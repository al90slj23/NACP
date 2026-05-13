# Requirements Document

## Introduction

Smart Relay Retry is an intelligent retry and channel health management system for the NACP AI API gateway. It replaces the current simplistic retry mechanism (which only drops to lower priority channels) with a comprehensive solution that includes a channel health state machine, same-priority channel switching, lightweight probing, transparent error interception, probe cost tracking, and low-channel alerting. The goal is to reduce client-visible errors to near zero by intercepting upstream failures and transparently retrying on healthy channels, while maintaining operator visibility into system health and probe costs.

## Glossary

- **Relay_Controller**: The main request relay loop in `controller/relay.go` that handles forwarding client requests to upstream AI providers and managing retries
- **Channel_Selector**: The channel selection subsystem (`service/channel_select.go` + `model/channel_cache.go`) responsible for choosing which upstream channel to use based on group, model, priority, and weight
- **Health_State_Machine**: The new subsystem (`service/channel_health.go`) that manages channel health state transitions (Healthy → Probing → Degraded → Recovering → Healthy)
- **Channel_Prober**: The new lightweight probing subsystem (`service/channel_probe.go`) that sends minimal requests to verify channel connectivity
- **Channel**: An upstream AI provider endpoint configured with API key, base URL, priority, weight, and supported models
- **Priority_Level**: A numeric value assigned to channels where higher values indicate higher priority; channels at the same priority level form a peer group
- **Probe_Request**: A minimal API request (`max_tokens=1`, simple message) used to verify channel connectivity without significant token consumption
- **Window_1_Error**: An error that occurs before any response data is sent to the client (HTTP error status code returned by upstream)
- **Health_Status**: One of five states a channel can be in: Healthy, Probing, Degraded, Recovering, or ManuallyDisabled
- **Recovery_Observation_Period**: A time window during which a channel in Recovering state is monitored for stability before being promoted to Healthy

## Requirements

### Requirement 1: Channel Health State Machine

**User Story:** As a system operator, I want channels to have granular health states instead of simple on/off, so that the system can intelligently manage channel availability based on observed behavior.

#### Acceptance Criteria

1. THE Health_State_Machine SHALL maintain each channel in exactly one of five states: Healthy, Probing, Degraded, Recovering, or ManuallyDisabled
2. WHEN a channel is in Healthy state and a user request to that channel returns an error, THE Health_State_Machine SHALL transition the channel to Probing state
3. WHILE a channel is in Probing state, THE Channel_Prober SHALL execute probe requests against the channel to verify connectivity
4. WHEN a channel in Probing state accumulates consecutive probe failures equal to the probe_fail_threshold (default: 2), THE Health_State_Machine SHALL transition the channel to Degraded state
5. WHEN a channel in Probing state receives a successful probe result, THE Health_State_Machine SHALL transition the channel back to Healthy state
6. WHILE a channel is in Degraded state, THE Channel_Selector SHALL exclude the channel from user request routing
7. WHILE a channel is in Degraded state, THE Channel_Prober SHALL execute periodic probe requests at the degraded_probe_interval (default: 5 minutes)
8. WHEN a channel in Degraded state accumulates consecutive probe successes equal to the recovery_success_threshold (default: 3), THE Health_State_Machine SHALL transition the channel to Recovering state
9. WHILE a channel is in Recovering state, THE Channel_Selector SHALL include the channel in normal priority-weighted routing
10. WHEN a channel in Recovering state receives any request error within the recovery_observation_period (default: 10 minutes), THE Health_State_Machine SHALL transition the channel back to Degraded state
11. WHEN a channel in Recovering state completes the recovery_observation_period without errors, THE Health_State_Machine SHALL transition the channel to Healthy state
12. WHILE a channel is in ManuallyDisabled state, THE Health_State_Machine SHALL not perform any automatic state transitions on the channel
13. THE Health_State_Machine SHALL persist health state to the channels database table and synchronize state to the in-memory channel cache

### Requirement 2: Same-Priority Channel Switching

**User Story:** As a client application, I want the gateway to try other channels at the same priority level before dropping to lower priority channels, so that I get the best available service quality.

#### Acceptance Criteria

1. WHEN a channel at a given priority level fails, THE Channel_Selector SHALL attempt other channels at the same priority level before selecting channels at a lower priority level
2. THE Channel_Selector SHALL track which channels have been attempted for the current request using the existing `use_channel` context list
3. THE Channel_Selector SHALL exclude previously attempted channel IDs when selecting the next channel at the same priority level
4. WHEN the number of same-priority channel attempts reaches the max_retry_same_priority limit (default: 3), THE Channel_Selector SHALL proceed to the next lower priority level
5. WHEN all channels at the current priority level have been attempted or the same-priority limit is reached, THE Channel_Selector SHALL select a channel from the next lower priority level

