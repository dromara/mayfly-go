import { describe, expect, it } from 'vitest';
import { evalCondition, isEmptyValue } from '../condition';
import type { JsonCondition } from '../schema';

describe('evalCondition - 简单条件', () => {
    const form = { env: 'prod', count: 3, name: 'demo', flag: true };

    it('eq 宽松相等：字符串与数字互通', () => {
        expect(evalCondition({ field: 'env', op: 'eq', value: 'prod' }, form)).toBe(true);
        expect(evalCondition({ field: 'count', op: 'eq', value: '3' }, form)).toBe(true);
        expect(evalCondition({ field: 'env', op: 'eq', value: 'dev' }, form)).toBe(false);
    });

    it('eq 的 null 与 undefined 视为相等', () => {
        expect(evalCondition({ field: 'missing', op: 'eq', value: null }, {})).toBe(true);
        expect(evalCondition({ field: 'missing', op: 'eq', value: '' }, {})).toBe(false);
    });

    it('ne', () => {
        expect(evalCondition({ field: 'env', op: 'ne', value: 'dev' }, form)).toBe(true);
        expect(evalCondition({ field: 'env', op: 'ne', value: 'prod' }, form)).toBe(false);
    });

    it('in / notIn', () => {
        expect(evalCondition({ field: 'env', op: 'in', value: ['prod', 'staging'] }, form)).toBe(true);
        expect(evalCondition({ field: 'env', op: 'in', value: ['dev'] }, form)).toBe(false);
        expect(evalCondition({ field: 'env', op: 'notIn', value: ['dev'] }, form)).toBe(true);
        expect(evalCondition({ field: 'env', op: 'notIn', value: ['prod'] }, form)).toBe(false);
        // value 非数组时 in 恒为 false、notIn 恒为 true
        expect(evalCondition({ field: 'env', op: 'in', value: 'prod' }, form)).toBe(false);
        expect(evalCondition({ field: 'env', op: 'notIn', value: 'prod' }, form)).toBe(true);
    });

    it('empty / notEmpty', () => {
        expect(evalCondition({ field: 'x', op: 'empty' }, {})).toBe(true);
        expect(evalCondition({ field: 'x', op: 'empty' }, { x: '' })).toBe(true);
        expect(evalCondition({ field: 'x', op: 'empty' }, { x: [] })).toBe(true);
        expect(evalCondition({ field: 'name', op: 'empty' }, form)).toBe(false);
        expect(evalCondition({ field: 'name', op: 'notEmpty' }, form)).toBe(true);
    });

    it('gt/gte/lt/lte 数值比较（字符串数字可比较）', () => {
        expect(evalCondition({ field: 'count', op: 'gt', value: 2 }, form)).toBe(true);
        expect(evalCondition({ field: 'count', op: 'gte', value: 3 }, form)).toBe(true);
        expect(evalCondition({ field: 'count', op: 'lt', value: '5' }, form)).toBe(true);
        expect(evalCondition({ field: 'count', op: 'lte', value: 2 }, form)).toBe(false);
    });

    it('数值比较无法转换时恒为 false', () => {
        expect(evalCondition({ field: 'env', op: 'gt', value: 1 }, form)).toBe(false);
        expect(evalCondition({ field: 'missing', op: 'lt', value: 1 }, form)).toBe(false);
    });

    it('布尔值仅严格比较', () => {
        expect(evalCondition({ field: 'flag', op: 'eq', value: true }, form)).toBe(true);
        expect(evalCondition({ field: 'flag', op: 'eq', value: 'true' }, form)).toBe(false);
    });
});

describe('evalCondition - 组合条件', () => {
    const form = { env: 'prod', count: 3 };

    it('all 与组合', () => {
        const cond: JsonCondition = {
            all: [
                { field: 'env', op: 'eq', value: 'prod' },
                { field: 'count', op: 'gte', value: 3 },
            ],
        };
        expect(evalCondition(cond, form)).toBe(true);
        expect(evalCondition({ all: [{ field: 'env', op: 'eq', value: 'dev' }, { field: 'count', op: 'gte', value: 3 }] }, form)).toBe(false);
    });

    it('any 或组合', () => {
        const cond: JsonCondition = {
            any: [
                { field: 'env', op: 'eq', value: 'dev' },
                { field: 'count', op: 'gt', value: 2 },
            ],
        };
        expect(evalCondition(cond, form)).toBe(true);
        expect(evalCondition({ any: [{ field: 'env', op: 'eq', value: 'dev' }] }, form)).toBe(false);
    });

    it('not 非组合', () => {
        expect(evalCondition({ not: { field: 'env', op: 'eq', value: 'dev' } }, form)).toBe(true);
        expect(evalCondition({ not: { field: 'env', op: 'eq', value: 'prod' } }, form)).toBe(false);
    });

    it('嵌套组合', () => {
        const cond: JsonCondition = {
            all: [
                { field: 'env', op: 'eq', value: 'prod' },
                { any: [{ field: 'count', op: 'lt', value: 2 }, { field: 'count', op: 'gt', value: 2 }] },
            ],
        };
        expect(evalCondition(cond, form)).toBe(true);
    });

    it('undefined 条件恒为 true', () => {
        expect(evalCondition(undefined, form)).toBe(true);
    });
});

describe('isEmptyValue', () => {
    it('覆盖空值形态', () => {
        expect(isEmptyValue(undefined)).toBe(true);
        expect(isEmptyValue(null)).toBe(true);
        expect(isEmptyValue('')).toBe(true);
        expect(isEmptyValue([])).toBe(true);
        expect(isEmptyValue(0)).toBe(false);
        expect(isEmptyValue(false)).toBe(false);
        expect(isEmptyValue([1])).toBe(false);
    });
});
