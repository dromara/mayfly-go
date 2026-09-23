/**
 * Schema 型方言的 DDL 共享基类
 *
 * 提取 schema 型方言（postgres/oracle/dm 及其兼容方言）的公共 DDL 基础设施：
 * - 标识符引用（双引号）
 * - 类型名模式匹配
 * - 列级 DDL 构建辅助（类型长度、默认值格式化、列名新旧值解析）
 * - 默认索引预设
 *
 * 各方言继承后仅覆写差异点（如 genColumnBasicSql 的自增表达、getDefaultValueSql 的类型判定等），
 * 新增 schema 型方言无需从零实现 DbDialect 全部方法——只需描述自身差异（开闭原则）。
 *
 * 归属 dialect 层的原因：这些方法是方言的词法/DDL 事实，由方言自描述后，
 * 调用方一律问方言、不硬编码 DbType 枚举。
 */

import type {
    DbDialect,
    DialectCapabilities,
    DialectInfo,
    EditorCompletion,
    IndexDefinition,
    ColumnDefinition,
    SqlSnippetTemplate,
    TableEditContext,
    TableInfoEditContext,
    ChangeDiff,
    DataType,
    DuplicateStrategy,
} from '../types';
import { matchType as _matchType, buildSchemaTable } from './utils';

/**
 * Schema 型方言 DDL 共享基类。
 *
 * 提供 schema 型方言（标识符以双引号引用、支持 COMMENT ON 语法）的公共 DDL 辅助方法。
 * 子类继承后仅覆写差异方法（如 genColumnBasicSql 的自增表达、getDefaultValueSql 的类型判定等），
 * 新增 schema 型方言无需从零实现全部 DbDialect 方法（开闭原则）。
 *
 * 子类须自行实现 getInfo / getCapabilities / getEditorCompletions 等元信息方法，
 * 以及 getDefaultSelectSql / getPageSql / getPreviewSql / getPageSnippet 等查询域方法，
 * 还有 getCreateTableSql / getModifyColumnSql 等方言差异较大的 DDL 方法。
 */
export abstract class SchemaDialectBase implements DbDialect {
    // ==================== 公共辅助方法（子类及外部均可调用） ====================

    /** 双引号引用标识符（schema 型方言统一使用双引号） */
    quoteIdentifier = (name: string): string => {
        return `"${name}"`;
    };

    /** 类型名模式匹配（大小写不敏感），判断类型名是否命中给定模式列表 */
    matchType(text: string, arr: string[]): boolean {
        return _matchType(text, arr);
    }

    /** 解析列名：有旧名且与当前名不同时返回旧名（用于 ALTER 语句中引用原始列名） */
    protected resolveColumnName(cl: ColumnDefinition): string {
        return cl.oldName && cl.name !== cl.oldName ? cl.oldName : cl.name;
    }

    // ==================== 列 DDL 构建辅助（子类可覆写以适配方言差异） ====================

    /**
     * 生成列类型长度 SQL 片段（含精度），如 `(20)` 或 `(10, 2)`。
     *
     * matchType 为大小写敏感的子串匹配，而各方言列类型大小写不一（postgres 用小写 varchar、
     * oracle/dm 用大写 VARCHAR/NUMBER），故先把类型名归一为小写再匹配，使本实现可同时服务三家。
     * 覆盖 char/time/bit/num/dec 前缀的类型（dec 同时命中 decimal 与 oracle 的 number 家族别名）。
     * 子类可覆写以适配方言特有的类型长度语法。
     */
    getTypeLengthSql(cl: ColumnDefinition): string {
        const type = (cl.type || '').toLowerCase();
        if (cl.length && this.matchType(type, ['char', 'time', 'bit', 'num', 'dec'])) {
            if (cl.numScale && this.matchType(type, ['num', 'dec'])) {
                return `(${cl.length}, ${cl.numScale})`;
            }
            return `(${cl.length})`;
        }
        return '';
    }

    /**
     * 默认值需要加引号的类型名列表（大小写不敏感匹配）。
     * 子类覆写以适配方言特有的类型判定（如 Oracle 额外包含 LONG/CLOB/BLOB 等）。
     */
    protected getDefaultValueQuotedTypeNames(): string[] {
        return ['char', 'time', 'date', 'text'];
    }

    /**
     * 默认值不加引号的函数名列表（小写，用于与 cl.value.toLowerCase() 比较）。
     * 仅当列类型属于 getDefaultValueQuotedTypeNames 中的 time/date 类时才生效。
     * 子类覆写以适配方言特有的时间函数（如 Oracle 的 SYSDATE、DM 的 SYSDATE/CURDATE 等）。
     */
    protected getUnquotedDefaultFunctionNames(): string[] {
        return ['current_timestamp', 'current_date', 'now()'];
    }

