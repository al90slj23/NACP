# CAF classic-plus-0.2.0 本地 API 深度测试计划

> 适用版本：`classic-plus-0.2.0-dev`
> 测试阶段：CAF Stage A，本地开发环境测试
> 本地入口：`http://localhost:5173/`
> 验证方式：以 API 调用为主，浏览器页面只做显示层抽查
> 生成日期：2026-05-16

---

## 1. 版本判断

### 1.1 版本关系

NACP 的版本线基于 NewAPI v0.13.2 classic 分支。

| 层级 | 版本 | 定位 |
|------|------|------|
| 上游基线 | NewAPI v0.13.2 | NACP 的基础代码版本 |
| NACP 初始基础版 | `classic-plus-0.1.0` | 第一个 NACP 增强版本，完成智能重试与渠道健康管理 |
| 当前待测版 | `classic-plus-0.2.0-dev` | 在 0.1.x 基础上补齐 SFT 结构化链路、日志兼容、CAF 和版本治理 |

结论：

```text
NewAPI v0.13.2 是上游基础。
NACP 的第一个增强版本是 classic-plus-0.1.0。
当前本地要验证的是 classic-plus-0.2.0-dev。
```

### 1.2 0.1.0 -> 0.2.0 升级摘要

0.1.0 解决“请求失败时如何更聪明地重试和切换渠道”。

0.2.0 解决“这些容错行为如何被结构化记录、兼容旧日志、被管理员正确理解、被标准化测试”。

| 领域 | 0.1.0 | 0.2.0-dev 本地要测什么 |
|------|-------|------------------------|
| 智能重试 | 同渠道重试、健康状态、预热探测、同优先级切换 | 这些行为是否仍能被 trace 还原，且不破坏最终结果 |
| 日志模型 | 曾出现新语义编号方案 | `logs.type` 只存 NewAPI 原生类型，新语义进入 `trace_role` |
| 链路结构 | 主要依赖 request_id/use_channel | `trace_id`、`trace_seq`、`trace_parent_id`、`trace_sibling_seq`、`trace_role` |
| 日志 API | 普通日志接口为主 | `/api/log/grouped`、`/api/log/traces`、`/api/log/trace` |
| 前端日志页 | 普通日志展示 | 类型派生、展开行、Log ID、字段完整、后台 probe 不刷空 |
| 兼容 | 重点在运行时容错 | 旧日志不强行推断，新日志可结构化还原 |
| 计费统计 | 关注最终请求是否成功 | 探测、拦截错误、取消请求不得污染余额和统计 |
| 测试体系 | 有专项测试计划 | CAF 正式约束本地 -> 测试服务器 -> 正式服 |
| 版本治理 | 初步 changelog | `VERSION`、`CHANGELOG.md`、release 文档、CAF 证据 |

---

## 2. 本地测试目标

本计划只验证 CAF Stage A：本地开发环境。

本地通过后，才能进入 Stage B：线上测试服务器测试。

本地测试要证明：

1. `http://localhost:5173/` 的 API proxy 正常指向本地后端。
2. 当前本地后端实际运行的是 0.2.0 代码。
3. 0.2.0 新增日志字段和 API 可用。
4. `/api/log/grouped` 默认列表不被后台 standalone probe 刷空。
5. `/api/log/traces` 同一 `request_id` 只聚合成一条摘要。
6. `/api/log/trace` 能完整返回链路步骤。
7. `logs.type` 不写入 `20/21/29/50/51/52/59`。
8. `trace_role` 正确表达 `consume/error_intercepted/error_visible/probe_success/probe_failed`。
9. 普通用户 token 的真实请求可在管理员日志中追踪。
10. 计费、计量、统计只统计最终消费，不被 probe 和 intercepted error 污染。
11. 旧日志仍能正常显示、筛选，不被强行推断成 SFT。
12. 本地 API 验证通过后，再少量打开 `/console/log` 做显示层抽查。

---

## 3. 本地环境门禁

