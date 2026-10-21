import { describe, expect, it, vi } from 'vitest';
import { flushPromises, mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createPinia, setActivePinia } from 'pinia';
import { createMemoryHistory, createRouter } from 'vue-router';
import { defineComponent } from 'vue';
import { useUserInfo } from '@/store/userInfo';

// 资源卡只在挂载时取一次列表，这里直接把一个总数喂进去，验证它是否真的上报到父级
vi.mock('@/views/ops/db/api', () => ({
    dbApi: {
        dbs: {
            request: vi.fn().mockResolvedValue({ list: [], total: 7 }),
        },
    },
}));

import Database from '../resources/Database.vue';

/**
 * 首页「资源总数」= 8 张资源卡各自上报的 total 之和。
 *
 * 资源卡外层（Database.vue 等）没有声明 loaded 事件，监听器靠 attrs 透传到根节点 Base.vue 的 emits。
 * 这条链路一旦被后续改动打断（外层加根节点、改成多根模板），首屏不会报错，只会永远显示「—」，
 * 因此这里用真实挂载把「包装组件 → Base → 父级」的透传钉住
 */
describe('资源卡总数上报', () => {
    it('加载完成后把 total 透传给父级监听器', async () => {
        const pinia = createPinia();
        setActivePinia(pinia);
        useUserInfo().userInfo = { username: 'admin', name: '管理员', permissions: ['resource:base'] };

        const router = createRouter({
            history: createMemoryHistory(),
            routes: [{ path: '/', component: { template: '<div />' } }],
        });

        const Parent = defineComponent({
            components: { Database },
            data() {
                return { totals: [] as number[] };
            },
            methods: {
                onLoaded(total: number) {
                    this.totals.push(total);
                },
            },
            template: '<Database @loaded="onLoaded" />',
        });

        const wrapper = mount(Parent, {
            global: {
                plugins: [pinia, router, ElementPlus],
                mocks: { $t: (key: string) => key },
            },
        });

        await flushPromises();

        expect((wrapper.vm as unknown as { totals: number[] }).totals).toEqual([7]);
        expect(wrapper.text()).toContain('7');
    });
});
