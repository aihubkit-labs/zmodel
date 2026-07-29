package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	AsyncImageStatusQueued    = "queued"
	AsyncImageStatusRunning   = "running"
	AsyncImageStatusSucceeded = "succeeded"
	AsyncImageStatusFailed    = "failed"

	AsyncImageOutputPending   = "pending"
	AsyncImageOutputArchiving = "archiving"
	AsyncImageOutputAvailable = "available"
	AsyncImageOutputExpired   = "expired"
	AsyncImageOutputFailed    = "failed"

	AsyncImageBillingReserved = "reserved"
	AsyncImageBillingSettled  = "settled"
	AsyncImageBillingRefunded = "refunded"

	StorageObjectBusinessAsyncImages = "zmodel@async-images"
	StorageObjectProviderS3          = "s3"
	StorageObjectStatusUploading     = "uploading"
	StorageObjectStatusAvailable     = "available"
	StorageObjectStatusFailed        = "failed"
	StorageObjectStatusDeletePending = "delete_pending"
	StorageObjectStatusDeleted       = "deleted"
	StorageStagingPending            = "pending"
	StorageStagingAvailable          = "available"
	StorageStagingFailed             = "failed"
	StorageStagingDeletePending      = "delete_pending"
	StorageStagingDeleted            = "deleted"
)

var ErrAsyncImageLeaseLost = errors.New("async image task lease lost")
var ErrAsyncImageInsufficientQuota = errors.New("async image quota is insufficient")

type AsyncImageTask struct {
	ID                         int64  `json:"id" gorm:"primaryKey"`
	TaskID                     string `json:"task_id" gorm:"type:varchar(64);uniqueIndex"`
	UserID                     int    `json:"user_id" gorm:"index"`
	TokenID                    int    `json:"token_id" gorm:"index"`
	Status                     string `json:"status" gorm:"type:varchar(32);index:idx_async_image_runnable,priority:1"`
	OutputAvailability         string `json:"output_availability" gorm:"type:varchar(32);index:idx_async_image_runnable,priority:2"`
	BillingStatus              string `json:"billing_status" gorm:"type:varchar(32);index"`
	BillingSource              string `json:"billing_source" gorm:"type:varchar(32)"`
	SubscriptionID             int    `json:"subscription_id"`
	ReservedQuota              int    `json:"reserved_quota"`
	ActualQuota                int    `json:"actual_quota"`
	TokenReservedQuota         int    `json:"token_reserved_quota"`
	TokenUnlimited             bool   `json:"token_unlimited"`
	OriginModelName            string `json:"origin_model_name" gorm:"type:varchar(191);index"`
	UsingGroup                 string `json:"using_group" gorm:"type:varchar(64)"`
	LastChannelID              int    `json:"last_channel_id" gorm:"index"`
	LastChannelType            int    `json:"last_channel_type"`
	RequestPayload             string `json:"-" gorm:"type:text"`
	BillingContext             string `json:"-" gorm:"type:text"`
	ArchiveManifest            string `json:"-" gorm:"type:text"`
	RetentionSeconds           int64  `json:"retention_seconds"`
	ArchiveTimeoutSeconds      int64  `json:"archive_timeout_seconds"`
	ArchiveMaxAttempts         int    `json:"archive_max_attempts"`
	ArchiveRetryDeadlineAt     int64  `json:"archive_retry_deadline_at" gorm:"index"`
	ArchiveAttempts            int    `json:"archive_attempts"`
	NextAttemptAt              int64  `json:"next_attempt_at" gorm:"index:idx_async_image_runnable,priority:3"`
	OutputExpiresAt            int64  `json:"output_expires_at" gorm:"index"`
	LeaseOwner                 string `json:"lease_owner" gorm:"type:varchar(128);index"`
	LeaseExpiresAt             int64  `json:"lease_expires_at" gorm:"index"`
	SourceKind                 string `json:"source_kind" gorm:"type:varchar(32)"`
	PublicErrorCode            string `json:"public_error_code" gorm:"type:varchar(64)"`
	PublicErrorMessage         string `json:"public_error_message" gorm:"type:text"`
	LastError                  string `json:"last_error,omitempty" gorm:"type:text"`
	GenerationCompletedAt      int64  `json:"generation_completed_at"`
	BillingFinalizedAt         int64  `json:"billing_finalized_at"`
	ManuallyRecoveredAt        int64  `json:"manually_recovered_at"`
	AdminNotificationState     string `json:"admin_notification_state" gorm:"type:varchar(32)"`
	AdminNotificationClaimedAt int64  `json:"admin_notification_claimed_at"`
	CreatedAt                  int64  `json:"created_at" gorm:"index"`
	StartedAt                  int64  `json:"started_at"`
	CompletedAt                int64  `json:"completed_at"`
	UpdatedAt                  int64  `json:"updated_at" gorm:"index"`
}

type StorageObject struct {
	ID                  int64  `json:"id" gorm:"primaryKey"`
	BusinessID          string `json:"business_id" gorm:"type:varchar(64);uniqueIndex:idx_storage_object_identity"`
	ResourceID          string `json:"resource_id" gorm:"type:varchar(64);uniqueIndex:idx_storage_object_identity"`
	ObjectIndex         int    `json:"object_index" gorm:"uniqueIndex:idx_storage_object_identity"`
	Provider            string `json:"provider" gorm:"type:varchar(32)"`
	Status              string `json:"status" gorm:"type:varchar(32);index"`
	Endpoint            string `json:"endpoint" gorm:"type:varchar(512)"`
	Region              string `json:"region" gorm:"type:varchar(128)"`
	Bucket              string `json:"bucket" gorm:"type:varchar(255)"`
	ObjectKey           string `json:"object_key" gorm:"type:varchar(768)"`
	MimeType            string `json:"mime_type" gorm:"type:varchar(128)"`
	Extension           string `json:"extension" gorm:"type:varchar(32)"`
	SizeBytes           int64  `json:"size_bytes"`
	ETag                string `json:"etag" gorm:"column:etag;type:varchar(255)"`
	UploadedAt          int64  `json:"uploaded_at"`
	ExpiresAt           int64  `json:"expires_at" gorm:"index"`
	DeletedAt           int64  `json:"deleted_at"`
	DeleteAttempts      int    `json:"delete_attempts"`
	LastError           string `json:"last_error,omitempty" gorm:"type:text"`
	StagingRelativePath string `json:"-" gorm:"type:varchar(768)"`
	StagingStatus       string `json:"staging_status" gorm:"type:varchar(32);index"`
	StagingSizeBytes    int64  `json:"staging_size_bytes"`
	StagingSHA256       string `json:"staging_sha256" gorm:"type:varchar(64)"`
	StagedAt            int64  `json:"staged_at"`
	StagingDeletedAt    int64  `json:"staging_deleted_at"`
	CreatedAt           int64  `json:"created_at" gorm:"index"`
	UpdatedAt           int64  `json:"updated_at" gorm:"index"`
}

