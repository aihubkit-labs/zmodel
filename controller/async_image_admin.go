package controller

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/objectstorage"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/storage_setting"

	"github.com/gin-gonic/gin"
)

const asyncImageRetryBatchLimit = 100

type asyncImageTaskListItem struct {
	TaskID                string `json:"task_id"`
	UserID                int    `json:"user_id"`
	Model                 string `json:"model"`
	Status                string `json:"status"`
	OutputAvailability    string `json:"output_availability"`
	BillingStatus         string `json:"billing_status"`
	ObjectAvailableCount  int    `json:"object_available_count"`
	ObjectTotalCount      int    `json:"object_total_count"`
	ArchiveAttempts       int    `json:"archive_attempts"`
	StagingIntegrity      string `json:"staging_integrity"`
	Error                 string `json:"error,omitempty"`
	CreatedAt             int64  `json:"created_at"`
	GenerationCompletedAt int64  `json:"generation_completed_at"`
	CompletedAt           int64  `json:"completed_at"`
	OutputExpiresAt       int64  `json:"output_expires_at"`
	ManuallyRecoveredAt   int64  `json:"manually_recovered_at"`
	ReservedQuota         int    `json:"reserved_quota,omitempty"`
	ActualQuota           int    `json:"actual_quota,omitempty"`
}

func GetSelfAsyncImageTasks(c *gin.Context) {
	filter := asyncImageTaskFilterFromQuery(c)
	filter.UserID = c.GetInt("id")
	writeAsyncImageTaskList(c, filter, false)
}

func GetAllAsyncImageTasks(c *gin.Context) {
	writeAsyncImageTaskList(c, asyncImageTaskFilterFromQuery(c), true)
}

func asyncImageTaskFilterFromQuery(c *gin.Context) model.AsyncImageTaskFilter {
	return model.AsyncImageTaskFilter{
		UserID:             queryInt(c, "user_id"),
		TaskID:             c.Query("task_id"),
		Model:              c.Query("model"),
		Status:             c.Query("status"),
		OutputAvailability: c.Query("output_availability"),
		BillingStatus:      c.Query("billing_status"),
		CreatedAfter:       queryInt64(c, "created_after"),
		CreatedBefore:      queryInt64(c, "created_before"),
		Page:               queryInt(c, "page"),
		PageSize:           queryInt(c, "page_size"),
	}
}

func writeAsyncImageTaskList(c *gin.Context, filter model.AsyncImageTaskFilter, root bool) {
	tasks, total, err := model.ListAsyncImageTasks(filter)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items := make([]asyncImageTaskListItem, 0, len(tasks))
	for index := range tasks {
		objects, objectErr := model.ListStorageObjects(tasks[index].TaskID)
		available := 0
		integrity := "not_staged"
		if objectErr == nil && len(objects) > 0 {
			integrity = "available"
			for _, object := range objects {
				if object.Status == model.StorageObjectStatusAvailable {
					available++
				}
				if object.StagingStatus == model.StorageStagingFailed {
					integrity = "failed"
				}
			}
		}
		item := asyncImageTaskListItem{
			TaskID: tasks[index].TaskID, UserID: tasks[index].UserID, Model: tasks[index].OriginModelName,
			Status: tasks[index].Status, OutputAvailability: tasks[index].OutputAvailability,
			BillingStatus: tasks[index].BillingStatus, ObjectAvailableCount: available, ObjectTotalCount: len(objects),
			ArchiveAttempts: tasks[index].ArchiveAttempts, StagingIntegrity: integrity,
			CreatedAt: tasks[index].CreatedAt, GenerationCompletedAt: tasks[index].GenerationCompletedAt,
			CompletedAt: tasks[index].CompletedAt, OutputExpiresAt: tasks[index].OutputExpiresAt,
			ManuallyRecoveredAt: tasks[index].ManuallyRecoveredAt,
		}
		if root {
			item.Error = tasks[index].LastError
			item.ReservedQuota = tasks[index].ReservedQuota
			item.ActualQuota = tasks[index].ActualQuota
		} else if tasks[index].Status == model.AsyncImageStatusFailed || tasks[index].OutputAvailability == model.AsyncImageOutputFailed {
			item.Error = tasks[index].PublicErrorMessage
		}
		items = append(items, item)
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total, "page": filter.Page, "page_size": filter.PageSize})
}

