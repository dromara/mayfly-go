/**
 * 草稿取数：接口调用 + 并发序号守卫 + 可中断。
 *
 * 与 useSqlExplain 同构：使用方只给 payload，不各自直连接口。序号守卫的作用是防止
 * 「用户连点两次生成」时先发的慢响应后到，把后发的快响应覆盖掉。
 */
import { reactive } from 'vue';
import { sqlDraftApi } from './api';
import type { SqlDraftRequest, SqlDraftResult } from './types';

/**
 * 客户端超时：后端有 55s 的模型预算，但连接可能永远不返回（后端重启打断、代理挂住）。
 * 没有这层兜底时，对话框会一直转圈、遮罩连关闭按钮都点不动——真机实测踩到过。
 */
const REQUEST_TIMEOUT_MS = 120_000;

/** 生成状态：SQL 助手面板与详情面板共用同一份，避免两处各自调接口产生两份结果 */
export interface DraftState {
    loading: boolean;
    result: SqlDraftResult | null;
    error: string;
    timedOut: boolean;
}

export function useSqlDraft(dbId: number, dbName: string) {
    const state = reactive<DraftState>({
        loading: false,
        result: null,
        error: '',
        timedOut: false,
    });
    let seq = 0;
    /**
     * 当前在途请求的中止控制器（同时只留一个）。
     *
     * 为什么不能只靠序号：此前「取消」只 seq++，请求仍在跑、loading 由在途计数派生，
     * 于是用户点了取消界面照旧转圈——那就是「没有取消」。必须真 abort，并立即把 loading 放下。
     */
    let controller: AbortController | null = null;

    /** 发起一次生成；返回结果（被更新的一次覆盖或被取消时为 null） */
    async function generate(payload: SqlDraftRequest): Promise<SqlDraftResult | null> {
        const mine = ++seq;
        // 新一轮生成意味着上一版结果不再被需要：继续等它只是白等一次模型耗时
        controller?.abort();
        const own = new AbortController();
        controller = own;
        state.loading = true;
        state.result = null;
        state.error = '';
        state.timedOut = false;
        let timeoutHandle: ReturnType<typeof setTimeout> | undefined;
        try {
            const result = await Promise.race([
                sqlDraftApi.draft.request(
                    {
                        ...payload,
                        dbId,
                        db: payload.db || dbName,
                    },
                    // RequestOptions 继承 RequestInit，signal 会透传到 fetch（不改公共接口）
                    { signal: own.signal }
                ),
                new Promise<null>((resolve) => {
                    timeoutHandle = setTimeout(() => resolve(null), REQUEST_TIMEOUT_MS);
                }),
            ]);
            if (mine !== seq) {
                return null;
            }
            if (result === null) {
                // 超时：清掉 loading 让用户能关闭/重试；迟到结果由序号守卫丢弃
                state.timedOut = true;
                state.result = null;
                return null;
            }
            state.result = result;
            return result;
        } catch (e: unknown) {
            // 主动取消不是错误：不弹红条、不留错误文案（用户就是不想要这一版）
            if (own.signal.aborted || mine !== seq) {
                return null;
            }
            state.result = null;
            state.error = e instanceof Error ? e.message : String(e);
            return null;
        } finally {
            if (timeoutHandle) {
                clearTimeout(timeoutHandle);
            }
            if (controller === own) {
                controller = null;
            }
            // 只有仍是最新的一次才有权改 loading：已被作废的那次不能把已收起的转圈再点亮
            if (mine === seq) {
                state.loading = false;
            }
        }
    }

    /**
     * 作废当前这一轮：中断在途请求、立即停止转圈、清掉结果与提示。
     *
     * 同一个动作同时服务三个场景：用户点「取消」/按 Esc、重开浮层、切换意图。
     * 注意中断只是不再等结果，后端的模型调用仍会跑完那一次（费用已发生），
     * 所以界面上不该把它说成「已停止生成」。
     */
    function cancel() {
        seq++;
        controller?.abort();
        controller = null;
        state.loading = false;
        state.result = null;
        state.error = '';
        state.timedOut = false;
    }

    return { state, generate, cancel };
}

/**
 * 草稿状态句柄：由编辑器持有一份，同时交给 SQL 助手面板与详情面板。
 *
 * 必须是同一份：两处各调一次接口就会出现两个结果，用户在浮层里看到的语句和
 * 面板里的校验明细可能对不上，而「采纳」写回的是哪一版已经说不清。
 */
export type DraftHandle = ReturnType<typeof useSqlDraft>;
