# MiniMax H3 (MegabyAI) 视频协议

本文记录平台对 MegabyAI MiniMax H3 上游协议的实际公开合同。上游原始文档位于项目外部，仅作为管理员线下核对模型能力的依据；本文件和代码定义平台客户可依赖的稳定行为。

## 1. 协议边界

- 渠道视频协议固定选择 `minimax-h3(megabyai)`。
- MiniMax 与 Seedance 复用 OpenAI/Sora 异步任务创建、轮询、响应转换和内容下载流程，但使用不同协议标识和参数校验。
- 平台不在运行时调用上游 `GET /v1/models`，不自动同步或直接公开上游动态模型 ID。
- 客户请求使用稳定公共模型名；管理员确认上游变更后，通过模型映射和模型能力配置维护上游 ID。
- 不维护 MiniMax 内置模型名单、时长名单、分辨率名单或画幅名单。

## 2. 渠道模型能力

管理员必须为映射后的每个上游模型 ID 配置：

```json
{
  "video_protocol": "minimax-h3(megabyai)",
  "video_model_capabilities": {
    "minimax-h3": {
      "resolutions": ["1440p"],
      "ratios": ["16:9", "1:1", "9:16", "21:9", "4:3", "3:4"],
      "max_reference_images": 5,
      "max_reference_videos": 0,
      "max_reference_audios": 3,
      "min_duration_seconds": 5,
      "max_duration_seconds": 15,
      "supports_generate_audio": true,
      "generate_audio_required": true,
      "supports_first_frame": true,
      "supports_last_frame": true,
      "last_frame_requires_first_frame": true,
      "reference_images_incompatible_with_frames": true,
      "audio_reference_requires_visual_reference": true
    }
  }
}
```

分辨率、画幅、时长和素材数量是当前上游文档的配置示例，不是代码常量，上游能力变化后由管理员更新配置。7个布尔字段同样由管理员按上游 `/v1/models` 和接口文档维护；新增 MiniMax H3 模型能力时默认全部开启，以符合当前合同，但页面允许调整。后端要求字段显式配置，并拒绝互相矛盾的组合，例如不支持原生音频却要求 `generate_audio=true`，或尾帧依赖首帧却不支持首帧。

## 3. 稳定请求字段

JSON 和 multipart 都只接受以下业务字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `model` | string | 客户公共模型名，转发时替换为模型映射后的上游 ID |
| `prompt` | string | 必填，当前 MiniMax H3 最长 2000 个字符 |
| `duration` | integer | 必填，按模型配置的最小和最大时长校验 |
| `resolution` | string | 必填，按模型配置动态校验 |
| `ratio` | string | 必填，按模型配置动态校验 |
| `referenceImages` | string[] | 普通参考图片 URL |
| `referenceVideos` | string[] | 普通参考视频 URL，是否支持取决于配置 |
| `referenceAudios` | string[] | 普通参考音频 URL |
| `generate_audio` | boolean | 是否生成原生音频，支持性和必填规则取决于配置 |
| `watermark` | boolean | 水印字段 |
| `first_image` | string | 首帧 HTTP/HTTPS URL；multipart 时也可为单个文件 |
| `last_image` | string | 尾帧 HTTP/HTTPS URL；multipart 时也可为单个文件 |
| `provider_options` | object | 仅限 JSON，命名空间固定为 `minimax-h3(megabyai)`，且不能覆盖受保护字段 |

平台不接受 `generation_mode`、`references`、`extra_params` 或上游字段别名。速度或质量档位应由稳定公共模型名和后台模型映射表达。

## 4. 素材组合

URL 必须使用 HTTP 或 HTTPS。multipart 本地文件字段为：

```text
referenceImageFiles
referenceVideoFiles
referenceAudioFiles
first_image
last_image
```

同类 URL 与文件合并计算素材数量。首帧和尾帧各只允许一个 URL 或一个文件，不能同时提交 URL 和文件。其余规则完全读取模型能力配置：

- `last_frame_requires_first_frame`：尾帧是否必须搭配首帧。
- `reference_images_incompatible_with_frames`：普通参考图是否与首尾帧互斥。
- `audio_reference_requires_visual_reference`：参考音频是否必须搭配普通参考图。
- 三类 `max_reference_*`：每类素材的最大总数，`0` 表示不支持。

## 5. 上游请求

适配器保持合法稳定字段的 JSON 类型、multipart 文件和数组顺序，只替换模型 ID并规范化分辨率、画幅和时长。MiniMax 与 Seedance 当前虽然使用相同任务路由，仍分别维护协议逻辑。

平台不会透传客户的 `Idempotency-Key`。每次创建上游任务时固定发送：

```http
Idempotency-Key: zmodel:{public_task_id}
```

这样同一个平台任务的上游提交具有稳定幂等边界，且不同客户不能通过自定义请求头制造键冲突。

## 6. 示例

```json
{
  "model": "public-minimax-h3",
  "prompt": "Create a smooth cinematic transition",
  "duration": 8,
  "resolution": "1440p",
  "ratio": "16:9",
  "generate_audio": true,
  "first_image": "https://media.example.com/start.png",
  "last_image": "https://media.example.com/end.png"
}
```

创建和查询响应继续使用平台公开任务 ID；完成后暴露给客户的视频地址统一为 `/v1/videos/{task_id}/content`。S3 优先、平台代理或跳转上游的选择仍由内容接口按渠道当前交付配置处理。
