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
import { appendLimitSql, buildSchemaTable, extractSchema, getDefaultDataType, matchType as _matchType, QuoteEscape, wrapValueDefault } from './shared/utils';
import { defineCapabilities, pgSplitOptions } from './shared/capabilities';
import { limitOffsetPageSnippet } from './shared/snippets';
import { DbType } from './dbType';
import { registerDbDialect } from './registry';
import type { TableEditContext, TableInfoEditContext, ChangeDiff } from './types';

const GAUSS_TYPE_LIST: sqlColumnType[] = [
    // 数值 - 整数型
    { udtName: 'int1', dataType: 'tinyint', desc: '微整数，别名为INT1', space: '1字节', range: '0 ~ +255' },
    { udtName: 'int2', dataType: 'smallint', desc: '小范围整数，别名为INT2。', space: '2字节', range: '-32,768 ~ +32,767' },
    { udtName: 'int4', dataType: 'integer', desc: '常用的整数，别名为INT4。', space: '4字节', range: '-2,147,483,648 ~ +2,147,483,647' },
    { udtName: 'int8', dataType: 'bigint', desc: '大范围的整数，别名为INT8。', space: '8字节', range: '很大' },

    // 数值 - 任意精度型
    {
        udtName: 'numeric',
        dataType: 'numeric',
        desc: '精度(总位数)取值范围为[1,1000]，标度(小数位数)取值范围为[0,精度]。',
        space: '每四位（十进制位）占用两个字节，然后在整个数据上加上八个字节的额外开销',
        range: '未指定精度的情况下，小数点前最大131,072位，小数点后最大16,383位',
    },
    // 数值 - 任意精度型
    { udtName: 'decimal', dataType: 'decimal', desc: '等同于number类型', space: '等同于number类型' },

    // 数值 - 序列整型
    { udtName: 'smallserial', dataType: 'smallserial', desc: '二字节序列整型。', space: '2字节', range: '-32,768 ~ +32,767' },
    { udtName: 'serial', dataType: 'serial', desc: '四字节序列整型。', space: '4字节', range: '-2,147,483,648 ~ +2,147,483,647' },
    { udtName: 'bigserial', dataType: 'bigserial', desc: '八字节序列整型', space: '8字节', range: '-9,223,372,036,854,775,808 ~ +9,223,372,036,854,775,807' },
    {
        udtName: 'largeserial',
        dataType: 'largeserial',
        desc: '默认插入十六字节序列整型，实际数值类型和numeric相同',
        space: '变长类型，每四位（十进制位）占用两个字节，然后在整个数据上加上八个字节的额外开销。',
        range: '小数点前最大131,072位，小数点后最大16,383位。',
    },

    // 数值 - 浮点类型（不常用 就不列出来了）

    // 货币类型
    { udtName: 'money', dataType: 'money', desc: '货币金额', space: '8字节', range: '-92233720368547758.08 ~ +92233720368547758.07' },

    // 布尔类型
    { udtName: 'bool', dataType: 'bool', desc: '布尔类型', space: '1字节', range: 'true：真 , false：假 , null：未知（unknown）' },

    // 字符类型
    { udtName: 'char', dataType: 'char', desc: '定长字符串，不足补空格。n是指字节长度，如不带精度n，默认精度为1。', space: '最大为10MB' },
    { udtName: 'character', dataType: 'character', desc: '定长字符串，不足补空格。n是指字节长度，如不带精度n，默认精度为1。', space: '最大为10MB' },
    { udtName: 'nchar', dataType: 'nchar', desc: '定长字符串，不足补空格。n是指字节长度，如不带精度n，默认精度为1。', space: '最大为10MB' },
    { udtName: 'varchar', dataType: 'varchar', desc: '变长字符串。PG兼容模式下，n是字符长度。其他兼容模式下，n是指字节长度。', space: '最大为10MB。' },
    { udtName: 'text', dataType: 'text', desc: '变长字符串。', space: '最大稍微小于1GB-1。' },
    { udtName: 'clob', dataType: 'clob', desc: '文本大对象。是TEXT类型的别名。', space: '最大稍微小于32TB-1。' },

    //特殊字符类型  用的很少，先屏蔽了
    // { udtName: 'name', dataType: 'name', desc: '用于对象名的内部类型。', space: '64字节。' },
    // { udtName: '"char"', dataType: '"char"', desc: '单字节内部类型。', space: '1字节。' },

    // 二进制类型
    { udtName: 'bytea', dataType: 'bytea', desc: '变长的二进制字符串', space: '4字节加上实际的二进制字符串。最大为1GB减去8203字节（即1073733621字节）。' },

    // 日期/时间类型
    { udtName: 'date', dataType: 'date', desc: '日期', space: '4字节' },
    { udtName: 'time', dataType: 'time', desc: 'TIME [(p)] 只用于一日内时间,p表示小数点后的精度，取值范围为0~6。', space: '8-12字节' },
    { udtName: 'timestamp', dataType: 'timestamp', desc: 'TIMESTAMP[(p)]日期和时间,p表示小数点后的精度，取值范围为0~6', space: '8字节' },
    // 带时区的时间戳用的少，先屏蔽了
    //{ udtName: 'TIMESTAMPTZ', dataType: 'TIMESTAMP WITH TIME ZONE', desc: '带时区的时间戳', space: '8字节' },
    {
        udtName: 'interval',
        dataType: 'interval',
        desc: '时间间隔', // 可以跟参数：YEAR，MONTH，DAY，HOUR，MINUTE，SECOND，DAY TO HOUR，DAY TO MINUTE，DAY TO SECOND，HOUR TO MINUTE，HOUR TO SECOND，MINUTE TO SECOND
        space: '精度取值范围为0~6，且参数为SECOND，DAY TO SECOND，HOUR TO SECOND或MINUTE TO SECOND时，参数p才有效',
    },
    // 几何类型
    { udtName: 'point', dataType: 'point', desc: '平面中的点， 如:(x,y)', space: '16字节' },
    { udtName: 'lseg', dataType: 'lseg', desc: '（有限）线段， 如:((x1,y1),(x2,y2))', space: '32字节' },
    { udtName: 'box', dataType: 'box', desc: '矩形， 如:((x1,y1),(x2,y2))', space: '32字节' },
    { udtName: 'path', dataType: 'path', desc: '闭合路径（与多边形类似）， 如:((x1,y1),...)', space: '16+16n字节' },
    { udtName: 'path', dataType: 'path', desc: '开放路径（与多边形类似）， 如:[(x1,y1),...]', space: '16+16n字节' },
    { udtName: 'polygon', dataType: 'polygon', desc: '多边形（与闭合路径相似）， 如:((x1,y1),...)', space: '40+16n字节' },
    { udtName: 'circle', dataType: 'polygon', desc: '圆,如:<(x,y),r> （圆心和半径）', space: '24 字节' },

    // 网络地址类型
    { udtName: 'cidr', dataType: 'cidr', desc: 'IPv4网络', space: '7字节' },
    { udtName: 'inet', dataType: 'inet', desc: 'IPv4主机和网络', space: '7字节' },
    { udtName: 'macaddr', dataType: 'macaddr', desc: 'MAC地址', space: '6字节' },
];

