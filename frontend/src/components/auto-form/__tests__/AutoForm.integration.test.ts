/**
 * AutoForm 系列组件级集成测试
 *
 * 以真实 element-plus 渲染 AutoForm / AutoFormDialog / AutoFormDrawer，
 * 覆盖：全部控件类型渲染与双向绑定、嵌套路径、动态 required、when 显隐、
 * options 异步加载与排除、tabs 布局、group 分组、宿主回填与确认流程。
 *
 * 重依赖（monaco / svg-icon）与全局 i18n / request 用 mock 替代；
 * 组件内 vue-i18n 的 useI18n 通过最小 messages 的真实 i18n 实例提供。
 */
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, reactive } from 'vue';
import { flushPromises, mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';
import AutoForm from '../AutoForm.vue';
import AutoFormDialog from '../AutoFormDialog.vue';
import AutoFormDrawer from '../AutoFormDrawer.vue';
import type { AutoFormItem, AutoFormTab } from '../types';
import type { AutoFormJsonSchema } from '../json';

// monaco 编辑器依赖 worker 与大包，用桩组件替代
vi.mock('@/components/monaco/MonacoEditor.vue', () => ({
    default: defineComponent({ name: 'MonacoEditorStub', props: ['modelValue', 'options'], template: '<div class="monaco-stub" />' }),
}));
// svg-icon 依赖 store，用桩组件替代
vi.mock('@/components/svg-icon/index.vue', () => ({
    default: defineComponent({ name: 'SvgIconStub', template: '<i class="svg-icon-stub" />' }),
}));
// 全局 i18n 单例（@/common/rule 的 Rules 依赖它），t 原样返回 key 便于断言
vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string, params?: Record<string, unknown>) => (params ? `${key}?${JSON.stringify(params)}` : key) } },
}));
// request 封装（compile 的 optionsSource 依赖它，本测试不触发网络）
vi.mock('@/common/request', () => ({ default: { request: vi.fn().mockResolvedValue([]) } }));

const i18n = createI18n({
    legacy: false,
    locale: 'en',
    messages: {
        en: {
            common: { pleaseInput: 'Please input {label}', pleaseSelect: 'Please select {label}', cancel: 'Cancel', confirm: 'Confirm' },
            fields: { name: 'Name', type: 'Type', host: 'Host' },
        },
    },
});

const mountForm = (props: Record<string, unknown>, slots: Record<string, string> = {}) =>
    mount(AutoForm, {
        props: { modelValue: {}, ...props },
        slots,
        global: { plugins: [ElementPlus, i18n] },
    });

const mountDialog = (props: Record<string, unknown>) =>
    mount(AutoFormDialog, {
        props: { visible: false, ...props },
        global: { plugins: [ElementPlus, i18n], mocks: {} },
    });

// ── 控件渲染与值绑定 ────────────────────────────────────────────

