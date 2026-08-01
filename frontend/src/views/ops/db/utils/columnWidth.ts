import { getTextWidth } from '@/common/utils/string';
import type { ColumnMetadata } from '../types';

/**
 * 根据列名和表数据自动计算列宽度
 * @param prop 列字段名
 * @param tableData 表数据
 * @returns 列宽度
 */
export function flexColumnWidth(prop: string, tableData: Record<string, unknown>[]): number | undefined {
    if (!prop || !prop.length || prop.length === 0 || prop === undefined) {
        return;
    }

    // 获取列名称的长度 加上排序图标长度、abc为字段类型简称占位符、更多/排序图标等
    const columnWidth: number = getTextWidth(prop + 'abc') + 25;
    // prop为该列的字段名(传字符串);tableData为该表格的数据源(传变量);
    if (!tableData || !tableData.length || tableData.length === 0 || tableData === undefined) {
        return columnWidth;
    }

    // 获取该列中最长的数据(内容)
    let maxWidthText = '';
    const length = tableData.length > 10 ? 10 : tableData.length; // 只取前几条数据计算宽度
    // 获取该列中最长的数据(内容)
    for (let i = 0; i < length; i++) {
        let nowValue = tableData[i][prop];
        if (!nowValue) {
            continue;
        }
        // 转为字符串比较长度
        let nowText = nowValue + '';
        if (nowText.length > maxWidthText.length) {
            maxWidthText = nowText;
        }
    }
    const contentWidth: number = getTextWidth(maxWidthText) + 3;
    const flexWidth: number = contentWidth > columnWidth ? contentWidth : columnWidth;
    return flexWidth > 500 ? 500 : flexWidth;
}

/**
 * 初始化所有列信息，完善需要显示的列类型，包含长度等，如varchar(20)
 * @param columns 列元数据数组
 */
export function initColumns(columns: ColumnMetadata[]) {
    if (!columns) {
        return;
    }
    for (let col of columns) {
        if (col.charMaxLength && col.charMaxLength > 0) {
            col.columnType = `${col.dataType}(${col.charMaxLength})`;
            col.showLength = col.charMaxLength;
            col.showScale = null;
            continue;
        }
        if (col.numPrecision && col.numPrecision > 0) {
            if (col.numScale && col.numScale > 0) {
                col.columnType = `${col.dataType}(${col.numPrecision},${col.numScale})`;
                col.showScale = col.numScale;
            } else {
                col.columnType = `${col.dataType}(${col.numPrecision})`;
                col.showScale = null;
            }

            col.showLength = col.numPrecision;
            continue;
        }

        col.columnType = col.dataType;
    }
}
