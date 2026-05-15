# NACP CAF 执行手册

> **CAF**：NACP Change Assurance Framework
> **用途**：每次新增、增强、修改、修复之后，用于生成标准化深度测试计划和执行记录。
> **创建日期**：2026-05-15

---

## 一、每次变更的固定流程

```text
1. 读取变更说明
2. 定位相关 NewAPI 原版代码
3. 定位 NACP 已改动代码
4. 查 `.ai` 既有系统知识
5. 画影响树
6. 分风险等级
7. 生成测试计划
8. 执行本地开发环境测试
9. 本地通过后，部署并执行线上测试服务器测试
10. 测试服务器通过后，部署正式服务器并观察
11. 留存证据
12. 回流知识
```

环境推进门禁：

```text
Local PASS -> Test Server PASS -> Production Deploy + Observe
```

任何 P0/P1 问题都必须回到本地修复，再重新走本地测试和测试服务器测试。

---

## 二、影响树模板

```text
变更主题：

直接影响：
- 

间接影响：
- 

数据影响：
- 表：
- 字段：
- 索引：
- 迁移：

计费/计量影响：
- 预扣费：
- 结算：
- 退款：
- token 计量：

日志/统计影响：
- logs.type：
- trace_*：
- other：
- dashboard：

前端影响：
- 页面：
- 表格列：
- 筛选：
- 详情：

线上影响：
- 真实渠道：
- 真实 token：
- 部署：
- 回滚：
```

---

## 三、测试计划表模板

| ID | 场景 | 风险 | 前置条件 | 步骤 | 预期结果 | 证据 | 结论 |
|----|------|------|----------|------|----------|------|------|
| CAF-001 |  | P0/P1/P2/P3 |  |  |  |  | 未执行 |

---

## 三点五、环境分阶段测试表模板

### Stage A：本地开发环境

| ID | 场景 | 命令/API/页面 | 预期 | 证据 | 结论 |
|----|------|---------------|------|------|------|
| L-001 | 相关单元测试 | `go test ...` / `bunx vitest ...` | PASS | 输出摘要 | 未执行 |
| L-002 | 本地 API 冒烟 | `/api/status`、目标 API | 字段正确 | API 摘要 | 未执行 |
| L-003 | 本地前端显示 | `http://localhost:5173/...` | 展示正确 | 截图或字段说明 | 未执行 |

本地未通过时，禁止进入测试服务器。

### Stage B：线上测试服务器

| ID | 场景 | 测试站环境 | 预期 | 证据 | 结论 |
|----|------|------------|------|------|------|
| T-001 | 版本确认 | commit / image / container | 与待发布版本一致 | 命令输出 | 未执行 |
| T-002 | 真实渠道请求 | 测试账号 + 测试 token + 测试渠道 | 请求成功或符合预期失败 | request_id / 日志 | 未执行 |
| T-003 | 计费与统计 | 用户、token、channel、logs | 不被错误和 probe 污染 | API/DB 摘要 | 未执行 |
| T-004 | 前端测试站显示 | 测试站页面 | 展示正确 | 截图或字段说明 | 未执行 |

测试服务器未通过时，禁止部署正式服务器。

### Stage C：正式服务器部署后观察

| ID | 场景 | 操作 | 预期 | 证据 | 结论 |
|----|------|------|------|------|------|
| P-001 | 部署确认 | 正式服版本/容器 | 新版本运行 | 命令输出 | 未执行 |
| P-002 | 低风险冒烟 | 健康检查、只读 API、少量真实请求 | 无明显异常 | API/日志摘要 | 未执行 |
| P-003 | 观察窗口 | 错误率、计费、日志、统计 | 无回归 | 监控/日志摘要 | 未执行 |

正式服不得作为首次发现核心问题的测试环境；正式服异常优先回滚或止血。

---

## 四、NACP 系统域初始清单

