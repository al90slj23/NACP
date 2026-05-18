package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func resetChannelGroupConfigTestTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.Exec("DELETE FROM abilities").Error)
	require.NoError(t, DB.Exec("DELETE FROM channel_group_configs").Error)
	require.NoError(t, DB.Exec("DELETE FROM channels").Error)
	t.Cleanup(func() {
		_ = DB.Exec("DELETE FROM abilities").Error
		_ = DB.Exec("DELETE FROM channel_group_configs").Error
		_ = DB.Exec("DELETE FROM channels").Error
	})
}

func TestChannelGroupConfigsCreatePerGroupAbilities(t *testing.T) {
	resetChannelGroupConfigTestTables(t)

	priorityA := int64(100)
	priorityB := int64(20)
	weightA := uint(80)
	weightB := uint(5)
	channel := Channel{
		Type:     1,
		Key:      "sk-test",
		Status:   common.ChannelStatusEnabled,
		Name:     "group-config-create",
		Models:   "gpt-a,gpt-b",
		Group:    "alpha,beta",
		Priority: common.GetPointer[int64](0),
		Weight:   common.GetPointer[uint](0),
		GroupConfigs: []ChannelGroupConfig{
			{Group: "alpha", Priority: &priorityA, Weight: &weightA},
			{Group: "beta", Priority: &priorityB, Weight: &weightB},
		},
	}
	require.NoError(t, channel.Insert())

	var abilities []Ability
	require.NoError(t, DB.Where("channel_id = ?", channel.Id).Find(&abilities).Error)
	require.Len(t, abilities, 4)
	for _, ability := range abilities {
		switch ability.Group {
		case "alpha":
			require.NotNil(t, ability.Priority)
			require.Equal(t, priorityA, *ability.Priority)
			require.Equal(t, weightA, ability.Weight)
		case "beta":
			require.NotNil(t, ability.Priority)
			require.Equal(t, priorityB, *ability.Priority)
			require.Equal(t, weightB, ability.Weight)
		default:
			t.Fatalf("unexpected group %q", ability.Group)
		}
	}
}

func TestChannelGroupConfigsUpdatePreservesExistingPerGroupAbilities(t *testing.T) {
	resetChannelGroupConfigTestTables(t)

	priorityA := int64(100)
	priorityB := int64(20)
	weightA := uint(80)
	weightB := uint(5)
	channel := Channel{
		Type:     1,
		Key:      "sk-test",
		Status:   common.ChannelStatusEnabled,
		Name:     "group-config-update",
		Models:   "gpt-a",
		Group:    "alpha,beta",
		Priority: common.GetPointer[int64](0),
		Weight:   common.GetPointer[uint](0),
		GroupConfigs: []ChannelGroupConfig{
			{Group: "alpha", Priority: &priorityA, Weight: &weightA},
			{Group: "beta", Priority: &priorityB, Weight: &weightB},
		},
	}
	require.NoError(t, channel.Insert())

	loaded, err := GetChannelById(channel.Id, true)
	require.NoError(t, err)
	loaded.Models = "gpt-a,gpt-c"
	loaded.GroupConfigs = nil
	require.NoError(t, loaded.Update())

	var beta Ability
	require.NoError(t, DB.Where("channel_id = ? AND "+commonGroupCol+" = ? AND model = ?", channel.Id, "beta", "gpt-c").First(&beta).Error)
	require.NotNil(t, beta.Priority)
	require.Equal(t, priorityB, *beta.Priority)
	require.Equal(t, weightB, beta.Weight)
}

func TestChannelGroupConfigsSurviveAbilityRebuild(t *testing.T) {
	resetChannelGroupConfigTestTables(t)

	priorityA := int64(100)
	priorityB := int64(20)
	weightA := uint(80)
	weightB := uint(5)
	channel := Channel{
		Type:     1,
		Key:      "sk-test",
		Status:   common.ChannelStatusEnabled,
		Name:     "group-config-rebuild",
		Models:   "gpt-a",
		Group:    "alpha,beta",
		Priority: common.GetPointer[int64](0),
		Weight:   common.GetPointer[uint](0),
		GroupConfigs: []ChannelGroupConfig{
			{Group: "alpha", Priority: &priorityA, Weight: &weightA},
			{Group: "beta", Priority: &priorityB, Weight: &weightB},
		},
	}
	require.NoError(t, channel.Insert())

	success, failed, err := FixAbility()
	require.NoError(t, err)
	require.Equal(t, 1, success)
	require.Equal(t, 0, failed)

	var alpha Ability
	require.NoError(t, DB.Where("channel_id = ? AND "+commonGroupCol+" = ? AND model = ?", channel.Id, "alpha", "gpt-a").First(&alpha).Error)
	require.NotNil(t, alpha.Priority)
	require.Equal(t, priorityA, *alpha.Priority)
	require.Equal(t, weightA, alpha.Weight)
}

