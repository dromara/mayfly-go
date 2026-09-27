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

/** 单次扫描默认取多少 key */
const DEFAULT_COUNT = 250;

/** 单次批量摘要请求的 key 数上限 */
const SUMMARY_BATCH = 300;

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
     * 批量拉取树的类型/过期信息：只请求还没取过的 key，并按 SUMMARY_BATCH 分批发送。
     * 「加载更多」会让 key 列表持续增长，一次管道塞入上万个命令会把连接与浏览器都拖住
     */
    async function loadSummaries(keys: string[]) {
        if (!canOperate.value) {
            return;
        }
        const pending = keys.filter((key) => !state.summaries[key]);
        for (let start = 0; start < pending.length; start += SUMMARY_BATCH) {
            const batch = pending.slice(start, start + SUMMARY_BATCH);
            const summaries = await redisApi.keySummary.request({ id: state.scanParam.id as number, db: state.scanParam.db as number, keys: batch });
            (summaries ?? []).forEach((item) => {
                state.summaries[item.key] = item;
            });
        }
    }

    /**
     * 模糊搜索时提高 scan count：SCAN 每次只扫描 count 个哈希桶，count 太小会让
     * 匹配结果稀疏到「点了加载更多半天不出数据」
     */
    function resolveScanCount(match: string): number {
        let count = DEFAULT_COUNT;
        if (match.includes('*') && match.length > 10) {
            count = state.dbsize > 100000 ? Math.floor(state.dbsize / 10) : 1000;
        }
        // 集群模式下后端会遍历所有 master 再合并，按 3 个 master 估算摊薄单次 count
        return state.scanParam.mode === 'cluster' ? Math.floor(count / 3) : count;
    }

    /**
     * 扫描一批 key。appendKey=false 表示重新搜索/刷新：必须从头开始扫，
     * 否则会接着上一次的游标继续，新关键词命中不到数据
     */
    async function scan(appendKey = true) {
        const match = state.scanParam.match?.trim() ?? '';
        if (!appendKey) {
            state.scanParam.cursor = {};
        }

        state.scanning = true;
        try {
            const res = await redisApi.scan.request({
                id: state.scanParam.id,
                db: state.scanParam.db,
                match,
                count: resolveScanCount(match),
                cursor: state.scanParam.cursor,
            });
            state.keys = appendKey ? [...state.keys, ...(res.keys ?? [])] : (res.keys ?? []);
            state.dbsize = res.dbSize;
            state.scanParam.cursor = res.cursor;
            await loadSummaries(state.keys);
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
