# OpenAI Video 兼容上游的任务与内容代理

## 1. 文档目的

本文档说明 zmodel 对 OpenAI Video 兼容上游的异步任务和视频内容下载代理方案。

当前直接场景包括通过 OpenAI 渠道类型接入 FriModel 和 MegabyAI 提供的 Seedance 2.0 封装接口，
以及通过同一视频入口接入 Agnes Video。Seedance 供应商复用 OpenAI Video 兼容协议；Agnes 的公共
业务语义相同，但时长线协议字段和专属参数存在差异，需要在 OpenAI/Sora task adaptor 边界内转换。
本文档描述协议兼容层和维护约束，不记录任何真实 API Key。

面向客户的接口、字段、响应和错误定义统一见
[`视频生成 API`](../openapi/public-video-api.md)，本文不重复定义客户合同。

代码和自动化测试是行为真相；本文档用于解释设计背景、关键契约和后续维护方式。

## 2. 背景与问题

上游提供异步任务创建、任务查询和视频内容下载三类 OpenAI Video 兼容能力。创建任务和查询任务均
需要上游 API Key。任务完成后，上游响应可能在以下字段返回受鉴权保护的下载地址：

```text
url
video_url
metadata.url
```

如果 zmodel 直接把上游下载地址返回给客户端，可能产生鉴权不匹配或绕过 zmodel 访问控制：

1. 客户端持有的是 zmodel Token。
2. 受保护的上游内容接口要求对应供应商的 API Key。
3. 客户端使用 zmodel Token 请求上游域名时会被上游拒绝。
4. 直接公开上游地址还会泄露供应商域名和上游任务 ID。

因此，视频任务响应统一返回 zmodel 的 `/v1/videos/{task_id}/content` 地址。客户端访问该地址时，
管理员可通过渠道的“代理视频内容”开关决定是由 zmodel 使用任务提交时实际选中的上游密钥代理下载，
还是重定向到上游视频地址；关闭代理时，管理员需要确认上游地址可由客户端直接访问。

## 3. 架构决策

### 3.1 复用现有 OpenAI Video 协议

供应商已经对 Seedance 2.0 接口进行了封装，并向下提供 OpenAI Video 兼容协议。因此 zmodel：

- 使用现有 OpenAI 渠道类型。
- 复用 OpenAI/Sora task adaptor。
- 不新增 Seedance 专用渠道类型。
- 不直接实现 Seedance 海外官方协议。
- 不新增视频提交、查询或内容下载路由。

视频协议按上游渠道定义，`megabyai` 和 `globalaiopc` 各自覆盖该上游使用同一套任务传输合同的所有视频模型。模型差异完全由渠道的 `video_model_capabilities` 定义，不通过模型名称分支实现。启用视频协议后，管理员必须从当前渠道的上游模型列表中选择模型并配置能力；模型映射场景按映射后的上游模型 ID 查询能力，未配置的模型会被拒绝。

对应的渠道 `setting` 数据结构如下，管理后台会以模型行、动态分辨率标签和数量输入框维护该结构：

```json
{
  "video_protocol": "megabyai",
  "video_model_capabilities": {
    "upstream-video-model": {
      "resolutions": ["720p", "1080p", "1440p"],
      "ratios": ["16:9", "9:16"],
      "ratio_required": true,
      "min_reference_images": 0,
      "max_reference_images": 2,
      "min_reference_videos": 0,
      "max_reference_videos": 1,
      "min_reference_audios": 0,
      "max_reference_audios": 0,
      "supports_duration": true,
      "duration_required": true,
      "min_duration_seconds": 4,
      "max_duration_seconds": 29,
      "supports_generate_audio": false,
      "generate_audio_required": false,
      "supports_first_frame": false,
      "first_frame_required": false,
      "supports_last_frame": false,
      "last_frame_required": false,
      "last_frame_requires_first_frame": false,
      "reference_images_incompatible_with_frames": false,
      "audio_reference_requires_visual_reference": false,
      "reference_media_incompatible_with_frames": false,
      "supports_seed": false,
      "supports_watermark": false
    }
  }
}
```

所有视频协议都要求每个模型至少配置一个分辨率。公共分辨率名称不使用全局白名单，`resolution_mappings` 可将 `1440p` 等稳定公共名称转换为上游的 `2k`、`1440P` 等值。参考素材最小值和最大值必须显式配置，最大值 `0` 表示该模型不支持对应素材。是否支持时长、时长是否必填以及范围也全部按模型配置。任务提交时会冻结最终命中的分辨率和时长计费能力，后续渠道配置变更不会改变进行中任务的结算规则。

管理端从视频能力模板接口加载与渠道协议匹配的模板。选择上游模型 ID 时会自动应用精确匹配模板，也可复制相似模板、批量应用所有匹配模型，并将调整后的能力保存为自定义模板。内置模板来自当前上游文档，只是可编辑的配置草稿，不参与公共模型路由；平台不会使用上游 `/v1/models` 自动修改客户模型合同。

统一视频请求支持以下 JSON 素材字段。zmodel 在调用上游前按模型能力校验数量，并保持字段名、数组顺序和 URL 原样透传：

```text
referenceImages
referenceVideos
referenceAudios
```

这样可以避免重复协议实现，并减少后续合并上游 new-api 代码时的冲突面。

### 3.2 统一视频参数与协议选择

客户端参数以[`视频生成 API`](../openapi/public-video-api.md)为准。适配器在渠道选定后读取视频协议和
模型能力，把统一字段转换为上游字段；任何协议差异都不得改变公共路由、任务状态或响应结构。

渠道 `setting` 只使用一个协议字段：

```json
{
  "video_protocol": "agnes_video_v2"
}
```

