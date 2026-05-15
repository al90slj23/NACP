# 2026-05-15 — 版本升级说明规范

## 操作内容

补充版本跃迁记录规范，明确从 `0.1` 升级到 `0.2` 这类版本变化必须有固定记录位置和固定结构。

## 文件变更

- 更新 `.ai/L3#Standards/standards/06.quality-03.version-release-management.md`
- 新增 `.ai/L5#Knowledge/release-classic-plus-0-2-0-upgrade-review.md`
- 更新 `CHANGELOG.md`
- 更新 `.ai/L5#Knowledge/README.md`

## 决策记录

1. 根目录 `CHANGELOG.md` 作为版本摘要和对外入口。
2. `L5#Knowledge (Knowledge)` 中的 `release-*-upgrade-review.md` 作为版本跃迁详版。
3. GitHub Release 正文从 `CHANGELOG.md` 和对应 L5 详版中提取。
4. `.ai/L4#Changelog (Changelog)` 只记录操作过程，不承载版本说明主体。

