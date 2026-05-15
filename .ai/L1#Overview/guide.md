# NACP — NewAPI Classic Plus

> **定位**：基于 QuantumNous/new-api v0.13.2 经典版本线的最终增强版
> **版本**：classic-plus-0.2.0-dev
> **创建日期**：2026-05-13
> **英文定位语**：The final enhanced edition of NewAPI Classic.

---

## 🎯 项目背景

NACP 是基于 NewAPI v0.13.2 classic 版本线的最终增强版。

**为什么 fork：**
- 上游质量不一，经常出现问题
- 原有重试机制不能满足业务需求
- 需要直接修改核心代码来解决紧急业务问题
- 为毒液（Venom）系统做技术迭代和积累

**与 Venom 的关系：**
- NACP 直接修改宿主代码，解决紧急需求
- Venom 后续读取 NACP 的迭代经验和代码，自行吸收
- 两者独立演进，不互相阻塞

---

## 📋 命名规范

| 项目 | 值 |
|------|-----|
| 产品名 | NewAPI Classic Plus |
| 简称 | NACP |
| 仓库名 | new-api-classic-plus |
| 短仓库名/镜像名 | nacp |
| 版本线 | classic-plus-0.x |
| 中文名 | NewAPI Classic 增强版 |

---

## 🏗️ 技术栈

- **Backend**: Go 1.22+, Gin, GORM v2
- **Frontend**: React 18, Vite, Semi Design UI
- **Databases**: SQLite / MySQL / PostgreSQL（三库兼容）
- **Cache**: Redis (go-redis) + 内存缓存
- **Auth**: JWT, WebAuthn/Passkeys, OAuth
- **Frontend PM**: Bun

---

## 📁 核心架构

```
Router → Controller → Service → Model

router/        — HTTP 路由
controller/    — 请求处理
service/       — 业务逻辑
model/         — 数据模型 (GORM)
relay/         — AI API 中继/代理
  relay/channel/ — 各供应商适配器 (openai/, claude/, gemini/, aws/ ...)
middleware/    — 中间件（认证、限流、分发等）
setting/       — 配置管理
common/        — 共享工具
dto/           — 数据传输对象
constant/      — 常量定义
types/         — 类型定义
i18n/          — 后端国际化
web/           — React 前端
```

---

## 🔑 核心规则（零容忍）

1. **JSON 操作必须用 `common/json.go` 封装** — 禁止直接 import `encoding/json`
2. **三库兼容** — 所有 DB 代码必须同时兼容 SQLite / MySQL / PostgreSQL
3. **前端用 Bun** — `bun install`, `bun run dev`, `bun run build`
4. **上游 DTO 保留显式零值** — 可选标量字段用指针类型 + `omitempty`
5. **受保护信息不可修改** — new-api / QuantumNous 相关品牌信息严禁删改
6. **Git 提交规范** — `<type>(<scope>): <subject>`
7. **版本线隔离** — 不跟 upstream v1 新 UI/新路线混合
8. **代码修改原则：新增优先，慎改源码** — 以新增文件/函数为主；必须改源码时，先全局检查调用方和依赖方，确认影响范围后再动手

---

## 🎯 当前迭代重点

### v0.1.0 — 智能重试与渠道健康管理 ✅ 已完成
- 渠道健康状态机（Healthy/Probing/Degraded/Recovering/ManuallyDisabled）
- 同渠道重试 + 并行预热顺位渠道
- 同优先级渠道切换（最多 3 个后才降级）
- 渠道亲和性健康感知（Degraded 跳过亲和）
- 低渠道告警 + 安全兜底
- 详见：`CHANGELOG.md`、`NACP.md`

### 下一步
- 轻量探测响应时间监控（预警延迟升高）
- 管理后台可视化渠道健康状态
- 配置项可通过管理后台调整

---

## 📂 .ai 文档结构（六层架构）

```
.ai/
├── L0#Execution/     # 工作执行（hooks, skills, specs, workflows）
├── L1#Overview/      # 项目概览（本文件）
├── L2#Index/         # 规范索引
├── L3#Standards/     # 完整规范
├── L4#Changelog/     # 操作日志
└── L5#Knowledge/     # 知识图谱
```

---

## 📚 参考

- 基线版本：QuantumNous/new-api v0.13.2 (commit bee339d)
- ZERO 框架：`./ZERO/` 目录
- AGENTS.md：项目级 AI 规则

---

**最后更新**：2026-05-13
