package openai

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

func (Adapter) Type() string { return "openai" }

func (Adapter) Aliases() []string { return nil }

func (Adapter) Capabilities() unitplatform.CapabilitySet {
	return unitplatform.ModelsOnlyCapabilities()
}

func (Adapter) CredentialSpec() unitplatform.CredentialSpec {
	return unitplatform.APIKeyCredentialSpec()
}

func (Adapter) Detect(_ context.Context, _ *http.Client, baseURL string) (unitplatform.Result, bool) {
	if strings.Contains(strings.ToLower(baseURL), "api.openai.com") {
		return unitplatform.Detected("openai", baseURL, "根据 URL 命中 OpenAI API。"), true
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
