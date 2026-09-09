<template>
    <div ref="wrapRef" class="w-full h-full overflow-hidden">
        <el-tree-v2
            ref="treeV2Ref"
            class="min-w-full inline-block"
            :data="props.data"
            :props="treeV2Props"
            :height="engineHeight"
            :indent="10"
            :expand-on-click-node="false"
            highlight-current
            :filter-method="filterMethod"
            :default-expanded-keys="props.expandedKeys"
            @node-click="(node: TreeNode) => emit('nodeClick', node)"
            @node-expand="onExpand"
            @node-collapse="onCollapse"
        >
            <template #default="{ data: node }">
                <TreeRowContent :node="node" :show-actions="props.showActions" @retry="(n) => emit('nodeRetry', n)" @row-contextmenu="(e, n) => emit('rowContextmenu', e, n)" />
            </template>
        </el-tree-v2>
    </div>
</template>

<script lang="ts" setup>
/**
 * 树引擎适配器：el-tree-v2 的唯一知识收口点。
 *
 * 容器（TreeContainer）只依赖本组件的 props/emits/expose 契约（见 types.ts TreeEngineExpose），
 * 不直接接触任何树库 API。若后续替换为 el-tree v1 或其他树组件：
 * 1. 新建 TreeEngineV1.vue 实现同一契约（data/expandedKeys 受控、事件转发、setCurrentKey/scrollToNode）；
 * 2. TreeContainer 更换一行 import 即完成切换。
 * 引擎职责边界：视口高度实测、字段映射、过滤谓词接线、展开/点击/右键事件转发；其余全部在容器。
 */
import { computed, ref, watch } from 'vue';
import { useResizeObserver } from '@vueuse/core';

import { isPrefixSubsequence } from '@/common/utils/string';

import { ERROR_KIND, LOADING_KIND, type TreeNode, type TreeNodeData, type TreeEngineExpose } from './types';
import TreeRowContent from './TreeRowContent.vue';

const props = withDefaults(
    defineProps<{
        /** 水合层数据（单一事实源） */
        data: TreeNode[];
        /** 受控展开态（容器拥有，引擎只回显） */
        expandedKeys: string[];
        /** 外部过滤文本（引擎内部防抖生效） */
        filterText?: string;
        /** 虚拟滚动视口高度（像素）；缺省实测容器高度自适应 */
        height?: number;
        /** 行操作按钮透传 */
        showActions?: boolean;
    }>(),
    { filterText: '', height: 0, showActions: true }
);

const emit = defineEmits<{
    nodeClick: [node: TreeNode];
    nodeExpand: [node: TreeNode];
    nodeCollapse: [node: TreeNode];
    nodeRetry: [node: TreeNode];
    rowContextmenu: [event: MouseEvent, node: TreeNode];
}>();

const treeV2Ref = ref<{
    filter: (val: string) => void;
    setCurrentKey: (key: string) => void;
    scrollToNode: (key: string, strategy?: 'auto' | 'center' | 'start' | 'end' | 'nearest') => void;
}>();

defineExpose<TreeEngineExpose>({
    setCurrentKey: (key) => treeV2Ref.value?.setCurrentKey(key),
    scrollToNode: (key, strategy) => treeV2Ref.value?.scrollToNode(key, strategy),
});

// el-tree-v2 需要像素高度：缺省实测容器高度自适应（ResizeObserver 跟随 Splitter 拖拽/全屏切换）
const wrapRef = ref<HTMLElement>();
const measuredHeight = ref(400);
useResizeObserver(wrapRef, (entries) => {
    const h = entries[0].contentRect.height;
    if (h > 0) {
        measuredHeight.value = h;
    }
});
const engineHeight = computed(() => props.height || measuredHeight.value);

// 字段映射（协议字段 → TreeV2 配置）
const treeV2Props = { value: 'key', label: 'label', children: 'children', disabled: 'disabled' };

// 过滤谓词与树库无关（前缀子序列匹配），由引擎接线到具体库的过滤机制
const filterMethod = (query: string, node: TreeNodeData) => !query || isPrefixSubsequence(query, node.label);

let filterTimer: ReturnType<typeof setTimeout> | null = null;
watch(
    () => props.filterText,
    (val) => {
        if (filterTimer) clearTimeout(filterTimer);
        filterTimer = setTimeout(() => treeV2Ref.value?.filter(val), 300);
    }
);

// 引擎只转发事件：占位/错误行不参与（水合层自管 expandedKeys）
const onExpand = (node: TreeNode) => {
    if (node.kind === LOADING_KIND || node.kind === ERROR_KIND) {
        return;
    }
    emit('nodeExpand', node);
};
const onCollapse = (node: TreeNode) => {
    if (node.kind === LOADING_KIND || node.kind === ERROR_KIND) {
        return;
    }
    emit('nodeCollapse', node);
};
</script>

<style lang="scss" scoped>
/* 树动效（树库 DOM 的唯一知识收口点，动效样式同归此处；
   缓动/时长对齐 PRODUCT.md 统一缓动语言：ease-out-quart、150–250ms） */

/* 行 hover/选中/当前高亮背景：内置为瞬时切换 → 150ms 柔和过渡 */
:deep(.el-tree-node__content) {
    transition: background-color 150ms var(--ease-out-quart);
}

/* 箭头旋转：内置 ease-in-out 0.3s → ease-out-quart 200ms，并补色彩过渡 */
:deep(.el-tree-node__expand-icon) {
    transition:
        transform 200ms var(--ease-out-quart),
        color 120ms var(--ease-out-quart);
}

/* 可展开行的 hover 主色提示（仅非禁用行：is-focusable 由树库按 disabled 求值） */
:deep(.el-tree-node.is-focusable .el-tree-node__content:hover .el-tree-node__expand-icon) {
    color: var(--el-color-primary);
}

/* 水合占位/错误行淡入：状态出现有过渡，替代瞬时闪现 */
:deep(.tree-row-placeholder) {
    animation: tree-row-fade-in 200ms var(--ease-out-quart);
}

@keyframes tree-row-fade-in {
    from {
        opacity: 0;
    }

    to {
        opacity: 1;
    }
}
</style>
