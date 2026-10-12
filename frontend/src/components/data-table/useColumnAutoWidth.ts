import { getCurrentInstance, nextTick, onMounted, onUnmounted, ref, toValue, watch, type MaybeRefOrGetter, type Ref } from 'vue';
import type { TableColumn } from './table-column';
import { refineColumnWidthsByDom } from './columnWidthMeasure';

interface ColumnAutoWidthOptions {
    /**
     * 表格包裹层 DOM：单元格内容实测宽的取量入口。
     * 结构最小接口（内部只读 .value），兼容 useTemplateRef 返回的 Readonly<ShallowRef<HTMLElement | null>>
     */
    tableWrapRef: { readonly value: HTMLElement | null | undefined };
    /** 列配置（getter 形式，跟随 props 响应式来源） */
    columns: () => TableColumn[];
    /** 表格数据 */
    tableData: MaybeRefOrGetter<Record<string, unknown>[]>;
    /** 是否渲染了 selection 列（影响回填时的 td 列序偏移） */
    showSelection: () => boolean;
    /**
     * 数据加载中标记（有 pageApi 的异步列表）：用于区分「首屏/加载中的假空」与「确认为空」。
     * 加载中的空数据不应提前显示表格（否则先画表头窄宽、数据到后再撑宽 → 列宽跳变）。
     */
    loading?: MaybeRefOrGetter<boolean>;
    /** 该表格数据是否由 api 异步加载：true 时空态需等首次加载结束再显示；false（静态 :data）直接显示 */
    asyncLoad?: () => boolean;
    /**
     * 首轮估算完成回调：minWidth 基线已写入列配置，此时才可安全显示表格
     * （先渲染后计宽会出现列宽跳变的闪烁）。
     */
    onEstimated?: () => void;
}

/**
 * 列宽自适应：估算打底 → DOM 实测回填 → 首轮定型。
 *
 * 1. 估算打底：首轮非空数据到达时，按各列 measureContentWidth（默认为格式化后的展示文本）
 *    离线测宽写入 minWidth 基线；此阶段表格尚未渲染，量不到插槽内图标/tag 等渲染占位；
 * 2. DOM 实测回填：表格渲染可见后遍历单元格，只把「真实溢出」的内容宽向上回填 minWidth
 *    （scrollWidth ≥ clientWidth 是 DOM 恒等式，只量溢出才不会把列宽自身当内容需求宽，造成单调膨胀）；
 * 3. 首轮定型：回填完成后冻结，此后翻页/刷新不再变动，保证分页浏览时布局稳定不跳动。
 *
 * 扩展性：新增特殊渲染形态的列只需在 TableColumn 上提供自己的 measureContentWidth，
 * 本 composable 与通用算法无需改动（开闭原则）。
 */
