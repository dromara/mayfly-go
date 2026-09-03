/**
 * 技能触发器（对齐 tokhub triggers/skillTrigger）
 *
 * / 触发技能选择菜单（列表模式），选中后插入 skill 芯片。
 *
 * 注意：不在本文件顶层调用 registerTrigger 自注册——会与 registry 形成
 * 循环依赖（TDZ），注册统一由 registry 底部调用 registerSkillTrigger 完成。
 */

import { registerTrigger } from './registry';
import type { TriggerDef, TriggerMenuItem, TriggerSelection } from './types';
import type { SkillItem } from '../types';

/** 按查询词过滤技能并构建菜单项 */
export function buildSkillItems(skills: SkillItem[], query: string): TriggerMenuItem[] {
    return (skills || [])
        .filter((s) => !query || s.name.toLowerCase().includes(query))
        .map((s) => ({
            id: s.id,
            label: s.name,
            description: s.description,
            icon: s.icon,
            kind: 'skill' as const,
            data: s as unknown as Record<string, unknown>,
        }));
}

/** 列表项选中 → 插入的 skill 芯片 */
export function selectSkillItem(item: TriggerMenuItem): TriggerSelection {
    return {
        chip: { type: 'skill', label: item.label, data: item.data || {} },
    };
}

/** 技能触发器定义 */
export function createSkillTrigger(): TriggerDef {
    return {
        kind: 'skill',
        chars: ['/', '／'],
        buildItems: (query, ctx) => buildSkillItems(ctx.skills, query),
        onSelect: selectSkillItem,
    };
}

// 供 registry 底部调用的注册入口（函数体调用时 registry 已初始化，无 TDZ）
export function registerSkillTrigger(): void {
    registerTrigger(createSkillTrigger());
}
