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
import { buildSchemaTable, extractSchema, getDefaultDataType, QuoteEscape, wrapValueMssql } from './shared/utils';
import { defineCapabilities, mssqlQuotePairs, mssqlSplitOptions } from './shared/capabilities';
import { diffPrimaryKey } from './shared/primaryKey';
import { offsetFetchPageSnippet } from './shared/snippets';
import { DbType } from './dbType';
import { registerDbDialect } from './registry';
import type { TableEditContext, TableInfoEditContext, ChangeDiff } from './types';

// 参考官方文档：https://docs.microsoft.com/zh-cn/sql/t-sql/data-types/data-types-transact-sql?view=sql-server-ver15
const MSSQL_TYPE_LIST = [
    //精确数字
    'bigint',
    'numeric',
    'bit',
    'smallint',
    'decimal',
    'smallmoney',
    'int',
    'tinyint',
    'money',
    // 近似数字
    'float',
    'real',
    // 日期和时间
    'date',
    'datetimeoffset',
    'datetime2',
    'smalldatetime',
    'datetime',
    'time',
    // 字符串
    'char',
    'varchar',
    'text',
    'nchar',
    'nvarchar',
    'ntext',
    'binary',
    'varbinary',

    // 其他
    'rowversion',
    'hierarchyid',
    'image',
    'uniqueidentifier',
    'sql_variant',
    'xml',
    // 空间几何类型 参照 https://learn.microsoft.com/zh-cn/sql/t-sql/spatial-geometry/spatial-types-geometry-transact-sql?view=sql-server-ver15
    'geometry',
    // 空间地理类型 参照 https://learn.microsoft.com/zh-cn/sql/t-sql/spatial-geography/spatial-types-geography?view=sql-server-ver15
    'geography',
];
// 函数参考官方文档 https://learn.microsoft.com/zh-cn/sql/t-sql/functions/functions?view=sql-server-ver15

let mssqlDialectInfo: DialectInfo;
let mssqlCompletions: EditorCompletion;

const customKeywords: EditorCompletionItem[] = [
    {
        label: 'select top ',
        description: 'keyword',
        insertText: 'select top 100 * from',
    },
    {
        label: 'select page ',
        description: 'keyword',
        insertText: 'SELECT *, 0 AS _ORDER_F_ FROM table_name \n ORDER BY _ORDER_F_ \n OFFSET 0 ROWS FETCH NEXT 25 ROWS ONLY;',
    },
];

/** 接受 `(长度[,小数位])` 的类型 */
const lengthArgTypes = ['char', 'varchar', 'nchar', 'nvarchar', 'binary', 'varbinary', 'decimal', 'numeric'];
/** 接受 `(小数秒精度)` 的时间类型（精度 0~7） */
const fspArgTypes = ['datetime2', 'time', 'datetimeoffset'];

/**
 * 列类型的参数片段。
 *
 * 编辑表单换类型时长度输入框会留着旧值，原样写出会生成 `int(64)`、`datetime(320)` 这类非法 DDL，
 * 故只对真正接受参数的类型族写出；时间类型额外限定小数秒精度区间（T-SQL 为 0~7）。
 */
function typeLengthSql(cl: ColumnDefinition): string {
    const type = (cl.type || '').toLowerCase().trim();
    if (!cl.length) {
        return '';
    }
    if (lengthArgTypes.includes(type)) {
        return cl.numScale ? `(${cl.length},${cl.numScale})` : `(${cl.length})`;
    }
    if (fspArgTypes.includes(type)) {
        const fsp = Number(cl.length);
        return Number.isInteger(fsp) && fsp >= 0 && fsp <= 7 ? `(${fsp})` : '';
    }
    return '';
}

class MssqlDialect implements DbDialect {
    getCapabilities(): DialectCapabilities {
        return defineCapabilities({
            supportsIndexComment: true,
            defaultIndexType: 'NONCLUSTERED',
            // T-SQL 撤销主键要 ALTER TABLE ... DROP CONSTRAINT <约束名>，名字由 t-keys 下发
            // IDENTITY 属性建表后不可变更，仅新建时可设置
            canEditAutoIncrementOnEdit: false,
            // go-mssqldb 一旦提供 CA 就同时校验主机名，无法只校链，故不声明 verify-ca
            tlsModes: ['require', 'verify-full'],
            quotePairs: mssqlQuotePairs,
            sqlSplitOptions: mssqlSplitOptions,
        });
    }

