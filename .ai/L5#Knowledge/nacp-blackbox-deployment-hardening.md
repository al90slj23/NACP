# NACP B2B 黑盒部署加固清单

## 目标

NACP 的黑盒模式解决的是“未认证应用层暴露面”：外部请求在没有有效浏览器 session、没有有效 API key、没有隐藏登录入口上下文时，应尽量只看到普通 404，不能通过公开页面、公开 API、错误体、响应头或静态资源判断这是 NewAPI、OneAPI、NACP 或 AI API 网关。

但应用代码不能完全控制网络层、TLS、反向代理、端口、Host、证书和基础设施 header。因此正式 B2B 部署必须同时做部署层收口。

## 应用层必须配置

正式 B2B 黑盒站点 `.env` 至少设置：

```env
NACP_SECURITY_PROFILE=blackbox
NACP_BLACKBOX_LOGIN_PATH=/替换成随机长路径
NACP_BLACKBOX_MASK_HEADERS=true
NACP_BLACKBOX_MASK_UNAUTH_RELAY=true
NACP_BLACKBOX_PUBLIC_REGISTER=false
NACP_BLACKBOX_PUBLIC_OAUTH=false
ENABLE_PPROF=false
```

隐藏登录路径要求：

- 不使用 `/login`、`/admin`、`/client-login`、`/console-login` 这类可猜路径。
- 推荐 24 位以上随机路径，例如 `/p/9f4b6c2a7d0e4a1c8b3d6f20`。
- 只给 B 端客户负责人或内部管理员，不写进公开文档、前端页面、邮件模板、客服 FAQ。
- 路径泄漏后要可以直接换新，并重启服务。

## Docker 与端口

正式部署原则：

- 应用容器端口只绑定内网或 `127.0.0.1`，不要把 `3000` 直接暴露公网。
- MySQL、Redis、Docker API、pprof、SSH 不对公网开放。
- 公网入口只允许 `80/443`。
- 如果使用双库，业务库和日志库容器也只允许应用容器或内网访问。

Docker Compose 方向示例：

```yaml
services:
  nacp:
    ports:
      - "127.0.0.1:3000:3000"
    environment:
      NACP_SECURITY_PROFILE: blackbox
      NACP_BLACKBOX_LOGIN_PATH: /p/replace-with-random-path
      ENABLE_PPROF: "false"

  mysql:
    ports: []

  redis:
    ports: []
```

如果必须临时开放数据库维护端口，必须限源 IP，并在维护完成后关闭。

## Host 白名单与直连 IP 404

目标：

- 访问正确域名才转发到 NACP。
- 直连服务器 IP 返回 404。
- 任意未知 Host 返回 404。
- 不要把未知 Host 301/302 到真实域名，这会泄漏站点入口。

Nginx 示例：

```nginx
server {
    listen 80 default_server;
    listen 443 ssl http2 default_server;
    server_name _;

    ssl_certificate     /etc/nginx/ssl/default.crt;
    ssl_certificate_key /etc/nginx/ssl/default.key;

    server_tokens off;
    return 404;
}

server {
    listen 80;
    server_name nacp.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name nacp.example.com;

    ssl_certificate     /etc/nginx/ssl/nacp.example.com.crt;
    ssl_certificate_key /etc/nginx/ssl/nacp.example.com.key;

    server_tokens off;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_hide_header X-New-Api-Version;
        proxy_hide_header X-Oneapi-Request-Id;
        proxy_hide_header Cache-Version;
        proxy_hide_header Auth-Version;
        proxy_hide_header X-Powered-By;
    }
}
```

说明：

- `server_tokens off` 只能隐藏 Nginx 版本，通常不能完全移除 `Server: nginx`。
- 如果部署环境支持 `headers_more`，可额外配置 `more_clear_headers Server;`。否则至少确保不暴露具体版本。
- 默认 TLS 证书不要使用真实业务域名证书，否则直连 IP 或未知 SNI 时仍可能通过证书看到域名。

## CDN / WAF / 云防火墙

如果有 Cloudflare、CDN、SLB 或云 WAF：

- 未知 Host / 非业务 Host：直接 404，不回源。
- 直连源站 IP：云防火墙只允许 CDN 回源 IP。
- 关闭或统一自定义错误页，避免 CDN 404、Nginx 404、应用 404 三种页面差异过大。
- 不给未认证路径添加暴露技术栈的 header，例如 `X-Powered-By`、框架名、内部 upstream 名。
- 限制异常扫描速率：同 IP 短时间大量 404、OPTIONS、HEAD、随机 path，应在边缘层限速。

## TLS 与证书侧信号

机器扫描不仅看 HTTP body，也可能看证书、SNI、ALPN、TLS 指纹和 IP 资产归属。

部署要求：

- 源站 IP 不对公网直接暴露，或至少直连 IP 只返回 default 404。
- 默认证书使用中性证书，不使用真实业务证书。
- 正式业务域名证书 SAN 中不要塞无关内部域名。
- TLS 版本和 cipher 使用常规安全配置，不要暴露过旧栈。

## 第三方服务与运维系统侧信号

黑盒扫描重点是公网未认证 HTTP，但部署时还要注意第三方系统和内部运维系统也可能记录项目特征。

当前源码中存在一些不会被普通公网扫描直接读到、但可能进入第三方或运维系统的名称：

