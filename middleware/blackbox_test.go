package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

func TestBlackboxLoginGateMasksWithoutHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldProfile := common.SecurityProfile
	oldLoginPath := common.BlackboxLoginPath
	t.Cleanup(func() {
		common.SecurityProfile = oldProfile
		common.BlackboxLoginPath = oldLoginPath
	})
	common.SecurityProfile = common.SecurityProfileBlackbox
	common.BlackboxLoginPath = "/hidden-login"

	r := gin.New()
	r.POST("/api/user/login", BlackboxLoginGate(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected blackbox login gate to mask missing header with 404, got %d", w.Code)
	}
}

func TestBlackboxLoginGateAllowsExpectedHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldProfile := common.SecurityProfile
	oldLoginPath := common.BlackboxLoginPath
	t.Cleanup(func() {
		common.SecurityProfile = oldProfile
		common.BlackboxLoginPath = oldLoginPath
	})
	common.SecurityProfile = common.SecurityProfileBlackbox
	common.BlackboxLoginPath = "/hidden-login"

	r := gin.New()
	r.POST("/api/user/login", BlackboxLoginGate(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/login", nil)
	req.Header.Set(BlackboxLoginHeader, "/hidden-login")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected blackbox login gate to allow expected header, got %d", w.Code)
	}
}

func TestSetSelectedAutoGroupContextRecordsGroupAndIndex(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)

	setSelectedAutoGroupContext(ctx, []string{"default", "vip", "trial"}, "vip")

	group := common.GetContextKeyString(ctx, constant.ContextKeyAutoGroup)
	if group != "vip" {
		t.Fatalf("expected selected auto group vip, got %q", group)
	}
	indexAny, ok := common.GetContextKey(ctx, constant.ContextKeyAutoGroupIndex)
	if !ok {
		t.Fatalf("expected selected auto group index to be set")
	}
	index, ok := indexAny.(int)
	if !ok || index != 1 {
		t.Fatalf("expected selected auto group index 1, got %#v", indexAny)
	}
}