    getInfo(): DialectInfo {
        if (mssqlDialectInfo) {
            return mssqlDialectInfo;
        }

        mssqlDialectInfo = {
            name: 'MSSQL',
            icon: 'icon db/sqlserver',
            defaultPort: 1433,
            formatSqlDialect: 'transactsql',
            columnTypes: MSSQL_TYPE_LIST.map((a) => ({ udtName: a, dataType: a, desc: '', space: '' })),
        };
        return mssqlDialectInfo;
    }

    /** 编辑器联想词由 monaco 语言定义派生，按需加载并缓存（详见 types.ts 的 DialectInfo 注释） */
    async getEditorCompletions(): Promise<EditorCompletion> {
        if (mssqlCompletions) {
            return mssqlCompletions;
        }

        const { language: sqlLanguage } = await import('monaco-editor/languages/definitions/sql/sql.js');
        let { keywords, operators, builtinVariables, builtinFunctions } = sqlLanguage;
        let functions = builtinFunctions.map((a: string): EditorCompletionItem => ({ label: a, insertText: `${a}()`, description: 'func' }));

        let excludeKeywords = new Set(operators);
        mssqlCompletions = {
            keywords: keywords
                .filter((a: string) => !excludeKeywords.has(a)) // 移除已存在的operator、function
                .map((a: string): EditorCompletionItem => ({ label: a, description: 'keyword' }))
                .concat(customKeywords)
                .concat(commonCustomKeywords.map((a): EditorCompletionItem => ({ label: a, description: 'keyword' }))),
            operators: operators.map((a: string): EditorCompletionItem => ({ label: a, description: 'operator' })),
            functions,
            variables: builtinVariables.map((a: string): EditorCompletionItem => ({ label: a, description: 'var' })),
        };
        return mssqlCompletions;
    }

    getDefaultSelectSql(db: string, table: string, condition: string, orderBy: string, pageNum: number, limit: number) {
        let schema = extractSchema(db);
        return `SELECT *, 0 AS _MAY_ORDER_F_ FROM ${this.quoteIdentifier(schema)}.${this.quoteIdentifier(table)} ${condition ? 'WHERE ' + condition : ''} ${
            orderBy ? orderBy + ', _MAY_ORDER_F_' : 'order by _MAY_ORDER_F_'
        } ${this.getPageSql(pageNum, limit)};`.toUpperCase();
    }

    getPageSql(pageNum: number, limit: number) {
        return ` offset ${(pageNum - 1) * limit} rows fetch next ${limit} rows only`.toUpperCase();
    }

    getPreviewSql(sql: string, limit = 1): string {
        // T-SQL 无 LIMIT，TOP 须包裹为子查询
        return `SELECT TOP ${limit} * FROM (${sql}) a`;
    }

    getPageSnippet(): SqlSnippetTemplate {
        return offsetFetchPageSnippet;
    }

    getDefaultColumns(): ColumnDefinition[] {
        return createDefaultColumns(defaultColumnConfigs.mssql);
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
        return `[${name}]`;
    };

    genColumnBasicSql(cl: ColumnDefinition): string {
        let val = cl.value ? (cl.value === 'CURRENT_TIMESTAMP' ? cl.value : `'${cl.value}'`) : '';
        let defVal = val ? `DEFAULT ${val}` : '';
        const parts = [this.quoteIdentifier(cl.name), cl.type + typeLengthSql(cl), cl.autoIncrement ? 'IDENTITY(1,1)' : '', defVal, cl.nullable ? 'NULL' : 'NOT NULL'];
        return parts.filter(Boolean).join(' ');
    }

    /** MSSQL ALTER COLUMN 专用：不允许 IDENTITY 和 DEFAULT */
    genAlterColumnSql(cl: ColumnDefinition): string {
        const parts = [this.quoteIdentifier(cl.name), cl.type + typeLengthSql(cl), cl.nullable ? 'NULL' : 'NOT NULL'];
        return parts.filter(Boolean).join(' ');
    }

