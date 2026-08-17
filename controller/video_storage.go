package controller

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/objectstorage"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/storage_setting"
	"github.com/gin-gonic/gin"
)

func videoObjectRoot(s3KeyPrefix string, businessID string) string {
	return s3KeyPrefix + "/user-files/" + businessID
}

func videoObjectKey(settings storage_setting.VideoSettings, task *model.Task, extension string) string {
	createdAt := time.Now().UTC()
	if task.SubmitTime > 0 {
		createdAt = time.Unix(task.SubmitTime, 0).UTC()
	}
	return fmt.Sprintf("%s/%d/%s/%s/%s/%s/original.%s",
		videoObjectRoot(settings.S3KeyPrefix, settings.BusinessID),
		task.UserId, createdAt.Format("2006"), createdAt.Format("01"), createdAt.Format("02"), task.TaskID, extension,
	)
}

func videoStagingRelativePath(task *model.Task) string {
	createdAt := time.Now().UTC()
	if task.SubmitTime > 0 {
		createdAt = time.Unix(task.SubmitTime, 0).UTC()
	}
	return filepath.Join(
		"videos", strconv.Itoa(task.UserId), createdAt.Format("2006"), createdAt.Format("01"),
		createdAt.Format("02"), task.TaskID, "original.video",
	)
}

func ensureVideoStorageObject(settings storage_setting.VideoSettings, task *model.Task) (*model.StorageObject, error) {
	object := &model.StorageObject{
		BusinessID: settings.BusinessID, ResourceID: task.TaskID, ObjectIndex: 0,
		Provider: model.StorageObjectProviderS3, Status: model.StorageObjectStatusUploading,
		Endpoint: settings.Endpoint, Region: settings.Region, Bucket: settings.Bucket,
		ObjectKey: videoObjectKey(settings, task, "mp4"), MimeType: "video/mp4",
		Extension: "mp4", StagingStatus: model.StorageStagingVideoPending,
		ArchiveMaxAttempts:     settings.ArchiveMaxAttempts,
		ArchiveRetryDeadlineAt: common.GetTimestamp() + settings.ArchiveRetryWindowSeconds,
	}
	if err := model.GetOrCreateStorageObject(object); err != nil {
		return nil, err
	}
	return object, nil
}

