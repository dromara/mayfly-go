/**
 * JSON 条件表达式安全求值器
 *
 * 纯手写比较逻辑，不使用 eval / new Function，条件定义可由后端或 AI 安全下发。
 */
import type { JsonCondition, JsonGroupCondition, JsonSimpleCondition } from './schema';

/** 判断是否为组合条件（否则视为简单条件） */
export const isGroupCondition = (cond: JsonCondition): cond is JsonGroupCondition => {
    return 'all' in cond || 'any' in cond || 'not' in cond;
};

/** 宽松相等：'1' 与 1 相等，null 与 undefined 相等；布尔值仅严格比较 */
const looseEq = (a: unknown, b: unknown): boolean => {
    if (a === b) {
        return true;
    }
    const aEmpty = a === undefined || a === null;
    const bEmpty = b === undefined || b === null;
    if (aEmpty || bEmpty) {
        return aEmpty && bEmpty;
    }
    if (typeof a === 'boolean' || typeof b === 'boolean') {
        return false;
    }
    return String(a) === String(b);
};

/** 值为空：undefined / null / 空字符串 / 空数组 */
export const isEmptyValue = (value: unknown): boolean => {
    if (value === undefined || value === null || value === '') {
        return true;
    }
    return Array.isArray(value) && value.length === 0;
};

/** 转为数值，无法转换（NaN）时返回 null */
const numOf = (value: unknown): number | null => {
    if (typeof value === 'number') {
        return Number.isNaN(value) ? null : value;
    }
    if (typeof value === 'string' && value.trim() !== '') {
        const n = Number(value);
        return Number.isNaN(n) ? null : n;
    }
    return null;
};

/** 求值简单条件 */
const evalSimpleCondition = (cond: JsonSimpleCondition, form: Record<string, unknown>): boolean => {
    const value = form[cond.field];
    switch (cond.op) {
        case 'eq':
            return looseEq(value, cond.value);
        case 'ne':
            return !looseEq(value, cond.value);
        case 'in':
            return Array.isArray(cond.value) && cond.value.some((v) => looseEq(value, v));
        case 'notIn':
            return !Array.isArray(cond.value) || !cond.value.some((v) => looseEq(value, v));
        case 'empty':
            return isEmptyValue(value);
        case 'notEmpty':
            return !isEmptyValue(value);
        case 'gt':
        case 'gte':
        case 'lt':
        case 'lte': {
            const left = numOf(value);
            const right = numOf(cond.value);
            if (left === null || right === null) {
                return false;
            }
            switch (cond.op) {
                case 'gt':
                    return left > right;
                case 'gte':
                    return left >= right;
                case 'lt':
                    return left < right;
                default:
                    return left <= right;
            }
        }
        default:
            return false;
    }
};

/**
 * 求值条件表达式
 *
 * @param cond 条件定义（undefined 时恒为 true，便于编译层直接透传可选条件）
 * @param form 当前表单数据
 */
export const evalCondition = (cond: JsonCondition | undefined, form: Record<string, unknown>): boolean => {
    if (!cond) {
        return true;
    }
    if (isGroupCondition(cond)) {
        if (cond.all && !cond.all.every((c) => evalCondition(c, form))) {
            return false;
        }
        if (cond.any && !cond.any.some((c) => evalCondition(c, form))) {
            return false;
        }
        if (cond.not && evalCondition(cond.not, form)) {
            return false;
        }
        return true;
    }
    return evalSimpleCondition(cond, form);
};
