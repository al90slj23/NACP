package unitplatform

import (
	"context"
	"fmt"
)

func FetchSnapshot(ctx context.Context, credentials Credentials) (*Snapshot, error) {
	credentials = NormalizeCredentials(credentials)
	if credentials.BaseURL == "" {
		return nil, fmt.Errorf("单位 API 地址为空")
	}
	adapter, err := MustGet(credentials.Platform)
	if err != nil {
		return nil, err
	}
	return adapter.FetchSnapshot(ctx, credentials)
}
