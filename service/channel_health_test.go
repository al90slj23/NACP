package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── State Machine Transition Tests ──────────────────────────────────────────

func resetHealthStates() {
	channelHealthStatesMu.Lock()
	channelHealthStates = make(map[int]*ChannelHealthState)
	channelHealthStatesMu.Unlock()
}

func TestHealthStatus_IsRoutable(t *testing.T) {
	assert.True(t, HealthStatusHealthy.IsRoutable())
	assert.True(t, HealthStatusProbing.IsRoutable())
	assert.True(t, HealthStatusRecovering.IsRoutable())
	assert.False(t, HealthStatusDegraded.IsRoutable())
	assert.False(t, HealthStatusDisabled.IsRoutable())
}

func TestOnUserRequestError_HealthyToProbing(t *testing.T) {
	resetHealthStates()
	channelID := 100

	OnUserRequestError(channelID, 502, "bad gateway")

	status := GetChannelHealthStatus(channelID)
	assert.Equal(t, HealthStatusProbing, status)
}

func TestOnUserRequestError_400_NoStateChange(t *testing.T) {
	resetHealthStates()
	channelID := 101

	// Start as healthy
	OnUserRequestError(channelID, 400, "bad request")

	// Should remain healthy — 400 is parameter issue
	status := GetChannelHealthStatus(channelID)
	assert.Equal(t, HealthStatusHealthy, status)
}

func TestOnUserRequestError_504_NoStateChange(t *testing.T) {
	resetHealthStates()
	channelID := 102

	OnUserRequestError(channelID, 504, "gateway timeout")

	status := GetChannelHealthStatus(channelID)
	assert.Equal(t, HealthStatusHealthy, status)
}

func TestOnUserRequestError_401_DisableKeyword_ImmediateDegraded(t *testing.T) {
	resetHealthStates()
	channelID := 103

	OnUserRequestError(channelID, 401, "Your credit balance is too low to continue")

	status := GetChannelHealthStatus(channelID)
	assert.Equal(t, HealthStatusDegraded, status)
}

func TestOnUserRequestError_401_NoKeyword_NoChange(t *testing.T) {
	resetHealthStates()
	channelID := 104

	OnUserRequestError(channelID, 401, "invalid api key")

	// 401 without disable keyword — no state change (auth issue, not transient)
	status := GetChannelHealthStatus(channelID)
	assert.Equal(t, HealthStatusHealthy, status)
}

func TestOnUserRequestError_ProbingToDegraded(t *testing.T) {
	resetHealthStates()
	channelID := 105

	// First error → Probing
	OnUserRequestError(channelID, 502, "bad gateway")
	assert.Equal(t, HealthStatusProbing, GetChannelHealthStatus(channelID))

	// Second error → Degraded (threshold = 2)
	OnUserRequestError(channelID, 502, "bad gateway")
	assert.Equal(t, HealthStatusDegraded, GetChannelHealthStatus(channelID))
}

func TestOnUserRequestError_RecoveringToDegraded(t *testing.T) {
	resetHealthStates()
	channelID := 106

	// Manually set to Recovering
	state := getOrCreateHealthState(channelID)
	state.mu.Lock()
	state.Status = HealthStatusRecovering
	state.RecoveryStartedAt = time.Now()
	state.mu.Unlock()

	// Any error during recovery → back to Degraded
	OnUserRequestError(channelID, 500, "internal error")
	assert.Equal(t, HealthStatusDegraded, GetChannelHealthStatus(channelID))
}

func TestOnUserRequestError_ManuallyDisabled_NoChange(t *testing.T) {
	resetHealthStates()
	channelID := 107

	state := getOrCreateHealthState(channelID)
	state.mu.Lock()
	state.Status = HealthStatusDisabled
	state.mu.Unlock()

	OnUserRequestError(channelID, 502, "bad gateway")
	assert.Equal(t, HealthStatusDisabled, GetChannelHealthStatus(channelID))
}

// ─── Success Handling Tests ──────────────────────────────────────────────────

func TestOnUserRequestSuccess_ProbingToHealthy(t *testing.T) {
	resetHealthStates()
	channelID := 200

	state := getOrCreateHealthState(channelID)
	state.mu.Lock()
	state.Status = HealthStatusProbing
	state.FailCount = 1
	state.mu.Unlock()

	OnUserRequestSuccess(channelID)
	assert.Equal(t, HealthStatusHealthy, GetChannelHealthStatus(channelID))
}

