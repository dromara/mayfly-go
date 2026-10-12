import { beforeEach, describe, expect, it, vi } from 'vitest';
import { nextTick, ref, type Ref } from 'vue';

// getTextWidth 依赖真实布局引擎（offsetWidth），单测环境量不到宽度，mock 成确定性函数：每个字符 10px
vi.mock('@/common/utils/string', () => ({
    getTextWidth: vi.fn((str: string) => str.length * 10),
}));

import { TableColumn } from '../index';
import { useColumnAutoWidth } from '../useColumnAutoWidth';

/** 构造 .el-table__body > tbody > tr > td > .cell 的最小模拟结构（无需真实渲染 el-table） */
function mountMockTable(rowCells: string[][]): Ref<HTMLElement | undefined> {
    const wrap = document.createElement('div');
    const bodyTable = document.createElement('table');
    bodyTable.className = 'el-table__body';
    const tbody = document.createElement('tbody');
    for (const cells of rowCells) {
        const tr = document.createElement('tr');
        for (const text of cells) {
            const td = document.createElement('td');
            const cell = document.createElement('div');
            cell.className = 'cell';
            cell.textContent = text;
            td.appendChild(cell);
            tr.appendChild(td);
            tbody.appendChild(tr);
        }
        bodyTable.appendChild(tbody);
        wrap.appendChild(bodyTable);
    }
    // 给容器一个真实宽度：runAutoWidth 靠 clientWidth>0 判定「可见」，隐藏(0宽)会推迟定型
    Object.defineProperty(wrap, 'clientWidth', { value: 1000, configurable: true });
    return ref(wrap);
}

/** 在指定元素上模拟布局尺寸（happy-dom 不做真实布局，scrollWidth/clientWidth 恒为 0） */
function mockLayout(el: HTMLElement, scrollWidth: number, clientWidth: number) {
    Object.defineProperty(el, 'scrollWidth', { value: scrollWidth, configurable: true });
    Object.defineProperty(el, 'clientWidth', { value: clientWidth, configurable: true });
}

/** 获取模拟表格指定行指定列（0 起）的 .cell */
function cellAt(wrap: Ref<HTMLElement | undefined>, row: number, col: number): HTMLElement {
    const cell = wrap.value?.querySelector<HTMLElement>(`.el-table__body tbody tr:nth-child(${row + 1}) td:nth-child(${col + 1}) .cell`);
    if (!cell) {
        throw new Error(`mock cell(${row}, ${col}) not found`);
    }
    return cell;
}

/** 跑完一轮「数据到达 → 估算 → nextTick 回填 → 定型」的完整调度 */
async function flushColumnWidthTicks() {
    await nextTick();
    await nextTick();
}

