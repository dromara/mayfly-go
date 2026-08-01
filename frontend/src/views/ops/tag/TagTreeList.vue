<template>
    <div class="tag-tree-list card p-2! h-full flex">
        <el-splitter>
            <el-splitter-panel size="24%" max="35%" class="flex flex-col flex-1">
                <div class="card p-1! mr-1! flex flex-row items-center justify-between overflow-hidden">
                    <el-input v-model="filterTag" clearable :placeholder="$t('tag.nameFilterPlaceholder')" class="mr-2" />
                    <el-button v-auth="'tag:save'" type="primary" icon="plus" @click="onShowSaveTagDialog(null)"></el-button>
                    <div>
                        <el-tooltip placement="top">
                            <template #content>
                                {{ $t('tag.tagTips1') }}
                                <br />
                                {{ $t('tag.tagTips2') }}
                                <br />
                                {{ $t('tag.tagTips3') }}
                                <br />
                                {{ $t('tag.tagTips4') }}
                            </template>
                            <SvgIcon class="ml-1" name="question-filled" />
                        </el-tooltip>
                    </div>
                </div>
                <el-scrollbar class="tag-tree-data">
                    <el-tree
                        class="min-w-full inline-block"
                        ref="tagTreeRef"
                        node-key="id"
                        highlight-current
                        :props="props"
                        :data="data"
                        @node-expand="onNodeExpand"
                        @node-collapse="onNodeCollapse"
                        @node-contextmenu="onNodeContextmenu"
                        @node-click="onTreeNodeClick"
                        :default-expanded-keys="defaultExpandedKeys"
                        :expand-on-click-node="false"
                        :filter-node-method="filterNode"
                    >
                        <template #default="{ data }">
                            <span class="custom-tree-node">
                                <SvgIcon
                                    :name="EnumValue.getEnumByValue(TagResourceTypeEnum, data.type)?.extra.icon"
                                    :color="EnumValue.getEnumByValue(TagResourceTypeEnum, data.type)?.extra.iconColor"
                                />

                                <span class="ml-1">
                                    {{ data.name }}
                                    <el-tag v-if="data.children !== null && data.id != allNode.id" size="small">{{ data.children.length }}</el-tag>
                                </span>
                            </span>
                        </template>
                    </el-tree>
                </el-scrollbar>
            </el-splitter-panel>

            <el-splitter-panel>
                <div class="ml-2 h-full">
                    <el-tabs class="h-full" @tab-change="onTabChange" v-model="state.activeTabName" v-if="currentTag">
                        <el-tab-pane v-if="currentTag.id != allNode.id" :label="$t('common.detail')" :name="TagDetail">
                            <el-descriptions :column="2" border>
                                <el-descriptions-item :label="$t('common.type')">
                                    <EnumTag :enums="TagResourceTypeEnum" :value="currentTag.type" />
                                </el-descriptions-item>
                                <el-descriptions-item label="code">{{ currentTag.code }}</el-descriptions-item>

                                <el-descriptions-item :label="$t('common.path')" :span="2">
                                    <TagCodePath :path="currentTag.codePath" />
                                </el-descriptions-item>

                                <el-descriptions-item :label="$t('common.name')">{{ currentTag.name }}</el-descriptions-item>
                                <el-descriptions-item :label="$t('common.remark')">{{ currentTag.remark }}</el-descriptions-item>

                                <el-descriptions-item :label="$t('common.creator')">{{ currentTag.creator }}</el-descriptions-item>
                                <el-descriptions-item :label="$t('common.createTime')">{{ formatDate(currentTag.createTime) }}</el-descriptions-item>
                                <el-descriptions-item :label="$t('common.modifier')">{{ currentTag.modifier }}</el-descriptions-item>
                                <el-descriptions-item :label="$t('common.updateTime')">{{ formatDate(currentTag.updateTime) }}</el-descriptions-item>
                            </el-descriptions>
                        </el-tab-pane>

                        <el-tab-pane
                            class="h-full"
                            :disabled="currentTag.type != TagResourceTypeEnum.Tag.value"
                            :label="`${$t(resource?.componentConf.name || '')} (${resourceCount[resource?.countKey || ''] || 0})`"
                            :name="index"
                            v-for="(resource, index) in resources"
                        >
                            <template #label>
                                <SvgIcon :name="resource?.componentConf.icon?.name" :color="resource?.componentConf.icon?.color" />
                                <span class="ml-1">
                                    {{ `${$t(resource?.componentConf.name || '')} (${resourceCount[resource?.countKey || ''] || 0})` }}
                                </span>
                            </template>

                            <div class="h-full" v-if="Number.isInteger(state.activeTabName) && Number.parseInt(String(state.activeTabName)) === index">
                                <component lazy :ref="(el: unknown) => setComponentRef(el as ComponentPublicInstance, index)" :is="resource?.componentConf.component"></component>
                            </div>
                        </el-tab-pane>
                    </el-tabs>
                </div>
            </el-splitter-panel>
        </el-splitter>

        <el-dialog width="500px" :title="saveTabDialog.title" :before-close="onCancelSaveTag" v-model="saveTabDialog.visible">
            <el-form ref="tagForm" :rules="rules" :model="saveTabDialog.form" label-width="auto">
                <el-form-item prop="name" :label="$t('common.name')" required>
                    <el-input v-model="saveTabDialog.form.name" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item :label="$t('common.remark')">
                    <el-input v-model="saveTabDialog.form.remark" auto-complete="off"></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="onCancelSaveTag()">{{ $t('common.cancel') }}</el-button>
                <el-button @click="onSaveTag" type="primary">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-dialog>

        <contextmenu :dropdown="state.contextmenu.dropdown" :items="state.contextmenu.items" ref="contextmenuRef" />
    </div>
