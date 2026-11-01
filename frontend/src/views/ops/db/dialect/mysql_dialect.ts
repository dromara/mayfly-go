import { commonCustomKeywords, DataType, DuplicateStrategy } from './types';
import type {
    DbDialect,
    DialectCapabilities,
    DialectInfo,
    EditorCompletion,
    EditorCompletionItem,
    IndexDefinition,
    ColumnDefinition,
    SqlSnippetTemplate,
} from './types';
import { createDefaultColumns, defaultColumnConfigs } from './shared/defaultColumns';
import { limitCommaPageSnippet } from './shared/snippets';
import { appendLimitSql, getDefaultDataType, QuoteEscape, wrapValueDefault } from './shared/utils';
import { backtickQuotePairs, defineCapabilities, mysqlSplitOptions } from './shared/capabilities';
import { diffPrimaryKey } from './shared/primaryKey';
import { DbType } from './dbType';
import { registerDbDialect, setFallbackDialect } from './registry';
import type { TableEditContext, TableInfoEditContext, ChangeDiff } from './types';

/**
 * 不能有字面量默认值的类型（MySQL：BLOB/TEXT/GEOMETRY/JSON 只允许表达式默认值）。
 * 换类型时旧的字符串默认值会残留，原样写出必报 1101。
 */
const MYSQL_NO_LITERAL_DEFAULT_TYPES = [
    'tinytext',
    'text',
    'mediumtext',
    'longtext',
    'tinyblob',
    'blob',
    'mediumblob',
    'longblob',
    'json',
    'geometry',
    'point',
    'linestring',
    'polygon',
    'multipoint',
    'multilinestring',
    'multipolygon',
    'geometrycollection',
];

// 参考官方文档：https://dev.mysql.com/doc/refman/8.0/en/data-types.html
//
// 只列「真正的数据类型」。unsigned / zerofill 是整数类型的修饰符而非类型，
// 单独用 MYSQL_UNSIGNED_TYPE_BASES 生成可选项（见 getInfo），避免把修饰符混进类型名。
const MYSQL_TYPE_LIST = [
    'bigint',
    'binary',
    'bit',
    'blob',
    'bool',
    'boolean',
    'char',
    'date',
    'datetime',
    'decimal',
    'double',
    'enum',
    'float',
    'geometry',
    'geometrycollection',
    'int',
    'integer',
    'json',
    'linestring',
    'longblob',
    'longtext',
    'mediumblob',
    'mediumint',
    'mediumtext',
    'multilinestring',
    'multipoint',
    'multipolygon',
    'numeric',
    'point',
    'polygon',
    'real',
    'set',
    'smallint',
    'text',
    'time',
    'timestamp',
    'tinyblob',
    'tinyint',
    'tinytext',
    'varbinary',
    'varchar',
    'year',
];

/**
 * 无符号整数：内部值沿用 `unsigned bigint` 前缀形态（与后端注册类型名、跨库类型匹配口径一致），
 * 下拉显示用 MySQL 的 DDL 语法形态 `bigint unsigned`。
 *
 * 必须作为可选项存在：列元数据回显的就是 `unsigned bigint`，清单里没有这一项时
 * el-select 只能把它当自由文本显示（宽度不够还会被截成「unsigned ...」），
 * 用户也选不回来——一旦改选成 `bigint`，保存后无符号属性就被静默丢弃。
 */
const MYSQL_UNSIGNED_TYPE_BASES = ['tinyint', 'smallint', 'mediumint', 'int', 'bigint'];

