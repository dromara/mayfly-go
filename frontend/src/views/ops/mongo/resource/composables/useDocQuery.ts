/**
 * 文档查询域：集合 tab 的查询状态与执行。
 *
 * 每个集合一个 tab，条件、页码、结果与统计都挂在 tab 上，切换 tab 不丢上下文；
 * 组件只消费这里暴露的状态与方法，不直接碰 API，也不自己拼请求参数。
 *
 * 条件以 JSON 文本承载而不是对象：文本能原样保留字段顺序（复合排序键的顺序是语义的一部分）
 * 与 `$oid`/`$date` 类型包装，交给后端解析即零猜测。
 */
import { computed, reactive } from 'vue';

import { i18n } from '@/i18n';
import { mongoApi } from '../../api';
import type { MongoCollectionStats, MongoDoc, MongoDocPage } from '../../types';
import { parseDocText } from '../../docview/extjson';

/** 与后端 config.Mongo.DefaultLimit 同源的兜底值（后端也会把非法 limit 归一到默认值） */
export const DEFAULT_LIMIT = 20;

/** 每页条数候选，上限不超过后端默认 MaxResultSet，避免选了就被静默截断 */
export const LIMIT_OPTIONS = [10, 20, 50, 100];

/** 未统计匹配总数时的占位值，与后端 dto.DocPage.Total 的 -1 语义一致 */
export const TOTAL_UNKNOWN = -1;

export interface DocQueryTab {
    key: string;
    label: string;
    mongoId: number;
    database: string;
    collection: string;
    /** 视图（view）不可写：树节点带进来后写入口就地收敛，而不是点了报错才知道 */
    readOnly: boolean;

    filterText: string;
    sortText: string;
    projectionText: string;
    limit: number;
    page: number;
    withCount: boolean;

    docs: MongoDoc[];
    total: number;
    truncated: boolean;
    effectiveLimit: number;
    loading: boolean;
    /** 查询失败的原因（与「没有匹配文档」区分开，前者要能看见） */
    error: string;
    /** 是否成功执行过至少一次查询 */
    loaded: boolean;
    stats?: MongoCollectionStats;
    statsError?: string;
}

/** 条件文本的解析结果 */
interface ParsedCondition {
    /** 校验不通过时面向用户的说明（已翻译，可直接显示） */
    error: string;
    filter?: unknown;
    sort?: unknown;
    projection?: unknown;
}

const CONDITION_FIELDS: { tabKey: keyof DocQueryTab; name: string }[] = [
    { tabKey: 'filterText', name: 'filter' },
    { tabKey: 'sortText', name: 'sort' },
    { tabKey: 'projectionText', name: 'projection' },
];

/** 查询域的状态形状（给 reactive 显式标注而不是内联断言：断言只能骗过编译器，推导仍会把 tabs 当成 {}） */
interface DocQueryState {
    activeKey: string;
    tabs: Record<string, DocQueryTab>;
}

