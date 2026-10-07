import { describe, expect, test } from 'vitest';

import type { MongoDoc } from '../types';
import { displayCell } from '../docview/extjson';
import {
    columnValue,
    columnWidthOf,
    COLUMN_MAX_WIDTH,
    COLUMN_MIN_WIDTH,
    DEFAULT_MAX_COLUMNS,
    inferColumns,
    isNumericColumn,
    nextSort,
    withEquality,
} from '../docview/schema';

function doc(source: Record<string, unknown>): MongoDoc {
    return { idToken: 't', idKind: 'objectId', hash: 'h', mode: 'plain', doc: source };
}

describe('inferColumns', () => {
    test('按首次出现顺序成列，_id 排在前', () => {
        const docs = [doc({ _id: '1', name: 'a', qty: 2 }), doc({ _id: '2', name: 'b' })];
        const { columns } = inferColumns(docs);
        expect(columns.map((column) => column.key)).toEqual(['_id', 'name', 'qty']);
    });

    test('低覆盖率字段进 optional 而不是被丢掉', () => {
        const docs = [doc({ _id: '1', common: 1 }), doc({ _id: '2', common: 2 }), doc({ _id: '3', common: 3 }), doc({ _id: '4', rare: 9 })];
        const { columns, optional } = inferColumns(docs);

        expect(columns.map((column) => column.key)).toEqual(['_id', 'common']);
        expect(optional.map((column) => column.key)).toEqual(['rare']);
        expect(optional[0].coverage).toBeCloseTo(0.25);
    });

    test('列数超过上限后，剩下的字段仍可被选回', () => {
        const wide = doc(Object.fromEntries(Array.from({ length: DEFAULT_MAX_COLUMNS + 4 }, (_, i) => [`f${i}`, i])));
        const docs = Array.from({ length: 3 }, () => ({ ...wide }));
        const { columns, optional } = inferColumns(docs);

        expect(columns.length).toBe(DEFAULT_MAX_COLUMNS);
        expect(optional.length).toBe(4);
    });

    test('固定的列即使覆盖率低也必须出现', () => {
        const docs = [doc({ _id: '1', a: 1 }), doc({ _id: '2', a: 2 }), doc({ _id: '3', a: 3 }), doc({ _id: '4', b: 1 })];
        const { columns, optional } = inferColumns(docs, { pinned: ['b'] });

        expect(columns.map((column) => column.key)).toContain('b');
        expect(optional.map((column) => column.key)).not.toContain('b');
    });

    test('同列出现多种类型时都记下来（异构字段需要在列头看出来）', () => {
        const docs = [doc({ _id: '1', v: 'text' }), doc({ _id: '2', v: 3 })];
        const { columns } = inferColumns(docs);
        const value = columns.find((column) => column.key === 'v');

        expect(value?.kinds).toEqual(['String', 'Number']);
    });

    test('类型包装列出的 kind 是 BSON 类型名而不是 Document', () => {
        const docs = [doc({ _id: '1', at: { $date: { $numberLong: '1760000000000' } } })];
        const { columns } = inferColumns(docs);

        expect(columns.find((column) => column.key === 'at')?.kinds).toEqual(['Date']);
    });

    test('空结果不炸', () => {
        expect(inferColumns([])).toEqual({ columns: [], optional: [] });
    });
});

describe('nextSort', () => {
    test('升序 → 降序 → 取消，且不动其他排序字段', () => {
        let sort: Record<string, unknown> = { createdAt: -1 };

        sort = nextSort(sort, 'qty');
        expect(sort).toEqual({ createdAt: -1, qty: 1 });

        sort = nextSort(sort, 'qty');
        expect(sort).toEqual({ createdAt: -1, qty: -1 });

        sort = nextSort(sort, 'qty');
        expect(sort).toEqual({ createdAt: -1 });
    });

    test('已有同名字段时保持优先级位置不变（键序即排序优先级）', () => {
        const sort = nextSort({ a: 1, b: -1 }, 'a');
        expect(Object.keys(sort)).toEqual(['a', 'b']);
        expect(sort.a).toBe(-1);
    });
});

