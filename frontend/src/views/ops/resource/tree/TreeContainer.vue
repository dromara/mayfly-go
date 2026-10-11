<template>
    <!-- 树库依赖已收口到通用 VirtualTree（引擎差异全在适配器）：容器只面向归一化契约，换树库仅换 defaultAdapter -->
    <VirtualTree
        ref="engineRef"
        class="w-full h-full overflow-hidden"
        :data="treeData"
        :expanded-keys="expandedKeysArray"
        :filter-text="filterText"
        :filter-method="filterMethod"
        :height="height"
        @filter-applied="onFilterApplied"
        @node-click="onNodeClick"
        @node-expand="onNodeExpand"
        @node-collapse="onNodeCollapse"
    >
        <!-- 行渲染/重试/右键在行插槽内直连业务组件（不再经引擎转发层） -->
        <template #default="{ data: node }">
            <TreeRowContent :node="node" :show-actions="showActions" @retry="onRetryError" @row-contextmenu="onNodeContextmenu" />
        </template>
    </VirtualTree>

    <Contextmenu ref="contextmenuRef" :dropdown="dropdown" :items="contextmenuItems" />
</template>

<script lang="ts" setup>
import { computed, inject, nextTick, onBeforeUnmount, onMounted, provide, ref, useTemplateRef, watch } from 'vue';

import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import { VirtualTree, type VirtualTreeHandle } from '@/components/virtual-tree';

import { findTriggerCommand, resolveNodeMenu } from './commands';
import { TreeApiKey, TreeFilterQueryKey } from './context';
import { DEFAULT_TREE_SCOPE, treeEvents } from './events';
import { collectMatchKeys, collectVisibleRowKeys, nodeMatchesQuery, type NodeVisibilityFilter } from './filter';
import { getContributor } from './registry';
import { type LocateOutcome, type LocateResolver, type TreeNode, type TreeNodeData, type TreeApi } from './types';
import TreeRowContent from './TreeRowContent.vue';
import { useLazyTree } from './useLazyTree';

/**
 * 通用资源树容器（核心对具体资产/树库双重零知识）：
 * - 通用 VirtualTree 隔离树库，容器只面向归一化契约，换树库仅换适配器；
 * - useLazyTree 懒加载水合层；节点渲染/单击/双击/右键/悬浮操作由贡献者与命令注册表声明；
 * - 失效/定位通过 treeEvents 事件总线，操作方无需持有树实例。
 */

const props = withDefaults(
    defineProps<{
        /** 根节点加载器（根刷新时调用） */
        loadRoot: () => Promise<TreeNodeData[]>;
        /** 虚拟滚动视口高度（像素），缺省自适应容器实测高度 */
        height?: number;
        /** 是否展示节点悬浮操作按钮 */
        showActions?: boolean;
        /** 交互模式：false 为引用/选择场景（不派发单击/双击命令与右键菜单，仅外抛 node-click） */
        interactive?: boolean;
        /** 节点装饰器：加载结果逐节点转换（如引用面板将库节点叶子化） */
        transformNode?: (node: TreeNodeData) => TreeNodeData;
        /** 外部过滤文本（防抖后生效） */
        filterText?: string;
        /**
         * 视图级附加可见性判定（如「按资源类型筛选」），与文本命中 AND 组合。
         *
         * 由宿主页面注入，容器不认识任何资产语义；传新函数即触发重过滤。
         */
        nodeFilter?: NodeVisibilityFilter;
        /** 事件作用域：匹配 treeEvents 事件的 target 才响应（多容器并存隔离） */
        eventScope?: string;
        /**
         * 定位标识解析器（由资产域注入，如 codePath → 节点 key）：
         * 容器只负责展开/高亮，不认识任何具体定位协议，新增协议或资产类型均无需改动容器
         */
        resolveLocate?: LocateResolver;
    }>(),
    { height: 0, showActions: true, interactive: true, filterText: '', eventScope: DEFAULT_TREE_SCOPE }
);

const emit = defineEmits<{ 'node-click': [node: TreeNode] }>();

const engineRef = useTemplateRef<VirtualTreeHandle>('engineRef');
const contextmenuRef = useTemplateRef<InstanceType<typeof Contextmenu>>('contextmenuRef');

