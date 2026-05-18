package model

import (
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

type NacpStatsOverview struct {
	GeneratedAt  int64                      `json:"generated_at"`
	RangeStart   int64                      `json:"range_start"`
	RangeEnd     int64                      `json:"range_end"`
	RangeHours   int                        `json:"range_hours"`
	UnitMonitors NacpUnitMonitorStats       `json:"unit_monitors"`
	Users        NacpUserStats              `json:"users"`
	Channels     NacpChannelStats           `json:"channels"`
	SFT          NacpSFTStats               `json:"sft"`
	Traffic      NacpTrafficStats           `json:"traffic"`
	Cost         NacpCostStats              `json:"cost"`
	Health       NacpOperationalHealth      `json:"health"`
	Trend        []NacpTrendPoint           `json:"trend"`
	Models       []NacpModelActivityStats   `json:"models"`
	TopUsers     []NacpUserActivityStats    `json:"top_users"`
	TopChannels  []NacpChannelActivityStats `json:"top_channels"`
}

type NacpUnitMonitorStats struct {
	Total           int64   `json:"total"`
	Enabled         int64   `json:"enabled"`
	OK              int64   `json:"ok"`
	Error           int64   `json:"error"`
	Pending         int64   `json:"pending"`
	CurrentBalance  float64 `json:"current_balance"`
	UsedAmount      float64 `json:"used_amount"`
	TokenCount      int64   `json:"token_count"`
	ModelCount      int64   `json:"model_count"`
	GroupCount      int64   `json:"group_count"`
	LastCheckedTime int64   `json:"last_checked_time"`
}

type NacpUserStats struct {
	Total        int64 `json:"total"`
	Enabled      int64 `json:"enabled"`
	Disabled     int64 `json:"disabled"`
	Admin        int64 `json:"admin"`
	Active24h    int64 `json:"active_24h"`
	ActiveUsers  int64 `json:"active_users"`
	Quota        int64 `json:"quota"`
	UsedQuota    int64 `json:"used_quota"`
	RequestCount int64 `json:"request_count"`
}

type NacpChannelStats struct {
	Total            int64 `json:"total"`
	Enabled          int64 `json:"enabled"`
	ManuallyDisabled int64 `json:"manually_disabled"`
	AutoDisabled     int64 `json:"auto_disabled"`
	Healthy          int64 `json:"healthy"`
	Degraded         int64 `json:"degraded"`
	Unhealthy        int64 `json:"unhealthy"`
	UsedQuota        int64 `json:"used_quota"`
}

type NacpSFTStats struct {
	SuccessSummary  int64 `json:"success_summary"`
	FailedSummary   int64 `json:"failed_summary"`
	Intercepted     int64 `json:"intercepted"`
	ClientVisible   int64 `json:"client_visible"`
	ProbeSuccess    int64 `json:"probe_success"`
	ProbeFailed     int64 `json:"probe_failed"`
	DirectConsume   int64 `json:"direct_consume"`
	RetryConsume    int64 `json:"retry_consume"`
	LegacyError     int64 `json:"legacy_error"`
	SuccessRatePerm int64 `json:"success_rate_perm"`
}

type NacpTrafficStats struct {
	RequestCount      int64 `json:"request_count"`
	SuccessCount      int64 `json:"success_count"`
	ErrorCount        int64 `json:"error_count"`
	UserVisibleErrors int64 `json:"user_visible_errors"`
	InterceptedErrors int64 `json:"intercepted_errors"`
	ProbeCount        int64 `json:"probe_count"`
	StreamCount       int64 `json:"stream_count"`
	StreamRatePerm    int64 `json:"stream_rate_perm"`
	Tokens            int64 `json:"tokens"`
	Quota             int64 `json:"quota"`
	AverageUseTime    int64 `json:"average_use_time"`
	SuccessRatePerm   int64 `json:"success_rate_perm"`
}

type NacpCostStats struct {
	DayStart             int64   `json:"day_start"`
	DayEnd               int64   `json:"day_end"`
	PlatformRevenueQuota int64   `json:"platform_revenue_quota"`
	PlatformRevenueUSD   float64 `json:"platform_revenue_usd"`
	UpstreamCostUSD      float64 `json:"upstream_cost_usd"`
	EstimatedProfitUSD   float64 `json:"estimated_profit_usd"`
	GrossMarginPerm      int64   `json:"gross_margin_perm"`
	MonitorCount         int64   `json:"monitor_count"`
	BaselineMonitorCount int64   `json:"baseline_monitor_count"`
	MissingBaselineCount int64   `json:"missing_baseline_count"`
	SnapshotCount        int64   `json:"snapshot_count"`
	Estimated            bool    `json:"estimated"`
	Note                 string  `json:"note"`
}

type NacpOperationalHealth struct {
	ScorePerm          int64    `json:"score_perm"`
	Level              string   `json:"level"`
	TrafficSuccessPerm int64    `json:"traffic_success_perm"`
	UnitHealthPerm     int64    `json:"unit_health_perm"`
	ChannelHealthPerm  int64    `json:"channel_health_perm"`
	CostCoveragePerm   int64    `json:"cost_coverage_perm"`
	Notes              []string `json:"notes"`
}

type NacpTrendPoint struct {
	BucketStart     int64 `json:"bucket_start"`
	RequestCount    int64 `json:"request_count"`
	SuccessCount    int64 `json:"success_count"`
	ErrorCount      int64 `json:"error_count"`
	ProbeCount      int64 `json:"probe_count"`
	Tokens          int64 `json:"tokens"`
	Quota           int64 `json:"quota"`
	SuccessRatePerm int64 `json:"success_rate_perm"`
}

type NacpModelActivityStats struct {
	ModelName    string `json:"model_name"`
	RequestCount int64  `json:"request_count"`
	Quota        int64  `json:"quota"`
	Tokens       int64  `json:"tokens"`
}

type NacpUserActivityStats struct {
	UserId       int    `json:"user_id"`
	Username     string `json:"username"`
	RequestCount int64  `json:"request_count"`
	Quota        int64  `json:"quota"`
	Tokens       int64  `json:"tokens"`
}

type NacpChannelActivityStats struct {
	ChannelId    int    `json:"channel_id"`
	ChannelName  string `json:"channel_name"`
	RequestCount int64  `json:"request_count"`
	Quota        int64  `json:"quota"`
	Tokens       int64  `json:"tokens"`
}

type NacpStatsRankMeta struct {
	Total      int64  `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	RangeStart int64  `json:"range_start"`
	RangeEnd   int64  `json:"range_end"`
	RangeHours int    `json:"range_hours"`
	Sort       string `json:"sort"`
}

type NacpModelRankPage struct {
	NacpStatsRankMeta
	Items []NacpModelRankStats `json:"items"`
}

type NacpUserRankPage struct {
	NacpStatsRankMeta
	Items []NacpUserRankStats `json:"items"`
}

type NacpChannelRankPage struct {
	NacpStatsRankMeta
	Items []NacpChannelRankStats `json:"items"`
}

type NacpCostRankPage struct {
	NacpStatsRankMeta
	Dimension string              `json:"dimension"`
	DayStart  int64               `json:"day_start"`
	DayEnd    int64               `json:"day_end"`
	Items     []NacpCostRankStats `json:"items"`
}

type NacpDimensionRankPage struct {
	NacpStatsRankMeta
	Dimension string                   `json:"dimension"`
	Items     []NacpDimensionRankStats `json:"items"`
}

type NacpModelCoveragePage struct {
	NacpStatsRankMeta
	Items []NacpModelCoverageStats `json:"items"`
}

type NacpModelRankStats struct {
	ModelName          string `json:"model_name"`
	RequestCount       int64  `json:"request_count"`
	SuccessCount       int64  `json:"success_count"`
	ErrorCount         int64  `json:"error_count"`
	DirectSuccessCount int64  `json:"direct_success_count"`
	RetrySuccessCount  int64  `json:"retry_success_count"`
	RetryFailedCount   int64  `json:"retry_failed_count"`
	LegacyErrorCount   int64  `json:"legacy_error_count"`
	StreamCount        int64  `json:"stream_count"`
	Tokens             int64  `json:"tokens"`
	Quota              int64  `json:"quota"`
	AverageUseTime     int64  `json:"average_use_time"`
	SuccessRatePerm    int64  `json:"success_rate_perm"`
	StreamRatePerm     int64  `json:"stream_rate_perm"`
}

type NacpUserRankStats struct {
	UserId             int    `json:"user_id"`
	Username           string `json:"username"`
	RequestCount       int64  `json:"request_count"`
	SuccessCount       int64  `json:"success_count"`
	ErrorCount         int64  `json:"error_count"`
	DirectSuccessCount int64  `json:"direct_success_count"`
	RetrySuccessCount  int64  `json:"retry_success_count"`
	RetryFailedCount   int64  `json:"retry_failed_count"`
	LegacyErrorCount   int64  `json:"legacy_error_count"`
	StreamCount        int64  `json:"stream_count"`
	Tokens             int64  `json:"tokens"`
	Quota              int64  `json:"quota"`
	AverageUseTime     int64  `json:"average_use_time"`
	SuccessRatePerm    int64  `json:"success_rate_perm"`
	StreamRatePerm     int64  `json:"stream_rate_perm"`
}

type NacpChannelRankStats struct {
	ChannelId          int    `json:"channel_id"`
	ChannelName        string `json:"channel_name"`
	Status             int    `json:"status"`
	HealthStatus       string `json:"health_status"`
	Priority           int64  `json:"priority"`
	Weight             uint   `json:"weight"`
	UnitID             int    `json:"unit_id"`
	UnitName           string `json:"unit_name"`
	UnitType           string `json:"unit_type"`
	UnitAccountID      int    `json:"unit_account_id"`
	UnitAccountName    string `json:"unit_account_name"`
	RequestCount       int64  `json:"request_count"`
	SuccessCount       int64  `json:"success_count"`
	ErrorCount         int64  `json:"error_count"`
	DirectSuccessCount int64  `json:"direct_success_count"`
	RetrySuccessCount  int64  `json:"retry_success_count"`
	RetryFailedCount   int64  `json:"retry_failed_count"`
	LegacyErrorCount   int64  `json:"legacy_error_count"`
	StreamCount        int64  `json:"stream_count"`
	Tokens             int64  `json:"tokens"`
	Quota              int64  `json:"quota"`
	AverageUseTime     int64  `json:"average_use_time"`
	SuccessRatePerm    int64  `json:"success_rate_perm"`
	StreamRatePerm     int64  `json:"stream_rate_perm"`
}

type NacpModelCoverageChannel struct {
	ChannelId    int    `json:"channel_id"`
	ChannelName  string `json:"channel_name"`
	Group        string `json:"group"`
	Status       int    `json:"status"`
	HealthStatus string `json:"health_status"`
	Priority     int64  `json:"priority"`
	Weight       uint   `json:"weight"`
	UnitID       int    `json:"unit_id"`
	AccountID    int    `json:"account_id"`
}

type NacpModelCoverageStats struct {
	ModelName            string                     `json:"model_name"`
	GroupCount           int64                      `json:"group_count"`
	EnabledGroupCount    int64                      `json:"enabled_group_count"`
	ChannelCount         int64                      `json:"channel_count"`
	EnabledChannelCount  int64                      `json:"enabled_channel_count"`
	HealthyChannelCount  int64                      `json:"healthy_channel_count"`
	DegradedChannelCount int64                      `json:"degraded_channel_count"`
	DisabledChannelCount int64                      `json:"disabled_channel_count"`
	CoverageRatePerm     int64                      `json:"coverage_rate_perm"`
	HealthyRatePerm      int64                      `json:"healthy_rate_perm"`
	RequestCount         int64                      `json:"request_count"`
	SuccessCount         int64                      `json:"success_count"`
	ErrorCount           int64                      `json:"error_count"`
	Tokens               int64                      `json:"tokens"`
	Quota                int64                      `json:"quota"`
	AverageUseTime       int64                      `json:"average_use_time"`
	SuccessRatePerm      int64                      `json:"success_rate_perm"`
	RiskLevel            string                     `json:"risk_level"`
	RiskReason           string                     `json:"risk_reason"`
	Groups               []string                   `json:"groups"`
	EnabledGroups        []string                   `json:"enabled_groups"`
	SampleChannels       []NacpModelCoverageChannel `json:"sample_channels"`
}

type NacpCostRankStats struct {
	Key                   string  `json:"key"`
	Label                 string  `json:"label"`
	Dimension             string  `json:"dimension"`
	UnitID                int     `json:"unit_id"`
	UnitName              string  `json:"unit_name"`
	UnitType              string  `json:"unit_type"`
	UnitAccountID         int     `json:"unit_account_id"`
	Account               string  `json:"account"`
	MonitorID             int     `json:"monitor_id"`
	ChannelId             int     `json:"channel_id"`
	ChannelName           string  `json:"channel_name"`
	ChannelStatus         int     `json:"channel_status"`
	ChannelHealthStatus   string  `json:"channel_health_status"`
	Priority              int64   `json:"priority"`
	Weight                uint    `json:"weight"`
	ModelName             string  `json:"model_name"`
	UserId                int     `json:"user_id"`
	Username              string  `json:"username"`
	RequestCount          int64   `json:"request_count"`
	Tokens                int64   `json:"tokens"`
	Quota                 int64   `json:"quota"`
	PlatformRevenueUSD    float64 `json:"platform_revenue_usd"`
	UpstreamCostUSD       float64 `json:"upstream_cost_usd"`
	EstimatedProfitUSD    float64 `json:"estimated_profit_usd"`
	GrossMarginPerm       int64   `json:"gross_margin_perm"`
	CurrentBalance        float64 `json:"current_balance"`
	UsedAmount            float64 `json:"used_amount"`
	BalanceUnit           string  `json:"balance_unit"`
	SnapshotCount         int64   `json:"snapshot_count"`
	BaselineReady         bool    `json:"baseline_ready"`
	Estimated             bool    `json:"estimated"`
	CostSource            string  `json:"cost_source"`
	FirstSnapshotTime     int64   `json:"first_snapshot_time"`
	LastSnapshotTime      int64   `json:"last_snapshot_time"`
	LastCheckedTime       int64   `json:"last_checked_time"`
	PlatformStatus        string  `json:"platform_status"`
	MissingSnapshotReason string  `json:"missing_snapshot_reason"`
}

type NacpDimensionRankStats struct {
	DimensionKey       string `json:"dimension_key"`
	DimensionID        int    `json:"dimension_id"`
	Label              string `json:"label"`
	RequestCount       int64  `json:"request_count"`
	SuccessCount       int64  `json:"success_count"`
	ErrorCount         int64  `json:"error_count"`
	DirectSuccessCount int64  `json:"direct_success_count"`
	RetrySuccessCount  int64  `json:"retry_success_count"`
	RetryFailedCount   int64  `json:"retry_failed_count"`
	LegacyErrorCount   int64  `json:"legacy_error_count"`
	StreamCount        int64  `json:"stream_count"`
	Tokens             int64  `json:"tokens"`
	Quota              int64  `json:"quota"`
	AverageUseTime     int64  `json:"average_use_time"`
	SuccessRatePerm    int64  `json:"success_rate_perm"`
	StreamRatePerm     int64  `json:"stream_rate_perm"`
}

func GetNacpStatsOverview(rangeHours int) (*NacpStatsOverview, error) {
	if rangeHours <= 0 {
		rangeHours = 24
	}
	now := time.Now().Unix()
	start := now - int64(rangeHours)*60*60
	overview := &NacpStatsOverview{
		GeneratedAt: now,
		RangeStart:  start,
		RangeEnd:    now,
		RangeHours:  rangeHours,
	}

	var (
		unitMonitors NacpUnitMonitorStats
		users        NacpUserStats
		channels     NacpChannelStats
		sft          NacpSFTStats
		traffic      NacpTrafficStats
		cost         NacpCostStats
		trend        []NacpTrendPoint
		models       []NacpModelActivityStats
		topUsers     []NacpUserActivityStats
		topChannels  []NacpChannelActivityStats
		wg           sync.WaitGroup
		errMu        sync.Mutex
		firstErr     error
	)

	run := func(fn func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(); err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
			}
		}()
	}

	run(func() error { return fillNacpUnitMonitorStats(&unitMonitors) })
	run(func() error { return fillNacpUserStats(&users, start, now) })
	run(func() error { return fillNacpChannelStats(&channels) })
	run(func() error { return fillNacpSFTStats(&sft, start, now) })
	run(func() error { return fillNacpTrafficStats(&traffic, start, now) })
	run(func() error { return fillNacpCostStats(&cost) })
	run(func() error {
		var err error
		trend, err = getNacpTrendStats(start, now, rangeHours)
		return err
	})
	run(func() error {
		var err error
		models, err = getNacpModelActivityStats(start, now, 8)
		return err
	})
	run(func() error {
		var err error
		topUsers, err = getNacpUserActivityStats(start, now, 8)
		return err
	})
	run(func() error {
		var err error
		topChannels, err = getNacpChannelActivityStats(start, now, 8)
		return err
	})
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	overview.UnitMonitors = unitMonitors
	overview.Users = users
	overview.Channels = channels
	overview.SFT = sft
	overview.Traffic = traffic
	overview.Cost = cost
	overview.Health = buildNacpOperationalHealth(unitMonitors, channels, traffic, cost)
	if trend == nil {
		trend = []NacpTrendPoint{}
	}
	if models == nil {
		models = []NacpModelActivityStats{}
	}
	if topUsers == nil {
		topUsers = []NacpUserActivityStats{}
	}
	if topChannels == nil {
		topChannels = []NacpChannelActivityStats{}
	}
	overview.Trend = trend
	overview.Models = models
	overview.TopUsers = topUsers
	overview.TopChannels = topChannels
	return overview, nil
}

func fillNacpUnitMonitorStats(stats *NacpUnitMonitorStats) error {
	type aggregate struct {
		Total           int64
		Enabled         int64
		OK              int64
		Error           int64
		Pending         int64
		CurrentBalance  float64
		UsedAmount      float64
		TokenCount      int64
		ModelCount      int64
		GroupCount      int64
		LastCheckedTime int64
	}
	var row aggregate
	if err := DB.Model(&UnitAccountMonitor{}).
		Select(
			"COUNT(*) AS total, COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END),0) AS enabled, COALESCE(SUM(CASE WHEN platform_status = ? THEN 1 ELSE 0 END),0) AS ok, COALESCE(SUM(CASE WHEN platform_status = ? THEN 1 ELSE 0 END),0) AS error, COALESCE(SUM(CASE WHEN platform_status = ? OR platform_status = ? THEN 1 ELSE 0 END),0) AS pending, COALESCE(SUM(current_balance),0) AS current_balance, COALESCE(SUM(used_amount),0) AS used_amount, COALESCE(SUM(token_count),0) AS token_count, COALESCE(SUM(model_count),0) AS model_count, COALESCE(SUM(group_count),0) AS group_count, COALESCE(MAX(last_checked_time),0) AS last_checked_time",
			UnitMonitorStatusEnabled,
			"ok",
			"error",
			"",
			"pending",
		).
		Scan(&row).Error; err != nil {
		return err
	}
	stats.Total = row.Total
	stats.Enabled = row.Enabled
	stats.OK = row.OK
	stats.Error = row.Error
	stats.Pending = row.Pending
	stats.CurrentBalance = row.CurrentBalance
	stats.UsedAmount = row.UsedAmount
	stats.TokenCount = row.TokenCount
	stats.ModelCount = row.ModelCount
	stats.GroupCount = row.GroupCount
	stats.LastCheckedTime = row.LastCheckedTime
	return nil
}

func fillNacpUserStats(stats *NacpUserStats, start int64, end int64) error {
	type aggregate struct {
		Total        int64
		Enabled      int64
		Disabled     int64
		Admin        int64
		Quota        int64
		UsedQuota    int64
		RequestCount int64
	}
	var row aggregate
	if err := DB.Unscoped().Model(&User{}).
		Select(
			"COUNT(*) AS total, COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END),0) AS enabled, COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END),0) AS disabled, COALESCE(SUM(CASE WHEN role = ? OR role = ? THEN 1 ELSE 0 END),0) AS admin, COALESCE(SUM(quota),0) AS quota, COALESCE(SUM(used_quota),0) AS used_quota, COALESCE(SUM(request_count),0) AS request_count",
			common.UserStatusEnabled,
			common.UserStatusDisabled,
			common.RoleAdminUser,
			common.RoleRootUser,
		).
		Scan(&row).Error; err != nil {
		return err
	}
	stats.Total = row.Total
	stats.Enabled = row.Enabled
	stats.Disabled = row.Disabled
	stats.Admin = row.Admin
	stats.Quota = row.Quota
	stats.UsedQuota = row.UsedQuota
	stats.RequestCount = row.RequestCount
	if LOG_DB != nil {
		if err := LOG_DB.Model(&Log{}).
			Where("created_at >= ? AND created_at <= ? AND user_id > 0 AND type IN ?", start, end, nacpUserVisibleLogTypes()).
			Distinct("user_id").
			Count(&stats.Active24h).Error; err != nil {
			return err
		}
		stats.ActiveUsers = stats.Active24h
	}
	return nil
}

func fillNacpChannelStats(stats *NacpChannelStats) error {
	type aggregate struct {
		Total            int64
		Enabled          int64
		ManuallyDisabled int64
		AutoDisabled     int64
		Healthy          int64
		Degraded         int64
		Unhealthy        int64
		UsedQuota        int64
	}
	var row aggregate
	if err := DB.Model(&Channel{}).
		Select(
			"COUNT(*) AS total, COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END),0) AS enabled, COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END),0) AS manually_disabled, COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END),0) AS auto_disabled, COALESCE(SUM(CASE WHEN health_status = ? OR health_status = ? THEN 1 ELSE 0 END),0) AS healthy, COALESCE(SUM(CASE WHEN health_status = ? THEN 1 ELSE 0 END),0) AS degraded, COALESCE(SUM(CASE WHEN health_status = ? THEN 1 ELSE 0 END),0) AS unhealthy, COALESCE(SUM(used_quota),0) AS used_quota",
			common.ChannelStatusEnabled,
			common.ChannelStatusManuallyDisabled,
			common.ChannelStatusAutoDisabled,
			"",
			"healthy",
			"degraded",
			"unhealthy",
		).
		Scan(&row).Error; err != nil {
		return err
	}
	stats.Total = row.Total
	stats.Enabled = row.Enabled
	stats.ManuallyDisabled = row.ManuallyDisabled
	stats.AutoDisabled = row.AutoDisabled
	stats.Healthy = row.Healthy
	stats.Degraded = row.Degraded
	stats.Unhealthy = row.Unhealthy
	stats.UsedQuota = row.UsedQuota
	return nil
}

func fillNacpSFTStats(stats *NacpSFTStats, start int64, end int64) error {
	if LOG_DB == nil {
		return nil
	}
	rows := []struct {
		Type  int
		Count int64
	}{}
	if err := LOG_DB.Model(&Log{}).
		Select("type, COUNT(*) AS count").
		Where("created_at >= ? AND created_at <= ? AND type IN ?", start, end, []int{
			LogTypeRetrySuccessSummary,
			LogTypeRetryFailedSummary,
			LogTypeErrorIntercepted,
			LogTypeErrorClientVisible,
			LogTypeProbeSuccess,
			LogTypeProbeFailed,
			LogTypeConsume,
			LogTypeRetryConsume,
			LogTypeError,
		}).
		Group("type").
		Scan(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		switch row.Type {
		case LogTypeRetrySuccessSummary:
			stats.SuccessSummary = row.Count
		case LogTypeRetryFailedSummary:
			stats.FailedSummary = row.Count
		case LogTypeErrorIntercepted:
			stats.Intercepted = row.Count
		case LogTypeErrorClientVisible:
			stats.ClientVisible = row.Count
		case LogTypeProbeSuccess:
			stats.ProbeSuccess = row.Count
		case LogTypeProbeFailed:
			stats.ProbeFailed = row.Count
		case LogTypeConsume:
			stats.DirectConsume = row.Count
		case LogTypeRetryConsume:
			stats.RetryConsume = row.Count
		case LogTypeError:
			stats.LegacyError = row.Count
		}
	}
	total := stats.SuccessSummary + stats.FailedSummary
	if total > 0 {
		stats.SuccessRatePerm = stats.SuccessSummary * 1000 / total
	}
	return nil
}

func fillNacpTrafficStats(stats *NacpTrafficStats, start int64, end int64) error {
	if LOG_DB == nil {
		return nil
	}
	type aggregate struct {
		RequestCount      int64
		SuccessCount      int64
		ErrorCount        int64
		UserVisibleErrors int64
		InterceptedErrors int64
		ProbeCount        int64
		StreamCount       int64
		Tokens            int64
		Quota             int64
		AverageUseTime    float64
	}
	var row aggregate
	if err := LOG_DB.Model(&Log{}).
		Select(
			"COALESCE(SUM(CASE WHEN type IN ? THEN 1 ELSE 0 END),0) AS request_count, COALESCE(SUM(CASE WHEN type IN ? THEN 1 ELSE 0 END),0) AS success_count, COALESCE(SUM(CASE WHEN type IN ? THEN 1 ELSE 0 END),0) AS error_count, COALESCE(SUM(CASE WHEN type IN ? THEN 1 ELSE 0 END),0) AS user_visible_errors, COALESCE(SUM(CASE WHEN type = ? THEN 1 ELSE 0 END),0) AS intercepted_errors, COALESCE(SUM(CASE WHEN type IN ? THEN 1 ELSE 0 END),0) AS probe_count, COALESCE(SUM(CASE WHEN type IN ? AND is_stream = ? THEN 1 ELSE 0 END),0) AS stream_count, COALESCE(SUM(CASE WHEN type IN ? THEN prompt_tokens + completion_tokens ELSE 0 END),0) AS tokens, COALESCE(SUM(CASE WHEN type IN ? THEN quota ELSE 0 END),0) AS quota, COALESCE(AVG(CASE WHEN type IN ? AND use_time > 0 THEN use_time ELSE NULL END),0) AS average_use_time",
			nacpUserVisibleLogTypes(),
			nacpSuccessSummaryLogTypes(),
			nacpUserVisibleErrorTypes(),
			nacpUserVisibleErrorTypes(),
			LogTypeErrorIntercepted,
			[]int{LogTypeProbeSuccess, LogTypeProbeFailed},
			nacpUserVisibleLogTypes(),
			true,
			nacpSuccessSummaryLogTypes(),
			nacpSuccessSummaryLogTypes(),
			nacpSuccessSummaryLogTypes(),
		).
		Where("created_at >= ? AND created_at <= ? AND type IN ?", start, end, nacpObservedLogTypes()).
		Scan(&row).Error; err != nil {
		return err
	}
	stats.RequestCount = row.RequestCount
	stats.SuccessCount = row.SuccessCount
	stats.ErrorCount = row.ErrorCount
	stats.UserVisibleErrors = row.UserVisibleErrors
	stats.InterceptedErrors = row.InterceptedErrors
	stats.ProbeCount = row.ProbeCount
	stats.StreamCount = row.StreamCount
	stats.Tokens = row.Tokens
	stats.Quota = row.Quota
	stats.AverageUseTime = int64(row.AverageUseTime + 0.5)
	if row.RequestCount > 0 {
		stats.StreamRatePerm = row.StreamCount * 1000 / row.RequestCount
		stats.SuccessRatePerm = row.SuccessCount * 1000 / row.RequestCount
	}
	return nil
}

func fillNacpCostStats(stats *NacpCostStats) error {
	start, end := nacpTodayRange()
	stats.DayStart = start
	stats.DayEnd = end
	if LOG_DB != nil {
		type revenueRow struct {
			Quota int64
		}
		var row revenueRow
		if err := LOG_DB.Model(&Log{}).
			Select("COALESCE(SUM(quota),0) AS quota").
			Where("created_at >= ? AND created_at <= ? AND type IN ?", start, end, nacpSuccessSummaryLogTypes()).
			Scan(&row).Error; err != nil {
			return err
		}
		stats.PlatformRevenueQuota = row.Quota
		if common.QuotaPerUnit > 0 {
			stats.PlatformRevenueUSD = float64(row.Quota) / common.QuotaPerUnit
		}
	}
	if DB != nil {
		if err := fillNacpUpstreamCostStats(stats, start, end); err != nil {
			return err
		}
	}
	stats.EstimatedProfitUSD = stats.PlatformRevenueUSD - stats.UpstreamCostUSD
	if stats.PlatformRevenueUSD > 0 {
		stats.GrossMarginPerm = int64((stats.EstimatedProfitUSD / stats.PlatformRevenueUSD) * 1000)
	}
	if stats.MonitorCount == 0 {
		stats.Estimated = true
		stats.Note = "未配置单位账号统计源，今日上游成本暂按 0 显示"
	} else if stats.MissingBaselineCount > 0 {
		stats.Estimated = true
		stats.Note = "部分统计源缺少今日两次以上快照，上游成本为可计算统计源的已用额度差值"
	} else {
		stats.Note = "上游成本按今日第一条和最新单位账号快照的已用额度差值计算"
	}
	return nil
}

func fillNacpUpstreamCostStats(stats *NacpCostStats, start int64, end int64) error {
	if err := DB.Model(&UnitAccountMonitor{}).Where("status = ?", UnitMonitorStatusEnabled).Count(&stats.MonitorCount).Error; err != nil {
		return err
	}
	if !DB.Migrator().HasTable(&UnitAccountMonitorSnapshot{}) {
		stats.MissingBaselineCount = stats.MonitorCount
		stats.Estimated = true
		stats.Note = "单位账号快照表尚未迁移，无法计算今日上游实际消耗"
		return nil
	}
	var snapshots []UnitAccountMonitorSnapshot
	if err := DB.Model(&UnitAccountMonitorSnapshot{}).
		Where("created_time >= ? AND created_time <= ? AND platform_status = ?", start, end, "ok").
		Order("monitor_id ASC, created_time ASC, id ASC").
		Find(&snapshots).Error; err != nil {
		return err
	}
	stats.SnapshotCount = int64(len(snapshots))
	type pair struct {
		first UnitAccountMonitorSnapshot
		last  UnitAccountMonitorSnapshot
		count int
	}
	byMonitor := make(map[int]*pair)
	for _, snapshot := range snapshots {
		if snapshot.BalanceUnit != "" && snapshot.BalanceUnit != "USD" {
			continue
		}
		current := byMonitor[snapshot.MonitorID]
		if current == nil {
			byMonitor[snapshot.MonitorID] = &pair{first: snapshot, last: snapshot, count: 1}
			continue
		}
		current.last = snapshot
		current.count++
	}
	for _, item := range byMonitor {
		if item.count < 2 {
			continue
		}
		delta := item.last.UsedAmount - item.first.UsedAmount
		if delta < 0 {
			delta = 0
		}
		stats.UpstreamCostUSD += delta
		stats.BaselineMonitorCount++
	}
	if stats.MonitorCount > stats.BaselineMonitorCount {
		stats.MissingBaselineCount = stats.MonitorCount - stats.BaselineMonitorCount
	}
	return nil
}

func getNacpTrendStats(start int64, end int64, rangeHours int) ([]NacpTrendPoint, error) {
	if LOG_DB == nil {
		return []NacpTrendPoint{}, nil
	}
	bucketSeconds := nacpTrendBucketSeconds(rangeHours)
	alignedStart := start / bucketSeconds * bucketSeconds
	rows := []struct {
		BucketStart  int64
		RequestCount int64
		SuccessCount int64
		ErrorCount   int64
		ProbeCount   int64
		Tokens       int64
		Quota        int64
	}{}
	if err := LOG_DB.Model(&Log{}).
		Select(
			nacpTrendBucketExpr()+" AS bucket_start, COALESCE(SUM(CASE WHEN type IN ? THEN 1 ELSE 0 END),0) AS request_count, COALESCE(SUM(CASE WHEN type IN ? THEN 1 ELSE 0 END),0) AS success_count, COALESCE(SUM(CASE WHEN type IN ? THEN 1 ELSE 0 END),0) AS error_count, COALESCE(SUM(CASE WHEN type IN ? THEN 1 ELSE 0 END),0) AS probe_count, COALESCE(SUM(CASE WHEN type IN ? THEN prompt_tokens + completion_tokens ELSE 0 END),0) AS tokens, COALESCE(SUM(CASE WHEN type IN ? THEN quota ELSE 0 END),0) AS quota",
			bucketSeconds,
			bucketSeconds,
			nacpUserVisibleLogTypes(),
			nacpSuccessSummaryLogTypes(),
			nacpUserVisibleErrorTypes(),
			[]int{LogTypeProbeSuccess, LogTypeProbeFailed},
			nacpSuccessSummaryLogTypes(),
			nacpSuccessSummaryLogTypes(),
		).
		Where("created_at >= ? AND created_at <= ? AND type IN ?", alignedStart, end, nacpObservedLogTypes()).
		Group("bucket_start").
		Order("bucket_start ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	byBucket := make(map[int64]NacpTrendPoint, len(rows))
	for _, row := range rows {
		point := NacpTrendPoint{
			BucketStart:  row.BucketStart,
			RequestCount: row.RequestCount,
			SuccessCount: row.SuccessCount,
			ErrorCount:   row.ErrorCount,
			ProbeCount:   row.ProbeCount,
			Tokens:       row.Tokens,
			Quota:        row.Quota,
		}
		if point.RequestCount > 0 {
			point.SuccessRatePerm = point.SuccessCount * 1000 / point.RequestCount
		}
		byBucket[point.BucketStart] = point
	}
	points := make([]NacpTrendPoint, 0, int((end-alignedStart)/bucketSeconds)+1)
	for bucket := alignedStart; bucket <= end; bucket += bucketSeconds {
		if point, ok := byBucket[bucket]; ok {
			points = append(points, point)
			continue
		}
		points = append(points, NacpTrendPoint{BucketStart: bucket})
	}
	return points, nil
}

func getNacpModelActivityStats(start int64, end int64, limit int) ([]NacpModelActivityStats, error) {
	if LOG_DB == nil {
		return []NacpModelActivityStats{}, nil
	}
	var rows []NacpModelActivityStats
	err := LOG_DB.Model(&Log{}).
		Select("model_name, COUNT(*) AS request_count, COALESCE(SUM(CASE WHEN type IN ? THEN quota ELSE 0 END),0) AS quota, COALESCE(SUM(CASE WHEN type IN ? THEN prompt_tokens + completion_tokens ELSE 0 END),0) AS tokens", nacpSuccessSummaryLogTypes(), nacpSuccessSummaryLogTypes()).
		Where("created_at >= ? AND created_at <= ? AND model_name <> '' AND type IN ?", start, end, nacpUserVisibleLogTypes()).
		Group("model_name").
		Order("request_count DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func getNacpUserActivityStats(start int64, end int64, limit int) ([]NacpUserActivityStats, error) {
	if LOG_DB == nil {
		return []NacpUserActivityStats{}, nil
	}
	var rows []NacpUserActivityStats
	err := LOG_DB.Model(&Log{}).
		Select("user_id, username, COUNT(*) AS request_count, COALESCE(SUM(CASE WHEN type IN ? THEN quota ELSE 0 END),0) AS quota, COALESCE(SUM(CASE WHEN type IN ? THEN prompt_tokens + completion_tokens ELSE 0 END),0) AS tokens", nacpSuccessSummaryLogTypes(), nacpSuccessSummaryLogTypes()).
		Where("created_at >= ? AND created_at <= ? AND user_id > 0 AND type IN ?", start, end, nacpUserVisibleLogTypes()).
		Group("user_id, username").
		Order("request_count DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func getNacpChannelActivityStats(start int64, end int64, limit int) ([]NacpChannelActivityStats, error) {
	if LOG_DB == nil {
		return []NacpChannelActivityStats{}, nil
	}
	var rows []NacpChannelActivityStats
	if err := LOG_DB.Model(&Log{}).
		Select("channel_id, COUNT(*) AS request_count, COALESCE(SUM(CASE WHEN type IN ? THEN quota ELSE 0 END),0) AS quota, COALESCE(SUM(CASE WHEN type IN ? THEN prompt_tokens + completion_tokens ELSE 0 END),0) AS tokens", nacpSuccessSummaryLogTypes(), nacpSuccessSummaryLogTypes()).
		Where("created_at >= ? AND created_at <= ? AND channel_id > 0 AND type IN ?", start, end, nacpUserVisibleLogTypes()).
		Group("channel_id").
		Order("request_count DESC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	channelIds := make([]int, 0, len(rows))
	for _, row := range rows {
		channelIds = append(channelIds, row.ChannelId)
	}
	if len(channelIds) == 0 {
		return rows, nil
	}
	var channels []Channel
	if err := DB.Model(&Channel{}).
		Select("id, name").
		Where("id IN ?", channelIds).
		Find(&channels).Error; err != nil {
		return nil, err
	}
	channelNames := make(map[int]string, len(channels))
	for _, channel := range channels {
		channelNames[channel.Id] = channel.Name
	}
	for i := range rows {
		rows[i].ChannelName = channelNames[rows[i].ChannelId]
	}
	return rows, nil
}

func GetNacpModelRankStats(rangeHours int, page int, pageSize int, sort string) (*NacpModelRankPage, error) {
	start, end, normalizedHours := nacpStatsRange(rangeHours)
	page, pageSize = nacpStatsPage(page, pageSize)
	result := &NacpModelRankPage{
		NacpStatsRankMeta: nacpStatsRankMeta(0, page, pageSize, start, end, normalizedHours, sort),
		Items:             []NacpModelRankStats{},
	}
	if LOG_DB == nil {
		return result, nil
	}
	query := LOG_DB.Model(&Log{}).
		Where("created_at >= ? AND created_at <= ? AND model_name <> '' AND type IN ?", start, end, nacpUserVisibleLogTypes())
	if err := query.Distinct("model_name").Count(&result.Total).Error; err != nil {
		return nil, err
	}
	selectSQL, selectArgs := nacpStatsRankSelect("model_name")
	rows := []struct {
		NacpModelRankStats
		AverageUseTime float64
	}{}
	if err := query.
		Select(selectSQL, selectArgs...).
		Group("model_name").
		Order(nacpStatsRankOrder(sort)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	result.Items = make([]NacpModelRankStats, 0, len(rows))
	for _, row := range rows {
		item := row.NacpModelRankStats
		item.AverageUseTime = int64(row.AverageUseTime + 0.5)
		fillNacpRankDerived(&item.SuccessRatePerm, &item.StreamRatePerm, item.RequestCount, item.SuccessCount, item.StreamCount)
		result.Items = append(result.Items, item)
	}
	return result, nil
}

func GetNacpUserRankStats(rangeHours int, page int, pageSize int, sort string) (*NacpUserRankPage, error) {
	start, end, normalizedHours := nacpStatsRange(rangeHours)
	page, pageSize = nacpStatsPage(page, pageSize)
	result := &NacpUserRankPage{
		NacpStatsRankMeta: nacpStatsRankMeta(0, page, pageSize, start, end, normalizedHours, sort),
		Items:             []NacpUserRankStats{},
	}
	if LOG_DB == nil {
		return result, nil
	}
	query := LOG_DB.Model(&Log{}).
		Where("created_at >= ? AND created_at <= ? AND user_id > 0 AND type IN ?", start, end, nacpUserVisibleLogTypes())
	if err := query.Distinct("user_id").Count(&result.Total).Error; err != nil {
		return nil, err
	}
	selectSQL, selectArgs := nacpStatsRankSelect("user_id, MAX(username) AS username")
	rows := []struct {
		NacpUserRankStats
		AverageUseTime float64
	}{}
	if err := query.
		Select(selectSQL, selectArgs...).
		Group("user_id").
		Order(nacpStatsRankOrder(sort)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	result.Items = make([]NacpUserRankStats, 0, len(rows))
	for _, row := range rows {
		item := row.NacpUserRankStats
		item.AverageUseTime = int64(row.AverageUseTime + 0.5)
		fillNacpRankDerived(&item.SuccessRatePerm, &item.StreamRatePerm, item.RequestCount, item.SuccessCount, item.StreamCount)
		result.Items = append(result.Items, item)
	}
	return result, nil
}

func GetNacpChannelRankStats(rangeHours int, page int, pageSize int, sort string) (*NacpChannelRankPage, error) {
	start, end, normalizedHours := nacpStatsRange(rangeHours)
	page, pageSize = nacpStatsPage(page, pageSize)
	result := &NacpChannelRankPage{
		NacpStatsRankMeta: nacpStatsRankMeta(0, page, pageSize, start, end, normalizedHours, sort),
		Items:             []NacpChannelRankStats{},
	}
	if LOG_DB == nil {
		return result, nil
	}
	query := LOG_DB.Model(&Log{}).
		Where("created_at >= ? AND created_at <= ? AND channel_id > 0 AND type IN ?", start, end, nacpUserVisibleLogTypes())
	if err := query.Distinct("channel_id").Count(&result.Total).Error; err != nil {
		return nil, err
	}
	selectSQL, selectArgs := nacpStatsRankSelect("channel_id")
	rows := []struct {
		NacpChannelRankStats
		AverageUseTime float64
	}{}
	if err := query.
		Select(selectSQL, selectArgs...).
		Group("channel_id").
		Order(nacpStatsRankOrder(sort)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	result.Items = make([]NacpChannelRankStats, 0, len(rows))
	channelIds := make([]int, 0, len(rows))
	for _, row := range rows {
		item := row.NacpChannelRankStats
		item.AverageUseTime = int64(row.AverageUseTime + 0.5)
		fillNacpRankDerived(&item.SuccessRatePerm, &item.StreamRatePerm, item.RequestCount, item.SuccessCount, item.StreamCount)
		result.Items = append(result.Items, item)
		channelIds = append(channelIds, item.ChannelId)
	}
	if len(channelIds) == 0 {
		return result, nil
	}
	var channels []Channel
	if err := DB.Model(&Channel{}).
		Select("id, name, status, health_status, priority, weight, unit_id, unit_account_id").
		Where("id IN ?", channelIds).
		Find(&channels).Error; err != nil {
		return nil, err
	}
	channelInfo := make(map[int]Channel, len(channels))
	unitIds := make([]int, 0, len(channels))
	accountIds := make([]int, 0, len(channels))
	for _, channel := range channels {
		channelInfo[channel.Id] = channel
		if channel.UnitID > 0 {
			unitIds = append(unitIds, channel.UnitID)
		}
		if channel.UnitAccountID > 0 {
			accountIds = append(accountIds, channel.UnitAccountID)
		}
	}
	unitLabels, accountLabels, err := nacpLoadUnitAccountLabelsByIDs(unitIds, accountIds)
	if err != nil {
		return nil, err
	}
	for i := range result.Items {
		channel, ok := channelInfo[result.Items[i].ChannelId]
		if !ok {
			continue
		}
		result.Items[i].ChannelName = channel.Name
		result.Items[i].Status = channel.Status
		result.Items[i].HealthStatus = channel.HealthStatus
		result.Items[i].UnitID = channel.UnitID
		result.Items[i].UnitAccountID = channel.UnitAccountID
		if unitLabel, ok := unitLabels[channel.UnitID]; ok {
			result.Items[i].UnitName = unitLabel.Name
			result.Items[i].UnitType = unitLabel.Type
		}
		result.Items[i].UnitAccountName = accountLabels[channel.UnitAccountID]
		if channel.Priority != nil {
			result.Items[i].Priority = *channel.Priority
		}
		if channel.Weight != nil {
			result.Items[i].Weight = *channel.Weight
		}
	}
	return result, nil
}

func GetNacpDimensionRankStats(rangeHours int, dimension string, page int, pageSize int, sort string) (*NacpDimensionRankPage, error) {
	start, end, normalizedHours := nacpStatsRange(rangeHours)
	page, pageSize = nacpStatsPage(page, pageSize)
	dimension = nacpNormalizeStatsDimension(dimension)
	result := &NacpDimensionRankPage{
		NacpStatsRankMeta: nacpStatsRankMeta(0, page, pageSize, start, end, normalizedHours, sort),
		Dimension:         dimension,
		Items:             []NacpDimensionRankStats{},
	}
	if LOG_DB == nil {
		return result, nil
	}

	dimensionSQL, groupSQL, whereSQL, countSQL := nacpStatsDimensionSQL(dimension)
	query := LOG_DB.Model(&Log{}).
		Where("created_at >= ? AND created_at <= ? AND type IN ?", start, end, nacpUserVisibleLogTypes()).
		Where(whereSQL)
	if err := query.Distinct(countSQL).Count(&result.Total).Error; err != nil {
		return nil, err
	}

	selectSQL, selectArgs := nacpStatsRankSelect(dimensionSQL)
	rows := []struct {
		NacpDimensionRankStats
		AverageUseTime float64
	}{}
	if err := query.
		Select(selectSQL, selectArgs...).
		Group(groupSQL).
		Order(nacpStatsRankOrder(sort)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	result.Items = make([]NacpDimensionRankStats, 0, len(rows))
	for _, row := range rows {
		item := row.NacpDimensionRankStats
		item.AverageUseTime = int64(row.AverageUseTime + 0.5)
		if item.Label == "" {
			item.Label = item.DimensionKey
		}
		if item.DimensionKey == "" && item.DimensionID > 0 {
			item.DimensionKey = strconv.Itoa(item.DimensionID)
		}
		fillNacpRankDerived(&item.SuccessRatePerm, &item.StreamRatePerm, item.RequestCount, item.SuccessCount, item.StreamCount)
		result.Items = append(result.Items, item)
	}
	return result, nil
}

func GetNacpModelCoverageStats(rangeHours int, page int, pageSize int, sortKey string) (*NacpModelCoveragePage, error) {
	start, end, normalizedHours := nacpStatsRange(rangeHours)
	page, pageSize = nacpStatsPage(page, pageSize)
	result := &NacpModelCoveragePage{
		NacpStatsRankMeta: nacpStatsRankMeta(0, page, pageSize, start, end, normalizedHours, sortKey),
		Items:             []NacpModelCoverageStats{},
	}

	var abilities []Ability
	if err := DB.Model(&Ability{}).
		Select(commonGroupCol + ", model, channel_id, enabled, priority, weight").
		Where("model <> ''").
		Find(&abilities).Error; err != nil {
		return nil, err
	}
	if len(abilities) == 0 {
		return result, nil
	}

	channelIDs := make([]int, 0, len(abilities))
	seenChannelIDs := map[int]struct{}{}
	for _, ability := range abilities {
		if ability.ChannelId <= 0 {
			continue
		}
		if _, ok := seenChannelIDs[ability.ChannelId]; ok {
			continue
		}
		seenChannelIDs[ability.ChannelId] = struct{}{}
		channelIDs = append(channelIDs, ability.ChannelId)
	}

	channelInfo := map[int]Channel{}
	if len(channelIDs) > 0 {
		var channels []Channel
		if err := DB.Model(&Channel{}).
			Select("id, name, status, health_status, priority, weight, unit_id, unit_account_id").
			Where("id IN ?", channelIDs).
			Find(&channels).Error; err != nil {
			return nil, err
		}
		for _, channel := range channels {
			channelInfo[channel.Id] = channel
		}
	}

	logStats, err := getNacpModelCoverageLogStats(start, end)
	if err != nil {
		return nil, err
	}

	type coverageAccumulator struct {
		item          NacpModelCoverageStats
		groups        map[string]struct{}
		enabledGroups map[string]struct{}
		channels      map[int]struct{}
		enabled       map[int]struct{}
		healthy       map[int]struct{}
		degraded      map[int]struct{}
		disabled      map[int]struct{}
	}
	itemsByModel := map[string]*coverageAccumulator{}
	for _, ability := range abilities {
		if ability.Model == "" {
			continue
		}
		acc, ok := itemsByModel[ability.Model]
		if !ok {
			acc = &coverageAccumulator{
				item: NacpModelCoverageStats{
					ModelName:      ability.Model,
					RiskLevel:      "ok",
					RiskReason:     "覆盖正常",
					Groups:         []string{},
					EnabledGroups:  []string{},
					SampleChannels: []NacpModelCoverageChannel{},
				},
				groups:        map[string]struct{}{},
				enabledGroups: map[string]struct{}{},
				channels:      map[int]struct{}{},
				enabled:       map[int]struct{}{},
				healthy:       map[int]struct{}{},
				degraded:      map[int]struct{}{},
				disabled:      map[int]struct{}{},
			}
			itemsByModel[ability.Model] = acc
		}

		if ability.Group != "" {
			acc.groups[ability.Group] = struct{}{}
		}
		acc.channels[ability.ChannelId] = struct{}{}

		channel, hasChannel := channelInfo[ability.ChannelId]
		enabled := ability.Enabled && hasChannel && channel.Status == common.ChannelStatusEnabled
		if enabled {
			acc.enabled[ability.ChannelId] = struct{}{}
			if ability.Group != "" {
				acc.enabledGroups[ability.Group] = struct{}{}
			}
			switch channel.HealthStatus {
			case common.ChannelHealthStatusHealthy, common.ChannelHealthStatusRecovering, "":
				acc.healthy[ability.ChannelId] = struct{}{}
			case common.ChannelHealthStatusDegraded, common.ChannelHealthStatusProbing:
				acc.degraded[ability.ChannelId] = struct{}{}
			default:
				acc.disabled[ability.ChannelId] = struct{}{}
			}
		} else {
			acc.disabled[ability.ChannelId] = struct{}{}
		}

		if hasChannel && len(acc.item.SampleChannels) < 10 {
			sample := NacpModelCoverageChannel{
				ChannelId:    channel.Id,
				ChannelName:  channel.Name,
				Group:        ability.Group,
				Status:       channel.Status,
				HealthStatus: channel.HealthStatus,
				Weight:       ability.Weight,
				UnitID:       channel.UnitID,
				AccountID:    channel.UnitAccountID,
			}
			if ability.Priority != nil {
				sample.Priority = *ability.Priority
			}
			acc.item.SampleChannels = append(acc.item.SampleChannels, sample)
		}
	}

	items := make([]NacpModelCoverageStats, 0, len(itemsByModel))
	for modelName, acc := range itemsByModel {
		item := acc.item
		item.GroupCount = int64(len(acc.groups))
		item.EnabledGroupCount = int64(len(acc.enabledGroups))
		item.ChannelCount = int64(len(acc.channels))
		item.EnabledChannelCount = int64(len(acc.enabled))
		item.HealthyChannelCount = int64(len(acc.healthy))
		item.DegradedChannelCount = int64(len(acc.degraded))
		item.DisabledChannelCount = int64(len(acc.disabled))
		if item.ChannelCount > 0 {
			item.CoverageRatePerm = item.EnabledChannelCount * 1000 / item.ChannelCount
		}
		if item.EnabledChannelCount > 0 {
			item.HealthyRatePerm = item.HealthyChannelCount * 1000 / item.EnabledChannelCount
		}
		item.Groups = nacpSortedStringSet(acc.groups)
		item.EnabledGroups = nacpSortedStringSet(acc.enabledGroups)
		if stats, ok := logStats[modelName]; ok {
			item.RequestCount = stats.RequestCount
			item.SuccessCount = stats.SuccessCount
			item.ErrorCount = stats.ErrorCount
			item.Tokens = stats.Tokens
			item.Quota = stats.Quota
			item.AverageUseTime = stats.AverageUseTime
			item.SuccessRatePerm = stats.SuccessRatePerm
		}
		item.RiskLevel, item.RiskReason = nacpModelCoverageRisk(item)
		items = append(items, item)
	}

	nacpSortModelCoverageItems(items, sortKey)
	result.Total = int64(len(items))
	startIndex := (page - 1) * pageSize
	if startIndex >= len(items) {
		return result, nil
	}
	endIndex := startIndex + pageSize
	if endIndex > len(items) {
		endIndex = len(items)
	}
	result.Items = items[startIndex:endIndex]
	return result, nil
}

func buildNacpOperationalHealth(unitMonitors NacpUnitMonitorStats, channels NacpChannelStats, traffic NacpTrafficStats, cost NacpCostStats) NacpOperationalHealth {
	trafficPerm := traffic.SuccessRatePerm
	if traffic.RequestCount == 0 {
		trafficPerm = 1000
	}
	unitPerm := int64(1000)
	if unitMonitors.Total > 0 {
		unitPerm = unitMonitors.OK * 1000 / unitMonitors.Total
	}
	channelPerm := int64(1000)
	if channels.Total > 0 {
		channelPerm = channels.Healthy * 1000 / channels.Total
	}
	costPerm := int64(1000)
	if cost.MonitorCount > 0 {
		costPerm = cost.BaselineMonitorCount * 1000 / cost.MonitorCount
	}
	score := (trafficPerm*45 + unitPerm*20 + channelPerm*20 + costPerm*15) / 100
	health := NacpOperationalHealth{
		ScorePerm:          score,
		Level:              "healthy",
		TrafficSuccessPerm: trafficPerm,
		UnitHealthPerm:     unitPerm,
		ChannelHealthPerm:  channelPerm,
		CostCoveragePerm:   costPerm,
		Notes:              []string{},
	}
	if score < 600 {
		health.Level = "critical"
	} else if score < 800 {
		health.Level = "warning"
	}
	if traffic.UserVisibleErrors > 0 {
		health.Notes = append(health.Notes, "当前范围存在用户可见错误")
	}
	if unitMonitors.Error > 0 {
		health.Notes = append(health.Notes, "存在异常单位账号统计源")
	}
	if channels.Unhealthy > 0 {
		health.Notes = append(health.Notes, "存在异常渠道")
	}
	if cost.MissingBaselineCount > 0 {
		health.Notes = append(health.Notes, "今日成本快照基线不完整")
	}
	return health
}

func getNacpModelCoverageLogStats(start int64, end int64) (map[string]NacpModelCoverageStats, error) {
	result := map[string]NacpModelCoverageStats{}
	if LOG_DB == nil {
		return result, nil
	}
	selectSQL, selectArgs := nacpStatsRankSelect("model_name")
	rows := []struct {
		NacpModelCoverageStats
		AverageUseTime float64
	}{}
	if err := LOG_DB.Model(&Log{}).
		Select(selectSQL, selectArgs...).
		Where("created_at >= ? AND created_at <= ? AND model_name <> '' AND type IN ?", start, end, nacpUserVisibleLogTypes()).
		Group("model_name").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		item := row.NacpModelCoverageStats
		item.AverageUseTime = int64(row.AverageUseTime + 0.5)
		fillNacpRankDerived(&item.SuccessRatePerm, nil, item.RequestCount, item.SuccessCount, 0)
		result[item.ModelName] = item
	}
	return result, nil
}

func nacpSortedStringSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func nacpModelCoverageRisk(item NacpModelCoverageStats) (string, string) {
	switch {
	case item.ChannelCount == 0:
		return "critical", "模型没有任何渠道覆盖"
	case item.EnabledChannelCount == 0:
		return "critical", "模型没有启用渠道"
	case item.HealthyChannelCount == 0:
		return "warning", "模型有启用渠道，但没有健康渠道"
	case item.RequestCount > 0 && item.SuccessRatePerm < 800:
		return "warning", "最近请求成功率偏低"
	case item.RequestCount == 0:
		return "idle", "当前统计范围内暂无真实请求"
	default:
		return "ok", "覆盖正常"
	}
}

func nacpSortModelCoverageItems(items []NacpModelCoverageStats, sortKey string) {
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		switch sortKey {
		case "healthy":
			if left.HealthyRatePerm == right.HealthyRatePerm {
				return left.EnabledChannelCount > right.EnabledChannelCount
			}
			return left.HealthyRatePerm < right.HealthyRatePerm
		case "coverage":
			if left.CoverageRatePerm == right.CoverageRatePerm {
				return left.ChannelCount > right.ChannelCount
			}
			return left.CoverageRatePerm < right.CoverageRatePerm
		case "channels":
			if left.EnabledChannelCount == right.EnabledChannelCount {
				return left.ChannelCount > right.ChannelCount
			}
			return left.EnabledChannelCount < right.EnabledChannelCount
		case "errors":
			return left.ErrorCount > right.ErrorCount
		case "latency":
			return left.AverageUseTime > right.AverageUseTime
		case "tokens":
			return left.Tokens > right.Tokens
		case "quota":
			return left.Quota > right.Quota
		default:
			return left.RequestCount > right.RequestCount
		}
	})
}

func GetNacpCostRankStats(dimension string, page int, pageSize int, sortKey string) (*NacpCostRankPage, error) {
	page, pageSize = nacpStatsPage(page, pageSize)
	start, end := nacpTodayRange()
	dimension = nacpNormalizeCostDimension(dimension)
	result := &NacpCostRankPage{
		NacpStatsRankMeta: nacpStatsRankMeta(0, page, pageSize, start, end, 24, sortKey),
		Dimension:         dimension,
		DayStart:          start,
		DayEnd:            end,
		Items:             []NacpCostRankStats{},
	}
	monitorCosts, accountCosts, totalUpstreamCost, hasMissingBaseline, err := getNacpMonitorCostRankMaps(start, end)
	if err != nil {
		return nil, err
	}
	var items []NacpCostRankStats
	switch dimension {
	case "unit":
		items, err = getNacpUnitCostRankStats(start, end, monitorCosts, accountCosts)
	case "channel":
		items, err = getNacpChannelCostRankStats(start, end, accountCosts)
	case "model":
		items, err = getNacpGenericCostRankStats(start, end, "model", totalUpstreamCost, hasMissingBaseline)
	case "user":
		items, err = getNacpGenericCostRankStats(start, end, "user", totalUpstreamCost, hasMissingBaseline)
	}
	if err != nil {
		return nil, err
	}
	nacpSortCostRankItems(items, sortKey)
	result.Total = int64(len(items))
	startIndex := (page - 1) * pageSize
	if startIndex >= len(items) {
		return result, nil
	}
	endIndex := startIndex + pageSize
	if endIndex > len(items) {
		endIndex = len(items)
	}
	result.Items = items[startIndex:endIndex]
	return result, nil
}

func nacpNormalizeCostDimension(dimension string) string {
	switch dimension {
	case "channel", "model", "user":
		return dimension
	default:
		return "unit"
	}
}

func getNacpUnitCostRankStats(start int64, end int64, monitorCosts map[int]NacpCostRankStats, accountCosts map[string]NacpCostRankStats) ([]NacpCostRankStats, error) {
	items := make([]NacpCostRankStats, 0, len(monitorCosts))
	for _, item := range monitorCosts {
		items = append(items, item)
	}
	channelRevenue, err := getNacpChannelRevenueRows(start, end)
	if err != nil {
		return nil, err
	}
	channelInfo, err := getNacpChannelInfoMap(nacpRevenueChannelIds(channelRevenue))
	if err != nil {
		return nil, err
	}
	unitLabels, accountLabels, err := nacpLoadUnitAccountLabelsFromChannels(channelInfo)
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]int, len(items))
	for i := range items {
		key := nacpUnitAccountCostKey(items[i].UnitID, items[i].UnitAccountID)
		byKey[key] = i
	}
	for _, row := range channelRevenue {
		channel, ok := channelInfo[row.ChannelId]
		if !ok {
			continue
		}
		key := nacpUnitAccountCostKey(channel.UnitID, channel.UnitAccountID)
		itemIndex, ok := byKey[key]
		if !ok {
			placeholder := accountCosts[key]
			placeholder.Key = key
			placeholder.Dimension = "unit"
			placeholder.UnitID = channel.UnitID
			placeholder.UnitAccountID = channel.UnitAccountID
			if unitLabel, ok := unitLabels[channel.UnitID]; ok {
				placeholder.UnitName = unitLabel.Name
				placeholder.UnitType = unitLabel.Type
			}
			placeholder.Account = accountLabels[channel.UnitAccountID]
			placeholder.Label = nacpUnitAccountCostLabel(placeholder.UnitName, placeholder.Account, "", key)
			placeholder.Estimated = true
			placeholder.CostSource = "missing_monitor"
			placeholder.MissingSnapshotReason = "渠道关联的单位账号没有统计源或今日没有可用快照"
			items = append(items, placeholder)
			itemIndex = len(items) - 1
			byKey[key] = itemIndex
		}
		nacpApplyRevenueToCostItem(&items[itemIndex], row.RequestCount, row.Tokens, row.Quota)
	}
	for i := range items {
		nacpFinalizeCostItem(&items[i])
	}
	return items, nil
}

func getNacpChannelCostRankStats(start int64, end int64, accountCosts map[string]NacpCostRankStats) ([]NacpCostRankStats, error) {
	revenueRows, err := getNacpChannelRevenueRows(start, end)
	if err != nil {
		return nil, err
	}
	channelIds := nacpRevenueChannelIds(revenueRows)
	channelInfo, err := getNacpChannelInfoMap(channelIds)
	if err != nil {
		return nil, err
	}
	unitLabels, accountLabels, err := nacpLoadUnitAccountLabelsFromChannels(channelInfo)
	if err != nil {
		return nil, err
	}
	accountQuota := make(map[string]int64)
	for _, row := range revenueRows {
		channel, ok := channelInfo[row.ChannelId]
		if !ok {
			continue
		}
		accountQuota[nacpUnitAccountCostKey(channel.UnitID, channel.UnitAccountID)] += row.Quota
	}
	items := make([]NacpCostRankStats, 0, len(revenueRows))
	for _, row := range revenueRows {
		channel, ok := channelInfo[row.ChannelId]
		if !ok {
			continue
		}
		key := nacpUnitAccountCostKey(channel.UnitID, channel.UnitAccountID)
		accountCost := accountCosts[key]
		unitLabel := unitLabels[channel.UnitID]
		item := NacpCostRankStats{
			Key:                 strconv.Itoa(row.ChannelId),
			Label:               channel.Name,
			Dimension:           "channel",
			ChannelId:           row.ChannelId,
			ChannelName:         channel.Name,
			ChannelStatus:       channel.Status,
			ChannelHealthStatus: channel.HealthStatus,
			UnitID:              channel.UnitID,
			UnitName:            unitLabel.Name,
			UnitType:            unitLabel.Type,
			UnitAccountID:       channel.UnitAccountID,
			Account:             accountLabels[channel.UnitAccountID],
			RequestCount:        row.RequestCount,
			Tokens:              row.Tokens,
			Quota:               row.Quota,
			CurrentBalance:      accountCost.CurrentBalance,
			UsedAmount:          accountCost.UsedAmount,
			BalanceUnit:         accountCost.BalanceUnit,
			BaselineReady:       accountCost.BaselineReady,
			Estimated:           true,
			CostSource:          "allocated_by_quota",
			SnapshotCount:       accountCost.SnapshotCount,
			LastCheckedTime:     accountCost.LastCheckedTime,
			PlatformStatus:      accountCost.PlatformStatus,
		}
		if channel.Priority != nil {
			item.Priority = *channel.Priority
		}
		if channel.Weight != nil {
			item.Weight = *channel.Weight
		}
		if accountQuota[key] > 0 {
			item.UpstreamCostUSD = accountCost.UpstreamCostUSD * float64(row.Quota) / float64(accountQuota[key])
		} else if accountCost.UpstreamCostUSD > 0 {
			item.MissingSnapshotReason = "该单位账号今日有上游消耗，但渠道成功消费额度为 0，无法分摊"
		}
		nacpFinalizeCostItem(&item)
		items = append(items, item)
	}
	return items, nil
}

func getNacpGenericCostRankStats(start int64, end int64, dimension string, totalUpstreamCost float64, hasMissingBaseline bool) ([]NacpCostRankStats, error) {
	rows, err := getNacpDimensionRevenueRows(start, end, dimension)
	if err != nil {
		return nil, err
	}
	var totalQuota int64
	for _, row := range rows {
		totalQuota += row.Quota
	}
	items := make([]NacpCostRankStats, 0, len(rows))
	for _, row := range rows {
		item := NacpCostRankStats{
			Key:           row.Key,
			Label:         row.Label,
			Dimension:     dimension,
			RequestCount:  row.RequestCount,
			Tokens:        row.Tokens,
			Quota:         row.Quota,
			Estimated:     true,
			CostSource:    "allocated_by_quota",
			BaselineReady: !hasMissingBaseline,
		}
		if dimension == "model" {
			item.ModelName = row.Key
		} else {
			item.UserId = row.IntKey
			item.Username = row.Label
		}
		if totalQuota > 0 {
			item.UpstreamCostUSD = totalUpstreamCost * float64(row.Quota) / float64(totalQuota)
		} else if totalUpstreamCost > 0 {
			item.MissingSnapshotReason = "今日有上游成本，但没有成功消费额度可用于分摊"
		}
		nacpFinalizeCostItem(&item)
		items = append(items, item)
	}
	return items, nil
}

type nacpChannelRevenueRow struct {
	ChannelId    int
	RequestCount int64
	Tokens       int64
	Quota        int64
}

type nacpDimensionRevenueRow struct {
	Key          string
	Label        string
	IntKey       int
	RequestCount int64
	Tokens       int64
	Quota        int64
}

func getNacpMonitorCostRankMaps(start int64, end int64) (map[int]NacpCostRankStats, map[string]NacpCostRankStats, float64, bool, error) {
	monitorCosts := map[int]NacpCostRankStats{}
	accountCosts := map[string]NacpCostRankStats{}
	var monitors []UnitAccountMonitor
	if err := DB.Model(&UnitAccountMonitor{}).
		Where("status = ?", UnitMonitorStatusEnabled).
		Find(&monitors).Error; err != nil {
		return monitorCosts, accountCosts, 0, false, err
	}
	unitLabels, accountNames, err := nacpLoadUnitAccountLabels(monitors)
	if err != nil {
		return monitorCosts, accountCosts, 0, false, err
	}
	type pair struct {
		first UnitAccountMonitorSnapshot
		last  UnitAccountMonitorSnapshot
		count int64
	}
	snapshotPairs := map[int]*pair{}
	if DB.Migrator().HasTable(&UnitAccountMonitorSnapshot{}) {
		var snapshots []UnitAccountMonitorSnapshot
		if err := DB.Model(&UnitAccountMonitorSnapshot{}).
			Where("created_time >= ? AND created_time <= ? AND platform_status = ?", start, end, "ok").
			Order("monitor_id ASC, created_time ASC, id ASC").
			Find(&snapshots).Error; err != nil {
			return monitorCosts, accountCosts, 0, false, err
		}
		for _, snapshot := range snapshots {
			if snapshot.BalanceUnit != "" && snapshot.BalanceUnit != "USD" {
				continue
			}
			current := snapshotPairs[snapshot.MonitorID]
			if current == nil {
				snapshotPairs[snapshot.MonitorID] = &pair{first: snapshot, last: snapshot, count: 1}
				continue
			}
			current.last = snapshot
			current.count++
		}
	}
	var totalUpstreamCost float64
	hasMissingBaseline := false
	for _, monitor := range monitors {
		unitLabel := unitLabels[monitor.UnitID]
		unitName := unitLabel.Name
		accountName := accountNames[monitor.UnitAccountID]
		key := nacpUnitAccountCostKey(monitor.UnitID, monitor.UnitAccountID)
		item := NacpCostRankStats{
			Key:                   key,
			Label:                 nacpUnitAccountCostLabel(unitName, accountName, monitor.Name, key),
			Dimension:             "unit",
			UnitID:                monitor.UnitID,
			UnitName:              unitName,
			UnitType:              unitLabel.Type,
			UnitAccountID:         monitor.UnitAccountID,
			Account:               accountName,
			MonitorID:             monitor.Id,
			CurrentBalance:        monitor.CurrentBalance,
			UsedAmount:            monitor.UsedAmount,
			BalanceUnit:           monitor.BalanceUnit,
			LastCheckedTime:       monitor.LastCheckedTime,
			PlatformStatus:        monitor.PlatformStatus,
			CostSource:            "snapshot_delta",
			MissingSnapshotReason: "今日成功快照不足两次，无法精确计算上游消耗差值",
			Estimated:             true,
		}
		if snapshotPair := snapshotPairs[monitor.Id]; snapshotPair != nil {
			item.SnapshotCount = snapshotPair.count
			item.FirstSnapshotTime = snapshotPair.first.CreatedTime
			item.LastSnapshotTime = snapshotPair.last.CreatedTime
			if snapshotPair.count >= 2 {
				item.BaselineReady = true
				item.Estimated = false
				item.MissingSnapshotReason = ""
				item.UpstreamCostUSD = snapshotPair.last.UsedAmount - snapshotPair.first.UsedAmount
				if item.UpstreamCostUSD < 0 {
					item.UpstreamCostUSD = 0
				}
				totalUpstreamCost += item.UpstreamCostUSD
			}
		}
		if !item.BaselineReady {
			hasMissingBaseline = true
		}
		monitorCosts[monitor.Id] = item
		accountCosts[key] = item
	}
	return monitorCosts, accountCosts, totalUpstreamCost, hasMissingBaseline, nil
}

type nacpUnitLabel struct {
	Name string
	Type string
}

func nacpLoadUnitAccountLabels(monitors []UnitAccountMonitor) (map[int]nacpUnitLabel, map[int]string, error) {
	unitIds := make([]int, 0, len(monitors))
	accountIds := make([]int, 0, len(monitors))
	for _, monitor := range monitors {
		if monitor.UnitID > 0 {
			unitIds = append(unitIds, monitor.UnitID)
		}
		if monitor.UnitAccountID > 0 {
			accountIds = append(accountIds, monitor.UnitAccountID)
		}
	}
	return nacpLoadUnitAccountLabelsByIDs(unitIds, accountIds)
}

func nacpLoadUnitAccountLabelsFromChannels(channels map[int]Channel) (map[int]nacpUnitLabel, map[int]string, error) {
	unitIds := make([]int, 0, len(channels))
	accountIds := make([]int, 0, len(channels))
	for _, channel := range channels {
		if channel.UnitID > 0 {
			unitIds = append(unitIds, channel.UnitID)
		}
		if channel.UnitAccountID > 0 {
			accountIds = append(accountIds, channel.UnitAccountID)
		}
	}
	return nacpLoadUnitAccountLabelsByIDs(unitIds, accountIds)
}

func nacpLoadUnitAccountLabelsByIDs(unitIds []int, accountIds []int) (map[int]nacpUnitLabel, map[int]string, error) {
	unitLabels := map[int]nacpUnitLabel{}
	accountNames := map[int]string{}
	if len(unitIds) > 0 {
		var units []Unit
		if err := DB.Model(&Unit{}).Select("id, name, type").Where("id IN ?", unitIds).Find(&units).Error; err != nil {
			return unitLabels, accountNames, err
		}
		for _, unit := range units {
			unitLabels[unit.Id] = nacpUnitLabel{Name: unit.Name, Type: unit.Type}
		}
	}
	if len(accountIds) > 0 {
		var accounts []UnitAccount
		if err := DB.Model(&UnitAccount{}).Select("id, account, account_id").Where("id IN ?", accountIds).Find(&accounts).Error; err != nil {
			return unitLabels, accountNames, err
		}
		for _, account := range accounts {
			accountNames[account.Id] = account.Account
			if accountNames[account.Id] == "" {
				accountNames[account.Id] = account.AccountID
			}
		}
	}
	return unitLabels, accountNames, nil
}

func getNacpChannelRevenueRows(start int64, end int64) ([]nacpChannelRevenueRow, error) {
	if LOG_DB == nil {
		return []nacpChannelRevenueRow{}, nil
	}
	var rows []nacpChannelRevenueRow
	err := LOG_DB.Model(&Log{}).
		Select("channel_id, COUNT(*) AS request_count, COALESCE(SUM(prompt_tokens + completion_tokens),0) AS tokens, COALESCE(SUM(quota),0) AS quota").
		Where("created_at >= ? AND created_at <= ? AND channel_id > 0 AND type IN ?", start, end, nacpSuccessSummaryLogTypes()).
		Group("channel_id").
		Scan(&rows).Error
	return rows, err
}

func getNacpDimensionRevenueRows(start int64, end int64, dimension string) ([]nacpDimensionRevenueRow, error) {
	if LOG_DB == nil {
		return []nacpDimensionRevenueRow{}, nil
	}
	if dimension == "model" {
		rows := []struct {
			ModelName    string
			RequestCount int64
			Tokens       int64
			Quota        int64
		}{}
		if err := LOG_DB.Model(&Log{}).
			Select("model_name, COUNT(*) AS request_count, COALESCE(SUM(prompt_tokens + completion_tokens),0) AS tokens, COALESCE(SUM(quota),0) AS quota").
			Where("created_at >= ? AND created_at <= ? AND model_name <> '' AND type IN ?", start, end, nacpSuccessSummaryLogTypes()).
			Group("model_name").
			Scan(&rows).Error; err != nil {
			return nil, err
		}
		result := make([]nacpDimensionRevenueRow, 0, len(rows))
		for _, row := range rows {
			result = append(result, nacpDimensionRevenueRow{
				Key:          row.ModelName,
				Label:        row.ModelName,
				RequestCount: row.RequestCount,
				Tokens:       row.Tokens,
				Quota:        row.Quota,
			})
		}
		return result, nil
	}
	rows := []struct {
		UserId       int
		Username     string
		RequestCount int64
		Tokens       int64
		Quota        int64
	}{}
	if err := LOG_DB.Model(&Log{}).
		Select("user_id, MAX(username) AS username, COUNT(*) AS request_count, COALESCE(SUM(prompt_tokens + completion_tokens),0) AS tokens, COALESCE(SUM(quota),0) AS quota").
		Where("created_at >= ? AND created_at <= ? AND user_id > 0 AND type IN ?", start, end, nacpSuccessSummaryLogTypes()).
		Group("user_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]nacpDimensionRevenueRow, 0, len(rows))
	for _, row := range rows {
		label := row.Username
		if label == "" {
			label = "#" + strconv.Itoa(row.UserId)
		}
		result = append(result, nacpDimensionRevenueRow{
			Key:          strconv.Itoa(row.UserId),
			Label:        label,
			IntKey:       row.UserId,
			RequestCount: row.RequestCount,
			Tokens:       row.Tokens,
			Quota:        row.Quota,
		})
	}
	return result, nil
}

func getNacpChannelInfoMap(channelIds []int) (map[int]Channel, error) {
	result := map[int]Channel{}
	if len(channelIds) == 0 {
		return result, nil
	}
	var channels []Channel
	if err := DB.Model(&Channel{}).
		Select("id, name, status, health_status, priority, weight, unit_id, unit_account_id").
		Where("id IN ?", channelIds).
		Find(&channels).Error; err != nil {
		return result, err
	}
	for _, channel := range channels {
		result[channel.Id] = channel
	}
	return result, nil
}

func nacpRevenueChannelIds(rows []nacpChannelRevenueRow) []int {
	seen := map[int]bool{}
	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		if row.ChannelId <= 0 || seen[row.ChannelId] {
			continue
		}
		seen[row.ChannelId] = true
		ids = append(ids, row.ChannelId)
	}
	return ids
}

func nacpUnitAccountCostKey(unitId int, unitAccountId int) string {
	return strconv.Itoa(unitId) + ":" + strconv.Itoa(unitAccountId)
}

func nacpUnitAccountCostLabel(unitName string, accountName string, monitorName string, fallback string) string {
	if unitName != "" && accountName != "" {
		return unitName + " / " + accountName
	}
	if monitorName != "" {
		return monitorName
	}
	return fallback
}

func nacpApplyRevenueToCostItem(item *NacpCostRankStats, requestCount int64, tokens int64, quota int64) {
	if item == nil {
		return
	}
	item.RequestCount += requestCount
	item.Tokens += tokens
	item.Quota += quota
}

func nacpFinalizeCostItem(item *NacpCostRankStats) {
	if item == nil {
		return
	}
	item.PlatformRevenueUSD = nacpQuotaToUSD(item.Quota)
	item.EstimatedProfitUSD = item.PlatformRevenueUSD - item.UpstreamCostUSD
	if item.PlatformRevenueUSD > 0 {
		item.GrossMarginPerm = int64((item.EstimatedProfitUSD / item.PlatformRevenueUSD) * 1000)
	}
}

func nacpQuotaToUSD(quota int64) float64 {
	if common.QuotaPerUnit <= 0 {
		return 0
	}
	return float64(quota) / common.QuotaPerUnit
}

func nacpSortCostRankItems(items []NacpCostRankStats, sortKey string) {
	sort.SliceStable(items, func(i int, j int) bool {
		left := items[i]
		right := items[j]
		switch sortKey {
		case "cost":
			if left.UpstreamCostUSD != right.UpstreamCostUSD {
				return left.UpstreamCostUSD > right.UpstreamCostUSD
			}
		case "profit":
			if left.EstimatedProfitUSD != right.EstimatedProfitUSD {
				return left.EstimatedProfitUSD > right.EstimatedProfitUSD
			}
		case "margin":
			if left.GrossMarginPerm != right.GrossMarginPerm {
				return left.GrossMarginPerm > right.GrossMarginPerm
			}
		case "tokens":
			if left.Tokens != right.Tokens {
				return left.Tokens > right.Tokens
			}
		case "quota":
			if left.Quota != right.Quota {
				return left.Quota > right.Quota
			}
		case "requests":
			if left.RequestCount != right.RequestCount {
				return left.RequestCount > right.RequestCount
			}
		default:
			if left.PlatformRevenueUSD != right.PlatformRevenueUSD {
				return left.PlatformRevenueUSD > right.PlatformRevenueUSD
			}
		}
		if left.RequestCount != right.RequestCount {
			return left.RequestCount > right.RequestCount
		}
		return left.Label < right.Label
	})
}

func nacpStatsRange(rangeHours int) (int64, int64, int) {
	if rangeHours <= 0 {
		rangeHours = 24
	}
	if rangeHours > 24*30 {
		rangeHours = 24 * 30
	}
	end := time.Now().Unix()
	start := end - int64(rangeHours)*60*60
	return start, end, rangeHours
}

func nacpTodayRange() (int64, int64) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return start.Unix(), now.Unix()
}

func nacpStatsPage(page int, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func nacpStatsRankMeta(total int64, page int, pageSize int, start int64, end int64, rangeHours int, sort string) NacpStatsRankMeta {
	if sort == "" {
		sort = "requests"
	}
	return NacpStatsRankMeta{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		RangeStart: start,
		RangeEnd:   end,
		RangeHours: rangeHours,
		Sort:       sort,
	}
}

func nacpStatsRankSelect(prefix string) (string, []interface{}) {
	return prefix + ", COUNT(*) AS request_count, " +
			"COALESCE(SUM(CASE WHEN type IN ? THEN 1 ELSE 0 END),0) AS success_count, " +
			"COALESCE(SUM(CASE WHEN type IN ? THEN 1 ELSE 0 END),0) AS error_count, " +
			"COALESCE(SUM(CASE WHEN type = ? THEN 1 ELSE 0 END),0) AS direct_success_count, " +
			"COALESCE(SUM(CASE WHEN type = ? THEN 1 ELSE 0 END),0) AS retry_success_count, " +
			"COALESCE(SUM(CASE WHEN type = ? THEN 1 ELSE 0 END),0) AS retry_failed_count, " +
			"COALESCE(SUM(CASE WHEN type = ? THEN 1 ELSE 0 END),0) AS legacy_error_count, " +
			"COALESCE(SUM(CASE WHEN type IN ? AND is_stream = ? THEN 1 ELSE 0 END),0) AS stream_count, " +
			"COALESCE(SUM(CASE WHEN type IN ? THEN prompt_tokens + completion_tokens ELSE 0 END),0) AS tokens, " +
			"COALESCE(SUM(CASE WHEN type IN ? THEN quota ELSE 0 END),0) AS quota, " +
			"COALESCE(AVG(CASE WHEN type IN ? AND use_time > 0 THEN use_time ELSE NULL END),0) AS average_use_time",
		[]interface{}{
			nacpSuccessSummaryLogTypes(),
			nacpUserVisibleErrorTypes(),
			LogTypeConsume,
			LogTypeRetrySuccessSummary,
			LogTypeRetryFailedSummary,
			LogTypeError,
			nacpUserVisibleLogTypes(),
			true,
			nacpSuccessSummaryLogTypes(),
			nacpSuccessSummaryLogTypes(),
			nacpSuccessSummaryLogTypes(),
		}
}

func nacpStatsRankOrder(sort string) string {
	switch sort {
	case "tokens":
		return "tokens DESC"
	case "quota":
		return "quota DESC"
	case "errors":
		return "error_count DESC"
	case "latency":
		return "average_use_time DESC"
	case "success":
		return "success_count DESC"
	default:
		return "request_count DESC"
	}
}

func nacpNormalizeStatsDimension(dimension string) string {
	switch dimension {
	case "token", "endpoint", "ip":
		return dimension
	default:
		return "group"
	}
}

func nacpStatsDimensionSQL(dimension string) (selectPrefix string, groupSQL string, whereSQL string, countSQL string) {
	switch dimension {
	case "token":
		return "token_id AS dimension_id, MAX(token_name) AS label",
			"token_id",
			"token_id > 0",
			"token_id"
	case "endpoint":
		return "content AS dimension_key, content AS label",
			"content",
			"content <> ''",
			"content"
	case "ip":
		return "ip AS dimension_key, ip AS label",
			"ip",
			"ip <> ''",
			"ip"
	default:
		return logGroupCol + " AS dimension_key, " + logGroupCol + " AS label",
			logGroupCol,
			logGroupCol + " <> ''",
			logGroupCol
	}
}

func fillNacpRankDerived(successRatePerm *int64, streamRatePerm *int64, requestCount int64, successCount int64, streamCount int64) {
	if requestCount <= 0 {
		return
	}
	if successRatePerm != nil {
		*successRatePerm = successCount * 1000 / requestCount
	}
	if streamRatePerm != nil {
		*streamRatePerm = streamCount * 1000 / requestCount
	}
}

func nacpSuccessSummaryLogTypes() []int {
	return []int{LogTypeConsume, LogTypeRetrySuccessSummary}
}

func nacpUserVisibleErrorTypes() []int {
	return []int{LogTypeRetryFailedSummary, LogTypeError}
}

func nacpUserVisibleLogTypes() []int {
	return []int{LogTypeConsume, LogTypeRetrySuccessSummary, LogTypeRetryFailedSummary, LogTypeError}
}

func nacpObservedLogTypes() []int {
	return []int{
		LogTypeConsume,
		LogTypeRetrySuccessSummary,
		LogTypeRetryFailedSummary,
		LogTypeError,
		LogTypeErrorIntercepted,
		LogTypeProbeSuccess,
		LogTypeProbeFailed,
	}
}

func nacpTrendBucketSeconds(rangeHours int) int64 {
	switch {
	case rangeHours <= 6:
		return 15 * 60
	case rangeHours <= 48:
		return 60 * 60
	case rangeHours <= 168:
		return 6 * 60 * 60
	default:
		return 24 * 60 * 60
	}
}

func nacpTrendBucketExpr() string {
	switch nacpEffectiveLogSqlType() {
	case common.DatabaseTypeMySQL:
		return "CAST(FLOOR(created_at / ?) * ? AS SIGNED)"
	case common.DatabaseTypePostgreSQL:
		return "CAST((created_at / ?) * ? AS BIGINT)"
	default:
		return "CAST((created_at / ?) * ? AS INTEGER)"
	}
}

func nacpEffectiveLogSqlType() string {
	if common.LogSqlType != common.DatabaseTypeSQLite || common.UsingSQLite {
		return common.LogSqlType
	}
	if LOG_DB == DB {
		if common.UsingMySQL {
			return common.DatabaseTypeMySQL
		}
		if common.UsingPostgreSQL {
			return common.DatabaseTypePostgreSQL
		}
	}
	return common.LogSqlType
}
