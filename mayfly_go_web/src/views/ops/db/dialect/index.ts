import { MysqlDialect } from './mysql_dialect';
import { PostgresqlDialect } from './postgres_dialect';
import { DMDialect } from '@/views/ops/db/dialect/dm_dialect';
import { OracleDialect } from '@/views/ops/db/dialect/oracle_dialect';
import { MariadbDialect } from '@/views/ops/db/dialect/mariadb_dialect';
import { SqliteDialect } from '@/views/ops/db/dialect/sqlite_dialect';
import { MssqlDialect } from '@/views/ops/db/dialect/mssql_dialect';
import { GaussDialect } from '@/views/ops/db/dialect/gauss_dialect';
import { KingbaseEsDialect } from '@/views/ops/db/dialect/kingbaseES_dialect';
import { VastbaseDialect } from '@/views/ops/db/dialect/vastbase_dialect';

export interface sqlColumnType {
    udtName: string;
    dataType: string;
    desc: string;
    space: string;
    range?: string;
}

export interface RowDefinition {
    name: string;
    oldName?: string;
    type: string;
    value: string;
    length: string;
    numScale: string;
    notNull: boolean;
    pri: boolean;
    auto_increment: boolean;
    remark: string;
}

export interface IndexDefinition {
    indexName: string;
    columnNames: string[];
    unique: boolean;
    indexType: string;
    indexComment?: string;
}
export const commonCustomKeywords = ['GROUP BY', 'ORDER BY', 'LEFT JOIN', 'RIGHT JOIN', 'INNER JOIN', 'SELECT * FROM'];

export interface EditorCompletionItem {
    /** 用于显示 */
    label: string;
    /** 用于插入编辑器，可预置一些变量方便使用函数 */
    insertText?: string;
    /** 用于描述 */
    description: string;
}

export interface EditorCompletion {
    /** 关键字 */
    keywords: EditorCompletionItem[];
    /** 操作关键字 */
    operators: EditorCompletionItem[];
    /** 函数,包括内置函数和自定义函数 */
    functions: EditorCompletionItem[];
    /** 内置变量 */
    variables: EditorCompletionItem[];
}

// 定义一个数据类型的枚举，包含字符串、数字、日期、时间、日期时间
export enum DataType {
    String = 'string',
    Number = 'number',
    Date = 'date',
    Time = 'time',
    DateTime = 'datetime',
}

/** 列数据类型角标 */
export const ColumnTypeSubscript = {
    /** 字符串 */
    string: 'abc',
    /** 数字 */
    number: '123',
    /** 日期 */
    date: 'icon-clock',
    /** 时间 */
    time: 'icon-clock',
    /** 日期时间 */
    datetime: 'icon-clock',
};

// 数据库基础信息
export interface DialectInfo {
    /**
     * 数据库类型label
     */
    name: string;

    /**
     * 图标
     */
    icon: string;

    /**
     * 默认端口
     */
    defaultPort: number;

    /**
     * 格式化sql的方言
     */
    formatSqlDialect: string;

    /**
     * 列字段类型
     */
    columnTypes: sqlColumnType[];

    /**
     * 编辑器一些固定代码提示（关键字、操作符）
     */
    editorCompletions: EditorCompletion;
}

export const DbType = {
    mysql: 'mysql',
    mariadb: 'mariadb',
    postgresql: 'postgres',
    gauss: 'gauss',
    dm: 'dm', // 达梦
    oracle: 'oracle',
    sqlite: 'sqlite',
    mssql: 'mssql', // ms sqlserver
    kingbaseEs: 'kingbaseEs', // 人大金仓 pgsql模式 https://help.kingbase.com.cn/v8/index.html
    vastbase: 'vastbase', // https://docs.vastdata.com.cn/zh/docs/VastbaseG100Ver2.2.5/doc/%E5%BC%80%E5%8F%91%E8%80%85%E6%8C%87%E5%8D%97/SQL%E5%8F%82%E8%80%83/SQL%E5%8F%82%E8%80%83.html
};

