package controller

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/QuantumNous/new-api/setting/storage_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTaskControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalRedisEnabled := common.RedisEnabled
	originalMainDatabaseType := common.MainDatabaseType()
	originalLogDatabaseType := common.LogDatabaseType()

	common.MemoryCacheEnabled = false
	common.RedisEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.Channel{}))
	require.NoError(t, db.AutoMigrate(&model.StorageObject{}))
	require.NoError(t, db.AutoMigrate(&model.Task{}))

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		common.RedisEnabled = originalRedisEnabled
		common.SetDatabaseTypes(originalMainDatabaseType, originalLogDatabaseType)
	})

	return db
}

func TestTasksToDtoIncludesChannelNameForAdminOnly(t *testing.T) {
	db := setupTaskControllerTestDB(t)
	channel := &model.Channel{
		Id:   3,
		Type: constant.ChannelTypeOpenAI,
		Key:  "test-key",
		Name: "seedance",
	}
	require.NoError(t, db.Create(channel).Error)
	tasks := []*model.Task{{ChannelId: channel.Id}}

	adminTasks := tasksToDto(tasks, true)
	userTasks := tasksToDto(tasks, false)

	require.Len(t, adminTasks, 1)
	require.Len(t, userTasks, 1)
	assert.Equal(t, "seedance", adminTasks[0].ChannelName)
	assert.Empty(t, userTasks[0].ChannelName)
}

func TestTasksToDtoIncludesFailureHTTPTraceForAdminOnly(t *testing.T) {
	setupTaskControllerTestDB(t)
	trace := &dto.TaskUpstreamHTTPTrace{
		SubmitRequest: &dto.TaskHTTPMessage{Method: http.MethodPost, URL: "https://upstream.example/v1/videos"},
		PollResponse:  &dto.TaskHTTPMessage{StatusCode: http.StatusForbidden, Body: `{"error":"forbidden"}`},
	}
	task := &model.Task{
		TaskID: "task_failure_trace", Status: model.TaskStatusFailure,
		PrivateData: model.TaskPrivateData{UpstreamHTTPTrace: trace},
	}

	adminTasks := tasksToDto([]*model.Task{task}, true)
	userTasks := tasksToDto([]*model.Task{task}, false)

	require.Len(t, adminTasks, 1)
	assert.Equal(t, trace, adminTasks[0].UpstreamHTTPTrace)
	require.Len(t, userTasks, 1)
	assert.Nil(t, userTasks[0].UpstreamHTTPTrace)
}

func TestTasksToDtoIncludesVideoStorageFailureForAdminOnly(t *testing.T) {
	db := setupTaskControllerTestDB(t)
	channelSetting := `{"video_content_proxy_enabled":true}`
	channel := &model.Channel{Id: 31, Type: constant.ChannelTypeOpenAI, Name: "video", Setting: &channelSetting}
	require.NoError(t, db.Create(channel).Error)
	common.OptionMapRWMutex.Lock()
	originalOptions := common.OptionMap
	common.OptionMap = map[string]string{storage_setting.OptionVideoBusinessID: "test@videos"}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptions
		common.OptionMapRWMutex.Unlock()
	})

	task := &model.Task{
		TaskID: "task_video_storage_failure", Action: constant.TaskActionGenerate,
		ChannelId: channel.Id, Status: model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{VideoS3StorageEnabled: true},
	}
	task.PrivateData.ResultURL = "https://upstream.example/video.mp4"
	require.NoError(t, db.Create(&model.StorageObject{
		BusinessID: "test@videos", ResourceID: task.TaskID, ObjectIndex: 0,
		Status: model.StorageObjectStatusFailed, LastError: "upload failed",
	}).Error)

	adminTasks := tasksToDto([]*model.Task{task}, true)
	userTasks := tasksToDto([]*model.Task{task}, false)

	require.Len(t, adminTasks, 1)
	assert.True(t, adminTasks[0].VideoS3StorageEnabled)
	assert.Equal(t, model.StorageObjectStatusFailed, adminTasks[0].VideoStorageStatus)
	assert.Equal(t, "upload failed", adminTasks[0].VideoStorageError)
	assert.Equal(t, taskcommon.BuildProxyURL(task.TaskID), adminTasks[0].ResultURL)
	require.Len(t, userTasks, 1)
	assert.False(t, userTasks[0].VideoS3StorageEnabled)
	assert.Empty(t, userTasks[0].VideoStorageStatus)
	assert.Empty(t, userTasks[0].VideoStorageError)
	assert.Equal(t, taskcommon.BuildProxyURL(task.TaskID), userTasks[0].ResultURL)
}

