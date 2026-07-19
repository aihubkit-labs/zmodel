# 阶梯媒体计费设计

## 状态

- 日期：2026-07-16
- 范围：图片和异步视频计费
- 表达式版本：`v2`
- 现有合法 `v1` 价格表达式：保持不变

## 1. 背景

当前计费表达式系统主要围绕 Token 定价设计。例如，表达式
`p * 2.5 + c * 10` 返回以「每百万 Token 美元价格」为单位的值，结算路径使用
以下公式进行转换：

```text
quota = expression_result / 1,000,000 * QuotaPerUnit * group_ratio
```

该方式适合 Token 计费，但不便于表达媒体固定价格。目前要表示每张生成图片
`$0.096`，必须在表达式中写成 `96000`。虽然数学结果正确，但不直观且容易配置
错误。另外，通过 `param()` 直接读取 `n`、`seconds` 等数量参数，会使未经校验的
请求数据直接参与计费计算。

需要支持以下行为：

- 根据标准化后的图片尺寸、分辨率或质量档位选择价格。
- 按成功生成的图片数量计费。
- 根据标准化后的视频分辨率或质量档位选择价格。
- 某个视频档位按每次生成视频计费。
- 另一个视频档位按生成时长或请求时长计费。
- 不同模型可以使用不同的表达式和档位定义。
- 正确处理预扣费、异步结算、退款、日志和分组倍率。

## 2. 目标

1. 在计费表达式中加入明确的固定美元费用。
2. 只向计费计算暴露经过校验和标准化的媒体维度。
3. 在同一个表达式中支持按份、按秒、按 Token 以及混合公式。
4. 从预扣费到最终结算，全程保留模型表达式和计费维度快照。
5. 继续以 `OriginModelName` 为计费配置键，支持任意数量模型使用独立定价规则。
6. 生成足够结构化的元数据，使定价页面和使用日志可以展示命中的档位、计费单位、
   单价、数量和总费用。
7. 保持所有返回有限非负价格的现有 `v1` 表达式语义不变；负数、NaN 或无穷结果
   按计费安全约束拒绝执行。

## 3. 非目标

- 自动信任任意请求字段并将其作为计费乘数。
- 自动理解未来所有供应商特有的参数。
- 使用新的表达式引擎替换 `expr-lang`。
- 在本次迭代中实现完整的强类型货币 DSL。
- 在表达式中支持多币种。表达式价格仍以美元为单位，展示时继续使用现有的显示货币
  转换机制。

## 4. 选定方案

引入计费表达式 `v2`，新增以下能力：

- `usd(amount)`：将明确的美元金额转换为表达式内部兼容微美元的值。
- 可信数值维度：`units`、`seconds`、`width` 和 `height`。
- 可信标准化字符串维度：`quality`、`resolution_tier` 和
  `image_size_tier`。
- 保留 `v1` 中现有的 Token 变量和函数。

`v2` 保持当前最终转换公式不变。在表达式内部：

```text
usd(amount) = amount * 1,000,000
```

因此，无需修改结算公式即可组合 Token 费用和固定费用：

```text
p * 2.5 + c * 10 + usd(0.02 * units)
```

经过现有的除以一百万转换后，该表达式表示：输入 Token `$2.50/1M`、输出 Token
`$10/1M`，再加上每个生成单位 `$0.02`。

## 5. 表达式语义

### 5.1 变量

| 变量 | 类型 | 含义 |
| --- | --- | --- |
| `p` | 数字 | 按现有分类排除规则处理后的可计费输入 Token |
| `c` | 数字 | 按现有分类排除规则处理后的可计费输出 Token |
| `len` | 数字 | 用于 Token 阶梯条件的完整输入上下文长度 |
| `cr`、`cc`、`cc1h` | 数字 | 缓存 Token 维度 |
| `img`、`img_o` | 数字 | 图片输入和输出 Token |
| `ai`、`ao` | 数字 | 音频输入和输出 Token |
| `units` | 数字 | 已校验的可计费输出数量 |
| `seconds` | 数字 | 每个输出对应的已校验可计费时长 |
| `width` | 数字 | 标准化后的输出宽度；不可用时为零 |
| `height` | 数字 | 标准化后的输出高度；不可用时为零 |
| `quality` | 字符串 | 标准化后的图片质量标识；不可用时为空字符串 |
| `resolution_tier` | 字符串 | 标准化后的视频分辨率档位 |
| `image_size_tier` | 字符串 | 标准化后的图片尺寸档位 |

`units` 和 `seconds` 是数量而不是价格。非 Token 公式中的价格必须明确写在
`usd()` 内。

### 5.2 函数

`v2` 保留全部 `v1` 函数，并新增：

```text
usd(amount) -> number
```

当 `usd()` 收到负数、NaN 或无穷值时，表达式求值必须失败。实现可以在注册函数
内部校验，也可以在求值器的校验边界统一处理。表达式最终结果同样必须是有限的
非负数。

### 5.3 图片示例

