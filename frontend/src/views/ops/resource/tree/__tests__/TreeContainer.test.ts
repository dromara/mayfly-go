/**
 * TreeContainer 挂载回归：复现「展开→水合→数据重建」链路下 el-tree-v2 虚拟列表在
 * vue 3.6-beta 下的补丁崩溃（Cannot set properties of null (setting '__vnode')）。
 */
import { mount } from '@vue/test-utils';
import { defineComponent, h, nextTick } from 'vue';
import { ElTreeV2 } from 'element-plus';
import { describe, expect, it, vi, beforeEach } from 'vitest';

import { VirtualTree } from '@/components/virtual-tree';
import { registerContributor } from '../registry';
import { registerCommand, registerMenu } from '../commands';
import TreeContainer from '../TreeContainer.vue';
import type { TreeNodeData } from '../types';

// @/components 依赖链隔离
vi.mock('@/i18n', () => ({ i18n: { global: { t: (k: string) => k, locale: 'zh-cn' } } }));
vi.mock('@/hooks/useI18n', () => ({ useI18n: () => ({ t: (k: string) => k }) }));
vi.mock('@/components/auth/auth', () => ({ hasPerm: vi.fn(() => true), hasPerms: vi.fn() }));
// 桩件要暴露 closeContextmenu：容器在每次 node-click 里都会调它（真实组件 expose 了该方法）
vi.mock('@/components/contextmenu', async () => {
    const item = await import('@/components/contextmenu/item');
    return {
        ...item,
        Contextmenu: defineComponent({
            name: 'ContextmenuStub',
            setup: (_props: Record<string, unknown>, ctx: { expose: (v: Record<string, unknown>) => void }) => {
                ctx.expose({ closeContextmenu: () => undefined });
                return () => h('div');
            },
        }),
    };
});

let seq = 0;
const makeChildren = (parent: string, kind: string) => Array.from({ length: 3 }, () => ({ key: `${parent}-c${++seq}`, kind, label: 'leaf' }));

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

