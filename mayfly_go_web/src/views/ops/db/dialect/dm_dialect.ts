import { DbDialect, sqlColumnType, DialectInfo, RowDefinition, IndexDefinition } from './index';

export { DMDialect, DM_TYPE_LIST };

// 参考文档:https://eco.dameng.com/document/dm/zh-cn/sql-dev/dmpl-sql-datatype.html#%E5%AD%97%E7%AC%A6%E6%95%B0%E6%8D%AE%E7%B1%BB%E5%9E%8B
const DM_TYPE_LIST: sqlColumnType[] = [
    // 字符数据类型
    { udtName: 'CHAR', dataType: 'VARCHAR', desc: '定长字符串', space: '', range: '1 - 32767' },
    { udtName: 'VARCHAR', dataType: 'VARCHAR', desc: '变长字符串', space: '', range: '1 - 32767' },

    // 精确数值数据类型 NUMERIC、DECIMAL、DEC 类型、NUMBER 类型、INTEGER 类型、INT 类型、BIGINT 类型、TINYINT 类型、BYTE 类型、SMALLINT
    { udtName: 'NUMERIC', dataType: 'NUMERIC', desc: '零、正负定点数', space: '1-38', range: '' },
    { udtName: 'DECIMAL', dataType: 'DECIMAL', desc: '与NUMERIC相似', space: '1-38', range: '' },
    { udtName: 'NUMBER', dataType: 'NUMBER', desc: '同NUMERIC', space: '1-38', range: '' },
    { udtName: 'INTEGER', dataType: 'INTEGER', desc: '有符号整数', space: '10', range: '-2^31-1 ~ 2^31-1' },
    { udtName: 'INT', dataType: 'INT', desc: '同INTEGER', space: '10', range: '' },
    { udtName: 'BIGINT', dataType: 'BIGINT', desc: '有符号整数', space: '19', range: '-2^63-1 ~ 2^63-1' },
    { udtName: 'TINYINT', dataType: 'TINYINT', desc: '有符号整数', space: '3', range: '-128~+127' },
    { udtName: 'BYTE', dataType: 'BYTE', desc: '与 TINYINT 相似', space: '3', range: '' },
    { udtName: 'SMALLINT', dataType: 'SMALLINT', desc: '有符号整数', space: '5', range: '-2^15-1 ~ 2^15-1' },
    // (用得少，忽略)近似数值类型包括：FLOAT 类型、DOUBLE 类型、REAL 类型、DOUBLE PRECISION 类型。
    // 位串数据类型 BIT 用于存储整数数据 1、0 或 NULL，只有 0 才转换为假，其他非空、非 0 值都会自动转换为真
    { udtName: 'BIT', dataType: 'BIT', desc: '用于存储整数数据 1、0 或 NULL', space: '1', range: '1' },
    // 一般日期时间数据类型 DATE TIME TIMESTAMP 默认精度 6
    // 多媒体数据类型 TEXT/LONG/LONGVARCHAR 类型：变长字符串类型  IMAGE/LONGVARBINARY 类型  BLOB CLOB BFILE  100G-1
    { udtName: 'DATE', dataType: 'DATE', desc: '年、月、日', space: '', range: '' },
    { udtName: 'TIME', dataType: 'TIME', desc: '时、分、秒', space: '', range: '' },
    {
        udtName: 'TIMESTAMP',
        dataType: 'TIMESTAMP',
        desc: '年、月、日、时、分、秒',
        space: '',
        range: '-4712-01-01 00:00:00.000000000 ~ 9999-12-31 23:59:59.999999999',
    },
    { udtName: 'TEXT', dataType: 'TEXT', desc: '变长字符串', space: '', range: '100G-1' },
    { udtName: 'LONG', dataType: 'LONG', desc: '同TEXT', space: '', range: '100G-1' },
    { udtName: 'LONGVARCHAR', dataType: 'LONGVARCHAR', desc: '同TEXT', space: '', range: '100G-1' },
    { udtName: 'IMAGE', dataType: 'IMAGE', desc: '图像二进制类型', space: '', range: '100G-1' },
    { udtName: 'LONGVARBINARY', dataType: 'LONGVARBINARY', desc: '同IMAGE', space: '', range: '100G-1' },
    { udtName: 'BLOB', dataType: 'BLOB', desc: '变长的二进制大对象', space: '', range: '100G-1' },
    { udtName: 'CLOB', dataType: 'CLOB', desc: '同TEXT', space: '', range: '100G-1' },
    { udtName: 'BFILE', dataType: 'BFILE', desc: '二进制文件', space: '', range: '100G-1' },
];

