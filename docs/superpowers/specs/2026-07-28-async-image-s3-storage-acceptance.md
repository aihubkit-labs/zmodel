# 异步图片生成/编辑与 S3 临时存储验收手册

## 文档关系

- 需求日期：2026-07-28
- 配套设计：[异步图片生成/编辑与 S3 临时存储设计](./2026-07-28-async-image-s3-storage-design.md)
- 验收范围：异步图片生成与编辑任务、JSON/multipart 重放、私有 S3 归档、持久暂存、计费、失败重传、重启恢复、对象清理和后台页面
- 结论记录：验收人应在本文各检查项后记录通过、失败或不适用，并保存关键任务 ID、截图和日志证据

本文是上述设计文档的配套执行手册，两份文档属于同一个需求。设计文档描述约束和实现方案，本文
描述发布前如何验证这些约束。

## 1. 验收环境

故障注入、IAM 权限回收、服务重启和对象过期测试必须在独立验收环境执行，不得直接操作生产环境。

准备以下信息：

- `BASE_URL`：待验收服务地址。
- `TOKEN`：有目标图片模型权限和足够额度的普通用户 Token。
- `OTHER_USER_TOKEN`：另一个用户的 Token，用于验证任务隔离。
- `INPUT_IMAGE`：本地有效图片文件，例如 `./input.png`。
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
`{s3_key_prefix}/user-files/zmodel@async-images/*` 下的 `PutObject`、`GetObject`（包括 `HeadObject`）和
`DeleteObject` 权限。保存设置时使用
`{s3_key_prefix}/user-files/zmodel@async-images/.probe/{random}` 完成写入、Head 和删除探针，不需要开放 Bucket
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
      "Resource": "arn:aws:s3:::<bucket>/<s3_key_prefix>/user-files/zmodel@async-images/*"
    }
  ]
}
```

若保存提示 `cleanup probe failed`，说明写入与 Head 已通过，但缺少 `s3:DeleteObject`；补充上述权限
后重新保存。

## 2. 部署和迁移检查

1. 启动服务，确认启动过程没有数据库迁移错误。
2. 确认数据库存在 `async_image_tasks` 和 `storage_objects` 表，并确认 `async_image_tasks` 已新增
   `request_path`、`request_content_type`、`request_body` 列。`request_body` 在 MySQL、PostgreSQL、
   SQLite 中应分别为 `longblob`、`bytea`、`blob`（允许数据库工具使用等价大小写显示）。
3. 确认后台准备填写的暂存目录位于持久卷，而不是容器临时文件系统。
4. 多节点环境分别进入各节点，确认相同相对路径指向同一文件。
5. 执行自动化验证。根 Go 包通过 `go:embed` 依赖 `web/default/dist` 和 `web/classic/dist`，新 worktree
   中这两个忽略目录通常不存在，所以必须先完成两个前端生产构建，再运行根包或全量 Go 测试：

   ```bash
   cd web/default
   bun install
   bun run typecheck
   bun run i18n:sync
   bun run build
   cd ../classic
   bun install
   bun run build
   cd ../..
   go test ./...
   ```

预期：全部命令通过；默认前端 i18n 检查不存在缺失、额外或未翻译条目。`dist` 和 `node_modules`
均为本地构建产物，不应出现在 Git 变更中。若 Go 测试报 `pattern web/*/dist: no matching files found`，
说明对应前端尚未构建，不是后端测试失败；完成上述生产构建后重新执行 Go 测试。

## 3. 对象存储设置

使用 Root 登录，进入“系统设置 → 对象存储”。确认进入页面后左侧仍是系统设置下钻菜单，并且
“对象存储”菜单可见且处于选中状态。

填写 Endpoint、Region、Bucket、Access Key、Secret Access Key、S3 对象键前缀和持久暂存目录。AWS S3 的
Endpoint 留空；MinIO 等兼容服务填写 HTTP(S) 地址。持久暂存目录填写
`/data/zmodel/async-image-staging`。生产环境前缀填写 `prod`，开发环境填写 `dev`。首次正常链路建议使用：

| 设置 | 验收值 |
| --- | ---: |
| S3 对象键前缀 | `dev`（开发环境）或 `prod`（生产环境） |
| 对象保留时间 | 600 秒 |
| 预签名 URL 有效期 | 60 秒 |
| 单次归档超时 | 30 秒 |
| 最大归档尝试次数 | 3 |
| 最长重试窗口 | 300 秒 |
| 清理间隔 | 60 秒 |

保存时系统会先对持久暂存目录执行创建、写入、`fsync`、原子重命名、读取和删除探针，再执行 S3
随机探针对象的 `Put -> Head -> Delete`。预期保存成功，暂存目录和 Bucket 中均不遗留探针文件。
刷新页面后持久暂存目录和 S3 对象键前缀保持原值；Secret 输入框必须为空，只显示“已配置”状态。

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

保存 `dev` 前缀后提交一个新异步图片任务，确认新对象写入
`dev/user-files/zmodel@async-images/...`，且探针也只使用 `dev` 前缀。将前缀切换为 `prod` 不应触发
Endpoint、Region 或 Bucket 的位置重绑定；切换前已创建对象的完整 Object Key 保持不变，只有新对象
使用 `prod`。

## 4. 同步图片回归

分别调用原同步生成和编辑接口。生成请求：

```bash
curl -sS "$BASE_URL/v1/images/generations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"YOUR_IMAGE_MODEL","prompt":"a red apple on a white table","n":1}' | jq
```

预期：响应结构、渠道选择、重试、计费、消费日志和错误行为与发布前一致；该调用不得创建
`async_image_tasks` 记录，也不得产生 S3 或暂存副作用。

编辑请求：

```bash
curl -sS "$BASE_URL/v1/images/edits" \
  -H "Authorization: Bearer $TOKEN" \
  -F "model=YOUR_IMAGE_EDIT_MODEL" \
  -F "prompt=make the sky blue" \
  -F "image=@$INPUT_IMAGE" \
  -F "n=1" | jq
```

预期：同步编辑的 multipart 文件、响应、渠道选择、重试和计费行为与发布前一致，也不得创建异步任务
或产生异步 S3/暂存副作用。

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

- 响应包含任务 `created_at`；`output.data` 数量等于请求 `n`，每项包含 `index`、`url`、
  `mime_type` 和 `size_bytes`，但不包含 `revised_prompt`。
- 使用预签名 URL 能下载并识别为有效图片；Bucket 的普通公开地址不能匿名访问。
- S3 Object Key 为
  `prod/user-files/zmodel@async-images/{yyyy}/{mm}/{dd}/{task_id}/{index}.{extension}`。
- UTC 年月日与首次完成暂存的时间一致，扩展名由实际图片内容决定。
- 用户只结算一次，任务 Root 视图显示 `billing_status=settled`。
- 全部对象可用前，查询接口不得返回任何部分图片地址。

随后分别验证 multipart 和 JSON 异步编辑。multipart 请求：

```bash
curl -i "$BASE_URL/v1/images/edits/tasks" \
  -H "Authorization: Bearer $TOKEN" \
  -F "model=YOUR_IMAGE_EDIT_MODEL" \
  -F "prompt=make the sky blue" \
  -F "image=@$INPUT_IMAGE" \
  -F "n=1"
```

对支持 JSON 图片引用的渠道，再提交：

```bash
curl -i "$BASE_URL/v1/images/edits/tasks" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"YOUR_IMAGE_EDIT_MODEL","prompt":"make the sky blue","image":"https://YOUR_TEST_HOST/input.png","n":1}'
```

两种请求都应返回 HTTP 202。分别保存编辑任务 ID 并轮询：

```bash
curl -sS "$BASE_URL/v1/images/edits/tasks/$EDIT_TASK_ID" \
  -H "Authorization: Bearer $TOKEN" | jq
```

预期状态序列、私有 S3 对象、签名 URL、计费和不返回部分结果的规则与生成任务相同；实际渠道收到的
路径是 `/v1/images/edits`。multipart 的 Content-Type boundary、文件名、图片字节和可选 mask 必须
完整，JSON 的图片引用与供应商扩展字段不得因异步持久化而丢失。Advanced Custom 渠道只配置同步
`/v1/images/edits` 路由也应能被任务入口选中，不需要额外配置 `/v1/images/edits/tasks`。

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
5. 用 `OTHER_USER_TOKEN` 查询 `$EDIT_TASK_ID`：HTTP 404，错误码
   `image_generation_task_not_found`。
6. 提交超过服务请求体上限的 multipart 图片或蒙版：请求应在创建任务前失败，不得写入截断正文或
   发生预扣费。

前三项及超限请求均不得创建可见任务或扣除额度。测试结束后立即恢复配置和目录权限。

在可暂停 Worker 的验收环境中暂停异步处理，再提交一个带唯一文件名和已知测试字节的编辑任务。
只查询长度、哈希或非敏感元数据，不把原始正文输出到终端或验收记录，并确认：

- `queued` 任务保存 `request_path=/v1/images/edits`、包含 boundary 的 `request_content_type`，且
  `request_body` 非空；`request_payload` 为空。
- 恢复 Worker 后，人为制造可重试的上游错误时，`running/reserved` 任务仍保留相同正文，以便下次
  租约领取重放。
- 成功结算进入 `settled` 或最终失败退款进入 `refunded` 后，`request_body` 与 `request_payload`
  均为空；来源获取或暂存事故走成功结算终态时也必须为空。
- 历史生成任务仅有 `request_payload` 时仍能执行，且继续使用 `/v1/images/generations` 和
  `application/json`。

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
- `{user_id}/{yyyy}/{mm}/{dd}/{task_id}/{index}.img` 暂存文件仍存在且哈希校验通过。
- Root 收到一条归档失败通知；普通上游生成失败不应发送该通知。

此时在对象存储设置页尝试修改持久暂存目录，预期保存被拒绝，提示仍有异步任务或保留的暂存文件；
原目录和数据库 Option 均不变。完成失败文件重传及暂存清理后，才允许切换目录。

恢复 IAM 权限，勾选任务执行“重新上传选中的文件”。预期任务进入 `archiving` 并最终恢复为
`available`。比较操作前后的渠道请求数和用户额度：不得再次调用上游，不得退款、补扣或重复扣费。
随后准备多个失败任务，验证“重新上传全部失败的图片文件”以及后台操作进度。

若页面将暂存完整性故障显示为不可重试事故，这是正确行为；普通 S3 上传失败必须允许重传。

## 9. 上游执行失败和退款

使用验收专用渠道让上游稳定返回 500，分别提交异步生成和编辑任务。两者预期终态均为：

```text
status=failed
output_availability=failed
billing_status=refunded
```

钱包或订阅额度以及 Token 额度应恢复到提交前。继续轮询并等待后台重复扫描，额度不能再次增加。
恢复测试渠道后，不应由“重新上传”操作重新生成该任务。

在“使用日志”中使用任务 ID 搜索，应精确找到一条类型为“退款”的日志，并确认：

- 请求 ID 与任务 ID 一致；
- 详情包含脱敏失败原因、错误码、模型、用户、分组、实际选中的渠道、退回额度和耗时；
- 没有选中渠道就失败时，渠道可以为空，但必须显示“无可用渠道”等具体失败原因；
- 继续等待 Worker 重复扫描后，同一任务 ID 仍只有这一条退款日志。

生成成功的消费日志和生成失败的退款/错误日志中，“路径”都应显示用户实际调用的任务入口：生成
任务为 `/v1/images/generations/tasks`，编辑任务为 `/v1/images/edits/tasks`。不得显示 Worker
内部用于渠道匹配的 `/v1/images/generations` 或 `/v1/images/edits`；旧任务没有 `request_path` 时
仍应显示生成任务入口。

## 10. 重启恢复

1. 恢复有效对象存储配置，把最大尝试次数设回 3 或以上。
2. 分别提交生成任务和 multipart 编辑任务，生成或编辑完成后暂时阻断 S3，使任务进入
   `succeeded/archiving`。
3. 记录上游渠道请求次数、用户额度和暂存文件。
4. 正常停止服务，确认进程在 `SHUTDOWN_TIMEOUT_SECONDS` 内排空活跃 Worker。
5. 使用同一数据库和同一暂存卷重新启动，恢复 S3 网络或权限。
6. 轮询至任务进入 `available`。

预期：恢复过程不重新调用上游、不重复结算，任务不会永久停留在 `running` 或 `uploading`。编辑任务
在上游调用完成前重启时仍可从数据库恢复原 Content-Type 与 multipart 正文；完成结算后正文已经清空，
归档恢复只读取持久暂存文件。对于 S3 Put 已成功但数据库尚未确认的对象，Worker 应通过
`HeadObject` 元数据和 `LastModified` 恢复，
不覆盖上传且不延长原有效期。

## 11. 到期和清理

把对象保留时间和清理间隔临时设为允许的最小值 60 秒，再生成一个成功任务并保存预签名 URL。
超过对象有效期后检查：

- 查询返回 `output.availability=expired` 和空 `data`，不再签发新 URL。
- 原预签名 URL 超过自身有效期后无法访问。
- 清理任务删除 S3 对象。
- `async_image_tasks` 和 `storage_objects` 数据库记录永久保留，并记录删除状态和时间。

## 12. 页面、语言和发布判定

普通用户进入“使用日志 → 异步图片任务”时只能看到自己的生成和编辑任务。Root 应能使用任务 ID、用户、
模型、生成状态、输出状态、计费状态和时间筛选；列表应展示提交时间、结束时间、渠道、用户、分组、
平台、模型、耗时、对象成功数/总数、尝试次数和脱敏错误，并能执行选中重传与全部失败重传。普通
用户列表不得返回或显示渠道、平台和其他用户信息。

点击页面右上角“查看”，确认可勾选提交时间、结束时间、渠道、用户、分组、平台、模型、耗时、
生成状态、输出状态、计费状态、对象数、尝试次数、暂存完整性和错误等列。任务 ID、详情以及
Root 的批量选择列应固定显示，不能被隐藏；普通用户的菜单不得出现渠道、用户和平台。关闭部分列并
刷新页面，显示选择应保持不变；分别以 Root 和普通用户调整列后，两类用户的选择不得相互覆盖。
恢复浏览器本地存储到初始状态后，Root 默认应显示提交时间、结束时间、用户、任务 ID、分组、渠道、
模型、耗时、生成状态、输出状态和详情；普通用户默认显示其中有权访问的列，其余可选列默认隐藏。
分组与渠道必须在列表中相邻。切换列时表头、数据单元格和空结果提示
必须保持对齐。

生成状态、上传状态和计费状态筛选器未选择时，应分别显示对应的条件名称作为 placeholder，不得显示
内部值 `all`。展开下拉菜单和选中任一状态后，确认“全部”及所有状态值均按当前界面语言显示；切换
语言后文案应同步变化，提交给列表接口的查询参数仍为原有英文枚举。

列表中的任务 ID 必须完整展示，不得使用省略号截断。输出可用性在页面中使用面向用户的“上传状态”
名称，其中 `available` 显示为“已上传”。点击任意任务的“详情”，确认弹窗立即打开并发起任务详情
请求，不得出现按钮点击后无反应的情况。
Root 列表的用户列应只显示用户名，不得拼接 `(#用户ID)`；缺少用户名时应显示带“用户 ID”标签的
回退值。详情弹窗应展示用户名和渠道名称，不展示用户 ID、渠道 ID 或 Token ID。

对一个 `available` 任务点击“详情”，确认详情弹窗同时展示图片画廊、预览、下载、生成、计费、归档
重试参数、生命周期和逐对象的上传、暂存、删除状态；单图对象详情应占满可用宽度。宽屏下左栏连续
展示生成详情与计费，右栏连续展示归档处理与生命周期，不得因区块高度差产生大块空白；请求参数跨
整行展示。
每张图片必须显示缩略预览；点击预览可在新窗口打开有效临时 URL，点击下载应得到正确扩展名和
内容的原图文件。详情接口应在点击时按需请求，列表每 15 秒轮询不得批量生成图片签名 URL。

使用普通用户查看自己的详情，确认可以预览和下载，但响应及页面不包含 Bucket、Object Key、ETag、
内部错误或其他用户任务。使用 Root 查看同一任务时，详情接口应返回 Token、订阅、渠道、Provider、
Endpoint、Region、Bucket、Object Key、ETag 和内部错误等审计字段，但界面不展示用户 ID、渠道 ID
或 Token ID；接口仍不得返回 S3 Secret 或持久暂存相对路径。请求参数应显示提交时安全快照中的操作
类型、模型、Prompt、数量和图片控制参数，不得包含图片、蒙版、
用户标识、任意扩展字段、Base64 或 data URI。改造前已清空原始请求的历史任务应显示“历史任务未
保留请求参数”，不得显示含糊的 `-`。对 `failed`、`expired` 或已删除对象打开详情，预期保留状态和错误信息，但不显示可用的预览、
下载链接。

通过普通用户任务 GET、用户/Root 列表接口和用户/Root 详情接口检查编辑任务响应。所有响应都不得
包含 `request_body`、`request_payload`、原始 Content-Type、multipart boundary、上传文件名、输入
图片/蒙版字节或可识别的 Base64/data URI 片段；数据库慢查询或调试日志也不得打印这些字段。列表
轮询应只加载任务摘要，不因在途编辑任务携带大正文而显著增加响应大小或数据库读取量。

依次切换 `en`、`zh`、`fr`、`ru`、`ja`、`vi`，检查对象存储设置页和异步图片任务页不存在缺失
文案、溢出、遮挡或错误回显 Secret。

只有以下条件全部满足时才允许发布：

- 同步图片生成和编辑接口无行为回归。
- 异步编辑的 JSON、multipart、Worker 重放、Advanced Custom 路径映射和编辑任务 GET 均正确。
- URL、Base64、data URI 均可归档。
- 生成成功但归档失败仍只结算一次，并可从暂存文件恢复。
- 上游生成失败只退款一次。
- 编辑请求正文只在排队/可重试执行阶段保留，任何结算或退款终态均已清空，且所有 API/列表查询不
  返回或加载原始正文。
- 用户隔离、Secret 脱敏、私有对象和预签名时限均正确。
- 重启恢复、到期停止签名、S3 删除和数据库永久记录均正确。
- 后端测试、默认前端类型检查/i18n/构建及 classic 前端构建全部通过。

验收失败时至少记录：任务 ID、用户 ID、模型、发生时间、任务三类状态、对象成功数/总数、暂存
完整性、相关脱敏日志和复现步骤。不得在验收记录中粘贴 S3 Secret、渠道 Key、Token Key、原始
Base64、data URI 或带签名的临时 URL。
