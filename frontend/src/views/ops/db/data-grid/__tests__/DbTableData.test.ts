import { describe, expect, it, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { defineComponent } from 'vue';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';

import DbTableData from '../DbTableData.vue';

/**
 * 加载遮罩与取消按钮的同步路径
 *
 * 取消/报错会把本组件卸载（结果页改渲染「已取消/失败」块），再次执行时父级先清
 * cancelled/errorMsg、再置 loading=true——组件重新挂载那一刻 props.loading 已是 true。
 * watch 不回调初值，曾导致整轮执行无遮罩、无取消按钮（页签标签转圈、表格区却看似空闲）。
 * 这里钉两条路径：挂载即 loading 的初值同步、挂载后 false→true 的 watch 同步。
 */

vi.mock('@/views/ops/db/db', () => ({
    DbInst: { getInst: () => ({ type: 'mysql' }) },
    DbThemeConfig: { showColumnComment: false },
}));

const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });

// 只关心遮罩插槽是否随 loading 出现：真 VirtualTable 需要尺寸测量环境，Stub 掉只保留 overlay 语义
const VirtualTableStub = defineComponent({
    name: 'VirtualTable',
    props: ['data', 'columns', 'loading'],
    // 组件内经 template ref 调滚动方法（setTableData/active），Stub 必须供同名空实现
    setup() {
        return { scrollTo: () => undefined, scrollToLeft: () => undefined };
    },
    template: '<div class="vt-stub"><slot name="overlay" v-if="loading" /></div>',
});

function mountTable(loading: boolean, abortFn?: () => void) {
    return mount(DbTableData, {
        props: { dbId: 1, db: 'db1', data: [], columns: [], loading, abortFn },
        global: {
            plugins: [i18n, ElementPlus],
            stubs: {
                VirtualTable: VirtualTableStub,
                SvgIcon: { template: '<i />' },
                Contextmenu: true,
                DbTableDataForm: true,
                DbTableRowDetail: true,
            },
        },
    });
}

describe('DbTableData 加载遮罩与取消入口', () => {
    it('挂载时 props.loading 已为 true：遮罩与取消按钮必须在（取消/报错后重跑的挂载场景）', async () => {
        const abort = vi.fn();
        const wrapper = mountTable(true, abort);
        await wrapper.vm.$nextTick();

        const mask = wrapper.find('.el-loading-mask');
        expect(mask.exists(), '挂载即 loading 时遮罩不能丢').toBe(true);
        const cancelBtn = mask.find('button');
        expect(cancelBtn.exists(), '遮罩内必须有取消按钮').toBe(true);
        await cancelBtn.trigger('click');
        expect(abort).toHaveBeenCalledTimes(1);
    });

    it('挂载后 loading 由 false 变 true 再变 false：遮罩随 watch 出现与收起', async () => {
        const wrapper = mountTable(false);
        expect(wrapper.find('.el-loading-mask').exists()).toBe(false);

        await wrapper.setProps({ loading: true });
        expect(wrapper.find('.el-loading-mask').exists()).toBe(true);

        await wrapper.setProps({ loading: false });
        expect(wrapper.find('.el-loading-mask').exists()).toBe(false);
    });

    it('loading 为 true 但无 abortFn：遮罩在而不渲染取消按钮（无可取消的请求时不给假入口）', async () => {
        const wrapper = mountTable(true);
        await wrapper.vm.$nextTick();
        const mask = wrapper.find('.el-loading-mask');
        expect(mask.exists()).toBe(true);
        expect(mask.find('button').exists()).toBe(false);
    });
});
