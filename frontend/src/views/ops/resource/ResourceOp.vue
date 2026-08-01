<template>
    <div class="h-full" :class="{ 'resource-op-fullscreen': isFullscreen }">
        <el-splitter @resize="onResizeOpPanel">
            <el-splitter-panel size="24%" max="40%">
                <el-card class="h-full flex" body-class="!p-0 flex flex-col w-full">
                    <div class="tag-tree-header flex justify-between items-center">
                        <el-input v-model="filterText" :placeholder="$t('tag.tagFilterPlaceholder')" clearable size="small" class="tag-tree-search w-full">
                            <template #prefix>
                                <SvgIcon class="tag-tree-search-icon" name="search" />
                            </template>
                        </el-input>
                    </div>

                    <el-scrollbar>
                        <el-tree
                            class="min-w-full inline-block"
                            ref="treeRef"
                            :highlight-current="true"
                            :indent="10"
                            :load="loadNode"
                            :props="treeProps"
                            lazy
                            node-key="key"
                            :expand-on-click-node="false"
                            :filter-node-method="filterNode"
                            @node-click="treeNodeClick"
                            @node-expand="treeNodeClick"
                            @node-collapse="onNodeCollapse"
                            @node-contextmenu="onNodeContextmenu"
                            :default-expanded-keys="state.defaultExpandedKeys"
                        >
                            <template #default="{ node, data }">
                                <div class="select-none" v-if="data.type == TagResourceTypeEnum.Tag.value">
                                    <span v-for="(value, i) in data.label.split('/')" :key="i">
                                        <el-text class="mr-[1.5px]! ml-[1.5px]!" v-if="i != 0" tag="b" type="primary" size="large">/</el-text>
                                        <el-text>{{ value }}</el-text>
                                    </span>
                                </div>

                                <component v-else-if="data.nodeComponent" :is="data.nodeComponent" :node="node" :data="data" />
                                <BaseTreeNode v-else :node="node" :data="data" />
                            </template>
                        </el-tree>
                    </el-scrollbar>
                </el-card>
            </el-splitter-panel>

            <el-splitter-panel>
                <el-card class="h-full" body-class="h-full !p-0 flex flex-col flex-1">
                    <!-- 标签栏：当存在带 tabKey 的组件时显示 -->
                    <div
                        v-if="resourceTabs.length > 0"
                        class="flex items-center gap-1 px-1.5 py-1 border-b border-(--el-border-color-light) shrink-0 overflow-x-auto min-h-9 [&::-webkit-scrollbar]:h-0.75"
                    >
                        <div
                            v-for="tab in resourceTabs"
                            :key="tab.key"
                            class="group flex items-center gap-1 px-2.5 py-1 rounded cursor-pointer text-sm whitespace-nowrap shrink-0 transition-all duration-200 ease-in-out"
                            :class="[
                                activeResourceOpTabKey === tab.key
                                    ? 'text-(--el-color-primary) bg-(--el-color-primary-light-9) border border-(--el-color-primary-light-5)'
                                    : 'text-(--el-text-color-regular) bg-(--el-fill-color-blank) border border-(--el-border-color-lighter) hover:bg-(--el-fill-color) hover:text-(--el-text-color-primary) hover:border-(--el-border-color)',
                            ]"
                            @click="activateTab(tab.key)"
                            @contextmenu.prevent="onTabContextmenu($event, tab)"
                        >
                            <!-- 自定义 tab 组件 -->
                            <component v-if="tab.tabComponent" :is="tab.tabComponent" v-bind="{ ...tab.tabComponentProps, tabName: $t(tab.name) }" />
                            <!-- 默认 tab 显示：icon + name -->
                            <template v-else>
                                <SvgIcon
                                    v-if="tab.tabComponentProps?.icon"
                                    :name="tab.tabComponentProps.icon.name"
                                    :color="tab.tabComponentProps.icon.color"
                                    class="text-sm shrink-0"
                                />
                                <span class="max-w-40 overflow-hidden text-ellipsis" :title="$t(tab.name)">{{ $t(tab.name) }}</span>
                            </template>
                            <!-- 激活的tab：始终显示所有按钮 -->
                            <span v-if="activeResourceOpTabKey === tab.key" class="inline-flex items-center gap-0.5 ml-1 shrink-0">
                                <span
                                    class="w-4 h-4 flex items-center justify-center rounded shrink-0 cursor-pointer transition-all duration-200 ease-in-out hover:bg-(--el-color-info-light-7)"
                                    @click.stop="refreshTab(tab.key)"
                                >
                                    <SvgIcon name="RefreshRight" class="text-(--el-text-color-secondary) hover:text-(--el-text-color-primary)" />
                                </span>
                                <span
                                    class="w-4 h-4 flex items-center justify-center rounded shrink-0 cursor-pointer transition-all duration-200 ease-in-out hover:bg-(--el-color-info-light-7)"
                                    @click.stop="closeTab(tab.key)"
                                >
                                    <SvgIcon name="Close" class="text-(--el-text-color-secondary) hover:text-(--el-text-color-primary) text-[12px]!" />
                                </span>
                                <span
                                    class="w-4 h-4 flex items-center justify-center rounded shrink-0 cursor-pointer transition-all duration-200 ease-in-out hover:bg-(--el-color-info-light-7)"
                                    @click.stop="toggleFullscreen"
                                >
                                    <SvgIcon
                                        v-if="!isFullscreen"
                                        name="FullScreen"
                                        class="text-(--el-text-color-secondary) hover:text-(--el-text-color-primary)"
                                    />
                                    <SvgIcon v-else name="crop" class="text-(--el-text-color-secondary) hover:text-(--el-text-color-primary)" />
                                </span>
                            </span>
                            <!-- 非激活的tab：悬浮时只显示关闭按钮 -->
                            <span
                                v-else
                                class="inline-flex items-center gap-0.5 h-4.5 max-w-0 overflow-hidden opacity-0 shrink-0 transition-all duration-300 ease-in-out group-hover:max-w-5 group-hover:ml-1 group-hover:opacity-100"
                            >
                                <span
                                    class="w-4 h-4 flex items-center justify-center rounded shrink-0 cursor-pointer transition-all duration-200 ease-in-out hover:bg-(--el-color-info-light-7)"
                                    @click.stop="closeTab(tab.key)"
                                >
                                    <SvgIcon name="Close" class="text-(--el-text-color-secondary) hover:text-(--el-text-color-primary) text-[12px]!" />
                                </span>
                            </span>
                        </div>
                    </div>
                    <div class="resource-tab-content">
                        <keep-alive :max="20">
                            <component
                                ref="activeCompRef"
                                :is="activeResourceTab?.component"
                                :key="activeResourceTab?.componentKey"
                                v-bind="activeResourceTab?.componentProps"
                            />
                        </keep-alive>
                    </div>
                </el-card>
            </el-splitter-panel>
        </el-splitter>

        <Contextmenu :dropdown="state.dropdown" :items="state.contextmenuItems" ref="contextmenuRef" />
        <Contextmenu :dropdown="tabDropdown" :items="tabContextmenuItems" ref="tabContextmenuRef" />

        <!-- 渲染注册的非 tab 组件（Overlay） -->
        <template v-for="overlay in overlayList" :key="overlay.key">
            <component v-if="overlay.visible" :is="overlay.component" v-bind="overlay.props" @update:visible="(val: boolean) => (overlay.visible = val)" />
        </template>
    </div>
