import { ref, type Ref } from 'vue';
import { DbInst } from '../../../db';

export class NowUpdateCell {
    rowIndex: number;
    colIndex: number;
    dataType: string;
    oldValue: unknown;
}

export class UpdatedRow {
    /**
     * 主键值
     */
    primaryValue: unknown;

    /**
     * 行数据
     */
    rowData: Record<string, unknown>;

    /**
     * 修改到的列信息, columnName -> tablecelldata
     */
    columnsMap = new Map<string, TableCellData>();
}

export class TableCellData {
    /**
     * 旧值
     */
    oldValue: unknown;
}

export interface UseTableEditOptions {
    dbId: () => number;
    db: () => string;
    table: () => string;
}

export function useTableEdit(options: UseTableEditOptions) {
    // 当前正在更新的单元格
    const nowUpdateCell: Ref<NowUpdateCell | null> = ref(null);

    // 更新单元格  key-> rowIndex  value -> 更新行
    const cellUpdateMap = ref(new Map<number, UpdatedRow>());

    const getNowDbInst = () => {
        return DbInst.getInst(options.dbId());
    };

    /**
     * 当前单元格是否允许编辑
     */
    const canEdit = (rowIndex: number, colIndex: number) => {
        return options.table() && nowUpdateCell.value?.rowIndex == rowIndex && nowUpdateCell.value?.colIndex == colIndex;
    };

    /**
     * 判断当前单元格是否被更新了
     */
    const isUpdated = (rowIndex: number, columnName: string) => {
        return cellUpdateMap.value.get(rowIndex)?.columnsMap.get(columnName);
    };

    const onEnterEditMode = (rowData: Record<string, unknown>, column: { key: string; dataType?: string }, rowIndex = 0, columnIndex = 0) => {
        // 不存在表，或者已经在编辑中，则不处理
        if (!options.table() || nowUpdateCell.value) {
            return;
        }

        nowUpdateCell.value = {
            rowIndex: rowIndex,
            colIndex: columnIndex,
            oldValue: rowData[column.key],
            dataType: column.dataType ?? '',
        };
    };

    const onExitEditMode = (rowData: Record<string, unknown>, column: { key: string }, rowIndex = 0) => {
        if (!nowUpdateCell.value) {
            return;
        }
        const oldValue = nowUpdateCell.value.oldValue;
        const newValue = rowData[column.key];

        // 未改变单元格值
        if (oldValue == newValue) {
            nowUpdateCell.value = null;
            return;
        }

        let updatedRow = cellUpdateMap.value.get(rowIndex);
        if (!updatedRow) {
            updatedRow = new UpdatedRow();
            updatedRow.rowData = rowData;
            cellUpdateMap.value.set(rowIndex, updatedRow);
        }

        const columnName = column.key;
        let cellData = updatedRow.columnsMap.get(columnName);
        if (cellData) {
            // 多次修改情况，可能又修改回原值，则移除该修改单元格
            if (cellData.oldValue == newValue) {
                cellUpdateMap.value.delete(rowIndex);
            }
        } else {
            cellData = new TableCellData();
            cellData.oldValue = oldValue;
            updatedRow.columnsMap.set(columnName, cellData);
        }

        nowUpdateCell.value = null;
    };

    const submitUpdateFields = async (onSuccess?: () => void) => {
        const dbInst = getNowDbInst();
        if (cellUpdateMap.value.size == 0) {
            return;
        }

        const db = options.db();
        const table = options.table();
        let res = '';

        for (let updateRow of cellUpdateMap.value.values()) {
            const rowData = { ...updateRow.rowData };
            let updateColumnValue: Record<string, unknown> = {};

            for (let k of updateRow.columnsMap.keys()) {
                const v = updateRow.columnsMap.get(k);
                if (!v) {
                    continue;
                }
                updateColumnValue[k] = rowData[k];
                // 将更新的字段对应的原始数据还原（主要应对可能更新修改了主键等）
                rowData[k] = v.oldValue;
            }
            res += await dbInst.genUpdateSql(db, table, updateColumnValue, rowData);
        }

        dbInst.promptExeSql(db, res, undefined, () => {
            cellUpdateMap.value.clear();
            onSuccess?.();
        });
    };

    const cancelUpdateFields = (onSuccess?: () => void) => {
        const updateRows = cellUpdateMap.value.values();
        // 恢复原值
        for (let updateRow of updateRows) {
            const rowData = updateRow.rowData;
            updateRow.columnsMap.forEach((v: TableCellData, k: string) => {
                rowData[k] = v.oldValue;
            });
        }
        cellUpdateMap.value.clear();
        onSuccess?.();
    };

    const clearEditState = () => {
        cellUpdateMap.value.clear();
        nowUpdateCell.value = null;
    };

    return {
        nowUpdateCell,
        cellUpdateMap,
        canEdit,
        isUpdated,
        onEnterEditMode,
        onExitEditMode,
        submitUpdateFields,
        cancelUpdateFields,
        clearEditState,
    };
}
