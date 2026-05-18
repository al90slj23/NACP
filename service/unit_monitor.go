package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/unitmonitor"
	"github.com/QuantumNous/new-api/service/unitplatform"
)

type UnitAccountMonitorSnapshot = unitmonitor.Snapshot

func CheckUnitAccountMonitor(ctx context.Context, monitor *model.UnitAccountMonitor) (*UnitAccountMonitorSnapshot, error) {
	if monitor == nil {
		return nil, fmt.Errorf("监控不存在")
	}
	unit, err := model.GetUnitById(monitor.UnitID)
	if err != nil {
		return nil, err
	}
	account, err := model.GetUnitAccountById(monitor.UnitAccountID)
	if err != nil {
		return nil, err
	}
	if account.UnitID != unit.Id {
		return nil, fmt.Errorf("监控账号不属于该单位")
	}
	return FetchUnitAccountMonitorSnapshot(ctx, unit, account)
}

func FetchUnitAccountMonitorSnapshot(ctx context.Context, unit *model.Unit, account *model.UnitAccount) (*UnitAccountMonitorSnapshot, error) {
	credentials, err := resolveUnitPlatformCredentials(unit, account)
	if err != nil {
		return nil, err
	}
	return unitmonitor.Fetch(ctx, unitmonitor.Credentials{
		Platform:    credentials.Platform,
		BaseURL:     credentials.BaseURL,
		AccessToken: credentials.AccessToken,
		AccountID:   credentials.AccountID,
	})
}

func resolveUnitMonitorCredentials(unit *model.Unit, account *model.UnitAccount) (baseURL string, accessToken string, accountID string, err error) {
	if unit == nil || account == nil {
		return "", "", "", fmt.Errorf("单位或账号不存在")
	}
	unit.Normalize()
	unitType := strings.ToLower(strings.TrimSpace(unit.Type))
	switch unitType {
	case model.UnitTypeNewAPI,
		model.UnitTypeSub2API,
		model.UnitTypeOneAPI,
		model.UnitTypeOneAPIFork,
		model.UnitTypeShellAPI,
		model.UnitTypeRixAPI,
		model.UnitTypeVeloera,
		model.UnitTypeOneHub,
		model.UnitTypeDoneHub,
		model.UnitTypeAnyRouter:
	default:
		return "", "", "", fmt.Errorf("当前单位类型 %s 暂不支持账号监控", unit.Type)
	}
	baseURL = unitplatform.NormalizeBaseURL(unit.EffectiveAPIURL)
	if baseURL == "" {
		baseURL = unitplatform.NormalizeBaseURL(unit.APIURL)
	}
	if baseURL == "" {
		baseURL = unitplatform.NormalizeBaseURL(unit.WebsiteURL)
	}
	if baseURL == "" {
		return "", "", "", fmt.Errorf("单位 API 地址为空")
	}
	accessToken = strings.TrimSpace(account.AccessToken)
	accountID = strings.TrimSpace(account.AccountID)
	if accessToken == "" || accountID == "" {
		return "", "", "", fmt.Errorf("所属账号缺少账户访问令牌或账户 ID，无法监控余额")
	}
	return baseURL, accessToken, accountID, nil
}

func ApplyUnitMonitorSnapshot(monitor *model.UnitAccountMonitor, snapshot *UnitAccountMonitorSnapshot) error {
	if monitor == nil || snapshot == nil {
		return fmt.Errorf("监控或快照为空")
	}
	rawBytes, err := common.Marshal(snapshot.Raw)
	if err != nil {
		return err
	}
	monitor.CurrentBalance = snapshot.Balance
	monitor.UsedAmount = snapshot.Used
	monitor.BalanceUnit = snapshot.BalanceUnit
	monitor.PlatformStatus = "ok"
	monitor.UpstreamUserID = snapshot.UpstreamUserID
	monitor.UpstreamUsername = snapshot.UpstreamUsername
	monitor.UpstreamGroup = snapshot.UpstreamGroup
	monitor.TokenCount = snapshot.TokenCount
	monitor.ModelCount = snapshot.ModelCount
	monitor.GroupCount = snapshot.GroupCount
	monitor.LastCheckedTime = common.GetTimestamp()
	monitor.ErrorMessage = ""
	monitor.RawJSON = string(rawBytes)
	if err := monitor.SaveCheckResult(); err != nil {
		return err
	}
	if err := monitor.RecordSnapshot(); err != nil {
		common.SysLog("failed to record unit monitor snapshot: " + err.Error())
	}
	return nil
}

func ApplyUnitMonitorError(monitor *model.UnitAccountMonitor, err error) error {
	if monitor == nil {
		return fmt.Errorf("监控不存在")
	}
	message := ""
	if err != nil {
		message = err.Error()
	}
	monitor.PlatformStatus = "error"
	monitor.LastCheckedTime = common.GetTimestamp()
	monitor.ErrorMessage = message
	return monitor.SaveCheckResult()
}