// 参考官方文档：https://dev.mysql.com/doc/refman/8.3/en/functions.html
const replaceFunctions: EditorCompletionItem[] = [
    /**  字符串相关函数  */
    { label: 'CONCAT', insertText: 'CONCAT(str1,str2,...)', description: '多字符串合并' },
    { label: 'ASCII', insertText: 'ASCII(char)', description: '返回字符的ASCII值' },
    { label: 'BIT_LENGTH', insertText: 'BIT_LENGTH(str1)', description: '多字符串合并' },
    { label: 'INSTR', insertText: 'INSTR(str,substr)', description: '返回字符串substr所在str位置' },
    { label: 'LEFT', insertText: 'LEFT(str,len)', description: '返回字符串str的左端len个字符' },
    { label: 'RIGHT', insertText: 'RIGHT(str,len)', description: '返回字符串str的右端len个字符' },
    { label: 'MID', insertText: 'MID(str,pos,len)', description: '返回字符串str的位置pos起len个字符' },
    { label: 'SUBSTRING', insertText: 'SUBSTRING(exp, start, length)', description: '截取字符串' },
    { label: 'REPLACE', insertText: 'REPLACE(str,from_str,to_str)', description: '替换字符串' },
    { label: 'REPEAT', insertText: 'REPEAT(str,count)', description: '重复字符串count遍' },
    { label: 'UPPER', insertText: 'UPPER(str)', description: '返回大写的字符串' },
    { label: 'LOWER', insertText: 'LOWER(str)', description: '返回小写的字符串' },
    { label: 'TRIM', insertText: 'TRIM(str)', description: '去除字符串首尾空格' },
    /**  数学相关函数  */
    { label: 'ABS', insertText: 'ABS(n)', description: '返回n的绝对值' },
    { label: 'FLOOR', insertText: 'FLOOR(n)', description: '返回不大于n的最大整数' },
    { label: 'CEILING', insertText: 'CEILING(n)', description: '返回不小于n的最小整数值' },
    { label: 'ROUND', insertText: 'ROUND(n,d)', description: '返回n的四舍五入值,保留d(默认0)位小数' },
    { label: 'RAND', insertText: 'RAND()', description: '返回在范围0到1.0内的随机浮点值' },

    /** 日期函数 */
    { label: 'DATE', insertText: "DATE('date')", description: '返回指定表达式的日期部分' },
    { label: 'WEEK', insertText: "WEEK('date')", description: '返回指定日期是一年中的第几周' },
    { label: 'MONTH', insertText: "MONTH('date')", description: '返回指定日期的月份' },
    { label: 'QUARTER', insertText: "QUARTER('date')", description: '返回指定日期是一年的第几个季度' },
    { label: 'YEAR', insertText: "YEAR('date')", description: '返回指定日期的年份' },
    { label: 'DATE_ADD', insertText: "DATE_ADD('date', interval 1 day)", description: '日期函数加减运算' },
    { label: 'DATE_SUB', insertText: "DATE_SUB('date', interval 1 day)", description: '日期函数加减运算' },
    { label: 'DATE_FORMAT', insertText: "DATE_FORMAT('date', '%Y-%m-%d %h:%i:%s')", description: '' },
    { label: 'CURDATE', insertText: 'CURDATE()', description: '返回当前日期' },
    { label: 'CURTIME', insertText: 'CURTIME()', description: '返回当前时间' },
    { label: 'NOW', insertText: 'NOW()', description: '返回当前日期时间' },
    { label: 'DATEDIFF', insertText: 'DATEDIFF(expr1,expr2)', description: '返回结束日expr1和起始日expr2之间的天数' },
    { label: 'UNIX_TIMESTAMP', insertText: 'UNIX_TIMESTAMP()', description: '返回指定时间(默认当前)unix时间戳' },
    { label: 'FROM_UNIXTIME', insertText: 'FROM_UNIXTIME(timestamp)', description: '把时间戳格式为年月日时分秒' },

    /**  逻辑函数 */
    { label: 'IFNULL', insertText: 'IFNULL(expression, alt_value)', description: '表达式为空取第二个参数值,否则取表达式值' },
    { label: 'IF', insertText: 'IF(expr1, expr2, expr3)', description: 'expr1为true则取expr2，否则取expr3' },
    { label: 'CASE', insertText: '\n(\n    CASE\n      WHEN expr1 THEN expr2\n      ELSE expr3\n     END\n   ) col', description: 'CASE WHEN THEN ELSE END' },
];

let mysqlDialectInfo: DialectInfo;
let mysqlCompletions: EditorCompletion;

