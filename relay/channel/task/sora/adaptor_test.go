package sora

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToOpenAIVideoRewritesTaskIdentityAndURLs(t *testing.T) {
	originalServerAddress := system_setting.ServerAddress
	system_setting.ServerAddress = "https://apimodel.aihubkit.com"
	t.Cleanup(func() {
		system_setting.ServerAddress = originalServerAddress
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
			"metadata":{"url":"https://api.frimodel.com/v1/videos/task_frimodel_upstream/content"}
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
			URL string `json:"url"`
		} `json:"metadata"`
	}
	require.NoError(t, common.Unmarshal(result, &payload))

	expectedURL := "https://apimodel.aihubkit.com/v1/videos/task_zmodel_public/content"
	assert.Equal(t, task.TaskID, payload.ID)
	assert.Equal(t, task.TaskID, payload.TaskID)
	assert.Equal(t, expectedURL, payload.URL)
	assert.Equal(t, expectedURL, payload.VideoURL)
	assert.Equal(t, expectedURL, payload.Metadata.URL)
	assert.NotContains(t, string(result), upstreamTaskID)
	assert.NotContains(t, string(result), "api.frimodel.com")
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
			"metadata":{"url":"https://api.frimodel.com/v1/videos/task_frimodel_upstream/content","request_id":"req_123"}
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
	assert.Equal(t, "req_123", metadata["request_id"])
	assert.NotContains(t, string(result), "api.frimodel.com")
}