    getCreateTableSql(data: TableEditContext): string {
        let schema = extractSchema(data.db);

        // 创建表结构
        let pks = [] as string[];
        let fields: string[] = [];
        let fieldComments: string[] = [];
        data.fields.res.forEach((item: ColumnDefinition) => {
            item.name && fields.push(this.genColumnBasicSql(item));
            item.comment &&
                fieldComments.push(
                    `EXECUTE sp_addextendedproperty N'MS_Description', N'${QuoteEscape(item.comment)}', N'SCHEMA', N'${schema}', N'TABLE', N'${data.tableName}', N'COLUMN', N'${item.name}'`
                );
            if (item.isPrimaryKey) {
                pks.push(this.quoteIdentifier(item.name));
            }
        });

        let baseTable = `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(data.tableName)}`;

        // 建表语句
        const pkClause = pks.length > 0 ? `,\n  PRIMARY KEY CLUSTERED (${pks.join(',')})` : '';
        let createTable = `CREATE TABLE ${baseTable} (\n  ${fields.join(',\n  ')}${pkClause}\n);`;

        let createIndexSql = this.getCreateIndexSql(data);

        // 表注释
        if (data.tableComment) {
            createTable += `\nEXECUTE sp_addextendedproperty N'MS_Description', N'${QuoteEscape(data.tableComment)}', N'SCHEMA', N'${schema}', N'TABLE', N'${data.tableName}';`;
        }

        return createTable + (createIndexSql ? '\n' + createIndexSql : '') + (fieldComments.length > 0 ? '\n' + fieldComments.join(';\n') : '');
    }

    getCreateIndexSql(data: TableEditContext): string {
        let schema = extractSchema(data.db);
        let baseTable = buildSchemaTable(this.quoteIdentifier, data.db, data.tableName);

        let indexComment = [] as string[];

        // 创建索引
        let sql: string[] = [];
        data.indexes.res.forEach((a: IndexDefinition) => {
            let columnNames = a.columnNames.map((b: string) => this.quoteIdentifier(b));
            sql.push(`CREATE ${a.unique ? 'UNIQUE ' : ''}NONCLUSTERED INDEX ${this.quoteIdentifier(a.indexName)} ON ${baseTable} (${columnNames.join(',')})`);
            if (a.indexComment) {
                indexComment.push(
                    `EXECUTE sp_addextendedproperty N'MS_Description', N'${QuoteEscape(a.indexComment ?? '')}', N'SCHEMA', N'${schema}', N'TABLE', N'${data.tableName}', N'INDEX', N'${a.indexName}'`
                );
            }
        });

        let arr = [];
        sql.length > 0 && arr.push(sql.join(';\n'));
        indexComment.length > 0 && arr.push(indexComment.join(';\n'));
        return arr.join(';\n');
    }

    getDropTableSql(db: string, table: string): string {
        // 必须带 schema 限定，否则会落到连接用户的默认 schema 而删错表
        return `DROP TABLE ${buildSchemaTable(this.quoteIdentifier, db, table)}`;
    }