export class MysqlDialect implements DbDialect {
    getCapabilities(): DialectCapabilities {
        return defineCapabilities({
            // MySQL 无 schema 层级（database 即最外层命名空间）
            supportsSchema: false,
            supportsIndexComment: true,
            // MySQL 协议兼容：表列表展示创建时间等专属列
            mysqlCompatible: true,
            // go-sql-driver 支持按名注册内存 tls.Config，三档（含只校链的 verify-ca）都可兑现
            tlsModes: ['require', 'verify-ca', 'verify-full'],
            quotePairs: backtickQuotePairs,
            sqlSplitOptions: mysqlSplitOptions,
        });
    }

    getInfo(): DialectInfo {
        if (mysqlDialectInfo) {
            return mysqlDialectInfo;
        }

        mysqlDialectInfo = {
            name: 'MySQL',
            icon: 'icon db/mysql',
            defaultPort: 3306,
            formatSqlDialect: 'mysql',
            columnTypes: [
                ...MYSQL_TYPE_LIST.map((a) => ({ udtName: a, dataType: a, desc: '', space: '' })),
                // 无符号整数：value 为归一形态，label 用 DDL 语法形态，回显与手动选择都能命中
                ...MYSQL_UNSIGNED_TYPE_BASES.map((base) => ({
                    udtName: `unsigned ${base}`,
                    dataType: `${base} unsigned`,
                    desc: '无符号整数（只存非负数）',
                    space: '',
                })),
            ],
        };
        return mysqlDialectInfo;
    }

    /** 编辑器联想词由 monaco 语言定义派生，按需加载并缓存（详见 types.ts 的 DialectInfo 注释） */
    async getEditorCompletions(): Promise<EditorCompletion> {
        if (mysqlCompletions) {
            return mysqlCompletions;
        }

        const { language: mysqlLanguage } = await import('monaco-editor/languages/definitions/mysql/mysql.js');
        let { keywords, operators, builtinVariables, builtinFunctions } = mysqlLanguage;
        let replaceFunctionNames = replaceFunctions.map((a) => a.label);
        let functions = builtinFunctions
            .filter((a: string) => replaceFunctionNames.indexOf(a) < 0) // 删除重写的函数
            .map((a: string): EditorCompletionItem => ({ label: a, insertText: `${a}()`, description: 'func' }))
            .concat(replaceFunctions);

        let excludeKeywords = new Set(builtinFunctions.concat(replaceFunctionNames).concat(operators));
        mysqlCompletions = {
            keywords: keywords
                .filter((a: string) => !excludeKeywords.has(a)) // 移除已存在的operator、function
                .map((a: string): EditorCompletionItem => ({ label: a, description: 'keyword' }))
                .concat(commonCustomKeywords.map((a): EditorCompletionItem => ({ label: a, description: 'keyword' }))),
            operators: operators.map((a: string): EditorCompletionItem => ({ label: a, description: 'operator' })),
            functions,
            variables: builtinVariables.map((a: string): EditorCompletionItem => ({ label: a, description: 'var' })),
        };
        return mysqlCompletions;
    }

    getDefaultSelectSql(db: string, table: string, condition: string, orderBy: string, pageNum: number, limit: number) {
        return `SELECT * FROM ${this.quoteIdentifier(table)} ${condition ? 'WHERE ' + condition : ''} ${orderBy ? orderBy : ''} ${this.getPageSql(
            pageNum,
            limit
        )};`;
    }

    getPageSql(pageNum: number, limit: number) {
        return ` LIMIT ${(pageNum - 1) * limit}, ${limit}`;
    }

    getPreviewSql(sql: string, limit = 1): string {
        return appendLimitSql(sql, limit);
    }

    getPageSnippet(): SqlSnippetTemplate {
        return limitCommaPageSnippet;
    }

    getDefaultColumns(): ColumnDefinition[] {
        return createDefaultColumns(defaultColumnConfigs.mysql);
    }

    getDefaultIndex(): IndexDefinition {
        return {
            indexName: '',
            columnNames: [],
            unique: false,
            // 索引类型取自能力声明，避免与 defaultIndexType 各写一份而漂移
            indexType: this.getCapabilities().defaultIndexType,
            indexComment: '',
        };
    }

    quoteIdentifier = (name: string) => {
        return `\`${name}\``;
    };

