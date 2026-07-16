# Tiered Media Billing Design

## Status

- Date: 2026-07-16
- Scope: image and asynchronous video billing
- Expression version: `v2`
- Existing `v1` behavior: unchanged

## 1. Background

The current billing expression system is designed around token pricing. An
expression such as `p * 2.5 + c * 10` returns a value denominated in
USD-per-million-token units, and the settlement path converts it with:

```text
quota = expression_result / 1,000,000 * QuotaPerUnit * group_ratio
```

This works well for token billing, but fixed media prices are awkward. A price
of `$0.096` per generated image must currently be represented as `96000`, which
is correct mathematically but unclear and error-prone. Reading quantities such
as `n` or `seconds` directly through `param()` also makes unvalidated request
data part of the billing calculation.

The required behavior includes:

- Image prices selected by normalized size, resolution, or quality tiers.
- Image charges multiplied by the number of successfully generated images.
- Video prices selected by normalized resolution or quality tiers.
- A video tier charged once per generated video.
- Another video tier charged by generated or requested duration.
- Different expressions and tier definitions for different models.
- Correct pre-consume, asynchronous settlement, refund, logging, and group
  ratio behavior.

## 2. Goals

1. Add explicit fixed-USD charges to billing expressions.
2. Expose only validated, normalized media dimensions to billing arithmetic.
3. Support per-unit, per-second, token, and mixed formulas in one expression.
4. Preserve a model's expression and dimensions from pre-consume through final
   settlement.
5. Support independent pricing rules for any number of models by continuing to
   key billing configuration by `OriginModelName`.
6. Produce structured enough metadata for pricing pages and usage logs to show
   the matched tier, unit, unit price, quantity, and total charge.
7. Preserve all existing `v1` expressions without semantic changes.

## 3. Non-Goals

- Automatically trusting arbitrary request fields as billing multipliers.
- Automatically understanding every future provider-specific parameter.
- Replacing `expr-lang` with a new expression engine.
- Implementing a fully typed monetary DSL in this iteration.
- Supporting multiple currencies inside expressions. Expression prices remain
  USD and use the existing display-currency conversion at presentation time.

## 4. Chosen Approach

Introduce billing expression `v2` with:

- `usd(amount)`: converts an explicit USD amount into the expression's internal
  micro-USD-compatible value.
- Trusted numeric dimensions: `units`, `seconds`, `width`, and `height`.
- Trusted normalized string dimensions: `quality`, `resolution_tier`, and
  `image_size_tier`.
- The existing token variables and functions from `v1`.

`v2` retains the current final conversion formula. Internally:

```text
usd(amount) = amount * 1,000,000
```

Therefore token and fixed charges compose without changing settlement math:

```text
p * 2.5 + c * 10 + usd(0.02 * units)
```

After the existing division by one million, this means `$2.50/1M` input
tokens, `$10/1M` output tokens, plus `$0.02` per generated unit.

## 5. Expression Semantics

### 5.1 Variables

| Variable | Type | Meaning |
| --- | --- | --- |
| `p` | number | Billable input tokens after existing category exclusion |
| `c` | number | Billable output tokens after existing category exclusion |
| `len` | number | Full input context length for token tier conditions |
| `cr`, `cc`, `cc1h` | number | Cache token dimensions |
| `img`, `img_o` | number | Image input and output tokens |
| `ai`, `ao` | number | Audio input and output tokens |
| `units` | number | Validated billable output count |
| `seconds` | number | Validated billable duration per output |
| `width` | number | Normalized output width, or zero when unavailable |
| `height` | number | Normalized output height, or zero when unavailable |
| `quality` | string | Normalized quality label, or empty when unavailable |
| `resolution_tier` | string | Normalized video resolution tier |
| `image_size_tier` | string | Normalized image size tier |

`units` and `seconds` are quantities, not prices. Prices must be explicit inside
`usd()` when the formula is not token-based.

### 5.2 Functions

`v2` keeps all `v1` functions and adds:

```text
usd(amount) -> number
```

Calling `usd()` with a negative, NaN, or infinite amount must make expression
evaluation fail; the implementation may enforce this inside the registered
function or in the evaluator's validation boundary. The expression's final
result must also be finite and non-negative.

### 5.3 Image Examples

Price by normalized image size and actual generated count:

