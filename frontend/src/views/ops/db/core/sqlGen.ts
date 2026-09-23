/**
 * 纯 SQL 语句生成函数：INSERT / UPDATE / DELETE / SELECT / COUNT。
 *
 * 与 DbInst 解耦：只接收方言、列元数据等原始参数，不依赖任何实例状态，
 * 便于单元测试与跨模块复用。DbInst 的 gen* 方法仅负责加载元数据后委托至本模块。
 */
import type { DbDialect } from '../dialect';
import type { ColumnMetadata } from '../types';

/**
 * 生成 INSERT 语句
 * @param dialect 方言
 * @param schema schema 前缀（已 wrapName，含尾部 '.'），空串表示无 schema
 * @param table 表名（未 wrap）
 * @param columns 表的全部列元数据
 * @param datas 要插入的数据行
 * @param skipNull 是否跳过 null 值字段
 * @param wrapName 标识符包裹函数
 */
export function buildInsertSql(
    dialect: DbDialect,
    schema: string,
    table: string,
    columns: ColumnMetadata[],
    datas: Record<string, unknown>[],
    skipNull: boolean,
    wrapName: (name: string) => string,
): string {
    if (!datas?.length) {
        return '';
    }
    const sqls: string[] = [];
    for (const data of datas) {
        const colNames: string[] = [];
        const values: (string | number | boolean | null)[] = [];
        for (const column of columns) {
            const colName = column.columnName;
            if (skipNull && data[colName] == null) {
                continue;
            }
            colNames.push(wrapName(colName));
            values.push(dialect.wrapValue(column.dataType, data[colName]));
        }
        sqls.push(`INSERT INTO ${schema}${wrapName(table)} (${colNames.join(', ')})
                       VALUES (${values.join(', ')})`);
    }
    return sqls.join(';\n') + ';';
}

/**
 * 生成 UPDATE 语句（按全部主键列定位行）
 * @returns UPDATE SQL；无主键时返回空串（调用方应提示用户）
 */
export function buildUpdateSql(
    dialect: DbDialect,
    schema: string,
    table: string,
    columnValue: Record<string, unknown>,
    updateDataTypes: Record<string, string>,
    keyColumns: ColumnMetadata[],
    rowData: Record<string, unknown>,
    wrapName: (name: string) => string,
): string {
    if (keyColumns.length === 0) {
        return '';
    }
    let sql = `UPDATE ${schema}${wrapName(table)} SET `;
    for (const k of Object.keys(columnValue)) {
        const dataType = updateDataTypes[k] ?? 'varchar';
        sql += ` ${wrapName(k)} = ${dialect.wrapValue(dataType, columnValue[k])},`;
    }
    sql = sql.substring(0, sql.length - 1);

    const where = keyColumns
        .map((c) => `${wrapName(c.columnName)} = ${dialect.wrapValue(c.dataType, rowData[c.columnName])}`)
        .join(' AND ');
    return sql + ` WHERE ${where} ;`;
}

/**
 * 生成 DELETE 语句（按主键定位行，支持联合主键）
 * @returns DELETE SQL；无主键时返回空串
 */
export function buildDeleteSql(
    dialect: DbDialect,
    table: string,
    keyColumns: ColumnMetadata[],
    datas: Record<string, unknown>[],
    wrapName: (name: string) => string,
): string {
    if (keyColumns.length === 0) {
        return '';
    }
    if (keyColumns.length === 1) {
        const col = keyColumns[0];
        const ids = datas.map((d: Record<string, unknown>) => `${dialect.wrapValue(col.dataType, d[col.columnName])}`).join(',');
        return `DELETE
                FROM ${wrapName(table)}
                WHERE ${wrapName(col.columnName)} IN (${ids})`;
    }
    // 联合主键：逐行按全部主键列 AND 组合，行间以 OR 连接
    const conditions = datas
        .map(
            (d: Record<string, unknown>) =>
                '(' +
                keyColumns.map((c) => `${wrapName(c.columnName)} = ${dialect.wrapValue(c.dataType, d[c.columnName])}`).join(' AND ') +
                ')',
        )
        .join(' OR ');
    return `DELETE
                FROM ${wrapName(table)}
                WHERE ${conditions}`;
}

/**
 * 生成 COUNT SQL
 */
export function buildCountSql(table: string, wrapName: (name: string) => string, condition?: string): string {
    return `SELECT COUNT(*) count
                FROM ${wrapName(table)} ${condition ? 'WHERE ' + condition : ''}`;
}

/**
 * 解析 schema 前缀：dbName 格式为 "instance/database" 时取 database 部分作为前缀，
 * 单段名称（无 '/'）不添加前缀（与回滚 SQL 生成逻辑一致）。
 */
export function resolveSchemaPrefix(dbName: string, wrapName: (name: string) => string): string {
    const arr = dbName.split('/');
    if (arr.length === 2) {
        return wrapName(arr[1]) + '.';
    }
    return '';
}
