<template>
    <div
        :id="props.node.key"
        class="w-full node-container flex items-center cursor-pointer select-none"
        :class="props.data.type?.nodeDblclickFunc ? 'select-none' : ''"
        @mouseenter="showActions = true"
        @mouseleave="showActions = false"
    >
        <!-- prefix -->
        <SvgIcon :size="13" v-if="data.icon" :name="data.icon.name" :color="data.icon.color" />
        <slot :node="node" :data="data" name="prefix"></slot>

        <!-- label -->
        <span class="ml-0.5" :title="data.labelRemark">
            <slot name="label" :data="data" v-if="!data.disabled"> {{ $t(data.label) }}</slot>

            <!-- 禁用状态 -->
            <slot name="disabledLabel" :data="data" v-else>
                <el-link type="danger" disabled underline="never">
                    {{ `${$t(data.label)}` }}
                </el-link>
            </slot>
        </span>

        <!-- 操作按钮 or suffix 区域 -->
        <span v-if="(showActions || dropdownVisible) && !data.disabled && contextMenuItems.length > 0" class="ml-auto pr-2.5 flex items-center">
            <el-dropdown size="small" trigger="click" @command="handleCommand" @visibleChange="(visible: boolean) => (dropdownVisible = visible)">
                <el-button text bg size="small" circle @click.stop type="primary">
                    <SvgIcon name="MoreFilled" />
                </el-button>

                <template #dropdown>
                    <el-dropdown-menu>
                        <template v-for="item in contextMenuItems" :key="item.clickId">
                            <el-dropdown-item v-if="!item.isHide(props.data)" :command="item">
                                <SvgIcon v-if="item.icon" :name="item.icon" class="mr-1" />{{ $t(item.txt) }}
                            </el-dropdown-item>
                        </template>
                    </el-dropdown-menu>
                </template>
            </el-dropdown>
        </span>

        <span v-else class="ml-auto pr-2 text-[10px] text-gray-400">
            <slot :node="node" :data="data" name="suffix"></slot>
        </span>
    </div>
</template>

<script lang="ts" setup>
import { ref, computed, inject } from 'vue';
import SvgIcon from '@/components/svgIcon/index.vue';

import { ContextmenuItem } from '@/components/contextmenu';
import { ResourceOpCtx, TagTreeNode } from '@/views/ops/component/tag';
import { ResourceOpCtxKey } from '@/views/ops/resource/resource';

const resourceOpCtx: ResourceOpCtx | undefined = inject(ResourceOpCtxKey, undefined);

const props = defineProps({
    node: {
        type: [Object],
        default: () => ({}),
    },
    data: {
        type: [TagTreeNode],
        default: () => ({}),
    },
});

const emit = defineEmits(['contextmenu']);

const showActions = ref(false);
const dropdownVisible = ref(false);

// 获取上下文菜单项
const contextMenuItems = computed(() => {
    let items = props.data.type.contextMenuItems;
    if (!items || items.length == 0) {
        // 如果 BaseTreeNode 组件无法直接访问父组件的 loadContextmenuItems 方法
        // 可以通过事件通知父组件处理
        return [];
    }
    return items;
});

// 处理命令点击
const handleCommand = (contextMenuItem: ContextmenuItem) => {
    contextMenuItem.onClickFunc({ ...props.data, ctx: resourceOpCtx });
};
</script>

<style lang="scss"></style>
