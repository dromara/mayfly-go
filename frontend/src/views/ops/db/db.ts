import { languages, type IRange } from 'monaco-editor';
import { dbApi } from './api';
import type { DbTableInfo, ColumnMetadata, DbInstInfo, DbNamesParam } from './types';
import SqlExecBox from './component/sqleditor/SqlExecBox';

import { Msg } from '@/hooks/useI18n';
import { type RemovableRef, useLocalStorage } from '@vueuse/core';
import { DbDialect, getDbDialect } from './dialect';
import { DbGetDbNamesMode } from './enums';
import { flexColumnWidth, initColumns } from './utils/columnWidth';

const hintsStorage: RemovableRef<Map<string, Record<string, string[]>>> = useLocalStorage('db-table-hints', new Map());
const tableStorage: RemovableRef<Map<string, DbTableInfo[]>> = useLocalStorage('db-tables', new Map());

const dbInstCache: Map<number, DbInst> = new Map();

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
        let tables = tableStorage.value.get(key);
        // 优先从 table 缓存中获取
        if (!reload && tables) {
            db.tables = tables;
            return tables;
        }
        // 重置列信息缓存与表提示信息
        db.columnsMap?.clear();
        tables = await dbApi.tableInfos.request({ id: this.id, db: dbName });
        tableStorage.value.set(key, tables);
        db.tables = tables;

        // 异步加载表提示信息
        this.loadDbHints(dbName, true).then(() => {});
        return tables;
    }

    async loadTableSuggestions(dbDialect: DbDialect, dbName: string, range: IRange, reload?: boolean) {
        const tables = await this.loadTables(dbName, reload);
        // 表名联想
        let suggestions: languages.CompletionItem[] = [];
        tables?.forEach((tableMeta: DbTableInfo, index: number) => {
            const { tableName, tableComment } = tableMeta;
            suggestions.push({
                label: {
                    label: tableName + ' - ' + tableComment,
                    description: 'table',
                },
                kind: languages.CompletionItemKind.File,
                detail: tableComment,
                insertText: dbDialect.quoteIdentifier(tableName),
                range,
                sortText: 300 + index + '',
            });
        });
        return { suggestions };
    }

    /** 加载列信息提示 */
    async loadTableColumnSuggestions(dbDialect: DbDialect, db: string, tableName: string, range: IRange) {
        let dbHits = await this.loadDbHints(db);
        let columns = dbHits[tableName];
        let suggestions: languages.CompletionItem[] = [];
        columns?.forEach((a: string, index: number) => {
            // 字段数据格式  字段名 字段注释，  如： create_time  [datetime][创建时间]
            const nameAndComment = a.split('  ');
            const fieldName = nameAndComment[0];
            suggestions.push({
                label: {
                    label: a,
                    description: 'column',
                },
                kind: languages.CompletionItemKind.Property,
                detail: '', // 不显示detail, 否则选中时备注等会被遮挡
                insertText: dbDialect.quoteIdentifier(fieldName), // create_time
                range,
                sortText: 100 + index + '', // 使用表字段声明顺序排序,排序需为字符串类型
            });
        });

        return { suggestions };
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
     * 获取指定表的指定信息
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
        let hints = hintsStorage.value.get(key);
        if (!reload && hints) {
            db.tableHints = hints;
            return hints;
        }
        hints = (await dbApi.hintTables.request({ id: this.id, db: db.name })) as unknown as Record<string, string[]>;
        db.tableHints = hints;
        hintsStorage.value.set(key, hints);
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
     * @param dbDialect db方言
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
            dbType: this.getDialect().getInfo().formatSqlDialect,
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
    static getOrNewInst(inst: DbInstInfo) {
        if (!inst) {
            throw new Error('inst不能为空');
        }
        let dbInst = dbInstCache.get(inst.id);
        if (dbInst) {
            // 可能同一个库关联多个标签，展示需要
            if (inst.tagPath) {
                dbInst.tagPath = inst.tagPath;
            }

            return dbInst;
        }
        dbInst = new DbInst();
        dbInst.tagPath = inst.tagPath || '';
        dbInst.id = inst.id;
        dbInst.host = inst.host || '';
        dbInst.name = inst.name || '';
        dbInst.type = inst.type || '';
        dbInst.databases = inst.databases || [];

        if (dbInst.databases?.[0]) {
            dbApi.getCompatibleDbVersion.request({ id: inst.id, db: dbInst.databases?.[0] }).then((version) => {
                dbInst.version = version;
            });
        }

        dbInstCache.set(dbInst.id, dbInst);
        return dbInst;
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
        let dbInst = dbInstCache.get(dbId);
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
        let dbInst = dbInstCache.get(dbId);
        if (dbInst) {
            return Promise.resolve(dbInst);
        }

        const dbInfoRes = await dbApi.dbs.request({ id: dbId });
        const db = dbInfoRes.list[0];
        return Promise.resolve(DbInst.getOrNewInst(db));
    }

    /**
     * 清空所有实例缓存信息
     */
    static clearAll() {
        dbInstCache.clear();
    }

    /**
     * 判断字段类型是否为数字类型
     * @param columnType 字段类型
     * @returns
     */
    static isNumber(columnType: string) {
        return columnType && columnType.match(/(int|uint|double|float|number|numeric|decimal|byte|bit)/gi);
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
        if (db.getDatabaseMode == DbGetDbNamesMode.Assign.value) {
            return (db.database as string).split(' ');
        }

        return await dbApi.getDbNamesByAc.request({ authCertName: db.authCertName });
    }
}

/**
 * 数据库实例信息
 */
class Db {
    name: string; // 库名
    tables: DbTableInfo[]; // 数据库实例表信息
    columnsMap: Map<string, ColumnMetadata[]> = new Map(); // table -> columns
    tableHints: Record<string, string[]> | null = null; // 提示词

    /**
     * 获取指定表列信息（前提需要dbInst.loadColumns）
     * @param table 表名
     */
    getColumns(table: string) {
        return this.columnsMap.get(table);
    }

    /**
     * 获取指定表中的指定列名信息，若列名为空则默认返回主键
     * @param table 表名
     * @param columnName 列名
     */
    getColumn(table: string, columnName: string = '') {
        const cols = this.getColumns(table);
        if (!cols) {
            return undefined;
        }
        if (!columnName) {
            const col = cols.find((c: ColumnMetadata) => c.isPrimaryKey);
            return col || cols[0];
        }
        return cols.find((c: ColumnMetadata) => c.columnName == columnName);
    }
}

// Re-exports for backward compatibility
export { TabType, TabInfo } from './models/TabInfo';
export type { TabParams, TabComponentRef } from './models/TabInfo';
export { registerDbCompletionItemProvider } from './services/completionService';

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
