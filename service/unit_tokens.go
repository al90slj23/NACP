package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/unitplatform"
	_ "github.com/QuantumNous/new-api/service/unitplatform/all"
)

type UnitPlatformToken = unitplatform.UnitPlatformToken
type UnitPlatformTokenGroupOption = unitplatform.UnitPlatformTokenGroupOption
type UnitPlatformTokenOptions = unitplatform.UnitPlatformTokenOptions
type CreateUnitPlatformTokenRequest = unitplatform.CreateUnitPlatformTokenRequest

func FetchUnitPlatformTokens(ctx context.Context, unit *model.Unit, account *model.UnitAccount) ([]UnitPlatformToken, error) {
	credentials, err := resolveUnitPlatformCredentials(unit, account)
	if err != nil {
		return nil, err
	}
	return unitplatform.FetchTokens(ctx, credentials)
}

func FetchUnitPlatformTokenOptions(ctx context.Context, unit *model.Unit, account *model.UnitAccount) (*UnitPlatformTokenOptions, error) {
	credentials, err := resolveUnitPlatformCredentials(unit, account)
	if err != nil {
		return nil, err
	}
	return unitplatform.FetchTokenOptions(ctx, credentials)
}

func CreateUnitPlatformToken(ctx context.Context, unit *model.Unit, account *model.UnitAccount, req CreateUnitPlatformTokenRequest) (*UnitPlatformToken, error) {
	credentials, err := resolveUnitPlatformCredentials(unit, account)
	if err != nil {
		return nil, err
	}
	return unitplatform.CreateToken(ctx, credentials, req)
}

func resolveUnitPlatformCredentials(unit *model.Unit, account *model.UnitAccount) (unitplatform.Credentials, error) {
	if unit == nil || account == nil {
		return unitplatform.Credentials{}, fmt.Errorf("单位或账号不存在")
	}
	unit.Normalize()
	baseURL := unitplatform.NormalizeBaseURL(unit.EffectiveAPIURL)
	if baseURL == "" {
		baseURL = unitplatform.NormalizeBaseURL(unit.APIURL)
	}
	if baseURL == "" {
		baseURL = unitplatform.NormalizeBaseURL(unit.WebsiteURL)
	}
	if baseURL == "" {
		return unitplatform.Credentials{}, fmt.Errorf("单位 API 地址为空")
	}
	credentials := unitplatform.Credentials{
		Platform:    strings.ToLower(strings.TrimSpace(unit.Type)),
		BaseURL:     baseURL,
		AccessToken: strings.TrimSpace(account.AccessToken),
		AccountID:   strings.TrimSpace(account.AccountID),
		APIKey:      strings.TrimSpace(account.AccessToken),
		Username:    strings.TrimSpace(account.Account),
		Password:    strings.TrimSpace(account.Password),
	}
	if credentials.Platform == "" {
		credentials.Platform = model.UnitTypeNewAPI
	}
	return credentials, nil
}
