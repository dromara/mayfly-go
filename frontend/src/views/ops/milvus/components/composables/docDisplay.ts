import type { Ref } from 'vue';

/**
 * Milvus 文档表格的显示域：单元格格式化、字段角色判定（主键/动态/向量）、列宽与标签、URL 识别与点击。
 * 主键字段是唯一的响应式输入（由宿主传入），其余为纯函数，便于复用与单测。
 */
export function useDocDisplay(primaryKey: Ref<string>) {
    // 格式化单元格值
    const formatCellValue = (value: unknown, field: string): string => {
        if (value === null || value === undefined) {
            return '';
        }

        // 动态字段
        if (field === 'dynamicFields') {
            if (typeof value === 'object') {
                return JSON.stringify(value);
            }
            return String(value);
        }

        // 向量字段
        if (Array.isArray(value)) {
            const str = JSON.stringify(value);
            return str.length > 100 ? str.substring(0, 100) + '...' : str;
        }
        // 其余对象数据
        if (typeof value === 'object') {
            return JSON.stringify(value);
        }
        return String(value);
    };

    // 判断是否为主键
    const isPrimaryKey = (field: string) => {
        return field === primaryKey.value;
    };

    // 判断是否为动态字段
    const isDynamicField = (field: string) => {
        return field === 'dynamicFields';
    };

    // 判断是否为向量字段
    const isVectorField = (field: string) => {
        return field === 'vector' || field.toLowerCase().includes('vector');
    };

    // 获取最小列宽
    const getMinWidth = (field: string) => {
        if (isVectorField(field)) return 300;
        if (isDynamicField(field)) return 250;
        if (isPrimaryKey(field)) return 180;
        return 150;
    };

    // 获取显示标签
    const getDisplayLabel = (field: string) => {
        // 主键字段显示 # 前缀
        if (isPrimaryKey(field)) {
            return `# ${field}`;
        }
        // 向量字段显示 ~ 前缀
        if (isVectorField(field)) {
            return `${field}`;
        }
        // 动态字段显示 "" 前缀
        if (isDynamicField(field)) {
            return `"${field}`;
        }
        return field;
    };

    // 判断是否为 URL
    const isUrl = (value: unknown): value is string => {
        return typeof value === 'string' && (value.startsWith('http://') || value.startsWith('https://'));
    };

    // 单元格点击事件
    const handleCellClick = (value: unknown, field: string) => {
        if (isUrl(value)) {
            window.open(value, '_blank');
        }
    };

    return { formatCellValue, isPrimaryKey, isDynamicField, isVectorField, getMinWidth, getDisplayLabel, isUrl, handleCellClick };
}
