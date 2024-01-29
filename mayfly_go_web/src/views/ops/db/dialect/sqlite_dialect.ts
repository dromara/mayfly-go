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
import { DbInst } from '@/views/ops/db/db';
import { language as sqlLanguage } from 'monaco-editor/esm/vs/basic-languages/sql/sql.js';

export { SqliteDialect };

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
    { label: 'sqlite_compileoption_used', insertText: 'sqlite_compileoption_used(X)', description: '检查SQLite编译时是否使用了指定的编译选项' },
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
    { label: 'time', insertText: 'time(time-value[, modifier, ...])', description: '将日期和时间字符串转换为特定的日期和时间格式' },
    { label: 'datetime', insertText: 'datetime(time-value[, modifier, ...])', description: '计算日期和时间的儒略日数' },
    { label: 'julianday', insertText: 'julianday(time-value[, modifier, ...])', description: '将日期和时间格式化为指定的字符串' },
];

let sqliteDialectInfo: DialectInfo;
class SqliteDialect implements DbDialect {
    getInfo(): DialectInfo {
        if (sqliteDialectInfo) {
            return sqliteDialectInfo;
        }

        let { keywords, operators, builtinVariables } = sqlLanguage;

        let editorCompletions: EditorCompletion = {
            keywords: keywords
                .filter((a: string) => addCustomKeywords.indexOf(a) === -1)
                .map((a: string): EditorCompletionItem => ({ label: a, description: 'keyword' }))
                .concat(commonCustomKeywords.map((a): EditorCompletionItem => ({ label: a, description: 'keyword' })))
                .concat(addCustomKeywords.map((a): EditorCompletionItem => ({ label: a, description: 'keyword' }))),
            operators: operators.map((a: string): EditorCompletionItem => ({ label: a, description: 'operator' })),
            functions,
            variables: builtinVariables.map((a: string): EditorCompletionItem => ({ label: a, description: 'var' })),
        };

        sqliteDialectInfo = {
            name: 'Sqlite',
            icon: 'iconfont icon-sqlite',
            defaultPort: 0,
            formatSqlDialect: 'sql',
            columnTypes: SQLITE_TYPE_LIST.sort((a, b) => a.udtName.localeCompare(b.udtName)),
            editorCompletions,
        };
        return sqliteDialectInfo;
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
            { name: 'id', type: 'integer', length: '', numScale: '', value: '', notNull: true, pri: true, auto_increment: true, remark: '主键ID' },
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
            { name: 'updator', type: 'varchar', length: '100', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '修改姓名' },
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
        return `\"${name}\"`;
    };

    genColumnBasicSql(cl: any): string {
        let val = cl.value ? (cl.value === 'CURRENT_TIMESTAMP' ? cl.value : `'${cl.value}'`) : '';
        let defVal = val ? `DEFAULT ${val}` : '';
        let length = cl.length ? `(${cl.length})` : '';
        let nullAble = cl.notNull ? 'NOT NULL' : 'NULL';
        if (cl.pri) {
            return ` ${this.quoteIdentifier(cl.name)} ${cl.type}${length} PRIMARY KEY ${cl.auto_increment ? 'AUTOINCREMENT' : ''} ${nullAble} `;
        }
        return ` ${this.quoteIdentifier(cl.name)} ${cl.type}${length} ${nullAble} ${defVal} `;
    }
    getCreateTableSql(data: any): string {
        // 创建表结构
        let fields: string[] = [];
        data.fields.res.forEach((item: any) => {
            item.name && fields.push(this.genColumnBasicSql(item));
        });

        return `CREATE TABLE ${this.quoteIdentifier(data.db)}.${this.quoteIdentifier(data.tableName)}
                  ( ${fields.join(',')} )`;
    }

    getCreateIndexSql(data: any): string {
        // 创建索引
        let sql = [] as string[];
        data.indexs.res.forEach((a: any) => {
            sql.push(
                `CREATE ${a.unique ? 'UNIQUE' : ''} INDEX ${this.quoteIdentifier(data.db)}.${this.quoteIdentifier(a.indexName)} ON "${data.tableName}" (${a.columnNames.join(',')})`
            );
        });
        return sql.join(';');
    }