func TestOnUserRequestSuccess_RecoveringToHealthy_AfterObservation(t *testing.T) {
	resetHealthStates()
	channelID := 201

	// Set recovering with observation period already elapsed
	state := getOrCreateHealthState(channelID)
	state.mu.Lock()
	state.Status = HealthStatusRecovering
	state.RecoveryStartedAt = time.Now().Add(-15 * time.Minute) // 15 min ago, threshold is 10 min
	state.mu.Unlock()

	OnUserRequestSuccess(channelID)
	assert.Equal(t, HealthStatusHealthy, GetChannelHealthStatus(channelID))
}

func TestOnUserRequestSuccess_RecoveringStaysRecovering_DuringObservation(t *testing.T) {
	resetHealthStates()
	channelID := 202

	// Set recovering with observation period NOT elapsed
	state := getOrCreateHealthState(channelID)
	state.mu.Lock()
	state.Status = HealthStatusRecovering
	state.RecoveryStartedAt = time.Now().Add(-2 * time.Minute) // 2 min ago, threshold is 10 min
	state.mu.Unlock()

	OnUserRequestSuccess(channelID)
	assert.Equal(t, HealthStatusRecovering, GetChannelHealthStatus(channelID))
}

// ─── Probe Result Tests ──────────────────────────────────────────────────────

func TestOnProbeResult_ProbingSuccess_ToHealthy(t *testing.T) {
	resetHealthStates()
	channelID := 300

	state := getOrCreateHealthState(channelID)
	state.mu.Lock()
	state.Status = HealthStatusProbing
	state.FailCount = 1
	state.mu.Unlock()

	OnProbeResult(channelID, true)
	assert.Equal(t, HealthStatusHealthy, GetChannelHealthStatus(channelID))
}

func TestOnProbeResult_ProbingFail_ToDegraded(t *testing.T) {
	resetHealthStates()
	channelID := 301

	state := getOrCreateHealthState(channelID)
	state.mu.Lock()
	state.Status = HealthStatusProbing
	state.FailCount = 1 // already 1, threshold is 2
	state.mu.Unlock()

	OnProbeResult(channelID, false)
	assert.Equal(t, HealthStatusDegraded, GetChannelHealthStatus(channelID))
}

func TestOnProbeResult_DegradedSuccess_ToRecovering(t *testing.T) {
	resetHealthStates()
	channelID := 302

	state := getOrCreateHealthState(channelID)
	state.mu.Lock()
	state.Status = HealthStatusDegraded
	state.SuccessCount = 2 // need 3 total, this will be the 3rd
	state.mu.Unlock()

	OnProbeResult(channelID, true)
	assert.Equal(t, HealthStatusRecovering, GetChannelHealthStatus(channelID))
}

func TestOnProbeResult_DegradedSuccess_NotEnough(t *testing.T) {
	resetHealthStates()
	channelID := 303

	state := getOrCreateHealthState(channelID)
	state.mu.Lock()
	state.Status = HealthStatusDegraded
	state.SuccessCount = 0 // need 3, this will be 1st
	state.mu.Unlock()

	OnProbeResult(channelID, true)
	assert.Equal(t, HealthStatusDegraded, GetChannelHealthStatus(channelID))

	// Check success count incremented
	state.mu.RLock()
	assert.Equal(t, 1, state.SuccessCount)
	state.mu.RUnlock()
}

func TestOnProbeResult_DegradedFail_ResetsSuccessCount(t *testing.T) {
	resetHealthStates()
	channelID := 304

	state := getOrCreateHealthState(channelID)
	state.mu.Lock()
	state.Status = HealthStatusDegraded
	state.SuccessCount = 2 // was building up
	state.mu.Unlock()

	OnProbeResult(channelID, false)

	state.mu.RLock()
	assert.Equal(t, 0, state.SuccessCount)
	assert.Equal(t, HealthStatusDegraded, state.Status)
	state.mu.RUnlock()
}

func TestOnProbeResult_RecoveringFail_ToDegraded(t *testing.T) {
	resetHealthStates()
	channelID := 305

	state := getOrCreateHealthState(channelID)
	state.mu.Lock()
	state.Status = HealthStatusRecovering
	state.RecoveryStartedAt = time.Now()
	state.mu.Unlock()

	OnProbeResult(channelID, false)
	assert.Equal(t, HealthStatusDegraded, GetChannelHealthStatus(channelID))
}

// ─── ShouldProbeOnError Tests ────────────────────────────────────────────────

func TestShouldProbeOnError(t *testing.T) {
	assert.False(t, ShouldProbeOnError(400))
	assert.False(t, ShouldProbeOnError(401))
	assert.False(t, ShouldProbeOnError(408))
	assert.False(t, ShouldProbeOnError(504))
	assert.False(t, ShouldProbeOnError(524))
	assert.True(t, ShouldProbeOnError(429))
	assert.True(t, ShouldProbeOnError(500))
	assert.True(t, ShouldProbeOnError(502))
	assert.True(t, ShouldProbeOnError(503))
}

