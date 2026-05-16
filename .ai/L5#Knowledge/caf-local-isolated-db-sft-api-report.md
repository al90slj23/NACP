# CAF 本地隔离数据库 SFT API 测试报告

> 日期：2026-05-16
> 阶段：CAF Stage A，本地隔离数据库
> 后端：临时本地后端 `http://127.0.0.1:3001`
> 数据库：Docker MySQL `nacp-mysql-dev`，`127.0.0.1:3307/nacp_dev`
> 上游：本地 OpenAI-compatible mock upstream `http://127.0.0.1:18080`

## 一、环境调整

本地 `.env` 已切换为本地 MySQL：

```text
SQL_DSN=nacp_dev:nacp_dev_pass@tcp(127.0.0.1:3307)/nacp_dev?charset=utf8mb4&parseTime=True&loc=Local
SKIP_DB_MIGRATION=false
```

当前用户已打开的 `http://localhost:5173/` 仍代理到运行中的 `http://localhost:3000`。该进程需要重启后才会读取新的 `.env`。本轮为了不中断用户已打开的前后端，使用已迁移完成的 `3001` 本地后端执行 API 测试。

## 二、测试夹具

通过 API 创建：

| 类型 | 结果 |
|------|------|
| 本地 mock 上游 | `/fail/v1` 固定返回 500；`/success/v1` 固定返回 200 + usage |
| 管理员登录 | 成功 |
| 普通用户 | `nlsft193226` |
| 普通用户 token | `tok-sft-193226`，不记录真实 key |
| 本地测试渠道 | OpenAI 类型，模型 `gpt-3.5-turbo`，分组 `default` |
| SFT option | `RetryTimes=2` |

## 三、关键结论

1. 本地隔离数据库可以完成迁移，`/api/log/grouped` 和 `/api/log/trace` 可正常查询。
2. `RetryTimes=0` 时，增强重试不会进入跨优先级重试链；本地 SFT 测试必须先设置 `RetryTimes >= 1`，建议为 `2`。
3. 直接成功链路生成 NewAPI 原生 `type=2`，不展开，符合设计。
4. 容错成功链路生成真实 summary `type=20`，展开步骤含 `51/59/29/21`，最终以 `21` 成功消费收尾。
5. 容错失败链路生成真实 summary `type=50`，展开步骤含 `51/59/52`，最终以 `52` 客户端可见错误收尾。
6. 默认 grouped 列表隐藏子步骤；显式 type 筛选可以查到 `21/29/51/52/59`。
7. 成功 probe `29` 能记录 usage/quota，不再只能显示 0 token、0 费用。

## 四、证据摘要

### 4.1 直接成功

| 字段 | 值 |
|------|----|
| grouped log id | `2` |
| type | `2` |
| request_id | `20260516113350975642000JRQnyRXYzNplYjje` |
| channel | `LOCAL-SFT-success-20260516192955` |
| prompt/completion/quota | `12 / 4 / 5` |

### 4.2 容错成功

| 字段 | 值 |
|------|----|
| summary log id | `12` |
| summary type | `20` |
| request_id / trace_id | `20260516114043803717000JRQnyRXYwnJE2ycN` |
| terminal channel | `LOCAL-SFT3-success-193907` |
| prompt/completion/quota | `12 / 4 / 5` |

展开步骤：

| log id | type | trace_role | trace_seq | channel | prompt | completion | quota |
|--------|------|------------|-----------|---------|--------|------------|-------|
| 6 | 51 | error_intercepted | 1 | LOCAL-SFT3-fail-193907 | 0 | 0 | 0 |
| 7 | 51 | error_intercepted | 2 | LOCAL-SFT3-fail-193907 | 0 | 0 | 0 |
| 8 | 59 | probe_failed | 3 | LOCAL-SFT2-fail-193600 | 0 | 0 | 0 |
| 10 | 29 | probe_success | 4 | LOCAL-SFT3-success-193907 | 12 | 4 | 5 |
| 9 | 51 | error_intercepted | 5 | LOCAL-SFT3-fail-193907 | 0 | 0 | 0 |
| 11 | 21 | consume | 6 | LOCAL-SFT3-success-193907 | 12 | 4 | 5 |

说明：数据库 `id` 与异步 probe 到达时间有关，不作为链路顺序依据；展开顺序以 `trace_seq` 为准。

### 4.3 容错失败

| 字段 | 值 |
|------|----|
| summary log id | `25` |
| summary type | `50` |
| request_id / trace_id | `20260516114203657536000JRQnyRXYbdET6OOA` |
| terminal channel | `LOCAL-SFT4-fail2-194136` |
| prompt/completion/quota | `0 / 0 / 0` |

展开步骤最后一行：

