import { DbInst } from '../db';
import {
    commonCustomKeywords,
    DataType,
    DbDialect,
    DialectInfo,
    EditorCompletion,
    EditorCompletionItem,
    IndexDefinition,
    RowDefinition,
    sqlColumnType,
} from './index';
import { language as pgsqlLanguage } from 'monaco-editor/esm/vs/basic-languages/pgsql/pgsql.js';

export { PostgresqlDialect, GAUSS_TYPE_LIST };

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

class PostgresqlDialect implements DbDialect {
    getInfo(): DialectInfo {
        if (pgDialectInfo) {
            return pgDialectInfo;
        }

        let { keywords, operators, builtinVariables, builtinFunctions } = pgsqlLanguage;
        let replaceFunctionNames = replaceFunctions.map((a) => a.label);
        let functions = builtinFunctions
            .filter((a: string) => replaceFunctionNames.indexOf(a) < 0)
            .map((a: string): EditorCompletionItem => ({ label: a, insertText: `${a}()`, description: 'func' }))
            .concat(replaceFunctions);
        let excludeKeywords = new Set(builtinFunctions.concat(replaceFunctionNames).concat(operators));

        let editorCompletions: EditorCompletion = {
            keywords: keywords
                .filter((a: string) => !excludeKeywords.has(a)) // 移除已存在的operator、function
                .map((a: string): EditorCompletionItem => ({ label: a, description: 'keyword' }))
                .concat(commonCustomKeywords.map((a): EditorCompletionItem => ({ label: a, description: 'keyword' }))),
            operators: operators.map((a: string): EditorCompletionItem => ({ label: a, description: 'operator' })),
            functions,
            variables: builtinVariables.map((a: string): EditorCompletionItem => ({ label: a, description: 'var' })),
        };

        pgDialectInfo = {
            name: 'PostgreSQL',
            icon: 'iconfont icon-op-postgres',
            defaultPort: 5432,
            formatSqlDialect: 'postgresql',
            columnTypes: GAUSS_TYPE_LIST.sort((a, b) => a.udtName.localeCompare(b.udtName)),
            editorCompletions,
        };
        return pgDialectInfo;
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

    getDefaultRows(): RowDefinition[] {
        return [
            { name: 'id', type: 'bigserial', length: '', numScale: '', value: '', notNull: true, pri: true, auto_increment: true, remark: '主键ID' },
            { name: 'creator_id', type: 'int8', length: '', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '创建人id' },
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
                type: 'timestamp',
                length: '',
                numScale: '',
                value: 'CURRENT_TIMESTAMP',
                notNull: true,
                pri: false,
                auto_increment: false,
                remark: '创建时间',
            },
            { name: 'updator_id', type: 'int8', length: '', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '修改人id' },
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
                type: 'timestamp',
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
            indexType: 'BTREE',
            indexComment: '',
        };
    }

    quoteIdentifier = (name: string) => {
        // 后端sql解析器暂不支持pgsql
        return name;
    };

    matchType(text: string, arr: string[]): boolean {
        if (!text || !arr || arr.length === 0) {
            return false;
        }
        for (let i = 0; i < arr.length; i++) {
            if (text.indexOf(arr[i]) > -1) {
                return true;
            }
        }
        return false;
    }

