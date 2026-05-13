# NACP Changelog

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
