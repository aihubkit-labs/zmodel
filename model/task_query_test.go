package model

import (
	"testing"

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

func TestTaskFilterOptionsRespectUserScope(t *testing.T) {
	setupTaskQueryTest(t)

	adminOptions, err := GetTaskFilterOptions(nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"alice", "bob"}, adminOptions.Usernames)
	assert.Equal(t, []string{"default", "vip"}, adminOptions.Groups)
	assert.Equal(t, []string{"upstream-only-model", "video-model-a", "video-model-b"}, adminOptions.Models)

	userID := 101
	userOptions, err := GetTaskFilterOptions(&userID)
	require.NoError(t, err)
	assert.Empty(t, userOptions.Usernames)
	assert.Equal(t, []string{"default", "vip"}, userOptions.Groups)
	assert.Equal(t, []string{"video-model-a", "video-model-b"}, userOptions.Models)
}