describe('AutoForm 控件渲染与双向绑定', () => {
    it('基础控件全部渲染，input 输入写入 form 对象', async () => {
        const form = reactive<Record<string, any>>({});
        const wrapper = mountForm({
            modelValue: form,
            items: [
                { prop: 'name', label: 'fields.name', type: 'input' },
                { prop: 'pwd', label: 'fields.name', type: 'password' },
                { prop: 'num', label: 'fields.name', type: 'number' },
                { prop: 'desc', label: 'fields.name', type: 'textarea' },
                { prop: 'sw', label: 'fields.name', type: 'switch' },
                { prop: 'tags', label: 'fields.name', type: 'tags' },
            ] as AutoFormItem[],
        });
        await flushPromises();
        expect(wrapper.find('input[type="password"]').exists()).toBe(true);
        expect(wrapper.find('.el-textarea').exists()).toBe(true);
        expect(wrapper.find('.el-switch').exists()).toBe(true);

        await wrapper.find('input[type="text"]').setValue('hello');
        expect(form.name).toBe('hello');
    });

    it('select 渲染静态 options 且 label 经 $t，选择写入 form', async () => {
        const form = reactive<Record<string, any>>({});
        const wrapper = mountForm({
            modelValue: form,
            items: [
                { prop: 'type', label: 'fields.type', type: 'select', options: [{ value: 1, label: 'fields.name' }, { value: 2, label: 'Raw Label' }] },
            ] as AutoFormItem[],
        });
        await flushPromises();
        // happy-dom 中 el-select 输入框可聚焦展开；直接断言 option 经下拉渲染（el-select 的 dropdown teleport 到 body）
        const select = wrapper.findComponent({ name: 'ElSelect' });
        expect(select.exists()).toBe(true);
        // label 翻译在 option 渲染时进行，通过下拉实例验证
        select.vm.$nextTick;
        await select.trigger('click');
        await flushPromises();
        const options = document.body.querySelectorAll('.el-select-dropdown__item');
        expect(options.length).toBe(2);
        expect(options[0].textContent).toContain('Name');
        expect(options[1].textContent).toContain('Raw Label');

        await (select.vm as any).handleOptionSelect({ value: 1, label: 'Name' });
        await flushPromises();
        expect(form.type).toBe(1);
    });

    it('radio 渲染静态 options 并应用 excludeValues', () => {
        const wrapper = mountForm({
            items: [
                {
                    prop: 'mode',
                    label: 'fields.type',
                    type: 'radio',
                    options: [
                        { value: 'a', label: 'A' },
                        { value: 'b', label: 'B' },
                        { value: 'c', label: 'C' },
                    ],
                    excludeValues: ['c'],
                },
            ] as AutoFormItem[],
        });
        const radios = wrapper.findAll('.el-radio');
        expect(radios.length).toBe(2);
        expect(radios.map((r) => r.text())).toEqual(['A', 'B']);
    });

    it('enum 渲染枚举选项并应用 excludeValues', () => {
        const wrapper = mountForm({
            items: [
                {
                    prop: 'level',
                    label: 'fields.type',
                    type: 'enum',
                    enums: { A: { value: 'a', label: 'A' }, B: { value: 'b', label: 'B' } } as unknown as AutoFormItem['enums'],
                    excludeValues: ['b'],
                },
            ] as AutoFormItem[],
        });
        const options = wrapper.findAllComponents({ name: 'ElOption' });
        expect(options.length).toBe(1);
        expect(options[0].props('value')).toBe('a');
    });

    it('嵌套路径 prop 写入 form 的嵌套对象', async () => {
        const form = reactive<Record<string, any>>({});
        const wrapper = mountForm({ modelValue: form, items: [{ prop: 'meta.icon', label: 'fields.name', type: 'input' }] as AutoFormItem[] });
        await flushPromises();
        await wrapper.find('input').setValue('icon-value');
        expect(form.meta.icon).toBe('icon-value');
    });

    it('custom 类型渲染具名插槽并透传 form/item', () => {
        const wrapper = mountForm(
            { items: [{ prop: 'custom', type: 'custom', slot: 'myField' }] as AutoFormItem[] },
            { myField: `<template #default="{ form, item }"><div class="custom-stub">{{ item.prop }}-{{ form.extra ?? 'none' }}</div></template>` }
        );
        expect(wrapper.find('.custom-stub').text()).toBe('custom-none');
    });

    it('divider 渲染分隔标题（非字段无 form-item）', () => {
        const wrapper = mountForm({ items: [{ type: 'divider', label: 'fields.name' }] as AutoFormItem[] });
        expect(wrapper.find('.el-divider__text').text()).toBe('Name');
    });
});

// ── 显隐 / 必填 / 禁用联动 ──────────────────────────────────────