    getModifyColumnSql(tableData: TableEditContext, tableName: string, changeData: ChangeDiff<ColumnDefinition>): string {
        let schema = extractSchema(tableData.db);
        let baseTable = buildSchemaTable(this.quoteIdentifier, tableData.db, tableName);

        let delSql = '';
        let addArr = [] as string[];
        let renameArr = [] as string[];
        let updArr = [] as string[];
        let changeCommentArr = [] as string[];
        let addCommentArr = [] as string[];

        if (changeData.del.length > 0) {
            delSql = `ALTER TABLE ${baseTable} DROP ${changeData.del.map((a) => 'COLUMN ' + this.quoteIdentifier(a.name)).join(',')};`;
        }
        if (changeData.add.length > 0) {
            changeData.add.forEach((a) => {
                addArr.push(`ALTER TABLE ${baseTable} ADD ${this.genColumnBasicSql(a)}`);
                if (a.comment) {
                    addCommentArr.push(
                        `EXECUTE sp_addextendedproperty N'MS_Description', N'${QuoteEscape(a.comment)}', N'SCHEMA', N'${schema}', N'TABLE', N'${tableName}', N'COLUMN', N'${a.name}'`
                    );
                }
            });
        }

        if (changeData.upd.length > 0) {
            changeData.upd.forEach((a) => {
                if (a.oldName && a.name !== a.oldName) {
                    renameArr.push(`EXEC sp_rename '${baseTable}.${this.quoteIdentifier(a.oldName)}', '${QuoteEscape(a.name)}', 'COLUMN'`);
                }
                // ALTER COLUMN 只允许 type 和 nullability，不允许 IDENTITY/DEFAULT
                updArr.push(`ALTER TABLE ${baseTable} ALTER COLUMN ${this.genAlterColumnSql(a)}`);
                if (a.comment) {
                    changeCommentArr.push(`IF ((SELECT COUNT(*) FROM fn_listextendedproperty('MS_Description',
'SCHEMA', N'${schema}',
'TABLE', N'${tableName}',
'COLUMN', N'${a.name}')) > 0)
  EXEC sp_updateextendedproperty
'MS_Description', N'${QuoteEscape(a.comment)}',
'SCHEMA', N'${schema}',
'TABLE', N'${tableName}',
'COLUMN', N'${a.name}'
ELSE
  EXEC sp_addextendedproperty
'MS_Description', N'${QuoteEscape(a.comment)}',
'SCHEMA', N'${schema}',
'TABLE', N'${tableName}',
'COLUMN', N'${a.name}'`);
                }
            });
        }

        // 主键是表级约束：T-SQL 同样没有 DROP PRIMARY KEY，要 ALTER TABLE ... DROP CONSTRAINT <名字>；
        // 名字由元数据 t-keys 下发，缺失时退回 `PK_<表>` 约定
        const primaryKey = diffPrimaryKey(tableData.fields.oldFields, tableData.fields.res);
        const hasOldPrimaryKey = tableData.fields.oldFields.some((field) => field.isPrimaryKey);
        const constraintName = tableData.primaryKeyName || `PK_${tableName}`;
        let pkSql = '';
        if (primaryKey.changed && hasOldPrimaryKey) {
            pkSql += `ALTER TABLE ${baseTable} DROP CONSTRAINT ${this.quoteIdentifier(constraintName)};\n`;
        }
        if (primaryKey.changed && primaryKey.columns.length > 0) {
            const cols = primaryKey.columns.map((name) => this.quoteIdentifier(name)).join(',');
            pkSql += `ALTER TABLE ${baseTable} ADD CONSTRAINT ${this.quoteIdentifier(constraintName)} PRIMARY KEY (${cols});\n`;
        }

        let arr = [];
        delSql && arr.push(delSql);
        addArr.length > 0 && arr.push(addArr.join(';\n'));
        renameArr.length > 0 && arr.push(renameArr.join(';\n'));
        updArr.length > 0 && arr.push(updArr.join(';\n'));
        changeCommentArr.length > 0 && arr.push(changeCommentArr.join(';\n'));
        addCommentArr.length > 0 && arr.push(addCommentArr.join(';\n'));
        pkSql && arr.push(pkSql.trim());

        return arr.join(';\n');
    }

