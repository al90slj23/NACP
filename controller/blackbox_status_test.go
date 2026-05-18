package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestGetStatusBlackboxUnauthenticatedReturnsMinimalPublicData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldProfile := common.SecurityProfile
	oldPasswordLogin := common.PasswordLoginEnabled
	oldTurnstileEnabled := common.TurnstileCheckEnabled
	oldTurnstileSiteKey := common.TurnstileSiteKey
	oldSetup := constant.Setup
	t.Cleanup(func() {
		common.SecurityProfile = oldProfile
		common.PasswordLoginEnabled = oldPasswordLogin
		common.TurnstileCheckEnabled = oldTurnstileEnabled
		common.TurnstileSiteKey = oldTurnstileSiteKey
		constant.Setup = oldSetup
	})
	common.SecurityProfile = common.SecurityProfileBlackbox
	common.PasswordLoginEnabled = true
	common.TurnstileCheckEnabled = true
	common.TurnstileSiteKey = "site-key"
	constant.Setup = true

	r := gin.New()
	r.Use(sessions.Sessions("session", cookie.NewStore([]byte("blackbox-test"))))
	r.GET("/api/status", GetStatus)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var payload map[string]any
	if err := common.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", payload["data"])
	}
	for _, forbidden := range []string{"version", "system_name", "logo", "footer_html", "docs_link", "custom_oauth_providers"} {
		if _, exists := data[forbidden]; exists {
			t.Fatalf("blackbox public status leaked %q", forbidden)
		}
	}
	if data["password_login"] != true || data["turnstile_check"] != true || data["turnstile_site_key"] != "site-key" {
		t.Fatalf("minimal status missing expected login fields: %#v", data)
	}
}