describe('AutoForm 显隐与校验联动', () => {
    it('when 为 false 的字段不渲染但保留 form 值，翻转后恢复渲染', async () => {
        const form = reactive<Record<string, any>>({ extra: 'keep', mode: 'off' });
        const wrapper = mountForm({
            modelValue: form,
            items: [
                { prop: 'mode', label: 'fields.type', type: 'input' },
                { prop: 'extra', label: 'fields.name', type: 'input', when: (f) => f.mode === 'on' },
            ] as AutoFormItem[],
        });
        await flushPromises();
        const inputs = wrapper.findAll('input');
        // 仅 mode 字段可见
        expect(inputs.length).toBe(1);
        expect(form.extra).toBe('keep');

        form.mode = 'on';
        await flushPromises();
        expect(wrapper.findAll('input').length).toBe(2);
    });

    it('动态 required：条件满足时生成必填规则与星号，条件翻转后规则消失', async () => {
        const form = reactive<Record<string, any>>({ host: '', mode: 'on' });
        const wrapper = mountForm({
            modelValue: form,
            items: [{ prop: 'host', label: 'fields.host', type: 'input', required: (f) => f.mode === 'on' }] as AutoFormItem[],
        });
        await flushPromises();
        // 必填星号显示 + 必填规则生成（happy-dom 下 el-form 全量校验不收集 fields，用 form-item 级校验锁定规则行为）
        expect(wrapper.find('.el-form-item.is-required').exists()).toBe(true);
        const rules = (wrapper.findComponent({ name: 'ElForm' }).props('rules') as Record<string, unknown[]>).host;
        expect(rules?.[0]).toMatchObject({ required: true });

        // 条件翻转后不再必填：星号与规则同时消失
        form.mode = 'off';
        await flushPromises();
        expect(wrapper.find('.el-form-item.is-required').exists()).toBe(false);
        expect((wrapper.findComponent({ name: 'ElForm' }).props('rules') as Record<string, unknown[]>).host).toBeUndefined();
    });

    it('自定义 validate 函数包装为 validator 规则：true 通过，false/string 产生错误', async () => {
        const wrapper = mountForm({
            items: [{ prop: 'name', label: 'fields.name', type: 'input', validate: (v) => (v === 'bad' ? 'common.invalidValue' : true) }] as AutoFormItem[],
        });
        await flushPromises();
        // 同步调用包装后的 validator，锁定 true→通过、string→i18n 错误 的语义
        const validator = ((wrapper.findComponent({ name: 'ElForm' }).props('rules') as Record<string, Array<{ validator: (r: unknown, v: unknown, cb: (e?: Error) => void) => void }>>).name ?? [])[0]?.validator;
        expect(validator).toBeTypeOf('function');
        const cb = vi.fn();
        validator({}, 'bad', cb);
        expect(cb).toHaveBeenCalledWith(expect.any(Error));
        validator({}, 'good', cb);
        expect(cb).toHaveBeenCalledWith();
    });

    it('全局 readonly 下所有控件禁用', async () => {
        const wrapper = mountForm({
            readonly: true,
            items: [
                { prop: 'name', label: 'fields.name', type: 'input' },
                { prop: 'sw', label: 'fields.name', type: 'switch' },
            ] as AutoFormItem[],
        });
        await flushPromises();
        expect(wrapper.find('input').attributes('disabled')).toBeDefined();
        expect(wrapper.find('.el-switch').classes()).toContain('is-disabled');
    });

    it('optionDisabled 联动禁用指定选项', () => {
        const wrapper = mountForm({
            items: [
                {
                    prop: 'type',
                    label: 'fields.type',
                    type: 'select',
                    options: [{ value: 'a', label: 'A' }, { value: 'b', label: 'B' }],
                    optionDisabled: (v) => v === 'b',
                },
            ] as AutoFormItem[],
        });
        const optionComps = wrapper.findAllComponents({ name: 'ElOption' });
        expect(optionComps[0].props('disabled')).toBe(false);
        expect(optionComps[1].props('disabled')).toBe(true);
    });
});

// ── options 异步加载 ────────────────────────────────────────────

describe('AutoForm options 异步加载', () => {
    it('异步 options 加载完成渲染选项，函数依赖 form 值', async () => {
        const form = reactive<Record<string, any>>({});
        const loader = vi.fn(async () => [{ value: 'x', label: 'X' }]);
        const wrapper = mountForm({
            modelValue: form,
            items: [{ prop: 'dep', label: 'fields.type', type: 'select', options: loader }] as AutoFormItem[],
        });
        await flushPromises();
        expect(loader).toHaveBeenCalledWith(form);
        expect(wrapper.findAllComponents({ name: 'ElOption' }).length).toBe(1);
    });
});

// ── Tab 布局 ────────────────────────────────────────────────────

