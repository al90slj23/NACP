package service

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

func TestLiveProbeRecordsUsageToNacpTestDB(t *testing.T) {
	if os.Getenv("NACP_LIVE_TEST") != "1" {
		t.Skip("set NACP_LIVE_TEST=1 to run against the nacp.m.srl test database")
	}

	const channelName = "NACP-test-claude"
	const modelName = "claude-haiku-4-5-20251001"

	var channel model.Channel
	if err := model.DB.Where("name = ?", channelName).First(&channel).Error; err != nil {
		t.Fatalf("failed to load live test channel %s: %v", channelName, err)
	}

	result := ProbeChannel(&channel, modelName)
	if result.Error != nil {
		t.Fatalf("probe returned error: %v", result.Error)
	}
	if !result.Success {
		t.Fatalf("probe failed: status=%d latency_ms=%d", result.StatusCode, result.LatencyMs)
	}
	if result.PromptTokens <= 0 {
		t.Fatalf("probe prompt tokens = %d, want > 0", result.PromptTokens)
	}
	if result.CompletionTokens <= 0 {
		t.Fatalf("probe completion tokens = %d, want > 0", result.CompletionTokens)
	}
	if result.EstimatedQuota <= 0 {
		t.Fatalf("probe estimated quota = %d, want > 0", result.EstimatedQuota)
	}

	requestId := fmt.Sprintf("nacp_live_probe_usage_%d", time.Now().UnixNano())
	recordProbeLog(&ProbeLog{
		ChannelID:           channel.Id,
		ChannelName:         channel.Name,
		ModelName:           modelName,
		Success:             result.Success,
		LatencyMs:           result.LatencyMs,
		StatusCode:          result.StatusCode,
		Trigger:             "live_test",
		Timestamp:           time.Now().Unix(),
		PromptTokens:        result.PromptTokens,
		CompletionTokens:    result.CompletionTokens,
		CacheReadTokens:     result.CacheReadTokens,
		CacheCreationTokens: result.CacheCreationTokens,
		EstimatedQuota:      result.EstimatedQuota,
	}, requestId)

	var logged model.Log
	deadline := time.Now().Add(5 * time.Second)
	for {
		err := model.LOG_DB.Where("request_id = ?", requestId).Order("id desc").First(&logged).Error
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("probe log was not written for request_id=%s: %v", requestId, err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	if logged.PromptTokens != result.PromptTokens {
		t.Fatalf("logged prompt tokens = %d, want %d", logged.PromptTokens, result.PromptTokens)
	}
	if logged.CompletionTokens != result.CompletionTokens {
		t.Fatalf("logged completion tokens = %d, want %d", logged.CompletionTokens, result.CompletionTokens)
	}
	if logged.Quota != result.EstimatedQuota {
		t.Fatalf("logged quota = %d, want %d", logged.Quota, result.EstimatedQuota)
	}
	if logged.TraceRole != model.TraceRoleProbeSuccess {
		t.Fatalf("logged trace role = %q, want %q", logged.TraceRole, model.TraceRoleProbeSuccess)
	}

	detail, err := GetTraceDetail(requestId)
	if err != nil {
		t.Fatalf("GetTraceDetail failed: %v", err)
	}
	if len(detail.Steps) != 1 {
		t.Fatalf("trace steps = %d, want 1", len(detail.Steps))
	}
	step := detail.Steps[0]
	if step.PromptTokens != result.PromptTokens || step.CompletionTokens != result.CompletionTokens || step.Quota != result.EstimatedQuota {
		t.Fatalf("trace step usage mismatch: prompt=%d completion=%d quota=%d, want prompt=%d completion=%d quota=%d",
			step.PromptTokens, step.CompletionTokens, step.Quota, result.PromptTokens, result.CompletionTokens, result.EstimatedQuota)
	}

	t.Logf("live probe request_id=%s channel=%d prompt=%d completion=%d quota=%d latency_ms=%d",
		requestId, channel.Id, result.PromptTokens, result.CompletionTokens, result.EstimatedQuota, result.LatencyMs)
}

func TestLiveSeedConsoleLogDisplayScenarios(t *testing.T) {
	if os.Getenv("NACP_LIVE_TEST") != "1" {
		t.Skip("set NACP_LIVE_TEST=1 to seed visible console/log scenarios")
	}

	stamp := time.Now().Format("20060102150405")
	now := time.Now().Unix()
	username := "nacp_log_ui_" + stamp
	tokenName := "log-ui-" + stamp
	modelName := "claude-haiku-4-5-20251001"

	makeOther := func(statusCode int, latencyMs int, extra map[string]interface{}) string {
		other := map[string]interface{}{
			"admin_info": map[string]interface{}{
				"status_code": statusCode,
				"latency_ms":  latencyMs,
			},
			"request_path":       "/v1/chat/completions",
			"request_conversion": []string{"OpenAI Compatible", "Claude Messages"},
		}
		for k, v := range extra {
			other[k] = v
		}
		return common.MapToJsonStr(other)
	}

	insert := func(log model.Log) {
		t.Helper()
		if log.CreatedAt == 0 {
			log.CreatedAt = now
		}
		if log.ModelName == "" {
			log.ModelName = modelName
		}
		if log.Username == "" {
			log.Username = username
		}
		if log.TokenName == "" {
			log.TokenName = tokenName
		}
		if log.Group == "" {
			log.Group = "default"
		}
		if err := model.LOG_DB.Create(&log).Error; err != nil {
			t.Fatalf("failed to insert seeded log: %v", err)
		}
	}

	directRequestId := "nacp_ui_direct_" + stamp
	insert(model.Log{
		RequestId:        directRequestId,
		Type:             model.LogTypeConsume,
		TraceRole:        model.TraceRoleConsume,
		ChannelId:        17,
		Quota:            16,
		PromptTokens:     8,
		CompletionTokens: 1,
		UseTime:          2,
		Content:          "NACP UI seed direct success",
		Other: makeOther(200, 1234, map[string]interface{}{
			"model_ratio":      0.5,
			"completion_ratio": 5,
			"group_ratio":      1,
		}),
	})

	successRequestId := "nacp_ui_sft_success_" + stamp
	insert(model.Log{
		RequestId: successRequestId,
		Type:      model.LogTypeErrorIntercepted,
		ChannelId: 12,
		UseTime:   1,
		Content:   "status_code=500, seeded intercepted upstream error",
		Other:     makeOther(500, 850, map[string]interface{}{"reason": "seeded intercepted upstream error"}),
	})
	insert(model.Log{
		RequestId:        successRequestId,
		Type:             model.LogTypeProbeSuccess,
		ChannelId:        17,
		Quota:            7,
		PromptTokens:     8,
		CompletionTokens: 1,
		UseTime:          2,
		Content:          "probe claude-haiku-4-5-20251001 channel #17",
		Other: makeOther(200, 2030, map[string]interface{}{
			"cost_reason":          "lightweight_probe",
			"cost_scope":           "platform",
			"probe_trigger":        "live_seed",
			"probe_usage_recorded": true,
			"probe_usage_source":   "upstream_response",
			"quota_estimated":      true,
		}),
	})
	insert(model.Log{
		RequestId: successRequestId,
		Type:      model.LogTypeProbeFailed,
		ChannelId: 14,
		UseTime:   1,
		Content:   "probe seeded failed channel #14",
		Other: makeOther(503, 900, map[string]interface{}{
			"cost_reason":          "lightweight_probe",
			"cost_scope":           "platform",
			"probe_trigger":        "live_seed",
			"probe_usage_recorded": false,
			"quota_estimated":      false,
			"error":                "seeded probe failed",
		}),
	})
	insert(model.Log{
		RequestId:        successRequestId,
		Type:             model.LogTypeConsume,
		TraceRole:        model.TraceRoleConsume,
		ChannelId:        17,
		Quota:            16,
		PromptTokens:     12,
		CompletionTokens: 4,
		UseTime:          3,
		Content:          "NACP UI seed SFT success",
		Other: makeOther(200, 3000, map[string]interface{}{
			"model_ratio":      0.5,
			"completion_ratio": 5,
			"group_ratio":      1,
			"admin_info": map[string]interface{}{
				"status_code": 200,
				"latency_ms":  3000,
				"use_channel": []int{12, 17},
			},
		}),
	})

	failedRequestId := "nacp_ui_sft_failed_" + stamp
	insert(model.Log{
		RequestId: failedRequestId,
		Type:      model.LogTypeErrorIntercepted,
		ChannelId: 12,
		UseTime:   1,
		Content:   "status_code=500, seeded intercepted upstream error",
		Other:     makeOther(500, 850, map[string]interface{}{"reason": "seeded intercepted upstream error"}),
	})
	insert(model.Log{
		RequestId:        failedRequestId,
		Type:             model.LogTypeProbeSuccess,
		ChannelId:        13,
		Quota:            7,
		PromptTokens:     8,
		CompletionTokens: 1,
		UseTime:          2,
		Content:          "probe claude-haiku-4-5-20251001 channel #13",
		Other: makeOther(200, 2100, map[string]interface{}{
			"cost_reason":          "lightweight_probe",
			"cost_scope":           "platform",
			"probe_trigger":        "live_seed",
			"probe_usage_recorded": true,
			"probe_usage_source":   "upstream_response",
			"quota_estimated":      true,
		}),
	})
	insert(model.Log{
		RequestId: failedRequestId,
		Type:      model.LogTypeProbeFailed,
		ChannelId: 14,
		UseTime:   1,
		Content:   "probe seeded failed channel #14",
		Other: makeOther(503, 900, map[string]interface{}{
			"cost_reason":          "lightweight_probe",
			"cost_scope":           "platform",
			"probe_trigger":        "live_seed",
			"probe_usage_recorded": false,
			"quota_estimated":      false,
			"error":                "seeded probe failed",
		}),
	})
	insert(model.Log{
		RequestId: failedRequestId,
		Type:      model.LogTypeErrorClientVisible,
		ChannelId: 14,
		UseTime:   4,
		Content:   "status_code=500, seeded final visible error",
		Other: makeOther(500, 4000, map[string]interface{}{
			"reason": "seeded final visible error",
			"admin_info": map[string]interface{}{
				"status_code": 500,
				"latency_ms":  4000,
				"use_channel": []int{12, 13, 14},
			},
		}),
	})

	t.Logf("seeded console/log scenarios username=%s token=%s direct=%s success=%s failed=%s",
		username, tokenName, directRequestId, successRequestId, failedRequestId)
}
