<template>
    <div class="card pd5">
        <el-input v-model="filterText" placeholder="输入关键字->搜索已展开节点信息" clearable size="small" class="mb5 w100" />
        <el-scrollbar class="tag-tree">
            <el-tree
                ref="treeRef"
                :highlight-current="true"
                :indent="10"
                :load="loadNode"
                :props="treeProps"
                lazy
                node-key="key"
                :expand-on-click-node="false"
                :filter-node-method="filterNode"
                @node-click="treeNodeClick"
                @node-expand="treeNodeClick"
                @node-contextmenu="nodeContextmenu"
                :default-expanded-keys="props.defaultExpandedKeys"
            >
                <template #default="{ node, data }">
                    <span :id="node.key" @dblclick="treeNodeDblclick(data)" :class="data.type.nodeDblclickFunc ? 'none-select' : ''">
                        <span v-if="data.type.value == TagTreeNode.TagPath">
                            <tag-info :tag-path="data.label" />
                        </span>

                        <slot v-else :node="node" :data="data" name="prefix"></slot>

                        <span class="ml3" :title="data.labelRemark">
                            <slot name="label" :data="data" v-if="!data.disabled"> {{ data.label }}</slot>
                            <!-- 禁用状态 -->
                            <slot name="disabledLabel" :data="data" v-else>
                                <el-link type="danger" disabled :underline="false">
                                    {{ `${data.label}` }}
                                </el-link>
                            </slot>
                        </span>

                        <span class="label-suffix">
                            <slot :node="node" :data="data" name="suffix"></slot>
                        </span>
                    </span>
                </template>
            </el-tree>

            <contextmenu :dropdown="state.dropdown" :items="state.contextmenuItems" ref="contextmenuRef" @currentContextmenuClick="onCurrentContextmenuClick" />
        </el-scrollbar>
    </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref, watch, toRefs, nextTick } from 'vue';
import { NodeType, TagTreeNode } from './tag';
import TagInfo from './TagInfo.vue';
import { Contextmenu } from '@/components/contextmenu';
import { tagApi } from '../tag/api';
import { isPrefixSubsequence } from '@/common/utils/string';

const props = defineProps({
    resourceType: {
        type: [Number],
        required: true,
    },
    defaultExpandedKeys: {
        type: [Array],
    },
    tagPathNodeType: {
        type: [NodeType],
        required: true,
    },
    load: {
        type: Function,
        required: false,
    },
    loadContextmenuItems: {
        type: Function,
        required: false,
    },
});

const treeProps = {
    label: 'name',
    children: 'zones',
    isLeaf: 'isLeaf',
};

const emit = defineEmits(['nodeClick', 'currentContextmenuClick']);
const treeRef: any = ref(null);
const contextmenuRef = ref();

const state = reactive({
    height: 600 as any,
    filterText: '',
    dropdown: {
        x: 0,
        y: 0,
    },
    contextmenuItems: [],
    opend: {},
});
const { filterText } = toRefs(state);

onMounted(async () => {});

watch(filterText, (val) => {
    treeRef.value?.filter(val);
});

const filterNode = (value: string, data: any) => {
    return !value || isPrefixSubsequence(value, data.label);
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
const loadNode = async (node: any, resolve: (data: any) => void, reject: () => void) => {
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
        // 调用 reject 以保持节点状态，并允许远程加载继续。
        return reject();
    }
    return resolve(nodes);
};

const treeNodeClick = (data: any) => {
    if (!data.disabled && !data.type.nodeDblclickFunc && data.type.nodeClickFunc) {
        emit('nodeClick', data);
        data.type.nodeClickFunc(data);
    }
    // 关闭可能存在的右击菜单
    contextmenuRef.value.closeContextmenu();
};

// 树节点双击事件
const treeNodeDblclick = (data: any) => {
    // emit('nodeDblick', data);
    if (!data.disabled && data.type.nodeDblclickFunc) {
        data.type.nodeDblclickFunc(data);
    }
    // 关闭可能存在的右击菜单
    contextmenuRef.value.closeContextmenu();
};

// 树节点右击事件
const nodeContextmenu = (event: any, data: any) => {
    if (data.disabled) {
        return;
    }

    // 加载当前节点是否需要显示右击菜单
    let items = data.type.contextMenuItems;
    if (!items || items.length == 0) {
        if (props.loadContextmenuItems) {
            items = props.loadContextmenuItems(data);
        }
    }
    if (!items) {
        return;
    }
    state.contextmenuItems = items;
    const { clientX, clientY } = event;
    state.dropdown.x = clientX;
    state.dropdown.y = clientY;
    contextmenuRef.value.openContextmenu(data);
};

const onCurrentContextmenuClick = (clickData: any) => {
    emit('currentContextmenuClick', clickData);
};

const reloadNode = (nodeKey: any) => {
    let node = getNode(nodeKey);
    node.loaded = false;
    node.expand();
};

const getNode = (nodeKey: any) => {
    let node = treeRef.value.getNode(nodeKey);
    if (!node) {
        throw new Error('未找到节点: ' + nodeKey);
    }
    return node;
};

const setCurrentKey = (nodeKey: any) => {
    treeRef.value.setCurrentKey(nodeKey);

    // 通过Id获取到对应的dom元素
    const node = document.getElementById(nodeKey);
    if (node) {
        setTimeout(() => {
            nextTick(() => {
                // 通过scrollIntoView方法将对应的dom元素定位到可见区域 【block: 'center'】这个属性是在垂直方向居中显示
                node.scrollIntoView({ block: 'center' });
            });
        }, 100);
    }
};

defineExpose({
    reloadNode,
    getNode,
    setCurrentKey,
});
</script>

<style lang="scss" scoped>
.tag-tree {
    height: calc(100vh - 148px);

    .el-tree {
        display: inline-block;
        min-width: 100%;
    }

    .label-suffix {
        position: absolute;
        right: 10px;
        color: #c4c9c4;
        font-size: 10px;
        margin-top: 2px;
    }
}
</style>
