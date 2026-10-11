import { describe, expect, it } from 'vitest';
import { applySort, buildOrderByClause, sortOrderIndex, type ColumnSort } from '../sortModel';

/**
 * 有序多列排序模型的守卫
 *
 * 存在理由：列表顺序就是 SQL 里 `ORDER BY a, b` 的先后，属于用户显式建立的事实。
 * 任何「按列名重排」「后点的列顶掉先点的列」都会悄悄改掉用户定的优先级，
 * 而结果集看起来仍然有序，没人能发现排序键已经换了。
 */

const sorts = (...items: [string, 'asc' | 'desc'][]): ColumnSort[] => items.map(([key, order]) => ({ key, order }));

describe('applySort', () => {
    it('新列追加到末尾，顺序即生效优先级', () => {
        let list: ColumnSort[] = [];
        list = applySort(list, 'dept', 'asc');
        list = applySort(list, 'name', 'desc');
        expect(list.map((item) => item.key)).toEqual(['dept', 'name']);
    });

    it('已参与的列只改方向，位置保持（先点的仍是主排序键）', () => {
        const list = applySort(sorts(['dept', 'asc'], ['name', 'asc']), 'dept', 'desc');
        expect(list).toEqual([
            { key: 'dept', order: 'desc' },
            { key: 'name', order: 'asc' },
        ]);
    });

    it('撤掉中间一列后，其余列相对顺序不变', () => {
        const list = applySort(sorts(['a', 'asc'], ['b', 'desc'], ['c', 'asc']), 'b', 'none');
        expect(list).toEqual([
            { key: 'a', order: 'asc' },
            { key: 'c', order: 'asc' },
        ]);
    });

    it('不修改入参数组（响应式列表被就地改会绕过 Vue 的变更追踪）', () => {
        const origin = sorts(['a', 'asc']);
        applySort(origin, 'b', 'desc');
        applySort(origin, 'a', 'none');
        expect(origin).toEqual([{ key: 'a', order: 'asc' }]);
    });

    it('重复点同一方向、撤掉未参与排序的列都是空操作，不产生新数组项', () => {
        const origin = sorts(['a', 'asc']);
        expect(applySort(origin, 'a', 'asc')).toEqual(origin);
        expect(applySort(origin, 'zz', 'none')).toEqual(origin);
    });
});

describe('排序序号与 ORDER BY 拼装', () => {
    it('序号从 1 起，未参与排序为 0', () => {
        const list = sorts(['a', 'asc'], ['b', 'desc']);
        expect(sortOrderIndex(list, 'a')).toBe(1);
        expect(sortOrderIndex(list, 'b')).toBe(2);
        expect(sortOrderIndex(list, 'c')).toBe(0);
    });

    it('标识符引用由调用方给出（方言专有），方向按 SQL 关键字输出', () => {
        const clause = buildOrderByClause(sorts(['id', 'asc'], ['a b', 'desc']), (name) => `\`${name}\``);
        expect(clause).toBe('ORDER BY `id` ASC, `a b` DESC');
    });

    it('无排序列时返回空串，由调用方整段省略 ORDER BY', () => {
        expect(buildOrderByClause([], (name) => name)).toBe('');
    });
});
