package storage_setting

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	OptionVideoS3Endpoint                = "VideoObjectStorageS3Endpoint"
	OptionVideoS3Region                  = "VideoObjectStorageS3Region"
	OptionVideoS3Bucket                  = "VideoObjectStorageS3Bucket"
	OptionVideoS3AccessKey               = "VideoObjectStorageS3AccessKey"
	OptionVideoS3SecretAccessKey         = "VideoObjectStorageS3SecretAccessKey"
	OptionVideoS3KeyPrefix               = "VideoObjectStorageS3KeyPrefix"
	OptionVideoBusinessID                = "VideoObjectStorageBusinessID"
	OptionVideoStagingDirectory          = "VideoObjectStorageStagingDirectory"
	OptionVideoRetentionSeconds          = "VideoObjectStorageRetentionSeconds"
	OptionVideoPresignSeconds            = "VideoObjectStoragePresignSeconds"
	OptionVideoArchiveTimeoutSeconds     = "VideoObjectStorageArchiveTimeoutSeconds"
	OptionVideoArchiveMaxAttempts        = "VideoObjectStorageArchiveMaxAttempts"
	OptionVideoArchiveRetryWindowSeconds = "VideoObjectStorageArchiveRetryWindowSeconds"
	OptionVideoCleanupInterval           = "VideoObjectStorageCleanupIntervalSeconds"
)

type VideoSettings struct {
	Endpoint                  string `json:"endpoint"`
	Region                    string `json:"region"`
	Bucket                    string `json:"bucket"`
	AccessKey                 string `json:"access_key"`
	SecretAccessKey           string `json:"-"`
	S3KeyPrefix               string `json:"s3_key_prefix"`
	BusinessID                string `json:"business_id"`
	StagingDirectory          string `json:"staging_directory"`
	RetentionSeconds          int64  `json:"retention_seconds"`
	PresignSeconds            int64  `json:"presign_seconds"`
	ArchiveTimeoutSeconds     int64  `json:"archive_timeout_seconds"`
	ArchiveMaxAttempts        int    `json:"archive_max_attempts"`
	ArchiveRetryWindowSeconds int64  `json:"archive_retry_window_seconds"`
	CleanupIntervalSeconds    int64  `json:"cleanup_interval_seconds"`
}

func GetVideoSettings() VideoSettings {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()

	prefix := strings.TrimSpace(common.OptionMap[OptionVideoS3KeyPrefix])
	if prefix == "" {
		prefix = DefaultS3KeyPrefix
	}
	return VideoSettings{
		Endpoint:                  strings.TrimSpace(common.OptionMap[OptionVideoS3Endpoint]),
		Region:                    strings.TrimSpace(common.OptionMap[OptionVideoS3Region]),
		Bucket:                    strings.TrimSpace(common.OptionMap[OptionVideoS3Bucket]),
		AccessKey:                 strings.TrimSpace(common.OptionMap[OptionVideoS3AccessKey]),
		SecretAccessKey:           common.OptionMap[OptionVideoS3SecretAccessKey],
		S3KeyPrefix:               prefix,
		BusinessID:                strings.TrimSpace(common.OptionMap[OptionVideoBusinessID]),
		StagingDirectory:          strings.TrimSpace(common.OptionMap[OptionVideoStagingDirectory]),
		RetentionSeconds:          parseInt64(common.OptionMap[OptionVideoRetentionSeconds], DefaultRetentionSeconds),
		PresignSeconds:            parseInt64(common.OptionMap[OptionVideoPresignSeconds], DefaultPresignSeconds),
		ArchiveTimeoutSeconds:     parseInt64(common.OptionMap[OptionVideoArchiveTimeoutSeconds], DefaultArchiveTimeout),
		ArchiveMaxAttempts:        int(parseInt64(common.OptionMap[OptionVideoArchiveMaxAttempts], DefaultArchiveMaxAttempts)),
		ArchiveRetryWindowSeconds: parseInt64(common.OptionMap[OptionVideoArchiveRetryWindowSeconds], DefaultArchiveRetryWindow),
		CleanupIntervalSeconds:    parseInt64(common.OptionMap[OptionVideoCleanupInterval], DefaultCleanupInterval),
	}
}

