package oneapi

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

func (Adapter) Type() string { return "oneapi" }

func (Adapter) Aliases() []string { return []string{"one-api", "one api"} }

func (Adapter) Capabilities() unitplatform.CapabilitySet { return unitplatform.PanelCapabilities() }

func (Adapter) CredentialSpec() unitplatform.CredentialSpec {
	return unitplatform.PanelCredentialSpec()
}

func (Adapter) Detect(ctx context.Context, client *http.Client, baseURL string) (unitplatform.Result, bool) {
	status := unitplatform.Probe(ctx, client, http.MethodGet, baseURL+"/api/status")
	if status.Status != http.StatusOK {
		return unitplatform.Result{}, false
	}
	statusJSON, ok := unitplatform.DecodeJSONResponse(status)
	if !ok {
		return unitplatform.Result{}, false
	}
	data := upcommon.ExtractStatusData(statusJSON)
	sysName := strings.TrimSpace(upcommon.AsString(data["system_name"]))
	if sysName == "One API" || sysName == "one-api" {
		return unitplatform.Detected("oneapi", baseURL, "检测到 One API 原版 system_name。"), true
	}
	if _, hasSystemName := data["system_name"]; !hasSystemName {
		if success, ok := statusJSON["success"].(bool); ok && success {
			return unitplatform.Detected("oneapi", baseURL, "检测到 One API 原版状态结构。"), true
		}
	}
	return unitplatform.Result{}, false
}

func (Adapter) FetchSnapshot(ctx context.Context, credentials unitplatform.Credentials) (*unitplatform.Snapshot, error) {
	return upcommon.FetchPanelSnapshot(ctx, credentials, upcommon.PanelSnapshotOptions{
		AdapterName: "oneapi",
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
