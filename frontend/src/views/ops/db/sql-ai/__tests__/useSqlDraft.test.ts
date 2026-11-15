import { describe, expect, it, vi, beforeEach } from 'vitest';

/**
 * 草稿取数的 loading / 竞态 / 取消语义。
 *
 * 取消这条是用户实报的缺陷：此前 cancel 只推进序号，请求仍在跑、loading 由在途计数派生，
 * 界面表现就是「点取消没反应、还在转圈」。所以这里钉三件事：
 *  1. 取消后 loading 必须**立刻**为 false（不能等接口回来）；
 *  2. 取消必须真的 abort 在途请求（signal.aborted）；
 *  3. 迟到结果既不写 result，也不弹错误文案（主动取消不是失败）。
 */
interface PendingCall {
    resolve: (value: unknown) => void;
    reject: (reason: unknown) => void;
    signal?: AbortSignal;
}
const pendingCalls: PendingCall[] = [];
const draftRequest = vi.fn(
    (_params: unknown, options?: { signal?: AbortSignal }) =>
        new Promise((resolve, reject) => {
            pendingCalls.push({ resolve, reject, signal: options?.signal });
        })
);

vi.mock('../api', () => ({
    sqlDraftApi: { draft: { request: draftRequest } },
}));

const { useSqlDraft } = await import('../useSqlDraft');

const flush = () => new Promise((r) => setTimeout(r, 0));

const resultOf = (sql: string) => ({
    intent: 'draft',
    statements: [{ sql }],
    verification: 'verified',
    checks: [],
    contextTables: [],
    repairs: 0,
    candidates: 0,
    dialect: 'mysql',
});

describe('草稿取数的 loading 与竞态', () => {
    beforeEach(() => {
        pendingCalls.length = 0;
        draftRequest.mockClear();
    });

    it('取消：立即停止转圈并中断在途请求，迟到结果被丢弃且不报错误', async () => {
        const { state, generate, cancel } = useSqlDraft(1, 'demo');
        const promise = generate({ db: 'demo', intent: 'draft', prompt: '统计' });
        await flush();
        expect(state.loading).toBe(true);

        cancel();
        expect(state.loading, '点取消后还在转圈就等于没有取消').toBe(false);
        expect(pendingCalls[0].signal?.aborted, '必须真的 abort 掉在途请求').toBe(true);

        // 迟到的响应回来：不能写结果、不能报错、不能把转圈再点亮
        pendingCalls[0].resolve(resultOf('select 1'));
        await promise;
        expect(state.result).toBeNull();
        expect(state.error).toBe('');
        expect(state.loading).toBe(false);
    });

    it('取消后接口抛 AbortError 时不当成失败展示', async () => {
        const { state, generate, cancel } = useSqlDraft(1, 'demo');
        const promise = generate({ db: 'demo', intent: 'draft', prompt: '统计' });
        await flush();
        cancel();
        pendingCalls[0].reject(new DOMException('aborted', 'AbortError'));
        await promise;
        expect(state.error).toBe('');
        expect(state.loading).toBe(false);
    });

    it('新一轮生成会中断上一轮在途请求（同时只留一个）', async () => {
        const { generate } = useSqlDraft(1, 'demo');
        const first = generate({ db: 'demo', intent: 'draft', prompt: '第一次' });
        await flush();
        const second = generate({ db: 'demo', intent: 'draft', prompt: '第二次' });
        await flush();
        expect(pendingCalls).toHaveLength(2);
        expect(pendingCalls[0].signal?.aborted).toBe(true);

        pendingCalls[1].resolve(resultOf('select 2'));
        pendingCalls[0].reject(new DOMException('aborted', 'AbortError'));
        await Promise.all([first, second]);
    });

    it('迟到的旧响应不覆盖新结果', async () => {
        const { state, generate } = useSqlDraft(1, 'demo');
        const first = generate({ db: 'demo', intent: 'draft', prompt: '第一次' });
        await flush();
        const second = generate({ db: 'demo', intent: 'draft', prompt: '第二次' });
        await flush();

        // 第二次先回
        pendingCalls[1].resolve(resultOf('select 2'));
        await second;
        expect(state.result?.statements?.[0]?.sql).toBe('select 2');

        // 第一次后回：必须被丢弃（这里让它正常 resolve，模拟没有 abort 的通道）
        pendingCalls[0].resolve(resultOf('select 1'));
        await first;
        expect(state.result?.statements?.[0]?.sql).toBe('select 2');
        expect(state.loading).toBe(false);
    });

    it('请求失败时回填错误并清 loading', async () => {
        const { state, generate } = useSqlDraft(1, 'demo');
        const promise = generate({ db: 'demo', intent: 'draft', prompt: '统计' });
        await flush();
        pendingCalls[0].reject(new Error('boom'));
        await promise;
        expect(state.loading).toBe(false);
        expect(state.error).toContain('boom');
    });

    it('signal 必须真的传给请求层，否则取消只是界面骗自己', async () => {
        const { generate } = useSqlDraft(1, 'demo');
        const promise = generate({ db: 'demo', intent: 'draft', prompt: '统计' });
        await flush();
        expect(pendingCalls[0].signal).toBeInstanceOf(AbortSignal);
        pendingCalls[0].resolve(resultOf('select 1'));
        await promise;
    });
});