| ID | 检查项 | API / 命令 | 通过标准 | 风险 |
|----|--------|------------|----------|------|
| L-GATE-01 | 前端 Vite | `curl http://localhost:5173/api/status` | `success=true` | P0 |
| L-GATE-02 | 后端 API | `/api/status` 返回 `server_address` | 指向本地后端，例如 `http://localhost:3000` | P0 |
| L-GATE-03 | 版本文件 | `cat VERSION` | `classic-plus-0.2.0-dev` | P1 |
| L-GATE-04 | 数据库环境 | 检查 `.env` / 启动日志 | 确认不是正式生产库 | P0 |
| L-GATE-05 | 管理员会话 | `/api/user/login` | 管理员登录成功，能访问日志 API | P0 |
| L-GATE-06 | 必要字段 | `/api/log/grouped` 返回字段 | 有 `id/type/trace_role/request_id/trace_id/trace_seq` | P0 |
| L-GATE-07 | 后台 probe 过滤 | `/api/log/grouped` 第一页 | 不被无 `request_id` 的 probe 占满 | P0 |
| L-GATE-08 | 本地测试隔离 | 测试用户名/token 前缀 | 使用唯一前缀 `nacp_l_v020_<stamp>` | P1 |

本地数据库特别要求：

```text
如果本地 SQL_DSN 指向远程测试库，可以继续测试。
如果指向正式生产库，禁止执行会写入数据的测试。
```

---

## 4. API 通用执行模板

以下命令是计划模板，不记录真实密码和渠道 key。

### 4.1 环境变量

```bash
export BASE='http://localhost:5173'
export COOKIE='/private/tmp/nacp-local-v020-cookie.txt'
export ADMIN_USER='<admin_username>'
export ADMIN_PASS='<admin_password>'
export STAMP="$(date +%Y%m%d%H%M%S)"
export TEST_PREFIX="nacp_l_v020_${STAMP}"
```

### 4.2 管理员登录

```bash
curl -sS -c "$COOKIE" \
  -H 'Content-Type: application/json' \
  -H 'New-API-User: -1' \
  -X POST "$BASE/api/user/login" \
  --data "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\"}" | jq
```

后续管理员 API 固定带：

```bash
-b "$COOKIE" -H 'New-API-User: 1'
```

### 4.3 基础状态

```bash
curl -sS "$BASE/api/status" | jq '{success, version:.data.version, system_name:.data.system_name, server_address:.data.server_address}'
```

### 4.4 日志核心接口

```bash
curl -sS -b "$COOKIE" -H 'New-API-User: 1' \
  "$BASE/api/log/grouped?p=1&page_size=20&type=0&start_timestamp=0&end_timestamp=9999999999" | jq

curl -sS -b "$COOKIE" -H 'New-API-User: 1' \
  "$BASE/api/log/traces?p=1&page_size=20&start_timestamp=0&end_timestamp=9999999999" | jq

curl -sS -b "$COOKIE" -H 'New-API-User: 1' \
  "$BASE/api/log/trace?request_id=<request_id>" | jq
```

### 4.5 用户 token 请求模板

```bash
curl -sS "$BASE/v1/chat/completions" \
  -H "Authorization: Bearer sk-<user_token>" \
  -H 'Content-Type: application/json' \
  --data '{
    "model":"claude-haiku-4-5-20251001",
    "messages":[{"role":"user","content":"NACP local v020 test"}],
    "max_tokens":16
  }' | jq

curl -sS "$BASE/v1/responses" \
  -H "Authorization: Bearer sk-<user_token>" \
  -H 'Content-Type: application/json' \
  --data '{
    "model":"gpt-5.3-codex",
    "input":"NACP local v020 codex test"
  }' | jq
```

---

## 5. 影响树

```text
classic-plus-0.2.0-dev
├── SFT 结构化链路
│   ├── trace_id
│   ├── trace_seq
│   ├── trace_parent_id
│   ├── trace_sibling_seq
│   └── trace_role
├── 日志 API
│   ├── /api/log/grouped
│   ├── /api/log/traces
│   └── /api/log/trace
├── 日志前端
│   ├── /console/log
│   ├── 类型派生
│   ├── 展开行
│   ├── Log ID
│   └── 后台 probe 隐藏
├── 计费/计量/统计
│   ├── 最终 consume 扣费
│   ├── probe 不扣费
│   ├── intercepted error 不扣费
│   └── dashboard/log stat 不污染
├── 用户/token/权限
│   ├── 管理员日志
│   ├── 普通用户 token
│   ├── token usage
│   └── 权限隔离
├── 兼容性
│   ├── NewAPI 原生日志 type 1..9
│   ├── 旧日志无 trace 字段
│   ├── 三数据库 SQL
│   └── 旧筛选入口
└── 版本/CAF
    ├── VERSION
    ├── CHANGELOG
    ├── .ai release review
    └── CAF Stage A/B/C 门禁
```

