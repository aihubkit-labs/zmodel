package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/objectstorage"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/storage_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

const (
	asyncImageWorkerConcurrency = 4
	asyncImageLeaseDuration     = 5 * time.Minute
	asyncImageLeaseRenewal      = time.Minute
	asyncImageProcessBatchSize  = 16
)

type asyncImageProcessHandler struct{}

func (asyncImageProcessHandler) Type() string            { return model.SystemTaskTypeAsyncImageProcess }
func (asyncImageProcessHandler) Enabled() bool           { return model.HasRunnableAsyncImageTasks() }
func (asyncImageProcessHandler) Interval() time.Duration { return 15 * time.Second }
func (asyncImageProcessHandler) NewPayload() any         { return nil }

func (asyncImageProcessHandler) Run(ctx context.Context, systemTask *model.SystemTask, runnerID string) {
	owner := runnerID + ":" + systemTask.TaskID
	claimed, err := model.ClaimRunnableAsyncImageTasks(owner, asyncImageProcessBatchSize, asyncImageLeaseDuration)
	if err != nil {
		finishSystemTaskHandler(systemTask, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}

	semaphore := make(chan struct{}, asyncImageWorkerConcurrency)
	var waitGroup sync.WaitGroup
	var processed atomic.Int64
	var failed atomic.Int64
	for index := range claimed {
		if ctx.Err() != nil {
			break
		}
		semaphore <- struct{}{}
		waitGroup.Add(1)
		task := claimed[index]
		go func() {
			defer waitGroup.Done()
			defer func() { <-semaphore }()
			if err := processClaimedAsyncImageTaskWithLeaseRenewal(ctx, &task, owner); err != nil {
				failed.Add(1)
				logger.LogWarn(context.Background(), fmt.Sprintf("async image task failed: task=%s err=%s", task.TaskID, common.LocalLogPreview(err.Error())))
			}
			processed.Add(1)
		}()
	}
	waitGroup.Wait()
	sendPendingAsyncImageNotifications()

	if model.HasRunnableAsyncImageTasks() {
		_, _, _ = service.EnqueueSystemTask(model.SystemTaskTypeAsyncImageProcess, nil)
	}
	finishSystemTaskHandler(systemTask, runnerID, model.SystemTaskStatusSucceeded, map[string]int{
		"claimed": len(claimed), "processed": int(processed.Load()), "failed": int(failed.Load()),
	}, nil)
}

func processClaimedAsyncImageTaskWithLeaseRenewal(ctx context.Context, task *model.AsyncImageTask, owner string) error {
	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(asyncImageLeaseRenewal)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := model.RenewAsyncImageTaskLease(task.TaskID, owner, asyncImageLeaseDuration); err != nil {
					logger.LogWarn(context.Background(), fmt.Sprintf("async image lease renewal failed: task=%s err=%s", task.TaskID, common.LocalLogPreview(err.Error())))
				}
			}
		}
	}()
	return processClaimedAsyncImageTask(ctx, task, owner)
}

func processClaimedAsyncImageTask(ctx context.Context, task *model.AsyncImageTask, owner string) error {
	if task.Status == model.AsyncImageStatusSucceeded && task.OutputAvailability == model.AsyncImageOutputArchiving {
		return archiveAsyncImageTask(ctx, task, owner)
	}
	if err := service.CheckAsyncImageStaging(); err != nil {
		return model.ScheduleAsyncImageGenerationRetry(task.TaskID, owner, common.GetTimestamp()+15, "archive staging is unavailable")
	}
	return generateAsyncImageTask(ctx, task, owner)
}

