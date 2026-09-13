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
import { buildSchemaTable, extractSchema, matchType as _matchType, matchNumericType, QuoteEscape, wrapValueOracle } from './shared/utils';
import { defineCapabilities, plsqlSplitOptions } from './shared/capabilities';
import { rownumPageSnippet } from './shared/snippets';
import { DbType } from './dbType';
import { registerDbDialect } from './registry';
import type { TableEditContext, TableInfoEditContext, ChangeDiff } from './types';

// 参考官方文档：https://docs.oracle.com/cd/B19306_01/server.102/b14200/sql_elements001.htm
const ORACLE_TYPE_LIST: sqlColumnType[] = [
    // 字符数据类型
    { udtName: 'CHAR', dataType: 'CHAR', desc: '定长字符串,自动在末尾用空格补全,非unicode', space: '', range: '1 - 2000' },
    { udtName: 'NCHAR', dataType: 'NCHAR', desc: '定长字符串,自动在末尾用空格补全,unicode', space: '', range: '1 - 1000' },
    { udtName: 'VARCHAR2', dataType: 'VARCHAR2', desc: '变长字符串,不自动补全空格,非unicode', space: '', range: '1 - 4000' },
    { udtName: 'NVARCHAR2', dataType: 'NVARCHAR2', desc: '变长字符串,不自动补全空格,unicode', space: '', range: '1 - 2000' },

    // 精确数值数据类型 NUMERIC、DECIMAL、DEC 类型、NUMBER 类型、INTEGER 类型、INT 类型、BIGINT 类型、TINYINT 类型、BYTE 类型、SMALLINT
    { udtName: 'NUMBER', dataType: 'NUMBER', desc: 'NUMBER(p,s)', space: '1-38', range: '' },
    { udtName: 'INTEGER', dataType: 'INTEGER', desc: '同于number(38)', space: '', range: '' },
    { udtName: 'INT', dataType: 'INT', desc: '同INTEGER', space: '10', range: '' },
    { udtName: 'SMALLINT', dataType: 'SMALLINT', desc: '同于number(38)', space: '', range: '' },
    { udtName: 'DECIMAL', dataType: 'DECIMAL', desc: 'decimal(p,s) 默认number(38)', space: '', range: '' },
    { udtName: 'FLOAT', dataType: 'FLOAT', desc: 'float(b二进制进度),b的取值范围[1,126]，默认126', space: '', range: '' },
    { udtName: 'REAL', dataType: 'REAL', desc: '同FLOAT(63)', space: '', range: '' },
    { udtName: 'BINARY_FLOAT', dataType: 'BINARY_FLOAT', desc: '32位单精度浮点数数据类型', space: '', range: '' },
    { udtName: 'BINARY_DOUBLE', dataType: 'BINARY_DOUBLE', desc: '64位双精度浮点数数据类型', space: '', range: '' },

    // 一般日期时间数据类型 DATE TIME TIMESTAMP 默认精度 6
    // 多媒体数据类型 TEXT/LONG/LONGVARCHAR 类型：变长字符串类型  IMAGE/LONGVARBINARY 类型  BLOB CLOB BFILE  100G-1
    { udtName: 'DATE', dataType: 'DATE', desc: '世纪，年，月，日，时，分，秒', space: '', range: '' },
    { udtName: 'TIMESTAMP', dataType: 'TIMESTAMP', desc: '', space: '', range: '' },
    // { udtName: 'timestamp(precision) with time zone', dataType: 'TIMESTAMP', desc: '在timestamp(precison)的基础上加入了时区偏移量的值', space: '', range: '' },
    // { udtName: 'timestamp with local time zone', dataType: 'TIMESTAMP', desc: '存储时转化为数据库时区进行规范化存储，但不存储时区信息，客户端检索时，按客户端时区的时间数据返回给客户端', space: '', range: '' },
    // { udtName: 'interval year(precision) to month', dataType: 'interval year(precision) to month', desc: '可以用来表示几年几月的时间间隔', space: '', range: '' },
    // { udtName: 'nterval day(days_precision) to second(seconds_precision)', dataType: 'nterval day(days_precision) to second(seconds_precision)', desc: '可以用来存储天、小时、分和秒的时间间隔', space: '', range: '' },

    { udtName: 'LONG', dataType: 'LONG', desc: '文本类型,不能作为主键或唯一约束', space: '', range: '最多达2GB' },
    { udtName: 'LONG RAW', dataType: 'LONG RAW', desc: '可变长二进制数据，不用进行字符集转换的数据', space: '', range: '最多达2GB' },
    { udtName: 'BLOB', dataType: 'BLOB', desc: '二进制大型对象', space: '', range: '最大长度4G' },
    { udtName: 'CLOB', dataType: 'CLOB', desc: '字符大型对象', space: '', range: '最大长度4G' },
    { udtName: 'NCLOB', dataType: 'NCLOB', desc: 'Unicode类型的数据', space: '', range: '最大长度4G' },
    { udtName: 'BFILE', dataType: 'BFILE', desc: '二进制文件', space: '', range: '' },
];

