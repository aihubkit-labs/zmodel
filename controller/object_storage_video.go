package controller

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/objectstorage"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/storage_setting"

	"github.com/gin-gonic/gin"
)

type videoObjectStorageSettingsResponse struct {
	Endpoint                  string `json:"endpoint"`
	Region                    string `json:"region"`
	Bucket                    string `json:"bucket"`
	AccessKey                 string `json:"access_key"`
	SecretConfigured          bool   `json:"secret_configured"`
	S3KeyPrefix               string `json:"s3_key_prefix"`
	BusinessID                string `json:"business_id"`
	StagingDirectory          string `json:"staging_directory"`
	RetentionSeconds          int64  `json:"retention_seconds"`
	PresignSeconds            int64  `json:"presign_seconds"`
	ArchiveTimeoutSeconds     int64  `json:"archive_timeout_seconds"`
	ArchiveMaxAttempts        int    `json:"archive_max_attempts"`
	ArchiveRetryWindowSeconds int64  `json:"archive_retry_window_seconds"`
	CleanupIntervalSeconds    int64  `json:"cleanup_interval_seconds"`
}

type updateVideoObjectStorageSettingsRequest struct {
	Endpoint                  string  `json:"endpoint"`
	Region                    string  `json:"region"`
	Bucket                    string  `json:"bucket"`
	AccessKey                 string  `json:"access_key"`
	SecretAccessKey           string  `json:"secret_access_key"`
	S3KeyPrefix               *string `json:"s3_key_prefix"`
	BusinessID                string  `json:"business_id"`
	StagingDirectory          string  `json:"staging_directory"`
	RetentionSeconds          int64   `json:"retention_seconds"`
	PresignSeconds            int64   `json:"presign_seconds"`
	ArchiveTimeoutSeconds     int64   `json:"archive_timeout_seconds"`
	ArchiveMaxAttempts        int     `json:"archive_max_attempts"`
	ArchiveRetryWindowSeconds int64   `json:"archive_retry_window_seconds"`
	CleanupIntervalSeconds    int64   `json:"cleanup_interval_seconds"`
}

func GetVideoObjectStorageSettings(c *gin.Context) {
	settings := storage_setting.GetVideoSettings()
	common.ApiSuccess(c, videoObjectStorageSettingsResponse{
		Endpoint:                  settings.Endpoint,
		Region:                    settings.Region,
		Bucket:                    settings.Bucket,
		AccessKey:                 settings.AccessKey,
		SecretConfigured:          strings.TrimSpace(settings.SecretAccessKey) != "",
		S3KeyPrefix:               settings.S3KeyPrefix,
		BusinessID:                settings.BusinessID,
		StagingDirectory:          settings.StagingDirectory,
		RetentionSeconds:          settings.RetentionSeconds,
		PresignSeconds:            settings.PresignSeconds,
		ArchiveTimeoutSeconds:     settings.ArchiveTimeoutSeconds,
		ArchiveMaxAttempts:        settings.ArchiveMaxAttempts,
		ArchiveRetryWindowSeconds: settings.ArchiveRetryWindowSeconds,
		CleanupIntervalSeconds:    settings.CleanupIntervalSeconds,
	})
}

