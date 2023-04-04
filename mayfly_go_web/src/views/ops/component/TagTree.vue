<template>
    <div class="instances-box layout-aside">
        <el-row type="flex" justify="space-between">
            <el-col :span="24" class="el-scrollbar flex-auto" style="overflow: auto">
                <el-input v-model="filterText" placeholder="输入关键字->搜索已展开节点信息" clearable size="small" class="mb5" />

                <el-tree ref="treeRef" :style="{ maxHeight: state.height, height: state.height, overflow: 'auto' }"
                    :highlight-current="true" :indent="7" :load="loadNode" :props="treeProps" lazy node-key="key"
                    :expand-on-click-node="true" :filter-node-method="filterNode" @node-click="treeNodeClick"
                    @node-expand="treeNodeClick">
                    <template #default="{ node, data }">
                        <span>
                            <span v-if="data.type == TagTreeNode.TagPath">
                                <tag-info :tag-path="data.label" />
                            </span>

                            <slot v-else :node="node" :data="data" name="prefix"></slot>

                            <span class="ml3">
                                <slot name="label" :data="data"> {{ data.label }}</slot>
                            </span>
                          
                            <span class="ml3">
                                <slot name="option" :data="data"></slot>
                            </span>
                          
                        </span>
                    </template>
                </el-tree>
            </el-col>
        </el-row>
    </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref, watch, toRefs } from 'vue';
import { TagTreeNode } from './tag';
import TagInfo from './TagInfo.vue';

const props = defineProps({
    height: {
        type: [Number, String],
        default: 0
    },
    load: {
        type: Function,
        required: true,
    }
})

const treeProps = {
    label: 'name',
    children: 'zones',
    isLeaf: 'isLeaf',
}

const emit = defineEmits(['nodeClick'])
const treeRef: any = ref(null)

const state = reactive({
    height: 600 as any,
    filterText: '',
    opend: {},
})
const { filterText } = toRefs(state)

onMounted(async () => {
    if (!props.height) {
        state.height = window.innerHeight - 147 + 'px';
    } else {
        state.height = props.height;
    }
})

watch(filterText, (val) => {
    treeRef.value?.filter(val)
})

const filterNode = (value: string, data: any) => {
    if (!value) return true
    return data.label.includes(value)
}

/**
* 加载树节点
* @param { Object } node
* @param { Object } resolve
*/
const loadNode = async (node: any, resolve: any) => {
    if (typeof resolve !== 'function') {
        return;
    }
    let nodes = []
    try {
        nodes = await props.load(node)
    } catch (e: any) {
        console.error(e);
    }
    return resolve(nodes)
};

const treeNodeClick = (data: any) => {
    emit('nodeClick', data);
}

const reloadNode = (nodeKey: any) => {
    let node = getNode(nodeKey);
    node.loaded = false;
    node.expand();
}

const getNode = (nodeKey: any) => {
    let node = treeRef.value.getNode(nodeKey);
    if (!node) {
        throw new Error('未找到节点: ' + nodeKey);
    }
    return node;
}

defineExpose({
    reloadNode,
})
</script>

<style lang="scss">
.instances-box {
    overflow: 'auto';

    .el-tree {
        display: inline-block;
        min-width: 100%;
    }
}
</style>