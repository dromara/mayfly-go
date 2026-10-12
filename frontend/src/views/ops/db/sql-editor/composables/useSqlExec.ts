import { notBlank } from '@/common/assert';
import { Msg } from '@/hooks/useI18n';
import type { editor, IRange } from 'monaco-editor';
import { ref, unref, type Ref, type UnwrapNestedRefs } from 'vue';
import { useI18n } from 'vue-i18n';

import { DbInst } from '../../db';
import { dbApi } from '../../api';
import { getDialectCapabilities } from '../../dialect';
import { isDdlSql } from '../../core/sqlKind';
import { confirmWarnAck } from '@/views/flow/warnAck';
import type { PolicyNotice, SqlExecRes, SqlExecResColumn, TableColumnDef } from '../../types';
import { getCurrentStatement, splitSqlStatements } from '../utils/sqlParser';

/** DbTableData 组件通过 defineExpose 暴露的方法 */
export interface DbTableDataRef {
    active: () => void;
    submitUpdateFields: () => void;
    cancelUpdateFields: () => void;
}

export class ExecResTab {
    id: number;

    /**
     * 当前结果集对应的sql
     */
    sql: string;

    /**
     * 响应式loading
     */
    loading: Ref<boolean>;

    dbTableRef: DbTableDataRef | null = null;

    abortFn?: Function;

    tableColumn: TableColumnDef[] = [];

    data: Record<string, unknown>[] = [];

    execTime: number;

    /**
     * 当前单表操作sql关联的表信息
     */
    table: string;

    /**
     * 是否有更新字段
     */
    hasUpdatedFields: boolean;

    errorMsg: string;

    /**
     * 本次执行是否被操作者取消（点了取消按钮，请求被中断）。
     *
     * 与 errorMsg 分开存：取消不是失败，不能渲染成红色「执行失败」；
     * 也不能什么都不显示——结果页签会落进「成功」分支，看起来像正常跑完了
     */
    cancelled = false;

    /**
     * 被触发策略要求审批的语句原文。
     *
     * 与 errorMsg 分开存：失败提示可能是语法错误、权限不足等任何原因，
     * 只有这一列非空才该出现「提交工单」入口
     */
    needApprovalSqls: string[] = [];

    /**
     * 命中「仅提醒」后操作者选择先不执行时的提示。
     *
     * 与 errorMsg 分开存：语句根本没跑，不是执行失败。混在 errorMsg 里会渲染成红色「执行失败」，
     * 操作者会以为语句出了错，而真实情况只是「等你确认或提单」
     */
    notice = '';

    /**
     * 结果集是否被行数上限截断（只由服务端给的 truncated 决定，不按行数反推）
     */
    truncated = false;

    /** 本次生效的行数上限，截断时用于告诉用户「只显示了前 N 行」 */
    rowLimit = 0;

    /**
     * 产出该结果集的那一条语句（只单条执行路径有值）。
     *
     * 全量导出需要重跑它；批量路径一个 tab 里是多条语句，导出哪一条没有答案，
     * 因此不供导出入口而不是「猜第一条」
     */
    exportSql = '';

    constructor(id: number) {
        this.id = id;
    }
}

/** reactive 解包后的结果集tab类型 (loading 字段的 Ref 被解包为 boolean) */
export type ExecResTabState = UnwrapNestedRefs<ExecResTab>;

/** 事件处理函数参数用的tab结构 (不含 loading，兼容原始与解包两种类型) */
export interface ExecResTabLike {
    id: number;
    dbTableRef: DbTableDataRef | null;
    hasUpdatedFields: boolean;
    data: Record<string, unknown>[];
    table: string;
}

interface UseSqlExecOptions {
    dbId: number;
    dbName: string;
    state: {
        execResTabs: ExecResTabState[];
        activeTab: number;
        tableDataEmptyText: string;
    };
    /** 是否以原子批次执行多条语句（取编辑器工具条上的开关状态，随每次执行读取） */
    atomic: () => boolean;
    /** 检测到「需审批」语句时的回调：由编辑器传入提单入口，用于自动打开预填好的工单抽屉；批量场景传多条合并后的语句 */
    onNeedApproval?: (sql: string) => void;
    /** 执行非查询语句前的确认框（备注 + 查看执行计划），由编辑器提供；resolve 备注文本，取消 resolve null */
    askExecRemark?: (sql: string) => Promise<string | null>;
    monacoEditor: editor.IStandaloneCodeEditor | null;
}

