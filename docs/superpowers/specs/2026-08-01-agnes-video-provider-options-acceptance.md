# Agnes 视频协议验收手册

## 文档关系

- 需求日期：2026-08-01
- 任务 slug：`agnes-video-provider-options`
- 开发分支：`codex/agnes-video-provider-options`
- 远端基线：`origin/main`
- 基线提交：`2013f642b48f38c623a5955c64a5d655cc11a2a3`
- 配套设计：[Agnes 视频协议适配设计](./2026-08-01-agnes-video-provider-options-design.md)
- Agnes 参考：<https://www.agnes-ai.com/zh-Hans/docs/agnes-video-v20>

本文中的请求通过平台 `/v1/videos` 发起，不直接请求 Agnes。成功创建任务会产生真实上游费用。

## 1. 验收准备

命令使用 `http://127.0.0.1:3000` 作为平台地址。替换以下占位符：

- `<TOKEN>`：平台访问令牌，不是 Agnes Key。
- `<AGNES_PUBLIC_MODEL>`：平台配置的 Agnes 公开模型名称。
- `<TASK_ID>`：创建接口返回的 `id` 或 `task_id`。

在管理员“渠道”页面编辑 Agnes 对应的 OpenAI/Sora 渠道：

```text
高级设置 → 渠道额外设置 → 视频协议 → Agnes Video V2
```

同时确认 Base URL、Key、公开模型、模型映射和分组可用。无需配置参数模式或供应商命名空间。

### 用户请求参数速查

Agnes 视频统一使用24fps。调用接口前按下面的范围填写整数 `duration`：

| `resolution` | 可填写的 `duration` | 默认值 | 最长视频 | 平台发送的最大 `num_frames` |
| --- | --- | --- | --- | --- |
| `480p` | 1–18 | — | 18秒 | 433 |
| `720p` | 1–18 | `resolution` 未传时默认使用 | 18秒 | 433 |
| `1080p` | 1–10 | — | 10秒 | 241 |

其他基础约定：

- `duration` 必须是整数；完全不传时默认5秒。
- `ratio` 支持 `16:9`、`9:16`、`1:1`、`4:3`、`3:4`，未传时默认 `16:9`。
- 上述时长范围适用于所有受支持的 `ratio`。
- 超出范围会在平台侧返回 HTTP 400 和 `invalid_seconds`，不会创建上游任务或产生上游生成费用。
- 平台固定使用24fps，不会为了延长视频而降低帧率。

## 2. 创建 10 秒视频

```bash
curl --request POST \
  --url 'http://127.0.0.1:3000/v1/videos' \
  --header 'Authorization: Bearer <TOKEN>' \
  --header 'Content-Type: application/json' \
  --data '{
    "model": "<AGNES_PUBLIC_MODEL>",
    "prompt": "A cinematic sunrise over a quiet lake, slow camera movement",
    "duration": 10,
    "resolution": "720p",
    "ratio": "16:9"
  }'
```

预期：请求成功，响应包含非空 `id` 和 `task_id`，并至少包含以下统一字段：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video",
  "model": "<AGNES_PUBLIC_MODEL>",
  "status": "queued",
  "duration": 10,
  "seconds": "10",
  "resolution": "720p",
  "ratio": "16:9",
  "size": "1280x720"
}
```

`size` 是上游具体像素尺寸兼容字段；若上游未返回则允许缺失。平台向 Agnes 发送
`num_frames: 241`、`frame_rate: 24`、`width: 1280`、`height: 720`，不发送公共 `duration`、
`seconds`、`resolution` 或 `ratio`。

### 2.1 验证 1080p 流畅度和时长上限

1080p 保持 24 fps，最长支持 10 秒：

```bash
curl --request POST \
  --url 'http://127.0.0.1:3000/v1/videos' \
  --header 'Authorization: Bearer <TOKEN>' \
  --header 'Content-Type: application/json' \
  --data '{
    "model": "<AGNES_PUBLIC_MODEL>",
    "prompt": "A cinematic sunrise over a quiet lake, slow camera movement",
    "duration": 10,
    "resolution": "1080p",
    "ratio": "16:9"
  }'
