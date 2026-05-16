package model

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"

	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

type Log struct {
	Id               int    `json:"id" gorm:"index:idx_created_at_id,priority:1;index:idx_user_id_id,priority:2"`
	UserId           int    `json:"user_id" gorm:"index;index:idx_user_id_id,priority:1"`
	CreatedAt        int64  `json:"created_at" gorm:"bigint;index:idx_created_at_id,priority:2;index:idx_created_at_type"`
	Type             int    `json:"type" gorm:"index:idx_created_at_type"`
	Content          string `json:"content"`
	Username         string `json:"username" gorm:"index;index:index_username_model_name,priority:2;default:''"`
	TokenName        string `json:"token_name" gorm:"index;default:''"`
	ModelName        string `json:"model_name" gorm:"index;index:index_username_model_name,priority:1;default:''"`
	Quota            int    `json:"quota" gorm:"default:0"`
	PromptTokens     int    `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens int    `json:"completion_tokens" gorm:"default:0"`
	UseTime          int    `json:"use_time" gorm:"default:0"`
	IsStream         bool   `json:"is_stream"`
	ChannelId        int    `json:"channel" gorm:"index"`
	ChannelName      string `json:"channel_name" gorm:"->"`
	TokenId          int    `json:"token_id" gorm:"default:0;index"`
	Group            string `json:"group" gorm:"index"`
	Ip               string `json:"ip" gorm:"index;default:''"`
	RequestId        string `json:"request_id,omitempty" gorm:"type:varchar(64);index:idx_logs_request_id;default:''"`
	TraceId          string `json:"trace_id,omitempty" gorm:"type:varchar(64);index:idx_logs_trace_id_seq,priority:1;default:''"`
	TraceSeq         int    `json:"trace_seq,omitempty" gorm:"index:idx_logs_trace_id_seq,priority:2;default:0"`
	TraceParentId    int    `json:"trace_parent_id,omitempty" gorm:"index;default:0"`
	TraceSiblingSeq  int    `json:"trace_sibling_seq,omitempty" gorm:"default:0"`
	TraceRole        string `json:"trace_role,omitempty" gorm:"type:varchar(32);index;default:''"`
	SummaryLogId     int    `json:"summary_log_id,omitempty" gorm:"index;default:0"`
	TerminalLogId    int    `json:"terminal_log_id,omitempty" gorm:"index;default:0"`
	TraceVersion     int    `json:"trace_version,omitempty" gorm:"default:0"`
	Other            string `json:"other"`
}

// don't use iota, avoid change log type value
const (
	LogTypeUnknown = 0
	LogTypeTopup   = 1
	LogTypeConsume = 2
	LogTypeManage  = 3
	LogTypeSystem  = 4
	LogTypeError   = 5 // Legacy: all errors (kept for historical data compatibility)
	LogTypeRefund  = 6

	// NACP: materialized SFT summary and step log types.
	LogTypeRetrySuccessSummary = 20 // Retry chain completed successfully; request-level summary row.
	LogTypeRetryConsume        = 21 // Terminal successful consume inside a retry chain.
	LogTypeRetryFailedSummary  = 50 // Retry chain exhausted/failed; request-level summary row.
	LogTypeErrorIntercepted    = 51 // Error intercepted by retry system (client did NOT see this error)
	LogTypeErrorClientVisible  = 52 // Error returned to client (all retries exhausted)

	LogTypeProbeSuccess = 29 // Probe request succeeded (lightweight health check)
	LogTypeProbeFailed  = 59 // Probe request failed (timeout or non-2xx)
)

const (
	TraceRoleSummarySuccess   = "summary_success"
	TraceRoleSummaryFailed    = "summary_failed"
	TraceRoleConsume          = "consume"
	TraceRoleErrorLegacy      = "error_legacy"
	TraceRoleErrorIntercepted = "error_intercepted"
	TraceRoleErrorVisible     = "error_visible"
	TraceRoleProbeSuccess     = "probe_success"
	TraceRoleProbeFailed      = "probe_failed"
	TraceRoleOther            = "other"
)

var (
	traceSeqMu    sync.Mutex
	traceSeqCache = map[string]int{}
)

func traceRoleForLogType(logType int) string {
	switch logType {
	case LogTypeConsume:
		return TraceRoleConsume
	case LogTypeRetrySuccessSummary:
		return TraceRoleSummarySuccess
	case LogTypeRetryConsume:
		return TraceRoleConsume
	case LogTypeRetryFailedSummary:
		return TraceRoleSummaryFailed
	case LogTypeError:
		return TraceRoleErrorLegacy
	case LogTypeErrorIntercepted:
		return TraceRoleErrorIntercepted
	case LogTypeErrorClientVisible:
		return TraceRoleErrorVisible
	case LogTypeProbeSuccess:
		return TraceRoleProbeSuccess
	case LogTypeProbeFailed:
		return TraceRoleProbeFailed
	default:
		return TraceRoleOther
	}
}

func normalizeLogTypeForStorage(logType int) int {
	return logType
}

func isTraceSummaryLogType(logType int) bool {
	return logType == LogTypeRetrySuccessSummary || logType == LogTypeRetryFailedSummary
}

func isTraceTerminalLog(log *Log) bool {
	if log == nil {
		return false
	}
	switch log.Type {
	case LogTypeRetryConsume, LogTypeErrorClientVisible:
		return true
	case LogTypeConsume:
		return log.TraceSeq > 1 || log.TraceRole == TraceRoleConsume
	case LogTypeError:
		return log.TraceRole == TraceRoleErrorVisible
	default:
		return false
	}
}

func isTraceChildLogType(logType int) bool {
	switch logType {
	case LogTypeRetryConsume, LogTypeProbeSuccess, LogTypeProbeFailed, LogTypeErrorIntercepted, LogTypeErrorClientVisible:
		return true
	default:
		return false
	}
}

func nextTraceSeq(traceId string) int {
	return nextTraceSeqWithDB(LOG_DB, traceId)
}

func nextTraceSeqWithDB(db *gorm.DB, traceId string) int {
	if traceId == "" {
		return 0
	}

	traceSeqMu.Lock()
	defer traceSeqMu.Unlock()

	current := traceSeqCache[traceId]
	if current == 0 && db != nil {
		var maxSeq int
		if err := db.Model(&Log{}).
			Where("trace_id = ? OR request_id = ?", traceId, traceId).
			Select("COALESCE(MAX(trace_seq), 0)").
			Scan(&maxSeq).Error; err == nil && maxSeq > current {
			current = maxSeq
		}
	}
	current++
	traceSeqCache[traceId] = current
	return current
}

func applyLogTraceFields(log *Log, db *gorm.DB) {
	if log == nil {
		return
	}
	if log.TraceId == "" {
		log.TraceId = log.RequestId
	}
	if log.TraceRole == "" {
		log.TraceRole = traceRoleForLogType(log.Type)
	}
	log.Type = normalizeLogTypeForStorage(log.Type)
	if log.TraceVersion == 0 && (isTraceChildLogType(log.Type) || isTraceSummaryLogType(log.Type)) {
		log.TraceVersion = 1
	}
	if isTraceSummaryLogType(log.Type) {
		return
	}
	if log.TraceSeq == 0 {
		log.TraceSeq = nextTraceSeqWithDB(db, log.TraceId)
	}
}

// ApplyLogTraceFields fills structural trace fields for a flat log row.
// Logs remain independent DB rows; these fields let readers reconstruct
// trace membership, logical order, role, and future parent-child links.
func ApplyLogTraceFields(log *Log) {
	applyLogTraceFields(log, LOG_DB)
}

func (log *Log) BeforeCreate(tx *gorm.DB) error {
	applyLogTraceFields(log, tx)
	return nil
}

func hasExistingSFTTraceSteps(requestId string) bool {
	if requestId == "" || LOG_DB == nil {
		return false
	}
	var count int64
	err := LOG_DB.Model(&Log{}).
		Where("(trace_id = ? OR request_id = ?) AND type IN ?", requestId, requestId, []int{
			LogTypeErrorIntercepted,
			LogTypeProbeSuccess,
			LogTypeProbeFailed,
			LogTypeErrorClientVisible,
			LogTypeRetryConsume,
			LogTypeRetrySuccessSummary,
			LogTypeRetryFailedSummary,
		}).
		Count(&count).Error
	return err == nil && count > 0
}

func traceTerminalRank(log Log) int {
	switch log.Type {
	case LogTypeErrorClientVisible:
		return 2
	case LogTypeRetryConsume:
		return 1
	case LogTypeError:
		if log.TraceRole == TraceRoleErrorVisible {
			return 2
		}
	case LogTypeConsume:
		if log.TraceSeq > 1 {
			return 1
		}
	}
	return 0
}

func betterTraceTerminal(candidate Log, current *Log) bool {
	if traceTerminalRank(candidate) == 0 {
		return false
	}
	if current == nil {
		return true
	}
	if candidate.TraceSeq != current.TraceSeq {
		return candidate.TraceSeq > current.TraceSeq
	}
	if candidate.CreatedAt != current.CreatedAt {
		return candidate.CreatedAt > current.CreatedAt
	}
	return candidate.Id > current.Id
}

func traceSummaryTypeForTerminal(terminal Log) int {
	if terminal.Type == LogTypeRetryConsume || terminal.Type == LogTypeConsume || terminal.TraceRole == TraceRoleConsume {
		return LogTypeRetrySuccessSummary
	}
	return LogTypeRetryFailedSummary
}

func traceSummaryRoleForType(logType int) string {
	if logType == LogTypeRetrySuccessSummary {
		return TraceRoleSummarySuccess
	}
	return TraceRoleSummaryFailed
}

func traceLogDisplayCost(log Log, summaryType int) (quota int, promptTokens int, completionTokens int) {
	if summaryType == LogTypeRetrySuccessSummary {
		return log.Quota, log.PromptTokens, log.CompletionTokens
	}
	return 0, 0, 0
}

func isFormalTraceRequestStep(log Log) bool {
	switch log.Type {
	case LogTypeConsume, LogTypeRetryConsume, LogTypeErrorIntercepted, LogTypeErrorClientVisible, LogTypeError:
		return true
	default:
		return false
	}
}

func buildTraceSummaryOther(steps []Log, terminal Log) string {
	otherMap, _ := common.StrToMap(terminal.Other)
	if otherMap == nil {
		otherMap = map[string]interface{}{}
	}

	channelPath := make([]int, 0, len(steps))
	lastFormalChannelId := 0
	var startedAt int64
	endedAt := terminal.CreatedAt
	platformProbeQuota := 0
	probeSuccessCount := 0
	probeFailedCount := 0
	interceptedCount := 0

	for _, step := range steps {
		if step.Id == 0 || isTraceSummaryLogType(step.Type) {
			continue
		}
		if startedAt == 0 || (step.CreatedAt != 0 && step.CreatedAt < startedAt) {
			startedAt = step.CreatedAt
		}
		if step.CreatedAt > endedAt {
			endedAt = step.CreatedAt
		}
		if isFormalTraceRequestStep(step) && step.ChannelId != 0 {
			if step.ChannelId != lastFormalChannelId {
				channelPath = append(channelPath, step.ChannelId)
				lastFormalChannelId = step.ChannelId
			}
		}
		switch step.Type {
		case LogTypeProbeSuccess:
			probeSuccessCount++
			platformProbeQuota += step.Quota
		case LogTypeProbeFailed:
			probeFailedCount++
		case LogTypeErrorIntercepted:
			interceptedCount++
		}
	}
	if startedAt == 0 {
		startedAt = terminal.CreatedAt
	}

	otherMap["summary"] = map[string]interface{}{
		"terminal_log_id":       terminal.Id,
		"terminal_trace_role":   terminal.TraceRole,
		"terminal_type":         terminal.Type,
		"step_count":            len(steps),
		"channel_path":          channelPath,
		"started_at":            startedAt,
		"ended_at":              endedAt,
		"duration_seconds":      endedAt - startedAt,
		"user_quota":            terminal.Quota,
		"prompt_tokens":         terminal.PromptTokens,
		"completion_tokens":     terminal.CompletionTokens,
		"platform_probe_quota":  platformProbeQuota,
		"probe_success_count":   probeSuccessCount,
		"probe_failed_count":    probeFailedCount,
		"intercepted_log_count": interceptedCount,
	}
	return common.MapToJsonStr(otherMap)
}

func UpsertTraceSummary(traceId string) (*Log, error) {
	if traceId == "" || LOG_DB == nil {
		return nil, nil
	}

	var logs []Log
	if err := LOG_DB.
		Where("(trace_id = ? OR request_id = ?) AND type NOT IN ?", traceId, traceId, []int{LogTypeRetrySuccessSummary, LogTypeRetryFailedSummary}).
		Order("CASE WHEN trace_seq > 0 THEN trace_seq ELSE 2147483647 END ASC, trace_sibling_seq ASC, id ASC").
		Find(&logs).Error; err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return nil, nil
	}

	var terminal *Log
	for i := range logs {
		if betterTraceTerminal(logs[i], terminal) {
			terminal = &logs[i]
		}
	}
	if terminal == nil {
		return nil, nil
	}

	hasChain := false
	for _, step := range logs {
		if step.Id == terminal.Id {
			continue
		}
		if isTraceChildLogType(step.Type) || step.TraceSeq > 1 {
			hasChain = true
			break
		}
	}
	if !hasChain && terminal.Type == LogTypeConsume {
		return nil, nil
	}

	summaryType := traceSummaryTypeForTerminal(*terminal)
	quota, promptTokens, completionTokens := traceLogDisplayCost(*terminal, summaryType)
	summaryOther := buildTraceSummaryOther(logs, *terminal)

	var summary Log
	err := LOG_DB.Where("(trace_id = ? OR request_id = ?) AND type IN ?", traceId, traceId, []int{LogTypeRetrySuccessSummary, LogTypeRetryFailedSummary}).
		Order("id DESC").
		First(&summary).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		summary = Log{
			UserId:           terminal.UserId,
			Username:         terminal.Username,
			CreatedAt:        common.GetTimestamp(),
			Type:             summaryType,
			Content:          terminal.Content,
			TokenName:        terminal.TokenName,
			ModelName:        terminal.ModelName,
			Quota:            quota,
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			UseTime:          int(common.GetTimestamp() - logs[0].CreatedAt),
			IsStream:         terminal.IsStream,
			ChannelId:        terminal.ChannelId,
			TokenId:          terminal.TokenId,
			Group:            terminal.Group,
			Ip:               terminal.Ip,
			RequestId:        terminal.RequestId,
			TraceId:          terminal.TraceId,
			TraceRole:        traceSummaryRoleForType(summaryType),
			TerminalLogId:    terminal.Id,
			TraceVersion:     1,
			Other:            summaryOther,
		}
		if summary.TraceId == "" {
			summary.TraceId = traceId
		}
		if summary.RequestId == "" {
			summary.RequestId = traceId
		}
		if err := LOG_DB.Create(&summary).Error; err != nil {
			return nil, err
		}
		if err := LOG_DB.Model(&Log{}).Where("id = ?", summary.Id).Updates(map[string]interface{}{
			"summary_log_id": summary.Id,
		}).Error; err != nil {
			return nil, err
		}
		summary.SummaryLogId = summary.Id
	} else if err != nil {
		return nil, err
	} else {
		updates := map[string]interface{}{
			"type":              summaryType,
			"trace_role":        traceSummaryRoleForType(summaryType),
			"terminal_log_id":   terminal.Id,
			"summary_log_id":    summary.Id,
			"trace_version":     1,
			"user_id":           terminal.UserId,
			"username":          terminal.Username,
			"token_name":        terminal.TokenName,
			"model_name":        terminal.ModelName,
			"quota":             quota,
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
			"is_stream":         terminal.IsStream,
			"channel_id":        terminal.ChannelId,
			"token_id":          terminal.TokenId,
			"group":             terminal.Group,
			"ip":                terminal.Ip,
			"content":           terminal.Content,
			"other":             summaryOther,
		}
		if err := LOG_DB.Model(&summary).Updates(updates).Error; err != nil {
			return nil, err
		}
		summary.Type = summaryType
		summary.TraceRole = traceSummaryRoleForType(summaryType)
		summary.TerminalLogId = terminal.Id
		summary.SummaryLogId = summary.Id
	}

	if err := LOG_DB.Model(&Log{}).
		Where("(trace_id = ? OR request_id = ?) AND type NOT IN ?", traceId, traceId, []int{LogTypeRetrySuccessSummary, LogTypeRetryFailedSummary}).
		Updates(map[string]interface{}{
			"summary_log_id": summary.Id,
			"trace_version":  1,
		}).Error; err != nil {
		return &summary, err
	}

	return &summary, nil
}

func FinalizeTraceAfterLogCreated(log *Log) {
	if log == nil || log.RequestId == "" || isTraceSummaryLogType(log.Type) {
		return
	}
	traceId := log.TraceId
	if traceId == "" {
		traceId = log.RequestId
	}
	if isTraceTerminalLog(log) {
		if _, err := UpsertTraceSummary(traceId); err != nil {
			common.SysLog("failed to upsert trace summary: " + err.Error())
		}
		return
	}

	var summary Log
	err := LOG_DB.Where("(trace_id = ? OR request_id = ?) AND type IN ?", traceId, traceId, []int{LogTypeRetrySuccessSummary, LogTypeRetryFailedSummary}).
		Order("id DESC").
		First(&summary).Error
	if err == nil && summary.Id > 0 {
		_ = LOG_DB.Model(log).Updates(map[string]interface{}{
			"summary_log_id": summary.Id,
			"trace_version":  1,
		}).Error
		if _, upsertErr := UpsertTraceSummary(traceId); upsertErr != nil {
			common.SysLog("failed to refresh trace summary: " + upsertErr.Error())
		}
	}
}

func formatUserLogs(logs []*Log, startIdx int) {
	for i := range logs {
		logs[i].ChannelName = ""
		var otherMap map[string]interface{}
		otherMap, _ = common.StrToMap(logs[i].Other)
		if otherMap != nil {
			// Remove admin-only debug fields.
			delete(otherMap, "admin_info")
			// delete(otherMap, "reject_reason")
			delete(otherMap, "stream_status")
		}
		logs[i].Other = common.MapToJsonStr(otherMap)
		logs[i].Id = startIdx + i + 1
	}
}

func GetLogByTokenId(tokenId int) (logs []*Log, err error) {
	err = LOG_DB.Model(&Log{}).Where("token_id = ?", tokenId).Order("id desc").Limit(common.MaxRecentItems).Find(&logs).Error
	formatUserLogs(logs, 0)
	return logs, err
}

func RecordLog(userId int, logType int, content string) {
	if logType == LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(userId, false)
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      logType,
		Content:   content,
	}
	ApplyLogTraceFields(log)
	err := LOG_DB.Create(log).Error
	if err != nil {
		common.SysLog("failed to record log: " + err.Error())
	}
}

// RecordLogWithAdminInfo 记录操作日志，并将管理员相关信息存入 Other.admin_info，
func RecordLogWithAdminInfo(userId int, logType int, content string, adminInfo map[string]interface{}) {
	if logType == LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(userId, false)
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      logType,
		Content:   content,
	}
	if len(adminInfo) > 0 {
		other := map[string]interface{}{
			"admin_info": adminInfo,
		}
		log.Other = common.MapToJsonStr(other)
	}
	ApplyLogTraceFields(log)
	if err := LOG_DB.Create(log).Error; err != nil {
		common.SysLog("failed to record log: " + err.Error())
	}
}

func RecordTopupLog(userId int, content string, callerIp string, paymentMethod string, callbackPaymentMethod string) {
	username, _ := GetUsernameById(userId, false)
	adminInfo := map[string]interface{}{
		"server_ip":               common.GetIp(),
		"node_name":               common.NodeName,
		"caller_ip":               callerIp,
		"payment_method":          paymentMethod,
		"callback_payment_method": callbackPaymentMethod,
		"version":                 common.Version,
	}
	other := map[string]interface{}{
		"admin_info": adminInfo,
	}
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      LogTypeTopup,
		Content:   content,
		Ip:        callerIp,
		Other:     common.MapToJsonStr(other),
	}
	ApplyLogTraceFields(log)
	err := LOG_DB.Create(log).Error
	if err != nil {
		common.SysLog("failed to record topup log: " + err.Error())
	}
}

func RecordErrorLog(c *gin.Context, userId int, channelId int, modelName string, tokenName string, content string, tokenId int, useTimeSeconds int,
	isStream bool, group string, other map[string]interface{}) {
	RecordErrorLogWithType(c, LogTypeErrorClientVisible, userId, channelId, modelName, tokenName, content, tokenId, useTimeSeconds, isStream, group, other)
}

// NACP: RecordErrorLogWithType allows specifying the log type (intercepted vs client-visible)
func RecordErrorLogWithType(c *gin.Context, logType int, userId int, channelId int, modelName string, tokenName string, content string, tokenId int, useTimeSeconds int,
	isStream bool, group string, other map[string]interface{}) {
	logger.LogInfo(c, fmt.Sprintf("record error log: userId=%d, channelId=%d, modelName=%s, tokenName=%s, content=%s", userId, channelId, modelName, tokenName, content))
	username := c.GetString("username")
	requestId := c.GetString(common.RequestIdKey)
	otherStr := common.MapToJsonStr(other)
	// 判断是否需要记录 IP
	needRecordIp := false
	if settingMap, err := GetUserSetting(userId, false); err == nil {
		if settingMap.RecordIpLog {
			needRecordIp = true
		}
	}
	log := &Log{
		UserId:           userId,
		Username:         username,
		CreatedAt:        common.GetTimestamp(),
		Type:             logType,
		Content:          content,
		PromptTokens:     0,
		CompletionTokens: 0,
		TokenName:        tokenName,
		ModelName:        modelName,
		Quota:            0,
		ChannelId:        channelId,
		TokenId:          tokenId,
		UseTime:          useTimeSeconds,
		IsStream:         isStream,
		Group:            group,
		Ip: func() string {
			if needRecordIp {
				return c.ClientIP()
			}
			return ""
		}(),
		RequestId: requestId,
		Other:     otherStr,
	}
	ApplyLogTraceFields(log)
	err := LOG_DB.Create(log).Error
	if err != nil {
		logger.LogError(c, "failed to record log: "+err.Error())
		return
	}
	FinalizeTraceAfterLogCreated(log)
}

type RecordConsumeLogParams struct {
	ChannelId        int                    `json:"channel_id"`
	PromptTokens     int                    `json:"prompt_tokens"`
	CompletionTokens int                    `json:"completion_tokens"`
	ModelName        string                 `json:"model_name"`
	TokenName        string                 `json:"token_name"`
	Quota            int                    `json:"quota"`
	Content          string                 `json:"content"`
	TokenId          int                    `json:"token_id"`
	UseTimeSeconds   int                    `json:"use_time_seconds"`
	IsStream         bool                   `json:"is_stream"`
	Group            string                 `json:"group"`
	Other            map[string]interface{} `json:"other"`
}

func RecordConsumeLog(c *gin.Context, userId int, params RecordConsumeLogParams) {
	if !common.LogConsumeEnabled {
		return
	}
	logger.LogInfo(c, fmt.Sprintf("record consume log: userId=%d, params=%s", userId, common.GetJsonString(params)))
	username := c.GetString("username")
	requestId := c.GetString(common.RequestIdKey)
	otherStr := common.MapToJsonStr(params.Other)
	// 判断是否需要记录 IP
	needRecordIp := false
	if settingMap, err := GetUserSetting(userId, false); err == nil {
		if settingMap.RecordIpLog {
			needRecordIp = true
		}
	}
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type: func() int {
			if hasExistingSFTTraceSteps(requestId) {
				return LogTypeRetryConsume
			}
			return LogTypeConsume
		}(),
		Content:          params.Content,
		PromptTokens:     params.PromptTokens,
		CompletionTokens: params.CompletionTokens,
		TokenName:        params.TokenName,
		ModelName:        params.ModelName,
		Quota:            params.Quota,
		ChannelId:        params.ChannelId,
		TokenId:          params.TokenId,
		UseTime:          params.UseTimeSeconds,
		IsStream:         params.IsStream,
		Group:            params.Group,
		Ip: func() string {
			if needRecordIp {
				return c.ClientIP()
			}
			return ""
		}(),
		RequestId: requestId,
		Other:     otherStr,
	}
	ApplyLogTraceFields(log)
	err := LOG_DB.Create(log).Error
	if err != nil {
		logger.LogError(c, "failed to record log: "+err.Error())
		return
	}
	FinalizeTraceAfterLogCreated(log)
	if common.DataExportEnabled {
		gopool.Go(func() {
			LogQuotaData(userId, username, params.ModelName, params.Quota, common.GetTimestamp(), params.PromptTokens+params.CompletionTokens)
		})
	}
}

type RecordTaskBillingLogParams struct {
	UserId    int
	LogType   int
	Content   string
	ChannelId int
	ModelName string
	Quota     int
	TokenId   int
	Group     string
	Other     map[string]interface{}
}

func RecordTaskBillingLog(params RecordTaskBillingLogParams) {
	if params.LogType == LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(params.UserId, false)
	tokenName := ""
	if params.TokenId > 0 {
		if token, err := GetTokenById(params.TokenId); err == nil {
			tokenName = token.Name
		}
	}
	log := &Log{
		UserId:    params.UserId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      params.LogType,
		Content:   params.Content,
		TokenName: tokenName,
		ModelName: params.ModelName,
		Quota:     params.Quota,
		ChannelId: params.ChannelId,
		TokenId:   params.TokenId,
		Group:     params.Group,
		Other:     common.MapToJsonStr(params.Other),
	}
	ApplyLogTraceFields(log)
	err := LOG_DB.Create(log).Error
	if err != nil {
		common.SysLog("failed to record task billing log: " + err.Error())
	}
}

func GetAllLogs(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, startIdx int, num int, channel int, group string, requestId string) (logs []*Log, total int64, err error) {
	var tx *gorm.DB
	if logType == LogTypeUnknown {
		tx = LOG_DB
	} else {
		tx = LOG_DB.Where("logs.type = ?", logType)
	}

	if modelName != "" {
		tx = tx.Where("logs.model_name like ?", modelName)
	}
	if username != "" {
		tx = tx.Where("logs.username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("logs.token_name = ?", tokenName)
	}
	if requestId != "" {
		tx = tx.Where("logs.request_id = ?", requestId)
	}
	if logType == LogTypeUnknown {
		tx = tx.Where("NOT (logs.summary_log_id > 0 AND logs.type NOT IN ?)", []int{LogTypeRetrySuccessSummary, LogTypeRetryFailedSummary})
	}
	if startTimestamp != 0 {
		tx = tx.Where("logs.created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("logs.created_at <= ?", endTimestamp)
	}
	if channel != 0 {
		tx = tx.Where("logs.channel_id = ?", channel)
	}
	if group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", group)
	}
	err = tx.Model(&Log{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = tx.Order("logs.id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	channelIds := types.NewSet[int]()
	for _, log := range logs {
		if log.ChannelId != 0 {
			channelIds.Add(log.ChannelId)
		}
	}

	if channelIds.Len() > 0 {
		var channels []struct {
			Id   int    `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		if common.MemoryCacheEnabled {
			// Cache get channel
			for _, channelId := range channelIds.Items() {
				if cacheChannel, err := CacheGetChannel(channelId); err == nil {
					channels = append(channels, struct {
						Id   int    `gorm:"column:id"`
						Name string `gorm:"column:name"`
					}{
						Id:   channelId,
						Name: cacheChannel.Name,
					})
				}
			}
		} else {
			// Bulk query channels from DB
			if err = DB.Table("channels").Select("id, name").Where("id IN ?", channelIds.Items()).Find(&channels).Error; err != nil {
				return logs, total, err
			}
		}
		channelMap := make(map[int]string, len(channels))
		for _, channel := range channels {
			channelMap[channel.Id] = channel.Name
		}
		for i := range logs {
			logs[i].ChannelName = channelMap[logs[i].ChannelId]
		}
	}

	return logs, total, err
}

