/* eslint-disable no-unused-vars */
import { dbApi } from './api';
import { getTextWidth } from '@/common/utils/string';
import SqlExecBox from './component/SqlExecBox';

const dbInstCache: Map<number, DbInst> = new Map();

export class DbInst {
    /**
     * 标签路径
     */
    tagPath: string

    /**
     * 实例id
     */
    id: number

    /**
     * 实例名
     */
    name: string

    /**
     * 数据库类型, mysql postgres
     */
    type: string

    /**
     * schema -> db
     */
    dbs: Map<string, Db> = new Map()

    /** 数据库schema，多个用空格隔开 */
    databases: string

    /**
     * 默认查询分页数量
     */
    static DefaultLimit = 20;

    /**
     * 获取指定数据库实例，若不存在则新建并缓存
     * @param dbName 数据库名
     * @returns db实例
     */
    getDb(dbName: string) {
        if (!dbName) {
            throw new Error('dbName不能为空')
        }
        let db = this.dbs.get(dbName)
        if (db) {
            return db;
        }
        console.info(`new db -> dbId: ${this.id}, dbName: ${dbName}`);
        db = new Db();
        db.name = dbName;
        this.dbs.set(dbName, db);
        return db;
    }

    /**
     * 加载数据库表信息
     * @param dbName 数据库名
     * @param reload 是否重新请求接口获取数据
     * @returns 表信息
     */
    async loadTables(dbName: string, reload?: boolean) {
        const db = this.getDb(dbName);
        // 优先从 table map中获取
        let tables = db.tables;
        if (!reload && tables) {
            return tables;
        }
        // 重置列信息缓存与表提示信息
        db.columnsMap?.clear();
        db.tableHints = null;
        console.log(`load tables -> dbName: ${dbName}`);
        tables = await dbApi.tableMetadata.request({ id: this.id, db: dbName });
        db.tables = tables;
        return tables;
    }

