package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type videoCapabilityTemplateAPIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func setupVideoCapabilityTemplateControllerTest(t *testing.T) {
	t.Helper()
	originalDB := model.DB
	t.Cleanup(func() { model.DB = originalDB })

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	require.NoError(t, model.DB.AutoMigrate(&model.VideoModelCapabilityTemplate{}))
	require.NoError(t, model.SeedVideoModelCapabilityTemplates())
}

func TestSaveVideoModelCapabilityTemplateCannotModifyBuiltInTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupVideoCapabilityTemplateControllerTest(t)
	templates, err := model.ListVideoModelCapabilityTemplates(dto.VideoProtocolGlobalAIOpc)
	require.NoError(t, err)
	require.NotEmpty(t, templates)
	builtIn, err := templates[0].ToDTO()
	require.NoError(t, err)
	builtIn.Name = "Modified built-in template"

	payload, err := common.Marshal(builtIn)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/channel/video_capability_templates", bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")

	SaveVideoModelCapabilityTemplate(ctx)

	var response videoCapabilityTemplateAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Success)
	assert.Equal(t, "built-in video model capability templates cannot be modified", response.Message)
}

func TestDeleteVideoModelCapabilityTemplateRejectsInvalidAndBuiltInIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupVideoCapabilityTemplateControllerTest(t)
	templates, err := model.ListVideoModelCapabilityTemplates(dto.VideoProtocolGlobalAIOpc)
	require.NoError(t, err)
	require.NotEmpty(t, templates)

	tests := []struct {
		name    string
		id      string
		message string
	}{
		{name: "invalid ID", id: "invalid", message: "invalid video model capability template ID"},
		{name: "built-in ID", id: fmt.Sprint(templates[0].ID), message: "built-in video model capability templates cannot be deleted"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodDelete, "/api/channel/video_capability_templates/"+test.id, nil)
			ctx.Params = gin.Params{{Key: "id", Value: test.id}}

			DeleteVideoModelCapabilityTemplate(ctx)

			var response videoCapabilityTemplateAPIResponse
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			assert.False(t, response.Success)
			assert.Equal(t, test.message, response.Message)
		})
	}
}
