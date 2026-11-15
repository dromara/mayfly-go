/**
 * 编辑器适配层：读材料、抓锚点、写回，以及 Esc 该归谁。
 *
 * 这里是唯一接触 monaco 的文件，其余逻辑都做成纯函数。抓锚点与写回刻意声明成
 * 结构化窄接口（只列出真正用到的方法）：monaco 的编辑器实例天然满足它，而单测
 * 可以传一个最小替身把「锚点失效就不写」这条判定钉住，无需拉起真编辑器。
 */
import type { editor } from 'monaco-editor';
import { getCurrentStatement, getSqlSplitOptions } from '../sql-editor/utils/sqlParser';
import type { DraftEditorContext } from './draftIntents';

/** 采集到的编辑器材料 */
export interface EditorMaterials extends DraftEditorContext {
    selection: string;
}

export interface DraftRange {
    startLineNumber: number;
    startColumn: number;
    endLineNumber: number;
    endColumn: number;
}

/** 写回所需的最小模型能力 */
export interface DraftModelHandle {
    uri: { toString(): string };
    getVersionId(): number;
    getLineCount(): number;
    getLineMaxColumn(lineNumber: number): number;
}

/** 写回所需的最小编辑器能力（monaco 的 IStandaloneCodeEditor 结构上满足） */
export interface DraftEditorHandle {
    getModel(): DraftModelHandle | null;
    getPosition(): { lineNumber: number; column: number } | null;
    executeEdits(source: string, edits: { range: DraftRange; text: string; forceMoveMarkers?: boolean }[]): unknown;
    focus(): void;
}

/**
 * 生成结果与编辑器内容的绑定锚点。
 *
 * 为什么必须有：模型调用是异步的（数秒），期间用户可能继续编辑甚至切库。回来后若
 * 直接写入，就会把 SQL 插到一个已经变了的语句中间，而界面上看不出任何异常。
 * 仓库里 formatSql 已经踩过同类问题（内容变了就丢弃结果），这里用同样的判据。
 */
export interface DraftAnchor {
    uri: string;
    versionId: number;
    /** 发起时的选区快照（用于「替换选区」；空选区时插入到光标处） */
    range: DraftRange | null;
}

/** 读取编辑器材料（选区 + 光标所在完整语句） */
export function readEditorMaterials(editorInstance: editor.IStandaloneCodeEditor | null, dbType?: string): EditorMaterials {
    const model = editorInstance?.getModel();
    const position = editorInstance?.getPosition();
    if (!model || !position) {
        return { selection: '', statement: '', hasSelection: false };
    }
    const selectionRange = editorInstance?.getSelection();
    const selection = selectionRange && !selectionRange.isEmpty() ? model.getValueInRange(selectionRange) : '';
    const statement = getCurrentStatement(model.getValue(), position, model, getSqlSplitOptions(dbType)) ?? '';
    return { selection, statement, hasSelection: selection.trim().length > 0 };
}

/** 抓锚点：记录当前模型标识、版本号与选区 */
export function captureAnchor(editorInstance: editor.IStandaloneCodeEditor | null): DraftAnchor | null {
    const model = editorInstance?.getModel();
    if (!model) {
        return null;
    }
    const selectionRange = editorInstance?.getSelection();
    const range =
        selectionRange && !selectionRange.isEmpty()
            ? {
                  startLineNumber: selectionRange.startLineNumber,
                  startColumn: selectionRange.startColumn,
                  endLineNumber: selectionRange.endLineNumber,
                  endColumn: selectionRange.endColumn,
              }
            : null;
    return { uri: model.uri.toString(), versionId: model.getVersionId(), range };
}

/** 写回失败原因 */
export type WritebackFailure = 'disposed' | 'stale';

/**
 * 把生成结果写回编辑器（可撤销）。
 *
 * 语义：有选区快照时替换该选区，否则插入到当前光标处。两种情况下都不做「整篇覆盖」——
 * 覆盖会连带清掉用户没选中的其它语句，那是不可挽回的破坏。
 */
export function writeback(editorInstance: DraftEditorHandle | null, anchor: DraftAnchor | null, text: string): WritebackFailure | null {
    const model = editorInstance?.getModel();
    if (!editorInstance || !model) {
        return 'disposed';
    }
    if (anchor) {
        // 版本号不一致说明期间有人动过内容：此时按旧坐标写入会插到错误位置
        if (model.uri.toString() !== anchor.uri || model.getVersionId() !== anchor.versionId) {
            return 'stale';
        }
    }
    const position = editorInstance.getPosition();
    const lastLine = model.getLineCount();
    const range: DraftRange = anchor?.range ?? {
        startLineNumber: position?.lineNumber ?? lastLine,
        startColumn: position?.column ?? model.getLineMaxColumn(lastLine),
        endLineNumber: position?.lineNumber ?? lastLine,
        endColumn: position?.column ?? model.getLineMaxColumn(lastLine),
    };
    editorInstance.executeEdits('sql-draft', [{ range, text, forceMoveMarkers: true }]);
    editorInstance.focus();
    return null;
}

/** Esc 该归谁：一次按键只能被一处消费，优先级写在一处才能说清「为什么是这个顺序」 */
export type EscapeOwner = 'modal' | 'findWidget' | 'bar' | 'none';

/**
 * 按下 Escape 时应由谁处理。
 *
 * 优先级背面的理由：
 *  - 模态弹层（提单抽屉、详情面板）在最上层，Esc 必须先关它，否则用户在弹层里按 Esc 却把背后浮层卸了；
 *  - monaco 的查找条可见时，Esc 的既定语义是收起查找条；
 *  - 最后才轮到 SQL 助手面板。
 */
export function resolveEscapeTarget(state: { modalOpen: boolean; findWidgetOpen: boolean; barVisible: boolean }): EscapeOwner {
    if (state.modalOpen) {
        return 'modal';
    }
    if (state.findWidgetOpen) {
        return 'findWidget';
    }
    if (state.barVisible) {
        return 'bar';
    }
    return 'none';
}
