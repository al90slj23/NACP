# 2026-05-13 — 项目初始化

## 操作内容

1. 从 QuantumNous/new-api v0.13.2 (commit bee339d) 克隆代码基线
2. 克隆 ZERO 框架到 `./ZERO/` 目录
3. 建立 `.ai/` 六层架构文档体系
4. 创建核心规范文档

## 文件变更

### 新增
- `.ai/L0#Execution/` — 工作执行层（目录结构）
- `.ai/L1#Overview/guide.md` — 项目指南
- `.ai/L1#Overview/README.md` — L1 说明
- `.ai/L2#Index/toc.md` — 规范索引
- `.ai/L2#Index/README.md` — L2 说明
- `.ai/L3#Standards/standards/01.arch-01.relay-retry.md` — 重试机制设计（待设计）
- `.ai/L3#Standards/standards/01.arch-02.baseline.md` — 基线版本说明
- `.ai/L3#Standards/standards/02.dev-01.json-usage.md` — JSON 操作规范
- `.ai/L3#Standards/standards/02.dev-02.db-compat.md` — 三库兼容规范
- `.ai/L3#Standards/standards/02.dev-03.dto-pointer.md` — DTO 指针类型规范
- `.ai/L3#Standards/standards/03.quality-01.git.md` — Git 规范
- `.ai/L4#Changelog/2026-05-13-project-init.md` — 本文件
- `ZERO/` — ZERO 框架（子目录）

## 决策记录

- 选择直接 fork 修改而非继续 Venom 外挂方式，原因：重试机制需要深入修改 relay 核心代码
- 版本线命名 classic-plus-0.x，明确与 upstream v1 隔离
- .ai 体系采用 ZERO 六层架构，但内容适配 Go + React 技术栈

---

**操作者**：AI + 用户