// 参考官方文档：https://docs.oracle.com/cd/B19306_01/server.102/b14200/functions001.htm
const replaceFunctions: EditorCompletionItem[] = [
    //  字符函数
    { label: 'ASCII', insertText: 'ASCII(x)', description: '返回字符X的ASCII码' },
    { label: 'CONCAT', insertText: 'CONCAT(x,y)', description: '连接字符串X和Y' },
    { label: 'INSTR', insertText: 'INSTR(X,STR[,START][,N)', description: '从X中查找str，可以指定从start开始，也可以指定从n开始' },
    { label: 'LENGTH', insertText: 'LENGTH(x)', description: '返回X的长度' },
    { label: 'LOWER', insertText: 'LOWER(X)', description: 'X转换成小写' },
    { label: 'UPPER', insertText: 'UPPER(X)', description: 'X转换成大写' },
    { label: 'LTRIM', insertText: 'LTRIM(X[,TRIM_STR])', description: '把X的左边截去trim_str字符串，缺省截去空格' },
    { label: 'RTRIM', insertText: 'RTRIM(X[,TRIM_STR])', description: '把X的右边截去trim_str字符串，缺省截去空格' },
    { label: 'TRIM', insertText: 'TRIM(X[,TRIM_STR])', description: '把X的两边截去trim_str字符串，缺省截去空格' },
    { label: 'REPLACE', insertText: 'REPLACE(X,old,new)', description: '在X中查找old，并替换成new' },
    { label: 'SUBSTR', insertText: 'SUBSTR(X,start[,length])', description: '返回X的字串，从start处开始，截取length个字符，缺省length，默认到结尾' },
    // 数值函数
    { label: 'ABS', insertText: 'ABS(X)', description: 'X的绝对值' },
    { label: 'ACOS', insertText: 'ACOS(X)', description: 'X的反余弦' },
    { label: 'COS', insertText: 'COS(X)', description: '余弦' },
    { label: 'CEIL', insertText: 'CEIL(X)', description: '大于或等于X的最小值' },
    { label: 'FLOOR', insertText: 'FLOOR(X)', description: '小于或等于X的最大值' },
    { label: 'LOG', insertText: 'LOG(X,Y)', description: 'X为底Y的对数' },
    { label: 'MOD', insertText: 'MOD(X,Y)', description: 'X除以Y的余数' },
    { label: 'POWER', insertText: 'POWER(X,Y)', description: 'X的Y次幂' },
    { label: 'ROUND', insertText: 'ROUND(X [,Y]})', description: 'X在第Y位四舍五入' },
    { label: 'SQRT', insertText: 'SQRT(n)', description: '求数值 n 的平方根' },
    { label: 'TRUNC', insertText: 'TRUNC(n [,m])', description: "截取数值函数，str 内只能为数字和'-', '+', '.' 的组合" },
    //日期时间函数
    { label: 'ADD_MONTHS', insertText: 'ADD_MONTHS(date,n)', description: '在输入日期上加上指定的几个月返回一个新日期' },
    { label: 'LAST_DAY', insertText: 'LAST_DAY(date)', description: '返回输入日期所在月份最后一天的日期' },
    { label: 'EXTRACT', insertText: 'EXTRACT(fmt FROM d)', description: '提取日期中的特定部分' },
    { label: 'CURRENT_DATE', insertText: 'CURRENT_DATE', description: '获取当前日期' },
    { label: 'CURRENT_TIMESTAMP', insertText: 'TIMESTAMP', description: '获取当前时间' },
    // 转换函数
    { label: 'TO_CHAR', insertText: `TO_CHAR(d|n, 'yyyy-MM-dd HH24:mi:ss')`, description: '把日期和数字转换为制定格式的字符串' },
    { label: 'TO_DATE', insertText: `TO_DATE(X, 'yyyy-MM-dd HH24:mi:ss')`, description: '把一个字符串以fmt格式转换成一个日期类型' },
    { label: 'TO_NUMBER', insertText: `TO_NUMBER(X, 'yyyy-MM-dd HH24:mi:ss')`, description: '把一个字符串以fmt格式转换为一个数字' },
    { label: 'TO_TIMESTAMP', insertText: `TO_TIMESTAMP(X, 'yyyy-MM-dd HH24:mi:ss.ff')`, description: '把一个字符串以fmt格式转换为日期类型' },
    // 其他
    { label: 'NVL', insertText: 'NVL(X,VALUE)', description: '如果X为空，返回value，否则返回X' },
    { label: 'NVL2', insertText: 'NVL2(x,value1,value2)', description: '如果x非空，返回value1，否则返回value2' },
];

