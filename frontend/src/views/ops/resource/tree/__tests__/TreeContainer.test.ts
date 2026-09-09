/**
 * TreeContainer 挂载回归：复现「展开→水合→数据重建」链路下 el-tree-v2 虚拟列表在
 * vue 3.6-beta 下的补丁崩溃（Cannot set properties of null (setting '__vnode')）。
 */
import { mount } from '@vue/test-utils';
import { defineComponent, h, nextTick } from 'vue';
import { ElTreeV2 } from 'element-plus';
import { describe, expect, it, vi, beforeEach } from 'vitest';

import { registerContributor } from '../registry';
import { registerCommand, registerMenu } from '../commands';
import TreeContainer from '../TreeContainer.vue';
import type { TreeNodeData } from '../types';

// @/components 依赖链隔离
vi.mock('@/i18n', () => ({ i18n: { global: { t: (k: string) => k, locale: 'zh-cn' } } }));
vi.mock('@/hooks/useI18n', () => ({ useI18n: () => ({ t: (k: string) => k }) }));
vi.mock('@/components/auth/auth', () => ({ hasPerm: vi.fn(() => true), hasPerms: vi.fn() }));
vi.mock('@/components/contextmenu', async () => {
    const item = await import('@/components/contextmenu/item');
    return { ...item, Contextmenu: defineComponent({ name: 'ContextmenuStub', setup: () => () => h('div') }) };
});

let seq = 0;
const makeChildren = (parent: string, kind: string) =>
    Array.from({ length: 3 }, () => ({ key: `${parent}-c${++seq}`, kind, label: 'leaf' }) as TreeNodeData);

beforeEach(() => {
    registerContributor({
        kind: 'root',
        hasChildren: true,
        loadRoots: async () => makeChildren('root', 'mid'),
    });
    registerContributor({
        kind: 'mid',
        hasChildren: true,
        loadChildren: async (n) => makeChildren(n.key, 'leaf'),
    });
    registerContributor({ kind: 'leaf' });
    registerCommand({ id: 't.cmd', txt: 't', handler: vi.fn() });
    registerMenu({ command: 't.cmd', kinds: ['leaf'], when: (ctx) => !!ctx.node });
});

describe('TreeContainer 挂载与水合', () => {
    it('根加载后展开触发水合且不抛补丁错误', async () => {
        const patchErrors: unknown[] = [];
        const err = vi.spyOn(console, 'error').mockImplementation((...args: unknown[]) => {
            patchErrors.push(args);
        });

        const wrapper = mount(TreeContainer, {
            props: {
                loadRoot: async () => [{ key: 'top1', kind: 'root', label: 'top' }],
                interactive: false,
            },
            global: {
                components: { ElTreeV2 },
                config: {
                    globalProperties: { $t: (k: string) => k } as any,
                    errorHandler: (_err) => {
                        patchErrors.push(_err);
                    },
                },
            },
        });
        await nextTick();
        await nextTick();
        expect(wrapper.exists()).toBe(true);

        // 模拟树内部展开事件：容器收到后执行水合并重建数据
        const treeV2 = wrapper.findComponent(ElTreeV2);
        expect(treeV2.exists()).toBe(true);
        treeV2.vm.$emit('node-expand', { key: 'top1', kind: 'root', label: 'top', params: {}, hasChildren: true });
        await nextTick();
        await nextTick();
        await nextTick();

        wrapper.unmount();

        const fatal = patchErrors.filter((c) => String(c).includes('__vnode') || String(c).includes('emitsOptions'));
        err.mockRestore();
        expect(fatal).toEqual([]);
    });

    it('交互模式：水合后行渲染与 when 判定正常（回归：ctxPayload 误传 ref 导致渲染期崩溃）', async () => {
        const patchErrors: unknown[] = [];
        const err = vi.spyOn(console, 'error').mockImplementation((...args: unknown[]) => {
            patchErrors.push(args);
        });

        const wrapper = mount(TreeContainer, {
            props: {
                loadRoot: async () => [{ key: 'top1', kind: 'root', label: 'top' }],
            },
            global: {
                components: { ElTreeV2 },
                config: {
                    globalProperties: { $t: (k: string) => k } as any,
                    errorHandler: (_err) => {
                        patchErrors.push(_err);
                    },
                },
            },
        });
        await nextTick();
        await nextTick();

        const treeV2 = wrapper.findComponent(ElTreeV2);
        treeV2.vm.$emit('node-expand', { key: 'top1', kind: 'root', label: 'top', params: {}, hasChildren: true });
        await nextTick();
        await nextTick();
        await nextTick();

        // 水合完成：DOM 出现叶子行（渲染链路完整未被中断）
        expect(wrapper.html()).toContain('leaf');

        // 叶子行挂着注册的菜单（when 读取 node.params 不抛错）
        expect(wrapper.findComponent({ name: 'TreeNodeRow' }).exists()).toBe(true);

        wrapper.unmount();

        const fatal = patchErrors.filter((c) => String(c).includes('__vnode') || String(c).includes('emitsOptions') || String(c).includes('TypeError'));
        err.mockRestore();
        expect(fatal).toEqual([]);
    });
});
