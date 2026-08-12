package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelValidateVideoRequestSettings(t *testing.T) {
	tests := []struct{ name, setting, wantErr string }{
		{name: "unconfigured", setting: `{}`},
		{name: "missing model capabilities", setting: `{"video_protocol":"openai_video"}`, wantErr: "video_model_capabilities is required"},
		{name: "Agnes Video V2", setting: `{"video_protocol":"agnes_video_v2","video_model_capabilities":{"agnes-video":{"resolutions":["720p"],"max_reference_images":1,"max_reference_videos":0,"max_reference_audios":0}}}`},
		{name: "invalid protocol", setting: `{"video_protocol":"unsafe"}`, wantErr: "unsupported video_protocol"},
		{name: "dynamic resolution", setting: `{"video_model_capabilities":{"tvideos":{"resolutions":["1440p"],"max_reference_images":1,"max_reference_videos":0,"max_reference_audios":0}}}`},
		{name: "duplicate normalized resolution", setting: `{"video_model_capabilities":{"tvideos":{"resolutions":["4k","4K"]}}}`, wantErr: "duplicate resolution"},
		{name: "Agnes multiple reference images", setting: `{"video_protocol":"agnes_video_v2","video_model_capabilities":{"agnes-video":{"resolutions":["720p"],"max_reference_images":2,"max_reference_videos":0,"max_reference_audios":0}}}`, wantErr: "cannot exceed 1"},
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

func validExtendedVideoCapability() dto.VideoModelCapability {
	trueValue := true
	falseValue := false
	return dto.VideoModelCapability{
		Resolutions:                           []string{"1440p"},
		Ratios:                                []string{"16:9"},
		RatioRequired:                         &trueValue,
		MinReferenceImages:                    common.GetPointer(0),
		MaxReferenceImages:                    common.GetPointer(5),
		MinReferenceVideos:                    common.GetPointer(0),
		MaxReferenceVideos:                    common.GetPointer(0),
		MinReferenceAudios:                    common.GetPointer(0),
		MaxReferenceAudios:                    common.GetPointer(3),
		SupportsDuration:                      &trueValue,
		DurationRequired:                      &trueValue,
		MinDurationSeconds:                    common.GetPointer(5),
		MaxDurationSeconds:                    common.GetPointer(15),
		SupportsGenerateAudio:                 &trueValue,
		GenerateAudioRequired:                 &trueValue,
		SupportsFirstFrame:                    &trueValue,
		FirstFrameRequired:                    &falseValue,
		SupportsLastFrame:                     &trueValue,
		LastFrameRequired:                     &falseValue,
		LastFrameRequiresFirstFrame:           &trueValue,
		ReferenceImagesIncompatibleWithFrames: &trueValue,
		AudioReferenceRequiresVisualReference: &trueValue,
		ReferenceMediaIncompatibleWithFrames:  &falseValue,
		SupportsSeed:                          &falseValue,
		SupportsWatermark:                     &falseValue,
	}
}

func TestChannelSettingsValidatesExtendedCapabilityRelationships(t *testing.T) {
	falseValue := false
	settings := dto.ChannelSettings{VideoProtocol: dto.VideoProtocolMegabyAI}

	tests := []struct {
		name      string
		configure func(*dto.VideoModelCapability)
		wantErr   string
	}{
		{
			name: "all capabilities can be disabled",
			configure: func(capability *dto.VideoModelCapability) {
				capability.SupportsGenerateAudio = &falseValue
				capability.GenerateAudioRequired = &falseValue
				capability.SupportsFirstFrame = &falseValue
				capability.SupportsLastFrame = &falseValue
				capability.LastFrameRequiresFirstFrame = &falseValue
				capability.ReferenceImagesIncompatibleWithFrames = &falseValue
				capability.AudioReferenceRequiresVisualReference = &falseValue
			},
		},
		{
			name: "required audio must be supported",
			configure: func(capability *dto.VideoModelCapability) {
				capability.SupportsGenerateAudio = &falseValue
			},
			wantErr: "cannot require generate_audio when it is unsupported",
		},
		{
			name: "required first frame must be supported",
			configure: func(capability *dto.VideoModelCapability) {
				capability.SupportsFirstFrame = &falseValue
			},
			wantErr: "cannot require a first frame when first frames are unsupported",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			capability := validExtendedVideoCapability()
			test.configure(&capability)
			testSettings := settings
			testSettings.VideoModelCapabilities = map[string]dto.VideoModelCapability{
				"minimax-h3": capability,
			}
			err := testSettings.ValidateVideoRequestSettings()
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestChannelSettingsGetVideoModelCapabilityNormalizesLookup(t *testing.T) {
	settings := dto.ChannelSettings{
		VideoModelCapabilities: map[string]dto.VideoModelCapability{
			" TVideos ": {
				Resolutions:                           []string{"720P", "4K"},
				Ratios:                                []string{"16:9", " 1:1 "},
				MaxReferenceImages:                    common.GetPointer(2),
				MaxReferenceVideos:                    common.GetPointer(1),
				MaxReferenceAudios:                    common.GetPointer(0),
				MinDurationSeconds:                    common.GetPointer(4),
				MaxDurationSeconds:                    common.GetPointer(29),
				SupportsGenerateAudio:                 common.GetPointer(true),
				GenerateAudioRequired:                 common.GetPointer(true),
				SupportsFirstFrame:                    common.GetPointer(true),
				SupportsLastFrame:                     common.GetPointer(true),
				LastFrameRequiresFirstFrame:           common.GetPointer(true),
				ReferenceImagesIncompatibleWithFrames: common.GetPointer(true),
				AudioReferenceRequiresVisualReference: common.GetPointer(true),
			},
		},
	}

	capability, ok := settings.GetVideoModelCapability("tvideos")
	require.True(t, ok)
	assert.Equal(t, []string{"720p", "4k"}, capability.Resolutions)
	assert.Equal(t, []string{"16:9", "1:1"}, capability.Ratios)
	assert.Equal(t, 2, *capability.MaxReferenceImages)
	assert.Equal(t, 1, *capability.MaxReferenceVideos)
	assert.Equal(t, 0, *capability.MaxReferenceAudios)
	assert.Equal(t, 4, *capability.MinDurationSeconds)
	assert.Equal(t, 29, *capability.MaxDurationSeconds)
	assert.True(t, *capability.SupportsGenerateAudio)
	assert.True(t, *capability.GenerateAudioRequired)
	assert.True(t, *capability.SupportsFirstFrame)
	assert.True(t, *capability.SupportsLastFrame)
	assert.True(t, *capability.LastFrameRequiresFirstFrame)
	assert.True(t, *capability.ReferenceImagesIncompatibleWithFrames)
	assert.True(t, *capability.AudioReferenceRequiresVisualReference)
}