// ─── Recovery Timer Tests ────────────────────────────────────────────────────

func TestCheckRecoveryTimers_PromotesExpired(t *testing.T) {
	resetHealthStates()
	channelID := 400

	state := getOrCreateHealthState(channelID)
	state.mu.Lock()
	state.Status = HealthStatusRecovering
	state.RecoveryStartedAt = time.Now().Add(-15 * time.Minute)
	state.mu.Unlock()

	CheckRecoveryTimers()

	assert.Equal(t, HealthStatusHealthy, GetChannelHealthStatus(channelID))
}

func TestCheckRecoveryTimers_DoesNotPromoteActive(t *testing.T) {
	resetHealthStates()
	channelID := 401

	state := getOrCreateHealthState(channelID)
	state.mu.Lock()
	state.Status = HealthStatusRecovering
	state.RecoveryStartedAt = time.Now().Add(-2 * time.Minute)
	state.mu.Unlock()

	CheckRecoveryTimers()

	assert.Equal(t, HealthStatusRecovering, GetChannelHealthStatus(channelID))
}

// ─── Default State Tests ─────────────────────────────────────────────────────

func TestGetChannelHealthStatus_UnknownChannel_ReturnsHealthy(t *testing.T) {
	resetHealthStates()
	status := GetChannelHealthStatus(99999)
	assert.Equal(t, HealthStatusHealthy, status)
}

func TestIsChannelRoutable_UnknownChannel_ReturnsTrue(t *testing.T) {
	resetHealthStates()
	assert.True(t, IsChannelRoutable(99999))
}

// ─── Property Tests ──────────────────────────────────────────────────────────

func TestProperty_NeverDirectDegradedToHealthy(t *testing.T) {
	// Property 5: A channel SHALL not transition from Degraded directly to Healthy
	resetHealthStates()
	channelID := 500

	state := getOrCreateHealthState(channelID)
	state.mu.Lock()
	state.Status = HealthStatusDegraded
	state.mu.Unlock()

	// Success should NOT promote directly to Healthy
	OnUserRequestSuccess(channelID)
	assert.NotEqual(t, HealthStatusHealthy, GetChannelHealthStatus(channelID))

	// Single probe success should NOT promote directly to Healthy
	OnProbeResult(channelID, true)
	status := GetChannelHealthStatus(channelID)
	// Should still be Degraded (need 3 successes to reach Recovering)
	assert.Equal(t, HealthStatusDegraded, status)
}

func TestProperty_ManualDisabledImmune(t *testing.T) {
	// Property 6: ManuallyDisabled has no automatic transitions
	resetHealthStates()
	channelID := 501

	state := getOrCreateHealthState(channelID)
	state.mu.Lock()
	state.Status = HealthStatusDisabled
	state.mu.Unlock()

	OnUserRequestError(channelID, 502, "error")
	assert.Equal(t, HealthStatusDisabled, GetChannelHealthStatus(channelID))

	OnUserRequestSuccess(channelID)
	assert.Equal(t, HealthStatusDisabled, GetChannelHealthStatus(channelID))

	OnProbeResult(channelID, true)
	assert.Equal(t, HealthStatusDisabled, GetChannelHealthStatus(channelID))

	OnProbeResult(channelID, false)
	assert.Equal(t, HealthStatusDisabled, GetChannelHealthStatus(channelID))
}

// ─── Config Tests ────────────────────────────────────────────────────────────

func TestDefaultHealthConfig(t *testing.T) {
	cfg := DefaultHealthConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, 2, cfg.SameChannelRetryCount)
	assert.Equal(t, 2, cfg.ProbeFailThreshold)
	assert.Equal(t, 3, cfg.RecoverySuccessThreshold)
	assert.Equal(t, 3, cfg.MaxRetrySamePriority)
	assert.Equal(t, 2, cfg.PreWarmChannelCount)
	assert.Equal(t, 2, cfg.LowChannelWarningThreshold)
	assert.Equal(t, 3*time.Second, cfg.ProbeTimeout)
	assert.Equal(t, 20*time.Second, cfg.FirstByteTimeout)
	assert.Equal(t, 60*time.Second, cfg.TotalRetryTimeout)
	assert.Equal(t, 5*time.Minute, cfg.DegradedProbeInterval)
	assert.Equal(t, 10*time.Minute, cfg.RecoveryObservationPeriod)
}

func TestGetHealthConfig_ReturnsSingleton(t *testing.T) {
	cfg1 := GetHealthConfig()
	cfg2 := GetHealthConfig()
	assert.Same(t, cfg1, cfg2)
}
