package common

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	projectcommon "github.com/QuantumNous/new-api/common"
)

func FetchOpenAICompatibleModels(ctx context.Context, baseURL string, headers map[string]string) ([]string, error) {
	return fetchModelPayload(ctx, resolveVersionedModelsURL(baseURL), headers)
}

func FetchClaudeModels(ctx context.Context, baseURL string, apiKey string) ([]string, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("缺少 Claude API Key")
	}
	models, err := FetchOpenAICompatibleModels(ctx, baseURL, map[string]string{
		"x-api-key":         apiKey,
		"anthropic-version": "2023-06-01",
	})
	if err == nil && len(models) > 0 {
		return models, nil
	}
	normalized := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(strings.ToLower(normalized), "/anthropic") {
		return FetchOpenAICompatibleModels(ctx, strings.TrimRight(normalized[:len(normalized)-len("/anthropic")], "/"), map[string]string{
			"Authorization": "Bearer " + apiKey,
		})
	}
	return models, err
}

func FetchGeminiModels(ctx context.Context, baseURL string, apiKey string) ([]string, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("缺少 Gemini API Key")
	}
	normalized := strings.TrimRight(baseURL, "/")
	if strings.Contains(strings.ToLower(normalized), "/openai") {
		models, err := FetchOpenAICompatibleModels(ctx, normalized, map[string]string{
			"Authorization": "Bearer " + apiKey,
		})
		if err == nil && len(models) > 0 {
			return normalizeGeminiModelNames(models), nil
		}
	}

	nativeURL := resolveGeminiNativeModelsURL(normalized, apiKey)
	models, err := fetchModelPayload(ctx, nativeURL, nil)
	if err == nil && len(models) > 0 {
		return normalizeGeminiModelNames(models), nil
	}

	models, err = FetchOpenAICompatibleModels(ctx, strings.TrimRight(normalized, "/")+"/v1beta/openai", map[string]string{
		"Authorization": "Bearer " + apiKey,
	})
	if err != nil {
		return nil, err
	}
	return normalizeGeminiModelNames(models), nil
}

func fetchModelPayload(ctx context.Context, targetURL string, headers map[string]string) ([]string, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	for key, value := range headers {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(value) != "" {
			req.Header.Set(key, value)
		}
	}
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
	models := extractModelIDs(payload)
	if len(models) == 0 {
		return nil, fmt.Errorf("读取模型失败：响应中没有模型列表")
	}
	return models, nil
}

func resolveVersionedModelsURL(baseURL string) string {
	normalized := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(strings.ToLower(normalized), "/models") {
		return normalized
	}
	if looksVersionedBase(normalized) {
		return normalized + "/models"
	}
	return normalized + "/v1/models"
}

func looksVersionedBase(baseURL string) bool {
	lower := strings.ToLower(strings.TrimRight(baseURL, "/"))
	return strings.HasSuffix(lower, "/v1") ||
		strings.HasSuffix(lower, "/v1beta") ||
		strings.HasSuffix(lower, "/v1alpha") ||
		strings.Contains(lower, "/v1beta/openai")
}

func resolveGeminiNativeModelsURL(baseURL string, apiKey string) string {
	normalized := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(strings.ToLower(normalized), "/models") {
		return appendQueryKey(normalized, apiKey)
	}
	if !strings.Contains(strings.ToLower(normalized), "/v1") {
		normalized += "/v1beta"
	}
	return appendQueryKey(strings.TrimRight(normalized, "/")+"/models", apiKey)
}

func appendQueryKey(targetURL string, apiKey string) string {
	separator := "?"
	if strings.Contains(targetURL, "?") {
		separator = "&"
	}
	return targetURL + separator + "key=" + apiKey
}

func extractModelIDs(payload map[string]any) []string {
	rawItems := extractModelItems(payload)
	seen := map[string]bool{}
	models := make([]string, 0, len(rawItems))
	for _, raw := range rawItems {
		model := ""
		switch item := raw.(type) {
		case string:
			model = item
		case map[string]any:
			model = firstModelField(item, "id", "name", "model")
		}
		model = strings.TrimSpace(model)
		if model == "" || seen[model] {
			continue
		}
		seen[model] = true
		models = append(models, model)
	}
	return models
}

func extractModelItems(payload map[string]any) []any {
	if payload == nil {
		return nil
	}
	for _, key := range []string{"data", "models", "items", "list"} {
		if items, ok := payload[key].([]any); ok {
			return items
		}
	}
	if data, ok := payload["data"].(map[string]any); ok {
		return extractModelItems(data)
	}
	return nil
}

func firstModelField(data map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := data[key]; ok && value != nil {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" && text != "<nil>" {
				return text
			}
		}
	}
	return ""
}

func normalizeGeminiModelNames(models []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(models))
	for _, model := range models {
		model = strings.TrimPrefix(strings.TrimSpace(model), "models/")
		if model == "" || seen[model] {
			continue
		}
		seen[model] = true
		result = append(result, model)
	}
	return result
}
