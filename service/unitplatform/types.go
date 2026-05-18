package unitplatform

import (
	"context"
	"net/http"
)

const (
	CapabilityDetect       = "detect"
	CapabilitySnapshot     = "snapshot"
	CapabilityTokens       = "tokens"
	CapabilityCreateToken  = "create_token"
	CapabilityTokenOptions = "token_options"
	CapabilityModels       = "models"
	CapabilityBalance      = "balance"
	CapabilityGroups       = "groups"
	CapabilityPricing      = "pricing"
	CapabilityCheckin      = "checkin"
	CapabilityOAuth        = "oauth"
)

type Result struct {
	OK      bool   `json:"ok"`
	Type    string `json:"type"`
	BaseURL string `json:"base_url"`
	Message string `json:"message"`
}

type CapabilitySet map[string]bool

type CredentialField struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Required    bool   `json:"required"`
	Secret      bool   `json:"secret"`
	Description string `json:"description,omitempty"`
}

type CredentialSpec struct {
	Mode   string            `json:"mode"`
	Fields []CredentialField `json:"fields"`
}

type Credentials struct {
	Platform    string
	BaseURL     string
	AccessToken string
	AccountID   string
	APIKey      string
	Username    string
	Password    string
}

type Snapshot struct {
	Platform         string         `json:"platform"`
	Balance          float64        `json:"balance"`
	Used             float64        `json:"used"`
	BalanceUnit      string         `json:"balance_unit"`
	UpstreamUserID   string         `json:"upstream_user_id"`
	UpstreamUsername string         `json:"upstream_username"`
	UpstreamGroup    string         `json:"upstream_group"`
	TokenCount       int            `json:"token_count"`
	ModelCount       int            `json:"model_count"`
	GroupCount       int            `json:"group_count"`
	Raw              map[string]any `json:"raw"`
}

type UnitPlatformToken struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Key                string `json:"key"`
	Group              string `json:"group"`
	Status             int    `json:"status"`
	ExpiredTime        int64  `json:"expired_time"`
	RemainQuota        int    `json:"remain_quota"`
	UnlimitedQuota     bool   `json:"unlimited_quota"`
	ModelLimitsEnabled bool   `json:"model_limits_enabled"`
	ModelLimits        string `json:"model_limits"`
}

type UnitPlatformTokenGroupOption struct {
	Value string  `json:"value"`
	Label string  `json:"label"`
	Ratio float64 `json:"ratio"`
}

type UnitPlatformTokenOptions struct {
	Groups []UnitPlatformTokenGroupOption `json:"groups"`
	Models []string                       `json:"models"`
}

type CreateUnitPlatformTokenRequest struct {
	Name               string  `json:"name"`
	ExpiredTime        int64   `json:"expired_time"`
	RemainQuota        int     `json:"remain_quota"`
	UnlimitedQuota     bool    `json:"unlimited_quota"`
	ModelLimitsEnabled bool    `json:"model_limits_enabled"`
	ModelLimits        string  `json:"model_limits"`
	AllowIps           *string `json:"allow_ips"`
	Group              string  `json:"group"`
	CrossGroupRetry    bool    `json:"cross_group_retry"`
}

type Adapter interface {
	Type() string
	Aliases() []string
	Capabilities() CapabilitySet
	CredentialSpec() CredentialSpec
	Detect(ctx context.Context, client *http.Client, baseURL string) (Result, bool)
	FetchSnapshot(ctx context.Context, credentials Credentials) (*Snapshot, error)
}

type TokenAdapter interface {
	FetchTokens(ctx context.Context, credentials Credentials) ([]UnitPlatformToken, error)
	FetchTokenOptions(ctx context.Context, credentials Credentials) (*UnitPlatformTokenOptions, error)
	CreateToken(ctx context.Context, credentials Credentials, req CreateUnitPlatformTokenRequest) (*UnitPlatformToken, error)
}

func Detected(platformType, baseURL, message string) Result {
	return Result{
		OK:      true,
		Type:    platformType,
		BaseURL: baseURL,
		Message: message,
	}
}

func Failed(platformType, baseURL, message string) Result {
	return Result{
		OK:      false,
		Type:    platformType,
		BaseURL: baseURL,
		Message: message,
	}
}

func PanelCredentialSpec() CredentialSpec {
	return CredentialSpec{
		Mode: "session",
		Fields: []CredentialField{
			{Name: "access_token", Label: "账户访问令牌", Required: true, Secret: true},
			{Name: "account_id", Label: "账户 ID", Required: true},
		},
	}
}

func APIKeyCredentialSpec() CredentialSpec {
	return CredentialSpec{
		Mode: "apikey",
		Fields: []CredentialField{
			{Name: "api_key", Label: "API Key", Required: true, Secret: true},
		},
	}
}

func OAuthCredentialSpec() CredentialSpec {
	return CredentialSpec{
		Mode: "oauth",
		Fields: []CredentialField{
			{Name: "access_token", Label: "Access Token", Required: true, Secret: true},
		},
	}
}

func PanelCapabilities() CapabilitySet {
	return CapabilitySet{
		CapabilityDetect:       true,
		CapabilitySnapshot:     true,
		CapabilityTokens:       true,
		CapabilityCreateToken:  true,
		CapabilityTokenOptions: true,
		CapabilityModels:       true,
		CapabilityBalance:      true,
		CapabilityGroups:       true,
	}
}

func ModelsOnlyCapabilities() CapabilitySet {
	return CapabilitySet{
		CapabilityDetect: true,
		CapabilityModels: true,
	}
}
