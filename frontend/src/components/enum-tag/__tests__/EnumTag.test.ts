import { mount } from '@vue/test-utils';
import { ElTag } from 'element-plus';
import { describe, expect, it } from 'vitest';

import EnumTag from '../EnumTag.vue';
import { EnumValue } from '@/common/Enum';
import { i18n } from '@/i18n';

const t = (key: string) => i18n.global.t(key);

/** 只登记两个取值的枚举源，label 用真实字典键，避免把测试文案钉死在中文上 */
const enums = {
    Up: EnumValue.of('up', 'agentops.feedbackUp').tagTypeSuccess(),
    Down: EnumValue.of('down', 'agentops.feedbackDown').tagTypeDanger(),
};

const mountTag = (value: unknown) =>
    mount(EnumTag, {
        props: { enums, value: value as string | number | boolean | null | undefined },
        global: { plugins: [i18n], components: { ElTag } },
    });

describe('EnumTag 的取值渲染契约', () => {
    it('命中的枚举值渲染对应标签文案', () => {
        const wrapper = mountTag('up');
        expect(wrapper.text()).toBe(t('agentops.feedbackUp'));
        expect(wrapper.find('.el-tag').classes()).toContain('el-tag--success');
    });

    it('空值不渲染任何内容（真机实测到旧行为会渲出一个没有文字的红框，看着像数据损坏）', () => {
        for (const empty of ['', null, undefined]) {
            const wrapper = mountTag(empty);
            expect(wrapper.find('.el-tag').exists(), `value=${String(empty)} 时空值必须整格不渲染`).toBe(false);
            expect(wrapper.text()).toBe('');
        }
    });

    it('0 与 false 是合法枚举值，不得被当空值吞掉（判据必须是空串/null，不能是 falsy）', () => {
        const zeroEnums = { Zero: EnumValue.of(0, 'agentops.feedbackUp').tagTypeInfo() };
        const wrapper = mount(EnumTag, {
            props: { enums: zeroEnums, value: 0 },
            global: { plugins: [i18n], components: { ElTag } },
        });
        expect(wrapper.find('.el-tag').exists(), '数字 0 是合法枚举值，用 falsy 判空会把它整格吞掉').toBe(true);
        expect(wrapper.text()).toBe(t('agentops.feedbackUp'));
    });

    it('有值但未登记展示元数据时仍回显原始值 + danger 色（这是刻意决策，不得被空值优化顺手改掉）', () => {
        const wrapper = mountTag('brand_new_from_backend');
        expect(wrapper.find('.el-tag').exists()).toBe(true);
        expect(wrapper.text()).toBe('brand_new_from_backend');
        expect(wrapper.find('.el-tag').classes()).toContain('el-tag--danger');
    });
});
