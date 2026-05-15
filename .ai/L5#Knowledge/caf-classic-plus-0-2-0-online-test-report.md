# CAF classic-plus-0.2.0 在线测试报告

> 测试日期：2026-05-16
> 测试站：`https://nacp.m.srl/`
> 测试版本：`classic-plus-0.2.0-dev`
> 版本响应头：`x-new-api-version: classic-plus-0.2.0-dev`
> 测试依据：[CAF classic-plus-0.2.0 深度测试计划](./caf-classic-plus-0-2-0-plan.md)

---

## 1. 总体结论

本轮已完成线上真实调用、SFT 容错成功、SFT 全失败、日志 API、数据库不变量、计费计量、渠道恢复和本地构建/单元测试的大部分验证。

当前结论：

| 类别 | 结论 |
|---|---|
| 线上版本门禁 | PASS |
| 普通用户创建 token | PASS |
| 真实 Claude / Codex 调用 | PASS |
| SFT 容错成功链路 | PASS |
| SFT 全失败链路 | PASS |
| `logs.type` 兼容 | PASS |
| 计费计量不变量 | PASS |
| 渠道恢复 | PASS |
| 前端生产构建 | PASS |
| 线上前端 UI 视觉验收 | BLOCKED |
| 本地 Go 相关测试 | PARTIAL PASS |

本轮发现 2 个需要记录的问题：

1. `relay/helper` 本地单元测试 `TestStreamScannerHandler_StreamStatus_PreInitialized` 失败，期望流状态为 `1`，实际为 `0`。
2. Codex 内置浏览器访问 `https://nacp.m.srl/login` 时页面白屏，DOM snapshot 为空，控制台无错误；HTTP 层 HTML/JS/CSS 均可访问，因此本轮不能把 UI 视觉验收记为通过。

---

## 2. 测试账号与 token

只记录非敏感信息，不记录 token key。

| 项 | 值 |
|---|---|
| 普通用户 | `nacp02_002336` |
| 用户 ID | `12` |
| Token 名 | `t_v020_002336` |
| Token ID | `12` |
| 初始测试额度 | `2500000` quota |

---

## 3. 真实调用结果

### 3.1 Claude 直接成功

| 项 | 值 |
|---|---|
| 接口 | `/v1/chat/completions` |
| 模型 | `claude-haiku-4-5-20251001` |
| HTTP 状态 | `200` |
| Request ID | `202605151624451140520698268d9d6PlYjDo2Z` |
| 命中渠道 | `12 - MOCK-Controllable-P100` |
| 日志 ID | `145` |
| `logs.type` | `2` |
| `trace_role` | `consume` |
| `trace_seq` | `1` |
| quota | `25` |
| 输入/输出 tokens | `20 / 6` |
| 结论 | PASS |

说明：这条是直接成功，不应展示为容错重试成功。

### 3.2 Codex 直接成功

| 项 | 值 |
|---|---|
| 接口 | `/v1/responses` |
| 模型 | `gpt-5.3-codex` |
| HTTP 状态 | `200` |
| Request ID | `202605151624473768823398268d9d6heW2jOtp` |
| 命中渠道 | `16 - NACP-test-codex` |
| 日志 ID | `146` |
| `logs.type` | `2` |
| `trace_role` | `consume` |
| `trace_seq` | `1` |
| quota | `328` |
| 输入/输出 tokens | `365 / 20` |
| 结论 | PASS |

---

## 4. SFT 容错成功链路

测试方法：

1. 将 Mock 12 控制状态设置为 `500`。
2. 使用普通用户 token 调用 Claude。
3. 请求完成后立即恢复 Mock 状态为 `200`。
4. 通过 `/api/log/grouped`、`/api/log/trace` 和数据库交叉验证。

| 项 | 值 |
|---|---|
| Request ID | `202605151634024149938738268d9d6m3vEE6rT` |
| HTTP 状态 | `200` |
| 用户最终结果 | 成功 |
| trace steps | `6` |
| 最终节点 | `type=2`, `trace_role=consume`, `channel_id=13` |
| 最终 quota | `111` |
| 结论 | PASS |

