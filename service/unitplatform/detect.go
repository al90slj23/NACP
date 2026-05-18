package unitplatform

import (
	"context"
	"net/http"
	"time"
)

func Detect(ctx context.Context, siteURL string) Result {
	if siteURL == "" {
		return Failed("", "", "请先填写单位网站地址。")
	}
	baseURL := NormalizeBaseURL(siteURL)
	if baseURL == "" {
		return Failed("", "", "单位网站地址为空或格式无效。")
	}

	client := NewClient(12 * time.Second)
	probeCtx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()

	root := Probe(probeCtx, client, http.MethodGet, baseURL)
	blocked := IsWAFChallenge(root)
	for _, adapter := range Adapters() {
		result, ok := adapter.Detect(probeCtx, client, baseURL)
		if ok {
			return result
		}
	}

	if blocked {
		return Failed("unknown", baseURL, "检测请求被 Cloudflare/WAF 挑战页拦截，无法判断类型。")
	}
	return Detected("newapi", baseURL, "未发现已知独立平台特征，暂按 newapi。")
}
