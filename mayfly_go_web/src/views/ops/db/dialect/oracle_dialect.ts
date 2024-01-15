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
import { language as sqlLanguage } from 'monaco-editor/esm/vs/basic-languages/sql/sql.js';

export { OracleDialect, ORACLE_TYPE_LIST };

// 参考文档:https://eco.dameng.com/document/dm/zh-cn/sql-dev/dmpl-sql-datatype.html#%E5%AD%97%E7%AC%A6%E6%95%B0%E6%8D%AE%E7%B1%BB%E5%9E%8B
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
    { label: 'TO_CHAR', insertText: 'TO_CHAR(d|n[,fmt])', description: '把日期和数字转换为制定格式的字符串' },
    { label: 'TO_DATE', insertText: 'TO_DATE(X,[,fmt])', description: '把一个字符串以fmt格式转换成一个日期类型' },
    { label: 'TO_NUMBER', insertText: 'TO_NUMBER(X,[,fmt])', description: '把一个字符串以fmt格式转换为一个数字' },
    { label: 'TO_TIMESTAMP', insertText: 'TO_TIMESTAMP(X,[,fmt])', description: '把一个字符串以fmt格式转换为日期类型' },
    // 其他
    { label: 'NVL', insertText: 'NVL(X,VALUE)', description: '如果X为空，返回value，否则返回X' },
    { label: 'NVL2', insertText: 'NVL2(x,value1,value2)', description: '如果x非空，返回value1，否则返回value2' },
];

const addCustomKeywords = ['ROWNUM', 'DUAL'];

let oracleDialectInfo: DialectInfo;
class OracleDialect implements DbDialect {
    getInfo(): DialectInfo {
        if (oracleDialectInfo) {
            return oracleDialectInfo;
        }

        let { keywords, operators, builtinVariables } = sqlLanguage;
        let functionNames = replaceFunctions.map((a) => a.label);
        let excludeKeywords = new Set(functionNames.concat(operators));

        let editorCompletions: EditorCompletion = {
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
                .concat(
                    // 加上自定义的关键字
                    addCustomKeywords.map(
                        (a): EditorCompletionItem => ({
                            label: a,
                            description: 'keyword',
                        })
                    )
                ),
            operators: operators.map((a: string): EditorCompletionItem => ({ label: a, description: 'operator' })),
            functions: replaceFunctions,
            variables: builtinVariables.map((a: string): EditorCompletionItem => ({ label: a, description: 'var' })),
        };

        oracleDialectInfo = {
            icon: 'iconfont icon-oracle',
            defaultPort: 1521,
            formatSqlDialect: 'plsql',
            columnTypes: ORACLE_TYPE_LIST.sort((a, b) => a.udtName.localeCompare(b.udtName)),
            editorCompletions,
        };
        return oracleDialectInfo;
    }

