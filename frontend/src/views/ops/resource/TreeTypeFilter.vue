<template>
    <el-dropdown trigger="click" size="small" @command="onCommand">
        <!--
            未筛选态只有一个漏斗图标；一旦生效就把「当前类型」写在控件上（图标 + 名称 + ▾）。
            窄侧栏里筛选态必须是自解释的可见值，而不是一个看不出差别的小图标。
        -->
        <button
            v-if="!selected"
            type="button"
            class="tree-type-trigger"
            :aria-label="$t('tag.treeTypeFilterTip')"
            :title="$t('tag.treeTypeFilterTip')"
        >
            <SvgIcon name="Filter" class="text-xs" />
        </button>
        <button v-else type="button" class="tree-type-chip" :title="$t('tag.treeTypeFilterTip')">
            <SvgIcon v-if="selected.icon" :name="selected.icon.name" :color="selected.icon.color" class="text-xs" />
            <span class="ml-0.5">{{ $t(selected.label) }}</span>
            <SvgIcon name="ArrowDown" class="text-[10px] ml-0.5 opacity-70" />
        </button>

        <template #dropdown>
            <el-dropdown-menu>
                <el-dropdown-item :command="ALL" :class="{ 'is-current': !modelValue }">
                    <span class="tree-type-item">
                        {{ $t('tag.allResource') }}
                        <SvgIcon v-if="!modelValue" name="Select" class="tree-type-check" />
                    </span>
                </el-dropdown-item>
                <el-dropdown-item v-for="option in options" :key="option.kind" :command="option.kind" :class="{ 'is-current': option.kind === modelValue }">
                    <span class="tree-type-item">
                        <SvgIcon v-if="option.icon" :name="option.icon.name" :color="option.icon.color" />
                        {{ $t(option.label) }}
                        <SvgIcon v-if="option.kind === modelValue" name="Select" class="tree-type-check" />
                    </span>
                </el-dropdown-item>
            </el-dropdown-menu>
        </template>
    </el-dropdown>
</template>

<script lang="ts" setup>
/**
 * 资源树「按类型筛选」控件
 *
 * 类型层级已从树里移除（标签下直接平铺资源），这里补回的是同一维度的**筛选**能力，
 * 对齐同类产品的做法（JumpServer 资产 tab、云资源中心的类型 facet）：类型是视图轴，不是树层级。
 *
 * 选项数据源是资源注册表本身（`getResourceConfigs()` 按 order 排好序 + 根贡献者声明的 kind）：
 * 新增一类资产只要注册自己的 ResourceConfig 与 Contributor，这里自动多一项，无需改动本组件。
 */
import { computed } from 'vue';

import SvgIcon from '@/components/svg-icon/index.vue';
import { deriveTypeOptions } from './tagResources';

/** 「全部资源」的 command 值（el-dropdown 的 command 不接受 null，用空串表达） */
const ALL = '';

const props = defineProps<{
    /** 当前生效的资源根节点 kind，null 表示不筛选 */
    modelValue?: string | null;
}>();

const emit = defineEmits<{ 'update:modelValue': [value: string | null] }>();


/** 可筛选的类型 = 注册了资源模块且声明了根贡献者的那些（无根贡献者的类型在树里不占一层） */
// 选项推导在 tagResources.deriveTypeOptions（纯函数，可脱离组件单测）：
// 新增一类资产只要注册自己的 ResourceConfig 与 Contributor，这里自动多一项
const options = computed(deriveTypeOptions);

const selected = computed(() => options.value.find((option) => option.kind === props.modelValue));

const onCommand = (kind: string) => {
    emit('update:modelValue', kind === ALL ? null : kind);
};
</script>

<style lang="scss" scoped>
/* 与查找入口同一套视觉：小尺寸、贴右上角、半透明避免与节点内容争视觉权重 */
.tree-type-trigger {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    color: var(--el-text-color-secondary);
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    opacity: 0.7;
    transition:
        opacity 0.15s ease,
        color 0.15s ease,
        border-color 0.15s ease;
}

.tree-type-trigger:hover {
    color: var(--el-color-primary);
    border-color: var(--el-color-primary-light-5);
    opacity: 1;
}

/* 生效态胶囊：主色描边 + 主色文字，一眼看出「现在只看某一类」 */
.tree-type-chip {
    display: flex;
    align-items: center;
    height: 20px;
    padding: 0 6px;
    font-size: 12px;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border: 1px solid var(--el-color-primary-light-5);
    border-radius: 10px;
}

.tree-type-item {
    display: inline-flex;
    align-items: center;
    gap: 4px;
}

/* 当前生效项打勾并着主色：重开菜单时必须能一眼看出「现在筛的是哪个」 */
.tree-type-check {
    margin-left: auto;
    color: var(--el-color-primary);
}

:deep(.el-dropdown-menu__item.is-current) {
    color: var(--el-color-primary);
    font-weight: 600;
}
</style>
