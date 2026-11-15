import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import { defineComponent, h } from 'vue';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';

import SqlExecResultTabs from '../SqlExecResultTabs.vue';
import { elTableV2Adapter } from '@/components/virtual-table/adapters';
import type { AdapterPropsInput } from '@/components/virtual-table/adapters';

/**
 * 结果表格的高度必须由布局给出，不能按像素硬算
 *
 * 曾经编辑器按「容器高 - 编辑器高 - 固定扣减」算出 tableDataHeight 传给结果表格，扣减里没算页签头
 * 与「表格/图表」切换行（合计约 58px），表格因此比可用空间高出一截；而 `.el-tabs__content` 是
 * `overflow: hidden`，被裁掉的正好是表格底部那几像素——el-table-v2 的横向滚动条就贴在那里，
 * 于是「列放不下却既看不到也拖不动横向滚动条」（真机实测：滚动条落在裁剪区下方 31px）。
 * 这里钉住两条不变式：结果表格不接收固定高度、而是撑满页签剩余空间；滚动条常驻可见。
 */

const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });

// 只关心宿主给表格的接线（高度与 flex 撑满），真表格需要尺寸测量环境，Stub 掉
const DbTableDataStub = defineComponent({
    name: 'DbTableData',
    props: {
        height: { type: String, default: '' },
        data: { type: Array, default: () => [] },
    },
    setup: (props) => () => h('div', { class: 'db-table-data-stub', 'data-height': props.height }),
});

const makeTab = () => ({
    id: 1,
    sql: 'select * from t',
    data: [{ id: 1 }],
    table: 't',
    tableColumn: [{ columnName: 'id', key: 'id', show: true }],
    errorMsg: '',
    notice: '',
    cancelled: false,
    truncated: false,
    hasUpdatedFields: false,
    needApprovalSqls: [],
    dbTableRef: null,
});

const mountResultTabs = () =>
    mount(SqlExecResultTabs, {
        props: {
            execResTabs: [makeTab()] as never,
            activeTab: 1,
            dbId: 1,
            db: 'db1',
            tableDataEmptyText: 'No Data',
        },
        global: {
            plugins: [i18n, ElementPlus],
            stubs: {
                DbTableData: DbTableDataStub,
                SvgIcon: { template: '<i />' },
                ElPopover: { template: '<div><slot /></div>' },
            },
        },
    });

describe('SQL 结果表格由布局撑满剩余高度', () => {
    it('不给表格传固定像素高度：高度只能来自页签的剩余空间', () => {
        const wrapper = mountResultTabs();
        const table = wrapper.findComponent(DbTableDataStub);
        expect(table.exists()).toBe(true);
        // 传了高度就会重新出现「按像素硬算 → 底部被 .el-tabs__content 裁掉 → 横向滚动条不可达」
        expect(table.props('height')).toBe('');
    });

    it('表格带 flex-1 与 min-h-0：随页签高度伸缩，且允许被压缩到可用空间内', () => {
        const wrapper = mountResultTabs();
        const cls = wrapper.findComponent(DbTableDataStub).attributes('class') ?? '';
        // min-h-0 缺位时 flex 子项的自动最小尺寸是内容高度，表格又会顶穿页签
        expect(cls).toContain('flex-1');
        expect(cls).toContain('min-h-0');
    });
});

describe('虚拟表格滚动条常驻', () => {
    const baseInput = {
        columns: [],
        data: [],
        width: 800,
        height: 300,
        headerHeight: 30,
        rowHeight: 30,
    } as unknown as AdapterPropsInput;

    it('开启 scrollbarAlwaysOn，不依赖鼠标悬停才显出可拖动的把手', () => {
        // EP 默认只在 hover 时显出 6px 细条，宽结果集里等于没有横向滚动入口
        expect(elTableV2Adapter.getProps(baseInput).scrollbarAlwaysOn).toBe(true);
    });
});