func ArchiveVideoTask(parent context.Context, task *model.Task, channel *model.Channel, source service.VideoArchiveSource) (resultErr error) {
	if task == nil || channel == nil {
		return errors.New("invalid video archive task")
	}
	settings := storage_setting.GetVideoSettings()
	if err := settings.Validate(); err != nil {
		return fmt.Errorf("invalid video object storage settings: %w", err)
	}
	object, err := ensureVideoStorageObject(settings, task)
	if err != nil {
		return err
	}
	if object.Status == model.StorageObjectStatusAvailable && object.ExpiresAt > common.GetTimestamp() {
		return nil
	}
	if object.ArchiveMaxAttempts == 0 || object.ArchiveRetryDeadlineAt == 0 {
		object.ArchiveMaxAttempts = settings.ArchiveMaxAttempts
		object.ArchiveRetryDeadlineAt = common.GetTimestamp() + settings.ArchiveRetryWindowSeconds
		if err := model.InitializeVideoStorageArchive(object.ID, object.ArchiveMaxAttempts, object.ArchiveRetryDeadlineAt); err != nil {
			return err
		}
	}
	operationID, err := common.GenerateRandomCharsKey(32)
	if err != nil {
		return err
	}
	leaseExpiresAt := common.GetTimestamp() + settings.ArchiveTimeoutSeconds
	if err := model.MarkVideoStorageArchiveUploading(
		object.ID,
		object.Status,
		object.ArchiveOperationID,
		operationID,
		leaseExpiresAt,
	); err != nil {
		return err
	}
	object.Status = model.StorageObjectStatusUploading
	object.ArchiveOperationID = operationID
	object.ArchiveLeaseExpiresAt = leaseExpiresAt
	defer func() {
		if resultErr != nil {
			now := common.GetTimestamp()
			nextAttemptAt := nextVideoArchiveAttemptAt(object, now)
			if err := model.MarkVideoStorageArchiveFailed(
				object.ID,
				object.ArchiveAttempts,
				object.ArchiveOperationID,
				object.ArchiveLeaseExpiresAt,
				object.ArchiveMaxAttempts,
				object.ArchiveRetryDeadlineAt,
				nextAttemptAt,
				common.LocalLogPreview(resultErr.Error()),
			); err != nil && !errors.Is(err, model.ErrVideoStorageArchiveStateChanged) {
				common.SysError("schedule video archive retry error: " + err.Error())
			}
		}
	}()

	ctx, cancel := context.WithTimeout(parent, time.Duration(settings.ArchiveTimeoutSeconds)*time.Second)
	defer cancel()

	staged := service.VideoStagedFile{
		RelativePath: object.StagingRelativePath,
		SizeBytes:    object.StagingSizeBytes,
		MimeType:     object.MimeType,
		Extension:    object.Extension,
		SHA256:       object.StagingSHA256,
	}
	var file *os.File
	if object.StagingStatus == model.StorageStagingVideoAvailable {
		file, err = service.OpenStagedVideo(settings.StagingDirectory, staged)
	}
	if file == nil || err != nil {
		if object.StagingStatus == model.StorageStagingVideoAvailable && err != nil {
			_ = model.MarkVideoStorageStagingFailed(object.ID)
		}
		staged, err = stageVideoArchiveSource(ctx, settings, channel, task, source)
		if err != nil {
			return err
		}
		if err := model.MarkVideoStorageStaged(
			object.ID,
			staged.RelativePath,
			staged.SizeBytes,
			staged.MimeType,
			staged.Extension,
			staged.SHA256,
		); err != nil {
			return err
		}
		file, err = service.OpenStagedVideo(settings.StagingDirectory, staged)
		if err != nil {
			_ = model.MarkVideoStorageStagingFailed(object.ID)
			return err
		}
	}
	defer file.Close()

	if err := model.DB.Model(&model.StorageObject{}).Where("id = ?", object.ID).Updates(map[string]any{
		"endpoint": settings.Endpoint, "region": settings.Region, "bucket": settings.Bucket,
		"object_key": videoObjectKey(settings, task, staged.Extension), "mime_type": staged.MimeType,
		"extension": staged.Extension, "size_bytes": staged.SizeBytes, "updated_at": common.GetTimestamp(),
	}).Error; err != nil {
		return err
	}
	object.Endpoint = settings.Endpoint
	object.Region = settings.Region
	object.Bucket = settings.Bucket
	object.ObjectKey = videoObjectKey(settings, task, staged.Extension)
	object.MimeType = staged.MimeType
	object.Extension = staged.Extension
	object.SizeBytes = staged.SizeBytes

	storage, err := objectstorage.NewStorage(ctx, objectstorage.Config{
		Endpoint: object.Endpoint, Region: object.Region,
		AccessKey: settings.AccessKey, SecretAccessKey: settings.SecretAccessKey,
	})
	if err != nil {
		return err
	}
	head, headErr := storage.HeadObject(ctx, objectstorage.HeadObjectInput{Bucket: object.Bucket, Key: object.ObjectKey})
	if headErr == nil && videoObjectHeadMatches(head, object, staged.SHA256) {
		uploadedAt := head.LastModified.Unix()
		if uploadedAt <= 0 {
			uploadedAt = common.GetTimestamp()
		}
		expiresAt := uploadedAt + settings.RetentionSeconds
		if expiresAt > common.GetTimestamp() {
			if err := model.MarkVideoStorageArchiveAvailable(
				object.ID,
				object.ArchiveOperationID,
				object.ArchiveLeaseExpiresAt,
				head.ETag,
				uploadedAt,
				expiresAt,
			); err != nil {
				return err
			}
			deleteVideoStagedFile(settings, object.ID, staged.RelativePath)
			return nil
		}
	}
	put, err := storage.PutObject(ctx, objectstorage.PutObjectInput{
		Bucket: object.Bucket, Key: object.ObjectKey, Body: file, ContentType: object.MimeType,
		Metadata: map[string]string{
			"sha256": staged.SHA256, "business-id": object.BusinessID,
			"resource-id": task.TaskID, "object-index": "0", "user-id": strconv.Itoa(task.UserId),
		},
	})
	if err != nil {
		return err
	}
	uploadedAt := common.GetTimestamp()
	if err := model.MarkVideoStorageArchiveAvailable(
		object.ID,
		object.ArchiveOperationID,
		object.ArchiveLeaseExpiresAt,
		put.ETag,
		uploadedAt,
		uploadedAt+settings.RetentionSeconds,
	); err != nil {
		return err
	}
	deleteVideoStagedFile(settings, object.ID, staged.RelativePath)
	return nil
}