export function useSqlExec(options: UseSqlExecOptions) {
    const { t } = useI18n();

    /**
     * 把策略提醒渲染成结果表格里的一列文本。
     *
     * 后端只下发 i18n key，翻译在前端完成；无提醒时留空而不是占位符，
     * 让「有提醒」成为这一列里唯一需要读的东西
     */
    const noticeTexts = (notices: PolicyNotice[] | undefined): string => (notices ?? []).map((notice) => t(notice.title)).join('；');
    const { dbId, dbName, state, atomic, onNeedApproval, askExecRemark } = options;

    /**
     * 每个结果tab的运行代号。
     *
     * 取消后旧一次运行的请求 promise 仍可能迟到结算（响应与 abort 赛跑）），
     * 其 catch/finally 若照写页签会把新一轮运行的状态抹掉——轻则清掉新运行的 abortFn（取消按钮消失），
     * 重则用旧空值覆盖新结果。只有持当前代号的写入才允许落盘；取消动作在收尾时把代号加一
     */
    const runGens = new Map<number, number>();
    const beginRun = (tabId: number): number => {
        const gen = (runGens.get(tabId) ?? 0) + 1;
        runGens.set(tabId, gen);
        return gen;
    };
    const isCurrentRun = (tabId: number, gen: number): boolean => runGens.get(tabId) === gen;

    /**
     * 从各类抛出物里取出可展示文案。
     *
     * 本地分支抛的是 { msg } 普通对象（提醒确认、服务端 errorMsg），请求层抛的是 Error 实例
     * （业务失败与网络错误，文案在 message 上，没有 msg）——只认一种会让另一类变成空白
     */
    const errorText = (e: unknown): string => {
        if (e instanceof Error) {
            return e.message;
        }
        const msg = (e as Record<string, unknown> | null)?.msg;
        return typeof msg === 'string' ? msg : String(e);
    };

    /** 请求被主动中断（操作者点了取消）时的拒绝类型：不是执行失败，但也不是成功 */
    const isAbortError = (e: unknown): boolean => (e as Error | null)?.name === 'AbortError';

    /** 一次执行开始前清掉上一次的展示态：残留的失败/提醒/取消/截断会盖住或误标本次结果 */
    const resetExecTab = (tab: ExecResTabState) => {
        tab.errorMsg = '';
        tab.notice = '';
        tab.cancelled = false;
        tab.needApprovalSqls = [];
        tab.truncated = false;
        tab.rowLimit = 0;
        tab.exportSql = '';
    };

    const getNowDbInst = () => {
        return DbInst.getInst(dbId);
    };

    /**
     * 组合「取消」：先请服务端中断该执行（至少不再往回读结果），再断本地 HTTP 收尾界面。
     *
     * 只 abort 前端请求是不够的：连接池会保留那条正在跑的语句，界面转圈结束了库里还在跑。
     * 服务端那一跳会尝试停掉本次执行上下文，但是否真停下来由数据库侧决定，
     * 所以这里不能说「已中断」，只告诉用户「已请求取消」。
     * 结果必须出声：只把遮罩收起而不告诉用户「到底取消了没有」，
     * 他会不知道那条语句是在库里继续跑还是已经停了（服务端可能已回「该执行已结束」）
     */
    const cancelExec = (execId: string, abort: () => void, terminate: () => void) => {
        // 一次取消动作只发一发：主线程卡顿时两次点击可能都落在重渲染之前，
        // 第二发到达时第一发早已结算注销，只能拿回「刚刚结束」——对已成功动作的纯浪费请求
        let fired = false;
        return () => {
            if (fired) {
                return;
            }
            fired = true;
            void dbApi.cancelSqlExec
                .request({ execId })
                .then(() => Msg.success('db.execCancelRequested'))
                // 失败不在这里另发一条：中心层已经把服务端的原话 toast 过了（如「该执行刚刚结束，无需取消」），
                // 再叠一条自己编的文案只会让用户看不出到底哪个是真结论
                .catch(() => undefined);
            abort();
            // 本地执行态同步收尾，不等请求 promise 结算：响应与 abort 无论谁赢，
            // 页签都必须立刻离开「执行中」，否则操作者既看不到取消结果也无法再次执行
            terminate();
        };
    };

    /**
     * 新建结果集tab并存入响应式数组。
     */
    const pushNewTab = (id: number): ExecResTabState => {
        const tab = new ExecResTab(id);
        // 写入 reactive 数组后，Vue 在代理层将实例内的 Ref 字段解包（loading: Ref<boolean> → boolean），
        // 对外形状即 ExecResTabState；该解包无法由「推入原始实例」这一静态行为推知，故在此唯一入口跨一次
        state.execResTabs.push(tab as unknown as ExecResTabState);
        return state.execResTabs[state.execResTabs.length - 1];
    };

    /**
     * 执行一批 SQL：
     * - 单条：非查询类先弹出备注输入，再在当前/新 tab 执行；
     * - 多条：按查询/非查询分类合并执行（见 runMultipleSqls）。
     *
     * @param sqls 待执行 SQL 列表
     * @param emptyMsg 列表为空时的提示（已解析的 i18n 文本）
     * @param newTab 是否在新 tab 执行
     */
    const runSqlList = async (sqls: string[], emptyMsg: string, newTab = false) => {
        if (!sqls?.length) {
            // 这里原先是 notBlank 抛错：整条调用链上没有捕获点，也没有全局处理器，
            // 编辑器为空时点「执行」就是彻底没反应，用户只会以为按钮坏了
            Msg.error(emptyMsg);
            return;
        }

        if (sqls.length == 1) {
            const oneSql = sqls[0];
            let execRemark = '';
            if (!getNowDbInst().isQuerySql(oneSql)) {
                // 确认框由编辑器提供（含「查看执行计划」）；取消返回 null → 中止本次执行
                const remark = await (askExecRemark?.(oneSql) ?? Promise.resolve(''));
                if (remark === null) return;
                execRemark = remark;
            }
            runSql(oneSql, execRemark, newTab);
            return;
        }

        // 处理多条SQL - 合并相同类型的结果
        await runMultipleSqls(sqls, newTab);
    };

    /*
     * 执行sql：有选区则执行选区，否则执行光标所在的单条语句
     */
    const onRunSql = async (newTab = false) => {
        await runSqlList(getSql(), t('db.noSelectRunSqlMsg'), newTab);
    };

    /*
     * 执行编辑器内的全部 SQL：忽略选区与光标位置，拆分为多条后逐类执行
     */
    const onRunAllSql = async (newTab = false) => {
        await runSqlList(getAllSql(), t('db.noSqlToRunMsg'), newTab);
    };

    /**
     * 执行多条SQL并合并结果
     */
    const runMultipleSqls = async (sqls: string[], newTab: boolean) => {
        state.execResTabs = [];
        // 分类SQL语句
        const nonQuerySqls: string[] = []; // 影响行数类SQL (UPDATE, INSERT, DELETE等)
        const querySqls: string[] = []; // 查询类SQL (SELECT等)

        const dbInst = getNowDbInst();
        // 分类SQL
        sqls.forEach((sql) => {
            if (!dbInst.isQuerySql(sql)) {
                nonQuerySqls.push(sql);
            } else {
                querySqls.push(sql);
            }
        });

        // 先执行非查询类SQL（可以合并结果）
        if (nonQuerySqls.length > 0) {
            await runNonQuerySqls(nonQuerySqls, newTab);
            newTab = true; // 后续查询需要新标签页
        }

        // 再执行查询类SQL（每条需要独立标签页）
        for (let i = 0; i < querySqls.length; i++) {
            const sql = querySqls[i];
            await runSql(sql, '', newTab || i > 0);
        }
    };

    /**
     * 执行非查询类SQL并合并结果
     */
    const runNonQuerySqls = async (sqls: string[], newTab: boolean) => {
        // 非查询批量也先弹一次执行备注确认（与单条一致）：整批非查询语句交给确认框，
        // 其「查看执行计划」可逐条选中；取消（null）即中止整批，不创建空结果 tab
        let remark = '';
        if (askExecRemark) {
            const asked = await askExecRemark(sqls.join(';\n'));
            if (asked === null) {
                return;
            }
            remark = asked;
        }

        let execRes: ExecResTabState;
        let i = 0;
        let id;

        // 获取或创建结果标签页
        if (newTab || state.execResTabs.length == 0) {
            id = state.execResTabs.length == 0 ? 1 : state.execResTabs[state.execResTabs.length - 1].id + 1;
            execRes = pushNewTab(id);
            i = state.execResTabs.length - 1;
        } else {
            i = state.execResTabs.findIndex((x) => x.id == state.activeTab);
            execRes = state.execResTabs[i];
            if (unref(execRes.loading)) {
                Msg.error('db.currentSqlTabIsRunning');
                return;
            }
            id = execRes.id;
        }

        state.activeTab = id;
        const startTime = new Date().getTime();
        const gen = beginRun(id);
        // loading 用自有可写 ref，见 runSql 同处注释：为何不把 vueuse 的 isFetching 原样存入。
        // 类型上 ExecResTabState 已把 loading 解包为 boolean，但运行时页签字段仍是 Ref 形状
        // （reactive set 会写穿到 oldValue.value），与 pushNewTab 同理在此唯一写入点跨一次类型
        const loadingRef = ref(false);
        execRes.loading = loadingRef as unknown as boolean;
        // 批量取消标记：取消后逐条循环必须停发新语句
        let batchCancelled = false;
        // 取消收尾：同步让页签离开「执行中」并把代号加一，使旧请求的迟到结算不能再写页签
        const terminate = () => {
            batchCancelled = true;
            loadingRef.value = false;
            execRes.cancelled = true;
            execRes.abortFn = undefined;
            runGens.set(id, gen + 1);
        };

        try {
            resetExecTab(execRes);
            execRes.sql = sqls.join('\n\n---\n\n'); // 显示所有SQL

            // 执行所有非查询SQL
            const results: Record<string, unknown>[] = [];
            const needApprovalSqls: string[] = [];

            // 原子批次：整批语句一次请求发出，由后端在同一事务里逐条执行、失败整批回滚。
            // 不能沿用下面的逐条请求：那等于 N 个独立事务，开关只体现在界面上，
            // 失败时前面的语句早已各自提交，界面却仍按「整批原子」显示 —— 属于假承诺
            if (atomic()) {
                const { data, execute, abort, execId } = getNowDbInst().execSql(dbName, sqls.join(';\n'), remark, { ask: false, atomic: true });
                execRes.abortFn = cancelExec(execId, abort, terminate);
                loadingRef.value = true;
                await execute();
                for (const item of (data.value as SqlExecRes[]) ?? []) {
                    if (item.needApproval && item.sql) {
                        needApprovalSqls.push(item.sql);
                    }
                    results.push({
                        sql: item.sql,
                        rowsAffected: item.res?.[0]?.rowsAffected,
                        error: item.errorMsg || '-',
                        notices: noticeTexts(item.notices),
                    });
                }
            } else {
                // 逐条请求也全程暴露取消入口：取消会中断当前语句并停发后续语句，
                // 否则长批量期间页签既无进度也无退出，「执行中」检查也不会拦同页签重跑
                loadingRef.value = true;
                for (const sql of sqls) {
                    if (batchCancelled) {
                        break;
                    }
                    try {
                        // 批量选区不逐条追问提醒：ask=false，命中的提醒随结果回显在「策略提醒」列。
                        // 这里不能改成 ask=true：批量本身就是一条一条发请求，逐条弹窗会让人没法干活
                        const { data, execute, abort, execId } = getNowDbInst().execSql(dbName, sql, remark, { ask: false });
                        execRes.abortFn = cancelExec(execId, abort, terminate);
                        await execute();
                        const result = (data.value as SqlExecRes[])[0];
                        if (result.needApproval && result.sql) {
                            needApprovalSqls.push(result.sql);
                        }
                        results.push({
                            sql: result.sql,
                            rowsAffected: result.res?.[0]?.rowsAffected,
                            error: result.errorMsg || '-',
                            notices: noticeTexts(result.notices),
                        });
                    } catch (error: unknown) {
                        if (batchCancelled) {
                            break;
                        }
                        results.push({
                            sql: sql,
                            error: error instanceof Error ? error.message : String(error),
                            notices: '',
                        });
                    }
                }
            }

            if (batchCancelled) {
                return;
            }

            // 设置表格列：提醒列只在真有提醒时出现（与单条路径一致），否则整列空值看着像缺数据
            const cols: TableColumnDef[] = [
                { columnName: 'SQL', key: 'sql', columnType: 'string', show: true },
                { columnName: 'RowsAffected', key: 'rowsAffected', columnType: 'number', show: true },
                { columnName: t('common.error'), key: 'error', columnType: 'string', show: true },
            ];
            if (results.some((r) => r.notices)) {
                cols.push({ columnName: t('db.policyNotices'), key: 'notices', columnType: 'string', show: true });
            }
            state.execResTabs[i].tableColumn = cols;

            state.execResTabs[i].data = results;
            state.execResTabs[i].needApprovalSqls = needApprovalSqls;
            // 批量（含原子）检测到需审批语句：合并成一张工单（同一次请求里已收集全部），审批通过后整单一起回放
            if (needApprovalSqls.length) {
                onNeedApproval?.(needApprovalSqls.join(';\n'));
            }
            cancelUpdateFields(execRes);
            // 批量执行含 DDL：失效补全元数据缓存，反映最新表/字段/注释
            if (sqls.some(isDdlSql)) {
                getNowDbInst().invalidateSchemaCache(dbName);
            }
        } catch (e: unknown) {
            if (!isCurrentRun(id, gen)) {
                return;
            }
            execRes.data = [];
            execRes.tableColumn = [];
            execRes.table = '';
            if (isAbortError(e) || batchCancelled) {
                // 操作者点了取消（请求被中断）：不是失败，置为已取消，让结果区如实显示状态
                state.execResTabs[i].cancelled = true;
            } else {
                state.execResTabs[i].errorMsg = errorText(e);
            }
            return;
        } finally {
            if (isCurrentRun(id, gen)) {
                loadingRef.value = false;
                execRes.execTime = new Date().getTime() - startTime;
                // 执行结束（无论成功/失败/取消）都清空取消函数：遮罩已收起，按钮不再可见，
                // 但引用残留会让下次打开同一 tab 时 v-if="abortFn" 误判为仍可取消
                execRes.abortFn = undefined;
            }
        }

        execRes.table = '';
    };

    /**
     * 执行单条sql
     *
     * @param sql 单条sql
     * @param remark 执行备注
     * @param newTab 是否新建tab
     */
    const runSql = async (sql: string, remark = '', newTab = false) => {
        let execRes: ExecResTabState;
        let i = 0;
        let id;
        // 新tab执行，或者tabs为0，则新建tab执行sql
        if (newTab || state.execResTabs.length == 0) {
            // 取最后一个tab的id + 1
            id = state.execResTabs.length == 0 ? 1 : state.execResTabs[state.execResTabs.length - 1].id + 1;
            execRes = pushNewTab(id);
            i = state.execResTabs.length - 1;
        } else {
            // 不是新建tab执行，则在当前激活的tab上执行sql
            i = state.execResTabs.findIndex((x) => x.id == state.activeTab);
            execRes = state.execResTabs[i];
            if (unref(execRes.loading)) {
                Msg.error('db.currentSqlTabIsRunning');
                return;
            }
            id = execRes.id;
        }

        state.activeTab = id;
        const startTime = new Date().getTime();
        const gen = beginRun(id);
        /**
         * loading 用自有可写 ref，而不是把 vueuse 的 isFetching 原样存进页签：
         * isFetching 是 shallowReadonly 包装的只读 ref，存进 reactive 页签后，
         * 任何「收尾写回布尔值」的赋值都会命中 Vue set 的 readonly 分支被静默吞掉（仅 dev 告警），
         * 页签会永久卡在「执行中」——取消后既看不到结果也无法再次执行。
         * 自有 ref 的值完全由本运行器驱动：发起前置 true，结算/取消收尾置 false
         */
        const loadingRef = ref(false);
        execRes.loading = loadingRef as unknown as boolean;
        // 取消收尾：同步让页签离开「执行中」并把代号加一，使旧请求的迟到结算不能抹掉新一轮运行的状态
        const terminate = () => {
            loadingRef.value = false;
            execRes.cancelled = true;
            execRes.abortFn = undefined;
            runGens.set(id, gen + 1);
        };
        try {
            resetExecTab(execRes);
            execRes.sql = '';

            // 一次执行请求的封装：asked=true 才能弹确认（单条场景），acknowledged 只在操作者选过「直接执行」后带上
            const runOnce = async (acknowledged: boolean) => {
                const { data, execute, abort, execId } = getNowDbInst().execSql(dbName, sql, remark, { ask: true, acknowledged });
                execRes.abortFn = cancelExec(execId, abort, terminate);
                loadingRef.value = true;
                await execute();
                return (data.value as SqlExecRes[])[0];
            };

            // 命中提醒后的三态确认：选「直接执行」就带确认重试一次；选「提交工单审批」则语句保持未执行，
            // 并把该语句交给结果区现成的提单入口；关掉弹窗只是暂缓，不给提单按钮
            const askWarnAck = async (res: SqlExecRes, retry: (acknowledged: boolean) => Promise<SqlExecRes>): Promise<SqlExecRes> => {
                const message = res.errorMsg || t('flow.warnAckTitle');
                const choice = await confirmWarnAck(message);
                // 弹窗打开期间可能已发生取消（terminate 已把代号加一并收尾页签）：
                // 此后无论选哪项都不得再写页签或重发语句，否则语句被真正执行、
                // 已收尾的页签被翻回「执行中」永久卡死（迟到结算全被代号守卫丢弃，无人再收尾）。
                // 模态遮挡下取消按钮点不到，但收尾正确性不能依赖这个外部前提
                if (!isCurrentRun(id, gen)) {
                    throw { msg: 'cancelled' };
                }
                if (choice === 'run') {
                    return await retry(true);
                }
                // needApprovalSqls 是提单按钮的唯一出现条件，改它必须先于任何提示分支
                state.execResTabs[i].needApprovalSqls = choice === 'ticket' ? [sql] : [];
                if (choice === 'ticket') {
                    onNeedApproval?.(sql);
                }
                // 语句没执行，必须把提示留在结果区：静默吞掉会让人以为已经跑过了
                throw { msg: message, warnAck: true };
            };

            let colAndData = await runOnce(false);
            if (colAndData.warnAck) {
                // 命中「仅提醒」：此刻语句还没执行，这是唯一能改走审批的时机
                colAndData = await askWarnAck(colAndData, runOnce);
            }
            // 取消在 await 期间发生时，后续写入全部作废：terminate 已把页签收尾，
            // 这里再写会把「已取消」覆盖成半截成功态
            if (!isCurrentRun(id, gen)) {
                return;
            }
            // 单条路径此前会丢掉策略结论：批量路径有提醒列而单条没有，
            // 「仅提醒」级别在最常用的单条执行场景下等于没有出口
            state.execResTabs[i].needApprovalSqls = colAndData.needApproval ? [sql] : [];
            // 单条命中需审批：自动打开预填好的提单抽屉（不再只靠结果区的按钮）
            if (colAndData.needApproval) {
                onNeedApproval?.(sql);
            }
            // 截断事实与源语句一起记下：没有它们，界面无法告知「这只是前 N 行」也发不出全量导出
            state.execResTabs[i].truncated = !!colAndData.truncated;
            state.execResTabs[i].rowLimit = colAndData.rowLimit ?? 0;
            state.execResTabs[i].exportSql = sql;
            if (colAndData.errorMsg) {
                throw { msg: colAndData.errorMsg };
            }

            if (colAndData.res?.length == 0) {
                state.tableDataEmptyText = 'No Data';
            }

            // 要实时响应，故需要用索引改变数据才生效
            const noticeText = noticeTexts(colAndData.notices);
            const rows = (colAndData.res ?? []).map((row: Record<string, unknown>) => ({ ...row, policyNotice: noticeText }));
            state.execResTabs[i].data = rows.length > 0 || !noticeText ? rows : [{ policyNotice: noticeText }];
            // 兼容表格字段配置
            state.execResTabs[i].tableColumn = (colAndData.columns ?? []).map((x: SqlExecResColumn) => {
                return {
                    columnName: x.name,
                    key: x.key,
                    columnType: x.type,
                    masked: x.masked,
                    show: true,
                };
            });
            // 提醒列只在真有提醒时出现：查询结果的列由语句自身决定，凭空挂一列恒空会被误读成表字段
            if (noticeText) {
                state.execResTabs[i].tableColumn.push({ columnName: t('db.policyNotices'), key: 'policyNotice', columnType: 'string', show: true });
            }
            cancelUpdateFields(execRes);
            // 执行了 DDL：失效补全元数据缓存，使表名/字段/注释联想反映最新结构
            if (isDdlSql(sql)) {
                getNowDbInst().invalidateSchemaCache(dbName);
            }
        } catch (e: unknown) {
            if (!isCurrentRun(id, gen)) {
                return;
            }
            execRes.data = [];
            execRes.tableColumn = [];
            execRes.table = '';
            const err = e as Record<string, unknown>;
            // 要实时响应，故需要用索引改变数据才生效
            if (isAbortError(e)) {
                // 操作者点了取消（请求被中断）：不是失败也不产出结果，
                // 落到取消分支如实显示，而不是空的 errorMsg 被当成执行成功
                state.execResTabs[i].cancelled = true;
            } else if (err.warnAck) {
                // 「仅提醒」命中而未执行：走提醒语义，不能渲染成红色的执行失败
                state.execResTabs[i].notice = err.msg as string;
            } else {
                state.execResTabs[i].errorMsg = errorText(e);
            }
            return;
        } finally {
            if (isCurrentRun(id, gen)) {
                loadingRef.value = false;
                execRes.sql = sql;
                execRes.execTime = new Date().getTime() - startTime;
                // 执行结束（无论成功/失败/取消）都清空取消函数，避免残留引用导致重复取消
                execRes.abortFn = undefined;
            }
        }

        // 即只有以该字符串开头的sql才可修改表数据内容
        if (sql.startsWith('SELECT *') || sql.startsWith('select *') || sql.startsWith('SELECT\n  *')) {
            const tableName = sql.split(/from/i)[1];
            if (tableName) {
                const tn = tableName.trim().split(' ')[0].split('\n')[0];
                // 去除表名前后的字符`或者"
                execRes.table = tn.replace(/`/g, '').replace(/"/g, '');
            } else {
                execRes.table = '';
            }
        } else {
            execRes.table = '';
        }
    };

    /**
     * 获取sql，如果有鼠标选中，则返回选中内容，否则返回当前光标附近的sql
     */
    const getSql = (): string[] => {
        const monacoEditor = options.monacoEditor;
        // 编辑器还没初始化
        if (!monacoEditor?.getModel()) {
            return [];
        }

        // 按方言切割语义，感知反引号/#注释/dollar-quote/字符串转义中的分号（切割错误会导致执行错误SQL）
        const splitOpts = getDialectCapabilities(getNowDbInst().getDialect()).sqlSplitOptions;

        let sql = '' as string | undefined;
        // 选择选中的sql
        let selection = monacoEditor.getSelection();
        if (selection) {
            sql = monacoEditor.getModel()?.getValueInRange(selection);
            sql = sql?.replace(/(^\s*)/g, '');
        }

        // 如果有选中的内容且不为空，直接返回
        if (sql && sql.trim()) {
            return splitSqlStatements(sql, ';', splitOpts).map((x) => x.text);
        }

        // 没有选中任何内容时，自动选择当前光标所在的SQL语句行
        const currentPosition = monacoEditor.getPosition();
        if (currentPosition) {
            const model = monacoEditor.getModel();
            if (model) {
                const fullSql = model.getValue();
                const sqlStatement = getCurrentStatement(fullSql, currentPosition, model, splitOpts);
                if (sqlStatement) {
                    return [sqlStatement];
                }
            }
        }

        return [];
    };

    /**
     * 获取编辑器内的全部 SQL 语句：忽略选区与光标，按方言切割语义拆分为多条。
     * 供「执行全部」使用，与 getSql（选区优先、否则光标所在单条）互补。
     */
    const getAllSql = (): string[] => {
        const monacoEditor = options.monacoEditor;
        // 编辑器还没初始化
        if (!monacoEditor?.getModel()) {
            return [];
        }

        // 按方言切割语义，感知反引号/#注释/dollar-quote/字符串转义中的分号（切割错误会导致执行错误SQL）
        const splitOpts = getDialectCapabilities(getNowDbInst().getDialect()).sqlSplitOptions;
        const fullSql = monacoEditor.getModel()?.getValue();
        if (!fullSql || !fullSql.trim()) {
            return [];
        }
        return splitSqlStatements(fullSql, ';', splitOpts).map((x) => x.text);
    };

    const changeUpdatedField = (hasUpdatedFields: boolean, dt: ExecResTabLike) => {
        // 存在待提交的单元格变更时，该结果页签才显示提交和取消按钮
        dt.hasUpdatedFields = hasUpdatedFields;
    };

    /**
     * 数据删除事件：真实 DELETE 已由数据网格按全主键列生成并执行，
     * 此处仅把被删行从当前结果页签的本地数据里剔除。
     * 必须按【全部主键列】联合匹配——只按首列会在联合主键下误删共享首列值的其它显示行；
     * 无主键无法可靠定位单行，跳过本地剔除（不兜底首列），交由刷新兜底。
     */
    const onDeleteData = async (deleteDatas: Record<string, unknown>[], dt: ExecResTabLike) => {
        const dbInst = getNowDbInst();
        const keyColumns = await dbInst.loadPrimaryKeys(dbName, dt.table);
        if (keyColumns.length === 0) {
            return;
        }
        const keyNames = keyColumns.map((c) => c.columnName);
        const rowKey = (row: Record<string, unknown>) => keyNames.map((k) => String(row[k])).join('\u0000');
        const deletedKeys = new Set(deleteDatas.map(rowKey));
        dt.data = dt.data.filter((d: Record<string, unknown>) => !deletedKeys.has(rowKey(d)));
    };

    const submitUpdateFields = (dt: ExecResTabLike) => {
        dt?.dbTableRef?.submitUpdateFields();
    };

    const cancelUpdateFields = (dt: ExecResTabLike) => {
        dt?.dbTableRef?.cancelUpdateFields();
    };

    const onRemoveTab = (targetId: number) => {
        let activeTab = state.activeTab;
        const tabs = [...state.execResTabs];
        for (let i = 0; i < tabs.length; i++) {
            const tabId = tabs[i].id;
            if (tabId !== targetId) {
                continue;
            }
            const nextTab = tabs[i + 1] || tabs[i - 1];
            if (nextTab) {
                activeTab = nextTab.id;
            } else {
                activeTab = 0;
            }
            state.execResTabs.splice(i, 1);
            state.activeTab = activeTab;
        }
    };

    const activeTab = () => {
        const resTab = state.execResTabs[state.activeTab - 1];
        if (!resTab || !resTab.dbTableRef) {
            return;
        }
        resTab.dbTableRef?.active();
    };

    return {
        getNowDbInst,
        pushNewTab,
        onRunSql,
        onRunAllSql,
        runSql,
        runMultipleSqls,
        runNonQuerySqls,
        getSql,
        getAllSql,
        changeUpdatedField,
        onDeleteData,
        submitUpdateFields,
        cancelUpdateFields,
        onRemoveTab,
        activeTab,
    };
}
