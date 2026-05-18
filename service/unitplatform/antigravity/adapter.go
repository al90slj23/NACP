package antigravity

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

func (Adapter) Type() string { return "antigravity" }

func (Adapter) Aliases() []string { return []string{"anti-gravity"} }

func (Adapter) Capabilities() unitplatform.CapabilitySet {
	capabilities := unitplatform.ModelsOnlyCapabilities()
	capabilities[unitplatform.CapabilityOAuth] = true
	return capabilities
}

func (Adapter) CredentialSpec() unitplatform.CredentialSpec {
	spec := unitplatform.OAuthCredentialSpec()
	spec.Fields = append(spec.Fields, unitplatform.CredentialField{
		Name:        "account_id",
		Label:       "项目 ID",
		Required:    false,
		Description: "Antigravity fetchAvailableModels 可携带 project 字段；为空时使用空请求体探测。",
	})
	return spec
}

func (Adapter) Detect(_ context.Context, _ *http.Client, baseURL string) (unitplatform.Result, bool) {
	if strings.Contains(strings.ToLower(baseURL), "antigravity") {
		return unitplatform.Detected("antigravity", baseURL, "根据 URL 命中 Antigravity API。"), true
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
	models, err := upcommon.FetchAntigravityModels(ctx, credentials.BaseURL, credentials.AccessToken, credentials.AccountID)
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
	return fmt.Errorf("当前单位类型 %s 是 OAuth/CLI 直连平台，不支持平台令牌管理", platform)
}