有效协议为 `openai_video`、`megabyai`、`globalaiopc` 和 `agnes_video_v2`。协议决定上游鉴权、任务端点、公共字段转换、状态解析和内容实时获取。
参数命名空间。空或缺失保持发布前的历史请求逻辑，不做隐式协议推断或数据迁移。管理后台只提供
“视频协议”下拉框，不提供严格、供应商选项或透传模式，也没有供应商命名空间输入框。

内部扩展参数仍在协议适配层完成白名单、深度和敏感字段校验，但不属于稳定客户合同，也不能选择
渠道或覆盖公共参数、计费、回调、鉴权和上游地址。面向客户的参数错误只返回稳定错误码和通用模型
表述，不返回渠道内部的 `video_protocol` 值或供应商命名空间。

### 3.3 MegabyAI

`megabyai` 是 MegabyAI 渠道级视频协议，Seedance、MiniMax H3 以及后续采用相同任务接口的模型都复用该协议。平台不会在运行时调用上游 `GET /v1/models`，也不会把上游动态模型 ID 直接暴露给客户；管理员可在线下查询上游模型目录，确认变化后在渠道的模型映射与模型能力中更新上游 ID。

适配器按当前上游模型 ID 的后台能力配置校验时长、分辨率、画幅、三类素材数量、首尾帧、原生音频、随机种子、水印和素材组合规则。能力开关不绑定特定模型名称，包括：

```text
supports_generate_audio
generate_audio_required
supports_first_frame
supports_last_frame
last_frame_requires_first_frame
reference_images_incompatible_with_frames
audio_reference_requires_visual_reference
```

这些开关全部由管理员按上游 `/v1/models` 和接口文档维护；后端只校验组合关系，例如要求 `generate_audio=true` 时必须同时支持原生音频，尾帧依赖首帧时也必须支持首帧。模型数值不是协议代码中的名单或固定参数，后续以上游确认结果更新后台配置或能力模板。

平台向上游发送的 `Idempotency-Key` 固定为 `zmodel:{public_task_id}`，不会透传客户请求头，因此重试同一个平台任务时上游幂等边界稳定，也不会让不同客户通过自定义键互相碰撞。

### 3.4 GlobalAiOpc

`globalaiopc` 与其他协议共用平台的创建、查询、内容、下载和任务 ID 合同，但在适配器内部使用 globalaiopc 的任务接口：

```text
POST /kyyReactApiServer/v2/model-center/tasks
GET  /kyyReactApiServer/v2/model-center/tasks/{upstream_task_id}
```

内部字段转换为：`ratio` 到 `aspect_ratio`，三类 camelCase 素材字段到对应 snake_case 上游字段；
分辨率按模型能力的 `resolution_mappings` 转换，例如 MiniMax H3 的公共 `1440p` 转为上游 `2k`。`generate_audio`、`seed` 和 `watermark` 仅在模型能力声明支持且客户显式传入时发送；`size` 不发送，避免与 `aspect_ratio` 冲突。

该协议当前只接受 `application/json` 和上游可访问的 HTTP/HTTPS 素材 URL，不支持 multipart 文件。渠道 Base URL 可配置为 `https://zcbservice.aizfw.cn` 或带 `/kyyReactApiServer` 前缀的地址，适配器会统一规范化。首批内置模板覆盖当前文档中的20个模型，所有模板应用后仍可由管理员调整。

GlobalAiOpc 的任务详情实际响应固定使用 `{ "data": { ... } }` 包装，适配器从 `data` 内解析
状态、视频地址、实际时长、分辨率及计费用量。按总 Token 结算时优先读取
`data.totalTokens`，不存在时回退 `data.usage.total_tokens`，并映射为渠道无关的
`TaskInfo.TokenUsage.TotalTokens`。该映射属于渠道适配器职责；后续其他渠道可以从不同响应字段
构造同一个 `TokenUsage`，任务结算层不依赖供应商字段名。

不论实际命中哪个视频协议，平台对客户重新构造统一响应，只保留平台字段。上游任务 ID、`result_url`、`amount`、`actualDuration`、供应商扩展 `metadata` 和真实下载 URL不会出现在创建或查询响应中。完成后的 `url` 与 `video_url` 始终指向平台 `/v1/videos/{public_task_id}/content`；S3 优先、平台代理或上游重定向仅由内容接口在访问时决定。

### 3.5 Agnes 时长协议转换

Agnes 渠道通过 `video_protocol=agnes_video_v2` 启用轻量转换，不按 Agnes 模型名称硬编码判断：

```text
客户端 duration（整数）
  -> 归一化并校验时长
  -> 使用归一化 duration 计费
  -> 删除上游请求中的 duration、seconds、num_frames 和 frame_rate
  -> 向 Agnes 发送 num_frames = duration * 24 + 1、frame_rate = 24
```

顶层字符串 `seconds` 暂时作为 Agnes 旧客户端的兼容别名；新客户端应使用公共 `duration`。两者同时
出现时必须一致，否则返回 HTTP 400 和 `duration_conflict`。响应解析兼容数字或字符串
`duration`，并在其缺失或无效时回退到数值字符串 `seconds`，包括 `"10.0"`。未传时长时 Agnes
渠道归一为约 5 秒。为保证视频流畅度，所有分辨率固定使用 24 fps：`480p`、`720p` 支持
1–18 秒；`1080p` 接受 1–18 秒，其中 11–18 秒自动按10秒处理，不通过降低帧率延长视频。

客户端可按下表选择时长：

| Agnes `resolution` | 公共 `duration` 允许值 | 固定帧率 | 最大请求帧数 |
| --- | --- | --- | --- |
| `480p` | 1–18 的整数 | 24 fps | 433 |
| `720p` | 1–18 的整数 | 24 fps | 433 |
| `1080p` | 1–18 的整数；11–18 归一为10 | 24 fps | 241 |

