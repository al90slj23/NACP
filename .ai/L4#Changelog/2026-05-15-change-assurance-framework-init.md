# 2026-05-15 — CAF 变更验证体系初始化

## 操作内容

新增 NACP 变更验证与上线保障体系（CAF），用于把每次新增、增强、修复、重构转化为标准化影响分析、测试计划、执行验证和证据留存流程。

## 文件变更

- 新增 `.ai/L3#Standards/standards/06.quality-02.change-assurance-framework.md`
- 新增 `.ai/L5#Knowledge/caf-change-assurance-framework-playbook.md`
- 更新 `.ai/L2#Index/toc.md`
- 更新 `.ai/L5#Knowledge/README.md`

## 决策记录

1. 正式命名为 `NACP Change Assurance Framework`，简称 `CAF`。
2. CAF 定位为质量保障体系，不只是测试计划。
3. CAF 七阶段为：系统知识定位、影响分析树、风险分级、测试计划生成、执行验证、验收证据、知识回流。
4. 后续每次 P0/P1 变更都必须至少完成影响树、测试计划、线上测试站验证和 `.ai` 知识回流。

