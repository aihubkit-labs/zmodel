# 异步图片生成与 S3 临时存储验收手册

## 文档关系

- 需求日期：2026-07-28
- 配套设计：[异步图片生成与 S3 临时存储设计](./2026-07-28-async-image-s3-storage-design.md)
- 验收范围：异步图片任务、私有 S3 归档、持久暂存、计费、失败重传、重启恢复、对象清理和后台页面
- 结论记录：验收人应在本文各检查项后记录通过、失败或不适用，并保存关键任务 ID、截图和日志证据

本文是上述设计文档的配套执行手册，两份文档属于同一个需求。设计文档描述约束和实现方案，本文
描述发布前如何验证这些约束。

## 1. 验收环境

故障注入、IAM 权限回收、服务重启和对象过期测试必须在独立验收环境执行，不得直接操作生产环境。

准备以下信息：

- `BASE_URL`：待验收服务地址。
- `TOKEN`：有目标图片模型权限和足够额度的普通用户 Token。
- `OTHER_USER_TOKEN`：另一个用户的 Token，用于验证任务隔离。
- Root 后台账号。
- 一个私有 S3 Bucket，以及可临时调整权限的验收专用 IAM 凭据。
- 至少一个真实图片渠道；验证 Base64 和 data URI 时还需要支持相应输出的渠道或模拟渠道。

部署时准备持久卷目录并配置统一停机时限：

```bash
export SHUTDOWN_TIMEOUT_SECONDS=120
mkdir -p /data/zmodel/async-image-staging
```

容器必须把后台填写的目录挂载到持久卷。多节点部署必须让全部 Worker 使用同一个共享卷和一致的
挂载路径。应用账号应能创建目录、写入、`fsync`、原子重命名、读取和删除文件，其他系统账号不
应具有不必要的访问权限。暂存目录只通过 Root 后台保存到数据库，不使用环境变量配置。

S3 Bucket 必须禁止匿名读取。应用凭据至少应具有
`prod/user-files/zmodel@async-images/*` 下的 `PutObject`、`GetObject`（包括 `HeadObject`）和
`DeleteObject` 权限。保存设置时使用
`prod/user-files/zmodel@async-images/.probe/{random}` 完成写入、Head 和删除探针，不需要开放 Bucket
根前缀。

AWS IAM 最小对象权限可参考以下策略，并把 `<bucket>` 替换为实际 Bucket 名：

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:DeleteObject"
      ],
      "Resource": "arn:aws:s3:::<bucket>/prod/user-files/zmodel@async-images/*"
    }
  ]
}
```

若保存提示 `cleanup probe failed`，说明写入与 Head 已通过，但缺少 `s3:DeleteObject`；补充上述权限
后重新保存。

## 2. 部署和迁移检查

1. 启动服务，确认启动过程没有数据库迁移错误。
2. 确认数据库存在 `async_image_tasks` 和 `storage_objects` 表。
3. 确认后台准备填写的暂存目录位于持久卷，而不是容器临时文件系统。
4. 多节点环境分别进入各节点，确认相同相对路径指向同一文件。
5. 执行自动化验证：

   ```bash
   go test ./...
   cd web/default
   bun run typecheck
   bun run i18n:sync
   bun run build
   cd ../classic
   bun run build
   ```

预期：全部命令通过；默认前端 i18n 检查不存在缺失、额外或未翻译条目。

## 3. 对象存储设置

使用 Root 登录，进入“系统设置 → 对象存储”。确认进入页面后左侧仍是系统设置下钻菜单，并且
“对象存储”菜单可见且处于选中状态。

填写 Endpoint、Region、Bucket、Access Key、Secret Access Key 和持久暂存目录。AWS S3 的
Endpoint 留空；MinIO 等兼容服务填写 HTTP(S) 地址。持久暂存目录填写
`/data/zmodel/async-image-staging`。首次正常链路建议使用：

| 设置 | 验收值 |
| --- | ---: |
| 对象保留时间 | 600 秒 |
| 预签名 URL 有效期 | 60 秒 |
| 单次归档超时 | 30 秒 |
| 最大归档尝试次数 | 3 |
| 最长重试窗口 | 300 秒 |
| 清理间隔 | 60 秒 |

保存时系统会先对持久暂存目录执行创建、写入、`fsync`、原子重命名、读取和删除探针，再执行 S3
随机探针对象的 `Put -> Head -> Delete`。预期保存成功，暂存目录和 Bucket 中均不遗留探针文件。
刷新页面后持久暂存目录保持原值；Secret 输入框必须为空，只显示“已配置”状态。

没有在途任务和保留文件时，把目录改为另一个非根绝对路径，预期探针通过并保存成功；确认数据库
中的 `ObjectStorageStagingDirectory` Option 已同步更新，再切回验收目录。把目录改为相对路径或
文件系统根目录，预期前端或后端拒绝且数据库原值不变。

通过浏览器网络面板或已认证请求检查：

```text
GET /api/option/object-storage
GET /api/option
```

在后台保存一组有效对象存储设置，确认只出现一次成功提示，且 `.probe` 临时对象已被删除。临时撤销
业务前缀的 `PutObject` 权限后再次保存，预期只出现一次失败提示，不得同时出现“设置已保存”；提示
中不得包含 AWS RequestID、HostID、IAM ARN、Bucket 名或完整 Object Key。恢复权限后继续验收。

预期：响应不包含 Secret 明文；通用 Option 响应也不包含
`ObjectStorageS3SecretAccessKey`。数据库 `options` 表中该 Option 按确认方案保存明文。再次保存时
Secret 留空，原值应继续有效；输入新 Secret 后应替换原值。

## 4. 同步图片回归

调用原同步接口：

```bash
curl -sS "$BASE_URL/v1/images/generations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"YOUR_IMAGE_MODEL","prompt":"a red apple on a white table","n":1}' | jq
```

预期：响应结构、渠道选择、重试、计费、消费日志和错误行为与发布前一致；该调用不得创建
`async_image_tasks` 记录，也不得产生 S3 或暂存副作用。

## 5. 异步正常链路

记录用户、订阅和 Token 当前额度，然后创建异步任务：

```bash
curl -i "$BASE_URL/v1/images/generations/tasks" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"YOUR_IMAGE_MODEL","prompt":"a red apple on a white table","n":1}'
```

预期 HTTP 202，响应包含非空 `id`、`status=queued`、`output.availability=pending` 和空 `data`。
保存任务 ID，并轮询：

```bash
curl -sS "$BASE_URL/v1/images/generations/tasks/$TASK_ID" \
  -H "Authorization: Bearer $TOKEN" | jq
