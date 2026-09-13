import { DbType } from './dbType';
import { registerDbDialect } from './registry';
import type { DialectInfo } from './types';
import { PostgresqlDialect } from './postgres_dialect';

let kbpgDialectInfo: DialectInfo;

/**
 * 人大金仓 KingbaseES（PG 模式）方言：语法与能力完全继承 PostgresqlDialect，
 * 差异仅在元信息（名称/图标）。
 *
 * @see https://help.kingbase.com.cn/v8/index.html
 */
class KingbaseEsDialect extends PostgresqlDialect {
    getInfo(): DialectInfo {
        if (kbpgDialectInfo) {
            return kbpgDialectInfo;
        }

        kbpgDialectInfo = {} as DialectInfo;
        Object.assign(kbpgDialectInfo, super.getInfo());
        kbpgDialectInfo.name = 'KingbaseES';
        kbpgDialectInfo.icon = 'icon db/kingbase';
        return kbpgDialectInfo;
    }
}

registerDbDialect(DbType.kingbaseEs, new KingbaseEsDialect());