---

## 6. 核心不变量

| ID | 不变量 | 本地验证方式 | 风险 |
|----|--------|--------------|------|
| INV-TYPE-01 | `logs.type` 不存 `20/21/29/50/51/52/59` | API type 查询或 DB 查询 | P0 |
| INV-TYPE-02 | 直接成功为 `type=2 + trace_role=consume` | `/api/log/grouped` | P0 |
| INV-TRACE-01 | 同一请求链路共享 `trace_id/request_id` | `/api/log/trace` | P0 |
| INV-TRACE-02 | 链路步骤按 `trace_seq` 正序 | `/api/log/trace` | P0 |
| INV-TRACE-03 | 成功链路最终是 `consume` | `/api/log/trace` | P0 |
| INV-TRACE-04 | 失败链路最终是 `error_visible` | `/api/log/trace` | P0 |
| INV-TRACE-05 | 同一 `request_id` 在 `/api/log/traces` 只有一条摘要 | `/api/log/traces` | P0 |
| INV-PROBE-01 | 无 `request_id` 的后台 standalone probe 不占默认日志列表 | `/api/log/grouped` 第一页 | P0 |
| INV-PROBE-02 | 有 `request_id` 的链路内 probe 必须保留 | `/api/log/trace` | P0 |
| INV-BILL-01 | probe/intercepted error quota 为 0 | `/api/log/trace` | P0 |
| INV-BILL-02 | 最终消费只扣一次 | 用户/token 前后额度 + consume 日志 | P0 |
| INV-COMPAT-01 | 旧日志正常显示，不强行推断 | `/api/log/grouped` + `/console/log` | P1 |
| INV-API-01 | 管理员接口必须带 `New-API-User` | 缺 header 应失败，带 header 应成功 | P1 |

---

## 7. 本地测试用例总表

### 7.1 Stage A 基础门禁

| ID | 场景 | API / 命令 | 预期 | 风险 |
|----|------|------------|------|------|
| L-0200-GATE-01 | 状态接口 | `GET /api/status` | success=true | P0 |
| L-0200-GATE-02 | 版本文件 | `cat VERSION` | `classic-plus-0.2.0-dev` | P1 |
| L-0200-GATE-03 | 管理员登录 | `POST /api/user/login` | 返回 role=100 | P0 |
| L-0200-GATE-04 | 管理员 header | 带/不带 `New-API-User` 调日志 API | 不带失败，带成功 | P1 |
| L-0200-GATE-05 | grouped 可用 | `GET /api/log/grouped` | success=true，items 可解析 | P0 |
| L-0200-GATE-06 | traces 可用 | `GET /api/log/traces` | success=true | P0 |
| L-0200-GATE-07 | trace 可用 | 使用已知 request_id 调 `/api/log/trace` | success=true | P0 |

### 7.2 版本升级覆盖

| ID | 场景 | API / 命令 | 预期 | 风险 |
|----|------|------------|------|------|
| L-0200-REL-01 | 0.1.0 基础确认 | `CHANGELOG.md` / `.ai` | 0.1.0 为智能重试与健康管理 | P2 |
| L-0200-REL-02 | 0.2.0 当前确认 | `VERSION` / `CHANGELOG.md` | 当前为 0.2.0-dev | P1 |
| L-0200-REL-03 | 升级范围一致 | release 文档对比 changelog | SFT/日志/CAF/版本治理一致 | P1 |
| L-0200-REL-04 | 保护信息未变更 | `git diff` 检查 README/元信息 | 不修改受保护 NewAPI/QuantumNous 标识 | P0 |

### 7.3 日志类型兼容

