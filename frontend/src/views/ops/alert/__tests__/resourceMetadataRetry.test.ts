import { describe, expect, it, vi } from 'vitest';

/**
 * 资源类型/指标元信息的取数降级语义。
 *
 * 判据存在的原因：`fetchResourceMetrics` 用一个模块级 Promise 做「进程内只请求一次」的缓存，
 * 失败分支曾把缓存位写成 `Promise.resolve([])`（而非放回 null），于是**一次瞬时故障就把降级
 * 状态钉死整个页面生命周期**：告警规则表单的「资源类型」下拉永久为空、已选值退化成裸数字 `1`，
 * 用户配不出任何新规则，只能整页刷新恢复。实测触发场景是后端重启期间页面恰好发起过该请求。
 */
const request = vi.fn();

vi.mock('../api', () => ({
    __esModule: true,
    alertRuleApi: { metrics: { request: (...args: unknown[]) => request(...args) } },
}));

/** 每个用例都取一份干净的模块实例：缓存位是模块私有的，复用会串味 */
async function loadUtils() {
    vi.resetModules();
    return await import('../utils');
}

const machineMetrics = [{ resourceType: 1, resourceKey: 'machine', metrics: [] }];

describe('告警资源类型元信息取数', () => {
    it('失败后下一次调用必须重新发请求（不得把空结果永久缓存）', async () => {
        const { fetchResourceMetrics } = await loadUtils();

        request.mockReset();
        request.mockRejectedValueOnce(new Error('backend restarted'));
        await expect(fetchResourceMetrics()).resolves.toEqual([]);
        expect(request).toHaveBeenCalledTimes(1);

        request.mockResolvedValueOnce(machineMetrics);
        await expect(fetchResourceMetrics()).resolves.toEqual(machineMetrics);
        // 关键断言：降级不是终态，恢复后能自己拿到数据
        expect(request).toHaveBeenCalledTimes(2);
    });

    it('成功后仍是进程内只请求一次（缓存不能被顺带改坏）', async () => {
        const { fetchResourceMetrics } = await loadUtils();

        request.mockReset();
        request.mockResolvedValue(machineMetrics);
        await fetchResourceMetrics();
        await fetchResourceMetrics();
        expect(request).toHaveBeenCalledTimes(1);
    });

    it('失败恢复后 supportedResourceTypes 能被填上（下拉不再空）', async () => {
        const { ensureResourceMetadata, supportedResourceTypes } = await loadUtils();

        request.mockReset();
        request.mockRejectedValueOnce(new Error('transient'));
        await ensureResourceMetadata();
        expect(supportedResourceTypes.value).toEqual([]);

        request.mockResolvedValueOnce(machineMetrics);
        await ensureResourceMetadata();
        expect(supportedResourceTypes.value.map((it) => it.value)).toEqual([1]);
    });
});
