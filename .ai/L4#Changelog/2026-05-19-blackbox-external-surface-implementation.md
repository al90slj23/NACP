# 2026-05-19 B2B 黑盒外部暴露面实施记录

## 背景

本次落地的是 B2B 部署形态下的“未认证黑盒化”第一阶段：外部扫描在没有有效 session、没有有效 API key、没有指定隐藏登录入口的情况下，尽量只能看到普通 404，不能通过响应头、错误体、公开 API、静态资源或默认 Web 入口识别系统类型。

本次没有删除、替换或弱化源码、文档、认证后界面里的受保护项目标识；处理范围只限未认证暴露面。

## 新增配置

新增环境变量：

- `NACP_SECURITY_PROFILE=normal|blackbox|strict`
- `NACP_BLACKBOX_LOGIN_PATH=/client-login`
- `NACP_BLACKBOX_MASK_HEADERS=true`
- `NACP_BLACKBOX_MASK_UNAUTH_RELAY=true`
- `NACP_BLACKBOX_PUBLIC_REGISTER=false`
- `NACP_BLACKBOX_PUBLIC_OAUTH=false`

默认 `normal` 保持原行为。`blackbox` 和 `strict` 启用未认证黑盒化。`strict` 额外用于更严格地隐藏权限不足等后台枚举场景。

## 后端公共能力

新增：

- `common/blackbox.go`
- `middleware/blackbox.go`
- `controller/blackbox.go`

核心能力：

- 统一判断安全剖面。
- 统一返回黑盒 404。
- 判断是否存在浏览器 session。
- 通过隐藏登录路径提供极简服务端登录壳。
- 登录 API 要求携带隐藏登录路径校验头，避免直接枚举 `/api/user/login`。该头名必须保持中性，不能带 `NACP`、`NewAPI`、`oneapi` 等系统特征。

## Web 暴露面调整

`router/web-router.go` 在黑盒模式下不再公开 `web/dist`：

- 未登录访问 `/`、`/login`、`/console/*`、未知普通路径：伪 404。
- 只有 `NACP_BLACKBOX_LOGIN_PATH` 返回极简登录页。
- `/assets/*`、`/logo.png`、`/favicon.ico`、`/manifest.json` 需要浏览器 session。
- 已登录 session 访问控制台路径时才返回完整 SPA。

这样公开 HTML 中的项目特征、前端 bundle 字符串、语义 chunk、控制台日志等不会在未登录状态下被直接下载。

## API 暴露面调整

`/api/status` 在黑盒未认证状态只返回登录所需最小字段：

- `password_login`
- `turnstile_check`
- `turnstile_site_key`
- `setup`
- 固定隐藏用户协议、隐私协议等公开展示项

不再返回：

- 版本号
- 系统名
- logo
- footer
- docs
- OAuth provider
- 模块配置
- 其他可指纹化字段

以下公开 API 在黑盒模式下默认需要 session 或隐藏登录入口：

- `/api/notice`
- `/api/about`
- `/api/home_page_content`
- `/api/pricing`
- `/api/user-agreement`
- `/api/privacy-policy`
- `/api/ratio_config`
- 注册、验证码、找回密码、OAuth、passkey 登录相关入口

## Relay 和鉴权调整

`TokenAuth`、`TokenAuthReadOnly`、`UserAuth`、`AdminAuth`、`RootAuth` 对未认证、无效凭据、禁用用户、无权分组、IP 限制失败等场景，在黑盒模式下优先返回伪 404。

`RelayNotFound`、`RelayNotImplemented` 在黑盒模式下不再返回 OpenAI 风格错误体，不暴露：

- `new_api_error`
- `api_not_implemented`
- `Invalid URL (METHOD path)`

`main.go` 启用 `HandleMethodNotAllowed`，`router/main.go` 将黑盒模式的 `NoMethod` 也统一成伪 404，避免 405 泄露路由存在。

## 响应头调整

黑盒模式默认隐藏：

- `X-New-Api-Version`
- `X-Oneapi-Request-Id`
- `Cache-Version`
- 普通用户指定渠道失败时的 `specific_channel_version`
- session 鉴权成功时的 `Auth-Version`

`OPTIONS` 预检在没有 session、没有 API key hint 时也返回伪 404。

## 测试结果

已新增测试：

- `middleware/blackbox_test.go`
- `controller/blackbox_status_test.go`
- `controller/blackbox_login_test.go`
- `router/blackbox_scan_test.go`
- `bin/blackbox_scan.sh`

第二阶段补充验证：