| ID | 场景 | API / 命令 | 预期 | 风险 |
|----|------|------------|------|------|
| L-0200-TYPE-01 | 20 查询 | `/api/log/grouped?type=20` | total=0 | P0 |
| L-0200-TYPE-02 | 21 查询 | `/api/log/grouped?type=21` | total=0 | P0 |
| L-0200-TYPE-03 | 29 查询 | `/api/log/grouped?type=29` | total=0 | P0 |
| L-0200-TYPE-04 | 50 查询 | `/api/log/grouped?type=50` | total=0 | P0 |
| L-0200-TYPE-05 | 51 查询 | `/api/log/grouped?type=51` | total=0 | P0 |
| L-0200-TYPE-06 | 52 查询 | `/api/log/grouped?type=52` | total=0 | P0 |
| L-0200-TYPE-07 | 59 查询 | `/api/log/grouped?type=59` | total=0 | P0 |
| L-0200-TYPE-08 | 原生消费查询 | `/api/log/grouped?type=2` | 只返回真实 consume | P0 |
| L-0200-TYPE-09 | 原生错误查询 | `/api/log/grouped?type=5` | 返回错误/拦截/probe_failed，但 type 保持 5 | P1 |
| L-0200-TYPE-10 | 原生系统查询 | `/api/log/grouped?type=4` | 可能含 probe_success，type 保持 4 | P1 |

### 7.4 grouped 默认列表

| ID | 场景 | API / 命令 | 预期 | 风险 |
|----|------|------------|------|------|
| L-0200-GRP-01 | 默认列表非空 | `/api/log/grouped?p=1&page_size=20&type=0` | items 不因 standalone probe 被前端过滤为空 | P0 |
| L-0200-GRP-02 | standalone probe 过滤 | 检查第一页 `request_id="" + trace_role=probe_*` | 默认页不应被此类行占满 | P0 |
| L-0200-GRP-03 | 链路内 probe 保留 | 指定 request_id 查询 | 有 request_id 的 probe 仍返回 | P0 |
| L-0200-GRP-04 | Log ID 唯一 | 检查 items[].id | 每行唯一且非空 | P1 |
| L-0200-GRP-05 | 字段完整 | 检查 items 字段 | id/type/trace_role/request_id/channel/user/token/quota/tokens 存在 | P0 |
| L-0200-GRP-06 | request_id 筛选 | `/api/log/grouped?request_id=...` | 只返回该请求链路 | P0 |
| L-0200-GRP-07 | token 筛选 | `token_name=<测试 token>` | 只返回对应 token | P1 |
| L-0200-GRP-08 | channel 筛选 | `channel=<id>` | 只返回对应 channel | P1 |
| L-0200-GRP-09 | pagination | p=1/2/3 | 排序稳定，无重复异常 | P1 |

### 7.5 traces 摘要列表

| ID | 场景 | API / 命令 | 预期 | 风险 |
|----|------|------------|------|------|
| L-0200-TRS-01 | traces 可查询 | `/api/log/traces` | success=true | P0 |
| L-0200-TRS-02 | request_id 唯一 | 同一 request_id 只一条摘要 | 不被 probe 空用户拆分 | P0 |
| L-0200-TRS-03 | 成功状态 | 有 consume 的链路 | status=success | P0 |
| L-0200-TRS-04 | 失败状态 | 无 consume 且有最终错误 | status=failed | P0 |
| L-0200-TRS-05 | token/user 聚合 | probe 没有 user/token | 摘要仍显示真实用户/token | P0 |
| L-0200-TRS-06 | channel_count | 多渠道链路 | channel_count 包含真实涉及渠道 | P1 |
| L-0200-TRS-07 | total_quota | 成功链路 | 只统计 type=2 quota | P0 |
| L-0200-TRS-08 | status 筛选 | `status=success/failed` | 结果准确 | P1 |

### 7.6 trace 详情

