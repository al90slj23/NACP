package unitdetect

import "net/http"

func detectSub2API(settings httpResult, baseURL string) (Result, bool) {
	if settings.Status != http.StatusOK {
		return Result{}, false
	}
	if _, ok := decodeJSONResponse(settings); !ok {
		return Result{}, false
	}
	return detected("sub2api", baseURL, "检测到 sub2api 公共设置接口。"), true
}
