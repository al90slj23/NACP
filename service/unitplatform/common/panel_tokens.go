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

type TokenKeysResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Keys map[string]string `json:"keys"`
	} `json:"data"`
}

type TokenCreateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type TokenKeyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Key string `json:"key"`
	} `json:"data"`
}

type PanelTokenOptions struct {
	UserHeaders []string
}

func FetchPanelTokens(ctx context.Context, credentials unitplatform.Credentials, options PanelTokenOptions) ([]unitplatform.UnitPlatformToken, error) {
	if err := unitplatform.RequirePanelCredentials(credentials); err != nil {
		return nil, err
	}
	if len(options.UserHeaders) == 0 {
		options.UserHeaders = []string{"New-Api-User"}
	}

	client := &http.Client{Timeout: 20 * time.Second}
	tokens, err := FetchPanelTokenPage(ctx, client, credentials, options.UserHeaders)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return tokens, nil
	}
	keys, err := FetchPanelTokenKeysBatch(ctx, client, credentials, options.UserHeaders, tokens)
	if err == nil {
		for i := range tokens {
			if key := strings.TrimSpace(keys[strconv.Itoa(tokens[i].ID)]); key != "" {
				tokens[i].Key = key
			}
		}
		return tokens, nil
	}

	for i := range tokens {
		key, keyErr := FetchPanelTokenKey(ctx, client, credentials, options.UserHeaders, tokens[i].ID)
		if keyErr == nil && strings.TrimSpace(key) != "" {
			tokens[i].Key = strings.TrimSpace(key)
		}
	}
	return tokens, nil
}

func FetchPanelTokenOptions(ctx context.Context, credentials unitplatform.Credentials, options PanelTokenOptions) (*unitplatform.UnitPlatformTokenOptions, error) {
	if err := unitplatform.RequirePanelCredentials(credentials); err != nil {
		return nil, err
	}
	if len(options.UserHeaders) == 0 {
		options.UserHeaders = []string{"New-Api-User"}
	}
	client := &http.Client{Timeout: 20 * time.Second}
	result := &unitplatform.UnitPlatformTokenOptions{
		Groups: make([]unitplatform.UnitPlatformTokenGroupOption, 0),
		Models: make([]string, 0),
	}
	if groups, err := FetchPanelTokenGroups(ctx, client, credentials, options.UserHeaders); err == nil {
		result.Groups = groups
	}
	if models, err := FetchPanelTokenModels(ctx, client, credentials, options.UserHeaders); err == nil {
		result.Models = models
	}
	return result, nil
}

func CreatePanelToken(ctx context.Context, credentials unitplatform.Credentials, req unitplatform.CreateUnitPlatformTokenRequest, options PanelTokenOptions) (*unitplatform.UnitPlatformToken, error) {
	if err := unitplatform.RequirePanelCredentials(credentials); err != nil {
		return nil, err
	}
	if len(options.UserHeaders) == 0 {
		options.UserHeaders = []string{"New-Api-User"}
	}
	client := &http.Client{Timeout: 20 * time.Second}
	if err := CreatePanelTokenOnly(ctx, client, credentials, options.UserHeaders, req); err != nil {
		return nil, err
	}
	tokens, err := FetchPanelTokenPage(ctx, client, credentials, options.UserHeaders)
	if err != nil {
		return nil, err
	}
	for i := range tokens {
		if strings.TrimSpace(tokens[i].Name) != req.Name {
			continue
		}
		key, keyErr := FetchPanelTokenKey(ctx, client, credentials, options.UserHeaders, tokens[i].ID)
		if keyErr != nil {
			return nil, keyErr
		}
		tokens[i].Key = strings.TrimSpace(key)
		return &tokens[i], nil
	}
	return nil, fmt.Errorf("平台令牌已创建，但未能定位新令牌")
}