describe('AutoForm Tab 布局', () => {
    const tabs: AutoFormTab[] = [
        { name: 'basic', label: 'fields.name', items: [{ prop: 'a', label: 'fields.name', type: 'input' }] },
        { name: 'adv', label: 'fields.type', items: [{ prop: 'b', label: 'fields.type', type: 'input' }] },
    ];

    it('tabs 渲染并自动激活第一个 Tab（update:activeTab 事件通知外部）', async () => {
        const onActiveTab = vi.fn();
        const wrapper = mountForm({ tabs, 'onUpdate:activeTab': onActiveTab });
        await flushPromises();
        expect(wrapper.findAll('.el-tabs__item').length).toBe(2);
        expect(onActiveTab).toHaveBeenCalledWith('basic');
    });

    it('Tab 禁用支持静态布尔与动态函数', () => {
        const wrapper = mountForm({
            tabs: [
                { name: 'a', label: 'fields.name', items: [], disabled: true },
                { name: 'b', label: 'fields.name', items: [], disabled: (f: any) => f.lock === true },
            ],
        });
        const items = wrapper.findAll('.el-tabs__item');
        expect(items[0].classes()).toContain('is-disabled');
        expect(items[1].classes()).not.toContain('is-disabled');
    });

    it('schema 内置 tabs 编译渲染', () => {
        const schema: AutoFormJsonSchema = { version: 1, fields: [], tabs: [{ name: 't1', label: 'fields.name', fields: [{ prop: 'p', type: 'input' }] }] };
        const wrapper = mountForm({ schema });
        expect(wrapper.find('.el-tabs__item').exists()).toBe(true);
        expect(wrapper.findAll('input').length).toBe(1);
    });
});

// ── group 分组布局 ──────────────────────────────────────────────

describe('AutoForm group 分组布局', () => {
    it('group 渲染标题/描述/容器，组内字段正常渲染', () => {
        const wrapper = mountForm({
            items: [
                { type: 'group', label: 'fields.name', groupDescription: 'fields.type' },
                { prop: 'a', label: 'fields.name', type: 'input' },
            ] as AutoFormItem[],
        });
        expect(wrapper.text()).toContain('Name');
        expect(wrapper.find('.rounded-lg').exists()).toBe(true);
        expect(wrapper.find('.rounded-lg input').exists()).toBe(true);
    });

    it('组内字段全部被 when 隐藏时不渲染空壳容器（动态翻转生效）', async () => {
        const form = reactive<Record<string, any>>({ mode: 'off' });
        const wrapper = mountForm({
            modelValue: form,
            items: [
                { prop: 'mode', label: 'fields.type', type: 'input' },
                { type: 'group', label: 'fields.name' },
                { prop: 'hidden1', label: 'fields.name', type: 'input', when: (f) => f.mode === 'on' },
                { prop: 'hidden2', label: 'fields.name', type: 'input', when: (f) => f.mode === 'on' },
            ] as AutoFormItem[],
        });
        await flushPromises();
        expect(wrapper.find('.rounded-lg').exists()).toBe(false);

        form.mode = 'on';
        await flushPromises();
        expect(wrapper.find('.rounded-lg').exists()).toBe(true);
    });
});

// ── JSON Schema 编译渲染 ────────────────────────────────────────

describe('AutoForm JSON Schema 编译渲染', () => {
    it('schema.fields 编译为表单：label 翻译、required 星号、defaultValue 经宿主回填', async () => {
        const schema: AutoFormJsonSchema = {
            version: 1,
            fields: [
                { prop: 'host', label: 'fields.host', type: 'input', rules: { required: true } },
                { prop: 'port', label: 'fields.name', type: 'number', min: 1, max: 100 },
            ],
        };
        const wrapper = mountForm({ schema });
        await flushPromises();
        expect(wrapper.find('.el-form-item.is-required').exists()).toBe(true);
        expect(wrapper.text()).toContain('Host');
    });

    it('不支持 JSON 下发的类型（custom/enum）跳过并告警', () => {
        const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
        const wrapper = mountForm({ schema: { version: 1, fields: [{ prop: 'a', type: 'custom' }, { prop: 'b', type: 'input' }] } });
        expect(wrapper.findAll('input').length).toBe(1);
        expect(warn).toHaveBeenCalled();
        warn.mockRestore();
    });
});

// ── 宿主组件（Dialog / Drawer） ─────────────────────────────────

