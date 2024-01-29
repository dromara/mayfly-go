import { DbInst } from '../db';
import { commonCustomKeywords, DataType, DbDialect, DialectInfo, EditorCompletion, EditorCompletionItem, IndexDefinition, RowDefinition } from './index';
import { language as sqlLanguage } from 'monaco-editor/esm/vs/basic-languages/sql/sql.js';

export { MSSQL_TYPE_LIST, MssqlDialect };

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
    'cursor',
    'rowversion',
    'hierarchyid',
    'uniqueidentifier',
    'sql_variant',
    'xml',
    'table',
    // 空间几何类型 参照 https://learn.microsoft.com/zh-cn/sql/t-sql/spatial-geometry/spatial-types-geometry-transact-sql?view=sql-server-ver15
    'geometry',
    // 空间地理类型 参照 https://learn.microsoft.com/zh-cn/sql/t-sql/spatial-geography/spatial-types-geography?view=sql-server-ver15
    'geography',
];
// 函数参考官方文档 https://learn.microsoft.com/zh-cn/sql/t-sql/functions/functions?view=sql-server-ver15

let mssqlDialectInfo: DialectInfo;

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

const fixedLengthTypes = [
    'int',
    'bigint',
    'smallint',
    'tinyint',
    'float',
    'real',
    'datetime',
    'smalldatetime',
    'date',
    'time',
    'datetime2',
    'datetimeoffset',
    'bit',
    'uniqueidentifier',
    'geometry',
    'geography',
];

class MssqlDialect implements DbDialect {
    getInfo(): DialectInfo {
        if (mssqlDialectInfo) {
            return mssqlDialectInfo;
        }

        let { keywords, operators, builtinVariables, builtinFunctions } = sqlLanguage;
        let functions = builtinFunctions.map((a: string): EditorCompletionItem => ({ label: a, insertText: `${a}()`, description: 'func' }));

        let excludeKeywords = new Set(operators);
        let editorCompletions: EditorCompletion = {
            keywords: keywords
                .filter((a: string) => !excludeKeywords.has(a)) // 移除已存在的operator、function
                .map((a: string): EditorCompletionItem => ({ label: a, description: 'keyword' }))
                .concat(customKeywords)
                .concat(commonCustomKeywords.map((a): EditorCompletionItem => ({ label: a, description: 'keyword' }))),
            operators: operators.map((a: string): EditorCompletionItem => ({ label: a, description: 'operator' })),
            functions,
            variables: builtinVariables.map((a: string): EditorCompletionItem => ({ label: a, description: 'var' })),
        };

        mssqlDialectInfo = {
            name: 'MSSQL',
            icon: 'iconfont icon-MSSQLNATIVE',
            defaultPort: 1433,
            formatSqlDialect: 'transactsql',
            columnTypes: MSSQL_TYPE_LIST.map((a) => ({ udtName: a, dataType: a, desc: '', space: '' })),
            editorCompletions,
        };
        return mssqlDialectInfo;
    }

    getDefaultSelectSql(db: string, table: string, condition: string, orderBy: string, pageNum: number, limit: number) {
        let schema = db.split('/')[1];
        return `SELECT *, 0 AS _MAY_ORDER_F_ FROM ${this.quoteIdentifier(schema)}.${this.quoteIdentifier(table)} ${condition ? 'WHERE ' + condition : ''} ${
            orderBy ? orderBy + ', _MAY_ORDER_F_' : 'order by _MAY_ORDER_F_'
        } ${this.getPageSql(pageNum, limit)};`.toUpperCase();
    }

    getPageSql(pageNum: number, limit: number) {
        return ` offset ${(pageNum - 1) * limit} rows fetch next ${limit} rows only`.toUpperCase();
    }