该范围是平台API合同，并统一适用于 Agnes 当前支持的全部宽高比。客户端必须显式传入渠道中为当前
模型配置的 `resolution`；不传 `duration` 时使用默认5秒。1080p 的 11–18 秒请求会在调用上游前
归一为10秒，
实际生成和计费均使用10秒；小于1秒或大于18秒返回 `invalid_seconds`。

Agnes 官方创建任务参数不包含 `seconds`；实际时长由 `num_frames` 和 `frame_rate` 控制，且
`num_frames` 遵循 `8n + 1`。官方文档给出的全局上限是 441 帧，但上游还会按分辨率和宽高比
设置更低上限；已确认 `1080p + 16:9` 最多接受 241 帧，因此该组合在 24 fps 下最长为 10 秒。
`seconds`、`num_frames` 和 `frame_rate` 均不属于
`provider_options.agnes`，也不能覆盖公共时长。只有无法归一为公共业务语义的 Agnes 专属能力才
放入供应商选项。

### 3.6 Agnes 分辨率协议转换

用户使用公共 `resolution` 和 `ratio`：

```json
{
  "resolution": "720p",
  "ratio": "16:9"
}
```

Agnes 适配器将其转换为 `width: 1280`、`height: 720`。分辨率名称由渠道按模型动态配置，Agnes
协议接受 `1p` 到 `4320p` 的数值型名称；支持的宽高比为 `16:9`、`9:16`、`1:1`、`4:3`、`3:4`。
`resolution` 必须显式传入，未传 `ratio` 时默认 `16:9`。Agnes 会把
名义尺寸映射到最近的内部标准输出尺寸，任务完成后以响应 `metadata.size_mapping.resolution` 和
`size` 为实际结果。

Agnes 统一模式不接受公共 `size`，也不允许 `provider_options.agnes` 覆盖 `width`、`height`、
`resolution` 或 `ratio`，避免请求、实际输出和计费使用不同分辨率。

### 3.7 统一参考图字段

用户统一使用 `referenceImages` 表达视频参考图片。`megabyai` 保持该数组字段和顺序发送上游；`globalaiopc` 转换为 `reference_images`。素材组合规则按模型能力校验。Agnes Video V2 只支持0或1张参考图，单张时由适配器转换为 Agnes 的顶层 `image` URL。
Agnes 请求不接受顶层 `image`、`images` 或 `input_reference`，也不允许通过
`provider_options.agnes` 覆盖参考图字段。多图、空 URL 或非 HTTP/HTTPS URL 在调用上游前返回
HTTP 400 和 `invalid_reference_images`。

Agnes 当前仅保证 `application/json` 中的图片 URL，不保证 multipart 本地文件上传。图片地址必须
能由 Agnes 上游直接访问。

### 3.8 区分公开任务 ID和上游任务 ID

任务包含两个不同用途的 ID：

| ID | 用途 | 是否可返回客户端 |
| --- | --- | --- |
| `Task.TaskID` | zmodel 公开任务 ID | 是 |
| `Task.PrivateData.UpstreamTaskID` | 上游真实任务 ID | 否 |

客户端提交、查询和下载时都使用 zmodel 公开任务 ID。zmodel 与上游通信时使用 `UpstreamTaskID`。

历史任务可能没有 `UpstreamTaskID`。此时 `Task.GetUpstreamTaskID()` 回退使用 `TaskID`，保持旧数据兼容。

### 3.9 保存任务实际使用的上游密钥

异步任务从创建到下载可能跨越较长时间。在此期间，渠道密钥可能发生以下变化：

- 管理员修改渠道密钥。
- 多 Key 渠道轮换到其他密钥。
- 请求重试后改用其他渠道或密钥。
- 原密钥仍能访问已创建任务，但当前渠道密钥不能访问。

因此，任务创建成功时必须保存本次请求最终实际使用的上游密钥：

```text
RelayInfo.ChannelMeta.ApiKey
    -> Task.PrivateData.Key
    -> task.private_data
```

`ChannelMeta.ApiKey` 来自请求上下文中的最终渠道密钥，因此能够反映多 Key 选择和重试后的结果。

该实现复用已有的 `TaskPrivateData.Key`，不增加数据库字段，也不需要数据库迁移。当前仅对原有 Gemini、Vertex AI 以及新增支持的 OpenAI、Sora 任务保存密钥，避免扩大其他任务平台的行为变化。

历史任务的 `PrivateData.Key` 可能为空。下载代理在这种情况下回退使用渠道当前密钥：

```text
Task.PrivateData.Key != "" ? Task.PrivateData.Key : Channel.Key
```

### 3.10 视频交付与 S3 归档独立配置

渠道包含三个相互独立的布尔配置：

| 配置 | 职责 |
| --- | --- |
| `video_content_proxy_enabled` | S3 优先未命中时，内容接口是否代理上游视频；访问内容接口时读取渠道当前值 |
| `video_s3_storage_enabled` | 任务成功后是否自动把视频归档到视频对象存储；任务创建时写入快照 |
| `video_s3_preferred` | 内容接口是否优先重定向到已有可用 S3 对象的临时签名地址；访问内容接口时读取渠道当前值 |

任务查询响应中的公开视频地址始终为：

```text
zmodel /v1/videos/{task_id}/content
```

客户端访问该内容接口时，再按以下顺序选择实际交付方式：

```text
S3 优先且对象可用 -> S3 临时签名地址
否则代理视频内容已开启 -> zmodel 流式返回上游视频
否则 -> 上游视频地址
```

自动归档开关不参与地址选择。管理员修改渠道的 `S3 优先`或`代理视频内容`后，该渠道所有已提交
任务后续访问内容接口时都会立即按当前配置重新选择实际交付方式。管理员也可以在任务日志中手动选择成功的视频任务
上传到 S3；因此即使任务创建时没有开启自动归档，只要后来存在可用的 S3 对象且渠道当前开启了
`S3 优先`，内容接口仍会重定向到 S3 临时签名地址。上传失败不改变视频任务状态和计费结果。

