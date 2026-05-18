# 2026-05-18 日志 IP 记录与用户可见性策略调整

## 背景

原版 NewAPI 中用户设置 `record_ip_log` 表示“是否记录请求与错误日志 IP”。这会导致用户关闭后，平台无法在日志中保留调用来源 IP，不利于安全审计和恶意使用追踪。

## 本次调整

- 消费日志与错误日志写入时默认保存 `ClientIP()`。
- 用户设置 `record_ip_log` 的语义调整为“是否在用户端日志中显示 IP”。
- `/api/log/self` 和 `/api/log/token` 在返回用户可见日志前按设置脱敏 `ip`。
- 管理员日志接口保持真实 IP 可见，用于后台审计、风控和统计。
- 用户侧个人设置文案保持原版：`记录请求与错误日志IP`，避免改变用户可见体验；内部语义调整为默认记录、用户侧按设置显示/隐藏。

## 验证

- `GOCACHE=/private/tmp/nacp-gocache go test ./model`
- `GOCACHE=/private/tmp/nacp-gocache go test ./controller ./service`
- `bun run build`
