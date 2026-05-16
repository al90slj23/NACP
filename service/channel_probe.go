/*
【文件职责】NACP 渠道轻量探测 — 并行预热顺位渠道
【核心架构】
  - 独立 HTTP client，不共享 relay 连接池
  - 最小请求体（max_tokens=1），3 秒超时
  - 并行探测多个顺位渠道，建立就绪队列
  - 探测成本独立记录（user_id=0，不计入用户账单）

【主要函数】
  - ProbeChannel: 同步探测单个渠道
  - ProbeNextChannels: 并行探测多个顺位渠道
  - StartDegradedProbeLoop: 后台定时探测降级渠道

【依赖关系】
  - common: JSON 封装、日志
  - model: Channel 数据模型
*/
package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/shopspring/decimal"
)

// ─── Types ────────────────────────────────────────────────────────────────────

// ProbeResult holds the outcome of a single probe request.
type ProbeResult struct {
	ChannelID           int
	ModelName           string
	Success             bool
	StatusCode          int
	LatencyMs           int64
	Error               error
	Timestamp           time.Time
	PromptTokens        int
	CompletionTokens    int
	CacheReadTokens     int
	CacheCreationTokens int
	EstimatedQuota      int
}

// ProbeLog represents a probe execution record for cost tracking.
type ProbeLog struct {
	ChannelID           int    `json:"channel_id"`
	ChannelName         string `json:"channel_name"`
	ModelName           string `json:"model_name"`
	Success             bool   `json:"success"`
	LatencyMs           int64  `json:"latency_ms"`
	StatusCode          int    `json:"status_code"`
	Trigger             string `json:"trigger"` // "pre_warm", "degraded_probe"
	Error               string `json:"error"`   // error message for failed probes
	Timestamp           int64  `json:"timestamp"`
	PromptTokens        int    `json:"prompt_tokens"`
	CompletionTokens    int    `json:"completion_tokens"`
	CacheReadTokens     int    `json:"cache_read_tokens"`
	CacheCreationTokens int    `json:"cache_creation_tokens"`
	EstimatedQuota      int    `json:"estimated_quota"`
}

type probeUsage struct {
	PromptTokens        int
	CompletionTokens    int
	CacheReadTokens     int
	CacheCreationTokens int
}

func (u probeUsage) totalTokens() int {
	return u.PromptTokens + u.CompletionTokens + u.CacheReadTokens + u.CacheCreationTokens
}

func (u probeUsage) hasUsage() bool {
	return u.totalTokens() > 0
}

// ─── HTTP Client ──────────────────────────────────────────────────────────────

var probeHTTPClient *http.Client

func initProbeHTTPClient() {
	cfg := GetHealthConfig()
	probeHTTPClient = &http.Client{
		Timeout: cfg.ProbeTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        20,
			MaxIdleConnsPerHost: 5,
			IdleConnTimeout:     30 * time.Second,
			DisableKeepAlives:   false,
		},
	}
	common.SysLog("NACP: probe HTTP client initialized")
}

// ─── Public API ───────────────────────────────────────────────────────────────

// ProbeChannel sends a minimal request to verify channel connectivity.
// Bypasses the full relay pipeline — direct HTTP call.
// Returns ProbeResult with success/failure, latency, and status code.
func ProbeChannel(channel *model.Channel, modelName string) *ProbeResult {
	if channel == nil {
		return &ProbeResult{Success: false, Error: fmt.Errorf("channel is nil")}
	}

	start := time.Now()
	result := &ProbeResult{
		ChannelID: channel.Id,
		ModelName: modelName,
		Timestamp: start,
	}

	// Build request
	body, err := buildProbeRequestBody(modelName)
	if err != nil {
		result.Error = err
		result.LatencyMs = time.Since(start).Milliseconds()
		return result
	}

	endpoint := getProbeEndpoint(channel)
	ctx, cancel := context.WithTimeout(context.Background(), GetHealthConfig().ProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(body)))
	if err != nil {
		result.Error = err
		result.LatencyMs = time.Since(start).Milliseconds()
		return result
	}

	req.Header.Set("Content-Type", "application/json")
	authHeader := getProbeAuthHeader(channel)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := probeHTTPClient.Do(req)
	result.LatencyMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Error = err
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
	if result.Success {
		bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if readErr == nil {
			usage := parseProbeUsage(bodyBytes)
			result.PromptTokens = usage.PromptTokens
			result.CompletionTokens = usage.CompletionTokens
			result.CacheReadTokens = usage.CacheReadTokens
			result.CacheCreationTokens = usage.CacheCreationTokens
			result.EstimatedQuota = estimateProbeQuota(modelName, usage)
		}
	}

	return result
}