func TestChannelDefaultSchedulingUpdateCreatesLegacyGroupConfigs(t *testing.T) {
	resetChannelGroupConfigTestTables(t)

	channel := Channel{
		Type:     1,
		Key:      "sk-test",
		Status:   common.ChannelStatusEnabled,
		Name:     "group-config-legacy-default",
		Models:   "gpt-a",
		Group:    "alpha,beta",
		Priority: common.GetPointer[int64](0),
		Weight:   common.GetPointer[uint](0),
		Tag:      common.GetPointer[string]("legacy-tag"),
	}
	require.NoError(t, channel.Insert())

	newPriority := int64(42)
	newWeight := uint(9)
	require.NoError(t, EditChannelByTag("legacy-tag", nil, nil, nil, nil, &newPriority, &newWeight, nil, nil))

	var configs []ChannelGroupConfig
	require.NoError(t, DB.Where("channel_id = ?", channel.Id).Find(&configs).Error)
	require.Len(t, configs, 2)
	for _, config := range configs {
		require.NotNil(t, config.Priority)
		require.NotNil(t, config.Weight)
		require.Equal(t, newPriority, *config.Priority)
		require.Equal(t, newWeight, *config.Weight)
	}

	var abilities []Ability
	require.NoError(t, DB.Where("channel_id = ?", channel.Id).Find(&abilities).Error)
	require.Len(t, abilities, 2)
	for _, ability := range abilities {
		require.NotNil(t, ability.Priority)
		require.Equal(t, newPriority, *ability.Priority)
		require.Equal(t, newWeight, ability.Weight)
	}
}

func TestChannelDefaultSchedulingUpdateDoesNotOverwriteExplicitGroupConfigs(t *testing.T) {
	resetChannelGroupConfigTestTables(t)

	priorityA := int64(100)
	priorityB := int64(20)
	weightA := uint(80)
	weightB := uint(5)
	channel := Channel{
		Type:     1,
		Key:      "sk-test",
		Status:   common.ChannelStatusEnabled,
		Name:     "group-config-explicit-default",
		Models:   "gpt-a",
		Group:    "alpha,beta",
		Priority: common.GetPointer[int64](0),
		Weight:   common.GetPointer[uint](0),
		Tag:      common.GetPointer[string]("explicit-tag"),
		GroupConfigs: []ChannelGroupConfig{
			{Group: "alpha", Priority: &priorityA, Weight: &weightA},
			{Group: "beta", Priority: &priorityB, Weight: &weightB},
		},
	}
	require.NoError(t, channel.Insert())

	newPriority := int64(42)
	newWeight := uint(9)
	require.NoError(t, EditChannelByTag("explicit-tag", nil, nil, nil, nil, &newPriority, &newWeight, nil, nil))

	var alpha Ability
	require.NoError(t, DB.Where("channel_id = ? AND "+commonGroupCol+" = ? AND model = ?", channel.Id, "alpha", "gpt-a").First(&alpha).Error)
	require.NotNil(t, alpha.Priority)
	require.Equal(t, priorityA, *alpha.Priority)
	require.Equal(t, weightA, alpha.Weight)

	var beta Ability
	require.NoError(t, DB.Where("channel_id = ? AND "+commonGroupCol+" = ? AND model = ?", channel.Id, "beta", "gpt-a").First(&beta).Error)
	require.NotNil(t, beta.Priority)
	require.Equal(t, priorityB, *beta.Priority)
	require.Equal(t, weightB, beta.Weight)

	var updated Channel
	require.NoError(t, DB.First(&updated, channel.Id).Error)
	require.NotNil(t, updated.Priority)
	require.NotNil(t, updated.Weight)
	require.Equal(t, newPriority, *updated.Priority)
	require.Equal(t, newWeight, *updated.Weight)
}
