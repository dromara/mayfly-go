import { dbApi } from './api';
import type { ColumnMetadata, DbInstInfo, DbNamesParam } from './types';
import SqlExecBox from './sql-editor/SqlExecBox';

import { Msg } from '@/hooks/useI18n';
import { DbDialect, getDbDialect, matchNumericType } from './dialect';
import { negotiateCapabilities, type NegotiatedCapabilities, type BackendCapabilities } from './dialect/registry';
import { flexColumnWidth, initColumns } from './core/columnWidth';
import { buildInsertSql, buildUpdateSql, buildDeleteSql, buildCountSql, resolveSchemaPrefix } from './core/sqlGen';

// core 模块
import { Db } from './core/db';
import {
    getOrNewDbInst,
    getCachedDbInst,
    cacheDbInst,
    clearAllDbInstCache,
    getCachedTables,
    setCachedTables,
    removeCachedTables,
    getDbNames,
} from './core/dbCache';

export class DbInst {
    /**
     * 标签路径
     */
    tagPath: string;

    /**
     * 实例id
     */
    id: number;

    /**
     * ip:port
     */
    host: string;

    /**
     * 实例名
     */
    name: string;

    /**
     * 数据库类型, mysql postgres
     */
    type: string;

    /** 兼容版本 */
    version: string;
    /**
     * dbName -> db
     */
    dbs: Map<string, Db> = new Map();

    /** 数据库，多个用空格隔开 */
    databases: string[];

    /**
     * 按库缓存后端原始能力（/capabilities 端点响应）。
     * 首次 loadCapabilities() 时填充，后续复用避免重复请求。
     */
    private _backendCaps?: Map<string, BackendCapabilities>;

    /**
     * 按库缓存协商后的能力（静态方言 ∩ 后端实际）。
     * 与 _backendCaps 同步填充，消费方直接取用无需再次协商。
     */
    private _negotiatedCaps?: Map<string, NegotiatedCapabilities>;

    /** 按库缓存视图列表（表名补全纳入视图）；结构变更时随 invalidateSchemaCache 失效 */
    private _viewsByDb?: Map<string, Array<{ name: string; comment?: string }>>;

    /**
     * 默认查询分页数量
     */
    static DefaultLimit = 25;

    /**
     * 获取指定数据库实例，若不存在则新建并缓存
     * @param dbName 数据库名
     * @returns db实例
     */
    getDb(dbName: string) {
        if (!dbName) {
            throw new Error('dbName不能为空');
        }
        let key = `${this.id}_${dbName}`;
        let db = this.dbs.get(key);
        if (db) {
            return db;
        }
        db = new Db();
        db.name = dbName;
        this.dbs.set(key, db);
        return db;
    }

    // 获取数据库实例方言
    getDialect(): DbDialect {
        return getDbDialect(this.type);
    }

    /**
     * 加载并缓存指定库的后端能力声明（/capabilities 端点），同时完成协商。
     *
     * 首次调用发请求并缓存；后续调用直接返回缓存的协商结果。
     * 协商 = 静态方言能力（天花板）∩ 后端实际能力（约束），
     * 消费方一律用本方法取能力，不分别查前端方言与后端端点再手动合并。
     *
     * @param dbName 数据库名
     * @returns 协商后的能力声明（含 backendFeatures 与 namespace）
     */
    async loadCapabilities(dbName: string): Promise<NegotiatedCapabilities> {
        if (!this._backendCaps) {
            this._backendCaps = new Map();
            this._negotiatedCaps = new Map();
        }
        const cached = this._negotiatedCaps!.get(dbName);
        if (cached) {
            return cached;
        }
        const backend = await dbApi.capabilities.request({ id: this.id, db: dbName });
        this._backendCaps.set(dbName, backend);
        const negotiated = negotiateCapabilities(this.getDialect(), backend);
        this._negotiatedCaps!.set(dbName, negotiated);
        return negotiated;
    }

    /**
     * 获取已缓存的协商能力（同步）。未加载过返回 undefined。
     * 调用方先检查返回值，未缓存时先调 loadCapabilities()。
     */
    getNegotiatedCapabilities(dbName: string): NegotiatedCapabilities | undefined {
        return this._negotiatedCaps?.get(dbName);
    }