const { data, expandedKeys, init, expandNode, collapseNode, refresh, appendChildren, ensureVisible, getNode, onAfterHydrate } = useLazyTree({
    loadChildren: async (node) => decorate((await getContributor(node.kind)?.loadChildren?.(node)) ?? []),
    loadRoot: async () => decorate(await props.loadRoot()),
});

/** 引用/选择场景的节点装饰器（transformNode 逐节点转换） */
const decorate = (nodes: TreeNodeData[]) => (props.transformNode ? nodes.map((n) => props.transformNode!(n)) : nodes);

const expandedKeysArray = computed(() => Array.from(expandedKeys.value));

/** 水合层 TreeNode（无索引签名的 interface）→ VirtualTree 归一化数据契约的桥接 */
const treeData = computed(() => data.value as unknown as Record<string, unknown>[]);

/**
 * 引擎过滤谓词：文本命中（tree/filter 纯函数）AND 宿主注入的视图级判定。
 * 与下面 matchedKeys / visibleRowKeys 用的是同一个组合，三处不会分叉。
 */
const filterMethod = (query: string, node: Parameters<NodeVisibilityFilter>[0]) =>
    nodeMatchesQuery(query, node) && (props.nodeFilter?.(node) ?? true);

/**
 * 引擎真正生效的查询词（由 VirtualTree 防抖后上报）：高亮与计数都只认它，不认 props.filterText。
 *
 * 防抖窗口内两者不一致：若拿「意图值」做高亮/计数，会出现已标色、已计数但该行还没被过滤掉；
 * 此时按 Enter 跳转，目标可能还在引擎旧隐藏集里（scrollToNode 对未 flatten 的节点失效）→ 静默跳空。
 */
const appliedQuery = ref('');

const onFilterApplied = (query: string) => {
    appliedQuery.value = query;
};


/** 生效查找文本透给行组件做命中片段高亮：与引擎过滤同一时相，「被标色的字」就是「让这行留下的字」 */
provide(TreeFilterQueryKey, computed(() => appliedQuery.value));

/**
 * 命中 key（文档序）：供查找条展示 n/N 计数并逐个跳转。
 *
 * 口径是「业务展开链上可见的命中」：只有这些节点真的在引擎 flattenTree 里，
 * 计数与 Enter 可跳次数才能严格相等（收起分支的命中跳到时 indexOf 为 -1，会静默跳空并复位滚动）。
 */
const matchedKeys = computed(() => collectMatchKeys(appliedQuery.value, data.value, expandedKeys.value, props.nodeFilter));

/**
 * 引擎在该查询词下实际渲染的行序（与 flattenTree 等价，规则见 filter.collectVisibleRowKeys）。
 * 只在定位时读（computed 惰性求值）：跳转要把目标滚到视口中间，而偏移 = 行序 × 行高。
 */
const visibleRowKeys = computed(() => collectVisibleRowKeys(appliedQuery.value, data.value, expandedKeys.value, props.nodeFilter));

/**
 * 过滤态下的两次「重跑引擎过滤」：el-tree-v2 只在 filter() 被调用时对全树求值，
 * 下面两种变化它都看不见，不重跑就会让新数据/新谓词绕过过滤：
 * - 视图级判定换了（如选了另一个资源类型）；
 * - data 变了（懒加载水合出新子树）——新节点不在旧隐藏集里，会直接漏出来。
 * 无过滤时不重跑（空词会触发引擎的全展开副作用，由 applyFilter 内的复位扛，但没必要白跑）。
 */
watch(
    () => props.nodeFilter,
    () => engineRef.value?.refilter(),
    // 取消筛选（谓词从有到 undefined）也必须重跑：引擎的隐藏集只在 doFilter 开头被 clear，
    // 不再次调用 filter 就会永久留着上一次的隐藏结果。post 保证读到的引擎状态是最新的
    { flush: 'post' }
);
// 必须 post-flush：pre 时本容器的 job id 小于引擎组件的，会抢在 useTree 的 setData 之前跑，
// 拿旧 treeNodes 求值等于白跑一次（新节点不在旧隐藏集里，照样漏出来）
watch(data, () => {
    if (appliedQuery.value || props.nodeFilter) {
        engineRef.value?.refilter();
    }
}, { flush: 'post' });

