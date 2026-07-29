# 异步图片生成与 S3 临时存储设计

## 状态

- 日期：2026-07-29
- 项目：zmodel
- 范围：OpenAI 兼容图片生成、异步任务、私有 S3 归档、对象生命周期、计费一致性、后台对象存储设置和任务管理
- 起点提交：`c0ffb2aca7b467db6b1f0688464033ee1216e244`
- 设计状态：已按用户第二轮复核意见确认，可进入实施
- 最新确认：S3 Secret 明文落现有 Option 表以支持动态配置；重启采用计划停机排空与持久化恢复，接受硬崩溃下无法彻底消除的极小重复上游请求窗口

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
15. 完整、有效的标准图片响应生成成功后立即结算；后续 S3 归档失败不自动退款，避免上游已扣费而平台承担损失。
16. 上游生成最终失败或标准图片响应无效时只退款一次；已结算任务不能由自动归档流程退款或重复扣费。
17. 所有图片字节在发起 S3 `PutObject` 前写入共享持久暂存目录；因此每个真正进入 S3 上传阶段的
    失败任务都保留可供人工重传的源文件。
18. 归档最终失败时向 Root 用户发送一条聚合管理员通知，并在独立的异步图片任务管理页支持批量重试。
19. 管理页提供“重新上传全部失败的图片文件”按钮，也支持勾选后批量重试；重试只读取持久暂存文件，不调用上游、不退款、不再次扣费。
20. 通知复用 `service.NotifyRootUser`，沿用 Root 用户的 Email、Webhook、Bark 或 Gotify 设置。
21. S3 Secret 按已确认方案以明文写入现有 Option 表；任何读取 API、页面和日志都不得回显。更新时空 Secret 保留原值，非空值替换原值。数据库读取者和数据库备份可接触该明文，这是选择明文存储后的明确安全边界。
22. 计划内服务重启必须停止领取新任务并在 `SHUTDOWN_TIMEOUT_SECONDS` 内等待活跃 Worker 排空；已发出的上游生成请求不得主动取消，取得响应后优先完成持久暂存。已提交暂存清单的 S3 归档允许中断，并通过数据库租约、确定性 Object Key、共享持久暂存文件和 `HeadObject` 恢复。
23. 机器宕机、`kill -9` 或排空超时恰好发生在“上游已生成成功、完整暂存清单尚未提交”窗口时，恢复后可能再次调用上游。系统仍保证用户只结算一次，但无法保证平台只被不支持幂等键的上游扣费一次；该残余风险已确认接受，首期不为此引入外部消息队列或分布式事务。

## 3. 目标

1. 复用现有图片适配器和渠道重试能力，不为每个图片渠道复制异步实现。
2. 在多节点、进程崩溃和任务重复执行情况下，避免重复结算或重复退款。
3. 在任何时刻都不向调用方暴露部分归档结果。
4. 不把图片字节写入数据库；把 URL、Base64 和 data URI 统一转换为受控的共享持久暂存文件，保证服务重启后仍可重新上传。
5. 不把渠道密钥写入任务、对象、暂存文件或日志；S3 Secret 只按用户确认的方案明文存入 Option 表，并严格禁止读取接口和日志回显。
6. 让所有已完成持久暂存的输出都能在进程或节点重启后继续归档。
7. 保持 SQLite、MySQL 5.7.8+ 和 PostgreSQL 9.6+ 兼容。
8. 让对象存储配置可在 Root 后台维护，并让新配置只在明确边界影响任务。
9. 让管理员可以查询全部归档失败任务并批量重新上传。

## 4. 非目标

- 不把同步图片接口改造成异步接口。
- 不支持异步图片编辑、variation 或 multipart 图片上传；本期只覆盖
  `POST /v1/images/generations/tasks`。
- 不增加公开对象、公共 Bucket 或长期不失效 URL。
- 不增加 S3 Key Prefix 设置，也不允许调用方自定义 Object Key。
- 不提供取消已经生成成功任务的能力。
- 不在归档重试按钮中调用上游生成；人工操作只使用已经持久暂存的图片文件重新上传，避免产生新的上游费用。
- 不把持久暂存目录当作用户可访问的长期对象存储；它只承担 S3 上传前后的可靠交接，并按明确生命周期清理。
- 不把现有视频、音乐等供应商轮询任务迁移到新表。
- 不承诺跨所有图片上游的严格 exactly-once 请求；上游未提供可靠幂等键时，硬崩溃窗口只能通过计划停机排空缩小，不能由本地数据库事务彻底消除。
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

### 5.4 历史兼容边界

本功能采用严格增量设计：

- 保留原有 `POST /v1/images/generations` 路由、请求/响应合同、同步计费、渠道重试、日志和错误
  行为；只有内部单渠道执行代码被抽取，且发布前必须通过同步回归测试。
- 新异步能力只挂载到新路由、新 `AsyncImageTask`/`StorageObject` 表和新 Option Key。未调用异步
  POST 时，不创建异步任务、不预扣异步额度、不访问 S3，也不启动与请求相关的归档流程。
- 不修改或迁移现有 `model.Task` 数据、`/api/task` 接口、视频/音乐轮询状态和现有 Task Logs 页面。
- 数据库迁移只新增表和索引，不重写历史业务表记录；对象清理任务只扫描固定
  `business_id=zmodel@async-images` 的新对象行。
- 对象存储配置不改变现有上传、文件代理或其他业务的存储位置；设置不完整只让新异步图片 POST
  返回 503。
- 默认前端新增页面和导航项；classic 前端既有页面和路由不改动。

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
  -> 原子写入共享持久暂存目录并校验大小、MIME、SHA-256
  -> 同库事务：持久化完整暂存清单 + 生成成功最终结算 + status succeeded
  -> 极端暂存基础设施故障：仍按有效标准响应结算，output failed 并触发数据完整性告警
  -> 私有 S3 上传及 StorageObject 状态更新
  -> 全部对象成功后标记 output available
  -> 清除原始请求正文；确认任务完整可用后清理暂存文件

GET /v1/images/generations/tasks/{task_id}
  -> TokenAuth
  -> 只查询当前用户的任务
  -> 根据任务和 StorageObject 判断可用性
  -> 对每个未过期对象动态生成预签名 GET URL

async-image-storage-cleanup
  -> 全局 SystemTask 租约
  -> 删除到期或 delete_pending 的 S3 对象
  -> 永久保留任务和对象数据库记录

Root 异步图片任务管理页
  -> 查询生成状态、输出可用性、计费状态和对象明细
  -> 勾选失败任务批量重试，或一键重新上传全部失败图片
  -> 从共享持久暂存读取源文件，不重新调用上游生成，不重复扣费
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
| `billing_status` | `varchar(32)` 索引 | `reserved/settled/refunded`；生成成功即 `settled` |
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
| `archive_manifest` | `text` | 输出索引、来源类型、暂存文件相对路径、大小、MIME、SHA-256 和 revised prompt；不含图片字节或来源 URL |
| `retention_seconds` | `int64` | 提交时冻结的对象有效期 |
| `archive_timeout_seconds` | `int64` | 提交时冻结，范围不超过 1200 秒 |
| `archive_max_attempts` | `int` | 提交时冻结 |
| `archive_retry_deadline_at` | `int64` 索引 | 提交时间加最长重试窗口 |
| `archive_attempts` | `int` | 已开始的业务归档尝试次数 |
| `next_attempt_at` | `int64` 索引 | 下次可领取时间 |
| `output_expires_at` | `int64` 索引 | 全部输出中最早的 `expires_at` |
| `lease_owner` | `varchar(128)` 索引 | 当前任务行租约持有者 |
| `lease_expires_at` | `int64` 索引 | 行租约到期时间 |
| `source_kind` | `varchar(32)` | `none/url/base64/data_uri/mixed`，仅用于来源审计，不单独决定重试资格 |
| `public_error_code` | `varchar(64)` | 对调用方稳定的错误码 |
| `public_error_message` | `text` | 已脱敏的错误说明 |
| `last_error` | `text` | 管理和日志使用的内部错误摘要，不包含密钥或 Base64 |
| `generation_completed_at` | `int64` | 已取得完整标准图片响应的时间 |
| `billing_finalized_at` | `int64` | 结算或退款完成时间 |
| `manually_recovered_at` | `int64` | 管理员重试成功后的审计时间；自动路径保持为 0 |
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
| `status` | `varchar(32)` 索引 | `uploading/available/failed/delete_pending/deleted` |
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
| `staging_relative_path` | `varchar(768)` | 相对 `ASYNC_IMAGE_STAGING_DIR` 的系统生成路径，不对 API 返回 |
| `staging_status` | `varchar(32)` 索引 | `pending/available/failed/delete_pending/deleted` |
| `staging_size_bytes` | `int64` | 暂存文件大小，必须与 `size_bytes` 一致 |
| `staging_sha256` | `varchar(64)` | 暂存内容 SHA-256，用于重启和人工重传前校验 |
| `staged_at` | `int64` | 原子写入并完成校验的时间 |
| `staging_deleted_at` | `int64` | 暂存文件删除确认时间 |
| `created_at` | `int64` 索引 | 创建时间 |
| `updated_at` | `int64` 索引 | 更新时间 |

