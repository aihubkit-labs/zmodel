package sora

import (
	"fmt"
	"io"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLingganyaTestContext(t *testing.T, body string) (*gin.Context, *relaycommon.RelayInfo) {
	t.Helper()
	ctx, info := newSeedanceTestContext(t, body)
	modelName := "sora-2"
	configureTestVideoModel(info, dto.VideoProtocolLingganya, modelName, nil, 8, 0, 0, 4, 12)
	capability := info.ChannelSetting.VideoModelCapabilities[modelName]
	capability.Ratios = []string{"16:9", "9:16"}
	capability.AllowedDurationSeconds = []int{4, 8, 12}
	capability.DefaultDurationSeconds = common.GetPointer(4)
	capability.ReferenceImagesIncompatibleWithFrames = common.GetPointer(false)
	capability.AudioReferenceRequiresVisualReference = common.GetPointer(false)
	capability.FramesAsReferenceImages = common.GetPointer(true)
	capability.OmitParameters = []string{"resolution"}
	info.ChannelSetting.VideoModelCapabilities[modelName] = capability
	info.OriginModelName = modelName
	info.UpstreamModelName = modelName
	return ctx, info
}

func TestLingganyaBuildRequestBodyMapsCompatibilityFields(t *testing.T) {
	ctx, info := newLingganyaTestContext(t, `{
		"model":"sora-2",
		"prompt":"city at night",
		"seconds":8,
		"size":"16:9",
		"referenceImages":["https://example.com/reference.jpg"],
		"images":["https://example.com/images.jpg","https://example.com/reference.jpg"],
		"image":"https://example.com/image.jpg",
		"input_reference":"https://example.com/input.jpg",
		"first_image":"https://example.com/first.jpg",
		"last_image":"https://example.com/last.jpg",
		"provider_options":{"lingganya":{"enhance_prompt":true,"camera":{"movement":"pan"}}}
	}`)
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

	parsed, err := relaycommon.GetTaskRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, 8, parsed.Duration)
	assert.Empty(t, parsed.Resolution)
	assert.Equal(t, "16:9", parsed.Size)
	assert.Equal(t, "16:9", parsed.Ratio)
	assert.Equal(t, []string{
		"https://example.com/reference.jpg",
		"https://example.com/images.jpg",
		"https://example.com/input.jpg",
		"https://example.com/image.jpg",
	}, parsed.ReferenceImages)

	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var upstream map[string]any
	require.NoError(t, common.Unmarshal(data, &upstream))
	assert.Equal(t, map[string]any{
		"model":   "sora-2",
		"prompt":  "city at night",
		"seconds": float64(8),
		"size":    "16:9",
		"images": []any{
			"https://example.com/reference.jpg",
			"https://example.com/images.jpg",
			"https://example.com/input.jpg",
			"https://example.com/image.jpg",
			"https://example.com/first.jpg",
			"https://example.com/last.jpg",
		},
		"extra": map[string]any{
			"enhance_prompt": true,
			"camera":         map[string]any{"movement": "pan"},
		},
	}, upstream)
}

func TestLingganyaRatioOnlyModelDoesNotInjectResolution(t *testing.T) {
	ctx, info := newLingganyaTestContext(t, `{"model":"sora-2","prompt":"demo","duration":4,"ratio":"16:9"}`)
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

	request, err := relaycommon.GetTaskRequest(ctx)
	require.NoError(t, err)
	assert.Empty(t, request.Resolution)
	assert.Equal(t, "16:9", request.Size)
	assert.Equal(t, "16:9", request.Ratio)

	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	var upstream map[string]any
	require.NoError(t, common.Unmarshal(data, &upstream))
	assert.Equal(t, "16:9", upstream["size"])
	assert.NotContains(t, upstream, "resolution")
}