const replaceFunctions: EditorCompletionItem[] = [];

let pgDialectInfo: DialectInfo;
let pgCompletions: EditorCompletion;

export class PostgresqlDialect implements DbDialect {
    getCapabilities(): DialectCapabilities {
        return defineCapabilities({
            // 自增由 serial/identity 列类型表达而非独立列属性，故表编辑器不提供自增勾选
            canEditAutoIncrementOnCreate: false,
            canEditAutoIncrementOnEdit: false,
            sqlSplitOptions: pgSplitOptions,
        });
    }

    getInfo(): DialectInfo {
        if (pgDialectInfo) {
            return pgDialectInfo;
        }

        pgDialectInfo = {
            name: 'PostgreSQL',
            icon: 'icon db/postgres',
            defaultPort: 5432,
            formatSqlDialect: 'postgresql',
            columnTypes: GAUSS_TYPE_LIST.sort((a, b) => a.udtName.localeCompare(b.udtName)),
        };
        return pgDialectInfo;
    }

    /** 编辑器联想词由 monaco 语言定义派生，按需加载并缓存（详见 types.ts 的 DialectInfo 注释） */
    async getEditorCompletions(): Promise<EditorCompletion> {
        if (pgCompletions) {
            return pgCompletions;
        }

        const { language: pgsqlLanguage } = await import('monaco-editor/languages/definitions/pgsql/pgsql.js');
        let { keywords, operators, builtinVariables, builtinFunctions } = pgsqlLanguage;
        let replaceFunctionNames = replaceFunctions.map((a) => a.label);
        let functions = builtinFunctions
            .filter((a: string) => replaceFunctionNames.indexOf(a) < 0)
            .map((a: string): EditorCompletionItem => ({ label: a, insertText: `${a}()`, description: 'func' }))
            .concat(replaceFunctions);
        let excludeKeywords = new Set(builtinFunctions.concat(replaceFunctionNames).concat(operators));

        pgCompletions = {
            keywords: keywords
                .filter((a: string) => !excludeKeywords.has(a)) // 移除已存在的operator、function
                .map((a: string): EditorCompletionItem => ({ label: a, description: 'keyword' }))
                .concat(commonCustomKeywords.map((a): EditorCompletionItem => ({ label: a, description: 'keyword' }))),
            operators: operators.map((a: string): EditorCompletionItem => ({ label: a, description: 'operator' })),
            functions,
            variables: builtinVariables.map((a: string): EditorCompletionItem => ({ label: a, description: 'var' })),
        };
        return pgCompletions;
    }

