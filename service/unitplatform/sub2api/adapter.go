package sub2api

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

type Adapter struct{}

func init() {
	unitplatform.Register(Adapter{})
}

func (Adapter) Type() string { return "sub2api" }

func (Adapter) Aliases() []string { return nil }

func (Adapter) Capabilities() unitplatform.CapabilitySet {
	return unitplatform.CapabilitySet{
		unitplatform.CapabilityDetect:   true,
		unitplatform.CapabilitySnapshot: false,
		unitplatform.CapabilityTokens:   true,
		unitplatform.CapabilityModels:   true,
		unitplatform.CapabilityGroups:   true,
	}
}

func (Adapter) CredentialSpec() unitplatform.CredentialSpec {
	return unitplatform.CredentialSpec{
		Mode: "jwt",
		Fields: []unitplatform.CredentialField{
			{Name: "access_token", Label: "JWT / Access Token", Required: true, Secret: true},
		},
	}
}

func (Adapter) Detect(ctx context.Context, client *http.Client, baseURL string) (unitplatform.Result, bool) {
	if strings.Contains(strings.ToLower(baseURL), "sub2api") {
		return unitplatform.Detected("sub2api", baseURL, "根据 URL 命中 sub2api 特征。"), true
	}
	settings := unitplatform.Probe(ctx, client, http.MethodGet, baseURL+"/api/v1/settings/public")
	if settings.Status == http.StatusOK {
		if _, ok := unitplatform.DecodeJSONResponse(settings); ok {
			return unitplatform.Detected("sub2api", baseURL, "检测到 sub2api 公共设置接口。"), true
		}
	}
	me := unitplatform.Probe(ctx, client, http.MethodGet, baseURL+"/api/v1/auth/me")
	body := strings.ToLower(string(me.Body))
	if strings.Contains(body, "api_key_required") || strings.Contains(body, "unauthorized") || strings.Contains(body, "auth") {
		return unitplatform.Detected("sub2api", baseURL, "检测到 sub2api 认证接口特征。"), true
	}
	return unitplatform.Result{}, false
}

func (Adapter) FetchSnapshot(ctx context.Context, credentials unitplatform.Credentials) (*unitplatform.Snapshot, error) {
	if strings.TrimSpace(credentials.AccessToken) == "" {
		return nil, fmt.Errorf("所属账号缺少 JWT / Access Token，无法监控 sub2api")
	}
	client := &http.Client{Timeout: 20 * time.Second}
	userData, err := fetchAuthMe(ctx, client, credentials)
	if err != nil {
		return nil, err
	}
	tokens, _ := fetchTokens(ctx, client, credentials)
	groups, _ := fetchGroups(ctx, client, credentials)
	models, _ := fetchModels(ctx, client, credentials)
	balance := extractFloat(userData, "balance")
	snapshot := &unitplatform.Snapshot{
		Platform:         credentials.Platform,
		Balance:          balance,
		Used:             0,
		BalanceUnit:      "USD",
		UpstreamUserID:   firstString(userData, "id", "user_id", "uid"),
		UpstreamUsername: displayName(userData),
		TokenCount:       len(tokens),
		ModelCount:       len(models),
		GroupCount:       len(groups),
		Raw: map[string]any{
			"collector": map[string]any{
				"platform": credentials.Platform,
				"adapter":  "sub2api",
				"endpoints": []string{
					"/api/v1/auth/me",
					"/api/v1/keys",
					"/api/v1/api-keys",
					"/api/v1/groups",
					"/v1/models",
				},
			},
			"auth_me":          userData,
			"tokens_count":     len(tokens),
			"groups_count":     len(groups),
			"models_count":     len(models),
			"collected_fields": []string{"balance", "upstream_user_id", "upstream_username", "token_count", "group_count", "model_count"},
		},
	}
	return snapshot, nil
}

func (Adapter) FetchTokens(ctx context.Context, credentials unitplatform.Credentials) ([]unitplatform.UnitPlatformToken, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	return fetchTokens(ctx, client, credentials)
}

func (Adapter) FetchTokenOptions(ctx context.Context, credentials unitplatform.Credentials) (*unitplatform.UnitPlatformTokenOptions, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	options := &unitplatform.UnitPlatformTokenOptions{
		Groups: make([]unitplatform.UnitPlatformTokenGroupOption, 0),
		Models: make([]string, 0),
	}
	if groups, err := fetchGroups(ctx, client, credentials); err == nil {
		options.Groups = groups
	}
	if models, err := fetchModels(ctx, client, credentials); err == nil {
		options.Models = models
	}
	return options, nil
}