## 4. 请求数据流

### 4.1 创建任务

```text
客户端
  -> POST zmodel /v1/videos
  -> zmodel Token 鉴权与渠道分配
  -> POST 上游 /v1/videos（携带上游 API Key）
  -> 上游返回真实任务 ID
  -> zmodel 返回公开任务 ID
  -> zmodel 保存公开任务 ID、上游任务 ID和最终上游密钥
```

任务私有数据示意：

```json
{
  "key": "<stored-upstream-key>",
  "upstream_task_id": "task_upstream_xxx"
}
```

`private_data` 不通过任务 JSON 响应返回客户端。

### 4.2 查询任务

```text
客户端
  -> GET zmodel /v1/videos/{public_task_id}
  -> zmodel 按用户 ID和公开任务 ID查询任务
  -> 读取任务轮询保存的上游响应
  -> 将任务 ID重写为公开任务 ID
  -> 将所有视频下载地址统一重写为 zmodel /v1/videos/{public_task_id}/content
  -> 返回 OpenAI Video 兼容响应
```

因此，同一已提交任务始终返回稳定的平台内容接口地址；任务查询不读取渠道交付开关、不查询 S3 对象，
也不生成 S3 签名地址。

### 4.3 下载视频

```text
客户端
  -> GET zmodel /v1/videos/{public_task_id}/content
  -> zmodel Token 或用户会话鉴权
  -> 校验任务归属和完成状态
  -> 查询任务对应渠道的当前配置
  -> 当前开启 S3 优先且对象可用时重定向到 S3 签名地址
  -> 否则使用 UpstreamTaskID 和任务保存的上游密钥实时查询当前有效的上游视频地址
  -> 当前关闭代理视频内容时重定向到上游 HTTPS 地址
  -> 当前开启代理视频内容时由 zmodel 流式返回内容
```

`download_name` 只在 S3 签名地址或平台流式代理分支中控制下载文件名。关闭代理并跳转上游时，平台
无法控制上游响应头，因此忽略该参数且不会为了修改文件名而转发视频。上游结果是 `data:` URI 时
没有可跳转的远程地址，只能由平台直接输出内容。

客户端不需要知道上游域名、上游任务 ID或上游 API Key。

对外复制视频地址、播放器 `src` 和下载入口都必须使用
`/v1/videos/{public_task_id}/content`。下载时可以追加 `download_name` 查询参数，但不得把解析出的
S3 签名地址或上游视频地址暴露为可复制、可保存的业务地址；这些地址只作为内容接口内部的临时交付目标。

OpenAI/Sora 视频渠道不会把轮询时取得的上游真实视频地址持久化为后续交付依据，因为该地址可能带有有效期。平台内容地址也不需要入库，始终根据公开任务 ID按 `/v1/videos/{public_task_id}/content` 规则生成。自动 S3 归档使用任务完成轮询当次返回的新鲜地址；`/content` 和后台手动上传 S3 则在实际执行时按所选视频协议重新查询上游任务详情。数据库只长期保存实时查询所需的上游任务 ID和实际任务密钥。

### 4.4 失败任务的上游 HTTP 诊断

视频任务提交成功后，平台临时保存协议适配器实际发送的上游请求和上游提交响应。任务最终失败时，
再保存最后一次状态轮询的请求和失败响应，并在管理端任务日志详情中展示“提交任务”和“失败任务轮询”
两组 HTTP 报文。普通用户任务接口不返回这些诊断数据。

如果连接超时、DNS 解析失败、连接被拒绝或代理不可用导致上游没有返回任何 HTTP 响应，平台仍保存
实际请求，并以“传输错误”记录底层错误原因；此时不存在可展示的上游状态码、响应头或响应正文。
提交阶段经过渠道重试后仍失败会直接写入失败任务日志。轮询阶段的单次网络异常不会立即终止任务，
但会保存最近一次异常；任务后来成功则与其他诊断数据一并清除，最终超时或失败则供管理员查看。

任务成功后清除临时诊断数据，避免成功任务长期占用数据库空间。每个请求或响应正文最多保存 64 KiB，
超过上限时标记为已截断。`Authorization`、Cookie、API Key、Token、Secret、签名、凭据字段以及
URL 查询参数中的敏感值必须在写入数据库前脱敏；诊断数据不得保存真实渠道密钥。

诊断信息忠实记录上游请求与响应，不推断上游没有明确返回的信息。例如上游仅返回通用 403 或参数
错误而未提供素材序号、字段路径或 URL 时，平台不能准确判断具体是哪一个参考素材不可访问，管理端
只展示上游原始错误和已脱敏的实际请求，供管理员与上游核对。

管理端以规范 HTTP 报文结构重建并复制诊断内容：请求行为
`METHOD request-target HTTP/version`，响应行为 `HTTP/version status-code reason-phrase`，随后逐行展示
`Header-Name: value`，空一行后展示 Body。请求同时补齐 `Host` 和已知的 `Content-Length`。报文中的
凭据和敏感查询参数已经脱敏，JSON Body 会格式化，上述内容属于可与上游直接核对的规范化报文，
不是网络抓包的原始字节；HTTP/2 等二进制协议也统一以可读的 HTTP 报文形式展示。

## 5. 查询响应转换约束

完整客户响应合同见[`视频生成 API`](../openapi/public-video-api.md)。OpenAI/Sora task adaptor 的
`ConvertToOpenAIVideo` 负责落实以下内部约束：

- `id` 和 `task_id` 均改写为平台公开任务 ID；
- 上游状态归一为四种平台状态；
- 完成任务的 `url` 和 `video_url` 均按公开任务 ID 动态构造为平台 `/content` 地址；
- 非完成任务不返回视频 URL；
- 不返回上游任务 ID、真实视频 URL、`metadata`、`result_url` 或供应商响应扩展；
- 已知时长、分辨率和画幅优先取实际结果，否则使用任务请求快照。

