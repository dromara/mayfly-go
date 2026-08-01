import { exportCsv, exportExcel, exportFile } from '@/common/utils/export';
import { formatDate } from '@/common/utils/format';
import { DbInst } from '../../../db';

export interface UseTableExportOptions {
    dbId: () => number;
    db: () => string;
    table: () => string;
    datas: () => Record<string, unknown>[];
    columns: () => { columnName?: string; show?: boolean; key: string; title: string }[];
}

export function useTableExport(options: UseTableExportOptions) {
    const getNowDbInst = () => {
        return DbInst.getInst(options.dbId());
    };

    const getVisibleColumnNames = (): string[] => {
        let columnNames: string[] = [];
        for (let column of options.columns()) {
            if (column.show) {
                columnNames.push(column.columnName ?? '');
            }
        }
        return columnNames;
    };

    /**
     * 导出CSV
     */
    const onExportCsv = () => {
        const dataList = options.datas();
        const columnNames = getVisibleColumnNames();
        exportCsv(`Data-${options.table()}-${formatDate(new Date(), 'YYYYMMDDHHmm')}`, columnNames, dataList);
    };

    /**
     * 导出Excel
     */
    const onExportExcel = async () => {
        const dataList = options.datas();
        const columnNames = getVisibleColumnNames();
        await exportExcel(`Data-${options.table()}-${formatDate(new Date(), 'YYYYMMDDHHmm')}`, [{ name: 'Data', columns: columnNames, datas: dataList }]);
    };

    /**
     * 导出SQL
     */
    const onExportSql = async () => {
        const dataList = options.datas();
        exportFile(`Data-${options.table()}-${formatDate(new Date(), 'YYYYMMDDHHmm')}.sql`, await getNowDbInst().genInsertSql(options.db(), options.table(), dataList));
    };

    /**
     * 生成JSON
     */
    const onGenerateJson = (selectionDatas: Record<string, unknown>[]): string => {
        // 按列字段重新排序对象key
        const jsonObj = [];
        for (let selectionData of selectionDatas) {
            let obj: Record<string, unknown> = {};
            for (let column of options.columns()) {
                if (column.show) {
                    obj[column.title] = selectionData[column.key];
                }
            }
            jsonObj.push(obj);
        }
        return JSON.stringify(jsonObj, null, 4);
    };

    /**
     * 生成Insert SQL
     */
    const onGenerateInsertSql = async (selectionDatas: Record<string, unknown>[]): Promise<string> => {
        return await getNowDbInst().genInsertSql(options.db(), options.table(), selectionDatas);
    };

    return {
        onExportCsv,
        onExportExcel,
        onExportSql,
        onGenerateJson,
        onGenerateInsertSql,
    };
}