func FetchPanelTokenGroups(ctx context.Context, client *http.Client, credentials unitplatform.Credentials, userHeaders []string) ([]unitplatform.UnitPlatformTokenGroupOption, error) {
	targetURL := projectcommon.BuildURL(credentials.BaseURL, "/api/user/self/groups")
	req, err := unitplatform.BuildAuthRequest(ctx, http.MethodGet, targetURL, credentials, nil, userHeaders)
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

func FetchPanelTokenModels(ctx context.Context, client *http.Client, credentials unitplatform.Credentials, userHeaders []string) ([]string, error) {
	targetURL := projectcommon.BuildURL(credentials.BaseURL, "/api/user/models")
	req, err := unitplatform.BuildAuthRequest(ctx, http.MethodGet, targetURL, credentials, nil, userHeaders)
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

func FetchPanelTokenPage(ctx context.Context, client *http.Client, credentials unitplatform.Credentials, userHeaders []string) ([]unitplatform.UnitPlatformToken, error) {
	targetURL := projectcommon.BuildURL(credentials.BaseURL, "/api/token/?p=1&page_size=100")
	req, err := unitplatform.BuildAuthRequest(ctx, http.MethodGet, targetURL, credentials, nil, userHeaders)
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
		return nil, fmt.Errorf("读取平台令牌失败：响应不是有效 JSON：%w", err)
	}
	if !data.Success {
		if strings.TrimSpace(data.Message) != "" {
			return nil, errors.New(data.Message)
		}
		return nil, fmt.Errorf("读取平台令牌失败")
	}
	return data.Data.Items, nil
}

func FetchPanelTokenKeysBatch(ctx context.Context, client *http.Client, credentials unitplatform.Credentials, userHeaders []string, tokens []unitplatform.UnitPlatformToken) (map[string]string, error) {
	ids := make([]int, 0, len(tokens))
	for _, token := range tokens {
		if token.ID > 0 {
			ids = append(ids, token.ID)
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("没有可读取的令牌 ID")
	}
	body, err := projectcommon.Marshal(map[string]any{"ids": ids})
	if err != nil {
		return nil, err
	}
	targetURL := projectcommon.BuildURL(credentials.BaseURL, "/api/token/batch/keys")
	req, err := unitplatform.BuildAuthRequest(ctx, http.MethodPost, targetURL, credentials, body, userHeaders)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("读取平台令牌密钥失败：HTTP %d", resp.StatusCode)
	}
	var data TokenKeysResponse
	if err := projectcommon.DecodeJson(resp.Body, &data); err != nil {
		return nil, err
	}
	if !data.Success {
		return nil, errors.New(data.Message)
	}
	return data.Data.Keys, nil
}

func CreatePanelTokenOnly(ctx context.Context, client *http.Client, credentials unitplatform.Credentials, userHeaders []string, payload unitplatform.CreateUnitPlatformTokenRequest) error {
	body, err := projectcommon.Marshal(payload)
	if err != nil {
		return err
	}
	targetURL := projectcommon.BuildURL(credentials.BaseURL, "/api/token/")
	req, err := unitplatform.BuildAuthRequest(ctx, http.MethodPost, targetURL, credentials, body, userHeaders)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("创建平台令牌失败：HTTP %d", resp.StatusCode)
	}
	var data TokenCreateResponse
	if err := projectcommon.DecodeJson(resp.Body, &data); err != nil {
		return err
	}
	if !data.Success {
		if strings.TrimSpace(data.Message) != "" {
			return errors.New(data.Message)
		}
		return fmt.Errorf("创建平台令牌失败")
	}
	return nil
}

func FetchPanelTokenKey(ctx context.Context, client *http.Client, credentials unitplatform.Credentials, userHeaders []string, tokenID int) (string, error) {
	targetURL := projectcommon.BuildURL(credentials.BaseURL, fmt.Sprintf("/api/token/%d/key", tokenID))
	req, err := unitplatform.BuildAuthRequest(ctx, http.MethodPost, targetURL, credentials, []byte("{}"), userHeaders)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("读取平台令牌密钥失败：HTTP %d", resp.StatusCode)
	}
	var data TokenKeyResponse
	if err := projectcommon.DecodeJson(resp.Body, &data); err != nil {
		return "", err
	}
	if !data.Success {
		return "", errors.New(data.Message)
	}
	return data.Data.Key, nil
}
