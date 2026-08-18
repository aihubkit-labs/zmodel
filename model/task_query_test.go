package model

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTaskQueryTest(t *testing.T) {
	t.Helper()
	truncateTables(t)
	require.NoError(t, DB.Exec("DELETE FROM tasks").Error)
	require.NoError(t, DB.Exec("DELETE FROM users").Error)

	users := []User{
		{Id: 101, Username: "alice", Password: "password", AffCode: "alice-aff"},
		{Id: 102, Username: "bob", Password: "password", AffCode: "bob-aff"},
	}
	require.NoError(t, DB.Create(&users).Error)

	tasks := []Task{
		{
			TaskID:     "task_alice_vip_model_a",
			UserId:     101,
			Group:      "vip",
			Properties: Properties{OriginModelName: "video-model-a"},
		},
		{
			TaskID:     "task_alice_default_model_b",
			UserId:     101,
			Group:      "default",
			Properties: Properties{OriginModelName: "video-model-b"},
		},
		{
			TaskID:     "task_bob_vip_model_a",
			UserId:     102,
			Group:      "vip",
			Properties: Properties{OriginModelName: "video-model-a"},
		},
		{
			TaskID:     "task_bob_upstream_model",
			UserId:     102,
			Group:      "default",
			Properties: Properties{UpstreamModelName: "upstream-only-model"},
		},
		{
			TaskID:     "task_bob_model_without_group",
			UserId:     102,
			Properties: Properties{OriginModelName: "model-without-group"},
		},
	}
	require.NoError(t, DB.Create(&tasks).Error)
}

func TestTaskAdminQueryFiltersByUsernameGroupAndModel(t *testing.T) {
	setupTaskQueryTest(t)

	params := SyncTaskQueryParams{
		Username: "alice",
		Group:    "vip",
		Model:    "video-model-a",
	}
	tasks := TaskGetAllTasks(0, 20, params)

	require.Len(t, tasks, 1)
	assert.Equal(t, "task_alice_vip_model_a", tasks[0].TaskID)
	assert.Equal(t, int64(1), TaskCountAllTasks(params))
}

func TestTaskModelQueryUsesDisplayedModelFallback(t *testing.T) {
	setupTaskQueryTest(t)

	params := SyncTaskQueryParams{Model: "upstream-only-model"}
	tasks := TaskGetAllTasks(0, 20, params)

	require.Len(t, tasks, 1)
	assert.Equal(t, "task_bob_upstream_model", tasks[0].TaskID)
	assert.Equal(t, int64(1), TaskCountAllTasks(params))
}

func TestTaskUserQueryKeepsOwnerBoundaryWithGroupAndModelFilters(t *testing.T) {
	setupTaskQueryTest(t)

	params := SyncTaskQueryParams{Group: "vip", Model: "video-model-a"}
	tasks := TaskGetAllUserTask(101, 0, 20, params)

	require.Len(t, tasks, 1)
	assert.Equal(t, "task_alice_vip_model_a", tasks[0].TaskID)
	assert.Equal(t, int64(1), TaskCountAllUserTask(101, params))
}

func TestTaskUserQueryDoesNotExposeUpstreamOnlyModel(t *testing.T) {
	setupTaskQueryTest(t)

	params := SyncTaskQueryParams{Model: "upstream-only-model"}
	tasks := TaskGetAllUserTask(102, 0, 20, params)

	assert.Empty(t, tasks)
	assert.Zero(t, TaskCountAllUserTask(102, params))
}

func TestTaskFilterOptionsRespectUserScope(t *testing.T) {
	setupTaskQueryTest(t)

	adminOptions, err := GetTaskFilterOptions(nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"alice", "bob"}, adminOptions.Usernames)
	assert.Equal(t, []string{"default", "vip"}, adminOptions.Groups)
	assert.Equal(t, []string{"model-without-group", "upstream-only-model", "video-model-a", "video-model-b"}, adminOptions.Models)

	userID := 101
	userOptions, err := GetTaskFilterOptions(&userID)
	require.NoError(t, err)
	assert.Empty(t, userOptions.Usernames)
	assert.Equal(t, []string{"default", "vip"}, userOptions.Groups)
	assert.Equal(t, []string{"video-model-a", "video-model-b"}, userOptions.Models)

	userID = 102
	userOptions, err = GetTaskFilterOptions(&userID)
	require.NoError(t, err)
	assert.NotContains(t, userOptions.Models, "upstream-only-model")
}

func TestPreparingVideoTasksUseDedicatedPollingQueue(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Exec("DELETE FROM tasks").Error)
	tasks := []Task{
		{TaskID: "preparing-oldest", Status: TaskStatusPreparing, Progress: "0%"},
		{TaskID: "preparing-next", Status: TaskStatusPreparing, Progress: "0%"},
		{TaskID: "queued", Status: TaskStatusQueued, Progress: "10%"},
		{TaskID: "completed", Status: TaskStatusSuccess, Progress: "100%"},
	}
	require.NoError(t, DB.Create(&tasks).Error)
	require.NoError(t, DB.Model(&Task{}).Where("id = ?", tasks[0].ID).UpdateColumn("updated_at", 1).Error)
	require.NoError(t, DB.Model(&Task{}).Where("id = ?", tasks[1].ID).UpdateColumn("updated_at", 2).Error)

	preparing := GetPreparingVideoTasks(1)
	require.Len(t, preparing, 1)
	assert.Equal(t, "preparing-oldest", preparing[0].TaskID)
	won, err := preparing[0].UpdatePrivateDataWithStatus(TaskStatusPreparing)
	require.NoError(t, err)
	require.True(t, won)
	preparing = GetPreparingVideoTasks(1)
	require.Len(t, preparing, 1)
	assert.Equal(t, "preparing-next", preparing[0].TaskID)

	ordinary := GetAllUnFinishSyncTasks(10)
	require.Len(t, ordinary, 1)
	assert.Equal(t, "queued", ordinary[0].TaskID)
	assert.Equal(t, dto.VideoStatusQueued, TaskStatusPreparing.ToVideoStatus())
}
