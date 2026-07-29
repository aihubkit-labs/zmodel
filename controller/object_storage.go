package controller

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/objectstorage"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/storage_setting"

	"github.com/gin-gonic/gin"
)

type objectStorageSettingsResponse struct {
	Endpoint                  string `json:"endpoint"`
	Region                    string `json:"region"`
	Bucket                    string `json:"bucket"`
	AccessKey                 string `json:"access_key"`
	SecretConfigured          bool   `json:"secret_configured"`
	RetentionSeconds          int64  `json:"retention_seconds"`
	PresignSeconds            int64  `json:"presign_seconds"`
	ArchiveTimeoutSeconds     int64  `json:"archive_timeout_seconds"`
	ArchiveMaxAttempts        int    `json:"archive_max_attempts"`
	ArchiveRetryWindowSeconds int64  `json:"archive_retry_window_seconds"`
	CleanupIntervalSeconds    int64  `json:"cleanup_interval_seconds"`
}

type updateObjectStorageSettingsRequest struct {
	Endpoint                  string `json:"endpoint"`
	Region                    string `json:"region"`
	Bucket                    string `json:"bucket"`
	AccessKey                 string `json:"access_key"`
	SecretAccessKey           string `json:"secret_access_key"`
	RetentionSeconds          int64  `json:"retention_seconds"`
	PresignSeconds            int64  `json:"presign_seconds"`
	ArchiveTimeoutSeconds     int64  `json:"archive_timeout_seconds"`
	ArchiveMaxAttempts        int    `json:"archive_max_attempts"`
	ArchiveRetryWindowSeconds int64  `json:"archive_retry_window_seconds"`
	CleanupIntervalSeconds    int64  `json:"cleanup_interval_seconds"`
}

func GetObjectStorageSettings(c *gin.Context) {
	settings := storage_setting.GetSettings()
	common.ApiSuccess(c, objectStorageSettingsResponse{
		Endpoint:                  settings.Endpoint,
		Region:                    settings.Region,
		Bucket:                    settings.Bucket,
		AccessKey:                 settings.AccessKey,
		SecretConfigured:          strings.TrimSpace(settings.SecretAccessKey) != "",
		RetentionSeconds:          settings.RetentionSeconds,
		PresignSeconds:            settings.PresignSeconds,
		ArchiveTimeoutSeconds:     settings.ArchiveTimeoutSeconds,
		ArchiveMaxAttempts:        settings.ArchiveMaxAttempts,
		ArchiveRetryWindowSeconds: settings.ArchiveRetryWindowSeconds,
		CleanupIntervalSeconds:    settings.CleanupIntervalSeconds,
	})
}

func UpdateObjectStorageSettings(c *gin.Context) {
	var request updateObjectStorageSettingsRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "invalid object storage settings")
		return
	}
	current := storage_setting.GetSettings()
	secret := request.SecretAccessKey
	if secret == "" {
		secret = current.SecretAccessKey
	}
	candidate := storage_setting.Settings{
		Endpoint:                  strings.TrimSpace(request.Endpoint),
		Region:                    strings.TrimSpace(request.Region),
		Bucket:                    strings.TrimSpace(request.Bucket),
		AccessKey:                 strings.TrimSpace(request.AccessKey),
		SecretAccessKey:           secret,
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
	physicalLocationChanged := current.Endpoint != candidate.Endpoint || current.Region != candidate.Region || current.Bucket != candidate.Bucket
	rebindTaskIDs := []string{}
	if physicalLocationChanged {
		activeCount, err := model.CountActiveStorageObjects()
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if activeCount > 0 {
			common.ApiErrorMsg(c, "cannot change object storage location while active objects exist")
			return
		}
		rebindTaskIDs, err = prepareAsyncImageStorageRebind(c.Request.Context(), current)
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}
	if err := probeObjectStorage(c.Request.Context(), candidate); err != nil {
		common.ApiError(c, fmt.Errorf("object storage probe failed: %w", err))
		return
	}
	values := map[string]string{
		storage_setting.OptionS3Endpoint:                candidate.Endpoint,
		storage_setting.OptionS3Region:                  candidate.Region,
		storage_setting.OptionS3Bucket:                  candidate.Bucket,
		storage_setting.OptionS3AccessKey:               candidate.AccessKey,
		storage_setting.OptionS3SecretAccessKey:         candidate.SecretAccessKey,
		storage_setting.OptionRetentionSeconds:          fmt.Sprintf("%d", candidate.RetentionSeconds),
		storage_setting.OptionPresignSeconds:            fmt.Sprintf("%d", candidate.PresignSeconds),
		storage_setting.OptionArchiveTimeoutSeconds:     fmt.Sprintf("%d", candidate.ArchiveTimeoutSeconds),
		storage_setting.OptionArchiveMaxAttempts:        fmt.Sprintf("%d", candidate.ArchiveMaxAttempts),
		storage_setting.OptionArchiveRetryWindowSeconds: fmt.Sprintf("%d", candidate.ArchiveRetryWindowSeconds),
		storage_setting.OptionCleanupIntervalSeconds:    fmt.Sprintf("%d", candidate.CleanupIntervalSeconds),
	}
	var updateErr error
	if physicalLocationChanged && len(rebindTaskIDs) > 0 {
		updateErr = model.UpdateObjectStorageOptionsWithRebind(
			values,
			rebindTaskIDs,
			candidate.Endpoint,
			candidate.Region,
			candidate.Bucket,
			common.GetTimestamp()+candidate.ArchiveRetryWindowSeconds,
		)
	} else {
		updateErr = model.UpdateOptionsBulk(values)
	}
	if updateErr != nil {
		common.ApiError(c, updateErr)
		return
	}
	if len(rebindTaskIDs) > 0 {
		_, _, _ = service.EnqueueSystemTask(model.SystemTaskTypeAsyncImageProcess, nil)
	}
	GetObjectStorageSettings(c)
}

