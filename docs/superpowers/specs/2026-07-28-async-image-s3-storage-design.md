# 异步图片生成与 S3 临时存储设计

## 状态

- 日期：2026-07-28
- 项目：zmodel
- 范围：OpenAI 兼容图片生成、异步任务、私有 S3 归档、对象生命周期、计费一致性、后台对象存储设置
- 起点提交：`c0ffb2aca7b467db6b1f0688464033ee1216e244`
- 设计状态：待用户复核

## 1. 背景

当前 `POST /v1/images/generations` 在一次 HTTP 请求中完成渠道选择、上游图片生成、
计费结算和响应返回。不同图片渠道最终都会通过图片适配器输出标准 OpenAI 图片响应，
但输出可能是公开或临时 URL、纯 Base64，或者 data URI。同步接口不能为调用方提供统一的
私有存储、固定有效期和可恢复的归档过程。

本功能新增异步图片任务接口。请求提交后先完成鉴权、校验、定价和全额预扣费，再由后台
执行图片生成，将所有结果归档到私有 S3。查询接口只在对象有效时动态返回预签名 GET URL。
原有同步接口及其行为保持不变。

## 2. 已确认需求

1. 保留 `POST /v1/images/generations`。
2. 新增：
   - `POST /v1/images/generations/tasks`
   - `GET /v1/images/generations/tasks/{task_id}`
3. 异步接口支持当前所有图片模型，并兼容 URL、纯 Base64 和 data URI 输出。
4. 异步接口不支持 `stream=true`。
5. 图片归档到私有 S3；不保存预签名 URL；查询时动态签名。
6. Object Key 固定为：

   ```text
   prod/user-files/zmodel@async-images/{user_id}/{yyyy}/{mm}/{task_id}/{index}.{extension}
   ```

7. 使用通用 `storage_objects` 表；本业务的 `business_id` 固定为
   `zmodel@async-images`，`resource_id` 为任务 ID，并单独保存 `object_index`。
8. 任务和对象记录永久保留。对象到期后立即停止签名，定时删除 S3 对象，但不删除数据库记录。
9. 图片有效期按每个对象上传完成时间固化。
10. 后台新增独立的“对象存储”设置页面。
11. 设置包括 S3 连接信息、对象有效期、预签名时长、单次归档超时、最大尝试次数、最长重试窗口和清理间隔。
12. 单次归档超时默认 10 分钟，最大 20 分钟；最长重试窗口默认 6 小时。
13. 不增加启用开关，不提供 Key Prefix 配置；文件大小限制复用现有 `MAX_FILE_DOWNLOAD_MB`。
14. 任务 `status` 与 `output.availability` 分离。
15. 归档最终失败时只退款一次，并向 Root 用户发送一条聚合管理员通知。
16. 已退款任务即使人工恢复，也不能自动重新扣费。
17. 通知复用 `service.NotifyRootUser`，沿用 Root 用户的 Email、Webhook、Bark 或 Gotify 设置。
18. Secret 不通过任何读取设置接口回显；更新时空 Secret 保留原值。

## 3. 目标

1. 复用现有图片适配器和渠道重试能力，不为每个图片渠道复制异步实现。
2. 在多节点、进程崩溃和任务重复执行情况下，避免重复结算或重复退款。
3. 在任何时刻都不向调用方暴露部分归档结果。
4. 不把 Base64 图片、data URI 图片、渠道密钥或 S3 Secret 写入数据库或磁盘。
5. 让 URL 输出能够在进程重启后继续归档。
6. 保持 SQLite、MySQL 5.7.8+ 和 PostgreSQL 9.6+ 兼容。
7. 让对象存储配置可在 Root 后台安全维护，并让新配置只在明确边界影响任务。

## 4. 非目标

- 不把同步图片接口改造成异步接口。
- 不支持异步图片编辑、variation 或 multipart 图片上传；本期只覆盖
  `POST /v1/images/generations/tasks`。
- 不增加公开对象、公共 Bucket 或长期不失效 URL。
- 不增加 S3 Key Prefix 设置，也不允许调用方自定义 Object Key。
- 不在本期提供任务列表、取消任务、重试任务或人工恢复的管理 UI/API。
- 不把现有视频、音乐等供应商轮询任务迁移到新表。
- 不保证外部通知服务的严格 exactly-once 投递；系统只生成一个聚合通知意图并做单任务去重。

## 5. 方案比较

### 5.1 选定方案：独立异步图片任务 + 通用对象表

新增 `AsyncImageTask` 承载图片生成、归档、计费和租约状态，新增通用
`StorageObject` 保存对象定位和生命周期。图片执行逻辑从同步处理器中抽出为不负责最终计费的
共享执行单元，同步和异步入口分别完成自己的响应与结算。

优点：

- 图片任务状态和供应商轮询任务状态不会互相污染。
- 可以明确表达“生成成功但归档失败”。
- 可以用任务行作为计费幂等边界。
- 通用对象表以后可以被其他临时文件业务复用。

代价：

- 需要新增任务模型、后台执行器和清理执行器。
- 需要把当前图片执行与同步计费拆开。

### 5.2 未选：复用现有 `model.Task`

现有 `model.Task` 面向具有上游任务 ID 的视频、音乐等轮询任务，状态值、平台、私有数据和
结算流程都与本功能不同。强行复用会让 `status`、输出可用性和归档重试混在同一组字段里，
也会继续依赖目前不具备崩溃幂等性的任务退款函数。

### 5.3 未选：引入外部消息队列

Kafka、RabbitMQ 或云队列可以提供更强的消费能力，但会新增部署依赖，并且不能替代数据库中的
计费状态和对象生命周期。当前已有带全局租约的系统任务框架，优先扩展该框架更符合项目现状。

## 6. 总体架构

