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
| `01.arch-01.relay-retry.md` | 重试机制增强技术方案（v0.1.0 已实现） |
| `01.arch-02.baseline.md` | 基线版本说明与 upstream 差异追踪 |

#### 开发规范

| 文档 | 说明 |
|------|------|
| `02.dev-01.json-usage.md` | JSON 操作规范（必须用 common/json.go） |
| `02.dev-02.db-compat.md` | 三库兼容开发规范 |
| `02.dev-03.dto-pointer.md` | 上游 DTO 指针类型规范 |

#### 质量与流程

| 文档 | 说明 |
|------|------|
| `03.quality-01.git.md` | Git 提交与分支规范 |
| `03.quality-02.version.md` | 版本线管理规范 |

---

### 🎯 当前重点

- **v0.1.0 已完成**: 智能重试与渠道健康管理
- **下一步**: 响应时间监控预警、管理后台可视化

---

### 📚 参考资料

- [L1 项目指南](../L1#Overview/guide.md)
- [AGENTS.md](../../AGENTS.md) — 项目级 AI 规则
- [ZERO 框架](../../ZERO/.ai/L1#Overview/guide.md) — .ai 体系参考

---

**最后更新**：2026-05-13
