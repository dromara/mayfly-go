<template>
    <el-tree-select
        v-bind="$attrs"
        ref="treeRef"
        :highlight-current="true"
        :indent="10"
        :load="loadNode"
        :props="treeProps"
        lazy
        node-key="key"
        :expand-on-click-node="true"
        filterable
        :filter-node-method="filterNode"
        v-model="modelValue"
        @change="changeNode"
    >
        <template #prefix="{ node, data }">
            <slot name="iconPrefix" :node="node" :data="data" />
        </template>
        <template #default="{ node, data }">
            <span>
                <span v-if="data.type.value == TagTreeNode.TagPath">
                    <tag-info :tag-path="data.label" />
                </span>

                <slot v-else :node="node" :data="data" name="prefix"></slot>

                <span class="ml3" :title="data.labelRemark">
                    <slot name="label" :data="data"> {{ data.label }}</slot>
                </span>

                <slot :node="node" :data="data" name="suffix"></slot>
            </span>
        </template>
    </el-tree-select>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref, toRefs, watch } from 'vue';
import { NodeType, TagTreeNode } from './tag';
import TagInfo from './TagInfo.vue';
import { tagApi } from '../tag/api';

const props = defineProps({
    resourceType: {
        type: [Number],
        required: true,
    },
    tagPathNodeType: {
        type: [NodeType],
        required: true,
    },
    load: {
        type: Function,
        required: false,
    },
});

const treeProps = {
    label: 'name',
    children: 'zones',
    isLeaf: 'isLeaf',
};

const emit = defineEmits(['change']);
const treeRef: any = ref(null);

const modelValue = defineModel('modelValue');

const state = reactive({
    height: 600 as any,
    filterText: '',
    opend: {},
});
const { filterText } = toRefs(state);

onMounted(async () => {});

watch(filterText, (val) => {
    treeRef.value?.filter(val);
});

const filterNode = (value: string, data: any) => {
    if (!value) return true;
    return data.label.includes(value);
};

/**
 * 加载标签树节点
 */
const loadTags = async () => {
    const tags = await tagApi.getResourceTagPaths.request({ resourceType: props.resourceType });
    const tagNodes = [];
    for (let tagPath of tags) {
        tagNodes.push(new TagTreeNode(tagPath, tagPath, props.tagPathNodeType));
    }
    return tagNodes;
};

/**
 * 加载树节点
 * @param { Object } node
 * @param { Object } resolve
 */
const loadNode = async (node: any, resolve: any) => {
    if (typeof resolve !== 'function') {
        return;
    }
    let nodes = [];
    try {
        if (node.level == 0) {
            nodes = await loadTags();
        } else if (props.load) {
            nodes = await props.load(node);
        } else {
            nodes = await node.data.loadChildren();
        }
    } catch (e: any) {
        console.error(e);
    }
    return resolve(nodes);
};

const getNode = (nodeKey: any) => {
    let node = treeRef.value.getNode(nodeKey);
    if (!node) {
        throw new Error('未找到节点: ' + nodeKey);
    }
    return node;
};

const changeNode = (val: any) => {
    // 触发改变时间，并传递节点数据
    emit('change', getNode(val)?.data);
};
</script>

<style lang="scss" scoped></style>
