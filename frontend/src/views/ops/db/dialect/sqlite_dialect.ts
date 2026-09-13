import { commonCustomKeywords, DataType, DuplicateStrategy } from './types';
import type {
    DbDialect,
    DialectCapabilities,
    DialectInfo,
    EditorCompletion,
    EditorCompletionItem,
    IndexDefinition,
    RowDefinition,
    sqlColumnType,
    SqlSnippetTemplate,
} from './types';
import { createDefaultRows, defaultRowsConfigs } from './shared/defaultRows';
import { appendLimitSql, getDefaultDataType, wrapValueDefault } from './shared/utils';
import { defineCapabilities, sqliteQuotePairs, sqliteSplitOptions } from './shared/capabilities';
import { limitCommaPageSnippet } from './shared/snippets';
import { DbType } from './dbType';
import { registerDbDialect } from './registry';
import type { TableEditContext, TableInfoEditContext, ChangeDiff } from './types';

// 参考官方文档：https://www.sqlite.org/datatype3.html
const SQLITE_TYPE_LIST: sqlColumnType[] = [
    // INTEGER
    { udtName: 'int', dataType: 'int', desc: '', space: '', range: '' },
    { udtName: 'integer', dataType: 'integer', desc: '', space: '', range: '' },
    { udtName: 'tinyint', dataType: 'tinyint', desc: '', space: '', range: '' },
    { udtName: 'smallint', dataType: 'smallint', desc: '', space: '', range: '' },
    { udtName: 'mediumint', dataType: 'mediumint', desc: '', space: '', range: '' },
    { udtName: 'bigint', dataType: 'bigint', desc: '', space: '', range: '' },
    { udtName: 'unsigned big int', dataType: 'unsigned big int', desc: '', space: '', range: '' },
    { udtName: 'int2', dataType: 'int2', desc: '', space: '', range: '' },
    { udtName: 'int8', dataType: 'int8', desc: '', space: '', range: '' },
    // TEXT
    { udtName: 'character', dataType: 'character', desc: '', space: '', range: '' },
    { udtName: 'varchar', dataType: 'varchar', desc: '', space: '', range: '' },
    { udtName: 'varying character', dataType: 'varying character', desc: '', space: '', range: '' },
    { udtName: 'nchar', dataType: 'nchar', desc: '', space: '', range: '' },
    { udtName: 'native character', dataType: 'native character', desc: '', space: '', range: '' },
    { udtName: 'nvarchar', dataType: 'nvarchar', desc: '', space: '', range: '' },
    { udtName: 'text', dataType: 'text', desc: '', space: '', range: '' },
    { udtName: 'clob', dataType: 'clob', desc: '', space: '', range: '' },
    // blob
    { udtName: 'blob', dataType: 'blob', desc: '', space: '', range: '' },
    { udtName: 'no datatype specified', dataType: 'no datatype specified', desc: '', space: '', range: '' },
    // REAL
    { udtName: 'real', dataType: 'real', desc: '', space: '', range: '' },
    { udtName: 'double', dataType: 'double', desc: '', space: '', range: '' },
    { udtName: 'double precision', dataType: 'double precision', desc: '', space: '', range: '' },
    { udtName: 'float', dataType: 'float', desc: '', space: '', range: '' },
    // NUMERIC
    { udtName: 'numeric', dataType: 'numeric', desc: '', space: '', range: '' },
    { udtName: 'decimal', dataType: 'decimal', desc: '', space: '', range: '' },
    { udtName: 'boolean', dataType: 'boolean', desc: '', space: '', range: '' },
    { udtName: 'date', dataType: 'date', desc: '', space: '', range: '' },
    { udtName: 'datetime', dataType: 'datetime', desc: '', space: '', range: '' },
];

const addCustomKeywords = ['PRAGMA', 'database_list', 'sqlite_master'];