## 6. 视频内容代理契约

渠道 `setting` 支持以下视频代理配置：

```json
{
  "video_content_proxy_enabled": true
}
```

配置行为：

- `true`：S3 优先未命中时，由 zmodel 获取并流式转发上游视频内容。
- `false`：默认值；S3 优先未命中时，zmodel 将客户端重定向到任务详情响应中的 HTTPS `url`。

`S3 优先`在上述代理判断之前执行，命中可用对象时直接重定向到 S3 签名地址。两个开关均在每次
访问内容接口时读取渠道当前值。关闭代理时，如果上游任务详情返回 HTTP `url`，接口返回错误
并提示管理员开启视频代理，避免 HTTPS 页面加载混合内容。

### 6.1 上游请求

每次访问公开内容接口时，OpenAI/Sora 渠道先实时请求上游任务详情：

```text
{Channel.BaseURL}/v1/videos/{upstream_task_id}
```

视频地址按以下明确顺序读取：

1. 响应顶层 `result_url`；
2. 响应顶层 `url`；
3. Grok 官方任务详情结构中的 `video.url`；
4. 响应顶层 `video_url`；
5. `metadata.url`；
6. `metadata.content_url`；
7. `metadata.local_url`；
8. `metadata.video_url`；
9. `metadata.final_video_url`。

解析时跳过非 URL 占位值，避免靠前的无效字段遮蔽后面的有效视频地址。以 `/`、`./` 或 `../`
开头的相对地址仅在没有绝对地址时使用，并基于本次任务详情请求地址转换为绝对地址。

不读取数据库快照、`metadata.origin_video_url` 或其他未定义的嵌套 URL 字段。

请求鉴权头为：

```text
Authorization: Bearer {stored_upstream_key}
```

当历史任务没有保存密钥时，使用当前 `Channel.Key`。

### 6.2 重定向交付

关闭视频代理时，zmodel 校验任务详情响应中的视频 URL，然后返回 `307 Temporary Redirect`。该 URL 必须使用 HTTPS，并通过 URL 与 SSRF 安全校验。

以下情况直接报错：

- 任务详情请求失败或返回非 2xx 状态；
- 任务详情响应无法解析；
- 所有受支持的视频 URL 字段均为空，或解析出的地址格式无效、使用 HTTP；
- `url` 被 URL 或 SSRF 安全策略拦截。

### 6.3 分段下载

视频播放器和下载工具通常依赖 HTTP Range。代理必须向上游转发：

```text
Range
If-Range
```

代理接受上游以下成功状态：

```text
200 OK
206 Partial Content
```

其他上游状态转换为 `502 Bad Gateway`，不直接把上游错误响应体透传给客户端。

### 6.4 响应头白名单

仅转发视频下载所需的响应头：

```text
Content-Type
Content-Length
Content-Range
Accept-Ranges
Content-Disposition
ETag
Last-Modified
```

不得无条件复制所有上游响应头，避免泄露上游内部信息、认证信息、调试信息或不适用于 zmodel 域名的缓存和 Cookie 设置。

下游缓存策略为：

```text
Cache-Control: private, max-age=86400
```

使用 `private` 是因为内容访问依赖用户身份和任务归属，不应被共享缓存存储为公共资源。

### 6.5 流式传输和超时

视频内容通过 `io.Copy` 流式转发，不在内存中读取完整视频。

内容代理不额外添加固定 60 秒的请求上下文超时。统一的 HTTP 客户端仍可受系统级 `RELAY_TIMEOUT` 配置约束。客户端断开连接时，请求上下文会取消上游请求。

## 7. 安全边界

### 7.1 密钥安全

- 上游密钥只保存在 `Task.PrivateData.Key`。
- `Task.PrivateData` 的 JSON 标签为 `json:"-"`，不得返回给客户端。
- 不得在日志、错误响应或测试快照中输出真实密钥。
- 自动化测试只能使用虚构密钥，例如 `stored-task-key`。
- 禁止把生产环境密钥写入文档、测试、配置样例或提交历史。

### 7.2 任务归属

普通用户访问内容接口时，必须使用当前用户 ID 和公开任务 ID 共同查询任务。管理员可以在自己的任务查询不到时按公开任务 ID 跨用户查询，以便查看和排查所有用户的任务；普通用户访问其他用户的任务统一返回未找到。

### 7.3 上游信息隔离

客户端响应不得暴露：

- 上游 API Key。
- 上游真实任务 ID。
- 上游下载 URL。
- 非必要的上游响应头。

### 7.4 SSRF 防护

内容代理继续使用现有 SSRF 校验和受保护 HTTP 客户端。不得为了兼容自定义 Base URL而全局关闭生产环境 SSRF 防护。

测试中可临时关闭 SSRF 防护以访问本地 `httptest.Server`，但测试结束后必须恢复全局设置。

## 8. 数据库与兼容性

本方案不新增字段，不需要迁移。

复用字段：

```text
Task.TaskID
Task.ChannelId
Task.PrivateData.Key
Task.PrivateData.UpstreamTaskID
Task.Data
```

兼容规则：

| 数据情况 | 处理方式 |
| --- | --- |
| 新任务有 `UpstreamTaskID` | 使用上游任务 ID请求内容接口 |
| 历史任务无 `UpstreamTaskID` | 回退使用 `TaskID` |
| 新任务有保存密钥 | 使用任务保存的密钥 |
| 历史任务无保存密钥 | 回退使用渠道当前密钥 |

`TaskPrivateData` 作为 JSON 存储，现有结构扩展和已有数据读取必须继续兼容 SQLite、MySQL 和 PostgreSQL。

## 9. 代码位置

主要实现位置：

