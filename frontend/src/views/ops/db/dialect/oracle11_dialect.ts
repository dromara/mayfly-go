/** oracle 11g 及以前的版本的一些语法兼容  */
import {OracleDialect} from '@/views/ops/db/dialect/oracle_dialect';
import {DialectInfo, RowDefinition} from '@/views/ops/db/dialect/index';

let oracle11DialectInfo: DialectInfo;

export class Oracle11Dialect extends OracleDialect {

    getInfo(): DialectInfo {
        if (oracle11DialectInfo) {
            return oracle11DialectInfo;
        }

        oracle11DialectInfo = {} as DialectInfo;
        Object.assign(oracle11DialectInfo, super.getInfo());
        oracle11DialectInfo.name = 'Oracle11x';
        return oracle11DialectInfo;
    }

    // 重写创建自增列sql

    genColumnBasicSql(cl: RowDefinition, create: boolean, data = {}): string {
        let length = this.getTypeLengthSql(cl);
        // 默认值
        let defVal = this.getDefaultValueSql(cl, false, data);
        // 忽略自增配置，11g不支持直接设置自增列，需要单独设置自增序列
        // 如果有原名以原名为准
        let name = cl.oldName && cl.name !== cl.oldName ? cl.oldName : cl.name;
        let baseSql = ` ${this.quoteIdentifier(name)} ${cl.type}${length}`;
        return ` ${baseSql} ${defVal} ${cl.notNull ? 'NOT NULL' : ''} `;
    }

    getDefaultValueSql(cl: RowDefinition, create?: boolean, data?: any): string {
        if (cl.value) {
            return ` DEFAULT ${cl.value}`;
        } else if (cl.auto_increment) {
            return ` DEFAULT ${data.tableName}_${cl.name}_SEQ.NEXTVAL`;
        }
        return '';
    }

    getOtherCreateTableSql(data: any): string {
        // 通过字段自增信息创建自增序列

        let result = '';
        data.fields.res.forEach((field: RowDefinition) => {
            let seqName = `${data.tableName}_${field.name}_SEQ`;
            if (field.auto_increment) {
                result += `CREATE SEQUENCE ${seqName} START WITH 1 INCREMENT BY 1 CACHE 20`;
            }
        });

        return result;
    }
}
