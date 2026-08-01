# Agnes 视频协议适配设计

## 状态与追溯

- 日期：2026-08-01
- 任务 slug：`agnes-video-provider-options`
- 开发分支：`codex/agnes-video-provider-options`
- 远端基线：`origin/main`
- 基线提交：`2013f642b48f38c623a5955c64a5d655cc11a2a3`
- 实现状态：代码已完成，待真实 Agnes 渠道验收
- 配套验收：[Agnes 视频协议验收手册](./2026-08-01-agnes-video-provider-options-acceptance.md)
- Agnes 参考：<https://www.agnes-ai.com/zh-Hans/docs/agnes-video-v20>

本文记录统一视频接口增加协议适配能力以及 Agnes Video V2 接入的最终设计。代码和自动化测试是
行为真相；验收手册记录部署后的真实渠道验证。

## 1. 目标

平台继续向用户提供统一的 `POST /v1/videos` 接口。用户使用公共业务字段表达需求，例如：

```json
{
  "model": "public-video-model",
  "prompt": "A cinematic sunrise over a quiet lake",
  "duration": 10,
  "resolution": "720p",
  "ratio": "16:9"
}
```

渠道协议适配器负责把公共字段转换成供应商线协议。用户不需要知道 Agnes 的 `num_frames`、
`frame_rate`、`width` 或 `height`。

本次设计满足以下目标：

1. 只用一个渠道设置选择视频协议，不暴露参数模式、供应商命名空间或透传开关。
2. Agnes 适配由协议配置启用，不依赖模型名称硬编码。
3. 公共 `duration`、`resolution`、`ratio` 和 `referenceImages` 是对应业务语义的稳定输入。
4. 无法抽象成公共字段的供应商能力仍可通过受保护的 `provider_options` 传入。
5. 空协议设置保持 `origin/main` 的历史视频行为，不做隐式协议推断或数据迁移。

## 2. 协议设置

渠道 `setting` 只新增一个字段：

```json
{
  "video_protocol": "agnes_video_v2"
}
```

有效值：

| 值 | 用途 |
| --- | --- |
| `openai_video` | 通用 OpenAI Video / Sora 兼容上游 |
| `seedance` | 当前项目已接入的 Seedance 兼容模型能力 |
| `agnes_video_v2` | Agnes Video V2 字段转换和校验 |
| 空或缺失 | 保持发布前历史逻辑 |

协议选择决定请求校验、字段转换和供应商扩展命名空间。系统不再提供 `strict`、
`provider_options` 或 `passthrough` 渠道模式，也没有 `video_provider` 设置。

当前生产环境只有一个 Seedance 视频渠道。发布后由管理员进入该渠道编辑页，显式选择
`Seedance Compatible` 即可；代码不为它添加隐藏默认值或自动迁移。Agnes 渠道必须显式选择
`Agnes Video V2`。

## 3. 协议边界

三种协议复用现有 OpenAI/Sora task adaptor 的任务编排、路由、鉴权、任务查询和内容下载能力，
只在请求边界使用轻量协议 profile：

```text
客户端统一请求
  -> 渠道分配和模型映射
  -> 读取 channel.setting.video_protocol
  -> 协议字段校验和公共参数归一化
  -> 协议出站转换
  -> 发送上游
```

### 3.1 OpenAI Video / Sora

保持通用 OpenAI Video 字段语义，不应用 Seedance 模型能力矩阵，也不执行 Agnes 转换。

### 3.2 Seedance

Seedance 继续使用 OpenAI Video 兼容路由，但作为独立协议 profile 维护已有模型能力矩阵：

| 上游模型 | 分辨率 | 时长 |
| --- | --- | --- |
| `videos-mini`、`videos-fast` | `480p`、`720p` | 4–15 秒 |
| `videos-standard` | `480p`、`720p`、`1080p`、`4K` | 4–15 秒 |
| `videos-4-mini`、`videos-4-fast`、`videos-4` | `480p`、`720p` | 4–15 秒 |

显式选择 Seedance 协议后，能力校验使用模型映射完成后的上游模型名，因此公开模型可以使用别名。

