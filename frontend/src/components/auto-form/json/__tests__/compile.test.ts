import { describe, expect, it, vi } from 'vitest';
import { compileJsonField, compileJsonForm } from '../compile';
import { buildDefaultForm, CONTROL_REGISTRY, type AutoFormItem } from '../../types';
import type { JsonField } from '../schema';

// mock request 封装：optionsSource 编译依赖它
vi.mock('@/common/request', () => ({
    default: {
        request: vi.fn(),
    },
}));

// mock i18n：避免单测环境初始化真实 i18n（依赖 localStorage）
vi.mock('@/i18n', () => ({
    i18n: {
        global: {
            t: (key: string) => key,
        },
    },
}));

import request from '@/common/request';

/** 调用编译产物的异步 options 函数 */
const loadOptions = (item: AutoFormItem, form: Record<string, unknown> = {}) =>
    (item.options as (form: Record<string, unknown>) => Promise<unknown[]>)(form);

describe('compileJsonField', () => {
    it('基础字段映射', () => {
        const item = compileJsonField({ prop: 'name', label: 'common.name', type: 'password', placeholder: 'common.pleaseInput', defaultValue: 'a' })!;
        expect(item).toMatchObject({ prop: 'name', label: 'common.name', type: 'password', placeholder: 'common.pleaseInput', defaultValue: 'a' });
        expect(item.required).toBeUndefined();
        expect(item.when).toBeUndefined();
    });

    it('type 缺省视为 input', () => {
        expect(compileJsonField({ prop: 'a' })!.type).toBe('input');
    });

    it('rules.required 提升为 item.required，其余规则编译为 FormItemRule', () => {
        const item = compileJsonField({
            prop: 'code',
            rules: { required: true, minLength: 2, maxLength: 10, pattern: '^[a-z]+$', message: 'common.patternRuleMsg', min: 1, max: 5 },
        })!;
        expect(item.required).toBe(true);
        const rules = item.rules as import('element-plus').FormItemRule[];
        expect(rules).toHaveLength(3);
        // 长度规则
        expect(rules[0]).toMatchObject({ min: 2, max: 10 });
        // pattern 规则（字符串编译为 RegExp）
        expect(rules[1]).toMatchObject({ pattern: /^[a-z]+$/ });
        // 数值范围规则
        expect(rules[2]).toMatchObject({ type: 'number', min: 1, max: 5 });
    });

    it('rule message 为延迟求值函数（语言切换后校验时取最新文案）', () => {
        const item = compileJsonField({
            prop: 'code',
            rules: { minLength: 2, maxLength: 10, message: 'common.patternRuleMsg' },
        })!;
        const rules = item.rules as import('element-plus').FormItemRule[];
        expect(typeof rules[0].message).toBe('function');
        // mock i18n 下 t 返回 key 本身：显式 message 优先于默认 key
        expect((rules[0].message as () => string)()).toBe('common.patternRuleMsg');
        // 未配置 message 时回退默认 key
        const fallback = compileJsonField({ prop: 'x', rules: { pattern: '^a' } })!;
        const fallbackRules = fallback.rules as import('element-plus').FormItemRule[];
        expect((fallbackRules[0].message as () => string)()).toBe('common.patternRuleMsg');
    });

    it('控件白名单与注册表 jsonCompilable 一致（防两处脱钩）', () => {
        const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
        for (const [type, descriptor] of Object.entries(CONTROL_REGISTRY)) {
            const compiled = compileJsonField({ prop: 'p', type: type as never });
            if (descriptor.jsonCompilable) {
                expect(compiled).toBeDefined();
            } else {
                expect(compiled).toBeUndefined();
            }
        }
        warn.mockRestore();
    });

    it('when / disabled 条件编译为求值函数', () => {
        const item = compileJsonField({
            prop: 'host',
            when: { field: 'enabled', op: 'eq', value: true },
            disabled: { field: 'mode', op: 'eq', value: 'fixed' },
        })!;
        expect(typeof item.when).toBe('function');
        expect(typeof item.disabled).toBe('function');
        expect(item.when!({ enabled: true })).toBe(true);
        expect(item.when!({ enabled: false })).toBe(false);
        expect((item.disabled as (form: Record<string, unknown>) => boolean)({ mode: 'fixed' })).toBe(true);
    });

    it('disabled 布尔值直接透传', () => {
        expect(compileJsonField({ prop: 'a', disabled: true })!.disabled).toBe(true);
        expect(compileJsonField({ prop: 'a' })!.disabled).toBeUndefined();
    });

    it('静态 options 直接透传', () => {
        const options = [
            { value: 'a', label: 'A' },
            { value: 'b', label: 'B' },
        ];
        const item = compileJsonField({ prop: 'type', type: 'select', options })!;
        expect(item.options).toEqual(options);
    });
});

