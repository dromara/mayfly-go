import { PostgresqlDialect } from '@/views/ops/db/dialect/postgres_dialect';
import { DialectInfo } from '@/views/ops/db/dialect/index';

let gsDialectInfo: DialectInfo;
export class GaussDialect extends PostgresqlDialect {
    getInfo(): DialectInfo {
        if (gsDialectInfo) {
            return gsDialectInfo;
        }

        gsDialectInfo = {} as DialectInfo;
        Object.assign(gsDialectInfo, super.getInfo());
        gsDialectInfo.icon = 'iconfont icon-gauss';
        gsDialectInfo.name = 'GaussDB';
        return gsDialectInfo;
    }
}
