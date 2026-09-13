import { dbApi } from './api';
import type { ColumnMetadata, DbInstInfo, DbNamesParam } from './types';
import SqlExecBox from './sql-editor/SqlExecBox';

import { Msg } from '@/hooks/useI18n';
import { DbDialect, getDbDialect, matchNumericType } from './dialect';
import { flexColumnWidth, initColumns } from './core/columnWidth';

// core 模块
import { Db } from './core/db';
import {
    getOrNewDbInst,
    getCachedDbInst,
    cacheDbInst,
    clearAllDbInstCache,
    getCachedTables,
    setCachedTables,
    getCachedHints,
    setCachedHints,
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
     * 加载数据库表信息
     * @param dbName 数据库名
     * @param reload 是否重新请求接口获取数据
     * @returns 表信息
     */
    async loadTables(dbName: string, reload?: boolean) {
        const db = this.getDb(dbName);
        let key = this.dbTablesKey(dbName);
        let tables = getCachedTables(key);
        // 优先从 table 缓存中获取
        if (!reload && tables) {
            db.tables = tables;
            return tables;
        }
        // 重置列信息缓存与表提示信息
        db.columnsMap?.clear();
        tables = await dbApi.tableInfos.request({ id: this.id, db: dbName });
        setCachedTables(key, tables);
        db.tables = tables;

        // 异步加载表提示信息
        this.loadDbHints(dbName, true).then(() => {});
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

    dbTableHintsKey(dbName: string) {
        return `db-table-hints_${this.id}_${dbName}`;
    }

    dbTablesKey(dbName: string) {
        return `db-tables_${this.id}_${dbName}`;
    }

    /**
     * 获取库信息提示
     */
    async loadDbHints(dbName: string, reload?: boolean): Promise<Record<string, string[]>> {
        const db = this.getDb(dbName);
        let key = this.dbTableHintsKey(dbName);
        let hints = getCachedHints(key);
        if (!reload && hints) {
            db.tableHints = hints;
            return hints;
        }
        hints = (await dbApi.hintTables.request({ id: this.id, db: db.name })) as unknown as Record<string, string[]>;
        db.tableHints = hints;
        setCachedHints(key, hints);
        return hints;
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
                Msg.error(`${re.sql} -> 执行失败: ${re.errorMsg}`);
            }
        }
        return res;
    }

    /**
     * 执行sql(可取消的)
     *
     * @param sql sql
     * @param remark 执行备注
     */
    execSql(dbName: string, sql: string, remark: string = '') {
        let dbId = this.id;
        return dbApi.sqlExec.useApi({
            id: dbId,
            db: dbName,
            sql: sql.trim(),
            remark,
        });
    }

    /**
     * 获取count sql
     * @param table 表名
     * @param condition 条件
     * @returns count sql
     */
    getDefaultCountSql = (table: string, condition?: string) => {
        return `SELECT COUNT(*) count
                FROM ${this.wrapName(table)} ${condition ? 'WHERE ' + condition : ''}`;
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
        let schema = '';
        let arr = dbName.split('/');
        if (arr.length == 1) {
            schema = this.wrapName(dbName) + '.';
        } else if (arr.length == 2) {
            schema = this.wrapName(arr[1]) + '.';
        }

        let dbDialect = this.getDialect();
        const columns = await this.loadColumns(dbName, table);
        const sqls = [];
        for (let data of datas) {
            let colNames = [];
            let values = [];
            for (let column of columns) {
                const colName = column.columnName;
                if (skipNull && data[colName] == null) {
                    continue;
                }
                colNames.push(this.wrapName(colName));
                values.push(dbDialect.wrapValue(column.dataType, data[colName]));
            }
            sqls.push(`INSERT INTO ${schema}${this.wrapName(table)} (${colNames.join(', ')})
                       VALUES (${values.join(', ')})`);
        }
        return sqls.join(';\n') + ';';
    }

    /**
     * 生成根据主键更新语句
     * @param dbName 数据库名
     * @param table 表名
     * @param columnValue 要更新的列以及对应的值 field->columnName; value->columnValue
     * @param rowData 表的一行完整数据（需要获取主键信息）
     */
    async genUpdateSql(dbName: string, table: string, columnValue: Record<string, unknown>, rowData: Record<string, unknown>) {
        let schema = '';
        let dbArr = dbName.split('/');
        if (dbArr.length == 2) {
            schema = this.wrapName(dbArr[1]) + '.';
        }

        let sql = `UPDATE ${schema}${this.wrapName(table)}
                   SET `;
        // 主键列信息
        const primaryKey = await this.loadTableColumn(dbName, table);
        let primaryKeyType = primaryKey!.dataType;
        let primaryKeyName = primaryKey!.columnName;
        let primaryKeyValue = rowData[primaryKeyName];
        const dialect = this.getDialect();
        for (let k of Object.keys(columnValue)) {
            const v = columnValue[k];
            // 更新字段列信息
            const updateColumn = await this.loadTableColumn(dbName, table, k);
            sql += ` ${this.wrapName(k)} = ${dialect.wrapValue(updateColumn!.dataType, v)},`;
        }
        sql = sql.substring(0, sql.length - 1);

        return sql + ` WHERE ${this.wrapName(primaryKeyName)} = ${this.getDialect().wrapValue(primaryKeyType, primaryKeyValue)} ;`;
    }

    /**
     * 生成根据主键删除的sql语句
     * @param db 数据库名
     * @param table 表名
     * @param datas 要删除的记录
     */
    async genDeleteByPrimaryKeysSql(db: string, table: string, datas: Record<string, unknown>[]) {
        const primaryKey = await this.loadTableColumn(db, table);
        const primaryKeyColumnName = primaryKey!.columnName;
        const ids = datas.map((d: Record<string, unknown>) => `${this.getDialect().wrapValue(primaryKey!.dataType, d[primaryKeyColumnName])}`).join(',');
        return `DELETE
                FROM ${this.wrapName(table)}
                WHERE ${this.wrapName(primaryKeyColumnName)} IN (${ids})`;
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

    /**
     * 是否缓存表信息
     */
    cacheTable: true,
};
