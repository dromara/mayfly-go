/**
 * 流程业务类型的前端注册完整性守卫。
 *
 * 存在理由：AI 提案审批接入时后端注册了 agent_remediation_flow 的 BizHandler，但前端枚举与
 * 详情组件都没跟上——待办列表的「业务类型」列显示 `-`、详情抽屉业务信息一片空白，工单看起来
 * 像「不存在的审批」。这类漂移（后端加 bizType、前端漏登记）没有任何编译期或运行时报错，
 * 只能靠一条把三方钉在一起的测试拦住。
 */
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import { FlowBizType, ManualFlowBizType } from '../enums';
import zhCn from '@/i18n/zh-cn/flow';
import en from '@/i18n/en/flow';

const bizTypeValues = Object.values(FlowBizType).map((ev) => ev.value);

describe('FlowBizType 前端注册完整性', () => {
    it('AI 提案审批业务类型已登记（与后端 agentops.AgentRemediationFlowBizType 同值）', () => {
        expect(bizTypeValues).toContain('agent_remediation_flow');
    });

    it('每个业务类型的 i18n 标签在中英文文案里都取得到', () => {
        for (const ev of Object.values(FlowBizType)) {
            const [ns, key] = ev.label.split('.');
            expect(ns, `标签键 ${ev.label} 应属于 flow 命名空间`).toBe('flow');
            expect((zhCn.flow as Record<string, unknown>)[key] ?? null, `zh-cn 缺少 ${ev.label}`).not.toBeNull();
            expect((en.flow as Record<string, unknown>)[key] ?? null, `en 缺少 ${ev.label}`).not.toBeNull();
        }
    });

    it('详情抽屉为每个业务类型都登记了业务信息组件', () => {
        // ProcinstDetail 按 bizComponents[bizType] 分发渲染；漏一条时该页签整个空白
        const source = readFileSync(join(import.meta.dirname, '../ProcinstDetail.vue'), 'utf8');
        const registry = source.slice(source.indexOf('const bizComponents'), source.indexOf('const bizComponents') + 800);
        for (const value of bizTypeValues) {
            expect(registry, `ProcinstDetail 缺少 ${value} 的业务组件登记`).toContain(value);
        }
    });

    it('手动提单表单不含 AI 提案类型（无提案背书的单批完只会静默忽略）', () => {
        expect(Object.values(ManualFlowBizType).map((ev) => ev.value)).not.toContain('agent_remediation_flow');
    });
});