```

允许观察到的正常状态序列为：

```text
queued/pending -> running/pending -> succeeded/archiving -> succeeded/available
```

中间状态可能因执行速度较快而跳过。最终检查：

- `output.data` 数量等于请求 `n`，每项包含 `index`、`url`、`mime_type` 和 `size_bytes`。
- 使用预签名 URL 能下载并识别为有效图片；Bucket 的普通公开地址不能匿名访问。
- S3 Object Key 为
  `prod/user-files/zmodel@async-images/{user_id}/{yyyy}/{mm}/{task_id}/{index}.{extension}`。
- UTC 年月与首次完成暂存的时间一致，扩展名由实际图片内容决定。
- 用户只结算一次，任务 Root 视图显示 `billing_status=settled`。
- 全部对象可用前，查询接口不得返回任何部分图片地址。

## 6. 来源格式兼容

分别验证 URL、纯 Base64 和 data URI 三种上游响应。可使用支持 `response_format=b64_json` 的真实
渠道验证 Base64，使用模拟渠道验证 data URI。

三种来源最终都应生成私有 S3 对象，用户查询只获得平台生成的预签名 URL。数据库、业务日志和
通知中不得出现原始 Base64、data URI 或上游临时 URL；暂存目录中只能出现规范图片字节。

## 7. 请求和权限边界

逐项执行并记录请求前后额度：

1. `stream=true`：HTTP 400，错误码 `async_image_stream_unsupported`。
2. 清空对象存储配置后提交：HTTP 503，错误码 `object_storage_not_configured`。
3. 让暂存目录不可写后提交：HTTP 503，错误码 `archive_staging_unavailable`。
4. 用 `OTHER_USER_TOKEN` 查询 `$TASK_ID`：HTTP 404，错误码
   `image_generation_task_not_found`。

前三项失败均不得创建可见任务或扣除额度。测试结束后立即恢复配置和目录权限。

## 8. S3 上传失败和人工重传

先用有效权限保存设置，并把最大归档尝试次数临时设为 1。保存成功后撤销验收 IAM 的
`PutObject` 权限，再提交一个异步任务。

等待终态后，在 Root 的“使用日志 → 异步图片任务”中确认：

```text
status=succeeded
output_availability=failed
billing_status=settled
```

同时确认：

- 用户已按生成成功扣费，失败没有触发退款。
- 任务不返回伪造或部分图片地址。
- `{user_id}/{yyyy}/{mm}/{task_id}/{index}.img` 暂存文件仍存在且哈希校验通过。
- Root 收到一条归档失败通知；普通上游生成失败不应发送该通知。

此时在对象存储设置页尝试修改持久暂存目录，预期保存被拒绝，提示仍有异步任务或保留的暂存文件；
原目录和数据库 Option 均不变。完成失败文件重传及暂存清理后，才允许切换目录。

恢复 IAM 权限，勾选任务执行“重新上传选中的文件”。预期任务进入 `archiving` 并最终恢复为
`available`。比较操作前后的渠道请求数和用户额度：不得再次调用上游，不得退款、补扣或重复扣费。
随后准备多个失败任务，验证“重新上传全部失败的图片文件”以及后台操作进度。

若页面将暂存完整性故障显示为不可重试事故，这是正确行为；普通 S3 上传失败必须允许重传。

## 9. 上游生成失败和退款

使用验收专用渠道让上游稳定返回 500，再提交异步任务。预期终态为：

```text
status=failed
output_availability=failed
billing_status=refunded
```

钱包或订阅额度以及 Token 额度应恢复到提交前。继续轮询并等待后台重复扫描，额度不能再次增加。
恢复测试渠道后，不应由“重新上传”操作重新生成该任务。

## 10. 重启恢复

1. 恢复有效对象存储配置，把最大尝试次数设回 3 或以上。
2. 提交任务，生成完成后暂时阻断 S3，使任务进入 `succeeded/archiving`。
3. 记录上游渠道请求次数、用户额度和暂存文件。
4. 正常停止服务，确认进程在 `SHUTDOWN_TIMEOUT_SECONDS` 内排空活跃 Worker。
5. 使用同一数据库和同一暂存卷重新启动，恢复 S3 网络或权限。
6. 轮询至任务进入 `available`。

预期：恢复过程不重新调用上游、不重复结算，任务不会永久停留在 `running` 或 `uploading`。对于
S3 Put 已成功但数据库尚未确认的对象，Worker 应通过 `HeadObject` 元数据和 `LastModified` 恢复，
不覆盖上传且不延长原有效期。

## 11. 到期和清理

把对象保留时间和清理间隔临时设为允许的最小值 60 秒，再生成一个成功任务并保存预签名 URL。
超过对象有效期后检查：

- 查询返回 `output.availability=expired` 和空 `data`，不再签发新 URL。
- 原预签名 URL 超过自身有效期后无法访问。
- 清理任务删除 S3 对象。
- `async_image_tasks` 和 `storage_objects` 数据库记录永久保留，并记录删除状态和时间。

## 12. 页面、语言和发布判定

普通用户进入“使用日志 → 异步图片任务”时只能看到自己的任务。Root 应能使用任务 ID、用户、
模型、生成状态、输出状态、计费状态和时间筛选；列表应展示提交时间、结束时间、渠道、用户、分组、
平台、模型、耗时、对象成功数/总数、尝试次数和脱敏错误，并能执行选中重传与全部失败重传。普通
用户列表不得返回或显示渠道、平台和其他用户信息。

点击页面右上角“查看”，确认可勾选提交时间、结束时间、渠道、用户、分组、平台、模型、耗时、
生成状态、输出状态、计费状态、对象数、尝试次数、暂存完整性和错误等列。任务 ID、预览、详情以及
Root 的批量选择列应固定显示，不能被隐藏；普通用户的菜单不得出现渠道、用户和平台。关闭部分列并
刷新页面，显示选择应保持不变；分别以 Root 和普通用户调整列后，两类用户的选择不得相互覆盖。
恢复浏览器本地存储到初始状态后，默认应显示提交时间、结束时间、用户（仅 Root）、任务 ID、模型、
耗时、生成状态、输出状态、预览和详情，其余可选列默认隐藏。切换列时表头、数据单元格和空结果提示
必须保持对齐。

对一个 `available` 任务点击“预览”，确认只展示图片画廊；点击“详情”，确认详情弹窗展示生成、
计费、归档重试参数、生命周期和逐对象的上传、暂存、删除状态。
每张图片必须显示缩略预览；点击预览可在新窗口打开有效临时 URL，点击下载应得到正确扩展名和
内容的原图文件。详情接口应在点击时按需请求，列表每 15 秒轮询不得批量生成图片签名 URL。

使用普通用户查看自己的详情，确认可以预览和下载，但响应及页面不包含 Bucket、Object Key、ETag、
内部错误或其他用户任务。使用 Root 查看同一任务时应能看到 Token、订阅、渠道、Provider、
Endpoint、Region、Bucket、Object Key、ETag 和内部错误，但仍不得返回 S3 Secret 或持久暂存相对
路径。对 `failed`、`expired` 或已删除对象打开详情，预期保留状态和错误信息，但不显示可用的预览、
下载链接。

依次切换 `en`、`zh`、`fr`、`ru`、`ja`、`vi`，检查对象存储设置页和异步图片任务页不存在缺失
文案、溢出、遮挡或错误回显 Secret。

只有以下条件全部满足时才允许发布：

- 同步图片接口无行为回归。
- URL、Base64、data URI 均可归档。
- 生成成功但归档失败仍只结算一次，并可从暂存文件恢复。
- 上游生成失败只退款一次。
- 用户隔离、Secret 脱敏、私有对象和预签名时限均正确。
- 重启恢复、到期停止签名、S3 删除和数据库永久记录均正确。
- 后端测试、默认前端类型检查/i18n/构建及 classic 前端构建全部通过。

验收失败时至少记录：任务 ID、用户 ID、模型、发生时间、任务三类状态、对象成功数/总数、暂存
完整性、相关脱敏日志和复现步骤。不得在验收记录中粘贴 S3 Secret、渠道 Key、Token Key、原始
Base64、data URI 或带签名的临时 URL。