func (task *AsyncImageTask) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	if task.CreatedAt == 0 {
		task.CreatedAt = now
	}
	task.UpdatedAt = now
	return nil
}

func (task *AsyncImageTask) BeforeUpdate(_ *gorm.DB) error {
	task.UpdatedAt = common.GetTimestamp()
	return nil
}

func (object *StorageObject) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	if object.CreatedAt == 0 {
		object.CreatedAt = now
	}
	object.UpdatedAt = now
	return nil
}

func (object *StorageObject) BeforeUpdate(_ *gorm.DB) error {
	object.UpdatedAt = common.GetTimestamp()
	return nil
}

func GenerateAsyncImageTaskID() (string, error) {
	key, err := common.GenerateRandomCharsKey(32)
	if err != nil {
		return "", err
	}
	return "task_" + key, nil
}

func CreateAsyncImageTask(task *AsyncImageTask) error {
	return DB.Create(task).Error
}

// CreateAsyncImageTaskWithReservation persists the task and reserves wallet or
// subscription funding plus token quota in one main-database transaction.
func CreateAsyncImageTaskWithReservation(task *AsyncImageTask, billingPreference string) error {
	if task == nil || task.TaskID == "" {
		return errors.New("invalid async image task")
	}
	if task.ReservedQuota < 0 || task.TokenReservedQuota < 0 {
		return errors.New("reserved quota cannot be negative")
	}
	var walletDelta int
	var tokenDelta int
	var tokenKey string
	err := DB.Transaction(func(tx *gorm.DB) error {
		if task.ReservedQuota > 0 {
			source, subscriptionID, err := reserveAsyncImageFundingTx(tx, task.UserID, task.TaskID, task.ReservedQuota, billingPreference)
			if err != nil {
				return err
			}
			task.BillingSource = source
			task.SubscriptionID = subscriptionID
			if source == "wallet" {
				walletDelta = task.ReservedQuota
			}
		} else {
			task.BillingSource = "wallet"
		}

		if task.TokenReservedQuota > 0 && !task.TokenUnlimited && task.TokenID > 0 {
			var token Token
			if err := lockForUpdate(tx).Where("id = ? AND user_id = ?", task.TokenID, task.UserID).First(&token).Error; err != nil {
				return err
			}
			if token.RemainQuota < task.TokenReservedQuota {
				return fmt.Errorf("%w: token remain quota is insufficient", ErrAsyncImageInsufficientQuota)
			}
			tokenKey = token.Key
			tokenDelta = task.TokenReservedQuota
			if err := tx.Model(&Token{}).Where("id = ?", token.Id).Updates(map[string]any{
				"remain_quota":  token.RemainQuota - task.TokenReservedQuota,
				"used_quota":    token.UsedQuota + task.TokenReservedQuota,
				"accessed_time": common.GetTimestamp(),
			}).Error; err != nil {
				return err
			}
		}
		return tx.Create(task).Error
	})
	if err == nil {
		syncAsyncImageBillingCaches(task.UserID, tokenKey, walletDelta, tokenDelta)
	}
	return err
}

func reserveAsyncImageFundingTx(tx *gorm.DB, userID int, requestID string, amount int, preference string) (string, int, error) {
	tryWallet := func() (string, int, error) {
		var user User
		if err := lockForUpdate(tx).Where("id = ?", userID).First(&user).Error; err != nil {
			return "", 0, err
		}
		if user.Quota < amount {
			return "", 0, fmt.Errorf("%w: wallet quota is insufficient", ErrAsyncImageInsufficientQuota)
		}
		if err := tx.Model(&User{}).Where("id = ?", user.Id).Update("quota", user.Quota-amount).Error; err != nil {
			return "", 0, err
		}
		return "wallet", 0, nil
	}
	trySubscription := func() (string, int, error) {
		now := common.GetTimestamp()
		var subscriptions []UserSubscription
		if err := lockForUpdate(tx).
			Where("user_id = ? AND status = ? AND end_time > ?", userID, "active", now).
			Order("end_time asc, id asc").Find(&subscriptions).Error; err != nil {
			return "", 0, err
		}
		for index := range subscriptions {
			subscription := &subscriptions[index]
			plan, err := getSubscriptionPlanByIdTx(tx, subscription.PlanId)
			if err != nil {
				return "", 0, err
			}
			if err := maybeResetUserSubscriptionWithPlanTx(tx, subscription, plan, now); err != nil {
				return "", 0, err
			}
			if subscription.AmountTotal > 0 && subscription.AmountTotal-subscription.AmountUsed < int64(amount) {
				continue
			}
			record := &SubscriptionPreConsumeRecord{
				RequestId: requestID, UserId: userID, UserSubscriptionId: subscription.Id,
				PreConsumed: int64(amount), Status: "consumed",
			}
			if err := tx.Create(record).Error; err != nil {
				return "", 0, err
			}
			if err := tx.Model(&UserSubscription{}).Where("id = ?", subscription.Id).
				Update("amount_used", subscription.AmountUsed+int64(amount)).Error; err != nil {
				return "", 0, err
			}
			return "subscription", subscription.Id, nil
		}
		return "", 0, fmt.Errorf("%w: subscription quota is insufficient", ErrAsyncImageInsufficientQuota)
	}

	preference = strings.TrimSpace(preference)
	switch preference {
	case "wallet_only":
		return tryWallet()
	case "subscription_only":
		return trySubscription()
	case "wallet_first":
		source, subscriptionID, err := tryWallet()
		if err == nil || !errors.Is(err, ErrAsyncImageInsufficientQuota) {
			return source, subscriptionID, err
		}
		return trySubscription()
	default:
		source, subscriptionID, err := trySubscription()
		if err == nil || !errors.Is(err, ErrAsyncImageInsufficientQuota) {
			return source, subscriptionID, err
		}
		var activeCount int64
		if countErr := tx.Model(&UserSubscription{}).
			Where("user_id = ? AND status = ? AND end_time > ?", userID, "active", common.GetTimestamp()).
			Count(&activeCount).Error; countErr != nil {
			return "", 0, countErr
		}
		if activeCount == 0 {
			return tryWallet()
		}
		var overflowCount int64
		if err := tx.Model(&UserSubscription{}).
			Where("user_id = ? AND status = ? AND end_time > ? AND allow_wallet_overflow = ?", userID, "active", common.GetTimestamp(), true).
			Count(&overflowCount).Error; err != nil {
			return "", 0, err
		}
		if overflowCount == 0 {
			return "", 0, err
		}
		return tryWallet()
	}
}

