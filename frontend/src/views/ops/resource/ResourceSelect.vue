<template>
    <el-tree-select
        v-bind="$attrs"
        ref="treeRef"
        popper-class="resource-select-tree-popper"
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
            <component v-if="contributorOf(data)?.renderer" :is="contributorOf(data)!.renderer" :data="data" :show-actions="isShowActions" />
            <TreeNodeRow v-else :data="data" :show-actions="isShowActions" />
        </template>
    </el-tree-select>
</template>

<script lang="ts" setup>
import { reactive, toRefs, useTemplateRef, watch } from 'vue';

import { isPrefixSubsequence } from '@/common/utils/string';

import { getContributor, isNodeSelectable, resolveHasChildren, toNodeMatcher, TreeApiKey, type NodeMatcher, type TreeApi, type TreeNode, type TreeNodeData } from './tree';
import { loadResourceTags } from './resource';
import TreeNodeRow from './tree/TreeNodeRow.vue';
import { provide } from 'vue';

/**
 * 资源选择下拉树：复用 ops 资源树的贡献者数据链路（tag 骨架 + 类型分组 + 资源懒加载）。
 * 选择粒度由 selectable prop 或贡献者 selectable 声明单源判定：
 * 仅命中选择粒度的节点才外抛 change 并关闭下拉（el-tree-select 任意节点点击都触发 change，
 * 守卫收口在此，消费方无需逐处 kind 判存）。
 */
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
    // 叶子节点 kind 数组，匹配到的节点强制标记为叶子节点（不展开）：选择粒度之下禁止继续钻取
    leafKinds: {
        type: Array as () => string[],
        default: () => [],
    },
    // 选择粒度（kind 清单或谓词函数）；缺省回退贡献者的 selectable 声明
    selectable: {
        type: [Array, Function] as unknown as () => NodeMatcher | undefined,
        default: undefined,
    },
    // 节点转换函数，在节点加载后调用，可用于动态标记叶子等属性
    transformNode: {
        type: Function as unknown as () => ((node: TreeNodeData) => TreeNodeData) | null,
        default: null,
    },
});

// 渲染器的 showActions 由 el-tree-select 节点默认插槽透传

// 选择器场景无树容器：提供空 TreeApi 桩，避免节点渲染器 inject 落空
provide(TreeApiKey, { locate: async () => {}, refresh: () => {}, getNode: () => undefined } as TreeApi);

const treeProps = {
    label: 'label',
    children: 'children',
    isLeaf: 'isLeaf',
};

const emit = defineEmits(['change']);
const treeRef = useTemplateRef<{ filter: (val: string) => void; getNode: (key: string | number) => { data: TreeNodeData } | undefined; blur: () => void }>('treeRef');

const modelValue = defineModel<string | number>('modelValue');

const state = reactive({
    filterText: '',
});
const { filterText } = toRefs(state);

watch(filterText, (val) => {
    treeRef.value?.filter(val);
});

const contributorOf = (data: TreeNodeData) => getContributor(data.kind);

// 与主资源树同一过滤语义（前缀子序列），避免同一产品两套搜索规则（主树能搜到 a-c，选择器搜不到）
const filterNode = (value: string, data: TreeNodeData) => {
    if (!value) return true;
    return isPrefixSubsequence(value, data.label);
};

/** 选择粒度判定：显式 selectable 优先，缺省走贡献者 selectable 声明（单源协议） */
const matchSelectable = (node: TreeNodeData): boolean => (props.selectable ? toNodeMatcher(props.selectable)(node) : isNodeSelectable(node));

/** 未声明可展开能力的 kind 一律视为叶子；带预构子树的节点（如标签骨架）可展开 */
const isLeafNode = (node: TreeNodeData) => {
    if (node.children?.length) {
        return false;
    }
    if (props.leafKinds.includes(node.kind)) {
        return true;
    }
    const contributor = getContributor(node.kind);
    return !(contributor && resolveHasChildren(contributor, node as TreeNode));
};

/**
 * 加载树节点（懒加载分发到对应贡献者）
 */
const loadNode = async (node: { level: number; data: TreeNodeData }, resolve: (data: TreeNodeData[]) => void) => {
    if (typeof resolve !== 'function') {
        return;
    }

    let nodes: TreeNodeData[] = [];
    try {
        if (node.level == 0) {
            nodes = await loadResourceTags([props.resourceType]);
        } else if (node.data.children?.length) {
            // 预构子树（如标签骨架→类型分组）：内存中直接透传，不走贡献者懒加载
            nodes = node.data.children;
        } else if (props.load) {
            nodes = await props.load(node);
        } else {
            nodes = (await getContributor(node.data.kind)?.loadChildren?.(node.data as TreeNode)) ?? [];
        }
    } catch (e: unknown) {
        console.error(e);
    }

    // 标记叶子（leafKinds 强制叶子 / 贡献者无展开能力默认叶子）；params 兜底非空（对齐 useLazyTree 契约，
    // el-tree-select 点击任意节点都会触发 change，下游 changeNode 依赖 params 可安全取参）
    nodes.forEach((n) => {
        n.params = n.params ?? {};
        (n as TreeNode & { isLeaf?: boolean }).isLeaf = isLeafNode(n);
    });

    // 如果提供了节点转换函数，调用它来处理每个节点
    if (props.transformNode) {
        nodes = nodes.map((n) => props.transformNode!(n));
    }

    resolve(nodes);
};

const changeNode = (val: string | number) => {
    const node = treeRef.value?.getNode(val);
    // 仅命中选择粒度的节点才是有效选择：标签/分组/中间层节点的点击仅用于展开，
    // 不外抛 change 也不关闭下拉（守卫单源收口，消费方 change 回调可直取 params）
    if (!node || !matchSelectable(node.data)) {
        return;
    }
    emit('change', node.data);

    // 选择后关闭下拉框
    setTimeout(() => {
        if (treeRef.value) {
            treeRef.value?.blur();
        }
    }, 100);
};
</script>

<style lang="scss" scoped></style>

<style lang="scss">
// 下拉面板 teleport 到 body，scoped 样式不可达：经 popper-class 全局命中本组件的所有弹层
.resource-select-tree-popper {
    // el-tree-select 会在行外包一层 li.el-select-dropdown__item（默认 height:20px + display:list-item +
    // overflow:hidden），行内容被顶对齐压低 → 展开箭头与文字错位 2px。改为 flex 居中、高度自适应，
    // 对齐不再依赖任何像素常量（箭头与行内容各自在同一中心线上）
    .el-select-dropdown__item {
        height: auto;
        display: flex;
        align-items: center;
        overflow: visible;
    }

    // 节点行高对齐行组件（默认 26px 偏紧，与树行视觉高度不一致）
    .el-tree-node__content {
        height: 28px;
    }
}
</style>
