# OpenAI Video 兼容上游的任务与内容代理

## 1. 文档目的

本文档说明 zmodel 对 OpenAI Video 兼容上游的异步任务和视频内容下载代理方案。

当前直接场景是通过 OpenAI 渠道类型接入 FriModel 和 MegabyAI 提供的 Seedance 2.0 封装接口。两个供应商复用同一套 OpenAI Video 兼容协议，供应商差异限于模型列表、定价和 `metadata` 扩展字段。本文档描述的是协议兼容层和维护约束，不记录任何真实 API Key。

代码和自动化测试是行为真相；本文档用于解释设计背景、关键契约和后续维护方式。

## 2. 背景与问题

上游提供以下 OpenAI Video 兼容接口：

```text
POST /v1/videos
GET  /v1/videos/{task_id}
GET  /v1/videos/{task_id}/content
```

创建任务和查询任务均需要上游 API Key。任务完成后，上游响应可能在以下字段返回受鉴权保护的下载地址：

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

因此，视频内容必须由 zmodel 代理下载。客户端始终访问 zmodel，由 zmodel 完成用户鉴权，并使用任务提交时实际选中的上游密钥访问上游。

## 3. 架构决策

### 3.1 复用现有 OpenAI Video 协议

供应商已经对 Seedance 2.0 接口进行了封装，并向下提供 OpenAI Video 兼容协议。因此 zmodel：

- 使用现有 OpenAI 渠道类型。
- 复用 OpenAI/Sora task adaptor。
- 不新增 Seedance 专用渠道类型。
- 不直接实现 Seedance 海外官方协议。
- 不新增视频提交、查询或内容下载路由。

当前通用 Seedance 模型能力如下：

| 模型 | 分辨率 | 时长 |
| --- | --- | --- |
| `videos-mini` | `480p`、`720p` | 4–15 秒 |
| `videos-fast` | `480p`、`720p` | 4–15 秒 |
| `videos-standard` | `480p`、`720p`、`1080p`、`4K` | 4–15 秒 |
| `videos-4-mini` | `480p`、`720p` | 4–15 秒 |
| `videos-4-fast` | `480p`、`720p` | 4–15 秒 |
| `videos-4` | `480p`、`720p` | 4–15 秒 |

其中 `videos-4*` 由 MegabyAI 提供。模型能力按模型 ID 定义，不通过供应商域名分支实现。

两个供应商均支持以下 JSON 素材字段，zmodel 保持字段名、数组顺序和 URL 原样透传，由上游执行数量、格式、时长和内容策略校验：

```text
referenceImages
referenceVideos
referenceAudios
```

这样可以避免重复协议实现，并减少后续合并上游 new-api 代码时的冲突面。

### 3.2 区分公开任务 ID和上游任务 ID

任务包含两个不同用途的 ID：

| ID | 用途 | 是否可返回客户端 |
| --- | --- | --- |
| `Task.TaskID` | zmodel 公开任务 ID | 是 |
| `Task.PrivateData.UpstreamTaskID` | 上游真实任务 ID | 否 |

客户端提交、查询和下载时都使用 zmodel 公开任务 ID。zmodel 与上游通信时使用 `UpstreamTaskID`。

历史任务可能没有 `UpstreamTaskID`。此时 `Task.GetUpstreamTaskID()` 回退使用 `TaskID`，保持旧数据兼容。

### 3.3 保存任务实际使用的上游密钥

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
  -> 重写任务 ID和视频下载地址
  -> 返回 OpenAI Video 兼容响应
```

完成状态下，响应中的下载地址统一指向：

```text
{ServerAddress}/v1/videos/{public_task_id}/content
```

### 4.3 下载视频

```text
客户端
  -> GET zmodel /v1/videos/{public_task_id}/content
  -> zmodel Token 或用户会话鉴权
  -> 校验任务归属和完成状态
  -> 查询任务对应渠道
  -> 使用 UpstreamTaskID 构造上游内容地址
  -> 使用任务保存的上游密钥请求上游
  -> 将视频内容流式返回客户端
