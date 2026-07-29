package storage_setting

import (
	"path/filepath"
	"testing"

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
		StagingDirectory:          "/data/zmodel/async-image-staging",
		RetentionSeconds:          DefaultRetentionSeconds,
		PresignSeconds:            DefaultPresignSeconds,
		ArchiveTimeoutSeconds:     DefaultArchiveTimeout,
		ArchiveMaxAttempts:        DefaultArchiveMaxAttempts,
		ArchiveRetryWindowSeconds: DefaultArchiveRetryWindow,
		CleanupIntervalSeconds:    DefaultCleanupInterval,
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

func TestSettingsRequiresSafeAbsoluteStagingDirectory(t *testing.T) {
	settings := validObjectStorageSettings()
	settings.StagingDirectory = "relative/staging"
	require.ErrorContains(t, settings.Validate(), "absolute path")

	settings.StagingDirectory = string(filepath.Separator)
	require.ErrorContains(t, settings.Validate(), "filesystem root")
}
