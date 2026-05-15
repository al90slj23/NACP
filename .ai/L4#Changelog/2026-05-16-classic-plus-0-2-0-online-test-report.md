# 2026-05-16 — classic-plus-0.2.0 在线测试报告

## 操作内容

- 按 CAF classic-plus-0.2.0 深度测试计划执行第一轮线上测试。
- 创建普通测试用户和 token，完成 Claude、Codex、SFT 容错成功、SFT 全失败、日志 API、数据库不变量和计费计量验证。
- 记录本地 Go 测试、前端构建和 UI 验证状态。

## 文件变更

- 新增 `.ai/L5#Knowledge/caf-classic-plus-0-2-0-online-test-report.md`
- 更新 `.ai/L5#Knowledge/README.md`

## 决策记录

- 测试报告不记录任何 token key、渠道 key 或管理员密码，只记录 Request ID、Log ID、trace 结构和非敏感测试账号信息。
- 本轮 UI 视觉验收因 Codex 内置浏览器白屏标记为 BLOCKED，不计入 PASS。
- `relay/helper` 流扫描单测失败和轻量探测准确性作为下一轮 P1 风险继续跟进。
