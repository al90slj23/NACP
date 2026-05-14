package service

import (
	"sort"
	"strconv"
	"strings"

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

// GroupedLogItem 统一的列表项（摘要行和普通行共用）
type GroupedLogItem struct {
	// 通用字段（两种行都有）
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
	Group            string `json:"group"`
	Ip               string `json:"ip"`
	Other            string `json:"other"`
	IsStream         bool   `json:"is_stream"`
	Content          string `json:"content"`
	SortAt           int64  `json:"-"`

	// 摘要行专用字段（普通行为零值）
	ChannelPath   string `json:"channel_path,omitempty"`
	TotalDuration int    `json:"total_duration,omitempty"`
	StepCount     int    `json:"step_count,omitempty"`
	IsSummary     bool   `json:"is_summary"`
}

// buildChannelPath removes consecutive duplicate channel_ids and joins them with "→".
// Example: [12, 12, 14, 14, 12] → "12→14→12"
func buildChannelPath(channelIds []int) string {
	if len(channelIds) == 0 {
		return ""
	}
	deduped := []int{channelIds[0]}
	for i := 1; i < len(channelIds); i++ {
		if channelIds[i] != channelIds[i-1] {
			deduped = append(deduped, channelIds[i])
		}
	}
	parts := make([]string, len(deduped))
	for i, id := range deduped {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, "→")
}

// applyCommonFilters applies shared filtering conditions (time range, model_name,
// username, token_name, channel_id, group) to a GORM query builder.
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

// traceSummaryRow is the raw row scanned from the GROUP BY aggregation query
// for grouped log listing.
type groupedTraceSummaryRow struct {
	Id                    int    `gorm:"column:id"`
	RequestId             string `gorm:"column:request_id"`
	CreatedAt             int64  `gorm:"column:created_at"`
	MaxCreatedAt          int64  `gorm:"column:max_created_at"`
	UserId                int    `gorm:"column:user_id"`
	ModelName             string `gorm:"column:model_name"`
	Username              string `gorm:"column:username"`
	TokenName             string `gorm:"column:token_name"`
	TokenId               int    `gorm:"column:token_id"`
	ChannelId             int    `gorm:"column:channel_id"`
	ChannelCount          int    `gorm:"column:channel_count"`
	HasSuccess            int    `gorm:"column:has_success"`
	TotalQuota            int    `gorm:"column:total_quota"`
	TotalPromptTokens     int    `gorm:"column:total_prompt_tokens"`
	TotalCompletionTokens int    `gorm:"column:total_completion_tokens"`
	LogCount              int    `gorm:"column:log_count"`
	HasError              int    `gorm:"column:has_error"`
	Group                 string `gorm:"column:group_val"`
	Ip                    string `gorm:"column:ip"`
	Other                 string `gorm:"column:other"`
	HasStream             int    `gorm:"column:has_stream"`
	Content               string `gorm:"column:content"`
}

// GetGroupedLogs returns a mixed list of summary rows and normal rows,
// sorted by created_at DESC with pagination.
func GetGroupedLogs(params GroupedLogParams) ([]GroupedLogItem, int64, error) {
	// Special branch: RequestId non-empty → keep retry traces grouped, keep normal requests flat.
	if params.RequestId != "" {
		return getLogsForRequestId(params)
	}

	// Special branch: LogType=2 → only normal consume rows (no retry activity)
	if params.LogType == model.LogTypeConsume {
		return getNormalLogsOnly(params)
	}

	// Special branch: LogType=51 or LogType=52 → only summary rows containing that type
	if params.LogType == model.LogTypeErrorIntercepted || params.LogType == model.LogTypeErrorClientVisible {
		return getSummaryLogsWithType(params)
	}

	// General branch: two-phase query + application-layer merge

	// Get all retry request_ids (for exclusion in Phase 2)
	retryReqIds := getRetryRequestIds(params)

	// Phase 1: Query summary rows
	var summaryRows []groupedTraceSummaryRow
	var summaryTotal int64

	summaryTx := traceSummaryQuery(retryReqIds)

	// Count summaries
	countTx := model.LOG_DB.Table("(?) AS sub", summaryTx).Count(&summaryTotal)
	if countTx.Error != nil {
		return nil, 0, countTx.Error
	}

	// Phase 2: Count normal rows
	normalTx := normalLogsQuery(params, retryReqIds)
	var normalTotal int64
	if err := normalTx.Count(&normalTotal).Error; err != nil {
		return nil, 0, err
	}

	total := summaryTotal + normalTotal

	// Fetch data for merge: get offset+pageSize from each source
	offset := (params.Page - 1) * params.PageSize
	fetchSize := offset + params.PageSize

	// Fetch summary rows
	if err := traceSummaryQuery(retryReqIds).
		Order("max_created_at DESC").
		Limit(fetchSize).
		Find(&summaryRows).Error; err != nil {
		return nil, 0, err
	}

	// Fetch normal rows
	var normalLogs []*model.Log
	if err := normalLogsQuery(params, retryReqIds).
		Order("id DESC").
		Limit(fetchSize).
		Find(&normalLogs).Error; err != nil {
		return nil, 0, err
	}

	// Get channel paths for summary rows
	reqIds := make([]string, 0, len(summaryRows))
	for _, row := range summaryRows {
		reqIds = append(reqIds, row.RequestId)
	}
	channelPaths, _ := getChannelPaths(reqIds)

	// Resolve channel names for normal logs
	resolveChannelNames(normalLogs)

	// Merge and paginate
	items := mergeAndPaginate(summaryRows, normalLogs, channelPaths, offset, params.PageSize)

	return items, total, nil
}

func getLogsForRequestId(params GroupedLogParams) ([]GroupedLogItem, int64, error) {
	var retryCount int64
	err := model.LOG_DB.Model(&model.Log{}).
		Where("request_id = ?", params.RequestId).
		Where("type IN (51, 52, 29, 59)").
		Count(&retryCount).Error
	if err != nil {
		return nil, 0, err
	}
	if retryCount > 0 {
		return getSummaryLogsForRequestId(params)
	}
	return getFlatLogsForRequestId(params)
}

func getSummaryLogsForRequestId(params GroupedLogParams) ([]GroupedLogItem, int64, error) {
	summaryTx := traceSummaryQueryForIds([]string{params.RequestId})

	var row groupedTraceSummaryRow
	err := summaryTx.First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return []GroupedLogItem{}, 0, nil
		}
		return nil, 0, err
	}

	channelPaths, _ := getChannelPaths([]string{row.RequestId})
	return []GroupedLogItem{summaryRowToGroupedItem(row, channelPaths)}, 1, nil
}