- 未认证 `/`、`/login`、`/register`、`/console/log`、`/assets/index.js` 返回黑盒 404。
- 未认证 `/api/notice` 返回黑盒 404。
- 未带隐藏登录头访问 `/api/user/login`、`/api/user/login/2fa` 返回黑盒 404。
- `PUT /api/status` 这类方法枚举返回黑盒 404。
- 未认证 `OPTIONS /v1/models` 返回黑盒 404。
- 未认证 `/v1/models`、`/v1/images/variations` 返回黑盒 404。
- 黑盒响应不应包含 `new_api`、`Invalid URL`、`API not implemented`、`success`、`error` 等可识别文本。
- 黑盒响应不应返回 `X-New-Api-Version`、`X-Oneapi-Request-Id`、`Cache-Version`。
- 黑盒 `/api/status` 仍只返回最小登录字段。
- 隐藏登录页已支持 2FA 二次验证码流程，并且 `/api/user/login/2fa` 同样必须带隐藏登录头。

第三阶段补充：

- 新增真实 HTTP 黑盒扫描脚本：`bin/blackbox_scan.sh`。
- `gogogo.sh` 新增菜单 `8) 黑盒外部暴露面扫描`。
- `gogogo.sh` 新增菜单 `9) 本地黑盒启动并扫描`，会停止本地旧后端、强制以 `NACP_SECURITY_PROFILE=blackbox` 启动本地后端、扫描 `23900`，最后停止临时后端。
- 扫描脚本新增预检：目标不可连接时只报一次；目标仍暴露 `X-New-Api-Version`、`X-Oneapi-Request-Id`、`version`、`system_name` 时，直接判定“不是黑盒模式”并提前退出。
- 修正扫描输出：同一个接口只要 header/body 断言失败，就不会再误打印 PASS。
- 修正 relay 性能保护泄露：黑盒模式下无认证提示的 relay 请求在 `SystemPerformanceCheck` 直接伪 404；生产 relay 路由顺序调整为先鉴权再执行系统性能保护，避免磁盘/CPU/内存过载信息盖过无效 token 掩蔽。

第四阶段深度审计补充：

- 对照 `ReferenceProjects/new-api-latest` 的公开入口继续扩展扫描面，补充 `/api/user-agreement`、`/api/privacy-policy`、`/api/about`、`/api/pricing`、`/api/ratio_config`、`/api/status/test`、`/api/perf-metrics`、`/api/rankings`、验证码、找回密码、OAuth、passkey、channel models、dashboard、OpenAI/Gemini/Claude/视频/任务类路径。
- 扫描脚本补充 `HEAD` 请求、invalid token 变体、`x-api-key`、`x-goog-api-key`、`mj-api-secret` 等常见认证 header，并给 curl 增加超时保护，避免异常 HEAD 或代理路径卡住扫描。
- 黑盒模式下强制忽略 `FRONTEND_BASE_URL` 的公开 NoRoute 重定向，避免任意路径 301 暴露前端入口。
- 黑盒模式下 panic recovery 统一伪 404，避免 `new_api_panic`、GitHub issue URL 或内部 panic 信息进入响应体。
- 黑盒模式下禁用 `ENABLE_PPROF` 启动的 `:8005` 调试服务，避免部署误配置导致 `/debug/pprof` 暴露 Go 调试面。
- 支付 webhook 空撞探测统一伪 404：Stripe/Creem/Waffo 必须带对应签名头才进入原验签流程；易支付类回调必须带 `sign` 参数才进入原验证流程。
- Midjourney 图片代理 `/mj/image/:id` 在黑盒模式下需要浏览器 session，避免公开图片代理路径被用作特征探测。
- 即梦官方兼容路由调整中间件顺序，先做 token 鉴权再做请求转换，避免无认证请求先返回 `Action query parameter is required` 与 `new_api_error`。
- 隐藏登录页的校验 header 从带项目名的 `X-NACP-Blackbox-Login` 改为中性 `X-Login-Path`，避免隐藏入口被猜中后直接从页面源码看到 NACP 特征。
- 扫描脚本的 body 泄漏关键字补充 `new-api` 与 `nacp`。
- 默认 `TranslateMessage` fallback 不在黑盒模式写入 `X-Translate-id`，避免极端初始化或测试路径出现额外固定特征头。
- 默认扫描 `http://localhost:23900`，可通过参数或环境变量改为测试站：

```bash
./gogogo.sh 8
./gogogo.sh 8 https://nacp.m.srl /client-login
./gogogo.sh 9
NACP_BLACKBOX_SCAN_BASE_URL=https://nacp.m.srl NACP_BLACKBOX_LOGIN_PATH=/client-login ./gogogo.sh 8
```