```text
POST /v1/images/generations/tasks
  -> TokenAuth / 限流
  -> 解析、校验、敏感词检查、模型权限和渠道可用性检查
  -> S3 配置完整性检查
  -> 定价并冻结计费与归档配置快照
  -> 同库事务：创建任务 + 全额预扣钱包/订阅/Token 配额
  -> 202 task response
  -> 唤醒全局 async-image-processing system task

async-image-processing
  -> 全局 SystemTask 租约
  -> CAS 领取 AsyncImageTask 行租约
  -> 用当前渠道配置执行共享图片中继逻辑
  -> 捕获标准 OpenAI ImageResponse
  -> URL/Base64/data URI 标准化
  -> 私有 S3 上传及 StorageObject 状态更新
  -> 同库事务：最终结算 + output available
  -> 完整归档清单持久化时清除原始请求正文

GET /v1/images/generations/tasks/{task_id}
  -> TokenAuth
  -> 只查询当前用户的任务
  -> 根据任务和 StorageObject 判断可用性
  -> 对每个未过期对象动态生成预签名 GET URL

async-image-storage-cleanup
  -> 全局 SystemTask 租约
  -> 删除到期或 delete_pending 的 S3 对象
  -> 永久保留任务和对象数据库记录
```

## 7. 数据模型

### 7.1 `async_image_tasks`

新增独立模型 `model.AsyncImageTask`。所有 JSON 快照使用 `TEXT` 持久化并通过
`common.Marshal`、`common.Unmarshal` 实现 `driver.Valuer`/`sql.Scanner`，避免数据库特有
JSON 类型成为迁移前提。

| 字段 | 类型/索引 | 含义 |
| --- | --- | --- |
| `id` | `int64` 主键 | GORM 生成主键 |
| `task_id` | `varchar(64)` 唯一索引 | 对外 `task_...` ID |
| `user_id` | `int` 索引 | 任务所有者 |
| `token_id` | `int` 索引 | 提交任务的 Token ID，不保存 Token Key |
| `status` | `varchar(32)` 索引 | `queued/running/succeeded/failed` |
| `output_availability` | `varchar(32)` 索引 | `pending/archiving/available/expired/failed` |
| `billing_status` | `varchar(32)` 索引 | `reserved/settled/refunded` |
| `billing_source` | `varchar(32)` | `wallet/subscription` |
| `subscription_id` | `int` | 订阅资金来源 ID |
| `reserved_quota` | `int` | 已事务性预扣额度 |
| `actual_quota` | `int` | 成功结算后的最终额度 |
| `token_reserved_quota` | `int` | 实际从 Token 配额扣除的额度；无限 Token 为 0 |
| `origin_model_name` | `varchar(191)` 索引 | 面向用户的计费模型名 |
| `using_group` | `varchar(64)` | 提交时解析后的实际分组；`auto` 必须解析为具体分组 |
| `last_channel_id` | `int` 索引 | 最后实际执行的渠道，仅用于日志和运行时重建，不保存 Key |
| `request_payload` | `text` | 原始 JSON 请求；完整归档清单持久化或生成失败后清空 |
| `billing_context` | `text` | 冻结的价格、表达式、分组倍率和请求输入快照 |
| `archive_manifest` | `text` | 输出索引、来源类型、URL 来源和 revised prompt；不含 Base64/data URI 内容 |
| `retention_seconds` | `int64` | 提交时冻结的对象有效期 |
| `archive_timeout_seconds` | `int64` | 提交时冻结，范围不超过 1200 秒 |
| `archive_max_attempts` | `int` | 提交时冻结 |
| `archive_retry_deadline_at` | `int64` 索引 | 提交时间加最长重试窗口 |
| `archive_attempts` | `int` | 已开始的业务归档尝试次数 |
| `next_attempt_at` | `int64` 索引 | 下次可领取时间 |
| `output_expires_at` | `int64` 索引 | 全部输出中最早的 `expires_at` |
| `lease_owner` | `varchar(128)` 索引 | 当前任务行租约持有者 |
| `lease_expires_at` | `int64` 索引 | 行租约到期时间 |
| `source_kind` | `varchar(32)` | `none/url/ephemeral/mixed`，供崩溃恢复判断 |
| `public_error_code` | `varchar(64)` | 对调用方稳定的错误码 |
| `public_error_message` | `text` | 已脱敏的错误说明 |
| `last_error` | `text` | 管理和日志使用的内部错误摘要，不包含密钥或 Base64 |
| `generation_completed_at` | `int64` | 已取得完整标准图片响应的时间 |
| `billing_finalized_at` | `int64` | 结算或退款完成时间 |
| `manually_recovered_at` | `int64` | 未来人工恢复输出的审计时间；自动路径保持为 0 |
| `admin_notification_state` | `varchar(32)` | `none/pending/claimed/sent` |
| `admin_notification_claimed_at` | `int64` | 聚合通知去重和超时恢复 |
| `created_at` | `int64` 索引 | 创建时间 |
| `started_at` | `int64` | 首次开始执行时间 |
| `completed_at` | `int64` | 业务终态时间 |
| `updated_at` | `int64` 索引 | 更新时间 |

复合索引：

- `(status, output_availability, next_attempt_at)`：领取可运行任务。
- `(lease_expires_at, next_attempt_at)`：回收过期租约。
- `(user_id, task_id)`：按所有者查询。
- `(billing_status, output_availability)`：计费终态检查与修复审计。

不使用 GORM 布尔默认标签，不使用数据库特有枚举、JSON 列或部分索引。

### 7.2 `storage_objects`

`StorageObject` 是业务无关的私有对象记录。本功能使用固定
`business_id = zmodel@async-images`。