根据标准化后的图片尺寸和实际生成数量定价：

```text
v2:image_size_tier == "1K"
  ? tier("1K", usd(0.05 * units))
  : image_size_tier == "4K"
    ? tier("4K", usd(0.15 * units))
    : tier("2K", usd(0.125 * units))
```

根据质量定价：

```text
v2:quality == "high"
  ? tier("high", usd(0.20 * units))
  : tier("standard", usd(0.08 * units))
```

### 5.4 视频示例

标准清晰度按每个生成视频计费，高清档按秒计费：

```text
v2:resolution_tier == "standard"
  ? tier("standard", usd(0.10 * units))
  : tier("hd", usd(0.025 * seconds * units))
```

三个分辨率档位使用不同的每秒价格：

```text
v2:resolution_tier == "1080p"
  ? tier("1080p", usd(0.04 * seconds * units))
  : resolution_tier == "720p"
    ? tier("720p", usd(0.025 * seconds * units))
    : tier("480p", usd(0.015 * seconds * units))
```

固定生成费用加时长费用：

```text
v2:tier("4k", usd((0.05 + 0.04 * seconds) * units))
```

最低计费时长：

```text
v2:tier("hd", usd(0.025 * max(seconds, 5) * units))
```

## 6. 可信计费维度

### 6.1 数据模型

在计费层增加共享的媒体计费上下文：

```go
type BillingDimensions struct {
    Units          float64 `json:"units"`
    Seconds        float64 `json:"seconds"`
    Width          float64 `json:"width"`
    Height         float64 `json:"height"`
    Quality        string  `json:"quality"`
    ResolutionTier string  `json:"resolution_tier"`
    ImageSizeTier  string  `json:"image_size_tier"`
    ImageSize      string  `json:"image_size"`
}
```

实际 Go 字段可以在合适的位置使用整数类型，但表达式环境接收的数值必须与
`expr-lang` 兼容。

`TokenParams` 继续负责 Token 维度。媒体维度单独传递，使 Token 标准化和媒体
标准化保持为两个独立概念。

### 6.2 数据来源和信任边界

只有完成请求解析和校验后才能生成计费维度：

```text
JSON / multipart / 元数据 / 供应商字段
    -> 请求 DTO 或任务适配器
    -> 校验并应用默认值
    -> 按供应商和模型进行标准化
    -> BillingDimensions
    -> 表达式引擎
```

表达式不得使用原始 `param("n")`、`param("seconds")` 或类似的元数据路径作为
计费乘数。`param()` 仍可用于不涉及数量的请求条件，但所有影响费用的用户可控数量
都必须使用可信计费维度。

异步任务如果使用 `param()` 或 `header()`，只持久化表达式中以字面量引用的具体值，
不保存完整请求正文。`Authorization`、`Cookie`、`Proxy-Authorization`、`X-Api-Key`
等认证头不得作为异步计费条件；动态路径或动态请求头引用也会被拒绝。冻结的请求
条件快照上限为 64 KiB。

### 6.3 校验

- 图片数量标准化为至少 `1`，并以 `dto.MaxImageN` 为上限。
- 视频时长必须为正数，并以 `relaycommon.MaxTaskDurationSeconds` 为上限。
- 宽度和高度在参与乘法或档位分类之前，必须完成正数校验和上限校验。
- 未知质量、尺寸或分辨率不得静默落入最便宜的档位。
- 供应商元数据和透传字段必须接受与标准 DTO 字段相同的校验。
- 从上游响应或媒体元数据中取得的实际值同样不可信，结算前必须进行校验。

不支持的值必须返回 HTTP 400，或者通过明确的供应商或模型标准化规则进行映射。
回退行为必须在配置和测试中可见。

### 6.4 标准化

不同供应商使用不同的字段和命名，因此标准化必须感知适配器。示例：

```text
1024x1024, provider "1k"                 -> image_size_tier "1K"
1536x1024, 1024x1536, provider "2k"      -> image_size_tier "2K"
3840x2160, provider "4k"                 -> image_size_tier "4K"
1920x1080, "1080P", "full_hd"           -> resolution_tier "1080p"
1280x720, "HD"                           -> resolution_tier "720p"
```

Frimodel 定制版的 `image_size_tier(size)` 使用最长边分类：最长边不超过 1024
为 `1K`，不超过 2048 为 `2K`，其余为 `4K`；缺失、`auto` 或无法识别的非数字
尺寸使用其上游兼容兜底值 `2K`。该映射属于供应商兼容规则，不应解释为所有图片
模型的通用尺寸定义。

`resolution_tier` 与 `image_size_tier` 虽然都可能包含 `4K`，但不能合并为同一个
计费变量：前者来自视频请求的 `resolution`，后者来自图片请求的 `size`，两者由
不同适配器校验，支持范围和回退规则也不同。当前 `quality` 由图片请求填充，因此
前端将其展示为“图片质量”；若未来视频模型提供独立质量参数，应新增明确的标准化
契约，而不是复用含义不清的字段。

