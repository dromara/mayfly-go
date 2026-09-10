import { mount } from '@vue/test-utils';
import { defineComponent, h } from 'vue';
import { describe, expect, it, vi, afterEach } from 'vitest';

// SvgIcon 带样式块，会触发 postcss 配置加载，测试中 mock 掉；auth/commands 链路隔离真实 i18n 初始化
vi.mock('@/components/svg-icon/index.vue', () => ({ default: { name: 'SvgIcon', render: () => null } }));
vi.mock('@/i18n', () => ({ i18n: { global: { t: (k: string) => k, locale: 'zh-cn' } } }));
vi.mock('@/hooks/useI18n', () => ({ useI18n: () => ({ t: (k: string) => k }) }));
vi.mock('@/components/auth/auth', () => ({ hasPerm: vi.fn(() => true), hasPerms: vi.fn() }));

import { registerContributor } from '../registry';
import { registerCommand, registerMenu } from '../commands';
import { TreeApiKey } from '../context';
import TreeNodeRow from '../TreeNodeRow.vue';
import type { TreeNode } from '../types';

/**
 * 通用行组件专测：labelRenderer 协议（kind 专属 label 渲染归贡献者，行组件零 kind 知识）。
 */
const TreeApiStub = { locate: async () => {}, refresh: () => {}, getNode: () => undefined };

const mountRow = (data: TreeNode, showActions = false) =>
    mount(TreeNodeRow, {
        props: { data, showActions },
        global: {
            mocks: { $t: (key: string) => key },
            provide: { [TreeApiKey as symbol]: TreeApiStub },
        },
    });

describe('tree/TreeNodeRow 通用行组件', () => {
    it('贡献者声明 labelRenderer 时行内 label 走专属渲染', () => {
        const Custom = defineComponent({
            props: { node: { type: Object, required: true } },
            setup: (p) => () => h('b', { class: 'custom-label' }, p.node.label),
        });
        registerContributor({ kind: 'row-test', labelRenderer: Custom });

        const w = mountRow({ key: 'k', kind: 'row-test', label: 'a/b', params: {}, hasChildren: false });
        expect(w.find('.custom-label').exists()).toBe(true);
        expect(w.text()).toContain('a/b');
    });

    it('未声明 labelRenderer 的 kind 用默认 i18n 文本渲染', () => {
        registerContributor({ kind: 'row-plain' });

        const w = mountRow({ key: 'k', kind: 'row-plain', label: 'plain.label', params: {}, hasChildren: false });
        expect(w.find('.custom-label').exists()).toBe(false);
        expect(w.text()).toContain('plain.label');
    });

    it('操作按钮驻留显示：悬浮即时不闪现，驻留 0.5s 后才出现，移开即隐藏', async () => {
        vi.useFakeTimers();
        registerContributor({ kind: 'row-park' });
        registerCommand({ id: 'row-park.cmd', txt: 'cmd', handler: vi.fn() });
        registerMenu({ command: 'row-park.cmd', kinds: ['row-park'] });

        const w = mountRow({ key: 'k', kind: 'row-park', label: 'x', params: {}, hasChildren: false }, true);

        // 悬浮即时：未驻留，不出现
        await w.trigger('mouseenter');
        expect(w.find('.tree-row-actions').exists()).toBe(false);

        // 驻留满阈值：出现
        vi.advanceTimersByTime(500);
        await w.vm.$nextTick();
        expect(w.find('.tree-row-actions').exists()).toBe(true);

        // 移开：立即隐藏（且未满阈值的悬浮不残留定时器）
        await w.trigger('mouseleave');
        await w.vm.$nextTick();
        expect(w.find('.tree-row-actions').exists()).toBe(false);

        // 再次短暂悬浮后移开：不出现（定时器已被清除）
        await w.trigger('mouseenter');
        vi.advanceTimersByTime(300);
        await w.trigger('mouseleave');
        vi.advanceTimersByTime(300);
        await w.vm.$nextTick();
        expect(w.find('.tree-row-actions').exists()).toBe(false);

        vi.useRealTimers();
        w.unmount();
    });

    afterEach(() => {
        vi.useRealTimers();
    });
});