func generateAsyncImageTask(ctx context.Context, task *model.AsyncImageTask, owner string) error {
	if err := model.MarkAsyncImageTaskRunning(task.TaskID, owner); err != nil {
		return err
	}

	request := &dto.ImageRequest{}
	if err := common.UnmarshalJsonStr(task.RequestPayload, request); err != nil {
		return model.RefundAsyncImageBilling(task.TaskID, "invalid_request_error", "stored image request is invalid")
	}
	c, recorder, writer, cleanup, err := newAsyncImageRelayContext(task, request)
	if err != nil {
		return model.RefundAsyncImageBilling(task.TaskID, "image_generation_failed", "image generation could not be started")
	}
	defer cleanup()

	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatOpenAIImage, request, nil)
	if err != nil {
		return model.RefundAsyncImageBilling(task.TaskID, "image_generation_failed", "image generation could not be started")
	}
	var billingContext asyncImageBillingContext
	if err := common.UnmarshalJsonStr(task.BillingContext, &billingContext); err != nil {
		return model.RefundAsyncImageBilling(task.TaskID, "image_generation_failed", "stored billing context is invalid")
	}
	relayInfo.PriceData = billingContext.PriceData
	relayInfo.TieredBillingSnapshot = billingContext.TieredBillingSnapshot
	relayInfo.BillingRequestInput = billingContext.BillingRequestInput
	relayInfo.TokenGroup = billingContext.TokenGroup
	relayInfo.UserGroup = billingContext.UserGroup
	relayInfo.SetEstimatePromptTokens(billingContext.EstimatedTokens)
	relayInfo.FinalPreConsumedQuota = task.ReservedQuota
	relayInfo.BillingSource = task.BillingSource
	relayInfo.SubscriptionId = task.SubscriptionID
	relayInfo.ForcePreConsume = true
	// Force the first call through the same random-channel selection primitive
	// used by retries; the submit-time channel is only a capability check.
	relayInfo.ChannelMeta = &relaycommon.ChannelMeta{}

	retryParam := &service.RetryParam{
		Ctx: c, TokenGroup: relayInfo.TokenGroup, ModelName: relayInfo.OriginModelName,
		RequestPath: "/v1/images/generations", Retry: common.GetPointer(0),
	}
	var execution *relay.ImageExecutionResult
	var finalError *types.NewAPIError
	for ; retryParam.GetRetry() <= common.RetryTimes; retryParam.IncreaseRetry() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		relayInfo.RetryIndex = retryParam.GetRetry()
		channel, channelErr := getChannel(c, relayInfo, retryParam)
		if channelErr != nil {
			finalError = channelErr
			break
		}
		if err := model.UpdateAsyncImageTaskWithLease(task.TaskID, owner, map[string]any{"last_channel_id": channel.Id}); err != nil {
			return err
		}
		task.LastChannelID = channel.Id
		addUsedChannel(c, channel.Id)
		bodyStorage, bodyErr := common.GetBodyStorage(c)
		if bodyErr != nil {
			finalError = types.NewError(bodyErr, types.ErrorCodeReadRequestBodyFailed, types.ErrOptionWithSkipRetry())
			break
		}
		_, _ = bodyStorage.Seek(0, io.SeekStart)
		c.Request.Body = io.NopCloser(bodyStorage)
		recorder.Body.Reset()
		writer.Reset()
		execution, finalError = relay.ExecuteImageAttempt(c, relayInfo)
		if finalError == nil && writer.Err() == nil {
			break
		}
		if writer.Err() != nil {
			finalError = types.NewError(writer.Err(), types.ErrorCodeBadResponse, types.ErrOptionWithSkipRetry())
		}
		relayInfo.LastError = finalError
		processChannelError(c, *types.NewChannelError(channel.Id, channel.Type, channel.Name, channel.ChannelInfo.IsMultiKey, common.GetContextKeyString(c, constant.ContextKeyChannelKey), channel.GetAutoBan()), finalError)
		if !shouldRetry(c, finalError, common.RetryTimes-retryParam.GetRetry()) {
			break
		}
	}
	if finalError != nil || execution == nil {
		code := "image_generation_failed"
		if finalError != nil && finalError.GetErrorCode() != "" {
			code = string(finalError.GetErrorCode())
		}
		return model.RefundAsyncImageBilling(task.TaskID, code, "image generation failed")
	}

	response := &dto.ImageResponse{}
	if err := common.Unmarshal(recorder.Body.Bytes(), response); err != nil || len(response.Data) == 0 || len(response.Data) > dto.MaxImageN {
		return model.RefundAsyncImageBilling(task.TaskID, "invalid_upstream_response", "image provider returned an invalid response")
	}
	requested := 1
	if request.N != nil && *request.N > 0 {
		requested = int(*request.N)
	}
	if len(response.Data) > requested {
		return model.RefundAsyncImageBilling(task.TaskID, "invalid_upstream_response", "image provider returned too many images")
	}

	stageCtx, cancel := context.WithTimeout(context.Background(), time.Duration(task.ArchiveTimeoutSeconds)*time.Second)
	manifest, sourceKind, stageErr := service.StageAsyncImageResponse(stageCtx, task.UserID, task.TaskID, response)
	cancel()
	if stageErr != nil {
		kind := service.AsyncImageStageErrorKindOf(stageErr)
		if kind == service.AsyncImageStageInvalid {
			return model.RefundAsyncImageBilling(task.TaskID, "invalid_upstream_response", "image provider returned invalid image data")
		}
		code := "archive_staging_failed"
		if kind == service.AsyncImageStageSourceFetch {
			code = "archive_source_fetch_failed"
		}
		relayInfo.Billing = &service.AsyncImageBillingSettler{
			TaskID: task.TaskID, PreConsumedQuota: task.ReservedQuota, LeaseOwner: owner,
			SourceKind: sourceKind, ArchiveErrorCode: code, ArchiveError: "generated image could not be archived",
		}
		if err := service.PostTextConsumeQuotaWithError(c, relayInfo, execution.Usage, execution.LogContent); err != nil {
			return err
		}
		return nil
	}

	manifestData, err := common.Marshal(manifest)
	if err != nil {
		return err
	}
	settings := storage_setting.GetSettings()
	objects := make([]model.StorageObject, 0, len(manifest))
	for _, item := range manifest {
		objects = append(objects, model.StorageObject{
			ObjectIndex: item.Index, Endpoint: settings.Endpoint, Region: settings.Region, Bucket: settings.Bucket,
			ObjectKey: asyncImageObjectKey(task.UserID, task.TaskID, item), MimeType: item.MimeType,
			Extension: item.Extension, SizeBytes: item.SizeBytes, StagingRelativePath: item.StagingRelativePath,
			StagingSizeBytes: item.SizeBytes, StagingSHA256: item.SHA256, StagedAt: common.GetTimestamp(),
		})
	}
	relayInfo.Billing = &service.AsyncImageBillingSettler{
		TaskID: task.TaskID, PreConsumedQuota: task.ReservedQuota, LeaseOwner: owner,
		Manifest: string(manifestData), SourceKind: sourceKind, Objects: objects,
	}
	if err := service.PostTextConsumeQuotaWithError(c, relayInfo, execution.Usage, execution.LogContent); err != nil {
		return err
	}
	updated, err := model.GetAsyncImageTaskByTaskID(task.TaskID)
	if err != nil {
		return err
	}
	return archiveAsyncImageTask(ctx, updated, owner)
}

