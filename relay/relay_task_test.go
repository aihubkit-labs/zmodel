package relay

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelayTaskSubmitValidatesMappedUpstreamVideoCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodPost, "/v1/videos", strings.NewReader(`{
		"model":"seedance-2.5",
		"prompt":"demo",
		"duration":15,
		"resolution":"1080p",
		"ratio":"16:9"
	}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req
	_, err := common.GetBodyStorage(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { common.CleanupBodyStorage(ctx) })

	zero := 0
	minDuration := 4
	maxDuration := 30
	falseValue := false
	settings := dto.ChannelSettings{
		VideoProtocol: dto.VideoProtocolGlobalAIOpc,
		VideoModelCapabilities: map[string]dto.VideoModelCapability{
			"seedance-2.5-c1": {
				Resolutions:                           []string{"480p", "720p"},
				Ratios:                                []string{"16:9"},
				RatioRequired:                         &falseValue,
				MinReferenceImages:                    &zero,
				MaxReferenceImages:                    &zero,
				MinReferenceVideos:                    &zero,
				MaxReferenceVideos:                    &zero,
				MinReferenceAudios:                    &zero,
				MaxReferenceAudios:                    &zero,
				SupportsDuration:                      common.GetPointer(true),
				DurationRequired:                      common.GetPointer(true),
				MinDurationSeconds:                    &minDuration,
				MaxDurationSeconds:                    &maxDuration,
				SupportsGenerateAudio:                 &falseValue,
				GenerateAudioRequired:                 &falseValue,
				SupportsFirstFrame:                    common.GetPointer(true),
				SupportsLastFrame:                     common.GetPointer(true),
				LastFrameRequiresFirstFrame:           &falseValue,
				ReferenceImagesIncompatibleWithFrames: &falseValue,
				AudioReferenceRequiresVisualReference: &falseValue,
			},
		},
	}
	common.SetContextKey(ctx, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
	common.SetContextKey(ctx, constant.ContextKeyChannelSetting, settings)
	common.SetContextKey(ctx, constant.ContextKeyChannelModelMapping, `{"seedance-2.5":"seedance-2.5-c1"}`)

	info := &relaycommon.RelayInfo{
		OriginModelName: "seedance-2.5",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
	}
	_, taskErr := RelayTaskSubmit(ctx, info)

	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_resolution", taskErr.Code)
	assert.Contains(t, taskErr.Message, `video model "seedance-2.5"`)
	assert.NotContains(t, taskErr.Message, "seedance-2.5-c1")
	assert.Equal(t, "seedance-2.5-c1", info.UpstreamModelName)
}

func TestTaskSubmitUpstreamErrorUsesStructuredPublicFields(t *testing.T) {
	taskErr := taskSubmitUpstreamError([]byte(`{
		"error": {
			"code": "invalid_request",
			"message": "resolution must be a quality label such as 720p or 4k",
			"type": "new_api_error"
		}
	}`), http.StatusBadRequest)

	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_request", taskErr.Code)
	assert.Equal(t, "resolution must be a quality label such as 720p or 4k", taskErr.Message)
	assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
	assert.NotContains(t, taskErr.Message, `{"error"`)
}

func TestTaskSubmitUpstreamErrorHidesInternalProtocolName(t *testing.T) {
	taskErr := taskSubmitUpstreamError([]byte(`{"error":{"code":"provider_error","message":"Lingganya request failed"}}`), http.StatusBadGateway)

	require.NotNil(t, taskErr)
	assert.Equal(t, "provider_error", taskErr.Code)
	assert.Equal(t, "upstream video request failed", taskErr.Message)
}

func TestTaskModel2DtoUsesPublicContentURLForVideoTask(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	originalAPIServerAddress, hadAPIServerAddress := common.OptionMap["ApiServerAddress"]
	common.OptionMap["ApiServerAddress"] = "https://api.example.com"
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		if hadAPIServerAddress {
			common.OptionMap["ApiServerAddress"] = originalAPIServerAddress
		} else {
			delete(common.OptionMap, "ApiServerAddress")
		}
	})

	task := &model.Task{
		TaskID: "task_public_id",
		Action: constant.TaskActionTextGenerate,
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://upstream.example/private-video.mp4",
		},
	}

	dtoTask := TaskModel2Dto(task)

	assert.Equal(t, "https://api.example.com/v1/videos/task_public_id/content", dtoTask.ResultURL)
}

func TestTaskModel2DtoAlwaysUsesPublicContentURL(t *testing.T) {
	task := &model.Task{
		TaskID: "task_delivery_policy", Action: constant.TaskActionGenerate,
		Status: model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://upstream.example/video.mp4",
		},
	}

	assert.Equal(t, taskcommon.BuildProxyURL(task.TaskID), TaskModel2Dto(task).ResultURL)
}

func TestTaskModel2DtoProvidesReadablePlatformName(t *testing.T) {
	task := &model.Task{
		Platform: constant.TaskPlatform("1"),
	}

	dtoTask := TaskModel2Dto(task)

	assert.Equal(t, "OpenAI", dtoTask.PlatformName)
}

func TestTaskModel2DtoHidesUpstreamModel(t *testing.T) {
	task := &model.Task{
		Properties: model.Properties{
			OriginModelName:   "public-model",
			UpstreamModelName: "provider-model-id",
		},
	}

	dtoTask := TaskModel2Dto(task)

	properties, ok := dtoTask.Properties.(model.Properties)
	require.True(t, ok)
	assert.Equal(t, "public-model", properties.OriginModelName)
	assert.Empty(t, properties.UpstreamModelName)
	assert.Equal(t, "provider-model-id", task.Properties.UpstreamModelName)
}

func TestTaskModel2DtoReportsTerminalTasksAsComplete(t *testing.T) {
	for _, status := range []model.TaskStatus{model.TaskStatusSuccess, model.TaskStatusFailure} {
		t.Run(string(status), func(t *testing.T) {
			task := &model.Task{
				Status:   status,
				Progress: "30%",
			}

			dtoTask := TaskModel2Dto(task)

			assert.Equal(t, taskcommon.ProgressComplete, dtoTask.Progress)
			assert.Equal(t, "30%", task.Progress)
		})
	}
}

func TestTaskModel2DtoKeepsNonVideoResultURL(t *testing.T) {
	task := &model.Task{
		TaskID: "task_audio_id",
		Action: constant.SunoActionMusic,
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://example.com/audio.mp3",
		},
	}

	dtoTask := TaskModel2Dto(task)

	assert.Equal(t, "https://example.com/audio.mp3", dtoTask.ResultURL)
}

func TestTaskModel2DtoRewritesVideoURLsWithoutMutatingStoredData(t *testing.T) {
	originalServerAddress := system_setting.ServerAddress
	system_setting.ServerAddress = "https://api.example.com"
	t.Cleanup(func() {
		system_setting.ServerAddress = originalServerAddress
	})

	storedData := []byte(`{
		"status":"completed",
		"url":"https://upstream.example/video.mp4",
		"video_url":"https://upstream.example/video.mp4",
		"metadata":{
			"url":"https://upstream.example/video.mp4",
			"content_url":"https://upstream.example/video.mp4",
			"local_url":"https://upstream.example/video.mp4",
			"video_url":"https://upstream.example/video.mp4",
			"final_video_url":"https://upstream.example/video.mp4",
			"origin_video_url":"https://origin.example/video.mp4",
			"cost_credits":70
		}
	}`)
	task := &model.Task{
		TaskID: "task_public_id",
		Action: constant.TaskActionTextGenerate,
		Status: model.TaskStatusSuccess,
		Data:   append([]byte(nil), storedData...),
	}

	dtoTask := TaskModel2Dto(task)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(dtoTask.Data, &payload))
	expectedURL := taskcommon.BuildProxyURL(task.TaskID)
	assert.Equal(t, expectedURL, payload["url"])
	assert.Equal(t, expectedURL, payload["video_url"])
	metadata, ok := payload["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, expectedURL, metadata["url"])
	assert.Equal(t, expectedURL, metadata["content_url"])
	assert.Equal(t, expectedURL, metadata["local_url"])
	assert.Equal(t, expectedURL, metadata["video_url"])
	assert.Equal(t, expectedURL, metadata["final_video_url"])
	assert.NotContains(t, metadata, "origin_video_url")
	assert.Equal(t, float64(70), metadata["cost_credits"])
	assert.Equal(t, storedData, []byte(task.Data))
}

func TestTaskModel2DtoRemovesVideoURLsBeforeCompletion(t *testing.T) {
	task := &model.Task{
		TaskID: "task_public_id",
		Action: constant.TaskActionGenerate,
		Status: model.TaskStatusInProgress,
		Data: []byte(`{
			"status":"processing",
			"url":"https://upstream.example/video.mp4",
			"video_url":"https://upstream.example/video.mp4",
			"origin_video_url":"https://origin.example/video.mp4",
			"metadata":{
				"url":"https://upstream.example/video.mp4",
				"origin_video_url":"https://origin.example/video.mp4",
				"request_id":"req_123"
			}
		}`),
	}

	dtoTask := TaskModel2Dto(task)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(dtoTask.Data, &payload))
	assert.NotContains(t, payload, "url")
	assert.NotContains(t, payload, "video_url")
	assert.NotContains(t, payload, "origin_video_url")
	metadata, ok := payload["metadata"].(map[string]any)
	require.True(t, ok)
	assert.NotContains(t, metadata, "url")
	assert.NotContains(t, metadata, "origin_video_url")
	assert.Equal(t, "req_123", metadata["request_id"])
}

func TestTaskModel2DtoKeepsInvalidVideoDataUnchanged(t *testing.T) {
	task := &model.Task{
		TaskID: "task_public_id",
		Action: constant.TaskActionGenerate,
		Status: model.TaskStatusSuccess,
		Data:   []byte(`not-json`),
	}

	dtoTask := TaskModel2Dto(task)

	assert.Equal(t, task.Data, dtoTask.Data)
}

func TestTaskModel2DtoExposesActualTokenUsageWithoutPrivateData(t *testing.T) {
	totalTokens := int64(123456)
	task := &model.Task{
		TaskID: "task_token_usage",
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				ActualTokenUsage: &billingexpr.TokenUsage{TotalTokens: &totalTokens},
			},
		},
	}

	dtoTask := TaskModel2Dto(task)
	require.NotNil(t, dtoTask.TokenUsage)
	require.NotNil(t, dtoTask.TokenUsage.TotalTokens)
	assert.Equal(t, totalTokens, *dtoTask.TokenUsage.TotalTokens)
}

func TestRewriteStoredVideoURLsKeepsOnlyCanonicalPublicVideoLocations(t *testing.T) {
	data := []byte(`{
		"url":"https://upstream.example/video.mp4",
		"video_url":"https://upstream.example/video.mp4",
		"metadata":{
			"content_url":"https://upstream.example/video.mp4",
			"local_url":"https://upstream.example/video.mp4",
			"video_url":"https://upstream.example/video.mp4",
			"final_video_url":"https://upstream.example/video.mp4"
		}
	}`)
	const publicURL = "https://api.example.com/v1/videos/task_public/content"

	updated, err := rewriteStoredVideoURLs(data, publicURL)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, common.Unmarshal(updated, &payload))
	assert.Equal(t, publicURL, payload["url"])
	assert.Equal(t, publicURL, payload["video_url"])
	assert.NotContains(t, payload, "metadata")
}
