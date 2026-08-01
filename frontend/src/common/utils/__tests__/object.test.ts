import { describe, expect, it } from 'vitest';
import { deepClone, getValueByPath, setValueByPath } from '../object';

describe('getValueByPath', () => {
    const obj = {
        user: { name: 'mayfly', age: 3 },
        orderNo: 1212211,
        products: [{ id: 12 }, { id: 24 }],
        jsonStr: '{"nested": "value"}',
    } as unknown as Record<string, unknown>;

    it('简单路径', () => {
        expect(getValueByPath(obj, 'orderNo')).toBe(1212211);
    });

    it('嵌套路径', () => {
        expect(getValueByPath(obj, 'user.name')).toBe('mayfly');
    });

    it('数组索引路径', () => {
        expect(getValueByPath(obj, 'products[0].id')).toBe(12);
        expect(getValueByPath(obj, 'products[1].id')).toBe(24);
    });

    it('JSON 字符串自动解析', () => {
        expect(getValueByPath(obj, 'jsonStr.nested')).toBe('value');
    });

    it('不存在的路径返回 undefined', () => {
        expect(getValueByPath(obj, 'not.exist')).toBeUndefined();
        expect(getValueByPath(obj, 'user.notExist')).toBeUndefined();
    });

    it('数组越界返回 undefined', () => {
        expect(getValueByPath(obj, 'products[99].id')).toBeUndefined();
    });
});

describe('setValueByPath', () => {
    it('设置嵌套值（路径已存在）', () => {
        const obj: Record<string, unknown> = { a: { b: 1 } };
        setValueByPath(obj, ['a', 'b'], 2);
        expect((obj.a as Record<string, unknown>).b).toBe(2);
    });

    it('自动创建不存在的路径', () => {
        const obj: Record<string, unknown> = {};
        setValueByPath(obj, ['x', 'y', 'z'], 'deep');
        expect(((obj.x as Record<string, unknown>).y as Record<string, unknown>).z).toBe('deep');
    });
});

describe('deepClone', () => {
    it('深度克隆对象', () => {
        const src = { a: 1, nested: { b: [1, 2, 3] } };
        const cloned = deepClone(src);
        expect(cloned).toEqual(src);
        expect(cloned).not.toBe(src);
        expect(cloned.nested).not.toBe(src.nested);
        expect(cloned.nested.b).not.toBe(src.nested.b);
    });

    it('处理循环引用', () => {
        const src: Record<string, unknown> = { a: 1 };
        src.self = src;
        const cloned = deepClone(src);
        expect(cloned.self).toBe(cloned);
    });

    it('克隆 Date 和 RegExp', () => {
        const date = new Date('2024-01-01');
        const regex = /abc/gi;
        const cloned = deepClone({ date, regex });
        expect(cloned.date).toEqual(date);
        expect(cloned.date).not.toBe(date);
        expect(cloned.regex.source).toBe('abc');
    });

    it('支持回调转换值', () => {
        const src = { keep: 'yes', drop: '' };
        const cloned = deepClone(src, (_key, value) => (value === '' ? null : value));
        expect(cloned.drop).toBeNull();
        expect(cloned.keep).toBe('yes');
    });

    it('基本类型直接返回', () => {
        expect(deepClone(42)).toBe(42);
        expect(deepClone('str')).toBe('str');
        expect(deepClone(null)).toBeNull();
    });
});
