<template>
    <div class="!w-full tag-tree-check">
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
                        value: $props.nodeKey,
                        label: 'codePath',
                        children: 'children',
                        disabled: 'disabled',
                    }"
                    @check="tagTreeNodeCheck"
                    :filter-node-method="filterNode"
                >
                    <template #default="{ data }">
                        <span>
                            <SvgIcon
                                :name="EnumValue.getEnumByValue(TagResourceTypeEnum, data.type)?.extra.icon"
                                :color="EnumValue.getEnumByValue(TagResourceTypeEnum, data.type)?.extra.iconColor"
                            />

                            <span class="!text-[13px] ml-1">
                                {{ data.name }}
                                <span style="color: #3c8dbc">【</span>
                                {{ data.code }}
                                <span style="color: #3c8dbc">】</span>
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
import { ref, reactive, onMounted } from 'vue';
import { tagApi } from '../tag/api';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import EnumValue from '@/common/Enum';
import { isPrefixSubsequence } from '@/common/utils/string';

const props = defineProps({
    height: {
        type: [String, Number],
        default: 'calc(100vh - 330px)',
    },
    tagType: {
        type: [Number, Array<Number>, String, Array<String>],
        default: TagResourceTypeEnum.Tag.value,
    },
    nodeKey: {
        type: String,
        default: 'codePath',
    },
});

const checkedTags = defineModel<Array<any>>('modelValue', {
    default: () => [],
});

const tagTreeRef: any = ref(null);
const filterTag = ref('');

const state = reactive({
    tags: [],
});

onMounted(() => {
    search();
});

const search = async () => {
    let tagType: any = props.tagType;
    if (Array.isArray(props.tagType)) {
        tagType = props.tagType.join(',');
    }

    state.tags = await tagApi.getTagTrees.request({ type: tagType });

    setTimeout(() => {
        const checkedNodes = tagTreeRef.value.getCheckedNodes();
        console.log('check nodes: ', checkedNodes);
        // 禁用选中节点的所有父节点，不可选中
        for (let checkNodeData of checkedNodes) {
            disableParentNodes(tagTreeRef.value.getNode(checkNodeData.codePath).parent);
        }
    }, 200);
};

const filterNode = (value: string, data: any) => {
    return !value || isPrefixSubsequence(value, data.codePath) || isPrefixSubsequence(value, data.name);
};

const onFilterValChanged = (val: string) => {
    tagTreeRef.value!.filter(val);
};

const tagTreeNodeCheck = (data: any) => {
    const node = tagTreeRef.value.getNode(data.codePath);
    console.log('check node: ', node);

    if (node.checked) {
        // 如果选中了子节点，则需要将父节点全部取消选中，并禁用父节点
        unCheckParentNodes(node.parent);
        disableParentNodes(node.parent);
    } else {
        // 如果取消了选中，则需要根据条件恢复父节点的选中状态
        disableParentNodes(node.parent, false);
    }

    // 更新绑定的值
    checkedTags.value = tagTreeRef.value.getCheckedKeys(false);
};

const unCheckParentNodes = (node: any) => {
    if (!node) {
        return;
    }
    tagTreeRef.value.setChecked(node, false, false);
    unCheckParentNodes(node.parent);
};

/**
 * 禁用该节点以及所有父节点
 * @param node 节点
 * @param disable 是否禁用
 */
const disableParentNodes = (node: any, disable = true) => {
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