</template>

<script lang="ts" setup>
import { computed, nextTick, onMounted, onUnmounted, provide, reactive, ref, toRefs, useTemplateRef, watch, type ComponentPublicInstance } from 'vue';

import { TagResourceTypeEnum } from '@/common/commonEnum';
import { isPrefixSubsequence } from '@/common/utils/string';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import SvgIcon from '@/components/svg-icon/index.vue';
import { useAutoOpenResource } from '@/store/autoOpenResource';
import { ResourceOpCtx } from '@/views/ops/resource/resourceOp';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import BaseTreeNode from './BaseTreeNode.vue';
import { TagTreeNode } from '@/views/ops/component/tag';
import { getResourceTypes, loadResourceTags } from './resource';
import {
    activateResourceOpTab,
    activeResourceOpTabKey,
    allResourceOpOverlays,
    allResourceOpTabs,
    getComponentInstance,
    getResourceOpTab,
    registerComponentInstance,
    removeResourceOpTab,
    ResourceOpCtxKey,
    ResourceOpTab,
} from './resourceOp';

const props = defineProps({
    load: {
        type: Function,
        required: false,
    },
    loadContextmenuItems: {
        type: Function,
        required: false,
    },
});

const treeProps = {
    label: 'name',
    children: 'zones',
    isLeaf: 'isLeaf',
};

