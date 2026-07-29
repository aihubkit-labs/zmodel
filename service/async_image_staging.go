package service

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

const asyncImageStagingEnv = "ASYNC_IMAGE_STAGING_DIR"

type AsyncImageStageErrorKind string

const (
	AsyncImageStageInvalid        AsyncImageStageErrorKind = "invalid_response"
	AsyncImageStageSourceFetch    AsyncImageStageErrorKind = "source_fetch"
	AsyncImageStageInfrastructure AsyncImageStageErrorKind = "infrastructure"
)

type AsyncImageStageError struct {
	Kind AsyncImageStageErrorKind
	Err  error
}

func (err *AsyncImageStageError) Error() string { return err.Err.Error() }
func (err *AsyncImageStageError) Unwrap() error { return err.Err }

func AsyncImageStageErrorKindOf(err error) AsyncImageStageErrorKind {
	var stageErr *AsyncImageStageError
	if errors.As(err, &stageErr) {
		return stageErr.Kind
	}
	return AsyncImageStageInfrastructure
}

type AsyncImageManifestItem struct {
	Index               int    `json:"index"`
	SourceType          string `json:"source_type"`
	StagingRelativePath string `json:"staging_relative_path"`
	SizeBytes           int64  `json:"size_bytes"`
	MimeType            string `json:"mime_type"`
	Extension           string `json:"extension"`
	SHA256              string `json:"sha256"`
	RevisedPrompt       string `json:"revised_prompt,omitempty"`
}

func CheckAsyncImageStaging() error {
	root, err := asyncImageStagingRoot()
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
	cleanup := func() {
		_ = os.Remove(temporaryPath)
		_ = os.Remove(finalPath)
	}
	defer cleanup()
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
	data, err := os.ReadFile(finalPath)
	if err != nil {
		return err
	}
	if string(data) != "ok" {
		return errors.New("staging read verification failed")
	}
	return syncDirectory(root)
}

func StageAsyncImageResponse(ctx context.Context, userID int, taskID string, response *dto.ImageResponse) ([]AsyncImageManifestItem, string, error) {
	if response == nil || len(response.Data) == 0 || len(response.Data) > dto.MaxImageN {
		return nil, "none", &AsyncImageStageError{Kind: AsyncImageStageInvalid, Err: errors.New("invalid image response data")}
	}
	manifest := make([]AsyncImageManifestItem, 0, len(response.Data))
	sourceKinds := make(map[string]struct{})
	for index, image := range response.Data {
		attempts := 1
		if strings.TrimSpace(image.Url) != "" {
			attempts = 3
		}
		var item AsyncImageManifestItem
		var err error
		for attempt := 0; attempt < attempts; attempt++ {
			item, err = stageAsyncImageData(ctx, userID, taskID, index, image)
			if err == nil || AsyncImageStageErrorKindOf(err) != AsyncImageStageSourceFetch || ctx.Err() != nil {
				break
			}
		}
		if err != nil {
			return nil, sourceKindFromSet(sourceKinds), err
		}
		manifest = append(manifest, item)
		sourceKinds[item.SourceType] = struct{}{}
	}
	return manifest, sourceKindFromSet(sourceKinds), nil
}

