package sora

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newGlobalAIOpcTestContext(t *testing.T, body string) (*gin.Context, *relaycommon.RelayInfo) {
	t.Helper()
	ctx, info := newSeedanceTestContext(t, body)
	info.UpstreamModelName = "minimax-h3"
	configureTestVideoModel(info, dto.VideoProtocolGlobalAIOpc, "minimax-h3", []string{"1440p"}, 5, 0, 1, 5, 15)
	capability := info.ChannelSetting.VideoModelCapabilities["minimax-h3"]
	capability.LastFrameRequiresFirstFrame = common.GetPointer(false)
	capability.ReferenceImagesIncompatibleWithFrames = common.GetPointer(false)
	info.ChannelSetting.VideoModelCapabilities["minimax-h3"] = capability
	return ctx, info
}

func TestGlobalAIOpcMapsUnifiedRequestToUpstreamContract(t *testing.T) {
	ctx, info := newGlobalAIOpcTestContext(t, `{
		"model":"minimax-h3",
		"prompt":"Create a cinematic performance",
		"duration":8,
		"resolution":"1440P",
		"ratio":"21:9",
		"generate_audio":true,
		"watermark":false,
		"referenceImages":["https://media.example.com/subject.png"],
		"referenceAudios":["https://media.example.com/voice.mp3"],
		"first_image":"https://media.example.com/start.png",
		"last_image":"https://media.example.com/end.png"
	}`)
	adaptor := &TaskAdaptor{}

	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	requestBody, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	upstreamBody, err := io.ReadAll(requestBody)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"model":"minimax-h3",
		"prompt":"Create a cinematic performance",
		"duration":8,
		"resolution":"2k",
		"aspect_ratio":"21:9",
		"generate_audio":true,
		"watermark":false,
		"reference_images":["https://media.example.com/subject.png"],
		"reference_audios":["https://media.example.com/voice.mp3"],
		"first_image":"https://media.example.com/start.png",
		"last_image":"https://media.example.com/end.png"
	}`, string(upstreamBody))
}

func TestGlobalAIOpcUsesProviderTaskEndpoints(t *testing.T) {
	service.InitHttpClient()
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}, TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
	info.ChannelSetting.VideoProtocol = dto.VideoProtocolGlobalAIOpc
	adaptor := &TaskAdaptor{baseURL: "https://zcbservice.aizfw.cn/", apiKey: "upstream-key"}

	createURL, err := adaptor.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://zcbservice.aizfw.cn/kyyReactApiServer/v2/model-center/tasks", createURL)

	adaptor.baseURL = "https://zcbservice.aizfw.cn/kyyReactApiServer/"
	createURL, err = adaptor.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://zcbservice.aizfw.cn/kyyReactApiServer/v2/model-center/tasks", createURL)

	var requestPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		assert.Equal(t, "Bearer upstream-key", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"upstream-task","status":"processing"}`))
	}))
	t.Cleanup(server.Close)
	response, err := adaptor.FetchTask(server.URL+"/kyyReactApiServer", "upstream-key", map[string]any{
		"task_id":        "upstream-task",
		"video_protocol": dto.VideoProtocolGlobalAIOpc,
	}, "")
	require.NoError(t, err)
	response.Body.Close()
	assert.Equal(t, "/kyyReactApiServer/v2/model-center/tasks/upstream-task", requestPath)
}

