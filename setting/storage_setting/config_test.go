package storage_setting

import (
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validObjectStorageSettings() Settings {
	return Settings{
		Endpoint:                  "https://s3.example.com",
		Region:                    "test-region",
		Bucket:                    "test-bucket",
		AccessKey:                 "test-access-key",
		SecretAccessKey:           "test-secret",
		S3KeyPrefix:               DefaultS3KeyPrefix,
		BusinessID:                "test@async-images",
		StagingDirectory:          "/data/zmodel/async-image-staging",
		RetentionSeconds:          DefaultRetentionSeconds,
		PresignSeconds:            DefaultPresignSeconds,
		ArchiveTimeoutSeconds:     DefaultArchiveTimeout,
		ArchiveMaxAttempts:        DefaultArchiveMaxAttempts,
		ArchiveRetryWindowSeconds: DefaultArchiveRetryWindow,
		CleanupIntervalSeconds:    DefaultCleanupInterval,
	}
}

func TestGetSettingsDefaultsS3KeyPrefixForExistingInstallations(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	original := common.OptionMap
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = original
		common.OptionMapRWMutex.Unlock()
	})

	assert.Equal(t, DefaultS3KeyPrefix, GetSettings().S3KeyPrefix)
}

func TestSettingsRequiresSafeS3KeyPrefix(t *testing.T) {
	settings := validObjectStorageSettings()
	settings.S3KeyPrefix = "dev"
	require.NoError(t, settings.Validate())

	invalidPrefixes := []string{"", "/dev", "dev/", "dev//images", "dev/../prod", `dev\images`, "dev images"}
	for _, prefix := range invalidPrefixes {
		settings.S3KeyPrefix = prefix
		require.ErrorContains(t, settings.Validate(), "S3 key prefix", prefix)
	}
}

func TestSettingsValidationProtectsArchiveTimingContract(t *testing.T) {
	settings := validObjectStorageSettings()
	require.NoError(t, settings.Validate())
	assert.True(t, settings.Configured())

	settings.ArchiveTimeoutSeconds = 1201
	require.ErrorContains(t, settings.Validate(), "archive timeout")

	settings = validObjectStorageSettings()
	settings.ArchiveRetryWindowSeconds = settings.ArchiveTimeoutSeconds - 1
	require.ErrorContains(t, settings.Validate(), "retry window")
}

func TestSettingsRequiresSecretAndValidEndpoint(t *testing.T) {
	settings := validObjectStorageSettings()
	settings.SecretAccessKey = ""
	assert.False(t, settings.Configured())
	require.ErrorContains(t, settings.Validate(), "secret access key")

	settings = validObjectStorageSettings()
	settings.Endpoint = "file:///tmp/object-storage"
	require.ErrorContains(t, settings.Validate(), "HTTP(S)")
}

func TestConfiguredSettingsRequireBusinessNamespace(t *testing.T) {
	settings := validObjectStorageSettings()
	settings.BusinessID = ""
	assert.False(t, settings.Configured())
	require.ErrorContains(t, settings.Validate(), "business ID")

	videoSettings := VideoSettings{
		Region: "test-region", Bucket: "test-bucket", AccessKey: "test-access-key",
		SecretAccessKey: "test-secret", S3KeyPrefix: "prod",
		StagingDirectory: "/data/zmodel/video-staging",
		RetentionSeconds: DefaultRetentionSeconds, PresignSeconds: DefaultPresignSeconds,
		ArchiveTimeoutSeconds: DefaultArchiveTimeout, ArchiveMaxAttempts: DefaultArchiveMaxAttempts,
		ArchiveRetryWindowSeconds: DefaultArchiveRetryWindow, CleanupIntervalSeconds: DefaultCleanupInterval,
	}
	assert.False(t, videoSettings.Configured())
	require.ErrorContains(t, videoSettings.Validate(), "business ID")

	videoSettings.BusinessID = "test@videos"
	assert.True(t, videoSettings.Configured())
	require.NoError(t, videoSettings.Validate())
}

func TestSettingsRequiresSafeAbsoluteStagingDirectory(t *testing.T) {
	settings := validObjectStorageSettings()
	settings.StagingDirectory = "relative/staging"
	require.ErrorContains(t, settings.Validate(), "absolute path")

	settings.StagingDirectory = string(filepath.Separator)
	require.ErrorContains(t, settings.Validate(), "filesystem root")
}

func TestVideoSettingsRequiresSafeStagingAndRetryWindow(t *testing.T) {
	settings := VideoSettings{
		Region: "test-region", Bucket: "test-bucket", AccessKey: "test-access-key",
		SecretAccessKey: "test-secret", S3KeyPrefix: "prod", BusinessID: "test@videos",
		StagingDirectory: "/data/zmodel/video-staging",
		RetentionSeconds: DefaultRetentionSeconds, PresignSeconds: DefaultPresignSeconds,
		ArchiveTimeoutSeconds: DefaultArchiveTimeout, ArchiveMaxAttempts: DefaultArchiveMaxAttempts,
		ArchiveRetryWindowSeconds: DefaultArchiveRetryWindow, CleanupIntervalSeconds: DefaultCleanupInterval,
	}
	require.NoError(t, settings.Validate())

	settings.StagingDirectory = "relative/video-staging"
	require.ErrorContains(t, settings.Validate(), "absolute path")

	settings.StagingDirectory = "/data/zmodel/video-staging"
	settings.ArchiveRetryWindowSeconds = settings.ArchiveTimeoutSeconds - 1
	require.ErrorContains(t, settings.Validate(), "retry window")
}