// 参考官方文档：https://www.sqlite.org/lang_corefunc.html
const functions: EditorCompletionItem[] = [
    //  字符函数
    { label: 'abs', insertText: 'abs(X)', description: '返回给定数值的绝对值' },
    { label: 'changes', insertText: 'changes()', description: '返回最近增删改影响的行数' },
    { label: 'coalesce', insertText: 'coalesce(X,Y,...)', description: '返回第一个不为空的值' },
    { label: 'hex', insertText: 'hex(X)', description: '返回给定字符的hex值' },
    { label: 'ifnull', insertText: 'ifnull(X,Y)', description: '返回第一个不为空的值' },
    { label: 'iif', insertText: 'iif(X,Y,Z)', description: '如果x为真则返回y，否则返回z' },
    { label: 'instr', insertText: 'instr(X,Y)', description: '返回字符y在x的第n个位置' },
    { label: 'length', insertText: 'length(X)', description: '返回给定字符的长度' },
    { label: 'load_extension', insertText: 'load_extension(X[,Y])', description: '加载扩展块' },
    { label: 'lower', insertText: 'lower(X)', description: '返回小写字符' },
    { label: 'ltrim', insertText: 'ltrim(X[,Y])', description: '左trim' },
    { label: 'nullif', insertText: 'nullif(X,Y)', description: '比较两值相等则返回null，否则返回第一个值' },
    { label: 'printf', insertText: "printf('%s',...)", description: '字符串格式化拼接,如%s %d' },
    { label: 'quote', insertText: 'quote(X)', description: '把字符串用引号包起来' },
    { label: 'random', insertText: 'random()', description: '生成随机数' },
    { label: 'randomblob', insertText: 'randomblob(N)', description: '生成一个包含N个随机字节的BLOB' },
    { label: 'replace', insertText: 'replace(X,Y,Z)', description: '替换字符串' },
    { label: 'round', insertText: 'round(X[,Y])', description: '将数值四舍五入到指定的小数位数' },
    { label: 'rtrim', insertText: 'rtrim(X[,Y])', description: '右trim' },
    { label: 'sign', insertText: 'sign(X)', description: '返回数字符号 1正 -1负 0零 null' },
    { label: 'soundex', insertText: 'soundex(X)', description: '返回字符串X的soundex编码字符串' },
    { label: 'sqlite_compileoption_get', insertText: 'sqlite_compileoption_get(N)', description: '获取指定编译选项的值' },
    {
        label: 'sqlite_compileoption_used',
        insertText: 'sqlite_compileoption_used(X)',
        description: '检查SQLite编译时是否使用了指定的编译选项',
    },
    { label: 'sqlite_source_id', insertText: 'sqlite_source_id()', description: '获取sqlite源代码标识符' },
    { label: 'sqlite_version', insertText: 'sqlite_version()', description: '获取sqlite版本' },
    { label: 'substr', insertText: 'substr(X,Y[,Z])', description: '截取字符串' },
    { label: 'substring', insertText: 'substring(X,Y[,Z])', description: '截取字符串' },
    { label: 'trim', insertText: 'trim(X[,Y])', description: '去除给定字符串前后的字符，默认空格' },
    { label: 'typeof', insertText: 'typeof(X)', description: '返回X的基本类型：null,integer,real,text,blob' },
    { label: 'unicode', insertText: 'unicode(X)', description: '返回与字符串X的第一个字符相对应的数字unicode代码点' },
    { label: 'unlikely', insertText: 'unlikely(X)', description: '返回大写字符' },
    { label: 'upper', insertText: 'upper(X)', description: '返回由0x00的N个字节组成的BLOB' },
    { label: 'zeroblob', insertText: 'zeroblob(N)', description: '返回分组中的平均值' },
    { label: 'avg', insertText: 'avg(X)', description: '返回总条数' },
    { label: 'count', insertText: 'count(*)', description: '返回分组中用给定非空字符串连接的值' },
    { label: 'group_concat', insertText: 'group_concat(X[,Y])', description: '返回分组中最大值' },
    { label: 'max', insertText: 'max(X)', description: '返回分组中最小值' },
    { label: 'min', insertText: 'min(X)', description: '返回分组中非空值的总和。' },
    { label: 'sum', insertText: 'sum(X)', description: '返回分组中非空值的总和。' },
    { label: 'total', insertText: 'total(X)', description: '返回YYYY-MM-DD格式的字符串' },
    { label: 'date', insertText: 'date(time-value[, modifier, ...])', description: '返回HH:MM:SS格式的字符串' },
    {
        label: 'time',
        insertText: 'time(time-value[, modifier, ...])',
        description: '将日期和时间字符串转换为特定的日期和时间格式',
    },
    { label: 'datetime', insertText: 'datetime(time-value[, modifier, ...])', description: '计算日期和时间的儒略日数' },
    {
        label: 'julianday',
        insertText: 'julianday(time-value[, modifier, ...])',
        description: '将日期和时间格式化为指定的字符串',
    },
];

let sqliteDialectInfo: DialectInfo;
let sqliteCompletions: EditorCompletion;

