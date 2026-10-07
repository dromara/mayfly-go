import { describe, expect, test } from 'vitest';

import { collectTypedFields, displayWrapper, parseDocText, prettyDoc, typedFieldsSummary, wrapperTypeOf } from '../docview/extjson';

describe('wrapperTypeOf', () => {
    test('单键 $ 包装识别为对应 BSON 类型', () => {
        expect(wrapperTypeOf({ $oid: '507f1f77bcf86cd799439011' })).toBe('ObjectID');
        expect(wrapperTypeOf({ $date: '2026-01-02T03:04:05.000Z' })).toBe('Date');
        expect(wrapperTypeOf({ $numberLong: '9007199254740993' })).toBe('NumberLong');
        expect(wrapperTypeOf({ $numberDecimal: '129.005' })).toBe('Decimal128');
        expect(wrapperTypeOf({ $binary: { base64: 'AAEC', subType: '04' } })).toBe('Binary');
        expect(wrapperTypeOf({ $timestamp: { t: 1700000000, i: 3 } })).toBe('Timestamp');
        expect(wrapperTypeOf({ $regularExpression: { pattern: '^a', options: 'i' } })).toBe('Regex');
        expect(wrapperTypeOf({ $minKey: {} })).toBe('MinKey');
    });

    test('普通文档与多键文档不是类型包装', () => {
        expect(wrapperTypeOf({ name: 'x' })).toBeUndefined();
        expect(wrapperTypeOf({ $oid: 'x', extra: 1 })).toBeUndefined();
        expect(wrapperTypeOf({ $unknownOp: 1 })).toBeUndefined();
        expect(wrapperTypeOf('text')).toBeUndefined();
        expect(wrapperTypeOf(null)).toBeUndefined();
        expect(wrapperTypeOf([{ $oid: 'x' }])).toBeUndefined();
    });
});

describe('displayWrapper', () => {
    test('日期毫秒与 ISO 两种形态都能读', () => {
        expect(displayWrapper('$date', '2026-01-02T03:04:05.000Z')).toBe('2026-01-02T03:04:05.000Z');
        expect(displayWrapper('$date', 1700000000000)).toMatch(/^202\d-\d{2}-\d{2} /);
    });

    test('Binary 显示子类型与字节数而不是整段 base64', () => {
        expect(displayWrapper('$binary', { base64: 'AAEC', subType: '04' })).toBe('BinData(4, 3 bytes)');
        expect(displayWrapper('$binary', { base64: '', subType: '00' })).toBe('BinData(0, 0 bytes)');
    });

    test('非有限浮点原样显示，不塌成 0 或空', () => {
        expect(displayWrapper('$numberDouble', 'NaN')).toBe('NaN');
        expect(displayWrapper('$numberDouble', '-Infinity')).toBe('-Infinity');
    });

    test('正则与时间戳给出结构化的可读形式', () => {
        expect(displayWrapper('$regularExpression', { pattern: '^a[0-9]+$', options: 'i' })).toBe('/^a[0-9]+$/i');
        expect(displayWrapper('$timestamp', { t: 1700000000, i: 3 })).toBe('Timestamp(1700000000, 3)');
    });
});

describe('collectTypedFields', () => {
    test('递归到数组与子文档并给出点号路径', () => {
        const doc = {
            _id: { $oid: '507f1f77bcf86cd799439011' },
            meta: { createdAt: { $date: '2026-01-02T03:04:05.000Z' } },
            items: [{ price: { $numberDecimal: '129.005' } }, { price: 1.5 }],
        };

        const fields = collectTypedFields(doc);
        expect(fields.map((f) => f.path)).toEqual(['_id', 'meta.createdAt', 'items[0].price']);
        expect(fields.map((f) => f.type)).toEqual(['ObjectID', 'Date', 'Decimal128']);
        expect(fields[2].display).toBe('129.005');
    });

    test('普通 JSON 文档没有类型标注', () => {
        expect(collectTypedFields({ name: 'x', n: 1, list: [1, 2], sub: { ok: true } })).toEqual([]);
    });
});

describe('typedFieldsSummary', () => {
    test('同类型聚合计数', () => {
        const doc = {
            _id: { $oid: 'a' },
            userId: { $oid: 'b' },
            createdAt: { $date: '2026-01-02T03:04:05.000Z' },
        };
        expect(typedFieldsSummary(collectTypedFields(doc))).toBe('ObjectID ×2、Date ×1');
    });

    test('无标注返回空串（前端据此不渲染提示条）', () => {
        expect(typedFieldsSummary(collectTypedFields({ a: 1 }))).toBe('');
    });
});

describe('prettyDoc / parseDocText', () => {
    test('缩进后仍可被解析回同一内容', () => {
        const doc = { _id: { $oid: 'x' }, n: 1, list: [1, 'a'] };
        const text = prettyDoc(doc);
        expect(text).toContain('\n    ');
        expect(parseDocText(text)).toEqual(doc);
    });

    test('非法 JSON 与空文本都返回 null，调用方据此报错而不是当空文档', () => {
        expect(parseDocText('{"a":')).toBeNull();
        expect(parseDocText('   ')).toBeNull();
    });

    test('缺字段与 null 文档都稳定输出', () => {
        expect(prettyDoc(undefined)).toBe('null');
        expect(prettyDoc(null)).toBe('null');
    });
});
