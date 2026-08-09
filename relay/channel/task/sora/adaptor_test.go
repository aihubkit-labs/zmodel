package sora

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoResponseNormalizesAgnesCreateResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	context.Set("task_request", relaycommon.TaskSubmitReq{
		Model:      "public-agnes-model",
		Duration:   10,
		Resolution: "720p",
		Ratio:      "16:9",
	})

	upstreamBody := `{
		"id":"task_agnes_upstream",
		"object":"video",
		"model":"agnes-video-v2.0",
		"status":"pending",
		"progress":0,
		"created_at":1785582142,
		"seconds":"10.0",
		"size":"1280x720",
		"metadata":{"provider_request_id":"req_agnes_123"}
	}`
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "public-agnes-model",
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task_zmodel_public",
		},
	}

	upstreamID, taskData, taskErr := (&TaskAdaptor{}).DoResponse(context, response, info)
	require.Nil(t, taskErr)
	assert.Equal(t, "task_agnes_upstream", upstreamID)
	assert.JSONEq(t, upstreamBody, string(taskData))

	var payload map[string]any
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	assert.Equal(t, "task_zmodel_public", payload["id"])
	assert.Equal(t, "task_zmodel_public", payload["task_id"])
	assert.Equal(t, "public-agnes-model", payload["model"])
	assert.Equal(t, dto.VideoStatusQueued, payload["status"])
	assert.Equal(t, float64(10), payload["duration"])
	assert.Equal(t, "10", payload["seconds"])
	assert.Equal(t, "720p", payload["resolution"])
	assert.Equal(t, "16:9", payload["ratio"])
	assert.Equal(t, "1280x720", payload["size"])
	metadata, ok := payload["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "req_agnes_123", metadata["provider_request_id"])
}

