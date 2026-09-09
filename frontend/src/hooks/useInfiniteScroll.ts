/**
 * useInfiniteScroll - 无限滚动分页 Hook
 *
 * 基于 usePageTable 的分页理念，但采用累加模式（追加而非替换），
 * 配合 IntersectionObserver（InfiniteCardList 内哨兵）实现滚动到底部自动加载更多。
 *
 * 适用于卡片列表、瀑布流等非表格场景。
 */
import { computed, ref, type Ref } from 'vue';

/** 无限滚动 fetcher 参数 */
export interface InfiniteScrollParams<TSearch> {
    page: number;
    pageSize: number;
    search: TSearch;
}

/** fetcher 返回结构（对齐后端 model.PageResult：{total, list}） */
export interface InfiniteScrollResult<TItem> {
    total: number;
    list: TItem[];
}

interface UseInfiniteScrollOptions<TItem, TSearch> {
    /** 数据加载函数 */
    fetcher: (params: InfiniteScrollParams<TSearch>) => Promise<InfiniteScrollResult<TItem>>;
    /** 搜索默认值 */
    defaultSearch?: TSearch;
    /** 每页条数，默认 12（卡片网格 3~4 列合适） */
    pageSize?: number;
}

/**
 * 无限滚动分页状态机
 *
 * - 首次加载 / 搜索 / 重置：page 归位 1 并替换 items
 * - loadMore：追加下一页，seq 竞态守卫避免慢响应串页
 */
export function useInfiniteScroll<TItem, TSearch extends Record<string, any>>(
    options: UseInfiniteScrollOptions<TItem, TSearch>,
) {
    const { fetcher, defaultSearch, pageSize = 12 } = options;

    const items = ref<TItem[]>([]) as Ref<TItem[]>;
    const total = ref(0);
    const loading = ref(false);
    const loadingMore = ref(false);
    const hasMore = ref(true);
    const searchValues = ref<TSearch>({ ...(defaultSearch ?? ({} as TSearch)) });

    // 页码与请求序号（竞态守卫：仅最新一次请求允许写回状态）
    let page = 1;
    let seq = 0;

    /** 首次加载 / 搜索 / 重置 / 手动重载：page 归位 1、替换 items */
    const reload = async () => {
        const current = ++seq;
        loading.value = true;
        page = 1;
        try {
            const res = await fetcher({ page: 1, pageSize, search: { ...searchValues.value } });
            if (current !== seq) return;
            items.value = res.list ?? [];
            total.value = res.total ?? 0;
            hasMore.value = items.value.length < total.value;
        } catch {
            // 失败保持现状（请求层已 toast）
        } finally {
            if (current === seq) loading.value = false;
        }
    };

    /** 加载下一页并追加（防重入） */
    const loadMore = async () => {
        if (loadingMore.value || loading.value || !hasMore.value) return;
        const current = ++seq;
        loadingMore.value = true;
        try {
            const res = await fetcher({ page: page + 1, pageSize, search: { ...searchValues.value } });
            if (current !== seq) return;
            const newItems = res.list ?? [];
            items.value = [...items.value, ...newItems];
            page += 1;
            total.value = res.total ?? 0;
            hasMore.value = items.value.length < total.value;
        } catch {
            // 失败保持现状（请求层已 toast）
        } finally {
            // loadingMore 仅有重入守卫、无并发写者，必须无条件复位；
            // 若沿用 seq 守卫，被 reload 抢占后将永久卡 true，滚动分页从此失效
            loadingMore.value = false;
        }
    };

    /** 执行搜索（重置到第一页） */
    const onSearch = () => reload();

    /** 重置搜索：恢复 defaultSearch 并重新加载 */
    const onReset = () => {
        searchValues.value = { ...(defaultSearch ?? ({} as TSearch)) };
        reload();
    };

    /**
     * 直接展开传给 InfiniteCardList 的 props，
     * 消除页面侧重复的逐个 props 绑定（bindCardList）
     */
    const bindCardList = computed(() => ({
        items: items.value,
        loading: loading.value,
        loadingMore: loadingMore.value,
        hasMore: hasMore.value,
        loadMore,
        searchValues: searchValues.value,
        onSearchChange: (v: Record<string, any>) => (searchValues.value = v as TSearch),
        onSearch,
        onReset,
    }));

    return {
        items,
        total,
        loading,
        loadingMore,
        hasMore,
        searchValues,
        reload,
        onSearch,
        onReset,
        loadMore,
        bindCardList,
    };
}
