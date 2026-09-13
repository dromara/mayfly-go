/**
 * 方言注册表
 *
 * 独立于 index.ts 的原因：各方言文件需要在模块体内自注册，若注册函数放在 index.ts，
 * 会形成「方言 → index → 方言」的模块循环——历史上已导致
 * `Cannot access 'DbType' before initialization` 的 TDZ 运行时报错。
 *
 * 本文件只依赖 dbType.ts（纯常量）与 types.ts（纯类型），属方言层的零依赖内核，
 * 任何模块（含方言自身）均可安全导入。
 */

import { DbType } from './dbType';
import type { DbDialect, DialectCapabilities } from './types';

/** dbType → 方言实例 */
const dbType2DialectMap = new Map<string, DbDialect>();
/** dbType + version → 方言实例（如 oracle11） */
const dbType2DialectVersionMap = new Map<string, DbDialect>();

/** 未注册类型的回退方言，由 index.ts 在方言加载完成后注入 */
let fallbackDialect: DbDialect | undefined;

/**
 * 设置回退方言。未识别的 dbType 将回退到它，避免调用方拿到 undefined。
 * 由 index.ts 调用，方言文件无需关心。
 */
export function setFallbackDialect(dialect: DbDialect) {
    fallbackDialect = dialect;
}

/** 注册基础方言 */
export function registerDbDialect(dbType: string, dd: DbDialect) {
    dbType2DialectMap.set(dbType, dd);
}

/**
 * 注册版本方言（如 Oracle 11g）
 * @param dbTypeWithVersion 已拼接版本的 key，如 `${DbType.oracle}11`
 */
export function registerDbDialectVersion(dbTypeWithVersion: string, dd: DbDialect) {
    dbType2DialectVersionMap.set(dbTypeWithVersion, dd);
}

/** 获取所有已注册的基础方言（key 为 dbType） */
export function getDbDialectMap(): Map<string, DbDialect> {
    return dbType2DialectMap;
}

/** 获取所有已注册的 dbType 列表 */
export function getRegisteredDbTypes(): string[] {
    return [...dbType2DialectMap.keys()];
}

/** 获取所有已注册的版本方言 key（如 oracle11） */
export function getRegisteredDialectVersions(): string[] {
    return [...dbType2DialectVersionMap.keys()];
}

/**
 * 获取方言实例：优先版本方言，回退基础方言，最终回退 MySQL 语法。
 *
 * @param dbType 数据库类型，见 DbType
 * @param version 数据库版本（可选），用于命中版本特化方言
 */
export function getDbDialect(dbType: string, version = ''): DbDialect {
    const matched = dbType2DialectVersionMap.get(dbType + version) || dbType2DialectMap.get(dbType);
    if (matched) {
        return matched;
    }

    // 未注册类型按最通用的 MySQL 语法处理（与历史行为一致）
    const fallback = fallbackDialect || dbType2DialectMap.get(DbType.mysql);
    if (!fallback) {
        // 仅在方言模块尚未加载时发生，属初始化顺序错误，快速失败以便定位
        throw new Error(`[db-dialect] 方言注册表未初始化，且未找到类型 ${dbType} 的方言`);
    }
    return fallback;
}

/** 方言能力声明缓存：能力在方言实例生命周期内恒定，按实例缓存一份即可 */
const capabilitiesCache = new WeakMap<DbDialect, DialectCapabilities>();

/**
 * 读取方言能力声明（带实例级缓存）
 *
 * 表单与表格的渲染路径会高频读取能力（如逐行判断自增列可否编辑、编辑器逐次按键取引号对），
 * 而各方言的 getCapabilities() 每次都合并缺省值新建对象，在热路径上造成无谓的分配与 GC 压力。
 *
 * 调用方一律用本函数取能力，不要直接调 dialect.getCapabilities()；
 * 新增方言无需为本函数做任何适配——照常实现 getCapabilities() 即自动享受缓存。
 */
export function getDialectCapabilities(dialect: DbDialect): DialectCapabilities {
    let capabilities = capabilitiesCache.get(dialect);
    if (!capabilities) {
        capabilities = dialect.getCapabilities();
        capabilitiesCache.set(dialect, capabilities);
    }
    return capabilities;
}