| 字段 | 类型/索引 | 含义 |
| --- | --- | --- |
| `id` | `int64` 主键 | GORM 生成主键 |
| `business_id` | `varchar(64)` 复合唯一索引 | 业务命名空间 |
| `resource_id` | `varchar(64)` 复合唯一索引 | 本业务为 `task_id` |
| `object_index` | `int` 复合唯一索引 | 从 0 开始的图片序号 |
| `provider` | `varchar(32)` | 本期固定 `s3` |
| `status` | `varchar(32)` 索引 | `uploading/available/delete_pending/deleted` |
| `endpoint` | `varchar(512)` | 创建对象记录时使用的非敏感 S3 Endpoint 快照 |
| `region` | `varchar(128)` | S3 Region 快照 |
| `bucket` | `varchar(255)` | Bucket 快照 |
| `object_key` | `varchar(768)` | 固定规则生成的 Key |
| `mime_type` | `varchar(128)` | 由实际内容识别的 MIME |
| `extension` | `varchar(32)` | 由实际内容识别的扩展名，不含点 |
| `size_bytes` | `int64` | 解码后的对象大小 |
| `etag` | `varchar(255)` | S3 返回的 ETag，可能为空 |
| `uploaded_at` | `int64` | S3 Put 成功时间 |
| `expires_at` | `int64` 索引 | `uploaded_at + task.retention_seconds` |
| `deleted_at` | `int64` | S3 删除确认时间 |
| `delete_attempts` | `int` | 删除尝试次数 |
| `last_error` | `text` | 脱敏后的上传/删除错误摘要 |
| `created_at` | `int64` 索引 | 创建时间 |
| `updated_at` | `int64` 索引 | 更新时间 |

唯一约束为 `(business_id, resource_id, object_index)`。重试对同一索引使用相同 Key 并覆盖，
不会创建第二条对象记录。

S3 Endpoint、Region 和 Bucket 在首次准备上传该索引时写入对象记录。后续重试继续使用该定位
快照，避免配置切换后同一任务的图片散落到不同 Bucket。Access Key 和 Secret 始终从当前设置
读取，绝不写入对象表。

### 7.3 迁移

两个模型都加入 `migrateDB` 和 `migrateDBFast`。迁移只使用 GORM `AutoMigrate` 和普通索引，
兼容 SQLite、MySQL 5.7.8+ 和 PostgreSQL 9.6+。不在迁移中使用 `ALTER COLUMN`、数据库特有
JSON 操作或方言专用默认值。

## 8. 状态机

### 8.1 生成状态 `status`

```text
queued -> running -> succeeded
                  -> failed
```

- `queued`：请求和预扣费已持久化，等待后台执行。
- `running`：正在选择渠道或等待上游图片结果。
- `succeeded`：已经取得并验证完整的标准图片响应。该状态不表示对象仍可访问。
- `failed`：上游生成在现有渠道重试后最终失败，或标准响应本身无效。

`succeeded` 和 `failed` 都是生成状态终态，不因对象到期、归档失败或人工恢复而改变。

### 8.2 输出可用性 `output.availability`

```text
pending -> archiving -> available -> expired
                    -> failed
pending -------------------------> failed
```

- `pending`：尚未取得可归档输出。
- `archiving`：已取得生成结果，正在上传、重试或完成计费终结。
- `available`：全部对象上传成功、未过期且计费已结算。
- `expired`：至少一个对象已到期；查询不返回任何部分结果。
- `failed`：生成失败或归档最终失败；查询不返回任何部分结果。

自动执行路径只能在全部对象均为 `available`、全部未过期且 `billing_status=settled` 时进入
`output.availability=available`。唯一例外是未来显式的人工恢复操作：它必须写入
`manually_recovered_at`，并允许保持 `billing_status=refunded`。

### 8.3 计费状态 `billing_status`

```text
reserved -> settled
         -> refunded
```

禁止 `refunded -> settled`。自动执行器领取任务和最终结算时都必须显式排除已退款任务。

未来的人工恢复工具可以把对象可用性从 `failed` 改为 `available`，但必须保留
`billing_status=refunded`，写入 `manually_recovered_at` 并记录“人工恢复且未重新计费”的审计
信息。该恢复能力不在本期 UI/API 范围内；本期只实现字段和状态保护，确保自动路径不会重扣。

### 8.4 典型组合

| 场景 | `status` | `output.availability` | `billing_status` |
| --- | --- | --- | --- |
| 等待执行 | `queued` | `pending` | `reserved` |
| 上游生成中 | `running` | `pending` | `reserved` |
| 已生成、归档中 | `succeeded` | `archiving` | `reserved` |
| 全部成功 | `succeeded` | `available` | `settled` |
| 上游最终失败 | `failed` | `failed` | `refunded` |
| 归档最终失败 | `succeeded` | `failed` | `refunded` |
| 对象过期 | `succeeded` | `expired` | `settled` |
| 已退款后人工恢复 | `succeeded` | `available` | `refunded` |

## 9. API 设计

### 9.1 提交任务

```http
POST /v1/images/generations/tasks
Authorization: Bearer sk-...
Content-Type: application/json
```

请求体沿用 `POST /v1/images/generations` 的 `dto.ImageRequest` 及额外字段兼容规则。后台持久化
可复用 BodyStorage 中的原始 JSON，而不是重新 marshal `ImageRequest`，以保留供应商扩展字段。

处理顺序：

1. `TokenAuth`、系统性能检查和模型请求限流。
2. 解析原始 JSON，执行现有图片请求校验，包括 `dto.MaxImageN`。
3. 若 `stream=true`，返回 OpenAI 风格 HTTP 400，错误码 `async_image_stream_unsupported`。
4. 执行 Token 模型权限、分组解析和渠道可用性检查。异步路径按同步路径
   `/v1/images/generations` 判断 Advanced Custom 渠道能力。
5. 执行现有敏感词检查、Token 估算、模型定价和计费表达式冻结。
6. 检查对象存储配置是否完整；不完整时在预扣费前返回 HTTP 503，错误码
   `object_storage_not_configured`。
7. 冻结归档配置并在同一数据库事务中创建任务、预扣资金来源和 Token 配额。
8. 提交事务后唤醒全局任务执行器并返回 HTTP 202。

成功响应：

```json
{
  "id": "task_xxx",
  "object": "image_generation.task",
  "created_at": 0,
  "status": "queued",
  "output": {
    "availability": "pending",
    "data": []
  }
}
```

若事务失败，不创建可见任务且不保留任何预扣费。任务调度唤醒失败不影响 202 响应，因为定时
调度器会再次发现已提交任务。

### 9.2 查询任务

```http
GET /v1/images/generations/tasks/{task_id}
Authorization: Bearer sk-...
```

查询使用 `TokenAuth`，但不使用 `Distribute()` 和模型请求限流。任务按
`task_id + 当前 user_id` 查询；同一用户的其他 Token 可以查询，其他用户统一得到 OpenAI 风格
HTTP 404，错误码 `image_generation_task_not_found`。

