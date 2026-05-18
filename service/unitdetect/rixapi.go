package unitdetect

import "strings"

func detectRixAPI(root httpResult, baseURL string) (Result, bool) {
	headers := strings.ToLower(root.Headers)
	body := strings.ToLower(string(root.Body))
	if strings.Contains(headers, "x-rix-api-version:") ||
		strings.Contains(body, "rix-api") ||
		strings.Contains(body, "rix_api") {
		return detected("rixapi", baseURL, "检测到 Rix-API 特征。"), true
	}
	return Result{}, false
}