唯一约束为 `(business_id, resource_id, object_index)`。重试对同一索引使用相同 Key 并覆盖，
不会创建第二条对象记录。单索引上传失败写为 `failed`，并保留 S3 定位、暂存相对路径、大小、
SHA-256 和脱敏错误，供自动或管理员批量重试。只有对象到期或明确执行删除流程时才进入
`delete_pending`；上传失败不得删除暂存源文件。

S3 Endpoint、Region 和 Bucket 在首次准备上传该索引时写入对象记录。后续重试继续使用该定位
快照，避免配置切换后同一任务的图片散落到不同 Bucket。Access Key 和 Secret 始终从当前 Option
设置读取，绝不写入任务表、对象表或暂存文件。

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
failed -- Root retry -----------> archiving
```

- `pending`：尚未取得可归档输出。
- `archiving`：已取得生成结果并完成计费结算，正在上传、重试或完成输出状态更新。
- `available`：全部对象上传成功、未过期且计费已结算。
- `expired`：至少一个对象已到期；查询不返回任何部分结果。
- `failed`：生成失败或归档最终失败；查询不返回任何部分结果。

自动执行路径只能在全部对象均为 `available`、全部未过期且 `billing_status=settled` 时进入
`output.availability=available`。管理员触发的归档重试成功后写入 `manually_recovered_at`，但
`billing_status` 仍保持原来的 `settled`，不会再次扣费。

### 8.3 计费状态 `billing_status`

```text
reserved -> settled
         -> refunded
```

完整、有效的标准图片响应一旦验证完成，就必须从 `reserved` 进入 `settled`：正常路径与完整暂存
清单在同一事务中提交，来源获取或暂存基础设施事故使用专门结算事务记录。只有上游生成最终失败
或标准响应无效时允许 `reserved -> refunded`。禁止 `settled -> refunded` 和
`refunded -> settled`；归档执行器和管理员重试接口都不得改变计费状态。

管理员归档重试可以把对象可用性从 `failed` 改回 `archiving`，成功后进入 `available`，但必须
保留 `billing_status=settled`，写入 `manually_recovered_at` 并记录“归档恢复且未重复计费”的
审计信息。所有 S3 上传失败都使用对应持久暂存文件，不通过该接口重新调用上游生成。

### 8.4 典型组合

| 场景 | `status` | `output.availability` | `billing_status` |
| --- | --- | --- | --- |
| 等待执行 | `queued` | `pending` | `reserved` |
| 上游生成中 | `running` | `pending` | `reserved` |
| 已生成、归档中 | `succeeded` | `archiving` | `settled` |
| 全部成功 | `succeeded` | `available` | `settled` |
| 上游最终失败 | `failed` | `failed` | `refunded` |
| 归档最终失败 | `succeeded` | `failed` | `settled` |
| 对象过期 | `succeeded` | `expired` | `settled` |
| 管理员归档恢复 | `succeeded` | `available` | `settled` |

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
6. 检查对象存储配置是否完整，并检查 `ASYNC_IMAGE_STAGING_DIR` 所在共享持久存储可创建、写入、
   `fsync`、原子重命名和读取测试文件；失败时在预扣费前返回 HTTP 503，错误码
   `object_storage_not_configured` 或 `archive_staging_unavailable`。
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
- 只有任务当前为 `output_availability=available` 时，查询发现任一对象到期才把整组输出原子更新为
  `expired`，即使清理任务尚未删除 S3 对象。归档失败任务中的部分对象到期仍保持 `failed`，以便
  Root 从暂存文件重新上传整组结果。
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
| 503 | `archive_staging_unavailable` | 提交时持久暂存目录不可用或不满足原子写入要求 |
| 503 | `object_storage_temporarily_unavailable` | 查询时预签名临时失败 |

异步后台错误写入任务状态，不在提交请求返回后另行改变 POST 的 HTTP 结果。

## 10. 图片执行复用

### 10.1 共享单渠道执行单元

把当前 `relay.ImageHelper` 拆成：

```go
type ImageExecutionResult struct {
    Usage      *dto.Usage
    Request    *dto.ImageRequest
    ImageCount uint
    LogContent []string
}