### 3.3 Agnes Video V2

Agnes profile 负责时长、分辨率和宽高比转换。它按渠道协议启用，新增或更换 Agnes 模型名称不需要
修改 Agnes 模型名单。

## 4. 供应商扩展参数

`provider_options` 是请求字段，不是渠道模式。命名空间由协议固定派生：

| 视频协议 | 请求命名空间 |
| --- | --- |
| `openai_video` | `provider_options.openai` |
| `seedance` | `provider_options.seedance` |
| `agnes_video_v2` | `provider_options.agnes` |

示例：

```json
{
  "model": "public-agnes-model",
  "prompt": "A cinematic sunrise over a quiet lake",
  "duration": 10,
  "provider_options": {
    "agnes": {
      "<agnes-specific-option>": "<value>"
    }
  }
}
```

配置了视频协议后，顶层只接受统一视频字段和 `provider_options`。适配器校验固定命名空间和扩展
字段，随后移除命名空间包装并把扩展字段合并到上游 JSON 顶层。字符串、数字、布尔值、对象和
数组保持原始 JSON 类型。multipart 请求不接受 `provider_options`。

`provider_options` 不参与渠道选择。渠道仍由公开模型、分组、优先级和模型映射选定。

## 5. Agnes 时长转换

### 5.1 入站

- 推荐字段：整数 `duration`。
- 兼容别名：字符串 `seconds`。
- 两者同时存在时必须相同，否则返回 HTTP 400 和 `duration_conflict`。
- 未传时长时使用 5 秒。
- 为保证视频流畅度，所有分辨率固定使用 24 fps，不通过降低帧率延长视频。

平台对外公开的基础约束如下：

| `resolution` | `duration` 允许值 | 固定帧率 | 最大请求帧数 | 说明 |
| --- | --- | --- | --- | --- |
| `480p` | 1–18 的整数 | 24 fps | 433 | 最长18秒 |
| `720p` | 1–18 的整数 | 24 fps | 433 | 默认分辨率 |
| `1080p` | 1–18 的整数 | 24 fps | 241 | 11–18 秒归一为10秒 |

该表是平台API合同，适用于当前支持的所有 `ratio`。Agnes 上游可能按分辨率和宽高比分别设置帧数
上限；平台对1080p统一采用已确认可用的10秒生成上限。用户请求 11–18 秒时，平台把公共
`duration` 归一为10秒后再进行计费和调用上游，避免业务流程被上游帧数限制中断。小于1秒或大于
18秒仍返回 HTTP 400，不创建上游任务。

### 5.2 出站

Agnes 官方创建任务没有 `seconds` 入参。实际时长由 `num_frames` 和 `frame_rate` 控制：

```text
frame_rate = 24
num_frames = duration * 24 + 1
```

适配器删除客户端请求中的 `duration`、`seconds`、`num_frames` 和 `frame_rate`，只发送转换后的
帧参数。10 秒转换为 `num_frames: 241`、`frame_rate: 24`。结果满足 `8n + 1`。Agnes 文档给出的
全局上限是 441 帧，但上游会按分辨率和宽高比进一步限制帧数；已确认 `1080p + 16:9` 最多为
241 帧。因此适配器在 1080p 下把 11–18 秒请求归一为 10 秒，而不是降低帧率生成不流畅的长视频。

计费、预扣、结算和任务输入快照统一读取归一化后的 `Duration`，不读取供应商扩展参数。任务创建时
把 `video_protocol` 保存到私有计费上下文，轮询完成结算按提交时协议校验实际时长：Seedance 为
4–15 秒，Agnes 的全局范围为 1–18 秒，请求阶段再把 1080p 的 11–18 秒归一为10秒，避免依赖
模型名称猜测协议。由于预扣和结算读取归一化后的 `Duration`，用户请求 18 秒时按实际的10秒计费。

### 5.3 响应

响应解析优先读取合法的数字或字符串 `duration`，缺失或无效时回退到 `seconds`。Agnes 返回的
小数字符串（例如 `"10.0"`）会安全转换为内部整数时长；NaN、Inf、负值和超大值不会进入计费。

