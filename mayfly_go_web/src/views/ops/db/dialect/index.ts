import { MysqlDialect } from './mysql_dialect';
import { PostgresqlDialect } from './postgres_dialect';

export interface sqlColumnType {
    udtName: string;
    dataType: string;
    desc: string;
    space: string;
    range?: string;
}

export const DbType = {
    mysql: 'mysql',
    postgresql: 'postgres',
};

export interface DbDialect {
    /**
     * 获取图标信息
     */
    getIcon(): string;

    /**
     * 获取默认查询sql
     * @param table  表名
     * @param condition 条件
     * @param orderBy 排序
     * @param pageNum  页数
     * @param limit  条数
     */
    getDefaultSelectSql(table: string, condition: string, orderBy: string, pageNum: number, limit: number): string;

    /**
     * 包裹数据库表名、字段名等，避免使用关键字为字段名或表名时报错
     * @param name 名称
     */
    wrapName(name: string): string;

    /**
     *  生成字段类型列表
     * */
    getColumnTypes(): sqlColumnType[];

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
    getModifyColumnSql(tableName: string, changeData: { del: any[]; add: any[]; upd: any[] }): string;

    /**
     * 生成编辑索引sql
     * @param tableName   表名
     * @param changeData  改变数据
     */
    getModifyIndexSql(tableName: string, changeData: { del: any[]; add: any[]; upd: any[] }): string;
}

let mysqlDialect = new MysqlDialect();
let postgresDialect = new PostgresqlDialect();

export const getDbDialect = (dbType: string | undefined): DbDialect => {
    if (dbType === DbType.mysql) {
        return mysqlDialect;
    }
    if (dbType === DbType.postgresql) {
        return postgresDialect;
    }
    throw new Error('不支持的数据库');
};
