# NACP Changelog

## classic-plus-0.2.0-dev (Unreleased)

### Summary

`classic-plus-0.2.0-dev` 从 `classic-plus-0.1.x` 的智能重试基础上，补齐 SFT 结构化日志、旧类型兼容、CAF 测试体系、`.ai` 命名治理和版本发布治理。

### NACP 智能容错链路（SFT）

- 保持 `logs.type` 与 NewAPI 原版类型兼容，避免 `20/21/29/50/51/52/59` 污染旧统计、计费和筛选。
- 新增结构化链路字段：`trace_id`、`trace_seq`、`trace_parent_id`、`trace_sibling_seq`、`trace_role`。
- 将容错语义收敛到 `trace_role`：`consume`、`error_intercepted`、`error_visible`、`probe_success`、`probe_failed`。
- `/api/log/grouped` 回归扁平真实日志行，不再合成 20/50 summary 行。
- `/api/log/trace` 支持按请求查看完整链路步骤，并返回完整日志字段。
- 日志展开页显示 Log ID、Trace ID、Trace Seq、Trace Role，并支持点击复制 Log ID。

### NACP CAF 变更验证体系

- 新增 NACP Change Assurance Framework（CAF）标准，用于每次变更后的影响分析、测试计划、执行验证和证据留存。
- 新增 CAF 执行手册，沉淀影响树、测试表、系统域、上线结论模板。

### AI 记忆体系治理

- 按 ZERO 六层架构规范收敛 `.ai` 文件命名。
- 新增 `.ai` 命名规范、L3/L4/L5 文件迁移记录。

### 版本与发布

- 将当前开发目标统一为 `classic-plus-0.2.0-dev`。
- `VERSION` 作为当前构建版本源；正式发布时由 Git tag / GitHub Release 固化。

### Internal References

- `.ai/L5#Knowledge/release-classic-plus-0-2-0-upgrade-review.md`
- `.ai/L5#Knowledge/sft-smart-failover-trace-analysis.md`
- `.ai/L5#Knowledge/caf-change-assurance-framework-playbook.md`

---

## v0.1.0 (2026-05-13)

### 🎯 智能重试与渠道健康管理

基于生产数据分析（4400+ 条日志样本，8.8% 错误率，100% 流式错误可重试）设计的智能重试系统。

#### 新增功能

- **渠道健康状态机** — 五状态模型（Healthy/Probing/Degraded/Recovering/ManuallyDisabled）
  - 错误触发状态转换，不再简单开/关
  - 内存优先 + 异步 DB 持久化
  - 重启后自动恢复状态

- **同渠道重试** — 发现错误后重试同渠道 2 次（应对偶发性错误）
  - 400 错误直接切换渠道（参数问题，不是渠道问题）
  - 401 + 禁用关键词直接降级（永久性问题）
  - 504/524/408 不重试（超时类）

- **并行预热** — 重试同渠道的同时，并行探测顺位渠道
  - 最小请求体（max_tokens=1），3 秒超时
  - 同渠道重试失败后，直接切到已验证健康的顺位渠道（零等待）

- **同优先级渠道切换** — 同优先级内最多尝试 3 个渠道后才降级
  - 排除已试过的渠道 ID
  - 用完同优先级才降到低优先级

- **渠道亲和性健康感知** — Degraded 渠道自动跳过亲和绑定
  - 解决"开亲和后渠道故障持续报错"的痛点
  - SwitchOnSuccess 自动更新亲和到新渠道

- **低渠道告警** — 可用渠道 ≤ 2 时 warning，全部降级时 critical

- **安全兜底** — 所有渠道被健康过滤后回退到不过滤（防止状态机 bug 导致服务中断）

- **后台恢复探测** — 定时探测 Degraded 渠道（5 分钟间隔），连续 3 次成功进入恢复期

#### 数据库变更

channels 表新增字段（GORM AutoMigrate 自动添加）：
- `health_status` VARCHAR(16) DEFAULT 'healthy'
- `health_updated_at` BIGINT DEFAULT 0
- `health_fail_count` INT DEFAULT 0
- `health_success_count` INT DEFAULT 0

#### 文件变更

新增：
- `service/channel_health.go`
- `service/channel_health_config.go`
- `service/channel_probe.go`

修改：
- `common/constants.go`
- `model/channel.go`
- `model/channel_cache.go`
- `controller/relay.go`
- `middleware/distributor.go`
- `main.go`

---

## v0.0.0 (2026-05-13)

项目初始化，基于 QuantumNous/new-api v0.13.2 (commit bee339d)。
