package unitdetect

import "strings"

func detectOneAPIStatusData(statusData map[string]interface{}, baseURL string) (Result, bool) {
	sysName := strings.TrimSpace(asString(statusData["system_name"]))
	if sysName == "One API" || sysName == "one-api" {
		return detected("oneapi", baseURL, "检测到 One API 原版 system_name。"), true
	}
	return Result{}, false
}

func detectOneAPIForkUser(self httpResult, baseURL string) (Result, bool) {
	if _, ok := decodeJSONResponse(self); ok {
		return detected("oneapifork", baseURL, "检测到 One API 系兼容用户接口，但无法进一步区分具体分支。"), true
	}
	return Result{}, false
}