describe('compileJsonField - 类型白名单', () => {
    it('custom / enum / 未知类型跳过并告警', () => {
        const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
        expect(compileJsonField({ prop: 'a', type: 'custom' })).toBeUndefined();
        expect(compileJsonField({ prop: 'b', type: 'enum' })).toBeUndefined();
        expect(compileJsonField({ prop: 'c', type: 'unknown-type' as never })).toBeUndefined();
        expect(warn).toHaveBeenCalledTimes(3);
        warn.mockRestore();
    });

    it('divider / monaco / time / tags 等可渲染类型保留', () => {
        for (const type of ['divider', 'monaco', 'time', 'tags'] as const) {
            expect(compileJsonField({ prop: 'x', type } as JsonField)).toBeDefined();
        }
    });
});

describe('compileJsonField - readonly / hidden / description', () => {
    it('readonly 布尔与条件编译', () => {
        expect(compileJsonField({ prop: 'a', readonly: true })!.readonly).toBe(true);
        const item = compileJsonField({ prop: 'a', readonly: { field: 'locked', op: 'eq', value: true } })!;
        expect(typeof item.readonly).toBe('function');
        expect((item.readonly as (form: Record<string, unknown>) => boolean)({ locked: true })).toBe(true);
        expect((item.readonly as (form: Record<string, unknown>) => boolean)({ locked: false })).toBe(false);
    });

    it('hidden / description 透传', () => {
        const item = compileJsonField({ prop: 'tenantId', hidden: true, description: 'common.tip' })!;
        expect(item.hidden).toBe(true);
        expect(item.description).toBe('common.tip');
    });
});

describe('compileJsonField - optionsSource', () => {
    it('编译为异步加载函数并按字段映射解析', async () => {
        vi.mocked(request.request).mockResolvedValue([
            { code: 'x', name: 'X' },
            { code: 'y', name: 'Y' },
        ]);
        const item = compileJsonField({
            prop: 'server',
            type: 'select',
            optionsSource: { url: '/api/servers', valueField: 'code', labelField: 'name' },
        })!;
        expect(await loadOptions(item)).toEqual([
            { value: 'x', label: 'X' },
            { value: 'y', label: 'Y' },
        ]);
        expect(request.request).toHaveBeenCalledWith('get', '/api/servers', {});
    });

    it('dataField 提取与 deps 合并进请求参数', async () => {
        vi.mocked(request.request).mockResolvedValue({ list: [{ value: 1, label: 'one' }] });
        const item = compileJsonField({
            prop: 'table',
            type: 'select',
            optionsSource: { url: '/api/tables', method: 'post', dataField: 'list', deps: ['dbId'] },
        })!;
        expect(await loadOptions(item, { dbId: 9 })).toEqual([{ value: 1, label: 'one' }]);
        expect(request.request).toHaveBeenLastCalledWith('post', '/api/tables', { dbId: 9 });
    });

    it('非法 url 全部拒绝（防开放代理，含大小写变体与非 / 开头路径）', async () => {
        const urls = ['http://evil.com/api', 'HTTP://evil.com/api', 'HTTPS://evil.com/api', '//evil.com/api', 'api/x', 'ftp://evil.com', ''];
        for (const url of urls) {
            const item = compileJsonField({ prop: 'x', optionsSource: { url } })!;
            await expect(loadOptions(item)).rejects.toThrow();
        }
        // 合法的站内绝对路径（含首尾空白）放行
        vi.mocked(request.request).mockResolvedValue([]);
        const ok = compileJsonField({ prop: 'x', optionsSource: { url: ' /api/ok ' } })!;
        await expect(loadOptions(ok)).resolves.toEqual([]);
        expect(request.request).toHaveBeenLastCalledWith('get', '/api/ok', {});
    });

    it('响应非数组时返回空选项', async () => {
        vi.mocked(request.request).mockResolvedValue({ list: 'not-array' });
        const item = compileJsonField({ prop: 'x', optionsSource: { url: '/api/x', dataField: 'list' } })!;
        await expect(loadOptions(item)).resolves.toEqual([]);
    });
});

