package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedAsyncImageWalletTask(t *testing.T, taskID string, status string, owner string, reservedQuota int, tokenReservedQuota int) {
	t.Helper()
	require.NoError(t, DB.Create(&User{
		Id:       9101,
		Username: "async-image-user",
		Quota:    1000,
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, DB.Create(&Token{
		Id:          9201,
		UserId:      9101,
		Key:         "async-image-token",
		Name:        "async image token",
		Status:      common.TokenStatusEnabled,
		RemainQuota: 1000,
	}).Error)
	require.NoError(t, CreateAsyncImageTaskWithReservation(&AsyncImageTask{
		TaskID:             taskID,
		UserID:             9101,
		TokenID:            9201,
		Status:             status,
		OutputAvailability: AsyncImageOutputPending,
		BillingStatus:      AsyncImageBillingReserved,
		ReservedQuota:      reservedQuota,
		TokenReservedQuota: tokenReservedQuota,
		OriginModelName:    "test-image-model",
		UsingGroup:         "default",
		LeaseOwner:         owner,
		RequestBody:        []byte{'i', 'm', 'a', 'g', 'e', 0, 1},
	}, "wallet_only"))
}

func asyncImageWalletAndTokenQuota(t *testing.T) (int, int, int) {
	t.Helper()
	var user User
	var token Token
	require.NoError(t, DB.First(&user, 9101).Error)
	require.NoError(t, DB.First(&token, 9201).Error)
	return user.Quota, token.RemainQuota, token.UsedQuota
}

func TestCreateAsyncImageTaskWithReservationRollsBackAllFunding(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Create(&User{
		Id:       9101,
		Username: "async-image-user",
		Quota:    1000,
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, DB.Create(&Token{
		Id:          9201,
		UserId:      9101,
		Key:         "async-image-token",
		Name:        "async image token",
		Status:      common.TokenStatusEnabled,
		RemainQuota: 10,
	}).Error)

	err := CreateAsyncImageTaskWithReservation(&AsyncImageTask{
		TaskID:             "task_async_reservation_rollback",
		UserID:             9101,
		TokenID:            9201,
		Status:             AsyncImageStatusQueued,
		OutputAvailability: AsyncImageOutputPending,
		BillingStatus:      AsyncImageBillingReserved,
		ReservedQuota:      200,
		TokenReservedQuota: 200,
	}, "wallet_only")
	require.ErrorIs(t, err, ErrAsyncImageInsufficientQuota)

	var user User
	var token Token
	require.NoError(t, DB.First(&user, 9101).Error)
	require.NoError(t, DB.First(&token, 9201).Error)
	assert.Equal(t, 1000, user.Quota)
	assert.Equal(t, 10, token.RemainQuota)
	assert.Equal(t, 0, token.UsedQuota)
	assert.ErrorIs(t, DB.Where("task_id = ?", "task_async_reservation_rollback").First(&AsyncImageTask{}).Error, gorm.ErrRecordNotFound)
}

func TestCountAsyncImageStagingInUseProtectsPathChanges(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Create(&AsyncImageTask{
		TaskID:             "task_staging_path_in_use",
		Status:             AsyncImageStatusQueued,
		OutputAvailability: AsyncImageOutputPending,
		BillingStatus:      AsyncImageBillingReserved,
	}).Error)

	count, err := CountAsyncImageStagingInUse()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	require.NoError(t, DB.Model(&AsyncImageTask{}).Where("task_id = ?", "task_staging_path_in_use").Update("status", AsyncImageStatusFailed).Error)
	require.NoError(t, DB.Create(&StorageObject{
		BusinessID:          StorageObjectBusinessAsyncImages,
		ResourceID:          "task_retained_staging",
		StagingRelativePath: "1/2026/07/29/task_retained_staging/0.img",
		StagingStatus:       StorageStagingAvailable,
	}).Error)

	count, err = CountAsyncImageStagingInUse()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestClaimRunnableAsyncImageTaskLoadsStoredRequestBodyAfterLeaseClaim(t *testing.T) {
	truncateTables(t)
	requestBody := []byte{'i', 'm', 'a', 'g', 'e', 0, 1}
	require.NoError(t, DB.Create(&AsyncImageTask{
		TaskID:             "task_claim_request_body",
		Status:             AsyncImageStatusQueued,
		OutputAvailability: AsyncImageOutputPending,
		BillingStatus:      AsyncImageBillingReserved,
		NextAttemptAt:      common.GetTimestamp(),
		RequestPath:        "/v1/images/edits",
		RequestBody:        requestBody,
	}).Error)

	claimed, err := ClaimRunnableAsyncImageTasks("runner-request-body", 1, time.Minute)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	assert.Equal(t, requestBody, claimed[0].RequestBody)
	assert.Equal(t, "runner-request-body", claimed[0].LeaseOwner)
}

func TestAsyncImageSubscriptionReservationRefundsExactlyOnce(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Create(&User{
		Id:       9101,
		Username: "async-image-subscription-user",
		Quota:    1000,
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, DB.Create(&SubscriptionPlan{
		Id:               9301,
		Title:            "Async image plan",
		Currency:         "USD",
		DurationUnit:     "month",
		DurationValue:    1,
		Enabled:          true,
		TotalAmount:      1000,
		QuotaResetPeriod: SubscriptionResetNever,
	}).Error)
	now := time.Now().Unix()
	require.NoError(t, DB.Create(&UserSubscription{
		Id:          9401,
		UserId:      9101,
		PlanId:      9301,
		AmountTotal: 1000,
		AmountUsed:  0,
		StartTime:   now,
		EndTime:     now + 86400,
		Status:      "active",
	}).Error)

	const taskID = "task_async_subscription_refund"
	require.NoError(t, CreateAsyncImageTaskWithReservation(&AsyncImageTask{
		TaskID:             taskID,
		UserID:             9101,
		Status:             AsyncImageStatusQueued,
		OutputAvailability: AsyncImageOutputPending,
		BillingStatus:      AsyncImageBillingReserved,
		ReservedQuota:      200,
		TokenUnlimited:     true,
	}, "subscription_only"))
	var subscription UserSubscription
	require.NoError(t, DB.First(&subscription, 9401).Error)
	assert.Equal(t, int64(200), subscription.AmountUsed)
	var record SubscriptionPreConsumeRecord
	require.NoError(t, DB.Where("request_id = ?", taskID).First(&record).Error)
	assert.Equal(t, "consumed", record.Status)
	assert.Equal(t, int64(200), record.PreConsumed)

	require.NoError(t, RefundAsyncImageBilling(taskID, "upstream_failed", "image generation failed", "image generation failed"))
	require.NoError(t, RefundAsyncImageBilling(taskID, "upstream_failed", "image generation failed", "image generation failed"))
	require.NoError(t, DB.First(&subscription, 9401).Error)
	assert.Equal(t, int64(0), subscription.AmountUsed)
	require.NoError(t, DB.Where("request_id = ?", taskID).First(&record).Error)
	assert.Equal(t, "refunded", record.Status)
	var user User
	require.NoError(t, DB.First(&user, 9101).Error)
	assert.Equal(t, 1000, user.Quota)
}

func TestCompleteAsyncImageGenerationSettlesExactlyOnce(t *testing.T) {
	truncateTables(t)
	const taskID = "task_async_settle_once"
	const owner = "runner-1"
	seedAsyncImageWalletTask(t, taskID, AsyncImageStatusRunning, owner, 200, 200)

	manifest := `[{"index":0,"source_type":"base64","staging_relative_path":"9101/2026/07/29/task_async_settle_once/0.img","size_bytes":16,"mime_type":"image/png","extension":"png","sha256":"0123456789abcdef"}]`
	objects := []StorageObject{{
		ObjectIndex:         0,
		Endpoint:            "https://s3.example.com",
		Region:              "test-region",
		Bucket:              "test-bucket",
		ObjectKey:           "prod/user-files/zmodel@async-images/9101/2026/07/29/task_async_settle_once/0.png",
		MimeType:            "image/png",
		Extension:           "png",
		SizeBytes:           16,
		StagingRelativePath: "9101/2026/07/29/task_async_settle_once/0.img",
		StagingSizeBytes:    16,
		StagingSHA256:       "0123456789abcdef",
	}}

	require.NoError(t, CompleteAsyncImageGeneration(taskID, owner, 150, manifest, "base64", objects))
	quota, tokenRemain, tokenUsed := asyncImageWalletAndTokenQuota(t)
	assert.Equal(t, 850, quota)
	assert.Equal(t, 850, tokenRemain)
	assert.Equal(t, 150, tokenUsed)

	task, err := GetAsyncImageTaskByTaskID(taskID)
	require.NoError(t, err)
	assert.Equal(t, AsyncImageStatusSucceeded, task.Status)
	assert.Equal(t, AsyncImageOutputArchiving, task.OutputAvailability)
	assert.Equal(t, AsyncImageBillingSettled, task.BillingStatus)
	assert.Equal(t, 150, task.ActualQuota)
	var storedTask AsyncImageTask
	require.NoError(t, DB.Select("request_body").Where("task_id = ?", taskID).First(&storedTask).Error)
	assert.Empty(t, storedTask.RequestBody)
	objectsAfter, err := ListStorageObjects(taskID)
	require.NoError(t, err)
	require.Len(t, objectsAfter, 1)
	assert.Equal(t, StorageObjectStatusUploading, objectsAfter[0].Status)

	require.NoError(t, CompleteAsyncImageGeneration(taskID, owner, 150, manifest, "base64", objects))
	quota, tokenRemain, tokenUsed = asyncImageWalletAndTokenQuota(t)
	assert.Equal(t, 850, quota)
	assert.Equal(t, 850, tokenRemain)
	assert.Equal(t, 150, tokenUsed)
	require.Error(t, RefundAsyncImageBilling(taskID, "should_not_refund", "should not refund", "should not refund"))
}

func TestRefundAsyncImageBillingRefundsExactlyOnceAndCannotSettle(t *testing.T) {
	truncateTables(t)
	const taskID = "task_async_refund_once"
	seedAsyncImageWalletTask(t, taskID, AsyncImageStatusRunning, "runner-1", 200, 200)
	startedAt := common.GetTimestamp() - 3
	require.NoError(t, DB.Model(&AsyncImageTask{}).Where("task_id = ?", taskID).Updates(map[string]any{
		"last_channel_id": 9301,
		"started_at":      startedAt,
		"request_path":    "/v1/images/edits",
	}).Error)

	const failureReason = "status_code=500, upstream image generation failed"
	require.NoError(t, RefundAsyncImageBilling(taskID, "upstream_failed", "image generation failed", failureReason))
	require.NoError(t, RefundAsyncImageBilling(taskID, "upstream_failed", "image generation failed", failureReason))
	quota, tokenRemain, tokenUsed := asyncImageWalletAndTokenQuota(t)
	assert.Equal(t, 1000, quota)
	assert.Equal(t, 1000, tokenRemain)
	assert.Equal(t, 0, tokenUsed)

	task, err := GetAsyncImageTaskByTaskID(taskID)
	require.NoError(t, err)
	assert.Equal(t, AsyncImageStatusFailed, task.Status)
	assert.Equal(t, AsyncImageBillingRefunded, task.BillingStatus)
	assert.Equal(t, "image generation failed", task.PublicErrorMessage)
	assert.Equal(t, failureReason, task.LastError)
	var storedTask AsyncImageTask
	require.NoError(t, DB.Select("request_body").Where("task_id = ?", taskID).First(&storedTask).Error)
	assert.Empty(t, storedTask.RequestBody)

	var logs []Log
	require.NoError(t, LOG_DB.Where("request_id = ? AND type = ?", taskID, LogTypeRefund).Find(&logs).Error)
	require.Len(t, logs, 1)
	refundLog := logs[0]
	assert.Equal(t, 9101, refundLog.UserId)
	assert.Equal(t, "async-image-user", refundLog.Username)
	assert.Equal(t, 9201, refundLog.TokenId)
	assert.Equal(t, "async image token", refundLog.TokenName)
	assert.Equal(t, 9301, refundLog.ChannelId)
	assert.Equal(t, "test-image-model", refundLog.ModelName)
	assert.Equal(t, "default", refundLog.Group)
	assert.Equal(t, 200, refundLog.Quota)
	assert.GreaterOrEqual(t, refundLog.UseTime, 3)
	assert.Equal(t, failureReason, refundLog.Content)
	var other map[string]interface{}
	require.NoError(t, common.UnmarshalJsonStr(refundLog.Other, &other))
	assert.Equal(t, taskID, other["task_id"])
	assert.Equal(t, "/v1/images/edits/tasks", other["request_path"])
	assert.Equal(t, "upstream_failed", other["error_code"])
	assert.Equal(t, failureReason, other["reason"])
	assert.Equal(t, true, other["async_image_task"])
	require.Error(t, CompleteAsyncImageGeneration(
		taskID,
		"",
		100,
		`[{"index":0}]`,
		"base64",
		[]StorageObject{{ObjectIndex: 0}},
	))
}

func TestScheduleAsyncImageGenerationRetryReleasesLeaseWithoutChangingBilling(t *testing.T) {
	truncateTables(t)
	const taskID = "task_async_generation_retry"
	seedAsyncImageWalletTask(t, taskID, AsyncImageStatusQueued, "runner-1", 100, 100)

	require.NoError(t, ScheduleAsyncImageGenerationRetry(taskID, "runner-1", 12345, "staging unavailable"))
	task, err := GetAsyncImageTaskByTaskID(taskID)
	require.NoError(t, err)
	assert.Equal(t, AsyncImageStatusQueued, task.Status)
	assert.Equal(t, AsyncImageOutputPending, task.OutputAvailability)
	assert.Equal(t, AsyncImageBillingReserved, task.BillingStatus)
	assert.Equal(t, int64(12345), task.NextAttemptAt)
	assert.Empty(t, task.LeaseOwner)
	assert.Zero(t, task.LeaseExpiresAt)
	assert.Empty(t, task.RequestBody)
	var storedTask AsyncImageTask
	require.NoError(t, DB.Select("request_body").Where("task_id = ?", taskID).First(&storedTask).Error)
	assert.Equal(t, []byte{'i', 'm', 'a', 'g', 'e', 0, 1}, storedTask.RequestBody)
	listed, total, err := ListAsyncImageTasks(AsyncImageTaskFilter{TaskID: taskID})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, listed, 1)
	assert.Empty(t, listed[0].RequestBody)
	assert.ErrorIs(t, ScheduleAsyncImageGenerationRetry(taskID, "runner-1", 12346, "staging unavailable"), ErrAsyncImageLeaseLost)
}

func TestUpdateObjectStorageOptionsRebindsFailedTaskAsOneTransaction(t *testing.T) {
	truncateTables(t)
	const taskID = "task_async_storage_rebind"
	seedAsyncImageWalletTask(t, taskID, AsyncImageStatusRunning, "runner-1", 100, 100)
	manifest := `[{"index":0,"source_type":"base64","staging_relative_path":"9101/2026/07/29/task_async_storage_rebind/0.img","size_bytes":16,"mime_type":"image/png","extension":"png","sha256":"0123456789abcdef"}]`
	require.NoError(t, CompleteAsyncImageGeneration(taskID, "runner-1", 100, manifest, "base64", []StorageObject{{
		ObjectIndex:         0,
		Endpoint:            "https://old.example.com",
		Region:              "old-region",
		Bucket:              "old-bucket",
		ObjectKey:           "old-key",
		MimeType:            "image/png",
		Extension:           "png",
		SizeBytes:           16,
		StagingRelativePath: "9101/2026/07/29/task_async_storage_rebind/0.img",
		StagingSizeBytes:    16,
		StagingSHA256:       "0123456789abcdef",
	}}))
	require.NoError(t, DB.Model(&AsyncImageTask{}).Where("task_id = ?", taskID).Updates(map[string]any{
		"output_availability": AsyncImageOutputFailed,
		"completed_at":        int64(100),
	}).Error)
	require.NoError(t, DB.Model(&StorageObject{}).Where("resource_id = ?", taskID).Update("status", StorageObjectStatusFailed).Error)

	values := map[string]string{"ObjectStorageS3Endpoint": "https://new.example.com"}
	require.NoError(t, UpdateObjectStorageOptionsWithRebind(values, []string{taskID}, "https://new.example.com", "new-region", "new-bucket", 99999))

	task, err := GetAsyncImageTaskByTaskID(taskID)
	require.NoError(t, err)
	assert.Equal(t, AsyncImageOutputArchiving, task.OutputAvailability)
	assert.Equal(t, int64(99999), task.ArchiveRetryDeadlineAt)
	objects, err := ListStorageObjects(taskID)
	require.NoError(t, err)
	require.Len(t, objects, 1)
	assert.Equal(t, StorageObjectStatusUploading, objects[0].Status)
	assert.Equal(t, "https://new.example.com", objects[0].Endpoint)
	assert.Equal(t, "new-region", objects[0].Region)
	assert.Equal(t, "new-bucket", objects[0].Bucket)
	var option Option
	require.NoError(t, DB.Where("key = ?", "ObjectStorageS3Endpoint").First(&option).Error)
	assert.Equal(t, "https://new.example.com", option.Value)
}
