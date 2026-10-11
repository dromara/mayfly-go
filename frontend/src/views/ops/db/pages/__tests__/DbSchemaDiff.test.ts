import { beforeEach, describe, expect, it, vi } from 'vitest';
import { flushPromises, mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';
import { defineComponent } from 'vue';

// 取数层全部打桩：这里要钉的是「服务端结论如何呈现在界面上」，不是请求怎么发
const schemaDiff = vi.fn();
const dbs = vi.fn();
const tableInfos = vi.fn();
const saveSchemaBaseline = vi.fn();
const checkSchemaBaseline = vi.fn();
const deleteSchemaBaseline = vi.fn();

// 基线行按用例可变：未建基线与已有基线两类呈现必须分开验证
let baselineRows: Array<Record<string, unknown>> = [];

// 目标库经资源树选择器选中：桩件把「选到哪条库记录 + 哪个具体库」暴露给用例设置
let stubTarget = { id: 2, db: 'other_db' };
const DbSelectTreeStub = defineComponent({
    name: 'DbSelectTree',
    props: { dbId: { type: Number, default: undefined }, dbName: { type: String, default: '' } },
    emits: ['update:dbId', 'update:dbName', 'selectDb'],
    setup(_props, { emit }) {
        return {
            pick: () => {
                emit('update:dbId', stubTarget.id);
                emit('update:dbName', stubTarget.db);
                emit('selectDb', {});
            },
        };
    },
    template: '<button class="stub-target" @click="pick">pick-target</button>',
});

vi.mock('../../api', () => ({
    dbApi: {
        schemaDiff: { request: (...args: unknown[]) => schemaDiff(...args) },
        dbs: { request: (...args: unknown[]) => dbs(...args) },
        tableInfos: { request: (...args: unknown[]) => tableInfos(...args) },
        saveSchemaBaseline: { request: (...args: unknown[]) => saveSchemaBaseline(...args) },
        listSchemaBaseline: { request: () => Promise.resolve(baselineRows) },
        checkSchemaBaseline: { request: (...args: unknown[]) => checkSchemaBaseline(...args) },
        deleteSchemaBaseline: { request: (...args: unknown[]) => deleteSchemaBaseline(...args) },
    },
}));

import DbSchemaDiff from '../DbSchemaDiff.vue';

// i18n 用空字典：t() 原样返回键名，断言直接针对键，改文案不会误伤判据
const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });

type Wrapper = ReturnType<typeof mountPanel>;

function mountPanel() {
    return mount(DbSchemaDiff, {
        props: { srcDbId: 1, srcDbs: ['mayfly_go'] },
        global: { plugins: [i18n, ElementPlus], stubs: { DbSelectTree: DbSelectTreeStub, TagCodePath: true } },
    });
}

/** 选中目标库：点桩件按钮即回传 dbId + 具体库名，与真人从资源树选到某个库同一条路径 */
async function selectTarget(wrapper: Wrapper) {
    await wrapper.find('.stub-target').trigger('click');
    await flushPromises();
}

async function clickCompare(wrapper: Wrapper) {
    const button = wrapper.findAllComponents({ name: 'ElButton' }).find((item) => item.text().includes('db.diffRun'));
    expect(button, '对比按钮必须存在').toBeTruthy();
    await button!.trigger('click');
    await flushPromises();
}

