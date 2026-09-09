import { nextTick, ref, shallowRef } from 'vue';

import { getContributor, resolveHasChildren } from './registry';
import { ERROR_KIND, LOADING_KIND, type TreeNode, type TreeNodeData } from './types';

/**
 * el-tree-v2 懒加载水合层（TreeV2 无原生 lazy）：
 * - data 为响应式单一事实源，变更后替换根引用触发 TreeV2 watch 重建 flatten（虚拟滚动只渲染可见行）；
 * - 展开态自管：expandedKeys 作为 default-expanded-keys 传回（TreeV2 每次 data 重建后用它恢复展开态）；
 * - 未加载的可展开节点以"加载中"占位子节点呈现（TreeV2 的 isLeaf 纯看 children，占位保证箭头显示）；
 * - 折叠时按 releaseOnCollapse 释放子节点回收内存（等价旧 collapseRemoveChildren）。
 */

export interface LazyTreeOptions {
    /** 按节点 kind 分发的子节点加载器（核心不认识任何具体 kind） */
    loadChildren: (node: TreeNode) => Promise<TreeNodeData[]>;
    /** 根加载器（refresh() 重载根时调用） */
    loadRoot: () => Promise<TreeNodeData[]>;
    /** 大子树折叠自动释放阈值：已加载子节点数达到该值的节点，折叠即释放子树（无视 releaseOnCollapse 声明），缺省 100 */
    autoReleaseThreshold?: number;
}

/** 缺省大子树阈值：DOM 由虚拟滚动约束，释放的是 JS 堆（子树节点对象）；再展开重新拉取，故仅对大子树自动生效 */
const DEFAULT_AUTO_RELEASE_THRESHOLD = 100;