class SqliteDialect implements DbDialect {
    getCapabilities(): DialectCapabilities {
        return defineCapabilities({
            supportsSchema: false,
            supportsTableComment: false,
            supportsColumnComment: false,
            defaultIndexType: 'BTREE',
            // 嵌入式库：以文件路径而非 host/port 连接
            connectionMode: 'file_path',
            quotePairs: sqliteQuotePairs,
            sqlSplitOptions: sqliteSplitOptions,
        });
    }

    getInfo(): DialectInfo {
        if (sqliteDialectInfo) {
            return sqliteDialectInfo;
        }

        sqliteDialectInfo = {
            name: 'Sqlite3',
            icon: 'icon db/sqlite',
            defaultPort: 0,
            formatSqlDialect: 'sql',
            columnTypes: SQLITE_TYPE_LIST.sort((a, b) => a.udtName.localeCompare(b.udtName)),
        };
        return sqliteDialectInfo;
    }

    /** 编辑器联想词由 monaco 语言定义派生，按需加载并缓存（详见 types.ts 的 DialectInfo 注释） */
    async getEditorCompletions(): Promise<EditorCompletion> {
        if (sqliteCompletions) {
            return sqliteCompletions;
        }

        const { language: sqlLanguage } = await import('monaco-editor/languages/definitions/sql/sql.js');
        let { keywords, operators, builtinVariables } = sqlLanguage;

        sqliteCompletions = {
            keywords: keywords
                .filter((a: string) => addCustomKeywords.indexOf(a) === -1)
                .map((a: string): EditorCompletionItem => ({ label: a, description: 'keyword' }))
                .concat(commonCustomKeywords.map((a): EditorCompletionItem => ({ label: a, description: 'keyword' })))
                .concat(addCustomKeywords.map((a): EditorCompletionItem => ({ label: a, description: 'keyword' }))),
            operators: operators.map((a: string): EditorCompletionItem => ({ label: a, description: 'operator' })),
            functions,
            variables: builtinVariables.map((a: string): EditorCompletionItem => ({ label: a, description: 'var' })),
        };
        return sqliteCompletions;
    }

    getDefaultSelectSql(db: string, table: string, condition: string, orderBy: string, pageNum: number, limit: number) {
        return `SELECT *
                FROM ${this.quoteIdentifier(table)} ${condition ? 'WHERE ' + condition : ''} ${orderBy ? orderBy : ''} ${this.getPageSql(pageNum, limit)};`;
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
        return createDefaultRows(defaultRowsConfigs.sqlite);
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
        return `\"${name}\"`;
    };

    genColumnBasicSql(cl: RowDefinition): string {
        let val = cl.value ? (cl.value === 'CURRENT_TIMESTAMP' ? cl.value : `'${cl.value}'`) : '';
        let defVal = val ? `DEFAULT ${val}` : '';
        let length = '';
        if (cl.length) {
            length = cl.numScale ? `(${cl.length},${cl.numScale})` : `(${cl.length})`;
        }
        let nullAble = cl.notNull ? 'NOT NULL' : 'NULL';
        if (cl.pri) {
            const parts = [this.quoteIdentifier(cl.name), cl.type + length, 'PRIMARY KEY', cl.auto_increment ? 'AUTOINCREMENT' : '', nullAble];
            return parts.filter(Boolean).join(' ');
        }
        const parts = [this.quoteIdentifier(cl.name), cl.type + length, nullAble, defVal];
        return parts.filter(Boolean).join(' ');
    }

    getCreateTableSql(data: TableEditContext): string {
        // 创建表结构
        let fields: string[] = [];
        data.fields.res.forEach((item: RowDefinition) => {
            item.name && fields.push(this.genColumnBasicSql(item));
        });

        return `CREATE TABLE ${this.quoteIdentifier(data.tableName)} (\n  ${fields.join(',\n  ')}\n);`;
    }

    getCreateIndexSql(data: TableEditContext): string {
        // 创建索引
        let sql = [] as string[];
        data.indexs.res.forEach((a: IndexDefinition) => {
            const cols = a.columnNames.map((c) => this.quoteIdentifier(c)).join(',');
            sql.push(`CREATE ${a.unique ? 'UNIQUE ' : ''}INDEX ${this.quoteIdentifier(a.indexName)} ON ${this.quoteIdentifier(data.tableName)} (${cols})`);
        });
        return sql.join(';');
    }

    getDropTableSql(db: string, table: string): string {
        // sqlite 为单文件嵌入式库，无 schema 层级
        return `DROP TABLE ${this.quoteIdentifier(table)}`;
    }

