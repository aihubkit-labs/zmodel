package controller

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

type videoArchiveTestStorage struct {
	head      objectstorage.HeadObjectResult
	putErr    error
	putInputs []objectstorage.PutObjectInput
	putBodies [][]byte
}

func (storage *videoArchiveTestStorage) PutObject(_ context.Context, input objectstorage.PutObjectInput) (objectstorage.PutObjectResult, error) {
	body, err := io.ReadAll(input.Body)
	if err != nil {
		return objectstorage.PutObjectResult{}, err
	}
	storage.putInputs = append(storage.putInputs, input)
	storage.putBodies = append(storage.putBodies, body)
	return objectstorage.PutObjectResult{ETag: "uploaded-etag"}, storage.putErr
}

func (storage *videoArchiveTestStorage) HeadObject(context.Context, objectstorage.HeadObjectInput) (objectstorage.HeadObjectResult, error) {
	return storage.head, nil
}

func (storage *videoArchiveTestStorage) DeleteObject(context.Context, objectstorage.DeleteObjectInput) error {
	return nil
}

func (storage *videoArchiveTestStorage) PresignGetObject(context.Context, objectstorage.PresignGetObjectInput) (string, error) {
	return "https://storage.example/video", nil
}

func setupVideoArchiveTest(t *testing.T, storage objectstorage.Storage) {
	t.Helper()
	originalDB := model.DB
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "video-storage.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.StorageObject{}))
	model.DB = db
	t.Cleanup(func() { model.DB = originalDB })

	common.OptionMapRWMutex.Lock()
	originalOptions := common.OptionMap
	common.OptionMap = map[string]string{
		storage_setting.OptionVideoS3Region:                  "test-region",
		storage_setting.OptionVideoS3Bucket:                  "test-bucket",
		storage_setting.OptionVideoS3AccessKey:               "test-access-key",
		storage_setting.OptionVideoS3SecretAccessKey:         "test-secret",
		storage_setting.OptionVideoS3KeyPrefix:               "dev",
		storage_setting.OptionVideoBusinessID:                "test@videos",
		storage_setting.OptionVideoStagingDirectory:          t.TempDir(),
		storage_setting.OptionVideoRetentionSeconds:          "3600",
		storage_setting.OptionVideoPresignSeconds:            "600",
		storage_setting.OptionVideoArchiveTimeoutSeconds:     "60",
		storage_setting.OptionVideoArchiveMaxAttempts:        "3",
		storage_setting.OptionVideoArchiveRetryWindowSeconds: "3600",
		storage_setting.OptionVideoCleanupInterval:           "900",
	}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptions
		common.OptionMapRWMutex.Unlock()
	})

	originalFactory := objectstorage.NewStorage
	objectstorage.NewStorage = func(context.Context, objectstorage.Config) (objectstorage.Storage, error) {
		return storage, nil
	}
	t.Cleanup(func() { objectstorage.NewStorage = originalFactory })
}

func TestVideoObjectKeyUsesTaskSubmissionDate(t *testing.T) {
	task := &model.Task{
		TaskID: "task_video_key", UserId: 42,
		SubmitTime: time.Date(2026, time.July, 29, 23, 30, 0, 0, time.UTC).Unix(),
	}
	key := videoObjectKey(storage_setting.VideoSettings{
		S3KeyPrefix: "prod", BusinessID: "test@videos",
	}, task, "mp4")
	assert.Equal(t, "prod/user-files/test@videos/42/2026/07/29/task_video_key/original.mp4", key)
}

func TestArchiveVideoTaskUploadsOnceAndPersistsObject(t *testing.T) {
	storage := &videoArchiveTestStorage{}
	setupVideoArchiveTest(t, storage)
	task := &model.Task{
		TaskID: "task_video_archive", UserId: 42,
		SubmitTime: time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC).Unix(),
	}
	channel := &model.Channel{Type: constant.ChannelTypeKling}

	require.NoError(t, ArchiveVideoTask(context.Background(), task, channel, service.VideoArchiveSource{
		URL: "data:video/mp4;base64,dmlkZW8tY29udGVudA==",
	}))
	require.Len(t, storage.putInputs, 1)
	assert.Equal(t, []byte("video-content"), storage.putBodies[0])
	assert.Equal(t, "dev/user-files/test@videos/42/2026/07/29/task_video_archive/original.mp4", storage.putInputs[0].Key)
	assert.Equal(t, "test@videos", storage.putInputs[0].Metadata["business-id"])
	assert.Equal(t, "task_video_archive", storage.putInputs[0].Metadata["resource-id"])

	object, err := model.GetStorageObjectByBusinessID("test@videos", task.TaskID, 0)
	require.NoError(t, err)
	assert.Equal(t, model.StorageObjectStatusAvailable, object.Status)
	assert.Equal(t, "uploaded-etag", object.ETag)
	assert.Greater(t, object.ExpiresAt, common.GetTimestamp())
	assert.Empty(t, object.ArchiveOperationID)
	assert.Zero(t, object.ArchiveLeaseExpiresAt)

	require.NoError(t, ArchiveVideoTask(context.Background(), task, channel, service.VideoArchiveSource{}))
	assert.Len(t, storage.putInputs, 1)
}

