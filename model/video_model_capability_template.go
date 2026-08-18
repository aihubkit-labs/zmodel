package model

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

type VideoModelCapabilityTemplate struct {
	ID             int               `json:"id"`
	VideoProtocol  dto.VideoProtocol `json:"video_protocol" gorm:"size:64;not null;uniqueIndex:uk_video_capability_template,priority:1"`
	ModelID        string            `json:"model_id" gorm:"size:128;not null;uniqueIndex:uk_video_capability_template,priority:2"`
	Name           string            `json:"name" gorm:"size:256;not null"`
	CapabilityJSON string            `json:"-" gorm:"type:text;not null"`
	Source         string            `json:"source" gorm:"size:32;not null"`
	SourceURL      string            `json:"source_url,omitempty" gorm:"type:text"`
	BuiltIn        bool              `json:"built_in"`
	CreatedTime    int64             `json:"created_time" gorm:"bigint"`
	UpdatedTime    int64             `json:"updated_time" gorm:"bigint"`
}

func (template *VideoModelCapabilityTemplate) ToDTO() (dto.VideoModelCapabilityTemplate, error) {
	var capability dto.VideoModelCapability
	if err := common.UnmarshalJsonStr(template.CapabilityJSON, &capability); err != nil {
		return dto.VideoModelCapabilityTemplate{}, err
	}
	return dto.VideoModelCapabilityTemplate{
		ID:            template.ID,
		VideoProtocol: template.VideoProtocol,
		ModelID:       template.ModelID,
		Name:          template.Name,
		Capability:    capability,
		Source:        template.Source,
		SourceURL:     template.SourceURL,
		BuiltIn:       template.BuiltIn,
		CreatedTime:   template.CreatedTime,
		UpdatedTime:   template.UpdatedTime,
	}, nil
}

func ListVideoModelCapabilityTemplates(protocol dto.VideoProtocol) ([]VideoModelCapabilityTemplate, error) {
	var templates []VideoModelCapabilityTemplate
	db := DB.Order("model_id ASC")
	if protocol != "" {
		db = db.Where("video_protocol = ?", protocol)
	}
	return templates, db.Find(&templates).Error
}

func SaveVideoModelCapabilityTemplate(template *VideoModelCapabilityTemplate) error {
	now := common.GetTimestamp()
	template.VideoProtocol = dto.VideoProtocol(strings.TrimSpace(string(template.VideoProtocol)))
	template.ModelID = strings.TrimSpace(template.ModelID)
	template.Name = strings.TrimSpace(template.Name)
	if template.Source == "" {
		template.Source = "manual"
	}
	template.UpdatedTime = now
	if template.ID != 0 {
		var existingByID VideoModelCapabilityTemplate
		result := DB.Where("id = ?", template.ID).Limit(1).Find(&existingByID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("video model capability template %d does not exist", template.ID)
		}
		if existingByID.BuiltIn {
			return fmt.Errorf("built-in video model capability templates cannot be modified")
		}
		template.CreatedTime = existingByID.CreatedTime
	} else {
		var existing VideoModelCapabilityTemplate
		result := DB.Where("video_protocol = ? AND model_id = ?", template.VideoProtocol, template.ModelID).
			Limit(1).
			Find(&existing)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			if existing.BuiltIn {
				return fmt.Errorf("a built-in video model capability template already exists for this protocol and model ID")
			}
			template.ID = existing.ID
			template.CreatedTime = existing.CreatedTime
		}
	}
	if template.ID == 0 {
		template.CreatedTime = now
		return DB.Create(template).Error
	}
	return DB.Model(&VideoModelCapabilityTemplate{}).
		Where("id = ?", template.ID).
		Updates(map[string]any{
			"video_protocol":  template.VideoProtocol,
			"model_id":        template.ModelID,
			"name":            template.Name,
			"capability_json": template.CapabilityJSON,
			"source":          template.Source,
			"source_url":      template.SourceURL,
			"built_in":        template.BuiltIn,
			"updated_time":    template.UpdatedTime,
		}).Error
}

func DeleteVideoModelCapabilityTemplate(id int) error {
	var template VideoModelCapabilityTemplate
	result := DB.Where("id = ?", id).Limit(1).Find(&template)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("video model capability template %d does not exist", id)
	}
	if template.BuiltIn {
		return fmt.Errorf("built-in video model capability templates cannot be deleted")
	}
	return DB.Delete(&template).Error
}

