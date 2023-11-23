import { DbOption, sqlType } from '@/views/ops/db/component/table/dbs/db-option';
import { GAUSS_TYPE_LIST } from '@/views/ops/db/component/table/service';

export class PostgresqlOption implements DbOption {
    getTypeList(): sqlType[] {
        return GAUSS_TYPE_LIST.sort((a, b) => a.udtName.localeCompare(b.udtName));
    }

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
