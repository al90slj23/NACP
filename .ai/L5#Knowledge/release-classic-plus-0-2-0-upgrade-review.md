# classic-plus-0.1.x -> classic-plus-0.2.0 升级说明

> **状态**：开发中
> **当前版本**：classic-plus-0.2.0-dev
> **基准版本**：classic-plus-0.1.x
> **创建日期**：2026-05-15
> **用途**：作为 `CHANGELOG.md`、GitHub Release、CAF 测试计划的详版依据。

---

## 一、版本定位

`classic-plus-0.2.0` 的重点不是继续扩大 relay retry 本身，而是把 v0.1.x 中已经出现的容错、探测、日志问题整理成可观测、可兼容、可测试的体系。

一句话：

```text
0.1.x 做智能重试和渠道健康；0.2.0 把容错链路、日志兼容、版本治理、CAF 测试体系补齐。
```

---

## 二、从 0.1 到 0.2 的核心变化

| 领域 | 0.1.x | 0.2.0-dev |
|------|-------|-----------|
| 容错重试 | 智能重试、同渠道重试、渠道健康、预热探测 | 正式命名为 SFT，并补齐结构化链路字段和日志兼容策略 |
| 日志类型 | 早期设计曾尝试 `51/52/29/59` 作为真实语义类型 | 最终 `logs.type` 保持 NewAPI 老类型，新语义进入 `trace_role` |
| 日志展示 | 普通日志页 + 初版分组想法 | 扁平真实日志 + `/api/log/trace` 展开链路 |
| 兼容策略 | 重点解决运行时问题 | 明确旧日志不强行推断，新日志通过 `trace_*` 还原 |
| 测试体系 | 有测试计划和线上测试记录 | 正式建立 CAF 变更验证与上线保障体系 |
| 知识体系 | `.ai` 已存在，但命名不统一 | 按 ZERO 六层规范收敛命名，并新增命名规范 |
| 版本发布 | 有 `CHANGELOG.md`，但 `VERSION` 为空 | `VERSION`、`CHANGELOG.md`、Git tag、GitHub Release 分工固定 |

---

## 三、SFT 日志兼容调整

### 3.1 最终原则

```text
logs.type 不存 20/21/29/50/51/52/59。
```

原因：

1. NewAPI 原版统计、计费、筛选依赖旧类型。
2. 线上真实运营站只有旧类型日志。
3. 老日志没有 NACP 容错链路，不能强行推断。
4. 新语义应该结构化表达，而不是污染 `type`。

### 3.2 语义映射

| 输入语义 | 最终落库 `type` | `trace_role` |
|----------|-----------------|--------------|
| 2 | 2 | `consume` |
| 21 | 2 | `consume` |
| 51 | 5 | `error_intercepted` |
| 52 | 5 | `error_visible` |
| 29 | 4 | `probe_success` |
| 59 | 5 | `probe_failed` |
| 20 | 2 | 展示/筛选语义，不作为真实落库值 |
| 50 | 5 | 展示/筛选语义，不作为真实落库值 |

---

## 四、数据结构变化

`logs` 表新增结构化链路字段：

```text
trace_id
trace_seq
trace_parent_id
trace_sibling_seq
trace_role
```

用途：

1. `trace_id`：链路归属。
2. `trace_seq`：链路内顺序。
3. `trace_role`：步骤角色。
4. `trace_parent_id`：未来树形父子关系。
5. `trace_sibling_seq`：未来并发同级顺序。

---

## 五、API 与前端变化

### 5.1 API

| API | 作用 |
|-----|------|
| `/api/log/grouped` | 保持接口兼容，但返回扁平真实日志，不再合成 20/50 摘要行 |
| `/api/log/trace` | 根据 `request_id` 返回完整链路步骤 |

### 5.2 前端日志页

变化：

1. 日志类型筛选保留 NewAPI 老类型。
2. 展开区按完整列展示链路步骤。
3. 每行显示 Log ID 标识。
4. tooltip 展示 Trace ID、Trace Seq、Trace Role、Parent Log ID、Sibling Seq、Request ID。
5. 点击 Log ID 可复制。

---

## 六、CAF 变更验证体系

0.2.0 新增 CAF：

```text
系统理解 -> 影响分析 -> 测试计划 -> 执行验证 -> 证据留存 -> 上线判断 -> 知识回流
```

固定文档：

| 文档 | 用途 |
|------|------|
| `06.quality-02.change-assurance-framework.md` | L3 标准 |
| `caf-change-assurance-framework-playbook.md` | L5 执行手册 |

---

## 七、版本治理变化

0.2.0 固化版本记录位置：

| 位置 | 用途 |
|------|------|
| `VERSION` | 构建版本源 |
| `CHANGELOG.md` | 仓库版本摘要 |
| `.ai/L4#Changelog/` | 内部操作审计 |
| `.ai/L5#Knowledge/release-*-upgrade-review.md` | 版本跃迁详版 |
| Git tag | 锁定正式版本代码点 |
| GitHub Release | 正式发布记录 |

---

## 八、测试要求

0.2.0 正式发布前至少要完成：

1. `logs.type` 不出现 `20/21/29/50/51/52/59`。
2. SFT 语义能通过 `trace_role` 找到。
3. 旧日志能正常显示，不被推断成容错链路。
4. 真实线上测试站能完成普通用户 token 调用。
5. 管理员日志能看到对应链路和最终计费。
6. `/api/log/grouped` 扁平列表不重复、不合成假行。
7. `/api/log/trace` 展开字段完整、顺序正确。
8. 计费、计量、统计不被探测日志污染。
9. Docker 构建读取 `VERSION`。
10. GitHub Release 说明可由本文档和 `CHANGELOG.md` 生成。

---

## 九、内部引用

- `CHANGELOG.md`
- `.ai/L3#Standards/standards/06.quality-02.change-assurance-framework.md`
- `.ai/L3#Standards/standards/06.quality-03.version-release-management.md`
- `.ai/L5#Knowledge/sft-smart-failover-trace-analysis.md`
- `.ai/L5#Knowledge/newapi-relay-retry-mechanism-analysis.md`
- `.ai/L5#Knowledge/test-nacp-online-execution-plan.md`
- `.ai/L5#Knowledge/test-newapi-v0-13-2-regression-plan.md`

