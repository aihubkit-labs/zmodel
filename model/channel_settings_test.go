package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelValidateVideoRequestSettings(t *testing.T) {
	tests := []struct {
		name    string
		setting string
		wantErr string
	}{
		{name: "unconfigured preserves historical behavior", setting: `{}`},
		{name: "OpenAI video", setting: `{"video_protocol":"openai_video"}`},
		{name: "Seedance", setting: `{"video_protocol":"seedance"}`},
		{
			name: "Seedance model capabilities",
			setting: `{
				"video_protocol":"seedance",
				"video_model_capabilities":{
					"tvideos":{"resolutions":["480p","720P","1080p","4K"]}
				}
			}`,
		},
		{name: "Agnes Video V2", setting: `{"video_protocol":"agnes_video_v2"}`},
		{name: "invalid protocol", setting: `{"video_protocol":"unsafe"}`, wantErr: "unsupported video_protocol"},
		{name: "empty model ID", setting: `{"video_model_capabilities":{"":{"resolutions":["480p"]}}}`, wantErr: "empty model ID"},
		{name: "empty resolution list", setting: `{"video_model_capabilities":{"tvideos":{"resolutions":[]}}}`, wantErr: "at least one resolution"},
		{name: "unknown resolution", setting: `{"video_model_capabilities":{"tvideos":{"resolutions":["2k"]}}}`, wantErr: "unsupported resolution"},
		{name: "duplicate normalized model", setting: `{"video_model_capabilities":{"tvideos":{"resolutions":["480p"]}," TVIDEOS ":{"resolutions":["720p"]}}}`, wantErr: "duplicate model ID"},
		{name: "duplicate normalized resolution", setting: `{"video_model_capabilities":{"tvideos":{"resolutions":["4k","4K"]}}}`, wantErr: "duplicate resolution"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			channel := &Channel{Setting: common.GetPointer(test.setting)}
			err := channel.ValidateSettings()
			if test.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestChannelSettingsGetVideoModelCapabilityNormalizesLookup(t *testing.T) {
	settings := dto.ChannelSettings{
		VideoModelCapabilities: map[string]dto.VideoModelCapability{
			" TVideos ": {Resolutions: []string{"720P", "4K"}},
		},
	}

	capability, ok := settings.GetVideoModelCapability("tvideos")
	require.True(t, ok)
	assert.Equal(t, []string{"720p", "4k"}, capability.Resolutions)
}