</template>

<script lang="ts" setup>
import { TagResourceTypeEnum } from '@/common/commonEnum';
import EnumValue from '@/common/Enum';
import { Rules } from '@/common/rule';
import { formatDate } from '@/common/utils/format';
import { isPrefixSubsequence } from '@/common/utils/string';
import { hasPerm } from '@/components/auth/auth';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu/index';
import EnumTag from '@/components/enum-tag/EnumTag.vue';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle, useI18nFormValidate } from '@/hooks/useI18n';
import { getResourceConfigs } from '@/views/ops/resource/resource';
import { computed, nextTick, onMounted, reactive, ref, toRefs, useTemplateRef, watch, type ComponentPublicInstance } from 'vue';
import type { FormInstance } from 'element-plus';
import { useI18n } from 'vue-i18n';
import TagCodePath from '../component/TagCodePath.vue';
import type { TagTree } from './types';
import { tagApi } from './api';

const compRefs = ref<ComponentPublicInstance[]>([]);
const setComponentRef = (el: ComponentPublicInstance | null, index: number) => {
    if (el) compRefs.value[index] = el;
};

const { t } = useI18n();

interface TreeNodeData {
    id: number;
    type?: number;
    codePath?: string;
    name: string;
    remark?: string;
    children?: TreeNodeData[];
    [key: string]: unknown;
}

interface TreeNode {
    data: TreeNodeData;
    childNodes: TreeNode[];
    expanded: boolean;
}

const tagForm = useTemplateRef<FormInstance>('tagForm');
const tagTreeRef = useTemplateRef<{ setCurrentKey: (key: number) => void; filter: (val: string) => void }>('tagTreeRef');
const filterTag = ref('');
const contextmenuRef = ref();

const TagDetail = 'tagDetail';

const allNode = {
    id: -1,
    name: t('tag.allResource'),
    type: TagResourceTypeEnum.Tag.value,
    children: [],
};

const resources = computed(() => {
    return getResourceConfigs()
        .filter((x) => {
            if (!x.manager?.componentConf) {
                return true;
            }
            if (!x.manager.permCode) {
                return true;
            }
            return hasPerm(x.manager.permCode);
        })
        .map((x) => x.manager);
});

const contextmenuAdd = new ContextmenuItem('addTag', 'tag.createSubTag')
    .withIcon('circle-plus')
    .withPermission('tag:save')
    .withHideFunc((data: unknown) => {
        const d = data as TreeNodeData;
        // 非标签类型不可添加子标签
        return d.type != TagResourceTypeEnum.Tag.value || (!!d.children && d.children?.[0].type != TagResourceTypeEnum.Tag.value);
    })
    .withOnClick((data: unknown) => onShowSaveTagDialog(data as TreeNodeData));

const contextmenuEdit = new ContextmenuItem('edit', 'common.edit')
    .withIcon('edit')
    .withPermission('tag:save')
    .withHideFunc((data: unknown) => {
        return (data as TreeNodeData).type != TagResourceTypeEnum.Tag.value;
    })
    .withOnClick((data: unknown) => onShowEditTagDialog(data as TreeNodeData));

const contextmenuDel = new ContextmenuItem('delete', 'common.delete')
    .withIcon('delete')
    .withPermission('tag:del')
    .withHideFunc((data: unknown) => {
        const d = data as TreeNodeData;
        // 存在子标签，则不允许删除
        return !!d.children || d.type != TagResourceTypeEnum.Tag.value;
    })
    .withOnClick((data: unknown) => onDeleteTag(data as TreeNodeData));

const state = reactive({
    data: [] as TagTree[],
    saveTabDialog: {
        title: '',
        visible: false,
        form: { id: 0, pid: 0, name: '', remark: '' },
    },
    resourceDialog: {
        title: '',
        visible: false,
        tagPath: '',
        data: null as TagTree | null,
    },
    // 展开的节点
    defaultExpandedKeys: [] as number[],
    contextmenu: {
        dropdown: {
            x: 0,
            y: 0,
        },
        items: [contextmenuEdit, contextmenuAdd, contextmenuDel],
    },
    activeTabName: TagDetail as string | number,
    currentTag: null as TagTree | null,
    resourceCount: {} as Record<string, number>,
});

const { data, saveTabDialog, currentTag, resourceCount, defaultExpandedKeys } = toRefs(state);

const props = {
    label: 'name',
    children: 'children',
};