| ID | 场景 | API / 命令 | 预期 | 风险 |
|----|------|------------|------|------|
| L-0200-TRC-01 | 直接成功详情 | `/api/log/trace?request_id=<direct>` | 单步或无容错链路，最终 consume | P0 |
| L-0200-TRC-02 | 容错成功详情 | `/api/log/trace?request_id=<sft_success>` | 51/59/29 + 最终 consume | P0 |
| L-0200-TRC-03 | 容错失败详情 | `/api/log/trace?request_id=<sft_failed>` | 最终 error_visible | P0 |
| L-0200-TRC-04 | 顺序 | steps[].trace_seq | 严格递增或可解释 | P0 |
| L-0200-TRC-05 | 字段完整 | steps[] | id/request_id/sequence/trace_id/trace_seq/trace_role/type/channel/quota/tokens/other | P0 |
| L-0200-TRC-06 | 计费合计 | detail.total_quota/tokens | 等于所有 type=2 之和 | P0 |
| L-0200-TRC-07 | 29/59 显示来源 | trace_role=probe_* | type 分别兼容为 4/5 | P1 |
| L-0200-TRC-08 | 51/52 显示来源 | trace_role=error_* | type 均兼容为 5 | P0 |
| L-0200-TRC-09 | 21 显示来源 | trace_role=consume 且 trace_seq>1 | 前端派生为容错重试成功 | P0 |

### 7.7 普通用户与 token

| ID | 场景 | API / 命令 | 预期 | 风险 |
|----|------|------------|------|------|
| L-0200-AUTH-01 | 管理员确认用户 | `/api/user/search` 或管理 API | 测试用户存在或可创建 | P0 |
| L-0200-AUTH-02 | 普通用户登录 | `/api/user/login` | role 普通用户 | P0 |
| L-0200-AUTH-03 | 创建 token | `POST /api/token/` | token 记录创建成功 | P0 |
| L-0200-AUTH-04 | 获取 token key | `POST /api/token/:id/key` | 可取到真实 key | P0 |
| L-0200-AUTH-05 | token usage | `/api/usage/token` 或 `/api/token/:id` | 额度字段可读 | P1 |
| L-0200-AUTH-06 | 普通用户不能看管理员日志 | 普通用户调 `/api/log/grouped` | 被拒绝 | P0 |
| L-0200-AUTH-07 | token 禁用 | 禁用后调用模型 | 请求失败，不产生 consume | P1 |

### 7.8 真实请求路径

| ID | 场景 | API / 命令 | 预期 | 风险 |
|----|------|------------|------|------|
| L-0200-REQ-01 | Claude 非流式 | `/v1/chat/completions` | HTTP 200 或可解释错误 | P0 |
| L-0200-REQ-02 | Claude 流式 | `/v1/chat/completions stream=true` | SSE 正常，日志流状态正常 | P0 |
| L-0200-REQ-03 | Codex responses | `/v1/responses` | HTTP 200，日志计费过程完整 | P0 |
| L-0200-REQ-04 | 非法模型 | `/v1/chat/completions` with invalid model | 不产生正常 consume | P1 |
| L-0200-REQ-05 | 低额度 token | token 额度不足 | 失败且不扣费 | P0 |
| L-0200-REQ-06 | 模型限制 token | token 限制模型 | 失败且不扣费 | P0 |
| L-0200-REQ-07 | 并发 10 个请求 | shell 并发 curl | request_id/trace_id 不串链 | P0 |

### 7.9 SFT 容错路径

| ID | 场景 | API / 操作 | 预期 | 风险 |
|----|------|------------|------|------|
| L-0200-SFT-01 | 直接成功 | 健康渠道直接调用 | 不生成 20/50 摘要，不误展开 | P0 |
| L-0200-SFT-02 | A 失败 B 成功 | Mock 渠道返回 500/429 | 用户最终成功，trace 最终 consume | P0 |
| L-0200-SFT-03 | 同渠道重试 | A1/A2 失败 A3 成功或转 B | trace_seq 可解释 | P0 |
| L-0200-SFT-04 | 预热探测成功 | 候选 probe_success | 29 语义可由 trace_role 识别，不扣费 | P0 |
| L-0200-SFT-05 | 预热探测失败 | 候选 probe_failed | 59 语义可由 trace_role 识别，不扣费 | P0 |
| L-0200-SFT-06 | 探测成功但真实请求失败 | B1- 成功 B2 失败 | B2 进入错误链路，不误判成功 | P0 |
| L-0200-SFT-07 | 全部失败 | 所有候选失败 | 最终 error_visible，traces status=failed | P0 |
| L-0200-SFT-08 | 50 必须 52 收尾 | 查看失败 trace | 最后客户端可见错误存在 | P0 |
| L-0200-SFT-09 | 20 不能 52 收尾 | 查看成功 trace | 最后 consume，不是 error_visible | P0 |
| L-0200-SFT-10 | 客户端取消 | curl --max-time 或断开 | 不产生 late success 扣费 | P0 |
| L-0200-SFT-11 | 上游超时 | Mock 延迟 | 失败或切换可解释，无重复扣费 | P0 |

