# NACP 智能容错链路（Smart Failover Trace, SFT）

**日期**：2026-05-15

---

## 一、命名

本次围绕 NewAPI 原生 relay retry 机制做的增强，统一命名为：

```text
NACP 智能容错链路
英文简称：SFT
英文全称：Smart Failover Trace
```

这个名字刻意不叫“重试日志”或“高级重试”，原因是它不是单点重试功能，而是一套完整链路：

```text
失败拦截 -> 同渠道重试 -> 候选渠道预热探测 -> 健康候选接管 -> 最终成功/失败收尾 -> 结构化日志还原
```

核心目标：

1. 用户侧尽量只看到最终结果，不暴露中间失败。
2. 管理员侧能看清每一步实际发生了什么。
3. 计费、统计、旧日志、旧筛选不能被新语义污染。
4. 新日志仍保持“一行就是一条真实日志”，链路关系由结构化字段还原。

---

## 二、相对 NewAPI 原版的最终调整总览

| 模块 | NewAPI 原版 | NACP SFT 最终调整 |
|------|-------------|-------------------|
| 重试方式 | 失败后按 `retry` 串行重选渠道并完整重发 | 保留原串行框架，同时增加同渠道重试、候选渠道预热探测、已尝试渠道排除 |
| 默认可见性 | 中间错误和最终错误没有明确角色区分 | 区分“已拦截错误”“最终客户端可见错误”“探测成功”“探测失败”“消费成功” |
| `logs.type` | 使用旧类型，例如 1/2/3/4/5/6 | 继续只存旧类型，兼容旧统计、旧筛选、旧日志 |
| 新语义编号 | 不存在 20/21/29/50/51/52/59 | 作为输入语义/展示语义存在，但落库前归一化到旧类型 |
| 链路关系 | 主要靠 `request_id` 和 `other.admin_info.use_channel` 弱关联 | 增加 `trace_id / trace_seq / trace_parent_id / trace_sibling_seq / trace_role` |
| 展开详情 | 普通日志详情，无严格步骤顺序 | `/api/log/trace` 按 `trace_seq` 返回步骤，前端按表格列对齐展示 |
| 分组日志 | 曾尝试做 20/50 聚合摘要 | 最终取消合成摘要，`/api/log/grouped` 返回扁平真实日志 |
| 旧日志兼容 | 原样显示 | 原样显示，不强行推断旧日志为 SFT 链路 |

---

## 三、最终日志模型

### 3.1 保持 `logs.type` 兼容

SFT 最关键的决定：

```text
logs.type 不存 20/21/29/50/51/52/59。
```

原因：

1. NewAPI 原版以及线上真实运营站已有日志只理解旧类型。
2. 计费统计、消费统计、错误统计大量依赖 `type=2` 和 `type=5`。
3. 如果把 `51/52/21/29/59` 当真实类型落库，会污染旧报表和旧筛选。
4. 老日志没有 SFT 机制，不能强行推断成 SFT 链路。

最终归一化规则在 `model/log.go`：

| 输入语义 | 含义 | 真实落库 `logs.type` | `trace_role` |
|----------|------|----------------------|--------------|
| `2` | 正常消费成功 | `2` | `consume` |
| `20` | 容错重试后成功的展示语义 | `2` | 由具体真实行决定 |
| `21` | 容错链路内最终消费成功 | `2` | `consume` |
| `50` | 容错重试后失败的展示语义 | `5` | 由具体真实行决定 |
| `51` | 中间错误已拦截，未返回用户 | `5` | `error_intercepted` |
| `52` | 最终错误客户端可见 | `5` | `error_visible` |
| `29` | 轻量探测成功 | `4` | `probe_success` |
| `59` | 轻量探测失败 | `5` | `probe_failed` |

注意：

```text
20/50 更适合作为前端或查询层的链路摘要语义。
21/51/52/29/59 更适合作为 trace_role 对应的步骤语义。
```

### 3.2 新增结构化链路字段

`model.Log` 新增字段：

```text
trace_id
trace_seq
trace_parent_id
trace_sibling_seq
trace_role
```

含义：

| 字段 | 当前作用 |
|------|----------|
| `trace_id` | 链路 ID。默认使用 `request_id`，让同一次请求内的日志可聚合。 |
| `trace_seq` | 链路内顺序。写日志时自动递增，详情页优先按它排序。 |
| `trace_role` | 日志在 SFT 链路中的角色，例如 `consume`、`error_intercepted`、`error_visible`、`probe_success`、`probe_failed`。 |
| `trace_parent_id` | 预留父日志 ID，用于未来严格树形父子关系。当前大多数记录为 0。 |
| `trace_sibling_seq` | 预留同级顺序，用于未来表达并发探测/同级分支顺序。当前大多数记录为 0。 |

当前已经能严谨表达：

```text
属于哪条链路：trace_id
链路内先后顺序：trace_seq
每一步是什么角色：trace_role
每一条日志唯一身份：id
```

未来如果要做严格树形：

```text
A1
├── A2
├── A3
├── B1-
└── C1-
```

可以继续填充：

