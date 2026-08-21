# Lingganya 视频渠道适配

常见调用场景的完整 curl 示例见 [`curl-examples.md`](./curl-examples.md)。

本文说明 Lingganya 视频渠道在平台中的配置、模型能力和协议转换。客户侧接口仍以
[`视频生成 API`](../openapi/public-video-api.md)为唯一合同。

## 1. 渠道配置

使用 OpenAI 或 Sora 渠道类型，并将 Base URL 配置为 Lingganya 服务地址。视频协议固定为：

```json
{
  "video_protocol": "lingganya_video"
}
```

创建、查询和内容接口分别映射到：

```text
POST /v1/videos
GET  /v1/videos/{upstream_task_id}
GET  /v1/videos/{upstream_task_id}/content
```

三类上游请求均使用渠道 API Key 的 Bearer 鉴权。该协议只接受
`application/json` 和上游可访问的 HTTP/HTTPS 图片 URL，不支持 multipart 文件上传或 remix。

## 2. 参数兼容

平台在选定渠道后完成下列转换：

| 平台字段 | 兼容别名 | Lingganya 字段 | 规则 |
| --- | --- | --- | --- |
| `duration` | `seconds` | `seconds` | 接受整数字符串或整数；两者同时出现时必须一致 |
| `resolution` | 无 | 模型对应字段 | 平台清晰度档位；按能力模板映射，模型不接收时不发送 |
| `ratio` | `size` | `size` | 平台画幅；与 `resolution` 组合后映射为上游比例或像素尺寸 |
| `referenceImages` | `images`、`image`、`input_reference` | `images` | 保持首次出现顺序并去重 |
| `first_image` | 无 | `images` | 追加到普通参考图之后 |
| `last_image` | 无 | `images` | 追加到首帧之后，并按模型能力校验首帧依赖 |
| `provider_options.lingganya` | 无 | `extra` | 只允许不能覆盖公共参数、鉴权、回调或计费的扩展值 |

新客户端应使用平台标准字段 `duration`、`ratio` 和 `referenceImages`。只有能力模板声明清晰度档位时
才传 `resolution`。比例型模型直接把 `ratio` 转换为上游 `size`，例如 `ratio=16:9` 会发送
`size=16:9`，不发送 `resolution`。像素型模型使用模板中的 `size_mappings` 将公共清晰度和画幅组合
转换为上游像素尺寸；未传 `resolution` 时使用能力模板的首个档位。只有 `size`、`ratio` 或显式像素
尺寸互相描述了不同画幅时才拒绝请求。

像素型模型的映射键固定使用 `resolution|ratio`：

```json
{
  "resolutions": ["720p", "1080p"],
  "ratios": ["16:9"],
  "size_mappings": {
    "720p|16:9": "1280x720",
    "1080p|16:9": "1792x1024"
  },
  "omit_parameters": ["resolution"]
}
```

如果客户未传时长，适配器使用当前模型能力模板的 `default_duration_seconds`。模板同时配置
`allowed_duration_seconds`，即使一个值落在最小时长与最大时长之间，只要不在离散允许列表中也会
返回 `invalid_seconds`。

## 3. 内置模型能力模板