    getModifyColumnSql(
        tableData: TableEditContext,
        tableName: string,
        changeData: ChangeDiff<RowDefinition>
    ): string {
        // sqlite修改表结构需要先删除再创建
        let sql = [] as string[];

        // 1.删除旧表索引
        tableData.indexs.res.forEach((a: IndexDefinition) => {
            a.indexName && sql.push(`DROP INDEX ${this.quoteIdentifier(a.indexName)}`);
        });

        // 2.重命名表，备份旧表
        let oldTableName = `_${tableName}_old_${new Date().getTime()}`;
        sql.push(`ALTER TABLE ${this.quoteIdentifier(tableName)} RENAME TO ${this.quoteIdentifier(oldTableName)}`);

        // 3.创建新表
        sql.push(this.getCreateTableSql(tableData));

        // 4.复制数据
        let delFields = changeData.del.map((a) => a.name);
        let addFields = changeData.add.map((a) => a.name);

        let queryFields = [] as string[];
        let insertFields = [] as string[];
        tableData.fields.res.forEach((a: RowDefinition) => {
            if (addFields.includes(a.name) || delFields.includes(a.name)) {
                return;
            }
            queryFields.push(this.quoteIdentifier(a.name === a.oldName ? a.name : (a.oldName ?? a.name)));
            insertFields.push(this.quoteIdentifier(a.name));
        });
        if (insertFields.length > 0 && queryFields.length > 0) {
            sql.push(
                `INSERT INTO ${this.quoteIdentifier(tableName)} (${insertFields.join(',')}) SELECT ${queryFields.join(',')} FROM ${this.quoteIdentifier(oldTableName)}`
            );
        }

        // 5.创建索引
        tableData.indexs.res.forEach((a: IndexDefinition) => {
            if (a.indexName) {
                const cols = a.columnNames.map((c) => this.quoteIdentifier(c)).join(',');
                sql.push(`CREATE ${a.unique ? 'UNIQUE ' : ''}INDEX ${this.quoteIdentifier(a.indexName)} ON ${this.quoteIdentifier(tableName)} (${cols})`);
            }
        });

        // 6.删除旧表
        sql.push(`DROP TABLE ${this.quoteIdentifier(oldTableName)}`);

        return sql.join(';') + ';';
    }

    getModifyIndexSql(tableData: TableEditContext, tableName: string, changeData: ChangeDiff<IndexDefinition>): string {
        let sql = [] as string[];

        if (changeData.del.length > 0) {
            changeData.del.forEach((a) => {
                sql.push(`DROP INDEX ${this.quoteIdentifier(a.indexName)}`);
            });
        }

        let indexData = [] as IndexDefinition[];
        if (changeData.add.length > 0) {
            indexData = indexData.concat(changeData.add);
        }
        if (changeData.upd.length > 0) {
            indexData = indexData.concat(changeData.upd);
        }

        if (indexData.length > 0) {
            indexData.forEach((a) => {
                const cols = a.columnNames.map((c) => this.quoteIdentifier(c)).join(',');
                sql.push(`CREATE ${a.unique ? 'UNIQUE ' : ''}INDEX ${this.quoteIdentifier(a.indexName)} ON ${this.quoteIdentifier(tableName)} (${cols})`);
            });
        }
        return sql.join(';');
    }

    getModifyTableInfoSql(tableData: TableInfoEditContext): string {
        // sqlite没有表注释
        let sql = '';
        if (tableData.tableName != tableData.oldTableName) {
            sql += `ALTER TABLE ${this.quoteIdentifier(tableData.oldTableName)} RENAME TO ${this.quoteIdentifier(tableData.tableName)};`;
        }
        return sql;
    }

    getDataType(columnType: string): DataType {
        return getDefaultDataType(columnType) as DataType;
    }

    wrapValue(columnType: string, value: unknown): string | number {
        return wrapValueDefault(columnType, value) as string | number;
    }

    getBatchInsertPreviewSql(tableName: string, fieldArr: string[], duplicateStrategy: DuplicateStrategy): string {
        let placeholder = '?'.repeat(fieldArr.length).split('').join(',');
        let prefix = 'insert into';
        if (duplicateStrategy === DuplicateStrategy.IGNORE) {
            prefix = 'insert or ignore into';
        } else if (duplicateStrategy === DuplicateStrategy.REPLACE) {
            prefix = 'insert or replace into';
        }
        return `${prefix} ${this.quoteIdentifier(tableName)}(${fieldArr.join(',')}) values (${placeholder});`;
    }
}

registerDbDialect(DbType.sqlite, new SqliteDialect());