通用标准化规则应放在共享媒体计费工具中，供应商特有别名则保留在对应适配器中，
避免全局函数不断积累协议特有逻辑。

## 7. 按模型配置

无需新增全局价格表。继续按照模型名称存储计费模式和表达式：

```json
{
  "image-model-a": "tiered_expr",
  "video-model-a": "tiered_expr"
}
```

```json
{
  "image-model-a": "v2:...image expression...",
  "video-model-a": "v2:...video expression..."
}
```

规则选择必须使用 `OriginModelName`。上游模型映射不得改变面向用户的计费约定。

每个模型可以定义不同的可识别档位、价格和公式。适配器负责提供标准化维度，模型
表达式负责决定这些维度如何影响价格。

## 8. 计费生命周期

### 8.1 图片预扣费

1. 解析并校验图片请求。
2. 根据标准化后的请求构造预估计费维度。
3. 将 `units` 设置为已校验的请求数量。
4. 使用冻结的模型表达式求值。
5. 使用现有配额公式和分组倍率进行转换。
6. 在 `RelayInfo` 中保存表达式快照和预估计费维度。

### 8.2 图片结算

1. 如果上游响应提供了可靠数据，则确定已确认成功生成的图片数量。
2. 使用 `dto.MaxImageN` 校验实际数量。
3. 使用实际数量替换 `units`。
4. 使用冻结的表达式和冻结的请求分类重新求值。
5. 结算与预扣费之间的差额。

如果上游没有提供可靠的实际数量，则结算使用已校验的请求数量。对于流式请求和
客户端断开连接，必须保留现有语义：客户端断开连接不自动表示上游生成了更少的
可计费图片。

### 8.3 视频预扣费

1. 任务适配器校验并标准化请求。
2. 适配器返回预估 `BillingDimensions`，而不再仅返回乘数倍率。
3. 使用请求数量、时长和标准化后的分辨率或质量，对模型的冻结表达式求值。
4. 预扣计算出的配额。
5. 将完整计费快照持久化到任务私有计费上下文中。

### 8.4 视频完成结算

1. 任务成功完成后，适配器提取所有可靠的实际数量、时长和输出分类。
2. 使用相同的安全上限校验实际计费维度。
3. 将实际维度覆盖到冻结的预估维度上；缺失的实际字段继续使用预估值。
4. 使用冻结的表达式和分组倍率重新求值。
5. 使用现有任务差额结算路径补扣或退还配额。
6. 任务失败时，使用现有全额退款路径。

适配器必须明确声明供应商是按请求时长还是实际输出时长计费。表达式只接收最终
选定的可信 `seconds` 值，不负责决定哪个数据源具有权威性。

## 9. 快照和持久化

扩展用于同步媒体结算的 `billingexpr.BillingSnapshot`，加入：

- 预估可信计费维度。
- 表达式结果和命中的档位。
- 对审计有帮助时，记录维度来源元数据。

扩展用于异步任务的 `model.TaskBillingContext`：

```go
BillingMode         string
ExprString          string
ExprHash            string
ExprVersion         int
GroupRatio          float64
EstimatedDimensions BillingDimensions
EstimatedTier       string
```

任务必须始终使用提交时冻结的表达式和分组倍率进行结算。后续配置变更只影响新任务。

不包含这些字段的现有任务记录继续使用当前的 `ModelPrice`、`ModelRatio` 和
`OtherRatios` 行为。

## 10. 后端集成

### 10.1 表达式引擎

更新 `pkg/billingexpr`：

- 注册 `v2` 编译环境。
- 增加 `usd()` 和可信媒体维度变量。
- 在运行和结算输入中接收媒体计费维度。
- 校验表达式结果必须为有限非负数。
- 保持 `v1` 编译和转换行为不变。
- 继续使用带检查的配额转换和配额饱和审计机制。

### 10.2 图片中继

更新图片请求和定价路径：

- 根据 `dto.ImageRequest` 构造预估媒体计费维度。
- 将计费维度传入阶梯预扣费流程。
- 独立于固定价格的 `PriceData.UsePrice` 行为记录实际生成数量。
- 使用实际 `units` 重新执行阶梯结算。

### 10.3 任务中继

扩展任务适配器计费契约。可采用以下兼容形式：

```go
EstimateBillingDimensions(c *gin.Context, info *RelayInfo) BillingDimensions
AdjustBillingDimensionsOnSubmit(info *RelayInfo, taskData []byte) *BillingDimensions
AdjustBillingDimensionsOnComplete(task *model.Task, result *TaskInfo) *BillingDimensions
```

迁移期间可保留现有倍率方法。阶梯表达式任务使用计费维度，旧版固定价格和倍率任务
继续使用 `OtherRatios`。

更新任务提交和轮询流程：

