# 视频生成 API

本文档是面向客户的视频 API 唯一人工可读合同。所有视频模型均使用相同的平台接口、字段名称、任务状态和响应结构；不同渠道或上游协议的差异由平台内部转换，不属于客户接口。

机器可读定义见 [`relay.json`](./relay.json)。

## 1. 接口概览

| 方法 | 路径 | 鉴权 | 用途 |
| --- | --- | --- | --- |
| `POST` | `/v1/videos` | Bearer Token | 创建视频生成任务 |
| `GET` | `/v1/videos/{task_id}` | Bearer Token | 查询任务状态和结果 |
| `GET` | `/v1/videos/{task_id}/content` | 无 | 播放或下载已完成的视频 |

除 `/content` 返回视频内容或重定向外，请求和响应均使用 `application/json`。

示例中的服务地址为：

```text
https://api.example.com
```

请替换为实际服务地址。创建和查询接口都使用 Bearer Token：

```http
Authorization: Bearer YOUR_API_KEY
```

创建接口还必须声明 JSON 请求体：

```http
Content-Type: application/json
```

`/content` 是由不可预测的 `task_id` 标识的公开能力地址，不要求 Bearer Token。获得该地址的人可以访问视频，因此不应将任务 ID 或内容地址泄露给无关人员。

## 2. 通用约定

### 2.1 公共模型与动态能力

客户只传平台提供的公共 `model` 名称。平台内部负责选择渠道并映射上游模型 ID。任务响应不会暴露渠道名称、视频协议名称、上游任务 ID 或上游真实视频地址。

每个模型支持的以下能力由平台后台动态配置：

- 时长范围及离散允许值；
- 分辨率名称；
- 画幅；
- 参考图片、视频和音频的数量；
- 首帧、尾帧和原生音频；
- 不同素材之间的组合限制。

请求是否支持 multipart 由平台所选适配器决定。因此，不应根据模型名称在客户端硬编码能力。请以服务方提供的当前模型能力和请求格式说明为准；不支持的值会在产生上游费用前返回参数错误。

### 2.2 字段命名

公共接口字段名区分大小写。三类普通参考素材使用 camelCase：

```text
referenceImages
referenceVideos
referenceAudios
```

首尾帧和原生音频使用 snake_case：

```text
first_image
last_image
generate_audio
```

不要传 `reference_images`、`reference_videos`、`reference_audios`、`aspect_ratio` 等上游字段别名，也不要传供应商专属字段。未知或当前模型不支持的字段会被拒绝。

### 2.3 可选字段

响应中未产生或不适用的可选字段会被省略，不会用 `null` 占位。例如：

- 任务未完成时不返回 `url` 和 `video_url`；
- 成功任务不返回 `error`；
- 尚无完成时间时不返回 `completed_at`；
- 上游未提供具体像素尺寸时不返回 `size`。

## 3. 创建视频任务

```http
POST /v1/videos HTTP/1.1
Authorization: Bearer YOUR_API_KEY
Content-Type: application/json
```

### 3.1 JSON 请求字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 平台提供的公共模型名称 |
| `prompt` | string | 是 | 视频生成提示词；不能为空，长度限制由模型能力决定 |
| `duration` | integer | 视模型而定 | 视频时长，单位为秒；必须为整数并位于当前模型的配置范围内 |
| `resolution` | string | 是 | 逻辑分辨率名称，例如 `720p`；按当前模型能力动态校验 |
| `ratio` | string | 视模型而定 | 视频画幅，例如 `16:9`、`9:16`；按当前模型能力动态校验 |
| `referenceImages` | string[] | 否 | 普通参考图片的公开 HTTP/HTTPS URL，数量按模型能力校验 |
| `referenceVideos` | string[] | 否 | 普通参考视频的公开 HTTP/HTTPS URL，数量按模型能力校验 |
| `referenceAudios` | string[] | 否 | 普通参考音频的公开 HTTP/HTTPS URL，数量和组合规则按模型能力校验 |
| `first_image` | string | 否 | 首帧图片的公开 HTTP/HTTPS URL；是否支持由模型能力决定 |
| `last_image` | string | 否 | 尾帧图片的公开 HTTP/HTTPS URL；是否支持及是否依赖首帧由模型能力决定 |
| `generate_audio` | boolean | 否 | 是否生成原生音频；是否支持、是否必须为 `true` 由模型能力决定 |
| `watermark` | boolean | 否 | 是否添加水印；仅在当前模型明确支持时传入 |