### 7.10 计费、计量、统计

| ID | 场景 | API / 命令 | 预期 | 风险 |
|----|------|------------|------|------|
| L-0200-BILL-01 | 直接成功扣费 | 用户/token 额度前后 | delta 等于 consume 日志 quota | P0 |
| L-0200-BILL-02 | 容错成功扣费一次 | SFT 成功链路 | 只最终 consume 扣费 | P0 |
| L-0200-BILL-03 | 拦截错误不扣费 | trace_role=error_intercepted | quota=0，不进消费统计 | P0 |
| L-0200-BILL-04 | probe 不扣费 | trace_role=probe_* | user_id=0 或 quota=0，不扣用户余额 | P0 |
| L-0200-BILL-05 | 最终错误不正常扣费 | error_visible | 不产生 type=2 consume | P0 |
| L-0200-BILL-06 | 日志统计 | `/api/log/stat` | 只统计真实消费 | P0 |
| L-0200-BILL-07 | 用户自统计 | `/api/log/self/stat` | 与管理员按 username 查询一致 | P1 |
| L-0200-BILL-08 | token usage | `/api/usage/token` | 与 token remain/used 一致 | P1 |
| L-0200-BILL-09 | 渠道用量 | 渠道列表或 DB | 最终成功渠道 used_quota 增加，probe 不污染 | P1 |

### 7.11 旧日志和 NewAPI 回归

| ID | 场景 | API / 命令 | 预期 | 风险 |
|----|------|------------|------|------|
| L-0200-COMP-01 | 旧 type=1 topup | `/api/log/grouped?type=1` | 正常返回和显示 | P1 |
| L-0200-COMP-02 | 旧 type=2 consume | `/api/log/grouped?type=2` | 正常返回和显示 | P0 |
| L-0200-COMP-03 | 旧 type=3 manage | `/api/log/grouped?type=3` | 正常返回和显示 | P2 |
| L-0200-COMP-04 | 旧 type=4 system | `/api/log/grouped?type=4` | 正常返回和显示 | P1 |
| L-0200-COMP-05 | 旧 type=5 error | `/api/log/grouped?type=5` | 正常返回和显示 | P1 |
| L-0200-COMP-06 | 无 trace 老日志 | 选择无 trace_id 日志 | 不强行生成 SFT 链路 | P1 |
| L-0200-COMP-07 | 原 `/api/log/` | 旧日志接口 | 未被 grouped/trace 改动破坏 | P1 |
| L-0200-COMP-08 | `/api/log/self` | 普通用户自日志 | 正常 | P1 |
| L-0200-COMP-09 | `/api/log/token` | token 读日志 | 正常 | P1 |

### 7.12 前端显示层抽查

只在 API 通过后执行。

| ID | 场景 | 页面 | 预期 | 风险 |
|----|------|------|------|------|
| L-0200-UI-01 | 日志页加载 | `/console/log` | 非空，无 JS 错误 | P0 |
| L-0200-UI-02 | 直接成功行 | `/console/log` | 显示正常消费成功 | P0 |
| L-0200-UI-03 | 容错成功行 | `/console/log` | 显示容错重试后成功，展开最终为 21 语义 | P0 |
| L-0200-UI-04 | 容错失败行 | `/console/log` | 显示容错重试后失败，展开最终为 52 语义 | P0 |
| L-0200-UI-05 | 字段完整 | 展开行 | 子行列与主行对齐，字段不缺 | P0 |
| L-0200-UI-06 | Log ID | hover/click | 能看完整 ID，能复制 | P1 |
| L-0200-UI-07 | 后台 probe | 默认页 | 不因隐藏 standalone probe 变空表 | P0 |
| L-0200-UI-08 | 筛选组合 | 时间 + request_id + token | 结果准确 | P1 |

