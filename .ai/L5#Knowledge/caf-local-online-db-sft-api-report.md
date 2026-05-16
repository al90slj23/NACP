# CAF 本地代码 + 测试站数据库 SFT API 复测报告

> 日期：2026-05-16
> 阶段：CAF Stage A，本地开发代码；测试数据库使用 `nacp.m.srl` 测试站 MySQL
> 前端/API 入口：`http://localhost:23901/`
> Relay 入口：`http://localhost:23900/`
> 上游夹具：本地 OpenAI-compatible mock upstream `http://127.0.0.1:18080`

## 一、测试目标

本轮目标是完成 SFT 容错重试日志链路的完整 API 复测，验证：

1. 真实普通用户创建 token 后可以发起 relay 请求。
2. 直接成功仍是普通 `2` 日志，不被误归为容错链路。
3. 容错成功生成 `20` 摘要，并以 `21` 作为最终成功步骤。
4. 容错失败生成 `50` 摘要，并以 `52` 作为最终客户端可见错误步骤。
5. `51/29/59/21/52` 子步骤可通过 `/api/log/trace` 精确还原。
6. `trace_seq` 是链路内排序依据，不能靠日志 `id` 或时间猜测。
7. `summary.channel_path` 只描述正式请求渠道路径，不被 `29/59` 探测渠道污染。
8. `/api/log/traces?token_name=...` 只能返回对应 token 的链路摘要。
9. 成功探测 `29` 如果上游返回 usage，应记录 token/quota；失败探测 `59` 可以为 0。
10. 计费和计量不被 51/59/52 等错误或探测日志错误污染。

## 二、测试方式

本轮坚持 API 驱动，不直接插入日志数据：

```text
管理员登录
-> 设置 ModelRatio 和 RetryTimes=2
-> 管理员 API 创建普通用户并给测试额度
-> 普通用户登录
-> 普通用户创建 token
-> 管理员 API 创建 6 个测试渠道
-> 等待渠道缓存刷新
-> 使用普通用户 token 调用 /v1/chat/completions
-> 管理员 API 查询 /api/log/grouped、/api/log/trace、/api/log/traces
-> 校验日志链路、摘要、排序、筛选、token/quota
```

测试脚本临时文件：

```text
/private/tmp/nacp_sft_online_db_full_test.js
```

测试输出证据目录：

```text
/private/tmp/nacp_c163426/
```

## 三、测试夹具

| 项 | 值 |
|----|----|
| run id | `c163426` |
| 普通用户 | `c163426u` |
| token 名称 | `c163426_token` |
| direct 模型 | `c163426-direct` |
| 容错成功模型 | `c163426-sft-success` |
| 容错失败模型 | `c163426-sft-fail` |
| RetryTimes | `2` |

本轮创建的渠道：

| 渠道 ID | 名称 | 模型 | base_url | 优先级 | 用途 |
|---------|------|------|----------|--------|------|
| `36` | `c163426-direct-ok` | `c163426-direct` | mock success | `900` | 直接成功 |
| `37` | `c163426-sft-a-fail` | `c163426-sft-success` | mock fail | `800` | 容错成功首选失败渠道 |
| `38` | `c163426-sft-b-ok` | `c163426-sft-success` | mock success | `700` | 容错成功后续成功渠道 |
| `39` | `c163426-fail-a` | `c163426-sft-fail` | mock fail | `600` | 容错失败 A |
| `40` | `c163426-fail-b` | `c163426-sft-fail` | mock fail | `500` | 容错失败 B |
| `41` | `c163426-fail-c` | `c163426-sft-fail` | mock fail | `400` | 容错失败 C |

## 四、发现并修复的问题

### 4.1 `/api/log/traces` 的 token_name 过滤失效

初次完整复测发现：

```text
/api/log/traces?token_name=<当前 token>
```

会返回其他 token 的链路摘要。这会导致日志页或链路页在按 token 过滤时混入无关结果。

修复：

1. `controller/trace.go` 读取 `token_name` 查询参数。
2. `service/trace.go` 在 `TraceListParams` 增加 `TokenName`。
3. 查询链路摘要时增加 `WHERE token_name = ?`。

复测结果：

| 检查 | 结果 |
|------|------|
| `/api/log/traces?token_name=c163426_token` | 只返回 `c163426_token` |
| 返回 total | `2` |
| 返回链路 | 容错成功、容错失败 |
| 是否混入其他 token | 否 |

### 4.2 `summary.channel_path` 被探测日志污染

初次完整复测发现，失败链路的摘要路径可能把 probe 渠道插入正式请求路径，甚至因为全局去重导致正式请求顺序失真。

修复：

1. `model/log.go` 增加正式请求步骤判断。
2. `summary.channel_path` 只统计 `2/5/21/51/52` 等正式请求步骤。
3. `29/59` 探测不进入 `channel_path`。
4. 路径只压缩相邻重复渠道，不做全局去重。

复测结果：

| 链路 | 期望正式路径 | 实际 summary.channel_path | 结果 |
|------|--------------|---------------------------|------|
| 容错成功 | `[37, 38]` | `[37, 38]` | 通过 |
| 容错失败 | `[39, 40]` | `[39, 40]` | 通过 |

说明：失败链路里存在对 `40/41` 的探测和正式请求，但 `channel_path` 代表“正式请求的相邻切换路径”，不是 probe 列表，也不是所有失败渠道全集。

## 五、最终复测结果

### 5.1 直接成功

