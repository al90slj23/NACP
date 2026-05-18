package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func withBlackboxScanConfig(t *testing.T) {
	t.Helper()
	oldProfile := common.SecurityProfile
	oldLoginPath := common.BlackboxLoginPath
	oldMaskHeaders := common.BlackboxMaskHeaders
	oldMaskRelay := common.BlackboxMaskUnauthRelay
	oldPublicRegister := common.BlackboxPublicRegister
	oldPublicOAuth := common.BlackboxPublicOAuth
	oldPasswordLogin := common.PasswordLoginEnabled
	oldTurnstileEnabled := common.TurnstileCheckEnabled
	oldTurnstileSiteKey := common.TurnstileSiteKey
	oldSetup := constant.Setup
	t.Cleanup(func() {
		common.SecurityProfile = oldProfile
		common.BlackboxLoginPath = oldLoginPath
		common.BlackboxMaskHeaders = oldMaskHeaders
		common.BlackboxMaskUnauthRelay = oldMaskRelay
		common.BlackboxPublicRegister = oldPublicRegister
		common.BlackboxPublicOAuth = oldPublicOAuth
		common.PasswordLoginEnabled = oldPasswordLogin
		common.TurnstileCheckEnabled = oldTurnstileEnabled
		common.TurnstileSiteKey = oldTurnstileSiteKey
		constant.Setup = oldSetup
	})
	common.SecurityProfile = common.SecurityProfileBlackbox
	common.BlackboxLoginPath = "/hidden-login"
	common.BlackboxMaskHeaders = true
	common.BlackboxMaskUnauthRelay = true
	common.BlackboxPublicRegister = false
	common.BlackboxPublicOAuth = false
	common.PasswordLoginEnabled = true
	common.TurnstileCheckEnabled = true
	common.TurnstileSiteKey = "site-key"
	constant.Setup = true
}

func newBlackboxScanRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.HandleMethodNotAllowed = true
	r.Use(middleware.RequestId())
	r.Use(middleware.PoweredBy())
	r.Use(sessions.Sessions("session", cookie.NewStore([]byte("blackbox-scan-test"))))
	r.Use(middleware.Cache())
	r.Use(middleware.CORS())
	r.NoMethod(middleware.AbortBlackboxNotFound)

	r.GET("/api/status", controller.GetStatus)
	r.GET("/api/notice", middleware.BlackboxSessionRequired(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	r.POST("/api/user/login", middleware.BlackboxLoginGate(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	r.POST("/api/user/login/2fa", middleware.BlackboxLoginGate(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	r.GET("/v1/models", middleware.TokenAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	r.POST("/v1/chat/completions", middleware.SystemPerformanceCheck(), middleware.TokenAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	r.POST("/v1/images/variations", middleware.TokenAuth(), controller.RelayNotImplemented)
	r.NoRoute(middleware.AbortBlackboxNotFound)
	return r
}

func TestBlackboxUnauthenticatedScanSurfaceIsMasked(t *testing.T) {
	withBlackboxScanConfig(t)
	r := newBlackboxScanRouter()

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/"},
		{http.MethodGet, "/login"},
		{http.MethodGet, "/register"},
		{http.MethodGet, "/console/log"},
		{http.MethodGet, "/assets/index.js"},
		{http.MethodGet, "/api/notice"},
		{http.MethodPost, "/api/user/login"},
		{http.MethodPost, "/api/user/login/2fa"},
		{http.MethodPut, "/api/status"},
		{http.MethodOptions, "/v1/models"},
		{http.MethodGet, "/v1/models"},
		{http.MethodPost, "/v1/chat/completions"},
		{http.MethodPost, "/v1/images/variations"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusNotFound {
				t.Fatalf("expected blackbox 404, got %d body=%q", w.Code, w.Body.String())
			}
			assertBlackboxHeadersMasked(t, w)
			body := strings.ToLower(w.Body.String())
			for _, leaked := range []string{"new_api", "invalid url", "api not implemented", "success", "error"} {
				if strings.Contains(body, leaked) {
					t.Fatalf("blackbox response leaked %q in body %q", leaked, w.Body.String())
				}
			}
		})
	}
}

func TestBlackboxStatusIsMinimalAndHeaderMasked(t *testing.T) {
	withBlackboxScanConfig(t)
	r := newBlackboxScanRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%q", w.Code, w.Body.String())
	}
	assertBlackboxHeadersMasked(t, w)
	var payload map[string]any
	if err := common.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", payload["data"])
	}
	for _, forbidden := range []string{"version", "system_name", "logo", "footer_html", "docs_link", "custom_oauth_providers"} {
		if _, exists := data[forbidden]; exists {
			t.Fatalf("blackbox status leaked %q", forbidden)
		}
	}
	if data["password_login"] != true || data["turnstile_check"] != true || data["turnstile_site_key"] != "site-key" {
		t.Fatalf("blackbox status missing minimal login fields: %#v", data)
	}
}

func TestBlackboxHiddenLoginGateAllowsOnlyHiddenPathHeader(t *testing.T) {
	withBlackboxScanConfig(t)
	r := newBlackboxScanRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/user/login", nil)
	req.Header.Set(middleware.BlackboxLoginHeader, "/hidden-login")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected hidden login header to pass, got %d body=%q", w.Code, w.Body.String())
	}
}

func assertBlackboxHeadersMasked(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	for _, header := range []string{"X-New-Api-Version", "X-Oneapi-Request-Id", "Cache-Version"} {
		if value := w.Header().Get(header); value != "" {
			t.Fatalf("expected %s to be masked, got %q", header, value)
		}
	}
}