模板来源为 [Lingganya 接口文档](https://lingganya.apifox.cn/9147007m0)。模板会在服务启动时按
`video_protocol + model_id` 更新内置记录，管理员可在渠道编辑器中按上游模型 ID 应用模板。

| 模型 | 允许时长（秒） | 默认值 | 平台 `resolution` | 平台 `ratio` | 上游 `size` |
| --- | --- | --- | --- | --- | --- |
| `sora-2` | 4、8、12 | 4 | 不设置 | `16:9`、`9:16` | 原比例 |
| `sora-2-pro` | 12 | 12 | 不设置 | `16:9`、`9:16` | 原比例 |
| `sora-2-vip` | 12 | 12 | 不设置 | `16:9`、`9:16`、`19:6` | 原比例 |
| `gemini_omni_flash` | 10 | 10 | 不设置 | `16:9`、`9:16` | 原比例 |
| `gemini-omni-flash-special` | 10 | 10 | 不设置 | `16:9`、`9:16` | 原比例 |
| `veo_3_1_fast` | 8 | 8 | 不设置 | `16:9`、`9:16` | 原比例 |
| `veo_3_1_fast_hd` | 8 | 8 | 不设置 | `16:9`、`9:16` | 原比例 |
| `veo_3_1_fast_fl_hd` | 8 | 8 | 不设置 | `16:9`、`9:16` | 原比例 |
| `grok-imagine-video-1.5-preview` | 10、15 | 15 | `720p`、`1080p` | `16:9`、`9:16`、`1:1` | `1280x720`、`720x1280`、`1024x1024`、`1792x1024`、`1024x1792` |
| `grok-image-video-special` | 10、15 | 15 | 不设置 | `16:9`、`9:16`、`1:1` | 原比例 |
| `grok-video-1.5-special` | 10、15 | 15 | 不设置 | `16:9`、`9:16`、`1:1` | 原比例 |
| `sd-2.0-vip` | 4–15 | 6 | `720p` | `16:9`、`9:16` | 原比例 |

图片数量按合并后的 `referenceImages`、兼容别名、`first_image` 和 `last_image` 总数计算。除
`sd-2.0-vip` 外，模板将首尾帧作为 Lingganya `images` 处理；这些模型不支持参考视频、参考音频、原生音频、
随机种子或水印。

`sd-2.0-vip` 是多模态参考模型：时长可以使用 4–15 秒的任意整数，默认 6 秒；支持最多 9 张图片、3 个
参考视频和 3 个参考音频。参考视频和参考音频必须至少同时提供一张参考图片，平台会把它们转换为上游
要求的 `extra.reference_videos` 和 `extra.reference_audios` 字段。该模型当前模板按专项文档示例配置为
`720p`，如上游模型广场增加档位，应由管理员更新模板。

`grok-imagine-video-1.5-preview` 的固定尺寸映射如下：

- `720p + 16:9` -> `1280x720`
- `720p + 9:16` -> `720x1280`
- `720p + 1:1` -> `1024x1024`
- `1080p + 16:9` -> `1792x1024`
- `1080p + 9:16` -> `1024x1792`

上游没有提供 `1080p + 1:1` 的固定尺寸，因此平台会拒绝这一组合。

除 `grok-imagine-video-1.5-preview` 外，上表中的模型都不接收分辨率档位，平台模板不配置
`resolution`。这些比例型模型只根据 `ratio` 生成上游 `size`，例如 `16:9` 会发送
`size=16:9`。

## 4. 任务状态与内容交付

Lingganya 状态按平台公共状态转换：

| Lingganya | 平台 |
| --- | --- |
| `queued` | `queued` |
| `in_progress` | `in_progress` |
| `completed` | `completed` |
| `failed` | `failed` |

创建响应和轮询响应都只返回平台公开任务 ID。上游任务 ID、真实 `video_url` 和供应商扩展字段不会
暴露给客户。任务完成后，`url` 与 `video_url` 均指向平台
`/v1/videos/{public_task_id}/content`。

Lingganya 的 `video_url` 和 `/content` 需要上游 Bearer API Key。平台不会强制开启视频内容代理；
是否代理、是否自动归档到 S3、是否优先使用 S3 都由管理员通过三个独立开关控制。代理或自动归档
开启时，平台使用任务提交时冻结的上游密钥，避免渠道改 Key 或多 Key 轮换导致历史任务无法下载。

如果 S3 优先未命中且视频内容代理关闭，平台不会消耗服务器带宽回退代理。由于当前上游返回的是
需要鉴权的 HTTP 内容地址，平台会返回明确错误；管理员可等待 S3 归档完成、修复归档失败，或评估后
手动开启视频内容代理。

## 5. 计费约束

接口文档没有提供可写入平台的统一价格，本适配不会猜测或内置价格。渠道仍使用平台现有模型倍率、
固定价格或分层计费配置。预扣与结算都使用校验后的 `duration` 和 `resolution`；默认时长也会在预扣
前写入请求快照，确保提交、计费和最终响应一致。
