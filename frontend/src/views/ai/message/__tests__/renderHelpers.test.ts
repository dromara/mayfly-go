import { describe, expect, it } from 'vitest';
import { groupPartsIntoSegments } from '../messageSegments';
import { truncateByCodePoints } from '../text';
import { buildCardAttachments } from '../attachmentCards';
import type { ContentSegment, MessageAttachment, MessagePart } from '../../protocol/types';

const text = (id: string, value: string): MessagePart => ({ type: 'text', id, text: value });
const reasoning = (id: string): MessagePart => ({ type: 'reasoning', id, text: '思考' });
const tool = (id: string, callId: string): MessagePart => ({
    type: 'tool_call',
    id,
    toolCallId: callId,
    toolName: 'DbQueryTables',
    arguments: '{}',
    output: 'ok',
    status: 'success',
});

describe('groupPartsIntoSegments', () => {
    it('连续过程段合并、文本/压缩/提示各自成段且顺序不变', () => {
        const segments = groupPartsIntoSegments([
            reasoning('r1'),
            tool('t1', 'c1'),
            text('x1', '正文'),
            { type: 'context_compaction', id: 'c1', originalTokens: 100, compressedTokens: 50, compressedMessageCount: 4 },
            { type: 'notice', id: 'n1', kind: 'stopped', text: '已停止' },
        ]);
        expect(segments.map((s) => s.type)).toEqual(['process', 'text', 'compaction', 'notice']);
        expect(segments[0].type === 'process' && segments[0].parts).toHaveLength(2);
    });

    it('每段 key 由 part.id 派生且互不相同（不得退回索引 key）', () => {
        const segments = groupPartsIntoSegments([text('x1', 'a'), text('x2', 'b'), reasoning('r1')]);
        const keys = segments.map((s) => s.key);
        expect(new Set(keys).size).toBe(keys.length);
        expect(keys).toContain('text:x1');
        expect(keys).toContain('text:x2');
        expect(keys).toContain('process:r1');
    });

    it('中途插入文本会把过程组拆成两段，且两段 key 不同（不会复用旧实例的展开态）', () => {
        const before = groupPartsIntoSegments([reasoning('r1'), tool('t1', 'c1')]);
        const after = groupPartsIntoSegments([reasoning('r1'), text('x1', '插入'), tool('t1', 'c1')]);
        expect(before).toHaveLength(1);
        expect(after).toHaveLength(3);
        const processKeys = after.filter((s) => s.type === 'process').map((s) => s.key);
        expect(processKeys).toEqual(['process:r1', 'process:t1']);
        expect(new Set(after.map((s) => s.key)).size).toBe(3);
    });

    it('空白正文段的文本段不生成（避免空 Markdown 块）', () => {
        expect(groupPartsIntoSegments([text('x1', '   ')])).toEqual([]);
    });
});

describe('truncateByCodePoints', () => {
    it('按码点截断，不劈开 emoji 代理对', () => {
        const value = 'a'.repeat(79) + '😀' + 'b';
        const out = truncateByCodePoints(value, 80);
        // 81 个码点超限：保留前 80 个（第 80 个是 emoji 本身）并补省略号
        expect(out).toBe('a'.repeat(79) + '😀' + '\u2026');
        // 输出不得含孤立代理项（旧实现 slice(0,80) 只拿到 emoji 的高位）
        expect(/[\uD800-\uDBFF](?![\uDC00-\uDFFF])|(?<![\uD800-\uDBFF])[\uDC00-\uDFFF]/.test(out)).toBe(false);
    });

    it('超出时补省略号，未超出原样返回', () => {
        expect(truncateByCodePoints('short', 80)).toBe('short');
        expect(truncateByCodePoints('x'.repeat(81), 80)).toBe('x'.repeat(80) + '\u2026');
    });
});

describe('buildCardAttachments', () => {
    const imageSeg = (fileKey: string, name: string): ContentSegment => ({
        type: 'image',
        text: '',
        extra: { fileKey, name },
    });
    const img = (fileKey: string, name: string, dataUrl?: string): MessageAttachment => ({
        kind: 'image',
        name,
        fileKey,
        ...(dataUrl ? { dataUrl } : {}),
    });

    it('image 段生成卡片，并带上同名本地预览', () => {
        const cards = buildCardAttachments([imageSeg('k1', 'a.png')], [img('k1', 'a.png', 'data:image/png;base64,x')], '图片');
        expect(cards).toHaveLength(1);
        expect(cards[0].dataUrl).toBe('data:image/png;base64,x');
    });

    it('旧消息：只有 attachments 没有 image 段时，图片也要显示（按 fileKey 去重）', () => {
        const legacyOnly = buildCardAttachments([], [img('k-legacy', 'old.png')], '图片');
        expect(legacyOnly.map((c) => c.fileKey)).toEqual(['k-legacy']);

        const both = buildCardAttachments([imageSeg('k1', 'a.png')], [img('k1', 'a.png'), img('k2', 'b.png')], '图片');
        expect(both.map((c) => c.fileKey)).toEqual(['k1', 'k2']);
    });

    it('文本/文件附件追加在图片之后，顺序稳定', () => {
        const cards = buildCardAttachments(
            [imageSeg('k1', 'a.png')],
            [img('k1', 'a.png'), { kind: 'text', name: 'log.txt', fileKey: 'k3' }, { kind: 'file', name: 'x.zip', fileKey: 'k4' }],
            '图片'
        );
        expect(cards.map((c) => c.kind)).toEqual(['image', 'text', 'file']);
    });
});
