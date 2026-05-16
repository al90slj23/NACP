# 2026-05-16 — API 驱动测试夹具规范回流

## 背景

SFT、日志、计费、渠道优先级等测试不能依赖手动 DB seed。真实测试应通过 API 构造测试环境：

1. 管理员 API 调整渠道分组和优先级。
2. 管理员 API 创建实际失效的测试渠道。
3. 普通用户 API 注册、登录、创建 token。
4. 使用普通用户 token 调用真实 relay。
5. 再通过管理员 API 检查日志、trace、计费和统计。

## 文件变更

新增：

1. `.ai/L3#Standards/standards/06.quality-04.api-driven-test-fixtures.md`

更新：

1. `.ai/L5#Knowledge/caf-change-assurance-framework-playbook.md`
2. `.ai/L5#Knowledge/README.md`

## 决策

1. DB seed 只能作为 UI fixture 或旧日志兼容测试，不作为真实链路测试结论。
2. 日常本地测试优先使用 `http://localhost:5173/api/*` 和本地 relay API。
3. 测试渠道、分组、优先级、失效渠道必须通过管理员 API 创建或恢复。
4. `.ai` 中不得记录真实 token key、渠道 key、管理员密码。

