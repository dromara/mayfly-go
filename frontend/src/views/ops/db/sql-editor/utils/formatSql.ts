import type { SqlFormatterLanguage } from '../../dialect/types';

/**
 * SQL 格式化的惰性入口（sql-formatter）
 *
 * sql-formatter 自带全部方言的语法表，产物约 200K（gz 约 55K），而格式化只发生在
 * 「点格式化按钮 / 查看 DDL / 弹出执行 SQL 弹窗」这三类用户动作里。静态 import 会把它并进
 * db 模块的共享 chunk，使表数据页、资源树、实例列表这些从不格式化的页面也一并下载。
 * 故与 completion/lazy.ts 采用同一约定：用到的那一刻才动态取。
 *
 * 该约定由 __tests__/monacoBoundary.test.ts 锁住（db 模块内禁止静态 import 'sql-formatter'）。
 */

type SqlFormatterModule = typeof import('sql-formatter');

/** 模块只加载一次；同一 tick 内的并发调用共享同一个 pending promise */
let formatterPromise: Promise<SqlFormatterModule> | null = null;

function loadFormatter(): Promise<SqlFormatterModule> {
    if (!formatterPromise) {
        // 与 completion/lazy.ts 同理：加载失败不缓存 rejection，否则本会话永久失去格式化能力
        formatterPromise = import('sql-formatter').catch((e: unknown) => {
            formatterPromise = null;
            throw e;
        });
    }
    return formatterPromise;
}

/**
 * 格式化 SQL；取不到格式化器时原样返回，不阻断调用方。
 *
 * 格式化失败不该让「看 DDL」「执行 SQL」这类动作整体失败，故降级策略集中在本函数，
 * 调用方无需各自 try/catch。
 *
 * @param sql 待格式化的语句
 * @param language 方言标识，一律取自 dialect.getInfo().formatSqlDialect（唯一事实源）
 */
export async function formatSql(sql: string, language: SqlFormatterLanguage): Promise<string> {
    try {
        const { format } = await loadFormatter();
        return format(sql, { language });
    } catch (e: unknown) {
        console.error('[db] format sql failed:', e);
        return sql;
    }
}
