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
 * monaco 的补全注册表按语言全局共享（一个 language 同时只有一个 provider 生效）。
 * 本站用 keep-alive + 多标签页，数据库标签页、同步任务抽屉、SQL 执行审批表单可能同时存活
 * 且各自持有不同库上下文。若 provider 闭包只记最后一次注册者，用户在 A 库编辑器里触发补全
 * 会拿到 B 库的提示（跨标签页/跨页面串扰）。
 *
 * 因此 provider 只注册一次，内部按 editorUri（每个编辑器实例的唯一标识）查 scopeByEditor
 * 路由到正确的库上下文；各使用方在 register 时把自己的 editorUri 登记到本表，provider
 * 触发时按 URI 命中；未命中则回落 owners 末位（兼容未上报 editorUri 的场景）。
 *
 * ## 为什么 provider 一旦安装就不再注销
 *
 * 若使用方全部释放时异步 dispose，会与「加载中的 claim」形成竞态：dispose 前一次
 * 迟到的 install 回调把 claimed 置 true 却又被 dispose，后续任何 claim 因 claimed=true
 * 直接短路，导致同一会话内永久失去联想。SQL 补全 provider 对未登记的编辑器返回 null
 * 与「provider 被 dispose」的行为等价（该编辑器无补全），故选择安装即常驻以彻底消除竞态。
 */
let modulePromise: Promise<typeof import('./index')> | null = null;

/** 使用方的联想上下文，与 registerDbCompletionItemProvider 的入参一一对应 */
type CompletionParams = {
    dbId: number;
    db: string;
    dbs: string[] | undefined;
    dbType: string;
    /** 编辑器实例唯一标识（monaco model.uri.toString()），多编辑器共存时按编辑器路由补全上下文 */
    editorUri?: string;
};

/** 存活的使用方，按申领先后排列，末位即未上报 editorUri 时的回落目标 */
const owners: CompletionParams[] = [];

/**
 * 编辑器 URI → 补全作用域（CompletionParams）映射表。
 * provider 按 editorUri 查表取参数，使多编辑器共存时各走各的库上下文，不再互相覆盖。
 * 与 owners（作用域列表，按申领先后排列）互补：owners 提供回落目标，本表提供精确路由。
 */
const scopeByEditor: Map<string, CompletionParams> = new Map();

/** provider 是否已安装；一旦安装即常驻，不再回到 false（见文件头「安装即常驻」说明） */
let providerInstalled = false;
/** 并发 claim 去重：多个使用方同时首启时只发一次 install */
let providerInstalling = false;

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

/**
 * 确保全局 provider 已安装：首次调用触发异步注册，后续调用为 no-op。
 * 未命中的编辑器由 resolver 返回 null，provider 直接放弃产出建议。
 */
function ensureProvider() {
    if (providerInstalled || providerInstalling) {
        return;
    }
    providerInstalling = true;
    loadCompletion()
        .then(({ registerDbCompletionItemProvider }) => {
            // provider 每次触发时按 editorUri 查 scopeByEditor 取对应库上下文；未登记时回落 owners 末位
            registerDbCompletionItemProvider((editorUri: string) => scopeByEditor.get(editorUri) || owners[owners.length - 1] || null);
            providerInstalled = true;
        })
        .catch((e: unknown) => {
            console.error('[db] load sql completion failed:', e);
            // 加载失败允许下次 claim 重试；成功安装即常驻
            providerInstalling = false;
        });
}

/** SQL 联想使用方作用域：一个使用方（页面/表单组件）对应一个实例 */
export type SqlCompletionScope = {
    /**
     * 注册或更新本使用方的联想上下文，并使其成为当前生效者；可反复调用（如切标签页）。
     *
     * @param dbs 实例下的库列表，用于 `.` 触发时的库名联想；缺省表示不做库名联想
     * @param dbType 数据库类型，决定方言（关键字、切分符、引用符）
     * @param editorUri 编辑器实例唯一标识（monaco model.uri.toString()），多编辑器共存时按编辑器路由补全上下文
     */
    register(dbId: number, db: string, dbs: string[] | undefined, dbType: string, editorUri?: string): void;
    /**
     * 提升本作用域在 owners 中的位置到末位（回落目标）。
     *
     * 用于 keep-alive 页面回到前台（onActivated）——期间其他页面可能已注册，
     * 回落目标需要指回本作用域。
     */
    refresh(): void;
    /** 释放本作用域：清理登记的 editorUri 映射并移出 owners；provider 本身保持安装状态 */
    release(): void;
    /**
     * 补登编辑器实例标识：编辑器 @ready 后才能拿到 URI，此时 register 已调用过，
     * 需单独补登使 provider 能按编辑器路由到本作用域。
     */
    setEditorUri(editorUri: string): void;
};

export function createSqlCompletionScope(): SqlCompletionScope {
    // 本作用域在 owners 中的槽位；register 首次调用时入列，release 时出列
    let slot: CompletionParams | null = null;

    /** 把本作用域的槽位移到末位（成为回落时的当前生效者） */
    function promote(active: CompletionParams) {
        const index = owners.indexOf(active);
        if (index === owners.length - 1) {
            return;
        }
        owners.splice(index, 1);
        owners.push(active);
    }

    return {
        register(dbId: number, db: string, dbs: string[] | undefined, dbType: string, editorUri?: string) {
            if (slot) {
                Object.assign(slot, { dbId, db, dbs, dbType, editorUri: editorUri ?? slot.editorUri });
            } else {
                slot = { dbId, db, dbs, dbType, editorUri };
                owners.push(slot);
            }
            if (slot.editorUri) {
                scopeByEditor.set(slot.editorUri, slot);
            }
            promote(slot);
            ensureProvider();
        },
        refresh() {
            if (!slot) {
                return;
            }
            promote(slot);
            ensureProvider();
        },
        setEditorUri(editorUri: string) {
            if (!slot) {
                return;
            }
            // 清理旧 editorUri 映射条目（同一 slot 的编辑器标识通常不变，但防御性清理避免残留）
            if (slot.editorUri && slot.editorUri !== editorUri) {
                scopeByEditor.delete(slot.editorUri);
            }
            slot.editorUri = editorUri;
            scopeByEditor.set(editorUri, slot);
            ensureProvider();
        },
        release() {
            if (!slot) {
                return;
            }
            // 清理本作用域登记的 editorUri
            if (slot.editorUri) {
                scopeByEditor.delete(slot.editorUri);
            }
            owners.splice(owners.indexOf(slot), 1);
            slot = null;
            // 不注销 provider：resolver 对未登记编辑器返回 null 与「dispose」行为等价，
            // 且避免「异步 install 未落定 + 立即 dispose」的状态分歧导致后续 claim 永久失效
        },
    };
}
