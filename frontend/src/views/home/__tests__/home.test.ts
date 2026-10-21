import { describe, expect, it, vi } from 'vitest';
import { flushPromises, mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createPinia, setActivePinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import { createMemoryHistory, createRouter } from 'vue-router';
import { defineComponent, h, onMounted } from 'vue';

// 首页时间口径走 i18n.global.t，这里只固定输出形态；组件内的 $t 用下面真实实例渲染中文
vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key } },
}));

vi.mock('@/views/ops/alert/api', () => ({
    alertEventApi: {
        list: {
            request: vi.fn().mockResolvedValue({
                total: 2,
                list: [
                    {
                        id: 1,
                        ruleName: 'CPU 使用率过高',
                        priority: 1,
                        resourceType: 1,
                        resourceId: 8,
                        resourceName: 'prod-web-01',
                        metric: 'cpu_rate',
                        currentValue: 93.5,
                        lastTriggerTime: new Date().toISOString(),
                    },
                ],
            }),
        },
    },
}));

vi.mock('@/views/flow/api', () => ({
    procinstTaskApi: {
        tasks: {
            request: vi.fn().mockResolvedValue({
                total: 3,
                list: [{ id: 11, procinstId: 7, nodeName: 'SQL 变更审批', bizKey: 'db_sql_exec_flow', createTime: new Date().toISOString() }],
            }),
        },
    },
}));

vi.mock('@/views/personal/api', () => ({
    personApi: {
        accountInfo: { request: vi.fn().mockResolvedValue({ roles: [{ id: 1, name: '管理员' }] }) },
        getUnreadMsgCount: { request: vi.fn().mockResolvedValue(9) },
    },
}));

vi.mock('@/views/ops/tag/api', () => ({
    tagApi: { listByQuery: { request: vi.fn().mockResolvedValue([]) } },
    resourceOpLogApi: {
        getAccountResourceOpLogs: {
            request: vi.fn().mockResolvedValue({
                total: 1,
                list: [{ id: 1, codePath: 'default/prod/', createTime: new Date().toISOString() }],
            }),
        },
    },
}));

// 资源卡只验证「总数上卷」这一个契约：每张卡上报 3，8 张卡合计 24
vi.mock('../resources', () => {
    const ResourceStub = defineComponent({
        name: 'ResourceStub',
        emits: ['loaded'],
        setup(_, { emit }) {
            onMounted(() => emit('loaded', 3));
            return () => h('div', { class: 'resource-stub' });
        },
    });
    return { resourceComponents: Array.from({ length: 8 }, () => ResourceStub) };
});

import zhHome from '@/i18n/zh-cn/home';
import zhCommon from '@/i18n/zh-cn/common';
import { useUserInfo } from '@/store/userInfo';
import Home from '../Home.vue';

const createTestI18n = () =>
    createI18n({
        legacy: false,
        locale: 'zh-cn',
        messages: { 'zh-cn': { ...zhHome, ...zhCommon } },
    });

const createTestRouter = () =>
    createRouter({
        history: createMemoryHistory(),
        routes: [
            { path: '/home', name: 'Home', component: { template: '<div />' } },
            { path: '/my-resource', name: 'ResourceOp', component: { template: '<div />' } },
            { path: '/flow/procinst-tasks', name: 'ProcinstTaskList', component: { template: '<div />' } },
            { path: '/alert/alert-events', name: 'AlertEventList', component: { template: '<div />' } },
            { path: '/ai/assistant', name: 'AiAssistant', component: { template: '<div />' } },
            { path: '/flow/procinsts', name: 'ProcinstList', component: { template: '<div />' } },
            { path: '/dbms/transfer', name: 'DbTransferList', component: { template: '<div />' } },
            { path: '/personal', name: 'Personal', component: { template: '<div />' } },
        ],
    });

describe('首页工作台', () => {
    it('渲染问候区、指标卡、待办与告警，并把资源卡总数汇总成资源总数', async () => {
        const pinia = createPinia();
        setActivePinia(pinia);
        // 快捷入口与面板跳转都按基础权限码控制，这里给全首页涉及的码
        useUserInfo().userInfo = {
            username: 'admin',
            name: '管理员',
            permissions: ['resource:base', 'ai:chat', 'flow:base', 'flow:task:base', 'alert:event'],
        };

        const wrapper = mount(Home, {
            global: {
                plugins: [pinia, createTestRouter(), createTestI18n(), ElementPlus],
                // main.ts 里 SvgIcon 是全局注册的，TagCodePath 这类间接子组件依赖它
                components: { SvgIcon: defineComponent({ name: 'SvgIcon', render: () => h('i') }) },
                mocks: { $t: (key: string) => key },
            },
        });

        await flushPromises();

        // 指标卡：待办 3 / 告警 2 / 未读 9 / 资源总数 8×3
        expect(wrapper.findAll('.stat-tile__value').map((node) => node.text())).toEqual(['3', '2', '9', '24']);

        // 工作台面板与最近操作都拿到了数据
        expect(wrapper.text()).toContain('SQL 变更审批');
        expect(wrapper.text()).toContain('CPU 使用率过高');
        expect(wrapper.text()).toContain('prod-web-01');
        expect(wrapper.findAll('.op-item').length).toBe(1);

        // 快捷操作四个入口都在（无权限的入口会按基础权限码自动隐藏）
        expect(wrapper.findAll('.quick-action').length).toBe(4);
    });
});