func TestDoResponseNormalizesSeedanceWithoutCompatibilityFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	context.Set("task_request", relaycommon.TaskSubmitReq{
		Model:      "public-seedance-model",
		Duration:   10,
		Resolution: "720p",
	})
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"id":"task_seedance_upstream","status":"queued"}`)),
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "public-seedance-model",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{PublicTaskID: "task_zmodel_seedance"},
	}

	_, _, taskErr := (&TaskAdaptor{}).DoResponse(context, response, info)
	require.Nil(t, taskErr)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	assert.Equal(t, float64(10), payload["duration"])
	assert.Equal(t, "10", payload["seconds"])
	assert.Equal(t, "720p", payload["resolution"])
	assert.NotContains(t, payload, "size")
	assert.NotContains(t, payload, "ratio")
}

func TestConvertToOpenAIVideoNormalizesAgnesQueryResponse(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_zmodel_public",
		Status:     model.TaskStatusSuccess,
		Progress:   "100%",
		SubmitTime: 1785582142,
		FinishTime: 1785582150,
		Properties: model.Properties{
			OriginModelName: "public-agnes-model",
			Input: `{
				"model":"public-agnes-model",
				"duration":10,
				"resolution":"720p",
				"ratio":"16:9"
			}`,
		},
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{VideoProtocol: dto.VideoProtocolAgnesVideoV2},
		},
		Data: []byte(`{
			"id":"task_agnes_upstream",
			"model":"agnes-video-v2.0",
			"status":"completed",
			"progress":100,
			"seconds":"5.0",
			"size":"1280x720",
			"metadata":{"provider_request_id":"req_agnes_123"}
		}`),
	}

	result, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(result, &payload))
	assert.Equal(t, task.TaskID, payload["id"])
	assert.Equal(t, task.TaskID, payload["task_id"])
	assert.Equal(t, "public-agnes-model", payload["model"])
	assert.Equal(t, dto.VideoStatusCompleted, payload["status"])
	assert.Equal(t, float64(100), payload["progress"])
	assert.Equal(t, float64(5), payload["duration"], "actual upstream duration must take precedence over the requested duration")
	assert.Equal(t, "5", payload["seconds"])
	assert.Equal(t, "720p", payload["resolution"])
	assert.Equal(t, "16:9", payload["ratio"])
	assert.Equal(t, "1280x720", payload["size"])
	assert.Equal(t, float64(task.SubmitTime), payload["created_at"])
	assert.Equal(t, float64(task.FinishTime), payload["completed_at"])
	metadata, ok := payload["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "req_agnes_123", metadata["provider_request_id"])
}

func TestConvertToOpenAIVideoAddsSeedanceResolutionFromRequestSnapshot(t *testing.T) {
	task := &model.Task{
		TaskID:   "task_zmodel_seedance",
		Status:   model.TaskStatusQueued,
		Progress: "0%",
		Properties: model.Properties{
			OriginModelName: "public-seedance-model",
			Input:           `{"model":"public-seedance-model","duration":10,"resolution":"720p"}`,
		},
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{VideoProtocol: dto.VideoProtocolSeedanceMegabyAI},
		},
		Data: []byte(`{
			"id":"task_seedance_upstream",
			"status":"queued",
			"seconds":"10",
			"size":"1280x720",
			"metadata":{"provider_request_id":"req_seedance_123"}
		}`),
	}

	result, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(result, &payload))
	assert.Equal(t, "public-seedance-model", payload["model"])
	assert.Equal(t, float64(10), payload["duration"])
	assert.Equal(t, "720p", payload["resolution"])
	assert.Equal(t, "1280x720", payload["size"])
	assert.NotContains(t, payload, "ratio")
}

func TestConvertToOpenAIVideoUsesFrozenResolutionWithInvalidRequestSnapshot(t *testing.T) {
	task := &model.Task{
		TaskID:   "task_historical",
		Status:   model.TaskStatusQueued,
		Progress: "0%",
		Properties: model.Properties{
			Input: "{invalid",
		},
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{VideoAllowedResolutions: []string{"720p"}},
		},
		Data: []byte(`{"id":"task_upstream","status":"queued","seconds":"8","size":"720P"}`),
	}

	result, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(result, &payload))
	assert.Equal(t, "task_historical", payload["id"])
	assert.Equal(t, float64(8), payload["duration"])
	assert.Equal(t, "720p", payload["resolution"])
}

func TestConvertToOpenAIVideoRewritesTaskIdentityAndURLs(t *testing.T) {
	originalServerAddress := system_setting.ServerAddress
	system_setting.ServerAddress = "https://frontend.example.com"
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	originalAPIServerAddress, hadAPIServerAddress := common.OptionMap["ApiServerAddress"]
	common.OptionMap["ApiServerAddress"] = "https://api.example.com"
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		system_setting.ServerAddress = originalServerAddress
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		if hadAPIServerAddress {
			common.OptionMap["ApiServerAddress"] = originalAPIServerAddress
		} else {
			delete(common.OptionMap, "ApiServerAddress")
		}
	})

	upstreamTaskID := "task_frimodel_upstream"
	task := &model.Task{
		TaskID: "task_zmodel_public",
		Status: model.TaskStatusSuccess,
		Data: []byte(`{
			"id":"task_frimodel_upstream",
			"task_id":"task_frimodel_upstream",
			"status":"completed",
			"url":"https://api.frimodel.com/v1/videos/task_frimodel_upstream/content",
			"video_url":"https://api.frimodel.com/v1/videos/task_frimodel_upstream/content",
			"metadata":{
				"url":"https://api.frimodel.com/v1/videos/task_frimodel_upstream/content",
				"content_url":"https://newapi.megabyai.cc/v1/videos/task_frimodel_upstream/content",
				"local_url":"https://newapi.megabyai.cc/v1/videos/task_frimodel_upstream/content",
				"video_url":"https://megavideos.example/videos/task_frimodel_upstream.mp4",
				"final_video_url":"https://megavideos.example/videos/task_frimodel_upstream.mp4",
				"origin_video_url":"https://megavideos.example/videos/task_frimodel_upstream.mp4",
				"cached":true,
				"cost_credits":70
			}
		}`),
	}

	result, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	require.NoError(t, err)

	var payload struct {
		ID       string `json:"id"`
		TaskID   string `json:"task_id"`
		URL      string `json:"url"`
		VideoURL string `json:"video_url"`
		Metadata struct {
			URL         string `json:"url"`
			ContentURL  string `json:"content_url"`
			LocalURL    string `json:"local_url"`
			VideoURL    string `json:"video_url"`
			FinalURL    string `json:"final_video_url"`
			OriginURL   string `json:"origin_video_url"`
			Cached      bool   `json:"cached"`
			CostCredits int    `json:"cost_credits"`
		} `json:"metadata"`
	}
	require.NoError(t, common.Unmarshal(result, &payload))

	expectedURL := "https://api.example.com/v1/videos/task_zmodel_public/content"
	assert.Equal(t, task.TaskID, payload.ID)
	assert.Equal(t, task.TaskID, payload.TaskID)
	assert.Equal(t, expectedURL, payload.URL)
	assert.Equal(t, expectedURL, payload.VideoURL)
	assert.Equal(t, expectedURL, payload.Metadata.URL)
	assert.Equal(t, expectedURL, payload.Metadata.ContentURL)
	assert.Equal(t, expectedURL, payload.Metadata.LocalURL)
	assert.Equal(t, expectedURL, payload.Metadata.VideoURL)
	assert.Equal(t, expectedURL, payload.Metadata.FinalURL)
	assert.Empty(t, payload.Metadata.OriginURL)
	assert.True(t, payload.Metadata.Cached)
	assert.Equal(t, 70, payload.Metadata.CostCredits)
	assert.NotContains(t, string(result), upstreamTaskID)
	assert.NotContains(t, string(result), "api.frimodel.com")
	assert.NotContains(t, string(result), "newapi.megabyai.cc")
	assert.NotContains(t, string(result), "megavideos.example")
}

func TestConvertToOpenAIVideoRemovesUpstreamURLsBeforeCompletion(t *testing.T) {
	task := &model.Task{
		TaskID: "task_zmodel_public",
		Status: model.TaskStatusFailure,
		Data: []byte(`{
			"id":"task_frimodel_upstream",
			"task_id":"task_frimodel_upstream",
			"status":"failed",
			"url":"https://api.frimodel.com/v1/videos/task_frimodel_upstream/content",
			"video_url":"https://api.frimodel.com/v1/videos/task_frimodel_upstream/content",
			"metadata":{
				"url":"https://api.frimodel.com/v1/videos/task_frimodel_upstream/content",
				"content_url":"https://newapi.megabyai.cc/v1/videos/task_frimodel_upstream/content",
				"local_url":"https://newapi.megabyai.cc/v1/videos/task_frimodel_upstream/content",
				"video_url":"https://megavideos.example/videos/task_frimodel_upstream.mp4",
				"final_video_url":"https://megavideos.example/videos/task_frimodel_upstream.mp4",
				"origin_video_url":"https://megavideos.example/videos/task_frimodel_upstream.mp4",
				"request_id":"req_123"
			}
		}`),
	}

	result, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(result, &payload))
	assert.Equal(t, task.TaskID, payload["id"])
	assert.Equal(t, task.TaskID, payload["task_id"])
	assert.NotContains(t, payload, "url")
	assert.NotContains(t, payload, "video_url")

	metadata, ok := payload["metadata"].(map[string]any)
	require.True(t, ok)
	assert.NotContains(t, metadata, "url")
	assert.NotContains(t, metadata, "content_url")
	assert.NotContains(t, metadata, "local_url")
	assert.NotContains(t, metadata, "video_url")
	assert.NotContains(t, metadata, "final_video_url")
	assert.NotContains(t, metadata, "origin_video_url")
	assert.Equal(t, "req_123", metadata["request_id"])
	assert.NotContains(t, string(result), "api.frimodel.com")
	assert.NotContains(t, string(result), "newapi.megabyai.cc")
	assert.NotContains(t, string(result), "megavideos.example")
}

func TestConvertToOpenAIVideoDoesNotAddAbsentMetadataVariants(t *testing.T) {
	originalServerAddress := system_setting.ServerAddress
	system_setting.ServerAddress = "https://apimodel.aihubkit.com"
	t.Cleanup(func() {
		system_setting.ServerAddress = originalServerAddress
	})

	task := &model.Task{
		TaskID: "task_zmodel_public",
		Status: model.TaskStatusSuccess,
		Data: []byte(`{
			"id":"task_frimodel_upstream",
			"task_id":"task_frimodel_upstream",
			"status":"completed",
			"metadata":{"url":"https://api.frimodel.com/v1/videos/task_frimodel_upstream/content"}
		}`),
	}

	result, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(result, &payload))
	metadata, ok := payload["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://apimodel.aihubkit.com/v1/videos/task_zmodel_public/content", metadata["url"])
	assert.NotContains(t, metadata, "content_url")
	assert.NotContains(t, metadata, "local_url")
	assert.NotContains(t, metadata, "video_url")
	assert.NotContains(t, metadata, "final_video_url")
	assert.NotContains(t, metadata, "origin_video_url")
}
