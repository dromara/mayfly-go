/**
 * AutoForm 共享逻辑单元测试
 *
 * 覆盖 resolveFormItems（字段配置解析优先级）、isItemRequired（动态必填判定）、
 * cloneFormData（编辑数据深拷贝防御）、buildDefaultForm（新增态缺省值）四个收敛点。
 */
import { describe, expect, it, vi } from 'vitest';
import { cloneFormData, isItemRequired, resolveFormItems } from '../shared';
import { buildDefaultForm, type AutoFormItem } from '../types';

// mock i18n：resolveFormItems 间接依赖 json 编译层，避免单测环境初始化真实 i18n（依赖 localStorage）
vi.mock('@/i18n', () => ({
    i18n: {
        global: {
            t: (key: string) => key,
        },
    },
}));

vi.mock('@/common/request', () => ({
    default: {
        request: vi.fn(),
    },
}));

describe('resolveFormItems', () => {
    it('tabs 存在时合并所有 Tab 字段（优先于 items/schema）', () => {
        const items = resolveFormItems({
            items: [{ prop: 'a' }],
            tabs: [
                { name: 't1', label: 'T1', items: [{ prop: 'b' }] },
                { name: 't2', label: 'T2', items: [{ prop: 'c' }, { prop: 'd' }] },
            ],
        });
        expect(items.value.map((i) => i.prop)).toEqual(['b', 'c', 'd']);
    });

    it('无 tabs 时 schema 优先编译，其次 items', () => {
        const bySchema = resolveFormItems({
            schema: { version: 1, fields: [{ prop: 's1', label: 'S1' }] },
            items: [{ prop: 'a' }],
        });
        expect(bySchema.value.map((i) => i.prop)).toEqual(['s1']);

        const byItems = resolveFormItems({ items: [{ prop: 'a' }] });
        expect(byItems.value.map((i) => i.prop)).toEqual(['a']);
    });

    it('全空时返回空数组', () => {
        expect(resolveFormItems({}).value).toEqual([]);
    });
});

describe('isItemRequired', () => {
    it('布尔与缺省值', () => {
        expect(isItemRequired({ prop: 'a', required: true }, {})).toBe(true);
        expect(isItemRequired({ prop: 'a' }, {})).toBe(false);
        expect(isItemRequired({ prop: 'a', required: false }, {})).toBe(false);
    });

    it('函数形式根据表单值动态计算', () => {
        const item: AutoFormItem = { prop: 'b', required: (form) => form.enabled === 1 };
        expect(isItemRequired(item, { enabled: 1 })).toBe(true);
        expect(isItemRequired(item, { enabled: 0 })).toBe(false);
    });
});

describe('cloneFormData', () => {
    it('深拷贝：嵌套对象不再与源数据共享引用', () => {
        const source = { id: 1, meta: { icon: 'a', tags: ['x'] } };
        const cloned = cloneFormData(source);
        cloned.meta.icon = 'changed';
        cloned.meta.tags.push('y');
        expect(source.meta.icon).toBe('a');
        expect(source.meta.tags).toEqual(['x']);
    });

    it('含不可结构化克隆值（函数）时退化为浅拷贝', () => {
        const handler = () => {};
        const source = { handler, name: 'n' };
        const cloned = cloneFormData(source);
        expect(cloned.name).toBe('n');
        expect(cloned.handler).toBe(handler);
    });
});

describe('buildDefaultForm', () => {
    it('仅对有 defaultValue 的字段赋值，multiple 缺省为空数组', () => {
        const form = buildDefaultForm([
            { prop: 'name', defaultValue: 'x' },
            { prop: 'empty' },
            { prop: 'tags', type: 'select', multiple: true },
        ] as AutoFormItem[]);
        expect(form).toEqual({ name: 'x', tags: [] });
    });

    // el-switch 挂载时 model-value 必须是 active/inactive 值之一，新增态缺省值为关闭态而非 undefined
    it('switch 无 defaultValue 时取关闭态值（兼容驼峰与 kebab props）', () => {
        const form = buildDefaultForm([
            { prop: 'boolSwitch', type: 'switch' },
            { prop: 'numStatus', type: 'switch', props: { activeValue: 1, inactiveValue: 0 } },
            { prop: 'kebabStatus', type: 'switch', props: { 'active-value': 1, 'inactive-value': -1 } },
            { prop: 'withDefault', type: 'switch', defaultValue: 1, props: { activeValue: 1, inactiveValue: 0 } },
        ] as AutoFormItem[]);
        expect(form).toEqual({ boolSwitch: false, numStatus: 0, kebabStatus: -1, withDefault: 1 });
    });
});