- 当模型计费模式为 `tiered_expr` 时使用阶梯表达式。
- 持久化表达式快照和计费维度。
- 在适配器配额覆盖和 Token 回退逻辑之前，结算表达式定价任务。
- 不要将表达式定价任务无条件标记为 `PerCallBilling`，因为这类任务可能需要在完成时
  根据时长或数量进行差额结算。

### 10.4 API 调用示例

以下示例均可直接执行。首次使用只需安装 `curl`、`jq`，然后修改 API Key；本地测试
和线上调用只需切换 `AIHUBKIT_BASE_URL`：

```bash
export AIHUBKIT_BASE_URL='http://localhost:3000'
# 线上地址：export AIHUBKIT_BASE_URL='https://apimodel.aihubkit.com'
export AIHUBKIT_API_KEY='替换为你的 API Key'
export IMAGE_MODEL='gpt-image-2-high'
```

不要把真实 API Key 写入文档、脚本仓库或聊天记录。若密钥已经公开，应立即删除并
重新创建。

#### 10.4.1 生成单张图片并保存 Base64 响应

下面的命令生成一张 `1024x1024` 高质量 PNG，并保存为当前目录下的
`gpt-image-2-high_1k.png`。`curl` 失败时命令会直接退出，不会继续触发二次 `jq`
错误：

```bash
response="$({ curl --fail-with-body --silent --show-error \
  "${AIHUBKIT_BASE_URL}/v1/images/generations" \
  -H "Authorization: Bearer ${AIHUBKIT_API_KEY}" \
  -H "Content-Type: application/json" \
  -d "$(jq -n \
    --arg model "$IMAGE_MODEL" \
    --arg prompt '一只坐在窗边的猫' \
    '{
      model: $model,
      prompt: $prompt,
      size: "1024x1024",
      quality: "high",
      response_format: "b64_json",
      output_format: "png",
      n: 1
    }')"; } 2>&1)" || {
  printf '%s\n' "$response" >&2
  exit 1
}

printf '%s' "$response" |
  jq -er '.data[0].b64_json' |
  base64 -d > "${IMAGE_MODEL}_1k.png"
```

`output_format` 可改为 `png`、`jpeg` 或 `webp`，文件扩展名也应同步修改。该字段、
`quality` 的具体可选值最终取决于上游模型协议；平台会透传这些字段，但不会把某个
图片供应商的可选值强制套用到所有模型。

`gpt-image-2-high` 可以是平台对外暴露的模型别名。若上游实际模型名为
`gpt-image-2`，应在渠道中配置模型映射：

```json
{
  "gpt-image-2-high": "gpt-image-2"
}
```

计费配置仍以用户请求的 `OriginModelName`（此处为 `gpt-image-2-high`）为键；模型映射
只改变发送给上游的模型名。该请求的 `size` 会标准化为 `image_size_tier = "1K"`，
`n` 会标准化为 `units = 1`。响应兼容 OpenAI 图片格式，图片位于 `data[].url` 或
`data[].b64_json`。

#### 10.4.2 覆盖 1K、2K、4K 图片尺寸

只需修改 `IMAGE_SIZE` 和输出文件名。以下三个命令分别覆盖当前 Frimodel 兼容映射
中的 `1K`、`2K`、`4K` 档位：

```bash
for spec in \
  '1024x1024:1k' \
  '2048x2048:2k' \
  '3840x2160:4k'
do
  IMAGE_SIZE="${spec%%:*}"
  FILE_TIER="${spec##*:}"
  response="$({ curl --fail-with-body --silent --show-error \
    "${AIHUBKIT_BASE_URL}/v1/images/generations" \
    -H "Authorization: Bearer ${AIHUBKIT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d "$(jq -n \
      --arg model "$IMAGE_MODEL" \
      --arg size "$IMAGE_SIZE" \
      --arg prompt '一只坐在窗边的猫，写实摄影' \
      '{
        model: $model,
        prompt: $prompt,
        size: $size,
        quality: "high",
        response_format: "b64_json",
        n: 1
      }')"; } 2>&1)" || {
    printf '%s\n' "$response" >&2
    exit 1
  }
  printf '%s' "$response" |
    jq -er '.data[0].b64_json' |
    base64 -d > "${IMAGE_MODEL}_${FILE_TIER}.png"
done
```

数值尺寸按最长边分类：不超过 `1024` 为 `1K`，不超过 `2048` 为 `2K`，其余合法
尺寸为 `4K`。这只是本次接入使用的上游兼容分类；上游是否接受某个精确尺寸仍由
模型协议决定。

#### 10.4.3 保存 URL 响应

当上游返回 `data[].url` 时，可以直接下载：

```bash
response="$({ curl --fail-with-body --silent --show-error \
  "${AIHUBKIT_BASE_URL}/v1/images/generations" \
  -H "Authorization: Bearer ${AIHUBKIT_API_KEY}" \
  -H "Content-Type: application/json" \
  -d "$(jq -n \
    --arg model "$IMAGE_MODEL" \
    --arg prompt '一只坐在窗边的猫' \
    '{
      model: $model,
      prompt: $prompt,
      size: "1024x1024",
      response_format: "url",
      n: 1
    }')"; } 2>&1)" || {
  printf '%s\n' "$response" >&2
  exit 1
}

image_url="$(printf '%s' "$response" | jq -er '.data[0].url')"
curl --fail-with-body --location --show-error \
  "$image_url" \
  --output "${IMAGE_MODEL}_url.png"
```