func nextVideoArchiveAttemptAt(object *model.StorageObject, now int64) int64 {
	attempts := object.ArchiveAttempts + 1
	if attempts >= object.ArchiveMaxAttempts || now >= object.ArchiveRetryDeadlineAt {
		return 0
	}
	delay := int64(15)
	for i := 1; i < attempts && delay < 900; i++ {
		delay *= 2
		if delay > 900 {
			delay = 900
		}
	}
	if now+delay >= object.ArchiveRetryDeadlineAt {
		return 0
	}
	return now + delay
}

func deleteVideoStagedFile(settings storage_setting.VideoSettings, objectID int64, relativePath string) {
	if err := model.MarkVideoStorageStagingDeletePending(objectID); err != nil {
		common.SysError("mark video staging delete pending error: " + err.Error())
		return
	}
	if err := service.DeleteStagedVideo(settings.StagingDirectory, relativePath); err != nil {
		common.SysError("delete staged video error: " + err.Error())
		return
	}
	if err := model.MarkVideoStorageStagingDeleted(objectID); err != nil {
		common.SysError("mark video staging deleted error: " + err.Error())
	}
}

func resolveVideoArchiveSource(ctx context.Context, channel *model.Channel, task *model.Task, source service.VideoArchiveSource) (string, string, error) {
	candidate := strings.TrimSpace(source.URL)
	if candidate == "" {
		candidate = strings.TrimSpace(source.RemoteURL)
	}
	if candidate != "" && !isTaskProxyContentURL(candidate, task.TaskID) {
		if channel.Type == constant.ChannelTypeGemini {
			key := strings.TrimSpace(task.PrivateData.Key)
			if key == "" {
				key = strings.TrimSpace(channel.Key)
			}
			return ensureAPIKey(candidate, key), key, nil
		}
		return candidate, "", nil
	}

	switch channel.Type {
	case constant.ChannelTypeGemini:
		key := strings.TrimSpace(task.PrivateData.Key)
		if key == "" {
			key = strings.TrimSpace(channel.Key)
		}
		resolved, err := getGeminiVideoURL(channel, task, key)
		return resolved, key, err
	case constant.ChannelTypeVertexAi:
		resolved, err := getVertexVideoURL(channel, task)
		return resolved, "", err
	case constant.ChannelTypeOpenAI, constant.ChannelTypeSora:
		key := strings.TrimSpace(task.PrivateData.Key)
		if key == "" {
			key = strings.TrimSpace(channel.Key)
		}
		baseURL := channel.GetBaseURL()
		if baseURL == "" {
			baseURL = "https://api.openai.com"
		}
		client := service.GetSSRFProtectedHTTPClient()
		proxy := channel.GetSetting().Proxy
		if proxy != "" {
			var err error
			client, err = service.GetHttpClientWithProxy(proxy)
			if err != nil {
				return "", "", err
			}
		}
		resolved, err := fetchOpenAIVideoTaskURLContext(ctx, client, baseURL, task.GetUpstreamTaskID(), key, proxy, task.GetVideoProtocol())
		return resolved, "", err
	default:
		candidate = strings.TrimSpace(task.GetResultURL())
		if candidate == "" || isTaskProxyContentURL(candidate, task.TaskID) {
			return "", "", errors.New("video source URL is unavailable")
		}
		return candidate, "", nil
	}
}

