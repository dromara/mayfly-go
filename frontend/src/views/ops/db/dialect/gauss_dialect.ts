import { PostgresqlDialect } from '@/views/ops/db/dialect/postgres_dialect';
import { DialectInfo, DuplicateStrategy } from '@/views/ops/db/dialect/index';

let gsDialectInfo: DialectInfo;
export class GaussDialect extends PostgresqlDialect {
    getInfo(): DialectInfo {
        if (gsDialectInfo) {
            return gsDialectInfo;
        }

        gsDialectInfo = {} as DialectInfo;
        Object.assign(gsDialectInfo, super.getInfo());
        gsDialectInfo.icon = 'icon db/guass';
        gsDialectInfo.name = 'GaussDB';
        return gsDialectInfo;
    }

    getBatchInsertPreviewSql(tableName: string, fieldArr: string[], duplicateStrategy: DuplicateStrategy): string {
        // 构建占位符字符串 "($1, $2, $3 ...)"
        let placeholder = fieldArr.map((_, i) => `$${i + 1}`).join(',');
        let suffix = '';
        if (duplicateStrategy === DuplicateStrategy.IGNORE) {
            suffix = '\nON DUPLICATE KEY UPDATE NOTHING';
        } else if (duplicateStrategy === DuplicateStrategy.REPLACE) {
            suffix = '\n-- 执行前会删除唯一键涉及到的字段 \nON DUPLICATE KEY UPDATE ' + fieldArr.map((a) => `${a}=excluded.${a}`).join(',');
        }

        return `INSERT INTO ${tableName} (${fieldArr.join(',')}) VALUES (${placeholder}) ${suffix};`;
    }
}
