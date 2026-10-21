/**
 * chipNode - TipTap 自定义内联原子节点
 *
 * 特性：
 * - inline: true — 内联在文本流中
 * - atom: true — 不可编辑的原子单元
 * - selectable: true — 可整体选中/删除
 * - contentEditable=false — 禁止内部编辑
 */
import { mergeAttributes, Node } from '@tiptap/core';
import { type DOMOutputSpec } from '@tiptap/pm/model';
import { chipIcon, FALLBACK_CHIP_COLOR, getChipType } from './chipRegistry';
// 副作用导入：芯片的行高固定（防药丸随 1.6 行框下坠）、图标 mask、选中轮廓等
// CSS 规则都在此文件；漏接此 import 时 renderHTML 内联样式仍在（芯片看着像个药丸），
// 但行高/图标 mask/轮廓全部缺失，芯片与文本错位（实测踩过）
import './chipStyles.scss';

export interface ChipNodeAttrs {
    type: string;
    label: string;
    data: Record<string, unknown>;
}

const CHIP_NODE_NAME = 'chip';

declare module '@tiptap/core' {
    interface Commands<ReturnType> {
        chip: {
            /** 插入一个芯片节点 */
            insertChip: (attrs: ChipNodeAttrs) => ReturnType;
        };
    }
}

export const ChipNode = Node.create({
    name: CHIP_NODE_NAME,
    group: 'inline',
    inline: true,
    atom: true,
    selectable: true,
    draggable: false,

    addAttributes() {
        return {
            type: {
                default: 'skill',
                parseHTML: (element) => element.getAttribute('data-chip-type'),
                renderHTML: (attributes) => ({ 'data-chip-type': attributes.type }),
            },
            label: {
                default: '',
                parseHTML: (element) => element.getAttribute('data-chip-label'),
                renderHTML: (attributes) => ({ 'data-chip-label': attributes.label }),
            },
            data: {
                default: {},
                parseHTML: (element) => {
                    const raw = element.getAttribute('data-chip-data');
                    try {
                        return raw ? JSON.parse(raw) : {};
                    } catch {
                        return {};
                    }
                },
                renderHTML: (attributes) => ({
                    'data-chip-data': JSON.stringify(attributes.data || {}),
                }),
            },
        };
    },

    parseHTML() {
        return [
            {
                tag: `span[data-chip-type]`,
            },
        ];
    },

    // 纯文本形态（getText/复制/落库指令文本）：由芯片类型注册表决定标记写法，
    // 未注册类型回落标签文本，保证文本里不会凭空少一截
    renderText({ node }) {
        const reg = getChipType(node.attrs.type);
        return reg ? reg.toText(node.attrs) : String(node.attrs.label || '');
    },

    renderHTML({ node, HTMLAttributes }) {
        const reg = getChipType(node.attrs.type);
        const color = reg?.color || FALLBACK_CHIP_COLOR;
        const data = (node.attrs.data || {}) as Record<string, unknown>;
        // 图标由注册表解析（资源子类型判定收在 chipTypes 注册处，与消息回显 InlineChip 同源）
        const icon = chipIcon(node.attrs.type, { label: String(node.attrs.label ?? ''), data });

        return [
            'span',
            mergeAttributes(HTMLAttributes, {
                class: 'chat-chip',
                contentEditable: 'false',
                // 内联只留「随注册表变化」的三项配色；几何（圆角/内距/字号/行高/flex）全部交回 .chat-chip。
                // 此前两处各写一份完整清单，而内联优先级恒高——chipStyles.scss 里除 line-height
                // 外全是永不生效的死规则（改 padding/字号界面不动，改行高才动，极易误判）
                style: `background:${color.bg};color:${color.text};border-color:${color.border};`,
            }),
            ['span', { class: 'chat-chip__icon', 'data-icon': icon }],
            node.attrs.label || node.attrs.type,
        ] as DOMOutputSpec;
    },

    addCommands() {
        return {
            insertChip:
                (attrs) =>
                ({ commands }) => {
                    return commands.insertContent({
                        type: CHIP_NODE_NAME,
                        attrs: {
                            type: attrs.type,
                            label: attrs.label,
                            data: attrs.data,
                        },
                    });
                },
        };
    },
});