链路顺序：

| seq | Log ID | type | trace_role | channel | status | quota |
|---:|---:|---:|---|---:|---:|---:|
| 1 | 147 | 5 | `error_intercepted` | 12 | - | 0 |
| 2 | 148 | 5 | `error_intercepted` | 12 | 500 | 0 |
| 3 | 149 | 5 | `error_intercepted` | 12 | 500 | 0 |
| 4 | 150 | 5 | `probe_failed` | 14 | 0 | 0 |
| 5 | 151 | 5 | `probe_failed` | 13 | 0 | 0 |
| 6 | 153 | 2 | `consume` | 13 | - | 111 |

验证点：

1. 容错成功链路最后是 `consume`，不是 `error_visible`。
2. 拦截错误和探测行 quota 都是 0。
3. 探测行 `user_id=0`、`token_id=0`。
4. `/api/log/grouped` 返回真实行，`is_summary=false`，没有虚拟 20 行。
5. `logs.type` 仍是 `2` 和 `5`，没有写入 `20/21/29/51/52/59`。

注意：本次轻量探测返回 `probe_failed`，但后续真实请求 channel 13 成功。这个现象需要后续单独评估探测实现是否完全适配 Claude/AWS 认证方式；它没有影响本次最终容错成功，但会影响“预探测健康判断”的准确性。

---

## 5. SFT 全失败链路

第一次尝试：

1. 临时禁用 13/14/15/17。
2. Mock 12 返回 500。
3. 返回 `503 No available channel`。
4. 没有进入 relay/SFT，因此没有日志链路。

该尝试不能作为 52 收尾证据。

正式测试方法：

1. 保持 13/14/15/17 为启用状态。
2. 临时将 13/14/15/17 的 `base_url` 指向 Mock。
3. Mock 返回 500。
4. 调用 Claude。
5. 请求完成后恢复 13/14/15/17 原 `base_url`，并恢复 Mock 为 200。

| 项 | 值 |
|---|---|
| Request ID | `202605151641575233889398268d9d6I356YgeP` |
| HTTP 状态 | `500` |
| 用户最终结果 | 失败，返回 `mock upstream status 500` |
| trace steps | `15` |
| 最终节点 | `type=5`, `trace_role=error_visible`, `channel_id=15` |
| 最终 quota | `0` |
| 结论 | PASS |

链路顺序：

| seq | Log ID | type | trace_role | channel | status | quota |
|---:|---:|---:|---|---:|---:|---:|
| 1 | 155 | 5 | `error_intercepted` | 12 | - | 0 |
| 2 | 156 | 5 | `error_intercepted` | 12 | 500 | 0 |
| 3 | 157 | 5 | `probe_failed` | 14 | 500 | 0 |
| 4 | 158 | 5 | `probe_failed` | 13 | 500 | 0 |
| 5 | 159 | 5 | `error_intercepted` | 12 | 500 | 0 |
| 6 | 160 | 5 | `error_intercepted` | 13 | - | 0 |
| 7 | 161 | 5 | `probe_failed` | 14 | 500 | 0 |
| 8 | 162 | 5 | `probe_failed` | 17 | 500 | 0 |
| 9 | 163 | 5 | `error_intercepted` | 13 | 500 | 0 |
| 10 | 164 | 5 | `error_intercepted` | 13 | 500 | 0 |
| 11 | 165 | 5 | `error_intercepted` | 15 | - | 0 |
| 12 | 166 | 5 | `probe_failed` | 17 | 500 | 0 |
| 13 | 167 | 5 | `error_intercepted` | 15 | 500 | 0 |
| 14 | 168 | 5 | `error_intercepted` | 15 | 500 | 0 |
| 15 | 169 | 5 | `error_visible` | 15 | - | 0 |

验证点：

1. 容错失败链路最终确实有 `error_visible` 收尾，对应展示语义 52。
2. 不存在“50 只有 51 没有 52”的问题。
3. 全链路失败没有产生成功消费扣费。
4. `/api/log/grouped` 返回 15 条真实行，全部 `is_summary=false`。
5. `trace_seq` 从 1 到 15，顺序可由结构化字段还原。

