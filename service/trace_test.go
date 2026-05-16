package service

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"pgregory.net/rapid"
)

// cleanLogs removes all log records between tests.
func cleanLogs(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		model.LOG_DB.Exec("DELETE FROM logs")
	})
}

// Feature: request-trace-view, Property 1: 链路详情查询过滤与排序
// For any set of Log records sharing the same request_id with random types,
// calling GetTraceDetail should return steps that:
// (a) only contain trace detail event types; summary rows are excluded.
// (b) are ordered by trace_seq ASC when present, otherwise created_at ASC
// (c) count ≤ 100
// **Validates: Requirements 1.1**
func TestTraceProperty1_DetailFilterAndSort(t *testing.T) {
	cleanLogs(t)

	rapid.Check(t, func(t *rapid.T) {
		// Clean up before each iteration
		model.LOG_DB.Exec("DELETE FROM logs")

		requestId := fmt.Sprintf("req_%s", rapid.StringMatching(`[a-z0-9]{8,16}`).Draw(t, "requestId"))
		numLogs := rapid.IntRange(1, 30).Draw(t, "numLogs")

		allTypes := []int{1, 2, 3, 4, 5, 20, 21, 29, 50, 51, 52, 59}
		var baseTime int64 = 1700000000

		for i := 0; i < numLogs; i++ {
			logType := allTypes[rapid.IntRange(0, len(allTypes)-1).Draw(t, fmt.Sprintf("type_%d", i))]
			createdAt := baseTime + int64(rapid.IntRange(0, 10000).Draw(t, fmt.Sprintf("time_%d", i)))

			log := model.Log{
				RequestId: requestId,
				Type:      logType,
				CreatedAt: createdAt,
				ChannelId: rapid.IntRange(1, 10).Draw(t, fmt.Sprintf("channel_%d", i)),
				ModelName: "gpt-4",
				Username:  "testuser",
				TokenName: "testtoken",
			}
			if err := model.LOG_DB.Create(&log).Error; err != nil {
				t.Fatalf("failed to insert log: %v", err)
			}
		}

		detail, err := GetTraceDetail(requestId)
		if err != nil {
			t.Fatalf("GetTraceDetail error: %v", err)
		}

		// (a) Only valid trace detail event types. 20/50 summary rows are main-list
		// records and should not appear in detail steps.
		validTypes := map[int]bool{
			model.LogTypeConsume:            true,
			model.LogTypeRetryConsume:       true,
			model.LogTypeProbeSuccess:       true,
			model.LogTypeProbeFailed:        true,
			model.LogTypeError:              true,
			model.LogTypeErrorIntercepted:   true,
			model.LogTypeErrorClientVisible: true,
		}
		for _, step := range detail.Steps {
			if !validTypes[step.Type] {
				t.Fatalf("step has invalid trace detail type %d", step.Type)
			}
		}

		// (b) Ordered by trace_seq ASC when present. Fresh writes get trace_seq
		// in insertion order even if created_at is not monotonic.
		for i := 1; i < len(detail.Steps); i++ {
			prev := detail.Steps[i-1]
			cur := detail.Steps[i]
			if prev.TraceSeq > 0 || cur.TraceSeq > 0 {
				if cur.TraceSeq < prev.TraceSeq {
					t.Fatalf("steps not ordered by trace_seq ASC: step[%d].trace_seq=%d < step[%d].trace_seq=%d",
						i, cur.TraceSeq, i-1, prev.TraceSeq)
				}
				continue
			}
			if cur.CreatedAt < prev.CreatedAt {
				t.Fatalf("steps not ordered by fallback created_at ASC: step[%d].created_at=%d < step[%d].created_at=%d",
					i, cur.CreatedAt, i-1, prev.CreatedAt)
			}
		}

		// (c) Count ≤ 100
		if len(detail.Steps) > 100 {
			t.Fatalf("steps count %d exceeds 100", len(detail.Steps))
		}
	})
}

