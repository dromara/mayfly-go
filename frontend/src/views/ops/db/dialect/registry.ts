/**
 * 方言注册表
 *
 * 独立于 index.ts 的原因：各方言文件需在模块体内自注册，若把注册函数并入 index.ts，
 * 会形成「方言 → index → 方言」的模块循环，触发
 * `Cannot access 'DbType' before initialization` 的 TDZ 运行时报错。
 *
 * 本文件只依赖 dbType.ts（纯常量）与 types.ts（纯类型），属方言层的零依赖内核，
 * 任何模块（含方言自身）均可安全导入。
 */

import { featureCapabilityMap, type BooleanCapabilityKey } from './shared/capabilities';
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

// ==================== 能力协商 ====================

/**
 * 后端命名空间层次（与后端 NamespaceHierarchy 对齐，字段首字母大写为 JSON 序列化原样）。
 */
export interface BackendNamespace {
    HasDatabase: boolean;
    HasSchema: boolean;
    HasCatalog: boolean;
}

/**
 * 后端能力协商响应（与 GET /dbs/{id}/capabilities 返回结构对齐）。
 */
export interface BackendCapabilities {
    dbType: string;
    features: string[];
    namespace: BackendNamespace;
}

/**
 * 协商后的能力声明：静态方言能力 ∩ 后端实际能力。
 *
 * 继承 DialectCapabilities 全部字段（消费方无需改动），额外提供：
 * - backendFeatures：后端原始能力清单（数据驱动渲染，如资源树扩展对象节点显隐）
 * - namespace：后端命名空间层次（database/schema/catalog）
 *
 * 协商语义：
 * - 前端方言静态声明为「天花板」——前端不支持的能力后端无法赋予
 * - 后端实际能力为「约束」——后端不支持的能力前端必须关闭
 * - 无后端对应 feature 的纯前端能力（如 supportsTableEdit）保持前端声明
 * - 无前端对应能力位的纯后端 feature（如 view/sequence）原样透传至 backendFeatures
 */
export interface NegotiatedCapabilities extends DialectCapabilities {
    /** 后端原始能力清单（SupportedFeatures 输出），供数据驱动渲染 */
    backendFeatures: string[];
    /** 后端命名空间层次 */
    namespace: BackendNamespace;
}

/**
 * 协商方言能力：合并静态方言声明与后端运行时能力。
 *
 * 算法：
 * 1. 从静态方言声明出发（天花板）
 * 2. 后端 feature 经 featureCapabilityMap 翻译为前端能力位覆盖
 * 3. 后端不支持的能力 → 前端对应能力位强制关闭（交集语义）
 * 4. 纯前端能力（无对应 feature 映射）保持不变
 * 5. 纯后端能力（view/sequence 等）经 backendFeatures 透传
 *
 * 调用方一律用本函数取协商结果，不要分别查 static + backend 再手动合并。
 * 新增方言或能力无需修改本函数——featureCapabilityMap 数据驱动映射。
 *
 * @param dialect 前端方言实例（提供静态能力声明）
 * @param backend 后端能力协商响应（/capabilities 端点返回）
 */
export function negotiateCapabilities(dialect: DbDialect, backend: BackendCapabilities): NegotiatedCapabilities {
    // 1. 静态方言声明为起点（天花板）
    const staticCaps = getDialectCapabilities(dialect);
    const negotiated: NegotiatedCapabilities = {
        ...staticCaps,
        backendFeatures: backend.features,
        namespace: backend.namespace,
    };

    // 2. 后端 feature 经映射桥翻译为前端能力位覆盖
    const backendEnabled: Record<string, boolean> = {};
    for (const feature of backend.features) {
        const override = featureCapabilityMap[feature];
        if (!override) continue;
        for (const [key, value] of Object.entries(override)) {
            backendEnabled[key] = (value) && true;
        }
    }

    // 3. 收集映射桥覆盖过的所有能力键（含后端未启用的）；直接遍历映射值，避开用字符串键回查映射表
    const mappedKeys = new Set<BooleanCapabilityKey>();
    for (const override of Object.values(featureCapabilityMap)) {
        for (const key of Object.keys(override) as BooleanCapabilityKey[]) {
            mappedKeys.add(key);
        }
    }

    // 4. 交集语义：后端未启用的映射能力位强制关闭
    for (const key of mappedKeys) {
        if (!backendEnabled[key]) {
            negotiated[key] = false;
        }
    }

    return negotiated;
}
