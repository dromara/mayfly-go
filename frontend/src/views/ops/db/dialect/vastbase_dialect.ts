import { DialectInfo } from './index';
import { PostgresqlDialect } from '@/views/ops/db/dialect/postgres_dialect';

let vastDialectInfo: DialectInfo;

export class VastbaseDialect extends PostgresqlDialect {
    getInfo(): DialectInfo {
        if (vastDialectInfo) {
            return vastDialectInfo;
        }

        vastDialectInfo = {} as DialectInfo;
        Object.assign(vastDialectInfo, super.getInfo());
        vastDialectInfo.name = 'VastbaseG100';
        vastDialectInfo.icon = 'iconfont icon-vastbase';
        return vastDialectInfo;
    }
}
