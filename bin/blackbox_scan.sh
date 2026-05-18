#!/usr/bin/env bash
set -u

BASE_URL="${1:-${NACP_BLACKBOX_SCAN_BASE_URL:-http://localhost:23900}}"
LOGIN_PATH="${2:-${NACP_BLACKBOX_LOGIN_PATH:-/client-login}}"

BASE_URL="${BASE_URL%/}"
FAILURES=0

red() { printf '\033[0;31m%s\033[0m\n' "$*"; }
green() { printf '\033[0;32m%s\033[0m\n' "$*"; }
yellow() { printf '\033[1;33m%s\033[0m\n' "$*"; }

fail() {
  FAILURES=$((FAILURES + 1))
  red "[FAIL] $*"
}

pass() {
  green "[PASS] $*"
}

tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/nacp-blackbox-scan.XXXXXX")"
trap 'rm -rf "$tmpdir"' EXIT

request() {
  local method="$1"
  local path="$2"
  local extra_header="${3:-}"
  local name
  name="$(printf '%s_%s' "$method" "$path" | tr '/:?' '___' | tr -cd '[:alnum:]_.-')"
  local header_file="${tmpdir}/${name}.headers"
  local body_file="${tmpdir}/${name}.body"
  local status_file="${tmpdir}/${name}.status"
  touch "$header_file" "$body_file" "$status_file"
  local curl_args=(-sS --max-time 10 -o "$body_file" -D "$header_file" -w "%{http_code}")
  if [ "$method" = "HEAD" ]; then
    curl_args+=(-I)
  else
    curl_args+=(-X "$method")
  fi
  if [ -n "$extra_header" ]; then
    curl_args+=(-H "$extra_header")
  fi
  if ! curl "${curl_args[@]}" "${BASE_URL}${path}" > "$status_file" 2>"${tmpdir}/${name}.stderr"; then
    printf '000' > "$status_file"
  fi
  printf '%s|%s|%s|%s\n' "$(cat "$status_file" 2>/dev/null)" "$header_file" "$body_file" "${tmpdir}/${name}.stderr"
}

body_preview() {
  local body="$1"
  if [ -f "$body" ]; then
    head -c 240 "$body"
  fi
}

assert_target_reachable_and_blackbox() {
  local result status headers body stderr lowered_headers lowered_body
  result="$(request GET /api/status)"
  IFS='|' read -r status headers body stderr <<< "$result"
  if [ "$status" = "000" ]; then
    red "[ERROR] 无法连接目标 ${BASE_URL}"
    red "        curl: $(cat "$stderr" 2>/dev/null)"
    yellow "        请先启动后端，或使用 ./gogogo.sh 9 自动启动本地黑盒后端并扫描。"
    exit 2
  fi
  lowered_headers="${tmpdir}/preflight.headers.lower"
  lowered_body="${tmpdir}/preflight.body.lower"
  tr '[:upper:]' '[:lower:]' < "$headers" > "$lowered_headers"
  tr '[:upper:]' '[:lower:]' < "$body" > "$lowered_body"
  if grep -q '^x-new-api-version:' "$lowered_headers" ||
    grep -q '^x-oneapi-request-id:' "$lowered_headers" ||
    grep -q '"version"' "$lowered_body" ||
    grep -q '"system_name"' "$lowered_body"; then
    red "[ERROR] 目标已响应，但不是黑盒模式：${BASE_URL}"
    yellow "        当前响应仍暴露版本/系统信息或特征头。"
    yellow "        本地请使用：./gogogo.sh 9"
    yellow "        或手动设置：NACP_SECURITY_PROFILE=blackbox NACP_BLACKBOX_LOGIN_PATH=${LOGIN_PATH}"
    exit 3
  fi
}

assert_masked_404() {
  local method="$1"
  local path="$2"
  local extra_header="${3:-}"
  local result status headers body stderr
  local before_failures="$FAILURES"
  result="$(request "$method" "$path" "$extra_header")"
  IFS='|' read -r status headers body stderr <<< "$result"
  if [ "$status" != "404" ]; then
    fail "$method $path expected 404, got ${status}; stderr=$(cat "$stderr") body=$(body_preview "$body")"
    return
  fi
  assert_headers_masked "$method $path" "$headers"
  assert_body_masked "$method $path" "$body"
  local lowered="${tmpdir}/body-404.lower"
  tr '[:upper:]' '[:lower:]' < "$body" > "$lowered"
  for marker in "success" "\"error\"" "invalid_request_error"; do
    if grep -q "$marker" "$lowered"; then
      fail "$method $path leaked structured error marker: $marker"
    fi
  done
  if [ "$FAILURES" = "$before_failures" ]; then
    pass "$method $path masked as 404"
  fi
}

assert_headers_masked() {
  local label="$1"
  local headers="$2"
  local lowered="${tmpdir}/headers.lower"
  tr '[:upper:]' '[:lower:]' < "$headers" > "$lowered"
  for header in "x-new-api-version:" "x-oneapi-request-id:" "cache-version:" "auth-version:"; do
    if grep -q "^${header}" "$lowered"; then
      fail "$label leaked header ${header%:}"
    fi
  done
}

assert_body_masked() {
  local label="$1"
  local body="$2"
  local lowered="${tmpdir}/body.lower"
  tr '[:upper:]' '[:lower:]' < "$body" > "$lowered"
  for marker in "new_api" "new-api" "invalid url" "api not implemented" "x-new-api-version" "oneapi" "quantumnous" "nacp"; do
    if grep -q "$marker" "$lowered"; then
      fail "$label leaked marker: $marker"
    fi
  done
}

