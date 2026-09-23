/** oracle 11g 及以前的版本的一些语法兼容 */
import { DbType } from './dbType';
import { registerDbDialectVersion } from './registry';
import type { DialectInfo, ColumnDefinition } from './types';
import type { TableEditContext } from './types';
import { OracleDialect } from './oracle_dialect';
import { buildSchemaTable, QuoteEscape } from './shared/utils';

let oracle11DialectInfo: DialectInfo;

class Oracle11Dialect extends OracleDialect {
    getInfo(): DialectInfo {
        if (oracle11DialectInfo) {
            return oracle11DialectInfo;
        }

        oracle11DialectInfo = {} as DialectInfo;
        Object.assign(oracle11DialectInfo, super.getInfo());
        oracle11DialectInfo.name = 'Oracle11x';
        return oracle11DialectInfo;
    }

    // ==================== Oracle 11g 差异：不支持 IDENTITY，使用序列 + 触发器 ====================

    /** 11g 列 DDL：忽略自增配置（不生成 IDENTITY），自增由序列单独处理 */
    private genOracle11ColumnSql(cl: ColumnDefinition, data?: TableEditContext): string {
        const length = this.getTypeLengthSql(cl);
        let defVal = this.getDefaultValueSql(cl);
        // 自增列使用序列 NEXTVAL 作为默认值
        if (cl.autoIncrement && data) {
            defVal = ` DEFAULT ${data.tableName}_${cl.name}_SEQ.NEXTVAL`;
        }
        const name = this.resolveColumnName(cl);
        return ` ${this.quoteIdentifier(name)} ${cl.type}${length} ${defVal} ${cl.nullable ? '' : 'NOT NULL'} `;
    }

    /** 基类 genColumnBasicSql 的 11g 适配：无 data 上下文时忽略自增序列 */
    genColumnBasicSql(cl: ColumnDefinition): string {
        return this.genOracle11ColumnSql(cl);
    }

    getOtherCreateTableSql(data: TableEditContext): string {
        // 通过字段自增信息创建自增序列
        let result = '';
        data.fields.res.forEach((field: ColumnDefinition) => {
            let seqName = `${data.tableName}_${field.name}_SEQ`;
            if (field.autoIncrement) {
                result += `CREATE SEQUENCE ${seqName} START WITH 1 INCREMENT BY 1 CACHE 20`;
            }
        });
        return result;
    }

    /** 覆写建表：使用 11g 专属列 DDL（含序列默认值） */
    getCreateTableSql(data: TableEditContext): string {
        const dbTable = buildSchemaTable(this.quoteIdentifier, data.db, data.tableName);

        let createSql = '';
        let tableCommentSql = '';
        let columCommentSql = '';
        let pris: string[] = [];

        let fields: string[] = [];
        data.fields.res.forEach((item: ColumnDefinition) => {
            item.name && fields.push(this.genOracle11ColumnSql(item, data));
            if (item.comment) {
                columCommentSql += `COMMENT ON COLUMN ${dbTable}.${this.quoteIdentifier(item.name)} IS '${QuoteEscape(item.comment)}';`;
            }
            if (item.isPrimaryKey) {
                pris.push(this.quoteIdentifier(item.name));
            }
        });

        let prisql = '';
        if (pris.length > 0) {
            prisql = `PRIMARY KEY (${pris.join(',')})`;
        }
        const fieldStr = fields.join(',');
        createSql = `CREATE TABLE ${dbTable} (${fieldStr}${prisql ? ', ' + prisql : ''});`;
        if (data.tableComment) {
            tableCommentSql = `COMMENT ON TABLE ${dbTable} IS '${QuoteEscape(data.tableComment)}';`;
        }
        let other = this.getOtherCreateTableSql(data);
        return createSql + tableCommentSql + columCommentSql + other;
    }
}

// 版本特化方言：仅当实例版本为 11 时命中，其余 Oracle 版本走基础方言
registerDbDialectVersion(DbType.oracle + '11', new Oracle11Dialect());
