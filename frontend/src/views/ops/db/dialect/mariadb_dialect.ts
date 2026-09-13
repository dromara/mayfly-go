import { DbType } from './dbType';
import { registerDbDialect } from './registry';
import type { DbDialect, DialectInfo } from './types';
import { MysqlDialect } from './mysql_dialect';

let mariadbDialectInfo: DialectInfo;

/**
 * MariaDB 方言：协议与语法完全兼容 MySQL，仅元信息（名称/图标）不同。
 * 能力声明、DDL 生成、切割语义等均继承 MysqlDialect，无需重复声明。
 */
class MariadbDialect extends MysqlDialect implements DbDialect {
    getInfo(): DialectInfo {
        if (mariadbDialectInfo) {
            return mariadbDialectInfo;
        }

        mariadbDialectInfo = {} as DialectInfo;
        Object.assign(mariadbDialectInfo, super.getInfo());
        mariadbDialectInfo.name = 'MariaDB';
        mariadbDialectInfo.icon = 'icon db/mariadb';
        return mariadbDialectInfo;
    }
}

registerDbDialect(DbType.mariadb, new MariadbDialect());