func ExecuteImageAttempt(c *gin.Context, info *relaycommon.RelayInfo) (*ImageExecutionResult, *types.NewAPIError)
```

`ExecuteImageAttempt` 负责一次已选定渠道上的模型映射、适配器选择、请求转换、上游调用、错误映射和适配器
`DoResponse`，但不调用 `service.PostTextConsumeQuota`，也不自行退款。

同步 `ImageHelper` 调用 `ExecuteImageAttempt` 后继续使用现有响应 Writer，并立即执行原有同步结算和日志。
因此 `POST /v1/images/generations` 的响应格式、渠道行为和计费语义保持不变。

异步执行器构建带硬上限的内存捕获 Writer，调用同一个 `ExecuteImageAttempt`，解析适配器写出的标准
`dto.ImageResponse`。上限用已校验的请求 `n` 和 `MAX_FILE_DOWNLOAD_MB` 计算，使用受检查的
`int64` 算术，并计入 Base64 4/3 膨胀及固定 JSON 开销；超过上限立即终止捕获并按无效上游响应
失败。任何单张解码后图片仍必须小于等于 `MAX_FILE_DOWNLOAD_MB`。捕获内容只存在于当前进程
内存，直到全部图片已经写入持久暂存并校验完成；之后立即释放响应正文和解码缓冲区。

### 10.2 渠道重试复用边界

复用适配器和渠道能力是本项目的最佳实践，但复用边界必须是 Go 内部执行单元和稳定的渠道选择/
重试原语，而不是让后台 Worker 递归请求本机 HTTP 接口，也不是直接调用包含响应写出、退款和日志
副作用的完整 `controller.Relay`。

- 保留同步 `controller.Relay` 现有重试循环、HTTP 返回、同步退款和日志时序，只把单渠道图片调用
  替换为 `ExecuteImageAttempt`。
- 把异步路径需要的渠道选择、`shouldRetry` 判断、模型映射、自动禁用和错误归一化能力提取或暴露
  为无 HTTP 响应副作用的共享原语；同步和异步入口调用同一实现。
- 异步 Worker 自己维护持久化任务状态和计费账本，在一次后台执行中按现有 `RetryTimes` 顺序调用
  共享原语和 `ExecuteImageAttempt`。它不复制任何供应商请求/响应转换代码。
- 新增图片渠道时仍只实现现有图片 adaptor；无需再实现一套异步 adaptor。渠道的参数覆盖、Header
  覆盖、错误映射、自动禁用和重试判定由共享路径自然生效。

这种拆分既消除每渠道重复实现，也避免把 HTTP 控制器、同步计费或响应 Writer 生命周期强行带入
后台任务。同步回归测试和共享执行合同测试是该重构的发布门槛。

### 10.3 后台上下文重建

任务不保存渠道 Key。执行时根据任务保存的用户、Token ID、模型和具体分组重新查询当前数据库
状态并选择当前可用渠道。后台只在内存构造的 Gin/Relay 上下文中把规范请求路径字段设为
`/v1/images/generations`，供现有路径判断逻辑读取；不会向本机发起 HTTP 请求。随后复用现有渠道
重试、模型映射、参数覆盖、Header 覆盖、自动禁用和错误处理。

Token 被删除或用户被禁用不会取消已经事务性预扣的任务；任务仍使用任务快照执行和结算。
但是后台只能从当前渠道配置读取上游凭据，所有渠道均不可用时任务按生成失败处理并退款。

## 11. 输出归档

### 11.1 标准化清单

完整 `ImageResponse` 验证成功后，任务的生成和计费结果确定为：

```text
status = succeeded
output_availability = archiving
billing_status = settled
generation_completed_at = now
```

正常路径先把全部图片可靠交接到持久暂存，再在同一个主数据库事务中提交完整暂存清单、状态转换
和最终额度调整。`status=succeeded` 的业务定义仍是“上游生成服务已经成功交付完整、有效的图片
结果，平台已经产生并确认生成成本”；因此即使 S3 或极端暂存基础设施随后失败也不能退款。S3 是
后续交付和托管能力，其失败只改变 `output_availability`，不回滚已经完成的生成计费。

同时保存不含图片字节和来源 URL 的 `archive_manifest`：

```json
[
  {
    "index": 0,
    "source_type": "url",
    "staging_relative_path": "42/2026/07/task_xxx/0.img",
    "size_bytes": 123,
    "mime_type": "image/png",
    "sha256": "...",
    "revised_prompt": "..."
  },
  {
    "index": 1,
    "source_type": "base64",
    "staging_relative_path": "42/2026/07/task_xxx/1.img",
    "size_bytes": 456,
    "mime_type": "image/webp",
    "sha256": "...",
    "revised_prompt": "..."
  }
]
```

URL、纯 Base64 和 data URI 只作为生成响应中的输入形式。URL 内容下载完成、Base64/data URI
解码完成并写入持久暂存后，不再持久化原始 URL、原始字符串或下载认证 Header。manifest 只保留
来源类型和已暂存内容的不可变元数据，因此后续自动重试、跨节点恢复和人工重传都不依赖上游 URL
继续有效，也不需要再次调用上游。

完整 manifest 和对应对象的暂存元数据持久化后，后台不再需要重新发送生成请求，因此在同一状态更新中清空
`request_payload`。生成最终失败时也清空 `request_payload`。这样原始 Prompt 和供应商扩展字段
只保留到生成阶段结束，不会随永久任务记录长期保存。

空 `data`、同一条目同时缺少 URL/Base64、超出 `dto.MaxImageN` 或索引重复都属于无效上游响应，
生成状态记为 `failed` 并退款。只有完整标准响应的条目和来源结构验证通过后才允许生成成功结算；
暂存或 S3 基础设施错误按本节定义的归档失败处理，不把有效上游响应改判为生成失败。

### 11.2 URL 来源

URL 使用现有 SSRF 保护下载能力，限制重定向、私网地址和下载大小。默认不附加调用方认证头。
如果某个适配器的结果 URL 需要渠道认证，可实现一个可选的归档来源解析接口，在运行时根据
当前 `ChannelMeta` 生成下载 Header；接口返回的 Header 只存在于内存，不写入 manifest。

下载使用响应内容类型作为辅助信息，但最终 MIME 和扩展名以实际内容识别为准。下载完成后必须
先写入持久暂存，之后才允许任务结算和清除原始响应；归档重试不再重新访问该 URL。

取得结构完整的 URL 响应后，如果 URL 下载持续失败、过期或被 SSRF 规则拒绝，当前 Worker 可在
内存中仍持有原始响应时重试下载，但不能把未取得的图片字节伪装成普通 S3 上传失败。重试终止时按
已经发生的上游生成成本执行专门结算事务，写入 `status=succeeded`、`billing_status=settled`、
`output_availability=failed` 和 `archive_source_fetch_failed`，并触发高优先级 Root 告警。由于设计
明确不持久化上游 URL，该类任务必须先由运维恢复来源文件并补齐持久暂存，之后才能进入人工上传；
它不属于“所有实际 S3 上传失败均可直接重传”的承诺范围，也不得退款或重新调用上游。

### 11.3 Base64 与 data URI

- 纯 Base64 和 data URI 在当前 Worker 内存中解码并验证。
- data URI 的 MIME 声明只作为辅助信息，必须与识别出的实际内容兼容。
- 不把原始字符串或图片字节写入数据库和日志；解码后的规范图片字节写入第 11.4 节定义的持久
  暂存文件。
- 暂存写入和数据库提交完成后立即释放原始字符串及内存解码字节。后续 S3 上传只读取暂存文件。

### 11.4 持久暂存协议

异步图片必须配置 `ASYNC_IMAGE_STAGING_DIR`，目录位于应用进程之外的持久存储。单节点部署可使用
持久磁盘；多节点部署必须把同一共享持久卷以一致根目录挂载到所有 Worker。没有共享卷时禁止启用
多节点异步图片 Worker，避免任务被另一个节点领取却无法读取源文件。

暂存路径只由系统生成，格式为：

```text
{user_id}/{yyyy}/{mm}/{task_id}/{index}.img
```

年月使用生成结果完成暂存时的 UTC 时间。数据库只保存相对路径；路径拼接必须清理并验证最终路径
仍位于配置根目录内，不能使用任何用户提供的文件名或路径片段。每个索引按以下协议写入：

1. 在目标目录创建权限为仅应用账号可读写的随机临时文件。
2. 流式写入规范图片字节，同时计算大小和 SHA-256，并再次执行单对象大小上限。
3. `fsync` 文件，关闭后以原子 `rename` 移到最终相对路径，再 `fsync` 父目录。
4. 重新打开最终文件，验证大小、SHA-256、MIME 和扩展名映射。
5. 全部索引完成后，在结算事务中写入 manifest 和 `StorageObject.staging_*` 元数据。

若标准响应已经完整有效，但任一索引因持久卷突然只读、空间耗尽或硬件损坏而无法暂存，不能走生成
失败退款，因为上游成本已经发生。系统必须用专门事务写入 `status=succeeded`、
`billing_status=settled`、`output_availability=failed`、`staging_status=failed` 和稳定错误码
`archive_staging_failed`，并触发高优先级 Root 数据完整性告警。仍在内存中的 Worker 应在租约和
排空期限内优先重试暂存；成功后即可转入普通人工/自动上传流程。若进程崩溃且字节未落入任何持久
介质，只能依赖持久卷快照或其他基础设施恢复，不能重新调用上游、退款或重复扣费。第 11.2 节的
URL 来源获取终止使用相同的 `succeeded/settled/failed` 计费原则，但使用独立错误码区分尚未取得
源字节与暂存介质损坏。

这属于 S3 `PutObject` 之前的灾难性持久化故障，不属于“上传失败”。所有实际发起过 S3 上传的任务
都已具备完整暂存文件，因此普通上传失败始终可以在管理后台人工重传。为把灾难窗口降到最低，POST
接收前和 Worker 领取前都执行暂存存储健康检查，并监控容量、inode、挂载可写性和共享卷可达性。

服务启动和周期性协调任务扫描数据库与暂存目录：删除没有对应已提交任务且超过安全宽限期的孤立
临时文件；对已提交但文件缺失、暂存失败或校验失败的任务标记高优先级运维事故并通知 Root。正常
上传失败的暂存文件永不被通用清理扫描删除。

暂存文件仅在以下条件之一满足后删除：

- 全部 S3 对象已通过 Put/Head 确认可用，任务 `output_availability=available` 已提交；
- 管理员未来执行明确的永久放弃流程，并确认不再承诺人工重传。

删除暂存文件前，必须在任务已经提交 `output_availability=available` 后把对象行从
`staging_status=available` 条件更新为 `staging_status=delete_pending`；删除成功后写入
`staging_status=deleted` 和 `staging_deleted_at`。删除失败或进程中断时保留 `delete_pending`，由
协调任务幂等补做，不能因为本地删除结果未知就提前写成 `deleted`。`output_availability=failed`
的任务不得进入该暂存清理状态，即使其中某个已成功 S3 对象后来到期或被物理清理。

### 11.5 内容类型和扩展名

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

纯 Base64 或 data URI 无法解码、实际内容无法识别、声明与内容冲突或不是允许图片类型，说明内联
标准响应本身无效，按生成失败退款，不进入 S3 上传重试。对于结构有效的 URL 响应，下载后的内容
校验属于来源获取阶段；若最终不是允许图片类型，按第 11.2 节的来源获取事故结算并记录
`archive_source_fetch_failed`，不能退款或伪装成普通 S3 上传失败。SVG 不在默认允许列表中，避免
预签名 URL 被当成可执行文档打开。已经取得有效图片内容后发生的持久卷或 S3 基础设施错误才属于
可从暂存恢复的归档失败。

### 11.6 Object Key

首次准备上传某个索引时捕获 UTC 时间，并在发起 Put 前把由该年月生成的 Object Key 持久化到对象
行；后续自动重试、人工重传、跨月恢复和 `HeadObject` 都必须复用该已保存 Key，不能按重试时间
重新计算。只有明确执行第 19.3 节的整任务位置重新绑定时才更新 Endpoint、Region 和 Bucket，Key
本身仍保持不变。Put 成功后才写入该次成功的 `uploaded_at`。对象有效期从 Put 成功返回时开始
计算，避免网络耗时缩短实际保留期。索引从 0 开始：

```text
prod/user-files/zmodel@async-images/42/2026/07/task_xxx/0.png
```

`user_id`、`task_id`、`index` 和扩展名都来自系统生成或固定映射，不接受调用方路径片段。

### 11.7 上传协议

每个对象按以下顺序处理：

1. 打开对应暂存文件并校验大小、SHA-256 和 MIME；不满足时停止上传、保留任务失败状态并触发
   高优先级运维通知，不能调用上游补生成。
2. 在数据库中 upsert 对应 `StorageObject` 为 `uploading`，写入 S3 定位快照和确定性 Key。
3. 使用私有 `PutObject` 上传，不设置公开 ACL；同时写入固定自定义元数据，包括内容 SHA-256、
   业务 ID、任务 ID 和对象索引，元数据值只来自系统和已校验暂存清单。
4. 上传成功后，以 Put 成功返回时的当前 UTC 时间写入 `uploaded_at`，并计算
   `expires_at = uploaded_at + task.retention_seconds`。
5. 在同一对象行更新中写入 MIME、扩展名、大小、ETag 和 `status=available`。

若 Put 请求成功但客户端在收到响应前崩溃，对象行仍是 `uploading`。恢复时必须先调用
`HeadObject` 检查确定性 Key：如果对象已经存在，且长度、内容类型、自定义 SHA-256/业务标识元数据
和 `LastModified` 都满足预期，则使用 S3 返回的 `LastModified` 补写 `uploaded_at`，重新计算
`expires_at`；只有计算结果仍大于当前时间时才把对象行补为 `available`，否则立即从持久暂存文件
覆盖上传并开始新的有效期。对象不存在、缺少可信 `LastModified` 或任一元数据不匹配时同样覆盖
上传。不能使用恢复进程的当前时间伪造原 Put 成功时间，也不能使用 multipart ETag 充当内容哈希。

每次自动或人工归档尝试都不能只处理 `status=failed` 的对象。对于同一任务中状态为 `deleted`，
或已标记 `available` 但 `expires_at <= now`、Head 检查发现缺失或元数据不匹配的对象，也必须从
暂存文件用相同 Key 覆盖上传并重新计算 `uploaded_at/expires_at`。这样多图任务即使部分对象早已
成功、另一部分长期失败，恢复后仍能得到一组全部未过期的图片。

`delete_pending` 表示清理器已经领取删除操作，上传器不得与它并发覆盖。人工重试可以先把任务改为
`archiving`，但遇到 `delete_pending` 对象时必须等待清理器把它提交为 `deleted`，再从暂存文件上传；
清理器也只能把 `available` 通过条件更新领取为 `delete_pending`，不能领取已经变成 `uploading` 的
对象。这样不会出现旧 DeleteObject 请求在新 PutObject 成功后把刚恢复的对象再次删除。

只有全部索引都为 `available` 才把 `output_availability` 标记为 `available`。任何部分成功都不会
出现在查询响应中，且此处不再发生计费调整。任务可用状态提交后，按第 11.4 节协议异步清理暂存
文件；如果暂存清理失败，不影响用户访问 S3 对象，由协调任务继续重试清理。

## 12. S3 客户端

新增可注入接口，业务代码不直接依赖具体 AWS 客户端：

```go
type ObjectStorage interface {
    PutObject(ctx context.Context, input PutObjectInput) (PutObjectResult, error)
    HeadObject(ctx context.Context, input HeadObjectInput) (HeadObjectResult, error)
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

客户端工厂接收一次不可变设置快照。Secret 明文存在于 Option 表、读取后的短生命周期设置快照和
AWS credential provider 中；错误日志必须经过脱敏，不能打印请求签名、Access Key、Secret 或
完整预签名 URL。

## 13. 后台执行与租约

### 13.1 系统任务类型

新增两个周期/唤醒调度类型和一个按需管理类型：

- `async_image_process`
- `async_image_storage_cleanup`
- `async_image_archive_bulk_retry`：只由 Root “重新上传全部失败图片”操作创建，完成分批状态重置后结束。

每个类型使用现有 `SystemTask`/`SystemTaskLock` 全局租约。系统不会为每个图片请求创建一个
`SystemTask`。提交任务只调用“确保存在一个活动处理任务并唤醒 Runner”；处理器完成一个批次后
结束，调度器在仍有工作时继续创建下一批。批量重试任务把扫描游标、稳定上界和 accepted/skipped
计数写入 SystemTask 进度，进程重启后从游标继续。

处理器使用固定的小规模并发执行已领取任务，不增加新的后台并发设置。并发上限作为代码常量，
避免一次领取大量响应造成无界内存占用。所有来源在完成持久暂存后具有相同的恢复语义：临时归档
失败可释放行租约，按 `next_attempt_at` 由任意挂载同一共享暂存卷的节点恢复。

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
- `succeeded + archiving + settled` 且到达 `next_attempt_at`：继续自动归档。

`refunded`、`available`、`expired` 和终态 `failed` 默认不进入自动生成路径。`settled + archiving`
是正常归档状态，自动执行器只能继续归档或标记归档失败，不能再调整用户额度。
生成失败状态和退款在第 16.4 节的同一事务中提交，因此正常数据库状态中不得出现
`status=failed AND billing_status=reserved`；启动协调器发现该组合时必须暂停该任务、记录高优先级
账务一致性告警并交给专用修复流程，不能把它当作普通归档或重新生成任务领取。

### 13.4 服务停机

现有 `http.Server.Shutdown` 只等待 HTTP 请求，不会自动等待 SystemTask 后台 goroutine。本功能必须
为 SystemTask Runner 增加显式停止和排空能力，并接入 `main.go` 的 SIGTERM/SIGINT 流程：

1. 收到停机信号后停止调度和领取新的异步图片任务。
2. 尚未开始调用上游的任务立即停止。已经向上游发出生成请求的 Worker 不主动取消，在排空时限内
   等待响应；一旦取得标准响应，优先完成 URL 下载、Base64/data URI 解码、原子暂存和 manifest
   事务提交，因为此时原始结果可能只存在于当前进程内存。尚未开始的 S3 上传可以取消；正在执行的
   单个 PutObject 可在排空时限内完成，超时后由恢复流程处理。
3. 在 `SHUTDOWN_TIMEOUT_SECONDS` 内等待活跃异步图片 Worker 退出，并在退出前停止续租。持有
   尚未提交暂存清单的 Worker 在排空期限内优先完成当前原子暂存步骤；已经提交暂存清单的 S3
   上传可以安全停止并由租约恢复。
4. 超时后进程退出，由其他节点或重启后的实例在租约过期后接管。

计划内滚动发布先停止领取，再等待已经发出的上游请求和仅存在于内存中的来源完成持久交接；已提交
暂存清单的 S3 归档不要求在停机前全部完成。进程崩溃、强制 kill 或超过停机时限时由第 14 节的
租约、暂存协调和 `HeadObject` 流程恢复；若完整清单尚未提交，则仍存在第 14.1 节所述重新发送
上游请求的极小窗口。正在写入但尚未原子重命名的临时文件不会被视为可结算结果，并由孤立文件
清理流程回收。

## 14. 崩溃恢复

### 14.1 上游请求窗口

上游请求不是数据库事务，无法与任务状态原子提交。如果进程在上游已经生成图片、但尚未持久化
完整 manifest 前崩溃，恢复后可能重新发送一次生成请求。系统保证本地只结算一次，但无法替不
支持幂等键的上游消除该极小窗口中的重复生成成本。计划停机通过停止领取和限时排空显著缩小该
窗口；机器宕机、`kill -9` 和排空超时仍可能触发它。首期接受“用户只扣一次、平台在极端情况下
可能承担一次额外上游费用”的残余风险，不引入外部消息队列、分布式事务或伪造的 exactly-once
承诺。若某个上游未来提供稳定幂等键，可在共享执行层按任务 ID 透传以进一步降低风险。

### 14.2 已暂存输出

完整暂存清单提交后，URL、Base64 和 data URI 具有相同恢复流程。恢复进程逐索引处理：

- 如果对应对象是 `available` 且尚未到期，仍用 `HeadObject` 校验长度、内容类型、自定义
  SHA-256/业务标识元数据和 `LastModified`；全部符合才跳过，缺失或不匹配时从暂存覆盖上传。
- 如果对象是 `uploading`，先用 `HeadObject` 检查确定性 Key；存在且全部元数据符合时，使用
  `LastModified` 补齐 `uploaded_at/expires_at` 和对象状态。
- 如果对象不存在、已经到期、已删除或 Head 校验不通过，从共享持久暂存文件读取并重新上传。
- 每次读取前核对相对路径、大小、SHA-256 和 MIME，避免损坏文件被上传。

自动重试窗口耗尽时只把任务标为归档失败并通知管理员，暂存文件继续保留。管理员稍后修复 S3
配置或服务后，可从管理页重新上传；该流程不依赖原进程、原上游 URL 或内存数据。

### 14.3 暂存提交窗口

在标准响应返回后、完整暂存清单提交前崩溃，数据库任务仍为 `running + reserved`。恢复时不会把
未提交文件当作结果，而是按 14.1 节重新执行生成；旧临时文件由协调任务按宽限期清理。暂存文件
已经原子重命名但数据库事务尚未提交时同样属于孤立文件，不可用于跳过生成，因为缺少完整、原子
提交的 manifest 和计费事实。

正常数据库事务提交后，`succeeded + settled`、完整 manifest 和全部 `staging_status=available`
同时可见。此后服务重启不会中断流程。若已提交任务的暂存文件缺失或哈希不匹配，这是持久存储
损坏，不是普通 S3 上传失败；系统保持 `output_availability=failed`、禁止退款和重新生成，并向
Root 发出高优先级数据完整性告警，直到运维从持久卷备份恢复该文件。第 11.4 节的极端暂存失败则
通过 `succeeded + settled + failed` 的专门事务显式记录，不伪装成生成失败。

### 14.4 最终状态写入窗口

正常路径的完整 manifest、全部暂存元数据、`status=succeeded`、最终额度调整和
`billing_status=settled` 在同一事务中提交，不会暴露 `succeeded + reserved` 中间态。若进程在该
事务提交前崩溃，按 14.1 的上游请求窗口恢复；若事务已提交，则后续只能继续归档。极端暂存失败
使用第 11.4 节的专门结算事务，同样不暴露 `succeeded + reserved`。若全部对象已上传但进程在
`output_availability=available` 写入前崩溃，恢复进程通过对象行和 `HeadObject` 发现完整结果，
并补写可用状态。

## 15. 重试策略

本功能的可配置重试只针对取得标准图片响应后的归档、对象存储和输出状态完成阶段。图片生成仍使用
现有渠道 `RetryTimes` 和 `shouldRetry` 规则。

- `archive_timeout_seconds`：一次任务归档尝试的上下文超时，默认 600，最大 1200。
- `archive_max_attempts`：业务归档尝试次数，默认 8。
- `archive_retry_window_seconds`：从提交时开始计算的最长窗口，默认 21600。
- 退避：从 15 秒开始指数增长，最多 15 分钟，不使用随机数，便于确定性测试。

所有已完成暂存的任务都把下一次时间写入 `next_attempt_at` 后释放租约，由任意合格节点恢复。

内联 Base64/data URI 无法解码、解码后图片超过限制、MIME 不受支持或响应结构无效发生在成功结算
前，按无效标准响应执行生成失败退款。结构有效的 URL 响应在下载、大小或内容校验阶段最终失败时，
按来源获取终止进入 `archive_source_fetch_failed` 并保留 `settled`；S3 永久错误和暂存完整性故障也
进入归档最终失败并保留 `settled`。普通 S3 上传失败可直接等待人工重传；来源获取终止需要运维
补齐源文件，暂存完整性故障需要先从持久卷备份恢复源文件，二者在恢复完整暂存前都不能进入上传
队列。

S3 网络超时、S3 错误和预期可由管理员修复的认证错误按普通归档临时错误重试。URL 下载临时失败
只在原始响应仍存在于当前 Worker 内存时重试；成功落入持久暂存后才进入普通 S3 归档重试。URL
来源获取最终失败按第 11.2 节进入 `archive_source_fetch_failed`，不能因缺少持久化来源而加入普通
批量上传队列。满足以下任一条件后停止普通归档重试：

```text
archive_attempts >= archive_max_attempts
当前时间 >= archive_retry_deadline_at
```

停止后把输出标记为 `failed`，保留 `billing_status=settled` 和全部暂存文件。管理员重试会重置
归档尝试计数和重试窗口，但不改变计费状态；所有普通 S3 上传失败都允许重试。

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

### 16.3 生成成功结算

完整标准图片响应经过结构、数量和内容来源校验后，立即使用冻结的计费上下文结算。计费图片数
必须复用同步 `ImageHelper` 的现有语义：优先使用渠道写入的 `ActualBillingDimensions.Units`，否则
使用已校验请求的 `n`，不能改为按响应数组长度计费。这样同一请求走同步或异步入口时价格一致，
也避免上游按请求数量收费而平台仅按返回条目数扣费。响应数组必须非空且不超过请求允许数量和
`dto.MaxImageN`；违反约束的响应按无效上游响应执行生成失败退款。

结算事务：

1. `lockForUpdate(tx)` 锁定任务。
2. 只有 `status=running AND billing_status=reserved` 可以结算；重复调用对 `settled` 幂等返回。
3. 正常路径再次验证完整 manifest 和全部已原子写入的暂存文件元数据；响应条目数只决定 manifest，
   不覆盖冻结的计费维度。URL 来源获取终止或极端暂存失败路径保存已知输出索引和完整性错误，但
   不把来源下载或暂存成功作为是否结算的条件。
4. 使用同步路径同一套计费维度计算 `actual_quota`，按 `actual_quota - reserved_quota` 调整
   钱包/订阅和 Token 配额。
5. 原子写入 `actual_quota`、`billing_status=settled`、`billing_finalized_at`、`status=succeeded` 和
   `generation_completed_at`；正常暂存路径写 `output_availability=archiving`，URL 来源获取终止或
   极端暂存失败路径写 `output_availability=failed` 和对应的 `archive_source_fetch_failed` 或
   `archive_staging_failed`。
6. 清空 `request_payload`，归档阶段从此不能重新调用上游，也不能再次调整计费。

如果任务已经 `refunded`，结算函数返回稳定的“禁止自动重新计费”结果。S3 对象是否已经上传不是
本事务的前置条件；归档成功只更新输出可用性，归档失败也不回滚本次结算。

### 16.4 生成失败幂等退款

退款只适用于现有渠道重试全部失败、无可用渠道或标准图片响应无效，不适用于 S3 归档失败。
生成失败事务：

1. `lockForUpdate(tx)` 锁定任务。
2. 只有 `billing_status=reserved` 可以退款；`refunded` 直接幂等返回，`settled` 明确拒绝自动退款。
3. 只按任务保存的 `reserved_quota` 和 `token_reserved_quota` 退回对应资金来源。
4. 原子写入 `status=failed`、`output_availability=failed`、`billing_status=refunded`、
   `billing_finalized_at`、稳定错误信息和 `admin_notification_state=none`。
5. 清空 `request_payload`，避免终态任务永久保存原始 Prompt、扩展字段或临时上游签名 URL。

资金变更和状态变更在同一数据库事务中，因此进程重启后不会重复退款。

### 16.5 归档失败不调整计费

普通 S3 归档达到重试终点时，只更新 `output_availability=failed`、错误信息和管理员通知意图，并
保留全部暂存文件。来源获取或暂存完整性事故通过生成成功专门结算事务进入相同输出失败状态，并
保留所有已经成功落盘的暂存文件及缺失索引信息。两类后续处理都必须要求
`status=succeeded AND billing_status=settled`，不得调用退款方法；普通归档执行器也不得再次调用
结算方法。

### 16.6 日志和统计

图片消费日志和用户/渠道统计在生成成功结算时产生；生成失败退款写任务退款日志。归档失败另写
不影响额度的运维/管理员日志，便于按 `task_id` 追踪上传错误和管理员重试。主额度事务不依赖日志
数据库成功。若日志数据库独立且写入失败，记录带 `task_id` 的系统告警，不回滚已经完成的资金
事务，也不通过再次资金调整来补日志。

## 17. 最终失败和通知

生成失败、普通 S3 上传失败和 S3 前完整性事故使用明确区分的终结路径：

1. 生成失败执行第 16.4 节的幂等退款，写入 `status=failed`，不发送对象存储告警。
2. 普通 S3 上传失败保持 `status=succeeded` 和 `billing_status=settled`，写入
   `output_availability=failed`；完整暂存文件、对象定位和 manifest 全部保留以供人工重新上传。
3. 来源获取终止或暂存完整性事故同样保持 `succeeded/settled/failed`，但写入独立稳定错误码和
   缺失/损坏索引。管理页必须显示为需要先恢复源文件的运维事故；在完整暂存校验通过前，勾选重试
   和一键重试都不得把它放入普通上传队列。
4. 不把部分成功对象标记为删除；人工重试通过 `HeadObject` 跳过仍存在且未过期的有效对象，重新
   上传失败、缺失、元数据不匹配或已经到期的索引。只有任务可用后正常到期，或明确永久删除流程
   才能转为 `delete_pending`。
5. 归档失败事务生成一个任务级聚合通知意图；内容包含任务 ID、用户 ID、模型、已尝试次数、对象
   总数、已上传数、暂存完整性状态和最后错误摘要，不包含 Prompt、图片字节、Secret 或预签名 URL。
6. 管理员重试被接受时，把上一轮已经终结的通知状态重置为 `none`，使本轮失败能够生成一条新的
   聚合通知；本轮成功后清除错误和通知状态，写入 `manually_recovered_at`。本轮失败则重新进入
   同一归档失败路径，仍不退款、不重新生成和不重复扣费。

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
是否已经运行。只有当前完整可用的任务到期后才立即停止签名，并将任务输出可用性更新为
`expired`。处于 `failed` 的任务不会因部分对象到期而转为 `expired`，其暂存文件和人工重传资格
继续保留。

### 18.2 物理删除

`async_image_storage_cleanup` 按设置的清理间隔运行，分批处理：

- `status=available AND expires_at <= now`
- `status=delete_pending`

S3 返回成功或对象不存在都视为删除成功，写入 `status=deleted` 和 `deleted_at`。删除失败时增加
`delete_attempts` 并保存脱敏错误，等待下一轮。数据库对象记录和任务记录永久保留。

若对象所属任务仍为 `output_availability=failed`，该清理只删除已到期的 S3 副本，必须保留
`staging_status=available` 和暂存文件，以便管理员稍后重新上传；只有任务曾完整进入 `available`
并已提交暂存清理意图时，才清理对应暂存文件。

清理任务每批使用短事务把符合条件的 `available` 对象 CAS 为 `delete_pending`，不在数据库事务内
执行网络请求。上传器不处理 `delete_pending`，必须等待其成为 `deleted`；对象状态完成更新仍使用
条件更新。该串行合同避免清理与正在上传的 Worker 互相覆盖或发生删除新对象的竞态。

## 19. 对象存储设置

### 19.1 Option Key 和默认值

新增设置快照，Option Key 使用稳定前缀：

| Option Key | 默认值 | 校验 |
| --- | --- | --- |
| `ObjectStorageS3Endpoint` | 空 | 空表示 AWS；非空必须是 HTTP(S) URL |
| `ObjectStorageS3Region` | 空 | 配置 Bucket 时必填 |
| `ObjectStorageS3Bucket` | 空 | 配置存储时必填，不允许空白字符 |
| `ObjectStorageS3AccessKey` | 空 | 配置存储时必填 |
| `ObjectStorageS3SecretAccessKey` | 空 | 按已确认方案以明文存入 Option 表；读取 API 永不返回 |
| `ObjectStorageRetentionSeconds` | `86400` | 60 到 31536000 |
| `ObjectStoragePresignSeconds` | `600` | 60 到 604800，实际签名受对象剩余寿命限制 |
| `ObjectStorageArchiveTimeoutSeconds` | `600` | 1 到 1200 |
| `ObjectStorageArchiveMaxAttempts` | `8` | 1 到 100 |
| `ObjectStorageArchiveRetryWindowSeconds` | `21600` | 60 到 604800，且不小于单次超时 |
| `ObjectStorageCleanupIntervalSeconds` | `900` | 60 到 86400 |

不新增 enable 字段。异步提交的可用条件是 Region、Bucket、Access Key 和 Secret 完整；Endpoint
可以为空。配置不完整只让异步 POST 返回 503，不影响同步图片生成和其他接口。

`ASYNC_IMAGE_STAGING_DIR` 是部署级路径而不是 Option，不在后台页面编辑，避免不同节点通过数据库
配置得到一个实际挂载不一致的本地路径。启动时必须存在且可用；多节点由部署系统保证同一共享卷
挂载。服务健康检查单独暴露暂存卷不可写、容量不足或共享卷不可达状态。

### 19.2 Secret 来源和持久化

按已确认要求，Root 在后台提交的 Secret 以明文写入现有 Option 表，Key 为
`ObjectStorageS3SecretAccessKey`。运行时设置快照从 Option 读取该值并交给 AWS credential provider；
不再设计环境变量优先级、应用层加密、加密主密钥或密文轮换流程。

这一选择意味着拥有主数据库读取权限的人，以及能够读取数据库备份、快照或副本的人，可以取得
S3 Secret。部署和备份权限必须按此风险边界收紧；数据库备份也应使用基础设施层加密和访问控制，
但应用不会声称数据库中不存在明文。

应用层仍执行以下保护：

- 专用 GET 和通用 Option GET 都不返回 Secret。
- 任何请求日志、设置变更审计、错误日志和通知都不得记录 Secret。
- PUT 的空 `secret_access_key` 表示保留 Option 中的现有明文；非空值替换现有明文。
- 全部字段先组成候选配置并校验，通过后使用 `model.UpdateOptionsBulk` 一次更新；任一字段失败时
  不部分保存。
- Secret 只允许出现在受 RootAuth 保护的 PUT 请求体、Option 表、短生命周期运行时设置快照和 AWS
  credential provider 中，不写入任务表、对象表、暂存文件或前端持久状态。

### 19.3 Root API

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

GET 不包含 Secret 字段。通用 `GET /api/option` 必须把 `ObjectStorageS3SecretAccessKey` 加入敏感
Key 过滤规则，不能因为它存于现有 Option 表而返回给浏览器。

PUT 接收完整非敏感配置和可选 `secret_access_key`：

- 空 Secret 保留 Option 中的旧明文，非空 Secret 替换旧明文。
- 若当前没有 Secret，而其他存储字段表示要启用配置，则空 Secret 校验失败。
- 全部字段先在内存中组成候选快照并整体校验，再通过 `model.UpdateOptionsBulk` 一次持久化。
- 任一字段失败时不部分更新。
- 保存非空配置前，用候选凭据向候选位置写入、Head 并删除一个随机探针对象；任一步失败都拒绝
  保存，探针 Key 不使用业务前缀且不得记录 Secret。删除探针失败时拒绝保存并告警，避免留下垃圾。

Endpoint、Region 或 Bucket 属于对象物理位置。存在当前 `available`、`uploading` 或
`delete_pending` 对象时，普通 PUT 拒绝改变这三个字段，避免旧对象因定位切换而无法签名、完成
上传或清理；物理迁移必须使用独立迁移流程，不在普通设置保存中隐式完成。若相关失败任务没有任何
仍存活的 S3 对象，所有对象均为 `failed/deleted` 且持久暂存完整，则允许 Root 保存新的物理位置。
作出“没有存活对象”的判断前，必须对 `failed` 行的确定性旧 Key 执行 `HeadObject`，避免 Put 已成功
但数据库未确认时把同一任务分散到两个位置。保存事务把这些对象按任务整体重新绑定到候选位置并
重置为待上传，不能只移动一个索引。探针和旧位置 Head 检查成功后才能执行该重绑定。

Access Key 和 Secret 可以在原位置轮换，调用方负责保证新凭据仍能访问同一 Bucket。每次上传、
Head、删除或签名尝试只使用开始该次尝试时取得的一份不可变设置快照；重试时重新读取最新凭据，
但对象定位始终使用对象行中的 Endpoint、Region 和 Bucket 快照。这样凭据轮换不会让单个 S3 请求
中途换凭据，也不需要在任务表中持久化 Secret 或配置版本。

除上述满足严格条件的失败任务整组重新绑定外，连接参数更新只影响尚未创建 `StorageObject` 的
对象和后续预签名/删除所使用的凭据。已创建对象继续使用自身保存的 Endpoint、Region 和 Bucket
定位快照。

## 20. 后台 API 和页面

### 20.1 异步图片任务 API

用户查询与 Root 运维使用独立资源，不复用现有视频/音乐 `/api/task` 接口：

```text
GET  /api/async-image-task/self
GET  /api/async-image-task
POST /api/async-image-task/retry
POST /api/async-image-task/retry-failed
```

- `/self` 使用 `UserAuth`，只返回当前用户任务，不包含内部错误、对象物理位置或管理操作。
- Root 列表和重试接口使用 `RootAuth`；列表支持任务 ID、用户、模型、生成状态、输出可用性、计费
  状态、暂存完整性和创建时间筛选，并返回分页结果和对象成功数/总数。
- `retry` 接收去重后的任务 ID 数组并设置合理批量上限；`retry-failed` 创建或复用一个
  `async_image_archive_bulk_retry` SystemTask，记录启动时的最大任务主键作为稳定扫描上界，并在
  后台分批处理该范围内全部归档失败任务，避免一个 HTTP 请求锁定全表或执行超时。
- 两个重试接口只接受 `status=succeeded AND billing_status=settled AND
  output_availability=failed`。事务内确认全部预期暂存文件存在且通过路径、大小、SHA-256 和 MIME
  校验，然后清除归档错误，把上一轮终结通知状态重置为 `none`，重置尝试次数、重试窗口和
  `next_attempt_at`，写入 `output_availability=archiving`，提交后唤醒处理器。每个任务使用独立短
  事务，单个完整性异常不能回滚同批其他可重试任务。
- `retry` 同步返回 `accepted_count`、`skipped_count` 和按稳定原因汇总的跳过数量；
  `retry-failed` 返回 `operation_id`，页面轮询操作进度并展示累计 accepted/skipped 数量。已过期
  输出、生成失败或仍在归档的任务不会被接受。暂存文件缺失或损坏的任务作为运维事故计入
  `integrity_error_count`，保持失败状态并通知 Root，不能被静默跳过。批处理进程重启后由 SystemTask
  租约继续，不重复改变已进入 `archiving` 的任务。
- `operation_id` 使用 SystemTask 的任务 ID；进度查询复用现有 Root-only
  `GET /api/system-task/:task_id`，不再增加另一套批处理状态表或轮询协议。
- 每次 Root 重试写操作审计，包含操作者、任务数量和筛选条件，不记录 Prompt、来源 URL 或凭据。
  重试只恢复归档，不调用上游、不退款、不补扣。
- 人工重传完成后，只有全部对象均为 `available` 且 `expires_at > now`，才原子更新任务为
  `output_availability=available`、清空归档错误并写入 `manually_recovered_at`。从该事务提交起，
  用户 GET 会动态签发有效图片 URL，用户可以查看或下载图片；随后再异步清理暂存文件。

### 20.2 对象存储设置页

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

时长在界面中使用明确单位的数字输入，提交时转换为 API 秒值。Secret 输入初始永远为空，只显示
“已配置/未配置”状态；空输入表示保留，非空输入表示替换。页面明确提醒 Root：该 Secret 按当前
方案以明文存入数据库 Option，数据库及备份读取者可以访问。页面不显示启用开关和 Key Prefix。

### 20.3 异步图片任务页

在 Usage Logs 下新增独立分区 `/usage-logs/async-images`，名称为 “Async Image Tasks”。不合并到
现有 “Task Logs”，因为后者对应 `model.Task` 的视频/音乐供应商轮询语义，状态、数据列和操作都
不兼容。普通用户可查看自己的异步图片任务；Root 用户在同一分区获得管理列和批量操作。

页面提供：

- 任务 ID、用户、模型、生成状态、输出可用性、计费状态、暂存完整性和时间筛选。
- 任务 ID、用户、模型、生成状态、输出状态、计费状态、对象成功数/总数、尝试次数、脱敏错误和
  时间列；详情抽屉展示逐对象状态，但不显示 S3 凭据或完整临时来源 URL。
- 表格勾选后的“Retry selected uploads”操作，可选中全部普通归档失败任务。
- Root 顶部“Retry all failed image uploads”按钮，中文为“重新上传全部失败的图片文件”；二次确认后
  调用批量接口，并展示已接受、已跳过和暂存完整性异常数量。本期范围是异步图片，因此不使用
  “失败的视频文件”文案。
- 暂存完整性异常显示明确事故状态和文件恢复要求；普通 S3 上传失败不得显示为不可重试。

所有用户可见文本使用 `useTranslation()` 和英文语义键，并同步六个 locale：`en`、`zh`、
`fr`、`ru`、`ja`、`vi`。页面保持现有紧凑后台视觉，不增加营销式 Hero 或装饰性卡片。

## 21. 安全和隐私

- S3 对象为私有对象，只通过短期预签名 GET 访问。
- 预签名 URL 不写数据库、不写业务日志。
- S3 Secret 按第 19.2 节以明文写入 Option 表，因此数据库及备份读取权限属于敏感权限；Secret 不写
  入任务表、对象表、暂存文件、日志或 API 响应。渠道 Key、Token Key 和运行时下载认证 Header
  也不持久化到任务、对象或暂存文件。
- URL、Base64 和 data URI 的原始表示不写数据库、暂存文件或日志；只把识别、校验后的规范图片
  字节写入权限受限的共享持久暂存目录。
- 暂存文件名和相对路径完全由系统生成，不接受用户路径；数据库只保存相对路径，API 不返回路径。
- 多节点必须使用共享持久卷；暂存卷和备份应使用基础设施层静态加密、最小权限和容量监控。
- 终态任务清空 `request_payload`；永久记录只保留模型、计费快照、输出元数据和脱敏错误。
- 上游 `source_url` 不进入已提交 manifest；下载并可靠暂存后立即释放。
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
- 完整标准响应验证成功即结算，S3 尚未上传时任务已是 `succeeded/archiving/settled`。
- 同一请求在同步和异步入口使用相同的请求 `n`/`ActualBillingDimensions.Units` 计费维度；响应数组
  长度不会降低上游已经产生的请求成本。
- 重复生成成功结算和重复生成失败退款测试验证额度只变化一次。
- 归档最终失败保持 `settled` 且额度不变；归档重试成功也不重复扣费。
- `refunded -> settled` 测试验证自动重新扣费被拒绝。
- SQL 实现只用 GORM 和 `lockForUpdate(tx)`，并通过现有 MySQL/PostgreSQL CI 验证迁移与行为。

### 22.2 设置和 S3

- GET 设置永不返回 Secret。
- 通用 Option GET 过滤 `ObjectStorageS3SecretAccessKey`，专用 GET 只返回 `secret_configured`。
- PUT 空 Secret 保留 Option 中原明文，非空 Secret 替换原明文；数据库测试明确验证该字段按需求明文
  保存，但任何 API 和日志均不出现该值。
- 非法设置组合不部分更新，`model.UpdateOptionsBulk` 的更新保持原子性。
- 候选 S3 设置必须通过 Put/Head/Delete 探针；没有存活 S3 对象且保有完整暂存的失败任务可按任务
  整体重新绑定新位置，仍有成功对象时普通设置 PUT 不执行隐式迁移。
- 存在 `available`、`uploading` 或 `delete_pending` 对象时拒绝修改 Endpoint、Region 或 Bucket；仅当
  失败任务全部对象均为 `failed/deleted`、暂存完整且旧 Key 经 `HeadObject` 确认不存在时，才允许
  在同一事务中按任务整组重新绑定，不能只重绑部分索引。原位置凭据轮换仍允许。
- 默认值和全部数值边界测试。
- 自定义 Endpoint 使用 path-style，AWS 空 Endpoint 使用标准解析。
- Fake ObjectStorage 验证私有 Put、跨月重试仍复用确定性 Key、`HeadObject` 校验长度/类型/自定义
  SHA-256 与业务标识元数据、使用 `LastModified` 恢复真实上传时间且不会延长已过期对象寿命、预签名
  剩余寿命上限和删除 404 幂等。

### 22.3 归档和恢复

- URL、纯 Base64 和 data URI 各自成功归档。
- 正常路径中三种来源都在结算前原子写入共享持久暂存，并在数据库中保存相对路径、大小、MIME 和
  SHA-256；极端暂存故障仍结算但显式进入完整性事故状态。
- URL 来源在原始响应仍位于 Worker 内存时允许重试下载；最终无法取得源字节时按
  `archive_source_fetch_failed` 结算并告警；URL 下载后的大小、MIME 或内容校验最终失败也走该路径，
  不错误退款或进入普通 S3 上传重试队列。
- 暂存目录不可写时异步 POST 在预扣前返回 503；Worker 领取前的健康检查失败时不发起上游请求。
- MIME/扩展名由实际内容决定。
- 单对象超过 `MAX_FILE_DOWNLOAD_MB` 失败。
- 多图部分上传不向 GET 暴露。
- URL、Base64 和 data URI 任务都能在进程或节点恢复后从共享暂存继续归档。
- Put 成功但对象行仍为 `uploading` 时用 `HeadObject` 和 `LastModified` 补齐真实上传时间与成功状态；
  已经过期、确认不存在或元数据不匹配时从暂存文件覆盖重传。
- 多图部分成功后跨过对象有效期再人工重传时，已过期索引也会覆盖上传，最终所有对象均未过期才
  恢复任务可用性。
- 清理器已把对象领取为 `delete_pending` 时，人工重传等待删除完成再上传；并发测试验证旧删除请求
  不会删除新上传对象。
- 归档失败任务中的部分成功对象到期后，任务仍保持 `failed` 且可被批量重传；不会错误转为
  `expired` 而失去重传资格。
- 暂存临时文件使用 `fsync + rename` 原子提交；崩溃遗留文件按宽限期清理，不会被误当作已结算输出。
- 上传失败保留暂存文件；全部对象可用且任务状态提交后以
  `staging_status=available -> delete_pending -> deleted` 幂等清理暂存文件，清理中断可恢复。
- 暂存文件缺失或哈希错误触发高优先级完整性告警，不退款、不重新生成、不静默跳过。
- 计划停机停止领取新任务并等待 Worker 排空；超过时限后由租约恢复，不留下永久 `running` 状态。
- 归档超时、最大尝试次数和最长窗口使用可控时钟测试。
- 上游生成失败退款但不通知；归档最终失败只产生一个聚合管理员通知意图。
- Root 勾选重试和一键重新上传全部失败图片覆盖所有普通归档失败项；前者返回 accepted/skipped
  数量，后者创建可恢复的后台批处理并累计进度，且两者都不调用上游、不退款、不补扣。
- 人工重试按任务使用独立事务；重试再次失败时每轮最多重新产生一条聚合通知，成功时清除错误和
  通知状态。
- 人工重传全部成功后，任务恢复为 `available`，用户 GET 立即返回可观看或下载的有效预签名 URL。
- 到期立即停止签名，清理成功后数据库行仍存在。

### 22.4 API 和图片中继

- 同步图片接口响应和计费回归测试保持不变。
- 异步 POST 返回 202，且 `stream=true` 返回 400。
- S3 未配置时在预扣费前返回 503。
- 原始扩展字段在后台执行时仍存在。
- GET 所有状态的 JSON 合同、用户隔离、404 和签名 503。
- 单渠道图片执行共享单元不自行结算或退款；同步入口保持原有结算一次，异步入口在标准响应验证后
  结算一次，归档阶段没有资金副作用。
- 同步入口和异步 Worker 对同一模拟渠道错误使用一致的选择、重试、模型映射和自动禁用规则。
- `/api/async-image-task/self` 只返回当前用户；Root 列表筛选和两个重试接口执行正确权限与状态保护。

### 22.5 前端

- Zod Schema 验证 URL、时长范围、最大 20 分钟和重试窗口关系。
- Secret 已配置时表单仍加载空值，空提交不清除 Secret。
- 设置页只显示 Secret 已配置状态，不向浏览器发送明文，并显示明文存入 Option 和数据库备份的风险提示。
- React Query 保存成功后刷新设置快照。
- 对象存储设置路由和侧边栏入口可访问。
- Usage Logs 异步图片分区对普通用户显示自有任务，对 Root 显示筛选、勾选重试和“重新上传全部
  失败的图片文件”操作；普通上传失败都可选择，暂存完整性异常单独显示事故状态。
- 运行 `bun run typecheck`、涉及文件 lint、i18n 同步/检查和生产构建。

## 23. 验收标准

1. 原有同步图片生成测试和行为不回归。
2. 异步任务对 URL、Base64、data URI 输出都能生成私有 S3 对象。
3. GET 只在全部对象有效时返回预签名 URL，任何时候都不返回部分结果。
4. 对象 Key、UTC 年月、索引和实际扩展名完全符合固定规则。
5. 对象到期后立即停止签名，清理后数据库记录仍保留。
6. 完整标准响应生成成功即完成结算；S3 归档失败保持 `status=succeeded` 和
   `billing_status=settled`，不会退款或再次扣费。
7. 上游生成最终失败或标准响应无效只退款一次；重复 Worker、进程崩溃和重复终结不会重复结算或退款。
8. 正常路径中 URL、Base64 和 data URI 都在结算前写入共享持久暂存，可在服务或节点重启后继续；
   计划停机排空活跃 Worker，`HeadObject` 使用对象元数据和 `LastModified` 恢复 Put 成功但数据库未
   确认的对象，且不会错误延长其有效期。URL 来源获取终止和灾难性暂存故障显式进入
   `succeeded/failed/settled` 完整性事故状态，不错误退款或伪装为普通上传失败。
9. 任何普通 S3 上传失败都保留暂存源文件并可由 Root 人工重新上传；不伪造可用地址，也不静默重新生成。
10. Root 可筛选、勾选批量重试和一键重新上传全部失败图片；重试不产生资金变化，成功后用户可以
    通过任务 GET 获得有效图片 URL 并查看或下载。
11. 普通用户在独立异步图片任务页只能查看自己的任务；现有视频/音乐 Task Logs 数据和接口不迁移、不改变。
12. URL/Base64/data URI 原始表示不出现在数据库、暂存文件或日志中；规范图片字节只存在于受控的
    共享持久暂存目录和私有 S3，并按生命周期清理。
13. Secret 以明文存入 `ObjectStorageS3SecretAccessKey` Option，但不从通用或专用读取设置 API、
    页面或日志返回；空 Secret 更新保留旧值，非空值替换旧值。
14. 对象存储设置页和异步图片任务页在六种语言下可用，并通过类型检查、lint 和生产构建。
15. 原有同步 `/v1/images/generations` 的路由、响应、渠道重试、计费、日志和错误行为通过回归测试保持不变；未调用新异步端点时不新增预扣或 S3 副作用。
16. SQLite 本地测试通过，代码不包含破坏 MySQL/PostgreSQL 兼容性的 SQL 或迁移。
17. 计划停机停止领取新任务，并在统一停机时限内等待已发出的上游生成请求完成响应交接；已暂存的
    S3 归档可在重启后恢复。硬崩溃窗口可能重复调用上游，但不得导致用户重复结算、重复退款或任务
    永久卡在 `running/uploading`。

## 24. 实施顺序

1. Secret 明文 Option、读取过滤、设置快照、专用 Root API、持久暂存配置和含 `HeadObject` 的可注入 S3 客户端。
2. `AsyncImageTask`、`StorageObject`、迁移、状态转换和 CAS 租约。
3. 事务性异步图片预扣、生成成功结算和生成失败退款，先固化计费不变量测试。
4. 提取单渠道图片执行和渠道选择/重试共享原语，保持同步接口回归测试。
5. 异步 POST/GET 合同。
6. 后台处理、URL/Base64/data URI 持久暂存与归档、崩溃恢复、SystemTask 停机排空和聚合通知。
7. Root/用户异步图片任务查询、勾选重试和一键重新上传全部失败图片 API。
8. 到期签名判断和清理系统任务。
9. 默认前端对象存储设置页、Usage Logs 异步图片任务页和六语言翻译。
10. 聚焦测试、全量后端测试、前端验证和最终差异审查。