func TestTraceDetailIncludesProbeLogs(t *testing.T) {
	cleanLogs(t)
	requireNoError := func(err error) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	const requestId = "trace_detail_probe_logs"
	rows := []model.Log{
		{RequestId: requestId, Type: model.LogTypeErrorIntercepted, CreatedAt: 100, ChannelId: 12},
		{RequestId: requestId, Type: model.LogTypeProbeSuccess, CreatedAt: 101, ChannelId: 13},
		{RequestId: requestId, Type: model.LogTypeProbeFailed, CreatedAt: 102, ChannelId: 14},
		{RequestId: requestId, Type: model.LogTypeErrorClientVisible, CreatedAt: 103, ChannelId: 12},
	}
	for _, row := range rows {
		requireNoError(model.LOG_DB.Create(&row).Error)
	}

	detail, err := GetTraceDetail(requestId)
	requireNoError(err)
	if len(detail.Steps) != len(rows) {
		t.Fatalf("expected %d steps, got %d", len(rows), len(detail.Steps))
	}
	expectedTypes := []int{model.LogTypeErrorIntercepted, model.LogTypeProbeSuccess, model.LogTypeProbeFailed, model.LogTypeErrorClientVisible}
	expectedRoles := []string{model.TraceRoleErrorIntercepted, model.TraceRoleProbeSuccess, model.TraceRoleProbeFailed, model.TraceRoleErrorVisible}
	for i, expectedType := range expectedTypes {
		if detail.Steps[i].Type != expectedType {
			t.Fatalf("step[%d] type: expected %d, got %d", i, expectedType, detail.Steps[i].Type)
		}
		if detail.Steps[i].TraceRole != expectedRoles[i] {
			t.Fatalf("step[%d] trace_role: expected %s, got %s", i, expectedRoles[i], detail.Steps[i].TraceRole)
		}
		if detail.Steps[i].RequestId != requestId {
			t.Fatalf("step[%d] request_id: expected %s, got %s", i, requestId, detail.Steps[i].RequestId)
		}
		if detail.Steps[i].Sequence != i+1 {
			t.Fatalf("step[%d] sequence: expected %d, got %d", i, i+1, detail.Steps[i].Sequence)
		}
	}
}