    /**
     * 加载数据库表信息：命中本地缓存则直接用，否则拉取并写入缓存。
     *
     * 不提供「强制重拉」入参：结构新鲜度由事件驱动（DDL 执行/树节点重载→{@link invalidateSchemaCache}），
     * 带表名过滤的搜索走 dbApi.tableInfos 的 `like` 下推（见资源树表菜单节点），与本全量缓存无关。
     */
    async loadTables(dbName: string) {
        const db = this.getDb(dbName);
        const key = this.dbTablesKey(dbName);
        const cached = getCachedTables(key);
        if (cached) {
            db.tables = cached;
            return cached;
        }
        // 重置按表列缓存，使表清单重新拉取后 SQL 补全的列也重新取
        db.columnsMap?.clear();
        const tables = await dbApi.tableInfos.request({ id: this.id, db: dbName });
        setCachedTables(key, tables);
        db.tables = tables;
        return tables;
    }

    /**
     * 获取表的所有列信息
     * @param dbName 数据库名
     * @param table 表名
     */
    async loadColumns(dbName: string, table: string) {
        const db = this.getDb(dbName);
        // 优先从 table map中获取
        let columns = db.getColumns(table);
        if (columns && columns.length > 0) {
            return columns;
        }
        columns = await dbApi.columnMetadata.request({
            id: this.id,
            db: dbName,
            tableName: table,
        });

        DbInst.initColumns(columns);

        db.columnsMap.set(table, columns);
        return columns;
    }

    /**
     * 获取指定表的指定列信息
     * @param table 表名
     */
    async loadTableColumn(dbName: string, table: string, columnName?: string) {
        // 确保该表的列信息都已加载
        await this.loadColumns(dbName, table);
        return this.getDb(dbName).getColumn(table, columnName);
    }

    /**
     * 获取指定表的全部主键列（联合主键返回多列）；无主键返回空数组
     */
    async loadPrimaryKeys(dbName: string, table: string): Promise<ColumnMetadata[]> {
        await this.loadColumns(dbName, table);
        return this.getDb(dbName).getPrimaryKeys(table);
    }

    dbTablesKey(dbName: string) {
        return `db-tables_${this.id}_${dbName}`;
    }

    /**
     * 失效指定库的表列表 + 按表列 + 视图本地缓存。
     *
     * 表结构变更（建表 / 改表 / 删表 / 执行 DDL、新增或修改列注释）后必须调用：
     * 否则 SQL 补全会命中过期的缓存，出现「编辑器里已填了列注释、补全却不显示」「新建表/视图补全不出来」等假象。
     * 清缓存后，下次 loadTables / loadColumns / loadViews 未命中即重新拉取最新元数据。
     */
    invalidateSchemaCache(dbName: string) {
        removeCachedTables(this.dbTablesKey(dbName));
        // 清空按表列缓存与视图缓存，使改表后 SQL 补全的按表列/视图重新拉取
        this.getDb(dbName).columnsMap?.clear();
        this._viewsByDb?.delete(dbName);
    }

    /**
     * 加载指定库的视图列表（名称 + 注释），供 SQL 补全把视图与表一起联想。
     * 方言不支持视图内省或请求失败时返回空数组（不影响表补全）。
     */
    async loadViews(dbName: string): Promise<Array<{ name: string; comment?: string }>> {
        if (!this._viewsByDb) {
            this._viewsByDb = new Map();
        }
        const cached = this._viewsByDb.get(dbName);
        if (cached) {
            return cached;
        }
        try {
            const objs = await dbApi.metaObjects.request({ id: this.id, db: dbName, kind: 'view' });
            const views = (objs ?? []).map((o) => ({ name: o.name, comment: o.comment }));
            this._viewsByDb.set(dbName, views);
            return views;
        } catch {
            return [];
        }
    }

    /**
     * 执行sql
     *
     * @param sql sql
     * @param remark 执行备注
     */
    async runSql(dbName: string, sql: string, remark: string = '') {
        const res = await dbApi.sqlExec.request({
            id: this.id,
            db: dbName,
            sql: sql.trim(),
            remark,
        });
        for (let re of res) {
            if (re.errorMsg) {
                Msg.error('db.sqlExecFailDetail', { sql: re.sql, error: re.errorMsg });
            }
        }
        return res;
    }