    getDefaultSelectSql(table: string, condition: string, orderBy: string, pageNum: number, limit: number) {
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

    // eslint-disable-next-line @typescript-eslint/no-unused-vars,no-unused-vars
    getPageSql(pageNum: number, limit: number) {
        return ``;
    }

    getDefaultRows(): RowDefinition[] {
        return [
            { name: 'ID', type: 'NUMBER', length: '', numScale: '', value: '', notNull: true, pri: true, auto_increment: true, remark: '主键ID' },
            { name: 'CREATOR_ID', type: 'NUMBER', length: '', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '创建人id' },
            {
                name: 'CREATOR',
                type: 'VARCHAR2',
                length: '100',
                numScale: '',
                value: '',
                notNull: true,
                pri: false,
                auto_increment: false,
                remark: '创建人姓名',
            },
            {
                name: 'CREATE_TIME',
                type: 'DATE',
                length: '',
                numScale: '',
                value: 'CURRENT_TIMESTAMP',
                notNull: true,
                pri: false,
                auto_increment: false,
                remark: '创建时间',
            },
            { name: 'UPDATOR_ID', type: 'NUMBER', length: '', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '修改人id' },
            {
                name: 'UPDATOR',
                type: 'VARCHAR2',
                length: '100',
                numScale: '',
                value: '',
                notNull: true,
                pri: false,
                auto_increment: false,
                remark: '修改人姓名',
            },
            {
                name: 'UPDATE_TIME',
                type: 'DATE',
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
            indexType: 'NORMAL',
            indexComment: '',
        };
    }

    quoteIdentifier = (name: string) => {
        return `"${name}"`;
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

    getTypeLengthSql(cl: any) {
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

    genColumnBasicSql(cl: RowDefinition): string {
        let length = this.getTypeLengthSql(cl);
        // 默认值
        let defVal = this.getDefaultValueSql(cl);
        let incr = cl.auto_increment ? 'generated by default as IDENTITY' : '';
        let pri = cl.pri ? 'PRIMARY KEY' : '';
        return ` ${cl.name.toUpperCase()} ${cl.type}${length} ${incr} ${pri} ${defVal} ${cl.notNull ? 'NOT NULL' : ''} `;
    }

    getCreateTableSql(data: any): string {
        let createSql = '';
        let tableCommentSql = '';
        let columCommentSql = '';

        // 创建表结构
        let fields: string[] = [];
        data.fields.res.forEach((item: any) => {
            item.name && fields.push(this.genColumnBasicSql(item));
            // 列注释
            if (item.remark) {
                columCommentSql += ` comment on column ${data.tableName?.toUpperCase()}.${item.name?.toUpperCase()} is '${item.remark}'; `;
            }
        });
        // 建表
        createSql = `CREATE TABLE ${data.tableName?.toUpperCase()} ( ${fields.join(',')} );`;
        // 表注释
        if (data.tableComment) {
            tableCommentSql = ` comment on table ${data.tableName?.toUpperCase()} is '${data.tableComment}'; `;
        }

        return createSql + tableCommentSql + columCommentSql;
    }

    getCreateIndexSql(tableData: any): string {
        // CREATE UNIQUE INDEX idx_column_name ON your_table (column1, column2);
        // COMMENT ON INDEX idx_column_name IS 'Your index comment here';
        // 创建索引
        let sql: string[] = [];
        tableData.indexs.res.forEach((a: any) => {
            sql.push(` CREATE ${a.unique ? 'UNIQUE' : ''} INDEX ${a.indexName} ON "${tableData.tableName}" ("${a.columnNames.join('","')})"`);
        });
        return sql.join(';');
    }

    getModifyColumnSql(tableName: string, changeData: { del: RowDefinition[]; add: RowDefinition[]; upd: RowDefinition[] }): string {
        let sql: string[] = [];
        if (changeData.add.length > 0) {
            changeData.add.forEach((a) => {
                sql.push(`ALTER TABLE "${tableName}" add COLUMN ${this.genColumnBasicSql(a)}`);
                if (a.remark) {
                    sql.push(`comment on COLUMN "${tableName}"."${a.name}" is '${a.remark}'`);
                }
            });
        }

        if (changeData.upd.length > 0) {
            changeData.upd.forEach((a) => {
                sql.push(`ALTER TABLE "${tableName}" MODIFY ${this.genColumnBasicSql(a)}`);
                if (a.remark) {
                    sql.push(`comment on COLUMN "${tableName}"."${a.name}" is '${a.remark}'`);
                }
            });
        }

        if (changeData.del.length > 0) {
            changeData.del.forEach((a) => {
                sql.push(`ALTER TABLE "${tableName}" DROP COLUMN ${a.name}`);
            });
        }
        return sql.join(';');
    }

    getModifyIndexSql(tableName: string, changeData: { del: any[]; add: any[]; upd: any[] }): string {
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
                    sql.push(`CREATE ${a.unique ? 'UNIQUE' : ''} INDEX ${a.indexName} ON "${tableName}" (${a.columnNames.join(',')})`);
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
        // 日期时间类型 oracle只有date和timestamp类型
        if (/timestamp|date/gi.test(columnType)) {
            return DataType.DateTime;
        }
        return DataType.String;
    }
    // eslint-disable-next-line @typescript-eslint/no-unused-vars,no-unused-vars
    wrapStrValue(columnType: string, value: string): string {
        if (value && this.getDataType(columnType) === DataType.DateTime) {
            return `to_timestamp('${value}', 'yyyy-mm-dd hh24:mi:ss')`;
        }
        return `'${value}'`;
    }
}