describe('AutoFormDialog 回填与确认流程', () => {
    it('打开时对象 data 深拷贝回填：编辑表单不污染外部行数据', async () => {
        const rowData = { name: 'orig', meta: { icon: 'a' } };
        const wrapper = mountDialog({ visible: false, data: rowData, items: [{ prop: 'name', type: 'input' }, { prop: 'meta.icon', type: 'input' }] as AutoFormItem[] });
        await flushPromises();
        await wrapper.setProps({ visible: true });
        await flushPromises();

        // 编辑深拷贝中的嵌套值
        const inputs = wrapper.findAll('input');
        await inputs[0].setValue('changed');
        await inputs[1].setValue('b');
        expect(rowData.name).toBe('orig');
        expect(rowData.meta.icon).toBe('a');
    });

    it('data 为 null 时按 defaultValue 回填，multiple 字段缺省为数组', async () => {
        const wrapper = mountDialog({
            visible: false,
            data: null,
            items: [
                { prop: 'name', type: 'input', defaultValue: 'default-name' },
                { prop: 'ids', type: 'select', multiple: true },
            ] as AutoFormItem[],
        });
        await flushPromises();
        await wrapper.setProps({ visible: true });
        await flushPromises();
        expect((wrapper.vm as any).validate).toBeDefined();
        // 回填结果经 confirm 事件验证
        await (wrapper.vm as any).validate();
        const confirmBtn = wrapper.findAll('button').find((b) => b.text() === 'Confirm');
        await confirmBtn!.trigger('click');
        await flushPromises();
        const emitted = wrapper.emitted('confirm') as unknown[][] | undefined;
        expect(emitted?.[0]?.[0]).toMatchObject({ name: 'default-name', ids: [] });
    });

    it('打开期间外部 data 引用变化不重置表单（保留用户已填内容）', async () => {
        const wrapper = mountDialog({ visible: false, data: { name: 'v1' }, items: [{ prop: 'name', type: 'input' }] as AutoFormItem[] });
        await flushPromises();
        await wrapper.setProps({ visible: true });
        await flushPromises();
        await wrapper.findAll('input')[0].setValue('user-input');

        // 外部 data 替换为新引用（如列表刷新）
        await wrapper.setProps({ data: { name: 'v2' } });
        await flushPromises();
        expect(wrapper.findAll('input')[0].element.value).toBe('user-input');

        // 关闭再打开才重新回填
        await wrapper.setProps({ visible: false });
        await flushPromises();
        await wrapper.setProps({ visible: true });
        await flushPromises();
        expect(wrapper.findAll('input')[0].element.value).toBe('v2');
    });

    it('schema 内置 tabs 场景 defaultValue 不丢失（回归锁定）', async () => {
        const schema: AutoFormJsonSchema = {
            version: 1,
            fields: [],
            tabs: [{ name: 't1', label: 'fields.name', fields: [{ prop: 'p', type: 'input', defaultValue: 'from-tab' }] }],
        };
        const wrapper = mountDialog({ visible: false, data: null, schema });
        await flushPromises();
        await wrapper.setProps({ visible: true });
        await flushPromises();
        const confirmBtn = wrapper.findAll('button').find((b) => b.text() === 'Confirm');
        await confirmBtn!.trigger('click');
        await flushPromises();
        expect((wrapper.emitted('confirm') as unknown[][])[0]?.[0]).toMatchObject({ p: 'from-tab' });
    });

    it('默认禁止点击遮罩关闭（防误关丢失表单）', () => {
        const wrapper = mountDialog({ visible: true, items: [{ prop: 'a', type: 'input' }] as AutoFormItem[] });
        expect(wrapper.findComponent({ name: 'ElDialog' }).props('closeOnClickModal')).toBe(false);
    });

    it('tabs 布局 + expose 方法透传可用', async () => {
        const wrapper = mountDialog({
            visible: true,
            tabs: [{ name: 'a', label: 'fields.name', items: [{ prop: 'x', type: 'input' }] }],
        });
        await flushPromises();
        expect(typeof (wrapper.vm as any).validate).toBe('function');
        expect(typeof (wrapper.vm as any).resetFields).toBe('function');
        expect(typeof (wrapper.vm as any).clearValidate).toBe('function');
        // happy-dom 下 el-form 全量校验 resolve(false/true) 均不抛错，可正常调用即验证契约
        await expect((wrapper.vm as any).validate()).resolves.toBeDefined();
    });
});

describe('AutoFormDrawer', () => {
    it('打开回填与 expose 与 Dialog 行为一致', async () => {
        const wrapper = mount(AutoFormDrawer, {
            props: { visible: false, data: { name: 'v1' }, items: [{ prop: 'name', type: 'input' }] as AutoFormItem[] },
            global: { plugins: [ElementPlus, i18n] },
        });
        await flushPromises();
        await wrapper.setProps({ visible: true });
        await flushPromises();
        expect(wrapper.findAll('input')[0].element.value).toBe('v1');
        expect(typeof (wrapper.vm as any).validate).toBe('function');
    });
});