- 启动日志、源码 import path、构建信息中包含原项目身份信息。这些属于受保护项，不通过删除源码解决，只能确保日志系统不公开。
- `PYROSCOPE_URL` 如果启用，会把应用名和主机标签发送到 Pyroscope。B2B 黑盒部署建议不启用外部 Pyroscope；如果必须启用，只接入内部私有 Pyroscope。
- 支付平台订单 reference、支付描述、回调 URL 可能进入 Stripe/Creem/Waffo/易支付后台。支付后台不是公开扫描面，但属于第三方可见面，生产前要确认这些字段不会被展示给终端客户或外部审计对象。
- 本地缓存目录、日志文件名、容器名、镜像 tag 可能包含项目名。它们不应被 Web server 静态目录、对象存储、日志下载接口或备份系统公开。

部署要求：

- 日志平台、APM、Pyroscope、Sentry、Prometheus、Grafana 不对公网开放。
- Web 根目录不要挂载项目根目录、日志目录、缓存目录、备份目录。
- 备份文件、`.env`、数据库 dump、构建临时目录不放进 Nginx 可访问目录。
- 支付平台里展示给用户的商品名、订单说明、收据文案应使用业务品牌，不使用内部项目名。

## 静态资源与前端包

黑盒模式下应用已经阻止未登录下载完整 SPA 与 `/assets/*`，部署时仍要确认：

- 生产环境不公开 source map：`web/dist/**/*.map` 不应存在。
- 未认证 `/assets/index.js`、`/favicon.ico`、`/manifest.json` 返回 404。
- 未认证 `/`、`/login`、`/console/*` 返回 404。
- 隐藏登录页源码不应包含 `NACP`、`new-api`、`New API`、`QuantumNous`、`oneapi`。

注意：登录后的管理员或客户浏览器一定可以拿到完整前端包，所以黑盒目标只覆盖未认证扫描，不覆盖“已授权用户拿到前端包后分析源码”。

## 支付回调与公开机器入口

支付 webhook 这类入口必须公开给支付平台，但不能被空撞识别：

- Stripe：没有 `Stripe-Signature` 时直接 404。
- Creem：没有 `creem-signature` 时直接 404。
- Waffo：没有 `X-SIGNATURE` 时直接 404。
- 易支付：没有 `sign` 时直接 404。

真实签名存在时仍进入原验签流程。支付平台回调 URL 不应公开在文档或前台页面。

## 上线前黑盒检查

应用层扫描：

```bash
./gogogo.sh 8 https://nacp.example.com /真实隐藏登录路径
```

基础 HTTP 检查：

```bash
curl -i https://nacp.example.com/
curl -i https://nacp.example.com/login
curl -i https://nacp.example.com/console/log
curl -i https://nacp.example.com/api/status
curl -i https://nacp.example.com/v1/models
curl -i -H 'Authorization: Bearer sk-invalid' https://nacp.example.com/v1/models
curl -I https://nacp.example.com/
```

预期：

- 除 `/api/status` 最小状态和真实隐藏登录路径外，未认证路径都是 404。
- 404 body 为空或不含结构化 JSON。
- header 不出现 `X-New-Api-Version`、`X-Oneapi-Request-Id`、`Cache-Version`、`Auth-Version`、`X-Powered-By`。
- body 不出现 `new_api`、`new-api`、`New API`、`oneapi`、`NACP`、`QuantumNous`、`Invalid URL`、`API not implemented`。

Host / IP 检查：

```bash
curl -ik https://服务器IP/
curl -ik -H 'Host: random.example' https://服务器IP/
curl -ik --resolve nacp.example.com:443:服务器IP https://nacp.example.com/
```

预期：

- 直连 IP：404。
- 未知 Host：404。
- 正确 Host：进入黑盒应用，未认证仍按应用规则 404。

端口检查：

```bash
nmap -sV -p 1-10000 服务器IP
nmap -sV --script=http-headers -p 80,443 nacp.example.com
```

预期：

- 公网只暴露 80/443。
- 不暴露 MySQL、Redis、Docker API、pprof、开发端口。
- HTTP headers 不带应用层特征。

## 当前仍需人工确认的风险

1. 隐藏登录路径一旦泄漏，攻击者能看到一个通用登录页。当前已去掉 NACP header 名，但隐藏路径仍应视为敏感入口，泄漏后轮换。
2. 已登录用户可以下载完整控制台包，这不属于未认证黑盒范围。B2B 客户本身是授权对象，不能靠黑盒隐藏替代合同和权限管理。
3. 反代默认错误页、CDN 错误页、TLS 默认证书可能形成侧面指纹，必须在部署层统一。
4. 如果临时开启 `NACP_BLACKBOX_PUBLIC_REGISTER=true` 或 `NACP_BLACKBOX_PUBLIC_OAUTH=true`，注册/OAuth 路径会重新成为明显特征入口，只适合非黑盒公网形态。
5. 新增路由、新增支付平台、新增兼容 API 时，必须同步加入 `bin/blackbox_scan.sh`。

## 放行标准

正式 B2B 黑盒部署满足以下条件才放行：

- `./gogogo.sh 8 https://域名 /隐藏路径` 通过。
- 直连 IP 和未知 Host 返回 404。
- 公网端口只开放 80/443。
- 未认证扫描不暴露应用 header、项目名、结构化 API 错误、前端 bundle。
- 有效 API key 正常调用模型。
- 管理员通过隐藏登录路径正常登录控制台。
- 支付平台真实回调能通过，空撞回调 404。