func TestTraceListCollapsesProbeRowsIntoSingleSummary(t *testing.T) {
	cleanLogs(t)
	model.LOG_DB.Exec("DELETE FROM logs")

	const requestId = "trace_list_probe_principal_split"
	rows := []model.Log{
		{
			RequestId: requestId,
			Type:      model.LogTypeErrorIntercepted,
			CreatedAt: 100,
			ChannelId: 12,
			ModelName: "claude-haiku-4-5-20251001",
			Username:  "trace_user",
			TokenName: "trace_token",
		},
		{
			RequestId: requestId,
			Type:      model.LogTypeProbeFailed,
			CreatedAt: 101,
			ChannelId: 14,
			ModelName: "claude-haiku-4-5-20251001",
		},
		{
			RequestId:        requestId,
			Type:             model.LogTypeConsume,
			CreatedAt:        102,
			ChannelId:        13,
			ModelName:        "claude-haiku-4-5-20251001",
			Username:         "trace_user",
			TokenName:        "trace_token",
			Quota:            111,
			PromptTokens:     121,
			CompletionTokens: 20,
		},
	}
	for _, row := range rows {
		if err := model.LOG_DB.Create(&row).Error; err != nil {
			t.Fatalf("failed to insert log: %v", err)
		}
	}

	results, total, err := GetTraceList(TraceListParams{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("GetTraceList error: %v", err)
	}

	matches := make([]TraceSummary, 0)
	for _, result := range results {
		if result.RequestId == requestId {
			matches = append(matches, result)
		}
	}
	if len(matches) != 1 {
		t.Fatalf("expected one summary for request_id %s, got %d: %#v", requestId, len(matches), matches)
	}
	if total != 1 {
		t.Fatalf("expected total=1, got %d", total)
	}

	summary := matches[0]
	if summary.Status != "success" {
		t.Fatalf("expected status success, got %q", summary.Status)
	}
	if summary.Username != "trace_user" || summary.TokenName != "trace_token" {
		t.Fatalf("expected principal trace_user/trace_token, got %q/%q", summary.Username, summary.TokenName)
	}
	if summary.ChannelCount != 3 {
		t.Fatalf("expected channel_count=3, got %d", summary.ChannelCount)
	}
	if summary.TotalQuota != 111 || summary.TotalPromptTokens != 121 || summary.TotalCompletionTokens != 20 {
		t.Fatalf("unexpected totals: quota=%d prompt=%d completion=%d",
			summary.TotalQuota, summary.TotalPromptTokens, summary.TotalCompletionTokens)
	}
}

func TestTraceListUsesMaterializedSummaryForBillingTotals(t *testing.T) {
	cleanLogs(t)
	model.LOG_DB.Exec("DELETE FROM logs")

	const requestId = "trace_list_materialized_summary_totals"
	rows := []model.Log{
		{
			RequestId: requestId,
			TraceId:   requestId,
			Type:      model.LogTypeErrorIntercepted,
			CreatedAt: 800,
			ChannelId: 12,
			ModelName: "claude-haiku-4-5-20251001",
			Username:  "trace_user",
			TokenName: "trace_token",
		},
		{
			RequestId:        requestId,
			TraceId:          requestId,
			Type:             model.LogTypeRetryConsume,
			CreatedAt:        801,
			ChannelId:        13,
			ModelName:        "claude-haiku-4-5-20251001",
			Username:         "trace_user",
			TokenName:        "trace_token",
			Quota:            123,
			PromptTokens:     11,
			CompletionTokens: 22,
		},
	}
	for _, row := range rows {
		if err := model.LOG_DB.Create(&row).Error; err != nil {
			t.Fatalf("failed to insert log: %v", err)
		}
	}
	summary, err := model.UpsertTraceSummary(requestId)
	if err != nil {
		t.Fatalf("UpsertTraceSummary error: %v", err)
	}
	if summary == nil || summary.Type != model.LogTypeRetrySuccessSummary {
		t.Fatalf("expected success summary, got %#v", summary)
	}

	results, total, err := GetTraceList(TraceListParams{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("GetTraceList error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total=1, got %d", total)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d: %#v", len(results), results)
	}
	got := results[0]
	if got.RequestId != requestId || got.Status != "success" {
		t.Fatalf("unexpected summary identity/status: %#v", got)
	}
	if got.TotalQuota != 123 || got.TotalPromptTokens != 11 || got.TotalCompletionTokens != 22 {
		t.Fatalf("summary totals should use materialized 20 only, got quota=%d prompt=%d completion=%d",
			got.TotalQuota, got.TotalPromptTokens, got.TotalCompletionTokens)
	}
}

// Feature: request-trace-view, Property 2: Other 字段 status_code 解析
// For any Log record, if Other is valid JSON with admin_info.status_code numeric field,
// the corresponding TraceStep.StatusCode should equal that value;
// if Other is empty, invalid JSON, or missing the field, StatusCode should be nil.
// **Validates: Requirements 1.3**
func TestTraceProperty2_StatusCodeParsing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate one of four cases
		caseType := rapid.IntRange(0, 3).Draw(t, "caseType")

		var other string
		var expectedCode *int

		switch caseType {
		case 0:
			// Valid JSON with admin_info.status_code
			code := rapid.IntRange(100, 599).Draw(t, "statusCode")
			other = fmt.Sprintf(`{"admin_info":{"status_code":%d,"channel_name":"test"}}`, code)
			expectedCode = &code
		case 1:
			// Valid JSON without admin_info.status_code
			other = `{"admin_info":{"channel_name":"test"}}`
			expectedCode = nil
		case 2:
			// Empty string
			other = ""
			expectedCode = nil
		case 3:
			// Invalid JSON
			other = rapid.StringMatching(`[a-z]{5,20}`).Draw(t, "invalidJson")
			expectedCode = nil
		}

		result := parseStatusCodeFromOther(other)

		if expectedCode == nil {
			if result != nil {
				t.Fatalf("expected nil status_code for other=%q, got %d", other, *result)
			}
		} else {
			if result == nil {
				t.Fatalf("expected status_code=%d for other=%q, got nil", *expectedCode, other)
			}
			if *result != *expectedCode {
				t.Fatalf("expected status_code=%d, got %d for other=%q", *expectedCode, *result, other)
			}
		}
	})
}

