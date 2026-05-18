package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

func TestBlackboxLoginPageSupportsTwoFAFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldLoginPath := common.BlackboxLoginPath
	oldTurnstileEnabled := common.TurnstileCheckEnabled
	oldTurnstileSiteKey := common.TurnstileSiteKey
	t.Cleanup(func() {
		common.BlackboxLoginPath = oldLoginPath
		common.TurnstileCheckEnabled = oldTurnstileEnabled
		common.TurnstileSiteKey = oldTurnstileSiteKey
	})
	common.BlackboxLoginPath = "/hidden-login"
	common.TurnstileCheckEnabled = true
	common.TurnstileSiteKey = "site-key"

	r := gin.New()
	r.GET("/hidden-login", BlackboxLoginPage)

	req := httptest.NewRequest(http.MethodGet, "/hidden-login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected hidden login page 200, got %d", w.Code)
	}
	body := w.Body.String()
	for _, expected := range []string{
		`X-Login-Path`,
		`/api/user/login?turnstile=`,
		`/api/user/login/2fa`,
		`Verification code`,
		`data-sitekey="site-key"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("hidden login page missing %q", expected)
		}
	}
	for _, forbidden := range []string{`X-NACP-Blackbox-Login`, `NACP`, `new-api`, `QuantumNous`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("hidden login page leaked %q", forbidden)
		}
	}
}
