package sora

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSeedanceTestContext(t *testing.T, body string) (*gin.Context, *relaycommon.RelayInfo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodPost, "/v1/videos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req
	_, err := common.GetBodyStorage(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		common.CleanupBodyStorage(ctx)
	})
	var request struct {
		Model string `json:"model"`
	}
	require.NoError(t, common.Unmarshal([]byte(body), &request))
	info := &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}
	configureTestVideoModel(info, dto.VideoProtocolMegabyAI, request.Model, []string{"480p", "720p", "1080p", "4k"}, 64, 64, 64)
	return ctx, info
}

func configureTestVideoModel(info *relaycommon.RelayInfo, protocol dto.VideoProtocol, modelName string, resolutions []string, imageLimit, videoLimit, audioLimit int, durationBounds ...int) {
	var minDurationSeconds *int
	var maxDurationSeconds *int
	extendedProtocol := protocol == dto.VideoProtocolMegabyAI || protocol == dto.VideoProtocolGlobalAIOpc
	minimaxDefaults := modelName == "minimax-h3"
	if extendedProtocol {
		minDuration := 4
		maxDuration := 15
		if len(durationBounds) >= 2 {
			minDuration = durationBounds[0]
			maxDuration = durationBounds[1]
		}
		minDurationSeconds = common.GetPointer(minDuration)
		maxDurationSeconds = common.GetPointer(maxDuration)
	}
	info.ChannelSetting.VideoProtocol = protocol
	info.ChannelSetting.VideoModelCapabilities = map[string]dto.VideoModelCapability{
		modelName: {
			Resolutions:                           resolutions,
			Ratios:                                []string{"16:9", "1:1", "9:16", "21:9", "4:3", "3:4"},
			RatioRequired:                         common.GetPointer(false),
			MinReferenceImages:                    common.GetPointer(0),
			MaxReferenceImages:                    common.GetPointer(imageLimit),
			MinReferenceVideos:                    common.GetPointer(0),
			MaxReferenceVideos:                    common.GetPointer(videoLimit),
			MinReferenceAudios:                    common.GetPointer(0),
			MaxReferenceAudios:                    common.GetPointer(audioLimit),
			SupportsDuration:                      common.GetPointer(extendedProtocol),
			DurationRequired:                      common.GetPointer(extendedProtocol),
			MinDurationSeconds:                    minDurationSeconds,
			MaxDurationSeconds:                    maxDurationSeconds,
			SupportsGenerateAudio:                 common.GetPointer(minimaxDefaults),
			GenerateAudioRequired:                 common.GetPointer(minimaxDefaults),
			SupportsFirstFrame:                    common.GetPointer(true),
			FirstFrameRequired:                    common.GetPointer(false),
			SupportsLastFrame:                     common.GetPointer(true),
			LastFrameRequired:                     common.GetPointer(false),
			LastFrameRequiresFirstFrame:           common.GetPointer(true),
			ReferenceImagesIncompatibleWithFrames: common.GetPointer(true),
			AudioReferenceRequiresVisualReference: common.GetPointer(true),
			ReferenceMediaIncompatibleWithFrames:  common.GetPointer(false),
			SupportsSeed:                          common.GetPointer(false),
			SupportsWatermark:                     common.GetPointer(minimaxDefaults),
		},
	}
	if protocol == dto.VideoProtocolGlobalAIOpc && modelName == "minimax-h3" {
		capability := info.ChannelSetting.VideoModelCapabilities[modelName]
		capability.ResolutionMappings = map[string]string{"1440p": "2k"}
		info.ChannelSetting.VideoModelCapabilities[modelName] = capability
	}
}

func TestSeedanceValidationAndBillingDimensions(t *testing.T) {
	tests := []string{"480p", "720p", "1080p", "4K", "1440p"}

	for _, resolution := range tests {
		t.Run(resolution, func(t *testing.T) {
			ctx, info := newSeedanceTestContext(t, fmt.Sprintf(
				`{"model":%q,"prompt":"demo","duration":15,"resolution":%q}`,
				"upstream-video-model", resolution,
			))
			configureTestVideoModel(info, dto.VideoProtocolMegabyAI, "upstream-video-model", tests, 0, 0, 0)
			adaptor := &TaskAdaptor{}
			require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

			dimensions, err := adaptor.EstimateBillingDimensions(ctx, info)
			require.NoError(t, err)
			assert.Equal(t, float64(1), dimensions.Units)
			assert.Equal(t, float64(15), dimensions.Seconds)
			assert.Equal(t, strings.ToLower(resolution), dimensions.ResolutionTier)
		})
	}
}

func TestSeedanceRejectsUnsupportedResolution(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{"model":"public-video-model","prompt":"demo","duration":5,"resolution":"1440p"}`)
	info.OriginModelName = "public-video-model"
	info.UpstreamModelName = "upstream-video-model"
	configureTestVideoModel(info, dto.VideoProtocolMegabyAI, "upstream-video-model", []string{"720p"}, 0, 0, 0)
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_resolution", taskErr.Code)
	assert.Equal(t, `video model "public-video-model" does not support resolution "1440p"; supported values: 720p`, taskErr.Message)
	assert.NotContains(t, taskErr.Message, "upstream-video-model")
	assert.Equal(t, dto.VideoParameterErrorData{
		Parameter:     "resolution",
		Received:      "1440p",
		AllowedValues: []any{"720p"},
	}, taskErr.Data)
}

func TestSeedanceRejectsDurationOutsideConfiguredRange(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{"model":"videos-mini","prompt":"demo","duration":16,"resolution":"720p"}`)
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_seconds", taskErr.Code)
	assert.Equal(t, dto.VideoParameterErrorData{
		Parameter: "duration",
		Received:  int64(16),
		Minimum:   common.GetPointer[int64](4),
		Maximum:   common.GetPointer[int64](15),
	}, taskErr.Data)
}