func TestTasksToDtoUsesContentURLWhenPreferredS3ObjectExists(t *testing.T) {
	db := setupTaskControllerTestDB(t)
	channelSetting := `{"video_s3_preferred":true}`
	channel := &model.Channel{Id: 32, Type: constant.ChannelTypeOpenAI, Name: "video", Setting: &channelSetting}
	require.NoError(t, db.Create(channel).Error)
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

	require.NoError(t, db.Create(&model.StorageObject{
		BusinessID: "test@videos", ResourceID: "task_video_stored", ObjectIndex: 0,
		Status: model.StorageObjectStatusAvailable, Region: "test-region",
		Bucket: "test-bucket", ObjectKey: "prod/user-files/test@videos/video.mp4",
		MimeType: "video/mp4", ExpiresAt: common.GetTimestamp() + 3600,
	}).Error)
	task := &model.Task{
		TaskID: "task_video_stored", Action: constant.TaskActionGenerate,
		ChannelId: channel.Id, Status: model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://upstream.example/video.mp4",
		},
	}
	task.SetData(map[string]any{"video_url": "https://upstream.example/video.mp4"})

	items := tasksToDto([]*model.Task{task}, true)

	require.Len(t, items, 1)
	assert.Equal(t, taskcommon.BuildProxyURL(task.TaskID), items[0].ResultURL)
	assert.Equal(t, model.StorageObjectStatusAvailable, items[0].VideoStorageStatus)
	var data map[string]any
	require.NoError(t, common.Unmarshal(items[0].Data, &data))
	assert.Equal(t, taskcommon.BuildProxyURL(task.TaskID), data["video_url"])
	assert.Equal(t, "https://upstream.example/video.mp4", task.PrivateData.ResultURL)

	channelSetting = `{}`
	require.NoError(t, db.Model(channel).Update("setting", channelSetting).Error)
	items = tasksToDto([]*model.Task{task}, true)
	require.Len(t, items, 1)
	assert.Equal(t, taskcommon.BuildProxyURL(task.TaskID), items[0].ResultURL)
}

func TestTasksToDtoIgnoresChannelVideoDeliveryPolicy(t *testing.T) {
	db := setupTaskControllerTestDB(t)
	channelSetting := `{"video_content_proxy_enabled":true}`
	channel := &model.Channel{Id: 33, Type: constant.ChannelTypeOpenAI, Name: "video", Setting: &channelSetting}
	require.NoError(t, db.Create(channel).Error)

	task := &model.Task{
		TaskID: "task_dynamic_delivery", ChannelId: channel.Id,
		Action: constant.TaskActionGenerate, Status: model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{ResultURL: "https://upstream.example/video.mp4"},
	}

	items := tasksToDto([]*model.Task{task}, false)
	require.Len(t, items, 1)
	assert.Equal(t, taskcommon.BuildProxyURL(task.TaskID), items[0].ResultURL)

	channelSetting = `{}`
	require.NoError(t, db.Model(channel).Update("setting", channelSetting).Error)
	items = tasksToDto([]*model.Task{task}, false)
	require.Len(t, items, 1)
	assert.Equal(t, taskcommon.BuildProxyURL(task.TaskID), items[0].ResultURL)
}
