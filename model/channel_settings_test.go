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
		{name: "Lingganya video", setting: `{"video_protocol":"lingganya_video","video_model_capabilities":{"sora-2":{"resolutions":["16:9"],"ratio_required":false,"min_reference_images":0,"max_reference_images":1,"min_reference_videos":0,"max_reference_videos":0,"min_reference_audios":0,"max_reference_audios":0,"supports_duration":true,"duration_required":false,"min_duration_seconds":4,"max_duration_seconds":12,"allowed_duration_seconds":[4,8,12],"default_duration_seconds":4,"supports_generate_audio":false,"generate_audio_required":false,"supports_first_frame":true,"first_frame_required":false,"supports_last_frame":false,"last_frame_required":false,"last_frame_requires_first_frame":false,"reference_images_incompatible_with_frames":false,"audio_reference_requires_visual_reference":false,"reference_media_incompatible_with_frames":false,"supports_seed":false,"supports_watermark":false}}}`},
		{name: "Lingganya duration range", setting: `{"video_protocol":"lingganya_video","video_model_capabilities":{"sd-2.0-vip":{"resolutions":["720p"],"ratios":["16:9"],"ratio_required":false,"min_reference_images":0,"max_reference_images":9,"min_reference_videos":0,"max_reference_videos":3,"min_reference_audios":0,"max_reference_audios":3,"supports_duration":true,"duration_required":false,"min_duration_seconds":4,"max_duration_seconds":15,"default_duration_seconds":6,"supports_generate_audio":false,"generate_audio_required":false,"supports_first_frame":false,"first_frame_required":false,"supports_last_frame":false,"last_frame_required":false,"last_frame_requires_first_frame":false,"reference_images_incompatible_with_frames":false,"audio_reference_requires_visual_reference":true,"reference_media_requires_visual_reference":true,"reference_media_incompatible_with_frames":false,"supports_seed":false,"supports_watermark":false}}}`},
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
			name: "size mapping",
			configure: func(capability *dto.VideoModelCapability) {
				capability.SizeMappings = map[string]string{"1440p|16:9": "2560x1440"}
				capability.OmitParameters = []string{"resolution", "size"}
			},
		},
		{
			name: "size mapping key format",
			configure: func(capability *dto.VideoModelCapability) {
				capability.SizeMappings = map[string]string{"1440p/16:9": "2560x1440"}
			},
			wantErr: "must use resolution|ratio",
		},
		{
			name: "size mapping resolution",
			configure: func(capability *dto.VideoModelCapability) {
				capability.SizeMappings = map[string]string{"720p|16:9": "1280x720"}
			},
			wantErr: "size mapping resolution",
		},
		{
			name: "size mapping ratio",
			configure: func(capability *dto.VideoModelCapability) {
				capability.SizeMappings = map[string]string{"1440p|9:16": "1440x2560"}
			},
			wantErr: "size mapping ratio",
		},
		{
			name: "size mapping value",
			configure: func(capability *dto.VideoModelCapability) {
				capability.SizeMappings = map[string]string{"1440p|16:9": ""}
			},
			wantErr: "invalid upstream size mapping",
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
		{
			name: "default duration must be allowed",
			configure: func(capability *dto.VideoModelCapability) {
				capability.AllowedDurationSeconds = []int{5, 10, 15}
				capability.DefaultDurationSeconds = common.GetPointer(8)
			},
			wantErr: "default_duration_seconds must be one of allowed_duration_seconds",
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
				ResolutionMappings:                    map[string]string{"720p": "720P"},
				SizeMappings:                          map[string]string{"720p|16:9": "1280x720"},
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
				AssetPreparationMode:                  dto.VideoAssetPreparationGlobalAIOpcSeedance,
			},
		},
	}

	capability, ok := settings.GetVideoModelCapability("tvideos")
	require.True(t, ok)
	assert.Equal(t, []string{"720p", "4k"}, capability.Resolutions)
	assert.Equal(t, []string{"16:9", "1:1"}, capability.Ratios)
	assert.Equal(t, "1280x720", capability.SizeMappings["720p|16:9"])
	capability.ResolutionMappings["720p"] = "changed"
	capability.SizeMappings["720p|16:9"] = "changed"
	assert.Equal(t, "720P", settings.VideoModelCapabilities[" TVideos "].ResolutionMappings["720p"])
	assert.Equal(t, "1280x720", settings.VideoModelCapabilities[" TVideos "].SizeMappings["720p|16:9"])
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
	assert.Equal(t, dto.VideoAssetPreparationGlobalAIOpcSeedance, capability.AssetPreparationMode)
}

