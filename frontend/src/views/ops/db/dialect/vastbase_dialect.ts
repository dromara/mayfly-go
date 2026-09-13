import { DbType } from './dbType';
import { registerDbDialect } from './registry';
import type { DialectInfo } from './types';
import { PostgresqlDialect } from './postgres_dialect';

let vastDialectInfo: DialectInfo;

/**
 * Vastbase G100 方言：语法与能力完全继承 PostgresqlDialect，差异仅在元信息（名称/图标）。
 *
 * @see https://docs.vastdata.com.cn/zh/docs/VastbaseG100Ver2.2.5/doc/开发者指南/SQL参考/SQL参考.html
 */
class VastbaseDialect extends PostgresqlDialect {
    getInfo(): DialectInfo {
        if (vastDialectInfo) {
            return vastDialectInfo;
        }

        vastDialectInfo = {} as DialectInfo;
        Object.assign(vastDialectInfo, super.getInfo());
        vastDialectInfo.name = 'VastbaseG100';
        vastDialectInfo.icon = 'icon db/vastbase';
        return vastDialectInfo;
    }
}

registerDbDialect(DbType.vastbase, new VastbaseDialect());
