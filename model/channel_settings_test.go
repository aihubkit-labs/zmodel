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
		{name: "missing model capabilities", setting: `{"video_protocol":"openai_video"}`, wantErr: "video_model_capabilities is required"},
		{
			name: "Seedance model capabilities",
			setting: `{
				"video_protocol":"seedance(megabyai)",
				"video_model_capabilities":{
					"tvideos":{"resolutions":["480p","720P","1080p","4K"],"max_reference_images":2,"max_reference_videos":1,"max_reference_audios":0,"min_duration_seconds":4,"max_duration_seconds":29}
				}
			}`,
		},
		{name: "Agnes Video V2", setting: `{"video_protocol":"agnes_video_v2","video_model_capabilities":{"agnes-video":{"resolutions":["720p"],"max_reference_images":1,"max_reference_videos":0,"max_reference_audios":0}}}`},
		{name: "legacy Seedance protocol is rejected", setting: `{"video_protocol":"seedance"}`, wantErr: "unsupported video_protocol"},
		{name: "invalid protocol", setting: `{"video_protocol":"unsafe"}`, wantErr: "unsupported video_protocol"},
		{name: "empty model ID", setting: `{"video_model_capabilities":{"":{"resolutions":["480p"]}}}`, wantErr: "empty model ID"},
		{name: "empty resolution list", setting: `{"video_model_capabilities":{"tvideos":{"resolutions":[]}}}`, wantErr: "at least one resolution"},
		{name: "dynamic resolution", setting: `{"video_model_capabilities":{"tvideos":{"resolutions":["1440p"],"max_reference_images":1,"max_reference_videos":0,"max_reference_audios":0}}}`},
		{name: "duplicate normalized model", setting: `{"video_model_capabilities":{"tvideos":{"resolutions":["480p"],"max_reference_images":1,"max_reference_videos":0,"max_reference_audios":0}," TVIDEOS ":{"resolutions":["720p"],"max_reference_images":1,"max_reference_videos":0,"max_reference_audios":0}}}`, wantErr: "duplicate model ID"},
		{name: "duplicate normalized resolution", setting: `{"video_model_capabilities":{"tvideos":{"resolutions":["4k","4K"]}}}`, wantErr: "duplicate resolution"},
		{name: "missing reference limit", setting: `{"video_model_capabilities":{"tvideos":{"resolutions":["480p"],"max_reference_images":1,"max_reference_videos":0}}}`, wantErr: "must configure max_reference_audios"},
		{name: "negative reference limit", setting: `{"video_model_capabilities":{"tvideos":{"resolutions":["480p"],"max_reference_images":-1,"max_reference_videos":0,"max_reference_audios":0}}}`, wantErr: "max_reference_images must be between"},
		{name: "oversized reference limit", setting: `{"video_model_capabilities":{"tvideos":{"resolutions":["480p"],"max_reference_images":0,"max_reference_videos":65,"max_reference_audios":0}}}`, wantErr: "max_reference_videos must be between"},
		{name: "Agnes multiple reference images", setting: `{"video_protocol":"agnes_video_v2","video_model_capabilities":{"agnes-video":{"resolutions":["720p"],"max_reference_images":2,"max_reference_videos":0,"max_reference_audios":0}}}`, wantErr: "cannot exceed 1"},
		{name: "Seedance missing minimum duration", setting: `{"video_protocol":"seedance(megabyai)","video_model_capabilities":{"seedance-2.0":{"resolutions":["720p"],"max_reference_images":0,"max_reference_videos":0,"max_reference_audios":0,"max_duration_seconds":15}}}`, wantErr: "must configure min_duration_seconds"},
		{name: "Seedance missing maximum duration", setting: `{"video_protocol":"seedance(megabyai)","video_model_capabilities":{"seedance-2.0":{"resolutions":["720p"],"max_reference_images":0,"max_reference_videos":0,"max_reference_audios":0,"min_duration_seconds":4}}}`, wantErr: "must configure max_duration_seconds"},
		{name: "Seedance reversed duration range", setting: `{"video_protocol":"seedance(megabyai)","video_model_capabilities":{"seedance-2.0":{"resolutions":["720p"],"max_reference_images":0,"max_reference_videos":0,"max_reference_audios":0,"min_duration_seconds":29,"max_duration_seconds":4}}}`, wantErr: "cannot exceed"},
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
			" TVideos ": {
				Resolutions:        []string{"720P", "4K"},
				MaxReferenceImages: common.GetPointer(2),
				MaxReferenceVideos: common.GetPointer(1),
				MaxReferenceAudios: common.GetPointer(0),
				MinDurationSeconds: common.GetPointer(4),
				MaxDurationSeconds: common.GetPointer(29),
			},
		},
	}

	capability, ok := settings.GetVideoModelCapability("tvideos")
	require.True(t, ok)
	assert.Equal(t, []string{"720p", "4k"}, capability.Resolutions)
	assert.Equal(t, 2, *capability.MaxReferenceImages)
	assert.Equal(t, 1, *capability.MaxReferenceVideos)
	assert.Equal(t, 0, *capability.MaxReferenceAudios)
	assert.Equal(t, 4, *capability.MinDurationSeconds)
	assert.Equal(t, 29, *capability.MaxDurationSeconds)
}
