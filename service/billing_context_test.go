package service

import (
	"context"
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

type fakeBillingSettler struct {
	preConsumed int
	settled     bool
	actualQuota int
}

func (f *fakeBillingSettler) Settle(actualQuota int) error {
	f.settled = true
	f.actualQuota = actualQuota
	return nil
}

func (f *fakeBillingSettler) Refund(_ *gin.Context) {}

func (f *fakeBillingSettler) NeedsRefund() bool {
	return !f.settled
}

func (f *fakeBillingSettler) GetPreConsumedQuota() int {
	return f.preConsumed
}

func (f *fakeBillingSettler) Reserve(targetQuota int) error {
	f.preConsumed = targetQuota
	return nil
}

func TestSettleBillingStillRunsAfterRequestContextDone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	reqCtx, cancel := context.WithCancel(context.Background())
	cancel()
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil).WithContext(reqCtx)

	billing := &fakeBillingSettler{}
	relayInfo := &relaycommon.RelayInfo{
		Billing: billing,
	}

	if err := SettleBilling(c, relayInfo, 0); err != nil {
		t.Fatalf("SettleBilling should not fail after a successful response just because request context is done: %v", err)
	}
	if !billing.settled {
		t.Fatal("expected billing settlement to run")
	}
	if WasBillingSkippedRequestCanceled(c) {
		t.Fatal("successful response settlement must not be marked as skipped client cancellation")
	}
}
