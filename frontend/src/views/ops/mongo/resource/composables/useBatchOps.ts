/**
 * 批量操作域：按条件批量更新/删除，含「先预览命中数」这道护栏。
 *
 * 预览不是装饰：服务端会把统计结果与提交的 expectCount 比对，不一致即中止。
 * 少了这一步，一个写错的 filter 会静默改掉整个集合且无法回退。
 * 因此预览值必须原样传给执行接口，中途改了条件就要重新预览（这里靠清空预览值强制）。
 */
import { reactive } from 'vue';

import { i18n } from '@/i18n';
import { mongoApi } from '../../api';
import type { CollectionParam, MongoWriteResult } from '../../types';
import { parseJsonObject, parseUpdateSpec } from '../../docview/json';

export type BatchMode = 'update' | 'delete';

interface BatchState {
    mode: BatchMode;
    filterText: string;
    updateText: string;
    upsert: boolean;
    /** 预览进行中 */
    previewing: boolean;
    submitting: boolean;
    /** 预览到的命中数；null 表示未预览或条件已改动需要重新预览 */
    previewed: number | null;
    error: string;
    /** 执行结果（用于说明实际影响条数：命中数不等于改动手数） */
    result: MongoWriteResult | null;
}

export function useBatchOps() {
    const state = reactive<BatchState>({
        mode: 'update',
        filterText: '{}',
        updateText: '{\n    "$set": { "status": "paid" }\n}',
        upsert: false,
        previewing: false,
        submitting: false,
        previewed: null,
        error: '',
        result: null,
    });

    /** 打开面板：带上当前查询条件作为起点，用户通常就是想改刚查出来的那一批 */
    function open(mode: BatchMode, filterText: string) {
        state.mode = mode;
        state.filterText = filterText?.trim() || '{}';
        state.error = '';
        state.result = null;
        state.previewed = null;
    }

    /** 条件一改，旧预览值即失效：否则就是「对 5 条做的决策作用到 5 万条上」 */
    function onFilterChange(text: string) {
        state.filterText = text;
        state.previewed = null;
    }

    /** 解析 filter 文本；空条件按 `{}` 处理（全集合），非法即报错，绝不退化成无条件全改 */
    function readFilter(): Record<string, unknown> | null {
        const text = state.filterText.trim();
        if (!text || text === '{}') {
            return {};
        }
        const parsed = parseJsonObject(text, 'mongo.conditionInvalid');
        if (!parsed.ok) {
            state.error = i18n.global.t(parsed.issue.key, { field: i18n.global.t('mongo.filter') });
            return null;
        }
        return parsed.value;
    }

    /** 预览命中数 */
    async function preview(target: CollectionParam) {
        state.error = '';
        const filter = readFilter();
        if (!filter) {
            return false;
        }

        state.previewing = true;
        try {
            const page = await mongoApi.queryDocs.request({ ...target, filter, limit: 1, withCount: true });
            state.previewed = page.total ?? 0;
            return true;
        } catch (e) {
            state.error = e instanceof Error ? e.message : '';
            return false;
        } finally {
            state.previewing = false;
        }
    }

    /**
     * 组装提交载荷，任一校验不过即返回 null 并把原因留在 state.error。
     *
     * 「未预览不能执行」是这里唯一的硬性顺序约束：服务端会拿 expectCount 与实时命中数比对，
     * 少了这道护栏，一个写错的 filter 会静默改掉整个集合。
     */
    function readPayload(): { filter: Record<string, unknown>; update?: unknown; expectCount: number } | null {
        state.error = '';
        if (state.previewed === null) {
            state.error = i18n.global.t('mongo.batchNeedPreview');
            return null;
        }
        if (state.previewed === 0) {
            // 命中 0 条时服务端不会写任何东西（expectCount 也要求非零），在这里拦下比等一次 400 明白
            state.error = i18n.global.t('mongo.batchNothingHit');
            return null;
        }

        const filter = readFilter();
        if (!filter) {
            return null;
        }

        // 预览值一并带出：它就是服务端要比对的 expectCount
        const expectCount = state.previewed;
        if (state.mode !== 'update') {
            return { filter, expectCount };
        }

        const parsed = parseUpdateSpec(state.updateText);
        if (!parsed.ok) {
            state.error = i18n.global.t(parsed.issue.key);
            return null;
        }
        return { filter, update: parsed.value, expectCount };
    }

    /**
     * 提交前预校验。
     *
     * 调用方必须在弹「确定继续？」之前先跑它：先让用户做完决定、再告诉他写不了，
     * 等于把一次无效确认压给他。
     */
    function validateSubmit(): boolean {
        return readPayload() !== null;
    }

    /** 执行批量操作 */
    async function submit(target: CollectionParam) {
        const payload = readPayload();
        if (!payload) {
            return false;
        }
        const { filter, update, expectCount } = payload;

        state.submitting = true;
        state.result = null;
        try {
            state.result =
                state.mode === 'update'
                    ? await mongoApi.updateByFilter.request({ ...target, filter, update, upsert: state.upsert, expectCount })
                    : await mongoApi.deleteByFilter.request({ ...target, filter, expectCount });
            return true;
        } catch (e) {
            // 命中数被并发改动时服务端会在这一跳拒绝，此时必须重新预览而不是照旧提交
            state.error = e instanceof Error ? e.message : '';
            state.previewed = null;
            return false;
        } finally {
            state.submitting = false;
        }
    }

    return { state, open, onFilterChange, preview, validateSubmit, submit };
}