    getDefaultRows(): RowDefinition[] {
        return [
            { name: 'id', type: 'bigint', length: '', numScale: '', value: '', notNull: true, pri: true, auto_increment: true, remark: '主键ID' },
            { name: 'creator_id', type: 'bigint', length: '20', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '创建人id' },
            {
                name: 'creator',
                type: 'varchar',
                length: '100',
                numScale: '',
                value: '',
                notNull: true,
                pri: false,
                auto_increment: false,
                remark: '创建人姓名',
            },
            {
                name: 'create_time',
                type: 'datetime2',
                length: '',
                numScale: '',
                value: 'CURRENT_TIMESTAMP',
                notNull: true,
                pri: false,
                auto_increment: false,
                remark: '创建时间',
            },
            { name: 'updator_id', type: 'bigint', length: '20', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '修改人id' },
            {
                name: 'updator',
                type: 'varchar',
                length: '100',
                numScale: '',
                value: '',
                notNull: true,
                pri: false,
                auto_increment: false,
                remark: '修改人姓名',
            },
            {
                name: 'update_time',
                type: 'datetime2',
                length: '',
                numScale: '',
                value: 'CURRENT_TIMESTAMP',
                notNull: true,
                pri: false,
                auto_increment: false,
                remark: '修改时间',
            },
        ];
    }

    getDefaultIndex(): IndexDefinition {
        return {
            indexName: '',
            columnNames: [],
            unique: false,
            indexType: 'NONCLUSTERED',
            indexComment: '',
        };
    }

    quoteIdentifier = (name: string) => {
        return `[${name}]`;
    };

    genColumnBasicSql(cl: any): string {
        let val = cl.value ? (cl.value === 'CURRENT_TIMESTAMP' ? cl.value : `'${cl.value}'`) : '';
        let defVal = val ? `DEFAULT ${val}` : '';
        // mssql哪些字段允许有长度
        let length = '';
        if (!fixedLengthTypes.includes(cl.type)) {
            length = cl.length ? `(${cl.length})` : '';
        }
        return ` ${this.quoteIdentifier(cl.name)} ${cl.type}${length} ${cl.auto_increment ? 'IDENTITY(1,1)' : ''} ${defVal} ${cl.notNull ? 'NOT NULL' : 'NULL'} `;
    }
    getCreateTableSql(data: any): string {
        let schema = data.db.split('/')[1];

        // 创建表结构
        let pks = [] as string[];
        let fields: string[] = [];
        let fieldComments: string[] = [];
        data.fields.res.forEach((item: any) => {
            item.name && fields.push(this.genColumnBasicSql(item));
            item.remark &&
                fieldComments.push(
                    `EXECUTE sp_addextendedproperty N'MS_Description', N'${item.remark}', N'SCHEMA', N'${schema}', N'TABLE', N'${data.tableName}', N'COLUMN', N'${item.name}'`
                );
            if (item.pri) {
                pks.push(`${this.quoteIdentifier(item.name)}`);
            }
        });

        let baseTable = `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(data.tableName)}`;

        // 建表语句
        let createTable = `CREATE TABLE ${baseTable}
                ( ${fields.join(',')}
                  ${pks.length > 0 ? `, PRIMARY KEY CLUSTERED (${pks.join(',')})` : ''}
                );`;

        let createIndexSql = this.getCreateIndexSql(data);

        // 表注释
        if (data.tableComment) {
            createTable += ` EXECUTE sp_addextendedproperty N'MS_Description', N'${data.tableComment}', N'SCHEMA', N'${schema}', N'TABLE', N'${data.tableName}';`;
        }

        return createTable + createIndexSql + fieldComments.join(';');
    }

    getCreateIndexSql(data: any): string {
        // CREATE UNIQUE NONCLUSTERED INDEX [aaa]
        // ON [dbo].[无标题] (
        //   [id],
        //   [name]
        // )
        let schema = data.db.split('/')[1];
        let baseTable = `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(data.tableName)}`;

        let indexComment = [] as string[];

        // 创建索引
        let sql: string[] = [];
        data.indexs.res.forEach((a: any) => {
            let columnNames = a.columnNames.map((b: string) => `${this.quoteIdentifier(b)}`);
            sql.push(` CREATE ${a.unique ? 'UNIQUE' : ''} NONCLUSTERED INDEX ${this.quoteIdentifier(a.indexName)} on ${baseTable} (${columnNames.join(',')})`);
            if (a.indexComment) {
                indexComment.push(
                    `EXECUTE sp_addextendedproperty N'MS_Description', N'${a.indexComment}', N'SCHEMA', N'${schema}', N'TABLE', N'${data.tableName}', N'INDEX', N'${a.indexName}'`
                );
            }
        });

        return sql.join(';') + ';' + indexComment.join(';');
    }