```text
v2:image_size_tier == "4K"
  ? tier("4K", usd(0.128 * units))
  : image_size_tier == "2K"
    ? tier("2K", usd(0.096 * units))
    : tier("1K", usd(0.096 * units))
```

Price by quality:

```text
v2:quality == "high"
  ? tier("high", usd(0.20 * units))
  : tier("standard", usd(0.08 * units))
```

### 5.4 Video Examples

Standard resolution charged per generated video, HD charged per second:

```text
v2:resolution_tier == "standard"
  ? tier("standard", usd(0.10 * units))
  : tier("hd", usd(0.025 * seconds * units))
```

Three resolution tiers with different per-second prices:

```text
v2:resolution_tier == "1080p"
  ? tier("1080p", usd(0.04 * seconds * units))
  : resolution_tier == "720p"
    ? tier("720p", usd(0.025 * seconds * units))
    : tier("480p", usd(0.015 * seconds * units))
```

Fixed creation fee plus duration charge:

```text
v2:tier("4k", usd((0.05 + 0.04 * seconds) * units))
```

Minimum billable duration:

```text
v2:tier("hd", usd(0.025 * max(seconds, 5) * units))
```

## 6. Trusted Billing Dimensions

### 6.1 Data Model

Add a shared media billing context owned by the billing layer:

```go
type BillingDimensions struct {
    Units          float64 `json:"units"`
    Seconds        float64 `json:"seconds"`
    Width          float64 `json:"width"`
    Height         float64 `json:"height"`
    Quality        string  `json:"quality"`
    ResolutionTier string  `json:"resolution_tier"`
    ImageSizeTier  string  `json:"image_size_tier"`
}
```

The exact Go field types may use integers where appropriate, but the expression
environment receives numeric values compatible with `expr-lang`.

`TokenParams` remains responsible for token dimensions. Media dimensions are
passed separately so token normalization and media normalization remain
independent concepts.

### 6.2 Sources and Trust Boundary

Dimensions are produced only after request parsing and validation:

```text
JSON / multipart / metadata / provider fields
    -> request DTO or task adaptor
    -> validation and defaults
    -> provider/model normalization
    -> BillingDimensions
    -> expression engine
```

The expression must not use raw `param("n")`, `param("seconds")`, or equivalent
metadata paths as billing multipliers. `param()` remains available for
non-quantity request conditions, but all user-controlled quantities that affect
cost use trusted dimensions.

### 6.3 Validation

- Image count is normalized to at least `1` and bounded by `dto.MaxImageN`.
- Video duration is positive and bounded by
  `relaycommon.MaxTaskDurationSeconds`.
- Width and height are positive and bounded before multiplication or tier
  classification.
- Unknown quality, size, or resolution values do not silently map to the
  cheapest tier.
- Provider metadata and passthrough fields receive the same checks as standard
  DTO fields.
- Actual values obtained from upstream responses or media metadata are also
  treated as untrusted and validated before settlement.

An unsupported value must either be rejected with HTTP 400 or mapped by an
explicit provider/model normalization rule. The fallback behavior must be
visible in configuration and tests.

### 6.4 Normalization

Normalization is adapter-aware because providers use different fields and
vocabularies. Examples include:

```text
1024x1024, 1024*1024, provider "square_hd" -> image_size_tier "1K"
2048x2048, provider "2k"                 -> image_size_tier "2K"
1920x1080, "1080P", "full_hd"           -> resolution_tier "1080p"
1280x720, "HD"                           -> resolution_tier "720p"
```

Common normalization rules should live in shared media billing utilities.
Provider-specific aliases remain in the relevant adaptor. This avoids a global
function accumulating protocol-specific behavior.

## 7. Per-Model Configuration

No new global price table is required. Continue storing billing mode and
expression by model name:

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

Rule selection must use `OriginModelName`. Upstream model mapping must not
change the user-facing billing contract.

Each model may define different recognized tiers, prices, and formulas. The
adaptor supplies normalized dimensions; the model expression decides how those
dimensions affect price.

## 8. Billing Lifecycle

### 8.1 Image Pre-Consume

1. Parse and validate the image request.
2. Build estimated dimensions from the normalized request.
3. Set `units` to the validated requested count.
4. Evaluate the frozen model expression.
5. Convert with the existing quota formula and group ratio.
6. Store the expression snapshot and estimated dimensions on `RelayInfo`.

### 8.2 Image Settlement

1. Determine the number of confirmed generated images from the upstream
   response when reliable.