    /**
     * 获取表的所有列信息
     * @param table 表名
     */
    async loadColumns(dbName: string, table: string) {
        const db = this.getDb(dbName);
        // 优先从 table map中获取
        let columns = db.getColumns(table);
        if (columns) {
            
            return columns;
        }
        console.log(`load columns -> dbName: ${dbName}, table: ${table}`);
        columns = await dbApi.columnMetadata.request({
            id: this.id,
            db: dbName,
            tableName: table,
        });
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

    /**
     * 获取库信息提示
     */
    async loadDbHints(dbName: string) {
        const db = this.getDb(dbName);
        if (db.tableHints) {
            return db.tableHints;
        }
        console.log(`load db-hits -> dbName: ${dbName}`);
        const hits = await dbApi.hintTables.request({ id: this.id, db: db.name, })
        db.tableHints = hits;
        return hits;
    }

    /**
    * 执行sql
    *
    * @param sql sql
    * @param remark 执行备注
    */
    async runSql(dbName: string, sql: string, remark: string = '') {
        return await dbApi.sqlExec.request({
            id: this.id,
            db: dbName,
            sql: sql.trim(),
            remark,
        });
    }

    // 获取指定表的默认查询sql
    getDefaultSelectSql(table: string, condition: string, orderBy: string, pageNum: number, limit: number = DbInst.DefaultLimit) {
        const baseSql = `SELECT * FROM ${table} ${condition ? 'WHERE ' + condition : ''} ${orderBy ? orderBy : ''}`;
        if (this.type == 'mysql') {
            return `${baseSql} LIMIT ${(pageNum - 1) * limit}, ${limit};`;
        }
        if (this.type == 'postgres') {
            return `${baseSql} OFFSET ${(pageNum - 1) * limit} LIMIT ${limit};`;
        }
        return baseSql;
    }

    /**
     * 生成指定数据的insert语句
     * @param dbName 数据库名
     * @param table 表名
     * @param datas 要生成的数据
     */
    genInsertSql(dbName: string, table: string, datas: any[]): string {
        if (!datas) {
            return '';
        }
        const columns = this.getDb(dbName).getColumns(table);
        const sqls = [];
        for (let data of datas) {
            let colNames = [];
            let values = [];
            for (let column of columns) {
                const colName = column.columnName;
                colNames.push(colName);
                values.push(DbInst.wrapValueByType(data[colName]));
            }
            sqls.push(`INSERT INTO ${table} (${colNames.join(', ')}) VALUES(${values.join(', ')})`);
        }
        return sqls.join(';\n') + ';'
    }

    /**
     * 生成根据主键删除的sql语句
     * @param table 表名
     * @param datas 要删除的记录
     */
    genDeleteByPrimaryKeysSql(db: string, table: string, datas: any[]) {
        const primaryKey = this.getDb(db).getColumn(table);
        const primaryKeyColumnName = primaryKey.columnName;
        const ids = datas.map((d: any) => `${DbInst.wrapColumnValue(primaryKey.columnType, d[primaryKeyColumnName])}`).join(',');
        return `DELETE FROM ${table} WHERE ${primaryKeyColumnName} IN (${ids})`;
    }

    /*
    * 弹框提示是否执行sql
    */
    promptExeSql = (db: string, sql: string, cancelFunc: any = null, successFunc: any = null) => {
        SqlExecBox({
            sql, dbId: this.id, db,
            runSuccessCallback: successFunc,
            cancelCallback: cancelFunc,
        });
    };

    /**
     * 获取或新建dbInst，如果缓存中不存在则新建，否则直接返回
     * @param inst 数据库实例，后端返回的列表接口中的信息
     * @returns DbInst
     */
    static getOrNewInst(inst: any) {
        if (!inst) {
            throw new Error('inst不能为空')
        }
        let dbInst = dbInstCache.get(inst.id);
        if (dbInst) {
            return dbInst;
        }
        console.info(`new dbInst: ${inst.id}, tagPath: ${inst.tagPath}`);
        dbInst = new DbInst();
        dbInst.tagPath = inst.tagPath;
        dbInst.id = inst.id;
        dbInst.name = inst.name;
        dbInst.type = inst.type;
        dbInst.databases = inst.databases;

        dbInstCache.set(dbInst.id, dbInst);
        return dbInst;
    }

    /**
    * 获取数据库实例id，若不存在，则新建一个并缓存
    * @param dbId 数据库实例id
    * @param dbType 第一次获取时为必传项，即第一次创建时
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
        throw new Error('dbInst不存在! 请在合适调用点使用DbInst.newInst()新建该实例');
    }

    /**
     * 清空所有实例缓存信息
     */
    static clearAll() {
        dbInstCache.clear();
    }

    /**
    * 获取count sql
    * @param table 表名
    * @param condition 条件
    * @returns count sql
    */
    static getDefaultCountSql = (table: string, condition?: string) => {
        return `SELECT COUNT(*) count FROM ${table} ${condition ? 'WHERE ' + condition : ''}`;
    };

    /**
     * 根据返回值包装值，若值为字符串类型则添加''
     * @param val 值
     * @returns 包装后的值
     */
    static wrapValueByType = (val: any) => {
        if (val == null) {
            return 'NULL';
        }
        if (typeof val == 'number') {
            return val;
        }
        return `'${val}'`;
    };

    /**
    * 根据字段类型包装字段值，如为字符串等则添加‘’，数字类型则直接返回即可
    */
    static wrapColumnValue(columnType: string, value: any) {
        if (this.isNumber(columnType)) {
            return value;
        }
        return `'${value}'`;
    };

    /**
     * 判断字段类型是否为数字类型
     * @param columnType 字段类型
     * @returns
     */
    static isNumber(columnType: string) {
        return columnType.match(/int|double|float|nubmer|decimal|byte|bit/gi);
    };

    /**
     *
     * @param str 字符串
     * @param tableData 表数据
     * @param flag 标志
     * @returns 列宽度
     */
    static flexColumnWidth = (prop: any, tableData: any) => {
        if (!prop || !prop.length || prop.length === 0 || prop === undefined) {
            return;
        }

        // 获取列名称的长度 加上排序图标长度
        const columnWidth: number = getTextWidth(prop) + 40;
        // prop为该列的字段名(传字符串);tableData为该表格的数据源(传变量);
        if (!tableData || !tableData.length || tableData.length === 0 || tableData === undefined) {
            return columnWidth;
        }

        // 获取该列中最长的数据(内容)
        let maxWidthText = ""
        let maxWidthValue
        // 获取该列中最长的数据(内容)
        for (let i = 0; i < tableData.length; i++) {
            let nowValue = tableData[i][prop]
            if (!nowValue) {
                continue;
            }
            // 转为字符串比较长度
            let nowText = nowValue + "";
            if (nowText.length > maxWidthText.length) {
                maxWidthText = nowText;
                maxWidthValue = nowValue;
            }
        }
        const contentWidth: number = getTextWidth(maxWidthText) + 15;
        const flexWidth: number = contentWidth > columnWidth ? contentWidth : columnWidth;
        return flexWidth > 500 ? 500 : flexWidth;
    };
}

/**
 * 数据库实例信息
 */
class Db {
    name: string  // 库名
    tables: []   // 数据库实例表信息
    columnsMap: Map<string, any> = new Map  // table -> columns
    tableHints: any = null // 提示词

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
        if (!columnName) {
            const col = cols.find((c: any) => c.columnKey == 'PRI');
            return col || cols[0];
        }
        return cols.find((c: any) => c.columnName == columnName);
    }
}

export enum TabType {
    /**
     * 表数据
     */
    TableData,

    /**
     * 查询框
     */
    Query,
}

export class TabInfo {
    /**
     * tab唯一key。与label、name都一致
     */
    key: string

    /**
     * 菜单树节点key
     */
    treeNodeKey: string

    /**
     * 数据库实例id
     */
    dbId: number

    /**
     * 库名
     */
    db: string = ''

    /**
     * tab 类型
     */
    type: TabType

    /**
     * tab需要的其他信息
     */
    params: any

    getNowDbInst() {
        return DbInst.getInst(this.dbId);
    }

    getNowDb() {
        return this.getNowDbInst().getDb(this.db);
    }
}

/** 修改表字段所需数据 */
export type UpdateFieldsMeta = {
    // 主键值
    primaryKey: string
    // 主键名
    primaryKeyName: string
    // 主键类型
    primaryKeyType: string
    // 新值
    fields: FieldsMeta[]
}

export type FieldsMeta = {
    // 字段所在div
    div: HTMLElement
    // 字段名
    fieldName: string
    // 字段所在的表格行数据
    row: any
    // 字段类型
    fieldType: string
    // 原值
    oldValue: string
    // 新值
    newValue: string
}
