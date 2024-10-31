import { DialectInfo } from './index';
import { PostgresqlDialect } from '@/views/ops/db/dialect/postgres_dialect';

let kbpgDialectInfo: DialectInfo;

export class KingbaseEsDialect extends PostgresqlDialect {
    getInfo(): DialectInfo {
        if (kbpgDialectInfo) {
            return kbpgDialectInfo;
        }

        kbpgDialectInfo = {} as DialectInfo;
        Object.assign(kbpgDialectInfo, super.getInfo());
        kbpgDialectInfo.name = 'KingbaseES';
        kbpgDialectInfo.icon = 'iconfont icon-kingbase';
        return kbpgDialectInfo;
    }
}
