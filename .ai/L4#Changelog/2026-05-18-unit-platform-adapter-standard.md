# 2026-05-18 单位平台适配器规范

## 变更

- 新增 `.ai/L3#Standards/standards/01.arch-03.unit-platform-adapter.md`。
- 明确 NACP 单位平台适配层遵守“平台 adapter 独立，协议级工具复用”。
- 明确 `newapi`、`rixapi`、`shellapi`、`oneapi`、`veloera`、`onehub`、`donehub`、`anyrouter`、`sub2api`、`openai`、`claude`、`gemini`、`geminicli`、`antigravity`、`cliproxyapi`、`codex` 等已识别平台必须独立目录和 adapter。
- 明确旧 `service/unitdetect`、`service/unitmonitor` 后续必须代理到 `service/unitplatform`。

## 原因

单位管理、单位详情、账号监控、平台令牌创建、NACP统计都会依赖不同平台类型的识别、凭据、余额、模型、分组和令牌接口。平台差异如果继续分散在多个 service 中，会导致平台逻辑重复、互相污染和后续扩展困难。

## 边界

- 允许复用 HTTP、URL、JSON、WAF、脱敏、标准模型解析等协议级工具。
- 不允许复用或混写平台专用识别、余额换算、令牌 payload、OAuth/session、订阅和公告等业务差异。

## 初始落地

- 新增 `service/unitplatform/` 适配层。
- `service/unitdetect` 已代理到 `unitplatform.Detect`。
- `service/unitmonitor` 已代理到 `unitplatform.FetchSnapshot`。
- `service/unit_tokens.go` 已缩为业务入口，平台令牌读取、令牌选项、令牌创建改为分发到 `unitplatform.TokenAdapter`。
- `newapi`、`rixapi`、`shellapi`、`oneapi`、`oneapifork`、`veloera`、`onehub`、`donehub`、`anyrouter` 已接入面板类账号快照和平台令牌能力。
- `sub2api` 已接入独立账号快照、平台令牌读取、分组选项、模型选项和保守创建令牌：
  - 账号信息：`/api/v1/auth/me`
  - 令牌：`/api/v1/keys`、`/api/v1/api-keys`
  - 分组：`/api/v1/groups/available`、`/api/v1/groups`、`/api/v1/group`
  - 模型：优先使用用户 API key 访问 `/v1/models`、`/api/v1/models`、`/v1beta/models`
  - 创建：优先 `POST /api/v1/keys`，失败 fallback `POST /api/v1/api-keys`
- `openai`、`claude`、`gemini`、`cliproxyapi` 已接入 API Key 直连模型读取：
  - OpenAI / CLIProxyAPI：OpenAI-compatible `/v1/models`
  - Claude：Anthropic `/v1/models`，并支持 `/anthropic` 反推 OpenAI-compatible fallback
  - Gemini：Gemini native `/v1beta/models?key=...`，并支持 OpenAI-compatible `/v1beta/openai`
  - 这些平台明确不支持平台令牌读取和创建，也不支持账号余额快照。
- `geminicli`、`antigravity`、`codex` 已接入独立识别、能力声明和模型选项读取：
  - `codex`：`GET <base>/models?client_version=1.0.0`，使用 `Originator: codex_cli_rs` 和可选 `Chatgpt-Account-Id`。
  - `geminicli`：校验 Google Cloud 项目已启用 `cloudaicompanion.googleapis.com` 后返回 Gemini CLI 静态模型清单。
  - `antigravity`：`POST /v1internal:fetchAvailableModels`，按单位 API 地址、Cloud Code 默认地址、daily、sandbox 顺序探测。
  - 这些 OAuth/CLI 直连平台未确认的余额/令牌管理能力明确返回不支持，不伪造数据。

## 验证

- `GOCACHE=/private/tmp/nacp-gocache go test ./service/unitplatform/... ./service/unitdetect ./service/unitmonitor ./service ./controller ./model`
