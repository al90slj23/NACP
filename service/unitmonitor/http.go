package unitmonitor

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

type selfResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

type tokenPageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Items []UnitPlatformToken `json:"items"`
		Total int                 `json:"total"`
	} `json:"data"`
}

type modelsResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Data    []string `json:"data"`
}

type groupsResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    map[string]struct {
		Desc  string  `json:"desc"`
		Ratio float64 `json:"ratio"`
	} `json:"data"`
}

func newRequest(ctx context.Context, method, targetURL, accessToken, accountID string, body []byte) (*http.Request, error) {
	reader := bytes.NewReader(body)
	req, err := http.NewRequestWithContext(ctx, method, targetURL, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("New-Api-User", accountID)
	return req, nil
}

func normalizeBaseURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	value = strings.TrimRight(value, "/")
	if strings.HasPrefix(strings.ToLower(value), "http://") || strings.HasPrefix(strings.ToLower(value), "https://") {
		return value
	}
	return "https://" + value
}

func extractMoneyValue(data map[string]any, keys ...string) float64 {
	value := extractPlainFloat(data, keys...)
	if value == 0 {
		return 0
	}
	return value / common.QuotaPerUnit
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
