import { notBlank } from '@/common/assert';
import { Msg } from '@/hooks/useI18n';
import { ElMessageBox } from 'element-plus';
import type { editor, IRange } from 'monaco-editor';
import { unref, type Ref, type UnwrapNestedRefs } from 'vue';
import { useI18n } from 'vue-i18n';

import { DbInst } from '../../db';
import { getDialectCapabilities } from '../../dialect';
import { isDdlSql } from '../../core/sqlKind';
import type { SqlExecRes, SqlExecResColumn, TableColumnDef } from '../../types';
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

    abortFn: Function;

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
    monacoEditor: editor.IStandaloneCodeEditor | null;
}

export function useSqlExec(options: UseSqlExecOptions) {
    const { t } = useI18n();
    const { dbId, dbName, state } = options;

    const getNowDbInst = () => {
        return DbInst.getInst(dbId);
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
        notBlank(sqls, emptyMsg);

        if (sqls.length == 1) {
            const oneSql = sqls[0];
            let execRemark;
            if (!getNowDbInst().isQuerySql(oneSql)) {
                const res = await ElMessageBox.prompt(t('db.enterExecRemarkTips'), 'Tip', {
                    confirmButtonText: t('common.confirm'),
                    cancelButtonText: t('common.cancel'),
                    inputErrorMessage: t('db.execRemarkPlaceholder'),
                });
                execRemark = res.value;
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

        try {
            execRes.errorMsg = '';
            execRes.sql = sqls.join('\n\n---\n\n'); // 显示所有SQL

            // 执行所有非查询SQL
            const results: Record<string, unknown>[] = [];
            for (const sql of sqls) {
                try {
                    const { data, execute } = getNowDbInst().execSql(dbName, sql, '');
                    await execute();
                    const result = (data.value as SqlExecRes[])[0];
                    results.push({
                        sql: result.sql,
                        rowsAffected: result.res?.[0]?.rowsAffected,
                        error: result.errorMsg || '-',
                    });
                } catch (error: unknown) {
                    results.push({
                        sql: sql,
                        error: error instanceof Error ? error.message : String(error),
                    });
                }
            }

            // 设置表格列
            state.execResTabs[i].tableColumn = [
                { columnName: 'SQL', key: 'sql', columnType: 'string', show: true },
                { columnName: 'RowsAffected', key: 'rowsAffected', columnType: 'number', show: true },
                { columnName: 'Error', key: 'error', columnType: 'string', show: true },
            ];

            state.execResTabs[i].data = results;
            cancelUpdateFields(execRes);
            // 批量执行含 DDL：失效补全元数据缓存，反映最新表/字段/注释
            if (sqls.some(isDdlSql)) {
                getNowDbInst().invalidateSchemaCache(dbName);
            }
        } catch (e: unknown) {
            execRes.data = [];
            execRes.tableColumn = [];
            execRes.table = '';
            state.execResTabs[i].errorMsg = e instanceof Error ? e.message : String(e);
            return;
        } finally {
            execRes.execTime = new Date().getTime() - startTime;
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
        try {
            execRes.errorMsg = '';
            execRes.sql = '';

            const { data, execute, isFetching, abort } = getNowDbInst().execSql(dbName, sql, remark);
            execRes.loading = isFetching;
            execRes.abortFn = abort;

            await execute();
            const colAndData = (data.value as SqlExecRes[])[0];
            if (colAndData.errorMsg) {
                throw { msg: colAndData.errorMsg };
            }

            if (colAndData.res?.length == 0) {
                state.tableDataEmptyText = 'No Data';
            }

            // 要实时响应，故需要用索引改变数据才生效
            state.execResTabs[i].data = colAndData.res ?? [];
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
            cancelUpdateFields(execRes);
            // 执行了 DDL：失效补全元数据缓存，使表名/字段/注释联想反映最新结构
            if (isDdlSql(sql)) {
                getNowDbInst().invalidateSchemaCache(dbName);
            }
        } catch (e: unknown) {
            execRes.data = [];
            execRes.tableColumn = [];
            execRes.table = '';
            // 要实时响应，故需要用索引改变数据才生效
            state.execResTabs[i].errorMsg = (e as Record<string, unknown>).msg as string;
            return;
        } finally {
            execRes.sql = sql;
            execRes.execTime = new Date().getTime() - startTime;
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
