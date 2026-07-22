package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskModel2DtoUsesPublicContentURLForVideoTask(t *testing.T) {
	task := &model.Task{
		TaskID: "task_public_id",
		Action: constant.TaskActionTextGenerate,
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://upstream.example/private-video.mp4",
		},
	}

	dtoTask := TaskModel2Dto(task)

	assert.Equal(t, taskcommon.BuildProxyURL(task.TaskID), dtoTask.ResultURL)
}

func TestTaskModel2DtoProvidesReadablePlatformName(t *testing.T) {
	task := &model.Task{
		Platform: constant.TaskPlatform("1"),
	}

	dtoTask := TaskModel2Dto(task)

	assert.Equal(t, "OpenAI", dtoTask.PlatformName)
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
