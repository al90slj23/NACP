package unitdetect

import "testing"

func TestDetectPlatformFeatureRules(t *testing.T) {
	t.Run("normalize host typo whitespace", func(t *testing.T) {
		got := normalizeBaseURL("528ai. cc")
		if got != "https://528ai.cc" {
			t.Fatalf("expected normalized URL, got %q", got)
		}
	})

	t.Run("newapi header", func(t *testing.T) {
		result, ok := detectNewAPIHeader(httpResult{
			Headers: "map[X-New-Api-Version:[v1.0.0-rc.4]]",
		}, "https://example.com")
		if !ok || result.Type != "newapi" {
			t.Fatalf("expected newapi, got ok=%v result=%#v", ok, result)
		}
	})

	t.Run("shellapi header", func(t *testing.T) {
		result, ok := detectShellAPIHeader(httpResult{
			Headers: "map[X-Shellapi-Request-Id:[test]]",
		}, "https://example.com")
		if !ok || result.Type != "shellapi" {
			t.Fatalf("expected shellapi, got ok=%v result=%#v", ok, result)
		}
	})

	t.Run("sub2api public settings", func(t *testing.T) {
		result, ok := detectSub2API(httpResult{
			Status: 200,
			Body:   []byte(`{"site":"sub2api"}`),
		}, "https://example.com")
		if !ok || result.Type != "sub2api" {
			t.Fatalf("expected sub2api, got ok=%v result=%#v", ok, result)
		}
	})

	t.Run("newapi status fields", func(t *testing.T) {
		result, ok := detectNewAPIStatusData(map[string]interface{}{
			"enable_task": true,
		}, "https://example.com")
		if !ok || result.Type != "newapi" {
			t.Fatalf("expected newapi, got ok=%v result=%#v", ok, result)
		}
	})

	t.Run("oneapi system name", func(t *testing.T) {
		result, ok := detectOneAPIStatusData(map[string]interface{}{
			"system_name": "One API",
		}, "https://example.com")
		if !ok || result.Type != "oneapi" {
			t.Fatalf("expected oneapi, got ok=%v result=%#v", ok, result)
		}
	})
}