| log id | type | trace_role | trace_seq | channel | content |
|--------|------|------------|-----------|---------|---------|
| 24 | 52 | error_visible | 11 | LOCAL-SFT4-fail2-194136 | `status_code=500, mock upstream 500` |

## 五、筛选验证

| grouped type filter | total |
|---------------------|-------|
| 20 | 1 |
| 21 | 1 |
| 29 | 1 |
| 50 | 2 |
| 51 | 11 |
| 52 | 2 |
| 59 | 4 |

默认 `/api/log/grouped?p=1&size=10`：

| 检查项 | 结果 |
|--------|------|
| total | 5 |
| page 内 20/50 summary 行 | 3 |
| page 内 21/29/51/52/59 子步骤行 | 0 |

## 六、后续注意

1. `http://localhost:5173/` 要真正使用本地库，需要重启当前本地后端 `3000`，使其读取新的 `.env`。
2. Stage B 线上测试站 `nacp.m.srl` 不再与本地 Stage A 共用数据库。
3. 每次 SFT 测试开始前必须通过 API 检查或设置 `RetryTimes`，否则只能验证直接成功和直接失败，不能验证容错重试。

## 七、2026-05-16 本地空库复测记录

> 前端：用户已打开的 `http://localhost:5173/`
> 后端：本地 `http://localhost:3000`
> 数据库：Docker MySQL `nacp-mysql-dev`，`127.0.0.1:3307/nacp_dev`
> 测试前缀：`cafsft195501`

### 7.1 环境校验

本轮开始时发现 `3000` 后端仍是旧进程，启动环境里 `SKIP_DB_MIGRATION=true`，导致 `/api/log/grouped` 查询新字段时报 `Unknown column 'summary_log_id'`。重启后端并读取当前 `.env` 后，迁移正常执行，`/api/log/grouped`、`/api/log/trace` 恢复可用。

本地 mock upstream 使用 OpenAI-compatible 协议：

| 路径 | 行为 |
|------|------|
| `/success/v1/...` | 返回 `200`，带 usage：`prompt_tokens=12`、`completion_tokens=4`、`total_tokens=16` |
| `/fail/v1/...` | 返回 `500`，OpenAI-like error：`mock upstream 500` |

### 7.2 API 夹具创建

| 类型 | 结果 |
|------|------|
| 普通用户 | `ucafsft195501` |
| 用户分组 | `default` |
| token | `cafsft195501-dtok-direct`、`cafsft195501-dtok-success`、`cafsft195501-dtok-fail` |
| 模型 | `cafsft195501-model-direct`、`cafsft195501-model-success`、`cafsft195501-model-fail` |
| 选项 | `RetryTimes=2` |
| 模型倍率 | 三个测试模型均加入 `ModelRatio=0.25` |

注意：先尝试使用临时自定义分组时，普通用户 token 访问返回 `403 无权访问 ... 分组`。这证明 SFT 测试夹具应优先使用 `default`，或在测试前显式配置用户可访问分组。

### 7.3 API 结果

#### 直接成功

| 字段 | 值 |
|------|----|
| summary/terminal log id | `31` |
| type | `2` |
| request_id / trace_id | `20260516120103408083000NWMoPSmYKqMke7k5` |
| token | `cafsft195501-dtok-direct` |
| channel | `16 - cafsft195501-dmodel-direct-success` |
| model | `cafsft195501-model-direct` |
| prompt/completion/quota | `12 / 4 / 4` |

直接成功只有 `2`，不需要折叠链路。

#### 容错成功

| 字段 | 值 |
|------|----|
| summary log id | `36` |
| summary type | `20` |
| request_id / trace_id | `20260516120103494489000NWMoPSmYrlM96rDV` |
| terminal log id | `35` |
| terminal type | `21` |
| token | `cafsft195501-dtok-success` |
| terminal channel | `18 - cafsft195501-smodel-success-b` |
| model | `cafsft195501-model-success` |
| prompt/completion/quota | `12 / 4 / 4` |

展开步骤：

| log id | type | trace_role | trace_seq | channel | prompt | completion | quota |
|--------|------|------------|-----------|---------|--------|------------|-------|
| 30 | 51 | error_intercepted | 1 | 17 - cafsft195501-smodel-fail-a | 0 | 0 | 0 |
| 32 | 51 | error_intercepted | 2 | 17 - cafsft195501-smodel-fail-a | 0 | 0 | 0 |
| 33 | 29 | probe_success | 3 | 18 - cafsft195501-smodel-success-b | 12 | 4 | 4 |
| 34 | 51 | error_intercepted | 4 | 17 - cafsft195501-smodel-fail-a | 0 | 0 | 0 |
| 35 | 21 | consume | 5 | 18 - cafsft195501-smodel-success-b | 12 | 4 | 4 |

