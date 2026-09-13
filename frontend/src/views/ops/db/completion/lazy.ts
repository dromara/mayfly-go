/**
 * SQL 联想的惰性入口 + 使用方作用域
 *
 * 补全实现依赖编辑器主体（约 3.9M / gz 约 1M），而它的调用方都是「进页面即挂载」的表单与标签页容器
 * （查询标签页、同步任务表单、SQL 执行审批表单）——用户打开这些页面时往往只是看表数据、填表单。
 * 静态 import `./index` 会把整份编辑器拖进它们的首屏，故统一经本模块在「真正注册联想的那一刻」才动态取用。
 *
 * 约束：本模块禁止引入 monaco 与 `./index`（类型导入除外），否则惰性失效。
 * 该约束由 ../../__tests__/monacoBoundary.test.ts 锁住。
 *
 * ## 为什么需要「使用方作用域」而不是一对准 register/dispose 函数
 *
 * monaco 的补全注册表按语言全局共享（一个 language 同时只有一个 provider 生效），provider 闭包持有
 * 注册时刻的库上下文。本站又用 keep-alive + 多标签页，于是数据库标签页、同步任务抽屉、SQL 执行审批
 * 表单可能同时存活且各自持有不同库上下文，因此：
 * - 任一使用方卸载不能无条件注销，否则其余使用方的联想一起失效，而它们只在挂载/选库时注册过一次，
 *   不会自愈；
 * - 注销后还要把 provider 交回**上一个仍在存活的注册者**，否则 provider 会继续停留在已卸载使用方的
 *   库上下文上；
 * - 被别的页面抢走过 provider 的使用方，回到前台时须重新申领（见 SqlCompletionScope#refresh）。
 *
 * 这里按「注册顺序」维护使用方列表而非做 model 级路由：补全上下文的归属单位是使用方（一个页面/表单），
 * 而 monaco 只在注册期接受闭包，改成 model 级分发需要每个编辑器暴露自己的 model uri，代价与收益不匹配。
 */
let modulePromise: Promise<typeof import('./index')> | null = null;

/** 使用方的联想上下文，与 registerDbCompletionItemProvider 的入参一一对应 */
type CompletionParams = {
    dbId: number;
    db: string;
    dbs: string[] | undefined;
    dbType: string;
};

/** 存活的使用方，按申领先后排列，末位即当前持有全局 provider 者 */
const owners: CompletionParams[] = [];

/**
 * 取用补全实现模块。
 *
 * 加载失败（弱网、发版后旧页面残留）不缓存 rejection：缓存下来就让本会话永久失去联想，
 * 而重试的代价只是「用户再次切 tab 时多一次请求 + 一条告警」。
 * 发版后旧页面的 chunk 404 属于必须刷新页面的故障，不在此处做退避。
 */
function loadCompletion() {
    if (!modulePromise) {
        modulePromise = import('./index').catch((e: unknown) => {
            modulePromise = null;
            throw e;
        });
    }
    return modulePromise;
}

/** 申领全局 provider：把 provider 闭包指向本使用方的上下文 */
function claim(params: CompletionParams) {
    // 同一时刻发起的多次申领挂在同一 promise 上，按附加顺序执行，故最后一次生效（与原同步语义一致）
    loadCompletion()
        .then(({ registerDbCompletionItemProvider }) => registerDbCompletionItemProvider(params.dbId, params.db, params.dbs, params.dbType))
        .catch((e: unknown) => console.error('[db] load sql completion failed:', e));
}

/** SQL 联想使用方作用域：一个使用方（页面/表单组件）对应一个实例 */
export type SqlCompletionScope = {
    /**
     * 注册或更新本使用方的联想上下文，并使其成为当前生效者；可反复调用（如切标签页）。
     *
     * @param dbs 实例下的库列表，用于 `.` 触发时的库名联想；缺省表示不做库名联想
     * @param dbType 数据库类型，决定方言（关键字、切分符、引用符）
     */
    register(dbId: number, db: string, dbs: string[] | undefined, dbType: string): void;
    /**
     * 重新申领：以上次注册的上下文把 provider 指回本使用方，未注册过则为空操作。
     *
     * 用于 keep-alive 页面回到前台（onActivated）——期间 provider 可能已被其他页面申领。
     */
    refresh(): void;
    /** 释放本使用方；交回上一个存活的使用方，无人存活时才注销全局 provider */
    release(): void;
};

export function createSqlCompletionScope(): SqlCompletionScope {
    // 本作用域在 owners 中的槽位；register 首次调用时入列，release 时出列
    let slot: CompletionParams | null = null;

    /** 把本作用域的槽位移到末位（成为当前生效者） */
    function promote(active: CompletionParams) {
        const index = owners.indexOf(active);
        if (index === owners.length - 1) {
            return;
        }
        owners.splice(index, 1);
        owners.push(active);
    }

    return {
        register(dbId: number, db: string, dbs: string[] | undefined, dbType: string) {
            if (slot) {
                Object.assign(slot, { dbId, db, dbs, dbType });
            } else {
                slot = { dbId, db, dbs, dbType };
                owners.push(slot);
            }
            promote(slot);
            claim(slot);
        },
        refresh() {
            if (!slot) {
                return;
            }
            promote(slot);
            claim(slot);
        },
        release() {
            if (!slot) {
                return;
            }
            // 只有本使用方正在持有 provider 时，退出才需要交接
            const wasActive = slot === owners[owners.length - 1];
            owners.splice(owners.indexOf(slot), 1);
            slot = null;
            // modulePromise 为空说明编辑器从未加载，注册表里不会有本语言 provider
            if (!modulePromise) {
                return;
            }
            const previous = owners[owners.length - 1];
            if (previous) {
                if (wasActive) {
                    claim(previous);
                }
                return;
            }
            // 清理路径不外抛：编辑器模块加载失败时本就无可注销的东西
            modulePromise
                .then(({ disposeDbCompletionItemProvider }) => disposeDbCompletionItemProvider())
                .catch(() => undefined);
        },
    };
}
