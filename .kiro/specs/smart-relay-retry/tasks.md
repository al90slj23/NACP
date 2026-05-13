# Implementation Plan: Smart Relay Retry

## Overview

This plan implements the Smart Relay Retry system — a channel health state machine with lightweight probing, same-priority channel switching, and enhanced retry logic. Tasks are ordered by dependency: data model first, then core logic, then integration into the existing relay path.

All new logic lives in new files (`service/channel_health_config.go`, `service/channel_health.go`, `service/channel_probe.go`). Existing files receive minimal, additive changes (new struct fields, filter additions, retry loop enhancement).

## Tasks

- [ ] 1. Data model and schema changes
  - [ ] 1.1 Add health status constants to `common/constants.go`
    - Add `ChannelHealthStatusHealthy`, `ChannelHealthStatusProbing`, `ChannelHealthStatusDegraded`, `ChannelHealthStatusRecovering`, `ChannelHealthStatusDisabled` string constants
    - These are purely additive — no existing code is affected
    - _Requirements: 7.1, 1.1_

  - [ ] 1.2 Add health fields to `model/channel.go` Channel struct
    - Add `HealthStatus string`, `HealthUpdatedAt int64`, `HealthFailCount int`, `HealthSuccessCount int` with GORM tags
    - Fields use `gorm:"type:varchar(16);default:'healthy'"` etc. for cross-DB compatibility
    - GORM AutoMigrate handles column addition on all three databases (SQLite, MySQL, PostgreSQL)
    - Verify no callers are broken (additive struct change — safe in Go)
    - _Requirements: 7.1, 7.3_

  - [ ]* 1.3 Write unit tests for Channel struct health field defaults
    - Verify new Channel instances have `HealthStatus == "healthy"` by default
    - Verify GORM tag parsing produces correct column definitions
    - _Requirements: 7.1_

- [ ] 2. Health configuration
  - [ ] 2.1 Create `service/channel_health_config.go` with configuration struct and defaults
    - Define `ChannelHealthConfig` struct with all threshold fields
    - Implement `DefaultHealthConfig()` returning hardcoded defaults
    - Implement `GetHealthConfig()` returning the global config singleton
    - Implement `UpdateHealthConfig()` for future admin UI integration
    - Implement `HealthConfigToMap()` for API response serialization (use `common.Marshal`)
    - _Requirements: 1.4, 1.7, 1.8, 1.11, 2.4, 3.3_

  - [ ]* 2.2 Write unit tests for health configuration
    - Test `DefaultHealthConfig()` returns expected default values
    - Test `UpdateHealthConfig()` updates the global config
    - Test `HealthConfigToMap()` produces correct map structure
    - _Requirements: 1.4, 1.7, 1.8, 2.4, 3.3_

