// 隔离 pinia（auth.ts）与 reka-ui（index.vue）依赖，仅保留纯逻辑模型
vi.mock('@/components/auth/auth', () => ({
    hasPerm: vi.fn((code: string) => code === 'granted:perm'),
    hasPerms: vi.fn(),
}));

vi.mock('@/components/contextmenu', async () => {
    const item = await import('@/components/contextmenu/item');
    return { ...item };
});

import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { TreeCommandCtx, TreeNode } from '../types';
import { findTriggerCommand, registerCommand, registerMenu, resetCommandsForTest, resolveNodeMenu, validateMenuCommands } from '../commands';

const makeNode = (kind: string, key = kind, params?: Record<string, unknown>): TreeNode => ({ key, kind, label: key, hasChildren: false, params: params ?? {} });
const makeCtx = (node: TreeNode): TreeCommandCtx => ({ node, tree: { locate: async () => {}, refresh: () => {}, getNode: () => undefined } });

describe('tree/commands 命令与菜单注册表', () => {
    beforeEach(() => {
        resetCommandsForTest();
        vi.clearAllMocks();
    });

    it('resolveNodeMenu 只返回挂载到对应 kind 的菜单项', () => {
        registerCommand({ id: 'a.cmd1', txt: 'cmd1', handler: vi.fn() });
        registerCommand({ id: 'a.cmd2', txt: 'cmd2', handler: vi.fn() });
        registerMenu({ command: 'a.cmd1', kinds: ['k1'] });
        registerMenu({ command: 'a.cmd2', kinds: ['k2'] });

        expect(resolveNodeMenu('k1').map((i) => i.clickId)).toEqual(['a.cmd1']);
        expect(resolveNodeMenu('k2').map((i) => i.clickId)).toEqual(['a.cmd2']);
        expect(resolveNodeMenu('k3')).toEqual([]);
    });

    it('group/order 排序：同组按 order，组间按组名字典序', () => {
        registerCommand({ id: 'c', txt: 'c', handler: vi.fn() });
        registerCommand({ id: 'a', txt: 'a', handler: vi.fn() });
        registerCommand({ id: 'b', txt: 'b', handler: vi.fn() });
        registerMenu({ command: 'c', kinds: ['k'], group: 'b', order: 1 });
        registerMenu({ command: 'a', kinds: ['k'], group: 'a', order: 9 });
        registerMenu({ command: 'b', kinds: ['k'], group: 'a', order: 1 });

        expect(resolveNodeMenu('k').map((i) => i.clickId)).toEqual(['b', 'a', 'c']);
    });

    it('permission 未授权的菜单项在展示判定时被隐藏（惰性求值，随用户态实时判定）', () => {
        registerCommand({ id: 'p1', txt: 'p1', handler: vi.fn() });
        registerCommand({ id: 'p2', txt: 'p2', handler: vi.fn() });
        registerMenu({ command: 'p1', kinds: ['k'], permission: 'granted:perm' });
        registerMenu({ command: 'p2', kinds: ['k'], permission: 'denied:perm' });

        // 菜单项保留（权限不在注册期过滤），展示时经 isHide 隐藏未授权项
        const items = resolveNodeMenu('k');
        expect(items.map((i) => i.clickId)).toEqual(['p1', 'p2']);
        const ctx = makeCtx(makeNode('k'));
        expect(items[0].isHide(ctx)).toBe(false);
        expect(items[1].isHide(ctx)).toBe(true);
    });

    it('when 以 hideFunc 惰性求值，菜单点击分发到命令 handler', () => {
        const handler = vi.fn();
        registerCommand({ id: 'w', txt: 'w', handler });
        registerMenu({ command: 'w', kinds: ['k'], when: (ctx) => ctx.node.params?.show === true });

        const items = resolveNodeMenu('k');
        const shown = makeCtx(makeNode('k', 'k1', { show: true }));
        const hidden = makeCtx(makeNode('k', 'k2', { show: false }));
        expect(items[0].isHide(shown)).toBe(false);
        expect(items[0].isHide(hidden)).toBe(true);

        items[0].onClickFunc!(shown);
        expect(handler).toHaveBeenCalledWith(shown);
    });

    it('findTriggerCommand 按触发方式过滤并求值 when/permission', () => {
        const handler = vi.fn();
        registerCommand({ id: 't1', txt: 't1', handler });
        registerCommand({ id: 't2', txt: 't2', handler });
        registerMenu({ command: 't1', kinds: ['k'], trigger: 'click' });
        registerMenu({ command: 't2', kinds: ['k'], trigger: 'dblclick', when: (ctx) => ctx.node.key === 'ok' });

        expect(findTriggerCommand(makeNode('k'), makeCtx(makeNode('k')).tree, 'click')?.id).toBe('t1');
        expect(findTriggerCommand(makeNode('k', 'ok'), makeCtx(makeNode('k', 'ok')).tree, 'dblclick')?.id).toBe('t2');
        expect(findTriggerCommand(makeNode('k', 'no'), makeCtx(makeNode('k', 'no')).tree, 'dblclick')).toBeUndefined();
        // 右键菜单不受 click/dblclick 声明影响
        expect(resolveNodeMenu('k')).toEqual([]);
    });

    it('validateMenuCommands 揪出挂了未注册命令的菜单项；渲染期防御性跳过不抛错', () => {
        const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
        registerCommand({ id: 'good.cmd', txt: 'ok', handler: vi.fn() });
        registerMenu({ command: 'good.cmd', kinds: ['k'] });
        registerMenu({ command: 'ghost.cmd', kinds: ['k'] });

        const problems = validateMenuCommands();
        expect(problems).toHaveLength(1);
        expect(problems[0]).toContain('ghost.cmd');
        expect(warn).toHaveBeenCalled();
        // 悬空菜单不进入渲染项（不抛错不崩树），正常菜单不受影响
        expect(resolveNodeMenu('k').map((i) => i.clickId)).toEqual(['good.cmd']);
    });

    it('注册表完整时 validateMenuCommands 静默无告警', () => {
        const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
        registerCommand({ id: 'only.cmd', txt: 'ok', handler: vi.fn() });
        registerMenu({ command: 'only.cmd', kinds: ['k'] });
        expect(validateMenuCommands()).toEqual([]);
        expect(warn).not.toHaveBeenCalled();
    });
});