```text
trace_parent_id = A1 的 log id
trace_sibling_seq = 同级分支顺序
```

---

## 四、SFT 后端逻辑

### 4.1 主流程

当前主入口仍是 `controller/relay.go` 的 `Relay`。

NewAPI 原版大致是：

```text
选渠道 A -> 发真实请求 A1
失败 -> 判断 shouldRetry
下一轮 retry -> 重新选渠道 B -> 发真实请求 B1
```

SFT 增强后变为：

```text
选渠道 A -> 发真实请求 A1
成功 -> 记录正常消费日志 type=2
失败 -> 记录 error_intercepted
     -> 判断是否可重试
     -> 启动候选渠道预热探测 B1-/C1-
     -> 对 A 做同渠道真实重试 A2/A3
     -> A 重试成功 -> 用户成功返回
     -> A 仍失败 -> 等待预热探测结果
     -> 选择探测成功的候选渠道 B/C 继续真实请求
     -> 最终成功 -> 用户成功返回
     -> 全部失败 -> 记录 error_visible，用户收到最终错误
```

### 4.2 同渠道重试

相关代码：

```text
controller/relay.go
- continueRetryTrace
- recordSameChannelRetryIntercepted
```

逻辑：

1. A1 失败后，先不马上把错误返回用户。
2. 如果错误类型允许重试，则进入 `continueRetryTrace`。
3. 对当前渠道执行 `SameChannelRetryCount` 次真实重试。
4. 每次同渠道重试失败，如果仍允许重试，就写一条 `error_intercepted` 角色日志。
5. 同渠道任意一次成功，整个请求成功返回，不再继续换渠道。

### 4.3 候选渠道预热探测

相关代码：

```text
controller/relay.go
- selectPreWarmChannels
- startPreWarmChannels
- waitPreWarmBatch
- orderedHealthyPreWarmChannels

service/channel_probe.go
- ProbeNextChannels
- recordProbeLog
```

逻辑：

1. 当前渠道 A 失败后，选择候选 B/C。
2. 候选选择会排除本次真实请求已经尝试过的渠道。
3. 预热探测异步并发执行。
4. 探测请求是轻量健康检查，不计入用户计费。
5. 探测日志使用：

```text
成功：type=4, trace_role=probe_success
失败：type=5, trace_role=probe_failed
user_id=0
quota=0
request_id=触发它的用户请求 request_id
```

### 4.4 候选渠道接管

当同渠道重试仍失败后：

1. 等待预热探测结果。
2. 只选择探测成功的候选渠道。
3. 按候选顺序发真实请求。
4. 候选真实请求失败后，它会成为新的“当前失败渠道”，递归进入同样的 SFT 流程。
5. 这个设计可以表达：

```text
A1 失败
A2/A3 失败
B1- 探测成功
C1- 探测成功
B2 真实请求失败
B3/B4 同渠道重试
C2- 新一轮探测
```

### 4.5 最终客户端可见错误

相关代码：

```text
controller/relay.go
- recordClientVisibleErrorLog
```

当全部可尝试路径都失败，并且错误最终要返回用户时，写最终收尾日志：

```text
输入语义：52
落库 type：5
trace_role：error_visible
```

这条日志的意义：

```text
这才是用户真正看到的失败。
```

中间的 `error_intercepted` 只说明系统内部已经拦截并继续尝试，不代表用户已经失败。

---

## 五、日志 API 调整

### 5.1 `/api/log/grouped`

相关代码：

```text
service/log_grouped.go
```

最终调整：

1. 保留接口路径和响应结构，避免前端/调用方断裂。
2. 不再生成假的 20/50 summary 行。
3. 返回真实扁平日志行。
4. `is_summary=false`。
5. 带上 `trace_*` 字段。

这样做的原因：

```text
20/50 聚合摘要容易造成“摘要行和真实行并行显示”“展开重复”“字段不完整”“52 收尾缺失”等误解。
```

最终原则是：

```text
列表里每一行都是数据库里真实存在的一条日志。
链路关系通过 trace 字段和详情接口还原。
```

### 5.2 `/api/log/trace`

相关代码：

```text
service/trace.go
```

作用：

1. 根据 `request_id` 查询完整链路步骤。
2. 返回每一步的完整字段：

```text
时间、渠道、用户、令牌、分组、类型、模型、用时/首字、输入、输出、花费、IP、重试、详情
```

3. 同时返回：

```text
id
trace_id
trace_seq
trace_parent_id
trace_sibling_seq
trace_role
```

4. 排序规则：

```text
trace_seq ASC -> created_at ASC -> id ASC
```

这解决了单纯按时间排序在并发探测下可能失真的问题。

---

## 六、日志页面调整

### 6.1 类型筛选

相关代码：

```text
web/src/components/table/usage-logs/UsageLogsFilters.jsx
```

最终只保留 NewAPI 原版类型：

```text
全部
1 充值
2 正常消费成功
3 管理
4 系统
5 错误
6 退款
```

没有把 `51/52/21/29/59/20/50` 放进下拉框。

原因：

