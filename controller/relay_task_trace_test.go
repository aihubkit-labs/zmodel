package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordFailedVideoTaskPersistsSubmissionTransportError(t *testing.T) {
	db := setupTaskControllerTestDB(t)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	ctx.Set("platform", "sora")
	trace := &dto.TaskUpstreamHTTPTrace{
		SubmitRequest:  &dto.TaskHTTPMessage{Method: http.MethodPost, URL: "https://upstream.example/v1/videos"},
		SubmitResponse: &dto.TaskHTTPMessage{Error: "context deadline exceeded"},
	}
	ctx.Set(relaycommon.TaskUpstreamHTTPTraceContextKey, trace)
	relayInfo := &relaycommon.RelayInfo{
		UserId: 12, UsingGroup: "default",
		ChannelMeta: &relaycommon.ChannelMeta{ChannelId: 34},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action: constant.TaskActionGenerate, PublicTaskID: "task_failed_submit_trace",
		},
	}
	taskErr := &dto.TaskError{Message: "upstream request timed out", StatusCode: http.StatusInternalServerError}

	require.NoError(t, recordFailedVideoTask(ctx, relayInfo, taskErr))

	var stored model.Task
	require.NoError(t, db.Where("task_id = ?", "task_failed_submit_trace").First(&stored).Error)
	assert.Equal(t, model.TaskStatus(model.TaskStatusFailure), stored.Status)
	assert.Equal(t, "100%", stored.Progress)
	assert.Equal(t, "upstream request timed out", stored.FailReason)
	assert.Equal(t, constant.TaskActionGenerate, stored.Action)
	require.NotNil(t, stored.PrivateData.UpstreamHTTPTrace)
	assert.Equal(t, "context deadline exceeded", stored.PrivateData.UpstreamHTTPTrace.SubmitResponse.Error)
	assert.Empty(t, stored.PrivateData.Key)
}