### 7.13 性能和稳定性

| ID | 场景 | API / 命令 | 预期 | 风险 |
|----|------|------------|------|------|
| L-0200-PERF-01 | grouped 首屏 | `/api/log/grouped?page_size=20` | 响应稳定 | P1 |
| L-0200-PERF-02 | grouped 100 条 | `page_size=100` | 不超时，字段完整 | P1 |
| L-0200-PERF-03 | trace 详情 | 复杂链路 request_id | 响应稳定，无明显 N+1 | P1 |
| L-0200-PERF-04 | traces 列表 | `/api/log/traces?page_size=100` | 不重复、不拆链 | P1 |
| L-0200-PERF-05 | 并发请求 | 10/20 并发 | trace_id 不串、扣费不串 | P0 |

### 7.14 三数据库本地静态/单测

本地 API 主环境可以是 MySQL，但 0.2.0 改到日志查询和 GORM 聚合，必须补兼容意识。

| ID | 场景 | 命令 | 预期 | 风险 |
|----|------|------|------|------|
| L-0200-DB-01 | Go 相关测试 | `go test ./service -run 'TestGroupedLogs|TestTrace'` | PASS | P0 |
| L-0200-DB-02 | service 全测试 | `go test ./service` | PASS 或记录既有失败 | P1 |
| L-0200-DB-03 | GORM SQL 检查 | review `service/trace.go` / `log_grouped.go` | 不使用 MySQL-only SQL | P1 |
| L-0200-DB-04 | reserved word | `group` 字段查询 | 使用 `logGroupCol()` | P1 |
| L-0200-DB-05 | SQLite/PostgreSQL 后续 | 单独环境或 CI | 迁移和查询可跑 | P1 |

---

## 8. 必跑自动化命令

```bash
go test ./service -run 'TestGroupedLogs|TestTrace'
go test ./service -run 'TestHealth|TestOnUser|TestOnProbe|TestShouldProbe|TestCheckRecovery|TestGetChannel|TestIsChannel|TestProperty|TestDefault|TestGetHealth'
cd web && bun run build
```

如果时间允许：

```bash
go test ./service ./controller ./model
go test ./...
cd web && bunx vitest run
```

如果 `go test ./...` 有既有失败，必须记录：

1. 失败包名。
2. 失败测试名。
3. 是否与 0.2.0 本次改动路径有关。
4. 是否阻断进入测试服务器。

---

## 9. API 检查脚本清单

### 9.1 forbidden type

```bash
for t in 20 21 29 50 51 52 59; do
  curl -sS -b "$COOKIE" -H 'New-API-User: 1' \
    "$BASE/api/log/grouped?p=1&page_size=1&type=$t&start_timestamp=0&end_timestamp=9999999999" \
    | jq -r "\"type=$t total=\" + ((.data.total // 0)|tostring)"
done
```

通过标准：

```text
每个 total 都是 0。
```

### 9.2 grouped 首屏污染检查

```bash
curl -sS -b "$COOKIE" -H 'New-API-User: 1' \
  "$BASE/api/log/grouped?p=1&page_size=20&type=0&start_timestamp=0&end_timestamp=9999999999" \
  | jq '[.data.items[] | {id,type,trace_role,request_id,channel,channel_name,username,token_name}]'
```

通过标准：

1. 不应全部是 `request_id=""` 且 `trace_role=probe_success/probe_failed`。
2. 如果 items 非空，前端默认页不应被过滤成空。

### 9.3 traces 去重检查

```bash
curl -sS -b "$COOKIE" -H 'New-API-User: 1' \
  "$BASE/api/log/traces?p=1&page_size=100&start_timestamp=0&end_timestamp=9999999999" \
  | jq -r '.data.items[].request_id' | sort | uniq -d
```

通过标准：

```text
输出为空。
```

### 9.4 trace 终态检查

```bash
curl -sS -b "$COOKIE" -H 'New-API-User: 1' \
  "$BASE/api/log/trace?request_id=<request_id>" \
  | jq '{request_id:.data.request_id,total_quota:.data.total_quota,steps:[.data.steps[] | {id,sequence,trace_seq,trace_role,type,channel_id,quota,prompt_tokens,completion_tokens}]}'
```