describe('DbSchemaDiff 结论呈现', () => {
    beforeEach(() => {
        schemaDiff.mockReset();
        dbs.mockReset();
        tableInfos.mockReset();
        saveSchemaBaseline.mockReset();
        checkSchemaBaseline.mockReset();
        deleteSchemaBaseline.mockReset();
        baselineRows = [];
        stubTarget = { id: 2, db: 'other_db' };
        dbs.mockResolvedValue({ list: [{ id: 1, code: 'c1', name: '本地库', creator: 'admin' }], total: 1 });
        tableInfos.mockResolvedValue([{ tableName: 't_user', tableComment: '', dataLength: 0, tableRows: 0, indexLength: 0 }]);
    });

    it('未建基线时说「尚未建立」，不拿「0 张表漂移」冒充结构稳定', async () => {
        const wrapper = mountPanel();
        await flushPromises();

        expect(wrapper.text()).toContain('db.baselineNone');
        expect(wrapper.text()).not.toContain('db.baselineNoDrift');
    });

    it('已有基线时给出跟踪表数与漂移数，比对结果逐表列出', async () => {
        baselineRows = [
            { table: 't_user', fingerprint: 'a', columnCount: 2, primaryKeys: 'id', indexCount: 0, drifted: true, driftDetail: '列数 2→3' },
            { table: 't_order', fingerprint: 'b', columnCount: 5, primaryKeys: 'id', indexCount: 1, drifted: false, driftDetail: '' },
        ];
        checkSchemaBaseline.mockResolvedValue({ dbId: 1, db: 'mayfly_go', driftedTables: [{ table: 't_user', detail: '列数 2→3' }] });

        const wrapper = mountPanel();
        await flushPromises();
        expect(wrapper.text()).toContain('db.baselineRowCount');
        expect(wrapper.text()).toContain('db.baselineDriftCount');

        await wrapper
            .findAllComponents({ name: 'ElButton' })
            .find((btn) => btn.text().includes('db.baselineCheck'))!
            .trigger('click');
        await flushPromises();

        expect(checkSchemaBaseline).toHaveBeenCalledWith({ id: 1, db: 'mayfly_go' });
        expect(wrapper.text()).toContain('t_user');
        expect(wrapper.text()).toContain('列数 2→3');
        expect(wrapper.text()).not.toContain('db.baselineNoDrift');
    });

    it('比对结果无漂移时给出明确一致结论，而不是空表格', async () => {
        baselineRows = [{ table: 't_user', fingerprint: 'a', columnCount: 2, primaryKeys: 'id', indexCount: 0, drifted: false, driftDetail: '' }];
        checkSchemaBaseline.mockResolvedValue({ dbId: 1, db: 'mayfly_go', driftedTables: [] });

        const wrapper = mountPanel();
        await flushPromises();
        await wrapper
            .findAllComponents({ name: 'ElButton' })
            .find((btn) => btn.text().includes('db.baselineCheck'))!
            .trigger('click');
        await flushPromises();

        expect(wrapper.text()).toContain('db.baselineNoDrift');
    });

    it('未选目标库时不发请求，也不给出「无差异」的空结果', async () => {
        const wrapper = mountPanel();
        await flushPromises();
        await clickCompare(wrapper);

        expect(schemaDiff).not.toHaveBeenCalled();
        expect(wrapper.text()).toContain('db.diffNeedTarget');
        expect(wrapper.text()).not.toContain('db.diffClean');
    });

    it('两侧落到同一个库时不发请求：自比的「无差异」不是结论', async () => {
        stubTarget = { id: 1, db: 'mayfly_go' };
        const wrapper = mountPanel();
        await flushPromises();
        await selectTarget(wrapper);

        await clickCompare(wrapper);
        expect(schemaDiff).not.toHaveBeenCalled();
        expect(wrapper.text()).toContain('db.diffSameDb');
        expect(wrapper.text()).not.toContain('db.diffClean');
    });

    it('对比失败时把原因留在面板上：只剩一张空表会被读成「比过了，没差异」', async () => {
        const wrapper = mountPanel();
        await flushPromises();
        await selectTarget(wrapper);

        schemaDiff.mockRejectedValue(new Error('target database is required'));
        await clickCompare(wrapper);

        expect(wrapper.text()).toContain('target database is required');
        expect(wrapper.text()).not.toContain('db.diffClean');
    });

    it('请求参数带两侧的真实库名（目标库由资源树选到具体库）', async () => {
        const wrapper = mountPanel();
        await flushPromises();
        await selectTarget(wrapper);

        schemaDiff.mockResolvedValue({ source: 's', target: 't', compared: 1, diffTables: 0 });
        await clickCompare(wrapper);

        expect(schemaDiff).toHaveBeenCalledWith({
            srcDbId: 1,
            srcDb: 'mayfly_go',
            dstDbId: 2,
            dstDb: 'other_db',
            tables: undefined,
        });
    });

    it('表清单候选跟着源库走：切了源库不重拉，用户会选到上一个库的表名', async () => {
        const wrapper = mount(DbSchemaDiff, {
            props: { srcDbId: 1, srcDbs: ['mayfly_go', 'tokhub'] },
            global: { plugins: [i18n, ElementPlus], stubs: { DbSelectTree: DbSelectTreeStub, TagCodePath: true } },
        });
        await flushPromises();
        expect(tableInfos).toHaveBeenLastCalledWith({ id: 1, db: 'mayfly_go', limit: 500, offset: 0 });

        // 换源库（多真实库名时的源库下拉）
        const sourceSelect = wrapper.findAllComponents({ name: 'ElSelect' })[0];
        await sourceSelect.vm.$emit('update:modelValue', 'tokhub');
        await flushPromises();

        expect(tableInfos).toHaveBeenLastCalledWith({ id: 1, db: 'tokhub', limit: 500, offset: 0 });
    });

    it('差异与「比不了」分开计数：合并成一个数字就等于把观测缺口算进结论', async () => {
        const wrapper = mountPanel();
        await flushPromises();
        await selectTarget(wrapper);

        schemaDiff.mockResolvedValue({
            source: 's',
            target: 't',
            compared: 12,
            diffTables: 1,
            note: 'only the first 50 tables were compared',
            tables: [
                {
                    table: 't_user',
                    kind: 'changed',
                    columns: [{ column: 'age', kind: 'column-type-changed', source: 'int', target: 'bigint' }],
                    unavailable: [{ aspect: 'indexes', reason: 'index introspection failed on the source' }],
                },
            ],
        });
        await clickCompare(wrapper);

        expect(wrapper.text()).toContain('db.diffChanged');
        expect(wrapper.text()).toContain('only the first 50 tables were compared');
        expect(wrapper.text()).toContain('db.diffColumnCount');
        expect(wrapper.text()).toContain('db.diffUnavailableCount');
    });
});