// Feature: request-trace-view, Property 3: 链路最终结果分类
// For any request_id and its associated log set, if the set contains at least one type=2 record,
// the trace status should be "success"; otherwise it should be "failed".
// **Validates: Requirements 2.4, 2.5**
func TestTraceProperty3_StatusClassification(t *testing.T) {
	cleanLogs(t)

	rapid.Check(t, func(t *rapid.T) {
		model.LOG_DB.Exec("DELETE FROM logs")

		requestId := fmt.Sprintf("req_%s", rapid.StringMatching(`[a-z0-9]{8,16}`).Draw(t, "requestId"))

		// Generate logs with types from the valid set for trace queries
		validTypes := []int{2, 5, 51, 52}
		numLogs := rapid.IntRange(2, 15).Draw(t, "numLogs") // at least 2 to pass HAVING filter
		hasType2 := rapid.Bool().Draw(t, "hasType2")

		var baseTime int64 = 1700000000

		for i := 0; i < numLogs; i++ {
			var logType int
			if hasType2 && i == 0 {
				// Ensure at least one type=2 if hasType2 is true
				logType = 2
			} else if !hasType2 {
				// Only non-consume types
				nonConsumeTypes := []int{5, 51, 52}
				logType = nonConsumeTypes[rapid.IntRange(0, 2).Draw(t, fmt.Sprintf("type_%d", i))]
			} else {
				logType = validTypes[rapid.IntRange(0, len(validTypes)-1).Draw(t, fmt.Sprintf("type_%d", i))]
			}

			log := model.Log{
				RequestId: requestId,
				Type:      logType,
				CreatedAt: baseTime + int64(i),
				ChannelId: rapid.IntRange(1, 5).Draw(t, fmt.Sprintf("ch_%d", i)),
				ModelName: "gpt-4",
				Username:  "testuser",
				TokenName: "testtoken",
			}
			if err := model.LOG_DB.Create(&log).Error; err != nil {
				t.Fatalf("failed to insert log: %v", err)
			}
		}

		// Query trace list
		results, _, err := GetTraceList(TraceListParams{
			Page:     1,
			PageSize: 100,
		})
		if err != nil {
			t.Fatalf("GetTraceList error: %v", err)
		}

		// Find our request_id in results
		var found *TraceSummary
		for i := range results {
			if results[i].RequestId == requestId {
				found = &results[i]
				break
			}
		}

		if found == nil {
			t.Fatalf("request_id %s not found in trace list results", requestId)
		}

		expectedStatus := "failed"
		if hasType2 {
			expectedStatus = "success"
		}

		if found.Status != expectedStatus {
			t.Fatalf("expected status=%q for hasType2=%v, got %q", expectedStatus, hasType2, found.Status)
		}
	})
}

// Feature: request-trace-view, Property 5: 链路摘要聚合正确性
// For any request_id and its associated log set:
// (a) channel_count = distinct channel_id count
// (b) total_duration = max(created_at) - min(created_at)
// (c) created_at = min(created_at)
// **Validates: Requirements 2.3**
func TestTraceProperty5_SummaryAggregation(t *testing.T) {
	cleanLogs(t)

	rapid.Check(t, func(t *rapid.T) {
		model.LOG_DB.Exec("DELETE FROM logs")

		requestId := fmt.Sprintf("req_%s", rapid.StringMatching(`[a-z0-9]{8,16}`).Draw(t, "requestId"))

		// Generate logs with valid types (must have >=2 to pass HAVING filter)
		validTypes := []int{2, 5, 51, 52}
		numLogs := rapid.IntRange(2, 20).Draw(t, "numLogs")

		var baseTime int64 = 1700000000
		channelSet := make(map[int]bool)
		var minCreatedAt, maxCreatedAt int64

		for i := 0; i < numLogs; i++ {
			logType := validTypes[rapid.IntRange(0, len(validTypes)-1).Draw(t, fmt.Sprintf("type_%d", i))]
			channelId := rapid.IntRange(1, 20).Draw(t, fmt.Sprintf("ch_%d", i))
			createdAt := baseTime + int64(rapid.IntRange(0, 10000).Draw(t, fmt.Sprintf("time_%d", i)))

			channelSet[channelId] = true
			if i == 0 || createdAt < minCreatedAt {
				minCreatedAt = createdAt
			}
			if i == 0 || createdAt > maxCreatedAt {
				maxCreatedAt = createdAt
			}

			log := model.Log{
				RequestId: requestId,
				Type:      logType,
				CreatedAt: createdAt,
				ChannelId: channelId,
				ModelName: "gpt-4",
				Username:  "testuser",
				TokenName: "testtoken",
			}
			if err := model.LOG_DB.Create(&log).Error; err != nil {
				t.Fatalf("failed to insert log: %v", err)
			}
		}

		results, _, err := GetTraceList(TraceListParams{
			Page:     1,
			PageSize: 100,
		})
		if err != nil {
			t.Fatalf("GetTraceList error: %v", err)
		}

		var found *TraceSummary
		for i := range results {
			if results[i].RequestId == requestId {
				found = &results[i]
				break
			}
		}

		if found == nil {
			t.Fatalf("request_id %s not found in trace list results", requestId)
		}

		// (a) channel_count = distinct channel_id count
		expectedChannelCount := len(channelSet)
		if found.ChannelCount != expectedChannelCount {
			t.Fatalf("channel_count: expected %d, got %d", expectedChannelCount, found.ChannelCount)
		}

		// (b) total_duration = max(created_at) - min(created_at)
		expectedDuration := maxCreatedAt - minCreatedAt
		if found.TotalDuration != expectedDuration {
			t.Fatalf("total_duration: expected %d, got %d", expectedDuration, found.TotalDuration)
		}

		// (c) created_at = min(created_at)
		if found.CreatedAt != minCreatedAt {
			t.Fatalf("created_at: expected %d, got %d", minCreatedAt, found.CreatedAt)
		}
	})
}