func stageVideoArchiveSource(
	ctx context.Context,
	settings storage_setting.VideoSettings,
	channel *model.Channel,
	task *model.Task,
	source service.VideoArchiveSource,
) (service.VideoStagedFile, error) {
	sourceURL, apiKey, err := resolveVideoArchiveSource(ctx, channel, task, source)
	if err != nil {
		return service.VideoStagedFile{}, err
	}
	var reader io.ReadCloser
	mimeType := ""
	if strings.HasPrefix(sourceURL, "data:") {
		parts := strings.SplitN(sourceURL, ",", 2)
		if len(parts) != 2 || !strings.Contains(parts[0], ";base64") {
			return service.VideoStagedFile{}, errors.New("invalid video data URL")
		}
		mimeType = strings.TrimSuffix(strings.TrimPrefix(parts[0], "data:"), ";base64")
		reader = io.NopCloser(base64.NewDecoder(base64.StdEncoding, strings.NewReader(parts[1])))
	} else {
		proxy := channel.GetSetting().Proxy
		if err := validateVideoFetchURL(sourceURL, proxy); err != nil {
			return service.VideoStagedFile{}, err
		}
		client := service.GetVideoContentHTTPClient()
		if proxy != "" {
			var err error
			client, err = service.GetHttpClientWithProxy(proxy)
			if err != nil {
				return service.VideoStagedFile{}, err
			}
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
		if err != nil {
			return service.VideoStagedFile{}, err
		}
		if apiKey != "" {
			request.Header.Set("x-goog-api-key", apiKey)
		}
		response, err := client.Do(request)
		if err != nil {
			return service.VideoStagedFile{}, err
		}
		if response.StatusCode != http.StatusOK {
			_ = response.Body.Close()
			return service.VideoStagedFile{}, fmt.Errorf("video source returned status %d", response.StatusCode)
		}
		reader = response.Body
		mimeType, _, _ = mime.ParseMediaType(response.Header.Get("Content-Type"))
	}
	defer reader.Close()
	return service.StageVideoFile(
		settings.StagingDirectory,
		videoStagingRelativePath(task),
		sourceURL,
		mimeType,
		reader,
	)
}

func videoObjectHeadMatches(head objectstorage.HeadObjectResult, object *model.StorageObject, checksum string) bool {
	if !head.Exists || head.ContentLength != object.SizeBytes || head.LastModified.IsZero() {
		return false
	}
	metadata := make(map[string]string, len(head.Metadata))
	for key, value := range head.Metadata {
		metadata[strings.ToLower(key)] = value
	}
	return metadata["sha256"] == checksum && metadata["business-id"] == object.BusinessID &&
		metadata["resource-id"] == object.ResourceID && metadata["object-index"] == "0"
}

type videoStorageRetryPayload struct {
	TaskID string `json:"task_id"`
}

var errVideoStorageDeletePending = errors.New("video object is pending deletion")

func videoStorageRetryActiveKey(taskID string) string {
	digest := common.Sha256Raw([]byte(taskID))
	return model.SystemTaskTypeVideoStorageRetry + ":" + fmt.Sprintf("%x", digest[:20])
}

type videoStorageRetryHandler struct{}

func (videoStorageRetryHandler) Type() string { return model.SystemTaskTypeVideoStorageRetry }

func (videoStorageRetryHandler) Enabled() bool {
	settings := storage_setting.GetVideoSettings()
	if !settings.Configured() {
		return false
	}
	now := common.GetTimestamp()
	legacyStaleBefore := now - settings.ArchiveTimeoutSeconds
	return model.HasDueVideoStorageArchiveRetries(settings.BusinessID, now) ||
		model.HasExpiredVideoStorageArchiveUploads(settings.BusinessID, now, legacyStaleBefore)
}

func (videoStorageRetryHandler) Interval() time.Duration { return 15 * time.Second }

func (videoStorageRetryHandler) NewPayload() any { return nil }

func (videoStorageRetryHandler) Run(ctx context.Context, systemTask *model.SystemTask, runnerID string) {
	payload := videoStorageRetryPayload{}
	if err := systemTask.DecodePayload(&payload); err != nil {
		finishSystemTaskHandler(systemTask, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	if payload.TaskID == "" {
		runScheduledVideoStorageRetries(ctx, systemTask, runnerID)
		return
	}
	if err := retryVideoStorageArchive(ctx, payload.TaskID); err != nil {
		finishSystemTaskHandler(systemTask, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	finishSystemTaskHandler(systemTask, runnerID, model.SystemTaskStatusSucceeded, map[string]string{
		"task_id": payload.TaskID,
	}, nil)
}

func runScheduledVideoStorageRetries(ctx context.Context, systemTask *model.SystemTask, runnerID string) {
	settings := storage_setting.GetVideoSettings()
	now := common.GetTimestamp()
	recovered, err := recoverExpiredVideoStorageUploads(ctx, settings, now, 10)
	if err != nil {
		finishSystemTaskHandler(systemTask, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	objects, err := model.ListDueVideoStorageArchiveRetries(settings.BusinessID, now, 10)
	if err != nil {
		finishSystemTaskHandler(systemTask, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	succeeded := 0
	failed := 0
	for _, object := range objects {
		if ctx.Err() != nil {
			break
		}
		if err := retryVideoStorageArchive(ctx, object.ResourceID); err != nil {
			failed++
			continue
		}
		succeeded++
	}
	finishSystemTaskHandler(systemTask, runnerID, model.SystemTaskStatusSucceeded, map[string]int{
		"recovered": recovered, "claimed": len(objects), "succeeded": succeeded, "failed": failed,
	}, ctx.Err())
}

func recoverExpiredVideoStorageUploads(
	ctx context.Context,
	settings storage_setting.VideoSettings,
	now int64,
	limit int,
) (int, error) {
	legacyStaleBefore := now - settings.ArchiveTimeoutSeconds
	objects, err := model.ListExpiredVideoStorageArchiveUploads(settings.BusinessID, now, legacyStaleBefore, limit)
	if err != nil {
		return 0, err
	}
	recovered := 0
	for i := range objects {
		object := &objects[i]
		nextAttemptAt := nextVideoArchiveAttemptAt(object, now)
		err := model.MarkVideoStorageArchiveFailed(
			object.ID,
			object.ArchiveAttempts,
			object.ArchiveOperationID,
			object.ArchiveLeaseExpiresAt,
			object.ArchiveMaxAttempts,
			object.ArchiveRetryDeadlineAt,
			nextAttemptAt,
			"video archive upload timed out",
		)
		if errors.Is(err, model.ErrVideoStorageArchiveStateChanged) {
			continue
		}
		if err != nil {
			return recovered, err
		}
		logger.LogWarn(ctx, fmt.Sprintf("Recovered expired video archive upload for task %s", object.ResourceID))
		recovered++
	}
	return recovered, nil
}

func retryVideoStorageArchive(ctx context.Context, taskID string) error {
	settings := storage_setting.GetVideoSettings()
	object, _ := model.GetStorageObjectByBusinessID(settings.BusinessID, taskID, 0)
	stopRetries := func(runErr error) error {
		if object != nil {
			_ = model.StopVideoStorageArchiveRetries(object.ID, common.LocalLogPreview(runErr.Error()))
		}
		return runErr
	}

	task, exists, err := model.GetByOnlyTaskId(taskID)
	if err == nil && (!exists || task == nil) {
		err = errors.New("video task not found")
	}
	if err == nil && task.Status != model.TaskStatusSuccess {
		err = errors.New("only successful video tasks can be uploaded")
	}
	if err == nil && !model.IsVideoTaskAction(task.Action) {
		err = errors.New("only video tasks can be uploaded")
	}
	if err == nil {
		err = settings.Validate()
	}
	if err != nil {
		return stopRetries(err)
	}
	channel, err := model.CacheGetChannel(task.ChannelId)
	if err != nil {
		return stopRetries(err)
	}
	return ArchiveVideoTask(ctx, task, channel, service.VideoArchiveSource{})
}

func enqueueVideoStorageUpload(task *model.Task, settings storage_setting.VideoSettings) (*model.SystemTask, bool, error) {
	object, err := ensureVideoStorageObject(settings, task)
	if err != nil {
		return nil, false, err
	}
	if object.Status == model.StorageObjectStatusDeletePending {
		return nil, false, errVideoStorageDeletePending
	}

	activeKey := videoStorageRetryActiveKey(task.TaskID)
	activeTask, err := model.GetActiveSystemTaskByKey(activeKey)
	if err != nil {
		return nil, false, err
	}
	if activeTask != nil {
		return activeTask, false, nil
	}
	if err := model.ResetVideoStorageArchiveForRetry(
		object.ID,
		settings.ArchiveMaxAttempts,
		common.GetTimestamp()+settings.ArchiveRetryWindowSeconds,
	); err != nil {
		return nil, false, err
	}
	systemTask, created, err := service.EnqueueSystemTaskWithKey(
		model.SystemTaskTypeVideoStorageRetry,
		activeKey,
		videoStorageRetryPayload{TaskID: task.TaskID},
	)
	if err != nil {
		_ = model.MarkStorageObjectFailed(object.ID, common.LocalLogPreview(err.Error()))
		return nil, false, err
	}
	return systemTask, created, nil
}

func RetryVideoStorageUpload(c *gin.Context) {
	taskID := strings.TrimSpace(c.Param("task_id"))
	task, exists, err := model.GetByOnlyTaskId(taskID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !exists || task == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "video task not found"})
		return
	}
	if task.Status != model.TaskStatusSuccess || !model.IsVideoTaskAction(task.Action) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "only successful video tasks can be uploaded"})
		return
	}
	settings := storage_setting.GetVideoSettings()
	if err := settings.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "video object storage is not configured: " + err.Error()})
		return
	}
	systemTask, created, err := enqueueVideoStorageUpload(task, settings)
	if err != nil {
		if errors.Is(err, errVideoStorageDeletePending) {
			c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
			return
		}
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    gin.H{"operation_id": systemTask.TaskID, "created": created},
	})
}

