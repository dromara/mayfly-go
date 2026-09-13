<template>
    <div class="w-full! tag-tree-check">
        <el-input v-model="filterTag" @input="onFilterValChanged" clearable :placeholder="$t('tag.keywordFilterPlaceholder')" size="small" />
        <div class="mt-0.5 tag-tree-box" :class="heightClass" :style="boxStyle">
            <el-tree
                v-bind="$attrs"
                ref="tagTreeRef"
                :data="state.tags"
                :default-expanded-keys="checkedTags"
                :default-checked-keys="checkedTags"
                multiple
                :render-after-expand="true"
                show-checkbox
                check-strictly
                :node-key="$props.nodeKey"
                :props="{
                    label: 'codePath',
                    children: 'children',
                    disabled: 'disabled',
                } as Record<string, string>"
                @check="tagTreeNodeCheck"
                :filter-node-method="filterNode"
            >
                <template #default="{ data }">
                    <span>
                        <SvgIcon
                            :name="EnumValue.getEnumByValue(TagResourceTypeEnum, data.type)?.extra.icon"
                            :color="EnumValue.getEnumByValue(TagResourceTypeEnum, data.type)?.extra.iconColor"
                        />

                        <span class="text-[13px]! ml-1">
                            {{ data.name }}
                            <el-tag v-if="data.children !== null" size="small">{{ data.children.length }} </el-tag>
                        </span>
                    </span>
                </template>
            </el-tree>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { computed, ref, reactive, onMounted, type PropType } from 'vue';
import { tagApi } from '../tag/api';
import type { TagTree } from '../tag/types';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import EnumValue from '@/common/Enum';
import { isPrefixSubsequence } from '@/common/utils/string';

interface TagTreeData {
    codePath: string;
    name: string;
    type: number;
    children?: TagTreeData[] | null;
    disabled?: boolean;
}

interface TreeNode {
    parent: TreeNode | null;
    checked: boolean;
    data: TagTreeData;
    childNodes: TreeNode[];
}

/** el-tree 组件实例方法（仅包含本组件使用的） */
interface TreeInstance {
    getCheckedNodes: () => TagTreeData[];
    getNode: (key: string) => TreeNode | undefined;
    filter: (val: string) => void;
    getCheckedKeys: (leafOnly?: boolean) => string[];
    setChecked: (node: TreeNode, checked: boolean, deep: boolean) => void;
}

const props = defineProps({
    /** 高度模式：'fixed'=固定高度（height），'max'=最大高度（max-height，内容少时自适应） */
    heightMode: {
        type: String as PropType<'fixed' | 'max'>,
        default: 'max',
    },
    /** 高度值：fixed 模式下为固定高度，max 模式下为最大高度上限 */
    height: {
        type: [String, Number],
        default: '45vh',
    },
    /** max 模式下的最小高度（内容少时盒子不低于此高度） */
    minHeight: {
        type: [String, Number],
        default: 120,
    },
    tagType: {
        type: [Number, String, Array] as PropType<number | string | (number | string)[]>,
        default: TagResourceTypeEnum.Tag.value,
    },
    nodeKey: {
        type: String,
        default: 'codePath',
    },
});

const checkedTags = defineModel<string[]>('modelValue', {
    default: () => [],
});

/** 归一化高度为 CSS 值 */
const heightCss = computed(() => (typeof props.height === 'number' ? `${props.height}px` : String(props.height)));
const minHeightCss = computed(() => (typeof props.minHeight === 'number' ? `${props.minHeight}px` : String(props.minHeight)));

/** 高度模式对应的 CSS 类名 */
const heightClass = computed(() => ({
    'tag-tree-box--fixed': props.heightMode === 'fixed',
    'tag-tree-box--max': props.heightMode === 'max',
}));

/** 盒子样式（高度值通过 CSS 变量传入，避免内联 !important） */
const boxStyle = computed(() => ({
    '--tree-box-height': heightCss.value,
    '--tree-box-min-height': minHeightCss.value,
}));

const tagTreeRef = ref<TreeInstance | null>(null);
const filterTag = ref('');

const state = reactive({
    tags: [] as TagTree[],
});

onMounted(() => {
    search();
});

const search = async () => {
    let tagType: string | number | Array<number | string> = props.tagType;
    if (Array.isArray(props.tagType)) {
        tagType = props.tagType.join(',');
    }

    state.tags = await tagApi.getTagTrees.request({ type: tagType });

    setTimeout(() => {
        const checkedNodes = tagTreeRef.value?.getCheckedNodes() ?? [];
        // 禁用选中节点的所有父节点，不可选中
        for (let checkNodeData of checkedNodes) {
            disableParentNodes(tagTreeRef.value?.getNode(checkNodeData.codePath)?.parent ?? null);
        }
    }, 200);
};

const filterNode = (value: unknown, data: unknown) => {
    const d = data as TagTreeData;
    return !value || isPrefixSubsequence(value as string, d.codePath) || isPrefixSubsequence(value as string, d.name);
};

const onFilterValChanged = (val: string) => {
    tagTreeRef.value!.filter(val);
};

const tagTreeNodeCheck = (data: TagTreeData) => {
    const node = tagTreeRef.value?.getNode(data.codePath);
    if (!node) return;

    if (node.checked) {
        // 如果选中了子节点，则需要将父节点全部取消选中，并禁用父节点
        unCheckParentNodes(node.parent);
        disableParentNodes(node.parent);
    } else {
        // 如果取消了选中，则需要根据条件恢复父节点的选中状态
        disableParentNodes(node.parent, false);
    }

    // 更新绑定的值
    checkedTags.value = tagTreeRef.value?.getCheckedKeys(false) ?? [];
};

const unCheckParentNodes = (node: TreeNode | null) => {
    if (!node) {
        return;
    }
    tagTreeRef.value?.setChecked(node, false, false);
    unCheckParentNodes(node.parent);
};

/**
 * 禁用该节点以及所有父节点
 * @param node 节点
 * @param disable 是否禁用
 */
const disableParentNodes = (node: TreeNode | null, disable = true) => {
    if (!node) {
        return;
    }
    if (!disable) {
        // 恢复为非禁用状态时，若同层级存在一个选中状态或者禁用状态，则继续禁用 不恢复非禁用状态。
        for (let oneLevelNodes of node.childNodes) {
            if (oneLevelNodes.checked || oneLevelNodes.data.disabled) {
                return;
            }
        }
    }
    node.data.disabled = disable;
    disableParentNodes(node.parent, disable);
};
</script>
<style lang="scss" scoped>
.tag-tree-check {
    .tag-tree-box {
        border: 1px solid var(--el-border-color);
        overflow-y: auto;
        // 背景色挂在固定高度的盒子上，而非树上，确保展开/收起时背景始终均匀铺满
        background: rgba(255, 255, 255, 0.22);
    }

    // 固定高度模式：盒子高度锁定，不随内容展开/收起变化
    .tag-tree-box--fixed {
        height: var(--tree-box-height);
    }

    // 最大高度模式：内容少时自适应，超出时滚动
    .tag-tree-box--max {
        min-height: var(--tree-box-min-height);
        max-height: var(--tree-box-height);
    }

    .el-tree {
        min-width: 100%;
        // 横向滚动生效
        display: inline-block;
        // 树本身透明，背景由父容器 .tag-tree-box 提供
        background: transparent !important;
    }
}
</style>
