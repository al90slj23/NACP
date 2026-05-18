package unitmonitor

import (
	"context"
	"fmt"
)

func fetchSub2API(ctx context.Context, credentials Credentials) (*Snapshot, error) {
	return nil, fmt.Errorf("sub2api 已单独分组，但余额监控适配尚未接入；需要确认该平台可用的账号余额接口")
}
