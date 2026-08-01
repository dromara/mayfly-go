import { DbDialect, DialectInfo, sqlColumnType, EditorCompletion, DataType, DbType, commonCustomKeywords, RowDefinition, IndexDefinition } from '@/views/ops/db/dialect/index';
import { QuoteEscape } from '@/views/ops/db/dialect/index';

export class ClickHouseDialect implements DbDialect {
    getInfo(): DialectInfo {
        return {
            name: 'ClickHouse',
            icon: 'icon db/clickhouse',
            defaultPort: 9000,
            formatSqlDialect: 'sql',
            columnTypes: this.getColumnTypes(),
            editorCompletions: this.getEditorCompletions(),
        };
    }

    getDefaultSelectSql(db: string, table: string, condition: string, orderBy: string, pageNum: number, limit: number): string {
        let sql = `SELECT * FROM ${this.quoteIdentifier(table)}`;
        if (condition) {
            sql += ` WHERE ${condition}`;
        }
        if (orderBy) {
            sql += ` ORDER BY ${orderBy}`;
        }
        if (limit > 0) {
            sql += ` LIMIT ${limit}`;
            if (pageNum > 1) {
                sql += ` OFFSET ${(pageNum - 1) * limit}`;
            }
        }
        return sql;
    }

    getPageSql(pageNum: number, limit: number): string {
        return `LIMIT ${limit} OFFSET ${(pageNum - 1) * limit}`;
    }

    getDefaultRows(): RowDefinition[] {
        return [
            {
                name: 'id',
                type: 'UInt64',
                value: '',
                length: '',
                numScale: '',
                notNull: true,
                pri: true,
                auto_increment: true,
                remark: '主键',
            },
        ];
    }

    getDefaultIndex(): IndexDefinition {
        return {
            indexName: '',
            columnNames: [],
            unique: false,
            indexType: 'INDEX',
            indexComment: '',
        };
    }

    quoteIdentifier(name: string): string {
        // ClickHouse uses backticks to quote identifiers
        return `\`${name}\``;
    }

    getCreateTableSql(tableData: Record<string, unknown>): string {
        const { tableName, columns, comment } = tableData as { tableName: string; columns: RowDefinition[]; comment: string };
        let sql = `CREATE TABLE ${this.quoteIdentifier(tableName)} (\n`;

        const columnDefs = columns.map((col: RowDefinition) => {
            let colDef = `  ${this.quoteIdentifier(col.name)} ${col.type}`;
            if (col.notNull) {
                colDef += ' NOT NULL';
            }
            if (col.auto_increment) {
                colDef += ' AUTO_INCREMENT';
            }
            if (col.remark) {
                colDef += ` COMMENT '${QuoteEscape(col.remark)}'`;
            }
            return colDef;
        });

        sql += columnDefs.join(',\n');
        sql += '\n) ENGINE = MergeTree() ORDER BY tuple()';

        if (comment) {
            sql += ` COMMENT '${QuoteEscape(comment)}'`;
        }

        return sql;
    }

    getCreateIndexSql(_tableData: Record<string, unknown>): string {
        // ClickHouse indexes are typically defined in the table creation statement
        // This is a simplified implementation
        return '-- ClickHouse indexes are typically defined in the CREATE TABLE statement';
    }

