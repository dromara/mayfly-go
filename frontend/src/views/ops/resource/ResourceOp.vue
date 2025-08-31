<template>
    <div class="h-full">
        <ResourceOpPanel @resize="onResizeOpPanel">
            <template #left>
                <el-card class="h-full flex tag-tree-card" body-class="!p-0 flex flex-col w-full">
                    <div class="tag-tree-header flex flex-row justify-between items-center">
                        <el-input v-model="filterText" :placeholder="$t('tag.tagFilterPlaceholder')" clearable size="small" class="tag-tree-search w-full">
                            <template #prefix>
                                <SvgIcon class="tag-tree-search-icon" name="search" />
                            </template>
                        </el-input>

                        <div class="ml-1" v-if="Object.keys(resourceComponents).length > 1">
                            <el-dropdown placement="bottom-start" @command="changeResourceOp">
                                <el-button type="primary" link plain><SvgIcon name="Switch" /> </el-button>

                                <template #dropdown>
                                    <el-dropdown-menu>
                                        <el-dropdown-item
                                            :command="{ name }"
                                            v-for="(compConf, name) in resourceComponents"
                                            :disabled="name == activeResourceComp"
                                        >
                                            <SvgIcon v-if="compConf.icon" :name="compConf.icon.name" :color="compConf.icon.color" />
                                            <div class="ml-1">{{ $t(name) }}</div>
                                        </el-dropdown-item>
                                    </el-dropdown-menu>
                                </template>
                            </el-dropdown>
                        </div>
                    </div>

                    <el-scrollbar>
                        <el-tree
                            class="min-w-full inline-block"
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
                            :default-expanded-keys="state.defaultExpandedKeys"
                        >
                            <template #default="{ node, data }">
                                <component v-if="data.nodeComponent" :is="data.nodeComponent" :node="node" :data="data" />
                                <BaseTreeNode v-else :node="node" :data="data" />
                            </template>
                        </el-tree>
                    </el-scrollbar>
                </el-card>
            </template>

            <template #right>
                <el-card class="h-full" body-class=" h-full !p-1 flex flex-col flex-1">
                    <transition name="slide-x" mode="out-in">
                        <keep-alive>
                            <component :is="resourceComponents[activeResourceComp]?.component" :key="activeResourceComp" @init="initResourceComp" />
                        </keep-alive>
                    </transition>
                </el-card>
            </template>
        </ResourceOpPanel>
    </div>
</template>

<script lang="ts" setup>
import { markRaw, nextTick, provide, reactive, ref, toRefs, useTemplateRef, watch } from 'vue';

import { isPrefixSubsequence } from '@/common/utils/string';
import SvgIcon from '@/components/svgIcon/index.vue';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import EnumValue from '@/common/Enum';
import { getResourceNodeType, getResourceTypes, ResourceOpCtxKey } from './resource';
import BaseTreeNode from './BaseTreeNode.vue';
import { tagApi } from '@/views/ops/tag/api';
import ResourceOpPanel from '@/views/ops/component/ResourceOpPanel.vue';
import { TagTreeNode, ResourceComponentConfig, ResourceOpCtx } from '@/views/ops/component/tag';
import { useI18n } from 'vue-i18n';
import { useAutoOpenResource } from '@/store/autoOpenResource';
import { storeToRefs } from 'pinia';

