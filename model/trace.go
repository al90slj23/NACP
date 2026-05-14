package model

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

// TraceListParams 链路列表查询参数
type TraceListParams struct {
	Page           int    `json:"page"`
	PageSize       int    `json:"page_size"`
	StartTimestamp int64  `json:"start_timestamp"`
	EndTimestamp   int64  `json:"end_timestamp"`
	ModelName      string `json:"model_name"`
	Username       string `json:"username"`
	Status         string `json:"status"` // "success" or "failed"
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
	Id          int    `json:"id"`
	ChannelId   int    `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Type        int    `json:"type"`
	StatusCode  *int   `json:"status_code"`
	UseTime     int    `json:"use_time"`
	ModelName   string `json:"model_name"`
	Quota       int    `json:"quota"`
	CreatedAt   int64  `json:"created_at"`
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

// traceAggRow represents a row from the GROUP BY aggregation query
type traceAggRow struct {
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

// GetTraceList queries trace summary list with GROUP BY aggregation
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

	tx := LOG_DB.Table("logs").
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
	countTx := LOG_DB.Table("(?) AS sub", tx).Count(&total)
	if countTx.Error != nil {
		return nil, 0, countTx.Error
	}

	// Apply ordering and pagination
	offset := (params.Page - 1) * params.PageSize
	var rows []traceAggRow
	err := tx.Order("created_at DESC").
		Limit(params.PageSize).
		Offset(offset).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	// Convert to TraceSummary with status filtering in application layer
	var results []TraceSummary
	for _, row := range rows {
		status := "failed"
		if row.HasSuccess == 1 {
			status = "success"
		}

		// Apply status filter in application layer
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

	// Adjust total count when status filter is applied
	if params.Status != "" {
		total = int64(len(results))
	}

	return results, total, nil
}

// GetTraceDetail queries trace detail for a specific request_id
func GetTraceDetail(requestId string) (*TraceDetail, error) {
	var logs []Log
	err := LOG_DB.Where("request_id = ? AND type IN (2, 5, 51, 52)", requestId).
		Order("created_at ASC").
		Limit(100).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}

	detail := &TraceDetail{
		RequestId: requestId,
		Steps:     make([]TraceStep, 0, len(logs)),
	}

	if len(logs) == 0 {
		return detail, nil
	}

	// Collect channel IDs for bulk name lookup
	channelIds := types.NewSet[int]()
	for _, l := range logs {
		if l.ChannelId != 0 {
			channelIds.Add(l.ChannelId)
		}
	}

	// Bulk query channel names (reuse CacheGetChannel pattern from GetAllLogs)
	channelMap := make(map[int]string)
	if channelIds.Len() > 0 {
		if common.MemoryCacheEnabled {
			for _, channelId := range channelIds.Items() {
				if cacheChannel, err := CacheGetChannel(channelId); err == nil {
					channelMap[channelId] = cacheChannel.Name
				}
			}
		} else {
			var channels []struct {
				Id   int    `gorm:"column:id"`
				Name string `gorm:"column:name"`
			}
			if err := DB.Table("channels").Select("id, name").Where("id IN ?", channelIds.Items()).Find(&channels).Error; err == nil {
				for _, ch := range channels {
					channelMap[ch.Id] = ch.Name
				}
			}
		}
	}

	// Build steps and aggregate summary info
	var totalQuota, totalPromptTokens, totalCompletionTokens int
	var minCreatedAt int64

	for i, l := range logs {
		// Track earliest created_at
		if i == 0 || l.CreatedAt < minCreatedAt {
			minCreatedAt = l.CreatedAt
		}

		// Parse status_code from Other JSON field
		var statusCode *int
		if l.Other != "" {
			otherMap, mapErr := common.StrToMap(l.Other)
			if mapErr == nil && otherMap != nil {
				if adminInfo, ok := otherMap["admin_info"]; ok {
					if adminInfoMap, ok := adminInfo.(map[string]interface{}); ok {
						if sc, ok := adminInfoMap["status_code"]; ok {
							if scFloat, ok := sc.(float64); ok {
								scInt := int(scFloat)
								statusCode = &scInt
							}
						}
					}
				}
			}
		}

		step := TraceStep{
			Id:          l.Id,
			ChannelId:   l.ChannelId,
			ChannelName: channelMap[l.ChannelId],
			Type:        l.Type,
			StatusCode:  statusCode,
			UseTime:     l.UseTime,
			ModelName:   l.ModelName,
			Quota:       l.Quota,
			CreatedAt:   l.CreatedAt,
		}
		detail.Steps = append(detail.Steps, step)

		// Aggregate totals from type=2 (Consume) logs
		if l.Type == LogTypeConsume {
			totalQuota += l.Quota
			totalPromptTokens += l.PromptTokens
			totalCompletionTokens += l.CompletionTokens
		}
	}

	// Set summary fields from first log (they share the same request)
	firstLog := logs[0]
	detail.CreatedAt = minCreatedAt
	detail.ModelName = firstLog.ModelName
	detail.Username = firstLog.Username
	detail.TokenName = firstLog.TokenName
	detail.TotalQuota = totalQuota
	detail.TotalPromptTokens = totalPromptTokens
	detail.TotalCompletionTokens = totalCompletionTokens

	return detail, nil
}