可用响应：

```json
{
  "id": "task_xxx",
  "object": "image_generation.task",
  "created_at": 0,
  "status": "succeeded",
  "output": {
    "availability": "available",
    "expires_at": 0,
    "data": [
      {
        "index": 0,
        "url": "https://presigned.example/...",
        "mime_type": "image/png",
        "size_bytes": 123,
        "revised_prompt": ""
      }
    ]
  }
}
```

等待、归档、失败和过期状态都返回 HTTP 200；只有请求本身错误、未找到任务或临时签名失败
使用非 200 状态。数据规则：

- `pending`、`archiving`、`failed`、`expired` 的 `data` 都是空数组。
- 生成失败时返回顶层 `error`，归档失败时返回 `output.error`，两者只包含稳定错误码和脱敏消息。
- `output.expires_at` 是全部对象中最早的到期时间。
- 查询时只要发现任一对象到期，立即把输出视为 `expired`，即使清理任务尚未删除 S3 对象。
- 不允许返回部分已签名结果。

动态签名失败时返回 OpenAI 风格 HTTP 503，错误码 `object_storage_temporarily_unavailable`，但不改变
任务和对象状态。每个预签名 URL 的有效期取以下两者较小值：

```text
当前 ObjectStoragePresignSeconds
对象 expires_at - 当前时间
```

数据库中永远不保存预签名 URL。

### 9.3 错误合同

| HTTP | 错误码 | 场景 |
| --- | --- | --- |
| 400 | `async_image_stream_unsupported` | 异步请求设置 `stream=true` |
| 400 | `invalid_request_error` | 请求解析或现有图片参数校验失败 |
| 403 | 现有鉴权/额度错误码 | Token 模型权限或预扣额度不足 |
| 404 | `image_generation_task_not_found` | 任务不存在或不属于当前用户 |
| 503 | `object_storage_not_configured` | 提交时 S3 配置不完整 |
| 503 | `object_storage_temporarily_unavailable` | 查询时预签名临时失败 |

异步后台错误写入任务状态，不在提交请求返回后另行改变 POST 的 HTTP 结果。

## 10. 图片执行复用

### 10.1 共享执行单元

把当前 `relay.ImageHelper` 拆成：

```go
type ImageExecutionResult struct {
    Usage      *dto.Usage
    Request    *dto.ImageRequest
    ImageCount uint
    LogContent []string
}

func ExecuteImage(c *gin.Context, info *relaycommon.RelayInfo) (*ImageExecutionResult, *types.NewAPIError)
```

`ExecuteImage` 负责当前的模型映射、适配器选择、请求转换、上游调用、错误映射和适配器
`DoResponse`，但不调用 `service.PostTextConsumeQuota`，也不自行退款。

同步 `ImageHelper` 调用 `ExecuteImage` 后继续使用现有响应 Writer，并立即执行原有同步结算和日志。
因此 `POST /v1/images/generations` 的响应格式、渠道行为和计费语义保持不变。

异步执行器构建带硬上限的内存捕获 Writer，调用同一个 `ExecuteImage`，解析适配器写出的标准
`dto.ImageResponse`。上限用已校验的请求 `n` 和 `MAX_FILE_DOWNLOAD_MB` 计算，使用受检查的
`int64` 算术，并计入 Base64 4/3 膨胀及固定 JSON 开销；超过上限立即终止捕获并按无效上游响应
失败。任何单张解码后图片仍必须小于等于 `MAX_FILE_DOWNLOAD_MB`。捕获内容只存在于当前进程
内存。

### 10.2 后台上下文重建

任务不保存渠道 Key。执行时根据任务保存的用户、Token ID、模型和具体分组重新查询当前数据库
状态并选择当前可用渠道。后台上下文的请求路径使用同步规范路径
`/v1/images/generations`，然后复用现有渠道重试、模型映射、参数覆盖、Header 覆盖、自动禁用和
错误处理。

Token 被删除或用户被禁用不会取消已经事务性预扣的任务；任务仍使用任务快照执行和结算。
但是后台只能从当前渠道配置读取上游凭据，所有渠道均不可用时任务按生成失败处理并退款。

## 11. 输出归档

### 11.1 标准化清单

完整 `ImageResponse` 验证成功后，任务立即转为：

```text
status = succeeded
output_availability = archiving
generation_completed_at = now
```

同时保存不含图片字节的 `archive_manifest`：

```json
[
  {
    "index": 0,
    "source_type": "url",
    "source_url": "https://temporary.example/image.png",
    "revised_prompt": "..."
  },
  {
    "index": 1,
    "source_type": "base64",
    "revised_prompt": "..."
  }
]
```

data URI 的 manifest 只记录 `source_type=data_uri`，不记录 URI。纯 Base64 和 data URI
统一属于不可跨进程恢复的 `ephemeral` 来源。URL 只保存上游已经返回的结果 URL，不保存请求或
渠道密钥。URL 可能本身带有临时签名，因此它只是恢复期间的瞬时数据；任务进入可用或最终失败
状态时必须从 manifest 中清除 `source_url`，永久记录只保留索引、来源类型和输出元数据。

完整 manifest 持久化后，后台不再需要重新发送生成请求，因此在同一状态更新中清空
`request_payload`。生成最终失败时也清空 `request_payload`。这样原始 Prompt 和供应商扩展字段
只保留到生成阶段结束，不会随永久任务记录长期保存。

空 `data`、同一条目同时缺少 URL/Base64、超出 `dto.MaxImageN` 或索引重复都属于无效上游响应，
生成状态记为 `failed` 并退款。

### 11.2 URL 来源

URL 使用现有 SSRF 保护下载能力，限制重定向、私网地址和下载大小。默认不附加调用方认证头。
如果某个适配器的结果 URL 需要渠道认证，可实现一个可选的归档来源解析接口，在运行时根据
当前 `ChannelMeta` 生成下载 Header；接口返回的 Header 只存在于内存，不写入 manifest。

下载使用响应内容类型作为辅助信息，但最终 MIME 和扩展名以实际内容识别为准。URL 可在任务
进程重启后从 manifest 重新下载；若 URL 已失效，则按归档重试策略处理。