成功链路通过标准：

```text
最后一个业务终态为 trace_role=consume。
total_quota 等于 type=2 步骤 quota 之和。
```

失败链路通过标准：

```text
不存在 type=2 consume。
最终可见错误为 trace_role=error_visible。
```

---

## 10. 本地执行顺序

严格按顺序执行：

1. `L-GATE` 本地门禁。
2. 自动化测试：`TestGroupedLogs|TestTrace`。
3. forbidden type 查询。
4. grouped 首屏污染检查。
5. traces 摘要去重检查。
6. trace 成功/失败详情检查。
7. 普通用户 token 调用 Claude/Codex。
8. 计费、计量、统计核对。
9. 旧日志兼容 API 检查。
10. 并发/分页/性能检查。
11. 打开 `/console/log` 做少量显示层抽查。
12. 汇总 Stage A 结论。

禁止跳步：

```text
如果本地 L-GATE 或 P0 API 检查失败，不得进入线上测试服务器。
```

---

## 11. Stage A 放行标准

本地阶段通过必须满足：

| 门禁 | 标准 |
|------|------|
| 基础服务 | `5173/api/status` 正常 |
| 自动化测试 | `go test ./service -run 'TestGroupedLogs|TestTrace'` PASS |
| 类型兼容 | 20/21/29/50/51/52/59 查询 total=0 |
| grouped 默认页 | 不被 standalone probe 刷空 |
| traces 摘要 | 同一 request_id 不重复 |
| trace 详情 | 成功/失败链路终态正确 |
| 计费 | probe/intercepted/error_visible 不产生正常消费 |
| 用户 token | 普通用户 token 可以完成至少一个真实请求或可解释失败 |
| 旧日志 | 老日志正常显示，不强行 SFT 化 |
| 前端抽查 | `/console/log` 非空、展开字段不乱 |
| 剩余风险 | 所有 P1/P2 记录清楚 |

Stage A 失败条件：

1. 日志接口 500 或无法登录。
2. `logs.type` 出现新编号。
3. 成功链路最终不是 consume。
4. 失败链路没有 error_visible 收尾。
5. grouped 默认页被 standalone probe 刷空。
6. 同一 request_id 在 traces 中重复摘要。
7. probe/intercepted error 影响用户余额或正常消费统计。
8. 本地连接到正式生产库且测试会写数据。

---

## 12. 证据记录模板

每个 P0/P1 场景记录：

```markdown
## <CASE-ID>

- 执行时间：
- 本地入口：http://localhost:5173
- 后端版本：
- Git commit：
- 数据库环境：
- 测试用户：
- token 名：
- 请求接口：
- 模型：
- Request ID：
- Log ID：
- trace_id：
- trace_role 序列：
- 用户余额前：
- 用户余额后：
- token 额度前：
- token 额度后：
- API 证据：
- DB/统计证据：
- 前端抽查：
- 结论：PASS / FAIL / BLOCKED
- 备注：
```

---

## 13. 本地通过后的下一步

本计划只覆盖本地 Stage A。

Stage A 通过后，才进入：

```text
Stage B：线上测试服务器测试
```

Stage B 需要基于既有文档：

- `caf-classic-plus-0-2-0-plan.md`
- `test-nacp-online-execution-plan.md`
- `caf-classic-plus-0-2-0-online-test-report.md`

Stage B 通过后，才允许：

```text
Stage C：正式服务器部署后观察
```

---

## 14. 关联资料

- [NACP CAF 执行手册](./caf-change-assurance-framework-playbook.md)
- [classic-plus-0.1.x -> classic-plus-0.2.0 升级说明](./release-classic-plus-0-2-0-upgrade-review.md)
- [CAF classic-plus-0.2.0 深度测试计划](./caf-classic-plus-0-2-0-plan.md)
- [NACP 智能容错链路（SFT）](./sft-smart-failover-trace-analysis.md)
- [NACP 在线测试执行计划](./test-nacp-online-execution-plan.md)
- [NewAPI v0.13.2 回归测试计划](./test-newapi-v0-13-2-regression-plan.md)