    getDefaultValueSql(cl: any): string {
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

    getTypeLengthSql(cl: any) {
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

    genColumnBasicSql(cl: any): string {
        let length = this.getTypeLengthSql(cl);
        // 默认值
        let defVal = this.getDefaultValueSql(cl);
        // 如果有原名以原名为准
        let name = cl.oldName && cl.name !== cl.oldName ? cl.oldName : cl.name;

        return ` ${this.quoteIdentifier(name)} ${cl.type}${length} ${cl.notNull ? 'NOT NULL' : ''} ${defVal} `;
    }

    getCreateTableSql(data: any): string {
        let createSql = '';
        let tableCommentSql = '';
        let columCommentSql = '';

        // 创建表结构
        let pks = [] as string[];
        let fields: string[] = [];
        data.fields.res.forEach((item: any) => {
            item.name && fields.push(this.genColumnBasicSql(item));
            if (item.pri) {
                pks.push(item.name);
            }
            // 列注释
            if (item.remark) {
                columCommentSql += ` comment on column ${data.tableName}.${item.name} is '${item.remark}'; `;
            }
        });
        // 建表
        createSql = `CREATE TABLE ${data.tableName}
                     (
                         ${fields.join(',')}
                             ${pks ? `, PRIMARY KEY (${pks.join(',')})` : ''}
                     );`;
        // 表注释
        if (data.tableComment) {
            tableCommentSql = ` comment on table ${data.tableName} is '${data.tableComment}'; `;
        }

        return createSql + tableCommentSql + columCommentSql;
    }

    getCreateIndexSql(tableData: any): string {
        // CREATE UNIQUE INDEX idx_column_name ON your_table (column1, column2);
        // COMMENT ON INDEX idx_column_name IS 'Your index comment here';
        // 创建索引
        let schema = tableData.db.split('/')[1];
        let dbTable = `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(tableData.tableName)}`;
        let sql: string[] = [];
        tableData.indexs.res.forEach((a: any) => {
            // 字段名用双引号包裹
            let colArr = a.columnNames.map((a: string) => `${this.quoteIdentifier(a)}`);
            sql.push(`CREATE ${a.unique ? 'UNIQUE' : ''} INDEX ${this.quoteIdentifier(a.indexName)} on ${dbTable} (${colArr.join(',')})`);
            if (a.indexComment) {
                sql.push(`COMMENT ON INDEX ${schema}.${this.quoteIdentifier(a.indexName)} IS '${a.indexComment}'`);
            }
        });
        return sql.join(';');
    }

    getModifyColumnSql(tableData: any, tableName: string, changeData: { del: RowDefinition[]; add: RowDefinition[]; upd: RowDefinition[] }): string {
        let schemaArr = tableData.db.split('/');
        let schema = schemaArr.length > 1 ? schemaArr[schemaArr.length - 1] : schemaArr[0];
        let dbTable = `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(tableName)}`;

        let dropPkSql = '';
        let modifySql = '';
        let dropSql = '';
        let renameSql = '';
        let addPkSql = '';
        let commentSql = '';

        if (changeData.add.length > 0) {
            changeData.add.forEach((a) => {
                modifySql += `alter table ${dbTable} add ${this.genColumnBasicSql(a)};`;
                if (a.remark) {
                    commentSql += `comment on column ${dbTable}.${this.quoteIdentifier(a.name)} is '${a.remark}';`;
                }
            });
        }

        if (changeData.upd.length > 0) {
            changeData.upd.forEach((a) => {
                let cmtSql = `comment on column ${dbTable}.${this.quoteIdentifier(a.name)} is '${a.remark}';`;
                if (a.remark && a.oldName === a.name) {
                    commentSql += cmtSql;
                }
                // 修改了字段名
                if (a.oldName !== a.name) {
                    renameSql += `alter table ${dbTable} rename column ${this.quoteIdentifier(a.oldName!)} to ${this.quoteIdentifier(a.name)};`;
                    if (a.remark) {
                        commentSql += cmtSql;
                    }
                }
                let typeLength = this.getTypeLengthSql(a);
                // 如果有原名以原名为准
                let name = a.oldName && a.name !== a.oldName ? a.oldName : a.name;
                modifySql += `alter table ${dbTable} alter column ${this.quoteIdentifier(name)} type ${a.type}${typeLength} ;`;
                let defaultSql = this.getDefaultValueSql(a);
                if (defaultSql) {
                    modifySql += `alter table ${dbTable} alter column ${this.quoteIdentifier(name)} set ${defaultSql} ;`;
                }
            });
        }

        if (changeData.del.length > 0) {
            changeData.del.forEach((a) => {
                dropSql += `alter table ${dbTable} drop column ${a.name};`;
            });
        }
        return dropPkSql + modifySql + dropSql + renameSql + addPkSql + commentSql;
    }

    getModifyIndexSql(tableData: any, tableName: string, changeData: { del: any[]; add: any[]; upd: any[] }): string {
        let schema = tableData.db.split('/')[1];
        let dbTable = `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(tableName)}`;

        // 不能直接修改索引名或字段、需要先删后加
        let dropIndexNames: string[] = [];
        let addIndexs: any[] = [];

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
                    sql.push(`DROP INDEX ${a}`);
                });
            }

            if (addIndexs.length > 0) {
                addIndexs.forEach((a) => {
                    // 字段名用双引号包裹
                    let colArr = a.columnNames.map((a: string) => `${this.quoteIdentifier(a)}`);
                    sql.push(`CREATE ${a.unique ? 'UNIQUE' : ''} INDEX ${this.quoteIdentifier(a.indexName)} on ${dbTable} (${colArr.join(',')})`);
                    if (a.indexComment) {
                        sql.push(`COMMENT ON INDEX ${schema}.${this.quoteIdentifier(a.indexName)} IS '${a.indexComment}'`);
                    }
                });
            }
            return sql.join(';');
        }
        return '';
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