    /**
     * 生成列默认值 SQL 片段（含 DEFAULT 关键字），如 `DEFAULT 'foo'` 或 `DEFAULT 0`。
     *
     * 默认实现：字符串/时间类型加引号，时间函数（如 CURRENT_TIMESTAMP）不加引号，
     * nextval 系列函数不加引号（序列型方言如 postgres/dm）。
     *
     * 子类可覆写以适配方言特有的默认值语法（如 Oracle 对 LONG/CLOB 类型的特殊处理）。
     */
    getDefaultValueSql(cl: ColumnDefinition): string {
        if (cl.value && cl.value.length > 0) {
            let needsQuotes = false;
            if (this.matchType(cl.type, this.getDefaultValueQuotedTypeNames())) {
                const lowerVal = cl.value.toLowerCase().replace(/\s+/g, '');
                if (
                    this.matchType(cl.type, ['time', 'date']) &&
                    this.getUnquotedDefaultFunctionNames().includes(lowerVal)
                ) {
                    needsQuotes = false;
                } else {
                    needsQuotes = true;
                }
            }
            // 序列函数（如 nextval）不需要引号
            if (this.matchType(cl.value, ['nextval'])) {
                needsQuotes = false;
            }
            return ` DEFAULT ${needsQuotes ? "'" : ''}${cl.value}${needsQuotes ? "'" : ''}`;
        }
        return '';
    }

    /**
     * 生成单列的基础 DDL 片段（列名 + 类型 + 长度 + 约束等）。
     *
     * 默认实现：`"name" type(length) NOT NULL DEFAULT value`。
     * 子类必须覆写以适配方言特有的列定义语法（如 MySQL 的 AUTO_INCREMENT/COMMENT、
     * Oracle 的 IDENTITY、DM 的 IDENTITY、SQLite 的 PRIMARY KEY AUTOINCREMENT 等）。
     */
    genColumnBasicSql(cl: ColumnDefinition): string {
        const length = this.getTypeLengthSql(cl);
        const defVal = this.getDefaultValueSql(cl);
        const name = this.resolveColumnName(cl);
        const parts = [
            this.quoteIdentifier(name),
            cl.type + length,
            cl.nullable ? '' : 'NOT NULL',
            defVal,
        ];
        return parts.filter(Boolean).join(' ');
    }

    // ==================== DbDialect 公共默认实现 ====================

    /** 默认索引预设（schema 型方言统一使用 BTREE） */
    getDefaultIndex(): IndexDefinition {
        return {
            indexName: '',
            columnNames: [],
            unique: false,
            indexType: this.getCapabilities().defaultIndexType,
            indexComment: '',
        };
    }

    /**
     * 生成 CREATE INDEX 语句（不含索引注释）。
     *
     * 默认实现为标准 `CREATE [UNIQUE] INDEX name ON table (cols)` 语法，
     * 适用于大多数 schema 型方言。需要索引注释的方言（如 postgres）应覆写本方法。
     */
    getCreateIndexSql(tableData: TableEditContext): string {
        const dbTable = buildSchemaTable(this.quoteIdentifier, tableData.db, tableData.tableName);
        const sql: string[] = [];
        tableData.indexes.res.forEach((a: IndexDefinition) => {
            const cols = a.columnNames.map((c: string) => this.quoteIdentifier(c)).join(',');
            sql.push(`CREATE ${a.unique ? 'UNIQUE ' : ''}INDEX ${this.quoteIdentifier(a.indexName)} ON ${dbTable} (${cols})`);
        });
        return sql.join(';');
    }

    // ==================== 以下方法为抽象要求，子类必须实现 ====================

    abstract getInfo(): DialectInfo;
    abstract getCapabilities(): DialectCapabilities;
    abstract getEditorCompletions(): Promise<EditorCompletion>;
    abstract getDefaultSelectSql(db: string, table: string, condition: string, orderBy: string, pageNum: number, limit: number): string;
    abstract getPageSql(pageNum: number, limit: number): string;
    abstract getPreviewSql(sql: string, limit?: number): string;
    abstract getPageSnippet(): SqlSnippetTemplate;
    abstract getDefaultColumns(): ColumnDefinition[];
    abstract getCreateTableSql(tableData: TableEditContext): string;
    abstract getDropTableSql(db: string, table: string): string;
    abstract getModifyColumnSql(tableData: TableEditContext, tableName: string, changeData: ChangeDiff<ColumnDefinition>): string;
    abstract getModifyIndexSql(tableData: TableEditContext, tableName: string, changeData: ChangeDiff<IndexDefinition>): string;
    abstract getModifyTableInfoSql(tableData: TableInfoEditContext): string;
    abstract getDataType(columnType: string): DataType;
    abstract wrapValue(columnType: string, value: unknown): string | number | boolean | null;
    abstract getBatchInsertPreviewSql(tableName: string, columns: string[], duplicateStrategy: DuplicateStrategy): string;
}