1. `logs.type` 不真实存这些值。
2. 放进去会让用户以为能直接用 `type=51` 查库。
3. 下拉会变长且语义混乱。
4. SFT 语义更适合后续做单独的“链路角色筛选”，例如按 `trace_role=error_visible` 搜索。

### 6.2 展开详情

相关代码：

```text
web/src/components/table/usage-logs/TraceExpandRender.jsx
```

最终展示方式：

1. 展开时调用 `/api/log/trace?request_id=...`。
2. 展开区使用和主列表对应的列：

```text
层级符号
时间
渠道
用户
令牌
分组
类型
模型
用时/首字
输入
输出
花费
IP
重试
详情
```

3. 每一条展开步骤都显示完整字段，不再只显示简化文本。
4. 在时间列前留出层级符号位置。
5. 每条日志都有 Log ID 标识：

```text
鼠标悬停：查看 Log ID / Trace ID / Trace Seq / Trace Role / Parent Log ID / Sibling Seq / Request ID
点击：复制 Log ID
```

### 6.3 类型展示

当前展开详情里的类型展示仍基于真实落库类型：

```text
2：正常消费成功
4：系统日志
5：普通错误
```

真正的 SFT 语义保存在 `trace_role`：

```text
error_intercepted
error_visible
probe_success
probe_failed
consume
```

后续如果要在 UI 上显示为：

```text
51：容错重试已拦截
52：容错重试客户端可见
29：容错探测成功
59：容错探测失败
21：容错重试成功
```

推荐从 `trace_role` 映射展示，而不是读取 `logs.type`。

---

## 七、为什么不再做 20/50 合成摘要行

之前设想：

```text
20 = 容错重试后成功
50 = 容错重试后失败
展开后显示 51/52/21/29/59
```

最终放弃把它做成真实或合成列表行，原因：

1. 20/50 不是数据库里的真实日志。
2. 合成行会和真实行并行出现，用户看到“重复”。
3. 20/50 摘要行需要补齐所有字段，容易和真实计费字段不一致。
4. 50 下面如果没有 52，说明收尾逻辑或查询逻辑有问题；合成摘要会掩盖问题。
5. 20 下面出现 52，也说明链路归属或收尾逻辑有问题；合成摘要同样会制造误导。
6. NewAPI 老日志没有这些机制，强行合成会污染历史数据理解。

最终采用：

```text
真实日志扁平展示
+ trace 字段结构化还原
+ 详情页按 trace_seq 展开
+ 后续可加 trace_role 语义筛选
```

---

## 八、后续测试重点

SFT 的测试必须围绕“最终用户结果”和“管理员链路可解释性”同时验证。

### 8.1 成功路径

| 场景 | 预期 |
|------|------|
| A1 直接成功 | 只有正常消费日志，`type=2`，不需要复杂展开 |
| A1 失败，A2 成功 | 用户成功；中间失败为 `type=5 trace_role=error_intercepted`；成功消费为 `type=2 trace_role=consume` |
| A1 失败，A2/A3 失败，B 探测成功，B2 成功 | 用户成功；探测日志可见；B2 消费日志计费完整 |

### 8.2 失败路径

| 场景 | 预期 |
|------|------|
| A 全部失败，无候选 | 最终有 `trace_role=error_visible` |
| A 失败，B/C 探测都失败 | 最终有 `error_visible`，探测失败为 `probe_failed` |
| B 探测成功但真实 B2 失败，后续无可用 | B2 错误先是 `error_intercepted`，最终必须有 `error_visible` |

### 8.3 日志一致性

必须检查：

1. `logs.type` 不出现 `20/21/29/50/51/52/59`。
2. 所有 SFT 语义都能通过 `trace_role` 找到。
3. 同一请求的 `trace_id` 一致。
4. `trace_seq` 单调递增。
5. 展开详情每一行字段完整。
6. 消费计费只来自真实消费日志，不来自探测日志。
7. 探测日志 `user_id=0`、`quota=0`。
8. 老日志仍按 NewAPI 原版类型正常显示。

---

## 九、当前实现边界

当前已经完成：

1. `logs.type` 兼容旧 NewAPI。
2. SFT 角色写入 `trace_role`。
3. 自动填充 `trace_id` 和 `trace_seq`。
4. 探测日志不计费。
5. 最终客户端可见错误有独立记录路径。
6. `/api/log/grouped` 回归扁平真实日志。
7. `/api/log/trace` 支持完整链路展开。
8. 前端展开区显示 Log ID，并支持复制。

当前仍属于增强空间：

1. `trace_parent_id` 和 `trace_sibling_seq` 已有字段，但当前主要是预留，尚未完整表达树形父子和并发同级顺序。
2. 前端类型列目前展示真实旧类型；如果要显示 `51/52/29/59/21`，应基于 `trace_role` 做展示映射。
3. 可以增加独立的 SFT 筛选器，例如“最终失败”“中间拦截”“探测失败”“探测成功”，查询 `trace_role` 而不是 `logs.type`。
4. 并发探测结果慢于同渠道重试时，当前流程会在同渠道失败后等待预热批次；后续可以继续优化为更细的竞态决策策略。
