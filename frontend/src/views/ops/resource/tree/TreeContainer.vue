<template>
    <!-- 树库依赖已收口到引擎适配器（TreeEngineV2）：容器只面向引擎契约，换树库仅换一行 import -->
    <TreeEngineV2
        ref="engineRef"
        class="w-full h-full overflow-hidden"
        :data="data"
        :expanded-keys="expandedKeysArray"
        :filter-text="filterText"
        :height="height"
        :show-actions="showActions"
        @node-click="onNodeClick"
        @node-expand="onNodeExpand"
        @node-collapse="onNodeCollapse"
        @node-retry="onRetryError"
        @row-contextmenu="onNodeContextmenu"
    />

    <Contextmenu ref="contextmenuRef" :dropdown="dropdown" :items="contextmenuItems" />
</template>

<script lang="ts" setup>
import { computed, inject, nextTick, onBeforeUnmount, onMounted, provide, ref, useTemplateRef } from 'vue';

import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';

import { findTriggerCommand, resolveNodeMenu } from './commands';
import { TreeApiKey } from './context';
import { DEFAULT_TREE_SCOPE, treeEvents } from './events';
import { getContributor } from './registry';
import { ERROR_KIND, LOADING_KIND, type TreeNode, type TreeNodeData, type TreeApi, type TreeEngineExpose } from './types';
import TreeEngineV2 from './TreeEngineV2.vue';
import { useLazyTree } from './useLazyTree';

/**
 * 通用资源树容器（核心对具体资产/树库双重零知识）：
 * - 引擎适配器（TreeEngineV2）隔离树库，容器只面向引擎契约，换树库仅换适配器；
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
        /** 事件作用域：匹配 treeEvents 事件的 target 才响应（多容器并存隔离） */
        eventScope?: string;
    }>(),
    { height: 0, showActions: true, interactive: true, filterText: '', eventScope: DEFAULT_TREE_SCOPE }
);

const emit = defineEmits<{ 'node-click': [node: TreeNode] }>();

const engineRef = useTemplateRef<TreeEngineExpose>('engineRef');
const contextmenuRef = useTemplateRef<InstanceType<typeof Contextmenu>>('contextmenuRef');

const { data, expandedKeys, init, expandNode, collapseNode, refresh, ensureVisible, getNode, onAfterHydrate } = useLazyTree({
    loadChildren: async (node) => decorate((await getContributor(node.kind)?.loadChildren?.(node)) ?? []),
    loadRoot: async () => decorate(await props.loadRoot()),
});

/** 引用/选择场景的节点装饰器（transformNode 逐节点转换） */
const decorate = (nodes: TreeNodeData[]) => (props.transformNode ? nodes.map((n) => props.transformNode!(n)) : nodes);

const expandedKeysArray = computed(() => Array.from(expandedKeys.value));

// 根加载
onMounted(() => {
    init().catch((e) => console.error('[tree] loadRoot failed:', e));
});

// ---------------------------------- TreeApi（provide 给命令/渲染器） ----------------------------------

/** 待定位节点：目标节点可能尚未水合出来，每次水合完成后重试 */
const pendingLocateKey = ref('');

async function tryLocate() {
    const key = pendingLocateKey.value;
    if (!key) {
        return;
    }
    if (!getNode(key)) {
        return;
    }
    pendingLocateKey.value = '';
    await ensureVisible(key);
    await nextTick();
    engineRef.value?.setCurrentKey(key);
    engineRef.value?.scrollToNode(key, 'center');
}

onAfterHydrate(() => {
    tryLocate();
});

const treeApi: TreeApi = {
    locate: async (key: string) => {
        pendingLocateKey.value = key;
        await tryLocate();
    },
    refresh: (key?: string) => {
        refresh(key).catch((e) => console.error('[tree] refresh failed:', key, e));
    },
    getNode,
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

const onNodeClick = (node: TreeNode) => {
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

const onNodeExpand = (node: TreeNode) => {
    if (node.kind === LOADING_KIND || node.kind === ERROR_KIND) {
        return;
    }
    expandNode(node.key);
};

const onNodeCollapse = (node: TreeNode) => {
    if (node.kind === LOADING_KIND || node.kind === ERROR_KIND) {
        return;
    }
    collapseNode(node.key);
};

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
    refresh: treeApi.refresh,
    getNode,
});
</script>