func TestSeedanceUsesConfiguredDurationRange(t *testing.T) {
	for _, test := range []struct {
		name     string
		duration int
		wantCode string
	}{
		{name: "Seedance 2.5 maximum", duration: 29},
		{name: "above Seedance 2.5 maximum", duration: 30, wantCode: "invalid_seconds"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, info := newSeedanceTestContext(t, fmt.Sprintf(
				`{"model":"seedance-2.5","prompt":"demo","duration":%d,"resolution":"1080p"}`,
				test.duration,
			))
			configureTestVideoModel(info, dto.VideoProtocolMegabyAI, "seedance-2.5", []string{"1080p"}, 0, 0, 0, 4, 29)

			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
			if test.wantCode == "" {
				require.Nil(t, taskErr)
				return
			}
			require.NotNil(t, taskErr)
			assert.Equal(t, test.wantCode, taskErr.Code)
			assert.Equal(t, dto.VideoParameterErrorData{
				Parameter: "duration",
				Received:  int64(test.duration),
				Minimum:   common.GetPointer[int64](4),
				Maximum:   common.GetPointer[int64](29),
			}, taskErr.Data)
		})
	}
}

func TestSeedanceRejectsMissingDuration(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{"model":"videos-mini","prompt":"demo","resolution":"720p"}`)
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_seconds", taskErr.Code)
}

func TestSeedanceBuildRequestBodyUsesNormalizedParameters(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{"model":"tvideos","prompt":"demo","duration":"15","resolution":"4K"}`)
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	info.UpstreamModelName = "tvideos"

	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var upstream map[string]any
	require.NoError(t, common.Unmarshal(data, &upstream))
	assert.Equal(t, "4k", upstream["resolution"])
	assert.Equal(t, float64(15), upstream["duration"])
}

func TestSeedanceBuildRequestBodyConvertsSecondsAliasToDuration(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{"model":"videos-fast","prompt":"demo","seconds":"5","resolution":"720P"}`)
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	info.UpstreamModelName = "videos-fast"

	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var upstream map[string]any
	require.NoError(t, common.Unmarshal(data, &upstream))
	assert.Equal(t, float64(5), upstream["duration"])
	assert.Equal(t, "720p", upstream["resolution"])
	assert.NotContains(t, upstream, "seconds")
}

func TestSeedanceBuildRequestBodyPreservesReferenceMedia(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{
		"model":"videos-4",
		"prompt":"demo",
		"duration":5,
		"resolution":"720p",
		"referenceImages":["https://example.com/1.jpg","https://example.com/2.webp"],
		"referenceVideos":["https://example.com/1.mp4"],
		"referenceAudios":["https://example.com/1.mp3"]
	}`)
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	info.UpstreamModelName = "videos-4"
	parsed, err := relaycommon.GetTaskRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://example.com/1.jpg", "https://example.com/2.webp"}, parsed.ReferenceImages)
	assert.Equal(t, []string{"https://example.com/1.mp4"}, parsed.ReferenceVideos)
	assert.Equal(t, []string{"https://example.com/1.mp3"}, parsed.ReferenceAudios)

	dimensions, err := adaptor.EstimateBillingDimensions(ctx, info)
	require.NoError(t, err)
	assert.Equal(t, float64(2), dimensions.ReferenceImageCount)
	assert.Equal(t, float64(1), dimensions.ReferenceVideoCount)
	assert.Equal(t, float64(1), dimensions.ReferenceAudioCount)

	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var upstream struct {
		ReferenceImages []string `json:"referenceImages"`
		ReferenceVideos []string `json:"referenceVideos"`
		ReferenceAudios []string `json:"referenceAudios"`
	}
	require.NoError(t, common.Unmarshal(data, &upstream))
	assert.Equal(t, []string{"https://example.com/1.jpg", "https://example.com/2.webp"}, upstream.ReferenceImages)
	assert.Equal(t, []string{"https://example.com/1.mp4"}, upstream.ReferenceVideos)
	assert.Equal(t, []string{"https://example.com/1.mp3"}, upstream.ReferenceAudios)
}

func TestVideoModelReferenceLimitsUseConfiguredUpstreamCapability(t *testing.T) {
	tests := []struct {
		name string
		body string
		code string
	}{
		{
			name: "images within limit",
			body: `{"model":"public-video-model","prompt":"demo","resolution":"720p","referenceImages":["a"]}`,
		},
		{
			name: "too many images",
			body: `{"model":"public-video-model","prompt":"demo","resolution":"720p","referenceImages":["a","b"]}`,
			code: "invalid_reference_images",
		},
		{
			name: "too many videos",
			body: `{"model":"public-video-model","prompt":"demo","resolution":"720p","referenceVideos":["a","b"]}`,
			code: "invalid_reference_videos",
		},
		{
			name: "too many audios",
			body: `{"model":"public-video-model","prompt":"demo","resolution":"720p","referenceAudios":["a","b"]}`,
			code: "invalid_reference_audios",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, info := newSeedanceTestContext(t, test.body)
			info.UpstreamModelName = "upstream-video-model"
			configureTestVideoModel(info, dto.VideoProtocolOpenAI, "upstream-video-model", []string{"720p"}, 1, 1, 1)
			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
			if test.code == "" {
				require.Nil(t, taskErr)
				return
			}
			require.NotNil(t, taskErr)
			assert.Equal(t, test.code, taskErr.Code)
			assert.Equal(t, dto.VideoParameterErrorData{
				Parameter: "reference_" + strings.TrimPrefix(test.code, "invalid_reference_"),
				Received:  int64(2),
				Minimum:   common.GetPointer[int64](0),
				Maximum:   common.GetPointer[int64](1),
			}, taskErr.Data)
		})
	}
}

