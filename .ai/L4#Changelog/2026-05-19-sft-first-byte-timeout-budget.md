# 2026-05-19 — SFT 首字超时与总预算硬限制

## 背景

排查 SFT 顺位容错队列时确认，旧的 `TotalRetryTimeout` 只是“准备启动下一次渠道尝试前”的软检查：

- 如果总耗时 29 秒时启动 D 渠道，旧逻辑不会中断 D。
- 如果 D 自己等待 60 秒才返回，用户总等待可能接近 90 秒。
- 这不符合“从 NACP 接收到用户请求开始，整条容错链路最多等待固定时间”的业务预期。

同时确认 `RELAY_TIMEOUT` 是 NewAPI 原生 HTTP client 全局生命周期超时，默认 `0` 表示关闭。它覆盖完整请求生命周期，可能包含完整流式输出，因此不能作为 SFT 首字等待上限直接使用。

## 调整

1. `service.ChannelHealthConfig`
   - 新增 `FirstByteTimeout`，默认 `20s`。
   - `TotalRetryTimeout` 默认从 `30s` 调整为 `60s`。

2. `service/sft_timeout.go`
   - 新增 `RelayRequestStartTime()`：优先读取 `ContextKeyRelayReceivedAt`，再回退到 `ContextKeyRequestStartTime`。
   - 新增 `SFTRetryTimeoutRemaining()` / `SFTRetryTimedOut()`。
   - 新增 `NewSFTFirstByteTimeoutContext()`，每次正式请求实际首字等待为 `min(FirstByteTimeout, TotalRetryTimeout 剩余时间)`。
   - 首字到达后停止首字 timer；流式响应体读取不受该 timer 限制。

3. `controller/relay.go`
   - SFT relay 循环在每次尝试前检查总预算。
   - 如果总预算已经耗尽且尚无上游错误，则生成 SFT 总超时错误。

4. `relay/channel/api_request.go`
   - 普通 HTTP relay 接入 SFT 首字 timeout context。
   - `client.Do(req)` 返回响应头后停止首字 timer。
   - 响应体关闭时再取消派生 context，避免资源泄漏。

5. `relay/channel/aws/relay-aws.go`
   - AWS Bedrock 非流式、流式、Nova 路径接入同样的首字 timeout 语义。

## 行为结论

当前 SFT 时间规则：

```text
总预算：60 秒，从 NACP 接收到 relay 请求开始。
单渠道首字：20 秒。
单次实际等待：min(20 秒, 总剩余时间)。
```

例子：

```text
A/B/C 已耗时 59 秒
-> D 仍可被选择
-> D 只能等待 1 秒首字
-> 1 秒内没有首字则超时收尾
```

首字限制只覆盖上游响应头/首字到达前的等待。一旦流式响应已经开始，完整流式输出不受 SFT 60 秒/20 秒限制，继续由原有流式扫描、保活和流式超时机制处理。

`RELAY_TIMEOUT` 仍保留：

- 默认 `0`，即关闭全局 HTTP client 生命周期 timeout。
- 非零时会限制完整 HTTP 请求生命周期。
- 可能截断长流式输出，因此不能代替 SFT 首字限制。

## 验证

- `GOCACHE=/private/tmp/nacp-gocache go test ./service ./relay/channel ./relay/channel/aws ./controller`
- `GOCACHE=/private/tmp/nacp-gocache go test ./...`

以上均通过。
