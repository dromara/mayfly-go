import { DbDialect, DialectInfo } from './index';
import { MysqlDialect } from '@/views/ops/db/dialect/mysql_dialect';

export { MariadbDialect };

let mariadbDialectInfo: DialectInfo;
class MariadbDialect extends MysqlDialect implements DbDialect {
    getInfo(): DialectInfo {
        if (mariadbDialectInfo) {
            return mariadbDialectInfo;
        }

        mariadbDialectInfo = {} as DialectInfo;
        Object.assign(mariadbDialectInfo, super.getInfo());
        mariadbDialectInfo.name = 'MariaDB';
        mariadbDialectInfo.icon = 'iconfont icon-mariadb';
        return mariadbDialectInfo;
    }
}