type retryAsyncImageTasksRequest struct {
	TaskIDs []string `json:"task_ids"`
}

func RetryAsyncImageTasks(c *gin.Context) {
	var request retryAsyncImageTasksRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "invalid retry request")
		return
	}
	if len(request.TaskIDs) == 0 || len(request.TaskIDs) > asyncImageRetryBatchLimit {
		common.ApiErrorMsg(c, fmt.Sprintf("task_ids must contain between 1 and %d items", asyncImageRetryBatchLimit))
		return
	}
	seen := make(map[string]struct{}, len(request.TaskIDs))
	accepted, skipped, integrityErrors := 0, 0, 0
	for _, taskID := range request.TaskIDs {
		if _, exists := seen[taskID]; exists {
			continue
		}
		seen[taskID] = struct{}{}
		ok, integrityErr, err := validateAndResetAsyncImageArchive(taskID)
		if err != nil {
			skipped++
			continue
		}
		if integrityErr {
			integrityErrors++
			skipped++
			continue
		}
		if ok {
			accepted++
		} else {
			skipped++
		}
	}
	if accepted > 0 {
		_, _, _ = service.EnqueueSystemTask(model.SystemTaskTypeAsyncImageProcess, nil)
	}
	common.ApiSuccess(c, gin.H{"accepted_count": accepted, "skipped_count": skipped, "integrity_error_count": integrityErrors})
}

func RetryAllFailedAsyncImageTasks(c *gin.Context) {
	active, err := model.GetActiveSystemTask(model.SystemTaskTypeAsyncImageBulkRetry)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if active != nil {
		common.ApiSuccess(c, gin.H{"operation_id": active.TaskID})
		return
	}
	maxID, err := model.MaxAsyncImageTaskID()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	task, _, err := service.EnqueueSystemTask(model.SystemTaskTypeAsyncImageBulkRetry, asyncImageBulkRetryPayload{MaxID: maxID})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"operation_id": task.TaskID})
}

func validateAndResetAsyncImageArchive(taskID string) (bool, bool, error) {
	task, err := model.GetAsyncImageTaskByTaskID(taskID)
	if err != nil {
		return false, false, err
	}
	if task.Status != model.AsyncImageStatusSucceeded || task.BillingStatus != model.AsyncImageBillingSettled || task.OutputAvailability != model.AsyncImageOutputFailed {
		return false, false, nil
	}
	manifest, err := decodeAsyncImageManifest(task.ArchiveManifest)
	if err != nil || len(manifest) == 0 {
		return false, true, errors.New("archive manifest is unavailable")
	}
	objects, err := model.ListStorageObjects(taskID)
	if err != nil || len(objects) != len(manifest) {
		return false, true, errors.New("archive object records are incomplete")
	}
	for index, item := range manifest {
		if objects[index].ObjectIndex != item.Index {
			return false, true, errors.New("archive object records do not match the manifest")
		}
		file, err := service.ReadStagedImage(item)
		if err != nil {
			_ = model.MarkStorageObjectStagingFailed(objects[index].ID, "staged image integrity check failed")
			_ = model.MarkAsyncImageStagingIntegrityIncident(taskID, "staged image integrity check failed")
			return false, true, err
		}
		_ = file.Close()
		if objects[index].StagingStatus == model.StorageStagingFailed {
			_ = model.MarkStorageObjectStagingAvailable(objects[index].ID)
		}
	}
	settings := storage_setting.GetSettings()
	accepted, err := model.ResetAsyncImageArchiveForRetry(taskID, common.GetTimestamp()+settings.ArchiveRetryWindowSeconds)
	return accepted, false, err
}

type asyncImageBulkRetryPayload struct {
	MaxID int64 `json:"max_id"`
}

type asyncImageBulkRetryState struct {
	CursorID            int64 `json:"cursor_id"`
	AcceptedCount       int   `json:"accepted_count"`
	SkippedCount        int   `json:"skipped_count"`
	IntegrityErrorCount int   `json:"integrity_error_count"`
	Progress            int   `json:"progress"`
}

type asyncImageBulkRetryHandler struct{}

func (asyncImageBulkRetryHandler) Type() string { return model.SystemTaskTypeAsyncImageBulkRetry }

