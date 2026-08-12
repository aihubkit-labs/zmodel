package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateModelMetaRejectsOversizedUsageGuidelines(t *testing.T) {
	gin.SetMode(gin.TestMode)
	payload, err := common.Marshal(map[string]any{
		"model_name":       "oversized-guidelines-model",
		"usage_guidelines": strings.Repeat("界", maxModelUsageGuidelinesLength+1),
	})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")

	CreateModelMeta(ctx)

	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Success)
	assert.Equal(t, "素材与场景限制不能超过 20000 个字符", response.Message)
}