#### 10.4.4 一次请求多张图片

下面请求 `n = 3`，并把响应数组中实际返回的所有 Base64 图片依次保存。保存数量以
`data` 数组实际长度为准：

```bash
response="$({ curl --fail-with-body --silent --show-error \
  "${AIHUBKIT_BASE_URL}/v1/images/generations" \
  -H "Authorization: Bearer ${AIHUBKIT_API_KEY}" \
  -H "Content-Type: application/json" \
  -d "$(jq -n \
    --arg model "$IMAGE_MODEL" \
    --arg prompt '一只坐在窗边的猫，每张构图不同' \
    '{
      model: $model,
      prompt: $prompt,
      size: "1024x1024",
      response_format: "b64_json",
      n: 3
    }')"; } 2>&1)" || {
  printf '%s\n' "$response" >&2
  exit 1
}

count="$(printf '%s' "$response" | jq -er '.data | length')"
[ "$count" -gt 0 ] || {
  printf '响应中没有图片\n' >&2
  exit 1
}

index=1
while [ "$index" -le "$count" ]; do
  printf '%s' "$response" |
    jq -er ".data[$((index - 1))].b64_json" |
    base64 -d > "${IMAGE_MODEL}_1k_${index}.png"
  index=$((index + 1))
done
```

请求中的 `n` 用于预扣费估算；非流式 OpenAI 兼容图片响应存在可靠 `data` 数组时，
最终结算使用实际返回数量。例如请求 `n = 3`、上游只返回 1 张时，系统按 1 张结算
并退还另外 2 张的预扣差额。部分上游虽然接受 `n`，但始终只生成 1 张，这不是平台
丢失图片。

#### 10.4.5 上游忽略 `n` 时可靠生成多张

需要确保得到 3 张图片时，使用三次 `n = 1` 的串行请求：

```bash
index=1
while [ "$index" -le 3 ]; do
  prompt="一只坐在窗边的猫，第 ${index} 张，构图不同"
  response="$({ curl --fail-with-body --silent --show-error \
    "${AIHUBKIT_BASE_URL}/v1/images/generations" \
    -H "Authorization: Bearer ${AIHUBKIT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d "$(jq -n \
      --arg model "$IMAGE_MODEL" \
      --arg prompt "$prompt" \
      '{
        model: $model,
        prompt: $prompt,
        size: "1024x1024",
        response_format: "b64_json",
        n: 1
      }')"; } 2>&1)" || {
    printf '%s\n' "$response" >&2
    exit 1
  }
  printf '%s' "$response" |
    jq -er '.data[0].b64_json' |
    base64 -d > "${IMAGE_MODEL}_serial_${index}.png"
  index=$((index + 1))
done
```

#### 10.4.6 Seedance 兼容视频请求范围

当前 Sora/Seedance 兼容适配器对以下对外模型执行严格校验：

| 模型 | 可选分辨率 | 时长 |
| --- | --- | --- |
| `videos-mini` | `480p`、`720p` | 4–15 秒 |
| `videos-fast` | `480p`、`720p` | 4–15 秒 |
| `videos-standard` | `480p`、`720p`、`1080p`、`4K` | 4–15 秒 |
| `videos-4-mini` | `480p`、`720p` | 4–15 秒 |
| `videos-4-fast` | `480p`、`720p` | 4–15 秒 |
| `videos-4` | `480p`、`720p` | 4–15 秒 |

下面的变量可覆盖六个模型和全部已支持分辨率。修改变量后直接执行提交命令即可：

```bash
export VIDEO_MODEL='videos-standard'
export VIDEO_RESOLUTION='1080p'
export VIDEO_DURATION='10'

# 其他有效组合：
# VIDEO_MODEL=videos-mini     VIDEO_RESOLUTION=480p
# VIDEO_MODEL=videos-mini     VIDEO_RESOLUTION=720p
# VIDEO_MODEL=videos-fast     VIDEO_RESOLUTION=480p
# VIDEO_MODEL=videos-fast     VIDEO_RESOLUTION=720p
# VIDEO_MODEL=videos-standard VIDEO_RESOLUTION=480p
# VIDEO_MODEL=videos-standard VIDEO_RESOLUTION=720p
# VIDEO_MODEL=videos-standard VIDEO_RESOLUTION=1080p
# VIDEO_MODEL=videos-standard VIDEO_RESOLUTION=4K
# VIDEO_MODEL=videos-4-mini   VIDEO_RESOLUTION=480p
# VIDEO_MODEL=videos-4-mini   VIDEO_RESOLUTION=720p
# VIDEO_MODEL=videos-4-fast   VIDEO_RESOLUTION=480p
# VIDEO_MODEL=videos-4-fast   VIDEO_RESOLUTION=720p
# VIDEO_MODEL=videos-4        VIDEO_RESOLUTION=480p
# VIDEO_MODEL=videos-4        VIDEO_RESOLUTION=720p
```

