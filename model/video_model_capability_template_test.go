package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBuiltInGlobalAIOpcCapabilityTemplatesAreValid(t *testing.T) {
	seeds := builtInGlobalAIOpcCapabilityTemplates()
	require.Len(t, seeds, 22)
	assetModels := make(map[string]dto.VideoModelCapability)
	for _, seed := range seeds {
		t.Run(seed.ModelID, func(t *testing.T) {
			settings := dto.ChannelSettings{
				VideoProtocol: seed.VideoProtocol,
				VideoModelCapabilities: map[string]dto.VideoModelCapability{
					seed.ModelID: seed.Capability,
				},
			}
			require.NoError(t, settings.ValidateVideoRequestSettings())
		})
		if seed.Capability.AssetPreparationMode != "" {
			assetModels[seed.ModelID] = seed.Capability
		}
	}
	require.Len(t, assetModels, 2)
	for _, modelID := range []string{"sd_2.5_discount_v1", "sd_2.5_special_v1"} {
		capability, ok := assetModels[modelID]
		require.True(t, ok)
		assert.Equal(t, dto.VideoAssetPreparationGlobalAIOpcSeedance, capability.AssetPreparationMode)
	}
}

func TestVideoModelCapabilityTemplatesSeedListAndCustomize(t *testing.T) {
	originalDB := DB
	t.Cleanup(func() { DB = originalDB })

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	require.NoError(t, DB.AutoMigrate(&VideoModelCapabilityTemplate{}))
	require.NoError(t, SeedVideoModelCapabilityTemplates())

	templates, err := ListVideoModelCapabilityTemplates(dto.VideoProtocolGlobalAIOpc)
	require.NoError(t, err)
	require.Len(t, templates, 22)
	var builtInTemplate VideoModelCapabilityTemplate
	for _, template := range templates {
		if template.ModelID == "minimax-h3" {
			builtInTemplate = template
			break
		}
	}
	require.NotZero(t, builtInTemplate.ID)
	require.NoError(t, DB.Model(&VideoModelCapabilityTemplate{}).
		Where("id = ?", builtInTemplate.ID).
		Updates(map[string]any{"name": "Stale built-in template", "capability_json": `{}`}).Error)
	require.NoError(t, SeedVideoModelCapabilityTemplates())
	templates, err = ListVideoModelCapabilityTemplates(dto.VideoProtocolGlobalAIOpc)
	require.NoError(t, err)
	for _, template := range templates {
		if template.ModelID == "minimax-h3" {
			assert.Equal(t, "MiniMax H3", template.Name)
			assert.NotEqual(t, `{}`, template.CapabilityJSON)
			break
		}
	}

	capability := builtInGlobalAIOpcCapabilityTemplates()[0].Capability
	capability.MaxReferenceImages = common.GetPointer(2)
	capabilityJSON, err := common.Marshal(capability)
	require.NoError(t, err)
	err = SaveVideoModelCapabilityTemplate(&VideoModelCapabilityTemplate{
		VideoProtocol:  dto.VideoProtocolGlobalAIOpc,
		ModelID:        "minimax-h3",
		Name:           "Customized MiniMax H3",
		CapabilityJSON: string(capabilityJSON),
		Source:         "manual",
	})
	require.EqualError(t, err, "a built-in video model capability template already exists for this protocol and model ID")

	customTemplate := &VideoModelCapabilityTemplate{
		VideoProtocol:  dto.VideoProtocolGlobalAIOpc,
		ModelID:        "custom-minimax-h3",
		Name:           "Customized MiniMax H3",
		CapabilityJSON: string(capabilityJSON),
		Source:         "manual",
	}
	require.NoError(t, SaveVideoModelCapabilityTemplate(customTemplate))
	require.NoError(t, SeedVideoModelCapabilityTemplates())

	templates, err = ListVideoModelCapabilityTemplates(dto.VideoProtocolGlobalAIOpc)
	require.NoError(t, err)
	require.Len(t, templates, 23)
	for _, template := range templates {
		if template.ModelID != "custom-minimax-h3" {
			continue
		}
		assert.Equal(t, "Customized MiniMax H3", template.Name)
		assert.False(t, template.BuiltIn)
		return
	}
	t.Fatal("customized template not found")
}

func TestVideoModelCapabilityTemplateDeleteProtectsBuiltInTemplates(t *testing.T) {
	originalDB := DB
	t.Cleanup(func() { DB = originalDB })

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	require.NoError(t, DB.AutoMigrate(&VideoModelCapabilityTemplate{}))
	require.NoError(t, SeedVideoModelCapabilityTemplates())

	templates, err := ListVideoModelCapabilityTemplates(dto.VideoProtocolGlobalAIOpc)
	require.NoError(t, err)
	require.NotEmpty(t, templates)
	require.EqualError(t, DeleteVideoModelCapabilityTemplate(templates[0].ID), "built-in video model capability templates cannot be deleted")

	capabilityJSON, err := common.Marshal(builtInGlobalAIOpcCapabilityTemplates()[0].Capability)
	require.NoError(t, err)
	custom := &VideoModelCapabilityTemplate{
		VideoProtocol:  dto.VideoProtocolGlobalAIOpc,
		ModelID:        "deletable-template",
		Name:           "Deletable template",
		CapabilityJSON: string(capabilityJSON),
		Source:         "manual",
	}
	require.NoError(t, SaveVideoModelCapabilityTemplate(custom))
	require.NoError(t, DeleteVideoModelCapabilityTemplate(custom.ID))

	var count int64
	require.NoError(t, DB.Model(&VideoModelCapabilityTemplate{}).Where("id = ?", custom.ID).Count(&count).Error)
	assert.Zero(t, count)
}