func TestConfiguredVideoProtocolsRequireResolution(t *testing.T) {
	tests := []struct {
		name     string
		protocol dto.VideoProtocol
		body     string
	}{
		{name: "OpenAI Video", protocol: dto.VideoProtocolOpenAI, body: `{"model":"video-model","prompt":"demo","duration":5}`},
		{name: "Seedance", protocol: dto.VideoProtocolMegabyAI, body: `{"model":"video-model","prompt":"demo","duration":5}`},
		{name: "MiniMax H3", protocol: dto.VideoProtocolMegabyAI, body: `{"model":"video-model","prompt":"demo","duration":5,"ratio":"16:9","generate_audio":true}`},
		{name: "Agnes Video V2", protocol: dto.VideoProtocolAgnesVideoV2, body: `{"model":"video-model","prompt":"demo","duration":5}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, info := newSeedanceTestContext(t, test.body)
			info.UpstreamModelName = "video-model"
			configureTestVideoModel(info, test.protocol, "video-model", []string{"720p"}, 0, 0, 0)

			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
			require.NotNil(t, taskErr)
			assert.Equal(t, "invalid_resolution", taskErr.Code)
		})
	}
}

func TestSeedanceProtocolUsesMappedUpstreamModelCapabilities(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{
		"model":"public-seedance-alias",
		"prompt":"demo",
		"duration":15,
		"resolution":"1080P"
	}`)
	info.UpstreamModelName = "tvideos"
	configureTestVideoModel(info, dto.VideoProtocolMegabyAI, "tvideos", []string{"1080p"}, 0, 0, 0)
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

	dimensions, err := adaptor.EstimateBillingDimensions(ctx, info)
	require.NoError(t, err)
	assert.Equal(t, float64(15), dimensions.Seconds)
	assert.Equal(t, "1080p", dimensions.ResolutionTier)

	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	var upstream map[string]any
	require.NoError(t, common.Unmarshal(data, &upstream))
	assert.Equal(t, "tvideos", upstream["model"])
	assert.Equal(t, float64(15), upstream["duration"])
	assert.Equal(t, "1080p", upstream["resolution"])
}

func TestSeedanceUsesConfiguredResolutionProfile(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{
		"model":"videos-standard",
		"prompt":"demo",
		"duration":5,
		"resolution":"720p"
	}`)
	configureTestVideoModel(info, dto.VideoProtocolMegabyAI, "videos-standard", []string{"480p"}, 0, 0, 0)

	adaptor := &TaskAdaptor{}
	taskErr := adaptor.ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_resolution", taskErr.Code)
}

func TestSeedanceCapabilityValidationUsesMappedModel(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{
		"model":"videos-standard",
		"prompt":"demo",
		"duration":10,
		"resolution":"4k"
	}`)
	info.UpstreamModelName = "tvideos"
	configureTestVideoModel(info, dto.VideoProtocolMegabyAI, "tvideos", []string{"480p", "720p", "1080p", "4k"}, 0, 0, 0)
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

	dimensions, err := adaptor.EstimateBillingDimensions(ctx, info)
	require.NoError(t, err)
	assert.Equal(t, "4k", dimensions.ResolutionTier)
	assert.Equal(t, []string{"480p", "720p", "1080p", "4k"}, info.VideoAllowedResolutions)
}

func TestSeedanceConfiguredUnknownMappedModelSetsSettlementSnapshot(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{
		"model":"public-seedance-alias",
		"prompt":"demo",
		"duration":10,
		"resolution":"1080p"
	}`)
	info.UpstreamModelName = "new-upstream-seedance"
	configureTestVideoModel(info, dto.VideoProtocolMegabyAI, "new-upstream-seedance", []string{"720p", "1080p"}, 0, 0, 0)
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

	dimensions, err := adaptor.EstimateBillingDimensions(ctx, info)
	require.NoError(t, err)
	assert.Equal(t, "1080p", dimensions.ResolutionTier)
	assert.Equal(t, []string{"720p", "1080p"}, info.VideoAllowedResolutions)
}

func TestSeedanceConfiguredUnknownModelFailsClosedWithoutCapability(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{
		"model":"public-seedance-alias",
		"prompt":"demo",
		"duration":10,
		"resolution":"720p"
	}`)
	info.UpstreamModelName = "unknown-upstream-model"
	adaptor := &TaskAdaptor{}
	taskErr := adaptor.ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "video_model_not_configured", taskErr.Code)
	assert.Equal(t, "video model is not configured for the selected channel", taskErr.Message)
	assert.NotContains(t, taskErr.Message, "unknown-upstream-model")
	assert.NotContains(t, taskErr.Message, string(dto.VideoProtocolMegabyAI))
}