// CompleteAsyncImageGeneration atomically records the durable staging
// manifest, creates the corresponding storage rows, and finalizes billing.
// The task row is the idempotency boundary: a repeated call after settlement
// is a no-op, while a refunded task can never be charged again.
func CompleteAsyncImageGeneration(taskID string, owner string, actualQuota int, manifest string, sourceKind string, objects []StorageObject) error {
	if actualQuota < 0 {
		return errors.New("actual quota cannot be negative")
	}
	if strings.TrimSpace(manifest) == "" || len(objects) == 0 {
		return errors.New("archive manifest is required")
	}
	var walletDelta int
	var tokenDelta int
	var userID int
	var tokenKey string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var task AsyncImageTask
		query := lockForUpdate(tx).Where("task_id = ?", taskID)
		if owner != "" {
			query = query.Where("lease_owner = ?", owner)
		}
		if err := query.First(&task).Error; err != nil {
			return err
		}
		if task.BillingStatus == AsyncImageBillingSettled {
			return nil
		}
		if task.BillingStatus == AsyncImageBillingRefunded {
			return errors.New("refunded async image task cannot be settled")
		}
		if task.Status != AsyncImageStatusRunning {
			return fmt.Errorf("async image task is not running: %s", task.Status)
		}

		fundingDelta := actualQuota - task.ReservedQuota
		if task.BillingSource != "subscription" || task.SubscriptionID <= 0 {
			walletDelta = fundingDelta
		}
		tokenActual := actualQuota
		if task.TokenUnlimited {
			tokenActual = 0
		}
		tokenDelta = tokenActual - task.TokenReservedQuota
		if err := adjustAsyncImageFundingTx(tx, &task, fundingDelta); err != nil {
			return err
		}
		if err := adjustAsyncImageTokenTx(tx, &task, tokenDelta, &tokenKey); err != nil {
			return err
		}

		now := common.GetTimestamp()
		for index := range objects {
			objects[index].BusinessID = StorageObjectBusinessAsyncImages
			objects[index].ResourceID = task.TaskID
			objects[index].Provider = StorageObjectProviderS3
			objects[index].Status = StorageObjectStatusUploading
			objects[index].StagingStatus = StorageStagingAvailable
			if err := tx.Create(&objects[index]).Error; err != nil {
				return err
			}
		}
		userID = task.UserID
		updates := map[string]any{
			"actual_quota":            actualQuota,
			"billing_status":          AsyncImageBillingSettled,
			"billing_finalized_at":    now,
			"status":                  AsyncImageStatusSucceeded,
			"output_availability":     AsyncImageOutputArchiving,
			"generation_completed_at": now,
			"archive_manifest":        manifest,
			"source_kind":             sourceKind,
			"request_payload":         "",
			"public_error_code":       "",
			"public_error_message":    "",
			"last_error":              "",
			"updated_at":              now,
		}
		result := tx.Model(&AsyncImageTask{}).
			Where("id = ? AND billing_status = ?", task.ID, AsyncImageBillingReserved).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("async image billing state changed during settlement")
		}
		return nil
	})
	if err == nil {
		syncAsyncImageBillingCaches(userID, tokenKey, walletDelta, tokenDelta)
	}
	return err
}

// CompleteAsyncImageGenerationWithArchiveFailure settles a valid upstream
// generation even when source retrieval or persistent staging failed. These
// failures are delivery incidents and must never refund an incurred upstream
// cost.
func CompleteAsyncImageGenerationWithArchiveFailure(taskID string, owner string, actualQuota int, sourceKind string, errorCode string, errorMessage string) error {
	if actualQuota < 0 {
		return errors.New("actual quota cannot be negative")
	}
	var walletDelta int
	var tokenDelta int
	var userID int
	var tokenKey string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var task AsyncImageTask
		query := lockForUpdate(tx).Where("task_id = ?", taskID)
		if owner != "" {
			query = query.Where("lease_owner = ?", owner)
		}
		if err := query.First(&task).Error; err != nil {
			return err
		}
		if task.BillingStatus == AsyncImageBillingSettled {
			return nil
		}
		if task.BillingStatus == AsyncImageBillingRefunded {
			return errors.New("refunded async image task cannot be settled")
		}
		fundingDelta := actualQuota - task.ReservedQuota
		if task.BillingSource != "subscription" || task.SubscriptionID <= 0 {
			walletDelta = fundingDelta
		}
		tokenActual := actualQuota
		if task.TokenUnlimited {
			tokenActual = 0
		}
		tokenDelta = tokenActual - task.TokenReservedQuota
		if err := adjustAsyncImageFundingTx(tx, &task, fundingDelta); err != nil {
			return err
		}
		if err := adjustAsyncImageTokenTx(tx, &task, tokenDelta, &tokenKey); err != nil {
			return err
		}
		now := common.GetTimestamp()
		userID = task.UserID
		result := tx.Model(&AsyncImageTask{}).
			Where("id = ? AND billing_status = ?", task.ID, AsyncImageBillingReserved).
			Updates(map[string]any{
				"actual_quota":                  actualQuota,
				"billing_status":                AsyncImageBillingSettled,
				"billing_finalized_at":          now,
				"status":                        AsyncImageStatusSucceeded,
				"output_availability":           AsyncImageOutputFailed,
				"generation_completed_at":       now,
				"completed_at":                  now,
				"source_kind":                   sourceKind,
				"request_payload":               "",
				"public_error_code":             errorCode,
				"public_error_message":          errorMessage,
				"last_error":                    errorMessage,
				"admin_notification_state":      "pending",
				"admin_notification_claimed_at": 0,
				"lease_owner":                   "",
				"lease_expires_at":              0,
				"updated_at":                    now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("async image billing state changed during settlement")
		}
		return nil
	})
	if err == nil {
		syncAsyncImageBillingCaches(userID, tokenKey, walletDelta, tokenDelta)
	}
	return err
}

