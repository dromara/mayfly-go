<template>
    <div ref="wrapRef" class="virtual-tree">
        <component :is="adapter.component" ref="treeRef" v-bind="adapterProps" class="virtual-tree__inner min-w-full inline-block">
            <!-- 行/空态插槽按名透传给引擎（适配义务：引擎须暴露 default/empty 插槽，见 adapters/types.ts）；
                 无行插槽时引擎回退内置 label 渲染，保持通用性 -->
            <template v-if="$slots.default" #default="slotProps">
                <slot v-bind="slotProps" />
            </template>
            <template v-if="$slots.empty" #empty>
                <slot name="empty" />
            </template>
        </component>
    </div>
</template>

<script lang="ts" setup>
/**
 * VirtualTree — 全局通用虚拟树容器
 *
 * 职责边界（严格遵守）：
 *   ✅ 容器尺寸管理（ResizeObserver，虚拟引擎需要像素高度）
 *   ✅ 适配器渲染（引擎可替换）
 *   ✅ 语义化 props → 适配器 props 的转发（容器不出现任何引擎 prop 名）
 *   ✅ 归一化事件转发（nodeClick / nodeExpand / nodeCollapse / check）
 *   ✅ 受控展开兜底（expandedKeys 变化同步引擎）与过滤防抖接线
 *   ✅ 插槽透传（default 行渲染 / empty 空态由消费者实现）
 *
 *   ❌ 不包含：右键菜单、行操作按钮、重试按钮（业务层在 default 插槽内自行处理）
 *   ❌ 不包含：分组、筛选、水合等业务状态（业务层 composable 自管）
 *
 * 使用方（Redis key 树 / 资源树 ...）通过 default 插槽注入行渲染，
 * 通过自身 composables 实现分组、勾选语义、右键菜单等功能。
 */
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from 'vue';
import { defaultAdapter, defaultFieldNames } from './adapters';
import type { VirtualTreeAdapter, VirtualTreeFieldNames, VirtualTreeInstance } from './adapters';

const props = withDefaults(
    defineProps<{
        /** 树数据（单一事实源，由业务层持有） */
        data: Record<string, unknown>[];
        /** 节点字段映射（默认 key/label/children/disabled） */
        fieldNames?: VirtualTreeFieldNames;
        /** 树引擎适配器（默认 el-tree-v2） */
        adapter?: VirtualTreeAdapter;
        /** 虚拟滚动视口高度（像素）；缺省实测容器高度自适应 */
        height?: number;
        /** 行高（像素）；虚拟引擎按此定位行，缺省用引擎默认（如 el-tree-v2 的 26） */
        rowHeight?: number;
        /** 受控展开集（业务层持有，容器回显并兜底同步引擎） */
        expandedKeys?: string[];
        /** 本地过滤文本（声明式入口，推荐）：变化后容器防抖 300ms 调引擎 filter，空串不过滤 */
        filterText?: string;
        /** 本地过滤谓词：只定义「匹配」规则，由适配器接线到引擎过滤机制（引擎不支持时 no-op）；触发时机取决于 filterText 或 filter() */
        filterMethod?: (query: string, node: Record<string, unknown>) => boolean;
        /** 是否显示勾选框 */
        checkable?: boolean;
        /** 点击叶子即切换勾选（批量选择场景） */
        checkOnClickLeaf?: boolean;
        /** 当前节点高亮 */
        highlightCurrent?: boolean;
        /** 点击行（非箭头区域）是否展开 */
        expandOnClickNode?: boolean;
    }>(),
    {
        fieldNames: () => defaultFieldNames,
        height: 0,
        expandedKeys: () => [],
        filterText: '',
        checkable: false,
        checkOnClickLeaf: false,
        highlightCurrent: true,
        expandOnClickNode: false,
    }
);

const emit = defineEmits<{
    nodeClick: [node: Record<string, unknown>];
    nodeExpand: [node: Record<string, unknown>];
    nodeCollapse: [node: Record<string, unknown>];
    /** 勾选变化：叶子 key 列表（口径由适配器归一） */
    check: [checkedLeafKeys: string[]];
}>();