const props = defineProps({
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

const autoOpenResourceStore = useAutoOpenResource();
const { autoOpenResource } = storeToRefs(autoOpenResourceStore);

const { t } = useI18n();

const emit = defineEmits(['nodeClick', 'currentContextmenuClick']);

const treeRef: any = useTemplateRef('treeRef');

// 存储所有注册的资源组件引用
const resourceComponents = ref<Record<string, ResourceComponentConfig>>({});
// 当前激活的资源组件
const activeResourceComp = ref<string>('');

const resourceComponentRefs = ref<Record<string, any>>({});

// :ref="(el: any) => setResourceComponentRefs(activeResourceComp, el)"
const setResourceComponentRefs = async (name: string, ref: any) => {
    if (!name || !ref) {
        return;
    }
    if (resourceComponentRefs.value[name]) {
        return;
    }
    resourceComponentRefs.value[name] = ref;
};

const state = reactive({
    defaultExpandedKeys: [] as string[],
    filterText: '',
});

const { filterText } = toRefs(state);

watch(filterText, (val) => {
    treeRef.value?.filter(val);
});

watch(
    () => autoOpenResource.value.codePath,
    (autoOpenCodePath: any) => {
        if (!autoOpenCodePath) {
            return;
        }

        const expandedKeys: string[] = [];
        let currentTagPath = '';
        const parts = autoOpenCodePath.split('/'); // 切分字符串并保留数字和对应的值部分
        let addResouceType = false;
        for (let part of parts) {
            if (!part) {
                continue;
            }
            let [key, value] = part.split('|'); // 分割数字和值部分
            // 如果不存在第二个参数，则说明为标签类型
            if (!value) {
                const tagPath = key + '/';
                currentTagPath = currentTagPath + tagPath;
                expandedKeys.push(currentTagPath);
                continue;
            }
            if (!addResouceType) {
                expandedKeys.push(currentTagPath + '-' + key);
                expandedKeys.push(value);
                addResouceType = true;
            } else {
                expandedKeys.push(value);
            }
        }

        state.defaultExpandedKeys = expandedKeys;
        autoOpenResourceStore.setCodePath('');
        setTimeout(() => {
            setCurrentKey(expandedKeys[expandedKeys.length - 1]);
        }, 500);
    },
    { immediate: true }
);

const filterNode = (value: string, data: any) => {
    return !value || isPrefixSubsequence(value, data.label);
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

let lastNodeClickTime = 0;

const treeNodeClick = async (data: any, node: any) => {
    const currentClickNodeTime = Date.now();
    if (currentClickNodeTime - lastNodeClickTime < 300) {
        treeNodeDblclick(data, node);
        return;
    }
    lastNodeClickTime = currentClickNodeTime;

    if (!data.disabled && !data.type.nodeDblclickFunc && data.type.nodeClickFunc) {
        emit('nodeClick', data);
        await data.type.nodeClickFunc(data);
    }
};

// 树节点双击事件
const treeNodeDblclick = (data: any, node: any) => {
    if (node.expanded) {
        node.collapse();
    } else {
        node.expand();
    }

    if (!data.disabled && data.type.nodeDblclickFunc) {
        data.type.nodeDblclickFunc(data);
    }
};

// 初始化资源组件ref
const initResourceComp = (val: any) => {
    if (!val.ref || resourceComponentRefs.value[val.name]) {
        return;
    }
    resourceComponentRefs.value[val.name] = val.ref;
};

const addResourceComponent = async (componentConf: ResourceComponentConfig) => {
    console.log(componentConf);
    const compName = componentConf.name;

    if (!resourceComponents.value[compName]) {
        // 使用 markRaw 标记组件，防止其被变成响应式对象
        resourceComponents.value[compName] = {
            ...componentConf,
            component: markRaw(componentConf.component),
        };
    }

    activeResourceComp.value = compName;

    // 使用一个 Promise 来确保组件引用已经被设置
    return new Promise((resolve) => {
        const checkRef = () => {
            if (resourceComponentRefs.value[compName]) {
                resolve(resourceComponentRefs.value[compName]);
            } else {
                // 如果引用还没有设置，稍后再检查
                setTimeout(checkRef, 10);
            }
        };
        // 先等待 nextTick 确保 DOM 更新
        nextTick().then(() => {
            checkRef();
        });
    });
};

const changeResourceOp = (data: any) => {
    activeResourceComp.value = data.name;
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

const onResizeOpPanel = () => {
    for (let name in resourceComponentRefs.value) {
        resourceComponentRefs.value[name]?.onResize?.();
    }
};

/**
 * 加载相关标签树节点
 */
const loadTags = async () => {
    const tags = await tagApi.getTagTrees.request({
        type: getResourceTypes().join(','),
    });
    const tagNodes = [];
    for (let tag of tags) {
        const tagNode = processTagNode(tag);
        tagNodes.push(tagNode);
    }
    return tagNodes;
};

const processTagNode = (tag: any): TagTreeNode => {
    const tagNode = new TagTreeNode(tag.codePath, tag.name, tag.type);

    if (!tag.children || !Array.isArray(tag.children) || tag.children.length == 0) {
        return tagNode;
    }

    // 子节点还是tag类型，则直接默认加载children即可
    if (tag.children[0].type == TagResourceTypeEnum.Tag.value) {
        tagNode.loadChildren = async () => {
            const childNodes = [];
            for (let child of tag.children) {
                const childNode = processTagNode(child);
                childNodes.push(childNode);
            }
            return childNodes;
        };
        return tagNode;
    }

    // 创建中间节点， 按类型分组
    const type2Tags = new Map<number, any>();
    tag.children.forEach((child: any) => {
        if (!type2Tags.has(child.type)) {
            type2Tags.set(child.type, [child]);
            return;
        }
        type2Tags.get(child.type).push(child);
    });

    tagNode.loadChildren = async () => {
        const childNodes = [];

        for (let [type, children] of type2Tags) {
            // 创建中间节点
            const typeEnum = EnumValue.getEnumByValue(TagResourceTypeEnum, type);
            const intermediateNode = new TagTreeNode(`${tag.codePath}-${type}`, t(typeEnum?.label || '未知'), getResourceNodeType(type))
                .withIcon({
                    name: typeEnum?.extra.icon,
                    color: typeEnum?.extra.iconColor,
                })
                .withIsLeaf(false)
                .withParams({ resourceCodes: children.map((c: any) => c.code), tagPath: tag.codePath })
                .withContext(ctx);

            childNodes.push(intermediateNode);
        }
        return childNodes;
    };

    return tagNode;
};

const ctx: ResourceOpCtx = {
    addResourceComponent,
    setCurrentTreeKey: setCurrentKey,
    getTreeNode: getNode,
    reloadTreeNode: reloadNode,
};

provide(ResourceOpCtxKey, ctx);
</script>

<style lang="scss" scoped>
.tag-tree-header {
    padding: 4px 6px;
    border-bottom: 1px solid var(--el-border-color-light);
}

.tag-tree-search {
    :deep(.el-input__wrapper) {
        border-radius: 14px;
        height: 24px;
    }
}
</style>
