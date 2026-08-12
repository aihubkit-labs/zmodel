package sora

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMinimaxH3TestContext(t *testing.T, body string) (*gin.Context, *relaycommon.RelayInfo) {
	t.Helper()
	ctx, info := newSeedanceTestContext(t, body)
	configureTestVideoModel(info, dto.VideoProtocolMegabyAI, "minimax-h3", []string{"1440p"}, 5, 0, 3, 5, 15)
	return ctx, info
}

func TestMinimaxH3StableJSONContractAndUpstreamMapping(t *testing.T) {
	body := `{
		"model":"public-minimax-h3",
		"prompt":"Create a cinematic performance",
		"duration":8,
		"resolution":"1440P",
		"ratio":"21:9",
		"generate_audio":true,
		"watermark":false,
		"first_image":"https://media.example.com/start.png",
		"last_image":"https://media.example.com/end.png"
	}`
	ctx, info := newSeedanceTestContext(t, body)
	info.UpstreamModelName = "minimax-h3"
	configureTestVideoModel(info, dto.VideoProtocolMegabyAI, "minimax-h3", []string{"1440p"}, 5, 0, 3, 5, 15)
	capability := info.ChannelSetting.VideoModelCapabilities["minimax-h3"]
	capability.ResolutionMappings = map[string]string{"1440p": "2k"}
	capability.FixedParameters = map[string]any{"quality": "high"}
	capability.OmitParameters = []string{"watermark"}
	info.ChannelSetting.VideoModelCapabilities["minimax-h3"] = capability
	adaptor := &TaskAdaptor{}

	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	dimensions, err := adaptor.EstimateBillingDimensions(ctx, info)
	require.NoError(t, err)
	assert.Equal(t, float64(8), dimensions.Seconds)
	assert.Equal(t, "1440p", dimensions.ResolutionTier)

	requestBody, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	upstreamBody, err := io.ReadAll(requestBody)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"model":"minimax-h3",
		"prompt":"Create a cinematic performance",
		"duration":8,
		"resolution":"2k",
		"ratio":"21:9",
		"generate_audio":true,
		"quality":"high",
		"first_image":"https://media.example.com/start.png",
		"last_image":"https://media.example.com/end.png"
	}`, string(upstreamBody))
}

func TestMinimaxH3RejectsUnsupportedFieldsAndCapabilityCombinations(t *testing.T) {
	tests := []struct {
		name string
		body string
		code string
	}{
		{
			name: "missing required native audio",
			body: `{"model":"minimax-h3","prompt":"demo","duration":5,"resolution":"1440p","ratio":"16:9"}`,
			code: "invalid_generate_audio",
		},
		{
			name: "unsupported ratio",
			body: `{"model":"minimax-h3","prompt":"demo","duration":5,"resolution":"1440p","ratio":"2:1","generate_audio":true}`,
			code: "invalid_ratio",
		},
		{
			name: "duration below configured minimum",
			body: `{"model":"minimax-h3","prompt":"demo","duration":4,"resolution":"1440p","ratio":"16:9","generate_audio":true}`,
			code: "invalid_seconds",
		},
		{
			name: "duration above configured maximum",
			body: `{"model":"minimax-h3","prompt":"demo","duration":16,"resolution":"1440p","ratio":"16:9","generate_audio":true}`,
			code: "invalid_seconds",
		},
		{
			name: "last frame without first frame",
			body: `{"model":"minimax-h3","prompt":"demo","duration":5,"resolution":"1440p","ratio":"16:9","generate_audio":true,"last_image":"https://example.com/end.png"}`,
			code: "invalid_last_image",
		},
		{
			name: "reference images combined with frames",
			body: `{"model":"minimax-h3","prompt":"demo","duration":5,"resolution":"1440p","ratio":"16:9","generate_audio":true,"referenceImages":["https://example.com/ref.png"],"first_image":"https://example.com/start.png"}`,
			code: "invalid_reference_images",
		},
		{
			name: "reference audio without ordinary reference image",
			body: `{"model":"minimax-h3","prompt":"demo","duration":5,"resolution":"1440p","ratio":"16:9","generate_audio":true,"referenceAudios":["https://example.com/ref.mp3"]}`,
			code: "invalid_reference_audios",
		},
		{
			name: "reference video is unsupported",
			body: `{"model":"minimax-h3","prompt":"demo","duration":5,"resolution":"1440p","ratio":"16:9","generate_audio":true,"referenceVideos":["https://example.com/ref.mp4"]}`,
			code: "invalid_reference_videos",
		},
		{
			name: "upstream aliases are not public contract",
			body: `{"model":"minimax-h3","prompt":"demo","duration":5,"resolution":"1440p","ratio":"16:9","generate_audio":true,"references":[]}`,
			code: "unsupported_parameter",
		},
		{
			name: "generation mode is selected by public model",
			body: `{"model":"minimax-h3","prompt":"demo","duration":5,"resolution":"1440p","ratio":"16:9","generate_audio":true,"generation_mode":"quality"}`,
			code: "unsupported_parameter",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, info := newMinimaxH3TestContext(t, test.body)
			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
			require.NotNil(t, taskErr)
			assert.Equal(t, test.code, taskErr.Code)
		})
	}
}

func TestMinimaxH3CapabilitySwitchesAreDynamic(t *testing.T) {
	ctx, info := newMinimaxH3TestContext(t, `{
		"model":"minimax-h3",
		"prompt":"demo",
		"duration":5,
		"resolution":"1440p",
		"ratio":"16:9",
		"first_image":"https://example.com/start.png"
	}`)
	capability := info.ChannelSetting.VideoModelCapabilities["minimax-h3"]
	capability.SupportsGenerateAudio = common.GetPointer(false)
	capability.GenerateAudioRequired = common.GetPointer(false)
	capability.SupportsFirstFrame = common.GetPointer(false)
	info.ChannelSetting.VideoModelCapabilities["minimax-h3"] = capability

	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_first_image", taskErr.Code)
	assert.Equal(t, dto.VideoParameterErrorData{
		Parameter: "first_image",
		Received:  "https://example.com/start.png",
	}, taskErr.Data)
}

func TestMinimaxH3RequiredFrameErrorIncludesParameterContract(t *testing.T) {
	ctx, info := newMinimaxH3TestContext(t, `{
		"model":"minimax-h3",
		"prompt":"demo",
		"duration":5,
		"resolution":"1440p",
		"ratio":"16:9",
		"generate_audio":true
	}`)
	capability := info.ChannelSetting.VideoModelCapabilities["minimax-h3"]
	capability.FirstFrameRequired = common.GetPointer(true)
	info.ChannelSetting.VideoModelCapabilities["minimax-h3"] = capability

	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_first_image", taskErr.Code)
	assert.Equal(t, dto.VideoParameterErrorData{
		Parameter: "first_image",
		Required:  common.GetPointer(true),
	}, taskErr.Data)
}

func TestMinimaxH3UsesPlatformTaskIDAsUpstreamIdempotencyKey(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "https://upstream.example/v1/videos", nil)
	request.Header.Set("Idempotency-Key", "client-controlled")
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	ctx.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_public_123"},
	}
	info.ChannelSetting.VideoProtocol = dto.VideoProtocolMegabyAI

	require.NoError(t, (&TaskAdaptor{apiKey: "upstream-key"}).BuildRequestHeader(ctx, request, info))
	assert.Equal(t, "zmodel:task_public_123", request.Header.Get("Idempotency-Key"))
	assert.Equal(t, "Bearer upstream-key", request.Header.Get("Authorization"))
}

func TestMinimaxH3MultipartFilesAreCountedAndForwarded(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range map[string]string{
		"model":          "minimax-h3",
		"prompt":         "Use the image and audio references",
		"duration":       "5",
		"resolution":     "1440p",
		"ratio":          "16:9",
		"generate_audio": "true",
	} {
		require.NoError(t, writer.WriteField(name, value))
	}
	imagePart, err := writer.CreateFormFile("referenceImageFiles", "subject.png")
	require.NoError(t, err)
	_, err = imagePart.Write([]byte("image-content"))
	require.NoError(t, err)
	audioPart, err := writer.CreateFormFile("referenceAudioFiles", "voice.wav")
	require.NoError(t, err)
	_, err = audioPart.Write([]byte("audio-content"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader(body.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = request
	_, err = common.GetBodyStorage(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { common.CleanupBodyStorage(ctx) })
	info := &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}
	info.UpstreamModelName = "minimax-h3"
	configureTestVideoModel(info, dto.VideoProtocolMegabyAI, "minimax-h3", []string{"1440p"}, 5, 0, 3, 5, 15)
	adaptor := &TaskAdaptor{}

	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	parsed, err := relaycommon.GetTaskRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, parsed.ReferenceImageFiles)
	assert.Equal(t, 1, parsed.ReferenceAudioFiles)

	requestBody, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	forwarded, err := io.ReadAll(requestBody)
	require.NoError(t, err)
	_, params, err := mime.ParseMediaType(ctx.Request.Header.Get("Content-Type"))
	require.NoError(t, err)
	forwardedForm, err := multipart.NewReader(bytes.NewReader(forwarded), params["boundary"]).ReadForm(1 << 20)
	require.NoError(t, err)
	defer forwardedForm.RemoveAll()
	assert.Equal(t, []string{"minimax-h3"}, forwardedForm.Value["model"])
	assert.Equal(t, []string{"5"}, forwardedForm.Value["duration"])
	assert.Equal(t, []string{"1440p"}, forwardedForm.Value["resolution"])
	assert.Equal(t, []string{"16:9"}, forwardedForm.Value["ratio"])
	assert.Len(t, forwardedForm.File["referenceImageFiles"], 1)
	assert.Len(t, forwardedForm.File["referenceAudioFiles"], 1)
}
