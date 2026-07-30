package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/objectstorage"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/storage_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type asyncImageDetailTestStorage struct {
	presignInputs []objectstorage.PresignGetObjectInput
}

func (s *asyncImageDetailTestStorage) PutObject(context.Context, objectstorage.PutObjectInput) (objectstorage.PutObjectResult, error) {
	return objectstorage.PutObjectResult{}, nil
}

func (s *asyncImageDetailTestStorage) HeadObject(context.Context, objectstorage.HeadObjectInput) (objectstorage.HeadObjectResult, error) {
	return objectstorage.HeadObjectResult{}, nil
}

func (s *asyncImageDetailTestStorage) DeleteObject(context.Context, objectstorage.DeleteObjectInput) error {
	return nil
}

func (s *asyncImageDetailTestStorage) PresignGetObject(_ context.Context, input objectstorage.PresignGetObjectInput) (string, error) {
	s.presignInputs = append(s.presignInputs, input)
	if strings.HasPrefix(input.ResponseDisposition, "attachment;") {
		return "https://storage.example/download", nil
	}
	return "https://storage.example/preview", nil
}

func setupAsyncImageDetailTest(t *testing.T) {
	t.Helper()
	originalDB := model.DB
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "async-image-detail.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.AsyncImageTask{}, &model.StorageObject{}, &model.User{}, &model.Channel{}))
	model.DB = db
	t.Cleanup(func() { model.DB = originalDB })

	common.OptionMapRWMutex.Lock()
	originalOptions := common.OptionMap
	common.OptionMap = map[string]string{
		storage_setting.OptionS3AccessKey:               "test-access-key",
		storage_setting.OptionS3SecretAccessKey:         "secret-that-must-not-leak",
		storage_setting.OptionPresignSeconds:            "600",
		storage_setting.OptionStagingDirectory:          t.TempDir(),
		storage_setting.OptionRetentionSeconds:          "86400",
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
}

func createAsyncImageDetailFixture(t *testing.T) {
	t.Helper()
	now := common.GetTimestamp()
	require.NoError(t, model.DB.Create(&model.User{
		Id: 42, Username: "async-image-user", Password: "test-password", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default",
	}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{
		Id: 9, Name: "openai-images", Type: constant.ChannelTypeOpenAI, Key: "test-key",
	}).Error)
	require.NoError(t, model.DB.Create(&model.AsyncImageTask{
		TaskID: "img_task_detail", UserID: 42, TokenID: 7,
		Status: model.AsyncImageStatusSucceeded, OutputAvailability: model.AsyncImageOutputAvailable,
		BillingStatus: model.AsyncImageBillingSettled, BillingSource: "wallet",
		ReservedQuota: 100, ActualQuota: 90, OriginModelName: "gpt-image-test",
		UsingGroup: "default", LastChannelID: 9, LastChannelType: constant.ChannelTypeOpenAI,
		RequestPayload:   `{"prompt":"red apple","images":["data:image/png;base64,private"]}`,
		RequestSnapshot:  `{"prompt":"red apple","n":1}`,
		RetentionSeconds: 86400, ArchiveTimeoutSeconds: 600, ArchiveMaxAttempts: 8,
		OutputExpiresAt: now + 3600, StartedAt: now - 20, GenerationCompletedAt: now - 10,
		BillingFinalizedAt: now - 10, CompletedAt: now - 5,
	}).Error)
	require.NoError(t, model.DB.Create(&model.StorageObject{
		BusinessID: model.StorageObjectBusinessAsyncImages, ResourceID: "img_task_detail", ObjectIndex: 0,
		Provider: model.StorageObjectProviderS3, Status: model.StorageObjectStatusAvailable,
		Endpoint: "https://s3.example.com", Region: "test-region", Bucket: "test-bucket",
		ObjectKey: "prod/user-files/test.img", MimeType: "image/png", Extension: "png",
		SizeBytes: 128, ETag: "test-etag", UploadedAt: now - 5, ExpiresAt: now + 3600,
		StagingRelativePath: "42/2026/07/29/img_task_detail/0.img",
		StagingStatus:       model.StorageStagingAvailable, StagingSizeBytes: 128, StagedAt: now - 10,
	}).Error)
}

func TestAsyncImageObjectKeyIncludesDayPartition(t *testing.T) {
	key := asyncImageObjectKey("task_day_partition", service.AsyncImageManifestItem{
		Index:               1,
		Extension:           "png",
		StagingRelativePath: "42/2026/07/29/task_day_partition/1.img",
	})
	assert.Equal(t, "prod/user-files/zmodel@async-images/2026/07/29/task_day_partition/1.png", key)
}

func TestAsyncImageTaskResponseUsesMinimalPublicContract(t *testing.T) {
	response := asyncImageTaskResponse(&model.AsyncImageTask{
		TaskID:             "task_public_contract",
		CreatedAt:          123,
		Status:             model.AsyncImageStatusSucceeded,
		OutputAvailability: model.AsyncImageOutputAvailable,
		OutputExpiresAt:    456,
	}, []dto.AsyncImageOutputData{{Index: 0, URL: "https://storage.example/image"}}, nil)

	encoded, err := common.Marshal(response)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"id":"task_public_contract",
		"status":"succeeded",
		"output":{
			"availability":"available",
			"expires_at":456,
			"data":[{"index":0,"url":"https://storage.example/image"}]
		}
	}`, string(encoded))
	assert.NotContains(t, string(encoded), "created_at")
	assert.NotContains(t, string(encoded), "mime_type")
	assert.NotContains(t, string(encoded), "size_bytes")
	assert.NotContains(t, string(encoded), "revised_prompt")
}

func TestAsyncImageRequestSnapshotExcludesImageInputs(t *testing.T) {
	var request dto.ImageRequest
	require.NoError(t, common.UnmarshalJsonStr(`{"model":"gpt-image","prompt":"red apple","n":2,"images":["data:image/png;base64,private"],"mask":"private-mask","custom":"private-value"}`, &request))

	snapshot, err := asyncImageRequestSnapshot(&request)
	require.NoError(t, err)
	assert.JSONEq(t, `{"model":"gpt-image","prompt":"red apple","n":2}`, snapshot)
}

func TestAsyncImageTaskDetailReturnsPreviewAndDownloadURLs(t *testing.T) {
	setupAsyncImageDetailTest(t)
	createAsyncImageDetailFixture(t)
	storage := &asyncImageDetailTestStorage{}
	originalFactory := objectstorage.NewStorage
	objectstorage.NewStorage = func(context.Context, objectstorage.Config) (objectstorage.Storage, error) {
		return storage, nil
	}
	t.Cleanup(func() { objectstorage.NewStorage = originalFactory })

	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Params = gin.Params{{Key: "task_id", Value: "img_task_detail"}}
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/api/async-image-task/img_task_detail", nil)
	GetAsyncImageTaskDetail(ginContext)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool                 `json:"success"`
		Data    asyncImageTaskDetail `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, "red apple", response.Data.Request.(map[string]any)["prompt"])
	assert.Equal(t, "async-image-user", response.Data.Username)
	assert.Equal(t, "openai-images", response.Data.ChannelName)
	assert.Equal(t, "OpenAI", response.Data.Platform)
	require.Len(t, response.Data.Objects, 1)
	assert.Equal(t, "https://storage.example/preview", response.Data.Objects[0].PreviewURL)
	assert.Equal(t, "https://storage.example/download", response.Data.Objects[0].DownloadURL)
	assert.Equal(t, "prod/user-files/test.img", response.Data.Objects[0].ObjectKey)
	require.Len(t, storage.presignInputs, 2)
	assert.Contains(t, storage.presignInputs[0].ResponseDisposition, "inline;")
	assert.Contains(t, storage.presignInputs[1].ResponseDisposition, "attachment;")
	assert.NotContains(t, recorder.Body.String(), "secret-that-must-not-leak")
	assert.NotContains(t, recorder.Body.String(), "42/2026/07/29/img_task_detail/0.img")
}

