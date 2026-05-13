/*
【文件职责】NACP 渠道健康状态机配置
【核心架构】定义所有健康状态机的可配置阈值参数
【主要函数】DefaultHealthConfig, GetHealthConfig, UpdateHealthConfig
【依赖关系】无外部依赖，纯配置定义
*/
package service

import "time"

// ChannelHealthConfig holds all configurable thresholds for the health state machine.
// Phase 1: hardcoded defaults. Phase 2: admin UI configurable.
type ChannelHealthConfig struct {
	// SameChannelRetryCount: how many times to retry the SAME channel with user request before switching
	SameChannelRetryCount int // default: 2

	// ProbeFailThreshold: consecutive probe failures to transition Probing → Degraded
	ProbeFailThreshold int // default: 2

	// DegradedProbeInterval: how often to probe a Degraded channel (auxiliary recovery)
	DegradedProbeInterval time.Duration // default: 5 minutes

	// RecoverySuccessThreshold: consecutive probe successes to transition Degraded → Recovering
	RecoverySuccessThreshold int // default: 3

	// RecoveryObservationPeriod: duration without errors to transition Recovering → Healthy
	RecoveryObservationPeriod time.Duration // default: 10 minutes

	// RecoveryFailTolerance: errors allowed in Recovering before going back to Degraded
	// 0 means any error immediately goes back to Degraded
	RecoveryFailTolerance int // default: 0

	// ProbeTimeout: max time for a single lightweight probe HTTP request
	ProbeTimeout time.Duration // default: 3 seconds

	// MaxRetrySamePriority: max different channels to try at the same priority level
	MaxRetrySamePriority int // default: 3

	// PreWarmChannelCount: how many standby channels to pre-warm in parallel
	PreWarmChannelCount int // default: 2

	// LowChannelWarningThreshold: emit warning when healthy channels <= this count
	LowChannelWarningThreshold int // default: 2
}

// DefaultHealthConfig returns the default configuration with sensible defaults.
func DefaultHealthConfig() *ChannelHealthConfig {
	return &ChannelHealthConfig{
		SameChannelRetryCount:      2,
		ProbeFailThreshold:         2,
		DegradedProbeInterval:      5 * time.Minute,
		RecoverySuccessThreshold:   3,
		RecoveryObservationPeriod:  10 * time.Minute,
		RecoveryFailTolerance:      0,
		ProbeTimeout:               3 * time.Second,
		MaxRetrySamePriority:       3,
		PreWarmChannelCount:        2,
		LowChannelWarningThreshold: 2,
	}
}

// globalHealthConfig is the active configuration singleton.
var globalHealthConfig = DefaultHealthConfig()

// GetHealthConfig returns the current health configuration.
func GetHealthConfig() *ChannelHealthConfig {
	return globalHealthConfig
}

// UpdateHealthConfig updates the global health configuration.
// Thread-safe for future admin UI integration.
func UpdateHealthConfig(config *ChannelHealthConfig) {
	if config != nil {
		globalHealthConfig = config
	}
}
