import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import CollapsibleSection from '../CollapsibleSection.vue';

/**
 * 折叠态回归：强制展开解除后必须保持展开。
 *
 * 缺陷类（真实发生过）：`open = forceOpen || innerOpen`，审批中卡片被强制展开（innerOpen 仍为 false），
 * 用户点完决定、forceOpen 变 false 的瞬间卡片**突然收起**——像内容被吞掉，而这是用户刚操作完的位置。
 */
describe('CollapsibleSection 强制展开解除', () => {
    const mountSection = (forceOpen: boolean) =>
        mount(CollapsibleSection, {
            props: { title: '工具调用', forceOpen },
            slots: { default: '<div class="detail">详情</div>' },
        });

    const expanded = (wrapper: ReturnType<typeof mountSection>) =>
        wrapper.find('.collapsible-section__collapse').classes().includes('is-expanded');

    it('强制展开期间展开，解除后保持展开', async () => {
        const wrapper = mountSection(true);
        expect(expanded(wrapper)).toBe(true);

        await wrapper.setProps({ forceOpen: false });
        expect(expanded(wrapper)).toBe(true);
    });

    it('非强制态默认收起，点击可展开、再点收起', async () => {
        const wrapper = mountSection(false);
        expect(expanded(wrapper)).toBe(false);

        await wrapper.find('button').trigger('click');
        expect(expanded(wrapper)).toBe(true);

        await wrapper.find('button').trigger('click');
        expect(expanded(wrapper)).toBe(false);
    });

    it('强制展开期间点击不改变内部态（解除后仍是展开）', async () => {
        const wrapper = mountSection(true);
        await wrapper.find('button').trigger('click');
        await wrapper.setProps({ forceOpen: false });
        expect(expanded(wrapper)).toBe(true);
    });
});
