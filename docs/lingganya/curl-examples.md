# Lingganya 视频接口 curl 用例

本文使用平台统一的视频接口，覆盖视频任务提交、查询、下载、参考素材、首尾帧、参数兼容和常见错误。客户端不需要直接访问上游地址，也不需要知道平台内部的视频协议名称。

先设置服务地址、平台 API Key 和公共模型名称：

```bash
export ZMODEL_BASE_URL="https://api.example.com"
export ZMODEL_API_KEY="YOUR_API_KEY"
export VIDEO_MODEL="YOUR_PUBLIC_VIDEO_MODEL"
```

`VIDEO_MODEL` 必须替换为平台中已经配置视频能力模板的公共模型名称。

## 1. 提交文生视频任务

```bash
curl --request POST "$ZMODEL_BASE_URL/v1/videos" \
  --header "Authorization: Bearer $ZMODEL_API_KEY" \
  --header "Content-Type: application/json" \
  --data-raw "{
	    \"model\": \"$VIDEO_MODEL\",
	    \"prompt\": \"一只橘猫在雨后的城市街道上缓慢行走，电影感镜头\",
	    \"duration\": 8,
	    \"ratio\": \"16:9\"
  }"
```

创建响应中的 `id` 和 `task_id` 是平台任务 ID，两者相同。保存其中一个用于后续查询。

## 2. 使用参考图片

参考素材必须是上游能够直接访问的 HTTP/HTTPS 地址：

```bash
curl --request POST "$ZMODEL_BASE_URL/v1/videos" \
  --header "Authorization: Bearer $ZMODEL_API_KEY" \
  --header "Content-Type: application/json" \
  --data-raw "{
	    \"model\": \"$VIDEO_MODEL\",
	    \"prompt\": \"保持人物外观，生成一段人物在海边回头微笑的视频\",
	    \"duration\": 8,
	    \"ratio\": \"9:16\",
    \"referenceImages\": [
      \"https://media.example.com/person.jpg\"
    ]
  }"
```

多个参考图片会按数组顺序传递，数量上限由当前模型能力模板决定：

```bash
curl --request POST "$ZMODEL_BASE_URL/v1/videos" \
  --header "Authorization: Bearer $ZMODEL_API_KEY" \
  --header "Content-Type: application/json" \
  --data-raw "{
	    \"model\": \"$VIDEO_MODEL\",
	    \"prompt\": \"将产品外观、材质和灯光融合为一段广告视频\",
	    \"duration\": 10,
	    \"ratio\": \"16:9\",
    \"referenceImages\": [
      \"https://media.example.com/product-front.jpg\",
      \"https://media.example.com/product-side.jpg\",
      \"https://media.example.com/product-detail.jpg\"
    ]
  }"
```

## 3. 使用参考视频和参考音频

只有能力模板允许对应素材的模型才能传 `referenceVideos` 或 `referenceAudios`：

```bash
curl --request POST "$ZMODEL_BASE_URL/v1/videos" \
  --header "Authorization: Bearer $ZMODEL_API_KEY" \
  --header "Content-Type: application/json" \
  --data-raw "{
	    \"model\": \"$VIDEO_MODEL\",
	    \"prompt\": \"让人物按照动作参考表演，并使画面节奏匹配音乐\",
	    \"duration\": 10,
	    \"ratio\": \"16:9\",
    \"referenceImages\": [
      \"https://media.example.com/character.jpg\"
    ],
    \"referenceVideos\": [
      \"https://media.example.com/motion.mp4\"
    ],
    \"referenceAudios\": [
      \"https://media.example.com/music.mp3\"
    ]
  }"
```

不支持参考视频或音频时，会在提交前返回 `invalid_reference_videos` 或 `invalid_reference_audios`，不会产生上游任务。

## 4. 使用首帧和尾帧

模型支持时，可以传首帧和尾帧。尾帧是否必须依赖首帧由能力模板决定：

