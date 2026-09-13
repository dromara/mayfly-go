/**
 * Db 类 - 数据库实例内的库信息（表、列、提示词等）
 */
import type { DbTableInfo, ColumnMetadata } from '../types';

/**
 * 数据库实例信息（某个实例下的某个库）
 */
export class Db {
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
