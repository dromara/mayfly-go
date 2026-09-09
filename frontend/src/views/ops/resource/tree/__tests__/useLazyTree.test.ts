import { beforeEach, describe, expect, it, vi } from 'vitest';

import { getContributor, registerContributor } from '../registry';
import { LOADING_KIND, type TreeNodeData } from '../types';
import { useLazyTree } from '../useLazyTree';

/**
 * 懒加载水合层专测：占位、水合、失败重试、折叠释放、定位展开、根刷新。
 * 全部 mock 数据驱动，无 i18n/API 依赖。
 */

let leafSeq = 0;
const leaf = (parent: string): TreeNodeData => ({ key: `${parent}-leaf-${++leafSeq}`, kind: 'leaf', label: 'leaf' });

const setupContributors = () => {
    registerContributor({ kind: 'leaf' });
};

beforeEach(() => {
    setupContributors();
});

describe('tree/useLazyTree 懒加载水合层', () => {
    it('根加载：可展开节点挂加载占位，叶子节点无占位', async () => {
        registerContributor({ kind: 'root', hasChildren: true });
        const { data, init } = useLazyTree({
            loadChildren: async () => [],
            loadRoot: async () => [{ key: 'r1', kind: 'root', label: 'r1' }, { key: 'l1', kind: 'leaf', label: 'l1' }],
        });
        await init();
        expect(data.value.map((n) => n.key)).toEqual(['r1', 'l1']);
        expect(data.value[0].children?.[0].kind).toBe(LOADING_KIND);
        expect(data.value[1].children).toEqual([]);
    });

    it('预构子树的节点展开能力直接成立（无贡献者的 kind 如标签节点可单击/双击切换展开）', async () => {
        // 故意不为 'tag' 注册贡献者：标签骨架由内存预构，展开能力来自 children 而非贡献者
        const { data, init, expandedKeys, expandNode, collapseNode } = useLazyTree({
            loadChildren: async () => [],
            loadRoot: async () => [
                { key: 'tag1', kind: 'tag', label: 'tag', children: [{ key: 'tag1-19', kind: 'group', label: 'g' }] },
            ],
        });
        await init();
        expect(data.value[0].hasChildren).toBe(true);
        expect(data.value[0].loaded).toBe(true);

        // 单击/双击切换走同一条 toggle 链路：expandNode/collapseNode 直接生效
        await expandNode('tag1');
        expect(expandedKeys.value.has('tag1')).toBe(true);
        collapseNode('tag1');
        expect(expandedKeys.value.has('tag1')).toBe(false);
    });

    it('展开水合：占位被真实子节点替换，展开态记录', async () => {
        registerContributor({
            kind: 'root',
            hasChildren: true,
            loadChildren: async () => [leaf('r1')],
        });
        const { data, init, expandNode, expandedKeys, getNode } = useLazyTree({
            loadChildren: async (n) => getContributor(n.kind)!.loadChildren!(n),
            loadRoot: async () => [{ key: 'r1', kind: 'root', label: 'r1' }],
        });
        await init();
        await expandNode('r1');

        expect(expandedKeys.value.has('r1')).toBe(true);
        expect(getNode('r1')!.loaded).toBe(true);
        expect(data.value[0].children?.map((c) => c.kind)).toEqual(['leaf']);
        expect(data.value[0].children?.[0].parentKey).toBe('r1');
    });

    it('水合失败：占位替换为错误行，可再次重试成功', async () => {
        registerContributor({
            kind: 'root',
            hasChildren: true,
        });
        let shouldFail = true;
        const { getNode, init, expandNode } = useLazyTree({
            loadChildren: async () => {
                if (shouldFail) {
                    throw new Error('boom');
                }
                return [leaf('r1')];
            },
            loadRoot: async () => [{ key: 'r1', kind: 'root', label: 'r1' }],
        });
        const error = vi.spyOn(console, 'error').mockImplementation(() => {});

        await init();
        await expandNode('r1');
        const errNode = getNode('r1')!.children?.[0];
        expect(errNode?.kind).toBe('__error__');
        expect(getNode('r1')!.loaded).toBe(false);

        shouldFail = false;
        await expandNode('r1');
        expect(getNode('r1')!.children?.map((c) => c.kind)).toEqual(['leaf']);
        error.mockRestore();
    });

    it('折叠释放：releaseOnCollapse 节点折叠后子树回到占位', async () => {
        registerContributor({
            kind: 'root',
            hasChildren: true,
            releaseOnCollapse: true,
            loadChildren: async () => [leaf('r1')],
        });
        const { getNode, init, expandNode, collapseNode } = useLazyTree({
            loadChildren: async (n) => getContributor(n.kind)!.loadChildren!(n),
            loadRoot: async () => [{ key: 'r1', kind: 'root', label: 'r1' }],
        });
        await init();
        await expandNode('r1');
        expect(getNode('r1')!.loaded).toBe(true);

        collapseNode('r1');
        expect(getNode('r1')!.loaded).toBe(false);
        expect(getNode('r1')!.children?.[0].kind).toBe(LOADING_KIND);

        // 未声明释放的节点折叠后保留已加载子树
        registerContributor({ kind: 'keep', hasChildren: true, loadChildren: async () => [leaf('k1')] });
        const t2 = useLazyTree({
            loadChildren: async (n) => getContributor(n.kind)!.loadChildren!(n),
            loadRoot: async () => [{ key: 'k1', kind: 'keep', label: 'k1' }],
        });
        await t2.init();
        await t2.expandNode('k1');
        t2.collapseNode('k1');
        expect(t2.getNode('k1')!.loaded).toBe(true);
    });

    it('大子树折叠自动释放：超过阈值即使未声明 releaseOnCollapse 也回收，小子树保留', async () => {
        // 大子树（无 releaseOnCollapse 声明）：3 个子节点，阈值设为 3 触发自动释放
        registerContributor({ kind: 'big', hasChildren: true, loadChildren: async () => [leaf('b1'), leaf('b2'), leaf('b3')] });
        // 小子树：2 个子节点低于阈值
        registerContributor({ kind: 'small', hasChildren: true, loadChildren: async () => [leaf('s1'), leaf('s2')] });
        const { getNode, init, expandNode, collapseNode } = useLazyTree({
            loadChildren: async (n) => getContributor(n.kind)!.loadChildren!(n),
            loadRoot: async () => [
                { key: 'b1', kind: 'big', label: 'big' },
                { key: 's1', kind: 'small', label: 'small' },
            ],
            autoReleaseThreshold: 3,
        });
        await init();
        await expandNode('b1');
        await expandNode('s1');

        collapseNode('b1');
        expect(getNode('b1')!.loaded).toBe(false);
        expect(getNode('b1')!.children?.[0].kind).toBe(LOADING_KIND);

        collapseNode('s1');
        // 低于阈值：子树保留，再展开不重新拉取
        expect(getNode('s1')!.loaded).toBe(true);
    });

    it('增量索引：多根交叉展开/释放后 getNode 定位互不干扰（索引只维护受影响子树）', async () => {
        registerContributor({
            kind: 'ra',
            hasChildren: true,
            releaseOnCollapse: true,
            loadChildren: async () => [{ key: 'a-child', kind: 'leaf', label: 'ac' }],
        });
        registerContributor({
            kind: 'rb',
            hasChildren: true,
            releaseOnCollapse: true,
            loadChildren: async () => [{ key: 'b-child', kind: 'leaf', label: 'bc' }],
        });
        const { init, expandNode, collapseNode, getNode } = useLazyTree({
            loadChildren: async (n) => getContributor(n.kind)!.loadChildren!(n),
            loadRoot: async () => [
                { key: 'a1', kind: 'ra', label: 'a' },
                { key: 'b1', kind: 'rb', label: 'b' },
            ],
        });
        await init();
        await expandNode('a1');
        await expandNode('b1');

        // 两棵子树均可在索引中定位，父子关系正确
        expect(getNode('a-child')?.parentKey).toBe('a1');
        expect(getNode('b-child')?.parentKey).toBe('b1');

        // 释放 a1 子树：后代索引被摘除、占位行重新入索引，兄弟子树不受影响
        collapseNode('a1');
        expect(getNode('a-child')).toBeUndefined();
        expect(getNode('a1')!.children?.[0].key).toBe('a1__loading');
        expect(getNode('b-child')?.parentKey).toBe('b1');

        // 再展开重新水合，索引恢复
        await expandNode('a1');
        expect(getNode('a-child')?.parentKey).toBe('a1');
        expect(getNode('b-child')).toBeDefined();
    });

    it('迟到响应作废：加载期间折叠释放，迟到的子树不覆盖占位；再展开重新拉取', async () => {
        const pending: Array<(v: TreeNodeData[]) => void> = [];
        registerContributor({
            kind: 'race',
            hasChildren: true,
            releaseOnCollapse: true,
            loadChildren: () => new Promise<TreeNodeData[]>((resolve) => pending.push(resolve)),
        });
        const { getNode, init, expandNode, collapseNode } = useLazyTree({
            loadChildren: async (n) => getContributor(n.kind)!.loadChildren!(n),
            loadRoot: async () => [{ key: 'r1', kind: 'race', label: 'r' }],
        });
        await init();

        // 发起加载（未完成）→ 折叠释放 → 迟到响应
        const expanding = expandNode('r1');
        collapseNode('r1');
        pending[0]([leaf('r1')]);
        await expanding;

        expect(getNode('r1')!.loaded).toBe(false);
        expect(getNode('r1')!.children?.[0].kind).toBe(LOADING_KIND);
        expect(getNode('r1-leaf-1')).toBeUndefined();

        // 再展开：旧请求已作废，重新发起并正常水合
        const second = expandNode('r1');
        pending[1]([leaf('r1')]);
        await second;
        expect(getNode('r1')!.loaded).toBe(true);
        expect(getNode('r1')!.children?.[0].kind).toBe('leaf');
    });

    it('refresh：指定 key 重载子树并保持展开态，根刷新重建整树', async () => {
        let version = 0;
        registerContributor({
            kind: 'root',
            hasChildren: true,
            loadChildren: async () => [{ key: `leaf-v${version}`, kind: 'leaf', label: 'leaf' }],
        });
        const rootLoader = vi.fn(async () => [{ key: `r-v${version}`, kind: 'root', label: 'r' }]);
        const { getNode, init, expandNode, refresh, data } = useLazyTree({ loadChildren: async (n) => getContributor(n.kind)!.loadChildren!(n), loadRoot: rootLoader });
        await init();
        await expandNode('r-v0');

        version = 1;
        await refresh('r-v0');
        expect(getNode('r-v0')!.children?.[0].key).toBe('leaf-v1');
        expect(expandedKeysValue(getNode, 'r-v0')).toBe(true);

        await refresh();
        expect(rootLoader).toHaveBeenCalledTimes(2);
        expect(data.value[0].key).toBe('r-v1');
    });

    it('折叠父节点同时收起后代展开态（防 TreeV2 setExpandedKeys 经后代链复活祖先）', async () => {
        registerContributor({ kind: 'sub', hasChildren: true });
        registerContributor({
            kind: 'chain',
            hasChildren: true,
            loadChildren: async () => [{ key: 'sub1', kind: 'sub', label: 'sub' }],
        });
        registerContributor({ kind: 'sub', hasChildren: true, loadChildren: async () => [leaf('sub1')] });
        const { expandedKeys, init, expandNode, collapseNode } = useLazyTree({
            loadChildren: async (n) => getContributor(n.kind)!.loadChildren!(n),
            loadRoot: async () => [{ key: 'top1', kind: 'chain', label: 'top' }],
        });
        await init();
        await expandNode('top1');
        await expandNode('sub1');
        expect(expandedKeys.value.has('top1')).toBe(true);
        expect(expandedKeys.value.has('sub1')).toBe(true);

        collapseNode('top1');
        // 父与后代展开态全部清除：回传 TreeV2 后不会经 sub1 的祖先链把 top1 重新展开
        expect(expandedKeys.value.has('top1')).toBe(false);
        expect(expandedKeys.value.has('sub1')).toBe(false);
    });

    it('ensureVisible：逐层水合展开祖先路径使目标可见', async () => {
        registerContributor({ kind: 'mid', hasChildren: true });
        registerContributor({
            kind: 'top',
            hasChildren: true,
            loadChildren: async () => [{ key: 'mid1', kind: 'mid', label: 'mid' }],
        });
        registerContributor({ kind: 'mid', hasChildren: true, loadChildren: async () => [{ key: 'target', kind: 'leaf', label: 't' }] });

        const { init, expandNode, collapseNode, ensureVisible, getNode } = useLazyTree({
            loadChildren: async (n) => getContributor(n.kind)!.loadChildren!(n),
            loadRoot: async () => [{ key: 'top1', kind: 'top', label: 'top' }],
        });
        await init();

        // 深链发现契约：占位之下的未水合 key 不在索引中，返回 false，
        // 由调用方（容器的 pendingLocate 水合后重试）逐层发现
        expect(await ensureVisible('target')).toBe(false);
        await expandNode('top1');
        expect(await ensureVisible('target')).toBe(false);
        await expandNode('mid1');

        // 祖先链已全部入索引：ensureVisible 负责补展开未展开的祖先并确认可见
        expect(await ensureVisible('target')).toBe(true);
        expect(getNode('target')).toBeDefined();
        expect(getNode('mid1')!.loaded).toBe(true);

        // 折叠 mid1（未释放子树）后，ensureVisible 重新展开祖先路径
        collapseNode('mid1');
        expect(await ensureVisible('target')).toBe(true);

        // 不存在的 key 返回 false
        expect(await ensureVisible('nope')).toBe(false);
    });

    /** 展开态以水合层的返回值为准（getNode 仅用于断言便捷性） */
    function expandedKeysValue(getNode: (key: string) => TreeNodeData | undefined, key: string) {
        return (getNode(key) as unknown as { loaded?: boolean }).loaded === true;
    }
});
