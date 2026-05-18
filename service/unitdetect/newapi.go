package unitdetect

import "strings"

func detectNewAPIHeader(response httpResult, baseURL string) (Result, bool) {
	if strings.Contains(strings.ToLower(response.Headers), "x-new-api-version:") {
		return detected("newapi", baseURL, "检测到 New API 响应头 X-New-Api-Version。"), true
	}
	return Result{}, false
}

func detectNewAPIStatusData(statusData map[string]interface{}, baseURL string) (Result, bool) {
	if hasAnyKey(
		statusData,
		"MjNotifyEnabled",
		"DataExportEnabled",
		"CheckinEnabled",
		"mj_notify_enabled",
		"enable_data_export",
		"checkin_enabled",
		"enable_drawing",
		"enable_task",
	) {
		return detected("newapi", baseURL, "检测到 New API 专有配置字段。"), true
	}
	return Result{}, false
}