// ProbeNextChannels probes multiple standby channels in parallel.
// Returns results in the same order as input channels.
// Used for pre-warming: while retrying channel A, simultaneously check B and C.
// requestId links probe logs to the triggering user request for trace grouping.
func ProbeNextChannels(channels []*model.Channel, modelName string, requestId string) []*ProbeResult {
	if len(channels) == 0 {
		return nil
	}

	results := make([]*ProbeResult, len(channels))
	var wg sync.WaitGroup

	for i, ch := range channels {
		wg.Add(1)
		idx := i
		channel := ch
		gopool.Go(func() {
			defer wg.Done()
			results[idx] = ProbeChannel(channel, modelName)
			// Record probe log for cost tracking
			var errMsg string
			if results[idx].Error != nil {
				errMsg = results[idx].Error.Error()
			}
			recordProbeLog(&ProbeLog{
				ChannelID:           channel.Id,
				ChannelName:         channel.Name,
				ModelName:           modelName,
				Success:             results[idx].Success,
				LatencyMs:           results[idx].LatencyMs,
				StatusCode:          results[idx].StatusCode,
				Trigger:             "pre_warm",
				Error:               errMsg,
				Timestamp:           time.Now().Unix(),
				PromptTokens:        results[idx].PromptTokens,
				CompletionTokens:    results[idx].CompletionTokens,
				CacheReadTokens:     results[idx].CacheReadTokens,
				CacheCreationTokens: results[idx].CacheCreationTokens,
				EstimatedQuota:      results[idx].EstimatedQuota,
			}, requestId)
		})
	}

	wg.Wait()
	return results
}

// StartDegradedProbeLoop starts a background goroutine that periodically
// probes all Degraded channels. Only runs on master node.
func StartDegradedProbeLoop() {
	if !common.IsMasterNode {
		return
	}

	gopool.Go(func() {
		cfg := GetHealthConfig()
		ticker := time.NewTicker(cfg.DegradedProbeInterval)
		defer ticker.Stop()

		for range ticker.C {
			probeDegradedChannels()
		}
	})
	common.SysLog("NACP: degraded probe loop started")
}

// ─── Internal Functions ───────────────────────────────────────────────────────

// buildProbeRequestBody constructs the minimal JSON body for probing.
func buildProbeRequestBody(modelName string) ([]byte, error) {
	body := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
		"max_tokens": 1,
		"stream":     false,
	}
	return common.Marshal(body)
}

func parseProbeUsage(body []byte) probeUsage {
	if len(body) == 0 {
		return probeUsage{}
	}
	var payload struct {
		Usage struct {
			PromptTokens      int `json:"prompt_tokens"`
			CompletionTokens  int `json:"completion_tokens"`
			InputTokens       int `json:"input_tokens"`
			OutputTokens      int `json:"output_tokens"`
			InputTokensCamel  int `json:"inputTokens"`
			OutputTokensCamel int `json:"outputTokens"`

			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			PromptTokensDetails      struct {
				CachedTokens         int `json:"cached_tokens"`
				CachedCreationTokens int `json:"cached_creation_tokens"`
			} `json:"prompt_tokens_details"`
			InputTokensDetails struct {
				CachedTokens         int `json:"cached_tokens"`
				CachedCreationTokens int `json:"cached_creation_tokens"`
			} `json:"input_tokens_details"`
			CacheCreation struct {
				Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens"`
				Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens"`
			} `json:"cache_creation"`
		} `json:"usage"`
	}
	if err := common.Unmarshal(body, &payload); err != nil {
		return probeUsage{}
	}

	usage := probeUsage{
		PromptTokens:        payload.Usage.PromptTokens,
		CompletionTokens:    payload.Usage.CompletionTokens,
		CacheReadTokens:     payload.Usage.CacheReadInputTokens,
		CacheCreationTokens: payload.Usage.CacheCreationInputTokens,
	}
	if usage.PromptTokens == 0 {
		usage.PromptTokens = payload.Usage.InputTokens
	}
	if usage.PromptTokens == 0 {
		usage.PromptTokens = payload.Usage.InputTokensCamel
	}
	if usage.CompletionTokens == 0 {
		usage.CompletionTokens = payload.Usage.OutputTokens
	}
	if usage.CompletionTokens == 0 {
		usage.CompletionTokens = payload.Usage.OutputTokensCamel
	}
	if usage.CacheReadTokens == 0 {
		usage.CacheReadTokens = payload.Usage.PromptTokensDetails.CachedTokens
	}
	if usage.CacheReadTokens == 0 {
		usage.CacheReadTokens = payload.Usage.InputTokensDetails.CachedTokens
	}
	if usage.CacheCreationTokens == 0 {
		usage.CacheCreationTokens = payload.Usage.PromptTokensDetails.CachedCreationTokens
	}
	if usage.CacheCreationTokens == 0 {
		usage.CacheCreationTokens = payload.Usage.InputTokensDetails.CachedCreationTokens
	}
	if usage.CacheCreationTokens == 0 {
		usage.CacheCreationTokens = payload.Usage.CacheCreation.Ephemeral5mInputTokens + payload.Usage.CacheCreation.Ephemeral1hInputTokens
	}
	return usage
}