const autoOpenResourceStore = useAutoOpenResource();
const { autoOpenResource } = storeToRefs(autoOpenResourceStore);

const { t } = useI18n();

const emit = defineEmits(['nodeClick', 'currentContextmenuClick']);

const treeRef = useTemplateRef<{
    filter: (val: string) => void;
    remove: (data: unknown) => void;
    getNode: (key: string) => { loaded: boolean; expand: () => void } | undefined;
    setCurrentKey: (key: string) => void;
}>('treeRef');
const contextmenuRef = useTemplateRef<InstanceType<typeof Contextmenu>>('contextmenuRef');
const tabContextmenuRef = useTemplateRef<InstanceType<typeof Contextmenu>>('tabContextmenuRef');

// 存储当前组件对应的最后操作的节点key，用户切换资源操作组件时，定位到相应的树节点
const resourceComponentsNodeKey = ref<Record<string, string>>({});

// 当前激活的 tab（从 allResourceOpTabs 获取）
const activeResourceTab = computed(() => {
    return getResourceOpTab(activeResourceOpTabKey.value);
});

const resourceTabs = computed(() => {
    return Array.from(allResourceOpTabs.values());
});

const overlayList = computed(() => {
    return Array.from(allResourceOpOverlays.values());
});

// Tab 右键菜单
const tabDropdown = reactive({ x: 0, y: 0 });
const tabContextmenuItems = ref<ContextmenuItem[]>([]);

const cmTabCloseAll = new ContextmenuItem('closeAll', 'layout.tagsView.closeAll').withIcon('Close').withOnClick(() => closeAllTabs());

const cmTabCloseLeft = new ContextmenuItem('closeLeft', 'layout.tagsView.closeLeft').withIcon('Back').withOnClick((data: unknown) => closeLeftTabs((data as Record<string, unknown>).tabKey as string));

const cmTabCloseRight = new ContextmenuItem('closeRight', 'layout.tagsView.closeRight')
    .withIcon('Right')
    .withOnClick((data: unknown) => closeRightTabs((data as Record<string, unknown>).tabKey as string));

const cmTabCloseOther = new ContextmenuItem('closeOther', 'layout.tagsView.closeOther')
    .withIcon('Switch')
    .withOnClick((data: unknown) => closeOtherTabs((data as Record<string, unknown>).tabKey as string));

tabContextmenuItems.value = [cmTabCloseLeft, cmTabCloseRight, cmTabCloseOther, cmTabCloseAll];

// 右侧面板全屏相关
const isFullscreen = ref(false);

const toggleFullscreen = () => {
    isFullscreen.value = !isFullscreen.value;
};

const onFullscreenKeydown = (e: KeyboardEvent) => {
    if (e.key === 'Escape' && isFullscreen.value) {
        isFullscreen.value = false;
    }
};

onMounted(() => {
    document.addEventListener('keydown', onFullscreenKeydown);
});

onUnmounted(() => {
    document.removeEventListener('keydown', onFullscreenKeydown);
    if (filterTimer) clearTimeout(filterTimer);
});

const activeCompRef = useTemplateRef<ComponentPublicInstance>('activeCompRef');

// 注册当前活跃组件实例到对应 tab
const registerActiveComp = (tabKey: string) => {
    const el = activeCompRef.value;
    if (!tabKey || !el) return false;

    registerComponentInstance(tabKey, el);
    return true;
};

