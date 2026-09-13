import { ref } from 'vue';

export function useTableSelection(datas: () => Record<string, unknown>[]) {
    // 选中的数据， key->rowIndex  value->rowData
    const selectionRowsMap = ref(new Map<number, Record<string, unknown>>());

    // 最后一次点击的行索引，用于 shift 批量选择
    let lastSelectedRowIndex: number | null = null;

    /**
     * 判断当前行是否被选中
     */
    const isSelection = (rowIndex: number): boolean => {
        return selectionRowsMap.value.has(rowIndex);
    };

    /**
     * 选中指定行
     * @param rowIndex 行索引
     * @param rowData 行数据
     * @param isMultiple 是否允许多选
     */
    const selectionRow = (rowIndex: number, rowData: Record<string, unknown>, isMultiple = false) => {
        if (isMultiple) {
            // 如果重复点击，则取消该选中数据
            if (selectionRowsMap.value.get(rowIndex)) {
                selectionRowsMap.value.delete(rowIndex);
                return;
            }
        } else {
            selectionRowsMap.value.clear();
        }
        selectionRowsMap.value.set(rowIndex, rowData);
        lastSelectedRowIndex = rowIndex;
    };

    /**
     * Shift 批量选择：选中起始行到当前行之间的所有行
     */
    const selectionRowRange = (startIndex: number, endIndex: number) => {
        const from = Math.min(startIndex, endIndex);
        const to = Math.max(startIndex, endIndex);
        const tableData = datas();
        for (let i = from; i <= to; i++) {
            const rowData = tableData[i];
            if (rowData) {
                selectionRowsMap.value.set(i, rowData);
            }
        }
    };

    /**
     * 行事件处理
     */
    const rowEventHandlers = {
        onClick: (e: { event: MouseEvent; rowIndex: number; rowData: Record<string, unknown> }) => {
            const event = e.event;
            const rowIndex = e.rowIndex;
            const rowData = e.rowData;
            // 按住ctrl/meta点击，则多选切换
            if (event.ctrlKey || event.metaKey) {
                selectionRow(rowIndex, rowData, true);
                return;
            }
            // 按住shift点击，则批量选择起始行到当前行
            if (event.shiftKey && lastSelectedRowIndex !== null) {
                selectionRowsMap.value.clear();
                selectionRowRange(lastSelectedRowIndex, rowIndex);
                return;
            }
            selectionRow(rowIndex, rowData);
        },
    };

    const rowClass = (row: { rowIndex: number }) => {
        if (isSelection(row.rowIndex)) {
            return 'data-selection';
        }
        return '';
    };

    const clearSelection = () => {
        selectionRowsMap.value.clear();
        lastSelectedRowIndex = null;
    };

    return {
        selectionRowsMap,
        isSelection,
        selectionRow,
        selectionRowRange,
        rowEventHandlers,
        rowClass,
        clearSelection,
    };
}
