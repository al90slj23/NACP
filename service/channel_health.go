/*
【文件职责】NACP 渠道健康状态机 — 管理渠道健康状态转换
【核心架构】
  - 五状态模型：Healthy → Probing → Degraded → Recovering → Healthy
  - 内存优先 + 异步 DB 持久化
  - 独立于现有 Channel.Status 字段（共存策略）
【主要函数】
  - InitChannelHealthStates: 启动时从 DB 加载状态
  - GetChannelHealthStatus: 读取渠道健康状态
  - IsChannelRoutable: 判断渠道是否可路由
  - OnUserRequestError: 用户请求失败时的状态转换
  - OnUserRequestSuccess: 用户请求成功时的恢复观察
  - OnProbeResult: 探测结果驱动的状态转换
  - ShouldProbeOnError: 判断是否应该触发探测
  - CheckRecoveryTimers: 定时检查恢复观察期
  - StartHealthManagement: 启动入口
【依赖关系】
  - common: 常量、日志、工具函数
  - model: Channel 数据模型、DB 操作
  - gopool: 异步任务
*/
package service

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/bytedance/gopkg/util/gopool"
)

// ─── Health Status Type ───────────────────────────────────────────────────────

// HealthStatus represents the health state of a channel.
type HealthStatus string

const (
	HealthStatusHealthy    HealthStatus = "healthy"
	HealthStatusProbing    HealthStatus = "probing"
	HealthStatusDegraded   HealthStatus = "degraded"
	HealthStatusRecovering HealthStatus = "recovering"
	HealthStatusDisabled   HealthStatus = "disabled" // manually disabled
)

// IsRoutable returns true if the channel should participate in request routing.
func (s HealthStatus) IsRoutable() bool {
	return s == HealthStatusHealthy || s == HealthStatusRecovering || s == HealthStatusProbing
}

// ─── Per-Channel Health State ─────────────────────────────────────────────────

// ChannelHealthState holds the runtime health state for a single channel.
type ChannelHealthState struct {
	mu                sync.RWMutex
	Status            HealthStatus
	FailCount         int
	SuccessCount      int
	UpdatedAt         time.Time
	RecoveryStartedAt time.Time
	LastProbeAt       time.Time
}

// ─── Global State Map ─────────────────────────────────────────────────────────

var (
	channelHealthStates   = make(map[int]*ChannelHealthState)
	channelHealthStatesMu sync.RWMutex
)

// ─── Initialization ───────────────────────────────────────────────────────────

// InitChannelHealthStates loads persisted health states from DB into memory.
// Called once at startup after InitChannelCache.
func InitChannelHealthStates() {
	var channels []*model.Channel
	model.DB.Select("id, health_status, health_fail_count, health_success_count, health_updated_at").Find(&channels)

	channelHealthStatesMu.Lock()
	defer channelHealthStatesMu.Unlock()

	for _, ch := range channels {
		status := HealthStatus(ch.HealthStatus)
		if status == "" {
			status = HealthStatusHealthy
		}
		// ManuallyDisabled channels map to our disabled state
		if ch.Status == common.ChannelStatusManuallyDisabled {
			status = HealthStatusDisabled
		}
		channelHealthStates[ch.Id] = &ChannelHealthState{
			Status:       status,
			FailCount:    ch.HealthFailCount,
			SuccessCount: ch.HealthSuccessCount,
			UpdatedAt:    time.Unix(ch.HealthUpdatedAt, 0),
		}
	}
	common.SysLog(fmt.Sprintf("NACP: loaded health states for %d channels", len(channels)))
}

// StartHealthManagement initializes the health system and starts background goroutines.
// Called once at application startup, after InitChannelCache.
func StartHealthManagement() {
	InitChannelHealthStates()
	initProbeHTTPClient()

	if common.IsMasterNode {
		// Recovery timer checker — every 30 seconds
		gopool.Go(func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				CheckRecoveryTimers()
			}
		})
		// Degraded probe loop — periodic auxiliary recovery
		StartDegradedProbeLoop()
	}
	common.SysLog("NACP: health management started")
}