type boundedGinWriter struct {
	gin.ResponseWriter
	maxBytes int64
	written  int64
	err      error
}

func (writer *boundedGinWriter) Write(data []byte) (int, error) {
	if writer.err != nil {
		return 0, writer.err
	}
	if int64(len(data)) > writer.maxBytes-writer.written {
		writer.err = errors.New("image response exceeds capture limit")
		return 0, writer.err
	}
	n, err := writer.ResponseWriter.Write(data)
	writer.written += int64(n)
	if err != nil {
		writer.err = err
	}
	return n, err
}

func (writer *boundedGinWriter) WriteString(value string) (int, error) {
	return writer.Write([]byte(value))
}

func (writer *boundedGinWriter) Reset() {
	writer.written = 0
	writer.err = nil
}

func (writer *boundedGinWriter) Err() error { return writer.err }

func newAsyncImageRelayContext(task *model.AsyncImageTask, imageRequest *dto.ImageRequest) (*gin.Context, *httptest.ResponseRecorder, *boundedGinWriter, func(), error) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	request, err := http.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewBufferString(task.RequestPayload))
	if err != nil {
		return nil, nil, nil, nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	c.Request = request
	c.Set(common.RequestIdKey, task.TaskID)
	common.SetContextKey(c, constant.ContextKeyRequestStartTime, time.Now())
	common.SetContextKey(c, constant.ContextKeyOriginalModel, task.OriginModelName)
	common.SetContextKey(c, constant.ContextKeyUsingGroup, task.UsingGroup)
	c.Set("id", task.UserID)

	user, userErr := model.GetUserCache(task.UserID)
	if userErr == nil && user != nil {
		user.WriteContext(c)
	} else {
		common.SetContextKey(c, constant.ContextKeyUserGroup, task.UsingGroup)
	}
	common.SetContextKey(c, constant.ContextKeyUsingGroup, task.UsingGroup)
	token, tokenErr := model.GetTokenByIds(task.TokenID, task.UserID)
	if tokenErr == nil && token != nil {
		if err := middleware.SetupContextForToken(c, token); err != nil {
			return nil, nil, nil, nil, err
		}
	} else {
		c.Set("token_id", task.TokenID)
		c.Set("token_unlimited_quota", task.TokenUnlimited)
		common.SetContextKey(c, constant.ContextKeyTokenGroup, task.UsingGroup)
	}
	common.SetContextKey(c, constant.ContextKeyUsingGroup, task.UsingGroup)
	common.SetContextKey(c, constant.ContextKeyOriginalModel, imageRequest.Model)

	n := int64(1)
	if imageRequest.N != nil && *imageRequest.N > 0 {
		n = int64(*imageRequest.N)
	}
	perImage := int64(constant.MaxFileDownloadMB) * 1024 * 1024
	maxBytes := n*(perImage+perImage/3+1024) + 1024*1024
	writer := &boundedGinWriter{ResponseWriter: c.Writer, maxBytes: maxBytes}
	c.Writer = writer
	cleanup := func() { common.CleanupBodyStorage(c) }
	return c, recorder, writer, cleanup, nil
}

