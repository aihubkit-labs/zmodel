package storage_setting

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	OptionS3Endpoint                      = "ObjectStorageS3Endpoint"
	OptionS3Region                        = "ObjectStorageS3Region"
	OptionS3Bucket                        = "ObjectStorageS3Bucket"
	OptionS3AccessKey                     = "ObjectStorageS3AccessKey"
	OptionS3SecretAccessKey               = "ObjectStorageS3SecretAccessKey"
	OptionStagingDirectory                = "ObjectStorageStagingDirectory"
	OptionRetentionSeconds                = "ObjectStorageRetentionSeconds"
	OptionPresignSeconds                  = "ObjectStoragePresignSeconds"
	OptionArchiveTimeoutSeconds           = "ObjectStorageArchiveTimeoutSeconds"
	OptionArchiveMaxAttempts              = "ObjectStorageArchiveMaxAttempts"
	OptionArchiveRetryWindowSeconds       = "ObjectStorageArchiveRetryWindowSeconds"
	OptionCleanupIntervalSeconds          = "ObjectStorageCleanupIntervalSeconds"
	DefaultRetentionSeconds         int64 = 86400
	DefaultPresignSeconds           int64 = 600
	DefaultArchiveTimeout           int64 = 600
	DefaultArchiveMaxAttempts             = 8
	DefaultArchiveRetryWindow       int64 = 21600
	DefaultCleanupInterval          int64 = 900
	EnvStagingDirectory                   = "ASYNC_IMAGE_STAGING_DIR"
)

type Settings struct {
	Endpoint                  string `json:"endpoint"`
	Region                    string `json:"region"`
	Bucket                    string `json:"bucket"`
	AccessKey                 string `json:"access_key"`
	SecretAccessKey           string `json:"-"`
	StagingDirectory          string `json:"staging_directory"`
	RetentionSeconds          int64  `json:"retention_seconds"`
	PresignSeconds            int64  `json:"presign_seconds"`
	ArchiveTimeoutSeconds     int64  `json:"archive_timeout_seconds"`
	ArchiveMaxAttempts        int    `json:"archive_max_attempts"`
	ArchiveRetryWindowSeconds int64  `json:"archive_retry_window_seconds"`
	CleanupIntervalSeconds    int64  `json:"cleanup_interval_seconds"`
}

func GetSettings() Settings {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	stagingDirectory := strings.TrimSpace(common.OptionMap[OptionStagingDirectory])
	if stagingDirectory == "" {
		stagingDirectory = strings.TrimSpace(os.Getenv(EnvStagingDirectory))
	}
	return Settings{
		Endpoint:                  strings.TrimSpace(common.OptionMap[OptionS3Endpoint]),
		Region:                    strings.TrimSpace(common.OptionMap[OptionS3Region]),
		Bucket:                    strings.TrimSpace(common.OptionMap[OptionS3Bucket]),
		AccessKey:                 strings.TrimSpace(common.OptionMap[OptionS3AccessKey]),
		SecretAccessKey:           common.OptionMap[OptionS3SecretAccessKey],
		StagingDirectory:          stagingDirectory,
		RetentionSeconds:          parseInt64(common.OptionMap[OptionRetentionSeconds], DefaultRetentionSeconds),
		PresignSeconds:            parseInt64(common.OptionMap[OptionPresignSeconds], DefaultPresignSeconds),
		ArchiveTimeoutSeconds:     parseInt64(common.OptionMap[OptionArchiveTimeoutSeconds], DefaultArchiveTimeout),
		ArchiveMaxAttempts:        int(parseInt64(common.OptionMap[OptionArchiveMaxAttempts], DefaultArchiveMaxAttempts)),
		ArchiveRetryWindowSeconds: parseInt64(common.OptionMap[OptionArchiveRetryWindowSeconds], DefaultArchiveRetryWindow),
		CleanupIntervalSeconds:    parseInt64(common.OptionMap[OptionCleanupIntervalSeconds], DefaultCleanupInterval),
	}
}

func (s Settings) Configured() bool {
	return s.Region != "" && s.Bucket != "" && s.AccessKey != "" && strings.TrimSpace(s.SecretAccessKey) != ""
}

func (s Settings) Validate() error {
	if s.Endpoint != "" {
		parsed, err := url.ParseRequestURI(s.Endpoint)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return errors.New("endpoint must be an HTTP(S) URL")
		}
	}
	if s.Region == "" {
		return errors.New("region is required")
	}
	if s.Bucket == "" || strings.ContainsAny(s.Bucket, " \t\r\n") {
		return errors.New("bucket is invalid")
	}
	if s.AccessKey == "" {
		return errors.New("access key is required")
	}
	if strings.TrimSpace(s.SecretAccessKey) == "" {
		return errors.New("secret access key is required")
	}
	if s.StagingDirectory == "" {
		return errors.New("staging directory is required")
	}
	if len(s.StagingDirectory) > 1024 || !filepath.IsAbs(s.StagingDirectory) {
		return errors.New("staging directory must be an absolute path of at most 1024 characters")
	}
	cleanStagingDirectory := filepath.Clean(s.StagingDirectory)
	if filepath.Dir(cleanStagingDirectory) == cleanStagingDirectory {
		return errors.New("staging directory cannot be a filesystem root")
	}
	if s.RetentionSeconds < 60 || s.RetentionSeconds > 31536000 {
		return errors.New("retention seconds must be between 60 and 31536000")
	}
	if s.PresignSeconds < 60 || s.PresignSeconds > 604800 {
		return errors.New("presign seconds must be between 60 and 604800")
	}
	if s.ArchiveTimeoutSeconds < 1 || s.ArchiveTimeoutSeconds > 1200 {
		return errors.New("archive timeout seconds must be between 1 and 1200")
	}
	if s.ArchiveMaxAttempts < 1 || s.ArchiveMaxAttempts > 100 {
		return errors.New("archive max attempts must be between 1 and 100")
	}
	if s.ArchiveRetryWindowSeconds < 60 || s.ArchiveRetryWindowSeconds > 604800 {
		return errors.New("archive retry window seconds must be between 60 and 604800")
	}
	if s.ArchiveRetryWindowSeconds < s.ArchiveTimeoutSeconds {
		return errors.New("archive retry window must not be shorter than archive timeout")
	}
	if s.CleanupIntervalSeconds < 60 || s.CleanupIntervalSeconds > 86400 {
		return errors.New("cleanup interval seconds must be between 60 and 86400")
	}
	return nil
}

func parseInt64(value string, fallback int64) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
