# 2026-05-14 — v0.1.1 错误日志拆分 + 本地开发环境

## 操作内容

1. 错误日志拆分为 type 51（已拦截）和 type 52（客户可见）
2. 前端日志页面新增两种类型的筛选和标签显示
3. gogogo.sh 本地开发环境完善（自动启动 Docker、构建 web/dist、连接远程 MySQL）
4. SKIP_DB_MIGRATION 环境变量优化本地开发启动速度
5. 部署流程：GitHub Actions 构建 → GHCR → 服务器 pull

## 文件变更

### 新增
- `docker-compose.dev.yml` — 本地开发 MySQL
- `.env` — 本地开发环境变量（gitignore 排除）
- `.env.example` — 环境变量模板

### 修改
- `model/log.go` — 新增 LogTypeErrorIntercepted(51) + LogTypeErrorClientVisible(52) + RecordErrorLogWithType()
- `controller/relay.go` — processChannelError 默认 type 51，循环结束记录 type 52
- `model/main.go` — SKIP_DB_MIGRATION 环境变量支持
- `web/src/components/table/usage-logs/UsageLogsFilters.jsx` — 下拉菜单新增 51/52
- `web/src/components/table/usage-logs/UsageLogsColumnDefs.jsx` — 标签渲染新增黄色"已拦截"/红色"客户可见"
- `gogogo.sh` — 本地开发环境（自动 Docker、web/dist、MySQL）

## 数据库迁移

部署后执行：
```sql
UPDATE logs SET type = 52 WHERE type = 5;
```

## 决策记录

- 错误类型用 51/52 而非 8/9 — 语义上是"5 的子类型"，一看就知道是错误相关
- 历史 type=5 数据迁移为 52 — 旧版本所有错误都是客户端可见的
- 本地开发优先连远程 MySQL — 避免空数据，SKIP_DB_MIGRATION 跳过慢迁移

---

**操作者**：AI + 用户
