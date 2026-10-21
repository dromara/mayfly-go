/**
 * runMultipleSqls 的结果 tab 生命周期测试。
 *
 * 钉住本次修复的行为：多条非查询语句执行前的「备注确认框」可取消，而清空旧结果是破坏性动作。
 * 清空必须发生在确认之后——否则弹窗一开上一批结果就没了，点「取消」也回不来（用户现场反馈）。
 *
 * 只 mock 执行链路的取数层（DbInst.execSql）与确认框，断言的是「旧结果是否被误清、是否误发请求」，
 * 不测请求怎么发。
 */
import { beforeEach, describe, expect, it, vi } from 'vitest';

// composable 顶层调用 useI18n()/Msg，测试环境无 app 实例，桩成恒等即可
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }));
vi.mock('@/hooks/useI18n', () => ({ Msg: { error: vi.fn(), success: vi.fn() } }));

// 执行链路依赖 DbInst（isQuerySql / execSql / invalidateSchemaCache）；取消用的 dbApi 与提醒框 confirmWarnAck 一并桩掉
const execSql = vi.fn();
const invalidateSchemaCache = vi.fn();
const isQuerySql = vi.fn((sql: string) => /^\s*select/i.test(sql));
vi.mock('../../../db', () => ({
    DbInst: { getInst: () => ({ isQuerySql, execSql, invalidateSchemaCache }) },
}));
vi.mock('../../../api', () => ({ dbApi: { cancelSqlExec: { request: vi.fn() } }, newSqlExecId: () => 'exec-id' }));
vi.mock('@/views/flow/warnAck', () => ({ confirmWarnAck: vi.fn() }));

import { useSqlExec } from '../useSqlExec';

/** 造一个已有结果的旧 tab（data 里放可辨识的哨兵值） */
const makeTab = (id: number) => ({ id, sql: '', data: [{ existing: 'prev' }], tableColumn: [], loading: false });

const setup = (askExecRemark: (sql: string) => Promise<string | null>) => {
    const state: any = { execResTabs: [makeTab(1)], activeTab: 1, tableDataEmptyText: '' };
    const api = useSqlExec({
        dbId: 1,
        dbName: 'testdb',
        state,
        atomic: () => false,
        askExecRemark,
        monacoEditor: null,
    });
    return { state, runMultipleSqls: api.runMultipleSqls, runSql: api.runSql, activeTab: api.activeTab };
};

/** 非查询执行回包：单条一个 rowsAffected */
const nonQueryRes = (sql: string) => ({
    data: { value: [{ sql, res: [{ rowsAffected: 1 }], columns: [] }] },
    execute: async () => undefined,
    abort: () => undefined,
    execId: 'exec-id',
});

describe('runMultipleSqls 备注确认与旧结果清空', () => {
    beforeEach(() => {
        execSql.mockReset();
        invalidateSchemaCache.mockReset();
        isQuerySql.mockClear();
    });

    it('取消备注确认：旧结果原样保留，不新建 tab，也不发任何执行请求', async () => {
        const ask = vi.fn(async () => null);
        const { state, runMultipleSqls } = setup(ask);
        const prevTabs = state.execResTabs;

        await runMultipleSqls(['ALTER TABLE t ADD COLUMN c INT', 'ALTER TABLE t2 ADD COLUMN c INT'], false);

        // 确认框弹了一次（整批一次），用户取消
        expect(ask).toHaveBeenCalledTimes(1);
        // 破坏性清空没有发生：仍是同一个数组、同一条旧结果
        expect(state.execResTabs).toBe(prevTabs);
        expect(state.execResTabs).toHaveLength(1);
        expect(state.execResTabs[0].data).toEqual([{ existing: 'prev' }]);
        // 取消即整体中止：一条都不执行
        expect(execSql).not.toHaveBeenCalled();
    });

    it('取消备注确认：同批混入的查询语句也不得被误跑', async () => {
        const ask = vi.fn(async () => null);
        const { state, runMultipleSqls } = setup(ask);

        await runMultipleSqls(['UPDATE t SET a = 1', 'SELECT * FROM t'], false);

        expect(ask).toHaveBeenCalledTimes(1);
        expect(state.execResTabs).toHaveLength(1);
        expect(state.execResTabs[0].data).toEqual([{ existing: 'prev' }]);
        expect(execSql).not.toHaveBeenCalled();
    });

    it('确认备注：旧结果被替换，整批非查询合并进一个 tab 并逐条执行', async () => {
        const ask = vi.fn(async () => '上线备注');
        execSql.mockImplementation((_db: string, sql: string) => nonQueryRes(sql));
        const { state, runMultipleSqls } = setup(ask);

        await runMultipleSqls(['ALTER TABLE t ADD c INT', 'UPDATE t SET a = 1'], false);

        expect(ask).toHaveBeenCalledTimes(1);
        // 两条非查询合并进同一个 tab（第一列 SQL，逐条一行），且旧哨兵结果已被本轮替换
        expect(state.execResTabs).toHaveLength(1);
        expect(state.execResTabs[0].data).toHaveLength(2);
        expect(JSON.stringify(state.execResTabs[0].data)).not.toContain('prev');
        // 逐条各发一次执行请求，备注透传到后端
        expect(execSql).toHaveBeenCalledTimes(2);
        expect(execSql.mock.calls[0][2]).toBe('上线备注');
    });

    it('纯查询多条：无需备注确认，直接清空并各自独立成 tab', async () => {
        const ask = vi.fn(async () => null);
        execSql.mockImplementation((_db: string, sql: string) => ({
            data: { value: [{ sql, res: [{ a: 1 }], columns: [{ name: 'a', key: 'a', type: 'number' }] }] },
            execute: async () => undefined,
            abort: () => undefined,
            execId: 'exec-id',
        }));
        const { state, runMultipleSqls } = setup(ask);

        await runMultipleSqls(['SELECT 1', 'SELECT 2'], false);

        // 查询不弹备注框，也就没有「取消后结果消失」的问题
        expect(ask).not.toHaveBeenCalled();
        expect(execSql).toHaveBeenCalledTimes(2);
        // 每条查询各占一个 tab
        expect(state.execResTabs).toHaveLength(2);
    });
});

