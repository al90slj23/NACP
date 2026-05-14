package service

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
)

// TraceListParams 链路列表查询参数
type TraceListParams struct {
	Page           int
	PageSize       int
	StartTimestamp int64
	EndTimestamp   int64
	ModelName      string
	Username       string
	Status         string // "success" or "failed"
}

// TraceSummary 链路摘要 DTO
type TraceSummary struct {
	RequestId             string `json:"request_id"`
	CreatedAt             int64  `json:"created_at"`
	ModelName             string `json:"model_name"`
	Username              string `json:"username"`
	TokenName             string `json:"token_name"`
	Status                string `json:"status"`
	ChannelCount          int    `json:"channel_count"`
	TotalDuration         int64  `json:"total_duration"`
	TotalQuota            int    `json:"total_quota"`
	TotalPromptTokens     int    `json:"total_prompt_tokens"`
	TotalCompletionTokens int    `json:"total_completion_tokens"`
}

// TraceStep 链路步骤 DTO
type TraceStep struct {
	Id               int    `json:"id"`
	ChannelId        int    `json:"channel_id"`
	ChannelName      string `json:"channel_name"`
	Type             int    `json:"type"`
	StatusCode       *int   `json:"status_code"`
	UseTime          int    `json:"use_time"`
	ModelName        string `json:"model_name"`
	Username         string `json:"username"`
	TokenName        string `json:"token_name"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	Quota            int    `json:"quota"`
	Group            string `json:"group"`
	Ip               string `json:"ip"`
	IsStream         bool   `json:"is_stream"`
	Content          string `json:"content"`
	CreatedAt        int64  `json:"created_at"`
	Other            string `json:"other,omitempty"`
}

// TraceDetail 链路详情 DTO
type TraceDetail struct {
	RequestId             string      `json:"request_id"`
	CreatedAt             int64       `json:"created_at"`
	ModelName             string      `json:"model_name"`
	Username              string      `json:"username"`
	TokenName             string      `json:"token_name"`
	TotalQuota            int         `json:"total_quota"`
	TotalPromptTokens     int         `json:"total_prompt_tokens"`
	TotalCompletionTokens int         `json:"total_completion_tokens"`
	Steps                 []TraceStep `json:"steps"`
}

// traceSummaryRow is the raw row scanned from the GROUP BY aggregation query.
type traceSummaryRow struct {
	RequestId             string `gorm:"column:request_id"`
	CreatedAt             int64  `gorm:"column:created_at"`
	MaxCreatedAt          int64  `gorm:"column:max_created_at"`
	ModelName             string `gorm:"column:model_name"`
	Username              string `gorm:"column:username"`
	TokenName             string `gorm:"column:token_name"`
	ChannelCount          int    `gorm:"column:channel_count"`
	HasSuccess            int    `gorm:"column:has_success"`
	TotalQuota            int    `gorm:"column:total_quota"`
	TotalPromptTokens     int    `gorm:"column:total_prompt_tokens"`
	TotalCompletionTokens int    `gorm:"column:total_completion_tokens"`
	LogCount              int    `gorm:"column:log_count"`
	HasError              int    `gorm:"column:has_error"`
}

// GetTraceList queries the trace summary list with pagination and filtering.
func GetTraceList(params TraceListParams) ([]TraceSummary, int64, error) {
	selectSQL := `
		request_id,
		MIN(created_at) AS created_at,
		MAX(created_at) AS max_created_at,
		model_name,
		username,
		token_name,
		COUNT(DISTINCT channel_id) AS channel_count,
		MAX(CASE WHEN type = 2 THEN 1 ELSE 0 END) AS has_success,
		SUM(CASE WHEN type = 2 THEN quota ELSE 0 END) AS total_quota,
		SUM(CASE WHEN type = 2 THEN prompt_tokens ELSE 0 END) AS total_prompt_tokens,
		SUM(CASE WHEN type = 2 THEN completion_tokens ELSE 0 END) AS total_completion_tokens,
		COUNT(*) AS log_count,
		MAX(CASE WHEN type IN (5, 51, 52) THEN 1 ELSE 0 END) AS has_error
	`

	// Build WHERE conditions
	tx := model.LOG_DB.Table("logs").
		Select(selectSQL).
		Where("type IN (2, 5, 51, 52)").
		Where("request_id != ''")

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

	tx = tx.Group("request_id, model_name, username, token_name").
		Having("log_count >= 2 OR has_error = 1")

	// Count total matching records using a subquery approach
	var total int64
	countTx := model.LOG_DB.Table("(?) AS sub", tx).Count(&total)
	if countTx.Error != nil {
		return nil, 0, countTx.Error
	}

	// Apply ordering and pagination
	offset := (params.Page - 1) * params.PageSize
	var rows []traceSummaryRow
	err := tx.Order("created_at DESC").
		Limit(params.PageSize).
		Offset(offset).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	// Convert rows to TraceSummary, applying status filter at application layer
	results := make([]TraceSummary, 0, len(rows))
	for _, row := range rows {
		status := "failed"
		if row.HasSuccess == 1 {
			status = "success"
		}

		// Apply status filter at application layer
		if params.Status != "" && params.Status != status {
			continue
		}

		results = append(results, TraceSummary{
			RequestId:             row.RequestId,
			CreatedAt:             row.CreatedAt,
			ModelName:             row.ModelName,
			Username:              row.Username,
			TokenName:             row.TokenName,
			Status:                status,
			ChannelCount:          row.ChannelCount,
			TotalDuration:         row.MaxCreatedAt - row.CreatedAt,
			TotalQuota:            row.TotalQuota,
			TotalPromptTokens:     row.TotalPromptTokens,
			TotalCompletionTokens: row.TotalCompletionTokens,
		})
	}

	// If status filter was applied at application layer, adjust total count
	if params.Status != "" {
		total = int64(len(results))
	}

	return results, total, nil
}

// traceLogRow is the raw log row for trace detail query.
type traceLogRow struct {
	Id        int    `gorm:"column:id"`
	ChannelId int    `gorm:"column:channel_id"`
	Type      int    `gorm:"column:type"`
	UseTime   int    `gorm:"column:use_time"`
	ModelName string `gorm:"column:model_name"`
	Quota     int    `gorm:"column:quota"`
	CreatedAt int64  `gorm:"column:created_at"`
	Other     string `gorm:"column:other"`
	// Fields for summary
	Username         string `gorm:"column:username"`
	TokenName        string `gorm:"column:token_name"`
	PromptTokens     int    `gorm:"column:prompt_tokens"`
	CompletionTokens int    `gorm:"column:completion_tokens"`
	Group            string `gorm:"column:group_val"`
	Ip               string `gorm:"column:ip"`
	IsStream         bool   `gorm:"column:is_stream"`
	Content          string `gorm:"column:content"`
}

// GetTraceDetail queries the full trace detail for a given request_id.
func GetTraceDetail(requestId string) (*TraceDetail, error) {
	var rows []traceLogRow
	err := model.LOG_DB.Table("logs").
		Select("id, channel_id, type, use_time, model_name, quota, created_at, other, username, token_name, prompt_tokens, completion_tokens, "+logGroupCol()+" AS group_val, ip, is_stream, content").
		Where("request_id = ?", requestId).
		Where("type IN (2, 5, 51, 52, 29, 59)").
		Order("created_at ASC, id ASC").
		Limit(100).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	detail := &TraceDetail{
		RequestId: requestId,
		Steps:     make([]TraceStep, 0, len(rows)),
	}

	if len(rows) == 0 {
		return detail, nil
	}

	// Collect channel IDs for batch name resolution
	channelIds := types.NewSet[int]()
	for _, row := range rows {
		if row.ChannelId != 0 {
			channelIds.Add(row.ChannelId)
		}
	}

	// Resolve channel names (reuse CacheGetChannel pattern from model/log.go)
	channelMap := make(map[int]string)
	if channelIds.Len() > 0 {
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
	}

	// Build steps and aggregate summary info
	var totalQuota, totalPromptTokens, totalCompletionTokens int
	var minCreatedAt int64

	for i, row := range rows {
		// Track summary info from first row
		if i == 0 {
			detail.ModelName = row.ModelName
			detail.Username = row.Username
			detail.TokenName = row.TokenName
			minCreatedAt = row.CreatedAt
		}
		if row.CreatedAt < minCreatedAt {
			minCreatedAt = row.CreatedAt
		}

		// Aggregate quota/tokens from type=2 (Consume) logs only
		if row.Type == model.LogTypeConsume {
			totalQuota += row.Quota
			totalPromptTokens += row.PromptTokens
			totalCompletionTokens += row.CompletionTokens
		}

		// Parse status_code from Other field JSON
		statusCode := parseStatusCodeFromOther(row.Other)

		step := TraceStep{
			Id:               row.Id,
			ChannelId:        row.ChannelId,
			ChannelName:      channelMap[row.ChannelId],
			Type:             row.Type,
			StatusCode:       statusCode,
			UseTime:          row.UseTime,
			ModelName:        row.ModelName,
			Username:         row.Username,
			TokenName:        row.TokenName,
			PromptTokens:     row.PromptTokens,
			CompletionTokens: row.CompletionTokens,
			Quota:            row.Quota,
			Group:            row.Group,
			Ip:               row.Ip,
			IsStream:         row.IsStream,
			Content:          row.Content,
			CreatedAt:        row.CreatedAt,
			Other:            row.Other,
		}
		detail.Steps = append(detail.Steps, step)
	}

	detail.CreatedAt = minCreatedAt
	detail.TotalQuota = totalQuota
	detail.TotalPromptTokens = totalPromptTokens
	detail.TotalCompletionTokens = totalCompletionTokens

	return detail, nil
}

// parseStatusCodeFromOther extracts admin_info.status_code from the Other JSON field.
// Returns nil if Other is empty, invalid JSON, or doesn't contain the field.
func parseStatusCodeFromOther(other string) *int {
	if other == "" {
		return nil
	}

	otherMap, err := common.StrToMap(other)
	if err != nil {
		return nil
	}

	adminInfoRaw, ok := otherMap["admin_info"]
	if !ok {
		return nil
	}

	adminInfo, ok := adminInfoRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	statusCodeRaw, ok := adminInfo["status_code"]
	if !ok {
		return nil
	}

	// JSON numbers are decoded as float64 by default
	switch v := statusCodeRaw.(type) {
	case float64:
		code := int(v)
		return &code
	case int:
		return &v
	default:
		// Try to handle via fmt conversion for edge cases
		var code int
		if _, err := fmt.Sscanf(fmt.Sprintf("%v", v), "%d", &code); err == nil {
			return &code
		}
		return nil
	}
}