func (settings VideoSettings) Configured() bool {
	return settings.Validate() == nil
}

func (settings VideoSettings) Validate() error {
	if settings.Endpoint != "" {
		parsed, err := url.ParseRequestURI(settings.Endpoint)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return errors.New("endpoint must be an HTTP(S) URL")
		}
	}
	if settings.Region == "" {
		return errors.New("region is required")
	}
	if settings.Bucket == "" || strings.ContainsAny(settings.Bucket, " \t\r\n") {
		return errors.New("bucket is invalid")
	}
	if settings.AccessKey == "" {
		return errors.New("access key is required")
	}
	if strings.TrimSpace(settings.SecretAccessKey) == "" {
		return errors.New("secret access key is required")
	}
	if err := validateS3KeyPrefix(settings.S3KeyPrefix); err != nil {
		return err
	}
	if err := validateBusinessID(settings.BusinessID); err != nil {
		return err
	}
	if err := validateStagingDirectory(settings.StagingDirectory); err != nil {
		return err
	}
	if settings.RetentionSeconds < 60 || settings.RetentionSeconds > 31536000 {
		return errors.New("retention seconds must be between 60 and 31536000")
	}
	if settings.PresignSeconds < 60 || settings.PresignSeconds > 604800 {
		return errors.New("presign seconds must be between 60 and 604800")
	}
	if settings.ArchiveTimeoutSeconds < 1 || settings.ArchiveTimeoutSeconds > 1200 {
		return errors.New("archive timeout seconds must be between 1 and 1200")
	}
	if settings.ArchiveMaxAttempts < 1 || settings.ArchiveMaxAttempts > 100 {
		return errors.New("archive max attempts must be between 1 and 100")
	}
	if settings.ArchiveRetryWindowSeconds < 60 || settings.ArchiveRetryWindowSeconds > 604800 {
		return errors.New("archive retry window seconds must be between 60 and 604800")
	}
	if settings.ArchiveRetryWindowSeconds < settings.ArchiveTimeoutSeconds {
		return errors.New("archive retry window must not be shorter than archive timeout")
	}
	if settings.CleanupIntervalSeconds < 60 || settings.CleanupIntervalSeconds > 86400 {
		return errors.New("cleanup interval seconds must be between 60 and 86400")
	}
	return nil
}

func (settings VideoSettings) OptionValues() map[string]string {
	return map[string]string{
		OptionVideoS3Endpoint:                settings.Endpoint,
		OptionVideoS3Region:                  settings.Region,
		OptionVideoS3Bucket:                  settings.Bucket,
		OptionVideoS3AccessKey:               settings.AccessKey,
		OptionVideoS3SecretAccessKey:         settings.SecretAccessKey,
		OptionVideoS3KeyPrefix:               settings.S3KeyPrefix,
		OptionVideoBusinessID:                settings.BusinessID,
		OptionVideoStagingDirectory:          filepath.Clean(settings.StagingDirectory),
		OptionVideoRetentionSeconds:          fmt.Sprintf("%d", settings.RetentionSeconds),
		OptionVideoPresignSeconds:            fmt.Sprintf("%d", settings.PresignSeconds),
		OptionVideoArchiveTimeoutSeconds:     fmt.Sprintf("%d", settings.ArchiveTimeoutSeconds),
		OptionVideoArchiveMaxAttempts:        fmt.Sprintf("%d", settings.ArchiveMaxAttempts),
		OptionVideoArchiveRetryWindowSeconds: fmt.Sprintf("%d", settings.ArchiveRetryWindowSeconds),
		OptionVideoCleanupInterval:           fmt.Sprintf("%d", settings.CleanupIntervalSeconds),
	}
}
