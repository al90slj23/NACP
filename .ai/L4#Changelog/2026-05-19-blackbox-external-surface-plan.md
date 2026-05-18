# 2026-05-19 B2B 黑盒外部暴露面收口计划

## 背景

本次讨论目标是让 NACP 在只服务 B 端客户的部署形态下，对未登录、无有效 API key、无有效 session 的外部扫描呈现“不可识别”的黑盒状态。

目标不是删除或改写项目身份信息，而是让未认证请求无法通过路由、响应头、错误体、公开 API、静态资源和前端包判断这是 NewAPI / OneAPI / NACP 或 AI API 网关。

根据项目保护规则，`new-api`、`QuantumNous` 等受保护项不能从源码、文档、授权后界面或项目元数据中删除、替换或弱化。黑盒方案只处理“未认证暴露面”。

## 已阅读的当前代码依据

- `router/web-router.go`：当前 `SetWebRouter` 会 `static.Serve("/", web/dist)`，并且大多数未知路径都会回退到完整 SPA；只有 `/v1`、`/api`、`/assets` 前缀特殊返回 relay not found。这意味着扫描任意普通路径可能拿到完整前端应用。
- `web/index.html`：当前公开 HTML 包含 `meta generator="new-api"`、`title New API` 和 AI 网关描述，未认证即可读取。
- `web/src/index.jsx`：当前浏览器控制台会打印 NEWAPI 与 GitHub 信息。源码内受保护信息不能删除，但未认证用户不应拿到完整控制台包。
- `controller/misc.go`：`/api/status` 当前公开返回 `version`、OAuth 状态、client id、系统名、logo、footer、docs、模块配置、价格显示配置等大量可指纹化字段。
- `router/api-router.go`：当前 `/api/status`、`/api/notice`、`/api/about`、`/api/home_page_content`、`/api/pricing`、OAuth、注册、找回密码、部分支付回调等接口未登录可访问。
- `middleware/cors.go`：当前 `PoweredBy` 全局写入 `X-New-Api-Version`，`CORS` 允许全部来源。
- `middleware/request-id.go`：当前全局写入 `X-Oneapi-Request-Id`。
- `middleware/auth.go`：`TokenAuth`、`TokenAuthReadOnly`、`UserAuth`、`AdminAuth`、`RootAuth` 对无效凭据会返回明确鉴权错误。
- `middleware/utils.go`：OpenAI 风格错误体会包含 `type: new_api_error`。
- `controller/relay.go`：`RelayNotImplemented` 返回 `API not implemented` 与 `new_api_error`，`RelayNotFound` 返回具体 `Invalid URL (METHOD path)`。
- `web/vite.config.js`：当前 chunk 名称包含 `react-core`、`semi-ui`、`tools`、`i18n`；前端应用内也存在大量管理功能字符串。若完整包未登录可下载，就会被静态扫描识别。

## 结论

该需求可以在当前架构内完整实现，不需要推翻现有 Router -> Controller -> Service -> Model 分层。

主要落点是：

- 后端新增黑盒安全剖面开关。
- 鉴权失败统一伪装 404。
- 未认证不再暴露完整 SPA。
- `/api/status` 拆分为“最小公开状态”和“认证后完整状态”。
- 静态资源分为公开登录壳与认证后控制台包。
- 响应头、错误体、CORS、NoRoute、NoMethod 统一收口。
- 增加黑盒扫描测试与有效用户 / 有效 API key 回归测试。

## 建议配置开关

```env
NACP_SECURITY_PROFILE=normal|blackbox|strict
NACP_BLACKBOX_LOGIN_PATH=/client-login-xxxx
NACP_BLACKBOX_MASK_HEADERS=true
NACP_BLACKBOX_MASK_UNAUTH_RELAY=true
NACP_BLACKBOX_PUBLIC_REGISTER=false
NACP_BLACKBOX_PUBLIC_OAUTH=false
```

