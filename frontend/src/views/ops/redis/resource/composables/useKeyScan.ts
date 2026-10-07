/**
 * Redis 数据操作页的 key 扫描域：SCAN 游标推进、key 摘要与视角描述符的批量拉取。
 *
 * 从 RedisDataOp 拆出的原因：扫描是纯数据获取（发请求、攒 key、算游标），
 * 与树渲染、tab 管理互不依赖；收敛后「切库要重置哪些状态」只需看 resetData，
 * 宿主组件只编排「扫描完成后让树重渲染」这一个会合点。
 */
import { computed, reactive } from 'vue';
import { redisApi } from '../../api';
import { defaultViewOf } from '../../keyview/descriptor';
import type { RedisKeySummary, RedisViewDescriptor } from '../../types';

/** 浏览态（无搜索词）单次扫描的 key 数：小步快跑尽快出首屏，也作为 scanParam.count 初值 */
const DEFAULT_COUNT = 250;
/** 搜索态单次扫描的 key 数：SCAN 只扫 count 个哈希桶、命中率低，放大 count 减少「点了加载更多半天不出数据」 */
const SEARCH_COUNT = 1000;
/** 大库搜索态的 count 上限：再大单次请求过重，封顶于此 */
const SEARCH_COUNT_LARGE_DB = 2000;
/** 库规模超过此阈值视为大库，搜索态用更大的 count */
const LARGE_DB_SIZE = 100000;
/** 集群模式后端遍历所有 master 再合并，按此估算的 master 数摊薄单次 count */
const CLUSTER_MASTER_ESTIMATE = 3;
/** 稀疏首屏自动续扫的最大轮数：本批 0 key 且游标未归零时再扫，凑出可见数据又不至于空转 */
const MAX_EMPTY_RESCAN = 3;

/** 扫描域的状态：显式声明而不是 {} as Record 断言，否则字段会被推导成空对象类型 */
export interface KeyScanState {
    scanning: boolean;
    keys: string[];
    summaries: Record<string, RedisKeySummary>;
    descriptors: RedisViewDescriptor[];
    dbsize: number;
    scanParam: {
        id: number | null;
        mode: string;
        db: number | null;
        match: string;
        count: number;
        cursor: Record<string, number>;
    };
}

/**
 * 计算单次 SCAN 的 count（纯函数，不读 reactive state，便于单测）：
 * - 浏览态（无搜索词）用小 count，尽快出首屏；
 * - 搜索态 SCAN 只扫 count 个哈希桶、命中率低，按库规模放大 count 减少「点了加载更多半天不出数据」，大库封顶 SEARCH_COUNT_LARGE_DB；
 * - 集群模式后端遍历所有 master 再合并，按估算的 master 数摊薄单次 count。
 */
export function resolveScanCount(match: string, dbsize: number, mode: string): number {
    const base = match ? (dbsize > LARGE_DB_SIZE ? SEARCH_COUNT_LARGE_DB : SEARCH_COUNT) : DEFAULT_COUNT;
    return mode === 'cluster' ? Math.floor(base / CLUSTER_MASTER_ESTIMATE) : base;
}

