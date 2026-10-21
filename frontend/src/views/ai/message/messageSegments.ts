/**
 * 消息分段分组（AssistantMessageBubble 的渲染段构建）。
 *
 * 为什么单独成模块：分段结果直接决定 Vue 的 key 稳定性。此前用索引做 key——
 * 流式中途插入一段文本会把原来的过程段拆开，索引位移导致 Vue 复用旧实例，
 * 把上一组的展开态带过来（视觉闪动）。这里统一给每段一个由 part.id 派生的稳定 key，
 * 并作为纯函数单测覆盖，避免退回索引 key。
 */
import type { CompactionPart, MessagePart, NoticePart } from '../protocol/types';
import { partMeta } from '../protocol/types';

export type RenderSegment =
    | { type: 'text'; key: string; text: string }
    | { type: 'process'; key: string; parts: MessagePart[] }
    | { type: 'compaction'; key: string; part: CompactionPart }
    | { type: 'notice'; key: string; part: NoticePart };

/** 过程段判定走 PART_META（单一真源）：新增一种过程段只需登记表 */
function isProcessPart(part: MessagePart): boolean {
    return partMeta(part.type)?.isProcess ?? false;
}

/** 连续 process parts 合并为一个 process 段；其余类型各自成段，顺序不变 */
export function groupPartsIntoSegments(parts: MessagePart[]): RenderSegment[] {
    const segments: RenderSegment[] = [];
    let processBuf: MessagePart[] = [];

    const flushProcess = () => {
        if (processBuf.length > 0) {
            // key 取组内首个 part.id：同一组内容重算后 key 不变；组被拆开时两段 key 天然不同，
            // Vue 不会拿旧实例承载新内容
            segments.push({ type: 'process', key: `process:${processBuf[0].id}`, parts: [...processBuf] });
            processBuf = [];
        }
    };

    for (const part of parts) {
        if (isProcessPart(part)) {
            processBuf.push(part);
        } else if (part.type === 'context_compaction') {
            flushProcess();
            segments.push({ type: 'compaction', key: `compaction:${part.id}`, part });
        } else if (part.type === 'notice') {
            flushProcess();
            segments.push({ type: 'notice', key: `notice:${part.id}`, part });
        } else {
            flushProcess();
            if (part.type === 'text' && part.text?.trim()) {
                segments.push({ type: 'text', key: `text:${part.id}`, text: part.text });
            }
        }
    }
    flushProcess();
    return segments;
}
