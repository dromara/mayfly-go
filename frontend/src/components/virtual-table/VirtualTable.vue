<template>
    <div ref="containerRef" class="virtual-table" :style="height ? { height } : {}">
        <component
            :is="adapter.component"
            ref="tableRef"
            v-bind="adapterProps"
            class="virtual-table__inner"
        >
            <!-- header slot: 仅当消费者提供时才透传，否则使用适配器默认表头 -->
            <template v-if="$slots.header" #header="slotProps">
                <slot name="header" v-bind="slotProps" />
            </template>
            <!-- cell slot: 仅当消费者提供时才透传 -->
            <template v-if="$slots.cell" #cell="slotProps">
                <slot name="cell" v-bind="slotProps" />
            </template>
            <!-- 加载覆盖层插槽：仅在 loading 时激活 -->
            <template v-if="loading && $slots.overlay" #overlay>
                <slot name="overlay" />
            </template>
        </component>
    </div>
</template>

<script lang="ts" setup>
/**
 * VirtualTable — 全局通用虚拟表格容器
 *
 * 职责边界（严格遵守）：
 *   ✅ 容器尺寸管理（ResizeObserver）
 *   ✅ 适配器渲染（引擎可替换）
 *   ✅ 加载覆盖层（loading overlay）
 *   ✅ 空数据占位行（维持横向滚动能力）
 *   ✅ 插槽透传（header / cell / overlay 由消费者自行实现）
 *
 *   ❌ 不包含：右键菜单（业务层通过 cell slot 自行处理）
 *   ❌ 不包含：行选择（业务层通过 composable + cell slot 自行处理）
 *   ❌ 不包含：行号列（业务层在列定义中自行添加）
 *   ❌ 不包含：默认列头（业务层通过 header slot 自行实现）
 *
 * 架构示意：
 *
 *   ┌──────────────────────────────────────────────────┐
 *   │  VirtualTable (通用容器)                           │
 *   │  ┌─────────────────────────────────────────────┐ │
 *   │  │  ResizeObserver → containerSize             │ │
 *   │  │  adapter.getProps() → <component :is />    │ │
 *   │  │  loading overlay (slot 透传)                │ │
 *   │  │  empty data spacer                         │ │
 *   │  └─────────────────────────────────────────────┘ │
 *   └──────────────────────────────────────────────────┘
 *
 * 使用方（DB / ES / Mongo ...）通过 slots 注入业务特有的渲染逻辑，
 * 通过自身 composables 实现行选择、右键菜单等功能。
 */
import { computed, onBeforeUnmount, onMounted, reactive, ref, useTemplateRef } from 'vue';
import { defaultAdapter, normalizeColumns } from './adapters';
import type { VirtualTableAdapter, VirtualTableColumn, NormalizedRowEventHandlers, NormalizedScrollEvent } from './adapters';

const props = defineProps<{
    /** 表格数据 */
    data: Record<string, unknown>[];
    /** 列定义 */
    columns: VirtualTableColumn[];
    /** 表格引擎适配器（默认 el-table-v2） */
    adapter?: VirtualTableAdapter;
    /** 容器高度 */
    height?: string;
    /** 行高 */
    rowHeight?: number;
    /** 表头高度 */
    headerHeight?: number;
    /** 是否显示加载状态 */
    loading?: boolean;
    /** 行样式回调（由业务层控制，如选中高亮） */
    rowClass?: (row: { rowIndex: number }) => string;
    /** 行事件处理器（由业务层控制，如点击选择） */
    rowEventHandlers?: NormalizedRowEventHandlers;
    /** 滚动事件回调 */
    onScroll?: (e: NormalizedScrollEvent) => void;
}>();

const emit = defineEmits<{
    scrollChange: [scroll: { scrollLeft: number; scrollTop: number }];
}>();

// ==================== 适配器 ====================

const adapter = computed(() => props.adapter ?? defaultAdapter);

// ==================== 容器尺寸 ====================

const containerRef = useTemplateRef<HTMLElement>('containerRef');
const tableRef = ref();
let resizeObserver: ResizeObserver | null = null;

const containerSize = reactive({ width: 800, height: 600 });

onMounted(() => {
    if (containerRef.value) {
        const rect = containerRef.value.getBoundingClientRect();
        containerSize.height = rect.height || 600;
        containerSize.width = rect.width || 800;

        resizeObserver = new ResizeObserver((entries) => {
            for (const entry of entries) {
                const { height, width } = entry.contentRect;
                if (height > 0) containerSize.height = height;
                if (width > 0) containerSize.width = width;
            }
        });
        resizeObserver.observe(containerRef.value);
    }
});

onBeforeUnmount(() => {
    resizeObserver?.disconnect();
});

// ==================== 适配器 Props 构建 ====================

const adapterProps = computed(() => {
    const normalizedCols = normalizeColumns(props.columns);

    // 当数据为空但列存在时，添加一个占位行以维持 scroll container 宽度，使 header 可横向滚动
    let data = props.data;
    if (data.length === 0 && props.columns.length > 0) {
        data = [{ __vt_empty_spacer__: true }];
    }

    return adapter.value.getProps({
        columns: normalizedCols,
        data,
        width: containerSize.width,
        height: containerSize.height,
        headerHeight: props.headerHeight ?? 30,
        rowHeight: props.rowHeight ?? 30,
        rowClass: props.rowClass,
        rowEventHandlers: props.rowEventHandlers,
        onScroll: (e) => emit('scrollChange', e),
    });
});

// ==================== 表格实例 ====================

const tableInstance = computed(() => adapter.value.getInstance(tableRef));

// ==================== 公开 API ====================

defineExpose({
    /** 滚动到指定位置 */
    scrollTo: (options: { scrollLeft?: number; scrollTop?: number }) => tableInstance.value.scrollTo(options),
    /** 水平滚动 */
    scrollToLeft: (left: number) => tableInstance.value.scrollToLeft(left),
});
</script>

<style lang="scss">
.virtual-table {
    position: relative;
    overflow: hidden;
    height: 100%;

    &__inner {
        border-left: var(--el-table-border);
        border-top: var(--el-table-border);
    }
}
</style>