创建和查询接口对外统一补齐以下公共字段：

- `model`：用户请求的公开模型名，不返回模型映射后的上游名称。
- `status`：统一为 `queued`、`in_progress`、`completed` 或 `failed`。
- `duration`：整数秒。查询时优先使用上游实际时长，缺失时回退到提交请求快照。
- `resolution`：逻辑分辨率档位。优先使用上游标准档位或 `metadata.size_mapping.resolution`，缺失时
  回退到提交请求快照。
- `ratio`：上游返回优先，缺失时回退到提交请求快照。

`seconds` 和 `size` 不是 Agnes 专属字段；Seedance 的部分 OpenAI Video/Sora 兼容封装也可能返回，
但各渠道不保证存在。平台在已知 `duration` 时把 `seconds` 统一为整数字符串以兼容历史客户端，
`size` 继续表示具体像素尺寸并作为可选兼容字段。用户应依赖公共 `duration` 和 `resolution`，不应
依赖供应商是否返回 `seconds` 或 `size`。供应商其他非敏感扩展字段继续保留。

## 6. Agnes 分辨率转换

公共请求使用 `resolution` 和 `ratio`。支持：

- 分辨率：`480p`、`720p`、`1080p`。
- 宽高比：`16:9`、`9:16`、`1:1`、`4:3`、`3:4`。
- 默认值：`720p + 16:9`。

适配器按分辨率短边和宽高比计算偶数名义尺寸，例如 `720p + 16:9` 转换为
`width: 1280`、`height: 720`。发送上游前删除公共 `resolution`、`ratio`、`size` 以及客户端原始
`width`、`height`。Agnes 协议不接受公共 `size`，避免多个分辨率事实来源。

内部结算解析优先使用顶层 `resolution`，其次使用 `metadata.size_mapping.resolution`。公共查询响应
只在 `size` 本身是 `480p`、`720p`、`1080p` 或 `4k` 档位时将其作为 `resolution`；像
`1280x720` 这样的具体尺寸不会被误当成逻辑分辨率，缺失档位时改为回退到提交请求快照。

## 7. 统一参考图与 Agnes 图生视频

平台使用已有公共字段 `referenceImages` 表达参考图片，不要求用户根据视频供应商切换成 Agnes 的
`image`、其他供应商的 `input_reference` 等线协议字段。当前已验证的协议映射为：

| 视频协议 | 公共输入 | 适配器出站行为 |
| --- | --- | --- |
| `seedance` | `referenceImages` | 保持数组和顺序，以同名字段发送上游 |
| `agnes_video_v2` | `referenceImages` | 只允许0或1张；单张转换为 Agnes 顶层 `image` URL |

Agnes 图生视频请求示例：

```json
{
  "model": "public-agnes-model",
  "prompt": "Animate the reference image with a slow camera movement",
  "referenceImages": [
    "https://example.com/reference.jpg"
  ],
  "duration": 10,
  "resolution": "720p",
  "ratio": "16:9"
}
```

Agnes 官方字段是单个图片 URL，因此适配器执行以下约束：

- 不传或传空数组时按文生视频处理。
- 传1个有效 HTTP/HTTPS URL 时，删除公共 `referenceImages` 并向 Agnes 发送 `image`。
- 传2张或更多图片时返回 HTTP 400 和 `invalid_reference_images`，不静默丢弃图片。
- 顶层 `image`、`images`、`input_reference` 不是平台 Agnes 公共合同，使用时返回
  `invalid_reference_images` 并提示改用 `referenceImages`。
- `provider_options.agnes` 不能覆盖 `image` 或 `referenceImages`。

当前 Agnes 官方文档定义的是可访问的图片 URL，不是本地文件上传。本次合同因此仅保证
`application/json` 请求中的 URL；URL 必须使用 HTTP 或 HTTPS，并能由 Agnes 上游访问。

## 8. 安全与计费保护

供应商扩展参数不能覆盖以下类别：