// 监听 tab 切换，主动获取当前活跃组件实例并注册
// 解决 keep-alive 场景下 :ref 回调不可靠的问题（缓存组件激活时 ref 回调不重新触发）
watch(activeResourceOpTabKey, (tabKey: string) => {
    if (!tabKey) return;
    let attempts = 0;
    const maxAttempts = 50; // 最多重试50次，防止无限轮询
    // 异步组件可能需要多轮 nextTick 才能拿到实例
    const tryRegister = () => {
        nextTick(() => {
            if (registerActiveComp(tabKey) || ++attempts >= maxAttempts) return;
            setTimeout(tryRegister, 50);
        });
    };
    tryRegister();
});

const state = reactive({
    defaultExpandedKeys: [] as string[],
    filterText: '',
    contextmenuItems: [] as ContextmenuItem[],
    dropdown: {
        x: 0,
        y: 0,
    },
});

const { filterText } = toRefs(state);

let filterTimer: ReturnType<typeof setTimeout> | null = null;
watch(filterText, (val: string) => {
    if (filterTimer) clearTimeout(filterTimer);
    filterTimer = setTimeout(() => {
        treeRef.value?.filter(val);
    }, 300);
});

watch(
    () => autoOpenResource.value.codePath,
    (autoOpenCodePath: string) => {
        if (!autoOpenCodePath) {
            return;
        }

        const expandedKeys: string[] = [];
        let currentTagPath = '';
        const parts = autoOpenCodePath.split('/'); // 切分字符串并保留数字和对应的值部分
        let addResouceType = false;
        for (let part of parts) {
            if (!part) {
                continue;
            }
            let [key, value] = part.split('|'); // 分割数字和值部分
            // 如果不存在第二个参数，则说明为标签类型
            if (!value) {
                const tagPath = key + '/';
                currentTagPath = currentTagPath + tagPath;
                expandedKeys.push(currentTagPath);
                continue;
            }
            if (!addResouceType) {
                expandedKeys.push(currentTagPath + '-' + key);
                expandedKeys.push(value);
                addResouceType = true;
            } else {
                expandedKeys.push(value);
            }
        }

        state.defaultExpandedKeys = expandedKeys;
        autoOpenResourceStore.setCodePath('');
        setTimeout(() => {
            setCurrentKey(expandedKeys[expandedKeys.length - 1]);
        }, 500);
    },
    { immediate: true }
);

const filterNode = (value: string, data: Record<string, unknown>) => {
    return !value || isPrefixSubsequence(value, data.label as string);
};

/**
 * 加载树节点
 * @param { Object } node
 * @param { Object } resolve
 */
const loadNode = async (node: Record<string, unknown>, resolve: (data: unknown[]) => void, reject: () => void) => {
    if (typeof resolve !== 'function') {
        return;
    }
    let nodes;
    try {
        if (node.level == 0) {
            nodes = await loadResourceTags(getResourceTypes(), ctx);
        } else if (props.load) {
            nodes = await props.load(node);
        } else {
            nodes = await (node.data as TagTreeNode).loadChildren();
        }
    } catch (e: unknown) {
        console.error(e);
        // 调用 reject 以保持节点状态，并允许远程加载继续。
        return reject();
    }
    return resolve(nodes);
};

let lastNodeClickTime = 0;

const treeNodeClick = async (data: Record<string, unknown>, node: Record<string, unknown>) => {
    // 关闭可能存在的右击菜单
    contextmenuRef.value?.closeContextmenu();

    const currentClickNodeTime = Date.now();
    // 双击节点
    if (currentClickNodeTime - lastNodeClickTime < 300) {
        await treeNodeDblclick(data, node);
    } else {
        lastNodeClickTime = currentClickNodeTime;
        const dataType = data.type as Record<string, unknown> | undefined;
        if (!data.disabled && !dataType?.nodeDblclickFunc && dataType?.nodeClickFunc) {
            emit('nodeClick', data);
            await (dataType.nodeClickFunc as Function)(data);
        }
    }

    setTimeout(() => {
        const activateKey = activeResourceOpTabKey.value;
        const nodeDataKey = data.key as string;
        if (activateKey && (nodeDataKey?.startsWith(activateKey) || nodeDataKey == activateKey || activateKey?.startsWith(nodeDataKey))) {
            resourceComponentsNodeKey.value[activeResourceOpTabKey.value] = data.key as string;
        }
    }, 50);
};