    getModifyIndexSql(tableData: TableEditContext, tableName: string, changeData: ChangeDiff<IndexDefinition>): string {
        let schema = extractSchema(tableData.db);
        let baseTable = buildSchemaTable(this.quoteIdentifier, tableData.db, tableName);

        let dropArr = [] as string[];
        let addArr = [] as string[];
        let commentArr = [] as string[];

        const pushDrop = (a: IndexDefinition) => {
            dropArr.push(`DROP INDEX ${this.quoteIdentifier(a.indexName)} ON ${baseTable}`);
        };
        const pushAdd = (a: IndexDefinition) => {
            addArr.push(
                `CREATE ${a.unique ? 'UNIQUE ' : ''}NONCLUSTERED INDEX ${this.quoteIdentifier(a.indexName)} ON ${baseTable} (${a.columnNames.map((b: string) => this.quoteIdentifier(b)).join(',')})`
            );
            if (a.indexComment) {
                commentArr.push(
                    `EXECUTE sp_addextendedproperty N'MS_Description', N'${QuoteEscape(a.indexComment)}', N'SCHEMA', N'${schema}', N'TABLE', N'${tableName}', N'INDEX', N'${a.indexName}'`
                );
            }
        };

        if (changeData.upd.length > 0) {
            changeData.upd.forEach((a) => {
                pushDrop(a);
                pushAdd(a);
            });
        }

        if (changeData.del.length > 0) {
            changeData.del.forEach((a) => {
                pushDrop(a);
            });
        }

        if (changeData.add.length > 0) {
            changeData.add.forEach((a) => pushAdd(a));
        }
        let dropSql = dropArr.join(';\n');
        let addSql = addArr.join(';\n');
        let commentSql = commentArr.join(';\n');

        let arr = [];
        dropSql && arr.push(dropSql);
        addSql && arr.push(addSql);
        commentSql && arr.push(commentSql);
        return arr.join(';\n');
    }

    getModifyTableInfoSql(tableData: TableInfoEditContext): string {
        let schema = extractSchema(tableData.db);

        let sql = '';

        if (tableData.oldTableName !== tableData.tableName) {
            let baseTable = `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(tableData.oldTableName)}`;
            sql += `EXEC sp_rename '${baseTable}', '${tableData.tableName}';\n`;
        }

        if (tableData.oldTableComment !== tableData.tableComment) {
            let tableComment = tableData.tableComment.replaceAll(/'/g, "'").replaceAll(/[\r\n]/g, ' ');
            sql += `IF ((SELECT COUNT(*) FROM fn_listextendedproperty('MS_Description',
'SCHEMA', N'${schema}',
'TABLE', N'${tableData.tableName}', NULL, NULL)) > 0)
  EXEC sp_updateextendedproperty
'MS_Description', N'${tableComment}',
'SCHEMA', N'${schema}',
'TABLE', N'${tableData.tableName}'
ELSE
  EXEC sp_addextendedproperty
'MS_Description', N'${tableComment}',
'SCHEMA', N'${schema}',
'TABLE', N'${tableData.tableName}'`;
        }
        return sql;
    }

    getDataType(columnType: string): DataType {
        return getDefaultDataType(columnType) as DataType;
    }

    wrapValue(columnType: string, value: unknown): string | number {
        return wrapValueMssql(columnType, value) as string | number;
    }

    getBatchInsertPreviewSql(tableName: string, fieldArr: string[], duplicateStrategy: DuplicateStrategy): string {
        let placeholder = '?'.repeat(fieldArr.length).split('').join(',');
        let baseSql = `INSERT INTO ${tableName} (${fieldArr.join(',')}) VALUES (${placeholder});`;
        if (duplicateStrategy === DuplicateStrategy.IGNORE) {
            let on = `ALTER TABLE ${tableName} ADD CONSTRAINT uniqueRows UNIQUE (id) WITH (IGNORE_DUP_KEY = ON);`;
            return on + '\n' + baseSql;
        }

        if (duplicateStrategy === DuplicateStrategy.REPLACE) {
            // 字段数组生成占位符sql
            let phs = [];
            let values = [];
            for (let i = 0; i < fieldArr.length; i++) {
                phs.push(`? ${fieldArr[i]}`);
                values.push(`T2.${fieldArr[i]}`);
            }
            let placeholder = phs.join(',');
            let sql = `MERGE INTO ${tableName} T1 USING 
        (
         SELECT ${placeholder}
        ) T2 ON (T1.id = T2.id) 
        WHEN NOT MATCHED THEN INSERT(${fieldArr.join(',')}) VALUES (${values.join(',')})
        WHEN MATCHED THEN UPDATE SET ${fieldArr.map((a) => `T1.${a} = T2.${a}`).join(',')}`;
            return sql;
        }

        return baseSql;
    }
}

registerDbDialect(DbType.mssql, new MssqlDialect());
