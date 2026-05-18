package unitmonitor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

func fetchNewAPICompatible(ctx context.Context, credentials Credentials) (*Snapshot, error) {
	if strings.TrimSpace(credentials.AccessToken) == "" || strings.TrimSpace(credentials.AccountID) == "" {
		return nil, fmt.Errorf("所属账号缺少账户访问令牌或账户 ID，无法监控余额")
	}
	client := &http.Client{Timeout: 20 * time.Second}
	selfData, err := fetchSelf(ctx, client, credentials)
	if err != nil {
		return nil, err
	}
	snapshot := &Snapshot{
		Platform:    credentials.Platform,
		BalanceUnit: "USD",
		Raw: map[string]any{
			"collector": map[string]any{
				"platform": credentials.Platform,
				"adapter":  "newapi-compatible",
				"endpoints": []string{
					"/api/status",
					"/api/user/self",
					"/api/token/?p=1&page_size=100",
					"/api/user/models",
					"/api/user/self/groups",
				},
			},
			"user_self": selfData,
		},
	}
	if statusData, err := fetchStatus(ctx, client, credentials); err == nil {
		snapshot.Raw["platform_status"] = statusData
	}
	snapshot.Balance = extractMoneyValue(selfData, "quota", "balance_quota")
	if snapshot.Balance == 0 {
		snapshot.Balance = extractPlainFloat(selfData, "balance", "current_balance")
	}
	snapshot.Used = extractMoneyValue(selfData, "used_quota")
	if snapshot.Used == 0 {
		snapshot.Used = extractPlainFloat(selfData, "used", "used_amount")
	}
	snapshot.UpstreamUserID = firstString(selfData, "id", "user_id", "uid")
	snapshot.UpstreamUsername = firstString(selfData, "username", "display_name", "name")
	snapshot.UpstreamGroup = firstString(selfData, "group", "group_name")

	collectedFields := []string{"balance", "used", "upstream_user_id", "upstream_username", "upstream_group"}
	if snapshot.Raw["platform_status"] != nil {
		collectedFields = append([]string{"platform_status"}, collectedFields...)
	}
	if tokens, err := fetchTokenPage(ctx, client, credentials); err == nil {
		snapshot.TokenCount = len(tokens)
		snapshot.Raw["tokens_count"] = len(tokens)
		collectedFields = append(collectedFields, "token_count")
	}
	if models, err := fetchModels(ctx, client, credentials); err == nil {
		snapshot.ModelCount = len(models)
		snapshot.Raw["models_count"] = len(models)
		collectedFields = append(collectedFields, "model_count")
	}
	if groups, err := fetchGroups(ctx, client, credentials); err == nil {
		snapshot.GroupCount = len(groups)
		snapshot.Raw["groups_count"] = len(groups)
		collectedFields = append(collectedFields, "group_count")
	}
	snapshot.Raw["collected_fields"] = collectedFields
	return snapshot, nil
}

func fetchStatus(ctx context.Context, client *http.Client, credentials Credentials) (map[string]any, error) {
	req, err := newRequest(ctx, http.MethodGet, common.BuildURL(credentials.BaseURL, "/api/status"), credentials.AccessToken, credentials.AccountID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("读取平台状态失败：HTTP %d", resp.StatusCode)
	}
	var data selfResponse
	if err := common.DecodeJson(resp.Body, &data); err != nil {
		return nil, err
	}
	if !data.Success {
		if strings.TrimSpace(data.Message) != "" {
			return nil, errors.New(data.Message)
		}
		return nil, fmt.Errorf("读取平台状态失败")
	}
	if data.Data == nil {
		return map[string]any{}, nil
	}
	return data.Data, nil
}

func fetchSelf(ctx context.Context, client *http.Client, credentials Credentials) (map[string]any, error) {
	req, err := newRequest(ctx, http.MethodGet, common.BuildURL(credentials.BaseURL, "/api/user/self"), credentials.AccessToken, credentials.AccountID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("读取账号信息失败：HTTP %d", resp.StatusCode)
	}
	var data selfResponse
	if err := common.DecodeJson(resp.Body, &data); err != nil {
		return nil, err
	}
	if !data.Success {
		if strings.TrimSpace(data.Message) != "" {
			return nil, errors.New(data.Message)
		}
		return nil, fmt.Errorf("读取账号信息失败")
	}
	if data.Data == nil {
		return map[string]any{}, nil
	}
	return data.Data, nil
}

func fetchTokenPage(ctx context.Context, client *http.Client, credentials Credentials) ([]UnitPlatformToken, error) {
	req, err := newRequest(ctx, http.MethodGet, common.BuildURL(credentials.BaseURL, "/api/token/?p=1&page_size=100"), credentials.AccessToken, credentials.AccountID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("读取平台令牌失败：HTTP %d", resp.StatusCode)
	}
	var data tokenPageResponse
	if err := common.DecodeJson(resp.Body, &data); err != nil {
		return nil, err
	}
	if !data.Success {
		return nil, errors.New(data.Message)
	}
	return data.Data.Items, nil
}

func fetchModels(ctx context.Context, client *http.Client, credentials Credentials) ([]string, error) {
	req, err := newRequest(ctx, http.MethodGet, common.BuildURL(credentials.BaseURL, "/api/user/models"), credentials.AccessToken, credentials.AccountID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("读取平台模型失败：HTTP %d", resp.StatusCode)
	}
	var data modelsResponse
	if err := common.DecodeJson(resp.Body, &data); err != nil {
		return nil, err
	}
	if !data.Success {
		return nil, errors.New(data.Message)
	}
	return data.Data, nil
}

func fetchGroups(ctx context.Context, client *http.Client, credentials Credentials) ([]UnitPlatformTokenGroupOption, error) {
	req, err := newRequest(ctx, http.MethodGet, common.BuildURL(credentials.BaseURL, "/api/user/self/groups"), credentials.AccessToken, credentials.AccountID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("读取平台分组失败：HTTP %d", resp.StatusCode)
	}
	var data groupsResponse
	if err := common.DecodeJson(resp.Body, &data); err != nil {
		return nil, err
	}
	if !data.Success {
		return nil, errors.New(data.Message)
	}
	groups := make([]UnitPlatformTokenGroupOption, 0, len(data.Data))
	for value, info := range data.Data {
		label := strings.TrimSpace(info.Desc)
		if label == "" {
			label = value
		}
		groups = append(groups, UnitPlatformTokenGroupOption{
			Value: value,
			Label: label,
			Ratio: info.Ratio,
		})
	}
	return groups, nil
}