---

## 6. 数据库不变量

### 6.1 字段与类型兼容

| 检查项 | 结果 |
|---|---:|
| `logs` trace 字段数量 | 5 |
| `type in (20,21,29,50,51,52,59)` | 0 |

结论：PASS。

### 6.2 用户与 token 扣费

| 项 | 值 |
|---|---:|
| 用户剩余额度 | 2499536 |
| 用户已用额度 | 464 |
| 用户请求数 | 3 |
| Token 剩余额度 | 2499536 |
| Token 已用额度 | 464 |

成功消费日志：

| role | type | 数量 | quota | 输入 tokens | 输出 tokens |
|---|---:|---:|---:|---:|---:|
| `consume` | 2 | 3 | 464 | 506 | 46 |

结论：PASS。三条成功消费日志 quota 合计为 `25 + 328 + 111 = 464`，与用户和 token 已用额度一致。

### 6.3 SFT 探测与错误不计费

按 SFT Request ID 汇总：

| Request ID | role | type | 数量 | quota | user_id 合计 |
|---|---|---:|---:|---:|---:|
| `202605151634024149938738268d9d6m3vEE6rT` | `error_intercepted` | 5 | 3 | 0 | 36 |
| `202605151634024149938738268d9d6m3vEE6rT` | `probe_failed` | 5 | 2 | 0 | 0 |
| `202605151634024149938738268d9d6m3vEE6rT` | `consume` | 2 | 1 | 111 | 12 |
| `202605151641575233889398268d9d6I356YgeP` | `error_intercepted` | 5 | 9 | 0 | 108 |
| `202605151641575233889398268d9d6I356YgeP` | `probe_failed` | 5 | 5 | 0 | 0 |
| `202605151641575233889398268d9d6I356YgeP` | `error_visible` | 5 | 1 | 0 | 12 |

探测行：

| 条件 | 结果 |
|---|---|
| `trace_role like 'probe%'` | `user_id=0`, `token_id=0`, `quota=0`, `prompt_tokens=0`, `completion_tokens=0` |

结论：PASS。

### 6.4 后台 degraded probe

本轮测试后，后台恢复探测产生了若干无 `trace_id` 的 probe 行：

| trace_role | 数量 | quota |
|---|---:|---:|
| `probe_failed` | 5 | 0 |
| `probe_success` | 3 | 0 |
| `other` | 1 | 0 |

这些行没有污染 `logs.type`，forbidden type 仍为 0。

---

## 7. 日志 API 验证

| API | 验证结果 |
|---|---|
| `/api/log/grouped?request_id=...` | PASS，返回真实日志行，不返回虚拟摘要行 |
| `/api/log/trace?request_id=...` | PASS，按 `trace_seq` 升序返回完整链路 |
| `/api/log/grouped?type=5` | PASS，类型筛选仍按 NewAPI 原生 `type=5` 工作 |

注意：

1. `/api/log/grouped` 的渠道字段名是 `channel`，不是 `channel_id`；`/api/log/trace` 使用 `channel_id`。这是当前接口契约差异，不是本次运行错误。
2. `grouped` 结果按 `id DESC`，所以视觉上最新日志在前；`trace` 结果按 `trace_seq ASC`，用于链路还原。

---

## 8. 渠道恢复验证

SFT 全失败测试后已恢复渠道配置：

| ID | 渠道 | 状态 | base_url |
|---:|---|---:|---|
| 12 | `MOCK-Controllable-P100` | 1 | `http://172.19.0.1:18080` |
| 13 | `NACP-test-CCM` | 1 | `https://ailink.dog` |
| 14 | `NACP-test-aws` | 1 | `https://ailink.dog` |
| 15 | `NACP-test-kiro` | 1 | `https://ailink.dog` |
| 16 | `NACP-test-codex` | 1 | `https://ailink.dog` |
| 17 | `NACP-test-claude` | 1 | `https://ailink.dog` |

Mock 控制状态已恢复为 `200`。

结论：PASS。

---

## 9. 本地测试与构建

### 9.1 Go 测试

