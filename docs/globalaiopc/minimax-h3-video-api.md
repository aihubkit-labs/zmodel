# GlobalAiOpc 渠道视频协议

本文只记录渠道配置和上游适配。面向客户的统一接口合同见
[`视频生成 API`](../openapi/public-video-api.md)。

## 渠道配置

- 渠道类型使用 OpenAI 或 Sora。
- Base URL 可配置为 `https://zcbservice.aizfw.cn` 或 `https://zcbservice.aizfw.cn/kyyReactApiServer`；适配器会规范化路径，避免重复追加 `/kyyReactApiServer`。
- 视频协议选择 `globalaiopc`。
- 公共模型名通过现有模型映射绑定到上游模型 ID，例如 `minimax-h3`、`wan2.7-r2v` 或 `seedance-2.5-c1`。
- 同一协议支持该上游采用相同任务端点和响应结构的所有视频模型，不再为每个模型新增协议代码。
- 模型能力在后台配置，不运行时调用上游模型目录改变客户合同。

## 上游转换

创建任务：

```text
POST {base_url}/kyyReactApiServer/v2/model-center/tasks
```

查询任务：

```text
GET {base_url}/kyyReactApiServer/v2/model-center/tasks/{upstream_task_id}
```

请求字段转换：

| 平台字段 | 上游字段 | 说明 |
| --- | --- | --- |
| `ratio` | `aspect_ratio` | 平台名称保持统一 |
| `referenceImages` | `reference_images` | 保持数组顺序 |
| `referenceVideos` | `reference_videos` | 当前模型能力建议配置上限为0 |
| `referenceAudios` | `reference_audios` | 按模型能力校验数量和组合 |
| `resolution` | `resolution` | 按模板或后台配置的 `resolution_mappings` 转换，例如 `1440p` 到 `2k` |
| `generate_audio` | `generate_audio` | 模型能力支持且客户显式传入时发送 |
| `seed` | `seed` | 模型能力支持且客户显式传入时发送 |
| `watermark` | `watermark` | 模型能力支持且客户显式传入时发送 |
| `size` | 不发送 | 避免与 `aspect_ratio` 冲突 |

该协议仅支持 `application/json` 和上游可访问的 HTTP/HTTPS 素材 URL。multipart 请求在平台侧返回 `unsupported_content_type`。

上游状态转换为平台状态：`queued` 保持 `queued`，`processing` 转为 `in_progress`，`completed` 转为 `completed`，`failed` 转为 `failed`。自动 S3 归档直接使用任务完成轮询当次返回的 `result_url`，缺失时回退 `video_url`。平台不会把该地址持久化为后续交付源，平台 `/content` 地址也不入库，而是根据公开任务 ID实时生成；访问 `/content` 或后台手动上传 S3 时，都使用上游任务 ID和任务密钥实时查询任务详情，取得当前有效地址。

## 能力模板

管理端内置当前上游文档对应的能力模板：

```text
minimax-h3                 MiniMax-H3-c4
hh-1.1-r2v-o               hh-1.1-i2v-o
hh-1.1-t2v-o               wan2.7-r2v
wan2.7-i2v                 wan2.7-t2v
wan2.7-videoedit           KlingO3
seedance-2.5-c1            seedance-2.5-c3
sd_2.0_fast_special        sd_2.0_special
sd_2.0_discount            sd_2.0_fast_discount
videos_933_c1              videos_fast_933_c1
videos_stable              videos_stable_fast
```

选择上游模型 ID 后会自动应用精确匹配模板；管理员也可复制相似模板、批量应用当前渠道中所有匹配模型，或将调整后的能力保存为自定义模板。模板只提供配置草稿，不是协议代码中的模型名单，也不参与公共模型路由。上游能力变化后由管理员修改渠道能力或模板。