    /**
     * 执行sql(可取消的)
     *
     * @param sql sql
     * @param remark 执行备注
     * @param warnAck 命中「仅提醒」时的处理方式，两个位都由调用方显式声明：
     *   - ask：本次能不能弹确认框。批量选区在客户端也是一条一条发请求的，
     *     逐条弹窗没人受得了，所以批量传 false（提醒只回显、不阻断）
     *   - acknowledged：操作者已在确认框里选了「直接执行」，后端据此不再追问。
     *     它挡不住「需审批」「禁止执行」，只影响提醒级别
     */
    execSql(dbName: string, sql: string, remark: string = '', warnAck: { ask: boolean; acknowledged?: boolean } = { ask: true }) {
        let dbId = this.id;
        return dbApi.sqlExec.useApi({
            id: dbId,
            db: dbName,
            sql: sql.trim(),
            remark,
            askWarn: warnAck.ask,
            ackWarn: warnAck.acknowledged === true,
        });
    }

    /**
     * 获取count sql
     * @param table 表名
     * @param condition 条件
     * @returns count sql
     */
    getDefaultCountSql = (table: string, condition?: string) => {
        return buildCountSql(table, this.wrapName, condition);
    };

    // 获取指定表的默认查询sql
    getDefaultSelectSql(db: string, table: string, condition: string, orderBy: string, pageNum: number, limit: number = DbInst.DefaultLimit) {
        return this.getDialect().getDefaultSelectSql(db, table, condition, orderBy, pageNum, limit);
    }

    /**
     * 生成指定数据的insert语句
     * @param dbName 数据库名
     * @param table 表名
     * @param datas 要生成的数据
     * @param skipNull 是否跳过空字段
     */
    async genInsertSql(dbName: string, table: string, datas: Record<string, unknown>[], skipNull = false) {
        if (!datas) {
            return '';
        }
        const schema = resolveSchemaPrefix(dbName, this.wrapName);
        const columns = await this.loadColumns(dbName, table);
        return buildInsertSql(this.getDialect(), schema, table, columns, datas, skipNull, this.wrapName);
    }

    /**
     * 生成根据主键更新语句
     * @param dbName 数据库名
     * @param table 表名
     * @param columnValue 要更新的列以及对应的值 field->columnName; value->columnValue
     * @param rowData 表的一行完整数据（需要获取主键信息）
     */
    async genUpdateSql(dbName: string, table: string, columnValue: Record<string, unknown>, rowData: Record<string, unknown>) {
        const schema = resolveSchemaPrefix(dbName, this.wrapName);
        const keyColumns = await this.loadPrimaryKeys(dbName, table);
        if (keyColumns.length === 0) {
            return '';
        }
        // 加载更新列的数据类型（用于 wrapValue）
        const updateDataTypes: Record<string, string> = {};
        for (const k of Object.keys(columnValue)) {
            const col = await this.loadTableColumn(dbName, table, k);
            updateDataTypes[k] = col?.dataType ?? 'varchar';
        }
        return buildUpdateSql(this.getDialect(), schema, table, columnValue, updateDataTypes, keyColumns, rowData, this.wrapName);
    }

    /**
     * 生成根据主键删除的sql语句
     * @param db 数据库名
     * @param table 表名
     * @param datas 要删除的记录
     */
    async genDeleteByPrimaryKeysSql(db: string, table: string, datas: Record<string, unknown>[]) {
        const keyColumns = await this.loadPrimaryKeys(db, table);
        return buildDeleteSql(this.getDialect(), table, keyColumns, datas, this.wrapName);
    }

    /*
     * 弹框提示是否执行sql
     */
    promptExeSql = (db: string, sql: string, cancelFunc: Function | undefined = undefined, successFunc: Function | undefined = undefined) => {
        SqlExecBox({
            sql,
            dbId: this.id,
            db,
            formatDialect: this.getDialect().getInfo().formatSqlDialect,
            runSuccessCallback: successFunc,
            cancelCallback: cancelFunc,
        });
    };

    /**
     * 包裹数据库表名、字段名等，避免使用关键字为字段名或表名时报错
     * @param name 表名、字段名、schema名
     * @returns 包裹后的字符串
     */
    wrapName = (name: string) => {
        return this.getDialect().quoteIdentifier(name);
    };