### 11.3 Base64 与 data URI

- 纯 Base64 和 data URI 只在当前 Worker 内存中解码。
- data URI 的 MIME 声明只作为辅助信息，必须与识别出的实际内容兼容。
- 不把原始字符串、解码字节写入任务表、对象表、日志或临时文件。
- 同一 Worker 内的临时失败重试期间保留尚未成功索引所需的字符串或解码字节；索引上传成功、
  任务最终失败或租约丢失后立即释放对应引用。

### 11.4 内容类型和扩展名

归档器通过内容特征识别常见图片格式，并使用固定映射生成 MIME 和扩展名，例如：

| MIME | 扩展名 |
| --- | --- |
| `image/png` | `png` |
| `image/jpeg` | `jpg` |
| `image/webp` | `webp` |
| `image/gif` | `gif` |
| `image/avif` | `avif` |
| `image/bmp` | `bmp` |
| `image/tiff` | `tiff` |

无法识别、声明与内容冲突或不是允许图片类型的输出属于永久归档错误。SVG 不在默认允许列表中，
避免预签名 URL 被当成可执行文档打开。

### 11.5 Object Key

上传时间和 Object Key 的年月使用 UTC。归档器在发起 Put 前捕获该次上传的 UTC 归档时间，
Object Key 使用该时间的年月；只有 Put 成功后才把它固化为 `uploaded_at`。对象有效期从 Put
成功返回时开始计算，避免网络耗时缩短实际保留期。索引从 0 开始：

```text
prod/user-files/zmodel@async-images/42/2026/07/task_xxx/0.png
```

`user_id`、`task_id`、`index` 和扩展名都来自系统生成或固定映射，不接受调用方路径片段。

### 11.6 上传协议

每个对象按以下顺序处理：

1. 在数据库中 upsert 对应 `StorageObject` 为 `uploading`，写入 S3 定位快照和确定性 Key。
2. 使用私有 `PutObject` 上传，不设置公开 ACL。
3. 上传成功后，以 Put 成功返回时的当前 UTC 时间写入 `uploaded_at`，并计算
   `expires_at = uploaded_at + task.retention_seconds`。
4. 在同一对象行更新中写入 MIME、扩展名、大小、ETag 和 `status=available`。

若 Put 请求成功但客户端在收到响应前崩溃，对象行仍是 `uploading`。URL 来源可用相同 Key 覆盖
重试；ephemeral 来源在无法重建字节时进入最终失败清理，删除该潜在对象。

只有全部索引都为 `available` 才进入最终结算。任何部分成功都不会出现在查询响应中。

## 12. S3 客户端

新增可注入接口，业务代码不直接依赖具体 AWS 客户端：

```go
type ObjectStorage interface {
    PutObject(ctx context.Context, input PutObjectInput) (PutObjectResult, error)
    DeleteObject(ctx context.Context, input DeleteObjectInput) error
    PresignGetObject(ctx context.Context, input PresignGetObjectInput) (string, error)
}
```

生产实现使用 AWS SDK for Go v2：

- `github.com/aws/aws-sdk-go-v2/config`
- `github.com/aws/aws-sdk-go-v2/credentials`
- `github.com/aws/aws-sdk-go-v2/service/s3`

Endpoint 为空时使用 AWS 标准端点；自定义 Endpoint 使用 path-style 访问，兼容 MinIO 和常见
S3 兼容服务。对象始终是私有对象。预签名时设置响应 Content-Type 和安全文件名，但不改变对象
ACL。

客户端工厂接收一次不可变设置快照。Secret 只存在于设置内存和 AWS credential provider 中，
错误日志必须经过脱敏，不能打印请求签名、Access Key、Secret 或完整预签名 URL。

## 13. 后台执行与租约

### 13.1 系统任务类型

新增两个调度类型：

- `async_image_process`
- `async_image_storage_cleanup`

每个类型使用现有 `SystemTask`/`SystemTaskLock` 全局租约。系统不会为每个图片请求创建一个
`SystemTask`。提交任务只调用“确保存在一个活动处理任务并唤醒 Runner”；处理器完成一个批次后
结束，调度器在仍有工作时继续创建下一批。

处理器使用固定的小规模并发执行已领取任务，不增加新的后台并发设置。并发上限作为代码常量，
避免一次领取大量 Base64 响应造成无界内存占用。

URL-only 任务在临时归档失败后可以释放行租约并通过 `next_attempt_at` 由任意节点恢复。
包含 Base64/data URI 的任务在取得标准响应后必须由当前 Worker 持续持有行租约和内存输出，
在同一进程中执行退避等待与后续尝试；等待期间继续续租，但不占用数据库事务。固定并发上限
同时限定这类长驻内存任务的最大数量。

### 13.2 任务行 CAS 租约

即使已有全局租约，每个 `AsyncImageTask` 仍使用行级 CAS 租约：

1. 查询少量候选任务 ID。
2. 对每个候选执行条件更新：任务可运行、`next_attempt_at <= now`，并且租约为空或已过期。
3. 只有 `RowsAffected == 1` 的 Runner 获得任务。
4. 执行期间周期性延长 `lease_expires_at`；延长时必须匹配 `lease_owner`。
5. 失去租约后取消当前上下文，停止后续上传、结算和状态写入。

不使用 `SELECT ... SKIP LOCKED`，保证三个数据库都能工作。需要事务行锁的计费路径使用项目的
`lockForUpdate(tx)`。

### 13.3 可运行任务

以下任务可被领取：

- `queued + pending + reserved`：开始生成。
- `running + pending + reserved` 且租约过期：恢复生成；上游请求可能被重新发送。
- `succeeded + archiving + reserved` 且到达 `next_attempt_at`：继续归档或最终结算。
- 已到重试终点但退款事务尚未成功的任务：继续尝试退款，不因重试窗口结束而放弃资金恢复。

`settled`、`refunded`、`available`、`expired` 和终态 `failed` 默认不进入自动生成/结算路径。

## 14. 崩溃恢复

### 14.1 上游请求窗口

