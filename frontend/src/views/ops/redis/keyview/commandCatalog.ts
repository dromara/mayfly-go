import { redisApi } from '../api';
import type { RedisCommandSpec } from '../types';

/**
 * 实例命令目录的共享缓存：目录由实例自描述且与库无关，重开控制台、切库、打开多个
 * 库面板都应命中同一份，而不是每次重拉几百条命令。
 *
 * 缓存必须放在模块作用域：`<script setup>` 里的变量是每个组件实例一份，tab 关闭重开就丢了；
 * 同一实例的在途请求共享同一个 Promise，避免并发双拉；拉取失败不写缓存（提示只是增强能力，
 * 下次打开还能重试），不可逆动作由 console.ts 的兜底确认名单挡住
 */
const catalogs = new Map<number, RedisCommandSpec[]>();
const inFlight = new Map<number, Promise<RedisCommandSpec[]>>();

/** 目录已就绪时直接取用（同步），供组件初始化时回填、避免首帧空提示 */
export function cachedCommandCatalog(id: number): RedisCommandSpec[] | undefined {
    return catalogs.get(id);
}

export function loadCommandCatalog(id: number, db: number): Promise<RedisCommandSpec[]> {
    const hit = catalogs.get(id);
    if (hit) {
        return Promise.resolve(hit);
    }
    const pending = inFlight.get(id);
    if (pending) {
        return pending;
    }

    const request = redisApi.commands.request({ id, db }).then((specs) => {
        const list = specs ?? [];
        catalogs.set(id, list);
        return list;
    });
    inFlight.set(id, request);
    // 无论成败都清掉在途标记：失败时下次调用重新发起（重试），成功时已落入 catalogs
    return request.finally(() => inFlight.delete(id));
}