    getDefaultSelectSql(db: string, table: string, condition: string, orderBy: string, pageNum: number, limit: number) {
        return `SELECT * FROM ${this.quoteIdentifier(table)} ${condition ? 'WHERE ' + condition : ''} ${orderBy ? orderBy : ''} ${this.getPageSql(
            pageNum,
            limit
        )};`;
    }

    getPageSql(pageNum: number, limit: number) {
        return ` OFFSET ${(pageNum - 1) * limit} LIMIT ${limit};`;
    }

    getPreviewSql(sql: string, limit = 1): string {
        return appendLimitSql(sql, limit);
    }

    getPageSnippet(): SqlSnippetTemplate {
        return limitOffsetPageSnippet;
    }

    getDefaultRows(): RowDefinition[] {
        return createDefaultRows(defaultRowsConfigs.postgresql);
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
        return `"${name}"`;
    };

    matchType(text: string, arr: string[]): boolean {
        return _matchType(text, arr);
    }

    getDefaultValueSql(cl: RowDefinition): string {
        if (cl.value && cl.value.length > 0) {
            // 哪些字段默认值需要加引号
            let marks = false;
            if (this.matchType(cl.type, ['char', 'time', 'date', 'text'])) {
                // 默认值是now()的time或date不需要加引号
                if (['pg_systimestamp()', 'current_timestamp'].includes(cl.value.toLowerCase()) && this.matchType(cl.type, ['time', 'date'])) {
                    marks = false;
                } else {
                    marks = true;
                }
            }
            // 哪些函数不需要加引号
            if (this.matchType(cl.value, ['nextval'])) {
                marks = false;
            }
            return ` DEFAULT ${marks ? "'" : ''}${cl.value}${marks ? "'" : ''}`;
        }
        return '';
    }

    getTypeLengthSql(cl: RowDefinition) {
        // 哪些字段可以指定长度
        if (cl.length && this.matchType(cl.type, ['char', 'time', 'bit', 'num', 'decimal'])) {
            // 哪些字段类型可以指定小数点
            if (cl.numScale && this.matchType(cl.type, ['num', 'decimal'])) {
                return `(${cl.length}, ${cl.numScale})`;
            } else {
                return `(${cl.length})`;
            }
        }
        return '';
    }

