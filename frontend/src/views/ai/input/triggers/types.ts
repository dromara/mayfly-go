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
    /** 菜单项种类（决定选中后插入的芯片类型；开放为字符串，新增触发器无需改本契约） */
    kind?: string;
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
 * - 面板模式：panelComponent + onPanelSelect，由 TriggerOverlays 动态渲染该组件。
 *
 * 面板模式也要求把「选中载荷 → 芯片」的转换交给注册表：否则新增一个面板触发类型
 * 就得改 TriggerOverlays 模板与处理器（此前就是这样：panelComponent 只被当布尔用，
 * 选回调硬写成了 resource 专属的 selectResource）。
 */
export interface TriggerDef {
    /** 触发器种类标识（决定插入的 ChipNode type；开放字符串以支持新增触发器） */
    kind: string;
    /** 触发字符（含全角变体，如中文输入法下的 ＠ ／） */
    chars: string[];
    /** hint 栏 i18n key（输入框底部动态渲染快捷键提示） */
    hintI18nKey: string;
    /** 列表模式：按查询词构建菜单项 */
    buildItems?: (query: string, ctx: TriggerContext) => TriggerMenuItem[];
    /** 面板模式组件（如 ResourceTreePanel） */
    panelComponent?: Component;
    /** 列表项选中 → 插入的芯片（缺省按 item.kind/data 推导） */
    onSelect?: (item: TriggerMenuItem) => TriggerSelection;
    /** 面板模式选中 → 插入的芯片（与 panelComponent 配对，缺省即视为该面板不支持选中） */
    onPanelSelect?: (payload: TriggerPanelSelectPayload) => TriggerSelection;
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