上游请求不是数据库事务，无法与任务状态原子提交。如果进程在上游已经生成图片、但尚未持久化
完整 manifest 前崩溃，恢复后可能重新发送一次生成请求。系统保证本地只结算一次，但无法替不
支持幂等键的上游消除该极小窗口中的重复生成成本。

### 14.2 URL 输出

完整响应和 URL manifest 持久化后，任何节点都可以继续下载尚未成功的索引。已经为
`available` 的索引直接跳过；其余索引使用确定性 Key 重试。

### 14.3 Base64/data URI 输出

恢复进程逐索引检查对象行：

- 如果对应对象已经是 `available`，该索引无需原始字节即可继续。
- 如果缺失或仅为 `uploading`，而 manifest 来源为 Base64/data URI，则无法安全重建。

存在不可重建索引时，任务直接进入最终归档失败流程：退款、把所有潜在对象标记为
`delete_pending`、尝试删除部分上传，并生成一条聚合管理员通知。不会重新调用上游来试图恢复
已经成功的生成结果。

当前 Worker 仍持有完整内存输出和行租约时，Base64/data URI 可以按普通退避策略重试。进程
退出或任务租约丢失后，当前 Worker 停止状态写入并释放内存输出；后续接管节点按上述不可重建
规则终结。绝不为了延续重试把图片字节写入数据库、磁盘或外部缓存。

### 14.4 最终状态写入窗口

若全部对象已上传但进程在最终结算前崩溃，恢复进程通过对象行发现完整结果，并继续结算。
若计费已在事务中结算但进程在响应状态更新后崩溃，任务行中的 `billing_status=settled` 和
`output_availability=available` 与额度变更属于同一事务，不会出现重复结算。

## 15. 重试策略

本功能的可配置重试只针对取得标准图片响应后的归档、对象存储和最终结算阶段。图片生成仍使用
现有渠道 `RetryTimes` 和 `shouldRetry` 规则。

- `archive_timeout_seconds`：一次任务归档尝试的上下文超时，默认 600，最大 1200。
- `archive_max_attempts`：业务归档尝试次数，默认 8。
- `archive_retry_window_seconds`：从提交时开始计算的最长窗口，默认 21600。
- 退避：从 15 秒开始指数增长，最多 15 分钟，不使用随机数，便于确定性测试。

URL-only 任务把下一次时间写入 `next_attempt_at` 后可释放租约。包含 ephemeral 来源的任务在
退避等待期间保留租约和内存输出，并逐次增加 `archive_attempts`；租约续期失败会取消等待并
停止当前 Worker，后续接管节点进入不可重建的最终失败流程。

永久错误立即进入最终失败，例如 Base64 无法解码、图片超过限制、MIME 不受支持、响应结构无效
或崩溃后 ephemeral 来源不可重建。

网络超时、URL 临时失败、S3 错误、预期可由管理员修复的认证错误和最终结算数据库错误按临时
错误重试。满足以下任一条件后停止普通归档重试：

```text
archive_attempts >= archive_max_attempts
当前时间 >= archive_retry_deadline_at
```

停止后必须先完成幂等退款事务，才能把输出标记为最终 `failed`。如果退款事务本身失败，任务
继续由后台领取并重试退款，不受归档尝试数和窗口限制。

## 16. 计费一致性

### 16.1 原则

现有 `BillingSession.Refund` 使用进程内标志并异步退款，`RefundTaskQuota` 也不能为本功能提供
崩溃后的 exactly-once 保证。因此异步图片任务使用任务行作为持久化幂等账本，不直接复用这两个
退款入口。

所有额度运算继续遵守项目计费安全规则：请求数量先由 `dto.MaxImageN` 限制，配额转换使用
`common.Quota*Checked` 或现有严格辅助函数，并把饱和信息写入计费快照和日志。

### 16.2 提交和预扣

异步任务设置 `ForcePreConsume=true`，禁用信任额度旁路。新增接收 `*gorm.DB` 的内部计费方法，
在单个主数据库事务中：

1. 创建 `AsyncImageTask`。
2. 使用 `lockForUpdate(tx)` 锁定所需用户、Token 和订阅行。
3. 预扣钱包或订阅额度。
4. 对非无限 Token 预扣 Token 配额。
5. 把实际预扣值和 `billing_status=reserved` 写入任务。

任何一步失败都回滚整个事务。事务提交后统一失效或更新相关缓存；缓存失败不能回滚数据库事实，
必须记录告警并以数据库为准。

### 16.3 最终结算

所有对象上传成功后，使用冻结的计费上下文和实际成功图片数计算最终额度。响应数量不得超过
请求允许数量和 `dto.MaxImageN`；违反约束的上游响应不进入正向补扣，而按无效响应失败退款。

结算事务：

1. `lockForUpdate(tx)` 锁定任务。
2. 只有 `billing_status=reserved` 可以结算。
3. 再次确认所有对象可用且未过期。
4. 按 `actual_quota - reserved_quota` 调整钱包/订阅和 Token 配额。
5. 写入 `actual_quota`、`billing_status=settled`、`billing_finalized_at`。
6. 同一事务写入 `output_availability=available`、最早 `output_expires_at`，并从 manifest
   清除所有 `source_url`。

如果任务已经 `refunded`，结算函数返回稳定的“禁止自动重新计费”结果，不做任何资金调整。

### 16.4 幂等退款

最终失败事务：

1. `lockForUpdate(tx)` 锁定任务。
2. `settled` 任务禁止自动退款；`refunded` 任务直接幂等返回。
3. 只按任务保存的 `reserved_quota` 和 `token_reserved_quota` 退回对应资金来源。
4. 写入 `billing_status=refunded`、`billing_finalized_at` 和失败状态；仅当失败阶段为归档时把
   聚合通知状态写为 `pending`，上游生成失败保持 `none`。
5. 清空残留的 `request_payload` 和 manifest 中的所有 `source_url`，避免终态任务永久保存
   原始 Prompt、扩展字段或临时上游签名 URL。

资金变更和状态变更在同一数据库事务中，因此进程重启后不会重复退款。

### 16.5 日志和统计