    /**
     * 判断sql是否为查询类sql
     * @param sql sql
     * @returns
     */
    isQuerySql(sql: string) {
        // 简单截取前十个字符
        const sqlPrefix = sql.slice(0, 10).toLowerCase();
        const nonQuery =
            sqlPrefix.startsWith('update') ||
            sqlPrefix.startsWith('insert') ||
            sqlPrefix.startsWith('delete') ||
            sqlPrefix.startsWith('alter') ||
            sqlPrefix.startsWith('drop') ||
            sqlPrefix.startsWith('create') ||
            sqlPrefix.startsWith('truncate') ||
            sqlPrefix.startsWith('rename') ||
            // 各方言的注释语句（comment on）及授权语句均为非查询类，由后端按方言 DDL 执行
            sqlPrefix.startsWith('comment') ||
            sqlPrefix.startsWith('grant') ||
            sqlPrefix.startsWith('revoke');
        return !nonQuery;
    }

    /**
     * 获取或新建dbInst，如果缓存中不存在则新建，否则直接返回
     * @param inst 数据库实例，后端返回的列表接口中的信息
     * @returns DbInst
     */
    static getOrNewInst(inst: DbInstInfo): DbInst {
        // 返回类型由工厂推断为 DbInst，缓存层按 CachedDbInst 契约存放，无需在调用点断言
        return getOrNewDbInst(inst, (instInfo) => {
            const dbInst = new DbInst();
            dbInst.tagPath = instInfo.tagPath || '';
            dbInst.id = instInfo.id;
            dbInst.host = instInfo.host || '';
            dbInst.name = instInfo.name || '';
            dbInst.type = instInfo.type || '';
            dbInst.databases = instInfo.databases || [];
            cacheDbInst(dbInst.id, dbInst);
            return dbInst;
        });
    }

    /**
     * 获取数据库实例id，若不存在，则新建一个并缓存
     * @param dbId 数据库实例id
     * @returns 数据库实例
     */
    static getInst(dbId?: number): DbInst {
        if (!dbId) {
            throw new Error('dbId不能为空');
        }
        const dbInst = getCachedDbInst<DbInst>(dbId);
        if (dbInst) {
            return dbInst;
        }
        throw new Error('dbInst不存在! 请在合适调用点使用DbInst.getInstA()新建该实例');
    }

    /**
     * 失效指定库的本地元数据缓存（表清单 / 按表列 / 视图）。
     *
     * 供「结构变更后清缓存」的通用调用点（SQL 执行弹框、资源树重载等）使用：
     * 实例尚未在客户端缓存时无缓存可失效，故为 no-op，不像 {@link getInst} 那样抛错而打断调用方的成功回调。
     */
    static invalidateSchema(dbId?: number, dbName?: string) {
        if (!dbId || !dbName) {
            return;
        }
        getCachedDbInst<DbInst>(dbId)?.invalidateSchemaCache(dbName);
    }

    /**
     * 获取数据库实例信息，若不存在，调接口获取数据库信息
     * @param dbId 数据库id
     * @returns
     */
    static async getInstA(dbId?: number): Promise<DbInst> {
        if (!dbId) {
            throw new Error('dbId不能为空');
        }
        const dbInst = getCachedDbInst<DbInst>(dbId);
        if (dbInst) {
            return dbInst;
        }

        const dbInfoRes = await dbApi.dbs.request({ id: dbId });
        const db = dbInfoRes.list[0];
        return DbInst.getOrNewInst(db);
    }

    /**
     * 清空所有实例缓存信息
     */
    static clearAll() {
        clearAllDbInstCache();
    }

    /**
     * 判断字段类型是否为数字类型
     *
     * 实现已下沉至 dialect/shared/utils 的 matchNumericType，此处仅委托以兼容既有调用方；
     * 方言层直接使用 matchNumericType，避免 dialect → db 的反向依赖（模块循环 + TDZ）。
     * @param columnType 字段类型
     * @returns 匹配结果，非数字类型时为 null
     */
    static isNumber(columnType: string) {
        return matchNumericType(columnType);
    }

    /**
     *
     * @param str 字符串
     * @param tableData 表数据
     * @param flag 标志
     * @returns 列宽度
     */
    static flexColumnWidth = flexColumnWidth;

    // 初始化所有列信息，完善需要显示的列类型，包含长度等，如varchar(20)
    static initColumns = initColumns;

    /**
     * 根据数据库配置信息获取对应的库名列表
     * @param db db配置信息
     * @returns 库名列表
     */
    static async getDbNames(db: DbNamesParam) {
        return getDbNames(db);
    }
}

/**
 * 数据库主题配置
 */
export const DbThemeConfig = {
    /**
     * 表数据表头是否显示备注
     */
    showColumnComment: true,

    /**
     * 是否自动定位至树节点
     */
    locationTreeNode: true,
};