- [ ] 3. Health state machine core logic
  - [ ] 3.1 Create `service/channel_health.go` — types and state storage
    - Define `HealthStatus` type and constants (`HealthStatusHealthy`, `HealthStatusProbing`, etc.)
    - Define `IsRoutable()` and `IsHealthy()` methods on `HealthStatus`
    - Define `ChannelHealthState` struct with `sync.RWMutex`, status, counters, timestamps
    - Define `channelHealthStates` map with its own `sync.RWMutex`
    - Implement `getOrCreateHealthState(channelID)` helper
    - _Requirements: 1.1, 1.13_

  - [ ] 3.2 Implement `InitChannelHealthStates()` — load persisted state from DB
    - Query all channels from DB, populate `channelHealthStates` map from `HealthStatus`, `HealthFailCount`, `HealthSuccessCount`, `HealthUpdatedAt` fields
    - Called once at startup after `InitChannelCache()`
    - _Requirements: 7.4, 1.13_

  - [ ] 3.3 Implement `GetChannelHealthStatus()` and `IsChannelRoutable()`
    - Thread-safe read of channel health status
    - Return `HealthStatusHealthy` if channel not found in map (safe default)
    - `IsChannelRoutable` returns true for Healthy, Probing, Recovering
    - _Requirements: 1.1, 1.6, 1.9_

  - [ ] 3.4 Implement `OnUserRequestError()` — state transitions on error
    - Healthy + non-400/non-timeout error → Probing (set FailCount=1)
    - Healthy + 401 with disable keyword → Degraded (immediate)
    - Probing + error → increment FailCount, if >= threshold → Degraded
    - Recovering + any error → Degraded (reset SuccessCount)
    - Degraded + error → increment FailCount (no further transition)
    - ManuallyDisabled → no-op
    - Implement `ShouldProbeOnError(statusCode)` — returns false for 400, 401, 408, 504, 524
    - _Requirements: 1.2, 1.4, 1.10, 1.12, 4.5, 4.7, 4.8_

  - [ ] 3.5 Implement `OnProbeResult()` — state transitions on probe result
    - Probing + success → Healthy (reset counters)
    - Probing + fail → increment FailCount, if >= threshold → Degraded
    - Degraded + success → increment SuccessCount, if >= recovery threshold → Recovering (set RecoveryStartedAt)
    - Degraded + fail → reset SuccessCount
    - Recovering + fail → Degraded (reset SuccessCount)
    - _Requirements: 1.5, 1.4, 1.8_

  - [ ] 3.6 Implement `OnUserRequestSuccess()` — recovery observation tracking
    - Recovering + success + observation period elapsed → Healthy
    - Probing + success → Healthy (user request succeeded while probing)
    - Other states → no-op
    - _Requirements: 1.11, 1.9_

  - [ ] 3.7 Implement `transitionTo()` and `persistHealthState()`
    - `transitionTo`: update in-memory state, log transition via `common.SysLog`
    - `persistHealthState`: async DB write via `gopool.Go()` — update `HealthStatus`, `HealthUpdatedAt`, `HealthFailCount`, `HealthSuccessCount` on channels table
    - Use GORM `Updates()` for cross-DB compatibility
    - _Requirements: 1.13, 7.2_

  - [ ]* 3.8 Write unit tests for state machine transitions
    - **Property 1: State Machine Validity** — verify only valid states are produced
    - **Property 2: Transition Determinism** — same input always produces same output
    - **Validates: Requirements 1.1, 1.2, 1.4, 1.5, 1.8, 1.10, 1.11, 1.12**

  - [ ]* 3.9 Write unit tests for recovery safety
    - **Property 5: Recovery Safety** — verify Degraded never transitions directly to Healthy
    - **Property 6: Manual Override Immunity** — verify ManuallyDisabled has no automatic transitions
    - **Validates: Requirements 1.8, 1.11, 1.12**

