package model

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

type cachedChannelAbility struct {
	ChannelId int
	Priority  int64
	Weight    uint
}

var group2model2channels map[string]map[string][]cachedChannelAbility // enabled ability entries
var channelsIDM map[int]*Channel                                      // all channels include disabled
var channelSyncLock sync.RWMutex

func InitChannelCache() {
	if !common.MemoryCacheEnabled {
		return
	}
	newChannelId2channel := make(map[int]*Channel)
	var channels []*Channel
	DB.Find(&channels)
	for _, channel := range channels {
		newChannelId2channel[channel.Id] = channel
	}
	var abilities []*Ability
	DB.Find(&abilities)
	newGroup2model2channels := make(map[string]map[string][]cachedChannelAbility)
	for _, ability := range abilities {
		if !ability.Enabled {
			continue
		}
		group := strings.TrimSpace(ability.Group)
		model := strings.TrimSpace(ability.Model)
		if group == "" || model == "" || ability.ChannelId <= 0 {
			continue
		}
		channel, ok := newChannelId2channel[ability.ChannelId]
		if !ok || channel.Status != common.ChannelStatusEnabled {
			continue
		}
		if _, ok := newGroup2model2channels[group]; !ok {
			newGroup2model2channels[group] = make(map[string][]cachedChannelAbility)
		}
		priority := channel.GetPriority()
		if ability.Priority != nil {
			priority = *ability.Priority
		}
		newGroup2model2channels[group][model] = append(newGroup2model2channels[group][model], cachedChannelAbility{
			ChannelId: ability.ChannelId,
			Priority:  priority,
			Weight:    ability.Weight,
		})
	}

	// Sort for deterministic sequential failover: group/model ability priority first, then weight.
	for group, model2channels := range newGroup2model2channels {
		for model, entries := range model2channels {
			sort.Slice(entries, func(i, j int) bool {
				left := entries[i]
				right := entries[j]
				if left.Priority != right.Priority {
					return left.Priority > right.Priority
				}
				if left.Weight != right.Weight {
					return left.Weight > right.Weight
				}
				return left.ChannelId < right.ChannelId
			})
			newGroup2model2channels[group][model] = entries
		}
	}

	channelSyncLock.Lock()
	group2model2channels = newGroup2model2channels
	//channelsIDM = newChannelId2channel
	for i, channel := range newChannelId2channel {
		if channel.ChannelInfo.IsMultiKey {
			channel.Keys = channel.GetKeys()
			if channel.ChannelInfo.MultiKeyMode == constant.MultiKeyModePolling {
				if oldChannel, ok := channelsIDM[i]; ok {
					// 存在旧的渠道，如果是多key且轮询，保留轮询索引信息
					if oldChannel.ChannelInfo.IsMultiKey && oldChannel.ChannelInfo.MultiKeyMode == constant.MultiKeyModePolling {
						channel.ChannelInfo.MultiKeyPollingIndex = oldChannel.ChannelInfo.MultiKeyPollingIndex
					}
				}
			}
		}
	}
	channelsIDM = newChannelId2channel
	channelSyncLock.Unlock()
	common.SysLog("channels synced from database")
}

func SyncChannelCache(frequency int) {
	for {
		time.Sleep(time.Duration(frequency) * time.Second)
		common.SysLog("syncing channels from database")
		InitChannelCache()
	}
}