describe('useColumnAutoWidth', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('空数据不触发估算/回填/定型', async () => {
        const col = TableColumn.new('name', 'common.name');
        const tableData = ref<Record<string, unknown>[]>([]);
        const onEstimated = vi.fn();
        const wrap = mountMockTable([]);
        const { columnWidthSettled } = useColumnAutoWidth({
            tableWrapRef: wrap,
            columns: () => [col],
            tableData,
            showSelection: () => false,
            onEstimated,
        });

        await flushColumnWidthTicks();

        expect(col.minWidth).toBeUndefined();
        expect(onEstimated).not.toHaveBeenCalled();
        expect(columnWidthSettled.value).toBe(false);
    });

    it('空数据赋值：按表头估宽并显示表格（不定型），首轮非空数据再按内容定型', async () => {
        const col = TableColumn.new('name', 'test.colName');
        const tableData = ref<Record<string, unknown>[]>([]);
        const onEstimated = vi.fn();
        const wrap = mountMockTable([['short']]);
        const { columnWidthSettled } = useColumnAutoWidth({
            tableWrapRef: wrap,
            columns: () => [col],
            tableData,
            showSelection: () => false,
            onEstimated,
        });

        // 接口返回空列表：表头+空态可见，列宽按表头文本估算，但未定型
        tableData.value = [];
        await flushColumnWidthTicks();
        // 'test.colName' 12 字符 → 120+30=150（mock 无表头节点，表头实测回填不触发，保持估算值）
        expect(col.minWidth).toBe(150);
        expect(onEstimated).toHaveBeenCalledTimes(1);
        expect(columnWidthSettled.value).toBe(false);

        // 首轮非空数据到达：内容估算 + 回填 + 定型
        tableData.value = [{ name: 'x'.repeat(25) }];
        await flushColumnWidthTicks();
        // 内容 25 字符 → 250+30=280 > 表头 150 → 280
        expect(col.minWidth).toBe(280);
        expect(columnWidthSettled.value).toBe(true);
    });

    it('异步列表：数据未到达的假空不显示，数据到达后一次定型（无表头窄宽→内容宽的跳变）', async () => {
        const col = TableColumn.new('ipPort', 'Ip:Port').isSlot();
        const tableData = ref<Record<string, unknown>[]>([]);
        const loading = ref(false);
        const onEstimated = vi.fn();
        const wrap = mountMockTable([['59.110.115.25']]);
        mockLayout(cellAt(wrap, 0, 0), 200, 50); // 插槽列离线量不到，可见内容溢出靠回填
        useColumnAutoWidth({
            tableWrapRef: wrap,
            columns: () => [col],
            tableData,
            showSelection: () => false,
            loading,
            asyncLoad: () => true,
            onEstimated,
        });

        // 挂载即开始加载：loading true、数据仍为空（假空）→ 不估算不显示
        loading.value = true;
        await flushColumnWidthTicks();
        expect(onEstimated).not.toHaveBeenCalled();
        expect(col.minWidth).toBeUndefined();

        // 数据到达：走估算+回填+定型，一次显示到位
        tableData.value = [{ ip: '59.110.115.25', port: 22 }];
        await flushColumnWidthTicks();
        expect(onEstimated).toHaveBeenCalled();
        expect(col.minWidth).toBe(230);
    });

    it('异步列表：加载结束仍为空才按表头估宽显示空态', async () => {
        const col = TableColumn.new('name', 'test.colName');
        const tableData = ref<Record<string, unknown>[]>([]);
        const loading = ref(false);
        const onEstimated = vi.fn();
        const wrap = mountMockTable([]);
        useColumnAutoWidth({
            tableWrapRef: wrap,
            columns: () => [col],
            tableData,
            showSelection: () => false,
            loading,
            asyncLoad: () => true,
            onEstimated,
        });

        // 加载中且为空：不显示
        loading.value = true;
        await flushColumnWidthTicks();
        expect(onEstimated).not.toHaveBeenCalled();

        // 加载结束仍为空：显示空态（'test.colName' 12 字符 → 120+30=150）
        loading.value = false;
        await flushColumnWidthTicks();
        expect(onEstimated).toHaveBeenCalled();
        expect(col.minWidth).toBe(150);
    });

    it('首轮非空数据：估算打底 → onEstimated 回调 → 回填后定型', async () => {
        const col = TableColumn.new('name', 'test.colName');
        const tableData = ref<Record<string, unknown>[]>([]);
        const onEstimated = vi.fn();
        const wrap = mountMockTable([['short']]); // 内容无溢出，回填零修正
        const { columnWidthSettled } = useColumnAutoWidth({
            tableWrapRef: wrap,
            columns: () => [col],
            tableData,
            showSelection: () => false,
            onEstimated,
        });

        tableData.value = [{ name: 'mayfly_uib_it' }];
        await flushColumnWidthTicks();

        // 估算：内容 13 字符 → 13*10+30=160，表头 'test.colName' 非真实 i18n key 回退原文 12 字符 → 120+30=150，取大者 160
        expect(col.minWidth).toBe(160);
        expect(onEstimated).toHaveBeenCalledTimes(1);
        expect(columnWidthSettled.value).toBe(true);
    });

    it('定型后翻页列宽冻结不变', async () => {
        const col = TableColumn.new('name', 'test.colName');
        const tableData = ref<Record<string, unknown>[]>([]);
        const wrap = mountMockTable([]);
        useColumnAutoWidth({
            tableWrapRef: wrap,
            columns: () => [col],
            tableData,
            showSelection: () => false,
        });

        tableData.value = [{ name: 'mayfly_uib_it' }];
        await flushColumnWidthTicks();
        expect(col.minWidth).toBe(160);

        // 翻页后出现超长内容：若重算会被封顶为 400，冻结则保持 160
        tableData.value = [{ name: 'x'.repeat(50) }];
        await flushColumnWidthTicks();
        expect(col.minWidth).toBe(160);
    });

    it('DOM 回填：溢出内容向上修正 minWidth', async () => {
        const col = TableColumn.new('name', 'test.colName');
        const tableData = ref<Record<string, unknown>[]>([]);
        const wrap = mountMockTable([['short']]);
        // 模拟单元格内容真实需求宽 200、可视宽 50 → 溢出
        mockLayout(cellAt(wrap, 0, 0), 200, 50);
        useColumnAutoWidth({
            tableWrapRef: wrap,
            columns: () => [col],
            tableData,
            showSelection: () => false,
        });

        tableData.value = [{ name: 'mayfly_uib_it' }];
        await flushColumnWidthTicks();

        // 回填：min(200+30, 400)=230 > 估算 160 → 修正为 230
        expect(col.minWidth).toBe(230);
    });

    it('DOM 回填：内容未溢出时不修正（防 scrollWidth=列宽 恒等式导致的单调膨胀）', async () => {
        const col = TableColumn.new('name', 'test.colName');
        const tableData = ref<Record<string, unknown>[]>([]);
        const wrap = mountMockTable([['short']]);
        // 内容窄于列宽：scrollWidth 恒等于 clientWidth（DOM 恒等式），不得当作内容需求宽
        mockLayout(cellAt(wrap, 0, 0), 170, 170);
        useColumnAutoWidth({
            tableWrapRef: wrap,
            columns: () => [col],
            tableData,
            showSelection: () => false,
        });

        tableData.value = [{ name: 'mayfly_uib_it' }];
        await flushColumnWidthTicks();
        expect(col.minWidth).toBe(160);
    });

    it('showSelection 时回填列序按 td 偏移一位对齐', async () => {
        const col = TableColumn.new('name', 'test.colName');
        const tableData = ref<Record<string, unknown>[]>([]);
        const wrap = mountMockTable([['checkbox', 'short']]);
        // selection 列故意放大溢出量：若未偏移错量到它，回填会被封顶 400 而非 280
        mockLayout(cellAt(wrap, 0, 0), 900, 50);
        mockLayout(cellAt(wrap, 0, 1), 250, 50);
        useColumnAutoWidth({
            tableWrapRef: wrap,
            columns: () => [col],
            tableData,
            showSelection: () => true,
        });

        tableData.value = [{ name: 'mayfly_uib_it' }];
        await flushColumnWidthTicks();

        // 回填取 td[1]：min(250+30, 400)=280 > 估算 160 → 280
        expect(col.minWidth).toBe(280);
    });

    it('声明弹性列：其余数值 minWidth 列固化为 width，弹性列保留 min-width', async () => {
        const flexibleCol = TableColumn.new('name', 'test.colName').setFlexible();
        const normalCol = TableColumn.new('prop', 'test.colProp');
        const tableData = ref<Record<string, unknown>[]>([]);
        const wrap = mountMockTable([['short', 'short']]);
        useColumnAutoWidth({
            tableWrapRef: wrap,
            columns: () => [flexibleCol, normalCol],
            tableData,
            showSelection: () => false,
        });

        tableData.value = [{ name: 'mayfly_uib_it', prop: 'abc' }];
        await flushColumnWidthTicks();

        // 固化：非弹性列 width = minWidth，退出容器富余摊派
        expect(normalCol.width).toBe(normalCol.minWidth);
        // 弹性列保留 min-width、width 未设置，继续吸收容器富余
        expect(flexibleCol.width).toBeUndefined();
        expect(flexibleCol.minWidth).toBeGreaterThan(0);
    });

    it('未声明弹性列：保持全员 min-width 均摊，不固化', async () => {
        const col = TableColumn.new('name', 'test.colName');
        const tableData = ref<Record<string, unknown>[]>([]);
        const wrap = mountMockTable([['short']]);
        useColumnAutoWidth({
            tableWrapRef: wrap,
            columns: () => [col],
            tableData,
            showSelection: () => false,
        });

        tableData.value = [{ name: 'mayfly_uib_it' }];
        await flushColumnWidthTicks();

        expect(col.minWidth).toBe(160);
        expect(col.width).toBeUndefined();
    });

    it('固化列的 DOM 回填向上修正 width 而非 minWidth', async () => {
        const flexibleCol = TableColumn.new('name', 'test.colName').setFlexible();
        const normalCol = TableColumn.new('prop', 'test.colProp');
        const tableData = ref<Record<string, unknown>[]>([]);
        const wrap = mountMockTable([['short', 'overflow-content']]);
        // 估算欠宽（图标/tag 占位量不到），渲染后溢出：回填应修固化列的生效约束 width
        mockLayout(cellAt(wrap, 0, 1), 200, 50);
        useColumnAutoWidth({
            tableWrapRef: wrap,
            columns: () => [flexibleCol, normalCol],
            tableData,
            showSelection: () => false,
        });

        tableData.value = [{ name: 'mayfly_uib_it', prop: 'abc' }];
        await flushColumnWidthTicks();

        // 回填 measured = min(200+30, 400)=230，固化列 width 从估算值 150 向上修正为 230
        expect(normalCol.width).toBe(230);
        expect(normalCol.minWidth).toBe(150);
    });

    it('表头实测回填：表头单行需求超出估算时向上加宽列（取代原先一律 +60 的固定预算）', async () => {
        const col = TableColumn.new('name', 'test.colName');
        const tableData = ref<Record<string, unknown>[]>([]);

        // 构造带表头的 mock：.el-table__header thead tr th .cell（表体留空，只验表头路径）
        const wrap = document.createElement('div');
        const headerTable = document.createElement('table');
        headerTable.className = 'el-table__header';
        const thead = document.createElement('thead');
        const htr = document.createElement('tr');
        const th = document.createElement('th');
        const hcell = document.createElement('div');
        hcell.className = 'cell';
        th.appendChild(hcell);
        htr.appendChild(th);
        thead.appendChild(htr);
        headerTable.appendChild(thead);
        wrap.appendChild(headerTable);
        Object.defineProperty(wrap, 'clientWidth', { value: 1000, configurable: true }); // 可见：让 runAutoWidth 走定型路径
        // 表头单行需求 200、可视 50 → 溢出（happy-dom 无 EP 样式，.cell 计算内边距为 0）
        mockLayout(hcell, 200, 50);

        useColumnAutoWidth({
            tableWrapRef: ref(wrap),
            columns: () => [col],
            tableData,
            showSelection: () => false,
        });

        // 内容 'ab' 2 字符 → 50，表头估算 12*10+30=150 → 估算取 150；表头实测溢出 200 → 回填 200+30=230
        tableData.value = [{ name: 'ab' }];
        await flushColumnWidthTicks();
        expect(col.minWidth).toBe(230);
    });

    it('隐藏期整轮推迟（不估算/不显示/不定型），切到可见后一次性算完定型（v-show tab 场景）', async () => {
        // 插槽列离线量不到内容（行里没有 ipPort 字段），列宽全靠渲染后 DOM 回填；
        // 隐藏期(clientWidth=0)若先估算并显示，会先画上脱离内容的估算宽，可见后补测再跳变。
        // 正确行为：隐藏期整轮推迟（minWidth 保持未初始化、不定型），切到可见才估算+回填+定型一次到位。
        const col = TableColumn.new('ipPort', 'Ip:Port').isSlot();
        const tableData = ref<Record<string, unknown>[]>([]);
        const wrap = mountMockTable([['59.110.115.25']]);
        mockLayout(cellAt(wrap, 0, 0), 200, 50); // 渲染后内容溢出，可见时应回填到 230
        Object.defineProperty(wrap.value, 'clientWidth', { value: 0, configurable: true }); // 先置为隐藏
        const { columnWidthSettled } = useColumnAutoWidth({
            tableWrapRef: wrap,
            columns: () => [col],
            tableData,
            showSelection: () => false,
        });

        tableData.value = [{ ip: '59.110.115.25', port: 22 }];
        await flushColumnWidthTicks();
        // 隐藏期：整轮推迟，不估算不显示不定型（minWidth 保持未初始化）
        expect(col.minWidth).toBeUndefined();
        expect(columnWidthSettled.value).toBe(false);

        // 变为可见（happy-dom 无 ResizeObserver，用再次赋值模拟可见后的重测触发）
        Object.defineProperty(wrap.value, 'clientWidth', { value: 1000, configurable: true });
        tableData.value = [{ ip: '59.110.115.25', port: 22 }];
        await flushColumnWidthTicks();
        // 可见后：按溢出内容回填 min(200+30,400)=230 并定型
        expect(col.minWidth).toBe(230);
        expect(columnWidthSettled.value).toBe(true);
    });
});