默认 `normal` 保持现有行为，测试站和 B2B 正式站使用 `blackbox` 或 `strict`。

## 实施计划

### 1. 后端统一黑盒响应

新增 `middleware/blackbox.go` 或 `controller/blackbox.go`：

- `BlackboxEnabled() bool`
- `BlackboxStrict() bool`
- `AbortBlackboxNotFound(c *gin.Context)`
- `IsBrowserSessionAuthenticated(c *gin.Context) bool`
- `IsKnownPublicBlackboxPath(path string) bool`

`AbortBlackboxNotFound` 统一返回：

- HTTP `404`
- 空 body 或固定 `404 page not found`
- 不返回 JSON `success/error/type/message`
- 不返回 request id、版本号、路径、方法、路由信息
- `HEAD` 空 body
- 未认证 `OPTIONS` 同样伪 404

### 2. 鉴权中间件收口

调整 `middleware/auth.go`：

- `UserAuth` / `AdminAuth` / `RootAuth` 在未登录、session 无效、access token 无效时，黑盒模式返回伪 404。
- strict 模式下普通用户访问管理员接口也返回伪 404，避免枚举后台接口。
- `TokenAuth` / `TokenAuthReadOnly` 在缺 token、无效 token、格式错误、IP 不允许、用户禁用时，黑盒模式返回伪 404。
- DB 错误仍可返回 500，但响应体必须泛化，不能出现 database/new_api/内部表述。
- 有效 API key 保持原有 relay 行为，不影响真实客户调用。

### 3. Relay 路由收口

调整 `router/relay-router.go` 与 `controller/relay.go`：

- `/v1/*`、`/v1beta/*`、`/mj/*`、`/suno/*`、`/pg/*` 缺失或无效 key 一律伪 404。
- `RelayNotImplemented` 在黑盒模式下不向未认证请求暴露 `API not implemented`。
- `RelayNotFound` 在黑盒模式下不返回 `Invalid URL (METHOD path)`。
- 启用 Gin `HandleMethodNotAllowed`，并让 `NoMethod` 返回同样伪 404，避免 405 暴露真实路由。

### 4. 响应头和 CORS 收口

调整 `main.go`、`middleware/cors.go`、`middleware/request-id.go`：

- 黑盒未认证响应不写 `X-New-Api-Version`。
- 黑盒未认证响应不写 `X-Oneapi-Request-Id`；认证后可继续使用，或后续统一迁移为中性 `X-Request-Id`。
- 未认证不写 `Cache-Version` 等特征头。
- CORS 不再对所有未认证请求 `AllowAllOrigins`。
- 有效 API key relay 请求继续兼容跨域。
- 部署层补充反向代理规则，去掉 `Server`、`X-Powered-By`，直连 IP 或非白名单 Host 返回同样 404。

### 5. 公开 API 收口

调整 `router/api-router.go` 与 `controller/misc.go`：

- 黑盒模式下 `/api/status` 不再公开返回完整配置。
- 新增最小公开状态接口，只给登录页必要字段，例如密码登录是否开启、Turnstile site key、语言，不返回 version、系统名、logo、OAuth provider、docs、模块配置。
- `/api/notice`、`/api/about`、`/api/home_page_content`、`/api/pricing`、`/api/user-agreement`、`/api/privacy-policy` 默认登录后可见，或只在 normal 模式公开。
- `/api/user/register`、`/api/verification`、`/api/reset_password`、OAuth 路由在 B2B 黑盒模式默认关闭或迁移到邀请流程。
- 支付 webhook 保留公开入口，但签名不合法时返回伪 404，不返回 `invalid signature` 等可枚举信息。

### 6. Web 路由改为认证后才给完整 SPA

调整 `router/web-router.go`：