- 公共请求：`prompt`、`model`、`duration`、`seconds`、`resolution`、`ratio`、`size` 等。
- 输出和计费：`n`、`count`、`output_count`、`num_frames`、`frame_rate`、`width`、`height`。
- 回调：`callback_url`、`webhook` 及兼容写法。
- 鉴权和路由：`api_key`、`authorization`、`base_url` 及兼容写法。
- 包装字段：`metadata`、`provider_options`。

相同保护递归应用于嵌套对象和数组，最大深度为 16。冲突在调用上游和产生费用前返回 HTTP 400
和 `provider_option_conflict`。

## 9. 管理后台

入口：管理员“渠道”页面 → 新增/编辑 OpenAI 或 Sora 渠道 → “高级设置” → “渠道额外设置”。

页面只显示一个“视频协议”下拉框，包含：

1. OpenAI Video / Sora Compatible
2. Seedance Compatible
3. Agnes Video V2

未选择时保存空字符串并保持历史逻辑。页面没有启用开关、严格模式、供应商选项模式、透传模式或
供应商命名空间输入框。文案覆盖 `en`、`zh`、`zh-TW`、`fr`、`ja`、`ru`、`vi`。

## 10. 代码映射

| 责任 | 文件 |
| --- | --- |
| 协议枚举和渠道设置校验 | `dto/channel_settings.go` |
| 渠道保存时的设置校验入口 | `model/channel.go` |
| 任务私有计费上下文中的协议快照 | `model/task.go`、`controller/relay.go` |
| 协议请求校验、扩展参数保护、Agnes 转换 | `relay/channel/task/sora/video_protocol.go` |
| Seedance profile、请求构造和统一响应 | `relay/channel/task/sora/adaptor.go` |
| 后端协议、转换、统一响应和计费回归测试 | `relay/channel/task/sora/adaptor_test.go`、`relay/channel/task/sora/media_billing_test.go` |
| 渠道设置回归测试 | `model/channel_settings_test.go` |
| 渠道编辑页协议下拉框 | `web/default/src/features/channels/components/drawers/channel-mutate-drawer.tsx` |
| 前端表单校验和设置序列化 | `web/default/src/features/channels/lib/channel-form.ts` |
| 前端类型和翻译 | `web/default/src/features/channels/types.ts`、`web/default/src/i18n/locales/*.json` |

## 11. 自动化测试

自动化测试保护以下合同：

1. 三个协议值和非法设置校验。
2. 空协议保持历史请求字段处理。
3. Seedance 协议使用映射后的上游模型能力。
4. OpenAI Video 协议不误用 Seedance profile。
5. 固定供应商命名空间、JSON 类型保持和安全字段冲突拒绝。
6. Agnes `duration -> num_frames + frame_rate` 且不依赖模型名称，并固定保持 24 fps。
7. Agnes `resolution + ratio -> width + height`、默认值和非法组合拒绝。
8. 归一化时长和分辨率进入预扣维度，完成结算按协议快照采用 Seedance 或 Agnes 的实际范围。
9. Agnes 创建响应统一公开模型、状态、`duration`、`resolution`、`ratio` 和兼容字段。
10. Agnes 查询优先返回实际时长，缺失的逻辑分辨率和比例回退到提交快照。
11. Seedance 即使不返回 `seconds` 或 `size`，也能稳定返回公共 `duration` 和 `resolution`。
12. 历史任务的请求快照无效时仍可查询，不因新增归一化逻辑失败。
13. Agnes 1080p 在 10 秒时发送 241 帧和 24 fps，11–18 秒归一为10秒并按10秒计费。
14. Agnes 公共 `referenceImages` 单图转换成上游 `image`，多图、非法 URL 和供应商原生字段被拒绝。

## 12. 发布与验收

1. 发布代码后，手动把现有生产 Seedance 渠道选择为 `Seedance Compatible`。
2. Agnes 验收渠道选择 `Agnes Video V2`。
3. 使用配套验收手册验证 10 秒时长、分辨率、任务查询和视频下载。
4. 确认实际视频时长和消费记录一致后再投入生产。
5. 保存验收任务 ID、`ffprobe` 输出、错误响应和后台配置截图。