export function useLazyTree(options: LazyTreeOptions) {
    // shallowRef：深层响应式无消费者（所有变更路径最终都经 bump 换根引用驱动 TreeV2 重建），
    // 免去大子树逐节点/params 深代理的内存与 CPU 开销
    const data = shallowRef<TreeNode[]>([]);
    const expandedKeys = ref<Set<string>>(new Set());
    const nodeIndex = new Map<string, TreeNode>();
    const parentIndex = new Map<string, string | undefined>();
    /** in-flight 加载：key → { seq, promise }；seq 与 node.loadSeq 不一致即为作废请求 */
    const loadingKeys = new Map<string, { seq: number; promise: Promise<void> }>();
    const afterHydrateHooks: Array<(node: TreeNode) => void> = [];

    /**
     * 登记一棵子树的索引（不清空旧条目）。
     * 双用途：setRoot 全量重建前先 clear 再调用；hydrate/release 后对受影响子树增量调用
     * （每次展开/释放只触碰该子树，O(子树) 而非 O(全树)，索引其余部分保持有效）
     */
    function indexTree(nodes: TreeNode[], parentKey?: string) {
        for (const n of nodes) {
            nodeIndex.set(n.key, n);
            parentIndex.set(n.key, parentKey);
            if (n.children?.length) {
                indexTree(n.children, n.key);
            }
        }
    }

    /** 摘除子树全部后代索引（子树被释放/替换时；节点自身保留，其 children 即将重建） */
    function unindexSubtree(node: TreeNode) {
        node.children?.forEach((c) => {
            nodeIndex.delete(c.key);
            parentIndex.delete(c.key);
            unindexSubtree(c);
        });
    }

    /** 替换根引用触发 TreeV2 重建（索引由各变更路径增量维护，不在此重建） */
    function bump() {
        data.value = [...data.value];
    }

    function loadingPlaceholder(node: TreeNode): TreeNode {
        return { key: `${node.key}__loading`, kind: LOADING_KIND, label: 'common.processing', hasChildren: false } as TreeNode;
    }

    function errorPlaceholder(node: TreeNode): TreeNode {
        // 携带 retryTarget：容器点击重试时重载该节点子树
        return { key: `${node.key}__error`, kind: ERROR_KIND, label: 'common.retry', hasChildren: false, params: { retryTarget: node.key } } as TreeNode;
    }

    /** 贡献者产出 → 容器节点：自带 children 视为已加载，否则可展开节点挂加载占位 */
    function normalize(d: TreeNodeData, parentKey?: string): TreeNode {
        const contributor = getContributor(d.kind);
        // params 兜底非空：下游命令/渲染器统一可安全取参
        const node: TreeNode = { ...d, params: d.params ?? {}, children: undefined } as TreeNode;
        node.parentKey = parentKey;
        if (node.releaseOnCollapse === undefined && contributor?.releaseOnCollapse) {
            node.releaseOnCollapse = true;
        }
        if (d.children?.length) {
            // 预构子树（如标签骨架）：展开能力直接成立（该 kind 可能无贡献者，如标签节点）
            node.hasChildren = true;
            node.loaded = true;
            node.children = d.children.map((c) => normalize(c, node.key));
        } else {
            if (node.hasChildren === undefined) {
                node.hasChildren = contributor ? resolveHasChildren(contributor, node) : false;
            }
            if (node.hasChildren) {
                node.loaded = false;
                node.children = [loadingPlaceholder(node)];
            } else {
                node.loaded = true;
                node.children = [];
            }
        }
        return node;
    }

    /** 懒加载子节点：统一走注入的按 kind 分发器（核心不认识任何具体 kind）；占位/错误行不参与 */
    async function dispatchLoadChildren(node: TreeNode): Promise<TreeNodeData[]> {
        return (await options.loadChildren(node)) ?? [];
    }

    function hydrate(node: TreeNode): Promise<void> {
        if (node.loaded) {
            return Promise.resolve();
        }
        const inflight = loadingKeys.get(node.key);
        if (inflight) {
            // 期间被折叠释放/刷新（loadSeq 递增）则旧请求已作废，重新发起；否则复用去重
            if (inflight.seq === node.loadSeq) {
                return inflight.promise;
            }
            loadingKeys.delete(node.key);
        }
        const seq = (node.loadSeq ?? 0) + 1;
        node.loadSeq = seq;
        const promise = dispatchLoadChildren(node)
            .then((children) => {
                loadingKeys.delete(node.key);
                // 迟到响应作废：请求期间节点被折叠释放/刷新，保持占位态，不覆盖不触发钩子
                if (seq !== node.loadSeq) {
                    return;
                }
                // 占位/错误行索引先摘除，再登记真实子树（增量维护，其余索引保持有效）
                unindexSubtree(node);
                node.children = children.map((c) => normalize(c, node.key));
                indexTree(node.children, node.key);
                node.loaded = true;
                bump();
                afterHydrateHooks.forEach((h) => h(node));
            })
            .catch((e) => {
                loadingKeys.delete(node.key);
                if (seq !== node.loadSeq) {
                    return;
                }
                console.error('[tree] loadChildren failed:', node.key, e);
                unindexSubtree(node);
                node.children = [errorPlaceholder(node)];
                indexTree(node.children, node.key);
                node.loaded = false;
                bump();
                afterHydrateHooks.forEach((h) => h(node));
            });
        loadingKeys.set(node.key, { seq, promise });
        return promise;
    }

    /** 设置根节点（首次加载 / 根刷新）：整树重建才全量重建索引 */
    function setRoot(nodes: TreeNodeData[]) {
        data.value = nodes.map((n) => normalize(n));
        nodeIndex.clear();
        parentIndex.clear();
        indexTree(data.value);
        // 清理展开态中已不存在的 key（避免引用旧树索引误判）
        const validKeys = new Set<string>();
        expandedKeys.value.forEach((k) => {
            if (nodeIndex.has(k)) {
                validKeys.add(k);
            }
        });
        expandedKeys.value = validKeys;
        bump();
        afterHydrateHooks.forEach((h) => h(data.value[0]));
    }

    /** 首次加载根节点（容器 onMounted 调用） */
    async function init() {
        setRoot(await options.loadRoot());
    }

    /** 程序化展开（含水合）：node-expand 事件与定位路径共用 */
    async function expandNode(key: string) {
        const node = nodeIndex.get(key);
        if (!node) {
            return;
        }
        if (!expandedKeys.value.has(key)) {
            expandedKeys.value = new Set(expandedKeys.value).add(key);
        }
        await hydrate(node);
        await nextTick();
    }

    /** 收集节点全部后代 key（折叠时需一并清除展开态，否则 TreeV2 setExpandedKeys 会经后代链把祖先重新展开） */
    function collectDescendantKeys(node: TreeNode, acc: Set<string>) {
        node.children?.forEach((c) => {
            acc.add(c.key);
            collectDescendantKeys(c, acc);
        });
    }

    /** 折叠：维护展开态（同时收起全部后代展开态），并按声明释放子节点回收内存 */
    function collapseNode(key: string) {
        const node = nodeIndex.get(key);
        const doomed = new Set<string>([key]);
        if (node) {
            collectDescendantKeys(node, doomed);
        }
        let changed = false;
        const next = new Set<string>();
        expandedKeys.value.forEach((k) => {
            if (!doomed.has(k)) {
                next.add(k);
            } else {
                changed = true;
            }
        });
        if (changed) {
            expandedKeys.value = next;
        }
        if (node?.releaseOnCollapse) {
            releaseChildren(node);
            return;
        }
        // 大子树自动释放：节点多（如千级表/集合）时折叠即回收 JS 堆，再展开重新懒加载
        const threshold = options.autoReleaseThreshold ?? DEFAULT_AUTO_RELEASE_THRESHOLD;
        if (node?.loaded && (node.children?.length ?? 0) >= threshold) {
            releaseChildren(node);
        }
    }

    /** 释放节点子树：回退到加载占位，再展开时重新懒加载；递增 loadSeq 作废 in-flight 迟到响应 */
    function releaseChildren(node: TreeNode) {
        node.loadSeq = (node.loadSeq ?? 0) + 1;
        unindexSubtree(node);
        node.loaded = false;
        node.children = [loadingPlaceholder(node)];
        indexTree(node.children, node.key);
        bump();
    }

    /** 刷新：key 缺省重载根，否则重载该节点子树（保持展开态） */
    async function refresh(key?: string) {
        if (!key) {
            const roots = await options.loadRoot();
            setRoot(roots);
            return;
        }
        const node = nodeIndex.get(key);
        if (!node) {
            return;
        }
        // 递增 loadSeq 作废 in-flight 旧数据，防迟到响应覆盖重载后的占位/新子树
        node.loadSeq = (node.loadSeq ?? 0) + 1;
        unindexSubtree(node);
        node.loaded = false;
        node.children = [loadingPlaceholder(node)];
        indexTree(node.children, node.key);
        if (expandedKeys.value.has(key)) {
            await hydrate(node);
        } else {
            bump();
        }
    }

    /** 展开祖先路径并水合，使 key 节点可见（供定位/深链展开） */
    async function ensureVisible(key: string): Promise<boolean> {
        const chain: TreeNode[] = [];
        let k: string | undefined = key;
        while (k) {
            const n = nodeIndex.get(k);
            if (!n) {
                return false;
            }
            chain.unshift(n);
            k = parentIndex.get(k);
        }
        for (const n of chain) {
            if (!n.hasChildren) {
                continue;
            }
            if (!expandedKeys.value.has(n.key) || !n.loaded) {
                await expandNode(n.key);
            }
        }
        return nodeIndex.has(key);
    }

    /** 水合完成回调（定位重试等） */
    function onAfterHydrate(cb: (node: TreeNode) => void) {
        afterHydrateHooks.push(cb);
    }

    return { data, expandedKeys, init, setRoot, hydrate, expandNode, collapseNode, refresh, ensureVisible, getNode: (key: string) => nodeIndex.get(key), onAfterHydrate };
}
