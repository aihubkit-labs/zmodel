package service

import (
	"context"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/storage_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var onePixelPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41,
	0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0xf0, 0x1f,
	0x00, 0x05, 0x00, 0x01, 0xff, 0x89, 0x99, 0x3d, 0x1d,
	0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
	0x42, 0x60, 0x82,
}

func setAsyncImageStagingDirectory(t *testing.T, directory string) {
	t.Helper()
	common.OptionMapRWMutex.Lock()
	originalOptions := common.OptionMap
	options := make(map[string]string, len(originalOptions)+1)
	for key, value := range originalOptions {
		options[key] = value
	}
	options[storage_setting.OptionStagingDirectory] = directory
	common.OptionMap = options
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptions
		common.OptionMapRWMutex.Unlock()
	})
}

func TestAsyncImageStagingUsesAdminConfiguredDirectory(t *testing.T) {
	persistedRoot := filepath.Join(t.TempDir(), "persisted")
	setAsyncImageStagingDirectory(t, persistedRoot)

	require.NoError(t, CheckAsyncImageStaging())
	assert.DirExists(t, persistedRoot)
	entries, err := os.ReadDir(persistedRoot)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestCheckAsyncImageStagingDirectoryProbesAdminPath(t *testing.T) {
	stagingRoot := filepath.Join(t.TempDir(), "admin-configured")

	require.NoError(t, CheckAsyncImageStagingDirectory(stagingRoot))
	assert.DirExists(t, stagingRoot)
	entries, err := os.ReadDir(stagingRoot)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestStageAsyncImageResponseNormalizesBase64AndDataURI(t *testing.T) {
	originalLimit := constant.MaxFileDownloadMB
	constant.MaxFileDownloadMB = 1
	t.Cleanup(func() { constant.MaxFileDownloadMB = originalLimit })
	root := t.TempDir()
	setAsyncImageStagingDirectory(t, root)
	encoded := base64.StdEncoding.EncodeToString(onePixelPNG)
	response := &dto.ImageResponse{Data: []dto.ImageData{
		{B64Json: encoded, RevisedPrompt: "first"},
		{B64Json: "data:image/png;base64," + encoded, RevisedPrompt: "second"},
	}}

	manifest, sourceKind, err := StageAsyncImageResponse(context.Background(), 42, "task_stage_formats", response)
	require.NoError(t, err)
	assert.Equal(t, "mixed", sourceKind)
	require.Len(t, manifest, 2)
	assert.Equal(t, "base64", manifest[0].SourceType)
	assert.Equal(t, "data_uri", manifest[1].SourceType)
	assert.Equal(t, "image/png", manifest[0].MimeType)
	assert.Equal(t, "png", manifest[0].Extension)
	assert.Equal(t, int64(len(onePixelPNG)), manifest[0].SizeBytes)
	assert.NotEmpty(t, manifest[0].SHA256)
	assert.Equal(t, "first", manifest[0].RevisedPrompt)

	for _, item := range manifest {
		file, err := ReadStagedImage(item)
		require.NoError(t, err)
		contents, err := io.ReadAll(file)
		require.NoError(t, err)
		require.NoError(t, file.Close())
		assert.Equal(t, onePixelPNG, contents)
		assert.False(t, filepath.IsAbs(item.StagingRelativePath))
	}
}

func TestReadStagedImageRejectsTampering(t *testing.T) {
	originalLimit := constant.MaxFileDownloadMB
	constant.MaxFileDownloadMB = 1
	t.Cleanup(func() { constant.MaxFileDownloadMB = originalLimit })
	root := t.TempDir()
	setAsyncImageStagingDirectory(t, root)
	encoded := base64.StdEncoding.EncodeToString(onePixelPNG)
	manifest, _, err := StageAsyncImageResponse(context.Background(), 42, "task_stage_integrity", &dto.ImageResponse{
		Data: []dto.ImageData{{B64Json: encoded}},
	})
	require.NoError(t, err)
	require.Len(t, manifest, 1)

	path := filepath.Join(root, manifest[0].StagingRelativePath)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	require.NoError(t, err)
	_, err = file.Write([]byte("tampered"))
	require.NoError(t, err)
	require.NoError(t, file.Close())

	_, err = ReadStagedImage(manifest[0])
	require.ErrorContains(t, err, "integrity check failed")
}

func TestStageAsyncImageResponseRejectsInvalidImageBytes(t *testing.T) {
	originalLimit := constant.MaxFileDownloadMB
	constant.MaxFileDownloadMB = 1
	t.Cleanup(func() { constant.MaxFileDownloadMB = originalLimit })
	setAsyncImageStagingDirectory(t, t.TempDir())
	encoded := base64.StdEncoding.EncodeToString([]byte("not an image payload"))

	_, _, err := StageAsyncImageResponse(context.Background(), 42, "task_stage_invalid", &dto.ImageResponse{
		Data: []dto.ImageData{{B64Json: encoded}},
	})
	require.Error(t, err)
	assert.Equal(t, AsyncImageStageInvalid, AsyncImageStageErrorKindOf(err))
}

func TestCleanupOrphanedAsyncImageStagingPreservesReferencedFiles(t *testing.T) {
	truncate(t)
	root := t.TempDir()
	setAsyncImageStagingDirectory(t, root)
	oldTime := time.Now().Add(-2 * time.Hour)

	referencedRelative := filepath.Join("42", "2026", "07", "task_referenced", "0.img")
	referencedPath := filepath.Join(root, referencedRelative)
	require.NoError(t, os.MkdirAll(filepath.Dir(referencedPath), 0o700))
	require.NoError(t, os.WriteFile(referencedPath, onePixelPNG, 0o600))
	require.NoError(t, os.Chtimes(referencedPath, oldTime, oldTime))
	require.NoError(t, model.DB.Create(&model.StorageObject{
		BusinessID:          model.StorageObjectBusinessAsyncImages,
		ResourceID:          "task_referenced",
		ObjectIndex:         0,
		StagingRelativePath: referencedRelative,
		StagingStatus:       model.StorageStagingAvailable,
	}).Error)

	orphanPath := filepath.Join(root, "42", "2026", "07", "task_orphan", "0.img")
	require.NoError(t, os.MkdirAll(filepath.Dir(orphanPath), 0o700))
	require.NoError(t, os.WriteFile(orphanPath, onePixelPNG, 0o600))
	require.NoError(t, os.Chtimes(orphanPath, oldTime, oldTime))
	temporaryPath := filepath.Join(root, ".async-image-orphan.tmp")
	require.NoError(t, os.WriteFile(temporaryPath, []byte("partial"), 0o600))
	require.NoError(t, os.Chtimes(temporaryPath, oldTime, oldTime))

	deleted, err := CleanupOrphanedAsyncImageStaging(context.Background(), time.Hour, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, deleted)
	assert.FileExists(t, referencedPath)
	assert.NoFileExists(t, orphanPath)
	assert.NoFileExists(t, temporaryPath)
}