#### 10.4.7 提交、轮询并下载视频

以下完整脚本提交任务，读取公开任务 ID，轮询到完成，并下载 MP4：

```bash
submit_response="$({ curl --fail-with-body --silent --show-error \
  "${AIHUBKIT_BASE_URL}/v1/videos" \
  -H "Authorization: Bearer ${AIHUBKIT_API_KEY}" \
  -H "Content-Type: application/json" \
  -d "$(jq -n \
    --arg model "$VIDEO_MODEL" \
    --arg resolution "$VIDEO_RESOLUTION" \
    --arg prompt '海边日落时缓慢移动的电影镜头' \
    --argjson duration "$VIDEO_DURATION" \
    '{
      model: $model,
      prompt: $prompt,
      resolution: $resolution,
      duration: $duration
    }')"; } 2>&1)" || {
  printf '%s\n' "$submit_response" >&2
  exit 1
}

VIDEO_ID="$(printf '%s' "$submit_response" | jq -er '.id // .task_id')"
printf 'video id: %s\n' "$VIDEO_ID"

while :; do
  status_response="$({ curl --fail-with-body --silent --show-error \
    "${AIHUBKIT_BASE_URL}/v1/videos/${VIDEO_ID}" \
    -H "Authorization: Bearer ${AIHUBKIT_API_KEY}"; } 2>&1)" || {
    printf '%s\n' "$status_response" >&2
    exit 1
  }
  status="$(printf '%s' "$status_response" | jq -er '.status')"
  progress="$(printf '%s' "$status_response" | jq -r '.progress // 0')"
  printf 'status=%s progress=%s%%\n' "$status" "$progress"
  case "$status" in
    completed)
      break
      ;;
    failed)
      printf '%s\n' "$status_response" | jq . >&2
      exit 1
      ;;
    queued|in_progress)
      sleep 5
      ;;
    *)
      printf '未知任务状态: %s\n' "$status" >&2
      exit 1
      ;;
  esac
done

curl --fail-with-body --location --show-error \
  "${AIHUBKIT_BASE_URL}/v1/videos/${VIDEO_ID}/content" \
  -H "Authorization: Bearer ${AIHUBKIT_API_KEY}" \
  --output "${VIDEO_MODEL}_${VIDEO_RESOLUTION}_${VIDEO_DURATION}s.mp4"
```

视频预扣费使用请求中校验后的 `resolution` 和 `duration`；任务完成后，如上游返回
可靠的实际时长或分辨率，则使用提交时冻结的表达式进行差额结算。

#### 10.4.8 常见错误排查

- `401`：API Key 缺失、仍为示例值、已失效，或没有访问权限。
- `400`：模型的分辨率或时长不在上表范围内，或者请求 JSON 无效。
- `503 No available compatible accounts`：当前没有支持该模型的可用渠道账号，不是
  阶梯表达式计算错误。
- `503 system disk overloaded`：节点磁盘使用率超过保护阈值，需要先释放磁盘空间。
- 请求失败时只有错误日志，不会生成消费日志。要把失败请求持久化到使用日志，部署
  后端时必须设置 `ERROR_LOG_ENABLED=true` 并重启后端。

#### 10.4.9 计费配置示例

每张或每个输出固定价格：

```text
v2:tier("base", usd(0.05 * units))
```

视频每秒价格：

```text
v2:tier("base", usd(0.025 * seconds * units))
```

每个视频固定费用 `$0.05`，再加每秒 `$0.04`：

```text
v2:tier("base", usd((0.05 + 0.04 * seconds) * units))
```

图片尺寸阶梯：

```text
v2:image_size_tier == "1K"
  ? tier("1K", usd(0.05 * units))
  : image_size_tier == "4K"
    ? tier("4K", usd(0.15 * units))
    : tier("2K", usd(0.125 * units))
```

最后一个分支是无条件兜底，不是对其档位名称的再次判断。上例中凡是没有命中
`1K`、`4K` 的值都会展示为并按 `2K` 计费；因此兜底档价格必须按模型标准化规则
谨慎设置，不能默认认为它只会接收到标签为 `2K` 的请求。

最终客户价格为表达式价格乘以实际使用分组倍率。例如基础价格 `$0.05 / 个`，分组
倍率为 `1.5` 时，客户价格为 `$0.075 / 个`。表达式本身不需要手工乘分组倍率，
系统会在预扣费和最终结算时统一应用并在价格页面展示。

#### 10.4.10 支持边界

`v2` 表达式可以按任意模型名称保存，但某个媒体变量能否用于计费，取决于该请求
路径是否已经提供可信、经过校验和标准化的维度：