func TestChannelSettingsValidatesVideoAssetPreparationMode(t *testing.T) {
	capability := validExtendedVideoCapability()
	capability.AssetPreparationMode = dto.VideoAssetPreparationGlobalAIOpcSeedance

	settings := dto.ChannelSettings{
		VideoProtocol:          dto.VideoProtocolGlobalAIOpc,
		VideoModelCapabilities: map[string]dto.VideoModelCapability{"sd_2.5_special_v1": capability},
	}
	require.NoError(t, settings.ValidateVideoRequestSettings())

	settings.VideoProtocol = dto.VideoProtocolMegabyAI
	require.ErrorContains(t, settings.ValidateVideoRequestSettings(), "requires the GlobalAIOpc video protocol")

	settings.VideoProtocol = dto.VideoProtocolGlobalAIOpc
	capability.AssetPreparationMode = "unknown"
	settings.VideoModelCapabilities["sd_2.5_special_v1"] = capability
	require.ErrorContains(t, settings.ValidateVideoRequestSettings(), "unsupported asset preparation mode")
}

func TestChannelSettingsResolvesGlobalAIOpcAssetPreparationDefaults(t *testing.T) {
	settings := dto.ChannelSettings{}

	resolved := settings.GetGlobalAIOpcAssetPreparationSettings()
	assert.Equal(t, 10, resolved.OperationsPerTaskPerPass)
	assert.Equal(t, 900, resolved.TimeoutSeconds)

	settings.GlobalAIOpcAssetPreparation = &dto.GlobalAIOpcAssetPreparationSettings{
		OperationsPerTaskPerPass: 25,
		TimeoutSeconds:           1200,
	}
	resolved = settings.GetGlobalAIOpcAssetPreparationSettings()
	assert.Equal(t, 25, resolved.OperationsPerTaskPerPass)
	assert.Equal(t, 1200, resolved.TimeoutSeconds)
}

func TestChannelSettingsValidatesGlobalAIOpcAssetPreparation(t *testing.T) {
	settings := dto.ChannelSettings{
		VideoProtocol: dto.VideoProtocolGlobalAIOpc,
		VideoModelCapabilities: map[string]dto.VideoModelCapability{
			"sd_2.5_special_v1": validExtendedVideoCapability(),
		},
		GlobalAIOpcAssetPreparation: &dto.GlobalAIOpcAssetPreparationSettings{},
	}
	require.NoError(t, settings.ValidateVideoRequestSettings())

	settings.GlobalAIOpcAssetPreparation.OperationsPerTaskPerPass = 51
	require.ErrorContains(t, settings.ValidateVideoRequestSettings(), "operations_per_task_per_pass must be between 1 and 50")

	settings.GlobalAIOpcAssetPreparation.OperationsPerTaskPerPass = 10
	settings.GlobalAIOpcAssetPreparation.TimeoutSeconds = 59
	require.ErrorContains(t, settings.ValidateVideoRequestSettings(), "timeout_seconds must be between 60 and 3600")

	settings.GlobalAIOpcAssetPreparation.TimeoutSeconds = 900
	settings.VideoProtocol = dto.VideoProtocolMegabyAI
	require.ErrorContains(t, settings.ValidateVideoRequestSettings(), "requires the GlobalAIOpc video protocol")
}
