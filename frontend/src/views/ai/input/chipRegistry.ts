/**
 * chipRegistry - 芯片类型注册表
 * 对齐 tokhub 的 chipRegistry.tsx
 *
 * 职责：芯片的视觉呈现（图标/配色）与数据提取（extractSegment）
 * 新增芯片类型只需调用 registerChipType()，零修改编辑器/消息渲染代码
 * （触发字符管理在 triggers/ 子系统，与呈现解耦）
 */
import type { ContentSegmentType } from '../protocol/types';

/** 芯片段：发送时提取的数据结构（type 对齐协议层 ContentSegmentType，与后端同形） */
export interface ChipSegment {
    type: ContentSegmentType;
    text: string;
    extra: Record<string, unknown>;
}

/** 芯片类型注册信息（消费方：chipNode 编辑器渲染 + InlineChip 消息回显，配色/图标同源） */
export interface ChipTypeRegistration {
    /** 芯片类型标识 */
    readonly type: string;
    /** 显示图标 SVG path 或组件 */
    readonly icon: string;
    /** 配色 */
    readonly color: {
        bg: string;
        text: string;
        border: string;
    };
    /** 从芯片节点提取段数据（发送时序列化） */
    extractSegment(nodeAttrs: Record<string, unknown>): ChipSegment;
}

const registry = new Map<string, ChipTypeRegistration>();

/** 注册芯片类型，返回卸载函数 */
export function registerChipType(reg: ChipTypeRegistration): () => void {
    registry.set(reg.type, reg);
    return () => registry.delete(reg.type);
}

/** 获取芯片类型注册 */
export function getChipType(type: string): ChipTypeRegistration | undefined {
    return registry.get(type);
}

/** 从文档节点提取所有段 */
export function extractSegmentsFromNode(node: { content: { forEach: (fn: (child: any) => void) => void } }): ChipSegment[] {
    const segments: ChipSegment[] = [];
    const doc = node.content;

    doc.forEach((block: any) => {
        block.content?.forEach((child: any) => {
            if (child.type.name === 'chip') {
                const chipType = child.attrs.type as string;
                const reg = getChipType(chipType);
                if (reg) {
                    segments.push(reg.extractSegment(child.attrs));
                }
            } else if (child.type.name === 'text' && child.text) {
                segments.push({
                    type: 'input_text' as ContentSegmentType,
                    text: child.text,
                    extra: {},
                });
            }
        });
    });

    // 合并相邻文本段
    const merged: ChipSegment[] = [];
    for (const seg of segments) {
        if (seg.type === 'input_text' && merged.length > 0) {
            const last = merged[merged.length - 1];
            if (last.type === 'input_text') {
                last.text += seg.text;
                continue;
            }
        }
        merged.push({ ...seg });
    }

    return merged.filter((s) => s.type !== 'input_text' || s.text.trim().length > 0);
}