- OpenAI 兼容图片生成路径提供 `units`、`quality`、`image_size`、
  `image_size_tier`，并在可靠响应中按实际图片数量结算。
- 当前 Sora/Seedance 兼容适配器为 `videos-mini`、`videos-fast`、
  `videos-standard`、`videos-4-mini`、`videos-4-fast`、`videos-4` 提供经过
  白名单校验的 `resolution_tier` 和 `seconds`。
- 其他视频适配器不能因为可以保存 `v2` 表达式，就被视为已支持上述视频维度。
  接入前必须为该适配器定义字段来源、默认值、合法范围及完成时实际值提取规则。
- 固定价格、模型倍率、`v1` Token 表达式以及不含 `v2` 快照的历史任务继续走原有
  计费路径，不受媒体编辑器配置影响。

#### 10.4.11 新视频厂商接入方式

媒体编辑器与厂商、渠道类型和模型名称解耦。新视频厂商只要能够把计费参数标准化为
现有可信维度，就不需要修改媒体编辑器：

- `units`：经过校验的生成视频数量。
- `seconds`：每个输出对应的经过校验的计费时长。
- `resolution_tier`：经过厂商和模型规则标准化的分辨率档位。

现有媒体编辑器可以直接表达：

- 按个计费。
- 按秒计费。
- 每个输出固定费用加每秒费用。
- 根据 `resolution_tier` 为不同分辨率设置不同档位。
- 为每个对外模型保存独立价格，并在结算时应用分组倍率。

例如，新厂商的 `1080p` 模型请求经适配器校验后产生：

```go
billingexpr.BillingDimensions{
    Units:          1,
    Seconds:        10,
    ResolutionTier: "1080p",
}
```

管理员即可使用现有媒体编辑器配置：

```text
v2:resolution_tier == "1080p"
  ? tier("1080p", usd(0.04 * seconds * units))
  : tier("default", usd(0.02 * seconds * units))
```

新厂商接入通常不只是增加一个文件，还必须完成以下后端契约：

1. 新增或扩展任务适配器，并注册对应渠道类型和任务平台。
2. 在 `ValidateRequestAndSetAction()` 中解析并校验该厂商的模型、分辨率、时长、
   数量和必填字段。
3. 在 `EstimateBillingDimensions()` 中返回可信的预估媒体维度。
4. 在 `BuildRequestBody()` 中转换成厂商实际要求的请求字段。
5. 在 `ParseTaskResult()` 中解析任务状态和厂商返回的实际媒体信息。
6. 在 `AdjustBillingDimensionsOnComplete()` 中校验并返回可靠的实际结算维度；缺失的
   实际值继续使用提交时冻结的预估值。
7. 添加合法值、非法值、上下边界、请求转换、预扣费、完成结算和失败退款测试。
8. 最后在管理后台为对外模型启用 `tiered_expr` 并配置价格。

分辨率白名单、时长范围、字段别名和默认值属于供应商协议契约，目前定义在对应
适配器代码中，不属于管理员价格配置。价格可以由管理员调整，但协议校验不能由普通
价格配置绕过。尚未实现可信媒体维度的适配器默认返回空维度；如果表达式引用
`seconds` 或 `resolution_tier`，请求会失败，而不会使用未经校验的原始参数计费。

出现以下新计费规则时，现有媒体编辑器可能需要扩展：

- 按帧数、帧率、总像素、宽高比或音频开关计费。
- 文生视频和图生视频使用不同价格，但适配器尚未提供对应可信维度。
- 按时长区间使用非线性价格，而不是简单的每秒价格。
- 按质量、优先级、并发等级或其他厂商专有参数分档。
- 一次生成多个视频时存在无法用 `units` 表达的特殊计费规则。

如果现有可信维度已经足够，只是可视化编辑器没有对应控件，可以先使用原始表达式
编辑器；如果表达式引擎也没有所需的可信维度，则必须同时扩展适配器、表达式环境、
媒体编辑器、日志展示和回归测试，不能直接通过 `param()` 读取用户输入作为计费乘数。

## 11. 前端设计

### 11.1 编辑器

可视化编辑器除 Token 价格外，还应支持媒体档位字段：

- 档位条件：质量、分辨率档位、图片尺寸档位、时长范围。
- 计费方式：按份、按秒、固定费用加每秒费用，或者高级原始表达式。
- 美元单价输入。
- 生成的 `v2` 表达式预览。

无法由可视化编辑器表示的公式仍可使用原始表达式编辑。

### 11.2 定价展示

标准表达式结构应展示为易于理解的计费单位：

| 档位 | 计费方式 | 价格 |
| --- | --- | --- |
| 1K | 按图片 | `$0.050 / 张` |
| 2K | 按图片 | `$0.125 / 张` |
| 4K | 按图片 | `$0.150 / 张` |
| standard | 按个 | `$0.100 / 个` |
| HD | 按时长 | `$0.025 / 秒` |

混合公式可以展示多个费用组成，例如：

