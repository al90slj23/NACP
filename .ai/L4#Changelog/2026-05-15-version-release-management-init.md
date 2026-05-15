# 2026-05-15 — 版本与发布管理初始化

## 操作内容

建立 NACP 版本记录与 GitHub Release 管理规则，明确 `VERSION`、`CHANGELOG.md`、Git tag、GitHub Release、`.ai/L4#Changelog (Changelog)` 的分工。

## 文件变更

- 更新 `VERSION` 为 `classic-plus-0.2.0-dev`
- 更新 `CHANGELOG.md`，新增 `classic-plus-0.2.0-dev (Unreleased)` 段落
- 更新 `.ai/L1#Overview/guide.md` 当前版本
- 新增 `.ai/L3#Standards/standards/06.quality-03.version-release-management.md`
- 更新 `.ai/L2#Index/toc.md`
- 更新 `.ai/L3#Standards/standards/06.quality-01.git-workflow.md`
- 新增 `.github/release.yml`，用于 GitHub 自动生成 Release Notes 分类

## 决策记录

1. `VERSION` 是当前构建版本源，不能为空。
2. `CHANGELOG.md` 是仓库内长期版本记录。
3. `.ai/L4#Changelog (Changelog)` 是内部操作过程审计。
4. 正式发布使用 Git tag + GitHub Release。
5. NACP tag 使用 `classic-plus-x.y.z`，不使用 `v0.2.0`，避免和上游 NewAPI tag 混淆。

