import { mount } from '@vue/test-utils';
import { ElInput } from 'element-plus';
import { defineComponent, h } from 'vue';
import { describe, expect, it, vi } from 'vitest';

// SvgIcon 带样式块会触发 postcss 加载；查找条不关心图标实现
vi.mock('@/components/svg-icon/index.vue', () => ({ default: { name: 'SvgIcon', render: () => null } }));

import TreeFindBar from '../TreeFindBar.vue';

const mountBar = (props: Partial<{ visible: boolean; query: string; total: number; no: number; countReady: boolean }> = {}) =>
    mount(TreeFindBar, {
        props: { visible: true, query: '', total: 0, no: 0, countReady: true, ...props },
        global: { mocks: { $t: (key: string) => key }, components: { ElInput } },
    });

/**
 * 查找条交互契约：常态零布局占位（只一个入口）、输入上抛、Enter/Shift+Enter 跳转、
 * Escape 就地消费（不得连带触发页面级全屏退出）、计数口径可读。
 */
describe('tree/TreeFindBar 查找条', () => {
    it('收起态只渲染悬浮入口，点击上抛 expand', async () => {
        const w = mountBar({ visible: false });
        expect(w.find('.tree-find-bar').exists()).toBe(false);

        const trigger = w.find('.tree-find-trigger');
        expect(trigger.exists()).toBe(true);
        await trigger.trigger('click');
        expect(w.emitted('expand')).toHaveLength(1);
    });

    it('输入即上抛查找词，由父组件持有过滤态', async () => {
        const w = mountBar();
        await w.find('input').setValue('ab');
        expect(w.emitted('update:query')?.at(-1)).toEqual(['ab']);
    });

    it('Enter 下一个、Shift+Enter 上一个', async () => {
        const w = mountBar({ query: 'ab' });
        await w.find('input').trigger('keydown', { key: 'Enter' });
        await w.find('input').trigger('keydown', { key: 'Enter', shiftKey: true });
        expect(w.emitted('step')).toEqual([[1], [-1]]);
    });

    it('输入法组合态的 Enter 不触发跳转', async () => {
        // EP 在 isComposing 期间不 emit input，但 keydown 照常上抛：不拦就是中文还没选完词树已按旧词跳走
        const w = mountBar({ query: 'ab' });
        await w.find('input').trigger('keydown', { key: 'Enter', isComposing: true });
        expect(w.emitted('step')).toBeUndefined();
    });

    it('Escape 就地消费并上抛 collapse（不得冒泡到页面级快捷键）', async () => {
        const w = mountBar();
        // 以行为为断言口径：页面级监听（全屏退出）收得到这次 Escape 就是双触发
        let reachedDocument = false;
        const onDocumentKeydown = () => (reachedDocument = true);
        document.addEventListener('keydown', onDocumentKeydown);
        try {
            await w.find('input').trigger('keydown', { key: 'Escape' });
        } finally {
            document.removeEventListener('keydown', onDocumentKeydown);
        }

        expect(w.emitted('collapse')).toHaveLength(1);
        expect(reachedDocument).toBe(false);
    });

    it('未跳转时只报总数，跳转后报位序', () => {
        expect(mountBar({ query: 'ab', total: 3 }).find('.tree-find-count').text()).toBe('3');
        expect(mountBar({ query: 'ab', total: 3, no: 2 }).find('.tree-find-count').text()).toBe('2/3');
    });

    it('有查找词但零命中时标记无匹配态', () => {
        expect(mountBar({ query: 'ab', total: 0 }).find('.tree-find-bar').classes()).toContain('is-no-match');
        expect(mountBar({ query: '', total: 0 }).find('.tree-find-bar').classes()).not.toContain('is-no-match');
    });

    it('计数未与引擎生效值对齐时宁可不显示（不报防抖窗口里的旧值/零命中）', () => {
        const stale = mountBar({ query: 'ab', total: 3, countReady: false });
        expect(stale.find('.tree-find-count').exists()).toBe(false);
        expect(stale.find('.tree-find-bar').classes()).not.toContain('is-no-match');
    });

    it('展开即调用输入框的 focus()', async () => {
        // 用桩件替掉真 ElInput：断的是「收起→展开这次变化会调 el-input 暴露的 focus()」这条自有契约，
        // 去抓 EP 内部的聚焦类名会把断言建在组件库实现细节上（版本一升就静默失真）
        const focus = vi.fn();
        const InputStub = defineComponent({
            setup: (_props: Record<string, unknown>, { expose }) => {
                expose({ focus });
                return () => h('input');
            },
        });
        const w = mount(TreeFindBar, {
            props: { visible: false, query: '', total: 0, no: 0 },
            global: { mocks: { $t: (k: string) => k }, components: { ElInput: InputStub } },
        });

        await w.setProps({ visible: true });
        expect(focus).toHaveBeenCalledTimes(1);

        // 反向对照：没再发生「收起→展开」就不重复抢焦点（否则用户每次回车跳转都会被掉回输入框）
        await w.setProps({ query: 'ab' });
        expect(focus).toHaveBeenCalledTimes(1);
        w.unmount();
    });
});
