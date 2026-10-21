/**
 * usePromptEditor - 提示词编辑器内核（TipTap 文档 + 芯片节点 + 触发器浮层状态）
 *
 * 为什么抽出来：聊天输入框与「任务/触发器指令」表单需要同一个编辑能力（`/` 引用技能、
 * `@` 引用资源、原子芯片、段序列化）。留在 ChatInput 里就会出现第二套 TipTap 装配——
 * 加一个扩展或改一条键盘语义时很容易只改一处，两处行为静默分叉（芯片渲染、触发词检测
 * 已经在 chipNode/triggers 单点，装配层也必须同源才完整）。
 *
 * 分工边界：本 composable 只管「文档模型 + 触发器状态 + 段的提取与回填」；附件、发送/停止、
 * 入队提示等聊天专属交互仍由 ChatInput 持有。
 */
import Document from '@tiptap/extension-document';
import HardBreak from '@tiptap/extension-hard-break';
import Paragraph from '@tiptap/extension-paragraph';
import Placeholder from '@tiptap/extension-placeholder';
import Text from '@tiptap/extension-text';
import { Editor } from '@tiptap/vue-3';
import type { Editor as CoreEditor } from '@tiptap/core';
import { onBeforeUnmount, onMounted, reactive, ref, shallowRef, watch } from 'vue';
import { ChipNode } from './chipNode';
import { extractSegmentsFromNode, getChipType } from './chipRegistry';
// 副作用导入：注册 skill/resource 芯片类型与触发器（漏掉会让芯片在序列化时被静默丢弃）
import './chipTypes';
import './triggers';
import type { ChipSegment } from './chipRegistry';
import type { SkillItem } from './types';
import { useTriggerState } from './triggers';

export interface UsePromptEditorOptions {
    /** 占位文案（回调式：可随触发器注册表或入队态变化） */
    placeholder: () => string;
    /** 可引用技能列表（`/` 菜单数据源） */
    skills: () => SkillItem[];
    /** 是否只读 */
    disabled?: () => boolean;
    /** 挂载后自动聚焦 */
    autoFocus?: () => boolean;
    /** Enter 提交回调；不传则 Enter 走默认换行（表单里的提示词不该一键提交） */
    onSubmit?: () => void;
    /**
     * Enter 是否真的提交（回调式开关）。
     *
     * 为什么不能只靠「传不传 onSubmit」区分：同一个内核既要服务「气泡原地编辑」（Enter 即重发）
     * 也要服务表单提示词（Enter 必须换行），传了 onSubmit 就必须决定消不消费 Enter，
     * 光凭函数存在与否无法表达「现在这个场景不该提交」
     */
    submitOnEnter?: () => boolean;
    /**
     * Escape 回调（返回 true 表示已消费）。聊天输入框用它实现「Esc 停止生成」；
     * 表单里的提示词编辑器不传，Esc 就还是普通的退出/冒泡行为
     */
    onEscape?: () => boolean;
    /** 内容变化后的附加回调（ChatInput 用它同步 canSend） */
    onChange?: () => void;
    /** 粘贴拦截：返回 true 表示已消费（ChatInput 用它把粘贴文件转附件） */
    handlePaste?: (event: ClipboardEvent) => boolean;
}

