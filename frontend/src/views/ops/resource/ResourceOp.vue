<template>
    <div class="h-full" :class="{ 'resource-op-fullscreen': isFullscreen }">
        <el-splitter @resize="onResizeOpPanel">
            <el-splitter-panel size="24%" max="40%">
                <el-card class="h-full flex" body-class="p-0! flex flex-col w-full">
                    <!-- 定位上下文：查找入口与查找条 absolute 浮在树右上角，不给树留任何一行的空间 -->
                    <div class="relative min-h-0 flex-1">
                        <!-- 悬浮工具行：类型筛选（查找条展开时让位，顶部一次只留一个浮层控件） -->
                        <div v-show="!findVisible" class="tree-view-tools">
                            <TreeTypeFilter v-model="typeFilter" />
                        </div>
                        <TreeFindBar
                            ref="findBarRef"
                            :visible="findVisible"
                            :query="filterText"
                            :total="matchTotal"
                            :no="matchNo"
                            :count-ready="countReady"
                            :scope-label-key="typeFilterLabelKey"
                            @update:query="onFindQueryChange"
                            @expand="openFind"
                            @collapse="closeFind"
                            @step="stepMatch"
                        />
                        <TreeContainer
                            ref="treeContainerRef"
                            :load-root="loadRoot"
                            :filter-text="filterText"
                            :node-filter="nodeFilter"
                            :resolve-locate="resolveCodePathLocate"
                            @node-click="onTreeNodeClick"
                        />
                    </div>
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
                            <!--
                                v-show 必须挂在本包装层，不能挂回 <component> 上：挂在组件上的指令要靠 fallthrough
                                透传到「组件根元素」，而 tab 组件顶层允许有第二个元素根（如 RedisDataOp 根 div 之外的
                                WorkTicketSubmit），此时 Vue 找不到唯一的根，指令被挂到 Fragment 上静默失效——该 tab
                                永远可见、盖住其它 tab（实测症状：激活 DB tab 仍显示 Redis 界面）。顶层注释不算根，
                                不影响透传。main.ts 置空了 warnHandler，连 Vue 那条 dev 告警都看不到，只能靠结构规避。

                                包装层是普通元素，v-show 恒生效，容器因此对 tab 组件的根结构零约束（同
                                components/tabs/Tabs.vue 的 .mf-tabs__pane 做法）；ref 仍留在 <component> 上取组件实例。
                                本不变式由 __tests__/dynamicComponentDirective.test.ts 钉住。
                            -->
                            <div v-show="tab.key === activeResourceOpTabKey" class="h-full">
                                <component
                                    :ref="(el: any) => el && registerComponentInstance(tab.key, el)"
                                    :is="tab.component"
                                    v-bind="tab.componentProps"
                                />
                            </div>
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
import { computed, nextTick, onActivated, onDeactivated, onMounted, onUnmounted, ref, useTemplateRef, watch } from 'vue';

import { useAutoOpenResource } from '@/store/autoOpenResource';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import SvgIcon from '@/components/svg-icon/index.vue';
import { storeToRefs } from 'pinia';
import TreeContainer from './tree/TreeContainer.vue';
import TreeFindBar from './tree/TreeFindBar.vue';
import TreeTypeFilter from './TreeTypeFilter.vue';
import { nextMatchIndex } from './tree';
import { resolveCodePathLocate } from './locateResource';
import { createTypeFilter, deriveTypeOptions } from './tagResources';
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
    locateIdentity: (identity: string) => Promise<void>;
    refresh: (key?: string) => void;
    getNode: (key: string) => TreeNode | undefined;
    matchedKeys: string[];
    appliedQuery: string;
}>('treeContainerRef');
const tabContextmenuRef = useTemplateRef<InstanceType<typeof Contextmenu>>('tabContextmenuRef');
const findBarRef = useTemplateRef<{ focusInput: () => void }>('findBarRef');

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

const cmTabCloseLeft = new ContextmenuItem('closeLeft', 'layout.tagsView.closeLeft')
    .withIcon('Back')
    .withOnClick((data: unknown) => closeLeftTabs((data as Record<string, unknown>).tabKey as string));

const cmTabCloseRight = new ContextmenuItem('closeRight', 'layout.tagsView.closeRight')
    .withIcon('Right')
    .withOnClick((data: unknown) => closeRightTabs((data as Record<string, unknown>).tabKey as string));

