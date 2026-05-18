# 2026-05-18 渠道单位账号平台令牌联动

## 变更

- 渠道创建/编辑弹窗接入所属单位、所属账号、平台令牌读取和平台令牌创建。
- 所属单位下拉显示单位名称、平台类型、账号数和有效 API 地址。
- 选择单位后自动清理旧账号和旧平台令牌状态，并回填单位 API 地址。
- 选择账号后再次回填单位 API 地址，并清理旧平台令牌缓存。
- 密钥旁按钮命名为“获取平台令牌”，明确获取的是上游单位平台 API 令牌。
- 前端请求平台令牌相关接口时优先展示后端 message。
- 后端 `/api/unit` 和 `/api/unit/` 同时兼容。
- 修复 `POST /api/channel/` 请求体缺少 `channel` 时可能空指针 panic 的问题，改为返回 `channel cannot be empty`。

## 验证

- `GOCACHE=/private/tmp/nacp-gocache go test ./controller ./router ./service ./model`
- `bun run build`
- 本地 API 验证：
  - `/api/unit` 返回启用单位。
  - `/api/unit/` 返回启用单位。
  - 重启本地开发环境后，`/api/unit` 无斜杠兼容确认生效。
  - `/api/unit/1/accounts` 返回挂载账号。
  - `/api/unit/1/accounts/1/token_options` 返回平台分组和模型。
  - `/api/unit/1/accounts/1/tokens` 返回平台令牌。
  - 新建禁用测试渠道 `nacp-unit-link-smoke-20260518` 成功保存 `unit_id=1`、`unit_account_id=1`、`base_url=https://528ai.cc`。
  - 编辑该测试渠道后单位账号关联继续保持。
  - 使用不存在的 `unit_account_id` 创建渠道返回“所属账号不存在”。
  - `POST /api/channel/` 请求体缺少 `channel` 时返回 `channel cannot be empty`，确认不再触发 panic。

## 注意

- 本地浏览器自动化没有可直接导入的 Playwright 包，本次未强行下载依赖。
- 前端构建存在既有 Browserslist、lottie eval、大 chunk 警告，不是本次改动引入的阻塞问题。
