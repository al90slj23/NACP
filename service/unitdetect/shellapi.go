package unitdetect

import "strings"

func detectShellAPIHeader(response httpResult, baseURL string) (Result, bool) {
	if strings.Contains(strings.ToLower(response.Headers), "x-shellapi-request-id:") {
		return detected("shellapi", baseURL, "检测到 Shell API 响应头 X-Shellapi-Request-Id。"), true
	}
	return Result{}, false
}

func detectShellAPIStatusData(statusData map[string]interface{}, baseURL string) (Result, bool) {
	if strings.TrimSpace(asString(statusData["system_name"])) == "Shell API" ||
		hasAnyKey(statusData, "ShellApiLogOptimizerEnabled", "SwitchUIEnabled") {
		return detected("shellapi", baseURL, "检测到 Shell API 专有配置字段。"), true
	}
	return Result{}, false
}
