import { beforeEach, describe, expect, it, vi } from 'vitest';
import { mount } from '@vue/test-utils';

// 组件在 setup 同步阶段就会调接口，这里把 API 与提示层整体替掉
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }));
vi.mock('@/hooks/useI18n', () => ({
    Msg: { success: vi.fn(), warning: vi.fn(), deleteSuccess: vi.fn() },
    useI18nDeleteConfirm: vi.fn(),
}));
vi.mock('../../api', () => ({
    machineApi: {
        files: { request: vi.fn() },
        addConf: { request: vi.fn() },
        delConf: { request: vi.fn() },
    },
}));

import FileConfTable from '../FileConfTable.vue';
import { machineApi } from '../../api';

// 只验证数据获取时机，Element Plus 组件全部空桩（不渲染插槽，避免 jsdom 下真实表格布局开销）
const stubs = {
    ElTable: true,
    ElTableColumn: true,
    ElInput: true,
    ElButton: true,
    ElPagination: true,
    EnumSelect: true,
};

// 模板里用全局 $t，测试环境没装 i18n 插件，直接回显 key
const global = { stubs, mocks: { $t: (key: string) => key } };

const filesRequest = machineApi.files.request as unknown as ReturnType<typeof vi.fn>;

describe('FileConfTable 配置列表拉取', () => {
    beforeEach(() => {
        filesRequest.mockReset();
    });

    /**
     * 回归：watch 是 immediate 的，setup 同步阶段就调用 getFiles。
     * getFiles 一旦写成 const 箭头函数（声明在 watch 之后）会命中 TDZ，
     * 抛「Cannot access 'getFiles' before initialization」，整个文件管理 tab 直接白掉。
     */
    it('带 machineId 挂载即拉取配置列表，不抛错', async () => {
        filesRequest.mockResolvedValue({ list: [], total: 0 });

        const wrapper = mount(FileConfTable, { props: { machineId: 7 }, global });
        await wrapper.vm.$nextTick();

        expect(filesRequest).toHaveBeenCalledTimes(1);
        expect(filesRequest).toHaveBeenCalledWith({ id: 7, pageNum: 1, pageSize: 8 });

        wrapper.unmount();
    });

    it('未选机器时不请求，且清空已有列表', async () => {
        const wrapper = mount(FileConfTable, { props: { machineId: null }, global });
        await wrapper.vm.$nextTick();

        expect(filesRequest).not.toHaveBeenCalled();

        wrapper.unmount();
    });

    it('machineId 变化时重新拉取', async () => {
        filesRequest.mockResolvedValue({ list: [], total: 0 });

        const wrapper = mount(FileConfTable, { props: { machineId: 7 }, global });
        await wrapper.setProps({ machineId: 8 });
        await wrapper.vm.$nextTick();

        expect(filesRequest).toHaveBeenCalledTimes(2);
        expect(filesRequest).toHaveBeenLastCalledWith({ id: 8, pageNum: 1, pageSize: 8 });

        wrapper.unmount();
    });
});