describe('TreeContainer 单击行的展开语义', () => {
    /**
     * 挂了单击命令（打开操作面板）的可展开节点，命令不能吞掉展开。
     *
     * 没有该命令的资源（DB/Redis 实例）点行本来就能展开，两类资源行为不一致时，
     * 用户点 mongo 实例只会看到一块空面板，误判成「点了没反应/展不开」。
     */
    it('命令执行的同时展开节点，且已展开时再单击不折叠', async () => {
        // 双击判定用的是 Date.now() 时间窗，需假时钟才能跨过 300ms 而不真的等
        vi.useFakeTimers();
        const handler = vi.fn();
        registerCommand({ id: 'clickable.open', txt: '', handler });
        registerMenu({ command: 'clickable.open', kinds: ['mid'], trigger: 'click' });

        const wrapper = mount(TreeContainer, {
            props: {
                loadRoot: async () => [{ key: 'c1', kind: 'mid', label: 'inst', params: {}, hasChildren: true }],
                interactive: true,
            },
            global: {
                config: { globalProperties: { $t: (k: string) => k } as any },
            },
        });
        await nextTick();
        await nextTick();

        const engine = wrapper.findComponent(VirtualTree);
        expect(engine.exists()).toBe(true);
        const node = { key: 'c1', kind: 'mid', label: 'inst', params: {}, hasChildren: true };

        expect(engine.props('expandedKeys') as string[]).not.toContain('c1');
        engine.vm.$emit('node-click', node);
        await nextTick();
        await nextTick();
        expect(handler).toHaveBeenCalledTimes(1);
        expect(engine.props('expandedKeys') as string[]).toContain('c1');

        // 第二次单击已越过双击判定窗口：仍只展开、不折叠（收起归箭头与双击管）
        vi.advanceTimersByTime(400);
        engine.vm.$emit('node-click', node);
        await nextTick();
        await nextTick();
        expect(handler).toHaveBeenCalledTimes(2);
        expect(engine.props('expandedKeys') as string[]).toContain('c1');

        wrapper.unmount();
        vi.useRealTimers();
    });

    it('无子节点的节点单击只执行命令，不产生展开状态', async () => {
        const handler = vi.fn();
        registerCommand({ id: 'leaf.open', txt: '', handler });
        registerMenu({ command: 'leaf.open', kinds: ['leaf'], trigger: 'click' });

        const wrapper = mount(TreeContainer, {
            props: {
                loadRoot: async () => [{ key: 'l1', kind: 'leaf', label: 'table', params: {}, hasChildren: false }],
                interactive: true,
            },
            global: { config: { globalProperties: { $t: (k: string) => k } as any } },
        });
        await nextTick();
        await nextTick();

        const engine = wrapper.findComponent(VirtualTree);
        engine.vm.$emit('node-click', { key: 'l1', kind: 'leaf', label: 'table', params: {}, hasChildren: false });
        await nextTick();
        await nextTick();
        expect(handler).toHaveBeenCalledTimes(1);
        expect(engine.props('expandedKeys') as string[]).not.toContain('l1');
        wrapper.unmount();
    });
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

describe('TreeContainer 查找接线', () => {
    /**
     * 计数与高亮都靠容器这一层接线（expose matchedKeys / provide 生效查询词），接错不会编译报错，
     * 表现是「计数永远为 0」或「整棵树都不标色」。
     *
     * 同时钉住「只认引擎已生效的词」：防抖未到点时派生 UI 不得抢跑，
     * 否则会出现已标色/已计数但该行还没被过滤掉，此时 Enter 跳转还会静默跳空。
     */
    const mountFiltered = async () => {
        const wrapper = mount(TreeContainer, {
            props: {
                loadRoot: async () => [
                    { key: 'a', kind: 'leaf', label: 't_order' },
                    { key: 'b', kind: 'leaf', label: 't_user' },
                ],
                filterText: '',
            },
            global: { components: { ElTreeV2 }, config: { globalProperties: { $t: (k: string) => k } as any } },
        });
        await nextTick();
        await nextTick();
        return wrapper;
    };

    it('防抖生效前不抢跑，生效后同步更新 matchedKeys 与命中片段标色', async () => {
        vi.useFakeTimers();
        const wrapper = await mountFiltered();

        await wrapper.setProps({ filterText: 'order' });
        expect(wrapper.vm.matchedKeys).toEqual([]);
        expect(wrapper.findAll('.tree-label-hit')).toHaveLength(0);

        // 防抖到点：VirtualTree 把词打到引擎并上报 filterApplied，容器据此重算
        vi.advanceTimersByTime(300);
        await nextTick();

        expect(wrapper.vm.matchedKeys).toEqual(['a']);
        const hits = wrapper.findAll('.tree-label-hit');
        expect(hits).toHaveLength(1);
        expect(hits[0].text()).toBe('order');

        // 自算行序必须与真引擎的 flatten 对得上：分叉时 locate 会发 DEV 告警并跳过滚动（就是上轮「计数在涨、视图不动」的形状）
        const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
        await wrapper.vm.locate('a');
        await nextTick();
        expect(warn.mock.calls.map((c) => String(c[0])).join('\n')).not.toContain('不在自算可见行序内');
        warn.mockRestore();

        wrapper.unmount();
        vi.useRealTimers();
    });

    it('宿主注入的视图级判定参与命中计数与引擎过滤（类型筛选与搜索同一口径）', async () => {
        vi.useFakeTimers();
        const wrapper = mount(TreeContainer, {
            props: {
                loadRoot: async () => [
                    { key: 'a', kind: 'leaf', label: 't_order' },
                    { key: 'b', kind: 'other', label: 'order_db' },
                ],
                filterText: '',
                // 只看 kind=leaf：b 文本上也命中，但不属于当前筛选类型
                nodeFilter: (node: { kind?: unknown }) => node.kind === 'leaf',
            },
            global: { components: { ElTreeV2 }, config: { globalProperties: { $t: (k: string) => k } as any } },
        });
        await nextTick();
        await nextTick();

        await wrapper.setProps({ filterText: 'order' });
        vi.advanceTimersByTime(300);
        await nextTick();

        expect(wrapper.vm.matchedKeys).toEqual(['a']);

        wrapper.unmount();
        vi.useRealTimers();
    });
});
