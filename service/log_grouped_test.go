package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func insertGroupedLogTestRow(t *testing.T, log model.Log) model.Log {
	t.Helper()
	require.NoError(t, model.LOG_DB.Create(&log).Error)
	return log
}

func TestGroupedLogsReturnsFlatRowsAndNormalizesTraceTypes(t *testing.T) {
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
		require.NotContains(t, []int{20, 21, 29, 50, 51, 52, 59}, item.Type)
		require.Equal(t, requestId, item.RequestId)
		require.Equal(t, requestId, item.TraceId)
		require.NotZero(t, item.TraceSeq)
	}

	require.Equal(t, model.LogTypeConsume, items[0].Type)
	require.Equal(t, model.TraceRoleConsume, items[0].TraceRole)
	require.Equal(t, model.LogTypeError, items[1].Type)
	require.Equal(t, model.TraceRoleErrorVisible, items[1].TraceRole)
	require.Equal(t, model.LogTypeError, items[2].Type)
	require.Equal(t, model.TraceRoleProbeFailed, items[2].TraceRole)
	require.Equal(t, model.LogTypeSystem, items[3].Type)
	require.Equal(t, model.TraceRoleProbeSuccess, items[3].TraceRole)
	require.Equal(t, model.LogTypeError, items[4].Type)
	require.Equal(t, model.TraceRoleErrorIntercepted, items[4].TraceRole)
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
