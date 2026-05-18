package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
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