/**
 * 页签切换与执行落点：在哪个结果页签执行就显示在哪个页签，而不是永远落到最后一个。
 *
 * 现场问题：state.activeTab 只在程序运行时被赋值，用户点页签不更新它，导致在结果-1 执行却写进结果-2。
 */
describe('页签切换与执行落点', () => {
    beforeEach(() => {
        execSql.mockReset();
    });

    it('点切到某个结果页签后，单条执行落在该页签而非最后一个', async () => {
        execSql.mockImplementation((_db: string, sql: string) => ({
            data: { value: [{ sql, res: [{ a: 1 }], columns: [{ name: 'a', key: 'a', type: 'number' }] }] },
            execute: async () => undefined,
            abort: () => undefined,
            execId: 'exec-id',
        }));
        const { state, runSql, activeTab } = setup(async () => '');
        // 已有两个结果页签，程序态 activeTab 停在 2（上一次运行留下的），用户正在看结果-1
        state.execResTabs = [makeTab(1), makeTab(2)];
        state.activeTab = 2;

        // 用户点击切到结果-1：activeTab 必须同步为 1
        activeTab(1);
        expect(state.activeTab).toBe(1);

        // 在当前页签执行单条查询（newTab=false）：结果写进结果-1，结果-2 不受影响
        await runSql('SELECT 1', '', false);
        expect(state.execResTabs[0].data).toEqual([{ a: 1, policyNotice: '' }]);
        expect(state.execResTabs[1].data).toEqual([{ existing: 'prev' }]);
    });

    it('无页签时执行自动新建一个', async () => {
        execSql.mockImplementation((_db: string, sql: string) => ({
            data: { value: [{ sql, res: [{ a: 1 }], columns: [{ name: 'a', key: 'a', type: 'number' }] }] },
            execute: async () => undefined,
            abort: () => undefined,
            execId: 'exec-id',
        }));
        const { state, runSql } = setup(async () => '');
        state.execResTabs = [];
        state.activeTab = 0;

        await runSql('SELECT 1', '', false);
        expect(state.execResTabs).toHaveLength(1);
        expect(state.activeTab).toBe(1);
    });

    it('按 id 而非下标定位页签：删过中间页签后仍写对', async () => {
        execSql.mockImplementation((_db: string, sql: string) => ({
            data: { value: [{ sql, res: [{ a: 1 }], columns: [{ name: 'a', key: 'a', type: 'number' }] }] },
            execute: async () => undefined,
            abort: () => undefined,
            execId: 'exec-id',
        }));
        const { state, runSql, activeTab } = setup(async () => '');
        // 剩下 id 为 1 和 3 的两个页签（id 2 已被关），当前激活 id=3
        state.execResTabs = [makeTab(1), makeTab(3)];
        state.activeTab = 3;

        activeTab(1);
        expect(state.activeTab).toBe(1);
        await runSql('SELECT 1', '', false);
        // 写入下标 0（id=1）那个页签
        expect(state.execResTabs[0].data).toEqual([{ a: 1, policyNotice: '' }]);
        expect(state.execResTabs[1].data).toEqual([{ existing: 'prev' }]);
    });
});
