package service

import (
	"context"
	"errors"
	"mime"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/objectstorage"
	"github.com/QuantumNous/new-api/setting/storage_setting"
)

type VideoArchiveSource struct {
	URL       string
	RemoteURL string
}

var ArchiveVideoTaskFunc func(context.Context, *model.Task, *model.Channel, VideoArchiveSource) error

func PresignVideoObject(ctx context.Context, taskID string, downloadName string) (string, error) {
	settings := storage_setting.GetVideoSettings()
	object, err := model.GetStorageObjectByBusinessID(settings.BusinessID, taskID, 0)
	if err != nil {
		return "", err
	}
	return presignVideoStorageObject(ctx, object, downloadName, settings)
}

func PresignVideoStorageObject(ctx context.Context, object *model.StorageObject, downloadName string) (string, error) {
	return presignVideoStorageObject(ctx, object, downloadName, storage_setting.GetVideoSettings())
}

func presignVideoStorageObject(
	ctx context.Context,
	object *model.StorageObject,
	downloadName string,
	settings storage_setting.VideoSettings,
) (string, error) {
	if object == nil {
		return "", errors.New("video object is missing")
	}
	now := common.GetTimestamp()
	if object.Status != model.StorageObjectStatusAvailable || object.ExpiresAt <= now {
		return "", errors.New("video object is not available")
	}
	if !settings.Configured() {
		return "", errors.New("video object storage is not configured")
	}
	storage, err := objectstorage.NewStorage(ctx, objectstorage.Config{
		Endpoint: object.Endpoint, Region: object.Region,
		AccessKey: settings.AccessKey, SecretAccessKey: settings.SecretAccessKey,
	})
	if err != nil {
		return "", err
	}
	disposition := ""
	if strings.TrimSpace(downloadName) != "" {
		disposition = mime.FormatMediaType("attachment", map[string]string{"filename": downloadName})
	}
	return storage.PresignGetObject(ctx, objectstorage.PresignGetObjectInput{
		Bucket: object.Bucket, Key: object.ObjectKey,
		Expires:             time.Duration(settings.PresignSeconds) * time.Second,
		ResponseContentType: object.MimeType, ResponseDisposition: disposition,
	})
}
