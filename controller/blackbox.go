package controller

import (
	"html/template"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

func BlackboxLoginPage(c *gin.Context) {
	loginPath := template.JSEscapeString(common.BlackboxLoginPath)
	turnstileEnabled := "false"
	turnstileWidget := ""
	if common.TurnstileCheckEnabled {
		turnstileEnabled = "true"
		turnstileWidget = `<script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>
    <div class="turnstile"><div class="cf-turnstile" data-sitekey="` + template.HTMLEscapeString(common.TurnstileSiteKey) + `" data-callback="onTurnstileToken"></div></div>`
	}
	html := `<!doctype html>
<html lang="zh">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <meta name="robots" content="noindex,nofollow">
  <title>Sign in</title>
  <style>
    :root{color-scheme:dark}body{margin:0;min-height:100vh;display:grid;place-items:center;background:#0f1115;color:#f3f4f6;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}.box{width:min(360px,calc(100vw - 40px));padding:28px;border:1px solid #2b3038;border-radius:12px;background:#171a20;box-shadow:0 18px 48px rgba(0,0,0,.35)}h1{margin:0 0 20px;font-size:22px;font-weight:650}label{display:block;margin:14px 0 7px;color:#b7bdc8;font-size:13px}input{box-sizing:border-box;width:100%;padding:12px 13px;border:1px solid #363c47;border-radius:8px;background:#101318;color:#f3f4f6;font-size:15px;outline:none}input:focus{border-color:#5f9bff}button{width:100%;margin-top:20px;padding:12px 13px;border:0;border-radius:8px;background:#4f8cff;color:white;font-size:15px;font-weight:650;cursor:pointer}button:disabled{opacity:.6;cursor:not-allowed}.msg{min-height:20px;margin-top:14px;color:#ff9f9f;font-size:13px;line-height:1.5}
  </style>
</head>
<body>
  <form class="box" id="login-form">
    <h1>Sign in</h1>
    <label for="username">Username</label>
    <input id="username" autocomplete="username" required>
    <label for="password">Password</label>
    <input id="password" type="password" autocomplete="current-password" required>
    <div id="twofa-box" style="display:none">
      <label for="twofa-code">Verification code</label>
      <input id="twofa-code" inputmode="numeric" autocomplete="one-time-code">
    </div>
    ` + turnstileWidget + `
    <button id="submit" type="submit">Continue</button>
    <div class="msg" id="message"></div>
  </form>
  <script>
    const loginPath = "` + loginPath + `";
    const turnstileEnabled = ` + turnstileEnabled + `;
    let turnstileToken = "";
    window.onTurnstileToken = (token) => { turnstileToken = token || ""; };
    const form = document.getElementById("login-form");
    const button = document.getElementById("submit");
    const message = document.getElementById("message");
    const twofaBox = document.getElementById("twofa-box");
    const twofaCode = document.getElementById("twofa-code");
    let awaiting2FA = false;
    async function postJSON(path, body) {
      const res = await fetch(path, {
        method: "POST",
        credentials: "same-origin",
        headers: {"Content-Type": "application/json", "X-Login-Path": loginPath},
        body: JSON.stringify(body)
      });
      if (res.status === 404) throw new Error("Sign in failed");
      return await res.json();
    }
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      message.textContent = "";
      if (!awaiting2FA && turnstileEnabled && !turnstileToken) {
        message.textContent = "Verification is still pending.";
        return;
      }
      button.disabled = true;
      try {
        let payload;
        if (awaiting2FA) {
          payload = await postJSON("/api/user/login/2fa", {code: twofaCode.value});
        } else {
          payload = await postJSON("/api/user/login?turnstile=" + encodeURIComponent(turnstileToken), {
            username: document.getElementById("username").value,
            password: document.getElementById("password").value
          });
        }
        if (!payload.success) throw new Error(payload.message || "Sign in failed");
        if (payload.data && payload.data.require_2fa) {
          awaiting2FA = true;
          twofaBox.style.display = "block";
          document.getElementById("username").disabled = true;
          document.getElementById("password").disabled = true;
          button.textContent = "Verify";
          twofaCode.focus();
          return;
        }
        localStorage.setItem("user", JSON.stringify(payload.data || {}));
        window.location.assign("/console");
      } catch (err) {
        message.textContent = err.message || "Sign in failed";
      } finally {
        button.disabled = false;
      }
    });
  </script>
</body>
</html>`
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
