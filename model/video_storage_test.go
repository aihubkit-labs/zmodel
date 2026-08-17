package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVideoStorageArchiveRetryLifecycle(t *testing.T) {
	truncateTables(t)
	now := common.GetTimestamp()
	object := &StorageObject{
		BusinessID: "test@videos", ResourceID: "task_video_retry_lifecycle", ObjectIndex: 0,
		Status: StorageObjectStatusUploading, ArchiveMaxAttempts: 3,
		ArchiveRetryDeadlineAt: now + 3600, StagingStatus: StorageStagingVideoAvailable,
	}
	require.NoError(t, DB.Create(object).Error)
	require.NoError(t, MarkVideoStorageArchiveUploading(
		object.ID, StorageObjectStatusUploading, "", "operation-1", now+60,
	))
	require.NoError(t, DB.First(object, object.ID).Error)
	assert.Equal(t, "operation-1", object.ArchiveOperationID)
	assert.Equal(t, now+60, object.ArchiveLeaseExpiresAt)

	require.NoError(t, MarkVideoStorageArchiveFailed(
		object.ID, 0, "operation-1", now+60, 3, now+3600, now+15, "s3 unavailable",
	))
	due, err := ListDueVideoStorageArchiveRetries("test@videos", now+15, 10)
	require.NoError(t, err)
	require.Len(t, due, 1)
	assert.Equal(t, 1, due[0].ArchiveAttempts)
	assert.Equal(t, "s3 unavailable", due[0].LastError)
	assert.Empty(t, due[0].ArchiveOperationID)
	assert.Zero(t, due[0].ArchiveLeaseExpiresAt)
	assert.True(t, HasDueVideoStorageArchiveRetries("test@videos", now+15))
	require.NoError(t, MarkVideoStorageArchiveUploading(
		object.ID, StorageObjectStatusFailed, "", "operation-2", now+120,
	))
	err = MarkVideoStorageArchiveAvailable(object.ID, "operation-1", now+60, "late-etag", now, now+3600)
	require.ErrorIs(t, err, ErrVideoStorageArchiveStateChanged)
	require.NoError(t, DB.First(object, object.ID).Error)
	assert.Equal(t, StorageObjectStatusUploading, object.Status)
	assert.Equal(t, "operation-2", object.ArchiveOperationID)

	require.NoError(t, ResetVideoStorageArchiveForRetry(object.ID, 5, now+7200))
	require.NoError(t, DB.First(object, object.ID).Error)
	assert.Equal(t, 0, object.ArchiveAttempts)
	assert.Equal(t, 5, object.ArchiveMaxAttempts)
	assert.Equal(t, now+7200, object.ArchiveRetryDeadlineAt)
	assert.GreaterOrEqual(t, object.ArchiveNextAttemptAt, now)
	assert.Empty(t, object.ArchiveOperationID)
	assert.Zero(t, object.ArchiveLeaseExpiresAt)

	require.NoError(t, StopVideoStorageArchiveRetries(object.ID, "source expired"))
	require.NoError(t, DB.First(object, object.ID).Error)
	assert.Equal(t, int64(0), object.ArchiveNextAttemptAt)
	assert.Equal(t, "source expired", object.LastError)
}
