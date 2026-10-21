import { describe, expect, it, vi } from 'vitest';
import { flushPromises, mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createPinia } from 'pinia';
import { createI18n } from 'vue-i18n';

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key } },
}));

// vi.mock 会被提升到文件顶部，共享 spy 必须一并提升，否则落到 TDZ
const { updateAccount } = vi.hoisted(() => ({ updateAccount: vi.fn().mockResolvedValue(undefined) }));

vi.mock('../api', () => ({
    personApi: {
        accountInfo: {
            request: vi.fn().mockResolvedValue({
                // 后端返回的是 roleName（vo.AccountRoleVO），不是角色实体的 name
                roles: [{ roleId: 1, roleName: '管理员', code: 'admin', status: 1 }],
            }),
        },
        authStatus: { request: vi.fn().mockResolvedValue({ enable: true, bind: false }) },
        updateAccount: { request: updateAccount },
        unbindOauth2: { request: vi.fn().mockResolvedValue(undefined) },
    },
}));

import zhCommon from '@/i18n/zh-cn/common';
import Personal from '../index.vue';

const createTestI18n = () =>
    createI18n({
        legacy: false,
        locale: 'zh-cn',
        messages: { 'zh-cn': zhCommon },
    });

describe('个人中心', () => {
    const mountPage = async () => {
        const wrapper = mount(Personal, {
            global: {
                plugins: [createPinia(), createTestI18n(), ElementPlus],
            },
        });
        await flushPromises();
        return wrapper;
    };

    it('按 roleName 渲染角色，不把角色名渲染成空白', async () => {
        const wrapper = await mountPage();
        expect(wrapper.findAll('.role-pill').map((node) => node.text())).toEqual(['管理员']);
    });

    it('两次密码不一致时只报校验错误，不发请求', async () => {
        const wrapper = await mountPage();
        const inputs = wrapper.findAll('input[type="password"]');
        expect(inputs.length).toBe(2);

        await inputs[0].setValue('Abc123!@#');
        await inputs[1].setValue('Abc123!@#x');
        await wrapper.find('.pwd-form button').trigger('click');
        await flushPromises();

        // 表单校验链路在测试环境会异常放行，提交处必须自己拦住不一致的口令
        expect(updateAccount).not.toHaveBeenCalled();
    });

    it('两次密码一致且符合强度规则时提交，并在成功后清空输入', async () => {
        updateAccount.mockClear();
        const wrapper = await mountPage();
        const inputs = wrapper.findAll('input[type="password"]');

        await inputs[0].setValue('Abc123!@#');
        await inputs[1].setValue('Abc123!@#');
        await wrapper.find('.pwd-form button').trigger('click');
        await flushPromises();

        expect(updateAccount).toHaveBeenCalledTimes(1);
        expect(updateAccount).toHaveBeenCalledWith({ password: 'Abc123!@#' });
        expect((inputs[0].element as HTMLInputElement).value).toBe('');
        expect((inputs[1].element as HTMLInputElement).value).toBe('');
    });
});
