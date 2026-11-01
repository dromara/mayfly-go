/**
 * 编辑表抽屉：长度回显 + 勾选「可为空」后生成的 ALTER 必须是能执行的合法 SQL。
 *
 * 为什么要有这组用例（真实故障）：列元数据里的长度由 initColumns 事后写入，而表单字段在它之前就构建好了，
 * 于是「长度」列恒为空；保存时 MODIFY COLUMN 丢掉长度/bigint 写成 `unsigned bigint`，
 * 生成 `varchar NULL` / `unsigned bigint` 这类非法 DDL，MySQL 报 1064，用户看到的就是
 * 「长度、改为可为空等都没法生效」。这里从真实元数据（mayfly_go.t_ai_memory）出发，
 * 一路走到「点保存后交给执行框的 SQL」，把回显与 DDL 两处一起钉住。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import type { Component } from 'vue';
import { createI18n } from 'vue-i18n';
import ElementPlus from 'element-plus';
import DbTableOp from '../DbTableOp.vue';
import { buildColumnDefinitions } from '../../core/columnDefinition';
import type { ColumnMetadata } from '../../types';

const sqlExec = vi.hoisted(() => vi.fn());
vi.mock('../../sql-editor/SqlExecBox', () => ({ default: (props: unknown) => sqlExec(props) }));
vi.mock('@/hooks/useI18n', () => ({
    Msg: { warning: vi.fn(), error: vi.fn(), success: vi.fn() },
}));

/** mayfly_go.t_ai_memory 的真实列元数据（information_schema 实测值） */
const columns: ColumnMetadata[] = [
    { columnName: 'id', dataType: 'unsigned bigint', numPrecision: 20, isPrimaryKey: true, autoIncrement: true, nullable: false },
    { columnName: 'is_deleted', dataType: 'tinyint', numPrecision: 3, columnDefault: '0', nullable: false },
    { columnName: 'creator_id', dataType: 'unsigned bigint', numPrecision: 20, nullable: false },
    { columnName: 'creator', dataType: 'varchar', charMaxLength: 32, nullable: false, columnComment: '创建人' },
    { columnName: 'type', dataType: 'varchar', charMaxLength: 32, nullable: false, columnDefault: 'fact', columnComment: '记忆类型' },
    { columnName: 'content', dataType: 'text', nullable: false, columnComment: '记忆内容' },
    { columnName: 'tags', dataType: 'varchar', charMaxLength: 512, nullable: true, columnComment: '标签JSON数组' },
];

const i18n = createI18n({
    legacy: false,
    globalInjection: true,
    locale: 'zh-cn',
    missingWarn: false,
    fallbackWarn: false,
    messages: { 'zh-cn': { common: { save: '保存', cancel: '取消', operation: '操作', type: '类型', delete: '删除' } } },
});

/** 打开抽屉（visible 由 false 翻 true 才会走初始化分支） */
async function openEditor(withPrebuiltFormData: boolean) {
    const fields = buildColumnDefinitions(columns);
    const data = withPrebuiltFormData
        ? {
              edit: true,
              row: { tableName: 't_ai_memory', tableComment: '记忆' },
              columns,
              formData: {
                  tableName: 't_ai_memory',
                  tableComment: '记忆',
                  oldTableName: 't_ai_memory',
                  oldTableComment: '记忆',
                  db: 'mayfly_go',
                  fields: { res: fields, oldFields: fields.map((f) => structuredClone(f)) },
                  indexes: { res: [], oldIndexes: [], columns: [] },
              },
          }
        : { edit: true, row: { tableName: 't_ai_memory', tableComment: '记忆' }, columns };

    // `visible` 是 defineModel 声明的 prop，不参与 SFC 公开 props 类型推导，但运行时就是普通 prop；
    // 按 Component 挂载，避免类型层挡掉「把抽屉打开」这一必要动作
    const wrapper = mount(DbTableOp as Component, {
        props: { visible: false, data, dbId: 1, db: 'mayfly_go', dbType: 'mysql', version: '8.0.46' },
        global: { plugins: [ElementPlus, i18n] },
        attachTo: document.body,
    });
    await wrapper.setProps({ visible: true });
    await flushPromises();
    return wrapper;
}

/** 按首列文本定位一行（抽屉 append-to-body，内容在 document.body 下） */
function rowOf(name: string): HTMLElement {
    const rows = Array.from(document.body.querySelectorAll<HTMLElement>('.el-table__body tbody tr'));
    const row = rows.find((r) => r.querySelector('input')?.value === name);
    if (!row) {
        throw new Error(`未找到列行：${name}`);
    }
    return row;
}

/** 行内第 index 个长度类输入（0=长度，1=精度），返回输入框 */
function numericInput(name: string, index: number): HTMLInputElement {
    const input = rowOf(name).querySelectorAll<HTMLInputElement>('input[type="number"]')[index];
    if (!input) {
        throw new Error(`${name} 行缺少第 ${index} 个数值输入`);
    }
    return input;
}

function nullableCheckbox(name: string): HTMLInputElement {
    const checkbox = rowOf(name).querySelectorAll<HTMLInputElement>('input.el-checkbox__original')[0];
    if (!checkbox) {
        throw new Error(`${name} 行缺少「是否可为空」勾选框`);
    }
    return checkbox;
}

function clickSave() {
    const button = Array.from(document.body.querySelectorAll('button')).find((b) => b.textContent?.trim() === '保存');
    if (!button) {
        throw new Error('未找到保存按钮');
    }
    button.click();
}

beforeEach(() => {
    sqlExec.mockClear();
    // 抽屉 append-to-body，每个用例前清空上一条挂载遗留的 DOM
    document.body.replaceChildren();
});

describe('编辑表抽屉：长度回显与列变更 DDL', () => {
    it('长度列回显真实声明长度，整数/文本类型不留长度', async () => {
        await openEditor(false);

        expect(numericInput('creator', 0).value).toBe('32');
        expect(numericInput('tags', 0).value).toBe('512');
        expect(numericInput('content', 0).value).toBe('');
        expect(numericInput('id', 0).value).toBe('');
    });

    it('勾选「可为空」后保存，生成的 ALTER 带完整长度且是合法 MySQL 语法', async () => {
        await openEditor(true);

        expect(nullableCheckbox('creator').checked).toBe(false);
        nullableCheckbox('creator').click();
        await flushPromises();

        clickSave();
        await flushPromises();

        expect(sqlExec).toHaveBeenCalledTimes(1);
        const sql = (sqlExec.mock.calls[0][0] as { sql: string }).sql;
        expect(sql).toContain('MODIFY COLUMN `creator` varchar(32) NULL');
        // 无长度 = `varchar NULL`，MySQL 直接 1064
        expect(sql).not.toMatch(/`creator` varchar\b(?!\()/);
    });

    it('改长度也只改长度，不会把无符号类型写成非法的 unsigned bigint', async () => {
        await openEditor(true);

        const tagsLength = numericInput('tags', 0);
        tagsLength.value = '256';
        tagsLength.dispatchEvent(new Event('input'));
        await flushPromises();

        clickSave();
        await flushPromises();

        expect(sqlExec).toHaveBeenCalledTimes(1);
        const sql = (sqlExec.mock.calls[0][0] as { sql: string }).sql;
        expect(sql).toContain('MODIFY COLUMN `tags` varchar(256) NULL');
        expect(sql).not.toContain('unsigned bigint');
    });
}, 40000);
