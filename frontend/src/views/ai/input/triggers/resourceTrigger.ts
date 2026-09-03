/**
 * 资源触发器（对齐 tokhub triggers/groupMemberTrigger 的面板特化模式）
 *
 * @ 触发资源选择（面板模式：ResourceTreePanel 替代通用列表菜单，
 * 因资源按机器/数据库分层且需懒加载），叶子选中后插入 resource 芯片。
 *
 * 注意：不在本文件顶层调用 registerTrigger 自注册——会与 registry 形成
 * 循环依赖（TDZ），注册统一由 registry 底部调用 registerResourceTrigger 完成。
 */

import { registerTrigger } from './registry';
import type { TriggerDef, TriggerPanelSelectPayload, TriggerSelection } from './types';
import ResourceTreePanel from './ResourceTreePanel.vue';

/** 树面板叶子选中 → 插入的 resource 芯片（extra 携带完整定位标识） */
export function selectResourceFromPanel(payload: TriggerPanelSelectPayload): TriggerSelection {
    return {
        chip: { type: 'resource', label: payload.label, data: payload.extra },
    };
}

/** 资源触发器定义（面板模式） */
export function createResourceTrigger(): TriggerDef {
    return {
        kind: 'resource',
        chars: ['@', '＠'],
        panelComponent: ResourceTreePanel,
    };
}

// 供 registry 底部调用的注册入口（函数体调用时 registry 已初始化，无 TDZ）
export function registerResourceTrigger(): void {
    registerTrigger(createResourceTrigger());
}