export function usePromptEditor(options: UsePromptEditorOptions) {
    /** 编辑器挂载点（消费方放进模板） */
    const editorContainer = ref<HTMLElement | null>(null);
    const editor = shallowRef<Editor | null>(null);
    /** 浮层根 DOM：由 TriggerOverlays 组件回填，供触发菜单定位与点击外部判定 */
    const overlayEl = ref<HTMLElement | null>(null);
    /** 文档纯文本（TipTap 不是 Vue 响应式的，需手动跟踪） */
    const text = ref('');
    /** 文档里的芯片段（驱动「是否有内容可提交」判定） */
    const chips = ref<ChipSegment[]>([]);

    const isDisabled = () => options.disabled?.() ?? false;

    // 触发器状态（检测/键盘导航/定位/面板选中转换均由 triggers/ 注册表驱动，
    // 本内核不再持有具体触发类型的处理函数）
    const trigger = reactive(
        useTriggerState({
            getEditor: () => editor.value,
            getMenuEl: () => overlayEl.value,
            onSubmit: () => options.onSubmit?.(),
            skills: () => options.skills(),
        })
    );

    const syncDoc = (e: CoreEditor) => {
        text.value = e.getText();
        chips.value = extractSegmentsFromNode(e.state.doc).filter((s) => s.type !== 'input_text');
        trigger.checkState(e);
        options.onChange?.();
    };

    onMounted(() => {
        if (!editorContainer.value) return;

        editor.value = new Editor({
            element: editorContainer.value,
            extensions: [
                Document,
                Paragraph,
                Text,
                HardBreak.configure({ keepMarks: true }),
                ChipNode,
                Placeholder.configure({ placeholder: () => options.placeholder() }),
            ],
            content: '',
            editable: !isDisabled(),
            autofocus: options.autoFocus?.() ? 'end' : false,
            editorProps: {
                handleKeyDown: (_view, event) => {
                    // 触发菜单打开时键盘优先给菜单（上下选择/Tab 补全）
                    if (trigger.visible) {
                        return trigger.handleKeydown(event);
                    }
                    // Esc 交给消费方（如聊天里的「停止生成」），未消费则走默认行为
                    if (event.key === 'Escape' && options.onEscape?.()) {
                        event.preventDefault();
                        return true;
                    }
                    // Enter 提交仅在消费方声明时生效（聊天发送）；表单里保持默认换行。
                    // 未提供 submitOnEnter 开关时沿用旧语义（传了 onSubmit 就提交），避免聊天侧行为变动
                    const wantsSubmit = options.submitOnEnter ? options.submitOnEnter() : !!options.onSubmit;
                    if (event.key === 'Enter' && !event.shiftKey && wantsSubmit) {
                        event.preventDefault();
                        options.onSubmit?.();
                        return true;
                    }
                    return false;
                },
                handlePaste: (_view, event) => options.handlePaste?.(event) ?? false,
            },
            onUpdate: ({ editor: e }) => syncDoc(e),
        });
    });

    onBeforeUnmount(() => {
        editor.value?.destroy();
    });

    watch(isDisabled, (val) => {
        editor.value?.setEditable(!val);
    });

    /**
     * 编辑器是否已就绪（onMounted 创建完成）。
     *
     * 回填类操作必须先判就绪：setText/setSegments 在编辑器不存在时是静默 no-op，
     * 调用方若误以为成功就会把草稿信箱清空（草稿被吞，不留任何痕迹）
     */
    const ready = () => !!editor.value;

    /** 当前文档的段序列（发送/保存时提交给后端，芯片的定位信息在 extra 里） */
    const segments = (): ChipSegment[] => (editor.value ? extractSegmentsFromNode(editor.value.state.doc) : []);

    /**
     * 纯文本 → 编辑器 HTML。
     *
     * setContent(字符串) 按 HTML 解析，裸 \n 会被当普通空白折叠，使多行提示词挤成一行；
     * 先转义再写成 <br>（hardBreak 的 renderText 就是换行），界面与提交内容才一致
     */
    const plainTextToHtml = (value: string) => value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/\n/g, '<br>');

    const escapeAttr = (value: string) => value.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;');

    /**
     * 段序列 → 文档（编辑回填）。
     *
     * 无法还原的段（未知类型、缺定位信息）按纯文本降级显示：宁可让用户看到一个普通词，
     * 也不能造出一个点开却定位不到资源的假芯片
     */
    const setSegments = (segs: ChipSegment[]) => {
        if (!editor.value) return;
        const html = (segs ?? [])
            .map((seg) => {
                if (seg.type === 'input_text') return plainTextToHtml(seg.text);
                const attrs = getChipType(seg.type)?.fromSegment(seg);
                if (!attrs) return plainTextToHtml(seg.text);
                return `<span data-chip-type="${escapeAttr(attrs.type)}" data-chip-label="${escapeAttr(attrs.label)}" data-chip-data="${escapeAttr(JSON.stringify(attrs.data))}"></span>`;
            })
            .join('');
        editor.value.commands.setContent(html);
        syncDoc(editor.value);
    };

    /** 纯文本灌入（草稿预填、队列消息回填等无芯片场景） */
    const setText = (value: string) => {
        if (!editor.value) return;
        editor.value.commands.setContent(plainTextToHtml(value ?? ''));
        syncDoc(editor.value);
    };

    const clear = () => {
        editor.value?.commands.clearContent();
        chips.value = [];
        text.value = '';
        trigger.close();
    };

    /**
     * 聚焦编辑器。
     *
     * 光标位置必须调用方显式声明：不传 position 时只把焦点交给编辑器并**保留当前选区**，
     * 传 'end' 才会把光标挑到文末。此前固定 'end'，导致用户点正文中间时 ProseMirror
     * 刚定位好的光标被砸回末尾（表单与聊天输入框同时踩到，实测缺陷）
     */
    const focus = (position?: 'start' | 'end' | 'all' | number) => {
        editor.value?.commands.focus(position);
    };

    return {
        editorContainer,
        editor,
        overlayEl,
        text,
        chips,
        trigger,
        ready,
        segments,
        setSegments,
        setText,
        clear,
        focus,
    };
}