成功结算继续生成图片消费日志并更新用户/渠道统计；退款生成任务退款日志。主额度事务不依赖日志
数据库成功。若日志数据库独立且写入失败，记录带 `task_id` 的系统告警，不回滚已经完成的资金
事务，也不通过再次资金调整来补日志。

## 17. 最终失败和通知

生成失败或归档最终失败都执行同一个幂等退款终结器，但保留不同的 `status`。只有归档最终失败
生成管理员通知；上游生成失败属于正常请求失败路径，不发送对象存储告警：

1. 在事务中完成幂等退款和任务失败状态。
2. 把所有 `uploading/available` 对象标记为 `delete_pending`。
3. 立即尝试删除，并由清理任务继续处理删除失败的对象。
4. 如果失败阶段是归档，在退款事务中生成一个任务级聚合通知意图；内容包含任务 ID、用户 ID、
   模型、已尝试次数、失败阶段、对象总数、已上传数和最后错误摘要，不包含 Prompt、Base64、
   Secret 或预签名 URL。生成失败保持 `admin_notification_state=none`。

通知调度使用任务的 `admin_notification_state` CAS 领取，只调用一次
`service.NotifyRootUser`。通知类型使用新的稳定类型，例如 `async_image_archive_failed`，从而继续受
现有通知限流保护，并自动使用 Root 用户选择的 Email、Webhook、Bark 或 Gotify。

外部通知调用与数据库不能组成分布式事务。实现采用带领取超时的单通知意图，避免同一失败中的
每张图片和每次重试各发一条通知；极端崩溃窗口仍按现有通知系统的 best-effort 语义处理。

## 18. 对象到期和清理

### 18.1 逻辑到期

对象的有效期在上传成功时固化：

```text
expires_at = uploaded_at + task.retention_seconds
```

任务 `output_expires_at` 取所有对象中最早值。GET 查询以数据库时间戳判断，不依赖 S3 Lifecycle
是否已经运行。到期后立即停止签名，并将任务输出可用性更新为 `expired`。

### 18.2 物理删除

`async_image_storage_cleanup` 按设置的清理间隔运行，分批处理：

- `status=available AND expires_at <= now`
- `status=delete_pending`

S3 返回成功或对象不存在都视为删除成功，写入 `status=deleted` 和 `deleted_at`。删除失败时增加
`delete_attempts` 并保存脱敏错误，等待下一轮。数据库对象记录和任务记录永久保留。

清理任务每批使用短事务领取对象，不在数据库事务内执行网络请求。对象状态更新使用条件更新，
避免与正在上传的 Worker 互相覆盖。

## 19. 对象存储设置

### 19.1 Option Key 和默认值

新增设置快照，Option Key 使用稳定前缀：

| Option Key | 默认值 | 校验 |
| --- | --- | --- |
| `ObjectStorageS3Endpoint` | 空 | 空表示 AWS；非空必须是 HTTP(S) URL |
| `ObjectStorageS3Region` | 空 | 配置 Bucket 时必填 |
| `ObjectStorageS3Bucket` | 空 | 配置存储时必填，不允许空白字符 |
| `ObjectStorageS3AccessKey` | 空 | 配置存储时必填 |
| `ObjectStorageS3SecretAccessKey` | 空 | 配置存储时必填；读取永不返回 |
| `ObjectStorageRetentionSeconds` | `86400` | 60 到 31536000 |
| `ObjectStoragePresignSeconds` | `600` | 60 到 604800，实际签名受对象剩余寿命限制 |
| `ObjectStorageArchiveTimeoutSeconds` | `600` | 1 到 1200 |
| `ObjectStorageArchiveMaxAttempts` | `8` | 1 到 100 |
| `ObjectStorageArchiveRetryWindowSeconds` | `21600` | 60 到 604800，且不小于单次超时 |
| `ObjectStorageCleanupIntervalSeconds` | `900` | 60 到 86400 |

不新增 enable 字段。异步提交的可用条件是 Region、Bucket、Access Key 和 Secret 完整；Endpoint
可以为空。配置不完整只让异步 POST 返回 503，不影响同步图片生成和其他接口。

### 19.2 Root API

新增 Root-only API：

```text
GET /api/option/object-storage
PUT /api/option/object-storage
```

GET 返回：

```json
{
  "success": true,
  "data": {
    "endpoint": "",
    "region": "",
    "bucket": "",
    "access_key": "",
    "secret_configured": false,
    "retention_seconds": 86400,
    "presign_seconds": 600,
    "archive_timeout_seconds": 600,
    "archive_max_attempts": 8,
    "archive_retry_window_seconds": 21600,
    "cleanup_interval_seconds": 900
  }
}
```

GET 不包含 Secret 字段。通用 `GET /api/option` 继续通过敏感 Key 过滤规则隐藏 Secret。

PUT 接收完整非敏感配置和可选 `secret_access_key`：

- 空 Secret 保留数据库和内存中的旧值。
- 非空 Secret 替换旧值。
- 若当前没有 Secret，而其他存储字段表示要启用配置，则空 Secret 校验失败。
- 全部字段先在内存中组成候选快照并整体校验，再通过 `model.UpdateOptionsBulk` 一次持久化。
- 任一字段失败时不部分更新。

Endpoint、Region 或 Bucket 属于对象物理位置。只要数据库中仍有 `uploading`、`available` 或
`delete_pending` 对象，PUT 就拒绝改变这三个字段，避免旧对象因定位或凭据切换而无法签名和
清理。待所有旧对象都成为 `deleted` 后才允许迁移到新的物理位置。Access Key 和 Secret 可以
在原位置轮换，调用方负责保证新凭据仍能访问同一 Bucket。

连接参数更新只影响尚未创建 `StorageObject` 的对象和后续预签名/删除所使用的凭据。已创建对象
继续使用自身保存的 Endpoint、Region 和 Bucket 定位快照。

## 20. 后台页面

在默认前端新增独立路由 `/system-settings/object-storage`，并在系统管理侧边栏新增直接入口
“Object Storage”，不藏在 Worker 或其他设置表单中。

页面使用现有系统设置布局、React Query、React Hook Form、Zod 和 UI 组件，包含：