func TestLingganyaMapsPublicDimensionsToPixelSize(t *testing.T) {
	ctx, info := newLingganyaTestContext(t, `{
		"model":"grok-imagine-video-1.5-preview",
		"prompt":"demo",
		"duration":10,
		"resolution":"1080p",
		"ratio":"16:9",
		"referenceImages":["https://example.com/reference.jpg"]
	}`)
	modelName := "grok-imagine-video-1.5-preview"
	capability := info.ChannelSetting.VideoModelCapabilities["sora-2"]
	capability.AllowedDurationSeconds = []int{10, 15}
	capability.MinDurationSeconds = common.GetPointer(10)
	capability.MaxDurationSeconds = common.GetPointer(15)
	capability.MinReferenceImages = common.GetPointer(1)
	capability.MaxReferenceImages = common.GetPointer(1)
	capability.Resolutions = []string{"720p", "1080p"}
	capability.SizeMappings = map[string]string{
		"720p|16:9":  "1280x720",
		"720p|9:16":  "720x1280",
		"1080p|16:9": "1792x1024",
		"1080p|9:16": "1024x1792",
	}
	info.ChannelSetting.VideoModelCapabilities = map[string]dto.VideoModelCapability{modelName: capability}
	info.OriginModelName = modelName
	info.UpstreamModelName = modelName

	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	request, err := relaycommon.GetTaskRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, "1080p", request.Resolution)
	assert.Equal(t, "16:9", request.Ratio)

	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	var upstream map[string]any
	require.NoError(t, common.Unmarshal(data, &upstream))
	assert.Equal(t, "1792x1024", upstream["size"])
	assert.NotContains(t, upstream, "resolution")
	assert.NotContains(t, upstream, "ratio")
}

func TestLingganyaModelWithoutResolutionOnlySendsAspectRatioAsSize(t *testing.T) {
	ctx, info := newLingganyaTestContext(t, `{
		"model":"grok-video-1.5-special",
		"prompt":"demo",
		"duration":10,
		"ratio":"16:9",
		"referenceImages":["https://example.com/reference.jpg"]
	}`)
	modelName := "grok-video-1.5-special"
	capability := info.ChannelSetting.VideoModelCapabilities["sora-2"]
	capability.AllowedDurationSeconds = []int{10, 15}
	capability.MinDurationSeconds = common.GetPointer(10)
	capability.MaxDurationSeconds = common.GetPointer(15)
	capability.MinReferenceImages = common.GetPointer(1)
	capability.MaxReferenceImages = common.GetPointer(1)
	capability.Resolutions = nil
	capability.SizeMappings = nil
	info.ChannelSetting.VideoModelCapabilities = map[string]dto.VideoModelCapability{modelName: capability}
	info.OriginModelName = modelName
	info.UpstreamModelName = modelName

	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	var upstream map[string]any
	require.NoError(t, common.Unmarshal(data, &upstream))
	assert.Equal(t, "16:9", upstream["size"])
	assert.NotContains(t, upstream, "resolution")
	assert.NotContains(t, upstream, "ratio")
}

func TestLingganyaRatioOnlyModelEstimatesBillingWithoutResolution(t *testing.T) {
	ctx, info := newLingganyaTestContext(t, `{
		"model":"grok-video-1.5-special",
		"prompt":"demo",
		"duration":10,
		"ratio":"16:9",
		"referenceImages":["https://example.com/reference.jpg"]
	}`)
	modelName := "grok-video-1.5-special"
	capability := info.ChannelSetting.VideoModelCapabilities["sora-2"]
	capability.AllowedDurationSeconds = []int{10, 15}
	capability.MinDurationSeconds = common.GetPointer(10)
	capability.MaxDurationSeconds = common.GetPointer(15)
	capability.MinReferenceImages = common.GetPointer(1)
	capability.MaxReferenceImages = common.GetPointer(1)
	capability.Resolutions = nil
	capability.SizeMappings = nil
	info.ChannelSetting.VideoModelCapabilities = map[string]dto.VideoModelCapability{modelName: capability}
	info.OriginModelName = modelName
	info.UpstreamModelName = modelName

	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	dimensions, err := adaptor.EstimateBillingDimensions(ctx, info)
	require.NoError(t, err)
	assert.Equal(t, float64(10), dimensions.Seconds)
	assert.Equal(t, "16:9", dimensions.ResolutionTier)
}

