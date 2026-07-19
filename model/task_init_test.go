package model

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitTaskStoresSelectedOpenAIChannelKey(t *testing.T) {
	truncateTables(t)
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(context, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
	common.SetContextKey(context, constant.ContextKeyChannelId, 202)
	common.SetContextKey(context, constant.ContextKeyChannelKey, "selected-upstream-key")

	relayInfo := &relaycommon.RelayInfo{
		UserId:     101,
		UsingGroup: "default",
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task_public_id",
		},
	}
	relayInfo.InitChannelMeta(context)

	task := InitTask(constant.TaskPlatform("1"), relayInfo)

	require.NotNil(t, task)
	assert.Equal(t, "task_public_id", task.TaskID)
	assert.Equal(t, "selected-upstream-key", task.PrivateData.Key)
	assert.Equal(t, 202, task.ChannelId)

	require.NoError(t, task.Insert())
	var persisted Task
	require.NoError(t, DB.First(&persisted, task.ID).Error)
	assert.Equal(t, "selected-upstream-key", persisted.PrivateData.Key)
}