func estimateProbeQuota(modelName string, usage probeUsage) int {
	if usage.totalTokens() == 0 {
		return 0
	}
	modelRatio, ok, _ := ratio_setting.GetModelRatio(modelName)
	if !ok || modelRatio <= 0 {
		return 0
	}
	completionRatio := ratio_setting.GetCompletionRatio(modelName)
	cacheRatio, _ := ratio_setting.GetCacheRatio(modelName)
	cacheCreationRatio, _ := ratio_setting.GetCreateCacheRatio(modelName)

	quota := decimal.NewFromInt(int64(usage.PromptTokens)).
		Add(decimal.NewFromInt(int64(usage.CompletionTokens)).Mul(decimal.NewFromFloat(completionRatio))).
		Add(decimal.NewFromInt(int64(usage.CacheReadTokens)).Mul(decimal.NewFromFloat(cacheRatio))).
		Add(decimal.NewFromInt(int64(usage.CacheCreationTokens)).Mul(decimal.NewFromFloat(cacheCreationRatio))).
		Mul(decimal.NewFromFloat(modelRatio))
	if quota.LessThanOrEqual(decimal.Zero) {
		return 0
	}
	rounded := int(quota.Round(0).IntPart())
	if rounded == 0 {
		return 1
	}
	return rounded
}

// getProbeEndpoint returns the full URL for the probe request based on channel type.
func getProbeEndpoint(channel *model.Channel) string {
	baseURL := channel.GetBaseURL()
	baseURL = strings.TrimRight(baseURL, "/")

	// Most channels use OpenAI-compatible chat completions endpoint
	switch channel.Type {
	case constant.ChannelTypeAnthropic:
		return baseURL + "/v1/messages"
	default:
		return baseURL + "/v1/chat/completions"
	}
}

// getProbeAuthHeader returns the Authorization header value for the channel.
func getProbeAuthHeader(channel *model.Channel) string {
	key := channel.Key
	if key == "" {
		keys := channel.GetKeys()
		if len(keys) > 0 {
			key = keys[0]
		}
	}
	if key == "" {
		return ""
	}

	switch channel.Type {
	case constant.ChannelTypeAnthropic:
		// Claude uses x-api-key, but for probe we use Authorization Bearer
		return "Bearer " + key
	default:
		return "Bearer " + key
	}
}

