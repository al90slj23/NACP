package service

import "testing"

func TestParseProbeUsageOpenAIFormat(t *testing.T) {
	usage := parseProbeUsage([]byte(`{"usage":{"prompt_tokens":12,"completion_tokens":1}}`))

	if usage.PromptTokens != 12 {
		t.Fatalf("PromptTokens = %d, want 12", usage.PromptTokens)
	}
	if usage.CompletionTokens != 1 {
		t.Fatalf("CompletionTokens = %d, want 1", usage.CompletionTokens)
	}
	if usage.totalTokens() != 13 {
		t.Fatalf("totalTokens = %d, want 13", usage.totalTokens())
	}
}

func TestParseProbeUsageAnthropicFormat(t *testing.T) {
	usage := parseProbeUsage([]byte(`{"usage":{"input_tokens":10,"output_tokens":1}}`))

	if usage.PromptTokens != 10 {
		t.Fatalf("PromptTokens = %d, want 10", usage.PromptTokens)
	}
	if usage.CompletionTokens != 1 {
		t.Fatalf("CompletionTokens = %d, want 1", usage.CompletionTokens)
	}
	if usage.totalTokens() != 11 {
		t.Fatalf("totalTokens = %d, want 11", usage.totalTokens())
	}
}

func TestParseProbeUsageCacheFields(t *testing.T) {
	body := []byte(`{"usage":{"prompt_tokens":20,"completion_tokens":2,"prompt_tokens_details":{"cached_tokens":7,"cached_creation_tokens":3}}}`)
	usage := parseProbeUsage(body)

	if usage.PromptTokens != 20 {
		t.Fatalf("PromptTokens = %d, want 20", usage.PromptTokens)
	}
	if usage.CompletionTokens != 2 {
		t.Fatalf("CompletionTokens = %d, want 2", usage.CompletionTokens)
	}
	if usage.CacheReadTokens != 7 {
		t.Fatalf("CacheReadTokens = %d, want 7", usage.CacheReadTokens)
	}
	if usage.CacheCreationTokens != 3 {
		t.Fatalf("CacheCreationTokens = %d, want 3", usage.CacheCreationTokens)
	}
}