// 根加载
onMounted(() => {
    init()
        .then(() => {
            rootLoaded = true;
            // 根加载完成后重试定位：ResourceOp 可能已在 onMounted 中发起 identity 定位，此时层级才就绪
            tryLocate();
        })
        .catch((e) => console.error('[tree] loadRoot failed:', e));
});

// ---------------------------------- TreeApi（provide 给命令/渲染器） ----------------------------------

/**
 * 待定位目标，两种语义：
 * - key：真实树节点 key（tab 激活/节点回定位），容器直接定位；
 * - identity：外部资源标识（如资产域 codePath），交由注入的 resolveLocate 解析为节点 key。
 * 目标层级尚未水合时挂起，根加载完成与水合后自动重试
 */
let pendingLocate: { mode: 'key' | 'identity'; value: string } | null = null;

/** 定位进行中标志：解析会展开层级触发 hydrate → onAfterHydrate → tryLocate，需防重入 */
let locating = false;

/** 根节点是否已加载完成：据此区分「层级未就绪可重试」与「目标确实不存在」，避免永久挂起 */
let rootLoaded = false;

/** 解析待定位目标为节点 key（容器不认识具体标识协议，identity 模式委托注入的解析器） */
async function resolvePendingLocate(pending: { mode: 'key' | 'identity'; value: string }): Promise<LocateOutcome> {
    if (pending.mode === 'key') {
        if (getNode(pending.value)) {
            return { status: 'resolved', key: pending.value };
        }
        return rootLoaded ? { status: 'missing' } : { status: 'pending' };
    }
    return (await props.resolveLocate?.(pending.value, { getNode, expandNode, rootLoaded })) ?? { status: 'missing' };
}

async function tryLocate() {
    if (locating) {
        return;
    }
    const pending = pendingLocate;
    if (!pending) {
        return;
    }
    locating = true;
    try {
        const outcome = await resolvePendingLocate(pending);
        if (outcome.status === 'pending') {
            return;
        }
        // resolved 执行定位；missing 放弃并清除挂起（防止此后水合时误高亮）
        pendingLocate = null;
        if (outcome.status === 'missing') {
            return;
        }
        await ensureVisible(outcome.key);
        await nextTick();
        engineRef.value?.setCurrentKey(outcome.key);
        // 滚动偏移按自算行序给引擎（不用引擎的 scrollToNode：它在过滤态下对合法下标返回错位偏移，实测把列表弹回顶部）
        const rowIndex = visibleRowKeys.value.indexOf(outcome.key);
        if (rowIndex >= 0) {
            engineRef.value?.scrollToIndex(rowIndex);
        } else if (import.meta.env.DEV) {
            console.warn(`[tree] 定位目标不在自算可见行序内，已跳过滚动：${outcome.key}（collectVisibleRowKeys 与引擎 flatten 分叉，需核规则）`);
        }
    } finally {
        locating = false;
    }
}

onAfterHydrate(() => {
    tryLocate();
});

const treeApi: TreeApi = {
    locate: async (key: string) => {
        pendingLocate = { mode: 'key', value: key };
        await tryLocate();
    },
    refresh: (key?: string) => {
        refresh(key).catch((e) => console.error('[tree] refresh failed:', key, e));
    },
    expandNode,
    getNode,
    // appendChildren 须与 loadChildren/loadRoot 同经 decorate：引用面板(transformNode)下续载节点不跳过装饰
    appendChildren: (key, nodes, options) => appendChildren(key, decorate(nodes), options),
};

/** 按外部资源标识定位（快捷跳转/最近操作）：交由注入的 resolveLocate 解析为节点 key 后展开高亮 */
const locateIdentity = async (identity: string) => {
    if (!identity) {
        return;
    }
    pendingLocate = { mode: 'identity', value: identity };
    await tryLocate();
};

provide(TreeApiKey, treeApi);

// 失效事件：操作完成后只发事件，容器统一重载（target 不匹配的事件忽略，如 ai 面板与主树隔离）
const unbindInvalidate = treeEvents.on('node:invalidate', ({ key, target }) => {
    if ((target ?? DEFAULT_TREE_SCOPE) !== props.eventScope) {
        return;
    }
    refresh(key).catch((e) => console.error('[tree] invalidate refresh failed:', key, e));
});
const unbindLocate = treeEvents.on('node:locate', ({ key, target }) => {
    if ((target ?? DEFAULT_TREE_SCOPE) !== props.eventScope) {
        return;
    }
    treeApi.locate(key);
});
onBeforeUnmount(() => {
    unbindInvalidate();
    unbindLocate();
});