第一次执行 `go test` 使用默认 Go cache，因沙箱无法写入 `~/Library/Caches/go-build` 失败。

第二次使用 `GOCACHE=/private/tmp/nacp-go-cache` 后结果：

| 包 | 结果 |
|---|---|
| `./service` | PASS |
| `./model` | PASS |
| `./relay/common` | PASS |
| `./dto` | PASS |
| `./common` | PASS |
| `./setting/operation_setting` | PASS |
| `./setting/model_setting` | PASS |
| `./relay/helper` | FAIL |

失败测试：

```text
TestStreamScannerHandler_StreamStatus_PreInitialized
expected: 1
actual:   0
```

结论：PARTIAL PASS。该失败属于流扫描状态单测，需要单独排查；本轮 SFT 日志、trace、计费测试未直接依赖该单测。

### 9.2 前端生产构建

命令：

```bash
cd web
bun run build
```

结果：PASS，构建成功。

警告：

1. `--localstorage-file` provided without a valid path。
2. `Browserslist` 数据 12 个月未更新。
3. `lottie-web` 使用 `eval`。
4. 多个 chunk 超过 500 kB。

这些是构建警告，不阻断本轮测试。

---

## 10. 前端 UI 验证状态

HTTP 层验证：

| 资源 | 状态 |
|---|---|
| `/` HTML | 200 |
| `/assets/index-D4nQHqPb.js` | 200 |
| `/assets/index-UfzNzYYF.css` | 200 |

Codex 内置浏览器验证：

| 项 | 结果 |
|---|---|
| URL | `https://nacp.m.srl/login` |
| title | `New API` |
| DOM snapshot | 空 |
| 页面截图 | 白屏 |
| console errors | 空 |

结论：BLOCKED。当前无法用 Codex 内置浏览器完成日志页面视觉验收；需要在普通 Chrome/手动浏览器或可用 Playwright 环境中复测：

1. 日志列表列对齐。
2. 容错成功链路展开。
3. 容错失败链路展开。
4. Log ID hover / copy。
5. 子行详情字段完整性。

---

## 11. 本轮通过项

| ID | 结论 |
|---|---|
| GATE-PRE-01 | PASS |
| GATE-PRE-04 | PASS |
| GATE-REL-04 | PASS |
| CAF-0200-AUTH-01/02/03 | PASS |
| CAF-0200-SFT-01 | PASS |
| CAF-0200-SFT-02 | PASS |
| CAF-0200-SFT-10 | PASS |
| CAF-0200-SFT-30/31 | PASS |
| CAF-0200-SFT-32 | PASS |
| CAF-0200-API-01/02/03/04/05/07 | PASS |
| CAF-0200-BILL-01/02/03/04/05/10 | PASS |
| CAF-0200-DEP-01 | PASS |

---

## 12. 待补测项

| 项 | 原因 | 优先级 |
|---|---|---|
| 日志页面视觉验收 | Codex 内置浏览器白屏，无法完成 | P1 |
| Log ID hover/copy | 需要真实 UI | P1 |
| 20/50 展开视觉语义 | API 已有证据，UI 未验收 | P1 |
| 客户端取消场景 | 本轮未执行，风险仍高 | P0 |
| 流式 SFT 场景 | 本轮未执行流式容错 | P1 |
| `probe_failed` 后真实请求成功的探测准确性 | 轻量探测返回失败但 channel 13 最终成功 | P1 |
| `relay/helper` 流扫描单测失败 | 本地测试失败 | P1 |
| SQLite/PostgreSQL 迁移运行 | 本轮只验证线上 MySQL | P1 |

---

## 13. 建议

1. 先把本轮结果作为 0.2.0 第一轮在线测试报告归档。
2. 单独排查 `TestStreamScannerHandler_StreamStatus_PreInitialized`。
3. 单独排查轻量探测对 Claude/AWS/Kiro 渠道的认证和请求体适配，避免探测误判。
4. 用可工作的浏览器环境复测日志 UI，重点验证你之前指出的 20/50/51/52/21 展开显示。
5. 下一轮继续补测客户端取消、流式容错、SQLite/PostgreSQL。