const logSearchCountLimit = 10000

func GetUserLogs(userId int, logType int, startTimestamp int64, endTimestamp int64, modelName string, tokenName string, startIdx int, num int, group string, requestId string) (logs []*Log, total int64, err error) {
	var tx *gorm.DB
	if logType == LogTypeUnknown {
		tx = LOG_DB.Where("logs.user_id = ?", userId)
	} else {
		tx = LOG_DB.Where("logs.user_id = ? and logs.type = ?", userId, logType)
	}

	if modelName != "" {
		modelNamePattern, err := sanitizeLikePattern(modelName)
		if err != nil {
			return nil, 0, err
		}
		tx = tx.Where("logs.model_name LIKE ? ESCAPE '!'", modelNamePattern)
	}
	if tokenName != "" {
		tx = tx.Where("logs.token_name = ?", tokenName)
	}
	if requestId != "" {
		tx = tx.Where("logs.request_id = ?", requestId)
	}
	if logType == LogTypeUnknown {
		tx = tx.Where("NOT (logs.summary_log_id > 0 AND logs.type NOT IN ?)", []int{LogTypeRetrySuccessSummary, LogTypeRetryFailedSummary})
	}
	if startTimestamp != 0 {
		tx = tx.Where("logs.created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("logs.created_at <= ?", endTimestamp)
	}
	if group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", group)
	}
	err = tx.Model(&Log{}).Limit(logSearchCountLimit).Count(&total).Error
	if err != nil {
		common.SysError("failed to count user logs: " + err.Error())
		return nil, 0, errors.New("查询日志失败")
	}
	err = tx.Order("logs.id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	if err != nil {
		common.SysError("failed to search user logs: " + err.Error())
		return nil, 0, errors.New("查询日志失败")
	}

	formatUserLogs(logs, startIdx)
	return logs, total, err
}