```

预期：成功创建任务；平台向 Agnes 发送 `num_frames: 241`、`frame_rate: 24`、`width: 1920`、
`height: 1080`。

下面的 18 秒请求应在平台侧直接拒绝，不再降低帧率调用 Agnes：

```bash
curl --request POST \
  --url 'http://127.0.0.1:3000/v1/videos' \
  --header 'Authorization: Bearer <TOKEN>' \
  --header 'Content-Type: application/json' \
  --write-out '\nHTTP_STATUS=%{http_code}\n' \
  --data '{
    "model": "<AGNES_PUBLIC_MODEL>",
    "prompt": "A cinematic sunrise over a quiet lake, slow camera movement",
    "duration": 18,
    "resolution": "1080p",
    "ratio": "16:9"
  }'
```

预期：HTTP 400，错误码为 `invalid_seconds`，错误信息说明 1080p 为保持 24 fps 最长支持 10 秒，
且不调用上游。

## 3. 图生视频：统一参考图字段

Agnes 图生视频仍使用平台公共字段 `referenceImages`，不要传 Agnes 原生 `image`：

```bash
curl --request POST \
  --url 'http://127.0.0.1:3000/v1/videos' \
  --header 'Authorization: Bearer <TOKEN>' \
  --header 'Content-Type: application/json' \
  --data '{
    "model": "<AGNES_PUBLIC_MODEL>",
    "prompt": "Animate the reference image with a gentle camera movement",
    "referenceImages": [
      "https://example.com/reference.jpg"
    ],
    "duration": 10,
    "resolution": "720p",
    "ratio": "16:9"
  }'
```

把示例 URL 替换为 Agnes 能直接访问的 HTTP/HTTPS 图片地址。预期：成功创建图生视频任务；平台
向 Agnes 发送 `image: "<图片URL>"`，不发送 `referenceImages`。本次不支持通过 multipart 上传
本地参考图文件。

下面的多图请求应被平台直接拒绝：

```bash
curl --request POST \
  --url 'http://127.0.0.1:3000/v1/videos' \
  --header 'Authorization: Bearer <TOKEN>' \
  --header 'Content-Type: application/json' \
  --write-out '\nHTTP_STATUS=%{http_code}\n' \
  --data '{
    "model": "<AGNES_PUBLIC_MODEL>",
    "prompt": "multiple reference image validation",
    "referenceImages": [
      "https://example.com/reference-1.jpg",
      "https://example.com/reference-2.jpg"
    ]
  }'
```

预期：HTTP 400，错误码为 `invalid_reference_images`，不调用上游。

## 4. 查询任务

```bash
curl --request GET \
  --url 'http://127.0.0.1:3000/v1/videos/<TASK_ID>' \
  --header 'Authorization: Bearer <TOKEN>'
```

预期：响应 `id`、`task_id` 与 `<TASK_ID>` 一致，`model` 为公开模型名，并稳定包含整数
`duration` 和逻辑档位 `resolution`。任务完成后 `duration` 优先反映上游实际生成时长；
`seconds` 和 `size` 仅为兼容字段，不作为验收公共合同。任务未完成时，稍后重新执行同一命令。

## 5. 下载视频

任务状态为 `completed` 后执行：

```bash
curl --request GET \
  --location \
  --url 'http://127.0.0.1:3000/v1/videos/<TASK_ID>/content' \
  --header 'Authorization: Bearer <TOKEN>' \
  --output 'agnes-video.mp4'
```

预期：得到非空且可播放的 `agnes-video.mp4`。

检查实际时长：

```bash
ffprobe -v error -show_entries format=duration \
  -of default=noprint_wrappers=1:nokey=1 'agnes-video.mp4'