| 文件 | 职责 |
| --- | --- |
| `model/task.go` | 保存任务密钥、上游任务 ID、自动 S3 归档开关和视频协议计费快照 |
| `dto/channel_settings.go` | 定义视频协议枚举和设置校验 |
| `model/channel.go` | 新增或更新渠道时校验视频请求设置 |
| `model/video_storage.go` | 保存视频 S3 对象、上传状态、重试状态和暂存文件元数据 |
| `relay/channel/task/sora/video_protocol.go` | 协议校验、安全保护、MiniMax 能力规则、供应商选项展开和 Agnes 转换 |
| `relay/channel/task/sora/adaptor.go` | 构造上游请求、MiniMax 幂等键、兼容响应时长并重写公开任务 ID和视频 URL |
| `relay/relay_task.go` | 将视频任务查询响应中的下载地址统一重写为平台内容接口地址 |
| `controller/relay.go` | 把自动 S3 归档开关和视频协议计费能力写入任务快照 |
| `controller/task.go` | 构建任务日志 DTO，并仅向管理员补充渠道名称、S3 对象状态和失败诊断 |
| `controller/video_proxy.go` | 动态重定向到 S3/上游或代理上游内容，处理 Range、下载文件名和响应头 |
| `controller/video_storage.go` | 自动或手动从上游获取视频，暂存后上传到视频 S3，并处理批量上传和重试 |
| `relay/channel/task/taskcommon/helpers.go` | 构造 zmodel 视频代理 URL |
| `router/video-router.go` | 注册现有视频提交、查询和内容代理路由 |

对应回归测试：

| 文件 | 覆盖范围 |
| --- | --- |
| `model/channel_settings_test.go` | 视频协议设置校验 |
| `relay/channel/task/sora/media_billing_test.go` | 协议 profile、供应商参数保护、Agnes 转换和计费时长 |
| `relay/channel/task/sora/minimax_h3_test.go` | MiniMax 稳定字段、能力组合、multipart、模型映射和幂等键 |
| `relay/channel/task/sora/adaptor_test.go` | ID和 URL重写、非成功状态地址清理 |
| `relay/channel/task/sora/live_e2e_test.go` | 可选真实接口 E2E，以及 E2E 流程自身的本地协议模拟测试 |
| `model/task_init_test.go` | 最终渠道密钥进入任务私有数据并成功落库 |
| `controller/task_test.go` | 任务日志始终返回平台内容接口地址，管理员仍可查看 S3 对象状态 |
| `controller/video_proxy_test.go` | S3/上游重定向、代理流式响应、`download_name`、Range、响应头和上游错误 |
| `controller/video_storage_test.go` | 自动归档、暂存重试、批量手动上传，以及关闭自动归档后的手动上传 |
| `service/task_polling_test.go` | 自动归档失败不影响任务结算和计费 |

## 10. 自动化测试矩阵

以下行为必须由自动化测试持续保护：

| 场景 | 预期结果 |
| --- | --- |
| 协议为空 | 保持迭代前的历史请求处理行为 |
| 已配置协议收到未知字段 | 返回 `unsupported_parameter`，不调用上游 |
| 供应商选项命名空间匹配 | 保持 JSON 类型并展开到上游请求顶层 |
| 供应商选项覆盖公共、计费或安全字段 | 返回 `provider_option_conflict`，不调用上游 |
| Agnes 请求使用公共 `duration: 10` | 上游收到 `num_frames: 241`、`frame_rate: 24`，计费使用 10 秒 |
| Agnes 请求使用 `1080p + duration: 10` | 上游收到 241 帧和 24 fps，保持流畅度 |
| Agnes 请求使用 `1080p + duration: 11–18` | 归一为10秒，上游收到241帧和24 fps，按10秒计费 |
| Agnes 请求使用 `720p + 16:9` | 上游收到 `width: 1280`、`height: 720` |
| Agnes 请求使用单个公共 `referenceImages` URL | 上游收到顶层 `image`，不残留公共字段 |
| Agnes 请求使用多图、原生参考图字段或 multipart 文件 | 返回 `invalid_reference_images`，不调用上游 |
| Agnes 响应包含 `metadata.size_mapping.resolution` | 使用标准化后的实际分辨率进行任务记录和结算 |
| Agnes 创建响应只含 `seconds: "10.0"` 和 `size: "1280x720"` | 返回公共 `duration: 10`、`resolution: "720p"`、`ratio: "16:9"`，保留兼容 `size` |
| Agnes 查询的实际时长与请求时长不同 | 公共 `duration` 返回上游实际时长，请求快照只作缺失回退 |
| Seedance 上游未返回 `seconds` 或 `size` | 仍从提交快照返回公共 `duration` 和 `resolution` |
| MiniMax 使用已配置的模型映射 | 上游收到映射后的模型 ID，客户模型名保持稳定 |
| MiniMax 请求缺少音频开关、画幅不支持或素材组合冲突 | 按后台模型能力返回对应 HTTP 400，不调用上游 |
| MiniMax JSON 或 multipart 请求 | 只接受稳定字段，URL 与文件合并计数并原样转发合法素材 |
| MiniMax 调用上游 | `Idempotency-Key` 使用 `zmodel:{public_task_id}`，忽略客户同名请求头 |
| 上游 `/v1/models` 返回模型目录变化 | 不影响运行时合同；管理员确认后更新模型映射与能力配置 |
| 任务能力快照完整但请求快照损坏 | 查询继续成功并使用能力快照允许的上游公共字段 |
| Agnes 完成结果为 1–18 秒 | 按任务保存的 Agnes 协议范围更新实际结算维度 |
| Agnes 同时收到不一致的 `duration` 和 `seconds` | 返回 `duration_conflict`，不调用上游 |
| Agnes 更换或新增模型名称 | 继续按渠道 `video_protocol` 转换，无需修改模型名单 |
| 成功任务查询 | `id` 和 `task_id` 为公开 ID |
| 成功任务查询 | 顶层 `url` 和兼容字段 `video_url` 均为 zmodel `/v1/videos/{task_id}/content` 地址，不返回 `metadata` |
| 已提交任务对应渠道的 S3 优先或代理开关发生变化 | 任务查询 URL 保持不变；下一次访问内容接口立即按渠道当前值选择交付方式 |
| 内容接口访问，S3 优先开启且对象可用 | 重定向到 S3 签名地址 |
| 内容接口访问，S3 不可用且代理开启 | 由 zmodel 流式代理上游视频 |
| 内容接口访问，S3 不可用且代理关闭 | 重定向到上游 HTTPS 视频地址 |
| 成功任务响应 | `id` 和 `task_id` 不包含上游任务 ID |
| 未完成或失败任务查询 | 删除所有上游下载 URL |
| 未完成或失败任务查询 | 不返回 `metadata` 或任何上游下载 URL |
| OpenAI 任务创建 | 保存请求上下文中的最终上游密钥 |
| 任务落库后重读 | 私有数据中的密钥保持不变 |
| 新任务下载 | 使用任务保存的密钥，而不是渠道当前密钥 |
| 历史任务下载 | 保存密钥为空时使用渠道当前密钥 |
| `proxy` Range 下载 | 转发 `Range` 和 `If-Range` |
| `proxy` 上游返回 `206` | 下游返回 `206` 和分段下载头 |
| `proxy` 上游返回 `200` | 下游流式返回完整内容 |
| `proxy` 上游返回非 `200/206` | 下游返回 `502` |
| `redirect` 上游返回重定向链 | 逐跳校验并解析最终视频地址，再向客户端返回绝对 `Location` |
| `redirect` 地址探测返回 `200/206` | 向客户端返回指向该地址的 `307`，不通过平台转发视频内容 |
| `redirect` 目标被安全策略拦截 | 返回 `403`，不暴露或访问目标内容 |
| `redirect` 请求携带 `download_name` | 仍重定向上游且忽略文件名，不通过平台转发视频 |
| 自动 S3 归档关闭的成功视频任务 | 管理员仍可从任务日志手动上传到 S3 |
| 自动或手动 S3 上传失败 | 不改变视频任务成功状态和计费结果 |
| 上游返回非白名单头 | 下游不转发该响应头 |