const dmDialectInfo: DialectInfo = {
    icon: 'iconfont icon-db-dm',
    defaultPort: 5236,
    formatSqlDialect: 'postgresql',
    columnTypes: DM_TYPE_LIST.sort((a, b) => a.udtName.localeCompare(b.udtName)),
};

class DMDialect implements DbDialect {
    getInfo() {
        return dmDialectInfo;
    }

    getDefaultSelectSql(table: string, condition: string, orderBy: string, pageNum: number, limit: number) {
        return `SELECT * FROM ${this.wrapName(table)} ${condition ? 'WHERE ' + condition : ''} ${orderBy ? orderBy : ''}  OFFSET ${
            (pageNum - 1) * limit
        } LIMIT ${limit};`;
    }

    getDefaultRows(): RowDefinition[] {
        return [
            { name: 'id', type: 'BIGINT', length: '', numScale: '', value: '', notNull: true, pri: true, auto_increment: true, remark: '主键ID' },
            { name: 'creator_id', type: 'BIGINT', length: '', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '创建人id' },
            {
                name: 'creator',
                type: 'VARCHAR',
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
                type: 'TIMESTAMP',
                length: '',
                numScale: '',
                value: 'SYSDATE',
                notNull: true,
                pri: false,
                auto_increment: false,
                remark: '创建时间',
            },
            { name: 'updator_id', type: 'BIGINT', length: '', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '修改人id' },
            {
                name: 'updator',
                type: 'VARCHAR',
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
                type: 'TIMESTAMP',
                length: '',
                numScale: '',
                value: 'SYSDATE',
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
    wrapName = (name: string) => {
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
            if (this.matchType(cl.type, ['CHAR', 'TIME', 'DATE', 'TEXT'])) {
                // 默认值是now()的time或date不需要加引号
                let val = cl.value.toUpperCase().replace(' ', '');
                if (this.matchType(cl.type, ['TIME', 'DATE']) && ['CURRENT_DATE', 'SYSDATE', 'CURDATE', 'CURTIME'].includes(val)) {
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
        let incr = cl.auto_increment ? 'IDENTITY' : '';
        return ` ${cl.name} ${cl.type}${length} ${incr} ${cl.notNull ? 'NOT NULL' : ''} ${defVal} `;
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
        let sql: string[] = [];
        tableData.indexs.res.forEach((a: any) => {
            sql.push(` CREATE ${a.unique ? 'UNIQUE' : ''} INDEX ${a.indexName} USING btree ("${a.columnNames.join('","')})"`);
        });
        return sql.join(';');
    }

    getModifyColumnSql(tableName: string, changeData: { del: RowDefinition[]; add: RowDefinition[]; upd: RowDefinition[] }): string {
        let sql: string[] = [];
        if (changeData.add.length > 0) {
            changeData.add.forEach((a) => {
                sql.push(`ALTER TABLE ${tableName} add COLUMN ${this.genColumnBasicSql(a)}`);
                if (a.remark) {
                    sql.push(`comment on COLUMN "${tableName}"."${a.name}" is '${a.remark}'`);
                }
            });
        }

        if (changeData.upd.length > 0) {
            changeData.upd.forEach((a) => {
                sql.push(`ALTER TABLE ${tableName} MODIFY ${this.genColumnBasicSql(a)}`);
                if (a.remark) {
                    sql.push(`comment on COLUMN "${tableName}"."${a.name}" is '${a.remark}'`);
                }
            });
        }

        if (changeData.del.length > 0) {
            changeData.del.forEach((a) => {
                sql.push(`ALTER TABLE ${tableName} DROP COLUMN ${a.name}`);
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
                    sql.push(`CREATE ${a.unique ? 'UNIQUE' : ''} INDEX ${a.indexName}(${a.columnNames.join(',')})`);
                    if (a.indexComment) {
                        sql.push(`COMMENT ON INDEX ${a.indexName} IS '${a.indexComment}'`);
                    }
                });
            }
            return sql.join(';');
        }
        return '';
    }
}
