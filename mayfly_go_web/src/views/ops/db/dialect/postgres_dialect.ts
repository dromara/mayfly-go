import { DbDialect, DialectInfo, sqlColumnType } from './index';

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

const postgresDialectInfo: DialectInfo = {
    icon: 'iconfont icon-op-postgres',
    defaultPort: 5432,
    formatSqlDialect: 'postgresql',
    columnTypes: GAUSS_TYPE_LIST.sort((a, b) => a.udtName.localeCompare(b.udtName)),
};

class PostgresqlDialect implements DbDialect {
    getInfo(): DialectInfo {
        return postgresDialectInfo;
    }

    getDefaultSelectSql(table: string, condition: string, orderBy: string, pageNum: number, limit: number) {
        return `SELECT * FROM ${this.wrapName(table)} ${condition ? 'WHERE ' + condition : ''} ${orderBy ? orderBy : ''}  OFFSET ${
            (pageNum - 1) * limit
        } LIMIT ${limit};`;
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
            if (this.matchType(cl.type, ['char', 'time', 'date', 'text'])) {
                // 默认值是now()的time或date不需要加引号
                if (cl.value.toLowerCase().replace(' ', '') === 'CURRENT_TIMESTAMP' && this.matchType(cl.type, ['time', 'date'])) {
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
        return ` ${cl.name} ${cl.type}${length} ${cl.notNull ? 'NOT NULL' : ''} ${defVal} `;
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
            if (a.indexComment) {
                sql.push(`COMMENT ON INDEX ${a.indexName} IS '${a.indexComment}'`);
            }
        });
        return sql.join(';');
    }

    getModifyColumnSql(tableName: string, changeData: { del: any[]; add: any[]; upd: any[] }): string {
        let sql: string[] = [];
        if (changeData.add.length > 0) {
            changeData.add.forEach((a) => {
                let typeLength = this.getTypeLengthSql(a);
                let defaultSql = this.getDefaultValueSql(a);
                sql.push(`ALTER TABLE ${tableName} add ${a.name} ${a.type}${typeLength} ${defaultSql}`);
            });
        }

        if (changeData.upd.length > 0) {
            changeData.upd.forEach((a) => {
                let typeLength = this.getTypeLengthSql(a);
                sql.push(`ALTER TABLE ${tableName} alter column ${a.name} type ${a.type}${typeLength}`);
                let defaultSql = this.getDefaultValueSql(a);
                if (defaultSql) {
                    sql.push(`alter table ${tableName} alter column ${a.name} set ${defaultSql}`);
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
