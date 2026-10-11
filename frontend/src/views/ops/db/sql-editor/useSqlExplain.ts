import { ref } from 'vue';
import { dbApi } from '../api';
import type { SqlExplainPlan } from '../types';

/**
 * 执行计划取数的单一入口。
 *
 * 封装「按库 + SQL 逐条 EXPLAIN」的接口调用、并发序号守卫与失败降级：使用方只给
 * dbId/db/sql，不各自直连 sqlExplain 接口，接口口径一旦调整只需改这一处。
 *
 * fetchPlans 返回本次调用的结果（供提单快照等需要确定返回值的场景），并把最新一次结果
 * 同步到 plans 响应式引用（供对话框这类直接渲染当前取数的场景）；序号守卫丢弃过期响应，
 * 避免并发时旧 SQL 的结果覆盖新 SQL。失败/无库无 SQL 一律降级为空列表，不抛出。
 */
export function useSqlExplain() {
    const plans = ref<SqlExplainPlan[]>([]);
    const loading = ref(false);
    let seq = 0;

    async function fetchPlans(dbId: number, db: string, sql: string): Promise<SqlExplainPlan[]> {
        if (!dbId || !sql?.trim()) {
            plans.value = [];
            return [];
        }
        const mine = ++seq;
        loading.value = true;
        try {
            const res = (await dbApi.sqlExplain.request({ id: dbId, db, sql })) ?? [];
            if (mine === seq) plans.value = res;
            return res;
        } catch {
            if (mine === seq) plans.value = [];
            return [];
        } finally {
            if (mine === seq) loading.value = false;
        }
    }

    return { plans, loading, fetchPlans };
}