// Feature: request-trace-view, Property 6: HAVING 过滤条件
// For any trace list query result, every returned record's request_id should satisfy:
// log_count >= 2 OR has_error (type ∈ {5, 51, 52})
// **Validates: Requirements 2.8**
func TestTraceProperty6_HavingFilter(t *testing.T) {
	cleanLogs(t)

	rapid.Check(t, func(t *rapid.T) {
		model.LOG_DB.Exec("DELETE FROM logs")

		var baseTime int64 = 1700000000
		numRequests := rapid.IntRange(3, 8).Draw(t, "numRequests")

		type requestInfo struct {
			requestId string
			logCount  int
			hasError  bool
		}
		var requests []requestInfo

		for r := 0; r < numRequests; r++ {
			reqId := fmt.Sprintf("req_%d_%s", r, rapid.StringMatching(`[a-z0-9]{6}`).Draw(t, fmt.Sprintf("rid_%d", r)))
			scenario := rapid.IntRange(0, 2).Draw(t, fmt.Sprintf("scenario_%d", r))

			var info requestInfo
			info.requestId = reqId

			switch scenario {
			case 0:
				// Single log, no error — should NOT appear in results
				log := model.Log{
					RequestId: reqId,
					Type:      2, // consume, not an error
					CreatedAt: baseTime + int64(r*100),
					ChannelId: 1,
					ModelName: "gpt-4",
					Username:  "testuser",
					TokenName: "testtoken",
				}
				model.LOG_DB.Create(&log)
				info.logCount = 1
				info.hasError = false

			case 1:
				// Multiple logs (>=2) — should appear in results
				numLogs := rapid.IntRange(2, 5).Draw(t, fmt.Sprintf("nlogs_%d", r))
				validTypes := []int{2, 5, 51, 52}
				for i := 0; i < numLogs; i++ {
					logType := validTypes[rapid.IntRange(0, len(validTypes)-1).Draw(t, fmt.Sprintf("t_%d_%d", r, i))]
					if logType == 5 || logType == 51 || logType == 52 {
						info.hasError = true
					}
					log := model.Log{
						RequestId: reqId,
						Type:      logType,
						CreatedAt: baseTime + int64(r*100+i),
						ChannelId: rapid.IntRange(1, 5).Draw(t, fmt.Sprintf("ch_%d_%d", r, i)),
						ModelName: "gpt-4",
						Username:  "testuser",
						TokenName: "testtoken",
					}
					model.LOG_DB.Create(&log)
				}
				info.logCount = numLogs

			case 2:
				// Single error log — should appear in results (has_error = 1)
				errorTypes := []int{5, 51, 52}
				logType := errorTypes[rapid.IntRange(0, 2).Draw(t, fmt.Sprintf("et_%d", r))]
				log := model.Log{
					RequestId: reqId,
					Type:      logType,
					CreatedAt: baseTime + int64(r*100),
					ChannelId: 1,
					ModelName: "gpt-4",
					Username:  "testuser",
					TokenName: "testtoken",
				}
				model.LOG_DB.Create(&log)
				info.logCount = 1
				info.hasError = true
			}

			requests = append(requests, info)
		}

		// Query trace list
		results, _, err := GetTraceList(TraceListParams{
			Page:     1,
			PageSize: 100,
		})
		if err != nil {
			t.Fatalf("GetTraceList error: %v", err)
		}

		// Verify: every returned record satisfies HAVING condition
		for _, result := range results {
			// Find the corresponding request info
			var info *requestInfo
			for i := range requests {
				if requests[i].requestId == result.RequestId {
					info = &requests[i]
					break
				}
			}
			if info == nil {
				// Result from a request we didn't generate — skip
				continue
			}

			if info.logCount < 2 && !info.hasError {
				t.Fatalf("request_id %s appeared in results but has logCount=%d and hasError=%v (should not pass HAVING filter)",
					info.requestId, info.logCount, info.hasError)
			}
		}

		// Also verify: requests that should NOT appear are absent
		for _, info := range requests {
			if info.logCount < 2 && !info.hasError {
				for _, result := range results {
					if result.RequestId == info.requestId {
						t.Fatalf("request_id %s should not appear in results (logCount=%d, hasError=%v)",
							info.requestId, info.logCount, info.hasError)
					}
				}
			}
		}
	})
}