    getModifyColumnSql(tableData: any, tableName: string, changeData: { del: RowDefinition[]; add: RowDefinition[]; upd: RowDefinition[] }): string {
        // sqlite修改表结构需要先删除再创建

        // 1.删除旧表索引  DROP INDEX "main"."aa";
        let sql = [] as string[];
        tableData.indexs.res.forEach((a: any) => {
            a.indexName && sql.push(`DROP INDEX ${this.quoteIdentifier(tableData.db)}.${this.quoteIdentifier(a.indexName)}`);
        });

        // 2.重命名表，备份旧表  ALTER TABLE "main"."t_sys_resource" RENAME TO "_t_sys_resource_old_20240118162712"; new Date().getTime()
        let oldTableName = `_${tableName}_old_${new Date().getTime()}`;
        sql.push(`ALTER TABLE ${this.quoteIdentifier(tableData.db)}.${this.quoteIdentifier(tableName)} RENAME TO ${this.quoteIdentifier(oldTableName)}`);

        // 3.创建新表
        sql.push(this.getCreateTableSql(tableData));

        // 4.复制数据 INSERT INTO "库名"."新表名" (${insertFields}) SELECT ${queryFields} FROM "库名"."旧表名";
        // 查询的字段数据类型和数量应与插入的字段一致
        // 判断哪些字段需要查询旧表，哪些字段需要插入新表
        // 解析changeData，统计需要查询旧表的字段，统计需要插入新表的字段
        let delFields = changeData.del.map((a) => a.name);
        let addFields = changeData.add.map((a) => a.name);

        let queryFields = [] as string[];
        let insertFields = [] as string[];
        tableData.fields.res.forEach((a: any) => {
            // 新增、删除的字段不需要查询旧表，不需要插入新表
            if (addFields.includes(a.name) || delFields.includes(a.name)) {
                return;
            }
            // 修改的字段需要查询和插入，判断是否修改了字段名，如果修改了字段名，需要查询旧表原名，插入新表新名
            // 其余未删除、未修改的字段，需要查询旧表，插入新表
            queryFields.push(this.quoteIdentifier(a.name === a.oldName ? a.name : a.oldName));
            insertFields.push(this.quoteIdentifier(a.name));
        });
        // 生成sql
        sql.push(
            `INSERT INTO ${this.quoteIdentifier(tableData.db)}.${this.quoteIdentifier(tableName)} (${insertFields.join(',')}) SELECT ${queryFields.join(
                ','
            )} FROM ${this.quoteIdentifier(tableData.db)}.${this.quoteIdentifier(oldTableName)}`
        );

        // 5.创建索引
        tableData.indexs.res.forEach((a: any) => {
            a.indexName &&
                sql.push(
                    `CREATE ${a.unique ? 'UNIQUE' : ''} INDEX ${this.quoteIdentifier(tableData.db)}.${this.quoteIdentifier(a.indexName)} ON "${tableName}" (${a.columnNames.join(',')})`
                );
        });

        return sql.join(';') + ';';
    }

    getModifyIndexSql(tableData: any, tableName: string, changeData: { del: any[]; add: any[]; upd: any[] }): string {
        // sqlite创建索引需要先删除再创建
        // CREATE INDEX "main"."aa1" ON "t_sys_resource" ( "ui_path" );

        let sql = [] as string[];

        if (changeData.del.length > 0) {
            changeData.del.forEach((a) => {
                sql.push(`DROP INDEX ${this.quoteIdentifier(a.indexName)}`);
            });
        }

        let indexData = [] as any[];
        if (changeData.add.length > 0) {
            indexData = indexData.concat(changeData.add);
        }
        if (changeData.upd.length > 0) {
            indexData = indexData.concat(changeData.upd);
        }

        if (indexData.length > 0) {
            indexData.forEach((a) => {
                sql.push(`CREATE ${a.unique ? 'UNIQUE' : ''} INDEX ${this.quoteIdentifier(a.indexName)} ON ${tableName} (${a.columnNames.join(',')})`);
            });
        }
        return sql.join(';');
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
