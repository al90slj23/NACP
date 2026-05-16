# 2026-05-16 — CAF 本地测试数据库隔离规则

## 背景

本地开发环境曾允许连接并共用 `nacp.m.srl` 测试站数据库。实际执行 SFT 日志和计费测试时，这会让本地代码、线上测试站数据、迁移状态和用户看到的日志混在一起，难以判断问题来自代码、前端代理、数据库结构还是测试站部署。

## 决策

1. CAF Stage A 本地测试改为使用本地隔离数据库。
2. 默认本地数据库为 Docker MySQL：
   - 容器：`nacp-mysql-dev`
   - 地址：`127.0.0.1:3307`
   - 数据库：`nacp_dev`
3. 本地测试仍优先通过 API 执行，不默认直连数据库。
4. `nacp.m.srl` 只作为 CAF Stage B 线上测试站入口，使用独立测试站数据库。
5. 如果线上测试需要复用本地测试基线，必须通过明确的 Docker 镜像、迁移脚本或受控数据库快照同步，不能让本地服务和测试站长期共用同一个实时数据库。

## 更新文件

1. `.ai/L3#Standards/standards/06.quality-02.change-assurance-framework.md`
2. `.ai/L3#Standards/standards/06.quality-04.api-driven-test-fixtures.md`
3. `.ai/L5#Knowledge/caf-change-assurance-framework-playbook.md`

