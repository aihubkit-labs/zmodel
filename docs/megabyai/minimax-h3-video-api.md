# MegabyAI 渠道视频协议

本文只记录平台对 MegabyAI 的渠道配置和上游适配。面向客户的统一接口合同见
[`视频生成 API`](../openapi/public-video-api.md)。上游原始文档只作为管理员线下核对模型能力的依据。

## 1. 协议边界

- 渠道视频协议固定选择 `megabyai`。
- MiniMax、Seedance 以及后续采用相同任务接口的模型统一使用 `megabyai` 协议；模型差异由后台能力配置表达。
- 平台不在运行时调用上游 `GET /v1/models`，不自动同步或直接公开上游动态模型 ID。
- 客户请求使用稳定公共模型名；管理员确认上游变更后，通过模型映射和模型能力配置维护上游 ID。
- 协议代码不维护模型名单、时长名单、分辨率名单或画幅名单。

## 2. 渠道模型能力

管理员必须为映射后的每个上游模型 ID 配置：

```json
{
  "video_protocol": "megabyai",
  "video_model_capabilities": {
    "minimax-h3": {
      "resolutions": ["1440p"],
      "ratios": ["16:9", "1:1", "9:16", "21:9", "4:3", "3:4"],
      "ratio_required": true,
      "min_reference_images": 0,
      "max_reference_images": 5,
      "min_reference_videos": 0,
      "max_reference_videos": 0,
      "min_reference_audios": 0,
      "max_reference_audios": 3,
      "supports_duration": true,
      "duration_required": true,
      "min_duration_seconds": 5,
      "max_duration_seconds": 15,
      "supports_generate_audio": true,
      "generate_audio_required": true,
      "supports_first_frame": true,
      "first_frame_required": false,
      "supports_last_frame": true,
      "last_frame_required": false,
      "last_frame_requires_first_frame": true,
      "reference_images_incompatible_with_frames": true,
      "audio_reference_requires_visual_reference": true,
      "reference_media_incompatible_with_frames": false,
      "supports_seed": false,
      "supports_watermark": false
    }
  }
}
```

分辨率、画幅、时长和素材数量是当前上游文档的配置示例，不是代码常量，上游能力变化后由管理员更新配置。能力开关同样由管理员按上游 `/v1/models` 和接口文档维护。后端要求字段显式配置，并拒绝互相矛盾的组合，例如不支持原生音频却要求 `generate_audio=true`，或尾帧依赖首帧却不支持首帧。

## 3. 素材组合校验

适配器按统一请求解析素材，同类 URL 与文件合并计算数量。首帧和尾帧各只允许一个 URL 或一个文件，不能同时提交 URL 和文件。其余规则完全读取模型能力配置：

- `last_frame_requires_first_frame`：尾帧是否必须搭配首帧。
- `reference_images_incompatible_with_frames`：普通参考图是否与首尾帧互斥。
- `audio_reference_requires_visual_reference`：参考音频是否必须搭配普通参考图。
- 三类 `max_reference_*`：每类素材的最大总数，`0` 表示不支持。

## 4. 上游请求

适配器保持合法稳定字段的 JSON 类型、multipart 文件和数组顺序，只替换模型 ID并规范化分辨率、画幅和时长。MiniMax 与 Seedance 使用同一渠道协议逻辑。

平台不会透传客户的 `Idempotency-Key`。每次创建上游任务时固定发送：

```http
Idempotency-Key: zmodel:{public_task_id}
```

这样同一个平台任务的上游提交具有稳定幂等边界，且不同客户不能通过自定义请求头制造键冲突。

创建和查询响应由公共响应构造器生成，不直接透传上游响应。公开视频地址只按平台公开任务 ID 动态
构造，S3 优先、平台代理或跳转上游的选择由内容接口在访问时处理。