assert_status_minimal() {
  local result status headers body stderr
  local before_failures="$FAILURES"
  result="$(request GET /api/status)"
  IFS='|' read -r status headers body stderr <<< "$result"
  if [ "$status" != "200" ]; then
    fail "GET /api/status expected 200, got ${status}; stderr=$(cat "$stderr") body=$(body_preview "$body")"
    return
  fi
  assert_headers_masked "GET /api/status" "$headers"
  for required in "password_login" "turnstile_check" "setup"; do
    if ! grep -q "$required" "$body"; then
      fail "GET /api/status missing minimal field $required"
    fi
  done
  for forbidden in "version" "system_name" "footer_html" "docs_link" "custom_oauth_providers"; do
    if grep -q "$forbidden" "$body"; then
      fail "GET /api/status leaked field $forbidden"
    fi
  done
  if [ "$FAILURES" = "$before_failures" ]; then
    pass "GET /api/status minimal public status"
  fi
}

assert_login_page() {
  local result status headers body stderr
  local before_failures="$FAILURES"
  result="$(request GET "$LOGIN_PATH")"
  IFS='|' read -r status headers body stderr <<< "$result"
  if [ "$status" != "200" ]; then
    fail "GET ${LOGIN_PATH} expected 200, got ${status}; stderr=$(cat "$stderr") body=$(body_preview "$body")"
    return
  fi
  assert_headers_masked "GET ${LOGIN_PATH}" "$headers"
  for required in "Sign in" "X-Login-Path" "/api/user/login"; do
    if ! grep -q "$required" "$body"; then
      fail "GET ${LOGIN_PATH} missing login marker $required"
    fi
  done
  assert_body_masked "GET ${LOGIN_PATH}" "$body"
  if [ "$FAILURES" = "$before_failures" ]; then
    pass "GET ${LOGIN_PATH} hidden login page"
  fi
}

echo "NACP blackbox scan"
echo "Base URL: ${BASE_URL}"
echo "Login path: ${LOGIN_PATH}"
echo ""

assert_target_reachable_and_blackbox
assert_status_minimal
assert_login_page

for item in \
  "HEAD /" \
  "GET /" \
  "GET /login" \
  "GET /register" \
  "GET /reset" \
  "GET /setup" \
  "GET /console" \
  "GET /console/log" \
  "GET /pricing" \
  "GET /about" \
  "GET /api/setup" \
  "GET /api/notice" \
  "GET /api/user-agreement" \
  "GET /api/privacy-policy" \
  "GET /api/about" \
  "GET /api/home_page_content" \
  "GET /api/pricing" \
  "GET /api/ratio_config" \
  "GET /api/status/test" \
  "GET /api/perf-metrics" \
  "GET /api/rankings" \
  "GET /api/verification" \
  "GET /api/reset_password" \
  "GET /api/oauth/state" \
  "GET /api/oauth/github" \
  "GET /api/oauth/wechat" \
  "GET /api/oauth/telegram/login" \
  "GET /api/user/groups" \
  "GET /api/user/self" \
  "GET /api/user/models" \
  "POST /api/user/register" \
  "GET /api/channel/" \
  "GET /api/channel/models" \
  "GET /api/token/" \
  "POST /api/user/login" \
  "POST /api/user/login/2fa" \
  "POST /api/user/passkey/login/begin" \
  "POST /api/user/passkey/login/finish" \
  "POST /api/stripe/webhook" \
  "POST /api/creem/webhook" \
  "POST /api/waffo/webhook" \
  "GET /api/user/epay/notify" \
  "GET /api/subscription/epay/notify" \
  "GET /api/subscription/epay/return" \
  "PUT /api/status" \
  "HEAD /v1/models" \
  "OPTIONS /v1/models" \
  "GET /v1/models" \
  "GET /v1/models/gpt-4o" \
  "POST /v1/chat/completions" \
  "POST /v1/messages" \
  "POST /v1/images/generations" \
  "POST /v1/audio/transcriptions" \
  "POST /v1/embeddings" \
  "POST /v1/responses" \
  "GET /v1/files" \
  "POST /v1/files" \
  "GET /v1/realtime" \
  "GET /v1beta/models" \
  "GET /v1beta/openai/models" \
  "POST /v1beta/models/gemini-pro:generateContent" \
  "POST /pg/chat/completions" \
  "POST /mj/submit/imagine" \
  "GET /mj/image/probe" \
  "GET /mj/task/probe/fetch" \
  "POST /suno/submit/foo" \
  "GET /suno/fetch/probe" \
  "POST /kling/v1/videos/text2video" \
  "POST /jimeng/" \
  "GET /dashboard/billing/usage" \
  "GET /v1/dashboard/billing/usage" \
  "GET /assets/index.js" \
  "GET /favicon.ico" \
  "GET /manifest.json" \
  "GET /robots.txt" \
  "GET /sitemap.xml" \
  "GET /.well-known/security.txt"; do
  method="${item%% *}"
  path="${item#* }"
  assert_masked_404 "$method" "$path"
done

assert_masked_404 "GET" "/v1/models" "Authorization: Bearer sk-invalid"
assert_masked_404 "GET" "/v1/models" "x-api-key: sk-invalid"
assert_masked_404 "GET" "/v1beta/models" "x-goog-api-key: sk-invalid"
assert_masked_404 "GET" "/mj/task/probe/fetch" "mj-api-secret: sk-invalid"
assert_masked_404 "POST" "/v1/chat/completions" "Authorization: Bearer sk-invalid"
assert_masked_404 "POST" "/v1/responses" "Authorization: Bearer sk-invalid"
assert_masked_404 "GET" "/dashboard/billing/usage" "Authorization: Bearer sk-invalid"

echo ""
if [ "$FAILURES" -gt 0 ]; then
  red "Blackbox scan failed: ${FAILURES} issue(s)"
  exit 1
fi

green "Blackbox scan passed"