- 黑盒模式下不再直接向所有请求 `static.Serve("/", web/dist)`。
- 未登录只允许访问 `NACP_BLACKBOX_LOGIN_PATH`。
- `/login`、`/register`、`/reset`、`/about`、`/pricing`、`/console/*`、`/assets/*` 未登录默认伪 404。
- 访问 secret login path 时只返回登录壳，不返回完整控制台 App。
- 已登录 session 访问 `/console/*` 时才返回完整 SPA。
- 静态资源分级：公开登录壳资源可访问；完整控制台资源需要 session。

### 7. 前端拆包

调整 `web/src/index.jsx`、`web/src/App.jsx`、`web/src/components/layout/PageLayout.jsx`、`web/vite.config.js`：

- 新增 `web/src/entries/login.jsx`，只包含极简登录页、必要 i18n、Turnstile、密码登录。
- 新增 `web/src/entries/app.jsx`，登录后加载完整控制台。
- `LoginForm` 调用最小公开状态接口，不依赖完整 `/api/status`。
- `PageLayout` 的完整 `/api/status` 只在认证后调用。
- 管理页面全部 lazy import，减少主入口包内功能字符串。
- 黑盒构建下关闭 source map。
- 黑盒构建下 chunk 文件名改为纯 hash，避免语义 chunk 名泄露。
- 保留源码和认证后包内的受保护版权与项目标识；未认证用户不应拿到完整包。

### 8. 登录入口策略

推荐黑盒模式下：

- `/` 返回伪 404。后续若要图片主页或品牌页，单独设计。
- `/login` 返回伪 404。
- `NACP_BLACKBOX_LOGIN_PATH` 才是真登录页。
- 登录成功后跳转 `/console`。
- 未登录访问 `/console` 返回伪 404，不跳登录。
- 可进一步把登录 API 从 `/api/user/login` 迁移到 secret login path 下；旧路径在黑盒模式返回伪 404。

## 测试计划

新增黑盒扫描测试脚本，覆盖：

```text
/
/login
/register
/reset
/setup
/console
/console/log
/pricing
/about
/api/status
/api/notice
/api/home_page_content
/api/user/self
/api/channel/
/api/token/
/v1/models
/v1/chat/completions
/v1/responses
/v1/files
/v1beta/models
/mj/submit/imagine
/suno/submit/foo
/dashboard/billing/usage
/assets/index.js
/favicon.ico
/manifest.json
```

每个路径测试：

- GET / POST / OPTIONS / HEAD
- 无 key
- 错 key
- 错 session
- 非白名单 Host
- 随机路径对比

放行标准：

- 未认证全部统一 404。
- 响应体不含 `new-api`、`oneapi`、`NACP`、`OpenAI compatible`、`Claude`、`Gemini`、`QuantumNous`、`new_api_error`、`X-New-Api-Version`、`X-Oneapi-Request-Id`。
- 已登录管理员可以正常访问控制台。
- 有效 API key 可以正常调用 `/v1/chat/completions`、`/v1/models`、Gemini/Claude 兼容接口。
- 支付 webhook 有效签名仍可用，无效签名 404。
- 普通用户和管理员权限隔离不被破坏。

## 风险与注意

- 该方案会改变未认证错误语义，外部无效 key 探测会从 401/403 变为 404。这是目标行为，但需要确认客户 SDK 是否依赖无效 key 的具体错误。
- OAuth、注册、找回密码是强特征入口。B2B 黑盒模式建议默认关闭公开 OAuth 和公开注册。
- 静态资源鉴权会让 Vite 开发态和生产态路径处理不同，需要明确只在黑盒生产构建中强收口，开发态保留便利性。
- 不能通过删除受保护项目标识实现黑盒化，只能通过认证边界隔离公开访问。

## 下一步

如确认进入实现，建议按以下顺序推进：

1. 后端黑盒开关、统一 404、鉴权中间件、响应头收口。
2. `/api/status` 最小公开状态与公开 API 收口。
3. Web 路由认证后才发完整 SPA。
4. 前端登录壳与控制台拆包。
5. 黑盒扫描测试、有效登录回归、有效 API key 回归。