const addCustomKeywords: EditorCompletionItem[] = [
    {
        label: 'ROWNUM',
        description: 'keyword',
        insertText: 'ROWNUM',
    },
    {
        label: 'DUAL',
        description: 'keyword',
        insertText: 'DUAL',
    },
    // 分页代码块
    {
        label: 'SELECT ROWNUM',
        description: 'code block',
        insertText: 'SELECT * from table_name where rownum <= 10',
    },
    {
        label: 'SELECT PAGE',
        description: 'code block',
        insertText: ` SELECT * FROM
    (
      SELECT t.*, ROWNUM AS rn
      FROM table_name t
      WHERE ROWNUM <= 25
    )
  WHERE rn > 0 \n`,
    },
];

let oracleDialectInfo: DialectInfo;
let oracleCompletions: EditorCompletion;
export class OracleDialect implements DbDialect {
    getCapabilities(): DialectCapabilities {
        return defineCapabilities({
            defaultIndexType: 'NORMAL',
            // 以 SID / Service Name 连接，库名列表为空时回落到实例 SID
            connectDescriptor: 'sid_service',
            sqlSplitOptions: plsqlSplitOptions,
        });
    }

    getInfo(): DialectInfo {
        if (oracleDialectInfo) {
            return oracleDialectInfo;
        }

        oracleDialectInfo = {
            name: 'Oracle',
            icon: 'icon db/oracle',
            defaultPort: 1521,
            formatSqlDialect: 'plsql',
            columnTypes: ORACLE_TYPE_LIST.sort((a, b) => a.udtName.localeCompare(b.udtName)),
        };
        return oracleDialectInfo;
    }

    /** 编辑器联想词由 monaco 语言定义派生，按需加载并缓存（详见 types.ts 的 DialectInfo 注释） */
    async getEditorCompletions(): Promise<EditorCompletion> {
        if (oracleCompletions) {
            return oracleCompletions;
        }

        const { language: sqlLanguage } = await import('monaco-editor/languages/definitions/sql/sql.js');
        let { keywords, operators, builtinVariables } = sqlLanguage;
        let functionNames = replaceFunctions.map((a) => a.label);
        let excludeKeywords = new Set(functionNames.concat(operators));
        excludeKeywords.add('SELECT');

        oracleCompletions = {
            keywords: keywords
                .filter((a: string) => !excludeKeywords.has(a)) // 移除已存在的operator、function
                .map((a: string): EditorCompletionItem => ({ label: a, description: 'keyword' }))
                .concat(
                    // 加上自定义的关键字
                    commonCustomKeywords.map(
                        (a): EditorCompletionItem => ({
                            label: a,
                            description: 'keyword',
                        })
                    )
                )
                .concat(addCustomKeywords),
            operators: operators.map((a: string): EditorCompletionItem => ({ label: a, description: 'operator' })),
            functions: replaceFunctions,
            variables: builtinVariables.map((a: string): EditorCompletionItem => ({ label: a, description: 'var' })),
        };
        return oracleCompletions;
    }