func TestOpenAIVideoProtocolDoesNotApplySeedanceModelProfile(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{
		"model":"videos-mini",
		"prompt":"demo",
		"duration":30,
		"resolution":"1080p"
	}`)
	info.ChannelSetting.VideoProtocol = dto.VideoProtocolOpenAI
	info.UpstreamModelName = "videos-mini"
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

	request, err := relaycommon.GetTaskRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, 30, request.Duration)
	assert.Equal(t, "1080p", request.Resolution)
}

func TestVideoRequestWithoutProtocolPreservesHistoricalFields(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{"model":"agnes-video-v2","prompt":"demo","custom_flag":false}`)
	info.ChannelSetting = dto.ChannelSettings{}
	info.UpstreamModelName = "agnes-video-v2"
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	var upstream map[string]any
	require.NoError(t, common.Unmarshal(data, &upstream))
	assert.Contains(t, upstream, "custom_flag")
	assert.Equal(t, false, upstream["custom_flag"])
}

func TestConfiguredVideoProtocolRejectsUnknownField(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{"model":"agnes-video-v2","prompt":"demo","custom_flag":false}`)
	info.ChannelSetting.VideoProtocol = dto.VideoProtocolAgnesVideoV2
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "unsupported_parameter", taskErr.Code)
	assert.Equal(t, `unsupported video parameter "custom_flag"`, taskErr.Message)
	assert.Equal(t, dto.VideoParameterErrorData{Parameter: "custom_flag"}, taskErr.Data)
}

func TestVideoProtocolValidationErrorsDoNotExposeInternalProtocol(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{
		"model":"seedance-2.5",
		"prompt":"demo",
		"duration":4,
		"resolution":"480p",
		"ratio":"16:123",
		"aspect_ratio":"12:23"
	}`)
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "unsupported_parameter", taskErr.Code)
	assert.Equal(t, `unsupported video parameter "aspect_ratio"`, taskErr.Message)
	assert.Equal(t, dto.VideoParameterErrorData{Parameter: "aspect_ratio"}, taskErr.Data)
	assert.NotContains(t, taskErr.Message, string(dto.VideoProtocolMegabyAI))
	assert.NotContains(t, taskErr.Message, "megabyai")
	assert.NotContains(t, taskErr.Message, "protocol")
}

func TestVideoRequestProviderOptionsFlattensConfiguredNamespace(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{
		"model":"public-agnes-alias",
		"prompt":"demo",
		"duration":10,
		"resolution":"720p",
		"provider_options":{
			"agnes":{
				"custom_flag":false,
				"custom_zero":0,
				"custom_object":{"enabled":true},
				"custom_array":["a",2]
			}
		}
	}`)
	info.UpstreamModelName = "agnes-video-v2"
	configureTestVideoModel(info, dto.VideoProtocolAgnesVideoV2, "agnes-video-v2", []string{"720p"}, 1, 0, 0)
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	dimensions, err := adaptor.EstimateBillingDimensions(ctx, info)
	require.NoError(t, err)
	assert.Equal(t, float64(10), dimensions.Seconds)
	assert.Equal(t, float64(10), adaptor.EstimateBilling(ctx, info)["seconds"])

	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	var upstream map[string]any
	require.NoError(t, common.Unmarshal(data, &upstream))
	assert.Equal(t, "agnes-video-v2", upstream["model"])
	assert.NotContains(t, upstream, "provider_options")
	assert.NotContains(t, upstream, "duration")
	assert.NotContains(t, upstream, "seconds")
	assert.Equal(t, float64(241), upstream["num_frames"])
	assert.Equal(t, float64(24), upstream["frame_rate"])
	assert.Equal(t, float64(1280), upstream["width"])
	assert.Equal(t, float64(720), upstream["height"])
	assert.Equal(t, false, upstream["custom_flag"])
	assert.Equal(t, float64(0), upstream["custom_zero"])
	assert.Equal(t, map[string]any{"enabled": true}, upstream["custom_object"])
	assert.Equal(t, []any{"a", float64(2)}, upstream["custom_array"])
}

func TestSeedanceMegabyAIProviderOptionsUsesProtocolNamespace(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{
		"model":"videos-standard",
		"prompt":"demo",
		"duration":10,
		"resolution":"720p",
		"provider_options":{"megabyai":{"custom_flag":false}}
	}`)
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	var upstream map[string]any
	require.NoError(t, common.Unmarshal(data, &upstream))
	assert.NotContains(t, upstream, "provider_options")
	assert.Equal(t, false, upstream["custom_flag"])
}