// ─── Public Read API ──────────────────────────────────────────────────────────

// GetChannelHealthStatus returns the current health status for a channel.
// Returns HealthStatusHealthy if channel not found (safe default).
func GetChannelHealthStatus(channelID int) HealthStatus {
	channelHealthStatesMu.RLock()
	state, exists := channelHealthStates[channelID]
	channelHealthStatesMu.RUnlock()

	if !exists {
		return HealthStatusHealthy
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.Status
}

// IsChannelRoutable returns true if the channel should participate in routing.
func IsChannelRoutable(channelID int) bool {
	return GetChannelHealthStatus(channelID).IsRoutable()
}

// ─── State Transition: User Request Error ─────────────────────────────────────

// OnUserRequestError is called when a user request to a channel fails.
// Drives state transitions based on error type.
func OnUserRequestError(channelID int, statusCode int, errMsg string) {
	// 400 errors are parameter issues, not channel health issues
	if statusCode == 400 {
		return
	}
	// Timeout errors — don't change health state
	if statusCode == 504 || statusCode == 524 || statusCode == 408 {
		return
	}

	state := getOrCreateHealthState(channelID)
	state.mu.Lock()
	defer state.mu.Unlock()

	switch state.Status {
	case HealthStatusHealthy:
		// Check for immediate degradation (auth failures with disable keywords)
		if statusCode == 401 && matchesDisableKeyword(errMsg) {
			transitionTo(channelID, state, HealthStatusDegraded)
			return
		}
		// 401 without disable keyword — auth issue, not transient, don't probe
		if statusCode == 401 {
			return
		}
		// Normal error → enter Probing state
		state.FailCount = 1
		transitionTo(channelID, state, HealthStatusProbing)

	case HealthStatusProbing:
		state.FailCount++
		cfg := GetHealthConfig()
		if state.FailCount >= cfg.ProbeFailThreshold {
			transitionTo(channelID, state, HealthStatusDegraded)
		}

	case HealthStatusRecovering:
		// Any error during recovery → back to Degraded
		state.SuccessCount = 0
		transitionTo(channelID, state, HealthStatusDegraded)

	case HealthStatusDegraded:
		// Already degraded, just increment fail count
		state.FailCount++

	case HealthStatusDisabled:
		// No automatic transitions for manually disabled channels
	}
}

// ─── State Transition: User Request Success ───────────────────────────────────

// OnUserRequestSuccess is called when a user request to a channel succeeds.
func OnUserRequestSuccess(channelID int) {
	channelHealthStatesMu.RLock()
	state, exists := channelHealthStates[channelID]
	channelHealthStatesMu.RUnlock()

	if !exists {
		return
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	switch state.Status {
	case HealthStatusProbing:
		// User request succeeded while probing → back to Healthy
		state.FailCount = 0
		state.SuccessCount = 0
		transitionTo(channelID, state, HealthStatusHealthy)

	case HealthStatusRecovering:
		// Check if observation period has elapsed
		cfg := GetHealthConfig()
		if time.Since(state.RecoveryStartedAt) >= cfg.RecoveryObservationPeriod {
			transitionTo(channelID, state, HealthStatusHealthy)
		}
		// else: still in observation, success is good but don't promote yet
	}
}

// ─── State Transition: Probe Result ───────────────────────────────────────────

// OnProbeResult is called when a lightweight probe completes.
// Drives transitions for Degraded channels (periodic recovery probing).
func OnProbeResult(channelID int, success bool) {
	state := getOrCreateHealthState(channelID)
	state.mu.Lock()
	defer state.mu.Unlock()

	state.LastProbeAt = time.Now()

	if success {
		switch state.Status {
		case HealthStatusProbing:
			state.FailCount = 0
			state.SuccessCount = 0
			transitionTo(channelID, state, HealthStatusHealthy)

		case HealthStatusDegraded:
			state.SuccessCount++
			cfg := GetHealthConfig()
			if state.SuccessCount >= cfg.RecoverySuccessThreshold {
				state.SuccessCount = 0
				state.RecoveryStartedAt = time.Now()
				transitionTo(channelID, state, HealthStatusRecovering)
			}
		}
	} else {
		switch state.Status {
		case HealthStatusProbing:
			state.FailCount++
			cfg := GetHealthConfig()
			if state.FailCount >= cfg.ProbeFailThreshold {
				transitionTo(channelID, state, HealthStatusDegraded)
			}

		case HealthStatusDegraded:
			// Still degraded, reset success counter
			state.SuccessCount = 0

		case HealthStatusRecovering:
			// Probe failure during recovery → back to Degraded
			state.SuccessCount = 0
			transitionTo(channelID, state, HealthStatusDegraded)
		}
	}
}

// ─── Decision Helpers ─────────────────────────────────────────────────────────

// ShouldProbeOnError determines if a probe should be triggered for this error type.
// Returns false for errors that are not channel health issues.
func ShouldProbeOnError(statusCode int) bool {
	switch statusCode {
	case 400: // parameter error
		return false
	case 401: // auth error — handled separately
		return false
	case 408: // request timeout (client-side)
		return false
	case 504, 524: // gateway timeout
		return false
	default:
		return true
	}
}

// ─── Recovery Timer ───────────────────────────────────────────────────────────

// CheckRecoveryTimers promotes Recovering channels that have completed their observation period.
// Called periodically by a background goroutine.
func CheckRecoveryTimers() {
	cfg := GetHealthConfig()

	channelHealthStatesMu.RLock()
	var candidates []int
	for id, state := range channelHealthStates {
		state.mu.RLock()
		if state.Status == HealthStatusRecovering {
			if time.Since(state.RecoveryStartedAt) >= cfg.RecoveryObservationPeriod {
				candidates = append(candidates, id)
			}
		}
		state.mu.RUnlock()
	}
	channelHealthStatesMu.RUnlock()

	// Promote candidates
	for _, id := range candidates {
		channelHealthStatesMu.RLock()
		state, exists := channelHealthStates[id]
		channelHealthStatesMu.RUnlock()
		if !exists {
			continue
		}
		state.mu.Lock()
		// Double-check after acquiring write lock
		if state.Status == HealthStatusRecovering {
			if time.Since(state.RecoveryStartedAt) >= cfg.RecoveryObservationPeriod {
				transitionTo(id, state, HealthStatusHealthy)
			}
		}
		state.mu.Unlock()
	}
}

// ─── Internal Helpers ─────────────────────────────────────────────────────────

// getOrCreateHealthState returns the health state for a channel, creating one if needed.
func getOrCreateHealthState(channelID int) *ChannelHealthState {
	channelHealthStatesMu.RLock()
	state, exists := channelHealthStates[channelID]
	channelHealthStatesMu.RUnlock()

	if exists {
		return state
	}

	// Create new state
	channelHealthStatesMu.Lock()
	defer channelHealthStatesMu.Unlock()

	// Double-check after acquiring write lock
	if state, exists = channelHealthStates[channelID]; exists {
		return state
	}

	state = &ChannelHealthState{
		Status:    HealthStatusHealthy,
		UpdatedAt: time.Now(),
	}
	channelHealthStates[channelID] = state
	return state
}

// transitionTo performs the state transition. Caller must hold state.mu.Lock().
func transitionTo(channelID int, state *ChannelHealthState, newStatus HealthStatus) {
	oldStatus := state.Status
	if oldStatus == newStatus {
		return
	}

	state.Status = newStatus
	state.UpdatedAt = time.Now()

	// Log the transition
	common.SysLog(fmt.Sprintf("NACP: channel #%d health transition: %s → %s (fail=%d, success=%d)",
		channelID, oldStatus, newStatus, state.FailCount, state.SuccessCount))

	// Async persist to DB
	failCount := state.FailCount
	successCount := state.SuccessCount
	updatedAt := state.UpdatedAt.Unix()
	gopool.Go(func() {
		persistHealthState(channelID, newStatus, failCount, successCount, updatedAt)
	})

	// Check low channel availability after degradation
	if newStatus == HealthStatusDegraded {
		gopool.Go(func() {
			checkLowChannelAvailability(channelID)
		})
	}
}

// persistHealthState writes health state to the database asynchronously.
func persistHealthState(channelID int, status HealthStatus, failCount int, successCount int, updatedAt int64) {
	err := model.DB.Model(&model.Channel{}).Where("id = ?", channelID).Updates(map[string]interface{}{
		"health_status":        string(status),
		"health_fail_count":    failCount,
		"health_success_count": successCount,
		"health_updated_at":    updatedAt,
	}).Error
	if err != nil {
		common.SysError(fmt.Sprintf("NACP: failed to persist health state for channel #%d: %v", channelID, err))
	}
}

// matchesDisableKeyword checks if the error message contains any auto-disable keyword.
func matchesDisableKeyword(errMsg string) bool {
	lowerMsg := strings.ToLower(errMsg)
	for _, keyword := range operation_setting.AutomaticDisableKeywords {
		if strings.Contains(lowerMsg, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// checkLowChannelAvailability checks if healthy channel count is critically low
// for any group+model combination involving this channel.
func checkLowChannelAvailability(channelID int) {
	channel, err := model.CacheGetChannel(channelID)
	if err != nil || channel == nil {
		return
	}

	cfg := GetHealthConfig()
	groups := channel.GetGroups()
	models := channel.GetModels()

	for _, group := range groups {
		for _, modelName := range models {
			healthyCount := countHealthyChannelsForGroupModel(group, modelName)
			if healthyCount <= 0 {
				emitHealthAlert("critical", channelID, group, modelName, healthyCount)
			} else if healthyCount <= cfg.LowChannelWarningThreshold {
				emitHealthAlert("warning", channelID, group, modelName, healthyCount)
			}
		}
	}
}

// countHealthyChannelsForGroupModel counts channels that are both enabled and healthy/recovering.
func countHealthyChannelsForGroupModel(group string, modelName string) int {
	// Use the existing channel cache to count
	channelHealthStatesMu.RLock()
	defer channelHealthStatesMu.RUnlock()

	count := 0
	for id, state := range channelHealthStates {
		state.mu.RLock()
		isRoutable := state.Status.IsRoutable()
		state.mu.RUnlock()

		if !isRoutable {
			continue
		}

		// Check if this channel serves this group+model
		ch, err := model.CacheGetChannel(id)
		if err != nil || ch == nil {
			continue
		}
		if ch.Status != common.ChannelStatusEnabled {
			continue
		}
		// Check group membership
		chGroups := ch.GetGroups()
		groupMatch := false
		for _, g := range chGroups {
			if g == group {
				groupMatch = true
				break
			}
		}
		if !groupMatch {
			continue
		}
		// Check model support
		chModels := ch.GetModels()
		modelMatch := false
		for _, m := range chModels {
			if m == modelName {
				modelMatch = true
				break
			}
		}
		if modelMatch {
			count++
		}
	}
	return count
}

// emitHealthAlert sends a notification about channel health changes.
func emitHealthAlert(level string, channelID int, group string, modelName string, healthyCount int) {
	var subject, content string

	switch level {
	case "warning":
		subject = fmt.Sprintf("⚠️ 分组 %s 模型 %s 可用渠道不足（剩余 %d）", group, modelName, healthyCount)
		content = fmt.Sprintf("分组「%s」模型「%s」的健康渠道仅剩 %d 个，请关注。\n渠道 #%d 状态变更触发此告警。", group, modelName, healthyCount, channelID)
	case "critical":
		subject = fmt.Sprintf("🚨 分组 %s 模型 %s 无可用渠道", group, modelName)
		content = fmt.Sprintf("分组「%s」模型「%s」的所有渠道均已降级或禁用，服务可能中断！", group, modelName)
	default:
		return
	}

	notifyType := fmt.Sprintf("channel_health_%s_%s_%s", level, group, modelName)
	NotifyRootUser(notifyType, subject, content)
}