相关包的无缓存验证命令：

```bash
GOCACHE=/tmp/zmodel-go-build go test -count=1 \
  ./relay/channel/task/sora \
  ./model \
  ./controller
```

其中内容代理测试使用本地 `httptest.Server`，测试环境必须允许监听本地临时端口。

### 10.1 可选真实端到端测试

`relay/channel/task/sora/live_e2e_test.go` 提供环境变量驱动的真实端到端测试。它不依赖 CI，也不会把密钥写入代码或仓库。

默认情况下真实测试会跳过：

```bash
GOCACHE=/tmp/zmodel-go-build go test -count=1 -v \
  -run TestLiveOpenAIVideoE2E \
  ./relay/channel/task/sora
```

只有显式设置以下开关时，测试才会创建可能产生费用的真实视频任务：

```bash
export OPENAI_VIDEO_E2E_ENABLED=true
```

支持三种目标：

| `OPENAI_VIDEO_E2E_TARGET` | 使用的凭据 | 作用 |
| --- | --- | --- |
| `zmodel` | `ZMODEL_API_KEY` | 验证客户端经过 zmodel 的完整任务和下载代理链路，默认值 |
| `upstream` | `OPENAI_VIDEO_UPSTREAM_API_KEY` | 直接验证任意 OpenAI Video 兼容上游的协议和内容接口 |
| `both` | 两套凭据 | 依次验证 zmodel 和指定上游，会创建两个真实任务 |

验证 zmodel 完整链路：

```bash
export OPENAI_VIDEO_E2E_ENABLED=true
export OPENAI_VIDEO_E2E_TARGET=zmodel
export ZMODEL_BASE_URL=https://your-zmodel.example.com
export ZMODEL_API_KEY='<zmodel-test-key>'

GOCACHE=/tmp/zmodel-go-build go test -count=1 -v \
  -run TestLiveOpenAIVideoE2E \
  ./relay/channel/task/sora
```

直接验证任意 OpenAI Video 兼容上游：

```bash
export OPENAI_VIDEO_E2E_ENABLED=true
export OPENAI_VIDEO_E2E_TARGET=upstream
export OPENAI_VIDEO_UPSTREAM_BASE_URL=https://newapi.megabyai.cc
export OPENAI_VIDEO_UPSTREAM_API_KEY='<upstream-test-key>'

GOCACHE=/tmp/zmodel-go-build go test -count=1 -v \
  -run TestLiveOpenAIVideoE2E \
  ./relay/channel/task/sora
```

环境变量说明：

| 变量 | 必填条件 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `OPENAI_VIDEO_E2E_ENABLED` | 运行真实测试时 | `false` | 费用保护开关，必须显式设为 `true` |
| `OPENAI_VIDEO_E2E_TARGET` | 否 | `zmodel` | `zmodel`、`upstream` 或 `both` |
| `ZMODEL_BASE_URL` | 目标包含 zmodel | 无 | 测试请求使用的 zmodel 地址 |
| `ZMODEL_PUBLIC_BASE_URL` | 否 | `ZMODEL_BASE_URL` | 查询响应中应出现的公开地址，适用于内外网地址不同的部署 |
| `ZMODEL_API_KEY` | 目标包含 zmodel | 无 | 专用 zmodel 测试 Token |
| `OPENAI_VIDEO_UPSTREAM_BASE_URL` | 否 | `https://api.frimodel.com` | OpenAI Video 兼容上游地址 |
| `OPENAI_VIDEO_UPSTREAM_API_KEY` | 目标包含 upstream | 无 | 专用上游测试密钥 |
| `FRIMODEL_BASE_URL` | 否 | 无 | 旧版兼容变量；未设置通用地址时作为回退 |
| `FRIMODEL_API_KEY` | 否 | 无 | 旧版兼容变量；未设置通用密钥时作为回退 |
| `OPENAI_VIDEO_E2E_MODEL` | 否 | `videos-mini` | 待测试模型 |
| `OPENAI_VIDEO_E2E_PROMPT` | 否 | 内置英文测试提示词 | 视频提示词 |
| `OPENAI_VIDEO_E2E_DURATION` | 否 | `5` | 视频时长，单位为秒 |
| `OPENAI_VIDEO_E2E_RATIO` | 否 | `16:9` | 视频宽高比 |
| `OPENAI_VIDEO_E2E_RESOLUTION` | 否 | `720p` | 视频分辨率 |
| `OPENAI_VIDEO_E2E_POLL_INTERVAL` | 否 | `5s` | 轮询间隔，使用 Go duration 格式 |
| `OPENAI_VIDEO_E2E_TIMEOUT` | 否 | `15m` | 单个目标的总超时时间 |