const adapter = computed(() => props.adapter ?? defaultAdapter);

// ==================== 容器尺寸 ====================

const wrapRef = useTemplateRef<HTMLElement>('wrapRef');
const measuredHeight = ref(400);
let resizeObserver: ResizeObserver | null = null;

// ==================== 实例与适配器 Props ====================

const treeRef = useTemplateRef<unknown>('treeRef');
const instance = computed<VirtualTreeInstance>(() => adapter.value.getInstance(treeRef));

const adapterProps = computed(() =>
    adapter.value.getProps({
        data: props.data,
        fieldNames: props.fieldNames,
        height: props.height || measuredHeight.value,
        rowHeight: props.rowHeight,
        expandedKeys: props.expandedKeys,
        checkable: props.checkable,
        checkOnClickLeaf: props.checkOnClickLeaf,
        highlightCurrent: props.highlightCurrent,
        expandOnClickNode: props.expandOnClickNode,
        filterMethod: props.filterMethod,
        events: {
            onNodeClick: (node) => emit('nodeClick', node),
            onNodeExpand: (node) => emit('nodeExpand', node),
            onNodeCollapse: (node) => emit('nodeCollapse', node),
            onCheck: (keys) => emit('check', keys),
        },
    })
);

// ==================== 受控展开与过滤 ====================

/**
 * 受控展开的引擎无关兜底（与引擎自身重放构成双保险，权威源始终是业务层的 expandedKeys）：
 * - 引擎侧：适配器把 expandedKeys 映射为 defaultExpandedKeys，el-tree-v2 在该 prop 变化与
 *   data 重建时都会自动重放展开集；
 * - 容器侧：这里再显式 setExpandedKeys 全量回写（幂等）。刻意保留是为了「换任意引擎都成立」——
 *   受控语义由容器统一保证，不依赖某引擎是否自动重放（开闭原则）。两条通道都只是把同一份
 *   expandedKeys 落到引擎，无第二真源。
 */
watch(
    () => props.expandedKeys,
    (keys) => {
        nextTick(() => instance.value.setExpandedKeys(keys));
    }
);

/** 过滤防抖（filterText 声明式入口的实现）：输入停顿 300ms 才打到引擎，避免逐字符重建过滤集 */
let filterTimer: ReturnType<typeof setTimeout> | null = null;
watch(
    () => props.filterText,
    (val) => {
        if (filterTimer) {
            clearTimeout(filterTimer);
        }
        filterTimer = setTimeout(() => instance.value.filter(val), 300);
    }
);

onMounted(() => {
    if (!wrapRef.value) {
        return;
    }
    const rect = wrapRef.value.getBoundingClientRect();
    if (rect.height > 0) {
        measuredHeight.value = rect.height;
    }
    resizeObserver = new ResizeObserver((entries) => {
        for (const entry of entries) {
            if (entry.contentRect.height > 0) {
                measuredHeight.value = entry.contentRect.height;
            }
        }
    });
    resizeObserver.observe(wrapRef.value);
});

onBeforeUnmount(() => {
    resizeObserver?.disconnect();
    if (filterTimer) {
        clearTimeout(filterTimer);
    }
});

// ==================== 公开 API ====================

defineExpose<VirtualTreeInstance>({
    setCurrentKey: (key) => instance.value.setCurrentKey(key),
    scrollToNode: (key, strategy) => instance.value.scrollToNode(key, strategy),
    scrollTo: (offset) => instance.value.scrollTo(offset),
    getCheckedKeys: (leafOnly) => instance.value.getCheckedKeys(leafOnly),
    setCheckedKeys: (keys) => instance.value.setCheckedKeys(keys),
    setExpandedKeys: (keys) => instance.value.setExpandedKeys(keys),
    filter: (query) => instance.value.filter(query),
});
</script>

<style lang="scss">
.virtual-tree {
    position: relative;
    overflow: hidden;
    height: 100%;
    width: 100%;
}
</style>
