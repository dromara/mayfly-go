/**
 * DB 实例缓存管理
 * 负责 DbInst 实例的全局缓存、表/列信息本地存储
 */
import { type RemovableRef, useLocalStorage } from '@vueuse/core';
import type { DbTableInfo, DbInstInfo, DbNamesParam } from '../types';
import { dbApi } from '../api';
import { DbGetDbNamesMode } from '../enums';

// ==================== 本地存储缓存 ====================

const hintsStorage: RemovableRef<Map<string, Record<string, string[]>>> = useLocalStorage('db-table-hints', new Map());
const tableStorage: RemovableRef<Map<string, DbTableInfo[]>> = useLocalStorage('db-tables', new Map());

// ==================== 实例缓存 ====================

/**
 * 实例缓存的最小结构契约。
 *
 * 本文件属 core 层，不能反向 import DbInst（会成环），故按「缓存实际读写的字段」定义契约：
 * `id` 作缓存键、`tagPath` 因一库可关联多标签需回填、`databases` 用于取首库查兼容版本、
 * `version` 由兼容版本接口异步补齐。其余字段沿用 DbInstInfo。
 *
 * 泛型 `T extends CachedDbInst` 让调用方（DbInst.getOrNewInst / getInst）取回自己的具体类型，
 * 不必在调用点断言；契约里没写的字段一律不可访问，写错字段名会直接编译报错。
 */
export interface CachedDbInst extends DbInstInfo {
    /** 兼容版本，首次建实例后由 getCompatibleDbVersion 异步回填 */
    version?: string;
}

const dbInstCache: Map<number, CachedDbInst> = new Map();

/**
 * 缓存边界上唯一的类型转换点。
 *
 * 缓存按契约类型存放，调用方要的是自己的具体实例类型，二者本就是同一个对象，
 * 故这里是「视图转换」而非「形状猜测」——与改造前散落在各字段的
 * `(dbInst as { tagPath: string }).tagPath` 有本质区别：那种写法编译器无从校验字段是否存在。
 */
function takeCached<T extends CachedDbInst>(dbId: number): T | undefined {
    return dbInstCache.get(dbId) as T | undefined;
}

/**
 * 获取或新建 DbInst 实例：命中缓存则回填标签路径后直接返回，否则交由 createFn 建实例并缓存。
 *
 * @param inst 实例信息（后端列表接口下发）
 * @param createFn 实例工厂，其返回类型即本函数的返回类型
 */
export function getOrNewDbInst<T extends CachedDbInst>(inst: DbInstInfo, createFn: (inst: DbInstInfo) => T): T {
    if (!inst) {
        throw new Error('inst不能为空');
    }
    const cached = takeCached<T>(inst.id);
    if (cached) {
        // 可能同一个库关联多个标签，展示需要
        if (inst.tagPath) {
            cached.tagPath = inst.tagPath;
        }
        return cached;
    }
    const dbInst = createFn(inst);
    // 以契约类型持有引用：对泛型 T 的属性写入会被判为不安全，而缓存要做的正是「按契约回填」
    const cachedView: CachedDbInst = dbInst;
    const firstDb = cachedView.databases?.[0];
    if (firstDb) {
        dbApi.getCompatibleDbVersion.request({ id: inst.id, db: firstDb }).then((version: string) => {
            cachedView.version = version;
        });
    }
    dbInstCache.set(cachedView.id, cachedView);
    return dbInst;
}

/** 获取缓存中的实例；未缓存时为 undefined */
export function getCachedDbInst<T extends CachedDbInst = CachedDbInst>(dbId: number): T | undefined {
    return takeCached<T>(dbId);
}

/** 缓存实例（由 createFn 内部登记，避免调用方拿到实例后忘记入缓存） */
export function cacheDbInst(id: number, inst: CachedDbInst): void {
    dbInstCache.set(id, inst);
}

/** 清空所有实例缓存 */
export function clearAllDbInstCache(): void {
    dbInstCache.clear();
}

// ==================== 表信息缓存 ====================

export function getCachedTables(key: string): DbTableInfo[] | undefined {
    return tableStorage.value.get(key);
}

export function setCachedTables(key: string, tables: DbTableInfo[]): void {
    tableStorage.value.set(key, tables);
}

// ==================== 表提示缓存 ====================

export function getCachedHints(key: string): Record<string, string[]> | undefined {
    return hintsStorage.value.get(key);
}

export function setCachedHints(key: string, hints: Record<string, string[]>): void {
    hintsStorage.value.set(key, hints);
}

// ==================== 库名获取 ====================

export async function getDbNames(db: DbNamesParam): Promise<string[]> {
    if (db.getDatabaseMode == DbGetDbNamesMode.Assign.value) {
        return (db.database as string).split(' ');
    }
    return await dbApi.getDbNamesByAc.request({ authCertName: db.authCertName });
}
