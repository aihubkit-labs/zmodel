package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/objectstorage"
	"github.com/QuantumNous/new-api/setting/storage_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type videoPresignTestStorage struct {
	input objectstorage.PresignGetObjectInput
}

func (*videoPresignTestStorage) PutObject(context.Context, objectstorage.PutObjectInput) (objectstorage.PutObjectResult, error) {
	return objectstorage.PutObjectResult{}, nil
}

func (*videoPresignTestStorage) HeadObject(context.Context, objectstorage.HeadObjectInput) (objectstorage.HeadObjectResult, error) {
	return objectstorage.HeadObjectResult{}, nil
}

func (*videoPresignTestStorage) DeleteObject(context.Context, objectstorage.DeleteObjectInput) error {
	return nil
}

func (storage *videoPresignTestStorage) PresignGetObject(_ context.Context, input objectstorage.PresignGetObjectInput) (string, error) {
	storage.input = input
	return "https://storage.example/signed-video", nil
}

func TestPresignVideoObjectUsesConfiguredBusinessNamespace(t *testing.T) {
	originalDB := model.DB
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "video-presign.db")), &gorm.Config{})
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
		storage_setting.OptionVideoS3KeyPrefix:               "prod",
		storage_setting.OptionVideoBusinessID:                "test@videos",
		storage_setting.OptionVideoStagingDirectory:          t.TempDir(),
		storage_setting.OptionVideoPresignSeconds:            "600",
		storage_setting.OptionVideoArchiveMaxAttempts:        "3",
		storage_setting.OptionVideoArchiveRetryWindowSeconds: "3600",
	}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptions
		common.OptionMapRWMutex.Unlock()
	})

	require.NoError(t, db.Create(&model.StorageObject{
		BusinessID: "test@videos", ResourceID: "task_video_presign", ObjectIndex: 0,
		Status: model.StorageObjectStatusAvailable, Endpoint: "https://s3.example.com",
		Region: "test-region", Bucket: "test-bucket", ObjectKey: "prod/user-files/test@videos/video.mp4",
		MimeType: "video/mp4", ExpiresAt: common.GetTimestamp() + 3600,
	}).Error)
	storage := &videoPresignTestStorage{}
	originalFactory := objectstorage.NewStorage
	objectstorage.NewStorage = func(context.Context, objectstorage.Config) (objectstorage.Storage, error) {
		return storage, nil
	}
	t.Cleanup(func() { objectstorage.NewStorage = originalFactory })

	url, err := PresignVideoObject(context.Background(), "task_video_presign", "result.mp4")
	require.NoError(t, err)
	assert.Equal(t, "https://storage.example/signed-video", url)
	assert.Equal(t, "test-bucket", storage.input.Bucket)
	assert.Equal(t, "prod/user-files/test@videos/video.mp4", storage.input.Key)
	assert.Equal(t, 10*time.Minute, storage.input.Expires)
	assert.Contains(t, storage.input.ResponseDisposition, "result.mp4")
}
