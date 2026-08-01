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
        <template #label="{ label, value }">
            <slot name="label" :label="label" :value="value" />
        </template>

        <template #default="{ node, data }">
            <component v-if="data.nodeComponent" :is="data.nodeComponent" :node="node" :data="data" />
            <BaseTreeNode v-else :node="node" :data="data" />
        </template>
    </el-tree-select>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref, toRefs, watch, provide, PropType, useTemplateRef } from 'vue';

import { NodeType, TagTreeNode } from '@/views/ops/component/tag';
import BaseTreeNode from '@/views/ops/resource/BaseTreeNode.vue';
import { loadResourceTags, IsShowActionsKey, LeafNodeTypesKey } from './resource';

const props = defineProps({
    resourceType: {
        type: [Number, String],
        required: true,
    },
    load: {
        type: Function,
        required: false,
    },
    // 是否显示操作按钮，默认 false（选择器模式）
    isShowActions: {
        type: Boolean,
        default: false,
    },
    // 叶子节点类型数组，匹配到的节点将标记为叶子节点
    leafNodeTypes: {
        type: Array as () => NodeType[],
        default: () => [],
    },
    // 节点转换函数，在节点加载后调用，可用于动态设置 isLeaf 等属性
    transformNode: {
        type: Function as PropType<(node: TagTreeNode) => TagTreeNode>,
        default: null,
    },
});

// 注入 isShowActions 到子组件
provide(IsShowActionsKey, props.isShowActions);

// 注入叶子节点类型到子组件
provide(LeafNodeTypesKey, props.leafNodeTypes);

const treeProps = {
    label: 'name',
    children: 'zones',
    isLeaf: 'isLeaf',
};

const emit = defineEmits(['change']);
const treeRef = useTemplateRef<{ filter: (val: string) => void; getNode: (key: string | number) => { data: TagTreeNode } | undefined; blur: () => void }>('treeRef');

const modelValue = defineModel<string | number>('modelValue');

const state = reactive({
    height: 600,
    filterText: '',
    opend: {},
});
const { filterText } = toRefs(state);

onMounted(async () => {});

watch(filterText, (val) => {
    treeRef.value?.filter(val);
});

const filterNode = (value: string, data: TagTreeNode) => {
    if (!value) return true;
    return data.label.includes(value);
};

/**
 * 加载树节点
 * @param { Object } node
 * @param { Object } resolve
 */
const loadNode = async (node: { level: number; data: TagTreeNode }, resolve: (data: TagTreeNode[]) => void) => {
    if (typeof resolve !== 'function') {
        return;
    }

    let nodes = [];
    try {
        if (node.level == 0) {
            nodes = await loadResourceTags([props.resourceType], null);
        } else if (props.load) {
            nodes = await props.load(node);
        } else {
            nodes = await node.data.loadChildren();
        }
    } catch (e: unknown) {
        console.error(e);
    }

    // 如果配置了叶子节点类型，检查并标记
    if (props.leafNodeTypes && props.leafNodeTypes.length > 0) {
        nodes.forEach((n: TagTreeNode) => {
            if (n.type && props.leafNodeTypes.some((type: NodeType) => type.value === n.type.value)) {
                n.isLeaf = true;
            }
        });
    }

    // 如果提供了节点转换函数，调用它来处理每个节点
    if (props.transformNode) {
        nodes = nodes.map((n: TagTreeNode) => props.transformNode(n));
    }

    return resolve(nodes);
};

const getNode = (nodeKey: string | number) => {
    let node = treeRef.value?.getNode(nodeKey);
    if (!node) {
        throw new Error('未找到节点: ' + nodeKey);
    }
    return node;
};

const changeNode = (val: string | number) => {
    // 触发改变事件，并传递节点数据
    emit('change', getNode(val)?.data);
    
    // 选择后关闭下拉框
    setTimeout(() => {
        if (treeRef.value) {
            treeRef.value?.blur();
        }
    }, 100);
};
</script>

<style lang="scss" scoped></style>
