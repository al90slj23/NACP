# 2026-05-16 — SFT 物化摘要日志实施计划

## 背景

决定将 `20/50` 从前端派生摘要升级为有真实 `logs.id` 的链路摘要日志，并将 `21/29/51/52/59` 作为 NACP 新日志标准的一部分持久化。

## 更新

新增计划文档：

```text
.ai/L5#Knowledge/sft-materialized-summary-implementation-plan.md
```

并同步更新：

```text
.ai/L5#Knowledge/sft-log-chain-structure-and-summary-ordering.md
```

## 核心决策

1. 主列表统一按 `logs.id DESC`。
2. 链路展开按 `trace_seq -> trace_sibling_seq -> id`。
3. 老日志不迁移、不补数据。
4. 新 summary 行通过 `summary_log_id/terminal_log_id` 和子行建立明确关系。
5. 用户消费统计调整为 `2 + 20`。
6. 平台探测消耗统计以 `29` 为主。

## 第一阶段实现

1. `20/50` 已成为真实 summary 日志行，拥有自己的 `logs.id`。
2. `21/29/51/52/59` 已作为真实 SFT 子日志类型写入 `logs.type`。
3. 子日志通过 `summary_log_id` 归属到 `20/50`；summary 通过 `terminal_log_id` 指向最终 `21/52`。
4. `/api/log/grouped` 默认隐藏已归属 summary 的子行，主表继续按 `logs.id DESC`。
5. `/api/log/trace` 返回完整子步骤，按 `trace_seq -> trace_sibling_seq -> id` 排序。
6. `/console/log` 日志类型筛选已补充 `20/21/29/50/51/52/59`。
7. 新增 `ContextKeyRelayReceivedAt`，SFT 计时起点前移到 NACP 接收到请求时。
8. SFT 总重试调度窗口默认 30 秒，超时后停止后续调度，并用最后一次正式请求错误收尾。

## 验证

已执行：

```text
GOCACHE=/private/tmp/nacp-go-build go test ./model ./service ./controller ./middleware ./relay/common -run 'TestGroupedLogs|TestTraceDetailIncludesProbeLogs|TestTraceListCollapsesProbeRowsIntoSingleSummary|TestTraceProperty1_DetailFilterAndSort|TestParseProbeUsage|TestDefaultHealthConfig' -count=1
bun run build
```