func TestGlobalAIOpcReportsHTTP200BusinessFailureWithoutLeakingUpstreamDetails(t *testing.T) {
	ctx, info := newGlobalAIOpcTestContext(t, `{"model":"minimax-h3","prompt":"demo","duration":5,"resolution":"1440p","ratio":"16:9","generate_audio":true}`)
	info.PublicTaskID = "task_public"
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"code":500,
			"data":null,
			"msg":"No handler found for POST /kyyReactApiServer/kyyReactApiServer/v2/model-center/tasks"
		}`)),
	}

	_, _, taskErr := (&TaskAdaptor{}).DoResponse(ctx, response, info)

	require.NotNil(t, taskErr)
	assert.Equal(t, "upstream_request_failed", taskErr.Code)
	assert.Equal(t, http.StatusBadGateway, taskErr.StatusCode)
	assert.Equal(t, "upstream rejected the video task request", taskErr.Message)
	assert.NotContains(t, taskErr.Message, "kyyReactApiServer")
}

func TestGlobalAIOpcParsesResultAndReturnsOnlyUnifiedFields(t *testing.T) {
	result, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"data": {
			"id":"upstream-task",
			"status":"completed",
			"progress":100,
			"result_url":"https://upstream.example/result.mp4",
			"video_url":"https://upstream.example/fallback.mp4",
			"actualDuration":8,
			"resolution":"2k",
			"amount":0.32,
			"totalTokens":"123456",
			"usage":{"output_tokens":"123456","total_tokens":"999999"}
		}
	}`))
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, result.Status)
	assert.Equal(t, "https://upstream.example/result.mp4", result.Url)
	assert.Equal(t, 8, result.Duration)
	assert.Equal(t, "2k", result.Resolution)
	require.NotNil(t, result.TokenUsage)
	require.NotNil(t, result.TokenUsage.TotalTokens)
	assert.Equal(t, int64(123456), *result.TokenUsage.TotalTokens)

	result, err = (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"data": {
			"status":"completed",
			"usage":{"total_tokens":"654321"}
		}
	}`))
	require.NoError(t, err)
	require.NotNil(t, result.TokenUsage)
	require.NotNil(t, result.TokenUsage.TotalTokens)
	assert.Equal(t, int64(654321), *result.TokenUsage.TotalTokens)

	task := &model.Task{
		TaskID:   "task_public",
		Status:   model.TaskStatusSuccess,
		Progress: "100%",
		Properties: model.Properties{Input: `{
			"model":"public-minimax-h3",
			"duration":8,
			"resolution":"1440p",
			"ratio":"21:9"
		}`},
		PrivateData: model.TaskPrivateData{BillingContext: &model.TaskBillingContext{
			VideoProtocol: dto.VideoProtocolGlobalAIOpc,
		}},
		Data: []byte(`{
			"id":"upstream-task",
			"status":"completed",
			"progress":100,
			"result_url":"https://upstream.example/result.mp4",
			"actualDuration":8,
			"resolution":"2k",
			"amount":0.32
		}`),
	}
	publicBody, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, common.Unmarshal(publicBody, &payload))
	assert.Equal(t, "task_public", payload["id"])
	assert.Equal(t, "public-minimax-h3", payload["model"])
	assert.Equal(t, "1440p", payload["resolution"])
	assert.Equal(t, payload["url"], payload["video_url"])
	assert.NotContains(t, payload, "result_url")
	assert.NotContains(t, payload, "amount")
	assert.NotContains(t, payload, "actualDuration")
	assert.NotContains(t, string(publicBody), "upstream.example")
}

func TestStoredSoraResponseDoesNotPersistTemporaryVideoURLs(t *testing.T) {
	stored, err := sanitizeStoredVideoResponse([]byte(`{
		"id":"upstream-task",
		"status":"completed",
		"result_url":"https://upstream.example/result.mp4",
		"video_url":"https://upstream.example/video.mp4",
		"metadata":{"origin_video_url":"https://upstream.example/origin.mp4","request_id":"request-1"}
	}`))
	require.NoError(t, err)
	assert.NotContains(t, string(stored), "upstream.example")
	var payload map[string]any
	require.NoError(t, common.Unmarshal(stored, &payload))
	assert.NotContains(t, payload, "result_url")
	assert.NotContains(t, payload, "video_url")
	metadata, ok := payload["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "request-1", metadata["request_id"])
	assert.NotContains(t, metadata, "origin_video_url")
}

func TestGlobalAIOpcRejectsUnsupportedTransportAndRemixBeforeUpstream(t *testing.T) {
	ctx, info := newGlobalAIOpcTestContext(t, `{"model":"minimax-h3","prompt":"demo","duration":5,"resolution":"1440p","ratio":"16:9","generate_audio":true}`)
	ctx.Request.Header.Set("Content-Type", "multipart/form-data; boundary=test")
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "unsupported_content_type", taskErr.Code)

	remixCtx, remixInfo := newGlobalAIOpcTestContext(t, `{"model":"minimax-h3","prompt":"remix"}`)
	remixInfo.Action = constant.TaskActionRemix
	taskErr = (&TaskAdaptor{}).ValidateRequestAndSetAction(remixCtx, remixInfo)
	require.NotNil(t, taskErr)
	assert.Equal(t, "unsupported_operation", taskErr.Code)
	assert.NotContains(t, strings.ToLower(taskErr.Message), "globalaiopc")
}