| 系统域 | 典型代码 | 必查风险 |
|--------|----------|----------|
| Relay 请求链路 | `controller/relay.go`, `relay/` | 请求转换、错误处理、流式、响应格式 |
| 渠道选择 | `middleware/`, `service/channel_*`, `model/ability.go` | 优先级、权重、auto 分组、健康状态、亲和 |
| 计费系统 | `service/billing*`, `helper.ModelPriceHelper` | 预扣费、结算、退款、订阅、违规扣费 |
| 计量系统 | token estimator, provider response usage | prompt/completion/cache/audio/task 计量 |
| 日志系统 | `model/log.go`, `service/trace.go`, `service/log_grouped.go` | 类型兼容、trace 字段、详情完整性 |
| 统计系统 | dashboard / usage stats | 消费统计、错误统计、渠道统计污染 |
| 用户与 token | `model/user.go`, token controller | 普通用户、管理员、token 权限 |
| 前端日志页 | `web/src/components/table/usage-logs/` | 列、筛选、展开、复制、i18n |
| 数据库迁移 | `model/main.go` | SQLite/MySQL/PostgreSQL 兼容 |
| 部署流程 | `gogogo.sh`, Docker | 构建、推送、迁移、回滚 |

---

## 五、CAF 输出结论模板

```text
结论：通过 / 有条件通过 / 不通过

已覆盖：
- 

未覆盖：
- 

发现问题：
- 

剩余风险：
- 

上线建议：
- 

需要回流到 .ai：
- 
```

---

## 六、日志页与链路功能测试经验

来源：2026-05-16 本地 `http://localhost:5173/console/log` 空表排查。

### 6.1 先用 API 定位，不先截图模拟

日志、trace、计费、统计类问题不要先从浏览器交互开始。标准顺序：

```text
1. curl /api/status
2. curl /api/user/login，保存 cookie
3. curl /api/log/grouped，检查列表原始数据
4. curl /api/log/trace?request_id=...，检查单条链路详情
5. curl /api/log/traces，检查链路摘要聚合
6. 必要时查 DB 原始 logs 行
7. 最后打开 /console/log 验证展示层
```

原因：前端日志页会做二次派生，包括隐藏、折叠、生成 20/50 摘要、把 `trace_role` 映射成 21/29/51/52/59。直接看页面容易把数据问题误判为 UI 问题，或把 UI 派生问题误判为底层逻辑问题。

### 6.2 `/api/log/grouped` 默认列表检查

必查字段：

| 字段 | 判断 |
|------|------|
| `total` | 后端查询总数是否合理 |
| `items.length` | 当前页是否有数据 |
| `id` | 每条日志唯一标识 |
| `type` | 必须保持 NewAPI 原生类型，不写入 20/21/29/50/51/52/59 |
| `trace_role` | NACP 展示语义来源 |
| `request_id` | 是否属于真实用户请求链路 |
| `trace_id` / `trace_seq` | 是否可还原链路顺序 |
| `quota` / token 字段 | probe/intercepted error 不得产生消费 |

经验规则：

1. 如果页面为空，先看 `/api/log/grouped` 第一页是否全是会被前端隐藏的数据。
2. 没有 `request_id` 的后台独立 probe 不能占满默认日志列表。
3. 有 `request_id` 的 probe 属于真实请求链路，必须保留给 trace 展开使用。
4. 默认日志列表不应该被后台健康任务、周期 probe、异步维护日志污染。

### 6.3 `/api/log/traces` 摘要聚合检查

必须验证：

1. 同一个 `request_id` 只能返回一条摘要。
2. 探测日志没有 `username/token_name` 时，不能把同一链路拆成“空用户 failed 摘要”和“真实用户 success 摘要”。
3. 成功链路状态以是否存在最终 `consume(type=2)` 为准。
4. 失败链路状态以无 `consume` 且存在 `error_visible` 或错误链路为准。
5. `total_quota`、`total_prompt_tokens`、`total_completion_tokens` 只统计 `type=2`。

回归测试锚点：

```bash
go test ./service -run 'TestGroupedLogs|TestTrace'
```

### 6.4 `/api/log/trace` 链路详情检查

必须验证：

1. 步骤按 `trace_seq` 正序输出。
2. 成功链路最后应为 `trace_role=consume`，展示层派生为 `21` 或 `2`。
3. 容错失败链路最后应为 `trace_role=error_visible`，展示层派生为 `52`。
4. `probe_success/probe_failed` 在链路内分别派生为 `29/59`。
5. `error_intercepted` 派生为 `51`，不得直接对用户可见。
6. probe/intercepted error 的 `quota/prompt_tokens/completion_tokens` 应为 0。

当前已知限制：

1. 旧日志没有 SFT trace 字段，不需要强行推断，只需正常显示。
2. 当前 `trace_parent_id/trace_sibling_seq` 可能仍为空或 0，线性顺序主要依赖 `trace_seq`。
3. 如果未来要展示严谨父子树，必须先让写日志路径结构化写入父子/同级字段，不能只靠时间推断。
