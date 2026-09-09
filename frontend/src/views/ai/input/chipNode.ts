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
import { Plugin, PluginKey } from '@tiptap/pm/state';
import { getChipType } from './chipRegistry';

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

    renderHTML({ node, HTMLAttributes }) {
        const reg = getChipType(node.attrs.type);
        const color = reg?.color || { bg: '#f0f0f0', text: '#333', border: '#ddd' };
        const data = (node.attrs.data || {}) as Record<string, unknown>;
        // 图标：resource 按子类型细分（机器/数据库），其余取注册图标（样式见 ChatInput 全局 .chat-chip__icon）
        const icon = node.attrs.type === 'resource' && data.resourceType === 'db' ? 'database' : reg?.icon || 'zap';

        return [
            'span',
            mergeAttributes(HTMLAttributes, {
                class: 'chat-chip',
                contentEditable: 'false',
                style: `background:${color.bg};color:${color.text};border:1px solid ${color.border};border-radius:12px;padding:1px 8px;font-size:12px;display:inline-flex;align-items:center;gap:4px;cursor:default;user-select:none;`,
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

    addProseMirrorPlugins() {
        return [
            new Plugin({
                key: new PluginKey('chipNodeProtection'),
                props: {
                    handleKeyDown: (_view, event) => {
                        // Backspace/Delete 选中芯片时，允许默认行为（删除原子节点）
                        if (event.key === 'Backspace' || event.key === 'Delete') {
                            return false;
                        }
                        return false;
                    },
                },
            }),
        ];
    },
});
