package oneapifork

import (
	"context"
	"net/http"

	"github.com/QuantumNous/new-api/service/unitplatform"
	upcommon "github.com/QuantumNous/new-api/service/unitplatform/common"
)

type Adapter struct{}

func init() {
	unitplatform.Register(Adapter{})
}

func (Adapter) Type() string { return "oneapifork" }

func (Adapter) Aliases() []string { return []string{"one-api-fork", "oneapi-fork"} }

func (Adapter) Capabilities() unitplatform.CapabilitySet { return unitplatform.PanelCapabilities() }

func (Adapter) CredentialSpec() unitplatform.CredentialSpec {
	return unitplatform.PanelCredentialSpec()
}

func (Adapter) Detect(ctx context.Context, client *http.Client, baseURL string) (unitplatform.Result, bool) {
	self := unitplatform.Probe(ctx, client, http.MethodGet, baseURL+"/api/user/self")
	if _, ok := unitplatform.DecodeJSONResponse(self); ok {
		return unitplatform.Detected("oneapifork", baseURL, "检测到 One API 系兼容用户接口，但无法进一步区分具体分支。"), true
	}
	return unitplatform.Result{}, false
}

func (Adapter) FetchSnapshot(ctx context.Context, credentials unitplatform.Credentials) (*unitplatform.Snapshot, error) {
	return upcommon.FetchPanelSnapshot(ctx, credentials, upcommon.PanelSnapshotOptions{
		AdapterName: "oneapifork",
		UserHeaders: userHeaders(),
	})
}

func (Adapter) FetchTokens(ctx context.Context, credentials unitplatform.Credentials) ([]unitplatform.UnitPlatformToken, error) {
	return upcommon.FetchPanelTokens(ctx, credentials, upcommon.PanelTokenOptions{UserHeaders: userHeaders()})
}

func (Adapter) FetchTokenOptions(ctx context.Context, credentials unitplatform.Credentials) (*unitplatform.UnitPlatformTokenOptions, error) {
	return upcommon.FetchPanelTokenOptions(ctx, credentials, upcommon.PanelTokenOptions{UserHeaders: userHeaders()})
}

func (Adapter) CreateToken(ctx context.Context, credentials unitplatform.Credentials, req unitplatform.CreateUnitPlatformTokenRequest) (*unitplatform.UnitPlatformToken, error) {
	return upcommon.CreatePanelToken(ctx, credentials, req, upcommon.PanelTokenOptions{UserHeaders: userHeaders()})
}

func userHeaders() []string {
	return []string{"New-Api-User", "New-API-User"}
}
