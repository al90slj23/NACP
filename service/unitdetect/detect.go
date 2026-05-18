package unitdetect

import (
	"context"

	"github.com/QuantumNous/new-api/service/unitplatform"
	_ "github.com/QuantumNous/new-api/service/unitplatform/all"
)

func Detect(ctx context.Context, siteURL string) Result {
	result := unitplatform.Detect(ctx, siteURL)
	return Result(result)
}
