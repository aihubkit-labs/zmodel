package controller

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/pkg/objectstorage"
	"github.com/QuantumNous/new-api/setting/storage_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type objectStorageProbeTestStorage struct {
	calls       []string
	putInput    objectstorage.PutObjectInput
	headInput   objectstorage.HeadObjectInput
	deleteInput objectstorage.DeleteObjectInput
	putErr      error
	deleteErr   error
}

func (s *objectStorageProbeTestStorage) PutObject(_ context.Context, input objectstorage.PutObjectInput) (objectstorage.PutObjectResult, error) {
	s.calls = append(s.calls, "put")
	s.putInput = input
	return objectstorage.PutObjectResult{}, s.putErr
}

func (s *objectStorageProbeTestStorage) HeadObject(_ context.Context, input objectstorage.HeadObjectInput) (objectstorage.HeadObjectResult, error) {
	s.calls = append(s.calls, "head")
	s.headInput = input
	return objectstorage.HeadObjectResult{Exists: true, ContentLength: 2}, nil
}

func (s *objectStorageProbeTestStorage) DeleteObject(_ context.Context, input objectstorage.DeleteObjectInput) error {
	s.calls = append(s.calls, "delete")
	s.deleteInput = input
	return s.deleteErr
}

func (s *objectStorageProbeTestStorage) PresignGetObject(context.Context, objectstorage.PresignGetObjectInput) (string, error) {
	return "", nil
}

func TestObjectStorageSettingsAPIsNeverReturnSecret(t *testing.T) {
	const secret = "secret-that-must-never-be-returned"
	stagingDirectory := t.TempDir()
	common.OptionMapRWMutex.Lock()
	original := common.OptionMap
	common.OptionMap = map[string]string{
		storage_setting.OptionS3Endpoint:                "https://s3.example.com",
		storage_setting.OptionS3Region:                  "test-region",
		storage_setting.OptionS3Bucket:                  "test-bucket",
		storage_setting.OptionS3AccessKey:               "test-access-key",
		storage_setting.OptionS3SecretAccessKey:         secret,
		storage_setting.OptionStagingDirectory:          stagingDirectory,
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
			StagingDirectory string `json:"staging_directory"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &dedicated))
	assert.True(t, dedicated.Success)
	assert.Equal(t, "test-access-key", dedicated.Data.AccessKey)
	assert.True(t, dedicated.Data.SecretConfigured)
	assert.Equal(t, stagingDirectory, dedicated.Data.StagingDirectory)

	recorder = httptest.NewRecorder()
	context, _ = gin.CreateTestContext(recorder)
	GetOptions(context)
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), secret)
	assert.NotContains(t, recorder.Body.String(), storage_setting.OptionS3SecretAccessKey)
}

func TestObjectStorageProbeUsesAuthorizedAsyncImagePrefixAndCleansUp(t *testing.T) {
	storage := &objectStorageProbeTestStorage{}
	originalFactory := objectstorage.NewStorage
	objectstorage.NewStorage = func(context.Context, objectstorage.Config) (objectstorage.Storage, error) {
		return storage, nil
	}
	t.Cleanup(func() { objectstorage.NewStorage = originalFactory })

	err := probeObjectStorage(context.Background(), storage_setting.Settings{Bucket: "test-bucket"})
	require.NoError(t, err)
	require.Equal(t, []string{"put", "head", "delete"}, storage.calls)
	assert.True(t, strings.HasPrefix(storage.putInput.Key, asyncImageObjectPrefix+"/.probe/"))
	assert.Equal(t, storage.putInput.Key, storage.headInput.Key)
	assert.Equal(t, storage.putInput.Key, storage.deleteInput.Key)
	assert.Equal(t, "test-bucket", storage.putInput.Bucket)
	assert.Equal(t, map[string]string{"probe": "true"}, storage.putInput.Metadata)
}

func TestObjectStorageProbeRequiresCleanupPermission(t *testing.T) {
	storage := &objectStorageProbeTestStorage{deleteErr: errors.New("AccessDenied")}
	originalFactory := objectstorage.NewStorage
	objectstorage.NewStorage = func(context.Context, objectstorage.Config) (objectstorage.Storage, error) {
		return storage, nil
	}
	t.Cleanup(func() { objectstorage.NewStorage = originalFactory })

	err := probeObjectStorage(context.Background(), storage_setting.Settings{Bucket: "test-bucket"})
	require.Error(t, err)
	assert.Equal(t, i18n.MsgObjectStorageProbeCleanupFailed, objectStorageProbeErrorMessageKey(err))
	assert.Equal(t, []string{"put", "head", "delete"}, storage.calls)
}

func TestUpdateObjectStorageSettingsDoesNotExposeProviderProbeError(t *testing.T) {
	stagingDirectory := t.TempDir()
	t.Setenv("ASYNC_IMAGE_STAGING_ALLOWED_ROOTS", stagingDirectory)
	common.OptionMapRWMutex.Lock()
	originalOptions := common.OptionMap
	common.OptionMap = map[string]string{
		storage_setting.OptionS3Endpoint:                "https://s3.example.com",
		storage_setting.OptionS3Region:                  "test-region",
		storage_setting.OptionS3Bucket:                  "test-bucket",
		storage_setting.OptionS3AccessKey:               "test-access-key",
		storage_setting.OptionS3SecretAccessKey:         "test-secret",
		storage_setting.OptionStagingDirectory:          stagingDirectory,
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
		common.OptionMap = originalOptions
		common.OptionMapRWMutex.Unlock()
	})

	storage := &objectStorageProbeTestStorage{
		putErr: errors.New("RequestID=SENSITIVE HostID=SENSITIVE arn:aws:iam::123456789:user/internal"),
	}
	originalFactory := objectstorage.NewStorage
	objectstorage.NewStorage = func(context.Context, objectstorage.Config) (objectstorage.Storage, error) {
		return storage, nil
	}
	t.Cleanup(func() { objectstorage.NewStorage = originalFactory })

	payload, err := common.Marshal(updateObjectStorageSettingsRequest{
		Endpoint:                  "https://s3.example.com",
		Region:                    "test-region",
		Bucket:                    "test-bucket",
		AccessKey:                 "test-access-key",
		StagingDirectory:          stagingDirectory,
		RetentionSeconds:          86400,
		PresignSeconds:            600,
		ArchiveTimeoutSeconds:     600,
		ArchiveMaxAttempts:        8,
		ArchiveRetryWindowSeconds: 21600,
		CleanupIntervalSeconds:    900,
	})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodPut, "/api/option/object-storage", bytes.NewReader(payload))
	ginContext.Request.Header.Set("Accept-Language", "en")
	require.NoError(t, i18n.Init())
	UpdateObjectStorageSettings(ginContext)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Success)
	assert.Equal(t, "Object storage write probe failed. Check PutObject permission for the async image prefix.", response.Message)
	assert.NotContains(t, recorder.Body.String(), "RequestID")
	assert.NotContains(t, recorder.Body.String(), "HostID")
	assert.NotContains(t, recorder.Body.String(), "arn:aws")
}
