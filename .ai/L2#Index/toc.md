# NACP L2#规范索引 (Index)

> **定位**：NACP 项目规范文档索引
> **版本**：v1.0.0
> **基线**：QuantumNous/new-api v0.13.2

---

## 📋 快速导航

NACP 是基于 NewAPI v0.13.2 classic 版本线的最终增强版，专注于解决上游质量问题和增强核心中继能力。

---

### 🗂️ 规范文档

#### 架构与核心

| 文档 | 说明 |
|------|------|
| `01.arch-01.smart-failover-trace.md` | NACP 智能容错链路（SFT）架构方案 |
| `01.arch-02.newapi-v0-13-2-baseline.md` | NewAPI v0.13.2 基线版本说明与 upstream 差异追踪 |

#### 开发规范

| 文档 | 说明 |
|------|------|
| `02.backend-01.json-wrapper-usage.md` | JSON 操作规范（必须用 common/json.go） |
| `02.backend-02.database-compatibility.md` | 三库兼容开发规范 |
| `02.backend-03.relay-dto-explicit-zero-values.md` | 上游 Relay DTO 显式零值保留规范 |

#### 质量与流程

| 文档 | 说明 |
|------|------|
| `06.quality-01.git-workflow.md` | Git 提交与分支规范 |
| `06.quality-02.change-assurance-framework.md` | NACP 变更验证与上线保障体系（CAF） |
| `06.quality-03.version-release-management.md` | 版本号、CHANGELOG、Git tag 与 GitHub Release 管理规范 |

#### AI 记忆体系

| 文档 | 说明 |
|------|------|
| `10.ai-memory-01.naming.md` | NACP `.ai` 六层记忆体系命名规范与迁移清单 |

---

### 🎯 当前重点

- **v0.1.0 已完成**: 智能重试与渠道健康管理
- **下一步**: 响应时间监控预警、管理后台可视化

---

### 📚 参考资料

- [L1 项目指南](../L1#Overview/guide.md)
- [AGENTS.md](../../AGENTS.md) — 项目级 AI 规则
- [ZERO 框架](../../ZERO/.ai/L1#Overview/guide.md) — .ai 体系参考
- [ZERO AI 记忆体系命名规范](../../ZERO/.ai/L3#Standards/standards/10.ai-memory-02.naming.md)
- [ZERO 六层架构官方命名对照表](../../ZERO/.ai/L3#Standards/standards/10.ai-memory-03.six-layer-naming.md)

---

**最后更新**：2026-05-15