func (Adapter) CreateToken(ctx context.Context, credentials unitplatform.Credentials, req unitplatform.CreateUnitPlatformTokenRequest) (*unitplatform.UnitPlatformToken, error) {
	if strings.TrimSpace(credentials.AccessToken) == "" {
		return nil, fmt.Errorf("所属账号缺少 JWT / Access Token，无法创建 sub2api 平台令牌")
	}
	client := &http.Client{Timeout: 20 * time.Second}
	payload := map[string]any{
		"name": strings.TrimSpace(req.Name),
	}
	if groupID, err := strconv.Atoi(strings.TrimSpace(req.Group)); err == nil && groupID > 0 {
		payload["group_id"] = groupID
	}
	if days := resolveExpiresInDays(req.ExpiredTime); days > 0 {
		payload["expires_in_days"] = days
	}
	if !req.UnlimitedQuota && req.RemainQuota > 0 {
		payload["quota"] = req.RemainQuota
	}
	body, err := projectcommon.Marshal(payload)
	if err != nil {
		return nil, err
	}
	endpoints := []string{"/api/v1/keys", "/api/v1/api-keys"}
	var lastErr error
	for _, endpoint := range endpoints {
		reqHTTP, err := unitplatform.BuildAuthRequest(ctx, http.MethodPost, projectcommon.BuildURL(credentials.BaseURL, endpoint), credentials, body, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(reqHTTP)
		if err != nil {
			lastErr = err
			continue
		}
		var payload map[string]any
		decodeErr := projectcommon.DecodeJson(resp.Body, &payload)
		_ = resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("创建 sub2api 平台令牌失败：HTTP %d", resp.StatusCode)
			continue
		}
		if decodeErr != nil {
			lastErr = decodeErr
			continue
		}
		data, err := parseEnvelope(payload, endpoint)
		if err != nil {
			lastErr = err
			continue
		}
		if token := normalizeToken(data, 0); token.Key != "" || token.ID > 0 {
			return &token, nil
		}
		tokens, err := fetchTokens(ctx, client, credentials)
		if err != nil {
			lastErr = err
			continue
		}
		for i := range tokens {
			if strings.TrimSpace(tokens[i].Name) == strings.TrimSpace(req.Name) {
				return &tokens[i], nil
			}
		}
		return nil, fmt.Errorf("sub2api 平台令牌已创建，但未能定位新令牌")
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("创建 sub2api 平台令牌失败")
}

func fetchAuthMe(ctx context.Context, client *http.Client, credentials unitplatform.Credentials) (map[string]any, error) {
	req, err := unitplatform.BuildAuthRequest(ctx, http.MethodGet, projectcommon.BuildURL(credentials.BaseURL, "/api/v1/auth/me"), credentials, nil, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("读取 sub2api 账号信息失败：HTTP %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := projectcommon.DecodeJson(resp.Body, &payload); err != nil {
		return nil, err
	}
	data, err := parseEnvelope(payload, "/api/v1/auth/me")
	if err != nil {
		return nil, err
	}
	return data, nil
}

func fetchTokens(ctx context.Context, client *http.Client, credentials unitplatform.Credentials) ([]unitplatform.UnitPlatformToken, error) {
	endpoints := []string{
		"/api/v1/keys?page=1&page_size=100",
		"/api/v1/api-keys?page=1&page_size=100",
	}
	var lastErr error
	for _, endpoint := range endpoints {
		req, err := unitplatform.BuildAuthRequest(ctx, http.MethodGet, projectcommon.BuildURL(credentials.BaseURL, endpoint), credentials, nil, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		var payload map[string]any
		decodeErr := projectcommon.DecodeJson(resp.Body, &payload)
		_ = resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("读取 sub2api 平台令牌失败：HTTP %d", resp.StatusCode)
			continue
		}
		if decodeErr != nil {
			lastErr = decodeErr
			continue
		}
		data, err := parseEnvelope(payload, endpoint)
		if err != nil {
			lastErr = err
			continue
		}
		tokens := parseTokenItems(data)
		if len(tokens) > 0 {
			return tokens, nil
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return []unitplatform.UnitPlatformToken{}, nil
}

func fetchGroups(ctx context.Context, client *http.Client, credentials unitplatform.Credentials) ([]unitplatform.UnitPlatformTokenGroupOption, error) {
	endpoints := []string{
		"/api/v1/groups/available",
		"/api/v1/groups?page=1&page_size=100",
		"/api/v1/groups",
		"/api/v1/group?page=1&page_size=100",
		"/api/v1/group",
	}
	seen := map[string]bool{}
	result := make([]unitplatform.UnitPlatformTokenGroupOption, 0)
	for _, endpoint := range endpoints {
		req, err := unitplatform.BuildAuthRequest(ctx, http.MethodGet, projectcommon.BuildURL(credentials.BaseURL, endpoint), credentials, nil, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		var payload map[string]any
		decodeErr := projectcommon.DecodeJson(resp.Body, &payload)
		_ = resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 || decodeErr != nil {
			continue
		}
		data, err := parseEnvelope(payload, endpoint)
		if err != nil {
			data = payload
		}
		for _, group := range parseGroupItems(data) {
			if group.Value == "" || seen[group.Value] {
				continue
			}
			seen[group.Value] = true
			result = append(result, group)
		}
		if len(result) > 0 {
			return result, nil
		}
	}
	tokens, err := fetchTokens(ctx, client, credentials)
	if err == nil {
		for _, token := range tokens {
			group := strings.TrimSpace(token.Group)
			if group == "" || seen[group] {
				continue
			}
			seen[group] = true
			result = append(result, unitplatform.UnitPlatformTokenGroupOption{Value: group, Label: group})
		}
	}
	if len(result) == 0 {
		result = append(result, unitplatform.UnitPlatformTokenGroupOption{Value: "default", Label: "default"})
	}
	return result, nil
}

func fetchModels(ctx context.Context, client *http.Client, credentials unitplatform.Credentials) ([]string, error) {
	tokens := []string{credentials.AccessToken, credentials.APIKey}
	if apiTokens, err := fetchTokens(ctx, client, credentials); err == nil {
		for _, token := range apiTokens {
			if token.Key != "" {
				tokens = append(tokens, token.Key)
			}
		}
	}
	endpoints := []string{"/v1/models", "/api/v1/models", "/v1beta/models", "/antigravity/v1beta/models"}
	seen := map[string]bool{}
	models := make([]string, 0)
	for _, token := range tokens {
		authToken := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(token, "Bearer "), "bearer "))
		if authToken == "" {
			continue
		}
		modelCredentials := credentials
		modelCredentials.AccessToken = authToken
		modelCredentials.APIKey = authToken
		for _, endpoint := range endpoints {
			req, err := unitplatform.BuildAuthRequest(ctx, http.MethodGet, projectcommon.BuildURL(credentials.BaseURL, endpoint), modelCredentials, nil, nil)
			if err != nil {
				continue
			}
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			var payload map[string]any
			decodeErr := projectcommon.DecodeJson(resp.Body, &payload)
			_ = resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode >= 300 || decodeErr != nil {
				continue
			}
			for _, model := range parseModelItems(payload) {
				if model == "" || seen[model] {
					continue
				}
				seen[model] = true
				models = append(models, model)
			}
			if len(models) > 0 {
				return models, nil
			}
		}
	}
	if len(models) == 0 {
		return nil, errors.New("未能读取 sub2api 模型列表")
	}
	return models, nil
}

func parseEnvelope(payload map[string]any, endpoint string) (map[string]any, error) {
	if payload == nil {
		return nil, fmt.Errorf("%s 响应为空", endpoint)
	}
	if success, ok := payload["success"].(bool); ok {
		if !success {
			return nil, fmt.Errorf("%s", firstString(payload, "message", "error"))
		}
		if data, ok := payload["data"].(map[string]any); ok {
			return data, nil
		}
		return payload, nil
	}
	if rawCode, ok := payload["code"]; ok {
		code := int(extractFloat(map[string]any{"code": rawCode}, "code"))
		if code != 0 {
			message := firstString(payload, "message", "error")
			if message == "" {
				message = fmt.Sprintf("%s 返回错误码 %d", endpoint, code)
			}
			return nil, errors.New(message)
		}
		if data, ok := payload["data"].(map[string]any); ok {
			return data, nil
		}
		if items, ok := payload["data"].([]any); ok {
			return map[string]any{"items": items}, nil
		}
		return map[string]any{}, nil
	}
	if data, ok := payload["data"].(map[string]any); ok {
		return data, nil
	}
	return payload, nil
}

func parseTokenItems(payload map[string]any) []unitplatform.UnitPlatformToken {
	rawItems := extractItems(payload)
	tokens := make([]unitplatform.UnitPlatformToken, 0, len(rawItems))
	for i, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		token := normalizeToken(item, i)
		if token.Key == "" && token.ID == 0 {
			continue
		}
		tokens = append(tokens, token)
	}
	return tokens
}

func normalizeToken(item map[string]any, index int) unitplatform.UnitPlatformToken {
	id := int(extractFloat(item, "id"))
	name := firstString(item, "name", "title")
	if name == "" {
		if id > 0 {
			name = fmt.Sprintf("token-%d", id)
		} else {
			name = fmt.Sprintf("token-%d", index+1)
		}
	}
	group := firstString(item, "group_id", "groupId", "group_name", "groupName", "group")
	status := 1
	if !parseEnabled(item["status"]) {
		status = 2
	}
	return unitplatform.UnitPlatformToken{
		ID:                 id,
		Name:               name,
		Key:                firstString(item, "key", "token", "api_key", "apiKey"),
		Group:              group,
		Status:             status,
		ExpiredTime:        int64(extractFloat(item, "expired_time", "expiredTime", "expires_at", "expiresAt")),
		RemainQuota:        int(extractFloat(item, "remain_quota", "remainQuota", "quota")),
		UnlimitedQuota:     extractFloat(item, "quota", "remain_quota", "remainQuota") == 0,
		ModelLimitsEnabled: false,
		ModelLimits:        "",
	}
}

func parseGroupItems(payload map[string]any) []unitplatform.UnitPlatformTokenGroupOption {
	rawItems := extractItems(payload)
	groups := make([]unitplatform.UnitPlatformTokenGroupOption, 0, len(rawItems))
	for _, raw := range rawItems {
		switch item := raw.(type) {
		case string:
			value := strings.TrimSpace(item)
			if value != "" {
				groups = append(groups, unitplatform.UnitPlatformTokenGroupOption{Value: value, Label: value})
			}
		case float64:
			if item > 0 {
				value := strconv.Itoa(int(item))
				groups = append(groups, unitplatform.UnitPlatformTokenGroupOption{Value: value, Label: value})
			}
		case map[string]any:
			value := firstString(item, "group_id", "groupId", "id", "value", "code", "name", "group_name", "groupName")
			label := firstString(item, "name", "group_name", "groupName", "title", "label")
			if label == "" {
				label = value
			}
			if value != "" {
				groups = append(groups, unitplatform.UnitPlatformTokenGroupOption{Value: value, Label: label})
			}
		}
	}
	return groups
}

func parseModelItems(payload map[string]any) []string {
	rawItems := extractItems(payload)
	models := make([]string, 0, len(rawItems))
	for _, raw := range rawItems {
		switch item := raw.(type) {
		case string:
			models = append(models, normalizeModelName(item))
		case map[string]any:
			models = append(models, normalizeModelName(firstString(item, "id", "name", "model")))
		}
	}
	return models
}

func extractItems(payload map[string]any) []any {
	if payload == nil {
		return nil
	}
	for _, key := range []string{"items", "list", "data", "groups", "models"} {
		if items, ok := payload[key].([]any); ok {
			return items
		}
	}
	if data, ok := payload["data"].(map[string]any); ok {
		return extractItems(data)
	}
	return nil
}

func normalizeModelName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "models/")
	return value
}