const cmTabCloseOther = new ContextmenuItem('closeOther', 'layout.tagsView.closeOther')
    .withIcon('Switch')
    .withOnClick((data: unknown) => closeOtherTabs((data as Record<string, unknown>).tabKey as string));

tabContextmenuItems.value = [cmTabCloseLeft, cmTabCloseRight, cmTabCloseOther, cmTabCloseAll];

// 右侧面板全屏相关（Escape 与查找条的优先级一并在下方 onKeydown 里裁决）
const isFullscreen = ref(false);

const toggleFullscreen = () => {
    isFullscreen.value = !isFullscreen.value;
};

onUnmounted(() => {
    // ResourceOp 卸载后内部 tab 组件实例已销毁，清除引用避免后续拿到失效实例
    allResourceOpTabs.forEach((tab) => (tab.componentInstance = undefined));
});

// ---------------------------------- 资源树查找 ----------------------------------

// 过滤文本经容器内部 300ms 防抖后生效
const filterText = ref('');

// 查找条展开态：由页面持有，键盘入口（⌘F）与 Escape 的优先级在此统一裁决
const findVisible = ref(false);

// 当前跳转位序（-1 表示还没按过 Enter 跳转）；命中集由树容器给出，口径与引擎过滤后的可见行一致
const matchCursor = ref(-1);

const matchKeys = computed(() => treeContainerRef.value?.matchedKeys ?? []);
const matchTotal = computed(() => matchKeys.value.length);
// 计数只在引擎真的按当前词过滤之后呈现：防抖窗口里报旧值（或空词时的 0）等于撒谎
const countReady = computed(() => (treeContainerRef.value?.appliedQuery ?? '') === filterText.value);
// 位序钳在总数内：搜索态下折叠大子树会释放节点（releaseOnCollapse）使命中集收缩，旧位序会显示成 6/3
const matchNo = computed(() => (matchCursor.value < 0 ? 0 : Math.min(matchCursor.value + 1, matchTotal.value)));

// 命中集收缩（收起分支、大子树折叠释放）时把指针夹回集合内：否则会出现「计数 1/3 而指针指着看不见的行」
watch(matchTotal, (total) => {
    if (matchCursor.value >= total) {
        matchCursor.value = total > 0 ? total - 1 : -1;
    }
});

// 换词即作废上一轮的跳转位序：旧下标对新命中集没有意义
const onFindQueryChange = (value: string) => {
    filterText.value = value;
    matchCursor.value = -1;
};

const openFind = () => {
    findVisible.value = true;
};

// 收起即清空过滤：留着过滤态收起，树会被看不到的规则筛着，用户只看到「树怎么少了几个节点」
const closeFind = () => {
    findVisible.value = false;
    onFindQueryChange('');
};

// ---------------------------------- 按资源类型筛选 ----------------------------------

// 生效的资源根节点 kind（null = 不筛选）；类型层级已从树里移除，这里把同一维度作为视图轴补回
const typeFilter = ref<string | null>(null);

// 谓词工厂在 tagResources.createTypeFilter（纯函数 + 可单测）：换类型即换函数引用，容器据此重过滤
const nodeFilter = computed(() =>
    typeFilter.value ? createTypeFilter(typeFilter.value, (key) => treeContainerRef.value?.getNode(key)) : undefined
);

// 查找条展开时入口 chip 让位，生效类型改由查找条上这枚标签呈现（不留看不见的过滤规则）
const typeFilterLabelKey = computed(() =>
    typeFilter.value ? deriveTypeOptions().find((option) => option.kind === typeFilter.value)?.label : undefined
);

// 在命中集内循环跳转（delta：1 下一个 / -1 上一个）：位序算式在 filter/nextMatchIndex（可单测）
const stepMatch = (delta: number) => {
    const keys = matchKeys.value;
    const next = nextMatchIndex(matchCursor.value, keys.length, delta);
    if (next < 0) {
        return;
    }
    matchCursor.value = next;
    // locate 自带展开祖先链 + 居中滚动
    treeContainerRef.value?.locate(keys[next]);
};

/**
 * 文本类控件与自带键位的面板（编辑器/终端/远程桌面）不接管快捷键：
 * 那些控件里的 ⌘F/Esc 属于它自己（Monaco 有自带查找面板、xterm/RDP 把 Esc 发给远端）。
 * contenteditable 只排除显式写了 false 的（裸写 / 空值按 HTML 语义等价 true）。
 */