    getModifyColumnSql(_tableData: Record<string, unknown>, tableName: string, changeData: { del: RowDefinition[]; add: RowDefinition[]; upd: RowDefinition[] }): string {
        const { del, add, upd } = changeData;
        let sql = '';

        // Handle deleted columns
        if (del && del.length > 0) {
            const dropColumns = del.map((col: RowDefinition) => `DROP COLUMN ${this.quoteIdentifier(col.name)}`).join(',\n');
            sql += `ALTER TABLE ${this.quoteIdentifier(tableName)}\n${dropColumns};\n\n`;
        }

        // Handle added columns
        if (add && add.length > 0) {
            const addColumns = add
                .map((col: RowDefinition) => {
                    let colDef = `ADD COLUMN ${this.quoteIdentifier(col.name)} ${col.type}`;
                    if (col.notNull) {
                        colDef += ' NOT NULL';
                    }
                    if (col.remark) {
                        colDef += ` COMMENT '${QuoteEscape(col.remark)}'`;
                    }
                    return colDef;
                })
                .join(',\n');
            sql += `ALTER TABLE ${this.quoteIdentifier(tableName)}\n${addColumns};\n\n`;
        }

        // Handle updated columns
        if (upd && upd.length > 0) {
            const modifyColumns = upd
                .map((col: RowDefinition) => {
                    let colDef = `MODIFY COLUMN ${this.quoteIdentifier(col.name)} ${col.type}`;
                    if (col.notNull) {
                        colDef += ' NOT NULL';
                    }
                    if (col.remark) {
                        colDef += ` COMMENT '${QuoteEscape(col.remark)}'`;
                    }
                    return colDef;
                })
                .join(',\n');
            sql += `ALTER TABLE ${this.quoteIdentifier(tableName)}\n${modifyColumns};\n\n`;
        }

        return sql.trim();
    }

    getModifyIndexSql(_tableData: Record<string, unknown>, _tableName: string, _changeData: Record<string, unknown>): string {
        // ClickHouse index modification is typically done through table alterations
        return '-- ClickHouse index modifications are typically done through ALTER TABLE statements';
    }

    getModifyTableInfoSql(tableData: Record<string, unknown>): string {
        const { tableName, comment } = tableData as { tableName: string; comment: string };
        if (comment) {
            return `ALTER TABLE ${this.quoteIdentifier(tableName)} MODIFY COMMENT '${QuoteEscape(comment)}'`;
        }
        return '';
    }

    getDataType(columnType: string): DataType {
        const type = columnType.toLowerCase();

        if (type.includes('int') || type.includes('uint')) {
            return DataType.Number;
        } else if (type.includes('float') || type.includes('decimal')) {
            return DataType.Number;
        } else if (type.includes('date') || type.includes('datetime')) {
            return DataType.DateTime;
        } else if (type.includes('bool')) {
            return DataType.Number; // ClickHouse uses 0/1 for booleans
        } else {
            return DataType.String;
        }
    }

    wrapValue(columnType: string, value: unknown): string | number {
        const type = columnType.toLowerCase();

        // For string types, wrap in quotes
        if (this.getDataType(columnType) === DataType.String) {
            if (value === null || value === undefined) {
                return 'NULL';
            }
            // Escape single quotes
            return `'${String(value).replace(/'/g, "''")}'`;
        }

        // For date/time types, might need special handling
        if (type.includes('date') || type.includes('datetime')) {
            if (value === null || value === undefined) {
                return 'NULL';
            }
            return `'${value}'`;
        }

        // For other types, return as is
        if (value === null || value === undefined) {
            return 'NULL';
        }
        return value as string | number;
    }

    getBatchInsertPreviewSql(tableName: string, columns: string[], _duplicateStrategy: unknown): string {
        const quotedColumns = columns.map((col) => this.quoteIdentifier(col)).join(', ');
        const placeholders = columns.map(() => '?').join(', ');
        return `INSERT INTO ${this.quoteIdentifier(tableName)} (${quotedColumns}) VALUES (${placeholders})`;
    }