2. Validate the actual count against `dto.MaxImageN`.
3. Replace `units` with the actual count.
4. Re-evaluate the frozen expression using the frozen request classification.
5. Settle the difference from pre-consume.

If the upstream does not expose a reliable count, settlement uses the validated
requested count. Streaming/client-disconnect semantics must preserve the
existing rule that a client disconnect does not automatically imply that the
upstream generated fewer billable images.

### 8.3 Video Pre-Consume

1. The task adaptor validates and normalizes the request.
2. It returns estimated `BillingDimensions`, not only multiplicative ratios.
3. Evaluate the model's frozen expression using requested count, duration, and
   normalized resolution/quality.
4. Pre-consume the resulting quota.
5. Persist the complete billing snapshot in the task private billing context.

### 8.4 Video Completion Settlement

1. On successful task completion, the adaptor extracts any reliable actual
   units, duration, and output classification.
2. Validate actual dimensions using the same safety bounds.
3. Merge actual dimensions over the frozen estimates. Missing actual fields
   retain their estimated values.
4. Re-evaluate the frozen expression and group ratio.
5. Use the existing task delta-settlement path to supplement or refund quota.
6. On task failure, use the existing full refund path.

The adaptor must explicitly define whether a provider bills requested duration
or actual output duration. The expression sees only the selected trusted
`seconds` value and does not decide which source is authoritative.

## 9. Snapshot and Persistence

Extend `billingexpr.BillingSnapshot` for synchronous media settlement with:

- Estimated trusted dimensions.
- Expression result and matched tier.
- Dimension source metadata where useful for audit.

Extend `model.TaskBillingContext` for asynchronous tasks with:

```go
BillingMode         string
ExprString          string
ExprHash            string
ExprVersion         int
GroupRatio          float64
EstimatedDimensions BillingDimensions
EstimatedTier       string
```

The task must always settle against the expression and group ratio frozen at
submission time. Later configuration changes affect new tasks only.

Existing task rows without these fields continue through the current
`ModelPrice`/`ModelRatio`/`OtherRatios` behavior.

## 10. Backend Integration

### 10.1 Expression Engine

Update `pkg/billingexpr` to:

- Register the `v2` compile environment.
- Add `usd()` and trusted media variables.
- Accept media dimensions in run and settlement inputs.
- Validate finite, non-negative expression results.
- Keep `v1` compile and conversion behavior unchanged.
- Continue using checked quota conversion and quota saturation auditing.

### 10.2 Image Relay

Update the image request/pricing path to:

- Build estimated media dimensions from `dto.ImageRequest`.
- Pass them into tiered pre-consume.
- Record actual generated count independently of fixed-price
  `PriceData.UsePrice` behavior.
- Re-run tiered settlement with actual `units`.

### 10.3 Task Relay

Evolve the task adaptor billing contract. A compatible shape is:

```go
EstimateBillingDimensions(c *gin.Context, info *RelayInfo) BillingDimensions
AdjustBillingDimensionsOnSubmit(info *RelayInfo, taskData []byte) *BillingDimensions
AdjustBillingDimensionsOnComplete(task *model.Task, result *TaskInfo) *BillingDimensions
```

Existing ratio methods can remain during migration. Tiered-expression tasks use
dimensions; legacy fixed-price and ratio tasks continue using `OtherRatios`.

Update task submission and polling to:

- Use the tiered expression when the model mode is `tiered_expr`.
- Persist the expression snapshot and dimensions.
- Settle expression-priced tasks before adaptor quota overrides and token
  fallback logic.
- Avoid marking expression-priced tasks as unconditional `PerCallBilling`,
  because they may require completion-time duration or unit reconciliation.

## 11. Frontend Design

### 11.1 Editor

The visual editor should support media tier fields in addition to token prices:

- Tier condition: quality, resolution tier, image size tier, seconds range.
- Charge type: per unit, per second, fixed plus per second, or advanced raw
  expression.
- USD unit price inputs.
- Generated `v2` expression preview.

Raw expression editing remains available for formulas that cannot be represented
by the visual editor.

### 11.2 Pricing Display

Standard expression shapes should render as meaningful units:

| Tier | Billing method | Price |
| --- | --- | --- |
| 1K | Per image | `$0.096 / image` |
| 4K | Per image | `$0.128 / image` |
| standard | Per video | `$0.100 / video` |
| HD | By duration | `$0.025 / second` |

Mixed formulas may display multiple components, for example:

```text
$0.050 / video + $0.040 / second
```

If the parser cannot safely structure an advanced expression, the UI displays
the raw expression rather than presenting an incorrect price table.

### 11.3 Logs

Usage and task logs should include:

- Billing mode and expression version.
- Matched tier.
- Estimated and actual trusted dimensions.
- Unit price and charge unit when structurally known.
- Pre-consumed quota, actual quota, and settlement delta.
- Quota saturation metadata under the existing admin-only audit location.

## 12. Error Handling and Safety

- Expression compilation failures block configuration save.
- Smoke tests cover media dimension vectors as well as token vectors.
- Negative, NaN, and infinite prices are rejected.
- Invalid or out-of-range quantity and duration values return HTTP 400 before
  pre-consume.
- Unknown media tiers cannot silently receive a cheaper price.
- Missing dimensions referenced by an expression use safe neutral defaults only
  where the model contract permits them; otherwise validation fails.
- All final conversions use the existing checked quota helpers.
- Pre-consume must fail with insufficient quota for an oversized valid charge;
  it must never wrap into a negative or smaller charge.
- Settlement errors fall back to the frozen pre-consumed amount and emit a
  correlated warning, matching the existing tiered settlement policy.

## 13. Compatibility and Migration

1. `v1` expressions remain byte-for-byte and semantically compatible.
2. Unprefixed expressions continue to mean `v1`.
3. New media expressions are saved with an explicit `v2:` prefix.
4. Existing fixed-price and ratio models are unchanged.
5. Existing tasks without a v2 snapshot use their current billing context.
6. Upstream pricing sync must preserve v2 expressions and must not flatten them
   into fixed-price or ratio entries.

No database-specific schema change is required if the additional task snapshot
fields remain inside the existing JSON private data. Any future column migration
must support SQLite, MySQL, and PostgreSQL.

## 14. Testing Strategy

### Expression engine

- `usd(0.096)` converts to exactly the expected quota.
- Token and USD components combine correctly.
- Every trusted variable is available only in `v2`.
- Existing `v1` expressions return unchanged results.
- Negative and non-finite results are rejected.

### Image billing

- 1K, 2K, and 4K inputs select the correct tier.
- Requested `n` is bounded and used for pre-consume.
- Actual successful image count adjusts settlement.
- Missing actual count falls back to requested count.
- Multipart and JSON requests normalize equivalently.
- Unknown image size does not fall into the cheapest tier silently.

### Video billing

- Standard resolution charges once per unit.
- HD charges unit price times seconds times units.
- Different models can assign different formulas to the same normalized tier.
- Requested duration is bounded by `MaxTaskDurationSeconds`.
- Provider metadata cannot bypass duration validation.
- Completion settlement uses actual values when the provider contract requires
  them and frozen estimates otherwise.
- Failed tasks refund the full pre-consumed amount.
- Expressions and group ratios remain frozen after configuration changes.

### Frontend

- Visual configurations generate the expected v2 expressions.
- Standard per-unit and per-second expressions parse into correct price tables.
- Advanced expressions fall back to raw display.
- All new UI text is covered by frontend i18n locales.

## 15. Implementation Order

1. Add v2 expression types, `usd()`, trusted dimensions, and engine tests.
2. Add shared image/video normalization and validation contracts.
3. Integrate image pre-consume and actual-count settlement.
4. Extend task billing snapshots and adaptor dimension hooks.
5. Integrate video pre-consume, completion settlement, and refunds.
6. Update logs and quota saturation audit data.
7. Update the visual editor, pricing breakdown, estimator, and i18n.
8. Run focused backend tests, full affected Go package tests, frontend typecheck,
   lint, and production build.

## 16. Acceptance Criteria

The design is complete when all of the following are true:

1. An administrator can configure independent v2 expressions for multiple
   image and video models.
2. Image tiers can charge a real USD amount per successfully generated image.
3. A video expression can charge one tier per generated video and another tier
   per second.
4. JSON, multipart, metadata, and provider-specific fields used by supported
   adaptors are normalized into trusted dimensions before billing.
5. Unsupported values are rejected or explicitly mapped, never silently priced
   at the cheapest tier.
6. Pre-consume and final settlement use a frozen expression and group ratio.
7. Existing v1 token expressions and legacy billing modes are unaffected.
8. Pricing pages and logs show the matched tier and understandable billing
   units for standard expression shapes.
