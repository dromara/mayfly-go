/**
 * 流程内条件编辑器（连线跳转条件 / 节点完成条件）交互回归。
 *
 * 钉住三条不变式，都是实测踩过的坑：
 * 1. 点「添加条件」必须真的出现可填写的条件行，而不是又回到按钮分支（空分组是死路）
 * 2. 套用预置得到的是深拷贝：改条件树不能改到后端下发、多节点共享的预置本体
 * 3. structuredClone 克隆不了响应式 Proxy，预置走 JSON 深拷贝
 */
import { describe, expect, it, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { reactive } from 'vue';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';

import FlowConditionEditor from '@/views/flow/components/FlowConditionEditor.vue';
import zhFlow from '@/i18n/zh-cn/flow';
import zhCommon from '@/i18n/zh-cn/common';
import type { PolicyScenario } from '@/components/policy-builder';

vi.mock('@/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }));

const i18n = createI18n({ legacy: false, globalInjection: true, locale: 'zh-cn', messages: { 'zh-cn': { ...zhCommon, ...zhFlow } } });

function conditionScenario(): PolicyScenario {
    return {
        bizType: 'flow_instance',
        fields: [
            {
                key: 'nrOfCompleted',
                titleKey: 'flow.field.nrOfCompleted',
                group: 'approval',
                type: 'number',
                editorKey: 'number',
                options: [],
                ops: [{ name: 'gte', labelKey: 'flow.op.gte', valueKind: 'single' }],
            },
        ],
        checks: [],
        conditionPresets: [
            {
                key: 'orSign',
                titleKey: 'flow.orSign',
                descriptionKey: 'flow.orSignTip',
                ruleNode: { kind: 'condition', field: 'nrOfCompleted', op: 'gte', value: 1 },
            },
        ],
    };
}

function mountEditor(scenario: PolicyScenario) {
    return mount(FlowConditionEditor, {
        props: { scenario, modelValue: null, showPresets: true },
        global: { plugins: [ElementPlus, i18n] },
    });
}

describe('流程内条件编辑器', () => {
    it('点「添加条件」出现可填写的条件行，而不是留在按钮分支', async () => {
        const wrapper = mountEditor(reactive(conditionScenario()));

        const add = wrapper.findAll('button').find((btn) => btn.text().includes('添加条件'));
        expect(add).toBeTruthy();
        await add!.trigger('click');

        const emitted = wrapper.emitted('update:modelValue');
        expect(emitted).toBeTruthy();
        const node = emitted![0][0] as { kind: string; items?: unknown[] };
        // 空分组会被保存校验判非法，也让面板回到「只有一个按钮」的死路，所以建组必须即给一行
        expect(node.kind).toBe('group');
        expect(node.items).toHaveLength(1);
    });

    it('套用预置得到深拷贝，改条件树不会改到共享的预置本体', async () => {
        const scenario = reactive(conditionScenario());
        const before = JSON.stringify(scenario.conditionPresets![0].ruleNode);

        const wrapper = mountEditor(scenario);
        const presetButton = wrapper.findAll('button').find((btn) => btn.text().includes('或签'));
        expect(presetButton).toBeTruthy();
        await presetButton!.trigger('click');

        const emitted = wrapper.emitted('update:modelValue');
        expect(emitted).toBeTruthy();
        const applied = emitted![0][0] as Record<string, unknown>;
        expect(applied.value).toBe(1);
        expect(applied).not.toBe(scenario.conditionPresets![0].ruleNode);

        // 模拟用户把阈值改大：预置必须保持原样，否则下一个节点套「或签」会拿到改过的值
        applied.value = 9;
        expect(JSON.stringify(scenario.conditionPresets![0].ruleNode)).toBe(before);
    });

    it('未配置条件时不渲染规则解读，避免给出一条空白摘要', () => {
        const wrapper = mountEditor(reactive(conditionScenario()));
        expect(wrapper.find('.editor-preview').exists()).toBe(false);
    });
});