- Endpoint URL
- Region
- Bucket
- Access Key
- Secret Access Key 密码输入
- Object retention
- Presigned URL lifetime
- Archive attempt timeout
- Maximum archive attempts
- Maximum retry window
- Cleanup interval

时长在界面中使用明确单位的数字输入，提交时转换为 API 秒值。Secret 输入初始永远为空，并在
字段说明中单独显示“已配置/未配置”状态；空输入表示保留。页面不显示启用开关和 Key Prefix。

所有用户可见文本使用 `useTranslation()` 和英文语义键，并同步六个 locale：`en`、`zh`、
`fr`、`ru`、`ja`、`vi`。页面保持现有紧凑后台视觉，不增加营销式 Hero 或装饰性卡片。

## 21. 安全和隐私

- S3 对象为私有对象，只通过短期预签名 GET 访问。
- 预签名 URL 不写数据库、不写业务日志。
- S3 Secret、渠道 Key、Token Key 和运行时下载认证 Header 不持久化到任务或对象表。
- Base64/data URI 输出不写数据库、磁盘或日志。
- 终态任务清空 `request_payload`；永久记录只保留模型、计费快照、输出元数据和脱敏错误。
- 上游 `source_url` 只在 URL 归档恢复期间持久化，任务可用或最终失败后立即清除。
- URL 下载使用 SSRF 保护和重定向校验。
- 上传和下载都执行 `MAX_FILE_DOWNLOAD_MB` 单对象大小限制。
- MIME 使用实际内容识别，不信任文件名或 URL 后缀。
- 错误信息不得包含完整 S3 请求、Authorization、签名查询参数或图片内容。
- 查询按当前 `user_id` 隔离，不通过不同错误暴露其他用户是否存在该 task ID。

## 22. 测试策略

实现严格使用 RED -> GREEN TDD。测试保护公开合同和跨模块不变量，不测试私有常量或仅追求覆盖率。

### 22.1 模型和数据库

- SQLite 集成测试验证任务、对象、唯一约束和迁移。
- CAS 领取测试验证同一任务只能被一个 Worker 获得。
- 过期租约测试验证另一 Runner 可恢复。
- 钱包、订阅和 Token 预扣事务测试验证任一步失败会整体回滚。
- 重复结算和重复退款测试验证额度只变化一次。
- `refunded -> settled` 测试验证自动重新扣费被拒绝。
- SQL 实现只用 GORM 和 `lockForUpdate(tx)`，并通过现有 MySQL/PostgreSQL CI 验证迁移与行为。

### 22.2 设置和 S3

- GET 设置永不返回 Secret。
- PUT 空 Secret 保留原值，非空 Secret 替换，非法组合不部分更新。
- 存在未删除对象时拒绝修改 Endpoint、Region 或 Bucket，但允许原位置凭据轮换。
- 默认值和全部数值边界测试。
- 自定义 Endpoint 使用 path-style，AWS 空 Endpoint 使用标准解析。
- Fake ObjectStorage 验证私有 Put、确定性 Key、预签名剩余寿命上限和删除 404 幂等。

### 22.3 归档和恢复

- URL、纯 Base64 和 data URI 各自成功归档。
- MIME/扩展名由实际内容决定。
- 单对象超过 `MAX_FILE_DOWNLOAD_MB` 失败。
- 多图部分上传不向 GET 暴露。
- URL 任务在进程恢复后继续归档。
- Base64/data URI 在缺少未上传字节时退款并清理部分对象。
- Put 成功但对象行仍为 `uploading` 时使用相同 Key 恢复或删除。
- 归档超时、最大尝试次数和最长窗口使用可控时钟测试。
- 上游生成失败退款但不通知；归档最终失败只产生一个聚合管理员通知意图。
- 到期立即停止签名，清理成功后数据库行仍存在。

### 22.4 API 和图片中继

- 同步图片接口响应和计费回归测试保持不变。
- 异步 POST 返回 202，且 `stream=true` 返回 400。
- S3 未配置时在预扣费前返回 503。
- 原始扩展字段在后台执行时仍存在。
- GET 所有状态的 JSON 合同、用户隔离、404 和签名 503。
- 图片执行共享单元不自行结算；同步入口结算一次，异步入口只在归档后结算一次。

### 22.5 前端

- Zod Schema 验证 URL、时长范围、最大 20 分钟和重试窗口关系。
- Secret 已配置时表单仍加载空值，空提交不清除 Secret。
- React Query 保存成功后刷新设置快照。
- 路由和侧边栏入口可访问。
- 运行 `bun run typecheck`、涉及文件 lint、i18n 同步/检查和生产构建。

## 23. 验收标准

1. 原有同步图片生成测试和行为不回归。
2. 异步任务对 URL、Base64、data URI 输出都能生成私有 S3 对象。
3. GET 只在全部对象有效时返回预签名 URL，任何时候都不返回部分结果。
4. 对象 Key、UTC 年月、索引和实际扩展名完全符合固定规则。
5. 对象到期后立即停止签名，清理后数据库记录仍保留。
6. 重复 Worker、进程崩溃和重复终结不会重复结算或退款。
7. Base64/data URI 不出现在数据库、文件系统或日志中。
8. Secret 不从通用或专用读取设置 API 返回，空 Secret 更新保留旧值。
9. 对象存储后台页在六种语言下可用，并通过类型检查、lint 和生产构建。
10. SQLite 本地测试通过，代码不包含破坏 MySQL/PostgreSQL 兼容性的 SQL 或迁移。

## 24. 实施顺序

1. 设置快照、专用 Root API 和 S3 可注入客户端。
2. `AsyncImageTask`、`StorageObject`、迁移、状态转换和 CAS 租约。
3. 事务性异步图片计费预扣、结算和退款。
4. 图片执行与同步结算拆分，保持同步回归测试。
5. 异步 POST/GET 合同。
6. 后台处理、URL/Base64/data URI 归档、崩溃恢复和聚合通知。
7. 到期签名判断和清理系统任务。
8. 默认前端对象存储页面和六语言翻译。
9. 聚焦测试、全量后端测试、前端验证和最终差异审查。