export function useColumnAutoWidth(options: ColumnAutoWidthOptions) {
    /** 列宽是否已按首轮数据定型：定型后翻页/刷新不再重算 */
    const columnWidthSettled: Ref<boolean> = ref(false);

    /** 是否已完成过至少一次数据加载（loading 由 true→false 至少一次）：区分「假空（数据未到）」与「确认为空」 */
    let hasLoadedOnce = false;

    /**
     * 渲染后按真实 DOM 回填列宽：把「怎么量」委托给 columnWidthMeasure（纯测量），本 composable 只负责
     * 在正确时机（可见、渲染后）把 wrap/列配置/selection 偏移喂给它。隐藏（display:none）时量到 0，列侧自动跳过。
     */
    const refineFromDom = () => {
        const wrap = options.tableWrapRef.value;
        if (wrap) {
            refineColumnWidthsByDom(wrap, toValue(options.columns), options.showSelection());
        }
    };

    /** 计算各列 minWidth 估算基线（离线文本测宽，渲染前执行） */
    const calculateTableColumnMinWidth = () => {
        const data = toValue(options.tableData);
        if (!data || data.length === 0) {
            return;
        }
        toValue(options.columns).forEach((item) => {
            // 隐藏列也参与离线估算（离屏测宽与 DOM 可见性无关）：否则列配置勾选启用时 minWidth 仍为
            // undefined，el-table 回落 80px 把长内容截断
            if (item.autoWidth) {
                item.autoCalculateMinWidth(data);
            }
        });
    };

    /**
     * 声明了「可见弹性列」时的紧凑固化：其余数值 minWidth 列把 minWidth 固化为 width，
     * 退出 el-table fit 的容器富余摊派，列宽贴内容；富余集中到弹性列（保留 min-width）。
     * 无可见弹性列时保持全员 min-width 均摊的既有默认，页面零回归。
     * 必须在估算后、表格渲染前调用：若等渲染后再固化，摊派宽会遮掩估算欠宽列的溢出，
     * 回填量不到真实需求。
     */
    // 已被本机制固化的列（width 由 compact 写入）：弹性列全部隐藏时据此撤销固化、交回 el-table 均摊
    const compacted = new Set<TableColumn>();

    const applyCompactWidths = () => {
        const cols = toValue(options.columns);
        // 以「可见的弹性列」为固化前提：弹性列被列配置勾掉后已无列吸收富余，若仍全员固化 width，
        // el-table 的 flexColumns 为空、会把表格写成固定 px 宽（比容器窄）→ 右侧出现无边界空白带
        if (!cols.some((col) => col.flexible && col.show)) {
            compacted.forEach((col) => (col.width = undefined));
            compacted.clear();
            return;
        }
        cols.forEach((col) => {
            // 字符串 minWidth（如 '250px'；el-table 仅认数字前缀 px、不支持百分比）与已显式设置 width 的列是调用方明确意图，不动
            if (!col.flexible && typeof col.minWidth === 'number' && col.width === undefined) {
                col.width = col.minWidth;
                compacted.add(col);
            }
        });
    };

    /** 空数据时的表头估宽：让空列表的表头+空态可见且有合理列宽；不定型，等首轮非空数据按内容重估 */
    const calculateHeaderOnlyMinWidth = () => {
        toValue(options.columns).forEach((item) => {
            if (item.autoWidth) {
                item.autoCalculateMinWidthByHeader();
            }
        });
    };

    /** 表格是否已具备真实布局宽度：v-show 未激活的 tab 内 clientWidth 为 0，此时 DOM 实测恒为 0。 */
    const hasRealWidth = () => (options.tableWrapRef.value?.clientWidth ?? 0) > 0;

    /**
     * 一轮「估算打底 → 显示 → 渲染后实测回填 → 定型」。
     *
     * 隐藏（v-show 未激活 tab / keep-alive 脱离 DOM）时 clientWidth=0，DOM 实测恒为 0：此时整轮直接推迟——
     * 不估算、不显示、不定型。若像先前那样在隐藏期先估算并 onEstimated 显示表格，表格会被画上「脱离内容的
     * 估算宽」（插槽列离线量不到图标/tag），等切到该 tab 可见时 ResizeObserver 再补测回填 → 用户看到列宽
     * 二次跳变，违背「算一次就冻结」。整轮推迟后，「切换到某 tab」与「刷新即在该 tab」走完全相同的
     * 估算→显示→回填→定型时序：表格只在可见那一刻出现一次并直接定型，无中间态、无跳变。
     */
    const runAutoWidth = () => {
        // 列宽在首轮非空数据到达且表格可见时一次性定型（估算打底 + 渲染后 DOM 回填精修），之后翻页不再变动
        if (columnWidthSettled.value) {
            return;
        }
        // 隐藏中：整轮推迟，等 ResizeObserver 在首次可见时重新触发（见下方容器尺寸观察器）
        if (!hasRealWidth()) {
            return;
        }
        const data = toValue(options.tableData);
        // 空数据：异步列表的「空」有两种——① 首屏/加载中，数据马上就到（假空），② 加载已结束确认为空。
        // 假空阶段若就显示表格，会先以「表头窄宽」画出（插槽列离线量不到内容），数据到后再撑宽 → 切换/挂载时肉眼可见的列宽跳变。
        // 故假空阶段保持隐藏（不估算、不显示），等数据到达走「估算→回填→定型」一次到位；确认为空才按表头估宽显示空态。
        if (!data || data.length === 0) {
            if (options.asyncLoad?.() && (toValue(options.loading) || !hasLoadedOnce)) {
                return;
            }
            calculateHeaderOnlyMinWidth();
            applyCompactWidths();
            options.onEstimated?.();
            nextTick(() => refineFromDom()); // 空态也量表头（排序/图标列自适应），不定型
            return;
        }
        calculateTableColumnMinWidth();
        applyCompactWidths();
        options.onEstimated?.();
        // 渲染可见后按单元格真实内容宽回填列宽：文本估算量不到插槽内图标/tag 等渲染占位。
        // 时序不变式（勿破）：必须在 el-table 重排（doLayout，走 rAF/ResizeObserver/50ms debounce）之前测——
        // 此刻列仍是隐藏期窄宽、内容必然溢出，「只量溢出」才量得到真实需求；若在重排后测，摊派宽会掩盖溢出导致回填失明。
        nextTick(() => {
            refineFromDom();
            columnWidthSettled.value = true;
        });
    };

    watch(() => toValue(options.tableData), runAutoWidth);

    // 加载态变化：首次加载结束（true→false）记为 hasLoadedOnce，并重跑一轮——
    // 数据到达走估算+回填+定型；加载结束仍为空则显示空态（此前假空阶段一直隐藏，无列宽跳变）。
    if (options.loading) {
        watch(
            () => toValue(options.loading),
            (isLoading, wasLoading) => {
                if (wasLoading && !isLoading) {
                    hasLoadedOnce = true;
                }
                runAutoWidth();
            }
        );
    }

    // 列可见性变化（列配置勾选/取消）后同步紧凑固化：弹性列被隐藏 → 撤销固化回落均摊（防右侧空白带）；
    // 重新勾选弹性列 → 恢复固化。settled 后 runAutoWidth 不再跑，故此 watch 是隐藏弹性列的唯一收口。
    watch(
        () => toValue(options.columns).some((col) => col.flexible && col.show),
        () => applyCompactWidths()
    );

    // 容器尺寸观察：仅在「未定型 + 首次获得真实宽度」时补测定型（覆盖 v-show 隐藏期到达数据的 tab）；
    // 定型后立即断开，保持「首轮定型后翻页/resize 冻结列宽」的既有语义（不会因窗口缩放而重算）。
    if (getCurrentInstance()) {
        let ro: ResizeObserver | null = null;
        const stopObserve = () => {
            ro?.disconnect();
            ro = null;
        };
        onMounted(() => {
            const el = options.tableWrapRef.value;
            if (!el || typeof ResizeObserver === 'undefined') {
                return;
            }
            ro = new ResizeObserver(() => {
                if (columnWidthSettled.value) {
                    stopObserve();
                    return;
                }
                runAutoWidth(); // 自身按 hasRealWidth 守卫：隐藏中不动，首次可见时补做整轮估算+回填+定型
            });
            ro.observe(el);
        });
        onUnmounted(stopObserve);
    }

    return { columnWidthSettled };
}