func TestAgnesProviderConvertsDurationToFrameParametersWithoutModelHardcoding(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{
		"model":"future-agnes-model-name",
		"prompt":"demo",
		"duration":10,
		"resolution":"720p",
		"provider_options":{"agnes":{"custom_flag":true}}
	}`)
	info.UpstreamModelName = "renamed-upstream-agnes-model"
	configureTestVideoModel(info, dto.VideoProtocolAgnesVideoV2, "renamed-upstream-agnes-model", []string{"720p"}, 1, 0, 0)
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	var upstream map[string]any
	require.NoError(t, common.Unmarshal(data, &upstream))
	assert.Equal(t, "renamed-upstream-agnes-model", upstream["model"])
	assert.NotContains(t, upstream, "duration")
	assert.NotContains(t, upstream, "seconds")
	assert.Equal(t, float64(241), upstream["num_frames"])
	assert.Equal(t, float64(24), upstream["frame_rate"])
	assert.Equal(t, float64(1280), upstream["width"])
	assert.Equal(t, float64(720), upstream["height"])
}

func TestAgnesProviderConvertsResolutionAndRatioToDimensions(t *testing.T) {
	for _, test := range []struct {
		resolution string
		ratio      string
		wantWidth  float64
		wantHeight float64
	}{
		{resolution: "480p", ratio: "16:9", wantWidth: 854, wantHeight: 480},
		{resolution: "720p", ratio: "9:16", wantWidth: 720, wantHeight: 1280},
		{resolution: "1080p", ratio: "1:1", wantWidth: 1080, wantHeight: 1080},
		{resolution: "1440p", ratio: "16:9", wantWidth: 2560, wantHeight: 1440},
		{resolution: "720p", ratio: "4:3", wantWidth: 960, wantHeight: 720},
		{resolution: "480p", ratio: "3:4", wantWidth: 480, wantHeight: 640},
	} {
		t.Run(test.resolution+"_"+test.ratio, func(t *testing.T) {
			ctx, info := newSeedanceTestContext(t, fmt.Sprintf(`{
				"model":"agnes-video-v2",
				"prompt":"demo",
				"duration":10,
				"resolution":%q,
				"ratio":%q
			}`, test.resolution, test.ratio))
			info.UpstreamModelName = "agnes-video-v2.0"
			configureTestVideoModel(info, dto.VideoProtocolAgnesVideoV2, "agnes-video-v2.0", []string{test.resolution}, 1, 0, 0)
			adaptor := &TaskAdaptor{}
			require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

			request, err := relaycommon.GetTaskRequest(ctx)
			require.NoError(t, err)
			assert.Equal(t, test.resolution, request.Resolution)
			assert.Equal(t, test.ratio, request.Ratio)

			body, err := adaptor.BuildRequestBody(ctx, info)
			require.NoError(t, err)
			data, err := io.ReadAll(body)
			require.NoError(t, err)
			var upstream map[string]any
			require.NoError(t, common.Unmarshal(data, &upstream))
			assert.Equal(t, test.wantWidth, upstream["width"])
			assert.Equal(t, test.wantHeight, upstream["height"])
			assert.NotContains(t, upstream, "resolution")
			assert.NotContains(t, upstream, "ratio")
			assert.NotContains(t, upstream, "size")
		})
	}
}

func TestAgnesProviderRejectsUnsupportedResolutionInputs(t *testing.T) {
	for _, test := range []struct {
		name     string
		body     string
		wantCode string
	}{
		{name: "resolution", body: `{"model":"agnes-video-v2","prompt":"demo","resolution":"4k"}`, wantCode: "invalid_resolution"},
		{name: "ratio", body: `{"model":"agnes-video-v2","prompt":"demo","resolution":"720p","ratio":"2:1"}`, wantCode: "invalid_ratio"},
		{name: "size", body: `{"model":"agnes-video-v2","prompt":"demo","size":"1280x720"}`, wantCode: "invalid_size"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, info := newSeedanceTestContext(t, test.body)
			info.ChannelSetting.VideoProtocol = dto.VideoProtocolAgnesVideoV2
			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
			require.NotNil(t, taskErr)
			assert.Equal(t, test.wantCode, taskErr.Code)
		})
	}
}

func TestAgnesProviderDurationDefaultsAndBounds(t *testing.T) {
	for _, test := range []struct {
		name      string
		body      string
		want      int
		wantCode  string
		wantFrame float64
	}{
		{name: "default", body: `{"model":"agnes-video-v2","prompt":"demo","resolution":"720p"}`, want: 5, wantFrame: 121},
		{name: "maximum", body: `{"model":"agnes-video-v2","prompt":"demo","duration":18,"resolution":"720p"}`, want: 18, wantFrame: 433},
		{name: "above maximum", body: `{"model":"agnes-video-v2","prompt":"demo","duration":19,"resolution":"720p"}`, wantCode: "invalid_seconds"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, info := newSeedanceTestContext(t, test.body)
			info.ChannelSetting.VideoProtocol = dto.VideoProtocolAgnesVideoV2
			info.UpstreamModelName = "agnes-video-v2"
			adaptor := &TaskAdaptor{}
			taskErr := adaptor.ValidateRequestAndSetAction(ctx, info)
			if test.wantCode != "" {
				require.NotNil(t, taskErr)
				assert.Equal(t, test.wantCode, taskErr.Code)
				return
			}
			require.Nil(t, taskErr)
			request, err := relaycommon.GetTaskRequest(ctx)
			require.NoError(t, err)
			assert.Equal(t, test.want, request.Duration)
			assert.Equal(t, "720p", request.Resolution)
			assert.Equal(t, "16:9", request.Ratio)
			dimensions, err := adaptor.EstimateBillingDimensions(ctx, info)
			require.NoError(t, err)
			assert.Equal(t, float64(test.want), dimensions.Seconds)
			assert.Equal(t, float64(test.want), adaptor.EstimateBilling(ctx, info)["seconds"])

			body, err := adaptor.BuildRequestBody(ctx, info)
			require.NoError(t, err)
			data, err := io.ReadAll(body)
			require.NoError(t, err)
			var upstream map[string]any
			require.NoError(t, common.Unmarshal(data, &upstream))
			assert.Equal(t, test.wantFrame, upstream["num_frames"])
			assert.Equal(t, float64(24), upstream["frame_rate"])
			assert.Equal(t, float64(1280), upstream["width"])
			assert.Equal(t, float64(720), upstream["height"])
		})
	}
}

func TestAgnesProviderCaps1080pDurationAndPreservesFrameRate(t *testing.T) {
	for _, test := range []struct {
		name         string
		duration     int
		wantDuration int
	}{
		{name: "maximum 1080p duration", duration: 10, wantDuration: 10},
		{name: "duration one second over limit", duration: 11, wantDuration: 10},
		{name: "global maximum duration", duration: 18, wantDuration: 10},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, info := newSeedanceTestContext(t, fmt.Sprintf(`{
				"model":"agnes-video-v2.0",
				"prompt":"demo",
				"duration":%d,
				"resolution":"1080p",
				"ratio":"16:9"
			}`, test.duration))
			info.UpstreamModelName = "agnes-video-v2.0"
			configureTestVideoModel(info, dto.VideoProtocolAgnesVideoV2, "agnes-video-v2.0", []string{"1080p"}, 1, 0, 0)
			adaptor := &TaskAdaptor{}
			require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

			request, err := relaycommon.GetTaskRequest(ctx)
			require.NoError(t, err)
			assert.Equal(t, test.wantDuration, request.Duration)
			dimensions, err := adaptor.EstimateBillingDimensions(ctx, info)
			require.NoError(t, err)
			assert.Equal(t, float64(test.wantDuration), dimensions.Seconds)

			body, err := adaptor.BuildRequestBody(ctx, info)
			require.NoError(t, err)
			data, err := io.ReadAll(body)
			require.NoError(t, err)
			var upstream map[string]any
			require.NoError(t, common.Unmarshal(data, &upstream))

			assert.Equal(t, float64(241), upstream["num_frames"])
			assert.Equal(t, float64(agnesPreferredFrameRate), upstream["frame_rate"])
			assert.Equal(t, float64(1920), upstream["width"])
			assert.Equal(t, float64(1080), upstream["height"])
		})
	}
}

func TestAgnesProviderMapsUnifiedReferenceImageToUpstreamImage(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{
		"model":"agnes-video-v2.0",
		"prompt":"animate the reference image",
		"referenceImages":[" https://example.com/reference.jpg?token=signed "],
		"duration":10,
		"resolution":"720p",
		"ratio":"16:9"
	}`)
	info.ChannelSetting.VideoProtocol = dto.VideoProtocolAgnesVideoV2
	info.UpstreamModelName = "agnes-video-v2.0"
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

	request, err := relaycommon.GetTaskRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://example.com/reference.jpg?token=signed"}, request.ReferenceImages)

	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	var upstream map[string]any
	require.NoError(t, common.Unmarshal(data, &upstream))

	assert.Equal(t, "https://example.com/reference.jpg?token=signed", upstream["image"])
	assert.NotContains(t, upstream, "referenceImages")
	assert.NotContains(t, upstream, "images")
	assert.NotContains(t, upstream, "input_reference")
}