```bash
curl --request POST "$ZMODEL_BASE_URL/v1/videos" \
  --header "Authorization: Bearer $ZMODEL_API_KEY" \
  --header "Content-Type: application/json" \
  --data-raw "{
	    \"model\": \"$VIDEO_MODEL\",
	    \"prompt\": \"从第一张图的构图平滑过渡到最后一张图的构图\",
	    \"duration\": 8,
	    \"ratio\": \"16:9\",
    \"first_image\": \"https://media.example.com/start.png\",
    \"last_image\": \"https://media.example.com/end.png\"
  }"
```

## 4.1 `sd-2.0-vip` 多模态参考

`sd-2.0-vip` 使用统一的平台字段，平台会将参考视频和音频放入上游要求的 `extra` 对象：

```bash
curl --request POST "$ZMODEL_BASE_URL/v1/videos" \
  --header "Authorization: Bearer $ZMODEL_API_KEY" \
  --header "Content-Type: application/json" \
  --data-raw '{
    "model": "sd-2.0-vip",
    "prompt": "结合人物、动作和音乐参考生成电影感短片",
    "duration": 6,
    "resolution": "720p",
    "ratio": "16:9",
    "referenceImages": ["https://media.example.com/character.jpg"],
    "referenceVideos": ["https://media.example.com/motion.mp4"],
    "referenceAudios": ["https://media.example.com/music.mp3"]
  }'
```

