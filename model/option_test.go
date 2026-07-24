package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestLoadOptionsUsesLegacyUsableGroupDescriptionsWhenSeparateOptionIsMissing(t *testing.T) {
	originalDB := DB
	originalUserUsableGroups := setting.UserUsableGroups2JSONString()
	common.OptionMapRWMutex.Lock()
	originalOptionMap := common.OptionMap
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		DB = originalDB
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUserUsableGroups))
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Option{}))
	DB = db

	legacyDescriptions := `{"default":"Custom default note","hidden":"Hidden note"}`
	require.NoError(t, DB.Create(&Option{
		Key:   "UserUsableGroups",
		Value: legacyDescriptions,
	}).Error)
	common.OptionMapRWMutex.Lock()
	common.OptionMap = map[string]string{
		"UserUsableGroups":  `{"default":"Default note"}`,
		"GroupDescriptions": `{"default":"Default note"}`,
	}
	common.OptionMapRWMutex.Unlock()

	loadOptionsFromDatabase()

	common.OptionMapRWMutex.RLock()
	groupDescriptions := common.OptionMap["GroupDescriptions"]
	common.OptionMapRWMutex.RUnlock()
	assert.JSONEq(t, legacyDescriptions, groupDescriptions)
}

func TestLoadOptionsKeepsExplicitGroupDescriptions(t *testing.T) {
	originalDB := DB
	originalUserUsableGroups := setting.UserUsableGroups2JSONString()
	common.OptionMapRWMutex.Lock()
	originalOptionMap := common.OptionMap
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		DB = originalDB
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUserUsableGroups))
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Option{}))
	DB = db

	require.NoError(t, DB.Create(&[]Option{
		{Key: "UserUsableGroups", Value: `{"default":"Selectable note"}`},
		{Key: "GroupDescriptions", Value: `{"default":"Saved note","hidden":"Hidden note"}`},
	}).Error)
	common.OptionMapRWMutex.Lock()
	common.OptionMap = map[string]string{
		"UserUsableGroups":  `{"default":"Default note"}`,
		"GroupDescriptions": `{"default":"Default note"}`,
	}
	common.OptionMapRWMutex.Unlock()

	loadOptionsFromDatabase()

	common.OptionMapRWMutex.RLock()
	groupDescriptions := common.OptionMap["GroupDescriptions"]
	common.OptionMapRWMutex.RUnlock()
	assert.JSONEq(t, `{"default":"Saved note","hidden":"Hidden note"}`, groupDescriptions)
}

func TestUpdateUserUsableGroupsMergesSelectableDescriptionsIntoSeparateOption(t *testing.T) {
	originalDB := DB
	originalUserUsableGroups := setting.UserUsableGroups2JSONString()
	common.OptionMapRWMutex.Lock()
	originalOptionMap := common.OptionMap
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		DB = originalDB
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUserUsableGroups))
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Option{}))
	DB = db
	common.OptionMapRWMutex.Lock()
	common.OptionMap = map[string]string{
		"UserUsableGroups":  `{"default":"Old selectable note"}`,
		"GroupDescriptions": `{"default":"Old saved note","hidden":"Hidden note"}`,
	}
	common.OptionMapRWMutex.Unlock()

	require.NoError(t, UpdateOption(
		"UserUsableGroups",
		`{"default":"Updated selectable note","vip":"VIP note"}`,
	))

	common.OptionMapRWMutex.RLock()
	userUsableGroups := common.OptionMap["UserUsableGroups"]
	groupDescriptions := common.OptionMap["GroupDescriptions"]
	common.OptionMapRWMutex.RUnlock()
	assert.JSONEq(t, `{"default":"Updated selectable note","vip":"VIP note"}`, userUsableGroups)
	assert.JSONEq(t, `{"default":"Updated selectable note","hidden":"Hidden note","vip":"VIP note"}`, groupDescriptions)

	var savedOptions []Option
	require.NoError(t, DB.Where("key IN ?", []string{"UserUsableGroups", "GroupDescriptions"}).Find(&savedOptions).Error)
	savedByKey := make(map[string]string, len(savedOptions))
	for _, option := range savedOptions {
		savedByKey[option.Key] = option.Value
	}
	assert.JSONEq(t, userUsableGroups, savedByKey["UserUsableGroups"])
	assert.JSONEq(t, groupDescriptions, savedByKey["GroupDescriptions"])
}

func TestUpdateUserUsableGroupsHandlesNullSeparateDescriptions(t *testing.T) {
	originalDB := DB
	originalUserUsableGroups := setting.UserUsableGroups2JSONString()
	common.OptionMapRWMutex.Lock()
	originalOptionMap := common.OptionMap
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		DB = originalDB
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUserUsableGroups))
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Option{}))
	DB = db
	common.OptionMapRWMutex.Lock()
	common.OptionMap = map[string]string{
		"UserUsableGroups":  `{}`,
		"GroupDescriptions": `null`,
	}
	common.OptionMapRWMutex.Unlock()

	require.NoError(t, UpdateOption("UserUsableGroups", `{"default":"Default note"}`))

	common.OptionMapRWMutex.RLock()
	groupDescriptions := common.OptionMap["GroupDescriptions"]
	common.OptionMapRWMutex.RUnlock()
	assert.JSONEq(t, `{"default":"Default note"}`, groupDescriptions)
}
