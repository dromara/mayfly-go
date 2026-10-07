/**
 * 聚合域：管道文本的执行与结果。
 *
 * 管道始终以文本为本源（而不是前端拼好的数组）：stage 的顺序就是执行顺序，
 * `$match` 放前面与放后面差的是几个数量级的扫描量，前端重排一次数组就把用户的写法改掉了。
 *
 * explain 与真执行走同一个入口：区别只在权限（explain 不执行任何 stage，含 $out 也只按只读放行），
 * 这个判断由服务端做，前端只负责把开关传下去并说明结果含义。
 */
import { reactive } from 'vue';

import { i18n } from '@/i18n';
import { mongoApi } from '../../api';
import type { CollectionParam, MongoDocPage } from '../../types';
import { parseJsonArray } from '../../docview/json';

/** 起手模板：先筛再聚合，比从空白写少一次「管道不能为空」的报错 */
export const DEFAULT_PIPELINE = '[\n    { "$match": {} },\n    { "$group": { "_id": "$status", "count": { "$sum": 1 } } }\n]';

interface AggState {
    pipelineText: string;
    allowDiskUse: boolean;
    explain: boolean;
    loading: boolean;
    error: string;
    /** 结果：与普通查询同构（docs/total/truncated/limit），因此可直接复用表格视图 */
    page: MongoDocPage | null;
}

export function useAggregation() {
    const state = reactive<AggState>({
        pipelineText: DEFAULT_PIPELINE,
        allowDiskUse: false,
        explain: false,
        loading: false,
        error: '',
        page: null,
    });

    async function run(target: CollectionParam) {
        const parsed = parseJsonArray(state.pipelineText, 'mongo.pipelineInvalid');
        if (!parsed.ok) {
            state.error = i18n.global.t(parsed.issue.key);
            return false;
        }

        state.loading = true;
        state.error = '';
        try {
            state.page = await mongoApi.aggregate.request({
                ...target,
                pipeline: parsed.value,
                allowDiskUse: state.allowDiskUse,
                explain: state.explain,
            });
            return true;
        } catch (e) {
            // 请求层已 toast 后端文案（管道语法错、缺内存、权限不足各有说法），这里留一份就地说明
            state.error = e instanceof Error ? e.message : '';
            state.page = null;
            return false;
        } finally {
            state.loading = false;
        }
    }

    /** 打开对话框：每个集合一份新管道，上一集合的 stage 拿到这里多半只是误导 */
    function reset() {
        state.pipelineText = DEFAULT_PIPELINE;
        state.error = '';
        state.page = null;
    }

    return { state, run, reset };
}