func SeedVideoModelCapabilityTemplates() error {
	for _, seed := range builtInGlobalAIOpcCapabilityTemplates() {
		settings := dto.ChannelSettings{
			VideoProtocol:          seed.VideoProtocol,
			VideoModelCapabilities: map[string]dto.VideoModelCapability{seed.ModelID: seed.Capability},
		}
		if err := settings.ValidateVideoRequestSettings(); err != nil {
			return err
		}
		capabilityJSON, err := common.Marshal(seed.Capability)
		if err != nil {
			return err
		}
		record := VideoModelCapabilityTemplate{
			VideoProtocol:  seed.VideoProtocol,
			ModelID:        seed.ModelID,
			Name:           seed.Name,
			CapabilityJSON: string(capabilityJSON),
			Source:         "official_docs",
			SourceURL:      seed.SourceURL,
			BuiltIn:        true,
			CreatedTime:    common.GetTimestamp(),
			UpdatedTime:    common.GetTimestamp(),
		}
		var existing VideoModelCapabilityTemplate
		result := DB.Where("video_protocol = ? AND model_id = ?", record.VideoProtocol, record.ModelID).
			Limit(1).
			Find(&existing)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			if err := DB.Create(&record).Error; err != nil {
				return err
			}
			continue
		}
		if !existing.BuiltIn {
			continue
		}
		record.ID = existing.ID
		record.CreatedTime = existing.CreatedTime
		if err := DB.Model(&VideoModelCapabilityTemplate{}).
			Where("id = ?", record.ID).
			Updates(map[string]any{
				"name":            record.Name,
				"capability_json": record.CapabilityJSON,
				"source":          record.Source,
				"source_url":      record.SourceURL,
				"updated_time":    record.UpdatedTime,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

type videoCapabilityTemplateSeed struct {
	VideoProtocol dto.VideoProtocol
	ModelID       string
	Name          string
	SourceURL     string
	Capability    dto.VideoModelCapability
}

func builtInGlobalAIOpcCapabilityTemplates() []videoCapabilityTemplateSeed {
	const docsBase = "https://docs.globalaiopc.com/api-reference/model-center/video-gen/"
	seeds := []videoCapabilityTemplateSeed{
		{ModelID: "minimax-h3", Name: "MiniMax H3", SourceURL: docsBase + "minimax-h3", Capability: globalAIOpcVideoCapability([]string{"1440p"}, []string{"16:9", "9:16"}, 5, 15, 5, 0, 3)},
		{ModelID: "MiniMax-H3-c4", Name: "MiniMax H3 c4", SourceURL: docsBase + "minimax-h3-c4", Capability: globalAIOpcVideoCapability([]string{"1440p"}, []string{"16:9", "9:16"}, 5, 15, 5, 0, 3)},
		{ModelID: "hh-1.1-r2v-o", Name: "HappyHorse 1.1 v2 Reference", SourceURL: docsBase + "hh-1.1-r2v-o", Capability: globalAIOpcVideoCapability([]string{"720p", "1080p"}, []string{"16:9", "9:16", "3:4", "4:3", "4:5", "5:4", "1:1", "9:21", "21:9"}, 3, 15, 9, 0, 0)},
		{ModelID: "hh-1.1-i2v-o", Name: "HappyHorse 1.1 v2 Image", SourceURL: docsBase + "hh-1.1-i2v-o", Capability: globalAIOpcVideoCapability([]string{"720p", "1080p"}, nil, 3, 15, 0, 0, 0)},
		{ModelID: "hh-1.1-t2v-o", Name: "HappyHorse 1.1 v2 Text", SourceURL: docsBase + "hh-1.1-t2v-o", Capability: globalAIOpcVideoCapability([]string{"720p", "1080p"}, []string{"16:9", "9:16", "3:4", "4:3", "4:5", "5:4", "1:1", "9:21", "21:9"}, 3, 15, 0, 0, 0)},
		{ModelID: "wan2.7-r2v", Name: "Wan2.7 Reference", SourceURL: docsBase + "wan2.7-r2v", Capability: globalAIOpcVideoCapability([]string{"720p", "1080p"}, []string{"16:9", "9:16", "1:1", "4:3", "3:4"}, 4, 15, 3, 0, 0)},
		{ModelID: "wan2.7-i2v", Name: "Wan2.7 Image", SourceURL: docsBase + "wan2.7-i2v", Capability: globalAIOpcVideoCapability([]string{"720p", "1080p"}, nil, 4, 15, 0, 0, 0)},
		{ModelID: "wan2.7-t2v", Name: "Wan2.7 Text", SourceURL: docsBase + "wan2.7-t2v", Capability: globalAIOpcVideoCapability([]string{"720p", "1080p"}, []string{"16:9", "9:16", "1:1", "4:3", "3:4"}, 4, 15, 0, 0, 0)},
		{ModelID: "wan2.7-videoedit", Name: "Wan2.7 Video Edit", SourceURL: docsBase + "wan2.7-videoedit", Capability: globalAIOpcVideoCapability([]string{"720p", "1080p"}, []string{"16:9", "9:16", "1:1", "3:4", "4:3"}, 0, 0, 4, 1, 0)},
		{ModelID: "KlingO3", Name: "Kling O3", SourceURL: docsBase + "klingo3", Capability: globalAIOpcVideoCapability([]string{"720p", "1080p"}, []string{"16:9", "9:16", "1:1"}, 3, 15, 3, 0, 0)},
		{ModelID: "seedance-2.5-c1", Name: "Seedance 2.5 c1", SourceURL: docsBase + "seedance-2.5-c1", Capability: globalAIOpcVideoCapability([]string{"480p", "720p"}, []string{"9:16", "16:9", "1:1"}, 4, 30, 30, 10, 10)},
		{ModelID: "seedance-2.5-c3", Name: "Seedance 2.5 c3", SourceURL: docsBase + "seedance-2.5-c3", Capability: globalAIOpcVideoCapability([]string{"720p"}, []string{"16:9", "1:1", "9:16"}, 4, 29, 30, 0, 10)},
		{ModelID: "sd_2.5_discount_v1", Name: "Seedance 2.5 Discount V1", SourceURL: docsBase + "sd_2.5_discount_v1", Capability: globalAIOpcVideoCapability([]string{"480p", "720p"}, []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9", "adaptive"}, 4, 30, 30, 10, 10)},
		{ModelID: "sd_2.5_special_v1", Name: "Seedance 2.5 Special V1", SourceURL: docsBase + "sd_2.5_special_v1", Capability: globalAIOpcVideoCapability([]string{"720p", "1080p"}, []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9", "adaptive"}, 4, 30, 30, 10, 10)},
		{ModelID: "sd_2.0_fast_special", Name: "Seedance 2.0 Fast Special", SourceURL: docsBase + "sd_2.0_fast_special", Capability: globalAIOpcVideoCapability([]string{"720p"}, []string{"16:9", "9:16", "1:1", "3:4", "4:3", "21:9", "adaptive"}, 4, 15, 9, 3, 3)},
		{ModelID: "sd_2.0_special", Name: "Seedance 2.0 Special", SourceURL: docsBase + "sd_2.0_special", Capability: globalAIOpcVideoCapability([]string{"720p", "1080p", "2k", "4k"}, []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9", "adaptive"}, 4, 15, 9, 3, 3)},
		{ModelID: "sd_2.0_discount", Name: "Seedance 2.0 Discount", SourceURL: docsBase + "sd_2.0_discount", Capability: globalAIOpcVideoCapability([]string{"480p", "720p", "1080p"}, []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9", "adaptive"}, 4, 15, 9, 3, 3)},
		{ModelID: "sd_2.0_fast_discount", Name: "Seedance 2.0 Fast Discount", SourceURL: docsBase + "sd_2.0_fast_discount", Capability: globalAIOpcVideoCapability([]string{"480p", "720p"}, []string{"16:9", "9:16", "1:1", "adaptive", "4:3", "3:4", "21:9"}, 4, 15, 9, 3, 3)},
		{ModelID: "videos_933_c1", Name: "Videos 933 c1", SourceURL: docsBase + "videos_933_c1", Capability: globalAIOpcVideoCapability([]string{"480p", "720p", "1080p"}, []string{"21:9", "16:9", "4:3", "1:1", "3:4", "9:16"}, 4, 15, 9, 3, 3)},
		{ModelID: "videos_fast_933_c1", Name: "Videos Fast 933 c1", SourceURL: docsBase + "videos_fast_933_c1", Capability: globalAIOpcVideoCapability([]string{"480p", "720p"}, []string{"21:9", "16:9", "4:3", "1:1", "3:4", "9:16"}, 4, 15, 9, 3, 3)},
		{ModelID: "videos_stable", Name: "Videos Stable", SourceURL: docsBase + "videos_stable", Capability: globalAIOpcVideoCapability([]string{"720p"}, []string{"9:16", "16:9", "1:1"}, 4, 15, 4, 3, 1)},
		{ModelID: "videos_stable_fast", Name: "Videos Stable Fast", SourceURL: docsBase + "videos_stable_fast", Capability: globalAIOpcVideoCapability([]string{"720p"}, []string{"9:16", "16:9", "1:1"}, 4, 15, 4, 3, 1)},
	}
	for index := range seeds {
		seeds[index].VideoProtocol = dto.VideoProtocolGlobalAIOpc
	}

	configureGlobalAIOpcTemplateDetails(seeds)
	return seeds
}

func globalAIOpcVideoCapability(resolutions, ratios []string, minDuration, maxDuration, maxImages, maxVideos, maxAudios int) dto.VideoModelCapability {
	zero := 0
	falseValue := false
	trueValue := true
	capability := dto.VideoModelCapability{
		Resolutions:                           resolutions,
		Ratios:                                ratios,
		ResolutionMappings:                    map[string]string{},
		RatioRequired:                         &falseValue,
		MinReferenceImages:                    &zero,
		MaxReferenceImages:                    common.GetPointer(maxImages),
		MinReferenceVideos:                    &zero,
		MaxReferenceVideos:                    common.GetPointer(maxVideos),
		MinReferenceAudios:                    &zero,
		MaxReferenceAudios:                    common.GetPointer(maxAudios),
		SupportsDuration:                      &trueValue,
		DurationRequired:                      &trueValue,
		MinDurationSeconds:                    common.GetPointer(minDuration),
		MaxDurationSeconds:                    common.GetPointer(maxDuration),
		SupportsGenerateAudio:                 &falseValue,
		GenerateAudioRequired:                 &falseValue,
		SupportsFirstFrame:                    &falseValue,
		FirstFrameRequired:                    &falseValue,
		SupportsLastFrame:                     &falseValue,
		LastFrameRequired:                     &falseValue,
		LastFrameRequiresFirstFrame:           &falseValue,
		ReferenceImagesIncompatibleWithFrames: &falseValue,
		AudioReferenceRequiresVisualReference: &falseValue,
		ReferenceMediaIncompatibleWithFrames:  &falseValue,
		SupportsSeed:                          &falseValue,
		SupportsWatermark:                     &falseValue,
		AutoReferenceMode:                     &falseValue,
		FramesAsReferenceImages:               &falseValue,
		FixedParameters:                       map[string]any{},
	}
	if minDuration == 0 && maxDuration == 0 {
		capability.SupportsDuration = &falseValue
		capability.DurationRequired = &falseValue
		capability.MinDurationSeconds = nil
		capability.MaxDurationSeconds = nil
		capability.OmitParameters = []string{"duration"}
	}
	return capability
}

func configureGlobalAIOpcTemplateDetails(seeds []videoCapabilityTemplateSeed) {
	for index := range seeds {
		capability := &seeds[index].Capability
		trueValue := true
		falseValue := false
		switch seeds[index].ModelID {
		case "minimax-h3":
			capability.ResolutionMappings = map[string]string{"1440p": "2k"}
			capability.RatioRequired = &trueValue
			capability.SupportsGenerateAudio = &trueValue
			capability.GenerateAudioRequired = &trueValue
			capability.SupportsFirstFrame = &trueValue
			capability.SupportsLastFrame = &trueValue
			capability.AudioReferenceRequiresVisualReference = &trueValue
		case "MiniMax-H3-c4":
			capability.ResolutionMappings = map[string]string{"1440p": "1440P"}
			capability.SupportsFirstFrame = &trueValue
			capability.SupportsLastFrame = &trueValue
			capability.LastFrameRequiresFirstFrame = &trueValue
			capability.ReferenceMediaIncompatibleWithFrames = &trueValue
			capability.AudioReferenceRequiresVisualReference = &trueValue
		case "hh-1.1-r2v-o", "wan2.7-r2v":
			capability.ResolutionMappings = map[string]string{"720p": "720P", "1080p": "1080P"}
			capability.MinReferenceImages = common.GetPointer(1)
			capability.SupportsSeed = &trueValue
			capability.MinSeed = common.GetPointer[int64](0)
			capability.MaxSeed = common.GetPointer[int64](2147483647)
			capability.SupportsWatermark = &trueValue
		case "hh-1.1-i2v-o":
			capability.ResolutionMappings = map[string]string{"720p": "720P", "1080p": "1080P"}
			capability.SupportsFirstFrame = &trueValue
			capability.FirstFrameRequired = &trueValue
			capability.SupportsSeed = &trueValue
			capability.MinSeed = common.GetPointer[int64](0)
			capability.MaxSeed = common.GetPointer[int64](2147483647)
			capability.SupportsWatermark = &trueValue
		case "wan2.7-i2v":
			capability.ResolutionMappings = map[string]string{"720p": "720P", "1080p": "1080P"}
			capability.SupportsFirstFrame = &trueValue
			capability.FirstFrameRequired = &trueValue
			capability.SupportsLastFrame = &trueValue
			capability.LastFrameRequired = &trueValue
			capability.SupportsSeed = &trueValue
			capability.MinSeed = common.GetPointer[int64](0)
			capability.MaxSeed = common.GetPointer[int64](2147483647)
			capability.SupportsWatermark = &trueValue
		case "hh-1.1-t2v-o", "wan2.7-t2v":
			capability.ResolutionMappings = map[string]string{"720p": "720P", "1080p": "1080P"}
			capability.SupportsSeed = &trueValue
			capability.MinSeed = common.GetPointer[int64](0)
			capability.MaxSeed = common.GetPointer[int64](2147483647)
			capability.SupportsWatermark = &trueValue
		case "wan2.7-videoedit":
			capability.ResolutionMappings = map[string]string{"720p": "720P", "1080p": "1080P"}
			capability.MinReferenceVideos = common.GetPointer(1)
			capability.SupportsSeed = &trueValue
			capability.MinSeed = common.GetPointer[int64](0)
			capability.MaxSeed = common.GetPointer[int64](2147483647)
			capability.SupportsWatermark = &trueValue
		case "KlingO3":
			capability.SupportsGenerateAudio = &trueValue
			capability.SupportsFirstFrame = &trueValue
			capability.SupportsLastFrame = &trueValue
		case "seedance-2.5-c1":
			capability.SupportsFirstFrame = &trueValue
			capability.SupportsLastFrame = &trueValue
		case "seedance-2.5-c3":
			capability.SupportsGenerateAudio = &trueValue
			capability.SupportsFirstFrame = &trueValue
			capability.SupportsLastFrame = &trueValue
			capability.ReferenceMediaIncompatibleWithFrames = &trueValue
		case "sd_2.5_discount_v1", "sd_2.5_special_v1":
			capability.SupportsGenerateAudio = &trueValue
			capability.SupportsSeed = &trueValue
			capability.MinSeed = common.GetPointer[int64](-1)
			capability.MaxSeed = common.GetPointer[int64](2147483647)
			capability.AssetPreparationMode = dto.VideoAssetPreparationGlobalAIOpcSeedance
		case "sd_2.0_fast_special", "sd_2.0_special", "sd_2.0_discount", "sd_2.0_fast_discount":
			capability.SupportsGenerateAudio = &trueValue
			capability.SupportsFirstFrame = &trueValue
			capability.SupportsLastFrame = &trueValue
			capability.ReferenceMediaIncompatibleWithFrames = &trueValue
			capability.AudioReferenceRequiresVisualReference = &trueValue
			capability.SupportsSeed = &trueValue
			capability.MinSeed = common.GetPointer[int64](-1)
			capability.MaxSeed = common.GetPointer[int64](2147483647)
			capability.SupportsWatermark = &trueValue
		case "videos_933_c1", "videos_fast_933_c1":
			capability.SupportsGenerateAudio = &trueValue
			capability.SupportsFirstFrame = &trueValue
			capability.SupportsLastFrame = &trueValue
			capability.ReferenceMediaIncompatibleWithFrames = &trueValue
			capability.AutoReferenceMode = &trueValue
			capability.ReferenceModeForReferences = "image"
			capability.ReferenceModeForFrames = "frame"
			capability.FramesAsReferenceImages = &trueValue
			capability.FixedParameters = map[string]any{"face_processing": true}
		case "videos_stable", "videos_stable_fast":
			capability.SupportsFirstFrame = &trueValue
			capability.SupportsLastFrame = &trueValue
		}
		if len(capability.Ratios) == 0 {
			capability.RatioRequired = &falseValue
		}
	}
}
