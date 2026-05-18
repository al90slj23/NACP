package shellapi

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

func (Adapter) Type() string { return "shellapi" }

func (Adapter) Aliases() []string { return []string{"shell-api", "shell api"} }

func (Adapter) Capabilities() unitplatform.CapabilitySet { return unitplatform.PanelCapabilities() }

func (Adapter) CredentialSpec() unitplatform.CredentialSpec {
	return unitplatform.PanelCredentialSpec()
}

func (Adapter) Detect(ctx context.Context, client *http.Client, baseURL string) (unitplatform.Result, bool) {
	root := unitplatform.Probe(ctx, client, http.MethodGet, baseURL)
	if strings.Contains(strings.ToLower(root.Headers), "x-shellapi-request-id:") {
		return unitplatform.Detected("shellapi", baseURL, "检测到 Shell API 响应头 X-Shellapi-Request-Id。"), true
	}
	status := unitplatform.Probe(ctx, client, http.MethodGet, baseURL+"/api/status")
	if statusJSON, ok := unitplatform.DecodeJSONResponse(status); ok {
		data := upcommon.ExtractStatusData(statusJSON)
		systemName := strings.TrimSpace(upcommon.AsString(data["system_name"]))
		version := strings.TrimSpace(upcommon.AsString(data["version"]))
		if systemName == "Shell API" ||
			strings.Contains(strings.ToLower(systemName), "shellapi") ||
			(strings.HasPrefix(version, "v") && strings.Contains(strings.ToLower(version), "alpha")) ||
			upcommon.HasAnyKey(data, "ShellApiLogOptimizerEnabled", "SwitchUIEnabled", "CustomThemeConfig", "DataExportInterval", "instanceId", "PureHomePageEnabled") {
			return unitplatform.Detected("shellapi", baseURL, "检测到 Shell API 专有配置字段。"), true
		}
	}
	return unitplatform.Result{}, false
}

func (Adapter) FetchSnapshot(ctx context.Context, credentials unitplatform.Credentials) (*unitplatform.Snapshot, error) {
	return upcommon.FetchPanelSnapshot(ctx, credentials, upcommon.PanelSnapshotOptions{
		AdapterName: "shellapi",
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
