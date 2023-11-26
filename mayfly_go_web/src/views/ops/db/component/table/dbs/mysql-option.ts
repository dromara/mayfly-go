import { DbOption, sqlType } from '@/views/ops/db/component/table/dbs/db-option';
import { MYSQL_TYPE_LIST } from '@/views/ops/db/component/table/service';

export class MysqlOption implements DbOption {
    getTypeList(): sqlType[] {
        return MYSQL_TYPE_LIST.map((a) => ({ udtName: a, dataType: a, desc: '', space: '' }));
    }

    genColumnBasicSql(cl: any): string {
        let val = cl.value ? (cl.value === 'CURRENT_TIMESTAMP' ? cl.value : `'${cl.value}'`) : '';
        let defVal = val ? `DEFAULT ${val}` : '';
        let length = cl.length ? `(${cl.length})` : '';
        let onUpdate = 'update_time' === cl.name ? ' ON UPDATE CURRENT_TIMESTAMP ' : '';
        return ` ${cl.name} ${cl.type}${length} ${cl.notNull ? 'NOT NULL' : 'NULL'} ${cl.auto_increment ? 'AUTO_INCREMENT' : ''} ${defVal} ${onUpdate} comment '${cl.remark || ''}' `;
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

    getModifyColumnSql(tableName: string, changeData: { del: any[]; add: any[]; upd: any[] }): string {
        let addSql = '',
            updSql = '',
            delSql = '';
        if (changeData.add.length > 0) {
            addSql = `ALTER TABLE ${tableName}`;
            changeData.add.forEach((a) => {
                addSql += ` ADD ${this.genColumnBasicSql(a)},`;
            });
            addSql = addSql.substring(0, addSql.length - 1);
            addSql += ';';
        }

        if (changeData.upd.length > 0) {
            updSql = `ALTER TABLE ${tableName}`;
            let arr = [] as string[];
            changeData.upd.forEach((a) => {
                arr.push(` MODIFY ${this.genColumnBasicSql(a)}`);
            });
            updSql += arr.join(',');
            updSql += ';';
        }

        if (changeData.del.length > 0) {
            changeData.del.forEach((a) => {
                delSql += ` ALTER TABLE ${tableName} DROP COLUMN ${a.name}; `;
            });
        }
        return addSql + updSql + delSql;
    }

    getModifyIndexSql(tableName: string, changeData: { del: any[]; add: any[]; upd: any[] }): string {
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
                    sql += ` ADD ${a.unique ? 'UNIQUE' : ''} INDEX ${a.indexName}(${a.columnNames.join(',')}) USING ${a.indexType} COMMENT '${a.indexComment}',`;
                });
                sql = sql.substring(0, sql.length - 1);
            }
            return sql;
        }
        return '';
    }
}
