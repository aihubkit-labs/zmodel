package service

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVideoStagingPersistsVerifiesAndDeletesContent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "video-staging")
	relativePath := filepath.Join("videos", "42", "2026", "08", "08", "task_video", "original.video")
	content := "video-content"

	require.NoError(t, CheckVideoStagingDirectory(root))
	staged, err := StageVideoFile(root, relativePath, "https://upstream.example/video.mp4", "video/mp4", strings.NewReader(content))
	require.NoError(t, err)
	assert.Equal(t, relativePath, staged.RelativePath)
	assert.Equal(t, int64(len(content)), staged.SizeBytes)
	assert.Equal(t, "video/mp4", staged.MimeType)
	assert.Equal(t, "mp4", staged.Extension)
	assert.FileExists(t, filepath.Join(root, relativePath))

	file, err := OpenStagedVideo(root, staged)
	require.NoError(t, err)
	read, err := io.ReadAll(file)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	assert.Equal(t, []byte(content), read)

	require.NoError(t, os.WriteFile(filepath.Join(root, relativePath), []byte("damaged"), 0o600))
	_, err = OpenStagedVideo(root, staged)
	require.ErrorContains(t, err, "integrity check failed")
	require.NoError(t, DeleteStagedVideo(root, relativePath))
	assert.NoFileExists(t, filepath.Join(root, relativePath))
}

func TestVideoStagingRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	_, err := StageVideoFile(root, "../outside.video", "", "video/mp4", strings.NewReader("video"))
	require.ErrorContains(t, err, "invalid video staging path")
}