真实测试执行以下检查：

1. `/v1/models` 返回配置的模型。
2. `/v1/videos` 成功创建任务并返回任务 ID。
3. 轮询任务直至 `completed`，失败或超时则测试失败。
4. zmodel 目标的查询响应始终使用公开任务 ID。
5. zmodel 目标在非完成状态不返回下载 URL。
6. 使用代理交付配置时，zmodel 完成响应中的顶层 `url` 和 `video_url` 均指向 zmodel 内容入口，且不返回上游 `metadata`。
7. 内容接口接受 zmodel Token 或上游密钥，并发送 `Range: bytes=0-1023`。
8. 代理交付配置下，内容接口返回 `200` 或 `206` 以及非空视频字节；返回 `206` 时同时验证 `Content-Range` 和 `Accept-Ranges`。

安全和运行约束：

- 建议使用低额度、可轮换的专用测试密钥。
- 密钥只从当前测试进程的环境变量读取。
- 测试不会输出 Authorization 请求头或密钥内容。
- 失败响应诊断会对当前测试密钥做脱敏处理，防止异常上游响应回显密钥。
- `both` 模式会创建两个任务，费用通常高于单目标模式。
- 常规回归测试不应设置 `OPENAI_VIDEO_E2E_ENABLED=true`。
- 运行结束后可使用 `unset` 清除当前 Shell 中的密钥环境变量。

真实 E2E 流程本身还有一个本地协议模拟测试 `TestRunLiveVideoE2EAgainstProtocolServer`。该测试不使用真实凭据、不产生费用，用于防止 E2E 测试代码因后续重构而失效。

## 11. 合并官方 new-api 代码时的检查项

zmodel 后续合并官方 new-api 源仓库时，应重点检查以下位置：

1. `model.InitTask` 是否仍保存 OpenAI/Sora 的最终上游密钥。
2. `TaskPrivateData.Key` 和 `UpstreamTaskID` 是否仍为私有字段并能正常持久化。
3. OpenAI/Sora adaptor 的 `ConvertToOpenAIVideo` 是否仍重写全部任务 ID和下载 URL。
4. 非成功状态是否仍会删除上游下载 URL。
5. `/v1/videos/:task_id/content` 是否仍经过 zmodel 鉴权和任务归属校验。
6. 内容代理是否优先使用任务保存的密钥，并保留历史任务回退逻辑。
7. `Range`、`If-Range`、`200` 和 `206` 支持是否完整。
8. 响应头是否仍采用白名单，而不是复制全部上游响应头。
9. SSRF 校验是否仍然有效。
10. `video_protocol` 四个协议值和空值行为是否正确。
11. Agnes 是否仍按 `video_protocol` 而不是模型名称执行 `duration -> num_frames + frame_rate` 转换，并固定保持 24 fps。
12. Agnes 是否仍把公共 `resolution + ratio` 转换为 `width + height`。
13. 供应商选项是否仍禁止覆盖时长、分辨率、计费、回调、鉴权和上游地址字段。
14. 本文档列出的回归测试是否全部通过。
15. MiniMax 是否仍只接受稳定字段、按动态模型能力校验，并使用平台公开任务 ID 构造上游幂等键。

为了降低合并冲突，维护时应继续遵循以下原则：

- 优先复用现有 OpenAI/Sora 任务编排，通过轻量协议 profile 隔离供应商字段差异。
- 优先复用现有字段、路由和 helper。
- 将供应商差异限制在已有 adaptor、任务初始化和内容代理扩展点。
- 不为单一供应商修改无关任务平台的通用行为。
- 每次行为变化同时更新测试和本文档。

## 12. 是否需要项目 Skill

当前实现不需要单独创建 Skill。本文档记录具体架构和维护契约，自动化测试保护实际行为。

当项目接入第二个或更多 OpenAI Video 兼容上游，并反复执行相同的接入、审查和测试流程时，可以创建项目级 Skill，例如：

```text
openai-video-provider-integration
```

Skill 应只维护可重复执行的流程清单，例如协议识别、任务 ID隔离、密钥快照、URL重写、Range 支持、安全检查和测试命令。具体实现细节仍以代码、测试和本文档为准，避免在 Skill 中复制整份设计说明而造成内容漂移。

## 13. Agnes 视频协议迭代文档

通用文档只记录已经进入 OpenAI Video 兼容层的稳定行为。本次迭代的设计决策、非目标、完整代码
映射、发布建议和可复制的 `curl` 验收步骤见：

- [Agnes 视频协议适配设计](../superpowers/specs/2026-08-01-agnes-video-provider-options-design.md)
- [Agnes 视频协议验收手册](../superpowers/specs/2026-08-01-agnes-video-provider-options-acceptance.md)

后续验收发现协议差异或实现发生调整时，应同时更新代码、自动化测试、对应迭代文档和本文档中的
稳定能力摘要。