func stageAsyncImageData(ctx context.Context, userID int, taskID string, index int, image dto.ImageData) (AsyncImageManifestItem, error) {
	var reader io.ReadCloser
	var sourceType string
	switch {
	case strings.TrimSpace(image.Url) != "":
		sourceType = "url"
		if err := ValidateSSRFProtectedFetchURL(image.Url); err != nil {
			return AsyncImageManifestItem{}, &AsyncImageStageError{Kind: AsyncImageStageSourceFetch, Err: err}
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, image.Url, nil)
		if err != nil {
			return AsyncImageManifestItem{}, &AsyncImageStageError{Kind: AsyncImageStageSourceFetch, Err: err}
		}
		response, err := GetSSRFProtectedHTTPClient().Do(request)
		if err != nil {
			return AsyncImageManifestItem{}, &AsyncImageStageError{Kind: AsyncImageStageSourceFetch, Err: err}
		}
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			_ = response.Body.Close()
			return AsyncImageManifestItem{}, &AsyncImageStageError{Kind: AsyncImageStageSourceFetch, Err: fmt.Errorf("image source returned status %d", response.StatusCode)}
		}
		reader = response.Body
	case strings.TrimSpace(image.B64Json) != "":
		encoded := strings.TrimSpace(image.B64Json)
		if strings.HasPrefix(strings.ToLower(encoded), "data:") {
			sourceType = "data_uri"
			comma := strings.IndexByte(encoded, ',')
			if comma < 0 || !strings.Contains(strings.ToLower(encoded[:comma]), ";base64") {
				return AsyncImageManifestItem{}, &AsyncImageStageError{Kind: AsyncImageStageInvalid, Err: errors.New("invalid image data URI")}
			}
			encoded = encoded[comma+1:]
		} else {
			sourceType = "base64"
		}
		reader = io.NopCloser(base64.NewDecoder(base64.StdEncoding, strings.NewReader(encoded)))
	default:
		return AsyncImageManifestItem{}, &AsyncImageStageError{Kind: AsyncImageStageInvalid, Err: errors.New("image response item has no URL or base64 data")}
	}
	defer reader.Close()
	relativePath := filepath.Join(fmt.Sprintf("%d", userID), time.Now().UTC().Format("2006/01"), taskID, fmt.Sprintf("%d.img", index))
	size, mimeType, extension, checksum, err := writeStagedImage(relativePath, reader)
	if err != nil {
		kind := AsyncImageStageErrorKindOf(err)
		if sourceType == "url" && kind == AsyncImageStageInvalid {
			kind = AsyncImageStageSourceFetch
		}
		return AsyncImageManifestItem{}, &AsyncImageStageError{Kind: kind, Err: err}
	}
	return AsyncImageManifestItem{
		Index:               index,
		SourceType:          sourceType,
		StagingRelativePath: relativePath,
		SizeBytes:           size,
		MimeType:            mimeType,
		Extension:           extension,
		SHA256:              checksum,
		RevisedPrompt:       image.RevisedPrompt,
	}, nil
}