func displayName(data map[string]any) string {
	username := firstString(data, "username", "name")
	if username != "" {
		return username
	}
	email := firstString(data, "email")
	if at := strings.Index(email, "@"); at > 0 {
		return email[:at]
	}
	return email
}

func firstString(data map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := data[key]
		if !ok || raw == nil {
			continue
		}
		value := strings.TrimSpace(fmt.Sprint(raw))
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func extractFloat(data map[string]any, keys ...string) float64 {
	for _, key := range keys {
		raw, ok := data[key]
		if !ok || raw == nil {
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

func parseEnabled(raw any) bool {
	if raw == nil {
		return true
	}
	switch value := raw.(type) {
	case bool:
		return value
	case float64:
		return value == 1
	case string:
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			return true
		}
		return !map[string]bool{"inactive": true, "disabled": true, "false": true, "0": true, "off": true}[normalized]
	default:
		return true
	}
}

func resolveExpiresInDays(expiredTime int64) int {
	if expiredTime <= 0 {
		return 0
	}
	expiresAt := expiredTime
	if expiresAt < 10000000000 {
		expiresAt *= 1000
	}
	delta := time.UnixMilli(expiresAt).Sub(time.Now())
	if delta <= 0 {
		return 1
	}
	days := int(delta.Hours() / 24)
	if days < 1 {
		return 1
	}
	if days > 3650 {
		return 3650
	}
	return days
}