    private getColumnTypes(): sqlColumnType[] {
        return [
            { udtName: 'UInt8', dataType: 'UInt8', desc: '8-bit unsigned integer', space: '数值' },
            { udtName: 'UInt16', dataType: 'UInt16', desc: '16-bit unsigned integer', space: '数值' },
            { udtName: 'UInt32', dataType: 'UInt32', desc: '32-bit unsigned integer', space: '数值' },
            { udtName: 'UInt64', dataType: 'UInt64', desc: '64-bit unsigned integer', space: '数值' },
            { udtName: 'Int8', dataType: 'Int8', desc: '8-bit signed integer', space: '数值' },
            { udtName: 'Int16', dataType: 'Int16', desc: '16-bit signed integer', space: '数值' },
            { udtName: 'Int32', dataType: 'Int32', desc: '32-bit signed integer', space: '数值' },
            { udtName: 'Int64', dataType: 'Int64', desc: '64-bit signed integer', space: '数值' },
            { udtName: 'Float32', dataType: 'Float32', desc: '32-bit floating point', space: '数值' },
            { udtName: 'Float64', dataType: 'Float64', desc: '64-bit floating point', space: '数值' },
            { udtName: 'String', dataType: 'String', desc: 'Variable-length string', space: '字符串' },
            { udtName: 'FixedString', dataType: 'FixedString(N)', desc: 'Fixed-length string', space: '字符串' },
            { udtName: 'DateTime', dataType: 'DateTime', desc: 'Date and time', space: '时间' },
            { udtName: 'DateTime64', dataType: 'DateTime64', desc: 'Date and time with precision', space: '时间' },
            { udtName: 'Date', dataType: 'Date', desc: 'Date', space: '时间' },
            { udtName: 'Date32', dataType: 'Date32', desc: 'Date (32-bit)', space: '时间' },
            { udtName: 'UUID', dataType: 'UUID', desc: 'Universally unique identifier', space: '其他' },
            { udtName: 'IPv4', dataType: 'IPv4', desc: 'IPv4 address', space: '其他' },
            { udtName: 'IPv6', dataType: 'IPv6', desc: 'IPv6 address', space: '其他' },
            { udtName: 'Bool', dataType: 'Bool', desc: 'Boolean', space: '其他' },
            { udtName: 'Decimal', dataType: 'Decimal(P, S)', desc: 'Exact decimal number', space: '数值' },
            { udtName: 'Enum8', dataType: 'Enum8', desc: '8-bit enumeration', space: '其他' },
            { udtName: 'Enum16', dataType: 'Enum16', desc: '16-bit enumeration', space: '其他' },
            { udtName: 'Array', dataType: 'Array(T)', desc: 'Array of type T', space: '复杂' },
            { udtName: 'Tuple', dataType: 'Tuple(T1, T2, ...)', desc: 'Tuple of types', space: '复杂' },
            { udtName: 'Map', dataType: 'Map(K, V)', desc: 'Map of key-value pairs', space: '复杂' },
            { udtName: 'Nullable', dataType: 'Nullable(T)', desc: 'Nullable type T', space: '其他' },
            { udtName: 'LowCardinality', dataType: 'LowCardinality(T)', desc: 'Low cardinality type T', space: '其他' },
        ];
    }