func (asyncImageBulkRetryHandler) Run(ctx context.Context, systemTask *model.SystemTask, runnerID string) {
	payload := asyncImageBulkRetryPayload{}
	state := asyncImageBulkRetryState{}
	if err := systemTask.DecodePayload(&payload); err != nil {
		finishSystemTaskHandler(systemTask, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	_ = systemTask.DecodeState(&state)
	for state.CursorID < payload.MaxID && ctx.Err() == nil {
		tasks, err := model.ListFailedAsyncImageTaskIDs(state.CursorID, payload.MaxID, 100)
		if err != nil {
			finishSystemTaskHandler(systemTask, runnerID, model.SystemTaskStatusFailed, nil, err)
			return
		}
		if len(tasks) == 0 {
			break
		}
		for _, task := range tasks {
			accepted, integrityErr, _ := validateAndResetAsyncImageArchive(task.TaskID)
			if accepted {
				state.AcceptedCount++
			} else {
				state.SkippedCount++
			}
			if integrityErr {
				state.IntegrityErrorCount++
			}
			state.CursorID = task.ID
		}
		if payload.MaxID > 0 {
			state.Progress = int(state.CursorID * 100 / payload.MaxID)
		}
		_ = model.UpdateSystemTaskState(systemTask.TaskID, runnerID, state)
	}
	state.Progress = 100
	_ = model.UpdateSystemTaskState(systemTask.TaskID, runnerID, state)
	if state.AcceptedCount > 0 {
		_, _, _ = service.EnqueueSystemTask(model.SystemTaskTypeAsyncImageProcess, nil)
	}
	finishSystemTaskHandler(systemTask, runnerID, model.SystemTaskStatusSucceeded, state, nil)
}

type asyncImageCleanupHandler struct{}

func (asyncImageCleanupHandler) Type() string  { return model.SystemTaskTypeAsyncImageCleanup }
func (asyncImageCleanupHandler) Enabled() bool { return storage_setting.GetSettings().Configured() }
func (asyncImageCleanupHandler) Interval() time.Duration {
	return time.Duration(storage_setting.GetSettings().CleanupIntervalSeconds) * time.Second
}
func (asyncImageCleanupHandler) NewPayload() any { return nil }

func (asyncImageCleanupHandler) Run(ctx context.Context, systemTask *model.SystemTask, runnerID string) {
	objects, err := model.ClaimExpiredStorageObjects(100)
	if err != nil {
		finishSystemTaskHandler(systemTask, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	settings := storage_setting.GetSettings()
	deleted, failed := 0, 0
	for _, object := range objects {
		if ctx.Err() != nil {
			break
		}
		storage, err := objectstorage.NewStorage(ctx, objectstorage.Config{
			Endpoint: object.Endpoint, Region: object.Region,
			AccessKey: settings.AccessKey, SecretAccessKey: settings.SecretAccessKey,
		})
		if err == nil {
			err = storage.DeleteObject(ctx, objectstorage.DeleteObjectInput{Bucket: object.Bucket, Key: object.ObjectKey})
		}
		if err != nil {
			failed++
			_ = model.MarkStorageObjectDeleteFailed(object.ID, common.LocalLogPreview(err.Error()))
			continue
		}
		deleted++
		_ = model.MarkStorageObjectDeleted(object.ID)
		_ = model.MarkExpiredAsyncImageTask(object.ResourceID, common.GetTimestamp())
	}
	stagingDeleted := 0
	pendingStaging, _ := model.ListStagingDeletePending(100)
	for _, object := range pendingStaging {
		if err := service.DeleteStagedImage(object.StagingRelativePath); err != nil {
			continue
		}
		if err := model.MarkStagingObjectDeleted(object.ID); err == nil {
			stagingDeleted++
		}
	}
	orphanDeleted, orphanErr := service.CleanupOrphanedAsyncImageStaging(ctx, time.Hour, 100)
	result := gin.H{"deleted": deleted, "failed": failed, "staging_deleted": stagingDeleted, "orphan_staging_deleted": orphanDeleted}
	if orphanErr != nil {
		result["orphan_staging_error"] = common.LocalLogPreview(orphanErr.Error())
	}
	finishSystemTaskHandler(systemTask, runnerID, model.SystemTaskStatusSucceeded, result, nil)
}

func queryInt(c *gin.Context, key string) int {
	value, _ := strconv.Atoi(c.Query(key))
	return value
}

func queryInt64(c *gin.Context, key string) int64 {
	value, _ := strconv.ParseInt(c.Query(key), 10, 64)
	return value
}