- [ ] 4. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 5. Lightweight probe implementation (parallel pre-warming)
  - [ ] 5.1 Create `service/channel_probe.go` — types and HTTP client
    - Define `ProbeResult` struct (ChannelID, ModelName, Success, StatusCode, LatencyMs, Error, Timestamp)
    - Define `ProbeLog` struct for cost tracking
    - Implement `initProbeHTTPClient()` with dedicated `*http.Client` (timeout from config, separate transport)
    - _Requirements: 3.1, 3.3_

  - [ ] 5.2 Implement `ProbeChannel()` — synchronous probe execution
    - Build minimal request body: `{"model":"X","messages":[{"role":"user","content":"hi"}],"max_tokens":1,"stream":false}` using `common.Marshal`
    - Determine probe endpoint based on channel type (most use `/v1/chat/completions`)
    - Set Authorization header from channel key
    - Execute HTTP POST with context timeout (3s)
    - Return `ProbeResult` based on HTTP status code (2xx = success)
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6_

  - [ ] 5.3 Implement `ProbeNextChannels()` — parallel pre-warming of standby channels
    - Accept list of candidate channels (B, C, ...) and model name
    - Launch goroutines to probe each candidate concurrently
    - Return ordered list of `ProbeResult` (preserving priority/weight order)
    - Used by retry loop: while retrying A with user request, simultaneously probe B and C
    - Results cached in context for immediate use when switching channels
    - _Requirements: 3.1, 3.2_

  - [ ] 5.4 Implement `buildProbeRequest()` helper functions
    - `getProbeEndpoint(channel)` — map channel type to probe URL path
    - `getProbeAuthHeader(channel)` — extract auth header from channel key
    - `buildProbeRequestBody(modelName)` — construct minimal JSON body via `common.Marshal`
    - _Requirements: 3.1, 3.2, 3.5_

  - [ ] 5.5 Implement `recordProbeLog()` — probe cost recording
    - Record probe execution with channel ID, model name, success/failure, latency
    - Use `user_id=0` to ensure no billing to any user account
    - Mark with distinct probe type identifier in the `other` JSON field
    - _Requirements: 5.1, 5.2, 5.3_

  - [ ]* 5.6 Write unit tests for probe implementation
    - Test `buildProbeRequestBody` produces valid minimal JSON
    - Test `getProbeEndpoint` returns correct paths for different channel types
    - Test `ProbeNextChannels` returns results in priority order
    - Test probe logs have user_id=0 and probe type marker
    - _Requirements: 3.1, 3.2, 5.1, 5.2_

- [ ] 6. Channel affinity health-awareness
  - [ ] 6.1 Modify `middleware/distributor.go` — skip affinity for Degraded channels
    - In the affinity check block (where `GetPreferredChannelByAffinity` is called), add health status check
    - If preferred channel's `health_status == "degraded"` → skip affinity, let normal channel selection proceed
    - Do NOT set `SkipRetryOnFailure` flag when skipping due to health (allow retry on other channels)
    - Existing `SwitchOnSuccess: true` will update affinity cache to the new successful channel
    - _Requirements: 1.6, 4.7_

  - [ ]* 6.2 Write unit tests for affinity + health interaction
    - Test: affinity channel healthy → use affinity channel
    - Test: affinity channel degraded → skip affinity, use normal selection
    - Test: after skip, successful request updates affinity cache to new channel
    - _Requirements: 1.6_

- [ ] 7. Channel selection enhancement
  - [ ] 7.1 Modify `model/channel_cache.go` `GetRandomSatisfiedChannel` — add health filtering and exclusion
    - Add variadic parameter `excludeIDs ...map[int]bool` to function signature (backward compatible)
    - After building `targetChannels` slice, filter out channels with `HealthStatus == "degraded"`
    - Filter out channels whose IDs are in the `excludeIDs` map
    - If no candidates remain at current priority, try next priority level (recursive or iterative)
    - _Requirements: 1.6, 2.1, 2.3_

  - [ ] 7.2 Modify `service/channel_select.go` `CacheGetRandomSatisfiedChannel` — pass exclusion context
    - Build `excludeIDs` map from `use_channel` context list (already tracked per-request)
    - Pass `excludeIDs` to `model.GetRandomSatisfiedChannel` calls (both auto-group and normal paths)
    - Ensure auto-group cross-group retry logic is not disrupted
    - _Requirements: 2.1, 2.2, 2.3, 2.5_

  - [ ]* 7.3 Write unit tests for health-aware channel selection
    - **Property 3: Degraded Channel Exclusion** — verify degraded channels are never returned
    - **Property 7: Same-Priority Exhaustion** — verify max same-priority attempts are bounded
    - **Property 8: Backward Compatibility** — verify default healthy channels behave identically to current
    - **Validates: Requirements 1.6, 2.1, 2.3, 2.4, 2.5**