    /**
     * 列类型表达式（类型名 + 长度参数 + 无符号修饰）。
     *
     * 元数据与迁移链路把无符号类型归一为 `unsigned bigint`（与后端注册类型名、跨库类型匹配口径一致），
     * 而 MySQL 的 DDL 语法是 `bigint unsigned`，且长度只能跟在基础类型后面（`bigint(20) unsigned`）：
     * 前缀形态原样写进 ALTER/CREATE 会直接 1064，表现为「改长度/改可空保存后没生效」。
     * 换序规则与后端 mysql sqlgen 保持一致。
     */
    private ddlColumnTypeSql(cl: ColumnDefinition): string {
        const type = (cl.type || '').trim();
        const length = this.typeLengthSql(cl);
        const unsignedPrefix = 'unsigned ';
        if (type.toLowerCase().startsWith(unsignedPrefix)) {
            return `${type.slice(unsignedPrefix.length)}${length} unsigned`;
        }
        return `${type}${length}`;
    }

    /**
     * 只对「类型声明里真的接受该参数」的类型写出 `(长度[,精度])`。
     *
     * 编辑表单换类型时长度输入框会留着旧值（varchar(64) → json/date/enum），原样写出直接语法错误：
     * 实测 MySQL 8.0.46 下 `json(64)`/`date(64)`/`enum(64)` 均 1064、`datetime(320)` 报 1426、
     * `int(300)` 报「Display width out of range」。故这里按类型白名单 + 取值区间双重把关。
     */
    private typeLengthSql(cl: ColumnDefinition): string {
        const type = (cl.type || '').toLowerCase().replace(/^unsigned\s+/, '').trim();
        if (!cl.length) {
            return '';
        }
        const raw = Number(cl.length);
        const scale = Number(cl.numScale) || 0;

        if (type === 'decimal' || type === 'numeric') {
            return scale > 0 ? `(${cl.length},${cl.numScale})` : `(${cl.length})`;
        }
        if (type === 'float' || type === 'double' || type === 'real') {
            // FLOAT(p)/FLOAT(M,D) 合法；DOUBLE 只写 (p) 是语法错误，必须有小数位
            if (scale > 0) {
                return `(${cl.length},${cl.numScale})`;
            }
            return type === 'float' ? `(${cl.length})` : '';
        }
        if (/^(tinyint|smallint|mediumint|int|bigint)$/.test(type)) {
            // 显示宽度上限 255
            return Number.isInteger(raw) && raw >= 1 && raw <= 255 ? `(${raw})` : '';
        }
        if (type === 'datetime' || type === 'time' || type === 'timestamp') {
            // 小数秒精度 0~6
            return Number.isInteger(raw) && raw >= 0 && raw <= 6 ? `(${raw})` : '';
        }
        if (type === 'year') {
            return raw === 4 ? '(4)' : '';
        }
        if (['char', 'varchar', 'binary', 'varbinary', 'bit'].includes(type)) {
            return `(${cl.length})`;
        }
        // text/blob 家族的 M 已被服务端忽略；json/date/enum/set/几何类型写参数即报错 → 一律不写
        return '';
    }

    /**
     * 默认值子句。
     *
     * 表达式默认值必须原样写进括号（`DEFAULT (uuid())`）：包上引号会退化成字符串字面量，
     * 「每次插入生成新 uuid」就静默变成「插入固定文本 uuid()」。CURRENT_TIMESTAMP 保持
     * 无括号形态，以兼容 5.7（5.7 不支持 8.0.13+ 的表达式默认值语法）。
     */
    private defaultClause(cl: ColumnDefinition): string {
        if (!cl.value) {
            return '';
        }
        // text/blob/json 等类型在 MySQL 里不能有**字面量**默认值（换类型时旧默认值会残留，直接写即 1101）。
        // 8.0 的表达式默认值 `DEFAULT ('x')` 虽然可执行，但元数据会把它回读成 `_latin1\'x\'` 这类转义文本，
        // 再写回就二次转义，故这类类型一律不写 DEFAULT——比生成一条必然失败的 DDL 更可控。
        if (MYSQL_NO_LITERAL_DEFAULT_TYPES.includes((cl.type || '').toLowerCase().replace(/^unsigned\s+/, '').trim())) {
            return '';
        }
        if (cl.valueIsExpression && cl.value.toUpperCase() !== 'CURRENT_TIMESTAMP') {
            return `DEFAULT (${cl.value})`;
        }
        if (cl.value === 'CURRENT_TIMESTAMP') {
            return `DEFAULT ${cl.value}`;
        }
        return `DEFAULT '${QuoteEscape(cl.value)}'`;
    }