```

客户端不需要知道上游域名、上游任务 ID或上游 API Key。

## 5. 查询响应转换契约

OpenAI/Sora task adaptor 的 `ConvertToOpenAIVideo` 负责转换任务响应。

### 5.1 所有状态

以下字段必须改为 zmodel 公开任务 ID：

```text
id
task_id
```

响应中不得残留上游任务 ID。

### 5.2 成功状态

任务状态为 `TaskStatusSuccess` 时，以下顶层字段必须改为 zmodel 内容代理地址：

```text
url
video_url
metadata.url
```

如果上游 `metadata` 原本包含以下扩展字段，也必须重写为同一个代理地址：

```text
metadata.content_url
metadata.local_url
metadata.video_url
metadata.final_video_url
```

`metadata.origin_video_url` 不属于稳定的最终结果下载契约，成功和非成功状态下均删除，避免暴露原始素材或上游存储地址。其他非 URL 元数据保持原样，例如 `cached`、`expires_in` 和 `cost_credits`。这样既兼容供应商扩展字段，也不在通用层引入供应商判断。

转换示例：

```json
{
  "id": "task_zmodel_xxx",
  "task_id": "task_zmodel_xxx",
  "status": "completed",
  "url": "https://zmodel.example.com/v1/videos/task_zmodel_xxx/content",
  "video_url": "https://zmodel.example.com/v1/videos/task_zmodel_xxx/content",
  "metadata": {
    "url": "https://zmodel.example.com/v1/videos/task_zmodel_xxx/content"
  }
}
```

### 5.3 非成功状态

任务未完成或失败时，必须删除以下字段中的上游下载地址：

```text
url
video_url
metadata.url
metadata.content_url
metadata.local_url
metadata.video_url
metadata.final_video_url
metadata.origin_video_url
```

删除 metadata URL 时必须保留 `metadata` 中的其他字段。这样既避免上游信息泄露，也不破坏其他响应元数据。

## 6. 视频内容代理契约

渠道 `setting` 支持以下内容交付配置：

```json
{
  "video_content_delivery": "proxy"
}
```

可选值：

- `proxy`：默认值，由 zmodel 获取并流式转发视频内容。
- `redirect`：要求上游内容接口返回 HTTP 重定向，由 zmodel 校验 `Location` 后将重定向返回客户端。

空值和历史渠道均按 `proxy` 处理。`redirect` 模式不会在任何错误场景回退到流式代理，避免异常消耗服务器出口带宽和长连接资源。

### 6.1 上游请求

OpenAI/Sora 渠道的上游内容地址为：

```text
{Channel.BaseURL}/v1/videos/{upstream_task_id}/content
```

请求鉴权头为：

```text
Authorization: Bearer {stored_upstream_key}
```

当历史任务没有保存密钥时，使用当前 `Channel.Key`。

### 6.2 重定向交付

`redirect` 模式禁止 HTTP 客户端自动跟随上游重定向，仅接受 `301`、`302`、`303`、`307` 和 `308`。zmodel 会解析相对 `Location`，执行 URL 与 SSRF 安全校验，再原样返回重定向状态和规范化后的绝对地址。

以下情况直接报错，不读取或转发上游视频流：

- 上游返回 `200`、`206` 或其他非重定向状态；
- 重定向缺少 `Location`；
- `Location` 无法解析；
- 重定向目标被 URL 或 SSRF 安全策略拦截；
- 上游请求失败。

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

### 6.4 流式传输和超时

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

内容代理必须使用当前用户 ID和公开任务 ID共同查询任务。不能仅凭任务 ID下载内容，否则可能造成跨用户访问。

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
| `model/task.go` | 创建任务时保存最终选择的上游密钥 |
| `relay/channel/task/sora/adaptor.go` | 重写公开任务 ID和视频代理 URL |
| `controller/video_proxy.go` | 鉴权后代理上游视频内容，处理 Range 和响应头 |
| `relay/channel/task/taskcommon/helpers.go` | 构造 zmodel 视频代理 URL |
| `router/video-router.go` | 注册现有视频提交、查询和内容代理路由 |

对应回归测试：

| 文件 | 覆盖范围 |
| --- | --- |
| `relay/channel/task/sora/adaptor_test.go` | ID和 URL重写、非成功状态地址清理 |
| `relay/channel/task/sora/live_e2e_test.go` | 可选真实接口 E2E，以及 E2E 流程自身的本地协议模拟测试 |
| `model/task_init_test.go` | 最终渠道密钥进入任务私有数据并成功落库 |
| `controller/video_proxy_test.go` | 保存密钥、历史回退、Range、200、206、响应头和上游错误 |

## 10. 自动化测试矩阵

以下行为必须由自动化测试持续保护：

| 场景 | 预期结果 |
| --- | --- |
| 成功任务查询 | `id` 和 `task_id` 为公开 ID |
| 成功任务查询 | 顶层和上游实际返回的 metadata 下载 URL 均指向 zmodel |
| 成功任务查询 | 响应不包含上游任务 ID和上游域名 |
| 未完成或失败任务查询 | 删除所有上游下载 URL |
| 未完成或失败任务查询 | 保留 `metadata` 中其他字段 |
| OpenAI 任务创建 | 保存请求上下文中的最终上游密钥 |
| 任务落库后重读 | 私有数据中的密钥保持不变 |
| 新任务下载 | 使用任务保存的密钥，而不是渠道当前密钥 |
| 历史任务下载 | 保存密钥为空时使用渠道当前密钥 |
| `proxy` Range 下载 | 转发 `Range` 和 `If-Range` |
| `proxy` 上游返回 `206` | 下游返回 `206` 和分段下载头 |
| `proxy` 上游返回 `200` | 下游流式返回完整内容 |
| `proxy` 上游返回非 `200/206` | 下游返回 `502` |
| `redirect` 上游返回重定向 | 不跟随重定向，校验并向客户端返回绝对 `Location` |
| `redirect` 上游返回 `200/206` | 返回 `502`，不回退到反向代理 |
| `redirect` 目标被安全策略拦截 | 返回 `403`，不暴露或访问目标内容 |
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
6. zmodel 完成响应中的顶层 URL 和上游实际返回的 metadata URL 均指向 zmodel 内容代理。
7. 内容接口接受 zmodel Token 或上游密钥，并发送 `Range: bytes=0-1023`。
8. 内容接口返回 `200` 或 `206` 以及非空视频字节；返回 `206` 时同时验证 `Content-Range` 和 `Accept-Ranges`。

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
10. 本文档列出的回归测试是否全部通过。

为了降低合并冲突，维护时应继续遵循以下原则：

- 优先扩展现有 OpenAI/Sora 通道，不新增供应商专用协议层。
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