describe('compileJsonForm', () => {
    it('完整 Schema 与字段数组两种入参', () => {
        const fields: JsonField[] = [{ prop: 'a' }, { prop: 'b', label: 'B' }];
        expect(compileJsonForm({ version: 1, fields })).toHaveLength(2);
        expect(compileJsonForm(fields)).toHaveLength(2);
        expect(compileJsonForm({ version: 1, fields: [] })).toEqual([]);
    });

    it('逐字段编译互不影响', () => {
        const items = compileJsonForm([{ prop: 'a', rules: { required: true } }, { prop: 'b' }]);
        expect(items[0].required).toBe(true);
        expect(items[1].required).toBeUndefined();
    });

    it('脏数据防御：不支持类型字段整体过滤，其余字段正常编译', () => {
        const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
        const items = compileJsonForm([{ prop: 'ok' }, { prop: 'bad', type: 'custom' }, { prop: 'ok2', type: 'select' }]);
        expect(items.map((i) => i.prop)).toEqual(['ok', 'ok2']);
        warn.mockRestore();
    });
});

describe('buildDefaultForm', () => {
    it('multiple 字段缺省值为数组，避免 undefined 传入多选控件', () => {
        const form = buildDefaultForm([
            { prop: 'tags', type: 'select', multiple: true },
            { prop: 'tags2', type: 'select', multiple: true, defaultValue: ['a'] },
            { prop: 'name' },
        ]);
        expect(form).toEqual({ tags: [], tags2: ['a'] });
    });

    it('无 prop 的 divider 字段不进入表单数据', () => {
        expect(buildDefaultForm([{ type: 'divider', label: 't' }, { prop: 'a', defaultValue: 1 }])).toEqual({ a: 1 });
    });
});

describe('布局能力编译（group / prefix / suffix / tabs）', () => {
    it('group 类型编译为分组容器项，透传 groupDescription', () => {
        const items = compileJsonForm([
            { prop: 'name' },
            { prop: 'g', type: 'group', label: 'adv.title', groupDescription: 'adv.desc' } as unknown as JsonField,
            { prop: 'host' },
        ]);
        expect(items).toHaveLength(3);
        expect(items[1]).toMatchObject({ type: 'group', label: 'adv.title', groupDescription: 'adv.desc' });
        expect(items[1].prop).toBe('g');
    });

    it('prefix / suffix 透传到编译产物', () => {
        const item = compileJsonField({ prop: 'port', type: 'number', prefix: 'common.portPrefix', suffix: 'ms' })!;
        expect(item.prefix).toBe('common.portPrefix');
        expect(item.suffix).toBe('ms');
    });

    it('compileJsonTabs 编译 schema 内置 tabs，tab 间字段独立编译', async () => {
        const { compileJsonTabs } = await import('../compile');
        const tabs = compileJsonTabs({
            version: 1,
            fields: [],
            tabs: [
                { name: 'basic', label: 'tab.basic', fields: [{ prop: 'name', rules: { required: true } }] },
                { name: 'adv', label: 'tab.adv', icon: 'Setting', fields: [{ prop: 'timeout', type: 'number', min: 1 }] },
            ],
        });
        expect(tabs).toHaveLength(2);
        expect(tabs[0]).toMatchObject({ name: 'basic', label: 'tab.basic', icon: undefined });
        expect(tabs[0].items[0]).toMatchObject({ prop: 'name', required: true });
        expect(tabs[1].items[0]).toMatchObject({ prop: 'timeout', type: 'number' });
    });

    it('无 tabs 的 schema 编译为空数组', async () => {
        const { compileJsonTabs } = await import('../compile');
        expect(compileJsonTabs({ version: 1, fields: [{ prop: 'a' }] })).toEqual([]);
    });
});
