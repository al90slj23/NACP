package cliproxyapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/service/unitplatform"
	upcommon "github.com/QuantumNous/new-api/service/unitplatform/common"
)

type Adapter struct{}

func init() {
	unitplatform.Register(Adapter{})
}

func (Adapter) Type() string { return "cliproxyapi" }

func (Adapter) Aliases() []string { return []string{"cpa", "cli-proxy-api"} }

func (Adapter) Capabilities() unitplatform.CapabilitySet {
	return unitplatform.ModelsOnlyCapabilities()
}

func (Adapter) CredentialSpec() unitplatform.CredentialSpec {
	return unitplatform.APIKeyCredentialSpec()
}

func (Adapter) Detect(ctx context.Context, client *http.Client, baseURL string) (unitplatform.Result, bool) {
	normalized := strings.ToLower(baseURL)
	if strings.Contains(normalized, "cliproxy") || strings.Contains(normalized, ":8317") {
		return unitplatform.Detected("cliproxyapi", baseURL, "根据 URL 命中 CLIProxyAPI。"), true
	}
	probe := unitplatform.Probe(ctx, client, http.MethodGet, baseURL+"/v0/management/openai-compatibility")
	headers := strings.ToLower(probe.Headers)
	body := strings.ToLower(string(probe.Body))
	if strings.Contains(headers, "x-cpa-version") ||
		strings.Contains(headers, "x-cpa-commit") ||
		strings.Contains(headers, "x-cpa-build-date") ||
		strings.Contains(body, "openai-compatibility") {
		return unitplatform.Detected("cliproxyapi", baseURL, "检测到 CLIProxyAPI 管理接口特征。"), true
	}
	return unitplatform.Result{}, false
}

func (Adapter) FetchSnapshot(ctx context.Context, credentials unitplatform.Credentials) (*unitplatform.Snapshot, error) {
	return upcommon.UnsupportedSnapshot(ctx, credentials.Platform)
}

func (Adapter) FetchTokens(_ context.Context, credentials unitplatform.Credentials) ([]unitplatform.UnitPlatformToken, error) {
	return nil, unsupportedTokenManagement(credentials.Platform)
}

func (Adapter) FetchTokenOptions(ctx context.Context, credentials unitplatform.Credentials) (*unitplatform.UnitPlatformTokenOptions, error) {
	models, err := upcommon.FetchOpenAICompatibleModels(ctx, credentials.BaseURL, map[string]string{
		"Authorization": "Bearer " + strings.TrimSpace(credentials.APIKey),
	})
	if err != nil {
		return nil, err
	}
	return &unitplatform.UnitPlatformTokenOptions{
		Groups: []unitplatform.UnitPlatformTokenGroupOption{},
		Models: models,
	}, nil
}

func (Adapter) CreateToken(_ context.Context, credentials unitplatform.Credentials, _ unitplatform.CreateUnitPlatformTokenRequest) (*unitplatform.UnitPlatformToken, error) {
	return nil, unsupportedTokenManagement(credentials.Platform)
}

func unsupportedTokenManagement(platform string) error {
	return fmt.Errorf("当前单位类型 %s 是 API Key 直连平台，不支持平台令牌管理", platform)
}
