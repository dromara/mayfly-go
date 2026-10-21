import { describe, expect, it, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createPinia, setActivePinia } from 'pinia';
import { createI18n } from 'vue-i18n';

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key } },
}));

import zhHome from '@/i18n/zh-cn/home';
import { useUserInfo } from '@/store/userInfo';
import QuickActions from '../components/QuickActions.vue';

const mountActions = (permissions: string[]) => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useUserInfo().userInfo = { username: 'operator', name: 'Operator', permissions };

    return mount(QuickActions, {
        global: {
            plugins: [pinia, createI18n({ legacy: false, locale: 'zh-cn', messages: { 'zh-cn': zhHome } }), ElementPlus],
            mocks: { $t: (key: string) => key },
        },
    });
};

/**
 * 入口可见性只看基础权限码（resource:base / ai:chat / flow:base）：
 * 菜单授权时会连带下发基础码，所以没有码就等于没有菜单，入口不渲染，避免点了跳不过去
 */
describe('快捷操作权限过滤', () => {
    it('持有四个基础权限码时渲染全部入口', () => {
        const wrapper = mountActions(['resource:base', 'alert:event', 'ai:chat', 'flow:base']);
        expect(wrapper.findAll('.quick-action').length).toBe(4);
    });

    it('缺少 ai:chat 时隐藏 AI 助手入口', () => {
        const wrapper = mountActions(['resource:base', 'alert:event', 'flow:base']);
        expect(wrapper.findAll('.quick-action').length).toBe(3);
        expect(wrapper.text()).not.toContain('home.goAiAssistant');
    });

    it('缺少 resource:base 时隐藏我的资源入口', () => {
        const wrapper = mountActions(['alert:event', 'ai:chat', 'flow:base']);
        expect(wrapper.findAll('.quick-action').length).toBe(3);
        expect(wrapper.text()).not.toContain('home.goMyResource');
    });

    it('缺少 alert:event 时隐藏告警中心入口', () => {
        const wrapper = mountActions(['resource:base', 'ai:chat', 'flow:base']);
        expect(wrapper.findAll('.quick-action').length).toBe(3);
        expect(wrapper.text()).not.toContain('home.goAlerts');
    });

    it('所有入口都无权限时展示空态而不是空面板', () => {
        const wrapper = mountActions([]);
        expect(wrapper.findAll('.quick-action').length).toBe(0);
        expect(wrapper.text()).toContain('home.noQuickAction');
    });
});