func asyncImageObjectKey(userID int, taskID string, item service.AsyncImageManifestItem) string {
	parts := strings.Split(filepathSlash(item.StagingRelativePath), "/")
	year, month := time.Now().UTC().Format("2006"), time.Now().UTC().Format("01")
	if len(parts) >= 4 {
		year, month = parts[1], parts[2]
	}
	return fmt.Sprintf("prod/user-files/zmodel@async-images/%d/%s/%s/%s/%d.%s", userID, year, month, taskID, item.Index, item.Extension)
}

func filepathSlash(value string) string { return strings.ReplaceAll(value, "\\", "/") }

func archiveAsyncImageTask(parent context.Context, task *model.AsyncImageTask, owner string) error {
	manifest, err := decodeAsyncImageManifest(task.ArchiveManifest)
	if err != nil || len(manifest) == 0 {
		return failAsyncImageArchive(task, owner, "archive_manifest_invalid", "archive manifest is invalid")
	}
	objects, err := model.ListStorageObjects(task.TaskID)
	if err != nil || len(objects) != len(manifest) {
		return failAsyncImageArchive(task, owner, "archive_manifest_invalid", "archive object records are incomplete")
	}
	settings := storage_setting.GetSettings()
	ctx, cancel := context.WithTimeout(parent, time.Duration(task.ArchiveTimeoutSeconds)*time.Second)
	defer cancel()
	for index := range objects {
		if objects[index].Status == model.StorageObjectStatusDeletePending {
			return retryAsyncImageArchive(task, owner, errors.New("object cleanup is still pending"))
		}
		item := manifest[index]
		if item.Index != objects[index].ObjectIndex {
			return failAsyncImageArchive(task, owner, "archive_manifest_invalid", "archive object records do not match the manifest")
		}
		file, err := service.ReadStagedImage(item)
		if err != nil {
			_ = model.MarkStorageObjectStagingFailed(objects[index].ID, "staged image integrity check failed")
			return failAsyncImageArchive(task, owner, "archive_staging_integrity_failed", "staged image is missing or damaged")
		}
		storage, err := objectstorage.NewStorage(ctx, objectstorage.Config{
			Endpoint: objects[index].Endpoint, Region: objects[index].Region,
			AccessKey: settings.AccessKey, SecretAccessKey: settings.SecretAccessKey,
		})
		if err != nil {
			_ = file.Close()
			return retryAsyncImageArchive(task, owner, err)
		}
		head, headErr := storage.HeadObject(ctx, objectstorage.HeadObjectInput{Bucket: objects[index].Bucket, Key: objects[index].ObjectKey})
		if headErr == nil && asyncImageHeadMatches(head, task.TaskID, item) {
			uploadedAt := head.LastModified.Unix()
			expiresAt := uploadedAt + task.RetentionSeconds
			if uploadedAt > 0 && expiresAt > common.GetTimestamp() {
				_ = file.Close()
				if objects[index].Status != model.StorageObjectStatusAvailable || objects[index].ExpiresAt != expiresAt {
					if err := model.MarkStorageObjectAvailable(objects[index].ID, objects[index].Status, head.ETag, uploadedAt, expiresAt); err != nil {
						return err
					}
				}
				continue
			}
		}
		if err := model.MarkStorageObjectUploading(objects[index].ID); err != nil {
			_ = file.Close()
			return err
		}
		put, err := storage.PutObject(ctx, objectstorage.PutObjectInput{
			Bucket: objects[index].Bucket, Key: objects[index].ObjectKey, Body: file, ContentType: item.MimeType,
			Metadata: map[string]string{
				"sha256": item.SHA256, "business-id": model.StorageObjectBusinessAsyncImages,
				"resource-id": task.TaskID, "object-index": strconv.Itoa(item.Index),
			},
		})
		_ = file.Close()
		if err != nil {
			_ = model.MarkStorageObjectFailed(objects[index].ID, common.LocalLogPreview(err.Error()))
			return retryAsyncImageArchive(task, owner, err)
		}
		uploadedAt := common.GetTimestamp()
		if err := model.MarkStorageObjectAvailable(objects[index].ID, model.StorageObjectStatusUploading, put.ETag, uploadedAt, uploadedAt+task.RetentionSeconds); err != nil {
			return err
		}
	}

	objects, err = model.ListStorageObjects(task.TaskID)
	if err != nil {
		return err
	}
	minExpiresAt := int64(0)
	for _, object := range objects {
		if object.Status != model.StorageObjectStatusAvailable || object.ExpiresAt <= common.GetTimestamp() {
			return retryAsyncImageArchive(task, owner, errors.New("not all archived images are available"))
		}
		if minExpiresAt == 0 || object.ExpiresAt < minExpiresAt {
			minExpiresAt = object.ExpiresAt
		}
	}
	manual := task.CompletedAt > 0
	if err := model.MarkAsyncImageOutputAvailable(task.TaskID, owner, minExpiresAt, manual); err != nil {
		return err
	}
	cleanupAsyncImageStaging(objects)
	return nil
}

