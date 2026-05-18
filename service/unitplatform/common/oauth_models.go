package common

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	projectcommon "github.com/QuantumNous/new-api/common"
)

const (
	CodexOriginator = "codex_cli_rs"

	GeminiCLIRequiredService = "cloudaicompanion.googleapis.com"
	GeminiCLIGoogleAPIClient = "google-genai-sdk/1.41.0 gl-node/v22.19.0"
	GeminiCLIUserAgent       = "GeminiCLI/0.31.0/unknown (win32; x64)"

	AntigravityDefaultBaseURL = "https://cloudcode-pa.googleapis.com"
	AntigravityDailyBaseURL   = "https://daily-cloudcode-pa.googleapis.com"
	AntigravitySandboxBaseURL = "https://daily-cloudcode-pa.sandbox.googleapis.com"
	AntigravityUserAgent      = "antigravity/1.19.6 darwin/arm64"
)

var GeminiCLIStaticModels = []string{
	"gemini-2.5-pro",
	"gemini-2.5-flash",
	"gemini-2.5-flash-lite",
	"gemini-3-pro-preview",
	"gemini-3.1-pro-preview",
	"gemini-3-flash-preview",
	"gemini-3.1-flash-lite-preview",
}

func FetchCodexModels(ctx context.Context, baseURL string, accessToken string, accountID string) ([]string, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, fmt.Errorf("缺少 Codex OAuth Access Token")
	}
	targetURL := strings.TrimRight(baseURL, "/") + "/models?client_version=1.0.0"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Originator", CodexOriginator)
	if strings.TrimSpace(accountID) != "" {
		req.Header.Set("Chatgpt-Account-Id", strings.TrimSpace(accountID))
	}
	return fetchOAuthModels(req)
}

func ValidateGeminiCLI(ctx context.Context, accessToken string, projectID string) error {
	accessToken = strings.TrimSpace(accessToken)
	projectID = strings.TrimSpace(projectID)
	if accessToken == "" {
		return fmt.Errorf("缺少 Gemini CLI OAuth Access Token")
	}
	if projectID == "" {
		return fmt.Errorf("缺少 Gemini CLI 项目 ID")
	}
	targetURL := fmt.Sprintf(
		"https://serviceusage.googleapis.com/v1/projects/%s/services/%s",
		projectID,
		GeminiCLIRequiredService,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", GeminiCLIUserAgent)
	req.Header.Set("X-Goog-Api-Client", GeminiCLIGoogleAPIClient)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Gemini CLI OAuth 校验失败：HTTP %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := projectcommon.DecodeJson(resp.Body, &payload); err != nil {
		return err
	}
	if strings.ToUpper(strings.TrimSpace(fmt.Sprint(payload["state"]))) != "ENABLED" {
		return fmt.Errorf("项目 %s 未启用 %s", projectID, GeminiCLIRequiredService)
	}
	return nil
}

func FetchAntigravityModels(ctx context.Context, baseURL string, accessToken string, projectID string) ([]string, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, fmt.Errorf("缺少 Antigravity OAuth Access Token")
	}
	requestBody := []byte("{}")
	if strings.TrimSpace(projectID) != "" {
		var err error
		requestBody, err = projectcommon.Marshal(map[string]string{"project": strings.TrimSpace(projectID)})
		if err != nil {
			return nil, err
		}
	}
	var lastErr error
	for _, candidate := range antigravityDiscoveryBaseURLs(baseURL) {
		targetURL := strings.TrimRight(candidate, "/") + "/v1internal:fetchAvailableModels"
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(requestBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", AntigravityUserAgent)
		models, err := fetchOAuthModels(req)
		if err == nil && len(models) > 0 {
			return models, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("未获取到 Antigravity 可用模型")
}

func antigravityDiscoveryBaseURLs(baseURL string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, 4)
	for _, candidate := range []string{
		baseURL,
		AntigravityDefaultBaseURL,
		AntigravityDailyBaseURL,
		AntigravitySandboxBaseURL,
	} {
		candidate = strings.TrimRight(strings.TrimSpace(candidate), "/")
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		result = append(result, candidate)
	}
	return result
}

func fetchOAuthModels(req *http.Request) ([]string, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("读取模型失败：HTTP %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := projectcommon.DecodeJson(resp.Body, &payload); err != nil {
		return nil, err
	}
	models := extractOAuthModelIDs(payload)
	if len(models) == 0 {
		return nil, fmt.Errorf("读取模型失败：响应中没有模型列表")
	}
	return models, nil
}

func extractOAuthModelIDs(payload map[string]any) []string {
	if payload == nil {
		return nil
	}
	if modelsObj, ok := payload["models"].(map[string]any); ok {
		models := make([]string, 0, len(modelsObj))
		for name := range modelsObj {
			name = strings.TrimSpace(name)
			if name != "" {
				models = append(models, name)
			}
		}
		return normalizeModelList(models)
	}
	rawItems := extractModelItems(payload)
	models := make([]string, 0, len(rawItems))
	for _, raw := range rawItems {
		switch item := raw.(type) {
		case string:
			models = append(models, item)
		case map[string]any:
			models = append(models, firstModelField(item, "id", "slug", "model", "name"))
		}
	}
	return normalizeModelList(models)
}

func normalizeModelList(models []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" || seen[strings.ToLower(model)] {
			continue
		}
		seen[strings.ToLower(model)] = true
		result = append(result, model)
	}
	return result
}
