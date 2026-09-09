<template>
    <div class="h-full" :class="{ 'resource-op-fullscreen': isFullscreen }">
        <el-splitter @resize="onResizeOpPanel">
            <el-splitter-panel size="24%" max="40%">
                <el-card class="h-full flex" body-class="p-0! flex flex-col w-full">
                    <div class="tag-tree-header flex justify-between items-center">
                        <el-input v-model="filterText" :placeholder="$t('tag.tagFilterPlaceholder')" clearable size="small" class="tag-tree-search w-full">
                            <template #prefix>
                                <SvgIcon class="tag-tree-search-icon" name="search" />
                            </template>
                        </el-input>
                    </div>

                    <TreeContainer ref="treeContainerRef" :load-root="loadRoot" :filter-text="filterText" @node-click="onTreeNodeClick" />
                </el-card>
            </el-splitter-panel>

            <el-splitter-panel>
                <el-card class="h-full" body-class="h-full p-0! flex flex-col flex-1">
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
                        <!--
                            所有打开的 tab 常驻挂载，v-show 切换：关闭 tab 即卸载实例（终端连接释放、内存回收）。
                            不能用 keep-alive + 动态 :is：KeepAlive 缓存仅随 include/exclude、max LRU、自身卸载淘汰，
                            已关闭的 tab 会永久残留缓存（连接不释放、内存无界增长）；而 max 又会静默淘汰活跃终端
                        -->
                        <template v-for="tab in resourceTabs" :key="tab.key">
                            <component
                                v-show="tab.key === activeResourceOpTabKey"
                                :ref="(el: any) => el && registerComponentInstance(tab.key, el)"
                                :is="tab.component"
                                v-bind="tab.componentProps"
                            />
                        </template>
                    </div>
                </el-card>
            </el-splitter-panel>
        </el-splitter>

        <Contextmenu :dropdown="tabDropdown" :items="tabContextmenuItems" ref="tabContextmenuRef" />

        <!-- 渲染注册的非 tab 组件（Overlay）：关闭即从注册表移除，避免常驻内存 -->
        <template v-for="overlay in overlayList" :key="overlay.key">
            <component
                v-if="overlay.visible"
                :is="overlay.component"
                v-bind="overlay.props"
                @update:visible="(val: boolean) => (val ? (overlay.visible = true) : removeResourceOpOverlay(overlay.key))"
            />
        </template>
    </div>
</template>

<script lang="ts" setup>
import { computed, nextTick, onMounted, onUnmounted, ref, useTemplateRef, watch } from 'vue';

import { useAutoOpenResource } from '@/store/autoOpenResource';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import SvgIcon from '@/components/svg-icon/index.vue';
import { storeToRefs } from 'pinia';
import TreeContainer from './tree/TreeContainer.vue';
import type { TreeNode } from './tree/types';
import { getResourceTypes, loadResourceTags } from './resource';
import {
    activateResourceOpTab,
    activeResourceOpTabKey,
    allResourceOpOverlays,
    allResourceOpTabs,
    getComponentInstance,
    getResourceOpTab,
    registerComponentInstance,
    removeResourceOpOverlay,
    removeResourceOpTab,
    ResourceOpTab,
} from './resourceOp';

const autoOpenResourceStore = useAutoOpenResource();
const { autoOpenResource } = storeToRefs(autoOpenResourceStore);

const treeContainerRef = useTemplateRef<{
    locate: (key: string) => Promise<void>;
    refresh: (key?: string) => void;
    getNode: (key: string) => TreeNode | undefined;
}>('treeContainerRef');
const tabContextmenuRef = useTemplateRef<InstanceType<typeof Contextmenu>>('tabContextmenuRef');

// 存储当前组件对应的最后操作的节点key，用户切换资源操作组件时，定位到相应的树节点
const resourceComponentsNodeKey = ref<Record<string, string>>({});

const resourceTabs = computed(() => {
    return Array.from(allResourceOpTabs.values());
});

const overlayList = computed(() => {
    return Array.from(allResourceOpOverlays.values());
});