| 字段 | 值 |
|------|----|
| relay status | `200` |
| request_id / trace_id | `20260516163624795610000FqOWjgm1ROw2KqvA` |
| summary log id | `610` |
| type | `2` |
| trace_role | `consume` |
| channel | `36` |
| prompt/completion/quota | `12 / 4 / 16` |
| trace 步骤 | `[2]` |

结论：直接成功只有一行 `2`，不需要展开为 SFT 链路。

### 5.2 容错成功

| 字段 | 值 |
|------|----|
| relay status | `200` |
| request_id / trace_id | `20260516163636689101000FqOWjgm17FBJrdiD` |
| summary log id | `617` |
| summary type | `20` |
| summary trace_role | `summary_success` |
| terminal log id | `615` |
| terminal type | `21` |
| terminal trace_role | `consume` |
| terminal channel | `38` |
| prompt/completion/quota | `12 / 4 / 16` |

展开步骤：

| log id | type | trace_role | trace_seq | channel | prompt | completion | quota |
|--------|------|------------|-----------|---------|--------|------------|-------|
| `611` | `51` | `error_intercepted` | `1` | `37` | `0` | `0` | `0` |
| `612` | `29` | `probe_success` | `2` | `38` | `12` | `4` | `16` |
| `613` | `51` | `error_intercepted` | `3` | `37` | `0` | `0` | `0` |
| `614` | `51` | `error_intercepted` | `4` | `37` | `0` | `0` | `0` |
| `615` | `21` | `consume` | `5` | `38` | `12` | `4` | `16` |

结论：

1. `20` 展开最终必须是 `21`，本轮通过。
2. `29` 成功探测记录了 usage/quota，符合“真实轻量请求产生平台运营消耗”的逻辑。
3. 用户扣费只来自最终成功消费，不来自 51 或 probe 子步骤。

### 5.3 容错失败

| 字段 | 值 |
|------|----|
| relay status | `500` |
| request_id / trace_id | `20260516163653743919000FqOWjgm1uXFguNsF` |
| summary log id | `627` |
| summary type | `50` |
| summary trace_role | `summary_failed` |
| terminal log id | `626` |
| terminal type | `52` |
| terminal trace_role | `error_visible` |
| terminal channel | `40` |
| prompt/completion/quota | `0 / 0 / 0` |

展开步骤：

| log id | type | trace_role | trace_seq | channel | prompt | completion | quota |
|--------|------|------------|-----------|---------|--------|------------|-------|
| `616` | `51` | `error_intercepted` | `1` | `39` | `0` | `0` | `0` |
| `619` | `59` | `probe_failed` | `2` | `41` | `0` | `0` | `0` |
| `618` | `59` | `probe_failed` | `3` | `40` | `0` | `0` | `0` |
| `620` | `51` | `error_intercepted` | `4` | `39` | `0` | `0` | `0` |
| `621` | `51` | `error_intercepted` | `5` | `39` | `0` | `0` | `0` |
| `622` | `51` | `error_intercepted` | `6` | `40` | `0` | `0` | `0` |
| `623` | `59` | `probe_failed` | `7` | `41` | `0` | `0` | `0` |
| `624` | `51` | `error_intercepted` | `8` | `40` | `0` | `0` | `0` |
| `625` | `51` | `error_intercepted` | `9` | `40` | `0` | `0` | `0` |
| `626` | `52` | `error_visible` | `10` | `40` | `0` | `0` | `0` |

结论：

1. `50` 展开最终必须是 `52`，本轮通过。
2. `51` 只表示被拦截的正式请求错误，不直接暴露给用户。
3. `59` 失败探测没有 usage，因此 token/quota 为 0，符合 mock 上游失败响应没有 usage 的结果。
4. 异步 probe 的 log id 与 trace_seq 不一定一致；展示必须按 `trace_seq`，不能按 `id`。

## 六、自动化断言

本轮脚本断言已覆盖：

| 断言 | 结果 |
|------|------|
| 直接请求 relay status 为 200 | 通过 |
| 容错成功 relay status 为 200 | 通过 |
| 容错失败 relay status 为 4xx/5xx | 通过 |
| grouped 中存在 `2/20/50` | 通过 |
| `20` trace 包含 `51/21` | 通过 |
| `20` trace 不包含 `52` | 通过 |
| `20` trace 最后一行是 `21` | 通过 |
| `50` trace 包含 `51/52/59` | 通过 |
| `50` trace 不包含 `21` | 通过 |
| `50` trace 最后一行是 `52` | 通过 |
| 所有 trace 的 `trace_seq` 单调递增 | 通过 |
| `summary.channel_path` 等于正式请求路径 | 通过 |
| `/api/log/traces?token_name=...` 不返回其他 token | 通过 |

## 七、补充验证

Go 层局部测试：

```text
GOCACHE=/private/tmp/nacp-go-test-cache go test ./model ./service
```

结果：

```text
ok github.com/QuantumNous/new-api/model
ok github.com/QuantumNous/new-api/service
```

## 八、剩余风险

1. 本轮重点是本地代码 + 测试站 MySQL + API 驱动测试，尚未做 SQLite/PostgreSQL 全量迁移验证。
2. 本轮未做浏览器截图级 UI 视觉验收；日志页视觉仍应在后续 Stage B 或发布前做一次人工/浏览器检查。
3. `/api/log/traces` 的复杂组合筛选、分页边界和 status 过滤仍建议补脚本化回归。
4. 测试夹具会在测试站数据库留下用户、token、渠道和日志数据；后续需要增加 API 清理步骤或按 tag 清理策略。