    private getEditorCompletions(): EditorCompletion {
        return {
            keywords: [
                { label: 'SELECT', description: '查询数据' },
                { label: 'INSERT', description: '插入数据' },
                { label: 'UPDATE', description: '更新数据' },
                { label: 'DELETE', description: '删除数据' },
                { label: 'CREATE', description: '创建对象' },
                { label: 'ALTER', description: '修改对象' },
                { label: 'DROP', description: '删除对象' },
                { label: 'FROM', description: '指定表' },
                { label: 'WHERE', description: '条件过滤' },
                { label: 'GROUP BY', description: '分组' },
                { label: 'ORDER BY', description: '排序' },
                { label: 'LIMIT', description: '限制结果数量' },
                { label: 'OFFSET', description: '偏移量' },
                { label: 'JOIN', description: '连接表' },
                { label: 'INNER JOIN', description: '内连接' },
                { label: 'LEFT JOIN', description: '左连接' },
                { label: 'RIGHT JOIN', description: '右连接' },
                { label: 'FULL JOIN', description: '全连接' },
                { label: 'ON', description: '连接条件' },
                { label: 'AS', description: '别名' },
                { label: 'DISTINCT', description: '去重' },
                { label: 'HAVING', description: '分组后过滤' },
                { label: 'UNION', description: '合并结果集' },
                { label: 'ALL', description: '所有' },
                { label: 'ANY', description: '任意' },
                { label: 'EXISTS', description: '存在' },
                { label: 'IN', description: '在...中' },
                { label: 'BETWEEN', description: '在...之间' },
                { label: 'LIKE', description: '模糊匹配' },
                { label: 'IS', description: '是' },
                { label: 'NULL', description: '空值' },
                { label: 'NOT', description: '非' },
                { label: 'AND', description: '与' },
                { label: 'OR', description: '或' },
                { label: 'CASE', description: '条件表达式' },
                { label: 'WHEN', description: '当' },
                { label: 'THEN', description: '那么' },
                { label: 'ELSE', description: '否则' },
                { label: 'END', description: '结束' },
                { label: 'ENGINE', description: '表引擎' },
                { label: 'ORDER BY', description: '排序键' },
                { label: 'PRIMARY KEY', description: '主键' },
                { label: 'PARTITION BY', description: '分区键' },
                { label: 'SAMPLE BY', description: '采样键' },
                { label: 'SETTINGS', description: '设置' },
                { label: 'FINAL', description: '最终一致性' },
                { label: 'PREWHERE', description: '预过滤' },
            ].concat(commonCustomKeywords.map((keyword) => ({ label: keyword, description: '' }))),
            operators: [
                { label: '=', description: '等于' },
                { label: '!=', description: '不等于' },
                { label: '<>', description: '不等于' },
                { label: '<', description: '小于' },
                { label: '>', description: '大于' },
                { label: '<=', description: '小于等于' },
                { label: '>=', description: '大于等于' },
                { label: '+', description: '加' },
                { label: '-', description: '减' },
                { label: '*', description: '乘' },
                { label: '/', description: '除' },
                { label: '%', description: '取模' },
                { label: '||', description: '字符串连接' },
                { label: 'LIKE', description: '模糊匹配' },
                { label: 'ILIKE', description: '忽略大小写模糊匹配' },
                { label: 'REGEXP', description: '正则表达式匹配' },
                { label: 'IN', description: '在...中' },
                { label: 'NOT IN', description: '不在...中' },
                { label: 'BETWEEN', description: '在...之间' },
                { label: 'NOT BETWEEN', description: '不在...之间' },
                { label: 'IS NULL', description: '为空' },
                { label: 'IS NOT NULL', description: '不为空' },
            ],
            functions: [
                { label: 'COUNT()', description: '计数' },
                { label: 'SUM()', description: '求和' },
                { label: 'AVG()', description: '平均值' },
                { label: 'MIN()', description: '最小值' },
                { label: 'MAX()', description: '最大值' },
                { label: 'ROUND()', description: '四舍五入' },
                { label: 'CEIL()', description: '向上取整' },
                { label: 'FLOOR()', description: '向下取整' },
                { label: 'ABS()', description: '绝对值' },
                { label: 'LENGTH()', description: '字符串长度' },
                { label: 'UPPER()', description: '转大写' },
                { label: 'LOWER()', description: '转小写' },
                { label: 'SUBSTRING()', description: '截取字符串' },
                { label: 'CONCAT()', description: '连接字符串' },
                { label: 'REPLACE()', description: '替换字符串' },
                { label: 'NOW()', description: '当前时间' },
                { label: 'TODAY()', description: '今天日期' },
                { label: 'TO_YEAR()', description: '提取年份' },
                { label: 'TO_MONTH()', description: '提取月份' },
                { label: 'TO_DAYOFMONTH()', description: '提取日期' },
                { label: 'TO_HOUR()', description: '提取小时' },
                { label: 'TO_MINUTE()', description: '提取分钟' },
                { label: 'TO_SECOND()', description: '提取秒' },
                { label: 'DATE_ADD()', description: '日期加法' },
                { label: 'DATE_SUB()', description: '日期减法' },
                { label: 'DATEDIFF()', description: '日期差' },
                { label: 'IF()', description: '条件函数' },
                { label: 'COALESCE()', description: '返回第一个非空值' },
                { label: 'NULLIF()', description: '如果相等则返回NULL' },
                { label: 'CAST()', description: '类型转换' },
                { label: 'TO_UNIX_TIMESTAMP()', description: '转换为Unix时间戳' },
                { label: 'FROM_UNIX_TIMESTAMP()', description: '从Unix时间戳转换' },
            ],
            variables: [
                { label: '@@version', description: '数据库版本' },
                { label: '@@hostname', description: '主机名' },
            ],
        };
    }
}
