package common

import (
	"context"

	"github.com/QuantumNous/new-api/service/unitplatform"
)

func UnsupportedSnapshot(_ context.Context, platformType string) (*unitplatform.Snapshot, error) {
	return unitplatform.UnsupportedSnapshot(platformType)
}
