/**
 * 触发器菜单运行时状态（对齐 tokhub triggers/useTriggerState）
 *
 * 聚合触发器注册表，提供菜单开合、触发词检测（含 IME/粘贴兜底路径）、
 * 键盘导航、选中插入芯片等能力；由 ChatInput 消费，编辑器侧不再出现
 * 具体触发类型的分支逻辑。
 */

import { computed, nextTick, ref } from 'vue';
import type { EditorLike } from './types';
import type { ChipNodeAttrs } from '../chipNode';
import type { SkillItem } from '../types';
import { buildTriggerTokenRegex, findTriggerDef, getTriggerChars } from './registry';
import { useMenuPosition } from './useMenuPosition';
import type { TriggerMenuItem, TriggerPanelSelectPayload, TriggerSelection } from './types';

export interface UseTriggerStateOptions {
    getEditor: () => EditorLike | null;
    /** 浮层根 DOM（列表菜单 expose menuRef / 面板组件 expose panelRef） */
    getMenuEl: () => HTMLElement | null | undefined;
    /** 菜单打开且无匹配项时按 Enter/Tab 放行发送，避免吞键导致无法提交 */
    onSubmit: () => void;
    /** 可用技能列表（传入 buildItems 上下文） */
    skills: () => SkillItem[];
}