export function useKeyScan() {
    const state = reactive<KeyScanState>({
        scanning: false,
        keys: [],
        summaries: {},
        descriptors: [],
        dbsize: 0,
        scanParam: {
            id: null,
            mode: '',
            db: null,
            match: '',
            count: DEFAULT_COUNT,
            cursor: {},
        },
    });

    const canOperate = computed(() => !!state.scanParam.id && state.scanParam.db != null);

    /** SCAN 游标未全部归零就说明库里还有没扫完的 key（集群/多 master 时每个节点各占一项） */
    const keysHasMore = computed(() => Object.values(state.scanParam.cursor).some((cursor) => Number(cursor) !== 0));

    const summaryOf = (key: string): RedisKeySummary | undefined => state.summaries[key];

    /** key 的默认视角描述符：树节点类型圆点与控制台快捷命令共用这一份（描述符未就绪时为 undefined） */
    const descriptorOf = (key: string): RedisViewDescriptor | undefined => {
        const summary = state.summaries[key];
        return summary ? defaultViewOf(state.descriptors, summary.type) : undefined;
    };

    /**
     * 视角描述符只服务于类型圆点、类型筛选与详情渲染，取不到不该拖死 key 列表：
     * 接口异常时降级成「无描述符」，列表照常扫描（报错提示由请求层统一给出）
     */
    async function loadDescriptors() {
        if (state.descriptors.length || !canOperate.value) {
            return;
        }
        try {
            state.descriptors = (await redisApi.views.request({ id: state.scanParam.id as number, db: state.scanParam.db as number })) ?? [];
        } catch {
            state.descriptors = [];
        }
    }

    /**
     * 扫描一批 key。appendKey=false 表示重新搜索/刷新：必须从头开始扫，
     * 否则会接着上一次的游标继续，新关键词命中不到数据。
     *
     * 稀疏首屏自动续扫：搜索态单批可能一个匹配桶都没扫到（返回 0 key 但游标未归零），
     * 此时自动再扫几轮凑出可见数据，最多 MAX_EMPTY_RESCAN 轮，避免用户对着空列表反复点「加载更多」。
     * @returns 本次扫描新增的原始 key（增量渲染树的依据；appendKey=false 时即本次的全量集）
     */
    async function scan(appendKey = true): Promise<string[]> {
        const match = state.scanParam.match?.trim() ?? '';
        if (!appendKey) {
            state.scanParam.cursor = {};
        }

        const collected: string[] = [];
        state.scanning = true;
        try {
            for (let attempt = 0; attempt <= MAX_EMPTY_RESCAN; attempt++) {
                const res = await redisApi.scan.request({
                    id: state.scanParam.id,
                    db: state.scanParam.db,
                    match,
                    // dbsize 首轮可能未知（为 0），拿到首批响应后即自校正，续扫轮次按真实库规模取 count
                    count: resolveScanCount(match, state.dbsize, state.scanParam.mode),
                    cursor: state.scanParam.cursor,
                });
                collected.push(...(res.keys ?? []));
                state.dbsize = res.dbSize;
                state.scanParam.cursor = res.cursor;
                // 摘要随 SCAN 批次返回，直接并入：树角标随批就位，无需再发独立摘要请求
                (res.summaries ?? []).forEach((item) => {
                    state.summaries[item.key] = item;
                });
                // 拿到数据、或游标已归零（整库扫完）即停；仅当本批空且还有剩余时才自动续扫
                if (collected.length > 0 || !keysHasMore.value) {
                    break;
                }
            }
            state.keys = appendKey ? [...state.keys, ...collected] : collected;
            return collected;
        } finally {
            state.scanning = false;
        }
    }

    /**
     * 登记新操作的实例与库（资源树双击与首次挂载都走这里）
     * @returns 是否真的切换了库——同库重复点击返回 false，宿主据此跳过重置与重扫
     */
    function applyDatabase(dbInfo: Record<string, unknown>): boolean {
        if (state.scanParam.db == dbInfo.db) {
            return false;
        }
        state.scanParam.id = dbInfo.id as number;
        state.scanParam.mode = dbInfo.mode as string;
        state.scanParam.db = dbInfo.db as number;
        return true;
    }

    /** 切库后清空本域全部数据：旧库的 key、摘要与描述符对下一个库毫无意义 */
    function resetData() {
        state.scanParam.match = '';
        state.scanParam.cursor = {};
        state.keys = [];
        state.summaries = {};
        state.descriptors = [];
        state.dbsize = 0;
    }

    return { state, canOperate, keysHasMore, summaryOf, descriptorOf, loadDescriptors, scan, applyDatabase, resetData };
}