新接入必须使用 `duration` 和 `resolution`。历史兼容字段 `seconds`、`size` 不属于新的统一请求合同，不应在新客户端中使用。
`resolution` 与 `ratio` 的含义不随模型变化；平台会按实际命中的模型将二者转换为上游需要的档位、比例或像素尺寸。例如客户端传 `720p + 16:9` 时，上游可能收到 `resolution=720p`、`size=16:9`，也可能只收到 `size=1280x720`；传 `1080p + 16:9` 时，采用固定尺寸的模型可收到 `size=1792x1024`。

### 3.2 素材 URL 要求

素材 URL 必须满足以下要求：

- 使用 `http` 或 `https`；
- 上游服务能够直接访问；
- 在任务完成前持续有效；
- 不依赖浏览器登录态、本地 Cookie 或客户侧内网；
- 文件类型、大小和媒体时长符合当前模型要求。

平台只能确认 URL 格式和部分组合规则。若上游在处理阶段无法下载某个素材，任务可能异步变为 `failed`，此时通过任务的 `error` 字段获取公开错误信息。

### 3.3 素材组合规则

以下规则不是全局常量，均以当前模型能力为准：

- 三类普通参考素材各自的最大数量；
- 尾帧是否必须同时提供首帧；
- 普通参考图片能否与首尾帧同时使用；
- 参考音频是否必须同时提供参考图片；
- `generate_audio` 是否支持或是否必须开启。

数组顺序具有业务含义，平台在内部协议转换时保持数组顺序。

### 3.4 JSON 请求示例

```bash
curl --request POST 'https://api.example.com/v1/videos' \
  --header 'Authorization: Bearer YOUR_API_KEY' \
  --header 'Content-Type: application/json' \
  --data-raw '{
    "model": "video-model",
    "prompt": "根据提供的素材生成一段自然流畅的视频",
    "duration": 8,
    "resolution": "720p",
    "ratio": "16:9",
    "referenceImages": [
      "https://media.example.com/reference-01.jpg"
    ],
    "referenceVideos": [
      "https://media.example.com/reference-01.mp4"
    ],
    "referenceAudios": [
      "https://media.example.com/reference-01.mp3"
    ],
    "first_image": "https://media.example.com/first.png",
    "last_image": "https://media.example.com/last.png",
    "generate_audio": true
  }'
```

示例同时展示了全部主要字段，不代表任意模型都允许这种素材组合。实际请求只传当前模型支持且业务需要的字段。

### 3.5 multipart 请求

部分模型允许通过 `multipart/form-data` 直接上传素材，部分模型只接受 JSON 公网 URL。仅在服务方明确说明当前模型支持 multipart 时使用。

multipart 使用与 JSON 相同的文本字段，并提供以下文件字段：

| 字段 | 数量 | 说明 |
| --- | --- | --- |
| `referenceImageFiles` | 可重复 | 普通参考图片文件 |
| `referenceVideoFiles` | 可重复 | 普通参考视频文件 |
| `referenceAudioFiles` | 可重复 | 普通参考音频文件 |
| `first_image` | 1 | 首帧图片文件 |
| `last_image` | 1 | 尾帧图片文件 |

同类 URL 和文件合并计算数量。首帧或尾帧不能同时以 URL 和文件两种方式提交。不支持 multipart 的模型返回 `unsupported_content_type`。

### 3.6 创建成功响应

创建接口通常返回已排队的任务，也可能直接返回其它有效状态：

```json
{
  "id": "task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "task_id": "task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "object": "video",
  "model": "video-model",
  "status": "queued",
  "progress": 0,
  "created_at": 1774836724,
  "duration": 8,
  "resolution": "720p",
  "ratio": "16:9",
  "seconds": "8"
}
```

必须保存 `id` 或 `task_id` 用于后续查询。两者始终相同；`task_id` 是为既有客户端保留的兼容字段。

## 4. 查询视频任务

```http
GET /v1/videos/{task_id} HTTP/1.1
Authorization: Bearer YOUR_API_KEY
```

只能查询当前 Token 所属用户创建的任务。建议每 5–10 秒查询一次，避免高频轮询。

```bash
curl --request GET \
  'https://api.example.com/v1/videos/task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx' \
  --header 'Authorization: Bearer YOUR_API_KEY'
```

### 4.1 任务状态

| `status` | 说明 | 是否终态 |
| --- | --- | --- |
| `queued` | 已创建，等待上游处理 | 否 |
| `in_progress` | 正在生成 | 否 |
| `completed` | 已完成，可访问视频内容 | 是 |
| `failed` | 生成失败 | 是 |

不要根据上游可能使用的其它状态字符串编写业务逻辑；平台统一只返回以上四种状态。

