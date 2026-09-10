<template>
    <div
        class="tree-row w-full flex items-center cursor-pointer select-none h-7 pr-1.5"
        :title="data.labelRemark"
        @mouseenter="onRowEnter"
        @mouseleave="onRowLeave"
    >
        <!-- 前缀插槽（如实例信息 popover），缺省用节点图标 -->
        <slot name="prefix" :data="data">
            <SvgIcon :size="13" v-if="data.icon" :name="data.icon.name" :color="data.icon.color" />
        </slot>

        <!-- 标签 -->
        <span class="ml-0.5 truncate" :title="data.labelRemark">
            <!-- 贡献者行内 label 渲染器优先（如标签路径 '/' 分段高亮），缺省 i18n 文本 -->
            <component :is="labelRenderer" v-if="labelRenderer" :node="data" />

            <el-link v-else-if="data.disabled" type="danger" disabled underline="never">
                {{ $t(data.label) }}
            </el-link>

            <template v-else>{{ $t(data.label) }}</template>
        </span>

        <!-- 悬浮操作按钮（与右键菜单同源：resolveNodeMenu）；出现时替代右侧信息区（对齐旧 BaseTreeNode 行为）；禁用节点不展示；
             菜单展开期间保持渲染（parked || dropdownVisible）：面板 teleport 到 body，鼠标移向面板会离开行，
             若仅依赖驻留态会连带锚点一起卸载导致弹层即开即消（无法点击菜单项）；
             与信息区互换走交叉淡入淡出，消除 v-if 切换的布局跳变 -->
        <Transition name="tree-row-swap" mode="out-in">
            <span v-if="!data.disabled && showActions && (parked || dropdownVisible) && visibleMenuItems.length" class="tree-row-actions ml-auto flex items-center shrink-0 pr-1">
                <el-dropdown size="small" trigger="click" @command="onMenuCommand" @visible-change="(v: boolean) => (dropdownVisible = v)">
                    <el-button text bg size="small" circle type="primary" @click.stop>
                        <SvgIcon name="MoreFilled" />
                    </el-button>
                    <template #dropdown>
                        <el-dropdown-menu>
                            <template v-for="item in visibleMenuItems" :key="item.clickId">
                                <el-dropdown-item :command="item">
                                    <SvgIcon v-if="item.icon" :name="item.icon" class="mr-1" />
                                    {{ $t(item.txt) }}
                                </el-dropdown-item>
                            </template>
                        </el-dropdown-menu>
                    </template>
                </el-dropdown>
            </span>

            <!-- 右侧信息区：角标 + suffix（凭证账号@主机、key 数量等），右对齐（字体样式对齐旧 BaseTreeNode） -->
            <span v-else class="ml-auto flex items-center gap-1 min-w-0 shrink-0 pr-2 text-[10px] text-gray-400">
                <span v-if="data.badge" class="text-[10px] text-gray-400 shrink-0">{{ data.badge }}</span>
                <slot name="suffix" :data="data" />
            </span>
        </Transition>
    </div>
</template>

<script lang="ts" setup>
import { computed, inject, onBeforeUnmount, ref } from 'vue';

import SvgIcon from '@/components/svg-icon/index.vue';

import { TreeApiKey } from './context';
import { resolveNodeMenu } from './commands';
import { getContributor } from './registry';
import { type TreeNode, type TreeCommandCtx } from './types';

const props = withDefaults(
    defineProps<{
        data: TreeNode;
        showActions?: boolean;
    }>(),
    { showActions: true }
);

const tree = inject(TreeApiKey)!;

/**
 * 操作按钮驻留显示：鼠标在行上停留 HOVER_PARK_MS 后才出现。
 * 滑动扫过行时不闪现 icon（替代即时的 hovered 布尔），驻留即视为有操作意图；
 * 菜单展开期间不受驻留态影响（锚点保持渲染，防 teleport 弹层连带关闭）
 */
const HOVER_PARK_MS = 300;
const parked = ref(false);
let parkTimer: ReturnType<typeof setTimeout> | null = null;

const onRowEnter = () => {
    if (parkTimer) {
        clearTimeout(parkTimer);
    }
    parkTimer = setTimeout(() => (parked.value = true), HOVER_PARK_MS);
};

const onRowLeave = () => {
    if (parkTimer) {
        clearTimeout(parkTimer);
        parkTimer = null;
    }
    parked.value = false;
};

onBeforeUnmount(() => {
    if (parkTimer) {
        clearTimeout(parkTimer);
    }
});

/** 贡献者声明的行内 label 渲染器（kind 专属渲染知识归贡献者，行组件零 kind 知识） */
const labelRenderer = computed(() => getContributor(props.data.kind)?.labelRenderer);

// 下拉菜单展开中：保持操作按钮区渲染（锚点卸载会连带关闭 teleport 弹层）
const dropdownVisible = ref(false);

const menuItems = computed(() => resolveNodeMenu(props.data.kind));

/** 悬浮按钮仅展示 when/permission 通过的项（与右键菜单同一份判定） */
const visibleMenuItems = computed(() => menuItems.value.filter((item) => !item.isHide(ctxPayload.value)));

const ctxPayload = computed<TreeCommandCtx>(() => ({ node: props.data, tree }));

const onMenuCommand = (item: (typeof menuItems.value)[number]) => {
    item.onClickFunc?.(ctxPayload.value);
};
</script>

<style scoped>
/* 操作按钮 ↔ 信息区交叉淡入淡出（进快出更快，避免拖沓；reduced-motion 由全局兜底瞬时切换） */
.tree-row-swap-enter-active {
    transition: opacity 120ms var(--ease-out-quart);
}

.tree-row-swap-leave-active {
    transition: opacity 80ms var(--ease-out-quart);
}

.tree-row-swap-enter-from,
.tree-row-swap-leave-to {
    opacity: 0;
}
</style>