func GetRandomSatisfiedChannel(group string, model string, retry int, excludeIDs ...map[int]bool) (*Channel, error) {
	// if memory cache is disabled, get channel directly from database
	if !common.MemoryCacheEnabled {
		return GetChannel(group, model, retry)
	}

	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	// First, try to find channels with the exact model name.
	channels := group2model2channels[group][model]

	// If no channels found, try to find channels with the normalized model name.
	if len(channels) == 0 {
		normalizedModel := ratio_setting.FormatMatchingModelName(model)
		channels = group2model2channels[group][normalizedModel]
	}

	if len(channels) == 0 {
		return nil, nil
	}

	// Build exclusion set from variadic parameter
	var excluded map[int]bool
	if len(excludeIDs) > 0 && excludeIDs[0] != nil {
		excluded = excludeIDs[0]
	}

	if len(channels) == 1 {
		chID := channels[0].ChannelId
		// Check exclusion and health
		if excluded != nil && excluded[chID] {
			return nil, nil
		}
		if channel, ok := channelsIDM[chID]; ok {
			if channel.HealthStatus == common.ChannelHealthStatusDegraded {
				return nil, nil
			}
			return channel, nil
		}
		return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", chID)
	}

	uniquePriorities := make(map[int]bool)
	for _, entry := range channels {
		if _, ok := channelsIDM[entry.ChannelId]; ok {
			uniquePriorities[int(entry.Priority)] = true
		} else {
			return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", entry.ChannelId)
		}
	}
	var sortedUniquePriorities []int
	for priority := range uniquePriorities {
		sortedUniquePriorities = append(sortedUniquePriorities, priority)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sortedUniquePriorities)))

	if retry >= len(uniquePriorities) {
		retry = len(uniquePriorities) - 1
	}
	targetPriority := int64(sortedUniquePriorities[retry])

	// get the priority for the given retry number
	var sumWeight = 0
	var targetChannels []cachedChannelAbility
	for _, entry := range channels {
		if channel, ok := channelsIDM[entry.ChannelId]; ok {
			if entry.Priority == targetPriority {
				// NACP: Skip degraded channels
				if channel.HealthStatus == common.ChannelHealthStatusDegraded {
					continue
				}
				// NACP: Skip excluded channels (same-priority retry)
				if excluded != nil && excluded[channel.Id] {
					continue
				}
				sumWeight += int(entry.Weight)
				targetChannels = append(targetChannels, entry)
			}
		} else {
			return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", entry.ChannelId)
		}
	}

	// NACP: Fallback — if all channels filtered out, retry with unfiltered list
	if len(targetChannels) == 0 && excluded == nil {
		// Try without health/exclusion filter as safety net
		for _, entry := range channels {
			if _, ok := channelsIDM[entry.ChannelId]; ok {
				if entry.Priority == targetPriority {
					sumWeight += int(entry.Weight)
					targetChannels = append(targetChannels, entry)
				}
			}
		}
		if len(targetChannels) > 0 {
			common.SysLog(fmt.Sprintf("NACP WARNING: all channels filtered by health/exclusion for group=%s model=%s priority=%d, falling back to unfiltered", group, model, targetPriority))
		}
	}

	if len(targetChannels) == 0 {
		return nil, errors.New(fmt.Sprintf("no channel found, group: %s, model: %s, priority: %d", group, model, targetPriority))
	}

	// smoothing factor and adjustment
	smoothingFactor := 1
	smoothingAdjustment := 0

	if sumWeight == 0 {
		// when all channels have weight 0, set sumWeight to the number of channels and set smoothing adjustment to 100
		// each channel's effective weight = 100
		sumWeight = len(targetChannels) * 100
		smoothingAdjustment = 100
	} else if sumWeight/len(targetChannels) < 10 {
		// when the average weight is less than 10, set smoothing factor to 100
		smoothingFactor = 100
	}

	// Calculate the total weight of all channels up to endIdx
	totalWeight := sumWeight * smoothingFactor

	// Generate a random value in the range [0, totalWeight)
	randomWeight := rand.Intn(totalWeight)

	// Find a channel based on its weight
	for _, entry := range targetChannels {
		randomWeight -= int(entry.Weight)*smoothingFactor + smoothingAdjustment
		if randomWeight < 0 {
			channel, ok := channelsIDM[entry.ChannelId]
			if !ok {
				return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", entry.ChannelId)
			}
			return channel, nil
		}
	}
	// return null if no channel is not found
	return nil, errors.New("channel not found")
}

// GetNextSatisfiedChannel returns the next enabled channel in priority order,
// excluding channels that were already tried in the current relay chain. The
// sequential failover chain intentionally does not skip degraded channels: the
// user request itself is the source of truth for this chain.
func GetNextSatisfiedChannel(group string, model string, excluded map[int]bool) (*Channel, error) {
	if !common.MemoryCacheEnabled {
		return GetNextChannel(group, model, excluded)
	}

	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	channels := group2model2channels[group][model]
	if len(channels) == 0 {
		normalizedModel := ratio_setting.FormatMatchingModelName(model)
		channels = group2model2channels[group][normalizedModel]
	}
	if len(channels) == 0 {
		return nil, nil
	}

	for _, entry := range channels {
		if excluded != nil && excluded[entry.ChannelId] {
			continue
		}
		channel, ok := channelsIDM[entry.ChannelId]
		if !ok {
			return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", entry.ChannelId)
		}
		if channel.Status != common.ChannelStatusEnabled {
			continue
		}
		return channel, nil
	}

	return nil, nil
}

func CacheGetChannel(id int) (*Channel, error) {
	if !common.MemoryCacheEnabled {
		return GetChannelById(id, true)
	}
	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	c, ok := channelsIDM[id]
	if !ok {
		return nil, fmt.Errorf("渠道# %d，已不存在", id)
	}
	return c, nil
}

func CacheGetChannelInfo(id int) (*ChannelInfo, error) {
	if !common.MemoryCacheEnabled {
		channel, err := GetChannelById(id, true)
		if err != nil {
			return nil, err
		}
		return &channel.ChannelInfo, nil
	}
	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	c, ok := channelsIDM[id]
	if !ok {
		return nil, fmt.Errorf("渠道# %d，已不存在", id)
	}
	return &c.ChannelInfo, nil
}

func CacheUpdateChannelStatus(id int, status int) {
	if !common.MemoryCacheEnabled {
		return
	}
	channelSyncLock.Lock()
	defer channelSyncLock.Unlock()
	if channel, ok := channelsIDM[id]; ok {
		channel.Status = status
	}
	if status != common.ChannelStatusEnabled {
		// delete the channel from group2model2channels
		for group, model2channels := range group2model2channels {
			for model, channels := range model2channels {
				for i, entry := range channels {
					if entry.ChannelId == id {
						// remove the channel from the slice
						group2model2channels[group][model] = append(channels[:i], channels[i+1:]...)
						break
					}
				}
			}
		}
	}
}

func CacheUpdateChannel(channel *Channel) {
	if !common.MemoryCacheEnabled {
		return
	}
	channelSyncLock.Lock()
	defer channelSyncLock.Unlock()
	if channel == nil {
		return
	}

	println("CacheUpdateChannel:", channel.Id, channel.Name, channel.Status, channel.ChannelInfo.MultiKeyPollingIndex)

	println("before:", channelsIDM[channel.Id].ChannelInfo.MultiKeyPollingIndex)
	channelsIDM[channel.Id] = channel
	println("after :", channelsIDM[channel.Id].ChannelInfo.MultiKeyPollingIndex)
}
