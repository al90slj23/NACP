package unitplatform

import (
	"fmt"
	"strings"
)

func NormalizeCredentials(credentials Credentials) Credentials {
	credentials.Platform = normalizeType(credentials.Platform)
	credentials.BaseURL = NormalizeBaseURL(credentials.BaseURL)
	credentials.AccessToken = strings.TrimSpace(credentials.AccessToken)
	credentials.AccountID = strings.TrimSpace(credentials.AccountID)
	credentials.APIKey = strings.TrimSpace(credentials.APIKey)
	credentials.Username = strings.TrimSpace(credentials.Username)
	credentials.Password = strings.TrimSpace(credentials.Password)
	return credentials
}

func RequirePanelCredentials(credentials Credentials) error {
	if strings.TrimSpace(credentials.AccessToken) == "" || strings.TrimSpace(credentials.AccountID) == "" {
		return fmt.Errorf("所属账号缺少账户访问令牌或账户 ID，无法监控余额")
	}
	return nil
}

func UnsupportedSnapshot(platformType string) (*Snapshot, error) {
	return nil, fmt.Errorf("当前单位类型 %s 暂不支持账号监控", platformType)
}