### Requirement 3: Lightweight Channel Probing

**User Story:** As a system operator, I want a fast, low-cost probe mechanism that verifies channel connectivity without the overhead of the existing test function, so that channel health can be assessed in real-time during request processing.

#### Acceptance Criteria

1. THE Channel_Prober SHALL send a minimal request body containing a single short message with max_tokens set to 1 and stream set to false
2. THE Channel_Prober SHALL make a direct HTTP POST request to the upstream provider using the channel base URL and API key, bypassing the full relay pipeline
3. THE Channel_Prober SHALL enforce a timeout of probe_timeout (default: 3 seconds) per probe request
4. THE Channel_Prober SHALL determine probe success based solely on receiving an HTTP 2xx status code from the upstream provider
5. THE Channel_Prober SHALL construct the probe request using the same model name that triggered the error, ensuring model-specific connectivity is verified
6. IF the upstream provider returns a non-2xx status code within the timeout period, THEN THE Channel_Prober SHALL report the probe as failed

### Requirement 4: Transparent Error Interception and Retry

**User Story:** As a client application, I want upstream errors to be intercepted and retried transparently, so that I experience slightly higher latency instead of receiving error responses.

#### Acceptance Criteria

1. WHEN the upstream provider returns a Window_1_Error, THE Relay_Controller SHALL intercept the error without returning it to the client
2. WHEN a Window_1_Error is intercepted, THE Relay_Controller SHALL trigger a probe against the failed channel
3. WHEN the probe succeeds after a Window_1_Error, THE Relay_Controller SHALL retry the original user request on the same channel (transient error case)
4. WHEN the retry on the same channel fails after a successful probe, THE Health_State_Machine SHALL immediately transition the channel to Degraded state and THE Relay_Controller SHALL switch to the next available channel
5. WHEN the probe fails after a Window_1_Error, THE Relay_Controller SHALL mark the channel for health state transition and switch to the next available channel
6. WHEN all available channels across all priority levels have been exhausted, THE Relay_Controller SHALL return the last error to the client
7. WHEN the upstream error has HTTP status code 400, THE Relay_Controller SHALL switch to the next channel without triggering any health state change on the failed channel
8. WHEN the upstream error has HTTP status code 401 and the error message contains a disable keyword, THE Health_State_Machine SHALL immediately transition the channel to Degraded state without probing

### Requirement 5: Probe Cost Tracking

**User Story:** As a system operator, I want probe token consumption tracked separately from user requests, so that I can monitor the cost of the health management system.

#### Acceptance Criteria

1. THE Channel_Prober SHALL record each probe request execution including channel ID, model name, success or failure status, and latency in milliseconds
2. THE Channel_Prober SHALL mark probe consumption records with a distinct identifier (probe type marker in the log other field) to distinguish them from user request logs
3. THE Relay_Controller SHALL not bill probe token consumption to any user account

### Requirement 6: Low-Channel Availability Alert

**User Story:** As a system operator, I want to be alerted when the number of healthy channels for a model in a group drops to critical levels, so that I can take manual action before service is fully degraded.

#### Acceptance Criteria

1. WHEN the count of Healthy or Recovering channels for a specific model within a group drops to 2 or fewer, THE Health_State_Machine SHALL emit a warning-level alert
2. WHEN all channels for a specific model within a group are in Degraded or ManuallyDisabled state, THE Health_State_Machine SHALL emit a critical-level alert
3. THE Health_State_Machine SHALL include the group name, model name, and count of remaining healthy channels in each alert message

### Requirement 7: Health State Database Schema

**User Story:** As a system operator, I want channel health state persisted in the database, so that health information survives system restarts and is visible across cluster nodes.

#### Acceptance Criteria

1. THE Health_State_Machine SHALL store health state using new columns on the channels table: health_status (VARCHAR(16), default 'healthy'), health_updated_at (BIGINT, default 0), health_fail_count (INT, default 0), health_success_count (INT, default 0)
2. THE Health_State_Machine SHALL update the in-memory channel cache immediately upon state change and write to the database asynchronously
3. THE Health_State_Machine SHALL support all three database backends (SQLite, MySQL, PostgreSQL) for the new schema columns
4. WHEN the system starts, THE Health_State_Machine SHALL load persisted health states from the database into the in-memory cache