    genColumnBasicSql(cl: RowDefinition): string {
        let length = this.getTypeLengthSql(cl);
        // 默认值
        let defVal = this.getDefaultValueSql(cl);
        // 如果有原名以原名为准
        let name = cl.oldName && cl.name !== cl.oldName ? cl.oldName : cl.name;

        const parts = [
            this.quoteIdentifier(name),
            cl.type + length,
            cl.notNull ? 'NOT NULL' : '',
            defVal,
        ];
        return parts.filter(Boolean).join(' ');
    }

    getCreateTableSql(data: TableEditContext): string {
        let createSql = '';
        let tableCommentSql = '';
        let columCommentSql = '';

        // 创建表结构
        let pks = [] as string[];
        let fields: string[] = [];
        data.fields.res.forEach((item: RowDefinition) => {
            item.name && fields.push(this.genColumnBasicSql(item));
            if (item.pri) {
                pks.push(this.quoteIdentifier(item.name));
            }
            // 列注释
            if (item.remark) {
                columCommentSql += `COMMENT ON COLUMN ${this.quoteIdentifier(data.tableName)}.${this.quoteIdentifier(item.name)} IS '${QuoteEscape(item.remark)}';`;
            }
        });
        // 建表
        const pkClause = pks.length > 0 ? `,\n  PRIMARY KEY (${pks.join(',')})` : '';
        createSql = `CREATE TABLE ${this.quoteIdentifier(data.tableName)} (\n  ${fields.join(',\n  ')}${pkClause}\n);`;
        // 表注释
        if (data.tableComment) {
            tableCommentSql = `COMMENT ON TABLE ${this.quoteIdentifier(data.tableName)} IS '${QuoteEscape(data.tableComment)}';`;
        }

        return createSql + tableCommentSql + columCommentSql;
    }

    getCreateIndexSql(tableData: TableEditContext): string {
        let dbTable = buildSchemaTable(this.quoteIdentifier, tableData.db, tableData.tableName);
        let schema = extractSchema(tableData.db);
        let sql: string[] = [];
        tableData.indexs.res.forEach((a: IndexDefinition) => {
            let colArr = a.columnNames.map((c: string) => this.quoteIdentifier(c));
            sql.push(`CREATE ${a.unique ? 'UNIQUE ' : ''}INDEX ${this.quoteIdentifier(a.indexName)} ON ${dbTable} (${colArr.join(',')})`);
            if (a.indexComment) {
                const idxRef = schema ? `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(a.indexName)}` : this.quoteIdentifier(a.indexName);
                sql.push(`COMMENT ON INDEX ${idxRef} IS '${QuoteEscape(a.indexComment)}'`);
            }
        });
        return sql.join(';');
    }

    getDropTableSql(db: string, table: string): string {
        // 必须带 schema 限定，否则会落到连接 search_path 的首个 schema 而删错表
        return `DROP TABLE ${buildSchemaTable(this.quoteIdentifier, db, table)}`;
    }

