package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func insertGroupedLogTestRow(t *testing.T, log model.Log) {
	t.Helper()
	require.NoError(t, model.LOG_DB.Create(&log).Error)
}

func TestGroupedLogsRetrySummaryDedupesRequestAndKeepsFields(t *testing.T) {
	cleanLogs(t)
	require.NoError(t, model.LOG_DB.Exec("DELETE FROM logs").Error)

	const requestId = "grouped_retry_complete_fields"
	insertGroupedLogTestRow(t, model.Log{
		RequestId: requestId,
		Type:      model.LogTypeErrorIntercepted,
		CreatedAt: 200,
		ChannelId: 12,
		Content:   "first intercepted",
		Other:     `{"admin_info":{"status_code":500}}`,
	})
	insertGroupedLogTestRow(t, model.Log{
		RequestId: requestId,
		Type:      model.LogTypeErrorIntercepted,
		CreatedAt: 201,
		ChannelId: 12,
		Content:   "second intercepted",
		Other:     `{"admin_info":{"status_code":500}}`,
	})
	insertGroupedLogTestRow(t, model.Log{
		RequestId: requestId,
		Type:      model.LogTypeErrorClientVisible,
		CreatedAt: 203,
		UserId:    7,
		Username:  "nacp_t_ctx",
		TokenId:   8,
		TokenName: "ctx-token",
		ModelName: "gpt-5.3-codex",
		Group:     "default",
		ChannelId: 14,
		UseTime:   3,
		Ip:        "127.0.0.1",
		Content:   "client visible error",
		Other:     `{"admin_info":{"status_code":503}}`,
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
		Other:            `{"frt":2000,"admin_info":{"status_code":200}}`,
	})

	items, total, err := GetGroupedLogs(GroupedLogParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)

	item := items[0]
	require.True(t, item.IsSummary)
	require.Equal(t, 20, item.Type)
	require.Equal(t, requestId, item.RequestId)
	require.Equal(t, "12→14→13", item.ChannelPath)
	require.Equal(t, "nacp_t_ctx", item.Username)
	require.Equal(t, "ctx-token", item.TokenName)
	require.Equal(t, "gpt-5.3-codex", item.ModelName)
	require.Equal(t, "default", item.Group)
	require.Equal(t, 13, item.ChannelId)
	require.Equal(t, 580, item.Quota)
	require.Equal(t, 360, item.PromptTokens)
	require.Equal(t, 13, item.CompletionTokens)
	require.True(t, item.IsStream)
	require.Equal(t, "127.0.0.1", item.Ip)
	require.Equal(t, "success content", item.Content)
	require.Equal(t, int64(200), item.CreatedAt)
	require.Equal(t, int64(214), item.SortAt)
}

func TestGroupedLogsChannelFilterKeepsFullRetryTrace(t *testing.T) {
	cleanLogs(t)
	require.NoError(t, model.LOG_DB.Exec("DELETE FROM logs").Error)

	const requestId = "grouped_retry_channel_filter"
	insertGroupedLogTestRow(t, model.Log{
		RequestId: requestId,
		Type:      model.LogTypeErrorIntercepted,
		CreatedAt: 300,
		ChannelId: 12,
	})
	insertGroupedLogTestRow(t, model.Log{
		RequestId: requestId,
		Type:      model.LogTypeErrorClientVisible,
		CreatedAt: 303,
		ChannelId: 14,
		Username:  "nacp_t_filter",
		TokenName: "filter-token",
		ModelName: "claude-sonnet",
		Group:     "default",
	})
	insertGroupedLogTestRow(t, model.Log{
		RequestId:        requestId,
		Type:             model.LogTypeConsume,
		CreatedAt:        309,
		ChannelId:        13,
		Username:         "nacp_t_filter",
		TokenName:        "filter-token",
		ModelName:        "claude-sonnet",
		Group:            "default",
		Quota:            152,
		PromptTokens:     18,
		CompletionTokens: 13,
	})

	items, total, err := GetGroupedLogs(GroupedLogParams{Page: 1, PageSize: 20, Channel: 13})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.Equal(t, "12→14→13", items[0].ChannelPath)
	require.Equal(t, 152, items[0].Quota)
	require.Equal(t, "nacp_t_filter", items[0].Username)
}

func TestGroupedLogsDoesNotCreateFailedSummaryWithoutTerminal52(t *testing.T) {
	cleanLogs(t)
	require.NoError(t, model.LOG_DB.Exec("DELETE FROM logs").Error)

	const requestId = "grouped_retry_incomplete_51_only"
	insertGroupedLogTestRow(t, model.Log{
		RequestId: requestId,
		Type:      model.LogTypeErrorIntercepted,
		CreatedAt: 400,
		ChannelId: 12,
		Content:   "intercepted but not terminal",
		Other:     `{"admin_info":{"status_code":500}}`,
	})
	insertGroupedLogTestRow(t, model.Log{
		RequestId: requestId,
		Type:      model.LogTypeErrorIntercepted,
		CreatedAt: 401,
		ChannelId: 12,
		Content:   "second intercepted but not terminal",
		Other:     `{"admin_info":{"status_code":499}}`,
	})

	items, total, err := GetGroupedLogs(GroupedLogParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.EqualValues(t, 0, total)
	require.Len(t, items, 0)
}

func TestGroupedLogsCreatesFailedSummaryOnlyWithTerminal52(t *testing.T) {
	cleanLogs(t)
	require.NoError(t, model.LOG_DB.Exec("DELETE FROM logs").Error)

	const requestId = "grouped_retry_terminal_52"
	insertGroupedLogTestRow(t, model.Log{
		RequestId: requestId,
		Type:      model.LogTypeErrorIntercepted,
		CreatedAt: 500,
		ChannelId: 12,
		Content:   "intercepted",
		Other:     `{"admin_info":{"status_code":500}}`,
	})
	insertGroupedLogTestRow(t, model.Log{
		RequestId: requestId,
		Type:      model.LogTypeProbeFailed,
		CreatedAt: 501,
		ChannelId: 13,
		Content:   "probe failed",
		Other:     `{"admin_info":{"status_code":503}}`,
	})
	insertGroupedLogTestRow(t, model.Log{
		RequestId: requestId,
		Type:      model.LogTypeErrorClientVisible,
		CreatedAt: 502,
		ChannelId: 14,
		Username:  "nacp_t_failed",
		TokenName: "failed-token",
		ModelName: "claude-haiku",
		Group:     "default",
		Content:   "client visible terminal error",
		Other:     `{"admin_info":{"status_code":503}}`,
	})

	items, total, err := GetGroupedLogs(GroupedLogParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.True(t, items[0].IsSummary)
	require.Equal(t, 50, items[0].Type)
	require.Equal(t, requestId, items[0].RequestId)
	require.Equal(t, "12→14", items[0].ChannelPath)
	require.Equal(t, "nacp_t_failed", items[0].Username)
}
