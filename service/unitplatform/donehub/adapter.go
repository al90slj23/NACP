package donehub

import (
	"context"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/service/unitplatform"
	upcommon "github.com/QuantumNous/new-api/service/unitplatform/common"
)

type Adapter struct{}

func init() {
	unitplatform.Register(Adapter{})
}

func (Adapter) Type() string { return "donehub" }

func (Adapter) Aliases() []string { return []string{"done-hub", "done hub"} }

func (Adapter) Capabilities() unitplatform.CapabilitySet {
	capabilities := unitplatform.PanelCapabilities()
	capabilities[unitplatform.CapabilityCheckin] = false
	return capabilities
}

func (Adapter) CredentialSpec() unitplatform.CredentialSpec {
	return unitplatform.PanelCredentialSpec()
}

func (Adapter) Detect(_ context.Context, _ *http.Client, baseURL string) (unitplatform.Result, bool) {
	normalized := strings.ToLower(baseURL)
	if strings.Contains(normalized, "donehub") || strings.Contains(normalized, "done-hub") {
		return unitplatform.Detected("donehub", baseURL, "根据 URL 命中 DoneHub 特征。"), true
	}
	return unitplatform.Result{}, false
}

func (Adapter) FetchSnapshot(ctx context.Context, credentials unitplatform.Credentials) (*unitplatform.Snapshot, error) {
	return upcommon.FetchPanelSnapshot(ctx, credentials, upcommon.PanelSnapshotOptions{
		AdapterName: "donehub",
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
