package rixapi

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

func (Adapter) Type() string { return "rixapi" }

func (Adapter) Aliases() []string { return []string{"rix-api", "rix api"} }

func (Adapter) Capabilities() unitplatform.CapabilitySet { return unitplatform.PanelCapabilities() }

func (Adapter) CredentialSpec() unitplatform.CredentialSpec {
	return unitplatform.PanelCredentialSpec()
}

func (Adapter) Detect(ctx context.Context, client *http.Client, baseURL string) (unitplatform.Result, bool) {
	root := unitplatform.Probe(ctx, client, http.MethodGet, baseURL)
	headers := strings.ToLower(root.Headers)
	body := strings.ToLower(string(root.Body))
	if strings.Contains(headers, "x-rix-api-version:") || strings.Contains(body, "rix-api") || strings.Contains(body, "rix_api") {
		return unitplatform.Detected("rixapi", baseURL, "检测到 Rix-API 特征。"), true
	}
	status := unitplatform.Probe(ctx, client, http.MethodGet, baseURL+"/api/status")
	if statusJSON, ok := unitplatform.DecodeJSONResponse(status); ok {
		data := upcommon.ExtractStatusData(statusJSON)
		if upcommon.HasAnyKey(data, "rix_license_enabled", "rix_version_message", "rixapi_license_type") {
			return unitplatform.Detected("rixapi", baseURL, "检测到 RixAPI 专有状态字段。"), true
		}
	}
	return unitplatform.Result{}, false
}

func (Adapter) FetchSnapshot(ctx context.Context, credentials unitplatform.Credentials) (*unitplatform.Snapshot, error) {
	return upcommon.FetchPanelSnapshot(ctx, credentials, upcommon.PanelSnapshotOptions{
		AdapterName: "rixapi",
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
	return []string{"New-Api-User", "New-API-User", "Rix-Api-User"}
}