type Stat struct {
	Quota int `json:"quota"`
	Rpm   int `json:"rpm"`
	Tpm   int `json:"tpm"`
}

func SumUsedQuota(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, channel int, group string) (stat Stat, err error) {
	tx := LOG_DB.Table("logs").Select("sum(quota) quota")

	// 为rpm和tpm创建单独的查询
	rpmTpmQuery := LOG_DB.Table("logs").Select("count(*) rpm, sum(prompt_tokens) + sum(completion_tokens) tpm")

	if username != "" {
		tx = tx.Where("username = ?", username)
		rpmTpmQuery = rpmTpmQuery.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
		rpmTpmQuery = rpmTpmQuery.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		modelNamePattern, err := sanitizeLikePattern(modelName)
		if err != nil {
			return stat, err
		}
		tx = tx.Where("model_name LIKE ? ESCAPE '!'", modelNamePattern)
		rpmTpmQuery = rpmTpmQuery.Where("model_name LIKE ? ESCAPE '!'", modelNamePattern)
	}
	if channel != 0 {
		tx = tx.Where("channel_id = ?", channel)
		rpmTpmQuery = rpmTpmQuery.Where("channel_id = ?", channel)
	}
	if group != "" {
		tx = tx.Where(logGroupCol+" = ?", group)
		rpmTpmQuery = rpmTpmQuery.Where(logGroupCol+" = ?", group)
	}

	if logType != LogTypeUnknown {
		tx = tx.Where("type = ?", logType)
		rpmTpmQuery = rpmTpmQuery.Where("type = ?", logType)
	} else {
		tx = tx.Where("type IN ?", []int{LogTypeConsume, LogTypeRetrySuccessSummary})
		rpmTpmQuery = rpmTpmQuery.Where("type IN ?", []int{LogTypeConsume, LogTypeRetrySuccessSummary})
	}

	// 只统计最近60秒的rpm和tpm
	rpmTpmQuery = rpmTpmQuery.Where("created_at >= ?", time.Now().Add(-60*time.Second).Unix())

	// 执行查询
	if err := tx.Scan(&stat).Error; err != nil {
		common.SysError("failed to query log stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}
	if err := rpmTpmQuery.Scan(&stat).Error; err != nil {
		common.SysError("failed to query rpm/tpm stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}

	return stat, nil
}

func SumUsedToken(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string) (token int) {
	tx := LOG_DB.Table("logs").Select("ifnull(sum(prompt_tokens),0) + ifnull(sum(completion_tokens),0)")
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	tx.Where("type IN ?", []int{LogTypeConsume, LogTypeRetrySuccessSummary}).Scan(&token)
	return token
}

func DeleteOldLog(ctx context.Context, targetTimestamp int64, limit int) (int64, error) {
	var total int64 = 0

	for {
		if nil != ctx.Err() {
			return total, ctx.Err()
		}

		result := LOG_DB.Where("created_at < ?", targetTimestamp).Limit(limit).Delete(&Log{})
		if nil != result.Error {
			return total, result.Error
		}

		total += result.RowsAffected

		if result.RowsAffected < int64(limit) {
			break
		}
	}

	return total, nil
}
