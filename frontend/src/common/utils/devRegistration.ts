/**
 * 开发期「重复注册」自检的聚合上报。
 *
 * 全局注册表（资源树命令 / 节点贡献者 / 资源配置）由模块在顶层求值时产生注册副作用，
 * 命中「键已存在」有两类成因，且无法按时序区分（resource.ts 只被懒加载组件与 glob 触达，
 * 首次注册可能发生在挂载前后的任一时刻）：
 *
 * - 真实冲突：两个不同模块注册同一 key —— 每个 key 各一次，通常条数很少；
 * - Vite HMR 重放：被改动的注册模块重新求值，而注册表仍是上一代内容 —— 表现为
 *   「同一批 key 在短时间内整批重复一次」，正是启动期刷出几十条噪声的来源。
 *
 * 因此这里不丢弃任何信号，而是把同一注册表的一批重复**合并为一条**告警：
 * 逐条明细保留在同一条消息里，并按批量特征提示是否为热更新重放，便于直接判断是否需要处理。
 */
type Pending = { keys: string[]; scheduled: boolean };

const pending = new Map<string, Pending>();

/** 单条告警的批量阈值：达到该数量的整批重复符合 HMR 重放特征 */
const HMR_LIKE_BATCH = 5;

function flush(registry: string, state: Pending) {
    state.scheduled = false;
    if (!state.keys.length) {
        return;
    }
    const keys = state.keys.splice(0, state.keys.length);
    const hint =
        keys.length >= HMR_LIKE_BATCH
            ? '（整批重复，符合 Vite HMR 重放注册模块的特征；若未改动过注册代码请排查是否重复导入）'
            : '';
    console.warn(`[dev-registration] ${registry} 重复注册 ${keys.length} 项，均已覆盖${hint}: ${keys.join(', ')}`);
}

/**
 * 登记一次重复注册。同一 registry 在同一次任务批次内的多次调用合并为一条告警。
 * @param registry 注册表标识，用于告警前缀（如 tree-command / tree-contributor / resource）
 * @param key 被重复注册的键
 */
export function reportDuplicateRegistration(registry: string, key: string) {
    if (!import.meta.env.DEV) {
        return;
    }
    let state = pending.get(registry);
    if (!state) {
        state = { keys: [], scheduled: false };
        pending.set(registry, state);
    }
    state.keys.push(key);
    if (state.scheduled) {
        return;
    }
    state.scheduled = true;
    const current = state;
    queueMicrotask(() => flush(registry, current));
}