// Feature: request-trace-view, Property 8: Quota 与 Token 聚合
// For any request_id and its associated log set:
// total_quota = sum(type=2 quota)
// total_prompt_tokens = sum(type=2 prompt_tokens)
// total_completion_tokens = sum(type=2 completion_tokens)
// When no type=2 records exist, all three should be 0.
// **Validates: Requirements 6.1, 6.2, 6.3**
func TestTraceProperty8_QuotaTokenAggregation(t *testing.T) {
	cleanLogs(t)

	rapid.Check(t, func(t *rapid.T) {
		model.LOG_DB.Exec("DELETE FROM logs")

		requestId := fmt.Sprintf("req_%s", rapid.StringMatching(`[a-z0-9]{8,16}`).Draw(t, "requestId"))

		// Generate a mix of logs, some type=2 with quota/tokens
		validTypes := []int{2, 5, 51, 52, 29, 59}
		numLogs := rapid.IntRange(2, 15).Draw(t, "numLogs")

		var baseTime int64 = 1700000000
		var expectedQuota, expectedPromptTokens, expectedCompletionTokens int

		for i := 0; i < numLogs; i++ {
			logType := validTypes[rapid.IntRange(0, len(validTypes)-1).Draw(t, fmt.Sprintf("type_%d", i))]

			quota := 0
			promptTokens := 0
			completionTokens := 0

			if logType == 2 {
				quota = rapid.IntRange(0, 100000).Draw(t, fmt.Sprintf("quota_%d", i))
				promptTokens = rapid.IntRange(0, 5000).Draw(t, fmt.Sprintf("pt_%d", i))
				completionTokens = rapid.IntRange(0, 5000).Draw(t, fmt.Sprintf("ct_%d", i))
				expectedQuota += quota
				expectedPromptTokens += promptTokens
				expectedCompletionTokens += completionTokens
			}

			log := model.Log{
				RequestId:        requestId,
				Type:             logType,
				CreatedAt:        baseTime + int64(i),
				ChannelId:        rapid.IntRange(1, 5).Draw(t, fmt.Sprintf("ch_%d", i)),
				ModelName:        "gpt-4",
				Username:         "testuser",
				TokenName:        "testtoken",
				Quota:            quota,
				PromptTokens:     promptTokens,
				CompletionTokens: completionTokens,
			}
			if err := model.LOG_DB.Create(&log).Error; err != nil {
				t.Fatalf("failed to insert log: %v", err)
			}
		}

		// Test via GetTraceDetail
		detail, err := GetTraceDetail(requestId)
		if err != nil {
			t.Fatalf("GetTraceDetail error: %v", err)
		}

		if detail.TotalQuota != expectedQuota {
			t.Fatalf("total_quota: expected %d, got %d", expectedQuota, detail.TotalQuota)
		}
		if detail.TotalPromptTokens != expectedPromptTokens {
			t.Fatalf("total_prompt_tokens: expected %d, got %d", expectedPromptTokens, detail.TotalPromptTokens)
		}
		if detail.TotalCompletionTokens != expectedCompletionTokens {
			t.Fatalf("total_completion_tokens: expected %d, got %d", expectedCompletionTokens, detail.TotalCompletionTokens)
		}
	})
}