func TestLingganyaModelWithoutResolutionRejectsResolution(t *testing.T) {
	ctx, info := newLingganyaTestContext(t, `{
		"model":"grok-video-1.5-special",
		"prompt":"demo",
		"duration":10,
		"resolution":"720p",
		"ratio":"16:9",
		"referenceImages":["https://example.com/reference.jpg"]
	}`)
	modelName := "grok-video-1.5-special"
	capability := info.ChannelSetting.VideoModelCapabilities["sora-2"]
	capability.AllowedDurationSeconds = []int{10, 15}
	capability.MinDurationSeconds = common.GetPointer(10)
	capability.MaxDurationSeconds = common.GetPointer(15)
	capability.MinReferenceImages = common.GetPointer(1)
	capability.MaxReferenceImages = common.GetPointer(1)
	capability.Resolutions = nil
	capability.SizeMappings = nil
	info.ChannelSetting.VideoModelCapabilities = map[string]dto.VideoModelCapability{modelName: capability}
	info.OriginModelName = modelName
	info.UpstreamModelName = modelName

	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_resolution", taskErr.Code)
	assert.NotContains(t, taskErr.Message, "lingganya_video")
}

func TestLingganyaUsesCapabilityDefaultDuration(t *testing.T) {
	ctx, info := newLingganyaTestContext(t, `{"model":"sora-2","prompt":"demo","resolution":"16:9"}`)
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

	parsed, err := relaycommon.GetTaskRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, 4, parsed.Duration)

	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	var upstream map[string]any
	require.NoError(t, common.Unmarshal(data, &upstream))
	assert.Equal(t, float64(4), upstream["seconds"])
	assert.Equal(t, "16:9", upstream["size"])
}

func TestLingganyaRejectsUnsupportedDiscreteDuration(t *testing.T) {
	ctx, info := newLingganyaTestContext(t, `{"model":"sora-2","prompt":"demo","duration":6,"resolution":"16:9"}`)
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_seconds", taskErr.Code)
	assert.Equal(t, []any{"4", "8", "12"}, taskErr.Data.(dto.VideoParameterErrorData).AllowedValues)
}

func TestLingganyaRejectsConflictingSizeAliases(t *testing.T) {
	ctx, info := newLingganyaTestContext(t, `{"model":"sora-2","prompt":"demo","duration":4,"resolution":"16:9","size":"9:16"}`)
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "size_conflict", taskErr.Code)
	assert.NotContains(t, taskErr.Message, "Lingganya")
}

func newSD20VIPLingganyaTestContext(t *testing.T, body string) (*gin.Context, *relaycommon.RelayInfo) {
	t.Helper()
	ctx, info := newSeedanceTestContext(t, body)
	modelName := "sd-2.0-vip"
	falseValue := false
	trueValue := true
	minDuration, maxDuration, defaultDuration := 4, 15, 6
	capability := dto.VideoModelCapability{
		Resolutions:                           []string{"720p"},
		Ratios:                                []string{"16:9", "9:16"},
		ResolutionMappings:                    map[string]string{},
		RatioRequired:                         &falseValue,
		MinReferenceImages:                    common.GetPointer(0),
		MaxReferenceImages:                    common.GetPointer(9),
		MinReferenceVideos:                    common.GetPointer(0),
		MaxReferenceVideos:                    common.GetPointer(3),
		MinReferenceAudios:                    common.GetPointer(0),
		MaxReferenceAudios:                    common.GetPointer(3),
		MaxReferenceMediaCount:                common.GetPointer(12),
		SupportsDuration:                      &trueValue,
		DurationRequired:                      &falseValue,
		MinDurationSeconds:                    &minDuration,
		MaxDurationSeconds:                    &maxDuration,
		DefaultDurationSeconds:                &defaultDuration,
		SupportsGenerateAudio:                 &falseValue,
		GenerateAudioRequired:                 &falseValue,
		SupportsFirstFrame:                    &falseValue,
		FirstFrameRequired:                    &falseValue,
		SupportsLastFrame:                     &falseValue,
		LastFrameRequired:                     &falseValue,
		LastFrameRequiresFirstFrame:           &falseValue,
		ReferenceImagesIncompatibleWithFrames: &falseValue,
		AudioReferenceRequiresVisualReference: &trueValue,
		ReferenceMediaRequiresVisualReference: &trueValue,
		ReferenceMediaIncompatibleWithFrames:  &falseValue,
		SupportsSeed:                          &falseValue,
		SupportsWatermark:                     &falseValue,
		AutoReferenceMode:                     &falseValue,
		FramesAsReferenceImages:               &falseValue,
		FixedParameters:                       map[string]any{},
	}
	info.ChannelSetting.VideoProtocol = dto.VideoProtocolLingganya
	info.ChannelSetting.VideoModelCapabilities = map[string]dto.VideoModelCapability{modelName: capability}
	info.OriginModelName = modelName
	info.UpstreamModelName = modelName
	return ctx, info
}

