package service

import (
	"bufio"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

type VideoStagedFile struct {
	RelativePath string
	SizeBytes    int64
	MimeType     string
	Extension    string
	SHA256       string
}

func CheckVideoStagingDirectory(directory string) error {
	root, err := validatedVideoStagingRoot(directory)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	probeName, err := common.GenerateRandomCharsKey(24)
	if err != nil {
		return err
	}
	temporaryPath := filepath.Join(root, ".probe-"+probeName+".tmp")
	finalPath := filepath.Join(root, ".probe-"+probeName)
	file, err := os.OpenFile(temporaryPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(temporaryPath)
		_ = os.Remove(finalPath)
	}()
	if _, err := file.Write([]byte("ok")); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, finalPath); err != nil {
		return err
	}
	content, err := os.ReadFile(finalPath)
	if err != nil {
		return err
	}
	if string(content) != "ok" {
		return errors.New("video staging read verification failed")
	}
	if err := os.Remove(finalPath); err != nil {
		return err
	}
	return syncVideoStagingDirectory(root)
}

func StageVideoFile(
	directory string,
	relativePath string,
	sourceURL string,
	mimeType string,
	reader io.Reader,
) (VideoStagedFile, error) {
	root, err := validatedVideoStagingRoot(directory)
	if err != nil {
		return VideoStagedFile{}, err
	}
	targetPath, err := safeVideoStagingPath(root, relativePath)
	if err != nil {
		return VideoStagedFile{}, err
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		return VideoStagedFile{}, err
	}
	temporary, err := os.CreateTemp(filepath.Dir(targetPath), ".video-stage-*.tmp")
	if err != nil {
		return VideoStagedFile{}, err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return VideoStagedFile{}, err
	}

	maxBytes := int64(constant.MaxFileDownloadMB) * 1024 * 1024
	if maxBytes <= 0 {
		maxBytes = 64 * 1024 * 1024
	}
	hasher := sha256.New()
	buffered := bufio.NewWriterSize(io.MultiWriter(temporary, hasher), 64*1024)
	written, err := io.Copy(buffered, io.LimitReader(reader, maxBytes+1))
	if err == nil {
		err = buffered.Flush()
	}
	if err != nil {
		_ = temporary.Close()
		return VideoStagedFile{}, err
	}
	if written == 0 {
		_ = temporary.Close()
		return VideoStagedFile{}, errors.New("video source is empty")
	}
	if written > maxBytes {
		_ = temporary.Close()
		return VideoStagedFile{}, fmt.Errorf("video exceeds maximum allowed size: %dMB", constant.MaxFileDownloadMB)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return VideoStagedFile{}, err
	}
	if mimeType == "" || mimeType == "application/octet-stream" {
		if _, err := temporary.Seek(0, io.SeekStart); err != nil {
			_ = temporary.Close()
			return VideoStagedFile{}, err
		}
		header := make([]byte, 512)
		read, readErr := temporary.Read(header)
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			_ = temporary.Close()
			return VideoStagedFile{}, readErr
		}
		mimeType = http.DetectContentType(header[:read])
	}
	if err := temporary.Close(); err != nil {
		return VideoStagedFile{}, err
	}
	if err := os.Rename(temporaryPath, targetPath); err != nil {
		return VideoStagedFile{}, err
	}
	if err := syncVideoStagingDirectory(filepath.Dir(targetPath)); err != nil {
		return VideoStagedFile{}, err
	}
	return VideoStagedFile{
		RelativePath: relativePath,
		SizeBytes:    written,
		MimeType:     mimeType,
		Extension:    stagedVideoExtension(sourceURL, mimeType),
		SHA256:       fmt.Sprintf("%x", hasher.Sum(nil)),
	}, nil
}

func OpenStagedVideo(directory string, staged VideoStagedFile) (*os.File, error) {
	root, err := validatedVideoStagingRoot(directory)
	if err != nil {
		return nil, err
	}
	path, err := safeVideoStagingPath(root, staged.RelativePath)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	hasher := sha256.New()
	written, err := io.Copy(hasher, file)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if written != staged.SizeBytes || fmt.Sprintf("%x", hasher.Sum(nil)) != staged.SHA256 {
		_ = file.Close()
		return nil, errors.New("staged video integrity check failed")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func DeleteStagedVideo(directory string, relativePath string) error {
	root, err := validatedVideoStagingRoot(directory)
	if err != nil {
		return err
	}
	path, err := safeVideoStagingPath(root, relativePath)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func CleanupOrphanedVideoStaging(
	ctx context.Context,
	directory string,
	businessID string,
	gracePeriod time.Duration,
	limit int,
) (int, error) {
	root, err := validatedVideoStagingRoot(directory)
	if err != nil {
		return 0, err
	}
	if limit <= 0 {
		limit = 100
	}
	if gracePeriod <= 0 {
		gracePeriod = time.Hour
	}
	cutoff := time.Now().Add(-gracePeriod)
	deleted := 0
	scanned := 0
	maxScanned := limit * 20
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return nil
		}
		scanned++
		if scanned > maxScanned || deleted >= limit {
			return fs.SkipAll
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.ModTime().After(cutoff) {
			return nil
		}
		name := entry.Name()
		isTemporary := strings.HasPrefix(name, ".video-stage-") && strings.HasSuffix(name, ".tmp")
		isProbe := strings.HasPrefix(name, ".probe-")
		if !isTemporary && !isProbe && filepath.Ext(name) != ".video" {
			return nil
		}
		if !isTemporary && !isProbe {
			relativePath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			referenced, err := model.HasVideoStorageStagingReference(businessID, relativePath)
			if err != nil {
				return err
			}
			if referenced {
				return nil
			}
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		deleted++
		return nil
	})
	return deleted, err
}

func validatedVideoStagingRoot(directory string) (string, error) {
	directory = strings.TrimSpace(directory)
	if directory == "" {
		return "", errors.New("video staging directory is not configured")
	}
	if !filepath.IsAbs(directory) {
		return "", errors.New("video staging directory must be absolute")
	}
	root, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	if filepath.Dir(root) == root {
		return "", errors.New("video staging directory cannot be a filesystem root")
	}
	return root, nil
}

func safeVideoStagingPath(root string, relativePath string) (string, error) {
	cleanRelative := filepath.Clean(relativePath)
	if cleanRelative == "." || filepath.IsAbs(cleanRelative) || strings.HasPrefix(cleanRelative, ".."+string(filepath.Separator)) || cleanRelative == ".." {
		return "", errors.New("invalid video staging path")
	}
	path := filepath.Join(root, cleanRelative)
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("video staging path escapes root")
	}
	return path, nil
}

func stagedVideoExtension(sourceURL string, mimeType string) string {
	switch strings.ToLower(mimeType) {
	case "video/webm":
		return "webm"
	case "video/quicktime":
		return "mov"
	case "video/x-matroska":
		return "mkv"
	case "video/mpeg":
		return "mpeg"
	case "video/mp4":
		return "mp4"
	}
	if parsed, err := url.Parse(sourceURL); err == nil {
		extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(parsed.Path)), ".")
		if extension != "" && len(extension) <= 8 {
			return extension
		}
	}
	return "mp4"
}

func syncVideoStagingDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