export function useTriggerState(options: UseTriggerStateOptions) {
    const { getEditor, getMenuEl, onSubmit } = options;

    const visible = ref(false);
    const char = ref('');
    const query = ref('');
    const items = ref<TriggerMenuItem[]>([]);
    const selectedIndex = ref(0);
    /** Escape 关闭时记录的光标前触发词，避免兜底检测立即重新打开；变更后重置 */
    let dismissedToken = '';

    /** 当前激活的触发器定义 */
    const activeDef = computed(() => findTriggerDef(char.value));
    /** 面板模式（如资源树面板）不走通用列表菜单与键盘选中 */
    const isPanelMode = computed(() => !!activeDef.value?.panelComponent);

    // 浮层定位（floating-ui + autoUpdate）
    const position = useMenuPosition();

    const getCaretCoords = () => {
        const editor = getEditor();
        if (!editor) return { left: 0, top: 0, bottom: 0 };
        const { from } = editor.state.selection;
        return editor.view.coordsAtPos(from);
    };

    const updateItems = (q: string) => {
        const ctx = { query: q, skills: options.skills() };
        items.value = activeDef.value?.buildItems ? activeDef.value.buildItems(q, ctx) : [];
        selectedIndex.value = 0;
    };

    const reposition = () => position.update(getCaretCoords, getMenuEl());

    const open = (trigger: string, q = '') => {
        visible.value = true;
        char.value = trigger;
        query.value = q;
        updateItems(q);
        // 菜单 v-if 渲染完成后再定位，否则拿不到浮层 DOM
        nextTick(reposition);
    };

    const close = () => {
        position.stop();
        visible.value = false;
        query.value = '';
    };

    /** Escape/点击外部关闭（记录当前触发词防兜底检测立即重开） */
    const dismiss = () => {
        dismissedToken = getTokenAtCursor();
        close();
    };

    /** 光标前的触发词（触发字符 + 查询词，如 '/sk'、'@web'），无则返回空串 */
    const getTokenAtCursor = (): string => {
        const editor = getEditor();
        if (!editor) return '';
        const { state } = editor;
        const { from } = state.selection;
        const textBefore = state.doc.textBetween(Math.max(0, from - 50), from, '\n');
        return buildTriggerTokenRegex().exec(textBefore)?.[1] || '';
    };

    /**
     * 触发词检测（onUpdate 调用）：
     * 未打开时兜底检测打开菜单——不依赖 handleTextInput（IME 组合输入/粘贴等
     * 路径不会触发），任何输入路径都能打开菜单；
     * 已打开时跟踪最近触发字符切换与空格关闭。
     */
    const checkState = (editor: EditorLike) => {
        const { state } = editor;
        const { from } = state.selection;
        const textBefore = state.doc.textBetween(Math.max(0, from - 50), from, '\n');

        if (!visible.value) {
            const match = buildTriggerTokenRegex().exec(textBefore);
            if (match) {
                // 用户刚 Escape 关闭且光标前仍是同一触发词时不重开，删除/修改后恢复
                if (match[1] !== dismissedToken) {
                    dismissedToken = '';
                    open(match[1][0], match[1].slice(1));
                }
            } else {
                dismissedToken = '';
            }
            return;
        }

        // 已打开：查找最近的触发字符（含全角变体，取更靠近光标者）
        const chars = getTriggerChars();
        let lastIdx = -1;
        for (const ch of chars) {
            const i = textBefore.lastIndexOf(ch);
            if (i > lastIdx) lastIdx = i;
        }
        if (lastIdx === -1) {
            close();
            return;
        }

        // 触发字符与当前菜单不一致时切换（如先 '/' 后紧跟 '@'）
        const foundChar = textBefore[lastIdx];
        if (foundChar !== char.value) {
            char.value = foundChar;
            selectedIndex.value = 0;
        }

        const q = textBefore.slice(lastIdx + 1);
        // 如果中间有空格或换行，关闭菜单
        if (/[\s\n]/.test(q)) {
            close();
            return;
        }

        query.value = q;
        updateItems(q);
        reposition();
    };

    /** 删除光标前的触发字符与查询文本 */
    const deleteTriggerText = () => {
        const editor = getEditor();
        if (!editor) return;
        const { state } = editor;
        const { from } = state.selection;
        const textBefore = state.doc.textBetween(Math.max(0, from - 50), from, '\n');
        const triggerIdx = textBefore.lastIndexOf(char.value);
        if (triggerIdx >= 0) {
            const deleteFrom = from - (textBefore.length - triggerIdx);
            editor.chain().focus().deleteRange({ from: deleteFrom, to: from }).run();
        }
    };

    /** 插入触发器选中结果对应的芯片（后跟空格分隔后续输入） */
    const insertSelection = (selection: TriggerSelection) => {
        const editor = getEditor();
        if (!editor) return;
        editor.chain().focus().insertChip(selection.chip).insertContent(' ').run();
    };

    /** 列表模式选中：删除触发词后按注册表 onSelect 插入芯片 */
    const selectItem = (index: number) => {
        const item = items.value[index];
        const def = activeDef.value;
        if (!item || !def) return;

        deleteTriggerText();

        const selection = def.onSelect
            ? def.onSelect(item)
            : ({
                  chip: { type: def.kind, label: item.label, data: item.data || {} },
              } as TriggerSelection);
        insertSelection(selection);

        close();
    };

    /** 面板模式选中（如资源树叶子节点）：删除触发词后插入芯片 */
    const selectPanelChip = (payload: TriggerPanelSelectPayload, toSelection: (p: TriggerPanelSelectPayload) => TriggerSelection) => {
        deleteTriggerText();
        insertSelection(toSelection(payload));
        close();
    };

    /** 菜单打开时的键盘拦截（返回 true 表示已消费，IME 安全） */
    const handleKeydown = (event: KeyboardEvent): boolean => {
        // 面板模式：无键盘选中项，Escape 关闭、Enter/Tab 关闭并放行发送，其余输入继续过滤
        if (isPanelMode.value) {
            switch (event.key) {
                case 'Escape':
                    event.preventDefault();
                    dismiss();
                    return true;
                case 'Enter':
                case 'Tab':
                    event.preventDefault();
                    close();
                    onSubmit();
                    return true;
                default:
                    return false;
            }
        }
        switch (event.key) {
            case 'ArrowDown':
                event.preventDefault();
                selectedIndex.value = Math.min(selectedIndex.value + 1, items.value.length - 1);
                return true;
            case 'ArrowUp':
                event.preventDefault();
                selectedIndex.value = Math.max(selectedIndex.value - 1, 0);
                return true;
            case 'Enter':
            case 'Tab':
                event.preventDefault();
                if (items.value.length > 0) {
                    selectItem(selectedIndex.value);
                } else {
                    // 无匹配项时放行发送，避免吞掉 Enter 导致无法提交
                    close();
                    onSubmit();
                }
                return true;
            case 'Escape':
                event.preventDefault();
                dismiss();
                return true;
            default:
                return false;
        }
    };

    return {
        /** 菜单是否可见 */
        visible,
        /** 当前触发字符（'/' 技能 / '@' 资源） */
        char,
        /** 查询词（触发字符后、光标前） */
        query,
        /** 列表模式菜单项 */
        items,
        /** 键盘选中索引 */
        selectedIndex,
        /** 浮层定位样式 */
        menuStyle: position.style,
        /** 当前激活的触发器定义 */
        activeDef,
        /** 是否面板模式（替代通用列表菜单） */
        isPanelMode,
        open,
        close,
        dismiss,
        /** onUpdate 触发词检测 */
        checkState,
        getTokenAtCursor,
        /** 菜单打开时的键盘拦截 */
        handleKeydown,
        selectItem,
        selectPanelChip,
    };
}

export type TriggerState = ReturnType<typeof useTriggerState>;

/** 兼容导出：芯片 attrs 类型（供触发器实现引用） */
export type { ChipNodeAttrs };
