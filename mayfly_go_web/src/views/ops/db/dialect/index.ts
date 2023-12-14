import { MysqlDialect } from './mysql_dialect';
import { PostgresqlDialect } from './postgres_dialect';
import { DMDialect } from '@/views/ops/db/dialect/dm_dialect';
import { SqlLanguage } from 'sql-formatter/lib/src/sqlFormatter';

export interface sqlColumnType {
    udtName: string;
    dataType: string;
    desc: string;
    space: string;
    range?: string;
}

export interface RowDefinition {
    name: string;
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

// 数据库基础信息
export interface DialectInfo {
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
    formatSqlDialect: SqlLanguage;

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
    postgresql: 'postgres',
    dm: 'dm', // 达梦
};

export interface DbDialect {
    /**
     * 获取一些数据库默认信息
     */
    getInfo(): DialectInfo;

    /**
     * 获取默认查询sql
     * @param table  表名
     * @param condition 条件
     * @param orderBy 排序
     * @param pageNum  页数
     * @param limit  条数
     */
    getDefaultSelectSql(table: string, condition: string, orderBy: string, pageNum: number, limit: number): string;

    getDefaultRows(): RowDefinition[];

    getDefaultIndex(): IndexDefinition;

    /**
     * 根据数据库完整字段类型，获取类型简称。如字符串为abc、数字为123、日期为date等。
     * @param columnType 数据库原始字段类型
     */
    getShortColumnType(columnType: string): string;

    /**
     * 包裹数据库表名、字段名等，避免使用关键字为字段名或表名时报错
     * @param name 名称
     */
    wrapName(name: string): string;

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
     * @param tableName 表名
     * @param changeData 改变信息
     */
    getModifyColumnSql(tableName: string, changeData: { del: RowDefinition[]; add: RowDefinition[]; upd: RowDefinition[] }): string;

    /**
     * 生成编辑索引sql
     * @param tableName   表名
     * @param changeData  改变数据
     */
    getModifyIndexSql(tableName: string, changeData: { del: any[]; add: any[]; upd: any[] }): string;
}

let mysqlDialect = new MysqlDialect();
let postgresDialect = new PostgresqlDialect();
let dmDialect = new DMDialect();

export const getDbDialect = (dbType: string | undefined): DbDialect => {
    if (!dbType) {
        return mysqlDialect;
    }
    if (dbType === DbType.mysql) {
        return mysqlDialect;
    }
    if (dbType === DbType.postgresql) {
        return postgresDialect;
    }
    if (dbType === DbType.dm) {
        return dmDialect;
    }
    throw new Error('不支持的数据库');
};