- [ ] 8. Enhanced retry loop in relay controller
  - [ ] 8.1 Modify `controller/relay.go` `Relay()` — implement enhanced retry loop with parallel pre-warming
    - Track `excludedChannelIDs` map (channels tried this request)
    - Track `samePriorityAttempts` counter and `currentPriorityRetry` level
    - On first error from channel A:
      1. **Don't return error to client**
      2. **Parallel fork:**
         - [Main] Retry user request to A (up to 2 more times, configurable `same_channel_retry_count`)
         - [Pre-warm] Simultaneously call `ProbeNextChannels(B, C)` to pre-check standby channels
      3. **If A retry succeeds** → return to client, discard pre-warm results
      4. **If A retries all fail** → check pre-warm results:
         - Pick first successful standby channel → send user request there
         - If all standby probes failed → return error to client
    - Error classification still applies:
      - 400 → switch channel immediately (no same-channel retry, parameter issue)
      - 401 + disable keyword → immediate Degraded, switch channel
      - 504/524/408 → don't retry
      - Other (429, 500, 502, 503) → same-channel retry + parallel pre-warm
    - On success: call `OnUserRequestSuccess(channelID)` for recovery observation tracking
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 4.7, 4.8, 2.1, 2.4, 3.1_

  - [ ] 8.2 Modify `controller/relay.go` `processChannelError()` — integrate health state machine
    - Add call to `service.OnUserRequestError(channelID, statusCode, errMsg)` at the beginning
    - Keep existing `ShouldDisableChannel` + `DisableChannel` logic (coexistence strategy)
    - The health state machine and auto-disable mechanism operate on separate fields
    - _Requirements: 1.2, 4.5, 4.8_

  - [ ]* 8.3 Write unit tests for enhanced retry loop logic
    - Test 400 error → switch channel immediately without same-channel retry
    - Test 502 error → retry A twice → A succeeds on 2nd try → return success
    - Test 502 error → retry A twice → A fails → pre-warm B success → send to B → success
    - Test 502 error → retry A fails → pre-warm B fails, C success → send to C → success
    - Test all channels exhausted → return last error to client
    - Test success after retry → OnUserRequestSuccess called
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 4.7_

- [ ] 9. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 10. Low-channel availability alerts
  - [ ] 10.1 Implement `checkLowChannelAvailability()` in `service/channel_health.go`
    - After each state transition to Degraded, count healthy+recovering channels for the affected group+model combinations
    - When count <= `LowChannelWarningThreshold` (default 2): emit warning via `NotifyRootUser`
    - When count == 0: emit critical alert
    - Include group name, model name, and remaining healthy count in alert message
    - _Requirements: 6.1, 6.2, 6.3_

  - [ ] 10.2 Implement `emitHealthAlert()` helper
    - Format alert subject and content with channel ID, group, model, healthy count
    - Use existing `service.NotifyRootUser()` infrastructure
    - Use distinct notify type to avoid duplicate alerts: `"channel_health_{level}_{group}_{model}"`
    - _Requirements: 6.1, 6.2, 6.3_

  - [ ]* 10.3 Write unit tests for low-channel alerts
    - Test warning emitted when healthy count drops to threshold
    - Test critical emitted when all channels degraded
    - Test alert includes correct group, model, and count information
    - _Requirements: 6.1, 6.2, 6.3_

