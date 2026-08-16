<template>
    <div class="w-full! tag-tree-check">
        <el-input v-model="filterTag" @input="onFilterValChanged" clearable :placeholder="$t('tag.keywordFilterPlaceholder')" size="small" />
        <div class="mt-0.5" style="border: 1px solid var(--el-border-color)">
            <el-scrollbar :style="{ height: props.height }">
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
            </el-scrollbar>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted, type PropType } from 'vue';
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
    height: {
        type: [String, Number],
        default: 'calc(100vh - 330px)',
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
    .el-tree {
        min-width: 100%;
        // 横向滚动生效
        display: inline-block;
    }
}
</style>