    genColumnBasicSql(cl: ColumnDefinition): string {
        const parts = [
            this.quoteIdentifier(cl.name),
            this.ddlColumnTypeSql(cl),
            cl.nullable ? 'NULL' : 'NOT NULL',
            cl.autoIncrement ? 'AUTO_INCREMENT' : '',
            this.defaultClause(cl),
            // 「自动更新」子句由元数据带回：MODIFY COLUMN 会重置未显式写出的属性，
            // 不带回来就会让「只改注释」的编辑把时间列的自动更新静默抹掉
            cl.onUpdate ?? '',
            cl.comment ? `COMMENT '${QuoteEscape(cl.comment)}'` : '',
        ];
        return parts.filter(Boolean).join(' ');
    }
    getCreateTableSql(data: TableEditContext): string {
        // 创建表结构
        let pks = [] as string[];
        let fields: string[] = [];
        data.fields.res.forEach((item: ColumnDefinition) => {
            item.name && fields.push(this.genColumnBasicSql(item));
            if (item.isPrimaryKey) {
                pks.push(this.quoteIdentifier(item.name));
            }
        });

        const pkClause = pks.length > 0 ? `,\n  PRIMARY KEY (${pks.join(',')})` : '';
        const commentClause = data.tableComment ? ` COMMENT='${QuoteEscape(data.tableComment)}'` : '';
        return `CREATE TABLE ${this.quoteIdentifier(data.tableName)} (\n  ${fields.join(',\n  ')}${pkClause}\n)${commentClause};`;
    }

    getCreateIndexSql(data: TableEditContext): string {
        // 创建索引
        const sqls: string[] = [];
        data.indexes.res.forEach((a: IndexDefinition) => {
            const unique = a.unique ? 'UNIQUE ' : '';
            const cols = a.columnNames.map((c) => this.quoteIdentifier(c)).join(',');
            sqls.push(`ADD ${unique}INDEX ${this.quoteIdentifier(a.indexName)}(${cols}) USING ${a.indexType} COMMENT '${QuoteEscape(a.indexComment ?? '')}'`);
        });
        if (sqls.length === 0) return '';
        return `ALTER TABLE ${this.quoteIdentifier(data.tableName)}\n  ${sqls.join(',\n  ')};`;
    }

    getDropTableSql(db: string, table: string): string {
        // MySQL 的 database 即最外层命名空间，执行时已由 db 参数选定，故不做库名限定
        return `DROP TABLE ${this.quoteIdentifier(table)}`;
    }

    getModifyColumnSql(tableData: TableEditContext, tableName: string, changeData: ChangeDiff<ColumnDefinition>): string {
        let arr = [] as string[];

        // 主键是表级约束，MODIFY 改不动它：勾选/取消「主键」必须另发 DROP/ADD PRIMARY KEY，
        // 且二者可与其他 alter_specification 合并进同一条 ALTER TABLE
        const primaryKey = diffPrimaryKey(tableData.fields.oldFields, tableData.fields.res);
        if (primaryKey.changed) {
            if (tableData.fields.oldFields.some((field) => field.isPrimaryKey)) {
                arr.push('DROP PRIMARY KEY');
            }
            if (primaryKey.columns.length > 0) {
                arr.push(`ADD PRIMARY KEY (${primaryKey.columns.map((name) => this.quoteIdentifier(name)).join(',')})`);
            }
        }

        if (changeData.del.length > 0) {
            changeData.del.forEach((a) => {
                arr.push(`DROP COLUMN ${this.quoteIdentifier(a.name)}`);
            });
        }
        if (changeData.add.length > 0) {
            changeData.add.forEach((a) => {
                arr.push(`ADD COLUMN ${this.genColumnBasicSql(a)}`);
            });
        }

        if (changeData.upd.length > 0) {
            changeData.upd.forEach((a) => {
                if (a.name === a.oldName) {
                    arr.push(`MODIFY COLUMN ${this.genColumnBasicSql(a)}`);
                } else {
                    arr.push(`CHANGE COLUMN ${this.quoteIdentifier(a.oldName!)} ${this.genColumnBasicSql(a)}`);
                }
            });
        }

        if (arr.length > 0) {
            const dbTable = `${this.quoteIdentifier(tableData.db)}.${this.quoteIdentifier(tableName)}`;
            return `ALTER TABLE ${dbTable}\n  ${arr.join(',\n  ')};`;
        }

        return '';
    }