### 4.2 响应字段

| 字段 | 类型 | 出现条件 | 说明 |
| --- | --- | --- | --- |
| `id` | string | 始终 | 平台公开任务 ID |
| `task_id` | string | 始终 | 兼容字段，与 `id` 相同 |
| `object` | string | 始终 | 固定为 `video` |
| `model` | string | 始终 | 请求时使用的公共模型名称 |
| `status` | string | 始终 | 统一任务状态 |
| `progress` | integer | 始终 | 0–100 的进度百分比；终态为 100 |
| `created_at` | integer | 始终 | Unix 时间戳，单位为秒 |
| `completed_at` | integer | 任务结束后 | 完成或失败时间，Unix 秒 |
| `expires_at` | integer | 上游提供时 | 上游结果过期时间，Unix 秒；不要据此保存临时下载地址 |
| `duration` | integer | 已知时 | 标准化后的视频时长，单位为秒 |
| `resolution` | string | 已知时 | 逻辑分辨率名称 |
| `ratio` | string | 已知时 | 逻辑画幅 |
| `seconds` | string | 已知时 | 历史兼容字段，值与 `duration` 表达相同秒数 |
| `size` | string | 上游提供时 | 历史兼容字段，具体像素尺寸，例如 `1280x720` |
| `url` | string | 仅 `completed` | 平台固定规则内容地址 `/v1/videos/{task_id}/content` |
| `video_url` | string | 仅 `completed` | 兼容字段，与 `url` 完全相同 |
| `error` | object | 仅 `failed` | 公开错误信息，包含 `code` 和 `message` |

响应不会包含 `metadata`、上游任务 ID、供应商域名、上游真实视频地址、`result_url`、计费金额或供应商扩展字段。

### 4.3 生成中响应

```json
{
  "id": "task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "task_id": "task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "object": "video",
  "model": "video-model",
  "status": "in_progress",
  "progress": 50,
  "created_at": 1774836724,
  "duration": 8,
  "resolution": "720p",
  "ratio": "16:9",
  "seconds": "8"
}
```

### 4.4 完成响应

```json
{
  "id": "task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "task_id": "task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "object": "video",
  "model": "video-model",
  "status": "completed",
  "progress": 100,
  "created_at": 1774836724,
  "completed_at": 1774836800,
  "duration": 8,
  "resolution": "720p",
  "ratio": "16:9",
  "seconds": "8",
  "url": "https://api.example.com/v1/videos/task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/content",
  "video_url": "https://api.example.com/v1/videos/task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/content"
}
```

客户应保存 `url`。复制视频地址、播放器 `src` 和下载入口也必须使用这个 `/content` 地址，不得保存其重定向目标。

### 4.5 失败响应

```json
{
  "id": "task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "task_id": "task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "object": "video",
  "model": "video-model",
  "status": "failed",
  "progress": 100,
  "created_at": 1774836724,
  "completed_at": 1774836750,
  "duration": 8,
  "resolution": "720p",
  "ratio": "16:9",
  "seconds": "8",
  "error": {
    "code": "generation_failed",
    "message": "video generation failed"
  }
}
```

失败任务不返回 `url` 或 `video_url`。

## 5. 获取视频内容

完成任务的固定视频地址为：

```text
GET /v1/videos/{task_id}/content
```

该地址不要求 Bearer Token，并始终保持同一格式。平台会在每次访问时按渠道当前配置选择实际交付来源：

1. 开启 S3 优先且 S3 文件可用时，返回 `307` 跳转到临时 S3 签名地址；
2. 否则，开启视频内容代理时，平台从当前有效上游地址获取并流式返回视频；
3. 否则，返回 `307` 跳转到当前有效的上游 HTTPS 视频地址。

视频是否自动上传 S3 与地址选择相互独立。即使任务生成时未自动上传，管理员之后也可以手动上传；开启 S3 优先后，后续访问同一个 `/content` 地址会自动使用可用的 S3 文件。

### 5.1 下载示例

客户端必须支持 HTTP 重定向。`curl` 使用 `-L`：

```bash
curl -L \
  'https://api.example.com/v1/videos/task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/content' \
  --output output.mp4
```

可通过 `download_name` 指定平台代理响应的下载文件名：

```bash
curl -L \
  'https://api.example.com/v1/videos/task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/content?download_name=my-video.mp4' \
  --output my-video.mp4
```

当平台返回 `307` 时，最终文件名由目标存储服务决定，`download_name` 不保证改变目标响应头。

### 5.2 Range 请求

平台代理模式支持播放器和断点续传需要的分段请求，并转发：