func TestAgnesProviderRejectsProviderSpecificReferenceImageFields(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "image", body: `{"model":"agnes-video-v2.0","prompt":"demo","image":"https://example.com/reference.jpg"}`},
		{name: "images", body: `{"model":"agnes-video-v2.0","prompt":"demo","images":["https://example.com/reference.jpg"]}`},
		{name: "input reference", body: `{"model":"agnes-video-v2.0","prompt":"demo","input_reference":"https://example.com/reference.jpg"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, info := newSeedanceTestContext(t, test.body)
			info.ChannelSetting.VideoProtocol = dto.VideoProtocolAgnesVideoV2
			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
			require.NotNil(t, taskErr)
			assert.Equal(t, "invalid_reference_images", taskErr.Code)
			assert.Contains(t, taskErr.Message, "referenceImages")
		})
	}
}

func TestAgnesProviderValidatesUnifiedReferenceImages(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
	}{
		{
			name: "multiple images",
			body: `{"model":"agnes-video-v2.0","prompt":"demo","referenceImages":["https://example.com/1.jpg","https://example.com/2.jpg"]}`,
		},
		{
			name: "empty URL",
			body: `{"model":"agnes-video-v2.0","prompt":"demo","referenceImages":["  "]}`,
		},
		{
			name: "non HTTP URL",
			body: `{"model":"agnes-video-v2.0","prompt":"demo","referenceImages":["file:///tmp/reference.jpg"]}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, info := newSeedanceTestContext(t, test.body)
			info.ChannelSetting.VideoProtocol = dto.VideoProtocolAgnesVideoV2
			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
			require.NotNil(t, taskErr)
			assert.Equal(t, "invalid_reference_images", taskErr.Code)
			if test.name == "multiple images" {
				assert.Equal(t, "Agnes supports at most one reference image", taskErr.Message)
			}
		})
	}
}