func UpdateVideoObjectStorageSettings(c *gin.Context) {
	var request updateVideoObjectStorageSettingsRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "invalid video object storage settings")
		return
	}

	current := storage_setting.GetVideoSettings()
	secret := request.SecretAccessKey
	if secret == "" {
		secret = current.SecretAccessKey
	}
	prefix := current.S3KeyPrefix
	if request.S3KeyPrefix != nil {
		prefix = strings.TrimSpace(*request.S3KeyPrefix)
	}
	candidate := storage_setting.VideoSettings{
		Endpoint:                  strings.TrimSpace(request.Endpoint),
		Region:                    strings.TrimSpace(request.Region),
		Bucket:                    strings.TrimSpace(request.Bucket),
		AccessKey:                 strings.TrimSpace(request.AccessKey),
		SecretAccessKey:           secret,
		S3KeyPrefix:               prefix,
		BusinessID:                strings.TrimSpace(request.BusinessID),
		StagingDirectory:          strings.TrimSpace(request.StagingDirectory),
		RetentionSeconds:          request.RetentionSeconds,
		PresignSeconds:            request.PresignSeconds,
		ArchiveTimeoutSeconds:     request.ArchiveTimeoutSeconds,
		ArchiveMaxAttempts:        request.ArchiveMaxAttempts,
		ArchiveRetryWindowSeconds: request.ArchiveRetryWindowSeconds,
		CleanupIntervalSeconds:    request.CleanupIntervalSeconds,
	}
	if err := candidate.Validate(); err != nil {
		common.ApiError(c, err)
		return
	}
	stagingDirectoryChanged := filepath.Clean(current.StagingDirectory) != filepath.Clean(candidate.StagingDirectory)
	if stagingDirectoryChanged {
		inUseCount, err := model.CountVideoStorageStagingInUse(current.BusinessID)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if inUseCount > 0 {
			common.ApiErrorMsg(c, "cannot change video staging directory while staged videos exist")
			return
		}
	}
	if err := service.CheckVideoStagingDirectory(candidate.StagingDirectory); err != nil {
		logger.LogWarn(c.Request.Context(), common.LocalLogPreview("video staging directory probe failed: "+err.Error()))
		common.ApiErrorMsg(c, "video staging directory is unavailable")
		return
	}

	physicalLocationChanged := current.Endpoint != candidate.Endpoint || current.Region != candidate.Region || current.Bucket != candidate.Bucket ||
		current.S3KeyPrefix != candidate.S3KeyPrefix || current.BusinessID != candidate.BusinessID
	if physicalLocationChanged {
		activeCount, err := model.CountUndeletedStorageObjectsByBusinessID(current.BusinessID)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if activeCount > 0 {
			common.ApiErrorMsg(c, "cannot change video object storage location while active objects exist")
			return
		}
	}
	if err := probeVideoObjectStorage(c.Request.Context(), candidate); err != nil {
		logger.LogWarn(c.Request.Context(), common.LocalLogPreview(err.Error()))
		common.ApiErrorMsg(c, "video object storage probe failed; check the S3 credentials and object permissions")
		return
	}
	if err := model.UpdateOptionsBulk(candidate.OptionValues()); err != nil {
		common.ApiError(c, err)
		return
	}
	GetVideoObjectStorageSettings(c)
}

func probeVideoObjectStorage(parent context.Context, settings storage_setting.VideoSettings) (resultErr error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	storage, err := objectstorage.NewStorage(ctx, objectstorage.Config{
		Endpoint: settings.Endpoint, Region: settings.Region,
		AccessKey: settings.AccessKey, SecretAccessKey: settings.SecretAccessKey,
	})
	if err != nil {
		return fmt.Errorf("create video object storage client: %w", err)
	}
	random, err := common.GenerateRandomCharsKey(24)
	if err != nil {
		return err
	}
	key := videoObjectRoot(settings.S3KeyPrefix, settings.BusinessID) + "/.probe/" + random
	if _, err := storage.PutObject(ctx, objectstorage.PutObjectInput{
		Bucket: settings.Bucket, Key: key, Body: bytes.NewReader([]byte("ok")),
		ContentType: "text/plain", Metadata: map[string]string{"probe": "true"},
	}); err != nil {
		return fmt.Errorf("write video object storage probe: %w", err)
	}
	defer func() {
		if err := storage.DeleteObject(ctx, objectstorage.DeleteObjectInput{Bucket: settings.Bucket, Key: key}); err != nil && resultErr == nil {
			resultErr = fmt.Errorf("delete video object storage probe: %w", err)
		}
	}()
	head, err := storage.HeadObject(ctx, objectstorage.HeadObjectInput{Bucket: settings.Bucket, Key: key})
	if err != nil {
		return fmt.Errorf("verify video object storage probe: %w", err)
	}
	if !head.Exists || head.ContentLength != 2 {
		return fmt.Errorf("video object storage probe returned an unexpected result")
	}
	return nil
}