export function useDocQuery() {
    const state = reactive<DocQueryState>({
        activeKey: '',
        tabs: {},
    });

    // 请求代际号：同一 tab 上连续触发查询时只认最后一次的结果，
    // 否则慢响应会在快响应之后落地，界面显示的是一个已被取代的条件组合
    const generations = new Map<string, number>();

    const activeTab = computed<DocQueryTab | undefined>(() => state.tabs[state.activeKey]);
    const tabKeys = computed(() => Object.keys(state.tabs));

    /** 打开集合 tab（已存在则激活），返回该 tab 以便调用方决定是否立即查询 */
    function openTab(mongoId: number, database: string, collection: string, readOnly = false): DocQueryTab {
        const key = `${database}.${collection}`;
        const existed = state.tabs[key];
        if (existed) {
            // 树上是权威来源：集合被改成视图（或反之）后重新进入要跟着更新，否则该关的写入口没关
            existed.readOnly = readOnly;
            state.activeKey = key;
            return existed;
        }

        const tab: DocQueryTab = {
            key,
            label: key,
            mongoId,
            database,
            collection,
            readOnly,
            filterText: '{}',
            sortText: '',
            projectionText: '',
            limit: DEFAULT_LIMIT,
            page: 1,
            withCount: false,
            docs: [],
            total: TOTAL_UNKNOWN,
            truncated: false,
            effectiveLimit: DEFAULT_LIMIT,
            loading: false,
            error: '',
            loaded: false,
        };
        state.tabs[key] = tab;
        state.activeKey = key;
        return tab;
    }

    /** 关闭 tab，并把激活态移到相邻 tab（与全站资源操作 tab 的邻居策略一致） */
    function closeTab(targetKey: string) {
        const keys = tabKeys.value;
        if (state.activeKey === targetKey) {
            const index = keys.indexOf(targetKey);
            const neighbor = keys[index + 1] ?? keys[index - 1];
            state.activeKey = neighbor ?? '';
        }
        delete state.tabs[targetKey];
    }

    /** 解析三个条件字段；文本非法时返回可直接展示的说明，不发请求 */
    function parseCondition(tab: DocQueryTab): ParsedCondition {
        const parsed: ParsedCondition = { error: '' };

        for (const field of CONDITION_FIELDS) {
            const text = String(tab[field.tabKey] ?? '');
            if (!text.trim()) {
                continue;
            }
            const value = parseDocText(text);
            // 条件必须是对象：数组与标量会被后端解成非法 BSON 文档，在这里拦下能指明是哪个字段
            if (value === null || typeof value !== 'object' || Array.isArray(value)) {
                return { error: i18n.global.t('mongo.conditionInvalid', { field: field.name }) };
            }
            if (field.name === 'filter') {
                parsed.filter = value;
            } else if (field.name === 'sort') {
                parsed.sort = value;
            } else {
                parsed.projection = value;
            }
        }

        return parsed;
    }

    /**
     * 执行查询。
     *
     * 返回是否成功，调用方据此决定是否提示「无匹配文档」而不是把失败当空结果。
     */
    async function runQuery(tabKey?: string): Promise<boolean> {
        const tab = state.tabs[tabKey ?? state.activeKey];
        if (!tab) {
            return false;
        }

        const condition = parseCondition(tab);
        if (condition.error) {
            tab.error = condition.error;
            return false;
        }

        const generation = (generations.get(tab.key) ?? 0) + 1;
        generations.set(tab.key, generation);
        tab.loading = true;
        tab.error = '';

        try {
            const page = await mongoApi.queryDocs.request({
                id: tab.mongoId,
                database: tab.database,
                collection: tab.collection,
                filter: condition.filter,
                sort: condition.sort,
                projection: condition.projection,
                skip: (tab.page - 1) * tab.limit,
                limit: tab.limit,
                withCount: tab.withCount,
            });
            // 已被后续查询取代的响应直接丢弃
            if (generations.get(tab.key) !== generation) {
                return false;
            }
            applyPage(tab, page);
            return true;
        } catch (e) {
            if (generations.get(tab.key) !== generation) {
                return false;
            }
            tab.docs = [];
            tab.total = TOTAL_UNKNOWN;
            tab.truncated = false;
            tab.loaded = false;
            // 错误文案已由请求层统一 toast，这里只留一份就地展示（清空态需要说明为什么是空的）
            tab.error = e instanceof Error ? e.message : '';
            return false;
        } finally {
            if (generations.get(tab.key) === generation) {
                tab.loading = false;
            }
        }
    }

    function applyPage(tab: DocQueryTab, page: MongoDocPage) {
        tab.docs = page.docs ?? [];
        tab.total = page.total ?? TOTAL_UNKNOWN;
        tab.truncated = Boolean(page.truncated);
        tab.effectiveLimit = page.limit ?? tab.limit;
        tab.loaded = true;
        // 统计取不到时后端只回原因，不阻断查询：头部读数据此显示占位并在 tooltip 里说明
        tab.stats = page.stats;
        tab.statsError = page.statsError;
    }

    /** 翻页：skip 由页码与每页条数推出，越界由后端上限截断并在读数区提示 */
    function goPage(delta: number, tabKey?: string) {
        const tab = state.tabs[tabKey ?? state.activeKey];
        if (!tab) {
            return;
        }
        const next = tab.page + delta;
        if (next < 1) {
            return;
        }
        tab.page = next;
    }

    /** 改每页条数后回到第一页：继续停在原页码会因 skip 变大而显示不相关内容 */
    function changeLimit(limit: number, tabKey?: string) {
        const tab = state.tabs[tabKey ?? state.activeKey];
        if (!tab) {
            return;
        }
        tab.limit = limit;
        tab.page = 1;
    }

    function setWithCount(withCount: boolean, tabKey?: string) {
        const tab = state.tabs[tabKey ?? state.activeKey];
        if (tab) {
            tab.withCount = withCount;
        }
    }

    /**
     * 改条件并回到第一页。
     *
     * 只传要改的那几项，未传的保持原值：单条 chip 的「只去掉排序」就是这种改法。
     * 条件变了还停在第 3 页，新条件下很可能没有那么多条，回第一页才符合预期。
     */
    function setCondition(patch: Partial<Pick<DocQueryTab, 'filterText' | 'sortText' | 'projectionText'>>, tabKey?: string) {
        const tab = state.tabs[tabKey ?? state.activeKey];
        if (!tab) {
            return;
        }
        Object.assign(tab, patch);
        tab.page = 1;
    }

    /** 该 tab 是否有可用于写回的主键令牌（投影排除 _id 的文档只能看不能改） */
    function locatable(doc: MongoDoc): boolean {
        return Boolean(doc.idToken);
    }

    return {
        state,
        activeTab,
        tabKeys,
        openTab,
        closeTab,
        runQuery,
        goPage,
        changeLimit,
        setWithCount,
        setCondition,
        locatable,
    };
}