func TestAgnesProviderRejectsMultipartReferenceImages(t *testing.T) {
	for _, test := range []struct {
		name      string
		fieldName string
		file      bool
	}{
		{name: "URL field", fieldName: "referenceImages"},
		{name: "provider image field", fieldName: "image"},
		{name: "uploaded file", fieldName: "referenceImages", file: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			require.NoError(t, writer.WriteField("model", "agnes-video-v2.0"))
			require.NoError(t, writer.WriteField("prompt", "animate the reference image"))
			if test.file {
				part, err := writer.CreateFormFile(test.fieldName, "reference.jpg")
				require.NoError(t, err)
				_, err = part.Write([]byte("fake image"))
				require.NoError(t, err)
			} else {
				require.NoError(t, writer.WriteField(test.fieldName, "https://example.com/reference.jpg"))
			}
			require.NoError(t, writer.Close())

			req := httptest.NewRequest(http.MethodPost, "/v1/videos", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = req
			_, err := common.GetBodyStorage(ctx)
			require.NoError(t, err)
			t.Cleanup(func() {
				common.CleanupBodyStorage(ctx)
			})

			info := &relaycommon.RelayInfo{
				ChannelMeta:   &relaycommon.ChannelMeta{},
				TaskRelayInfo: &relaycommon.TaskRelayInfo{},
			}
			info.ChannelSetting.VideoProtocol = dto.VideoProtocolAgnesVideoV2
			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
			require.NotNil(t, taskErr)
			assert.Equal(t, "invalid_reference_images", taskErr.Code)
			assert.Contains(t, taskErr.Message, "application/json")
		})
	}
}

func TestAgnesProviderNormalizesSecondsAliasAndRejectsConflicts(t *testing.T) {
	for _, test := range []struct {
		name     string
		body     string
		wantCode string
		want     int
	}{
		{name: "seconds alias", body: `{"model":"agnes-video-v2","prompt":"demo","seconds":"10","resolution":"720p"}`, want: 10},
		{name: "matching fields", body: `{"model":"agnes-video-v2","prompt":"demo","duration":10,"seconds":"10","resolution":"720p"}`, want: 10},
		{name: "conflicting fields", body: `{"model":"agnes-video-v2","prompt":"demo","duration":5,"seconds":"10","resolution":"720p"}`, wantCode: "duration_conflict"},
		{name: "non-integer alias", body: `{"model":"agnes-video-v2","prompt":"demo","seconds":"ten","resolution":"720p"}`, wantCode: "invalid_seconds"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, info := newSeedanceTestContext(t, test.body)
			info.ChannelSetting.VideoProtocol = dto.VideoProtocolAgnesVideoV2
			adaptor := &TaskAdaptor{}
			taskErr := adaptor.ValidateRequestAndSetAction(ctx, info)
			if test.wantCode != "" {
				require.NotNil(t, taskErr)
				assert.Equal(t, test.wantCode, taskErr.Code)
				return
			}
			require.Nil(t, taskErr)
			request, err := relaycommon.GetTaskRequest(ctx)
			require.NoError(t, err)
			assert.Equal(t, test.want, request.Duration)
			assert.Empty(t, request.Seconds)
		})
	}
}

func TestVideoRequestProviderOptionsRejectsWrongNamespace(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{
		"model":"agnes-video-v2",
		"prompt":"demo",
		"provider_options":{"other":{"custom_flag":true}}
	}`)
	info.ChannelSetting.VideoProtocol = dto.VideoProtocolAgnesVideoV2
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_provider_options", taskErr.Code)
	assert.Equal(t, `provider_options contains an unsupported namespace "other"`, taskErr.Message)
	assert.NotContains(t, taskErr.Message, string(dto.VideoProtocolAgnesVideoV2))
}

func TestVideoRequestProviderOptionsRejectsProtectedOverrides(t *testing.T) {
	for _, field := range []string{"model", "mode", "image", "referenceImages", "duration", "seconds", "num_frames", "frame_rate", "width", "height", "resolution", "n", "callback_url", "authorization"} {
		t.Run(field, func(t *testing.T) {
			ctx, info := newSeedanceTestContext(t, fmt.Sprintf(`{
				"model":"agnes-video-v2",
				"prompt":"demo",
				"provider_options":{"agnes":{%q:0}}
			}`, field))
			info.ChannelSetting.VideoProtocol = dto.VideoProtocolAgnesVideoV2
			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
			require.NotNil(t, taskErr)
			assert.Equal(t, "provider_option_conflict", taskErr.Code)
		})
	}
}

func TestVideoRequestProviderOptionsRejectsTopLevelCollision(t *testing.T) {
	ctx, info := newSeedanceTestContext(t, `{
		"model":"agnes-video-v2",
		"prompt":"demo",
		"metadata":{"source":"client"},
		"provider_options":{"agnes":{"metadata":{"source":"provider"}}}
	}`)
	info.ChannelSetting.VideoProtocol = dto.VideoProtocolAgnesVideoV2
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "provider_option_conflict", taskErr.Code)
}

func TestVideoRequestProviderOptionsRejectsNestedBillingOverride(t *testing.T) {
	for _, field := range []string{"duration", "num_frames", "frame_rate"} {
		t.Run(field, func(t *testing.T) {
			ctx, info := newSeedanceTestContext(t, fmt.Sprintf(`{
				"model":"agnes-video-v2",
				"prompt":"demo",
				"duration":10,
				"provider_options":{"agnes":{"parameters":{%q:999999}}}
			}`, field))
			info.ChannelSetting.VideoProtocol = dto.VideoProtocolAgnesVideoV2
			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
			require.NotNil(t, taskErr)
			assert.Equal(t, "provider_option_conflict", taskErr.Code)
		})
	}
}

func TestParseTaskResultSupportsDurationNumberAndString(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want int
	}{
		{name: "number", body: `{"status":"completed","duration":10}`, want: 10},
		{name: "string", body: `{"status":"completed","duration":"12"}`, want: 12},
		{name: "duration preferred", body: `{"status":"completed","duration":8,"seconds":"6"}`, want: 8},
		{name: "seconds fallback", body: `{"status":"completed","duration":"invalid","seconds":"7"}`, want: 7},
		{name: "decimal seconds fallback", body: `{"status":"completed","seconds":"10.0"}`, want: 10},
		{name: "NaN seconds ignored", body: `{"status":"completed","seconds":"NaN"}`, want: 0},
		{name: "oversized ignored", body: `{"status":"completed","duration":3601,"seconds":"999999"}`, want: 0},
		{name: "negative ignored", body: `{"status":"completed","duration":-1,"seconds":"-2"}`, want: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := (&TaskAdaptor{}).ParseTaskResult([]byte(test.body))
			require.NoError(t, err)
			assert.Equal(t, test.want, result.Duration)
		})
	}
}

func TestParseTaskResultUsesAgnesNormalizedResolution(t *testing.T) {
	result, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"status":"completed",
		"seconds":"10.0",
		"size":"1088x832",
		"metadata":{"size_mapping":{"resolution":"720p","ratio":"4:3"}}
	}`))
	require.NoError(t, err)
	assert.Equal(t, 10, result.Duration)
	assert.Equal(t, "720p", result.Resolution)
}