扫描脚本会验证：

- `/api/status` 只返回最小公开状态。
- 隐藏登录路径返回极简登录页。
- `/`、`/login`、`/register`、`/reset`、`/setup`、`/console`、`/console/log`、`/pricing`、`/about` 默认 404。
- `/api/notice`、`/api/home_page_content`、`/api/user/self`、`/api/channel/`、`/api/token/` 默认 404。
- `/api/user/login`、`/api/user/login/2fa` 没有隐藏登录头时默认 404。
- `PUT /api/status`、`OPTIONS /v1/models` 默认 404。
- `/v1/models`、`/v1/chat/completions`、`/v1/responses`、`/v1/files`、`/v1beta/models`、`/mj/*`、`/suno/*` 默认 404。
- `/dashboard/billing/usage`、`/assets/index.js`、`/favicon.ico`、`/manifest.json` 默认 404。
- `HEAD /`、`HEAD /v1/models` 默认 404。
- latest NewAPI 常见公开 API、OAuth、passkey、pricing、rankings、perf metrics 路径默认 404。
- 空撞支付 webhook 默认 404；真实签名回调继续进入原有支付验签逻辑。
- invalid token、`x-api-key`、`x-goog-api-key`、`mj-api-secret` 探测默认 404。
- 响应头不能出现 `X-New-Api-Version`、`X-Oneapi-Request-Id`、`Cache-Version`。
- 响应体不能出现 `new_api`、`Invalid URL`、`API not implemented`、`oneapi` 等可识别特征。

已执行：

```bash
GOCACHE=/private/tmp/nacp-gocache go test . ./common ./middleware ./controller ./router
GOCACHE=/private/tmp/nacp-gocache go test ./...
bash -n bin/blackbox_scan.sh
bash -n gogogo.sh
```

结果：全部通过。

第四阶段本地黑盒实扫：

```bash
./gogogo.sh 9
```

结果：扩展后的真实 HTTP 黑盒扫描通过，覆盖 HEAD、更多 NewAPI 公开路径、支付 webhook 空撞、Midjourney 图片代理、即梦兼容路径、invalid token/header 变体。

前端曾执行：

```bash
cd web
bun install
bun run build
```

结果：构建通过。`bun install` 是为了补齐本地 `web/node_modules`，不属于源码改动。

## 部署层配合要求

应用层黑盒化不能完全替代反向代理层收口。正式 B2B 黑盒部署建议：

- 容器只绑定内网端口，不直接把 `3000` 暴露给公网；公网只允许反代入口。
- 非白名单 Host 直接返回 404，不转发到应用。
- 直连 IP 返回 404，避免通过 IP 绕过域名策略。
- 反代层隐藏或弱化 `Server`、`X-Powered-By` 等网关特征头。
- 对公网只开放 80/443；MySQL、Redis、Docker API、SSH 等不应对公网开放。
- 黑盒站点 `.env` 至少设置：

```env
NACP_SECURITY_PROFILE=blackbox
NACP_BLACKBOX_LOGIN_PATH=/client-login-替换为随机路径
NACP_BLACKBOX_MASK_HEADERS=true
NACP_BLACKBOX_MASK_UNAUTH_RELAY=true
NACP_BLACKBOX_PUBLIC_REGISTER=false
NACP_BLACKBOX_PUBLIC_OAUTH=false
```

Nginx 方向示例：

```nginx
server {
    listen 80 default_server;
    listen 443 ssl default_server;
    server_name _;
    return 404;
}

server {
    listen 443 ssl http2;
    server_name example.com;

    server_tokens off;
    proxy_hide_header X-Powered-By;
    proxy_hide_header X-New-Api-Version;
    proxy_hide_header X-Oneapi-Request-Id;
    proxy_hide_header Cache-Version;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Caddy 方向示例：

```caddyfile
:443 {
    respond 404
}

example.com {
    reverse_proxy 127.0.0.1:3000 {
        header_down -X-Powered-By
        header_down -X-New-Api-Version
        header_down -X-Oneapi-Request-Id
        header_down -Cache-Version
    }
}
```

## 后续建议

- 增加隐藏登录页 passkey 兼容流程。
- 在线测试站开启 `NACP_SECURITY_PROFILE=blackbox` 后，用 `./gogogo.sh 8 https://nacp.m.srl /实际隐藏登录路径` 做真实扫描。
- 若后续需要公开品牌主页，应单独做图片化或极简静态页，不复用完整控制台 SPA。