func asyncImageHeadMatches(head objectstorage.HeadObjectResult, taskID string, item service.AsyncImageManifestItem) bool {
	if !head.Exists || head.ContentLength != item.SizeBytes || head.ContentType != item.MimeType || head.LastModified.IsZero() {
		return false
	}
	metadata := make(map[string]string, len(head.Metadata))
	for key, value := range head.Metadata {
		metadata[strings.ToLower(key)] = value
	}
	return metadata["sha256"] == item.SHA256 &&
		metadata["business-id"] == model.StorageObjectBusinessAsyncImages &&
		metadata["resource-id"] == taskID && metadata["object-index"] == strconv.Itoa(item.Index)
}

func retryAsyncImageArchive(task *model.AsyncImageTask, owner string, archiveErr error) error {
	attempts := task.ArchiveAttempts + 1
	now := common.GetTimestamp()
	if attempts >= task.ArchiveMaxAttempts || now >= task.ArchiveRetryDeadlineAt {
		return failAsyncImageArchiveWithAttempts(task, owner, attempts, "archive_upload_failed", "generated images could not be uploaded")
	}
	delay := int64(15)
	for i := 1; i < attempts && delay < 900; i++ {
		delay *= 2
		if delay > 900 {
			delay = 900
		}
	}
	return model.ScheduleAsyncImageArchiveRetry(task.TaskID, owner, attempts, now+delay, common.LocalLogPreview(archiveErr.Error()))
}

func failAsyncImageArchive(task *model.AsyncImageTask, owner string, code string, message string) error {
	return failAsyncImageArchiveWithAttempts(task, owner, task.ArchiveAttempts+1, code, message)
}

func failAsyncImageArchiveWithAttempts(task *model.AsyncImageTask, owner string, attempts int, code string, message string) error {
	return model.FailAsyncImageArchive(task.TaskID, owner, attempts, code, message)
}

func cleanupAsyncImageStaging(objects []model.StorageObject) {
	for _, object := range objects {
		if object.StagingStatus != model.StorageStagingAvailable {
			continue
		}
		if err := model.DB.Model(&model.StorageObject{}).
			Where("id = ? AND staging_status = ?", object.ID, model.StorageStagingAvailable).
			Update("staging_status", model.StorageStagingDeletePending).Error; err != nil {
			continue
		}
		if err := service.DeleteStagedImage(object.StagingRelativePath); err != nil {
			continue
		}
		_ = model.DB.Model(&model.StorageObject{}).
			Where("id = ? AND staging_status = ?", object.ID, model.StorageStagingDeletePending).
			Updates(map[string]any{"staging_status": model.StorageStagingDeleted, "staging_deleted_at": common.GetTimestamp(), "updated_at": common.GetTimestamp()}).Error
	}
}

func sendPendingAsyncImageNotifications() {
	tasks, err := model.ClaimPendingAsyncImageNotifications(20)
	if err != nil {
		return
	}
	for _, task := range tasks {
		objects, _ := model.ListStorageObjects(task.TaskID)
		available := 0
		for _, object := range objects {
			if object.Status == model.StorageObjectStatusAvailable {
				available++
			}
		}
		content := fmt.Sprintf("异步图片归档失败：任务 %s，用户 %d，模型 %s，尝试 %d 次，对象 %d/%d 可用。错误：%s",
			task.TaskID, task.UserID, task.OriginModelName, task.ArchiveAttempts, available, len(objects), common.LocalLogPreview(task.LastError))
		service.NotifyRootUser("async_image_archive_failed", "异步图片归档失败", content)
		_ = model.MarkAsyncImageNotificationSent(task.TaskID)
	}
}
