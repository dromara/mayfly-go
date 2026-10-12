import type { TableColumn } from './table-column';

/**
 * 列宽的「渲染后 DOM 实测」：读真实布局里单元格/表头的溢出宽，回填到列模型。
 *
 * 与 useColumnAutoWidth（生命周期编排：watch / ResizeObserver / 定型冻结）分开，是为了让
 * 「怎么从 DOM 量出需求宽」这一测量职责独立成可单测的纯函数，编排侧只在合适的时机调用它。
 */

/**
 * 单元格内容真实需求宽：只在内容溢出当前列宽时返回溢出叶子的完整宽，否则返回 0（无需修正）。
 * 必须只量溢出：scrollWidth ≥ clientWidth 是 DOM 恒等式，内容窄于列宽时元素的 scrollWidth
 * 等于列宽本身，若把它当内容需求宽回填，每轮 minWidth 恒被抬升一个 CELL_PADDING，
 * 列宽单调膨胀直到上限；只有真正溢出（scrollWidth > clientWidth）才说明内容需求超出当前列宽。
 * 溢出只发生在声明了 overflow:hidden 的 truncate 叶子元素里，它的 scrollWidth 才是完整内容宽；
 * 图标/tag 等非溢出兄弟占位由 CELL_PADDING 余量覆盖。
 *
 * 不变式（勿破）：「只量横向溢出」意味着——内容不横向溢出的列，本回填一律返回 0、不生效。两类列因此
 * 必须自带离线估算下限（setSlotText / setAddWidth / setMinWidth 三者之一），否则塌成表头宽：
 *   ① 定宽交互控件（el-switch/el-select）：列被 fit 撑宽那一帧控件不溢出；
 *   ② 取消 show-overflow-tooltip 的换行插槽列（多 tag flex-wrap）：内容换行而非横向溢出，永远量不到。
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
 * 表头单元格单行需求宽。表头默认 white-space:normal 会换行、横向永不溢出→量不到真实需求，
 * 故临时置 nowrap 读出单行宽：仅当内容超出当前列宽（有排序箭头/表头图标等额外占位）才返回内容宽
 * （扣掉 .cell 自身内边距，与内容列同口径，refineMinWidthByDom 再统一补 CELL_PADDING），量完立即还原。
 */
const measureHeaderCellWidth = (th: HTMLElement): number => {
    const cell = th.querySelector<HTMLElement>('.cell');
    if (!cell) {
        return 0;
    }
    const prevWhiteSpace = cell.style.whiteSpace;
    cell.style.whiteSpace = 'nowrap';
    let need = 0;
    if (cell.scrollWidth > cell.clientWidth + 1) {
        const cs = getComputedStyle(cell);
        const padX = (parseFloat(cs.paddingLeft) || 0) + (parseFloat(cs.paddingRight) || 0);
        need = cell.scrollWidth - padX;
    }
    cell.style.whiteSpace = prevWhiteSpace;
    return need;
};

/**
 * 渲染后按真实 DOM 回填列宽（只向上修正）：
 * - 表头：量单行需求宽（含排序箭头/表头图标），让列宽自适应真实表头，取代原先一律 +HEADER_PADDING 的固定预算；
 * - 表体：只量溢出（scrollWidth ≥ clientWidth 是恒等式，不判溢出会把列宽当内容需求回填 → 单调膨胀）。
 * 单元格顺序 = selection 列（若有）+ show 的列，表头/表体同偏移按位对齐防错位；
 * 隐藏（display:none）时量到 0，refineMinWidthByDom 内部跳过，不写脏值。
 */
export function refineColumnWidthsByDom(wrap: HTMLElement, columns: TableColumn[], showSelection: boolean): void {
    const visibleColumns = columns.filter((c) => c.show);
    const offset = showSelection ? 1 : 0;

    const headerRow = wrap.querySelector<HTMLElement>('.el-table__header thead tr');
    if (headerRow) {
        visibleColumns.forEach((col, i) => {
            if (!col.autoWidth) {
                return;
            }
            const th = headerRow.children[i + offset] as HTMLElement | undefined;
            if (th) {
                col.refineMinWidthByDom(measureHeaderCellWidth(th));
            }
        });
    }

    const bodyTable = wrap.querySelector<HTMLElement>('.el-table__body');
    const rows = bodyTable ? bodyTable.querySelectorAll('tbody tr') : [];
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
}
