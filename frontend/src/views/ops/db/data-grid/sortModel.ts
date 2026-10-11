/**
 * 网格的有序多列排序模型。
 *
 * 排序列的顺序就是 SQL 里 `ORDER BY a, b` 的先后，属于用户显式建立的事实，因此这里
 * 既不能按列名字典序重排（会把用户点的顺序改掉），也不能让后点的列顶掉先点的列
 * （单列排序的老行为）。标识符引用交由调用方传入的 quote 完成：引用规则属于方言，
 * 在这里写死双引号会让 MySQL 的 SQL 直接语法错误。
 */

/** 用户可下达的排序动作；none 表示把该列从排序列表里撤掉 */
export type SortAction = 'asc' | 'desc' | 'none';

export interface ColumnSort {
    key: string;
    order: 'asc' | 'desc';
}

/**
 * 应用一次排序动作，返回新的有序列表。
 *
 * - 未参与排序的列：追加到末尾（顺序即生效优先级）
 * - 已参与的列：只改方向，**位置不变**（先点的列仍是主排序键）
 * - none：移除该列，其余列的相对顺序保持
 */
export function applySort(sorts: ColumnSort[], key: string, action: SortAction): ColumnSort[] {
    const index = sorts.findIndex((item) => item.key === key);
    if (action === 'none') {
        return index < 0 ? [...sorts] : sorts.filter((item) => item.key !== key);
    }
    if (index < 0) {
        return [...sorts, { key, order: action }];
    }
    if (sorts[index].order === action) {
        return [...sorts];
    }
    return sorts.map((item, i) => (i === index ? { key, order: action } : item));
}

/** 该列的排序序号（1 起）；0 表示未参与排序 */
export function sortOrderIndex(sorts: ColumnSort[], key: string): number {
    return sorts.findIndex((item) => item.key === key) + 1;
}

/** 拼 ORDER BY 子句；无排序列时返回空串（由调用方决定整段省略，而不是写个合法的废话） */
export function buildOrderByClause(sorts: ColumnSort[], quote: (name: string) => string): string {
    if (!sorts.length) {
        return '';
    }
    return `ORDER BY ${sorts.map((item) => `${quote(item.key)} ${item.order === 'desc' ? 'DESC' : 'ASC'}`).join(', ')}`;
}
