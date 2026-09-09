/**
 * 触发器子系统类型定义
 *
 * 可扩展的触发器框架：/ 技能、@ 资源，后续 # 文档引用等新触发类型
 * 只需实现 TriggerDef 并 registerTrigger 注册，编辑器侧检测/菜单/选中
 * 逻辑均由注册表驱动，无需改动 ChatInput 分支。
 */
import type { Component } from 'vue';
import type { ChipNodeAttrs } from '../chipNode';
import type { SkillItem } from '../types';

/**
 * 最小编辑器命令接口（结构化兼容 @tiptap/core 与 @tiptap/vue-3 两套 Editor 类型，
 * 避免双包类型互不相赋的冲突）
 */
export interface EditorCommands {
    focus(): EditorCommands;
    deleteRange(range: { from: number; to: number }): EditorCommands;
    insertChip(attrs: ChipNodeAttrs): EditorCommands;
    insertContent(content: string): EditorCommands;
    run(): boolean;
}

/** 触发器子系统依赖的最小编辑器能力 */
export interface EditorLike {
    state: {
        doc: { textBetween(from: number, to: number, blockSeparator: string): string };
        selection: { from: number };
    };
    /** 光标坐标（浮层定位用，ProseMirror EditorView 结构化子集） */
    view: {
        coordsAtPos(pos: number): { left: number; top: number; bottom: number };
    };
    chain(): EditorCommands;
}

/** 触发器菜单中的一个选项（TriggerItem） */
export interface TriggerMenuItem {
    id: string;
    label: string;
    description?: string;
    icon?: string;
    /** 菜单项种类（决定默认图标与选中后插入的芯片类型） */
    kind?: 'skill' | 'resource';
    /** 资源类型（kind=resource 时用于分组渲染：machine / db） */
    resourceType?: string;
    data?: Record<string, unknown>;
}

/** 触发器上下文（传递给 buildItems，TriggerContext） */
export interface TriggerContext {
    /** 当前查询词（触发字符之后、光标之前的文本） */
    query: string;
    /** 可用技能列表（父组件传入） */
    skills: SkillItem[];
}

/** 触发器选中结果：插入到编辑器的芯片（TriggerSelection） */
export interface TriggerSelection {
    chip: ChipNodeAttrs;
}

/**
 * 触发器定义（TriggerConfig）
 *
 * 两种呈现模式二选一：
 * - 列表模式：buildItems + onSelect，由通用 TriggerMenu 呈现；
 * - 面板模式：panelComponent（如资源树面板），列表项概念不存在。
 */
export interface TriggerDef {
    /** 触发器种类标识（决定插入的 ChipNode type） */
    kind: 'skill' | 'resource';
    /** 触发字符（含全角变体，如中文输入法下的 ＠ ／） */
    chars: string[];
    /** 列表模式：按查询词构建菜单项 */
    buildItems?: (query: string, ctx: TriggerContext) => TriggerMenuItem[];
    /** 面板模式组件（如 ResourceTreePanel） */
    panelComponent?: Component;
    /** 列表项选中 → 插入的芯片（缺省按 item.kind/data 推导） */
    onSelect?: (item: TriggerMenuItem) => TriggerSelection;
}

/**
 * 面板模式组件的选中事件载荷（面板组件需按此约定 emit('select') 与
 * defineExpose({ panelRef })——panelRef 为浮层根 DOM，供 floating-ui 定位）
 */
export interface TriggerPanelSelectPayload {
    label: string;
    description?: string;
    /** 芯片携带的定位元数据（透传 ChipNode data，发送时进 segment extra） */
    extra: Record<string, string>;
}