    getModifyIndexSql(tableData: TableEditContext, tableName: string, changeData: ChangeDiff<IndexDefinition>): string {
        let dropIndexNames: string[] = [];
        let addIndexes: IndexDefinition[] = [];

        if (changeData.upd.length > 0) {
            changeData.upd.forEach((a) => {
                dropIndexNames.push(a.indexName);
                addIndexes.push(a);
            });
        }

        if (changeData.del.length > 0) {
            changeData.del.forEach((a) => {
                dropIndexNames.push(a.indexName);
            });
        }

        if (changeData.add.length > 0) {
            changeData.add.forEach((a) => {
                addIndexes.push(a);
            });
        }

        if (dropIndexNames.length > 0 || addIndexes.length > 0) {
            const parts: string[] = [];
            if (dropIndexNames.length > 0) {
                dropIndexNames.forEach((a) => {
                    parts.push(`DROP INDEX ${this.quoteIdentifier(a)}`);
                });
            }

            if (addIndexes.length > 0) {
                addIndexes.forEach((a) => {
                    const unique = a.unique ? 'UNIQUE ' : '';
                    const cols = a.columnNames.map((c) => this.quoteIdentifier(c)).join(',');
                    parts.push(
                        `ADD ${unique}INDEX ${this.quoteIdentifier(a.indexName)}(${cols}) USING ${a.indexType} COMMENT '${QuoteEscape(a.indexComment ?? '')}'`
                    );
                });
            }
            // 与列语句一致带上库限定：索引 DDL 只写裸表名时，一旦连接默认库不同就会作用到别的库
            const dbTable = `${this.quoteIdentifier(tableData.db)}.${this.quoteIdentifier(tableName)}`;
            return `ALTER TABLE ${dbTable}\n  ${parts.join(',\n  ')};`;
        }
        return '';
    }

    getModifyTableInfoSql(tableData: TableInfoEditContext): string {
        let sql = '';
        const dbTable = `${this.quoteIdentifier(tableData.db)}.${this.quoteIdentifier(tableData.oldTableName)}`;
        if (tableData.tableComment !== tableData.oldTableComment) {
            sql += `ALTER TABLE ${dbTable} COMMENT '${QuoteEscape(tableData.tableComment)}';`;
        }

        if (tableData.tableName !== tableData.oldTableName) {
            sql += `ALTER TABLE ${dbTable} RENAME TO ${this.quoteIdentifier(tableData.tableName)};`;
        }
        return sql;
    }

    getDataType(columnType: string): DataType {
        return getDefaultDataType(columnType) as DataType;
    }

    wrapValue(columnType: string, value: unknown): string | number {
        // MySQL 默认把反斜杠当转义符，字符串值里的 \ 必须加倍，否则 a\b 会被解释成退格
        return wrapValueDefault(columnType, value, { escapeBackslash: true }) as string | number;
    }

    getBatchInsertPreviewSql(tableName: string, fieldArr: string[], duplicateStrategy: number): string {
        let placeholder = '?'.repeat(fieldArr.length).split('').join(',');
        let prefix = 'insert into';
        if (duplicateStrategy === DuplicateStrategy.IGNORE) {
            prefix = 'insert ignore into';
        } else if (duplicateStrategy === DuplicateStrategy.REPLACE) {
            prefix = 'replace into';
        }
        return `${prefix} ${this.quoteIdentifier(tableName)}(${fieldArr.join(',')}) values (${placeholder});`;
    }
}

// 自注册：MySQL 同时作为注册表的回退方言（未识别类型按最通用的 MySQL 语法处理）
const mysqlDialect = new MysqlDialect();
setFallbackDialect(mysqlDialect);
registerDbDialect(DbType.mysql, mysqlDialect);