describe('columnValue / displayCell / withEquality', () => {
    test('缺失列返回 undefined，由展示层显示空', () => {
        expect(columnValue(doc({ _id: '1' }), 'nope')).toBeUndefined();
        expect(displayCell(undefined)).toBe('null');
    });

    test('单元格里的类型包装转成可读值', () => {
        expect(displayCell({ $oid: '507f1f77bcf86cd799439011' })).toBe('507f1f77bcf86cd799439011');
        expect(displayCell({ $numberDecimal: '129.005' })).toBe('129.005');
        expect(displayCell({ $binary: { base64: 'AAEC', subType: '04' } })).toBe('BinData(4, 3 bytes)');
    });

    test('嵌套文档与数组里的类型包装也要转成可读值', () => {
        const cell = displayCell({ buyer: { name: 'momo' }, at: { $date: { $numberLong: '1760000000000' } } });
        const parsed = JSON.parse(cell) as Record<string, unknown>;

        expect(parsed.buyer).toEqual({ name: 'momo' });
        expect(String(parsed.at)).toMatch(/^202\d-\d{2}-\d{2} /);
        expect(cell).not.toContain('$numberLong');
    });

    test('数组单元格保持顺序', () => {
        expect(displayCell([1, 'x', true])).toBe('[1,"x",true]');
    });

    test('按字段过滤只追加条件，不清空已有条件', () => {
        expect(withEquality({ status: 'paid' }, 'buyer.level', 2)).toEqual({ status: 'paid', 'buyer.level': 2 });
    });
});

describe('columnWidthOf', () => {
    /** 注入测宽函数：脱开浏览器与 canvas 也能断言相对关系（每字符 8px） */
    const measure = (text: string) => text.length * 8;
    const metrics = (over = {}) => ({ measureCell: measure, ...over });
    const col = (key: string, kinds: string[] = ['String']) => ({ key, kinds, coverage: 1 });

    test('内容越长列越宽，短内容列不会被撑成空白', () => {
        const shortDocs = [doc({ status: 'paid' })];
        const longDocs = [doc({ status: 'a'.repeat(30) })];
        expect(columnWidthOf(col('status'), longDocs, metrics())).toBeGreaterThan(columnWidthOf(col('status'), shortDocs, metrics()));
    });

    test('表头按自己的字重量：粗体更宽时列宽跟着表头走', () => {
        // 正文只有 1 个字符，表头 12 字符且按 2 倍宽量 → 列宽必须由表头决定
        const width = columnWidthOf(col('createdAt'), [doc({ createdAt: 'a' })], metrics({ measureHeader: (text: string) => text.length * 16 }));
        expect(width).toBeGreaterThanOrEqual('createdAt'.length * 16);
    });

    test('内容为空时按表头与下限兜底', () => {
        const width = columnWidthOf(col('createdAt'), [doc({ createdAt: null })], metrics());
        expect(width).toBeGreaterThanOrEqual('createdAt'.length * 8 + 40);
        expect(width).toBeGreaterThanOrEqual(COLUMN_MIN_WIDTH);
    });

    test('嵌套文档等超长值有上限，不会把整表挤成一列可见', () => {
        const huge = [doc({ payload: { blob: 'x'.repeat(2000) } })];
        expect(columnWidthOf(col('payload', ['Document']), huge, metrics())).toBe(COLUMN_MAX_WIDTH);
    });

    test('主键列在存在 BSON 类型文档时额外留出徽标宽度', () => {
        const hex = '670000000000000000000001';
        const withBson = [doc({ _id: hex }), { ...doc({ _id: hex }), mode: 'extjson' } as MongoDoc];
        const plain = [doc({ _id: hex })];
        expect(columnWidthOf(col('_id', ['ObjectID']), withBson, metrics({ idBadgeWidth: 44 }))).toBeGreaterThan(
            columnWidthOf(col('_id', ['ObjectID']), plain, metrics({ idBadgeWidth: 44 }))
        );
    });

    test('缺字段的行不参与计宽（否则每列都被空占位拉平）', () => {
        const onlyMissing = [doc({ other: 1 }), doc({ other: 2 })];
        const width = columnWidthOf(col('note'), onlyMissing, metrics());
        expect(width).toBeGreaterThanOrEqual(COLUMN_MIN_WIDTH);
        expect(width).toBeLessThan(COLUMN_MAX_WIDTH);
    });

    test('量宽不可用（字体未就绪）时退回下限，不谎报成放得下', () => {
        const huge = [doc({ payload: 'x'.repeat(500) })];
        expect(columnWidthOf(col('payload'), huge, metrics({ measureCell: () => 0 }))).toBe(COLUMN_MIN_WIDTH);
    });
});

describe('isNumericColumn', () => {
    test('整列都是数值类型才右对齐', () => {
        expect(isNumericColumn({ key: 'qty', kinds: ['Int32'], coverage: 1 })).toBe(true);
        expect(isNumericColumn({ key: 'amount', kinds: ['Decimal128', 'Double'], coverage: 1 })).toBe(true);
    });

    test('混合类型列不右对齐（只有部分是数字，对齐反而乱）', () => {
        expect(isNumericColumn({ key: 'v', kinds: ['String', 'Number'], coverage: 1 })).toBe(false);
        expect(isNumericColumn({ key: 'at', kinds: ['Date'], coverage: 1 })).toBe(false);
        expect(isNumericColumn({ key: 'x', kinds: [], coverage: 1 })).toBe(false);
    });
});