func GetAsyncImageTaskForUser(taskID string, userID int) (*AsyncImageTask, error) {
	var task AsyncImageTask
	if err := DB.Where("task_id = ? AND user_id = ?", taskID, userID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func GetAsyncImageTaskByTaskID(taskID string) (*AsyncImageTask, error) {
	var task AsyncImageTask
	if err := DB.Where("task_id = ?", taskID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func ClaimRunnableAsyncImageTasks(owner string, limit int, leaseDuration time.Duration) ([]AsyncImageTask, error) {
	if owner == "" {
		return nil, errors.New("lease owner is required")
	}
	if limit <= 0 {
		limit = 4
	}
	now := common.GetTimestamp()
	leaseUntil := now + int64(leaseDuration.Seconds())
	var candidates []AsyncImageTask
	err := DB.Where("next_attempt_at <= ? AND (lease_owner = '' OR lease_expires_at <= ?)", now, now).
		Where("(status IN ? AND output_availability = ? AND billing_status = ?) OR (status = ? AND output_availability = ? AND billing_status = ?)",
			[]string{AsyncImageStatusQueued, AsyncImageStatusRunning}, AsyncImageOutputPending, AsyncImageBillingReserved,
			AsyncImageStatusSucceeded, AsyncImageOutputArchiving, AsyncImageBillingSettled).
		Order("id asc").Limit(limit * 3).Find(&candidates).Error
	if err != nil {
		return nil, err
	}
	claimed := make([]AsyncImageTask, 0, limit)
	for _, candidate := range candidates {
		result := DB.Model(&AsyncImageTask{}).
			Where("id = ? AND next_attempt_at <= ? AND (lease_owner = '' OR lease_expires_at <= ?)", candidate.ID, now, now).
			Updates(map[string]any{"lease_owner": owner, "lease_expires_at": leaseUntil, "updated_at": now})
		if result.Error != nil {
			return claimed, result.Error
		}
		if result.RowsAffected != 1 {
			continue
		}
		candidate.LeaseOwner = owner
		candidate.LeaseExpiresAt = leaseUntil
		claimed = append(claimed, candidate)
		if len(claimed) >= limit {
			break
		}
	}
	return claimed, nil
}

func RenewAsyncImageTaskLease(taskID string, owner string, leaseDuration time.Duration) error {
	now := common.GetTimestamp()
	result := DB.Model(&AsyncImageTask{}).
		Where("task_id = ? AND lease_owner = ? AND lease_expires_at > ?", taskID, owner, now).
		Updates(map[string]any{"lease_expires_at": now + int64(leaseDuration.Seconds()), "updated_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrAsyncImageLeaseLost
	}
	return nil
}

func ReleaseAsyncImageTaskLease(taskID string, owner string) error {
	result := DB.Model(&AsyncImageTask{}).
		Where("task_id = ? AND lease_owner = ?", taskID, owner).
		Updates(map[string]any{"lease_owner": "", "lease_expires_at": 0, "updated_at": common.GetTimestamp()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrAsyncImageLeaseLost
	}
	return nil
}

func ListStorageObjects(resourceID string) ([]StorageObject, error) {
	var objects []StorageObject
	err := DB.Where("business_id = ? AND resource_id = ?", StorageObjectBusinessAsyncImages, resourceID).
		Order("object_index asc").Find(&objects).Error
	return objects, err
}

func CountActiveStorageObjects() (int64, error) {
	var count int64
	err := DB.Model(&StorageObject{}).
		Where("status IN ?", []string{StorageObjectStatusUploading, StorageObjectStatusAvailable, StorageObjectStatusDeletePending}).
		Count(&count).Error
	return count, err
}

func CountAsyncImageStagingInUse() (int64, error) {
	var taskCount int64
	if err := DB.Model(&AsyncImageTask{}).
		Where("status IN ?", []string{AsyncImageStatusQueued, AsyncImageStatusRunning}).
		Count(&taskCount).Error; err != nil {
		return 0, err
	}
	var objectCount int64
	if err := DB.Model(&StorageObject{}).
		Where("business_id = ? AND staging_status IN ?", StorageObjectBusinessAsyncImages, []string{
			StorageStagingPending,
			StorageStagingAvailable,
			StorageStagingFailed,
			StorageStagingDeletePending,
		}).Count(&objectCount).Error; err != nil {
		return 0, err
	}
	return taskCount + objectCount, nil
}

type AsyncImageStorageRebindCandidate struct {
	Task    AsyncImageTask
	Objects []StorageObject
}

func ListAsyncImageStorageRebindCandidates() ([]AsyncImageStorageRebindCandidate, error) {
	var tasks []AsyncImageTask
	if err := DB.Where("status = ? AND billing_status = ? AND output_availability = ?", AsyncImageStatusSucceeded, AsyncImageBillingSettled, AsyncImageOutputFailed).
		Order("id asc").Find(&tasks).Error; err != nil {
		return nil, err
	}
	candidates := make([]AsyncImageStorageRebindCandidate, 0, len(tasks))
	for _, task := range tasks {
		objects, err := ListStorageObjects(task.TaskID)
		if err != nil {
			return nil, err
		}
		if len(objects) == 0 {
			continue
		}
		candidates = append(candidates, AsyncImageStorageRebindCandidate{Task: task, Objects: objects})
	}
	return candidates, nil
}

func UpdateObjectStorageOptionsWithRebind(values map[string]string, taskIDs []string, endpoint string, region string, bucket string, retryDeadlineAt int64) error {
	now := common.GetTimestamp()
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := updateOptionsBulkTx(tx, values); err != nil {
			return err
		}
		for _, taskID := range taskIDs {
			var task AsyncImageTask
			if err := lockForUpdate(tx).
				Where("task_id = ? AND status = ? AND billing_status = ? AND output_availability = ?", taskID, AsyncImageStatusSucceeded, AsyncImageBillingSettled, AsyncImageOutputFailed).
				First(&task).Error; err != nil {
				return err
			}
			var objects []StorageObject
			if err := lockForUpdate(tx).
				Where("business_id = ? AND resource_id = ?", StorageObjectBusinessAsyncImages, taskID).
				Order("object_index asc").Find(&objects).Error; err != nil {
				return err
			}
			if len(objects) == 0 {
				return errors.New("failed async image task has no storage objects")
			}
			for _, object := range objects {
				if object.StagingStatus != StorageStagingAvailable || (object.Status != StorageObjectStatusFailed && object.Status != StorageObjectStatusDeleted) {
					return errors.New("async image storage rebind state changed")
				}
			}
			if err := tx.Model(&StorageObject{}).
				Where("business_id = ? AND resource_id = ?", StorageObjectBusinessAsyncImages, taskID).
				Updates(map[string]any{
					"endpoint":        endpoint,
					"region":          region,
					"bucket":          bucket,
					"status":          StorageObjectStatusUploading,
					"etag":            "",
					"uploaded_at":     0,
					"expires_at":      0,
					"deleted_at":      0,
					"delete_attempts": 0,
					"last_error":      "",
					"updated_at":      now,
				}).Error; err != nil {
				return err
			}
			result := tx.Model(&AsyncImageTask{}).
				Where("id = ? AND output_availability = ?", task.ID, AsyncImageOutputFailed).
				Updates(map[string]any{
					"output_availability":           AsyncImageOutputArchiving,
					"archive_attempts":              0,
					"archive_retry_deadline_at":     retryDeadlineAt,
					"next_attempt_at":               now,
					"public_error_code":             "",
					"public_error_message":          "",
					"last_error":                    "",
					"admin_notification_state":      "none",
					"admin_notification_claimed_at": 0,
					"lease_owner":                   "",
					"lease_expires_at":              0,
					"updated_at":                    now,
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.New("async image storage rebind state changed")
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for key, value := range values {
		if err := updateOptionMap(key, value); err != nil {
			return err
		}
	}
	return nil
}

func UpdateAsyncImageTaskWithLease(taskID string, owner string, updates map[string]any) error {
	updates["updated_at"] = common.GetTimestamp()
	result := DB.Model(&AsyncImageTask{}).Where("task_id = ? AND lease_owner = ?", taskID, owner).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrAsyncImageLeaseLost
	}
	return nil
}

func MarkExpiredAsyncImageTask(taskID string, now int64) error {
	result := DB.Model(&AsyncImageTask{}).
		Where("task_id = ? AND output_availability = ? AND output_expires_at > 0 AND output_expires_at <= ?", taskID, AsyncImageOutputAvailable, now).
		Updates(map[string]any{"output_availability": AsyncImageOutputExpired, "updated_at": now})
	return result.Error
}

func MarkAsyncImageTaskRunning(taskID string, owner string) error {
	now := common.GetTimestamp()
	result := DB.Model(&AsyncImageTask{}).
		Where("task_id = ? AND lease_owner = ? AND status IN ? AND billing_status = ?", taskID, owner, []string{AsyncImageStatusQueued, AsyncImageStatusRunning}, AsyncImageBillingReserved).
		Updates(map[string]any{
			"status":     AsyncImageStatusRunning,
			"started_at": gorm.Expr("CASE WHEN started_at = 0 THEN ? ELSE started_at END", now),
			"last_error": "",
			"updated_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrAsyncImageLeaseLost
	}
	return nil
}

func GetStorageObject(resourceID string, objectIndex int) (*StorageObject, error) {
	var object StorageObject
	if err := DB.Where("business_id = ? AND resource_id = ? AND object_index = ?", StorageObjectBusinessAsyncImages, resourceID, objectIndex).First(&object).Error; err != nil {
		return nil, err
	}
	return &object, nil
}

func MarkStorageObjectAvailable(id int64, expectedStatus string, etag string, uploadedAt int64, expiresAt int64) error {
	result := DB.Model(&StorageObject{}).
		Where("id = ? AND status = ?", id, expectedStatus).
		Updates(map[string]any{
			"status":      StorageObjectStatusAvailable,
			"etag":        etag,
			"uploaded_at": uploadedAt,
			"expires_at":  expiresAt,
			"last_error":  "",
			"updated_at":  common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("storage object state changed")
	}
	return nil
}

func MarkStorageObjectUploading(id int64) error {
	result := DB.Model(&StorageObject{}).
		Where("id = ? AND status NOT IN ?", id, []string{StorageObjectStatusDeletePending}).
		Updates(map[string]any{
			"status":     StorageObjectStatusUploading,
			"last_error": "",
			"updated_at": common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("storage object is not uploadable")
	}
	return nil
}

func MarkStorageObjectFailed(id int64, errorMessage string) error {
	return DB.Model(&StorageObject{}).
		Where("id = ? AND status <> ?", id, StorageObjectStatusDeletePending).
		Updates(map[string]any{
			"status":     StorageObjectStatusFailed,
			"last_error": errorMessage,
			"updated_at": common.GetTimestamp(),
		}).Error
}

func MarkStorageObjectStagingFailed(id int64, errorMessage string) error {
	return DB.Model(&StorageObject{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":         StorageObjectStatusFailed,
			"staging_status": StorageStagingFailed,
			"last_error":     errorMessage,
			"updated_at":     common.GetTimestamp(),
		}).Error
}

func MarkStorageObjectStagingAvailable(id int64) error {
	return DB.Model(&StorageObject{}).
		Where("id = ? AND staging_status = ?", id, StorageStagingFailed).
		Updates(map[string]any{
			"staging_status": StorageStagingAvailable,
			"last_error":     "",
			"updated_at":     common.GetTimestamp(),
		}).Error
}

func MarkAsyncImageStagingIntegrityIncident(taskID string, errorMessage string) error {
	now := common.GetTimestamp()
	return DB.Model(&AsyncImageTask{}).
		Where("task_id = ? AND status = ? AND billing_status = ?", taskID, AsyncImageStatusSucceeded, AsyncImageBillingSettled).
		Updates(map[string]any{
			"output_availability":           AsyncImageOutputFailed,
			"public_error_code":             "archive_staging_integrity_failed",
			"public_error_message":          "staged image is missing or damaged",
			"last_error":                    errorMessage,
			"admin_notification_state":      "pending",
			"admin_notification_claimed_at": 0,
			"updated_at":                    now,
		}).Error
}

func MarkAsyncImageOutputAvailable(taskID string, owner string, expiresAt int64, manual bool) error {
	updates := map[string]any{
		"output_availability":           AsyncImageOutputAvailable,
		"output_expires_at":             expiresAt,
		"completed_at":                  common.GetTimestamp(),
		"public_error_code":             "",
		"public_error_message":          "",
		"last_error":                    "",
		"admin_notification_state":      "none",
		"admin_notification_claimed_at": 0,
		"lease_owner":                   "",
		"lease_expires_at":              0,
		"updated_at":                    common.GetTimestamp(),
	}
	if manual {
		updates["manually_recovered_at"] = common.GetTimestamp()
	}
	result := DB.Model(&AsyncImageTask{}).
		Where("task_id = ? AND lease_owner = ? AND status = ? AND billing_status = ?", taskID, owner, AsyncImageStatusSucceeded, AsyncImageBillingSettled).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrAsyncImageLeaseLost
	}
	return nil
}

func ScheduleAsyncImageArchiveRetry(taskID string, owner string, attempts int, nextAttemptAt int64, errorMessage string) error {
	result := DB.Model(&AsyncImageTask{}).
		Where("task_id = ? AND lease_owner = ? AND status = ? AND billing_status = ?", taskID, owner, AsyncImageStatusSucceeded, AsyncImageBillingSettled).
		Updates(map[string]any{
			"output_availability": AsyncImageOutputArchiving,
			"archive_attempts":    attempts,
			"next_attempt_at":     nextAttemptAt,
			"last_error":          errorMessage,
			"lease_owner":         "",
			"lease_expires_at":    0,
			"updated_at":          common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrAsyncImageLeaseLost
	}
	return nil
}

func ScheduleAsyncImageGenerationRetry(taskID string, owner string, nextAttemptAt int64, errorMessage string) error {
	result := DB.Model(&AsyncImageTask{}).
		Where("task_id = ? AND lease_owner = ? AND status IN ? AND output_availability = ? AND billing_status = ?", taskID, owner, []string{AsyncImageStatusQueued, AsyncImageStatusRunning}, AsyncImageOutputPending, AsyncImageBillingReserved).
		Updates(map[string]any{
			"next_attempt_at":  nextAttemptAt,
			"last_error":       errorMessage,
			"lease_owner":      "",
			"lease_expires_at": 0,
			"updated_at":       common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrAsyncImageLeaseLost
	}
	return nil
}

func FailAsyncImageArchive(taskID string, owner string, attempts int, errorCode string, errorMessage string) error {
	now := common.GetTimestamp()
	result := DB.Model(&AsyncImageTask{}).
		Where("task_id = ? AND lease_owner = ? AND status = ? AND billing_status = ?", taskID, owner, AsyncImageStatusSucceeded, AsyncImageBillingSettled).
		Updates(map[string]any{
			"output_availability":           AsyncImageOutputFailed,
			"archive_attempts":              attempts,
			"completed_at":                  now,
			"public_error_code":             errorCode,
			"public_error_message":          errorMessage,
			"last_error":                    errorMessage,
			"admin_notification_state":      "pending",
			"admin_notification_claimed_at": 0,
			"lease_owner":                   "",
			"lease_expires_at":              0,
			"updated_at":                    now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrAsyncImageLeaseLost
	}
	return nil
}

func HasRunnableAsyncImageTasks() bool {
	now := common.GetTimestamp()
	var count int64
	err := DB.Model(&AsyncImageTask{}).
		Where("next_attempt_at <= ? AND (lease_owner = '' OR lease_expires_at <= ?)", now, now).
		Where("(status IN ? AND output_availability = ? AND billing_status = ?) OR (status = ? AND output_availability = ? AND billing_status = ?)",
			[]string{AsyncImageStatusQueued, AsyncImageStatusRunning}, AsyncImageOutputPending, AsyncImageBillingReserved,
			AsyncImageStatusSucceeded, AsyncImageOutputArchiving, AsyncImageBillingSettled).
		Count(&count).Error
	return err == nil && count > 0
}

type AsyncImageTaskFilter struct {
	UserID             int
	TaskID             string
	Model              string
	Status             string
	OutputAvailability string
	BillingStatus      string
	CreatedAfter       int64
	CreatedBefore      int64
	Page               int
	PageSize           int
}

func ListAsyncImageTasks(filter AsyncImageTaskFilter) ([]AsyncImageTask, int64, error) {
	query := DB.Model(&AsyncImageTask{})
	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.TaskID != "" {
		query = query.Where("task_id LIKE ?", "%"+filter.TaskID+"%")
	}
	if filter.Model != "" {
		query = query.Where("origin_model_name LIKE ?", "%"+filter.Model+"%")
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.OutputAvailability != "" {
		query = query.Where("output_availability = ?", filter.OutputAvailability)
	}
	if filter.BillingStatus != "" {
		query = query.Where("billing_status = ?", filter.BillingStatus)
	}
	if filter.CreatedAfter > 0 {
		query = query.Where("created_at >= ?", filter.CreatedAfter)
	}
	if filter.CreatedBefore > 0 {
		query = query.Where("created_at <= ?", filter.CreatedBefore)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	var tasks []AsyncImageTask
	err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks).Error
	return tasks, total, err
}

func ResetAsyncImageArchiveForRetry(taskID string, retryDeadlineAt int64) (bool, error) {
	now := common.GetTimestamp()
	result := DB.Model(&AsyncImageTask{}).
		Where("task_id = ? AND status = ? AND billing_status = ? AND output_availability = ?", taskID, AsyncImageStatusSucceeded, AsyncImageBillingSettled, AsyncImageOutputFailed).
		Updates(map[string]any{
			"output_availability":           AsyncImageOutputArchiving,
			"archive_attempts":              0,
			"archive_retry_deadline_at":     retryDeadlineAt,
			"next_attempt_at":               now,
			"public_error_code":             "",
			"public_error_message":          "",
			"last_error":                    "",
			"admin_notification_state":      "none",
			"admin_notification_claimed_at": 0,
			"lease_owner":                   "",
			"lease_expires_at":              0,
			"updated_at":                    now,
		})
	return result.RowsAffected == 1, result.Error
}

func ListFailedAsyncImageTaskIDs(afterID int64, maxID int64, limit int) ([]AsyncImageTask, error) {
	if limit <= 0 {
		limit = 100
	}
	var tasks []AsyncImageTask
	err := DB.Where("id > ? AND id <= ? AND status = ? AND billing_status = ? AND output_availability = ?", afterID, maxID, AsyncImageStatusSucceeded, AsyncImageBillingSettled, AsyncImageOutputFailed).
		Order("id asc").Limit(limit).Find(&tasks).Error
	return tasks, err
}

func MaxAsyncImageTaskID() (int64, error) {
	var maxID int64
	err := DB.Model(&AsyncImageTask{}).Select("COALESCE(MAX(id), 0)").Scan(&maxID).Error
	return maxID, err
}

func ClaimExpiredStorageObjects(limit int) ([]StorageObject, error) {
	if limit <= 0 {
		limit = 100
	}
	now := common.GetTimestamp()
	var candidates []StorageObject
	if err := DB.Where("business_id = ? AND ((status = ? AND expires_at > 0 AND expires_at <= ?) OR status = ?)", StorageObjectBusinessAsyncImages, StorageObjectStatusAvailable, now, StorageObjectStatusDeletePending).
		Order("id asc").Limit(limit * 2).Find(&candidates).Error; err != nil {
		return nil, err
	}
	claimed := make([]StorageObject, 0, limit)
	for _, object := range candidates {
		if object.Status == StorageObjectStatusAvailable {
			result := DB.Model(&StorageObject{}).
				Where("id = ? AND status = ? AND expires_at <= ?", object.ID, StorageObjectStatusAvailable, now).
				Updates(map[string]any{"status": StorageObjectStatusDeletePending, "updated_at": now})
			if result.Error != nil {
				return claimed, result.Error
			}
			if result.RowsAffected != 1 {
				continue
			}
			object.Status = StorageObjectStatusDeletePending
		}
		claimed = append(claimed, object)
		if len(claimed) >= limit {
			break
		}
	}
	return claimed, nil
}

func MarkStorageObjectDeleted(id int64) error {
	now := common.GetTimestamp()
	return DB.Model(&StorageObject{}).
		Where("id = ? AND status = ?", id, StorageObjectStatusDeletePending).
		Updates(map[string]any{
			"status":     StorageObjectStatusDeleted,
			"deleted_at": now,
			"last_error": "",
			"updated_at": now,
		}).Error
}

func MarkStorageObjectDeleteFailed(id int64, errorMessage string) error {
	return DB.Model(&StorageObject{}).
		Where("id = ? AND status = ?", id, StorageObjectStatusDeletePending).
		Updates(map[string]any{
			"delete_attempts": gorm.Expr("delete_attempts + 1"),
			"last_error":      errorMessage,
			"updated_at":      common.GetTimestamp(),
		}).Error
}

func ListStagingDeletePending(limit int) ([]StorageObject, error) {
	if limit <= 0 {
		limit = 100
	}
	var objects []StorageObject
	err := DB.Where("business_id = ? AND staging_status = ?", StorageObjectBusinessAsyncImages, StorageStagingDeletePending).
		Order("id asc").Limit(limit).Find(&objects).Error
	return objects, err
}

func HasStorageObjectForStagingPath(relativePath string) (bool, error) {
	var count int64
	err := DB.Model(&StorageObject{}).
		Where("business_id = ? AND staging_relative_path = ?", StorageObjectBusinessAsyncImages, relativePath).
		Count(&count).Error
	return count > 0, err
}

func MarkStagingObjectDeleted(id int64) error {
	now := common.GetTimestamp()
	return DB.Model(&StorageObject{}).
		Where("id = ? AND staging_status = ?", id, StorageStagingDeletePending).
		Updates(map[string]any{"staging_status": StorageStagingDeleted, "staging_deleted_at": now, "updated_at": now}).Error
}

func ClaimPendingAsyncImageNotifications(limit int) ([]AsyncImageTask, error) {
	if limit <= 0 {
		limit = 20
	}
	now := common.GetTimestamp()
	staleBefore := now - 300
	var candidates []AsyncImageTask
	if err := DB.Where("admin_notification_state = ? OR (admin_notification_state = ? AND admin_notification_claimed_at <= ?)", "pending", "claimed", staleBefore).
		Order("id asc").Limit(limit * 2).Find(&candidates).Error; err != nil {
		return nil, err
	}
	claimed := make([]AsyncImageTask, 0, limit)
	for _, task := range candidates {
		result := DB.Model(&AsyncImageTask{}).
			Where("id = ? AND (admin_notification_state = ? OR (admin_notification_state = ? AND admin_notification_claimed_at <= ?))", task.ID, "pending", "claimed", staleBefore).
			Updates(map[string]any{"admin_notification_state": "claimed", "admin_notification_claimed_at": now, "updated_at": now})
		if result.Error != nil {
			return claimed, result.Error
		}
		if result.RowsAffected == 1 {
			task.AdminNotificationState = "claimed"
			claimed = append(claimed, task)
		}
		if len(claimed) >= limit {
			break
		}
	}
	return claimed, nil
}

func MarkAsyncImageNotificationSent(taskID string) error {
	return DB.Model(&AsyncImageTask{}).
		Where("task_id = ? AND admin_notification_state = ?", taskID, "claimed").
		Updates(map[string]any{"admin_notification_state": "sent", "updated_at": common.GetTimestamp()}).Error
}

func ValidateAsyncImageTaskState(task *AsyncImageTask) error {
	if task == nil || task.TaskID == "" {
		return errors.New("invalid async image task")
	}
	if task.BillingStatus == AsyncImageBillingSettled && task.Status == AsyncImageStatusFailed {
		return fmt.Errorf("settled task cannot have failed generation status: %s", task.TaskID)
	}
	return nil
}

func SettleAsyncImageBilling(taskID string, actualQuota int) error {
	if actualQuota < 0 {
		return errors.New("actual quota cannot be negative")
	}
	var walletDelta int
	var tokenDelta int
	var userID int
	var tokenKey string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var task AsyncImageTask
		if err := lockForUpdate(tx).Where("task_id = ?", taskID).First(&task).Error; err != nil {
			return err
		}
		if task.BillingStatus == AsyncImageBillingSettled {
			return nil
		}
		if task.BillingStatus == AsyncImageBillingRefunded {
			return errors.New("refunded async image task cannot be settled")
		}
		fundingDelta := actualQuota - task.ReservedQuota
		if task.BillingSource != "subscription" || task.SubscriptionID <= 0 {
			walletDelta = fundingDelta
		}
		tokenActual := actualQuota
		if task.TokenUnlimited {
			tokenActual = 0
		}
		tokenDelta = tokenActual - task.TokenReservedQuota
		if err := adjustAsyncImageFundingTx(tx, &task, fundingDelta); err != nil {
			return err
		}
		if err := adjustAsyncImageTokenTx(tx, &task, tokenDelta, &tokenKey); err != nil {
			return err
		}
		now := common.GetTimestamp()
		userID = task.UserID
		return tx.Model(&AsyncImageTask{}).Where("id = ? AND billing_status = ?", task.ID, AsyncImageBillingReserved).
			Updates(map[string]any{
				"actual_quota":         actualQuota,
				"billing_status":       AsyncImageBillingSettled,
				"billing_finalized_at": now,
				"updated_at":           now,
			}).Error
	})
	if err == nil {
		syncAsyncImageBillingCaches(userID, tokenKey, walletDelta, tokenDelta)
	}
	return err
}

func RefundAsyncImageBilling(taskID string, errorCode string, errorMessage string) error {
	var walletDelta int
	var tokenDelta int
	var userID int
	var tokenKey string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var task AsyncImageTask
		if err := lockForUpdate(tx).Where("task_id = ?", taskID).First(&task).Error; err != nil {
			return err
		}
		if task.BillingStatus == AsyncImageBillingRefunded {
			return nil
		}
		if task.BillingStatus == AsyncImageBillingSettled {
			return errors.New("settled async image task cannot be refunded")
		}
		fundingDelta := -task.ReservedQuota
		if task.BillingSource != "subscription" || task.SubscriptionID <= 0 {
			walletDelta = fundingDelta
		}
		tokenDelta = -task.TokenReservedQuota
		if err := adjustAsyncImageFundingTx(tx, &task, fundingDelta); err != nil {
			return err
		}
		if task.BillingSource == "subscription" && task.SubscriptionID > 0 {
			if err := tx.Model(&SubscriptionPreConsumeRecord{}).
				Where("request_id = ? AND status = ?", task.TaskID, "consumed").
				Updates(map[string]any{"status": "refunded", "updated_at": common.GetTimestamp()}).Error; err != nil {
				return err
			}
		}
		if err := adjustAsyncImageTokenTx(tx, &task, tokenDelta, &tokenKey); err != nil {
			return err
		}
		now := common.GetTimestamp()
		userID = task.UserID
		return tx.Model(&AsyncImageTask{}).Where("id = ? AND billing_status = ?", task.ID, AsyncImageBillingReserved).
			Updates(map[string]any{
				"status":                   AsyncImageStatusFailed,
				"output_availability":      AsyncImageOutputFailed,
				"billing_status":           AsyncImageBillingRefunded,
				"billing_finalized_at":     now,
				"completed_at":             now,
				"public_error_code":        errorCode,
				"public_error_message":     errorMessage,
				"last_error":               errorMessage,
				"request_payload":          "",
				"admin_notification_state": "none",
				"lease_owner":              "",
				"lease_expires_at":         0,
				"updated_at":               now,
			}).Error
	})
	if err == nil {
		syncAsyncImageBillingCaches(userID, tokenKey, walletDelta, tokenDelta)
	}
	return err
}

func ReserveAsyncImageBilling(taskID string, targetQuota int) error {
	if targetQuota < 0 {
		return errors.New("target quota cannot be negative")
	}
	var walletDelta int
	var tokenDelta int
	var userID int
	var tokenKey string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var task AsyncImageTask
		if err := lockForUpdate(tx).Where("task_id = ?", taskID).First(&task).Error; err != nil {
			return err
		}
		if task.BillingStatus != AsyncImageBillingReserved || targetQuota <= task.ReservedQuota {
			return nil
		}
		fundingDelta := targetQuota - task.ReservedQuota
		if task.BillingSource != "subscription" || task.SubscriptionID <= 0 {
			walletDelta = fundingDelta
		}
		tokenTarget := targetQuota
		if task.TokenUnlimited {
			tokenTarget = 0
		}
		tokenDelta = tokenTarget - task.TokenReservedQuota
		if err := adjustAsyncImageFundingTx(tx, &task, fundingDelta); err != nil {
			return err
		}
		if err := adjustAsyncImageTokenTx(tx, &task, tokenDelta, &tokenKey); err != nil {
			return err
		}
		userID = task.UserID
		return tx.Model(&AsyncImageTask{}).Where("id = ?", task.ID).Updates(map[string]any{
			"reserved_quota":       targetQuota,
			"token_reserved_quota": tokenTarget,
			"updated_at":           common.GetTimestamp(),
		}).Error
	})
	if err == nil {
		syncAsyncImageBillingCaches(userID, tokenKey, walletDelta, tokenDelta)
	}
	return err
}

func adjustAsyncImageFundingTx(tx *gorm.DB, task *AsyncImageTask, delta int) error {
	if delta == 0 {
		return nil
	}
	if task.BillingSource == "subscription" && task.SubscriptionID > 0 {
		var subscription UserSubscription
		if err := lockForUpdate(tx).Where("id = ?", task.SubscriptionID).First(&subscription).Error; err != nil {
			return err
		}
		newUsed := subscription.AmountUsed + int64(delta)
		if newUsed < 0 {
			return errors.New("subscription quota refund exceeds used amount")
		}
		if subscription.AmountTotal > 0 && newUsed > subscription.AmountTotal {
			return fmt.Errorf("subscription quota insufficient, need=%d", delta)
		}
		return tx.Model(&UserSubscription{}).Where("id = ?", subscription.Id).Update("amount_used", newUsed).Error
	}
	var user User
	if err := lockForUpdate(tx).Where("id = ?", task.UserID).First(&user).Error; err != nil {
		return err
	}
	newQuota := user.Quota - delta
	if newQuota < 0 {
		return fmt.Errorf("user quota insufficient, need=%d", delta)
	}
	return tx.Model(&User{}).Where("id = ?", user.Id).Update("quota", newQuota).Error
}

func adjustAsyncImageTokenTx(tx *gorm.DB, task *AsyncImageTask, delta int, tokenKey *string) error {
	if delta == 0 || task.TokenUnlimited || task.TokenID <= 0 {
		return nil
	}
	var token Token
	if err := lockForUpdate(tx).Where("id = ?", task.TokenID).First(&token).Error; err != nil {
		return err
	}
	newRemain := token.RemainQuota - delta
	if newRemain < 0 {
		return fmt.Errorf("token quota insufficient, need=%d", delta)
	}
	newUsed := token.UsedQuota + delta
	if newUsed < 0 {
		return errors.New("token quota refund exceeds used amount")
	}
	*tokenKey = token.Key
	return tx.Model(&Token{}).Where("id = ?", token.Id).Updates(map[string]any{
		"remain_quota":  newRemain,
		"used_quota":    newUsed,
		"accessed_time": common.GetTimestamp(),
	}).Error
}

func syncAsyncImageBillingCaches(userID int, tokenKey string, walletDelta int, tokenDelta int) {
	if common.RedisEnabled && userID > 0 && walletDelta != 0 {
		_ = cacheDecrUserQuota(userID, int64(walletDelta))
	}
	if common.RedisEnabled && tokenKey != "" && tokenDelta != 0 {
		_ = cacheDecrTokenQuota(tokenKey, int64(tokenDelta))
	}
}