```http
Range: bytes=0-1048575
If-Range: "etag-value"
```

可能的成功响应：

| HTTP 状态 | 说明 |
| --- | --- |
| `200 OK` | 返回完整视频内容 |
| `206 Partial Content` | 返回请求的字节范围 |
| `307 Temporary Redirect` | 临时跳转到 S3 签名地址或上游 HTTPS 地址 |

`307` 响应的 `Location` 可能过期，也可能在管理员调整交付配置后变化。它只用于当前 HTTP 请求，不是可持久化的视频业务地址。

## 6. 错误处理

### 6.1 创建和查询错误

创建或查询接口的同步错误使用以下结构：

```json
{
  "code": "invalid_resolution",
  "message": "video model \"seedance-2.5-c1\" does not support resolution \"1440p\"; supported values: 480p, 720p",
  "data": {
    "parameter": "resolution",
    "received": "1440p",
    "allowed_values": ["480p", "720p"]
  }
}
```

参数类错误会尽可能在 `message` 中直接说明合法值或范围，并在 `data` 中提供可供程序读取的结构化信息：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `parameter` | string | 出错的公共参数名，统一使用文档中的规范名称 |
| `received` | 任意 | 客户端实际提交的值或数量；参数缺失时可省略 |
| `allowed_values` | array | 当前模型支持的合法值列表 |
| `minimum` | integer | 合法最小值，常用于时长、素材数量和随机种子 |
| `maximum` | integer | 合法最大值 |
| `required` | boolean | 参数是否为当前模型必填 |
| `related_parameters` | string[] | 与当前错误组合相关的其他公共参数 |

不同错误只返回适用字段。例如素材数量超限会返回 `minimum` 和 `maximum`，未知公共参数会返回 `parameter`，素材组合冲突会返回 `related_parameters`。错误内容不会暴露渠道内部协议名称、上游供应商名称或上游模型映射。

常见 HTTP 状态和错误码包括：

| HTTP 状态 | 常见 `code` | 说明 |
| --- | --- | --- |
| `400` | `invalid_request`、`invalid_json` | 请求体或必填字段无效 |
| `400` | `unsupported_parameter` | 字段不是公共字段或当前模型不支持 |
| `400` | `unsupported_content_type` | 当前模型不支持所用的请求格式 |
| `400` | `invalid_seconds` | 时长格式或范围无效 |
| `400` | `invalid_resolution`、`invalid_ratio` | 分辨率或画幅不受当前模型支持 |
| `400` | `invalid_reference_images`、`invalid_reference_videos`、`invalid_reference_audios` | 素材数量、URL 或组合不符合模型能力 |
| `400` | `invalid_first_image`、`invalid_last_image` | 首尾帧参数不符合模型能力 |
| `400` | `invalid_generate_audio` | 原生音频参数不符合模型能力 |
| `400` | `task_not_exist` | 查询的任务不存在或不属于当前用户 |
| `401` | 由鉴权模块返回 | Token 缺失或无效 |
| `403` | 由额度模块返回 | 额度不足或没有访问权限 |
| `429` | 由限流模块返回 | 请求过多或当前上游容量已满 |
| `500`、`502`、`503` | `server_error` 或具体服务错误码 | 平台或上游暂时不可用 |

错误消息用于直接定位和修改当前请求，不应依赖其完整文本编写业务分支；客户端应优先判断 HTTP 状态、`code` 和结构化 `data`。

### 6.2 内容接口错误

`/content` 的错误结构与任务接口不同：

```json
{
  "error": {
    "message": "Task not found",
    "type": "invalid_request_error"
  }
}
```

| HTTP 状态 | 说明 |
| --- | --- |
| `400` | 任务尚未完成或任务状态不允许下载 |
| `403` | 实际视频地址被 URL 或 SSRF 安全策略拦截 |
| `404` | 任务不存在 |
| `500` | 无法读取任务或渠道配置 |
| `502` | 无法从上游解析或获取视频内容 |

## 7. 完整调用流程

1. 调用 `POST /v1/videos` 创建任务。
2. 保存响应中的 `id` 或 `task_id`。
3. 每 5–10 秒调用 `GET /v1/videos/{task_id}`。
4. `status=completed` 时保存并使用 `url`。
5. `status=failed` 时读取 `error.code` 和 `error.message`，停止轮询。
6. 播放或下载时访问 `/v1/videos/{task_id}/content`，并允许客户端跟随 `307` 重定向。

对外展示和持久化的视频地址始终是平台 `/content` 地址。S3 签名地址和上游视频地址只是平台在访问当下选择的临时传输目标。
