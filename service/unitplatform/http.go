package unitplatform

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

const MaxProbeBodyBytes = 512 * 1024

type HTTPResult struct {
	Status  int
	Headers string
	Body    []byte
}

func NormalizeBaseURL(rawURL string) string {
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

func NewClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 12 * time.Second
	}
	return &http.Client{Timeout: timeout}
}

func Probe(ctx context.Context, client *http.Client, method, targetURL string) HTTPResult {
	req, err := http.NewRequestWithContext(ctx, method, targetURL, nil)
	if err != nil {
		return HTTPResult{Status: 0, Body: []byte(err.Error())}
	}
	req.Header.Set("Accept", "application/json,text/html;q=0.8,*/*;q=0.5")
	req.Header.Set("User-Agent", "NACP-Unit-Platform-Detector/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return HTTPResult{Status: 0, Body: []byte(err.Error())}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, MaxProbeBodyBytes))
	return HTTPResult{
		Status:  resp.StatusCode,
		Headers: fmt.Sprintf("%v", resp.Header),
		Body:    body,
	}
}

func IsWAFChallenge(response HTTPResult) bool {
	headers := strings.ToLower(response.Headers)
	body := strings.ToLower(string(response.Body))
	return strings.Contains(headers, "cf-mitigated:[challenge") ||
		strings.Contains(body, "just a moment") ||
		strings.Contains(body, "challenges.cloudflare.com") ||
		strings.Contains(body, "cf-chl")
}

func DecodeJSONResponse(response HTTPResult) (map[string]interface{}, bool) {
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

func BuildAuthRequest(ctx context.Context, method, targetURL string, credentials Credentials, body []byte, userHeaders []string) (*http.Request, error) {
	reader := strings.NewReader("")
	if body != nil {
		reader = strings.NewReader(string(body))
	}
	req, err := http.NewRequestWithContext(ctx, method, targetURL, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	token := strings.TrimSpace(credentials.AccessToken)
	if token == "" {
		token = strings.TrimSpace(credentials.APIKey)
	}
	if token != "" {
		if strings.HasPrefix(strings.ToLower(token), "bearer ") {
			req.Header.Set("Authorization", token)
		} else {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	accountID := strings.TrimSpace(credentials.AccountID)
	for _, header := range userHeaders {
		if strings.TrimSpace(header) != "" && accountID != "" {
			req.Header.Set(header, accountID)
		}
	}
	return req, nil
}
