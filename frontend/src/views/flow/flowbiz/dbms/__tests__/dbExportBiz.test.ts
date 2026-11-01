/**
 * 导出工单详情（业务信息页签）。
 *
 * 钉住三件事：
 * 1. 工单未通过（status!=2）时**不给下载按钮**，只说明原因——否则用户点了会拿到一句
 *    「尚未审批通过」，看起来像功能坏了；
 * 2. 通过后按钮出现，且下载地址里带的是**这张工单的 id** 与被批的库名（凭据即工单）；
 * 3. bizForm 不是本组件认识的形态时按空表单渲染，不能在详情页抛错（历史工单/别的业务串号）。
 */
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import ElementPlus from 'element-plus';
import DbExportBiz from '../DbExportBiz.vue';

vi.mock('@/common/config', () => ({ default: { baseApiUrl: '/api' } }));
vi.mock('@/common/request', () => ({ joinClientParams: () => 'token=t&clientId=c' }));

const i18n = createI18n({
    legacy: false,
    globalInjection: true,
    locale: 'zh-cn',
    missingWarn: false,
    fallbackWarn: false,
    messages: { 'zh-cn': {} },
});

const authDirective = { mounted: () => undefined, updated: () => undefined };

function ticket(status: number, bizForm: string, id = 900001) {
    return { id, status, bizType: 'db_export_flow', bizForm };
}

function mountDetail(procinst: Record<string, unknown>) {
    return mount(DbExportBiz, {
        props: { procinst },
        global: { plugins: [i18n, ElementPlus], directives: { auth: authDirective } },
    });
}

const openSpy = vi.fn();
beforeEach(() => {
    openSpy.mockReset();
    vi.stubGlobal('open', openSpy);
});
afterEach(() => vi.unstubAllGlobals());

describe('导出工单详情', () => {
    it('未通过时不给下载按钮，只说明原因', () => {
        const wrapper = mountDetail(ticket(1, '{"dbId":1,"dbName":"mayfly_go","tables":["t_a"],"dumpType":"3"}'));
        expect(wrapper.findAll('button')).toHaveLength(0);
        expect(wrapper.text()).toContain('flow.exportNotApproved');
    });

    it('通过后按钮出现，下载地址带工单 id 与被批的库名', async () => {
        const wrapper = mountDetail(ticket(2, '{"dbId":1,"dbName":"mayfly_go","tables":["t_a"],"dumpType":"3"}'));
        const buttons = wrapper.findAll('button');
        expect(buttons).toHaveLength(1);

        await buttons[0].trigger('click');
        expect(openSpy).toHaveBeenCalledTimes(1);
        const url = openSpy.mock.calls[0][0] as string;
        expect(url).toContain('/dbs/1/export-tickets/900001/download');
        expect(url).toContain('db=mayfly_go');
        // token 由查询参数携带：与即席导出同一套下载方式
        expect(url).toContain('token=t');
    });

    it('bizForm 不可解析时按空表单渲染且不抛错', () => {
        const wrapper = mountDetail(ticket(2, 'not-json'));
        // 没有库信息就不构成可下载的凭据：按钮不应出现
        expect(wrapper.findAll('button')).toHaveLength(0);
        expect(wrapper.text()).not.toContain('undefined');
    });
});
