import { hasPerm } from '@/components/auth/auth';
import { ContextmenuItem } from '@/components/contextmenu';

import type { TreeApi, TreeCommandCtx } from './types';

/**
 * 命令/菜单注册表：操作与节点类型解耦（VS Code commands+menus 模式）。
 * - registerCommand 定义命令（文案/图标/handler）
 * - registerMenu 声明命令挂在哪些节点类型上（when 条件 + 权限 + 分组排序 + 触发方式）
 * - 容器右键菜单与节点悬浮按钮共用 resolveNodeMenu 单源，可见性判定单点收口
 * - 菜单项按 kind 缓存；when 以 isHide 惰性求值（openContextmenu 传入 ctx），渲染期零开销
 */

export interface TreeCommand {
    id: string;
    /** i18n key */
    txt: string;
    icon?: string;
    handler: (ctx: TreeCommandCtx) => void | Promise<void>;
}

export type CommandTrigger = 'menu' | 'click' | 'dblclick';

export interface MenuEntry {
    command: string;
    /** 挂载的节点类型 */
    kinds: string[];
    /** 分组（同组按 order 排，未声明按注册顺序） */
    group?: string;
    order?: number;
    permission?: string;
    /** 显示条件（返回 true 显示；语义与旧 hideFunc 相反） */
    when?: (ctx: TreeCommandCtx) => boolean;
    trigger?: CommandTrigger;
}

const commands = new Map<string, TreeCommand>();
const menuEntries: MenuEntry[] = [];
const menuCache = new Map<string, ContextmenuItem[]>();

export function registerCommand(cmd: TreeCommand) {
    if (import.meta.env.DEV && commands.has(cmd.id)) {
        console.warn(`[tree] 命令重复注册: ${cmd.id}，将覆盖已有命令`);
    }
    commands.set(cmd.id, cmd);
}

export function registerMenu(entry: MenuEntry) {
    menuEntries.push(entry);
}

function sortedEntries(trigger: CommandTrigger, kind: string): MenuEntry[] {
    return menuEntries
        .map((e, i) => ({ e, seq: i }))
        .filter((x) => (x.e.trigger ?? 'menu') === trigger && x.e.kinds.includes(kind))
        .sort((a, b) => (a.e.group ?? '').localeCompare(b.e.group ?? '') || (a.e.order ?? a.seq) - (b.e.order ?? b.seq))
        .map((x) => x.e);
}

/** 节点右键/悬浮菜单项（按 kind 缓存；permission/when 均以 isHide 惰性求值，payload 为 TreeCommandCtx） */
export function resolveNodeMenu(kind: string): ContextmenuItem[] {
    let items = menuCache.get(kind);
    if (!items) {
        items = sortedEntries('menu', kind)
            // 防御：菜单挂了未注册命令时跳过（DEV 由 validateMenuCommands 显式告警，不在渲染期抛错）
            .filter((entry) => commands.has(entry.command))
            .map((entry) => {
                const cmd = commands.get(entry.command)!;
                const item = new ContextmenuItem(entry.command, cmd.txt);
                if (cmd.icon) {
                    item.withIcon(cmd.icon);
                }
                // permission 与 when 均惰性求值：权限随用户态实时判定（旧 BaseTreeNode 语义），不因首次渲染时权限未就绪被缓存永久隐藏
                item.withHideFunc((payload) => {
                    if (entry.permission && !hasPerm(entry.permission)) {
                        return true;
                    }
                    return entry.when ? !entry.when(payload as TreeCommandCtx) : false;
                });
                // 缓存的菜单项不绑定具体节点：点击时从 payload（{ node, tree }）取上下文分发
                item.withOnClick((payload) => {
                    commands.get(entry.command)?.handler(payload as TreeCommandCtx);
                });
                return item;
            });
        menuCache.set(kind, items);
    }
    return items;
}

/**
 * 注册表完整性自检（DEV 启动期调用，对标 VS Code contribution validation）：
 * 菜单挂了未注册命令会静默丢菜单项（拼写错误难发现），此处显式告警暴露
 * @returns 问题描述列表（空数组表示注册表完整）
 */
export function validateMenuCommands(): string[] {
    const problems: string[] = [];
    const seen = new Set<string>();
    for (const entry of menuEntries) {
        if (!commands.has(entry.command) && !seen.has(entry.command)) {
            seen.add(entry.command);
            problems.push(`菜单挂载引用了未注册的命令: ${entry.command}`);
        }
    }
    if (import.meta.env.DEV) {
        problems.forEach((p) => console.warn(`[tree] ${p}`));
    }
    return problems;
}

/** 单击/双击触发命令（无则返回 undefined；触发时求值 when/permission） */
export function findTriggerCommand(node: TreeCommandCtx['node'], tree: TreeApi, trigger: 'click' | 'dblclick'): TreeCommand | undefined {
    const entry = sortedEntries(trigger, node.kind).find((e) => (!e.permission || hasPerm(e.permission)) && (!e.when || e.when({ node, tree })));
    return entry ? commands.get(entry.command) : undefined;
}

/** 测试辅助：清空注册表 */
export function resetCommandsForTest() {
    commands.clear();
    menuEntries.length = 0;
    menuCache.clear();
}
