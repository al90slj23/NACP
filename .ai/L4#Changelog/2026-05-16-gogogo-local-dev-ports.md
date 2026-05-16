# 2026-05-16 — gogogo.sh 本地开发端口和本地库启动规则

> 后续修订：本地 MySQL 方案已取消，见 `2026-05-16-gogogo-online-test-db.md`。当前保留本文作为端口决策和历史操作记录。

## 操作内容

1. 调整 `gogogo.sh` 本地开发入口，固定本地开发端口：
   - 前端 Vite：`23901`
   - 后端 Go：`23900`
   - 本地 MySQL：`23906`
2. 本地开发启动时强制优先使用本地隔离数据库：

```text
SQL_DSN=nacp_dev:nacp_dev_pass@tcp(127.0.0.1:23906)/nacp_dev?charset=utf8mb4&parseTime=True&loc=Local
```

3. 新增非交互式重启方式：

```bash
./gogogo.sh 0 restart
```

4. 启动前自动清理 `23900`、`23901` 端口占用；`23906` 若被非 `nacp-mysql-dev` Docker 容器占用，会先停止占用容器。
5. 修复脚本直接 `source .env` 的问题，避免未加引号的 `SQL_DSN=...tcp(...)...` 被 shell 解析失败。
6. 增强 Docker Desktop 自动拉起逻辑：
   - 先检测 `docker info`，daemon 已可用则直接继续。
   - daemon 不可用时尝试固定路径 `/Applications/Docker.app`、`~/Applications/Docker.app`。
   - 再尝试 bundle id `com.docker.docker`。
   - 再通过 `mdfind` 查找 Docker Desktop。
   - 最后回退到 `open -a Docker` / `open -a "Docker Desktop"`。
   - 等待时间提升到 120 秒。

## 决策记录

本地 Stage A 测试必须与 `nacp.m.srl` 测试站数据库隔离。以后需要重启本地测试环境时，优先使用 `gogogo.sh`，避免手动启动造成端口、数据库、迁移状态不一致。

端口段选择 `239**`，用于避开 YYSYYF 已固定使用的 `23200/23201`、Graphiti `23220-23222`、以及临时多开 `23280-23299`。

## 验证

1. `bash -n gogogo.sh` 通过。
2. 在当前环境 Docker Desktop 不可自动启动时，`./gogogo.sh 0 e` 能明确失败并提示手动启动 Docker，不再卡住等待。