```

预期：实际时长接近 10 秒。编码器可能产生少量帧级偏差；如果明显回退为约 5 秒，应检查渠道是否
选择了 `Agnes Video V2`。

## 6. seconds 兼容别名

```bash
curl --request POST \
  --url 'http://127.0.0.1:3000/v1/videos' \
  --header 'Authorization: Bearer <TOKEN>' \
  --header 'Content-Type: application/json' \
  --data '{
    "model": "<AGNES_PUBLIC_MODEL>",
    "prompt": "A calm ocean at dawn",
    "seconds": "10"
  }'
```

预期：成功创建约 10 秒任务。新客户端仍应使用公共 `duration`。

## 7. 时长冲突保护

```bash
curl --request POST \
  --url 'http://127.0.0.1:3000/v1/videos' \
  --header 'Authorization: Bearer <TOKEN>' \
  --header 'Content-Type: application/json' \
  --write-out '\nHTTP_STATUS=%{http_code}\n' \
  --data '{
    "model": "<AGNES_PUBLIC_MODEL>",
    "prompt": "duration conflict acceptance test",
    "duration": 5,
    "seconds": "10"
  }'
```

预期：HTTP 400，错误码为 `duration_conflict`，不调用上游、不产生消费记录。

## 8. 扩展参数保护

`provider_options` 由 Agnes 协议自动识别固定的 `agnes` 命名空间，无需后台配置。以下请求尝试
覆盖计费时长控制字段：

```bash
curl --request POST \
  --url 'http://127.0.0.1:3000/v1/videos' \
  --header 'Authorization: Bearer <TOKEN>' \
  --header 'Content-Type: application/json' \
  --write-out '\nHTTP_STATUS=%{http_code}\n' \
  --data '{
    "model": "<AGNES_PUBLIC_MODEL>",
    "prompt": "protected provider option acceptance test",
    "duration": 10,
    "provider_options": {
      "agnes": {
        "parameters": {
          "num_frames": 441
        }
      }
    }
  }'
```

预期：HTTP 400，错误码为 `provider_option_conflict`。

## 9. 协议字段校验

```bash
curl --request POST \
  --url 'http://127.0.0.1:3000/v1/videos' \
  --header 'Authorization: Bearer <TOKEN>' \
  --header 'Content-Type: application/json' \
  --write-out '\nHTTP_STATUS=%{http_code}\n' \
  --data '{
    "model": "<AGNES_PUBLIC_MODEL>",
    "prompt": "unsupported field acceptance test",
    "custom_flag": true
  }'
```

预期：渠道已选择 Agnes 协议时返回 HTTP 400 和 `unsupported_parameter`。供应商专属字段必须放在
受保护的 `provider_options.agnes` 中。

## 10. 验收记录

| 检查项 | 结果 | 证据 |
| --- | --- | --- |
| 渠道视频协议为 Agnes Video V2 | 待验收 | 截图 |
| `duration: 10` 对应约 10 秒视频 | 待验收 | 任务 ID、`ffprobe` 输出 |
| `720p + 16:9` 映射为 Agnes 标准输出 | 待验收 | 任务响应、视频尺寸 |
| `referenceImages` 单图成功创建图生视频 | 待验收 | 任务 ID、输入图片和生成视频 |
| Agnes 多参考图返回 `invalid_reference_images` | 待验收 | HTTP 响应 |
| `seconds` 兼容别名 | 待验收 | 任务 ID |
| 时长冲突返回 `duration_conflict` | 待验收 | HTTP 响应 |
| `num_frames` 覆盖返回 `provider_option_conflict` | 待验收 | HTTP 响应 |
| 未知顶层字段返回 `unsupported_parameter` | 待验收 | HTTP 响应 |

验收结束后撤销或轮换专用 Token，并清理下载的视频文件。不要把 Token、Agnes Key 或完整请求日志
提交到 Git。