func writeStagedImage(relativePath string, reader io.Reader) (int64, string, string, string, error) {
	root, err := asyncImageStagingRoot()
	if err != nil {
		return 0, "", "", "", &AsyncImageStageError{Kind: AsyncImageStageInfrastructure, Err: err}
	}
	targetPath, err := safeStagingPath(root, relativePath)
	if err != nil {
		return 0, "", "", "", &AsyncImageStageError{Kind: AsyncImageStageInfrastructure, Err: err}
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		return 0, "", "", "", &AsyncImageStageError{Kind: AsyncImageStageInfrastructure, Err: err}
	}
	temporary, err := os.CreateTemp(filepath.Dir(targetPath), ".async-image-*.tmp")
	if err != nil {
		return 0, "", "", "", &AsyncImageStageError{Kind: AsyncImageStageInfrastructure, Err: err}
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return 0, "", "", "", &AsyncImageStageError{Kind: AsyncImageStageInfrastructure, Err: err}
	}
	maxBytes := int64(constant.MaxFileDownloadMB) * 1024 * 1024
	hasher := sha256.New()
	buffered := bufio.NewWriterSize(io.MultiWriter(temporary, hasher), 64*1024)
	written, err := io.Copy(buffered, io.LimitReader(reader, maxBytes+1))
	if err == nil {
		err = buffered.Flush()
	}
	if err != nil {
		_ = temporary.Close()
		kind := AsyncImageStageInfrastructure
		var corrupt base64.CorruptInputError
		if errors.As(err, &corrupt) {
			kind = AsyncImageStageInvalid
		}
		return 0, "", "", "", &AsyncImageStageError{Kind: kind, Err: err}
	}
	if written == 0 {
		_ = temporary.Close()
		return 0, "", "", "", &AsyncImageStageError{Kind: AsyncImageStageInvalid, Err: errors.New("image content is empty")}
	}
	if written > maxBytes {
		_ = temporary.Close()
		return 0, "", "", "", &AsyncImageStageError{Kind: AsyncImageStageInvalid, Err: fmt.Errorf("image exceeds maximum allowed size: %dMB", constant.MaxFileDownloadMB)}
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return 0, "", "", "", &AsyncImageStageError{Kind: AsyncImageStageInfrastructure, Err: err}
	}
	if err := temporary.Close(); err != nil {
		return 0, "", "", "", &AsyncImageStageError{Kind: AsyncImageStageInfrastructure, Err: err}
	}
	content, err := os.ReadFile(temporaryPath)
	if err != nil {
		return 0, "", "", "", &AsyncImageStageError{Kind: AsyncImageStageInfrastructure, Err: err}
	}
	mimeType, extension, err := detectImageType(content)
	if err != nil {
		return 0, "", "", "", &AsyncImageStageError{Kind: AsyncImageStageInvalid, Err: err}
	}
	if err := os.Rename(temporaryPath, targetPath); err != nil {
		return 0, "", "", "", &AsyncImageStageError{Kind: AsyncImageStageInfrastructure, Err: err}
	}
	if err := syncDirectory(filepath.Dir(targetPath)); err != nil {
		return 0, "", "", "", &AsyncImageStageError{Kind: AsyncImageStageInfrastructure, Err: err}
	}
	return written, mimeType, extension, fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func DeleteStagedImage(relativePath string) error {
	root, err := asyncImageStagingRoot()
	if err != nil {
		return err
	}
	path, err := safeStagingPath(root, relativePath)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func CleanupOrphanedAsyncImageStaging(ctx context.Context, gracePeriod time.Duration, limit int) (int, error) {
	root, err := asyncImageStagingRoot()
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
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
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
		isTemporary := strings.HasPrefix(name, ".async-image-") && strings.HasSuffix(name, ".tmp")
		isProbe := strings.HasPrefix(name, ".probe-")
		if !isTemporary && !isProbe && filepath.Ext(name) != ".img" {
			return nil
		}
		if !isTemporary && !isProbe {
			relativePath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			referenced, err := model.HasStorageObjectForStagingPath(relativePath)
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

func ReadStagedImage(item AsyncImageManifestItem) (*os.File, error) {
	root, err := asyncImageStagingRoot()
	if err != nil {
		return nil, err
	}
	path, err := safeStagingPath(root, item.StagingRelativePath)
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
	if written != item.SizeBytes || fmt.Sprintf("%x", hasher.Sum(nil)) != item.SHA256 {
		_ = file.Close()
		return nil, errors.New("staged image integrity check failed")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, err
	}
	header := make([]byte, 512)
	n, readErr := file.Read(header)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		_ = file.Close()
		return nil, readErr
	}
	mimeType, extension, typeErr := detectImageType(header[:n])
	if typeErr != nil || mimeType != item.MimeType || extension != item.Extension {
		_ = file.Close()
		return nil, errors.New("staged image type verification failed")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func asyncImageStagingRoot() (string, error) {
	root := strings.TrimSpace(os.Getenv(asyncImageStagingEnv))
	if root == "" {
		return "", fmt.Errorf("%s is not configured", asyncImageStagingEnv)
	}
	return filepath.Abs(root)
}

func safeStagingPath(root string, relativePath string) (string, error) {
	cleanRelative := filepath.Clean(relativePath)
	if cleanRelative == "." || filepath.IsAbs(cleanRelative) || strings.HasPrefix(cleanRelative, ".."+string(filepath.Separator)) || cleanRelative == ".." {
		return "", errors.New("invalid staging path")
	}
	path := filepath.Join(root, cleanRelative)
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("staging path escapes root")
	}
	return path, nil
}

func detectImageType(content []byte) (string, string, error) {
	if len(content) < 12 {
		return "", "", errors.New("image content is too short")
	}
	mimeType := http.DetectContentType(content)
	switch mimeType {
	case "image/png":
		return mimeType, "png", nil
	case "image/jpeg":
		return mimeType, "jpg", nil
	case "image/gif":
		return mimeType, "gif", nil
	case "image/webp":
		return mimeType, "webp", nil
	case "image/bmp":
		return mimeType, "bmp", nil
	case "image/tiff":
		return mimeType, "tiff", nil
	}
	if bytes.Equal(content[4:8], []byte("ftyp")) && (bytes.Equal(content[8:12], []byte("avif")) || bytes.Equal(content[8:12], []byte("avis"))) {
		return "image/avif", "avif", nil
	}
	if bytes.HasPrefix(content, []byte{'I', 'I', 42, 0}) || bytes.HasPrefix(content, []byte{'M', 'M', 0, 42}) {
		return "image/tiff", "tiff", nil
	}
	return "", "", fmt.Errorf("unsupported image content type: %s", mimeType)
}

func sourceKindFromSet(kinds map[string]struct{}) string {
	if len(kinds) == 0 {
		return "none"
	}
	if len(kinds) > 1 {
		return "mixed"
	}
	for kind := range kinds {
		return kind
	}
	return "none"
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
