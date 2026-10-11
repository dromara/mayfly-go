import { nextTick, ref, toValue, watch, type MaybeRefOrGetter, type Ref } from 'vue';
import { TableColumn } from './index';

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

    /**
     * 单元格内容真实需求宽：只在内容溢出当前列宽时返回溢出叶子的完整宽，否则返回 0（无需修正）。
     * 必须只量溢出：scrollWidth ≥ clientWidth 是 DOM 恒等式，内容窄于列宽时元素的 scrollWidth
     * 等于列宽本身，若把它当内容需求宽回填，每轮 minWidth 恒被抬升一个 CELL_PADDING，
     * 列宽单调膨胀直到上限；只有真正溢出（scrollWidth > clientWidth）才说明内容需求超出当前列宽。
     * 溢出只发生在声明了 overflow:hidden 的 truncate 叶子元素里，它的 scrollWidth 才是完整内容宽；
     * 图标/tag 等非溢出兄弟占位由 CELL_PADDING 余量覆盖。
     */
    const measureCellContentWidth = (cell: HTMLElement): number => {
        if (cell.scrollWidth > cell.clientWidth + 1) {
            return cell.scrollWidth;
        }
        for (const el of cell.querySelectorAll<HTMLElement>('*')) {
            if (el.scrollWidth > el.clientWidth + 1) {
                return el.scrollWidth;
            }
        }
        return 0;
    };

    /**
     * 遍历各列单元格，取内容真实占位宽回填 minWidth。
     * td 顺序 = selection 列（若有）+ show 的列（含不回填的固定宽列），按位对齐防错位；
     * 隐藏（外层 display:none）时 scrollWidth 为 0，refineMinWidthByDom 内部会跳过，不写脏值。
     */
    const refineColumnWidthsByDom = () => {
        const bodyTable = options.tableWrapRef.value?.querySelector<HTMLElement>('.el-table__body');
        if (!bodyTable) {
            return;
        }
        const rows = bodyTable.querySelectorAll('tbody tr');
        if (!rows.length) {
            return;
        }
        const visibleColumns = toValue(options.columns).filter((c) => c.show);
        const offset = options.showSelection() ? 1 : 0;
        for (const tr of rows) {
            visibleColumns.forEach((col, i) => {
                if (!col.autoWidth) {
                    return;
                }
                const cell = (tr.children[i + offset] as HTMLElement | undefined)?.querySelector<HTMLElement>('.cell');
                if (cell) {
                    col.refineMinWidthByDom(measureCellContentWidth(cell));
                }
            });
        }
    };

    /** 计算各列 minWidth 估算基线（离线文本测宽，渲染前执行） */
    const calculateTableColumnMinWidth = () => {
        const data = toValue(options.tableData);
        if (!data || data.length === 0) {
            return;
        }
        toValue(options.columns).forEach((item) => {
            if (item.autoWidth && item.show) {
                item.autoCalculateMinWidth(data);
            }
        });
    };

    /**
     * 声明了弹性列时的紧凑固化：其余数值 minWidth 列把 minWidth 固化为 width，
     * 退出 el-table fit 的容器富余摊派，列宽贴内容；富余集中到弹性列（保留 min-width）。
     * 未声明弹性列时保持全员 min-width 均摊的既有默认，页面零回归。
     * 必须在估算后、表格渲染前调用：若等渲染后再固化，摊派宽会遮掩估算欠宽列的溢出，
     * 回填量不到真实需求。
     */
    const applyCompactWidths = () => {
        const cols = toValue(options.columns);
        if (!cols.some((col) => col.flexible)) {
            return;
        }
        cols.forEach((col) => {
            // 字符串 minWidth（如 '10%'）与已显式设置 width 的列是调用方明确意图，不动
            if (!col.flexible && typeof col.minWidth === 'number' && col.width === undefined) {
                col.width = col.minWidth;
            }
        });
    };

    /** 空数据时的表头估宽：让空列表的表头+空态可见且有合理列宽；不定型，等首轮非空数据按内容重估 */
    const calculateHeaderOnlyMinWidth = () => {
        toValue(options.columns).forEach((item) => {
            if (item.autoWidth && item.show) {
                item.autoCalculateMinWidthByHeader();
            }
        });
    };

    watch(
        () => toValue(options.tableData),
        () => {
            // 列宽在首轮非空数据到达时一次性定型（估算打底 + 渲染后 DOM 回填精修），
            // 之后翻页不再变动；定型后回填已收敛（只量溢出，无累积路径），不存在跨页膨胀
            if (columnWidthSettled.value) {
                return;
            }
            const data = toValue(options.tableData);
            // 空数据：仅按表头文本估宽并显示表格（表头+空态可见），不定型——
            // 首轮非空数据到达时仍走完整估算+回填+定型
            if (!data || data.length === 0) {
                calculateHeaderOnlyMinWidth();
                applyCompactWidths();
                options.onEstimated?.();
                return;
            }
            calculateTableColumnMinWidth();
            applyCompactWidths();
            options.onEstimated?.();
            // 渲染可见后按单元格真实内容宽回填列宽：文本估算量不到插槽内图标/tag 等渲染占位
            nextTick(() => {
                refineColumnWidthsByDom();
                columnWidthSettled.value = true;
            });
        }
    );

    return { columnWidthSettled };
}
