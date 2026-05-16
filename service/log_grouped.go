package service

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"gorm.io/gorm"
)

// GroupedLogParams 分组日志查询参数
type GroupedLogParams struct {
	Page           int
	PageSize       int
	StartTimestamp int64
	EndTimestamp   int64
	LogType        int
	ModelName      string
	Username       string
	TokenName      string
	Channel        int
	Group          string
	RequestId      string
}

// GroupedLogItem keeps the historical /api/log/grouped response shape.
// The endpoint now returns flat log rows only; IsSummary stays false.
type GroupedLogItem struct {
	Id               int    `json:"id"`
	UserId           int    `json:"user_id"`
	Type             int    `json:"type"`
	CreatedAt        int64  `json:"created_at"`
	ModelName        string `json:"model_name"`
	Username         string `json:"username"`
	TokenName        string `json:"token_name"`
	TokenId          int    `json:"token_id"`
	Quota            int    `json:"quota"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	UseTime          int    `json:"use_time"`
	ChannelId        int    `json:"channel"`
	ChannelName      string `json:"channel_name"`
	RequestId        string `json:"request_id"`
	TraceId          string `json:"trace_id,omitempty"`
	TraceSeq         int    `json:"trace_seq,omitempty"`
	TraceParentId    int    `json:"trace_parent_id,omitempty"`
	TraceSiblingSeq  int    `json:"trace_sibling_seq,omitempty"`
	TraceRole        string `json:"trace_role,omitempty"`
	SummaryLogId     int    `json:"summary_log_id,omitempty"`
	TerminalLogId    int    `json:"terminal_log_id,omitempty"`
	TraceVersion     int    `json:"trace_version,omitempty"`
	Group            string `json:"group"`
	Ip               string `json:"ip"`
	Other            string `json:"other"`
	IsStream         bool   `json:"is_stream"`
	Content          string `json:"content"`
	SortAt           int64  `json:"-"`

	ChannelPath   string `json:"channel_path,omitempty"`
	TotalDuration int    `json:"total_duration,omitempty"`
	StepCount     int    `json:"step_count,omitempty"`
	IsSummary     bool   `json:"is_summary"`
}

// applyCommonFilters applies shared filtering conditions (time range,
// model_name, username, token_name, channel_id, group) to a GORM query builder.
// Compatible with SQLite, MySQL, and PostgreSQL.
func applyCommonFilters(tx *gorm.DB, params GroupedLogParams) *gorm.DB {
	if params.StartTimestamp != 0 {
		tx = tx.Where("created_at >= ?", params.StartTimestamp)
	}
	if params.EndTimestamp != 0 {
		tx = tx.Where("created_at <= ?", params.EndTimestamp)
	}
	if params.ModelName != "" {
		tx = tx.Where("model_name = ?", params.ModelName)
	}
	if params.Username != "" {
		tx = tx.Where("username = ?", params.Username)
	}
	if params.TokenName != "" {
		tx = tx.Where("token_name = ?", params.TokenName)
	}
	if params.Channel != 0 {
		tx = tx.Where("channel_id = ?", params.Channel)
	}
	if params.Group != "" {
		tx = tx.Where(logGroupCol()+" = ?", params.Group)
	}
	return tx
}

func logGroupCol() string {
	col := model.GetLogGroupCol()
	if col != "" {
		return col
	}
	if common.UsingPostgreSQL {
		return `"group"`
	}
	return "`group`"
}

// GetGroupedLogs returns the log list used by /console/log.
// New SFT traces are represented by materialized 20/50 summary rows. Child
// rows are hidden from the main list and loaded by /api/log/trace. Transitional
// traces without a materialized summary are still returned as flat rows so the
// frontend compatibility grouping can keep displaying older in-flight data.
func GetGroupedLogs(params GroupedLogParams) ([]GroupedLogItem, int64, error) {
	tx := model.LOG_DB.Model(&model.Log{})
	if params.LogType != model.LogTypeUnknown {
		tx = tx.Where("type = ?", params.LogType)
	}
	if params.RequestId != "" {
		tx = tx.Where("request_id = ?", params.RequestId)
	} else {
		tx = tx.Where("NOT (request_id = '' AND trace_role IN ?)", []string{
			model.TraceRoleProbeSuccess,
			model.TraceRoleProbeFailed,
		})
	}
	if params.LogType == model.LogTypeUnknown {
		tx = tx.Where("NOT (summary_log_id > 0 AND type NOT IN ?)", []int{
			model.LogTypeRetrySuccessSummary,
			model.LogTypeRetryFailedSummary,
		})
	}
	tx = applyCommonFilters(tx, params)

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (params.Page - 1) * params.PageSize
	var logs []*model.Log
	if err := tx.Order("id DESC").Limit(params.PageSize).Offset(offset).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	resolveChannelNames(logs)

	items := make([]GroupedLogItem, 0, len(logs))
	for _, l := range logs {
		items = append(items, logToGroupedItem(l))
	}
	return items, total, nil
}

func logToGroupedItem(l *model.Log) GroupedLogItem {
	return GroupedLogItem{
		Id:               l.Id,
		UserId:           l.UserId,
		Type:             l.Type,
		CreatedAt:        l.CreatedAt,
		ModelName:        l.ModelName,
		Username:         l.Username,
		TokenName:        l.TokenName,
		TokenId:          l.TokenId,
		Quota:            l.Quota,
		PromptTokens:     l.PromptTokens,
		CompletionTokens: l.CompletionTokens,
		UseTime:          l.UseTime,
		ChannelId:        l.ChannelId,
		ChannelName:      l.ChannelName,
		RequestId:        l.RequestId,
		TraceId:          l.TraceId,
		TraceSeq:         l.TraceSeq,
		TraceParentId:    l.TraceParentId,
		TraceSiblingSeq:  l.TraceSiblingSeq,
		TraceRole:        l.TraceRole,
		SummaryLogId:     l.SummaryLogId,
		TerminalLogId:    l.TerminalLogId,
		TraceVersion:     l.TraceVersion,
		Group:            l.Group,
		Ip:               l.Ip,
		Other:            l.Other,
		IsStream:         l.IsStream,
		Content:          l.Content,
		SortAt:           l.CreatedAt,
		IsSummary:        l.Type == model.LogTypeRetrySuccessSummary || l.Type == model.LogTypeRetryFailedSummary,
	}
}

func resolveChannelNames(logs []*model.Log) {
	if len(logs) == 0 {
		return
	}

	channelIds := types.NewSet[int]()
	for _, l := range logs {
		if l.ChannelId != 0 {
			channelIds.Add(l.ChannelId)
		}
	}
	if channelIds.Len() == 0 {
		return
	}

	channelMap := make(map[int]string)
	if common.MemoryCacheEnabled {
		for _, channelId := range channelIds.Items() {
			if ch, err := model.CacheGetChannel(channelId); err == nil {
				channelMap[channelId] = ch.Name
			}
		}
	} else {
		var channels []struct {
			Id   int    `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		if err := model.DB.Table("channels").Select("id, name").Where("id IN ?", channelIds.Items()).Find(&channels).Error; err == nil {
			for _, ch := range channels {
				channelMap[ch.Id] = ch.Name
			}
		}
	}

	for i := range logs {
		logs[i].ChannelName = channelMap[logs[i].ChannelId]
	}
}
