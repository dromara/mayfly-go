<template>
    <!-- 懒加载占位行（淡入，见引擎动效样式） -->
    <div v-if="node.kind === LOADING_KIND" class="tree-row-placeholder h-7 flex items-center text-(--el-text-color-secondary) select-none" :style="{ paddingLeft: '4px' }">
        <SvgIcon name="Loading" class="is-loading text-xs" />
        <span class="ml-1 text-xs">{{ $t('common.loadingMore') }}</span>
    </div>

    <!-- 加载失败行：点击重试加载父节点子树 -->
    <div
        v-else-if="node.kind === ERROR_KIND"
        class="tree-row-placeholder h-7 flex items-center text-(--el-color-danger) cursor-pointer select-none"
        :style="{ paddingLeft: '4px' }"
        @click.stop="$emit('retry', node)"
    >
        <SvgIcon name="RefreshRight" class="text-xs" />
        <span class="ml-1 text-xs">{{ $t('common.retry') }}</span>
    </div>

    <!-- 具体类型节点：贡献者自定义渲染器优先，缺省通用行 -->
    <component
        :is="getContributor(node.kind)?.renderer ?? TreeNodeRow"
        v-else
        :data="node"
        :show-actions="showActions"
        @contextmenu.prevent="$emit('rowContextmenu', $event, node)"
    />
</template>

<script lang="ts" setup>
/**
 * 树行内容渲染（引擎无关）：
 * 占位行/错误行/贡献者渲染器分发的统一实现，任意树引擎适配器（TreeEngineV2 等）
 * 在行插槽中透传节点即可复用，换树库不影响行渲染协议。
 */
import { getContributor } from './registry';
import { ERROR_KIND, LOADING_KIND, type TreeNode } from './types';
import TreeNodeRow from './TreeNodeRow.vue';

withDefaults(
    defineProps<{
        node: TreeNode;
        showActions?: boolean;
    }>(),
    { showActions: true }
);

defineEmits<{
    retry: [node: TreeNode];
    rowContextmenu: [event: MouseEvent, node: TreeNode];
}>();
</script>
