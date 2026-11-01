import { describe, expect, it, vi } from 'vitest';

import { firstMissingRequired, resolveFormAction } from '../formMode';

/**
 * 回归：新增行必须生成 INSERT。
 *
 * 历史缺陷：insert/update 由「表单当前是否有值」判定，用户点「+」后一填字段就被当成
 * 编辑；上一次编辑残留的 oldValue 也会让新增变成 `UPDATE ... WHERE id = NULL`。
 * 现在语义由弹窗用途（mode）决定。
 */
describe('resolveFormAction', () => {
    it('create 模式下即使已有表单值，也必须走 insert', () => {
        expect(resolveFormAction('create', { id: 1, name: 'filled' })).toBe('insert');
        expect(resolveFormAction('create', null)).toBe('insert');
    });

    it('edit 模式且有旧值基线时走 update', () => {
        expect(resolveFormAction('edit', { id: 1, name: 'old' })).toBe('update');
    });

    it('edit 模式但基线缺失时退回 insert（不盲更新）', () => {
        expect(resolveFormAction('edit', null)).toBe('insert');
        expect(resolveFormAction(undefined, null)).toBe('insert');
    });
});

// ---------------------------------------------------------------------------
// 关闭清理：取消 / 右上角 × / Esc 都必须把表单清空（用户反馈「点取消后数据没清」）
// ---------------------------------------------------------------------------

import { mount } from '@vue/test-utils';
import { nextTick } from 'vue';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';

import DbTableDataForm from '../DbTableDataForm.vue';

vi.mock('@/hooks/useI18n', () => ({
    useI18nFormValidate: vi.fn(async () => true),
    Msg: { warning: vi.fn(), error: vi.fn(), success: vi.fn(), info: vi.fn() },
}));

vi.mock('../widgets/ColumnValueInput.vue', () => ({
    default: { name: 'ColumnValueInput', props: ['modelValue'], template: '<input />' },
}));

const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });

function mountDialog(modelValue: Record<string, unknown>) {
    return mount(DbTableDataForm, {
        props: {
            dbInst: {
                getDialect: () => ({ getDataType: (t: string) => t }),
                genInsertSql: vi.fn(async () => 'INSERT'),
                genUpdateSql: vi.fn(async () => 'UPDATE'),
                promptExeSql: vi.fn(),
            } as never,
            dbName: 'd',
            tableName: 't',
            columns: [{ columnName: 'name', type: 'varchar(10)' }] as never,
            title: 'add',
            visible: true,
            mode: 'create',
            modelValue,
        },
        global: { plugins: [i18n, ElementPlus] },
    });
}

describe('DbTableDataForm 关闭清理', () => {
    it('取消按钮：清空表单并回写父级 modelValue', async () => {
        const wrapper = mountDialog({ name: 'dirty' });
        await nextTick();
        const cancel = wrapper.findAll('button').find((b) => b.text().includes('cancel') || b.text().includes('取消'));
        if (!cancel) throw new Error('cancel button not found');
        await cancel.trigger('click');
        await nextTick();
        const emitted = wrapper.emitted('update:modelValue') ?? [];
        expect(emitted.some((args) => JSON.stringify(args[0]) === '{}')).toBe(true);
    });

    it('非取消途径关闭（× / Esc / 遮罩）也清空：dialog close 事件即重置', async () => {
        const wrapper = mountDialog({ name: 'dirty' });
        await nextTick();
        const dialog = wrapper.findComponent({ name: 'ElDialog' });
        dialog.vm.$emit('close');
        await nextTick();
        const emitted = wrapper.emitted('update:modelValue') ?? [];
        expect(emitted.some((args) => JSON.stringify(args[0]) === '{}')).toBe(true);
    });
});

// ---------------------------------------------------------------------------
// 打开即校验：必填项在用户尚未交互前不得显示错误态
// ---------------------------------------------------------------------------

describe('DbTableDataForm 校验时机', () => {
    it('接线：关闭 validate-on-rule-change（规则建立时不自动跑校验）', async () => {
        // happy-dom 下 is-error 类不可观测（el-form 全量 validate 不收集 fields），
        // 因此回归锁必须钉接线本身；删掉该属性这条必红。
        const wrapper = mountDialog({});
        await nextTick();
        const form = wrapper.findComponent({ name: 'ElForm' });
        expect(form.exists()).toBe(true);
        expect(form.props('validateOnRuleChange')).toBe(false);
    });

    it('必填校验只在提交时触发：firstMissingRequired 是唯一入口（纯函数可断言）', () => {
        // 这条替代了原先「打开后无 is-error」的断言 —— 后者在当前实现下恒绿（已无任何
        // el-form 规则，is-error 永不出现），测不到任何东西。
        const columns = [
            { columnName: 'id', isPrimaryKey: true },
            { columnName: 'name', nullable: false },
            { columnName: 'note', nullable: true },
            { columnName: 'seq', nullable: false, autoIncrement: true },
        ];
        expect(firstMissingRequired(columns, {})).toBe('name');
        expect(firstMissingRequired(columns, { name: '   ' })).toBe('name');
        expect(firstMissingRequired(columns, { name: null })).toBe('name');
        expect(firstMissingRequired(columns, { name: 'ok' })).toBeNull();
        // 主键/自增/可空列即使为空也不拦
        expect(firstMissingRequired([{ columnName: 'id', isPrimaryKey: true }], {})).toBeNull();
        expect(firstMissingRequired([{ columnName: 'seq', autoIncrement: true }], {})).toBeNull();
        expect(firstMissingRequired([{ columnName: 'note', nullable: true }], {})).toBeNull();
    });

    it('填写后清空（关闭重置）也不应残留错误态', async () => {
        const wrapper = mountDialog({ name: 'x' });
        await nextTick();
        const dialog = wrapper.findComponent({ name: 'ElDialog' });
        dialog.vm.$emit('close');
        await nextTick();
        await nextTick();
        expect(wrapper.findAll('.el-form-item.is-error').length).toBe(0);
    });
});

// ---------------------------------------------------------------------------
// 打开即校验根因：el-form-item 的 required 会生成真实规则，字段动态创建时立刻校验
// ---------------------------------------------------------------------------

describe('DbTableDataForm 必填只做视觉标记', () => {
    it('不向 el-form-item 传 required（改由提交时手动校验），但保留星号 class', async () => {
        const wrapper = mountDialog({});
        await nextTick();
        const items = wrapper.findAllComponents({ name: 'ElFormItem' });
        expect(items.length).toBeGreaterThan(0);
        // 一旦传 required，EP 会生成规则并在字段注册时立刻校验 → 打开即红
        expect(items.every((i) => !i.props('required'))).toBe(true);
        // 星号仍要显示
        expect(items.some((i) => i.classes().includes('is-required'))).toBe(true);
    });
});
