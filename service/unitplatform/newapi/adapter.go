package newapi

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

func (Adapter) Type() string { return "newapi" }

func (Adapter) Aliases() []string {
	return []string{"new-api", "new api", "newapi"}
}

func (Adapter) Capabilities() unitplatform.CapabilitySet { return unitplatform.PanelCapabilities() }

func (Adapter) CredentialSpec() unitplatform.CredentialSpec {
	return unitplatform.PanelCredentialSpec()
}

func (Adapter) Detect(ctx context.Context, client *http.Client, baseURL string) (unitplatform.Result, bool) {
	root := unitplatform.Probe(ctx, client, http.MethodGet, baseURL)
	if strings.Contains(strings.ToLower(root.Headers), "x-new-api-version:") {
		return unitplatform.Detected("newapi", baseURL, "检测到 New API 响应头 X-New-Api-Version。"), true
	}
	status := unitplatform.Probe(ctx, client, http.MethodGet, baseURL+"/api/status")
	if statusJSON, ok := unitplatform.DecodeJSONResponse(status); ok {
		data := upcommon.ExtractStatusData(statusJSON)
		if upcommon.HasAnyKey(data, "MjNotifyEnabled", "DataExportEnabled", "CheckinEnabled", "mj_notify_enabled", "enable_data_export", "checkin_enabled", "enable_drawing", "enable_task") {
			return unitplatform.Detected("newapi", baseURL, "检测到 New API 专有配置字段。"), true
		}
	}
	return unitplatform.Result{}, false
}

func (Adapter) FetchSnapshot(ctx context.Context, credentials unitplatform.Credentials) (*unitplatform.Snapshot, error) {
	return upcommon.FetchPanelSnapshot(ctx, credentials, upcommon.PanelSnapshotOptions{
		AdapterName: "newapi",
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
