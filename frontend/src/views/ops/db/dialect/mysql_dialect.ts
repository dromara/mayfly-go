import { commonCustomKeywords, DataType, DuplicateStrategy } from './types';
import type { DbDialect, DialectCapabilities, DialectInfo, EditorCompletion, EditorCompletionItem, IndexDefinition, RowDefinition, SqlSnippetTemplate } from './types';
import { createDefaultRows, defaultRowsConfigs } from './shared/defaultRows';
import { limitCommaPageSnippet } from './shared/snippets';
import { appendLimitSql, getDefaultDataType, QuoteEscape, wrapValueDefault } from './shared/utils';
import { backtickQuotePairs, defineCapabilities, mysqlSplitOptions } from './shared/capabilities';
import { DbType } from './dbType';
import { registerDbDialect, setFallbackDialect } from './registry';
import type { TableEditContext, TableInfoEditContext, ChangeDiff } from './types';

// 参考官方文档：https://dev.mysql.com/doc/refman/8.0/en/data-types.html
const MYSQL_TYPE_LIST = [
    'bigint',
    'binary',
    'blob',
    'char',
    'datetime',
    'date',
    'decimal',
    'double',
    'enum',
    'float',
    'int',
    'json',
    'longblob',
    'longtext',
    'mediumblob',
    'mediumtext',
    'set',
    'smallint',
    'text',
    'time',
    'timestamp',
    'tinyint',
    'varbinary',
    'varchar',
];

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
            columnTypes: MYSQL_TYPE_LIST.map((a) => ({ udtName: a, dataType: a, desc: '', space: '' })),
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

    getDefaultRows(): RowDefinition[] {
        return createDefaultRows(defaultRowsConfigs.mysql);
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

    genColumnBasicSql(cl: RowDefinition): string {
        let val = cl.value ? (cl.value === 'CURRENT_TIMESTAMP' ? cl.value : `'${cl.value}'`) : '';
        let defVal = val ? `DEFAULT ${val}` : '';
        let length = cl.length;
        if (length) {
            length = cl.numScale ? `(${cl.length},${cl.numScale})` : `(${cl.length})`;
        } else {
            length = '';
        }
        let onUpdate = 'update_time' === cl.name ? ' ON UPDATE CURRENT_TIMESTAMP' : '';
        const parts = [
            this.quoteIdentifier(cl.name),
            cl.type + length,
            cl.notNull ? 'NOT NULL' : 'NULL',
            cl.auto_increment ? 'AUTO_INCREMENT' : '',
            defVal,
            onUpdate,
            cl.remark ? `COMMENT '${QuoteEscape(cl.remark)}'` : '',
        ];
        return parts.filter(Boolean).join(' ');
    }
    getCreateTableSql(data: TableEditContext): string {
        // 创建表结构
        let pks = [] as string[];
        let fields: string[] = [];
        data.fields.res.forEach((item: RowDefinition) => {
            item.name && fields.push(this.genColumnBasicSql(item));
            if (item.pri) {
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
        data.indexs.res.forEach((a: IndexDefinition) => {
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

    getModifyColumnSql(tableData: TableEditContext, tableName: string, changeData: ChangeDiff<RowDefinition>): string {
        let arr = [] as string[];
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
        let addIndexs: IndexDefinition[] = [];

        if (changeData.upd.length > 0) {
            changeData.upd.forEach((a) => {
                dropIndexNames.push(a.indexName);
                addIndexs.push(a);
            });
        }

        if (changeData.del.length > 0) {
            changeData.del.forEach((a) => {
                dropIndexNames.push(a.indexName);
            });
        }

        if (changeData.add.length > 0) {
            changeData.add.forEach((a) => {
                addIndexs.push(a);
            });
        }

        if (dropIndexNames.length > 0 || addIndexs.length > 0) {
            const parts: string[] = [];
            if (dropIndexNames.length > 0) {
                dropIndexNames.forEach((a) => {
                    parts.push(`DROP INDEX ${this.quoteIdentifier(a)}`);
                });
            }

            if (addIndexs.length > 0) {
                addIndexs.forEach((a) => {
                    const unique = a.unique ? 'UNIQUE ' : '';
                    const cols = a.columnNames.map((c) => this.quoteIdentifier(c)).join(',');
                    parts.push(`ADD ${unique}INDEX ${this.quoteIdentifier(a.indexName)}(${cols}) USING ${a.indexType} COMMENT '${QuoteEscape(a.indexComment ?? '')}'`);
                });
            }
            return `ALTER TABLE ${this.quoteIdentifier(tableName)}\n  ${parts.join(',\n  ')};`;
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
        return wrapValueDefault(columnType, value) as string | number;
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
