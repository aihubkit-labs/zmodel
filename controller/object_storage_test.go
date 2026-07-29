package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/storage_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObjectStorageSettingsAPIsNeverReturnSecret(t *testing.T) {
	const secret = "secret-that-must-never-be-returned"
	common.OptionMapRWMutex.Lock()
	original := common.OptionMap
	common.OptionMap = map[string]string{
		storage_setting.OptionS3Endpoint:                "https://s3.example.com",
		storage_setting.OptionS3Region:                  "test-region",
		storage_setting.OptionS3Bucket:                  "test-bucket",
		storage_setting.OptionS3AccessKey:               "test-access-key",
		storage_setting.OptionS3SecretAccessKey:         secret,
		storage_setting.OptionRetentionSeconds:          "86400",
		storage_setting.OptionPresignSeconds:            "600",
		storage_setting.OptionArchiveTimeoutSeconds:     "600",
		storage_setting.OptionArchiveMaxAttempts:        "8",
		storage_setting.OptionArchiveRetryWindowSeconds: "21600",
		storage_setting.OptionCleanupIntervalSeconds:    "900",
	}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = original
		common.OptionMapRWMutex.Unlock()
	})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	GetObjectStorageSettings(context)
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), secret)
	var dedicated struct {
		Success bool `json:"success"`
		Data    struct {
			AccessKey        string `json:"access_key"`
			SecretConfigured bool   `json:"secret_configured"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &dedicated))
	assert.True(t, dedicated.Success)
	assert.Equal(t, "test-access-key", dedicated.Data.AccessKey)
	assert.True(t, dedicated.Data.SecretConfigured)

	recorder = httptest.NewRecorder()
	context, _ = gin.CreateTestContext(recorder)
	GetOptions(context)
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), secret)
	assert.NotContains(t, recorder.Body.String(), storage_setting.OptionS3SecretAccessKey)
}