func TestSeedanceCompletionIgnoresUnknownResolution(t *testing.T) {
	dimensions := (&TaskAdaptor{}).AdjustBillingDimensionsOnComplete(nil, &relaycommon.TaskInfo{
		Duration:   8,
		Resolution: "unknown-cheap-tier",
	})
	require.NotNil(t, dimensions)
	assert.Equal(t, float64(8), dimensions.Seconds)
	assert.Empty(t, dimensions.ResolutionTier)
}

func TestSeedanceCompletionRejectsResolutionOutsideFrozenCapabilities(t *testing.T) {
	task := &model.Task{
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				VideoProtocol:           dto.VideoProtocolMegabyAI,
				VideoAllowedResolutions: []string{"720p"},
			},
		},
	}
	dimensions := (&TaskAdaptor{}).AdjustBillingDimensionsOnComplete(task, &relaycommon.TaskInfo{
		Resolution: "1080p",
	})

	assert.Nil(t, dimensions)
}

func TestAgnesCompletionUsesAgnesDurationAndResolutionBounds(t *testing.T) {
	task := &model.Task{
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				VideoProtocol:           dto.VideoProtocolAgnesVideoV2,
				VideoAllowedResolutions: []string{"1080p"},
			},
		},
	}
	dimensions := (&TaskAdaptor{}).AdjustBillingDimensionsOnComplete(task, &relaycommon.TaskInfo{
		Duration:   18,
		Resolution: "1080p",
	})
	require.NotNil(t, dimensions)
	assert.Equal(t, float64(18), dimensions.Seconds)
	assert.Equal(t, "1080p", dimensions.ResolutionTier)

	dimensions = (&TaskAdaptor{}).AdjustBillingDimensionsOnComplete(task, &relaycommon.TaskInfo{
		Duration:   19,
		Resolution: "4k",
	})
	assert.Nil(t, dimensions)
}

func TestSeedanceCompletionUsesMappedUpstreamModelCapabilities(t *testing.T) {
	task := &model.Task{
		Properties: model.Properties{
			OriginModelName:   "public-seedance-alias",
			UpstreamModelName: "videos-mini",
		},
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				VideoProtocol:           dto.VideoProtocolMegabyAI,
				VideoMinDurationSeconds: 4,
				VideoMaxDurationSeconds: 15,
			},
		},
	}
	dimensions := (&TaskAdaptor{}).AdjustBillingDimensionsOnComplete(task, &relaycommon.TaskInfo{
		Duration:   10,
		Resolution: "1080p",
	})
	require.NotNil(t, dimensions)
	assert.Equal(t, float64(10), dimensions.Seconds)
	assert.Empty(t, dimensions.ResolutionTier)
}

func TestSeedanceCompletionUsesFrozenConfiguredCapabilities(t *testing.T) {
	task := &model.Task{
		Properties: model.Properties{UpstreamModelName: "new-upstream-seedance"},
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				VideoProtocol:           dto.VideoProtocolMegabyAI,
				VideoAllowedResolutions: []string{"720p", "1080p"},
				VideoMinDurationSeconds: 4,
				VideoMaxDurationSeconds: 29,
			},
		},
	}

	dimensions := (&TaskAdaptor{}).AdjustBillingDimensionsOnComplete(task, &relaycommon.TaskInfo{
		Duration:   29,
		Resolution: "1080p",
	})
	require.NotNil(t, dimensions)
	assert.Equal(t, float64(29), dimensions.Seconds)
	assert.Equal(t, "1080p", dimensions.ResolutionTier)

	dimensions = (&TaskAdaptor{}).AdjustBillingDimensionsOnComplete(task, &relaycommon.TaskInfo{
		Duration:   30,
		Resolution: "1080p",
	})
	require.NotNil(t, dimensions)
	assert.Zero(t, dimensions.Seconds)
	assert.Equal(t, "1080p", dimensions.ResolutionTier)

	dimensions = (&TaskAdaptor{}).AdjustBillingDimensionsOnComplete(task, &relaycommon.TaskInfo{
		Duration:   10,
		Resolution: "4k",
	})
	require.NotNil(t, dimensions)
	assert.Empty(t, dimensions.ResolutionTier)
}
