import { describe, expect, it } from 'vitest';
import { resolveEscapeTarget, writeback, type DraftAnchor, type DraftEditorHandle } from '../draftEditor';

/** 最小编辑器替身：只实现写回路径真正用到的方法 */
function fakeEditor(options: { versionId?: number; position?: { lineNumber: number; column: number } | null } = {}) {
    const edits: { range: unknown; text: string }[] = [];
    const handle: DraftEditorHandle = {
        getModel: () => ({
            uri: { toString: () => 'inmemory://model/1' },
            getVersionId: () => options.versionId ?? 7,
            getLineCount: () => 3,
            getLineMaxColumn: () => 20,
        }),
        getPosition: () => (options.position === undefined ? { lineNumber: 2, column: 5 } : options.position),
        executeEdits: (_source, list) => {
            edits.push(...list);
            return true;
        },
        focus: () => undefined,
    };
    return { handle, edits };
}

describe('SQL 草稿写回', () => {
    it('锚点版本一致时按锚点位置写入', () => {
        const { handle, edits } = fakeEditor({ versionId: 7 });
        const anchor: DraftAnchor = { uri: 'inmemory://model/1', versionId: 7, range: null };
        expect(writeback(handle, anchor, 'select 1;')).toBeNull();
        expect(edits).toHaveLength(1);
        expect(edits[0].text).toBe('select 1;');
    });

    it('内容在生成期间被改动时拒绝写入（stale），一个字符都不落盘', () => {
        const { handle, edits } = fakeEditor({ versionId: 8 });
        const anchor: DraftAnchor = { uri: 'inmemory://model/1', versionId: 7, range: null };
        expect(writeback(handle, anchor, 'select 1;')).toBe('stale');
        expect(edits).toHaveLength(0);
    });

    it('模型标识变化（切了 tab）同样拒绝写入', () => {
        const { handle, edits } = fakeEditor({ versionId: 7 });
        const anchor: DraftAnchor = { uri: 'inmemory://model/2', versionId: 7, range: null };
        expect(writeback(handle, anchor, 'select 1;')).toBe('stale');
        expect(edits).toHaveLength(0);
    });

    it('有选区快照时替换该选区，而不是插入到光标处', () => {
        const { handle, edits } = fakeEditor({ versionId: 7, position: { lineNumber: 1, column: 1 } });
        const range = { startLineNumber: 2, startColumn: 1, endLineNumber: 2, endColumn: 10 };
        expect(writeback(handle, { uri: 'inmemory://model/1', versionId: 7, range }, 'select 2;')).toBeNull();
        expect(edits[0].range).toEqual(range);
    });

    it('编辑器已销毁时返回 disposed 而不是抛错', () => {
        expect(writeback(null, null, 'select 1;')).toBe('disposed');
    });

    it('拿不到光标位置时退化为「文件末尾插入」而不是失败', () => {
        const { handle, edits } = fakeEditor({ versionId: 7, position: null });
        expect(writeback(handle, null, 'select 1;')).toBeNull();
        expect(edits[0].range).toEqual({ startLineNumber: 3, startColumn: 20, endLineNumber: 3, endColumn: 20 });
    });
});

describe('Esc 的让位优先级', () => {
    const state = (over: Partial<Parameters<typeof resolveEscapeTarget>[0]> = {}) => ({
        modalOpen: false,
        findWidgetOpen: false,
        barVisible: false,
        ...over,
    });

    it('什么都没开时谁也不处理', () => {
        expect(resolveEscapeTarget(state())).toBe('none');
    });

    it('模态弹层在时 Esc 归弹层，不能把背后的面板卸掉', () => {
        expect(resolveEscapeTarget(state({ modalOpen: true, barVisible: true }))).toBe('modal');
        expect(resolveEscapeTarget(state({ modalOpen: true, findWidgetOpen: true }))).toBe('modal');
    });

    it('查找条可见时 Esc 先收查找条（monaco 的既定语义）', () => {
        expect(resolveEscapeTarget(state({ findWidgetOpen: true, barVisible: true }))).toBe('findWidget');
    });

    it('助手面板在时 Esc 收面板', () => {
        expect(resolveEscapeTarget(state({ barVisible: true }))).toBe('bar');
    });
});
