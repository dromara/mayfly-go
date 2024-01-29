import { DbInst } from '../db';
import { commonCustomKeywords, DataType, DbDialect, DialectInfo, EditorCompletion, EditorCompletionItem, IndexDefinition, RowDefinition } from './index';
import { language as mysqlLanguage } from 'monaco-editor/esm/vs/basic-languages/mysql/mysql.js';

export { MYSQL_TYPE_LIST, MysqlDialect };

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

class MysqlDialect implements DbDialect {
    getInfo(): DialectInfo {
        if (mysqlDialectInfo) {
            return mysqlDialectInfo;
        }

        let { keywords, operators, builtinVariables, builtinFunctions } = mysqlLanguage;
        let replaceFunctionNames = replaceFunctions.map((a) => a.label);
        let functions = builtinFunctions
            .filter((a: string) => replaceFunctionNames.indexOf(a) < 0) // 删除重写的函数
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

        mysqlDialectInfo = {
            name: 'MySQL',
            icon: 'iconfont icon-op-mysql',
            defaultPort: 3306,
            formatSqlDialect: 'mysql',
            columnTypes: MYSQL_TYPE_LIST.map((a) => ({ udtName: a, dataType: a, desc: '', space: '' })),
            editorCompletions,
        };
        return mysqlDialectInfo;
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

    getDefaultRows(): RowDefinition[] {
        return [
            { name: 'id', type: 'bigint', length: '20', numScale: '', value: '', notNull: true, pri: true, auto_increment: true, remark: '主键ID' },
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
                type: 'datetime',
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
                type: 'datetime',
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
        return `\`${name}\``;
    };

    genColumnBasicSql(cl: any): string {
        let val = cl.value ? (cl.value === 'CURRENT_TIMESTAMP' ? cl.value : `'${cl.value}'`) : '';
        let defVal = val ? `DEFAULT ${val}` : '';
        let length = cl.length ? `(${cl.length})` : '';
        let onUpdate = 'update_time' === cl.name ? ' ON UPDATE CURRENT_TIMESTAMP ' : '';
        return ` ${this.quoteIdentifier(cl.name)} ${cl.type}${length} ${cl.notNull ? 'NOT NULL' : 'NULL'} ${
            cl.auto_increment ? 'AUTO_INCREMENT' : ''
        } ${defVal} ${onUpdate} comment '${cl.remark || ''}' `;
    }
    getCreateTableSql(data: any): string {
        // 创建表结构
        let pks = [] as string[];
        let fields: string[] = [];
        data.fields.res.forEach((item: any) => {
            item.name && fields.push(this.genColumnBasicSql(item));
            if (item.pri) {
                pks.push(item.name);
            }
        });

        return `CREATE TABLE ${data.tableName}
                  ( ${fields.join(',')}
                      ${pks ? `, PRIMARY KEY (${pks.join(',')})` : ''}
                  ) COMMENT='${data.tableComment}';`;
    }

    getCreateIndexSql(data: any): string {
        // 创建索引
        let sql = `ALTER TABLE ${data.tableName}`;
        data.indexs.res.forEach((a: any) => {
            sql += ` ADD ${a.unique ? 'UNIQUE' : ''} INDEX ${a.indexName}(${a.columnNames.join(',')}) USING ${a.indexType} COMMENT '${a.indexComment}',`;
        });
        return sql.substring(0, sql.length - 1) + ';';
    }

    getModifyColumnSql(tableData: any, tableName: string, changeData: { del: RowDefinition[]; add: RowDefinition[]; upd: RowDefinition[] }): string {
        let sql = `ALTER TABLE ${this.quoteIdentifier(tableData.db)}.${this.quoteIdentifier(tableName)}`;
        let arr = [] as string[];
        if (changeData.del.length > 0) {
            changeData.del.forEach((a) => {
                arr.push(` DROP COLUMN  ${this.quoteIdentifier(a.name)} `);
            });
        }
        if (changeData.add.length > 0) {
            changeData.add.forEach((a) => {
                arr.push(` ADD COLUMN ${this.genColumnBasicSql(a)} `);
            });
        }

        if (changeData.upd.length > 0) {
            changeData.upd.forEach((a) => {
                if (a.name === a.oldName) {
                    arr.push(` MODIFY COLUMN ${this.genColumnBasicSql(a)} `);
                } else {
                    arr.push(` CHANGE COLUMN ${this.quoteIdentifier(a.oldName!)} ${this.genColumnBasicSql(a)} `);
                }
            });
        }

        return sql + arr.join(',') + ';';
    }

    getModifyIndexSql(tableData: any, tableName: string, changeData: { del: any[]; add: any[]; upd: any[] }): string {
        // 搜集修改和删除的索引，添加到drop index xx
        // 收集新增和修改的索引，添加到ADD xx
        // ALTER TABLE `test1`
        // DROP INDEX `test1_name_uindex`,
        // DROP INDEX `test1_column_name4_index`,
        // ADD UNIQUE INDEX `test1_name_uindex`(`id`) USING BTREE COMMENT 'ASDASD',
        // ADD INDEX `111`(`column_name4`) USING BTREE COMMENT 'zasf';

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
            let sql = `ALTER TABLE ${tableName} `;
            if (dropIndexNames.length > 0) {
                dropIndexNames.forEach((a) => {
                    sql += `DROP INDEX ${a},`;
                });
                sql = sql.substring(0, sql.length - 1);
            }

            if (addIndexs.length > 0) {
                if (dropIndexNames.length > 0) {
                    sql += ',';
                }
                addIndexs.forEach((a) => {
                    sql += ` ADD ${a.unique ? 'UNIQUE' : ''} INDEX ${a.indexName}(${a.columnNames.join(',')}) USING ${a.indexType} COMMENT '${
                        a.indexComment
                    }',`;
                });
                sql = sql.substring(0, sql.length - 1);
            }
            return sql;
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