func TestArchiveVideoTaskMarksObjectFailedWhenSourceCannotBeDownloaded(t *testing.T) {
	setupVideoArchiveTest(t, &videoArchiveTestStorage{})
	task := &model.Task{TaskID: "task_video_failed", UserId: 42}

	err := ArchiveVideoTask(context.Background(), task, &model.Channel{Type: constant.ChannelTypeKling}, service.VideoArchiveSource{
		URL: "data:video/mp4,not-base64",
	})
	require.ErrorContains(t, err, "invalid video data URL")

	object, getErr := model.GetStorageObjectByBusinessID("test@videos", task.TaskID, 0)
	require.NoError(t, getErr)
	assert.Equal(t, model.StorageObjectStatusFailed, object.Status)
	assert.Contains(t, object.LastError, "invalid video data URL")
	assert.Equal(t, 1, object.ArchiveAttempts)
	assert.Greater(t, object.ArchiveNextAttemptAt, common.GetTimestamp())
	assert.Empty(t, object.ArchiveOperationID)
	assert.Zero(t, object.ArchiveLeaseExpiresAt)
}

func TestResolveVideoArchiveSourcePreservesLingganyaTaskKey(t *testing.T) {
	setupVideoProxyTest(t)

	var authorization string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"completed","video_url":"https://video.example/protected.mp4"}`))
	}))
	t.Cleanup(upstream.Close)

	baseURL := upstream.URL
	channel := &model.Channel{
		Type:    constant.ChannelTypeOpenAI,
		Key:     "current-channel-key",
		BaseURL: &baseURL,
	}
	task := &model.Task{
		PrivateData: model.TaskPrivateData{
			Key:            "stored-lingganya-key",
			UpstreamTaskID: "task_lingganya_upstream",
			BillingContext: &model.TaskBillingContext{
				VideoProtocol: dto.VideoProtocolLingganya,
			},
		},
	}

	resolvedURL, apiKey, err := resolveVideoArchiveSource(
		context.Background(),
		channel,
		task,
		service.VideoArchiveSource{},
	)

	require.NoError(t, err)
	assert.Equal(t, "https://video.example/protected.mp4", resolvedURL)
	assert.Equal(t, "stored-lingganya-key", apiKey)
	assert.Equal(t, "Bearer stored-lingganya-key", authorization)
}

func TestArchiveVideoTaskRetriesFromPersistentStaging(t *testing.T) {
	storage := &videoArchiveTestStorage{putErr: errors.New("s3 unavailable")}
	setupVideoArchiveTest(t, storage)
	task := &model.Task{
		TaskID: "task_video_staged_retry", UserId: 42,
		SubmitTime: time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC).Unix(),
	}
	channel := &model.Channel{Type: constant.ChannelTypeKling}

	err := ArchiveVideoTask(context.Background(), task, channel, service.VideoArchiveSource{
		URL: "data:video/mp4;base64,dmlkZW8tY29udGVudA==",
	})
	require.ErrorContains(t, err, "s3 unavailable")
	object, getErr := model.GetStorageObjectByBusinessID("test@videos", task.TaskID, 0)
	require.NoError(t, getErr)
	assert.Equal(t, model.StorageObjectStatusFailed, object.Status)
	assert.Equal(t, model.StorageStagingVideoAvailable, object.StagingStatus)
	assert.Equal(t, 1, object.ArchiveAttempts)
	stagedPath := filepath.Join(storage_setting.GetVideoSettings().StagingDirectory, object.StagingRelativePath)
	assert.FileExists(t, stagedPath)

	storage.putErr = nil
	require.NoError(t, ArchiveVideoTask(context.Background(), task, channel, service.VideoArchiveSource{}))
	object, getErr = model.GetStorageObjectByBusinessID("test@videos", task.TaskID, 0)
	require.NoError(t, getErr)
	assert.Equal(t, model.StorageObjectStatusAvailable, object.Status)
	assert.Equal(t, model.StorageStagingVideoDeleted, object.StagingStatus)
	assert.NoFileExists(t, stagedPath)
	require.Len(t, storage.putBodies, 2)
	assert.Equal(t, storage.putBodies[0], storage.putBodies[1])
}

func TestArchiveVideoTaskReuploadsExpiredMatchingObject(t *testing.T) {
	content := []byte("video-content")
	checksum := fmt.Sprintf("%x", sha256.Sum256(content))
	lastModified := time.Now().Add(-2 * time.Hour)
	storage := &videoArchiveTestStorage{head: objectstorage.HeadObjectResult{
		Exists: true, ContentLength: int64(len(content)), LastModified: lastModified,
		Metadata: map[string]string{
			"sha256": checksum, "business-id": "test@videos",
			"resource-id": "task_video_expired", "object-index": "0",
		},
	}}
	setupVideoArchiveTest(t, storage)
	task := &model.Task{
		TaskID: "task_video_expired", UserId: 42,
		SubmitTime: time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC).Unix(),
	}
	objectKey := "dev/user-files/test@videos/42/2026/07/29/task_video_expired/original.mp4"
	require.NoError(t, model.DB.Create(&model.StorageObject{
		BusinessID: "test@videos", ResourceID: task.TaskID, ObjectIndex: 0,
		Provider: model.StorageObjectProviderS3, Status: model.StorageObjectStatusAvailable,
		Region: "test-region", Bucket: "test-bucket", ObjectKey: objectKey,
		MimeType: "video/mp4", Extension: "mp4", SizeBytes: int64(len(content)),
		ExpiresAt: common.GetTimestamp() - 1,
	}).Error)

	require.NoError(t, ArchiveVideoTask(context.Background(), task, &model.Channel{Type: constant.ChannelTypeKling}, service.VideoArchiveSource{
		URL: "data:video/mp4;base64,dmlkZW8tY29udGVudA==",
	}))
	require.Len(t, storage.putInputs, 1)
	assert.True(t, bytes.Equal(content, storage.putBodies[0]))

	stored, err := model.GetStorageObjectByBusinessID("test@videos", task.TaskID, 0)
	require.NoError(t, err)
	assert.Equal(t, model.StorageObjectStatusAvailable, stored.Status)
	assert.Greater(t, stored.ExpiresAt, common.GetTimestamp())
}

func TestRetryVideoStorageUploadDeduplicatesActiveTask(t *testing.T) {
	setupVideoArchiveTest(t, &videoArchiveTestStorage{})
	require.NoError(t, model.DB.AutoMigrate(&model.Task{}, &model.SystemTask{}))
	task := &model.Task{
		TaskID: "task_video_retry_" + strings.Repeat("x", 80), UserId: 42, Status: model.TaskStatusSuccess,
		Action: constant.TaskActionGenerate,
	}
	require.NoError(t, model.DB.Create(task).Error)

	retry := func() map[string]any {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/task/"+task.TaskID+"/retry-video-upload", nil)
		ctx.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
		RetryVideoStorageUpload(ctx)
		assert.Equal(t, http.StatusOK, recorder.Code)
		var response map[string]any
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		return response
	}

	first := retry()
	second := retry()
	firstData, ok := first["data"].(map[string]any)
	require.True(t, ok)
	secondData, ok := second["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, firstData["created"])
	assert.Equal(t, false, secondData["created"])
	assert.Equal(t, firstData["operation_id"], secondData["operation_id"])

	var count int64
	require.NoError(t, model.DB.Model(&model.SystemTask{}).
		Where("type = ?", model.SystemTaskTypeVideoStorageRetry).
		Count(&count).Error)
	assert.Equal(t, int64(1), count)
	object, err := model.GetStorageObjectByBusinessID("test@videos", task.TaskID, 0)
	require.NoError(t, err)
	assert.Equal(t, model.StorageObjectStatusFailed, object.Status)
	assert.Equal(t, 0, object.ArchiveAttempts)
	assert.Equal(t, 3, object.ArchiveMaxAttempts)
	assert.Greater(t, object.ArchiveRetryDeadlineAt, common.GetTimestamp())
	assert.GreaterOrEqual(t, object.ArchiveNextAttemptAt, common.GetTimestamp())
}

func TestBatchVideoStorageUploadAcceptsTaskWithoutAutomaticStorageAndSkipsOthers(t *testing.T) {
	setupVideoArchiveTest(t, &videoArchiveTestStorage{})
	require.NoError(t, model.DB.AutoMigrate(&model.Task{}, &model.SystemTask{}))
	require.NoError(t, model.DB.Create([]*model.Task{
		{
			TaskID: "task_video_batch", Status: model.TaskStatusSuccess, Action: constant.TaskActionGenerate,
			PrivateData: model.TaskPrivateData{VideoS3StorageEnabled: false},
		},
		{TaskID: "task_video_pending", Status: model.TaskStatusInProgress, Action: constant.TaskActionGenerate},
		{TaskID: "task_audio_batch", Status: model.TaskStatusSuccess, Action: constant.SunoActionMusic},
	}).Error)

	body := []byte(`{"task_ids":["task_video_batch","task_video_pending","task_audio_batch","missing_task"]}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/task/batch-video-upload", bytes.NewReader(body))
	BatchVideoStorageUpload(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Accepted []batchVideoStorageUploadAccepted `json:"accepted"`
			Skipped  []batchVideoStorageUploadSkipped  `json:"skipped"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	require.Len(t, response.Data.Accepted, 1)
	assert.Equal(t, "task_video_batch", response.Data.Accepted[0].TaskID)
	assert.True(t, response.Data.Accepted[0].Created)
	assert.Len(t, response.Data.Skipped, 3)
}

func TestVideoStorageRetryHandlerSchedulesDueObjectsOnly(t *testing.T) {
	setupVideoArchiveTest(t, &videoArchiveTestStorage{})
	now := common.GetTimestamp()
	require.NoError(t, model.DB.Create(&model.StorageObject{
		BusinessID: "test@videos", ResourceID: "task_video_due", ObjectIndex: 0,
		Status: model.StorageObjectStatusFailed, ArchiveAttempts: 1, ArchiveMaxAttempts: 3,
		ArchiveRetryDeadlineAt: now + 3600, ArchiveNextAttemptAt: now,
	}).Error)

	assert.True(t, (videoStorageRetryHandler{}).Enabled())
	require.NoError(t, model.DB.Model(&model.StorageObject{}).
		Where("resource_id = ?", "task_video_due").
		Update("archive_next_attempt_at", now+60).Error)
	assert.False(t, (videoStorageRetryHandler{}).Enabled())
}

func TestVideoStorageRetryHandlerRecoversExpiredAndLegacyUploads(t *testing.T) {
	setupVideoArchiveTest(t, &videoArchiveTestStorage{})
	now := common.GetTimestamp()
	objects := []*model.StorageObject{
		{
			BusinessID: "test@videos", ResourceID: "task_video_expired_lease", ObjectIndex: 0,
			Status: model.StorageObjectStatusUploading, ArchiveAttempts: 0, ArchiveMaxAttempts: 3,
			ArchiveRetryDeadlineAt: now - 1, ArchiveOperationID: "expired-operation",
			ArchiveLeaseExpiresAt: now - 1,
		},
		{
			BusinessID: "test@videos", ResourceID: "task_video_legacy_upload", ObjectIndex: 0,
			Status: model.StorageObjectStatusUploading, ArchiveAttempts: 1, ArchiveMaxAttempts: 3,
			ArchiveRetryDeadlineAt: now - 1,
		},
		{
			BusinessID: "test@videos", ResourceID: "task_video_active_upload", ObjectIndex: 0,
			Status: model.StorageObjectStatusUploading, ArchiveAttempts: 0, ArchiveMaxAttempts: 3,
			ArchiveRetryDeadlineAt: now + 3600, ArchiveOperationID: "active-operation",
			ArchiveLeaseExpiresAt: now + 60,
		},
	}
	require.NoError(t, model.DB.Create(objects).Error)
	require.NoError(t, model.DB.Model(&model.StorageObject{}).
		Where("resource_id IN ?", []string{"task_video_legacy_upload", "task_video_active_upload"}).
		UpdateColumn("updated_at", now-61).Error)

	assert.True(t, (videoStorageRetryHandler{}).Enabled())
	recovered, err := recoverExpiredVideoStorageUploads(
		context.Background(), storage_setting.GetVideoSettings(), now, 10,
	)
	require.NoError(t, err)
	assert.Equal(t, 2, recovered)

	for _, taskID := range []string{"task_video_expired_lease", "task_video_legacy_upload"} {
		object, getErr := model.GetStorageObjectByBusinessID("test@videos", taskID, 0)
		require.NoError(t, getErr)
		assert.Equal(t, model.StorageObjectStatusFailed, object.Status)
		assert.Contains(t, object.LastError, "timed out")
		assert.Zero(t, object.ArchiveNextAttemptAt)
		assert.Empty(t, object.ArchiveOperationID)
		assert.Zero(t, object.ArchiveLeaseExpiresAt)
	}

	active, err := model.GetStorageObjectByBusinessID("test@videos", "task_video_active_upload", 0)
	require.NoError(t, err)
	assert.Equal(t, model.StorageObjectStatusUploading, active.Status)
	assert.Equal(t, "active-operation", active.ArchiveOperationID)
	assert.Equal(t, now+60, active.ArchiveLeaseExpiresAt)
	assert.False(t, (videoStorageRetryHandler{}).Enabled())
}

func TestNextVideoArchiveAttemptStopsAtRetryDeadline(t *testing.T) {
	object := &model.StorageObject{
		ArchiveAttempts:        0,
		ArchiveMaxAttempts:     3,
		ArchiveRetryDeadlineAt: 1015,
	}

	assert.Equal(t, int64(0), nextVideoArchiveAttemptAt(object, 1000))
}
