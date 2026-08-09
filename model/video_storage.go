package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
)

const (
	StorageStagingVideoPending       = "video_pending"
	StorageStagingVideoAvailable     = "video_available"
	StorageStagingVideoFailed        = "video_failed"
	StorageStagingVideoDeletePending = "video_delete_pending"
	StorageStagingVideoDeleted       = "video_deleted"
)

func CountVideoStorageStagingInUse(businessID string) (int64, error) {
	if businessID == "" {
		return 0, nil
	}
	var count int64
	err := DB.Model(&StorageObject{}).
		Where("business_id = ? AND staging_status IN ?", businessID, []string{
			StorageStagingVideoPending,
			StorageStagingVideoAvailable,
			StorageStagingVideoFailed,
			StorageStagingVideoDeletePending,
		}).Count(&count).Error
	return count, err
}

func InitializeVideoStorageArchive(id int64, maxAttempts int, retryDeadlineAt int64) error {
	result := DB.Model(&StorageObject{}).
		Where("id = ? AND (archive_max_attempts = ? OR archive_retry_deadline_at = ?)", id, 0, 0).
		Updates(map[string]any{
			"archive_max_attempts":      maxAttempts,
			"archive_retry_deadline_at": retryDeadlineAt,
			"archive_next_attempt_at":   0,
			"updated_at":                common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("video storage archive retry state changed")
	}
	return nil
}

func MarkVideoStorageArchiveFailed(
	id int64,
	previousAttempts int,
	maxAttempts int,
	retryDeadlineAt int64,
	nextAttemptAt int64,
	errorMessage string,
) error {
	result := DB.Model(&StorageObject{}).
		Where("id = ? AND archive_attempts = ? AND status <> ?", id, previousAttempts, StorageObjectStatusDeletePending).
		Updates(map[string]any{
			"status":                    StorageObjectStatusFailed,
			"archive_attempts":          previousAttempts + 1,
			"archive_max_attempts":      maxAttempts,
			"archive_retry_deadline_at": retryDeadlineAt,
			"archive_next_attempt_at":   nextAttemptAt,
			"last_error":                errorMessage,
			"updated_at":                common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("video storage archive state changed")
	}
	return nil
}

func ResetVideoStorageArchiveForRetry(id int64, maxAttempts int, retryDeadlineAt int64) error {
	now := common.GetTimestamp()
	result := DB.Model(&StorageObject{}).
		Where("id = ? AND status <> ?", id, StorageObjectStatusDeletePending).
		Updates(map[string]any{
			"status":                    StorageObjectStatusFailed,
			"archive_attempts":          0,
			"archive_max_attempts":      maxAttempts,
			"archive_retry_deadline_at": retryDeadlineAt,
			"archive_next_attempt_at":   now,
			"last_error":                "",
			"updated_at":                now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("video storage object is not retryable")
	}
	return nil
}

func ListDueVideoStorageArchiveRetries(businessID string, now int64, limit int) ([]StorageObject, error) {
	if businessID == "" {
		return []StorageObject{}, nil
	}
	if limit <= 0 {
		limit = 10
	}
	var objects []StorageObject
	err := DB.Where(
		"business_id = ? AND status = ? AND archive_next_attempt_at > ? AND archive_next_attempt_at <= ? AND archive_attempts < archive_max_attempts AND archive_retry_deadline_at > ?",
		businessID, StorageObjectStatusFailed, 0, now, now,
	).Order("archive_next_attempt_at asc").Limit(limit).Find(&objects).Error
	return objects, err
}

func HasDueVideoStorageArchiveRetries(businessID string, now int64) bool {
	if businessID == "" {
		return false
	}
	var count int64
	err := DB.Model(&StorageObject{}).Where(
		"business_id = ? AND status = ? AND archive_next_attempt_at > ? AND archive_next_attempt_at <= ? AND archive_attempts < archive_max_attempts AND archive_retry_deadline_at > ?",
		businessID, StorageObjectStatusFailed, 0, now, now,
	).Limit(1).Count(&count).Error
	return err == nil && count > 0
}

func StopVideoStorageArchiveRetries(id int64, errorMessage string) error {
	return DB.Model(&StorageObject{}).
		Where("id = ? AND status <> ?", id, StorageObjectStatusDeletePending).
		Updates(map[string]any{
			"status":                  StorageObjectStatusFailed,
			"archive_next_attempt_at": 0,
			"last_error":              errorMessage,
			"updated_at":              common.GetTimestamp(),
		}).Error
}

func MarkVideoStorageStaged(
	id int64,
	relativePath string,
	sizeBytes int64,
	mimeType string,
	extension string,
	checksum string,
) error {
	now := common.GetTimestamp()
	return DB.Model(&StorageObject{}).Where("id = ?", id).Updates(map[string]any{
		"staging_relative_path": relativePath,
		"staging_status":        StorageStagingVideoAvailable,
		"staging_size_bytes":    sizeBytes,
		"staging_sha256":        checksum,
		"mime_type":             mimeType,
		"extension":             extension,
		"staged_at":             now,
		"staging_deleted_at":    0,
		"updated_at":            now,
	}).Error
}

func MarkVideoStorageStagingFailed(id int64) error {
	return DB.Model(&StorageObject{}).Where("id = ?", id).Updates(map[string]any{
		"staging_status": StorageStagingVideoFailed,
		"updated_at":     common.GetTimestamp(),
	}).Error
}

func MarkVideoStorageStagingDeletePending(id int64) error {
	result := DB.Model(&StorageObject{}).
		Where("id = ? AND staging_status = ?", id, StorageStagingVideoAvailable).
		Updates(map[string]any{
			"staging_status": StorageStagingVideoDeletePending,
			"updated_at":     common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("video staging object is not deletable")
	}
	return nil
}

func MarkVideoStorageStagingDeleted(id int64) error {
	now := common.GetTimestamp()
	return DB.Model(&StorageObject{}).
		Where("id = ? AND staging_status = ?", id, StorageStagingVideoDeletePending).
		Updates(map[string]any{
			"staging_status":     StorageStagingVideoDeleted,
			"staging_deleted_at": now,
			"updated_at":         now,
		}).Error
}

func ListVideoStorageStagingCleanupCandidates(businessID string, limit int) ([]StorageObject, error) {
	if limit <= 0 {
		limit = 100
	}
	var objects []StorageObject
	err := DB.Where(
		"business_id = ? AND (staging_status = ? OR (staging_status = ? AND status IN ?))",
		businessID,
		StorageStagingVideoDeletePending,
		StorageStagingVideoAvailable,
		[]string{StorageObjectStatusAvailable, StorageObjectStatusDeleted},
	).
		Order("id asc").Limit(limit).Find(&objects).Error
	return objects, err
}

func HasVideoStorageStagingReference(businessID string, relativePath string) (bool, error) {
	var count int64
	err := DB.Model(&StorageObject{}).
		Where("business_id = ? AND staging_relative_path = ?", businessID, relativePath).
		Count(&count).Error
	return count > 0, err
}