// ---------------------------------- 节点交互 ----------------------------------

let lastClick: { key: string; time: number } | null = null;

const onNodeClick = (raw: Record<string, unknown>) => {
    const node = raw as unknown as TreeNode;
    // 关闭可能存在的右键菜单
    contextmenuRef.value?.closeContextmenu();
    emit('node-click', node);

    // 引用/选择场景：不派发命令；单击可展开节点仅切换展开态（展开箭头之外的行区域也能展开）
    if (!props.interactive) {
        if (!node.disabled && node.hasChildren) {
            toggleNode(node.key);
        }
        return;
    }
    if (node.disabled) {
        return;
    }

    const now = Date.now();
    // 双击检测（TreeV2 无 dblclick 事件）：300ms 内二次点击同一节点
    if (lastClick && lastClick.key === node.key && now - lastClick.time < 300) {
        lastClick = null;
        onNodeDblclick(node);
        return;
    }
    lastClick = { key: node.key, time: now };

    const cmd = findTriggerCommand(node, treeApi, 'click');
    if (cmd) {
        cmd.handler({ node, tree: treeApi });
        // 挂了单击命令的实例行（打开操作面板）也要能就地展开：
        // 没有该命令的资源（DB/Redis 实例）点行本来就能展开，两类资源行为不一致会让人以为「点了没反应」。
        // 这里只展开不折叠——收起仍归箭头与双击管，避免连击把刚打开的子树又收掉。
        if (node.hasChildren && !expandedKeys.value.has(node.key)) {
            expandNode(node.key);
        }
        return;
    }
    // 无单击命令的可展开节点：单击直接切换展开（展开态单源自管，比双击更顺滑）
    if (node.hasChildren) {
        toggleNode(node.key);
    }
};

/** 展开态切换（单源自管：经 default-expanded-keys 回传同步 TreeV2 内部状态） */
const toggleNode = (key: string) => {
    if (expandedKeys.value.has(key)) {
        collapseNode(key);
    } else {
        expandNode(key);
    }
};

const onNodeDblclick = (node: TreeNode) => {
    if (node.hasChildren) {
        toggleNode(node.key);
    }
    const cmd = findTriggerCommand(node, treeApi, 'dblclick');
    if (cmd) {
        cmd.handler({ node, tree: treeApi });
    }
};

// 占位/错误行的展开/折叠过滤已下沉 useLazyTree 入口（数据层语义自治）：容器只透传 key
const onNodeExpand = (raw: Record<string, unknown>) => expandNode(String(raw.key));

const onNodeCollapse = (raw: Record<string, unknown>) => collapseNode(String(raw.key));

const onRetryError = (node: TreeNode) => {
    const target = (node.params?.retryTarget as string) ?? '';
    if (target) {
        refresh(target).catch((e) => console.error('[tree] retry failed:', target, e));
    }
};

// ---------------------------------- 右键菜单（与悬浮按钮同源） ----------------------------------

const dropdown = ref({ x: 0, y: 0 });
const contextmenuItems = ref<ContextmenuItem[]>([]);

const onNodeContextmenu = (event: MouseEvent, node: TreeNode) => {
    if (!props.interactive || node.disabled) {
        return;
    }
    const items = resolveNodeMenu(node.kind);
    if (!items.length) {
        return;
    }
    contextmenuItems.value = items;
    dropdown.value = { x: event.clientX, y: event.clientY };
    // payload 统一为 TreeCommandCtx：when/onClickFunc 入参形状一致
    contextmenuRef.value?.openContextmenu({ node, tree: treeApi });
};

// ---------------------------------- 对外暴露 ----------------------------------

defineExpose({
    locate: treeApi.locate,
    locateIdentity,
    refresh: treeApi.refresh,
    getNode,
    matchedKeys,
    /** 引擎已生效的查询词：宿主据此判断派生值是否跟上了输入（防抖期内别拿旧计数示人） */
    appliedQuery,
});
</script>
