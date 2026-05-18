package common

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	projectcommon "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service/unitplatform"
)

type SelfResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

type TokenPageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Items []unitplatform.UnitPlatformToken `json:"items"`
		Total int                              `json:"total"`
	} `json:"data"`
}

type ModelsResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Data    []string `json:"data"`
}

type GroupsResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    map[string]struct {
		Desc  string  `json:"desc"`
		Ratio float64 `json:"ratio"`
	} `json:"data"`
}

type PanelSnapshotOptions struct {
	AdapterName string
	UserHeaders []string
	QuotaUnit   float64
}

func FetchPanelSnapshot(ctx context.Context, credentials unitplatform.Credentials, options PanelSnapshotOptions) (*unitplatform.Snapshot, error) {
	if err := unitplatform.RequirePanelCredentials(credentials); err != nil {
		return nil, err
	}
	if options.AdapterName == "" {
		options.AdapterName = credentials.Platform
	}
	if len(options.UserHeaders) == 0 {
		options.UserHeaders = []string{"New-Api-User"}
	}
	if options.QuotaUnit <= 0 {
		options.QuotaUnit = projectcommon.QuotaPerUnit
	}

	client := &http.Client{Timeout: 20 * time.Second}
	selfData, err := fetchSelf(ctx, client, credentials, options.UserHeaders)
	if err != nil {
		return nil, err
	}
	snapshot := &unitplatform.Snapshot{
		Platform:    credentials.Platform,
		BalanceUnit: "USD",
		Raw: map[string]any{
			"collector": map[string]any{
				"platform": credentials.Platform,
				"adapter":  options.AdapterName,
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
	if statusData, err := fetchStatus(ctx, client, credentials, options.UserHeaders); err == nil {
		snapshot.Raw["platform_status"] = statusData
	}
	snapshot.Balance = extractMoneyValue(selfData, options.QuotaUnit, "quota", "balance_quota")
	if snapshot.Balance == 0 {
		snapshot.Balance = extractPlainFloat(selfData, "balance", "current_balance")
	}
	snapshot.Used = extractMoneyValue(selfData, options.QuotaUnit, "used_quota")
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
	if tokens, err := fetchTokenPage(ctx, client, credentials, options.UserHeaders); err == nil {
		snapshot.TokenCount = len(tokens)
		snapshot.Raw["tokens_count"] = len(tokens)
		collectedFields = append(collectedFields, "token_count")
	}
	if models, err := fetchModels(ctx, client, credentials, options.UserHeaders); err == nil {
		snapshot.ModelCount = len(models)
		snapshot.Raw["models_count"] = len(models)
		collectedFields = append(collectedFields, "model_count")
	}
	if groups, err := fetchGroups(ctx, client, credentials, options.UserHeaders); err == nil {
		snapshot.GroupCount = len(groups)
		snapshot.Raw["groups_count"] = len(groups)
		collectedFields = append(collectedFields, "group_count")
	}
	snapshot.Raw["collected_fields"] = collectedFields
	return snapshot, nil
}

func fetchStatus(ctx context.Context, client *http.Client, credentials unitplatform.Credentials, userHeaders []string) (map[string]any, error) {
	req, err := unitplatform.BuildAuthRequest(ctx, http.MethodGet, projectcommon.BuildURL(credentials.BaseURL, "/api/status"), credentials, nil, userHeaders)
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
	var data SelfResponse
	if err := projectcommon.DecodeJson(resp.Body, &data); err != nil {
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

func fetchSelf(ctx context.Context, client *http.Client, credentials unitplatform.Credentials, userHeaders []string) (map[string]any, error) {
	req, err := unitplatform.BuildAuthRequest(ctx, http.MethodGet, projectcommon.BuildURL(credentials.BaseURL, "/api/user/self"), credentials, nil, userHeaders)
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
	var data SelfResponse
	if err := projectcommon.DecodeJson(resp.Body, &data); err != nil {
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

func fetchTokenPage(ctx context.Context, client *http.Client, credentials unitplatform.Credentials, userHeaders []string) ([]unitplatform.UnitPlatformToken, error) {
	req, err := unitplatform.BuildAuthRequest(ctx, http.MethodGet, projectcommon.BuildURL(credentials.BaseURL, "/api/token/?p=1&page_size=100"), credentials, nil, userHeaders)
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
	var data TokenPageResponse
	if err := projectcommon.DecodeJson(resp.Body, &data); err != nil {
		return nil, err
	}
	if !data.Success {
		return nil, errors.New(data.Message)
	}
	return data.Data.Items, nil
}

func fetchModels(ctx context.Context, client *http.Client, credentials unitplatform.Credentials, userHeaders []string) ([]string, error) {
	req, err := unitplatform.BuildAuthRequest(ctx, http.MethodGet, projectcommon.BuildURL(credentials.BaseURL, "/api/user/models"), credentials, nil, userHeaders)
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
	var data ModelsResponse
	if err := projectcommon.DecodeJson(resp.Body, &data); err != nil {
		return nil, err
	}
	if !data.Success {
		return nil, errors.New(data.Message)
	}
	return data.Data, nil
}

func fetchGroups(ctx context.Context, client *http.Client, credentials unitplatform.Credentials, userHeaders []string) ([]unitplatform.UnitPlatformTokenGroupOption, error) {
	req, err := unitplatform.BuildAuthRequest(ctx, http.MethodGet, projectcommon.BuildURL(credentials.BaseURL, "/api/user/self/groups"), credentials, nil, userHeaders)
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
	var data GroupsResponse
	if err := projectcommon.DecodeJson(resp.Body, &data); err != nil {
		return nil, err
	}
	if !data.Success {
		return nil, errors.New(data.Message)
	}
	groups := make([]unitplatform.UnitPlatformTokenGroupOption, 0, len(data.Data))
	for value, info := range data.Data {
		label := strings.TrimSpace(info.Desc)
		if label == "" {
			label = value
		}
		groups = append(groups, unitplatform.UnitPlatformTokenGroupOption{
			Value: value,
			Label: label,
			Ratio: info.Ratio,
		})
	}
	return groups, nil
}

func ExtractStatusData(statusJSON map[string]interface{}) map[string]interface{} {
	if data, ok := statusJSON["data"].(map[string]interface{}); ok {
		return data
	}
	return statusJSON
}

func HasAnyKey(data map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		if _, ok := data[key]; ok {
			return true
		}
	}
	return false
}

func AsString(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

func ExtractPlainFloat(data map[string]any, keys ...string) float64 {
	return extractPlainFloat(data, keys...)
}

func extractMoneyValue(data map[string]any, quotaUnit float64, keys ...string) float64 {
	value := extractPlainFloat(data, keys...)
	if value == 0 {
		return 0
	}
	return value / quotaUnit
}

func extractPlainFloat(data map[string]any, keys ...string) float64 {
	for _, key := range keys {
		raw, ok := data[key]
		if !ok {
			continue
		}
		switch value := raw.(type) {
		case float64:
			return value
		case float32:
			return float64(value)
		case int:
			return float64(value)
		case int64:
			return float64(value)
		case string:
			parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if err == nil {
				return parsed
			}
		}
	}
	return 0
}

func firstString(data map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := data[key]
		if !ok {
			continue
		}
		value := strings.TrimSpace(fmt.Sprint(raw))
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}