// 树节点双击事件
const treeNodeDblclick = async (data: Record<string, unknown>, node: Record<string, unknown>) => {
    if (node.expanded) {
        (node.collapse as Function)();
    } else {
        (node.expand as Function)();
    }

    const dataType = data.type as Record<string, unknown> | undefined;
    if (!data.disabled && dataType?.nodeDblclickFunc) {
        await (dataType.nodeDblclickFunc as Function)(data);
    }
};

// 递归查找子树中标记了 collapseRemoveChildren 的节点，删除其子节点
const removeCollapseChildrenNodes = (node: Record<string, unknown>) => {
    const nodeData = node.data as Record<string, unknown> | undefined;
    const nodeType = nodeData?.type as Record<string, unknown> | undefined;
    if (nodeType?.collapseRemoveChildren) {
        // 找到标记节点，删除其所有子节点，不再往下递归
        const childNodes = [...(node.childNodes as Record<string, unknown>[])];
        childNodes.forEach((child: Record<string, unknown>) => {
            treeRef.value?.remove((child.data as Record<string, unknown>) || child);
        });
        node.loaded = false;
        node.loading = false;
        node.isLeaf = false;
        node.expanded = false;
        return;
    }
    // 当前节点无标记，继续往子节点查找
    for (const child of [...(node.childNodes as Record<string, unknown>[])]) {
        removeCollapseChildrenNodes(child);
    }
};

// 树节点折叠事件 - 删除子树中标记了 collapseRemoveChildren 的节点的子节点，释放内存
const onNodeCollapse = (data: unknown, node: Record<string, unknown>) => {
    removeCollapseChildrenNodes(node);
};

// 树节点右击事件
const onNodeContextmenu = (event: MouseEvent, data: Record<string, unknown>) => {
    if (data.disabled) {
        return;
    }

    // 加载当前节点是否需要显示右击菜单
    let items = (data.type as Record<string, unknown>).contextMenuItems as ContextmenuItem[] | undefined;
    if (!items || items.length == 0) {
        if (props.loadContextmenuItems) {
            items = props.loadContextmenuItems(data) as ContextmenuItem[];
        }
    }
    if (!items) {
        return;
    }
    state.contextmenuItems = items;
    const { clientX, clientY } = event;
    state.dropdown.x = clientX;
    state.dropdown.y = clientY;
    contextmenuRef.value?.openContextmenu(data);
};

// 激活指定标签页
const activateTab = (tabKey: string) => {
    activateResourceOpTab(tabKey);
    if (!tabKey) {
        return;
    }
    // 定位到左侧资源树对应节点
    if (resourceComponentsNodeKey.value[tabKey]) {
        setCurrentKey(resourceComponentsNodeKey.value[tabKey]);
    }
    nextTick(() => {
        // 调用该tab的激活回调
        getComponentInstance(tabKey)?.onActivate?.();
    });
};

// 刷新标签页（通过改变 key 强制重新渲染）
const refreshTab = (tabKey: string) => {
    // 调用该 tab 注册的刷新回调
    getComponentInstance(tabKey)?.onRefresh?.();
};

// Tab 右键菜单处理
const onTabContextmenu = (event: MouseEvent, tab: ResourceOpTab) => {
    // 关闭可能存在的树节点右键菜单
    contextmenuRef.value?.closeContextmenu();
    tabDropdown.x = event.clientX;
    tabDropdown.y = event.clientY;
    tabContextmenuRef.value?.openContextmenu({ tabKey: tab.key });
};

// 关闭标签页
const closeTab = (tabKey: string, isChangeTab: boolean = true) => {
    // 清除组件实例和缓存
    removeResourceOpTab(tabKey);
    // 清理节点映射关系
    delete resourceComponentsNodeKey.value[tabKey];
    // 调用该 tab 的关闭回调
    getComponentInstance(tabKey)?.onClose?.();

    // 如果关闭的是当前活动标签，切换到相邻标签
    if (activeResourceOpTabKey.value === tabKey) {
        const remainingTabs: string[] = Array.from(allResourceOpTabs.keys());
        if (remainingTabs.length > 0) {
            // 切换到最后一个tab
            activateTab(remainingTabs[remainingTabs.length - 1]);
        } else {
            activeResourceOpTabKey.value = '';
        }
    }
};

