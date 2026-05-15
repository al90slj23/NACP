# 2026-05-13 — v0.1.0 智能重试与渠道健康管理

## 操作内容

实现 NACP 首个功能版本：智能重试与渠道健康管理系统。

基于生产数据分析（4400+ 条日志，8.8% 错误率，100% 流式错误发生在窗口 1 可重试）设计。

## 核心设计决策

1. **不做开关操作** — 用五状态健康状态机替代简单开/关
2. **同渠道重试 + 并行预热** — 错误后重试同渠道 2 次，同时并行探测顺位渠道
3. **新增优先，慎改源码** — 核心逻辑在 3 个新文件中，现有文件最小改动
4. **安全兜底** — 所有渠道被过滤时回退到不过滤
5. **亲和性协同** — Degraded 渠道跳过亲和绑定
6. **400 不改健康状态** — 参数问题不是渠道问题
7. **向后兼容** — 默认所有渠道 healthy，行为与原版一致

## 文件变更

### 新增
- `service/channel_health.go` — 健康状态机（~280 行）
- `service/channel_health_config.go` — 配置/阈值（~70 行）
- `service/channel_probe.go` — 轻量探测 + 并行预热（~230 行）
- `NACP.md` — 项目 README
- `CHANGELOG.md` — 版本更新记录

### 修改
- `common/constants.go` — +10 行（健康状态常量 + SameChannelRetryCount）
- `model/channel.go` — +5 行（Channel struct 新增 4 个健康字段）
- `model/channel_cache.go` — +30 行（GetRandomSatisfiedChannel 健康过滤 + 排除 + 兜底）
- `controller/relay.go` — +80 行（增强重试循环 + 并行预热 + processChannelError 集成状态机）
- `middleware/distributor.go` — +4 行（亲和性健康感知）
- `main.go` — +3 行（启动时初始化健康管理）
- `.gitignore` — +2 行（ZERO/ + .analysis/）

### 数据库
- channels 表新增 4 个字段（GORM AutoMigrate 自动添加，三库兼容）

## 影响分析

- `GetRandomSatisfiedChannel` 签名改为 variadic（向后兼容，现有调用方无需修改）
- `processChannelError` 新增一行状态机调用（不影响现有禁用逻辑）
- 健康状态存在独立 map 中，不受 `InitChannelCache` 周期性同步影响
- 计费逻辑不受影响（预扣/退款在循环外，探测不走 billing）

---

**操作者**：AI + 用户