const maxBatchVideoStorageUploads = 100

type batchVideoStorageUploadRequest struct {
	TaskIDs []string `json:"task_ids"`
}

type batchVideoStorageUploadAccepted struct {
	TaskID      string `json:"task_id"`
	OperationID string `json:"operation_id"`
	Created     bool   `json:"created"`
}

type batchVideoStorageUploadSkipped struct {
	TaskID string `json:"task_id"`
	Reason string `json:"reason"`
}

func BatchVideoStorageUpload(c *gin.Context) {
	request := batchVideoStorageUploadRequest{}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request body"})
		return
	}
	if len(request.TaskIDs) == 0 || len(request.TaskIDs) > maxBatchVideoStorageUploads {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": fmt.Sprintf("task_ids must contain between 1 and %d items", maxBatchVideoStorageUploads),
		})
		return
	}

	taskIDs := make([]string, 0, len(request.TaskIDs))
	seen := make(map[string]struct{}, len(request.TaskIDs))
	for _, rawTaskID := range request.TaskIDs {
		taskID := strings.TrimSpace(rawTaskID)
		if taskID == "" {
			continue
		}
		if _, exists := seen[taskID]; exists {
			continue
		}
		seen[taskID] = struct{}{}
		taskIDs = append(taskIDs, taskID)
	}
	if len(taskIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "task_ids must contain at least one non-empty task ID"})
		return
	}

	settings := storage_setting.GetVideoSettings()
	if err := settings.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "video object storage is not configured: " + err.Error()})
		return
	}
	tasks, err := model.GetByOnlyTaskIds(taskIDs)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	tasksByID := make(map[string]*model.Task, len(tasks))
	for _, task := range tasks {
		tasksByID[task.TaskID] = task
	}

	accepted := make([]batchVideoStorageUploadAccepted, 0, len(taskIDs))
	skipped := make([]batchVideoStorageUploadSkipped, 0)
	for _, taskID := range taskIDs {
		task := tasksByID[taskID]
		if task == nil {
			skipped = append(skipped, batchVideoStorageUploadSkipped{TaskID: taskID, Reason: "video task not found"})
			continue
		}
		if task.Status != model.TaskStatusSuccess || !model.IsVideoTaskAction(task.Action) {
			skipped = append(skipped, batchVideoStorageUploadSkipped{TaskID: taskID, Reason: "only successful video tasks can be uploaded"})
			continue
		}
		systemTask, created, enqueueErr := enqueueVideoStorageUpload(task, settings)
		if enqueueErr != nil {
			skipped = append(skipped, batchVideoStorageUploadSkipped{TaskID: taskID, Reason: enqueueErr.Error()})
			continue
		}
		accepted = append(accepted, batchVideoStorageUploadAccepted{
			TaskID: taskID, OperationID: systemTask.TaskID, Created: created,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"accepted": accepted,
			"skipped":  skipped,
		},
	})
}

