import { DbType } from './dbType';
import { registerDbDialect } from './registry';
import { DuplicateStrategy } from './types';
import type { DialectInfo } from './types';
import { PostgresqlDialect } from './postgres_dialect';

let gsDialectInfo: DialectInfo;

/**
 * GaussDB 方言：兼容 PostgreSQL 语法，能力声明与 DDL 生成均继承 PostgresqlDialect。
 * 差异仅在元信息（名称/图标）与批量插入的冲突处理语法。
 */
class GaussDialect extends PostgresqlDialect {
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

registerDbDialect(DbType.gauss, new GaussDialect());
