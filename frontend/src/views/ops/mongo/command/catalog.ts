/**
 * 命令目录缓存。
 *
 * 缓存必须挂在模块作用域而不是组件实例上：控制台弹窗会在同一实例上反复开关，
 * 放在 setup 顶层等于「每次打开都重新拉取、关掉就丢」，还让同实例的并发打开各自发一跳。
 *
 * 目录的唯一真源是后端（级别、要求的权限码、是否需要确认、模板都在那里），
 * 前端不再另立一份命令名单，否则鉴权口径与确认口径迟早分叉。
 */
import { mongoApi } from '../api';
import type { MongoCommandSpec } from '../types';

const catalogCache = new Map<number, MongoCommandSpec[]>();
const inflight = new Map<number, Promise<MongoCommandSpec[]>>();

export async function loadCommandCatalog(id: number): Promise<MongoCommandSpec[]> {
    const cached = catalogCache.get(id);
    if (cached) {
        return cached;
    }

    // 同一实例的并发请求合流，避免多次打开弹窗打出多跳相同请求
    const pending = inflight.get(id);
    if (pending) {
        return pending;
    }

    const request = mongoApi.commands
        .request({ id })
        .then((specs) => {
            // 只在成功时写缓存，失败保留可重试的语义
            const list = specs ?? [];
            catalogCache.set(id, list);
            return list;
        })
        .finally(() => {
            inflight.delete(id);
        });

    inflight.set(id, request);
    return request;
}

/** 清缓存：实例配置变更后需要重新拉取，测试也用它隔离状态 */
export function clearCommandCatalog(id?: number) {
    if (id === undefined) {
        catalogCache.clear();
        return;
    }
    catalogCache.delete(id);
}
