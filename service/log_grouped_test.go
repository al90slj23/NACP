package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func insertGroupedLogTestRow(t *testing.T, log model.Log) model.Log {
	t.Helper()
	require.NoError(t, model.LOG_DB.Create(&log).Error)
	return log
}

func TestGroupedLogsReturnsMaterializedTraceTypes(t *testing.T) {
	cleanLogs(t)
	require.NoError(t, model.LOG_DB.Exec("DELETE FROM logs").Error)

	const requestId = "grouped_flat_trace_types"
	insertGroupedLogTestRow(t, model.Log{
		RequestId: requestId,
		Type:      model.LogTypeErrorIntercepted,
		CreatedAt: 200,
		ChannelId: 12,
		Content:   "intercepted",
	})
	insertGroupedLogTestRow(t, model.Log{
		RequestId: requestId,
		Type:      model.LogTypeProbeSuccess,
		CreatedAt: 201,
		ChannelId: 13,
		Content:   "probe success",
	})
	insertGroupedLogTestRow(t, model.Log{
		RequestId: requestId,
		Type:      model.LogTypeProbeFailed,
		CreatedAt: 202,
		ChannelId: 14,
		Content:   "probe failed",
	})
	insertGroupedLogTestRow(t, model.Log{
		RequestId: requestId,
		Type:      model.LogTypeErrorClientVisible,
		CreatedAt: 203,
		ChannelId: 12,
		Content:   "visible error",
	})
	insertGroupedLogTestRow(t, model.Log{
		RequestId:        requestId,
		Type:             model.LogTypeConsume,
		CreatedAt:        214,
		UserId:           7,
		Username:         "nacp_t_ctx",
		TokenId:          8,
		TokenName:        "ctx-token",
		ModelName:        "gpt-5.3-codex",
		Group:            "default",
		ChannelId:        13,
		Quota:            580,
		PromptTokens:     360,
		CompletionTokens: 13,
		UseTime:          14,
		IsStream:         true,
		Ip:               "127.0.0.1",
		Content:          "success content",
	})

	items, total, err := GetGroupedLogs(GroupedLogParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.EqualValues(t, 5, total)
	require.Len(t, items, 5)

	for _, item := range items {
		require.False(t, item.IsSummary)
		require.Equal(t, requestId, item.RequestId)
		require.Equal(t, requestId, item.TraceId)
		require.NotZero(t, item.TraceSeq)
	}

	require.Equal(t, model.LogTypeConsume, items[0].Type)
	require.Equal(t, model.TraceRoleConsume, items[0].TraceRole)
	require.Equal(t, model.LogTypeErrorClientVisible, items[1].Type)
	require.Equal(t, model.TraceRoleErrorVisible, items[1].TraceRole)
	require.Equal(t, model.LogTypeProbeFailed, items[2].Type)
	require.Equal(t, model.TraceRoleProbeFailed, items[2].TraceRole)
	require.Equal(t, model.LogTypeProbeSuccess, items[3].Type)
	require.Equal(t, model.TraceRoleProbeSuccess, items[3].TraceRole)
	require.Equal(t, model.LogTypeErrorIntercepted, items[4].Type)
	require.Equal(t, model.TraceRoleErrorIntercepted, items[4].TraceRole)
}

func TestGroupedLogsReturnsSummaryAndHidesChildren(t *testing.T) {
	cleanLogs(t)
	require.NoError(t, model.LOG_DB.Exec("DELETE FROM logs").Error)

	const requestId = "grouped_materialized_summary"
	insertGroupedLogTestRow(t, model.Log{
		RequestId: requestId,
		Type:      model.LogTypeErrorIntercepted,
		CreatedAt: 500,
		ChannelId: 12,
		Content:   "intercepted",
	})
	insertGroupedLogTestRow(t, model.Log{
		RequestId:    requestId,
		Type:         model.LogTypeRetryConsume,
		CreatedAt:    501,
		ChannelId:    13,
		Quota:        152,
		PromptTokens: 18,
		Content:      "retry success",
	})
	summary, err := model.UpsertTraceSummary(requestId)
	require.NoError(t, err)
	require.NotNil(t, summary)
	require.Equal(t, model.LogTypeRetrySuccessSummary, summary.Type)

	items, total, err := GetGroupedLogs(GroupedLogParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.True(t, items[0].IsSummary)
	require.Equal(t, model.LogTypeRetrySuccessSummary, items[0].Type)
	require.Equal(t, summary.Id, items[0].SummaryLogId)
	require.Equal(t, summary.TerminalLogId, items[0].TerminalLogId)
	require.Equal(t, 152, items[0].Quota)
}

func TestGroupedLogsExplicitChildTypeFilterShowsSummaryChildren(t *testing.T) {
	cleanLogs(t)
	require.NoError(t, model.LOG_DB.Exec("DELETE FROM logs").Error)

	const requestId = "grouped_child_filter_after_summary"
	insertGroupedLogTestRow(t, model.Log{
		RequestId: requestId,
		Type:      model.LogTypeErrorIntercepted,
		CreatedAt: 600,
		ChannelId: 12,
		Content:   "intercepted",
	})
	insertGroupedLogTestRow(t, model.Log{
		RequestId: requestId,
		Type:      model.LogTypeErrorClientVisible,
		CreatedAt: 601,
		ChannelId: 12,
		Content:   "visible error",
	})
	summary, err := model.UpsertTraceSummary(requestId)
	require.NoError(t, err)
	require.NotNil(t, summary)
	require.Equal(t, model.LogTypeRetryFailedSummary, summary.Type)

	items, total, err := GetGroupedLogs(GroupedLogParams{
		Page:     1,
		PageSize: 20,
		LogType:  model.LogTypeErrorIntercepted,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.False(t, items[0].IsSummary)
	require.Equal(t, model.LogTypeErrorIntercepted, items[0].Type)
	require.Equal(t, summary.Id, items[0].SummaryLogId)
}

func TestUsageStatsCountDirectAndSummaryConsumptionOnly(t *testing.T) {
	cleanLogs(t)
	require.NoError(t, model.LOG_DB.Exec("DELETE FROM logs").Error)

	now := common.GetTimestamp()
	rows := []model.Log{
		{
			RequestId:        "direct_success",
			Type:             model.LogTypeConsume,
			CreatedAt:        now,
			Quota:            10,
			PromptTokens:     3,
			CompletionTokens: 4,
		},
		{
			RequestId:        "sft_summary",
			Type:             model.LogTypeRetrySuccessSummary,
			CreatedAt:        now,
			Quota:            20,
			PromptTokens:     5,
			CompletionTokens: 6,
		},
		{
			RequestId:        "sft_child_success",
			Type:             model.LogTypeRetryConsume,
			CreatedAt:        now,
			Quota:            20,
			PromptTokens:     5,
			CompletionTokens: 6,
			SummaryLogId:     2,
		},
		{
			RequestId:        "probe_success",
			Type:             model.LogTypeProbeSuccess,
			CreatedAt:        now,
			Quota:            7,
			PromptTokens:     8,
			CompletionTokens: 1,
		},
	}
	for _, row := range rows {
		insertGroupedLogTestRow(t, row)
	}

	stat, err := model.SumUsedQuota(model.LogTypeUnknown, 0, 9999999999, "", "", "", 0, "")
	require.NoError(t, err)
	require.Equal(t, 30, stat.Quota)
	require.Equal(t, 18, stat.Tpm)
	require.Equal(t, 18, model.SumUsedToken(model.LogTypeUnknown, 0, 9999999999, "", "", ""))
}

func TestGroupedLogsExcludesStandaloneProbeRowsByDefault(t *testing.T) {
	cleanLogs(t)
	require.NoError(t, model.LOG_DB.Exec("DELETE FROM logs").Error)

	insertGroupedLogTestRow(t, model.Log{
		Type:      model.LogTypeProbeSuccess,
		CreatedAt: 400,
		ChannelId: 13,
		Content:   "standalone probe success",
	})
	insertGroupedLogTestRow(t, model.Log{
		Type:      model.LogTypeProbeFailed,
		CreatedAt: 401,
		ChannelId: 14,
		Content:   "standalone probe failed",
	})
	insertGroupedLogTestRow(t, model.Log{
		RequestId: "trace_with_probe",
		Type:      model.LogTypeProbeFailed,
		CreatedAt: 402,
		ChannelId: 14,
		Content:   "trace probe failed",
	})
	insertGroupedLogTestRow(t, model.Log{
		RequestId: "trace_with_probe",
		Type:      model.LogTypeConsume,
		CreatedAt: 403,
		ChannelId: 13,
		Content:   "trace consume",
	})

	items, total, err := GetGroupedLogs(GroupedLogParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, items, 2)
	for _, item := range items {
		require.Equal(t, "trace_with_probe", item.RequestId)
		require.NotEmpty(t, item.TraceId)
	}

	items, total, err = GetGroupedLogs(GroupedLogParams{
		Page:      1,
		PageSize:  20,
		RequestId: "trace_with_probe",
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, items, 2)
}

func TestGroupedLogsFiltersFlatRowsByRequestAndChannel(t *testing.T) {
	cleanLogs(t)
	require.NoError(t, model.LOG_DB.Exec("DELETE FROM logs").Error)

	insertGroupedLogTestRow(t, model.Log{
		RequestId: "target_trace",
		Type:      model.LogTypeErrorIntercepted,
		CreatedAt: 300,
		ChannelId: 12,
		Username:  "nacp_t_filter",
		TokenName: "filter-token",
		ModelName: "claude-sonnet",
		Group:     "default",
	})
	insertGroupedLogTestRow(t, model.Log{
		RequestId:    "target_trace",
		Type:         model.LogTypeConsume,
		CreatedAt:    309,
		ChannelId:    13,
		Username:     "nacp_t_filter",
		TokenName:    "filter-token",
		ModelName:    "claude-sonnet",
		Group:        "default",
		Quota:        152,
		PromptTokens: 18,
	})
	insertGroupedLogTestRow(t, model.Log{
		RequestId: "other_trace",
		Type:      model.LogTypeConsume,
		CreatedAt: 310,
		ChannelId: 13,
		Username:  "nacp_t_filter",
		TokenName: "filter-token",
		ModelName: "claude-sonnet",
		Group:     "default",
	})

	items, total, err := GetGroupedLogs(GroupedLogParams{
		Page:      1,
		PageSize:  20,
		RequestId: "target_trace",
		Channel:   13,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.False(t, items[0].IsSummary)
	require.Equal(t, model.LogTypeConsume, items[0].Type)
	require.Equal(t, "target_trace", items[0].RequestId)
	require.Equal(t, 13, items[0].ChannelId)
	require.Equal(t, 152, items[0].Quota)
	require.Equal(t, "nacp_t_filter", items[0].Username)
}