const KEY_TAKEOVER_EXCLUDE =
    'input, textarea, select, [contenteditable]:not([contenteditable="false"]), .monaco-editor, .terminal-container, .xterm, .rdpDialog';

const isTextEntryTarget = (target: EventTarget | null) => !!(target as HTMLElement | null)?.closest?.(KEY_TAKEOVER_EXCLUDE);

/**
 * 页面级键盘接管：⌘/Ctrl+F 唤起查找，Escape 先收查找条再退出全屏。
 *
 * 一次按键只做一个动作：查找条展开时 Escape 就地收查找条（输入框内已 stopPropagation，
 * 焦点在别处时走本函数的 findVisible 优先分支）。
 */
const onKeydown = (e: KeyboardEvent) => {
    // 已被其它处理器消费（Monaco 等）的按键不再插手
    if (e.defaultPrevented) {
        return;
    }
    if ((e.metaKey || e.ctrlKey) && !e.shiftKey && !e.altKey && e.key.toLowerCase() === 'f') {
        if (findVisible.value) {
            // 不拦默认行为会唤出浏览器「在页面中查找」浮层盖过面板查找；Shift/Alt 组合（如 Ctrl+Shift+F 全局搜索）已在上边放行
            e.preventDefault();
            if (!isTextEntryTarget(e.target)) {
                findBarRef.value?.focusInput();
            }
            return;
        }
        if (isTextEntryTarget(e.target)) {
            return;
        }
        e.preventDefault();
        openFind();
        return;
    }
    if (e.key !== 'Escape' || isTextEntryTarget(e.target)) {
        return;
    }
    if (findVisible.value) {
        closeFind();
        return;
    }
    if (isFullscreen.value) {
        isFullscreen.value = false;
    }
};

// keep-alive 页面 deactivated 后仍 mounted，监听必须随激活态挂/摘，否则在其它页面抢掉 ⌘F/Esc
onMounted(() => document.addEventListener('keydown', onKeydown));
onActivated(() => document.addEventListener('keydown', onKeydown));
onDeactivated(() => document.removeEventListener('keydown', onKeydown));
onUnmounted(() => document.removeEventListener('keydown', onKeydown));

// 根节点加载：标签骨架（该标签含哪些资源类型记在 params.resourceTypes），资源行由标签展开时按需取数
const loadRoot = () => loadResourceTags(getResourceTypes());

// ---------------------------------- autoOpen 定位 ----------------------------------

// autoOpen 待定位的资源 codePath（原始串，解析与逐层下钻交由注入树容器的 resolveCodePathLocate 完成）
let pendingLocateCodePath = '';

// 消费待定位 codePath：tree 就绪即发起定位
// 首次挂载由 onMounted 触发、keep-alive 缓存后再次进入由 onActivated 触发（此时不会再次 onMounted）
// 注意：仅在 DOM 已挂载/重新挂载时消费，避免在 deactivated 状态对脱离的 DOM 定位导致滚动失效
const consumePendingLocate = () => {
    if (!pendingLocateCodePath) {
        return;
    }
    treeContainerRef.value?.locateIdentity(pendingLocateCodePath);
    pendingLocateCodePath = '';
};

watch(
    () => autoOpenResource.value.codePath,
    (autoOpenCodePath: string) => {
        if (!autoOpenCodePath) {
            return;
        }
        pendingLocateCodePath = autoOpenCodePath;
        // 捕获后立即清空 store，避免重复触发或返回页面时再次定位
        autoOpenResourceStore.setCodePath('');
    },
    { immediate: true }
);

onMounted(consumePendingLocate);
onActivated(consumePendingLocate);

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
.resource-tab-content {
    flex: 1;
    min-height: 0;
    overflow: hidden;
    padding: 4px;
}

/* 悬浮工具行：贴在查找入口左侧（入口占 right:8 的 20px），与它同一套右上角视觉 */
.tree-view-tools {
    position: absolute;
    top: 2px;
    right: 32px;
    z-index: 2;
    display: flex;
    gap: 2px;
}

/* 水平分割面板 flex 最小尺寸修复：
 * 面板内 Monaco 等内容有较大的 intrinsic min-content，
 * flexbox 默认 min-width:auto 会阻止面板缩小到内容尺寸以下，
 * 导致分割线只能向左（面板缩小方向）不能向右（面板需进一步缩小方向）。
 * 设 min-width:0 覆盖自动最小尺寸，允许面板自由收缩。 */
:deep(.el-splitter__horizontal > .el-splitter-panel) {
    min-width: 0;
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