该模型支持 4–15 秒任意整数，最多 9 张图片、3 个视频和 3 个音频；传视频或音频时至少需要一张图片。
平台会生成 `seconds`、`size`、`images` 以及 `extra.reference_videos` / `extra.reference_audios`。
能力模板来源：[SD（seedance）视频模型对接统一调用格式](https://lingganya.apifox.cn/9229531m0)。

## 5. 生成原生音频

模型声明支持原生音频时，可以显式传 `generate_audio`：

```bash
curl --request POST "$ZMODEL_BASE_URL/v1/videos" \
  --header "Authorization: Bearer $ZMODEL_API_KEY" \
  --header "Content-Type: application/json" \
  --data-raw "{
	    \"model\": \"$VIDEO_MODEL\",
	    \"prompt\": \"夜晚街头的电影镜头，包含自然环境声和脚步声\",
	    \"duration\": 8,
	    \"ratio\": \"16:9\",
    \"generate_audio\": true
  }"
```

## 6. 分辨率、画幅和兼容别名

平台的 `resolution` 表示清晰度档位，`ratio` 表示画幅。比例型模型只传 `ratio`，平台会将其转换为上游 `size`：

```bash
curl --request POST "$ZMODEL_BASE_URL/v1/videos" \
  --header "Authorization: Bearer $ZMODEL_API_KEY" \
  --header "Content-Type: application/json" \
  --data-raw "{
    \"model\": \"$VIDEO_MODEL\",
    \"prompt\": \"生成横屏产品展示视频\",
    \"duration\": 8,
    \"ratio\": \"16:9\"
  }"
```

该请求会转换为上游 `size=16:9`，不会发送 `resolution`。

`grok-imagine-video-1.5-preview` 的 `1080p + 16:9` 会转换为上游
`size=1792x1024`：

```bash
curl --request POST "$ZMODEL_BASE_URL/v1/videos" \
  --header "Authorization: Bearer $ZMODEL_API_KEY" \
  --header "Content-Type: application/json" \
  --data-raw '{
    "model": "grok-imagine-video-1.5-preview",
    "prompt": "根据参考图生成高清横屏产品展示视频",
    "duration": 10,
    "resolution": "1080p",
    "ratio": "16:9",
    "referenceImages": ["https://media.example.com/product.jpg"]
  }'
```

竖屏 `1080p + 9:16` 会转换为 `size=1024x1792`。该模型没有
`1080p + 1:1` 对应的上游固定尺寸，因此这一组合会返回 `invalid_ratio`。

除 `grok-imagine-video-1.5-preview` 外，本渠道内置模板中的模型都没有分辨率档位，
请求时只传公共画幅字段 `ratio`，不要传 `resolution`：

```bash
curl --request POST "$ZMODEL_BASE_URL/v1/videos" \
  --header "Authorization: Bearer $ZMODEL_API_KEY" \
  --header "Content-Type: application/json" \
  --data-raw '{
    "model": "grok-video-1.5-special",
    "prompt": "保持参考图主体特征，生成横屏动态视频",
    "duration": 15,
    "ratio": "16:9",
    "referenceImages": ["https://media.example.com/reference.jpg"]
  }'
```

平台会将 `ratio=16:9` 转换为上游 `size=16:9`，并且不会向上游发送
`resolution`。如果客户端给比例型模型显式传入 `resolution=720p`，平台会返回
HTTP 400 和 `invalid_resolution`，避免将上游不支持的参数静默吞掉。

历史客户端可以用 `size` 代替 `ratio`：

```bash
curl --request POST "$ZMODEL_BASE_URL/v1/videos" \
  --header "Authorization: Bearer $ZMODEL_API_KEY" \
  --header "Content-Type: application/json" \
  --data-raw "{
    \"model\": \"$VIDEO_MODEL\",
    \"prompt\": \"生成竖屏短视频\",
    \"duration\": 8,
    \"size\": \"9:16\"
  }"
```

`size` 与 `ratio` 同时出现时必须表示相同画幅。下面的请求会返回 `size_conflict`：

```bash
curl --request POST "$ZMODEL_BASE_URL/v1/videos" \
  --header "Authorization: Bearer $ZMODEL_API_KEY" \
  --header "Content-Type: application/json" \
  --data-raw "{
    \"model\": \"$VIDEO_MODEL\",
    \"prompt\": \"测试尺寸冲突\",
    \"duration\": 8,
    \"size\": \"16:9\",
    \"ratio\": \"9:16\"
  }"
```

对配置了清晰度档位的像素型模型，`resolution=720p` 与 `ratio=16:9` 不冲突，因为一个表示清晰度、一个表示画幅。

## 7. 查询任务状态

```bash
export VIDEO_TASK_ID="task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

curl --request GET "$ZMODEL_BASE_URL/v1/videos/$VIDEO_TASK_ID" \
  --header "Authorization: Bearer $ZMODEL_API_KEY"
```

建议每 5–10 秒查询一次，直到 `status` 为 `completed` 或 `failed`。平台统一返回 `queued`、`in_progress`、`completed`、`failed` 四种状态。

## 8. 下载已完成视频

`/content` 地址不需要客户端 Bearer Token。以下命令同时兼容平台流式返回内容和重定向交付：

```bash
curl --fail --location \
  "$ZMODEL_BASE_URL/v1/videos/$VIDEO_TASK_ID/content" \
  --output generated-video.mp4
```

该渠道的上游 `/content` 必须携带上游 Bearer API Key，直接访问上游地址会得到 HTTP 401。内容
交付严格遵循管理员配置：S3 优先命中时重定向到 S3 签名地址；代理开启时由平台返回 HTTP 200/206
视频内容；两者均未命中时返回明确错误，不会强制占用平台带宽代理上游内容。

也可以从查询响应中取出平台 `url`：

```bash
curl --fail --location \
  "$(curl --silent --show-error \
    --header "Authorization: Bearer $ZMODEL_API_KEY" \
    "$ZMODEL_BASE_URL/v1/videos/$VIDEO_TASK_ID" \
    | jq -r '.url')" \
  --output generated-video.mp4
```

不要保存重定向后的上游地址或对象存储签名地址，只保存平台 `/content` 地址。

## 9. 轮询直到结束

以下示例依赖 `jq`：

```bash
while true; do
  response="$(curl --silent --show-error \
    --header "Authorization: Bearer $ZMODEL_API_KEY" \
    "$ZMODEL_BASE_URL/v1/videos/$VIDEO_TASK_ID")"
  status="$(printf '%s' "$response" | jq -r '.status // empty')"
  progress="$(printf '%s' "$response" | jq -r '.progress // 0')"
  printf 'status=%s progress=%s%%\n' "$status" "$progress"

  case "$status" in
    completed)
      printf '%s\n' "$(printf '%s' "$response" | jq -r '.url')"
      break
      ;;
    failed)
      printf '%s\n' "$(printf '%s' "$response" | jq -r '.error.message // \"video generation failed\"')" >&2
      exit 1
      ;;
  esac
  sleep 8
done
```

## 10. 常见错误用例

### 比例型模型传入不支持的分辨率

`gemini_omni_flash` 没有清晰度档位，下面的请求会被拒绝：

```bash
curl --request POST "$ZMODEL_BASE_URL/v1/videos" \
  --header "Authorization: Bearer $ZMODEL_API_KEY" \
  --header "Content-Type: application/json" \
  --data-raw "{
    \"model\": \"gemini_omni_flash\",
    \"prompt\": \"测试不支持的分辨率\",
    \"duration\": 10,
    \"resolution\": \"720p\",
    \"ratio\": \"16:9\"
  }"
```

预期返回 HTTP 400 和 `invalid_resolution`。

### 时长不在允许列表

```bash
curl --request POST "$ZMODEL_BASE_URL/v1/videos" \
  --header "Authorization: Bearer $ZMODEL_API_KEY" \
  --header "Content-Type: application/json" \
  --data-raw "{
    \"model\": \"$VIDEO_MODEL\",
    \"prompt\": \"测试不支持的时长\",
    \"duration\": 6,
    \"ratio\": \"16:9\"
  }"
```

预期返回 HTTP 400 和 `invalid_seconds`，具体允许值以当前模型能力模板为准。

### 无效参考素材 URL

```bash
curl --request POST "$ZMODEL_BASE_URL/v1/videos" \
  --header "Authorization: Bearer $ZMODEL_API_KEY" \
  --header "Content-Type: application/json" \
  --data-raw "{
    \"model\": \"$VIDEO_MODEL\",
    \"prompt\": \"测试无效素材地址\",
    \"duration\": 8,
    \"ratio\": \"16:9\",
    \"referenceImages\": [\"file:///tmp/reference.jpg\"]
  }"
```

预期返回 HTTP 400 和 `invalid_reference_images`。本地文件路径、内网地址和需要登录才能访问的 URL 不适合作为参考素材。

### 不支持的 multipart 请求

该渠道适配器只接受 JSON 公网 URL；下面的请求会返回 `unsupported_content_type`：

```bash
curl --request POST "$ZMODEL_BASE_URL/v1/videos" \
  --header "Authorization: Bearer $ZMODEL_API_KEY" \
  --form "model=$VIDEO_MODEL" \
  --form 'prompt=不支持 multipart 的模型' \
  --form 'duration=8' \
  --form 'resolution=720p' \
  --form 'referenceImageFiles=@reference.jpg'
```

### 不支持 remix

不要调用 `/remix`，也不要在请求体中传入供应商专属 remix 字段。该渠道不提供平台 remix 操作。

## 11. 调用注意事项

- 创建和查询请求使用平台 Bearer Token，不要替换为上游 Key。
- 参考图片、视频和音频必须是上游可访问的公网 HTTP/HTTPS URL。
- 不要根据模型名称猜测能力，先查看平台后台的视频能力模板。
- 创建成功后只轮询已返回的任务 ID，不要因网络重试重复提交任务。
- 失败时读取 `error.code` 和结构化字段，不要依赖完整错误文本。
- 只持久化平台任务 ID 和 `/content` 地址，不要持久化上游任务 ID、签名 URL 或供应商域名。