// Tab 右键菜单
const tabDropdown = ref({ x: 0, y: 0 });
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
    // ResourceOp 卸载后内部 tab 组件实例已销毁，清除引用避免后续拿到失效实例
    allResourceOpTabs.forEach((tab) => (tab.componentInstance = undefined));
});

// 过滤文本经容器内部 300ms 防抖后生效
const filterText = ref('');

// 根节点加载：标签骨架（含类型分组节点预构），资源子树由各贡献者懒加载
const loadRoot = () => loadResourceTags(getResourceTypes());

// ---------------------------------- autoOpen 定位 ----------------------------------

// autoOpen 待定位的节点 key：目标节点可能尚未水合，容器在每次水合完成后自动重试
watch(
    () => autoOpenResource.value.codePath,
    (autoOpenCodePath: string) => {
        if (!autoOpenCodePath) {
            return;
        }

        // 解析 codePath 的最后一段作为定位目标（中间层级由容器按需展开）
        const parts = autoOpenCodePath.split('/');
        let lastKey = '';
        let currentTagPath = '';
        let lastResourceKey = '';
        for (const part of parts) {
            if (!part) {
                continue;
            }
            const [key, value] = part.split('|');
            if (!value) {
                currentTagPath = currentTagPath + key + '/';
                lastKey = currentTagPath;
                continue;
            }
            // 资源段：'-' 前为资源类型，值为资源根节点 code
            if (!lastResourceKey) {
                lastKey = `${currentTagPath}-${key}`;
            }
            lastResourceKey = value;
        }
        autoOpenResourceStore.setCodePath('');
        if (lastKey || lastResourceKey) {
            treeContainerRef.value?.locate(lastResourceKey && lastResourceKey !== lastKey ? lastResourceKey : lastKey);
        }
    },
    { immediate: true }
);

// ---------------------------------- 节点点击 → tab 联动 ----------------------------------

const onTreeNodeClick = (node: TreeNode) => {
    // 节点点击激活了 tab（新建或切换）时，同步记录 tab -> 树节点映射，用于切回时定位
    const prevActiveKey = activeResourceOpTabKey.value;
    nextTick(() => {
        const nowActiveKey = activeResourceOpTabKey.value;
        if (nowActiveKey && nowActiveKey !== prevActiveKey) {
            resourceComponentsNodeKey.value[nowActiveKey] = node.key;
        }
    });
};

// ---------------------------------- Tab 管理 ----------------------------------

// 激活指定标签页
const activateTab = (tabKey: string) => {
    activateResourceOpTab(tabKey);
    if (!tabKey) {
        return;
    }
    // 定位到左侧资源树对应节点（优先 tab 创建/交互时显式登记的 nodeKey）
    const nodeKey = getResourceOpTab(tabKey)?.nodeKey ?? resourceComponentsNodeKey.value[tabKey];
    if (nodeKey) {
        treeContainerRef.value?.locate(nodeKey as string);
    }
    nextTick(() => {
        // 调用该tab的激活回调
        getComponentInstance(tabKey)?.onActivate?.();
    });
};

// 刷新标签页（调用该 tab 注册的刷新回调）
const refreshTab = (tabKey: string) => {
    getComponentInstance(tabKey)?.onRefresh?.();
};

// Tab 右键菜单处理
const onTabContextmenu = (event: MouseEvent, tab: ResourceOpTab) => {
    tabDropdown.value = { x: event.clientX, y: event.clientY };
    tabContextmenuRef.value?.openContextmenu({ tabKey: tab.key });
};

// 关闭标签页
const closeTab = (tabKey: string) => {
    // 先触发该 tab 的关闭回调，再清理实例与映射（清理后实例不可再获取）
    getComponentInstance(tabKey)?.onClose?.();
    removeResourceOpTab(tabKey);
    delete resourceComponentsNodeKey.value[tabKey];

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
        closeTab(key);
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
        closeTab(key);
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
        closeTab(key);
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
        closeTab(key);
    });
    // 如果当前激活的标签被关闭，切换到目标标签
    if (keysToClose.includes(activeResourceOpTabKey.value)) {
        activateTab(targetTabKey);
    }
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