func TestSelfAsyncImageTaskDetailEnforcesOwnershipAndHidesStorageLocation(t *testing.T) {
	setupAsyncImageDetailTest(t)
	createAsyncImageDetailFixture(t)
	storage := &asyncImageDetailTestStorage{}
	originalFactory := objectstorage.NewStorage
	objectstorage.NewStorage = func(context.Context, objectstorage.Config) (objectstorage.Storage, error) {
		return storage, nil
	}
	t.Cleanup(func() { objectstorage.NewStorage = originalFactory })

	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Params = gin.Params{{Key: "task_id", Value: "img_task_detail"}}
	ginContext.Set("id", 42)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/api/async-image-task/self/img_task_detail", nil)
	GetSelfAsyncImageTaskDetail(ginContext)
	require.NotContains(t, recorder.Body.String(), "prod/user-files/test.img")
	require.NotContains(t, recorder.Body.String(), "test-bucket")
	require.Contains(t, recorder.Body.String(), "https://storage.example/preview")

	recorder = httptest.NewRecorder()
	ginContext, _ = gin.CreateTestContext(recorder)
	ginContext.Params = gin.Params{{Key: "task_id", Value: "img_task_detail"}}
	ginContext.Set("id", 43)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/api/async-image-task/self/img_task_detail", nil)
	GetSelfAsyncImageTaskDetail(ginContext)
	assert.NotContains(t, recorder.Body.String(), "red apple")
}

func TestAsyncImageTaskListIncludesRootDisplayMetadataOnly(t *testing.T) {
	setupAsyncImageDetailTest(t)
	createAsyncImageDetailFixture(t)
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", 9).Update("type", constant.ChannelTypeAzure).Error)

	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/api/async-image-task?page=1&page_size=20", nil)
	GetAllAsyncImageTasks(ginContext)
	require.Equal(t, http.StatusOK, recorder.Code)
	var rootResponse struct {
		Success bool `json:"success"`
		Data    struct {
			Items []asyncImageTaskListItem `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &rootResponse))
	require.True(t, rootResponse.Success)
	require.Len(t, rootResponse.Data.Items, 1)
	assert.Equal(t, "async-image-user", rootResponse.Data.Items[0].Username)
	assert.Equal(t, "openai-images", rootResponse.Data.Items[0].ChannelName)
	assert.Equal(t, constant.GetChannelTypeName(constant.ChannelTypeOpenAI), rootResponse.Data.Items[0].Platform)
	assert.Equal(t, "default", rootResponse.Data.Items[0].UsingGroup)

	recorder = httptest.NewRecorder()
	ginContext, _ = gin.CreateTestContext(recorder)
	ginContext.Set("id", 42)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/api/async-image-task/self?page=1&page_size=20", nil)
	GetSelfAsyncImageTasks(ginContext)
	assert.NotContains(t, recorder.Body.String(), "async-image-user")
	assert.NotContains(t, recorder.Body.String(), "openai-images")
	assert.NotContains(t, recorder.Body.String(), `"platform"`)
	assert.Contains(t, recorder.Body.String(), `"using_group":"default"`)
}

var _ objectstorage.Storage = (*asyncImageDetailTestStorage)(nil)