// mysql兼容的数据库
export const noSchemaTypes = [DbType.mysql, DbType.mariadb, DbType.sqlite];

// 有schema层的数据库
export const schemaDbTypes = [DbType.postgresql, DbType.gauss, DbType.dm, DbType.oracle, DbType.mssql, DbType.kingbaseEs, DbType.vastbase];

export const editDbTypes = [...noSchemaTypes, ...schemaDbTypes];

export const compatibleMysql = (dbType: string): boolean => {
    switch (dbType) {
        case DbType.mysql:
        case DbType.mariadb:
            return true;
        default:
            return false;
    }
};

export interface DbDialect {
    /**
     * 获取一些数据库默认信息
     */
    getInfo(): DialectInfo;

    /**
     * 获取默认查询sql
     * @param db  数据库信息
     * @param table  表名
     * @param condition 条件
     * @param orderBy 排序
     * @param pageNum  页数
     * @param limit  条数
     */
    getDefaultSelectSql(db: string, table: string, condition: string, orderBy: string, pageNum: number, limit: number): string;

    getPageSql(pageNum: number, limit: number): string;

    getDefaultRows(): RowDefinition[];

    getDefaultIndex(): IndexDefinition;

    /**
     * 引用标识符，包裹数据库表名、字段名等，避免使用关键字为字段名或表名时报错
     * @param name 名称
     */
    quoteIdentifier(name: string): string;

    /**
     * 生成创建表sql
     * @param tableData 建表数据
     */
    getCreateTableSql(tableData: any): string;

    /**
     * 生成创建索引sql
     * @param tableData
     */
    getCreateIndexSql(tableData: any): string;

    /**
     * 生成编辑列sql
     * @param tableData 表数据，包含表名、列数据、索引数据
     * @param tableName 表名
     * @param changeData 改变信息
     */
    getModifyColumnSql(tableData: any, tableName: string, changeData: { del: RowDefinition[]; add: RowDefinition[]; upd: RowDefinition[] }): string;

    /**
     * 生成编辑索引sql
     * @param tableData 表数据，包含表名、列数据、索引数据
     * @param tableName   表名
     * @param changeData  改变数据
     */
    getModifyIndexSql(tableData: any, tableName: string, changeData: { del: any[]; add: any[]; upd: any[] }): string;

    /** 通过数据库字段类型，返回基本数据类型 */
    getDataType(columnType: string): DataType;

    /** 包装字符串数据， 如：oracle需要把date类型改为 to_date(str, 'yyyy-mm-dd hh24:mi:ss') */
    wrapStrValue(columnType: string, value: string): string;
}

let mysqlDialect = new MysqlDialect();

let dbType2DialectMap: Map<string, DbDialect> = new Map();

export const registerDbDialect = (dbType: string, dd: DbDialect) => {
    dbType2DialectMap.set(dbType, dd);
};

export const getDbDialectMap = () => {
    return dbType2DialectMap;
};

export const getDbDialect = (dbType?: string): DbDialect => {
    return dbType2DialectMap.get(dbType!) || mysqlDialect;
};

(function () {
    console.log('init register db dialect');
    registerDbDialect(DbType.mysql, mysqlDialect);
    registerDbDialect(DbType.mariadb, new MariadbDialect());
    registerDbDialect(DbType.postgresql, new PostgresqlDialect());
    registerDbDialect(DbType.gauss, new GaussDialect());
    registerDbDialect(DbType.dm, new DMDialect());
    registerDbDialect(DbType.oracle, new OracleDialect());
    registerDbDialect(DbType.sqlite, new SqliteDialect());
    registerDbDialect(DbType.mssql, new MssqlDialect());
    registerDbDialect(DbType.kingbaseEs, new KingbaseEsDialect());
    registerDbDialect(DbType.vastbase, new VastbaseDialect());
})();