    getDefaultSelectSql(db: string, table: string, condition: string, orderBy: string, pageNum: number, limit: number) {
        return `
        SELECT *
        FROM (
            SELECT t.*, ROWNUM AS rn
            FROM "${table}" t
            WHERE ROWNUM <=${pageNum * limit} ${condition ? ' and ' + condition : ''}
            ${orderBy ? orderBy : ''}
        )
        WHERE rn > ${(pageNum - 1) * limit}
        `;
    }

    getPageSql(pageNum: number, limit: number) {
        return ``;
    }

    getPreviewSql(sql: string, limit = 1): string {
        // Oracle 11g 及以前无 OFFSET/FETCH，以 ROWNUM 伪列限行
        const hasCondition = /where/i.test(sql);
        return `${sql} ${hasCondition ? 'AND' : 'WHERE'} ROWNUM <= ${limit}`;
    }

    getPageSnippet(): SqlSnippetTemplate {
        return rownumPageSnippet;
    }

    getDefaultRows(): RowDefinition[] {
        return createDefaultRows(defaultRowsConfigs.oracle);
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
            if (this.matchType(cl.type, ['CHAR', 'TIME', 'DATE', 'LONG', 'CLOB', 'BLOB', 'BFILE'])) {
                // 默认值是时间日期函数的必须要加引号
                let val = cl.value.toUpperCase().replace(' ', '');
                if (this.matchType(cl.type, ['DATE', 'TIMESTAMP']) && ['CURRENT_DATE', 'CURRENT_TIMESTAMP'].includes(val)) {
                    marks = false;
                } else {
                    marks = true;
                }
            }
            return ` DEFAULT ${marks ? "'" : ''}${cl.value}${marks ? "'" : ''}`;
        }
        return '';
    }

    getTypeLengthSql(cl: RowDefinition) {
        // 哪些字段可以指定长度  VARCHAR/VARCHAR2/CHAR/BIT/NUMBER/NUMERIC/TIME、TIMESTAMP(可以指定小数秒精度)
        if (cl.length && this.matchType(cl.type, ['CHAR', 'BIT', 'TIME', 'NUM', 'DEC'])) {
            // 哪些字段类型可以指定小数点
            if (cl.numScale && this.matchType(cl.type, ['NUM', 'DEC'])) {
                return `(${cl.length}, ${cl.numScale})`;
            } else {
                return `(${cl.length})`;
            }
        }
        return '';
    }

    genColumnBasicSql(cl: RowDefinition, create: boolean, data = {}): string {
        let length = this.getTypeLengthSql(cl);
        // 默认值
        let defVal = this.getDefaultValueSql(cl);
        let incr = '';
        if (cl.auto_increment && create) {
            cl.type = 'number';
            length = '';
            incr = 'generated by default as IDENTITY';
        }
        // 如果有原名以原名为准
        let name = cl.oldName && cl.name !== cl.oldName ? cl.oldName : cl.name;
        let baseSql = ` ${this.quoteIdentifier(name)} ${cl.type}${length} ${incr}`;
        return incr ? baseSql : ` ${baseSql} ${defVal} ${cl.notNull ? 'NOT NULL' : ''} `;
    }

    getOtherCreateTableSql(_data: TableEditContext) {
        return '';
    }

    getCreateTableSql(data: TableEditContext): string {
        let dbTable = buildSchemaTable(this.quoteIdentifier, data.db, data.tableName);

        let createSql = '';
        let tableCommentSql = '';
        let columCommentSql = '';
        let pris = [] as string[];

        // 创建表结构
        let fields: string[] = [];
        data.fields.res.forEach((item: RowDefinition) => {
            item.name && fields.push(this.genColumnBasicSql(item, true, data));
            // 列注释
            if (item.remark) {
                columCommentSql += `COMMENT ON COLUMN ${dbTable}.${this.quoteIdentifier(item.name)} IS '${QuoteEscape(item.remark)}';`;
            }
            // 主键
            if (item.pri) {
                pris.push(this.quoteIdentifier(item.name));
            }
        });
        // 主键语句
        let prisql = '';
        if (pris.length > 0) {
            prisql = `PRIMARY KEY (${pris.join(',')})`;
        }
        // 建表
        const fieldStr = fields.join(',');
        createSql = `CREATE TABLE ${dbTable} (${fieldStr}${prisql ? ', ' + prisql : ''});`;
        // 表注释
        if (data.tableComment) {
            tableCommentSql = `COMMENT ON TABLE ${dbTable} IS '${QuoteEscape(data.tableComment)}';`;
        }

        // 其余建表信息，如：自增字段在老版本的使用方式是创建自增序列
        let other = this.getOtherCreateTableSql(data);
        return createSql + tableCommentSql + columCommentSql + other;
    }

    getCreateIndexSql(tableData: TableEditContext): string {
        let dbTable = buildSchemaTable(this.quoteIdentifier, tableData.db, tableData.tableName);

        let sql: string[] = [];
        tableData.indexs.res.forEach((a: IndexDefinition) => {
            const cols = a.columnNames.map((c) => this.quoteIdentifier(c)).join(',');
            sql.push(`CREATE ${a.unique ? 'UNIQUE ' : ''}INDEX ${this.quoteIdentifier(a.indexName)} ON ${dbTable} (${cols})`);
        });
        return sql.join(';');
    }

    getDropTableSql(db: string, table: string): string {
        // 必须带 schema（Oracle 中即 owner）限定，否则会解析到当前登录用户的同名对象
        return `DROP TABLE ${buildSchemaTable(this.quoteIdentifier, db, table)}`;
    }

    getModifyColumnSql(tableData: TableEditContext, tableName: string, changeData: ChangeDiff<RowDefinition>): string {
        let dbTable = buildSchemaTable(this.quoteIdentifier, tableData.db, tableName);

        let baseSql = `ALTER TABLE ${dbTable}`;

        let modifyArr: string[] = [];
        let dropArr: string[] = [];
        let renameArr: string[] = [];
        let commentArr: string[] = [];

        // 主键字段
        let priArr = new Set();

        if (changeData.upd.length > 0) {
            changeData.upd.forEach((a) => {
                let commentSql = `COMMENT ON COLUMN ${dbTable}.${this.quoteIdentifier(a.name)} IS '${QuoteEscape(a.remark)}'`;
                if (a.remark && a.oldName === a.name) {
                    commentArr.push(commentSql);
                }
                // 修改了字段名
                if (a.oldName !== a.name) {
                    renameArr.push(`${baseSql} RENAME COLUMN ${this.quoteIdentifier(a.oldName!)} TO ${this.quoteIdentifier(a.name)}`);
                    if (a.remark) {
                        commentArr.push(commentSql);
                    }
                }
                modifyArr.push(`${baseSql} MODIFY (${this.genColumnBasicSql(a, false, tableData)})`);
                if (a.pri) {
                    priArr.add(this.quoteIdentifier(a.name));
                }
            });
        }

        if (changeData.add.length > 0) {
            changeData.add.forEach((a) => {
                modifyArr.push(`${baseSql} ADD (${this.genColumnBasicSql(a, false, tableData)})`);
                if (a.remark) {
                    commentArr.push(`COMMENT ON COLUMN ${dbTable}.${this.quoteIdentifier(a.name)} IS '${QuoteEscape(a.remark)}'`);
                }
                if (a.pri) {
                    priArr.add(this.quoteIdentifier(a.name));
                }
            });
        }

        if (changeData.del.length > 0) {
            changeData.del.forEach((a) => {
                dropArr.push(this.quoteIdentifier(a.name));
            });
        }

        let dropPkSql = '';
        if (priArr.size > 0) {
            let resPri = tableData.fields.res.find((a: RowDefinition) => a.pri);
            if (resPri) {
                priArr.add(this.quoteIdentifier(resPri.name));
            }
            if (tableData.fields.oldFields.find((a: RowDefinition) => a.pri)) {
                dropPkSql = `ALTER TABLE ${dbTable} DROP PRIMARY KEY;`;
            }
        }

        let modifySql = modifyArr.length > 0 ? modifyArr.join(';\n') + ';' : '';
        let dropSql = dropArr.length > 0 ? `${baseSql} DROP (${dropArr.join(',')});` : '';
        let renameSql = renameArr.join(';\n') + (renameArr.length > 0 ? ';' : '');
        let addPkSql = priArr.size > 0 ? `ALTER TABLE ${dbTable} ADD CONSTRAINT "PK_${tableName}" PRIMARY KEY (${Array.from(priArr).join(',')});` : '';
        let commentSql = commentArr.join(';\n') + (commentArr.length > 0 ? ';' : '');

        return dropPkSql + modifySql + dropSql + renameSql + addPkSql + commentSql;
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
            let sql: string[] = [];
            if (dropIndexNames.length > 0) {
                dropIndexNames.forEach((a) => {
                    sql.push(`DROP INDEX ${this.quoteIdentifier(a)}`);
                });
            }

            if (addIndexs.length > 0) {
                addIndexs.forEach((a) => {
                    const cols = a.columnNames.map((c) => this.quoteIdentifier(c)).join(',');
                    sql.push(`CREATE ${a.unique ? 'UNIQUE ' : ''}INDEX ${this.quoteIdentifier(a.indexName)} ON ${this.quoteIdentifier(tableName)} (${cols})`);
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
        if (matchNumericType(columnType)) {
            return DataType.Number;
        }
        // 日期时间类型 oracle只有date和timestamp类型
        if (/timestamp|date/gi.test(columnType)) {
            return DataType.DateTime;
        }
        return DataType.String;
    }

    wrapValue(columnType: string, value: unknown): string | number {
        return wrapValueOracle(columnType, value) as string | number;
    }

    getBatchInsertPreviewSql(tableName: string, fieldArr: string[], duplicateStrategy: DuplicateStrategy): string {
        if (duplicateStrategy == DuplicateStrategy.REPLACE) {
            // 字段数组生成占位符sql
            let phs = [];
            let values = [];
            let insertFields = [];
            for (let i = 0; i < fieldArr.length; i++) {
                phs.push(`:${i + 1} ${fieldArr[i]}`);
                values.push(`T2.${fieldArr[i]}`);
                insertFields.push(`T1.${fieldArr[i]}`);
            }
            let placeholder = phs.join(',');
            let sql = `MERGE INTO ${this.quoteIdentifier(tableName)} T1 USING 
        (
         SELECT ${placeholder} FROM dual
        ) T2 ON (T1.id = T2.id) 
        WHEN NOT MATCHED THEN INSERT (${insertFields.join(',')}) VALUES (${values.join(',')})
        WHEN MATCHED THEN UPDATE SET ${fieldArr.map((a) => `T1.${a} = T2.${a}`).join(',')}`;
            return sql;
        } else {
            // 字段数组生成占位符sql
            let phs = [];
            for (let i = 0; i < fieldArr.length; i++) {
                phs.push(`:${i + 1} ${fieldArr[i]}`);
            }
            let ignore = '';
            if (duplicateStrategy == DuplicateStrategy.IGNORE) {
                ignore = `/*+ IGNORE_ROW_ON_DUPKEY_INDEX(${tableName}(id)) */`;
            }
            let placeholder = phs.join(',');
            return `INSERT ${ignore} INTO ${tableName} (${fieldArr.join(',')}) VALUES (${placeholder});`;
        }
    }
}

registerDbDialect(DbType.oracle, new OracleDialect());