// 关闭所有标签
const closeAllTabs = () => {
    const allKeys: string[] = Array.from(allResourceOpTabs.keys());
    allKeys.forEach((key) => {
        closeTab(key, false);
    });
    allResourceOpTabs.clear();
    resourceComponentsNodeKey.value = {};
    activateTab('');
};

// 关闭左侧标签
const closeLeftTabs = (targetTabKey: string) => {
    const allKeys: string[] = Array.from(allResourceOpTabs.keys());
    const targetIndex = allKeys.indexOf(targetTabKey);
    if (targetIndex <= 0) return;
    const keysToClose = allKeys.slice(0, targetIndex);
    keysToClose.forEach((key: string) => {
        closeTab(key, false);
    });
    // 如果当前激活的标签被关闭，切换到目标标签
    if (keysToClose.includes(activeResourceOpTabKey.value)) {
        activateTab(targetTabKey);
    }
};

// 关闭其他标签
const closeOtherTabs = (targetTabKey: string) => {
    const allKeys: string[] = Array.from(allResourceOpTabs.keys());
    const keysToClose = allKeys.filter((key) => key !== targetTabKey);
    keysToClose.forEach((key: string) => {
        closeTab(key, false);
    });
    activateTab(targetTabKey);
};

// 关闭右侧标签
const closeRightTabs = (targetTabKey: string) => {
    const allKeys: string[] = Array.from(allResourceOpTabs.keys());
    const targetIndex = allKeys.indexOf(targetTabKey);
    if (targetIndex === -1 || targetIndex === allKeys.length - 1) return;
    const keysToClose = allKeys.slice(targetIndex + 1);
    keysToClose.forEach((key: string) => {
        closeTab(key, false);
    });
    // 如果当前激活的标签被关闭，切换到目标标签
    if (keysToClose.includes(activeResourceOpTabKey.value)) {
        activateTab(targetTabKey);
    }
};

const reloadNode = (nodeKey: string) => {
    let node = getNode(nodeKey);
    node.loaded = false;
    node.expand();
};

const getNode = (nodeKey: string) => {
    let node = treeRef.value?.getNode(nodeKey);
    if (!node) {
        throw new Error('未找到节点: ' + nodeKey);
    }
    return node;
};

const setCurrentKey = (nodeKey: string) => {
    treeRef.value?.setCurrentKey(nodeKey);
    // 延迟查询 DOM，确保节点已展开渲染，再用 rAF 滚动避免强制同步布局
    setTimeout(() => {
        requestAnimationFrame(() => {
            document.getElementById(nodeKey)?.scrollIntoView({ block: 'center' });
        });
    }, 100);
};

let resizeRAF = 0;
const onResizeOpPanel = () => {
    // 用 requestAnimationFrame 节流，Splitter 拖拽时高频触发 resize 事件
    if (resizeRAF) return;
    resizeRAF = requestAnimationFrame(() => {
        resizeRAF = 0;
        const key = activeResourceOpTabKey.value;
        if (key) {
            getComponentInstance(key)?.onResize?.();
        }
    });
};

const ctx: ResourceOpCtx = {
    setCurrentTreeKey: setCurrentKey,
    getTreeNode: getNode,
    reloadTreeNode: reloadNode,
};

provide(ResourceOpCtxKey, ctx);
</script>

<style lang="scss" scoped>
.tag-tree-header {
    padding: 4px 6px;
    border-bottom: 1px solid var(--el-border-color-light);
}

.tag-tree-search {
    :deep(.el-input__wrapper) {
        border-radius: 14px;
        height: 24px;
    }
}

.resource-tab-content {
    flex: 1;
    min-height: 0;
    overflow: hidden;
    padding: 4px;
}
</style>

<style lang="scss">
.resource-op-fullscreen {
    position: fixed !important;
    inset: 0;
    z-index: 2000;
    background: var(--el-bg-color);
}
</style>
