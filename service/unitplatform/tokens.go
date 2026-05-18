package unitplatform

import (
	"context"
	"fmt"
	"strings"
)

func FetchTokens(ctx context.Context, credentials Credentials) ([]UnitPlatformToken, error) {
	adapter, credentials, err := tokenAdapterFor(credentials)
	if err != nil {
		return nil, err
	}
	return adapter.FetchTokens(ctx, credentials)
}

func FetchTokenOptions(ctx context.Context, credentials Credentials) (*UnitPlatformTokenOptions, error) {
	adapter, credentials, err := tokenAdapterFor(credentials)
	if err != nil {
		return nil, err
	}
	return adapter.FetchTokenOptions(ctx, credentials)
}

func CreateToken(ctx context.Context, credentials Credentials, req CreateUnitPlatformTokenRequest) (*UnitPlatformToken, error) {
	adapter, credentials, err := tokenAdapterFor(credentials)
	if err != nil {
		return nil, err
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, fmt.Errorf("令牌名称不能为空")
	}
	if req.ExpiredTime == 0 {
		req.ExpiredTime = -1
	}
	if req.UnlimitedQuota {
		req.RemainQuota = 0
	}
	req.ModelLimits = strings.TrimSpace(req.ModelLimits)
	req.ModelLimitsEnabled = req.ModelLimits != ""
	req.Group = strings.TrimSpace(req.Group)
	if req.AllowIps != nil {
		trimmed := strings.TrimSpace(*req.AllowIps)
		req.AllowIps = &trimmed
	}
	return adapter.CreateToken(ctx, credentials, req)
}

func tokenAdapterFor(credentials Credentials) (TokenAdapter, Credentials, error) {
	credentials = NormalizeCredentials(credentials)
	if credentials.BaseURL == "" {
		return nil, credentials, fmt.Errorf("单位 API 地址为空")
	}
	adapter, err := MustGet(credentials.Platform)
	if err != nil {
		return nil, credentials, err
	}
	tokenAdapter, ok := adapter.(TokenAdapter)
	if !ok {
		return nil, credentials, fmt.Errorf("当前单位类型 %s 暂不支持平台令牌管理", credentials.Platform)
	}
	return tokenAdapter, credentials, nil
}
