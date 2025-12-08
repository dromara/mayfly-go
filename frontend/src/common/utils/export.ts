import * as XLSX from 'xlsx';

/**
 * 导出CSV文件
 * @param filename 文件名
 * @param columns 列信息
 * @param datas 数据
 */
export function exportCsv(filename: string, columns: string[], datas: []) {
    // 二维数组
    const cvsData = [columns];
    for (let data of datas) {
        // 数据值组成的一维数组
        let dataValueArr: any = [];
        for (let column of columns) {
            let val: any = data[column];
            if (val == null || val == undefined) {
                val = '';
            } else if (val && typeof val == 'string') {
                // 替换换行符
                val = val.replace(/[\r\n]/g, '\\n');

                // csv格式如果有逗号，整体用双引号括起来；如果里面还有双引号就替换成两个双引号，这样导出来的格式就不会有问题了
                if (val.indexOf(',') != -1) {
                    // 如果还有双引号，先将双引号转义，避免两边加了双引号后转义错误
                    if (val.indexOf('"') != -1) {
                        val = val.replace(/"/g, '""');
                    }
                    // 再将逗号转义
                    val = `"${val}"`;
                }
            }
            dataValueArr.push(String(val));
        }
        cvsData.push(dataValueArr);
    }
    const csvString = cvsData.map((e) => e.join(',')).join('\n');
    exportFile(`${filename}.csv`, csvString);
}

/**
 * 导出文件
 * @param filename 文件名
 * @param content 文件内容
 */
export function exportFile(filename: string, content: string) {
    // 导出
    let link = document.createElement('a');
    let exportContent = '\uFEFF';
    let blob = new Blob([exportContent + content], {
        type: 'text/plain;charset=utf-8',
    });
    link.id = 'download-file';
    link.setAttribute('href', URL.createObjectURL(blob));
    link.setAttribute('download', `${filename}`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link); // 下载完成后移除元素
}

/**
 * 计算字符串显示宽度（考虑中英文字符差异）
 * @param str 要计算的字符串
 * @returns 计算后的宽度值
 */
function getStringWidth(str: string): number {
    if (!str) return 0;

    // 统计中文字符数量（包括中文标点）
    const chineseChars = str.match(/[\u4e00-\u9fa5\u3000-\u303f\uff00-\uffef]/g);
    const chineseCount = chineseChars ? chineseChars.length : 0;

    // 英文字符数量
    const englishCount = str.length - chineseCount;

    // 中文字符按2个单位宽度计算，英文字符按1个单位宽度计算
    return chineseCount * 2 + englishCount;
}

/**
 * 导出Excel文件
 * @param filename 文件名
 * @param sheets 多个工作表数据，每个工作表包含名称、列信息和数据
 * 示例: [{name: 'Sheet1', columns: ['列1', '列2'], datas: [{col1: '值1', col2: '值2'}]}]
 */
export function exportExcel(filename: string, sheets: { name: string; columns: string[]; datas: any[] }[]) {
    // 创建工作簿
    const wb = XLSX.utils.book_new();

    // 处理每个工作表
    sheets.forEach((sheet) => {
        // 准备表头
        const headers: any = {};
        sheet.columns.forEach((col) => {
            headers[col] = col;
        });

        // 准备数据
        const data = [headers, ...sheet.datas];

        // 创建工作表
        const ws = XLSX.utils.json_to_sheet(data, { skipHeader: true });

        // 设置列宽自适应
        const colWidths: { wch: number }[] = [];
        sheet.columns.forEach((col, index) => {
            // 计算列宽：取表头和前几行数据的最大宽度
            let maxWidth = getStringWidth(col); // 表头宽度
            const checkCount = Math.min(sheet.datas.length, 10); // 只检查前10行数据

            for (let i = 0; i < checkCount; i++) {
                const cellData = sheet.datas[i][col];
                const cellStr = cellData ? String(cellData) : '';
                const cellWidth = getStringWidth(cellStr);
                if (cellWidth > maxWidth) {
                    maxWidth = cellWidth;
                }
            }

            // 设置最小宽度为8，最大宽度为80
            colWidths.push({ wch: Math.min(Math.max(maxWidth + 2, 8), 80) });
        });

        // 应用列宽设置
        ws['!cols'] = colWidths;

        // 添加工作表到工作簿
        XLSX.utils.book_append_sheet(wb, ws, sheet.name);
    });

    // 导出文件
    XLSX.writeFile(wb, `${filename}.xlsx`);
}
