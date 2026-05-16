# 2026-05-16 — CAF 本地入口与共享测试库规则回流

## 背景

NACP 同时存在：

1. 线上测试站：`nacp.m.srl`。
2. 本地前后端：常用入口 `http://localhost:5173/`。
3. 本地环境可连接并共用 `nacp.m.srl` 的测试数据库。

因此日常测试不应默认把“本地测试”理解为本地独立数据库，也不应默认直接访问数据库。

## 固化规则

已更新：

1. `.ai/L3#Standards/standards/06.quality-02.change-assurance-framework.md`
   - 明确 Stage A 本地测试入口为 `http://localhost:5173/`。
   - 明确本地环境可以连接 `nacp.m.srl` 测试库。
   - 明确常规测试优先通过本地 API 执行，不默认直连数据库。

2. `.ai/L5#Knowledge/caf-change-assurance-framework-playbook.md`
   - 新增“本地前端入口 / dashboard API / relay API / 测试库 / 数据库直连”约定表。
   - 将日志测试顺序中的 DB 检查改为“只有 API 证据不足时”才执行。

## 后续执行原则

常规验证顺序：

```text
http://localhost:5173/api/status
-> 本地 API 创建/查询/触发行为
-> 本地前端页面确认展示
-> 只有 API 无法证明时补充数据库查询
```

对于 relay 请求，先从 `/api/status.server_address` 获取本地后端地址，再调用对应 `/v1/*`。

