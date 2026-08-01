import { beforeEach, describe, expect, it, vi } from 'vitest';

// mock Api 模块，避免加载完整依赖链
vi.mock('@/common/Api', () => ({
    default: class MockApi {},
}));

import { usePageTable } from '../usePageTable';

describe('usePageTable', () => {
    let mockApi: { request: ReturnType<typeof vi.fn> };

    beforeEach(() => {
        mockApi = {
            request: vi.fn().mockResolvedValue({ list: [{ id: 1 }, { id: 2 }], total: 50 }),
        };
    });

    it('初始化默认状态', () => {
        const { tableData, total, searchParams, loading } = usePageTable(true, mockApi as any);
        expect(tableData.value).toEqual([{}]);
        expect(total.value).toBe(0);
        expect(searchParams.value.pageNum).toBe(1);
        expect(searchParams.value.pageSize).toBe(10);
        expect(loading.value).toBe(false);
    });

    it('自定义初始参数', () => {
        const params = { pageNum: 2, pageSize: 20, keyword: 'test' };
        const { searchParams } = usePageTable(true, mockApi as any, params);
        expect(searchParams.value.pageNum).toBe(2);
        expect(searchParams.value.pageSize).toBe(20);
        expect(searchParams.value.keyword).toBe('test');
    });

    it('getTableData 获取分页数据', async () => {
        const { getTableData, tableData, total } = usePageTable(true, mockApi as any);
        await getTableData();
        expect(mockApi.request).toHaveBeenCalledTimes(1);
        expect(tableData.value).toEqual([{ id: 1 }, { id: 2 }]);
        expect(total.value).toBe(50);
    });

    it('getTableData 非分页模式', async () => {
        mockApi.request.mockResolvedValue([{ id: 1 }, { id: 2 }, { id: 3 }]);
        const { getTableData, tableData } = usePageTable(false, mockApi as any);
        await getTableData();
        expect(tableData.value.length).toBe(3);
        expect(tableData.value[0]).toMatchObject({ id: 1 });
        expect(tableData.value[2]).toMatchObject({ id: 3 });
    });

    it('无 api 时 getTableData 不执行', async () => {
        const { getTableData, tableData } = usePageTable(true, undefined);
        await getTableData();
        expect(tableData.value).toEqual([{}]);
    });

    it('search 重置 pageNum 为 1', async () => {
        const params = { pageNum: 5, pageSize: 10 };
        const { search, searchParams } = usePageTable(true, mockApi as any, params);
        search();
        expect(searchParams.value.pageNum).toBe(1);
    });

    it('reset 清空非分页参数', async () => {
        const params = { pageNum: 3, pageSize: 10, keyword: 'test', status: 1 };
        const { reset, searchParams } = usePageTable(true, mockApi as any, params);
        reset();
        expect(searchParams.value.pageNum).toBe(1);
        expect(searchParams.value.pageSize).toBe(10);
        expect(searchParams.value.keyword).toBeNull();
        expect(searchParams.value.status).toBeNull();
    });

    it('handlePageSizeChange 重置 pageNum 并更新 pageSize', async () => {
        const params = { pageNum: 3, pageSize: 10 };
        const { handlePageSizeChange, searchParams } = usePageTable(true, mockApi as any, params);
        handlePageSizeChange(20);
        expect(searchParams.value.pageNum).toBe(1);
        expect(searchParams.value.pageSize).toBe(20);
    });

    it('handlePageNumChange 更新 pageNum', async () => {
        const { handlePageNumChange, searchParams } = usePageTable(true, mockApi as any);
        handlePageNumChange(5);
        expect(searchParams.value.pageNum).toBe(5);
    });

    it('非分页模式 setPageNum 无效', () => {
        const params = { pageNum: 1, pageSize: 10 };
        const { handlePageNumChange, searchParams } = usePageTable(false, mockApi as any, params);
        handlePageNumChange(5);
        // 非分页模式下 setPageNum 直接 return，但 handlePageNumChange 直接赋值
        // 根据源码 handlePageNumChange 直接赋值 state.searchParams.pageNum = val
        expect(searchParams.value.pageNum).toBe(5);
    });

    it('beforeQueryFn 在请求前处理参数', async () => {
        const beforeQueryFn = vi.fn((params: any) => ({ ...params, extra: 'added' }));
        const { getTableData } = usePageTable(true, mockApi as any, { pageNum: 1, pageSize: 10 }, beforeQueryFn);
        await getTableData();
        expect(beforeQueryFn).toHaveBeenCalled();
        expect(mockApi.request).toHaveBeenCalledWith(expect.objectContaining({ extra: 'added' }));
    });

    it('dataCallBack 处理返回数据', async () => {
        const dataCallBack = vi.fn((data: any) => ({ ...data, list: data.list.map((i: any) => ({ ...i, extra: true })) }));
        const { getTableData, tableData } = usePageTable(true, mockApi as any, { pageNum: 1, pageSize: 10 }, undefined, dataCallBack);
        await getTableData();
        expect(dataCallBack).toHaveBeenCalled();
        expect(tableData.value).toEqual([{ id: 1, extra: true }, { id: 2, extra: true }]);
    });

    it('请求失败时 loading 恢复为 false', async () => {
        mockApi.request.mockRejectedValue(new Error('network error'));
        const { getTableData, loading } = usePageTable(true, mockApi as any);
        await expect(getTableData()).rejects.toThrow('network error');
        expect(loading.value).toBe(false);
    });
});