func prepareAsyncImageStorageRebind(ctx context.Context, current storage_setting.Settings) ([]string, error) {
	candidates, err := model.ListAsyncImageStorageRebindCandidates()
	if err != nil {
		return nil, err
	}
	taskIDs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		manifest, err := decodeAsyncImageManifest(candidate.Task.ArchiveManifest)
		if err != nil || len(manifest) != len(candidate.Objects) {
			return nil, fmt.Errorf("failed task %s does not have a complete archive manifest", candidate.Task.TaskID)
		}
		for index, object := range candidate.Objects {
			item := manifest[index]
			if item.Index != object.ObjectIndex || object.StagingStatus != model.StorageStagingAvailable {
				return nil, fmt.Errorf("failed task %s does not have complete staged files", candidate.Task.TaskID)
			}
			file, err := service.ReadStagedImage(item)
			if err != nil {
				return nil, fmt.Errorf("failed task %s has missing or damaged staged files", candidate.Task.TaskID)
			}
			_ = file.Close()
			switch object.Status {
			case model.StorageObjectStatusDeleted:
				continue
			case model.StorageObjectStatusFailed:
				storage, err := objectstorage.NewStorage(ctx, objectstorage.Config{
					Endpoint: object.Endpoint, Region: object.Region,
					AccessKey: current.AccessKey, SecretAccessKey: current.SecretAccessKey,
				})
				if err != nil {
					return nil, fmt.Errorf("failed to inspect the old object storage location: %w", err)
				}
				head, err := storage.HeadObject(ctx, objectstorage.HeadObjectInput{Bucket: object.Bucket, Key: object.ObjectKey})
				if err != nil {
					return nil, fmt.Errorf("failed to inspect the old object storage location: %w", err)
				}
				if head.Exists {
					return nil, fmt.Errorf("cannot change object storage location because task %s still has an object in the old location", candidate.Task.TaskID)
				}
			default:
				return nil, fmt.Errorf("cannot change object storage location while task %s has active objects", candidate.Task.TaskID)
			}
		}
		taskIDs = append(taskIDs, candidate.Task.TaskID)
	}
	return taskIDs, nil
}

func probeObjectStorage(parent context.Context, settings storage_setting.Settings) error {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	storage, err := objectstorage.NewStorage(ctx, objectstorage.Config{
		Endpoint:        settings.Endpoint,
		Region:          settings.Region,
		AccessKey:       settings.AccessKey,
		SecretAccessKey: settings.SecretAccessKey,
	})
	if err != nil {
		return err
	}
	random, err := common.GenerateRandomCharsKey(24)
	if err != nil {
		return err
	}
	key := "zmodel-object-storage-probe/" + random
	_, err = storage.PutObject(ctx, objectstorage.PutObjectInput{
		Bucket:      settings.Bucket,
		Key:         key,
		Body:        bytes.NewReader([]byte("ok")),
		ContentType: "text/plain",
		Metadata:    map[string]string{"probe": "true"},
	})
	if err != nil {
		return err
	}
	head, err := storage.HeadObject(ctx, objectstorage.HeadObjectInput{Bucket: settings.Bucket, Key: key})
	if err != nil {
		return err
	}
	if !head.Exists || head.ContentLength != 2 {
		return fmt.Errorf("probe object verification failed")
	}
	return storage.DeleteObject(ctx, objectstorage.DeleteObjectInput{Bucket: settings.Bucket, Key: key})
}