结论：`20` 必须以 `21` 作为成功终端；`29` 成功探测能记录非零 usage/quota，展示为平台运营消耗，不计入用户账单。

#### 容错失败

| 字段 | 值 |
|------|----|
| summary log id | `48` |
| summary type | `50` |
| request_id / trace_id | `20260516120103556076000NWMoPSmYH0wksG85` |
| terminal log id | `47` |
| terminal type | `52` |
| token | `cafsft195501-dtok-fail` |
| terminal channel | `21 - cafsft195501-fmodel-fail-c` |
| model | `cafsft195501-model-fail` |
| prompt/completion/quota | `0 / 0 / 0` |
| terminal error | `status_code=500, mock upstream 500` |

展开步骤：

| log id | type | trace_role | trace_seq | channel |
|--------|------|------------|-----------|---------|
| 37 | 51 | error_intercepted | 1 | 19 - cafsft195501-fmodel-fail-a |
| 39 | 59 | probe_failed | 2 | 20 - cafsft195501-fmodel-fail-b |
| 38 | 51 | error_intercepted | 3 | 19 - cafsft195501-fmodel-fail-a |
| 40 | 59 | probe_failed | 4 | 21 - cafsft195501-fmodel-fail-c |
| 41 | 51 | error_intercepted | 5 | 19 - cafsft195501-fmodel-fail-a |
| 42 | 51 | error_intercepted | 6 | 20 - cafsft195501-fmodel-fail-b |
| 43 | 59 | probe_failed | 7 | 21 - cafsft195501-fmodel-fail-c |
| 44 | 51 | error_intercepted | 8 | 20 - cafsft195501-fmodel-fail-b |
| 45 | 51 | error_intercepted | 9 | 20 - cafsft195501-fmodel-fail-b |
| 46 | 51 | error_intercepted | 10 | 21 - cafsft195501-fmodel-fail-c |
| 47 | 52 | error_visible | 11 | 21 - cafsft195501-fmodel-fail-c |

结论：`50` 必须以 `52` 作为失败终端；本轮 API 数据符合该约束。

### 7.4 grouped 列表和筛选

`/api/log/grouped?p=1&page_size=20&username=ucafsft195501` 默认列表：

| log id | type | 含义 |
|--------|------|------|
| 48 | 50 | 容错重试后失败 summary |
| 36 | 20 | 容错重试后成功 summary |
| 31 | 2 | 正常消费成功 |
| 26 | 3 | 管理操作 |

默认列表不直接混入 `21/29/51/52/59` 子步骤。按 `request_id` 或显式 type 可以查到子步骤。

按 username 筛选时，`29/59` 可能为 `user_id=0` 的平台探测日志，因此不会出现在 username 筛选结果里；应使用 `request_id` 查询探测步骤：

| 查询 | 结果 |
|------|------|
| `type=29 + success request_id` | 1 条，含 `prompt=12/completion=4/quota=4` |
| `type=59 + fail request_id` | 3 条 |

### 7.5 日志页面显示验证

使用 Browser 对 `http://localhost:5173/console/log` 页面验证：

| 检查项 | 结果 |
|--------|------|
| 50 summary 行显示 token/model/channel path | 通过：`cafsft195501-dtok-fail`、`cafsft195501-model-fail`、`19 → 20 → 21` |
| 20 summary 行显示 token/model/channel path | 通过：`cafsft195501-dtok-success`、`cafsft195501-model-success`、`17 → 18` |
| 2 正常消费行 | 通过：`cafsft195501-dtok-direct`、`cafsft195501-model-direct` |
| 摘要行输入/输出/费用 | 通过：20 显示 `12/4/$0.000008`，50 显示 `0/0/$0.000000` |
| 展开步骤 | 通过：可见 `51/52/59/29/21` |
| 展开详情 | 通过：可见日志类型、消耗 Token、产生费用、时间记录、请求转换、计费模式 |
| 唯一标识 | 通过：展开行首显示 `#log_id`，可作为唯一 log id 标识 |

本轮发现并修复一个前端显示问题：后端 `20/50` summary 行数据完整，但前端列规则仍只把旧 `2/5` 当作请求类日志，导致 20/50 的令牌、模型、用时、输入/输出、花费列为空。已调整为 `20/21/29/50/51/52/59` 均按请求链路日志显示字段，并让 20/50 渠道路径从 `other.summary.channel_path` 渲染。

### 7.6 验证命令

| 命令 | 结果 |
|------|------|
| `bun run build` | 通过 |
| API 创建用户/token/channel/模型倍率 | 通过 |
| 直接成功 relay | 200 |
| 容错成功 relay | 200 |
| 容错失败 relay | 500 |
| `/api/log/grouped` | 通过 |
| `/api/log/trace` | 通过 |
| Browser 页面验证 | 通过 |
