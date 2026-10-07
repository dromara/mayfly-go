/**
 * 数据进出口域：按条件导出文件、从文件导入文档。
 *
 * 两条链路都不走「当前页结果」：导出由服务端按同一份 filter 重新取数（有硬上限），
 * 导入按批提交并在中途失败时如实报已写入条数——把「部分成功」说成成功，
 * 用户就会以为文件里那一千条都进去了。
 */
import { reactive } from 'vue';

import { i18n } from '@/i18n';
import { mongoApi } from '../../api';
import type { CollectionParam, ExportFormat } from '../../types';
import { chunk, IMPORT_BATCH_SIZE, parseImportText, type ImportIssue } from '../../io/parse';
import { downloadTextFile } from '../../io/download';

interface ImportProgress {
    done: number;
    total: number;
}

interface IoState {
    format: ExportFormat;
    exporting: boolean;
    importing: boolean;
    /** 上一次导出的条数（等于服务端上限即被截断，界面要说明） */
    exported: number | null;
    /** 待导入文档与解析失败行 */
    docs: Record<string, unknown>[];
    issues: ImportIssue[];
    /** 当前选中的文件名，让用户知道即将导入的是刚才那一份 */
    fileName: string;
    progress: ImportProgress;
    error: string;
}

export function useDocIO() {
    const state = reactive<IoState>({
        format: 'json',
        exporting: false,
        importing: false,
        exported: null,
        docs: [],
        issues: [],
        fileName: '',
        progress: { done: 0, total: 0 },
        error: '',
    });

    /**
     * 导出。
     *
     * filter 由调用方给出（当前条件或选中行的主键集合）：条数上限由服务端配置决定，
     * 前端不传 limit，避免「以为能导全量」。
     */
    async function exportDocs(target: CollectionParam, filter: unknown) {
        state.exporting = true;
        state.error = '';
        state.exported = null;
        try {
            const file = await mongoApi.exportDocs.request({ ...target, filter, format: state.format });
            downloadTextFile(file.fileName, file.content, file.contentType);
            state.exported = file.count;
            return true;
        } catch (e) {
            state.error = e instanceof Error ? e.message : '';
            return false;
        } finally {
            state.exporting = false;
        }
    }

    /** 读取文件并解析；返回可导入的条数，解析问题只记录不抛错 */
    async function readImportFile(file: File) {
        state.error = '';
        state.docs = [];
        state.issues = [];
        state.fileName = file.name;
        state.progress = { done: 0, total: 0 };

        const text = await file.text();
        const parsed = parseImportText(text);
        state.docs = parsed.docs;
        state.issues = parsed.issues;
        state.progress = { done: 0, total: parsed.docs.length };
        return parsed;
    }

    function resetImport() {
        state.docs = [];
        state.issues = [];
        state.fileName = '';
        state.progress = { done: 0, total: 0 };
        state.error = '';
    }

    /**
     * 分批导入。
     *
     * 返回已写入条数：中途某一批失败（唯一索引冲突、文档过大）时前面的批次其实已经落库，
     * 只有如实报出这个数，调用方才能给出「已导入 n/m」而不是「导入失败」。
     */
    async function importDocs(target: CollectionParam) {
        if (!state.docs.length) {
            return 0;
        }

        state.importing = true;
        state.error = '';
        let inserted = 0;
        const batches = chunk(state.docs, IMPORT_BATCH_SIZE);

        try {
            for (const batch of batches) {
                const res = await mongoApi.insertDocs.request({ ...target, docs: batch });
                // 有序插入可能在批内中断，实际写入条数以服务端回包为准
                inserted += res?.insertedCount ?? 0;
                state.progress = { done: Math.min(inserted, state.docs.length), total: state.docs.length };
            }
            return inserted;
        } catch (e) {
            state.error = e instanceof Error ? e.message : i18n.global.t('mongo.importFailed');
            return inserted;
        } finally {
            state.importing = false;
        }
    }

    return { state, exportDocs, readImportFile, resetImport, importDocs };
}