    getModifyColumnSql(tableData: any, tableName: string, changeData: { del: RowDefinition[]; add: RowDefinition[]; upd: RowDefinition[] }): string {
        // sql执行顺序
        // 1. 删除字段
        // 2. 添加字段
        // 3. 修改字段名字
        // 4. 修改字段类型
        // 5. 修改字段注释
        // 6. 添加字段注释

        let schema = tableData.db.split('/')[1];
        let baseTable = `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(tableName)}`;

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
                addArr.push(` ALTER TABLE ${baseTable} ADD COLUMN ${this.genColumnBasicSql(a)}`);
                if (a.remark) {
                    addCommentArr.push(
                        `EXECUTE sp_addextendedproperty N'MS_Description', N'${a.remark}', N'SCHEMA', N'${schema}', N'TABLE', N'${tableName}', N'COLUMN', N'${a.name}'`
                    );
                }
            });
        }

        if (changeData.upd.length > 0) {
            changeData.upd.forEach((a) => {
                if (a.oldName && a.name !== a.oldName) {
                    renameArr.push(` EXEC sp_rename '${baseTable}.${this.quoteIdentifier(a.oldName)}', '${a.name}', 'COLUMN' `);
                } else {
                    updArr.push(` ALTER TABLE ${baseTable} ALTER COLUMN ${this.genColumnBasicSql(a)} `);
                }
                if (a.remark) {
                    changeCommentArr.push(`IF ((SELECT COUNT(*) FROM fn_listextendedproperty('MS_Description',
'SCHEMA', N'${schema}',
'TABLE', N'${tableName}',
'COLUMN', N'${a.name}')) > 0)
  EXEC sp_updateextendedproperty
'MS_Description', N'${a.remark}',
'SCHEMA', N'${schema}',
'TABLE', N'${tableName}',
'COLUMN', N'${a.name}'
ELSE
  EXEC sp_addextendedproperty
'MS_Description', N'${a.remark}',
'SCHEMA', N'${schema}',
'TABLE', N'${tableName}',
'COLUMN',N'${a.name}'`);
                }
            });
        }

        return (
            delSql +
            (addArr.length > 0 ? addArr.join(';') + ';' : '') +
            (renameArr.length > 0 ? renameArr.join(';') + ';' : '') +
            (updArr.length > 0 ? updArr.join(';') + ';' : '') +
            (changeCommentArr.length > 0 ? changeCommentArr.join(';') + ';' : '') +
            (addCommentArr.length > 0 ? addCommentArr.join(';') + ';' : '')
        );
    }

    getModifyIndexSql(tableData: any, tableName: string, changeData: { del: any[]; add: any[]; upd: any[] }): string {
        let schema = tableData.db.split('/')[1];
        let baseTable = `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(tableName)}`;

        let dropArr = [] as string[];
        let addArr = [] as string[];
        let commentArr = [] as string[];

        const pushDrop = (a: any) => {
            dropArr.push(` DROP INDEX ${this.quoteIdentifier(a.indexName)} ON ${baseTable} `);
        };
        const pushAdd = (a: any) => {
            addArr.push(
                ` CREATE ${a.unique ? 'UNIQUE' : ''} NONCLUSTERED INDEX ${this.quoteIdentifier(a.indexName)} ON ${baseTable} (${a.columnNames.map((b: string) => this.quoteIdentifier(b)).join(',')}) `
            );
            if (a.indexComment) {
                commentArr.push(
                    ` EXEC sp_addextendedproperty N'MS_Description', N'${a.indexComment}', N'SCHEMA', N'${schema}', N'TABLE', N'${tableName}', N'INDEX', N'${a.indexName}' `
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
        let dropSql = dropArr.join(';');
        let addSql = addArr.join(';');
        let commentSql = commentArr.join(';');
        return dropSql + ';' + addSql + ';' + commentSql + ';';
    }

    getDataType(columnType: string): DataType {
        if (DbInst.isNumber(columnType)) {
            return DataType.Number;
        }
        // 日期时间类型
        if (/datetime|timestamp/gi.test(columnType)) {
            return DataType.DateTime;
        }
        // 日期类型
        if (/date/gi.test(columnType)) {
            return DataType.Date;
        }
        // 时间类型
        if (/time/gi.test(columnType)) {
            return DataType.Time;
        }
        return DataType.String;
    }
    // eslint-disable-next-line @typescript-eslint/no-unused-vars,no-unused-vars
    wrapStrValue(columnType: string, value: string): string {
        return `'${value}'`;
    }
}
