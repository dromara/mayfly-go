/**
 * 同步任务 Extra 字段的类型化访问器。
 *
 * 后端 entity.DataSyncTask 把 cursorInclusivity / sleepBetweenBatchesMs / skipIndexValidation
 * 等非查询维度配置存到 Extra（json 列，无 SQL 索引），HTTP 响应体里以 `extra` 顶层字段回传。
 *
 * 本模块集中 key 名，避免：
 * - 后端换 key 时前端散落各处的 `extra.foo` 静默读到 undefined 后回落默认，导致"配了但无效果"
 * - 前后端 key 拼写分叉无编译期告警
 *
 * key 需与服务端 entity.ExtraKeyCursorInclusivity 等常量保持字面一致，改名时双向同步。
 */
export const TASK_EXTRA_KEYS = {
    cursorInclusivity: 'cursorInclusivity',
    sleepBetweenBatchesMs: 'sleepBetweenBatchesMs',
    skipIndexValidation: 'skipIndexValidation',
} as const;

type TaskExtra = Record<string, unknown> | undefined | null;

/** 读取数值型 extra，缺失/非数值时回 fallback。JSON 反序列化后 number 已是 number，无需强转 */
export function readExtraNumber(extra: TaskExtra, key: string, fallback = 0): number {
    const v = extra?.[key];
    return typeof v === 'number' && Number.isFinite(v) ? v : fallback;
}

/** 读取布尔型 extra，只有显式 true 才返 true（保持与后端 setter "非 true 即清 key" 的语义对称） */
export function readExtraBool(extra: TaskExtra, key: string): boolean {
    return extra?.[key] === true;
}