// recordProbeLog records probe execution to the logs table.
// Uses gopool.Go for async write to avoid blocking the probe/retry flow.
// user_id=0 ensures no user billing impact; quota is only a platform-cost estimate for log display.
func recordProbeLog(probeLog *ProbeLog, requestId string) {
	if probeLog == nil {
		return
	}
	// Log at debug level to avoid flooding
	if common.DebugEnabled {
		common.SysLog(fmt.Sprintf("NACP probe: channel=#%d model=%s success=%v latency=%dms trigger=%s",
			probeLog.ChannelID, probeLog.ModelName, probeLog.Success, probeLog.LatencyMs, probeLog.Trigger))
	}

	// Determine the input role; model.Log normalizes it before storage.
	logType := model.LogTypeProbeSuccess
	if !probeLog.Success {
		logType = model.LogTypeProbeFailed
	}

	hasUsage := probeUsage{
		PromptTokens:        probeLog.PromptTokens,
		CompletionTokens:    probeLog.CompletionTokens,
		CacheReadTokens:     probeLog.CacheReadTokens,
		CacheCreationTokens: probeLog.CacheCreationTokens,
	}.hasUsage()

	// Build Other field
	other := map[string]interface{}{
		"probe_trigger":        probeLog.Trigger,
		"cost_scope":           "platform",
		"cost_reason":          "lightweight_probe",
		"probe_usage_recorded": hasUsage,
		"quota_estimated":      hasUsage && probeLog.EstimatedQuota > 0,
		"admin_info": map[string]interface{}{
			"status_code":  probeLog.StatusCode,
			"latency_ms":   probeLog.LatencyMs,
			"channel_name": probeLog.ChannelName,
		},
	}
	if hasUsage {
		other["probe_usage_source"] = "upstream_response"
	}
	if probeLog.CacheReadTokens > 0 {
		other["cache_tokens"] = probeLog.CacheReadTokens
	}
	if probeLog.CacheCreationTokens > 0 {
		other["cache_creation_tokens"] = probeLog.CacheCreationTokens
	}
	if !probeLog.Success && probeLog.Error != "" {
		other["error"] = probeLog.Error
	}
	otherStr := common.MapToJsonStr(other)

	// Use time in seconds (milliseconds / 1000, round up)
	useTimeSec := int((probeLog.LatencyMs + 999) / 1000)

	log := &model.Log{
		UserId:           0, // probe cost not billed to any user
		Username:         "",
		CreatedAt:        probeLog.Timestamp,
		Type:             logType,
		Content:          fmt.Sprintf("probe %s channel #%d", probeLog.ModelName, probeLog.ChannelID),
		ModelName:        probeLog.ModelName,
		Quota:            probeLog.EstimatedQuota,
		PromptTokens:     probeLog.PromptTokens,
		CompletionTokens: probeLog.CompletionTokens,
		ChannelId:        probeLog.ChannelID,
		UseTime:          useTimeSec,
		RequestId:        requestId,
		Other:            otherStr,
	}
	model.ApplyLogTraceFields(log)

	gopool.Go(func() {
		if err := model.LOG_DB.Create(log).Error; err != nil {
			common.SysLog("failed to record probe log: " + err.Error())
			return
		}
		model.FinalizeTraceAfterLogCreated(log)
	})
}

// probeDegradedChannels probes all channels in Degraded state.
func probeDegradedChannels() {
	cfg := GetHealthConfig()

	channelHealthStatesMu.RLock()
	var degradedIDs []int
	for id, state := range channelHealthStates {
		state.mu.RLock()
		if state.Status == HealthStatusDegraded {
			if time.Since(state.LastProbeAt) >= cfg.DegradedProbeInterval {
				degradedIDs = append(degradedIDs, id)
			}
		}
		state.mu.RUnlock()
	}
	channelHealthStatesMu.RUnlock()

	if len(degradedIDs) == 0 {
		return
	}

	common.SysLog(fmt.Sprintf("NACP: probing %d degraded channels", len(degradedIDs)))

	for _, id := range degradedIDs {
		channel, err := model.CacheGetChannel(id)
		if err != nil || channel == nil {
			continue
		}
		if channel.Status != common.ChannelStatusEnabled {
			continue
		}

		// Use test model or first model
		modelName := getProbeModelForChannel(channel)
		if modelName == "" {
			continue
		}

		result := ProbeChannel(channel, modelName)
		OnProbeResult(id, result.Success)

		var errMsg string
		if result.Error != nil {
			errMsg = result.Error.Error()
		}
		recordProbeLog(&ProbeLog{
			ChannelID:           id,
			ChannelName:         channel.Name,
			ModelName:           modelName,
			Success:             result.Success,
			LatencyMs:           result.LatencyMs,
			StatusCode:          result.StatusCode,
			Trigger:             "degraded_probe",
			Error:               errMsg,
			Timestamp:           time.Now().Unix(),
			PromptTokens:        result.PromptTokens,
			CompletionTokens:    result.CompletionTokens,
			CacheReadTokens:     result.CacheReadTokens,
			CacheCreationTokens: result.CacheCreationTokens,
			EstimatedQuota:      result.EstimatedQuota,
		}, "") // empty requestId — degraded probe not linked to any user request

		// Small delay between probes to avoid burst
		time.Sleep(500 * time.Millisecond)
	}
}

// getProbeModelForChannel returns the model name to use for probing a channel.
func getProbeModelForChannel(channel *model.Channel) string {
	if channel.TestModel != nil && *channel.TestModel != "" {
		return strings.TrimSpace(*channel.TestModel)
	}
	models := channel.GetModels()
	if len(models) > 0 {
		return strings.TrimSpace(models[0])
	}
	return ""
}