type videoStorageCleanupHandler struct{}

func (videoStorageCleanupHandler) Type() string { return model.SystemTaskTypeVideoStorageCleanup }
func (videoStorageCleanupHandler) Enabled() bool {
	return storage_setting.GetVideoSettings().Configured()
}
func (videoStorageCleanupHandler) Interval() time.Duration {
	return time.Duration(storage_setting.GetVideoSettings().CleanupIntervalSeconds) * time.Second
}
func (videoStorageCleanupHandler) NewPayload() any { return nil }

func (videoStorageCleanupHandler) Run(ctx context.Context, systemTask *model.SystemTask, runnerID string) {
	settings := storage_setting.GetVideoSettings()
	objects, err := model.ClaimExpiredStorageObjectsByBusinessID(settings.BusinessID, 100)
	if err != nil {
		finishSystemTaskHandler(systemTask, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	deleted, failed := 0, 0
	for _, object := range objects {
		if ctx.Err() != nil {
			break
		}
		storage, storageErr := objectstorage.NewStorage(ctx, objectstorage.Config{
			Endpoint: object.Endpoint, Region: object.Region,
			AccessKey: settings.AccessKey, SecretAccessKey: settings.SecretAccessKey,
		})
		if storageErr == nil {
			storageErr = storage.DeleteObject(ctx, objectstorage.DeleteObjectInput{Bucket: object.Bucket, Key: object.ObjectKey})
		}
		if storageErr != nil {
			failed++
			_ = model.MarkStorageObjectDeleteFailed(object.ID, common.LocalLogPreview(storageErr.Error()))
			continue
		}
		deleted++
		_ = model.MarkStorageObjectDeleted(object.ID)
	}
	stagingDeleted := 0
	pendingStaging, _ := model.ListVideoStorageStagingCleanupCandidates(settings.BusinessID, 100)
	for _, object := range pendingStaging {
		if object.StagingStatus == model.StorageStagingVideoAvailable {
			if err := model.MarkVideoStorageStagingDeletePending(object.ID); err != nil {
				continue
			}
		}
		if err := service.DeleteStagedVideo(settings.StagingDirectory, object.StagingRelativePath); err != nil {
			continue
		}
		if err := model.MarkVideoStorageStagingDeleted(object.ID); err == nil {
			stagingDeleted++
		}
	}
	orphanDeleted, orphanErr := service.CleanupOrphanedVideoStaging(
		ctx, settings.StagingDirectory, settings.BusinessID, time.Hour, 100,
	)
	result := gin.H{
		"claimed": len(objects), "deleted": deleted, "failed": failed,
		"staging_deleted": stagingDeleted, "orphan_staging_deleted": orphanDeleted,
	}
	if orphanErr != nil {
		result["orphan_staging_error"] = common.LocalLogPreview(orphanErr.Error())
	}
	finishSystemTaskHandler(systemTask, runnerID, model.SystemTaskStatusSucceeded, result, ctx.Err())
}