    getModifyColumnSql(tableData: TableEditContext, tableName: string, changeData: ChangeDiff<RowDefinition>): string {
        let dbTable = buildSchemaTable(this.quoteIdentifier, tableData.db, tableName);

        let modifySql = '';
        let dropSql = '';
        let renameSql = '';
        let commentSql = '';

        if (changeData.add.length > 0) {
            changeData.add.forEach((a) => {
                modifySql += `ALTER TABLE ${dbTable} ADD ${this.genColumnBasicSql(a)};`;
                if (a.remark) {
                    commentSql += `COMMENT ON COLUMN ${dbTable}.${this.quoteIdentifier(a.name)} IS '${QuoteEscape(a.remark)}';`;
                }
            });
        }

        if (changeData.upd.length > 0) {
            changeData.upd.forEach((a) => {
                let cmtSql = `COMMENT ON COLUMN ${dbTable}.${this.quoteIdentifier(a.name)} IS '${QuoteEscape(a.remark)}';`;
                if (a.remark && a.oldName === a.name) {
                    commentSql += cmtSql;
                }
                // 修改了字段名
                if (a.oldName !== a.name) {
                    renameSql += `ALTER TABLE ${dbTable} RENAME COLUMN ${this.quoteIdentifier(a.oldName!)} TO ${this.quoteIdentifier(a.name)};`;
                    if (a.remark) {
                        commentSql += cmtSql;
                    }
                }
                let typeLength = this.getTypeLengthSql(a);
                // 如果有原名以原名为准
                let name = a.oldName && a.name !== a.oldName ? a.oldName : a.name;
                modifySql += `ALTER TABLE ${dbTable} ALTER COLUMN ${this.quoteIdentifier(name)} TYPE ${a.type}${typeLength};`;
                let defaultSql = this.getDefaultValueSql(a);
                if (defaultSql) {
                    modifySql += `ALTER TABLE ${dbTable} ALTER COLUMN ${this.quoteIdentifier(name)} SET${defaultSql};`;
                }
            });
        }

        if (changeData.del.length > 0) {
            changeData.del.forEach((a) => {
                dropSql += `ALTER TABLE ${dbTable} DROP COLUMN ${this.quoteIdentifier(a.name)};`;
            });
        }
        return modifySql + dropSql + renameSql + commentSql;
    }

    getModifyIndexSql(tableData: TableEditContext, tableName: string, changeData: ChangeDiff<IndexDefinition>): string {
        let dbTable = buildSchemaTable(this.quoteIdentifier, tableData.db, tableName);
        let schema = extractSchema(tableData.db);

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
            let sql: string[] = [];
            if (dropIndexNames.length > 0) {
                dropIndexNames.forEach((a) => {
                    const idxRef = schema ? `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(a)}` : this.quoteIdentifier(a);
                    sql.push(`DROP INDEX ${idxRef}`);
                });
            }

            if (addIndexs.length > 0) {
                addIndexs.forEach((a) => {
                    let colArr = a.columnNames.map((c: string) => this.quoteIdentifier(c));
                    sql.push(`CREATE ${a.unique ? 'UNIQUE ' : ''}INDEX ${this.quoteIdentifier(a.indexName)} ON ${dbTable} (${colArr.join(',')})`);
                    if (a.indexComment) {
                        const idxRef = schema ? `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(a.indexName)}` : this.quoteIdentifier(a.indexName);
                        sql.push(`COMMENT ON INDEX ${idxRef} IS '${QuoteEscape(a.indexComment)}'`);
                    }
                });
            }
            return sql.join(';');
        }
        return '';
    }

    getModifyTableInfoSql(tableData: TableInfoEditContext): string {
        let schema = extractSchema(tableData.db);

        let sql = '';
        if (tableData.tableComment != tableData.oldTableComment) {
            let dbTable = `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(tableData.oldTableName)}`;
            sql = `COMMENT ON TABLE ${dbTable} IS '${QuoteEscape(tableData.tableComment)}';`;
        }
        if (tableData.tableName != tableData.oldTableName) {
            let dbTable = `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(tableData.oldTableName)}`;
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

    getBatchInsertPreviewSql(tableName: string, fieldArr: string[], duplicateStrategy: DuplicateStrategy): string {
        // 构建占位符字符串 "($1, $2, $3 ...)"
        let placeholder = fieldArr.map((_, i) => `$${i + 1}`).join(',');
        let suffix = '';
        if (duplicateStrategy === DuplicateStrategy.IGNORE) {
            suffix = ' ON CONFLICT DO NOTHING';
        } else if (duplicateStrategy === DuplicateStrategy.REPLACE) {
            suffix = ' ON CONFLICT ON CONSTRAINT {your_constraint_name1} DO UPDATE SET ' + fieldArr.map((a) => `${a}=excluded.${a}`).join(',');
        }

        return `INSERT INTO ${tableName} (${fieldArr.join(',')}) VALUES (${placeholder}) \n ${suffix};`;
    }
}

registerDbDialect(DbType.postgresql, new PostgresqlDialect());