// getFlatLogsForRequestId returns all non-retry logs for a specific request_id.
func getFlatLogsForRequestId(params GroupedLogParams) ([]GroupedLogItem, int64, error) {
	tx := model.LOG_DB.Where("request_id = ?", params.RequestId)
	if params.StartTimestamp != 0 {
		tx = tx.Where("created_at >= ?", params.StartTimestamp)
	}
	if params.EndTimestamp != 0 {
		tx = tx.Where("created_at <= ?", params.EndTimestamp)
	}

	var total int64
	if err := tx.Model(&model.Log{}).Count(&total).Error; err != nil {
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

// getNormalLogsOnly returns only normal consume rows (type=2) without retry activity.
func getNormalLogsOnly(params GroupedLogParams) ([]GroupedLogItem, int64, error) {
	retryReqIds := getRetryRequestIds(params)

	tx := model.LOG_DB.Where("type = ?", model.LogTypeConsume)
	if len(retryReqIds) > 0 {
		tx = tx.Where("(request_id NOT IN ? OR request_id = '')", retryReqIds)
	}
	tx = applyCommonFilters(tx, params)

	var total int64
	if err := tx.Model(&model.Log{}).Count(&total).Error; err != nil {
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

// getSummaryLogsWithType returns only summary rows that contain the specified log type (51 or 52).
func getSummaryLogsWithType(params GroupedLogParams) ([]GroupedLogItem, int64, error) {
	// Find request_ids that have the specified type
	subQuery := model.LOG_DB.Table("logs").
		Select("DISTINCT request_id").
		Where("type = ?", params.LogType).
		Where("request_id != ''")
	if params.StartTimestamp != 0 {
		subQuery = subQuery.Where("created_at >= ?", params.StartTimestamp)
	}
	if params.EndTimestamp != 0 {
		subQuery = subQuery.Where("created_at <= ?", params.EndTimestamp)
	}

	var targetReqIds []string
	if err := subQuery.Pluck("request_id", &targetReqIds).Error; err != nil {
		return nil, 0, err
	}
	targetReqIds = filterTraceRequestIdsByCommonFilters(params, targetReqIds)

	if len(targetReqIds) == 0 {
		return []GroupedLogItem{}, 0, nil
	}

	// Build summary query for these specific request_ids
	summaryTx := traceSummaryQueryForIds(targetReqIds)

	var total int64
	countTx := model.LOG_DB.Table("(?) AS sub", summaryTx).Count(&total)
	if countTx.Error != nil {
		return nil, 0, countTx.Error
	}

	offset := (params.Page - 1) * params.PageSize
	var summaryRows []groupedTraceSummaryRow
	if err := summaryTx.Order("max_created_at DESC").
		Limit(params.PageSize).
		Offset(offset).
		Find(&summaryRows).Error; err != nil {
		return nil, 0, err
	}

	// Get channel paths
	reqIds := make([]string, 0, len(summaryRows))
	for _, row := range summaryRows {
		reqIds = append(reqIds, row.RequestId)
	}
	channelPaths, _ := getChannelPaths(reqIds)

	items := make([]GroupedLogItem, 0, len(summaryRows))
	for _, row := range summaryRows {
		items = append(items, summaryRowToGroupedItem(row, channelPaths))
	}
	return items, total, nil
}

// getRetryRequestIds returns request_ids that have retry activity and match the
// current list filters. The returned ids are candidates only; summary rows are
// later aggregated from the full trace so child fields are not cut apart.
func getRetryRequestIds(params GroupedLogParams) []string {
	tx := model.LOG_DB.Table("logs").
		Select("DISTINCT request_id").
		Where("type IN (51, 52, 29, 59)").
		Where("request_id != ''")
	if params.StartTimestamp != 0 {
		tx = tx.Where("created_at >= ?", params.StartTimestamp)
	}
	if params.EndTimestamp != 0 {
		tx = tx.Where("created_at <= ?", params.EndTimestamp)
	}

	var retryReqIds []string
	tx.Pluck("request_id", &retryReqIds)
	return filterTraceRequestIdsByCommonFilters(params, retryReqIds)
}

func filterTraceRequestIdsByCommonFilters(params GroupedLogParams, reqIds []string) []string {
	if len(reqIds) == 0 {
		return reqIds
	}
	tx := model.LOG_DB.Table("logs").
		Select("DISTINCT request_id").
		Where("request_id IN ?", reqIds).
		Where("type IN (2, 5, 51, 52, 29, 59)")
	tx = applyCommonFilters(tx, params)

	var filtered []string
	tx.Pluck("request_id", &filtered)
	return filtered
}

func emptyTraceSummaryQuery() *gorm.DB {
	return model.LOG_DB.Table("logs").
		Select(emptyTraceSummarySelectSQL()).
		Where("1 = 0")
}

func emptyTraceSummarySelectSQL() string {
	return "0 AS id, '' AS request_id, 0 AS created_at, 0 AS max_created_at, 0 AS user_id, '' AS model_name, '' AS username, '' AS token_name, 0 AS token_id, 0 AS channel_id, 0 AS channel_count, 0 AS has_success, 0 AS total_quota, 0 AS total_prompt_tokens, 0 AS total_completion_tokens, 0 AS log_count, 0 AS has_error, '' AS group_val, '' AS ip, '' AS other, 0 AS has_stream, '' AS content"
}

func traceSummarySelectSQL() string {
	groupCol := logGroupCol()
	return `
		MAX(id) AS id,
		request_id,
		MIN(created_at) AS created_at,
		MAX(created_at) AS max_created_at,
		MAX(user_id) AS user_id,
		COALESCE(NULLIF(MAX(CASE WHEN type = 2 THEN model_name ELSE '' END), ''), MAX(model_name)) AS model_name,
		COALESCE(NULLIF(MAX(CASE WHEN type = 2 THEN username ELSE '' END), ''), MAX(username)) AS username,
		COALESCE(NULLIF(MAX(CASE WHEN type = 2 THEN token_name ELSE '' END), ''), MAX(token_name)) AS token_name,
		MAX(token_id) AS token_id,
		COALESCE(NULLIF(MAX(CASE WHEN type = 2 THEN channel_id ELSE 0 END), 0), MAX(channel_id)) AS channel_id,
		COUNT(DISTINCT channel_id) AS channel_count,
		MAX(CASE WHEN type = 2 THEN 1 ELSE 0 END) AS has_success,
		SUM(CASE WHEN type = 2 THEN quota ELSE 0 END) AS total_quota,
		SUM(CASE WHEN type = 2 THEN prompt_tokens ELSE 0 END) AS total_prompt_tokens,
		SUM(CASE WHEN type = 2 THEN completion_tokens ELSE 0 END) AS total_completion_tokens,
		COUNT(*) AS log_count,
		MAX(CASE WHEN type IN (5, 51, 52, 59) THEN 1 ELSE 0 END) AS has_error,
		COALESCE(NULLIF(MAX(CASE WHEN type = 2 THEN ` + groupCol + ` ELSE '' END), ''), MAX(` + groupCol + `)) AS group_val,
		COALESCE(NULLIF(MAX(CASE WHEN type = 2 THEN ip ELSE '' END), ''), MAX(ip)) AS ip,
		COALESCE(NULLIF(MAX(CASE WHEN type = 2 THEN other ELSE '' END), ''), NULLIF(MAX(CASE WHEN type = 52 THEN other ELSE '' END), ''), NULLIF(MAX(CASE WHEN type = 5 THEN other ELSE '' END), ''), NULLIF(MAX(CASE WHEN type = 51 THEN other ELSE '' END), ''), MAX(other)) AS other,
		MAX(CASE WHEN is_stream THEN 1 ELSE 0 END) AS has_stream,
		COALESCE(NULLIF(MAX(CASE WHEN type = 2 THEN content ELSE '' END), ''), NULLIF(MAX(CASE WHEN type = 52 THEN content ELSE '' END), ''), NULLIF(MAX(CASE WHEN type = 5 THEN content ELSE '' END), ''), NULLIF(MAX(CASE WHEN type = 51 THEN content ELSE '' END), ''), MAX(content)) AS content
	`
}

// traceSummaryQuery builds one summary row per request_id with full trace fields.
func traceSummaryQuery(retryReqIds []string) *gorm.DB {
	if len(retryReqIds) == 0 {
		// No retry request_ids → return empty result query
		return emptyTraceSummaryQuery()
	}

	tx := model.LOG_DB.Table("logs").
		Select(traceSummarySelectSQL()).
		Where("type IN (2, 5, 51, 52, 29, 59)").
		Where("request_id IN ?", retryReqIds)

	tx = tx.Group("request_id")

	return tx
}

// traceSummaryQueryForIds builds the GROUP BY aggregation query for specific request_ids.
func traceSummaryQueryForIds(reqIds []string) *gorm.DB {
	if len(reqIds) == 0 {
		return emptyTraceSummaryQuery()
	}
	tx := model.LOG_DB.Table("logs").
		Select(traceSummarySelectSQL()).
		Where("type IN (2, 5, 51, 52, 29, 59)").
		Where("request_id IN ?", reqIds)

	tx = tx.Group("request_id")

	return tx
}

// normalLogsQuery builds the query for normal log rows (type IN 2, 5) excluding retry request_ids.
func normalLogsQuery(params GroupedLogParams, retryReqIds []string) *gorm.DB {
	tx := model.LOG_DB.Model(&model.Log{}).
		Where("type IN (2, 5)")

	// Exclude request_ids that are covered by summary rows
	if len(retryReqIds) > 0 {
		tx = tx.Where("(request_id NOT IN ? OR request_id = '')", retryReqIds)
	}

	tx = applyCommonFilters(tx, params)

	return tx
}

// getChannelPaths batch queries channel steps for each request_id and builds channel path strings.
func getChannelPaths(requestIds []string) (map[string]string, error) {
	if len(requestIds) == 0 {
		return nil, nil
	}

	type channelStep struct {
		RequestId string `gorm:"column:request_id"`
		Id        int    `gorm:"column:id"`
		ChannelId int    `gorm:"column:channel_id"`
		CreatedAt int64  `gorm:"column:created_at"`
	}

	var steps []channelStep
	err := model.LOG_DB.Table("logs").
		Select("request_id, id, channel_id, created_at").
		Where("request_id IN ?", requestIds).
		Where("type IN (2, 51, 52)"). // Exclude probe records (29/59) per requirement 4.1
		Order("created_at ASC, id ASC").
		Find(&steps).Error
	if err != nil {
		return nil, err
	}

	// Group by request_id
	grouped := make(map[string][]int)
	for _, s := range steps {
		grouped[s.RequestId] = append(grouped[s.RequestId], s.ChannelId)
	}

	// Build channel path for each request_id
	result := make(map[string]string, len(requestIds))
	for reqId, channelIds := range grouped {
		result[reqId] = buildChannelPath(channelIds)
	}

	return result, nil
}

// mergeAndPaginate merges summary items and normal items, sorts by created_at DESC,
// and applies offset/limit pagination.
func mergeAndPaginate(summaries []groupedTraceSummaryRow, normals []*model.Log,
	channelPaths map[string]string, offset, limit int) []GroupedLogItem {

	// Convert summary rows to GroupedLogItem
	summaryItems := make([]GroupedLogItem, 0, len(summaries))
	for _, row := range summaries {
		summaryItems = append(summaryItems, summaryRowToGroupedItem(row, channelPaths))
	}

	// Convert normal rows to GroupedLogItem
	normalItems := make([]GroupedLogItem, 0, len(normals))
	for _, l := range normals {
		normalItems = append(normalItems, logToGroupedItem(l))
	}

	// Merge all items
	all := make([]GroupedLogItem, 0, len(summaryItems)+len(normalItems))
	all = append(all, summaryItems...)
	all = append(all, normalItems...)

	// Sort by newest activity DESC, while summary rows still display the trace start time.
	sort.Slice(all, func(i, j int) bool {
		return itemSortAt(all[i]) > itemSortAt(all[j])
	})

	// Apply pagination
	if offset >= len(all) {
		return []GroupedLogItem{}
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end]
}

func itemSortAt(item GroupedLogItem) int64 {
	if item.SortAt > 0 {
		return item.SortAt
	}
	return item.CreatedAt
}

// summaryRowToGroupedItem converts a groupedTraceSummaryRow to a GroupedLogItem.
func summaryRowToGroupedItem(row groupedTraceSummaryRow, channelPaths map[string]string) GroupedLogItem {
	// Determine type: 20 (success) if has_success, otherwise 50 (failed)
	summaryType := 50
	if row.HasSuccess == 1 {
		summaryType = 20
	}

	channelPath := ""
	if channelPaths != nil {
		channelPath = channelPaths[row.RequestId]
	}

	totalDuration := int(row.MaxCreatedAt - row.CreatedAt)
	if totalDuration < 0 {
		totalDuration = 0
	}

	return GroupedLogItem{
		Id:               row.Id,
		UserId:           row.UserId,
		Type:             summaryType,
		CreatedAt:        row.CreatedAt,
		ModelName:        row.ModelName,
		Username:         row.Username,
		TokenName:        row.TokenName,
		TokenId:          row.TokenId,
		Quota:            row.TotalQuota,
		PromptTokens:     row.TotalPromptTokens,
		CompletionTokens: row.TotalCompletionTokens,
		UseTime:          totalDuration,
		ChannelId:        row.ChannelId,
		ChannelName:      "",
		RequestId:        row.RequestId,
		Group:            row.Group,
		Ip:               row.Ip,
		Other:            row.Other,
		IsStream:         row.HasStream > 0,
		Content:          row.Content,
		SortAt:           row.MaxCreatedAt,
		ChannelPath:      channelPath,
		TotalDuration:    totalDuration,
		StepCount:        row.LogCount,
		IsSummary:        true,
	}
}

// logToGroupedItem converts a model.Log to a GroupedLogItem (normal row).
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
		Group:            l.Group,
		Ip:               l.Ip,
		Other:            l.Other,
		IsStream:         l.IsStream,
		Content:          l.Content,
		SortAt:           l.CreatedAt,
		IsSummary:        false,
	}
}

// resolveChannelNames batch resolves channel names for a slice of logs.
// Uses CacheGetChannel when memory cache is enabled, otherwise queries DB directly.
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