const rules = {
    name: [Rules.requiredInput('common.name')],
};

onMounted(() => {
    search();
    tagTreeRef.value?.setCurrentKey(allNode.id);
    onTreeNodeClick(allNode);
});

watch(filterTag, (val) => {
    tagTreeRef.value?.filter(val);
});

watch(
    () => state.currentTag,
    (val: TagTree | null) => {
        if (val?.type == TagResourceTypeEnum.Tag.value) {
            tagApi.countTagResource.request({ tagPath: val.codePath }).then((res: Record<string, number>) => {
                state.resourceCount = res;
            });
        }

        setNowTabData();
    }
);

const onTabChange = () => {
    setNowTabData();
};

const setNowTabData = async () => {
    if (Number.isInteger(state.activeTabName) && state.currentTag) {
        (await getResouceCompRef(Number.parseInt(String(state.activeTabName)))).search(state.currentTag.codePath);
    }
};

const getResouceCompRef = (index: number): Promise<{ search: (tagPath: string) => void }> => {
    // 使用一个 Promise 来确保组件引用已经被设置
    return new Promise((resolve) => {
        const checkRef = () => {
            if (compRefs.value[index]) {
                resolve(compRefs.value[index] as unknown as { search: (tagPath: string) => void });
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

const filterNode = (value: string, data: TreeNodeData) => {
    return !value || isPrefixSubsequence(value, data.codePath || '') || isPrefixSubsequence(value, data.name);
};

const search = async () => {
    let res = await tagApi.getTagTrees.request({ flatten: '0' });
    res.unshift(allNode as unknown as TagTree);
    state.data = res;
};

const getDetail = async (id: number) => {
    const tags = await tagApi.listByQuery.request({ id });
    return tags?.[0];
};

// 树节点右击事件
const onNodeContextmenu = (event: MouseEvent, data: TreeNodeData) => {
    const { clientX, clientY } = event;
    state.contextmenu.dropdown.x = clientX;
    state.contextmenu.dropdown.y = clientY;
    contextmenuRef.value?.openContextmenu(data);
};

const onTreeNodeClick = async (data: TreeNodeData) => {
    // 关闭可能存在的右击菜单
    contextmenuRef.value?.closeContextmenu();

    if (data.id == allNode.id) {
        state.currentTag = data as unknown as TagTree;
        state.activeTabName = 0 as string | number;
        onTabChange();
        return;
    }

    state.currentTag = await getDetail(data.id);
    state.activeTabName = TagDetail;
};

const onShowSaveTagDialog = (data: TreeNodeData | null) => {
    if (data) {
        state.saveTabDialog.form.pid = data.id;
        state.saveTabDialog.title = t('tag.createSubTagTitle', { codePath: data.codePath });
    } else {
        state.saveTabDialog.title = useI18nCreateTitle('tag.rootTag');
    }
    state.saveTabDialog.visible = true;
};

const onShowEditTagDialog = (data: TreeNodeData) => {
    state.saveTabDialog.form.id = data.id;
    state.saveTabDialog.form.name = data.name;
    state.saveTabDialog.form.remark = data.remark ?? '';
    state.saveTabDialog.title = useI18nEditTitle(data.codePath ?? '');
    state.saveTabDialog.visible = true;
};

const onSaveTag = async () => {
    await useI18nFormValidate(tagForm);
    const form = state.saveTabDialog.form;
    await tagApi.saveTagTree.request(form);
    Msg.saveSuccess();
    search();
    onCancelSaveTag();
    state.currentTag = null;
};

const onCancelSaveTag = () => {
    state.saveTabDialog.visible = false;
    state.saveTabDialog.form = { id: 0, pid: 0, name: '', remark: '' };
    tagForm.value?.resetFields();
};

const onDeleteTag = async (data: TreeNodeData) => {
    await useI18nDeleteConfirm(data.codePath);
    await tagApi.delTagTree.request({ id: data.id });
    Msg.deleteSuccess();
    search();
};

// 节点被展开时触发的事件
const onNodeExpand = (data: TreeNodeData, node: TreeNode) => {
    const id = node.data.id;
    if (!state.defaultExpandedKeys.includes(id)) {
        state.defaultExpandedKeys.push(id);
    }
};

// 关闭节点
const onNodeCollapse = (data: TreeNodeData, node: TreeNode) => {
    removeDeafultExpandId(node.data.id);

    let childNodes = node.childNodes;
    for (let cn of childNodes) {
        if (cn.expanded) {
            removeDeafultExpandId(cn.data.id);
        }
        // 递归删除展开的子节点节点id
        onNodeCollapse(data, cn);
    }
};

const removeDeafultExpandId = (id: number) => {
    let index = state.defaultExpandedKeys.indexOf(id);
    if (index > -1) {
        state.defaultExpandedKeys.splice(index, 1);
    }
};
</script>
<style lang="scss" scoped>
.tag-tree-list {
    .tag-tree-data {
        // .el-tree-node__content {
        //     height: 40px;
        //     line-height: 40px;
        // }
    }
}
</style>
