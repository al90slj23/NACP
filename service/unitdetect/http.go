package unitdetect

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const maxProbeBodyBytes = 512 * 1024

func normalizeBaseURL(rawURL string) string {
	rawURL = strings.Join(strings.Fields(strings.TrimSpace(rawURL)), "")
	if rawURL == "" {
		return ""
	}
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	if parsed.Path == "/" {
		parsed.Path = ""
	}
	return strings.TrimRight(parsed.String(), "/")
}

func request(ctx context.Context, client *http.Client, method, targetURL string) httpResult {
	req, err := http.NewRequestWithContext(ctx, method, targetURL, nil)
	if err != nil {
		return httpResult{Status: 0, Body: []byte(err.Error())}
	}
	req.Header.Set("Accept", "application/json,text/html;q=0.8,*/*;q=0.5")
	req.Header.Set("User-Agent", "NACP-Unit-Type-Detector/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return httpResult{Status: 0, Body: []byte(err.Error())}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxProbeBodyBytes))
	return httpResult{
		Status:  resp.StatusCode,
		Headers: fmt.Sprintf("%v", resp.Header),
		Body:    body,
	}
}

func isWAFChallenge(response httpResult) bool {
	headers := strings.ToLower(response.Headers)
	body := strings.ToLower(string(response.Body))
	return strings.Contains(headers, "cf-mitigated:[challenge") ||
		strings.Contains(body, "just a moment") ||
		strings.Contains(body, "challenges.cloudflare.com") ||
		strings.Contains(body, "cf-chl")
}

func decodeJSONResponse(response httpResult) (map[string]interface{}, bool) {
	body := strings.TrimSpace(string(response.Body))
	if body == "" || strings.HasPrefix(body, "<") {
		return nil, false
	}
	var data map[string]interface{}
	if err := common.Unmarshal(response.Body, &data); err != nil {
		return nil, false
	}
	return data, true
}

func newClient() *http.Client {
	return &http.Client{Timeout: 12 * time.Second}
}