- [ ] 11. Background goroutines
  - [ ] 11.1 Implement `CheckRecoveryTimers()` — periodic recovery promotion
    - Iterate all Recovering channels, check if observation period has elapsed
    - Promote to Healthy if period completed without errors
    - Use double-check locking pattern (RLock to find candidates, Lock to promote)
    - _Requirements: 1.11_

  - [ ] 11.2 Implement `StartDegradedProbeLoop()` — periodic degraded channel probing
    - Only run on master node (`common.IsMasterNode`)
    - Ticker at `DegradedProbeInterval` (default 5 minutes)
    - Iterate all Degraded channels, probe each sequentially with 500ms gap
    - Feed results to `OnProbeResult()` and record probe logs
    - Use channel's test model or first model for probe
    - This is the auxiliary recovery mechanism (main recovery is via user traffic)
    - _Requirements: 1.7, 3.1_

  - [ ] 11.3 Implement `StartHealthManagement()` — initialization entry point
    - Call `InitChannelHealthStates()` to load persisted state
    - Call `initProbeHTTPClient()` to set up dedicated HTTP client
    - Start recovery timer checker goroutine (every 30s) via `gopool.Go()`
    - Start degraded probe loop (master node only)
    - This function is called once at application startup
    - _Requirements: 7.4, 1.7, 1.11_

  - [ ]* 11.4 Write unit tests for background goroutine logic
    - Test `CheckRecoveryTimers` promotes channels past observation period
    - Test `CheckRecoveryTimers` does not promote channels still in observation
    - Test degraded probe loop respects `DegradedProbeInterval`
    - _Requirements: 1.7, 1.11_

- [ ] 12. Integration and initialization
  - [ ] 12.1 Wire `StartHealthManagement()` into application startup
    - Add call to `service.StartHealthManagement()` in the main initialization sequence (after `model.InitChannelCache()`)
    - Ensure it runs after DB migration and channel cache initialization
    - _Requirements: 7.4_

  - [ ] 12.2 Ensure `InitChannelCache` does NOT overwrite health state
    - Verify that periodic `SyncChannelCache` only syncs channel configuration (models, keys, priority, weight)
    - Health state fields in `channelsIDM` are managed exclusively by the health state machine
    - If `InitChannelCache` overwrites the full channel object, preserve health fields from the health state map
    - _Requirements: 7.2, 1.13_

  - [ ] 12.3 Handle interaction with existing auto-disable mechanism
    - Ensure `processChannelError` still calls existing `ShouldDisableChannel` + `DisableChannel` for severe errors
    - Health state machine operates on `health_status` field; auto-disable operates on `Status` field
    - A channel must have `Status == ChannelStatusEnabled` AND `health_status != "degraded"` to be routable
    - Document coexistence strategy in code comments
    - _Requirements: 4.8, 1.6_

  - [ ]* 12.4 Write integration tests for full request flow
    - Test happy path: request succeeds, no health changes
    - Test error → probe success → retry same channel → success
    - Test error → probe fail → switch channel → success
    - Test degraded channel excluded from routing
    - Test recovery flow: degraded → probe successes → recovering → observation → healthy
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 1.5, 1.8, 1.11_

- [ ] 13. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document
- Unit tests validate specific examples and edge cases
- All JSON operations use `common/json.go` wrappers (Rule 1)
- All DB changes are cross-compatible with SQLite, MySQL, PostgreSQL via GORM AutoMigrate (Rule 2)
- New logic lives in new files; existing files receive minimal additive changes
- The health state machine coexists with the existing auto-disable mechanism (separate fields)
- Background goroutines use `gopool.Go()` consistent with existing project patterns

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "1.2", "2.1"] },
    { "id": 1, "tasks": ["1.3", "2.2", "3.1"] },
    { "id": 2, "tasks": ["3.2", "3.3", "5.1"] },
    { "id": 3, "tasks": ["3.4", "3.5", "3.6", "5.4"] },
    { "id": 4, "tasks": ["3.7", "5.2", "5.3"] },
    { "id": 5, "tasks": ["3.8", "3.9", "5.5", "6.1"] },
    { "id": 6, "tasks": ["6.2", "7.1"] },
    { "id": 7, "tasks": ["7.2", "7.3"] },
    { "id": 8, "tasks": ["8.1", "8.2"] },
    { "id": 9, "tasks": ["8.3", "10.1", "10.2"] },
    { "id": 10, "tasks": ["10.3", "11.1", "11.2"] },
    { "id": 11, "tasks": ["11.3", "11.4"] },
    { "id": 12, "tasks": ["12.1", "12.2", "12.3"] },
    { "id": 13, "tasks": ["12.4"] }
  ]
}
```