```text
$0.050 / 个 + $0.040 / 秒
```

如果解析器无法可靠地结构化高级表达式，界面应显示原始表达式，而不是展示错误的
价格表。

### 11.3 日志

使用日志和任务日志应包含：

- 计费模式和表达式版本。
- 命中的档位。
- 预估和实际可信计费维度。
- 可以从表达式结构中确定时，记录单价和计费单位。
- 预扣配额、实际配额和结算差额。
- 在现有仅管理员可见的审计位置记录配额饱和元数据。

## 12. 错误处理和安全性

- 表达式编译失败时禁止保存配置。
- 冒烟测试除 Token 向量外，还必须覆盖媒体计费维度向量。
- 拒绝负数、NaN 和无穷价格。
- 无效或超出范围的数量和时长必须在预扣费前返回 HTTP 400。
- 未知媒体档位不得静默获得更低价格。
- 只有模型契约明确允许时，表达式引用的缺失维度才可以使用安全中性默认值；否则
  校验必须失败。
- 所有最终转换均使用现有带检查的配额辅助函数。
- 对于金额很大但仍合法的费用，预扣费必须因配额不足而失败，不得溢出为负数或
  更小金额。
- 结算发生错误时，回退到冻结的预扣金额并记录带请求关联信息的警告，与现有阶梯
  结算策略保持一致。

## 13. 兼容性和迁移

1. 返回有限非负价格的 `v1` 表达式保持语义兼容；负数、NaN 或无穷结果会被拒绝，
   防止负扣费或溢出。
2. 不带版本前缀的表达式继续表示 `v1`。
3. 新媒体表达式保存时必须包含明确的 `v2:` 前缀。
4. 现有固定价格和倍率模型保持不变。
5. 不包含 v2 快照的现有任务继续使用当前计费上下文。
6. 上游价格同步必须保留 v2 表达式，不得将其展平为固定价格或倍率配置。

如果新增任务快照字段继续存放在现有 JSON 私有数据中，则无需执行数据库特有的
结构变更。未来如需新增数据库列，迁移必须同时支持 SQLite、MySQL 和 PostgreSQL。

## 14. 测试策略

### 表达式引擎

- `usd(0.096)` 精确转换为预期配额。
- Token 费用和美元费用可以正确组合。
- 所有可信媒体变量仅在 `v2` 中可用。
- 现有 `v1` 表达式返回结果不变。
- 拒绝负数和非有限结果。

### 图片计费

- 1K、2K 和 4K 输入选择正确档位。
- 请求 `n` 受上限约束并用于预扣费。
- 实际成功生成的图片数量会调整最终结算。
- 缺少实际数量时回退到请求数量。
- multipart 和 JSON 请求得到相同的标准化结果。
- 未知图片尺寸不会静默落入最便宜档位。

### 视频计费

- 标准清晰度按每个输出单位计费一次。
- 高清档按照单价乘以秒数再乘以输出数量计费。
- 不同模型可以为相同标准化档位配置不同公式。
- 请求时长受 `MaxTaskDurationSeconds` 上限约束。
- 供应商元数据无法绕过时长校验。
- 供应商计费契约要求使用实际值时，完成结算使用实际值；否则使用冻结的预估值。
- 失败任务全额退还预扣费。
- 配置变更后，已提交任务的表达式和分组倍率仍保持冻结。

### 前端

- 可视化配置能够生成预期的 v2 表达式。
- 标准按份和按秒表达式可以解析为正确的价格表。
- 高级表达式回退为原始文本展示。
- 所有新增界面文本均覆盖前端 i18n 语言文件。

## 15. 实施顺序

1. 增加 v2 表达式类型、`usd()`、可信计费维度和表达式引擎测试。
2. 增加共享的图片和视频标准化及校验契约。
3. 集成图片预扣费和基于实际数量的结算。
4. 扩展任务计费快照和适配器计费维度钩子。
5. 集成视频预扣费、完成结算和退款。
6. 更新日志和配额饱和审计数据。
7. 更新可视化编辑器、价格明细、费用估算器和 i18n。
8. 运行针对性后端测试、受影响 Go 包的完整测试、前端类型检查、代码检查和生产构建。

## 16. 验收标准

满足以下全部条件时，该设计视为完成：

1. 管理员可以为多个图片和视频模型配置独立的 v2 表达式。
2. 图片档位可以按照每张成功生成图片的实际美元金额计费。
3. 同一个视频表达式可以让一个档位按每次生成视频计费，让另一个档位按秒计费。
4. 受支持适配器使用的 JSON、multipart、metadata 和供应商特有字段，在参与计费前
   均被标准化为可信计费维度。
5. 不支持的值会被拒绝或显式映射，绝不静默使用最便宜的价格。
6. 预扣费和最终结算使用冻结的表达式和分组倍率。
7. 现有 v1 Token 表达式和旧版计费模式不受影响。
8. 对于标准表达式结构，定价页面和日志能够展示命中的档位及易于理解的计费单位。