func TestLingganyaSD20VIPBuildsUnifiedRequest(t *testing.T) {
	ctx, info := newSD20VIPLingganyaTestContext(t, `{
		"model":"sd-2.0-vip",
		"prompt":"cinematic short film",
		"duration":6,
		"resolution":"720p",
		"ratio":"16:9",
		"referenceImages":["https://example.com/character.jpg"],
		"referenceVideos":["https://example.com/motion.mp4"],
		"referenceAudios":["https://example.com/music.mp3"],
		"provider_options":{"lingganya":{"enhance_prompt":true}}
	}`)

	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	var upstream map[string]any
	require.NoError(t, common.Unmarshal(data, &upstream))
	assert.Equal(t, "sd-2.0-vip", upstream["model"])
	assert.Equal(t, "cinematic short film", upstream["prompt"])
	assert.Equal(t, float64(6), upstream["seconds"])
	assert.Equal(t, "720p", upstream["resolution"])
	assert.Equal(t, "16:9", upstream["size"])
	assert.NotContains(t, upstream, "ratio")
	assert.Equal(t, []any{"https://example.com/character.jpg"}, upstream["images"])
	assert.Equal(t, map[string]any{
		"reference_videos": []any{"https://example.com/motion.mp4"},
		"reference_audios": []any{"https://example.com/music.mp3"},
		"enhance_prompt":   true,
	}, upstream["extra"])
}

func TestLingganyaSD20VIPRequiresImageForReferenceMedia(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "video", body: `{"model":"sd-2.0-vip","prompt":"demo","duration":6,"resolution":"720p","referenceVideos":["https://example.com/motion.mp4"]}`},
		{name: "audio", body: `{"model":"sd-2.0-vip","prompt":"demo","duration":6,"resolution":"720p","referenceAudios":["https://example.com/music.mp3"]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, info := newSD20VIPLingganyaTestContext(t, test.body)
			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
			require.NotNil(t, taskErr)
			assert.Equal(t, "invalid_reference_images", taskErr.Code)
			assert.NotContains(t, taskErr.Message, "lingganya_video")
		})
	}
}

func TestLingganyaSD20VIPUsesDurationRange(t *testing.T) {
	for _, duration := range []int{3, 16} {
		ctx, info := newSD20VIPLingganyaTestContext(t, fmt.Sprintf(`{"model":"sd-2.0-vip","prompt":"demo","duration":%d,"resolution":"720p"}`, duration))
		taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
		require.NotNil(t, taskErr)
		assert.Equal(t, "invalid_seconds", taskErr.Code)
	}
}

func TestLingganyaSD20VIPRejectsUnsupportedResolution(t *testing.T) {
	ctx, info := newSD20VIPLingganyaTestContext(t, `{"model":"sd-2.0-vip","prompt":"demo","duration":6,"resolution":"1080p"}`)
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_resolution", taskErr.Code)
	assert.NotContains(t, taskErr.Message, "lingganya_video")
}

func TestLingganyaSD20VIPRejectsExcessTotalReferenceMedia(t *testing.T) {
	images := `"referenceImages":["https://example.com/1.jpg","https://example.com/2.jpg","https://example.com/3.jpg","https://example.com/4.jpg","https://example.com/5.jpg","https://example.com/6.jpg","https://example.com/7.jpg","https://example.com/8.jpg","https://example.com/9.jpg"]`
	body := fmt.Sprintf(`{"model":"sd-2.0-vip","prompt":"demo","duration":6,"resolution":"720p",%s,"referenceVideos":["https://example.com/1.mp4","https://example.com/2.mp4","https://example.com/3.mp4"],"referenceAudios":["https://example.com/1.mp3"]}`, images)
	ctx, info := newSD20VIPLingganyaTestContext(t, body)
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_reference_media", taskErr.Code)
}
