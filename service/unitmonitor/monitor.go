package unitmonitor

import (
	"context"
	"strings"

	"github.com/QuantumNous/new-api/service/unitplatform"
	_ "github.com/QuantumNous/new-api/service/unitplatform/all"
)

func Fetch(ctx context.Context, credentials Credentials) (*Snapshot, error) {
	credentials.Platform = strings.ToLower(strings.TrimSpace(credentials.Platform))
	return unitplatform.FetchSnapshot(ctx, unitplatform.Credentials(credentials))
}
