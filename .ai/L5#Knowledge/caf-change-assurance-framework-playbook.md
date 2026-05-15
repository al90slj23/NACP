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
8. 执行本地/线上验证
9. 留存证据
10. 回流知识
```

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

