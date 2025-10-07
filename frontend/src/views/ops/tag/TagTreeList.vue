<template>
    <div class="tag-tree-list card !p-2 h-full flex">
        <el-splitter>
            <el-splitter-panel size="24%" max="35%" class="flex flex-col flex-1">
                <div class="card !p-1 !mr-1 flex flex-row items-center justify-between overflow-hidden">
                    <el-input v-model="filterTag" clearable :placeholder="$t('tag.nameFilterPlaceholder')" class="mr-2" />
                    <el-button
                        v-if="useUserInfo().userInfo.username == 'admin'"
                        v-auth="'tag:save'"
                        type="primary"
                        icon="plus"
                        @click="onShowSaveTagDialog(null)"
                    ></el-button>
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
                        :indent="10"
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
                                    <template v-if="data.code">
                                        <span style="color: #3c8dbc">【</span>
                                        {{ data.code }}
                                        <span style="color: #3c8dbc">】</span>
                                    </template>
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

                            <div class="h-full" v-if="Number.isInteger(state.activeTabName) && Number.parseInt(state.activeTabName) === index">
                                <component lazy :ref="(el: any) => setComponentRef(el, index)" :is="resource?.componentConf.component"></component>
                            </div>
                        </el-tab-pane>
                    </el-tabs>
                </div>
            </el-splitter-panel>
        </el-splitter>

        <el-dialog width="500px" :title="saveTabDialog.title" :before-close="onCancelSaveTag" v-model="saveTabDialog.visible">
            <el-form ref="tagForm" :rules="rules" :model="saveTabDialog.form" label-width="auto">
                <el-form-item prop="code" :label="$t('tag.code')" required>
                    <el-input :disabled="saveTabDialog.form.id ? true : false" v-model="saveTabDialog.form.code" auto-complete="off"></el-input>
                </el-form-item>
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
import { toRefs, ref, watch, reactive, onMounted, computed, nextTick, useTemplateRef } from 'vue';
import { tagApi } from './api';
import { formatDate } from '@/common/utils/format';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu/index';
import { useUserInfo } from '@/store/userInfo';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import EnumTag from '@/components/enumtag/EnumTag.vue';
import EnumValue from '@/common/Enum';
import TagCodePath from '../component/TagCodePath.vue';
import { isPrefixSubsequence } from '@/common/utils/string';
import { useI18n } from 'vue-i18n';
import {
    useI18nCreateTitle,
    useI18nDeleteConfirm,
    useI18nDeleteSuccessMsg,
    useI18nEditTitle,
    useI18nFormValidate,
    useI18nSaveSuccessMsg,
} from '@/hooks/useI18n';
import { Rules } from '@/common/rule';
import { getResourceConfigs } from '@/views/ops/resource/resource';
import { hasPerm } from '@/components/auth/auth';

const compRefs = ref<Array<any>>([]);
const setComponentRef = (el: any, index: number) => {
    compRefs.value[index] = el;
};

const { t } = useI18n();

interface Tree {
    id: number;
    codePath: string;
    name: string;
    children?: Tree[];
}

const tagForm: any = ref(null);
const tagTreeRef: any = useTemplateRef('tagTreeRef');
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
    .withHideFunc((data: any) => {
        // 非标签类型不可添加子标签
        return data.type != TagResourceTypeEnum.Tag.value || (data.children && data.children?.[0].type != TagResourceTypeEnum.Tag.value);
    })
    .withOnClick((data: any) => onShowSaveTagDialog(data));

const contextmenuEdit = new ContextmenuItem('edit', 'common.edit')
    .withIcon('edit')
    .withPermission('tag:save')
    .withHideFunc((data: any) => {
        return data.type != TagResourceTypeEnum.Tag.value;
    })
    .withOnClick((data: any) => onShowEditTagDialog(data));

const contextmenuDel = new ContextmenuItem('delete', 'common.delete')
    .withIcon('delete')
    .withPermission('tag:del')
    .withHideFunc((data: any) => {
        // 存在子标签，则不允许删除
        return data.children || data.type != TagResourceTypeEnum.Tag.value;
    })
    .withOnClick((data: any) => onDeleteTag(data));

const state = reactive({
    data: [],
    saveTabDialog: {
        title: '',
        visible: false,
        form: { id: 0, pid: 0, code: '', name: '', remark: '' },
    },
    resourceDialog: {
        title: '',
        visible: false,
        tagPath: '',
        data: null as any,
    },
    // 展开的节点
    defaultExpandedKeys: [] as any,
    contextmenu: {
        dropdown: {
            x: 0,
            y: 0,
        },
        items: [contextmenuEdit, contextmenuAdd, contextmenuDel],
    },
    activeTabName: TagDetail,
    currentTag: null as any,
    resourceCount: {} as any,
});

const { data, saveTabDialog, currentTag, resourceCount, defaultExpandedKeys } = toRefs(state);

const props = {
    label: 'name',
    children: 'children',
};

const rules = {
    code: [Rules.requiredInput('tag.code')],
    name: [Rules.requiredInput('common.name')],
};

onMounted(() => {
    search();
    tagTreeRef.value.setCurrentKey(allNode.id);
    onTreeNodeClick(allNode);
});

watch(filterTag, (val) => {
    tagTreeRef.value!.filter(val);
});

watch(
    () => state.currentTag,
    (val: any) => {
        if (val?.type == TagResourceTypeEnum.Tag.value) {
            tagApi.countTagResource.request({ tagPath: val.codePath }).then((res: any) => {
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
    if (Number.isInteger(state.activeTabName)) {
        (await getResouceCompRef(Number.parseInt(state.activeTabName))).search(state.currentTag.codePath);
    }
};

const getResouceCompRef = (index: number): Promise<any> => {
    // 使用一个 Promise 来确保组件引用已经被设置
    return new Promise((resolve) => {
        const checkRef = () => {
            if (compRefs.value[index]) {
                resolve(compRefs.value[index]);
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

const filterNode = (value: string, data: Tree) => {
    return !value || isPrefixSubsequence(value, data.codePath) || isPrefixSubsequence(value, data.name);
};

const search = async () => {
    let res = await tagApi.getTagTrees.request(null);
    res.unshift(allNode);
    state.data = res;
};

const getDetail = async (id: number) => {
    const tags = await tagApi.listByQuery.request({ id });
    return tags?.[0];
};

// 树节点右击事件
const onNodeContextmenu = (event: any, data: any) => {
    const { clientX, clientY } = event;
    state.contextmenu.dropdown.x = clientX;
    state.contextmenu.dropdown.y = clientY;
    contextmenuRef.value.openContextmenu(data);
};

const onTreeNodeClick = async (data: any) => {
    // 关闭可能存在的右击菜单
    contextmenuRef.value.closeContextmenu();

    if (data.id == allNode.id) {
        state.currentTag = data;
        state.activeTabName = 0 as any;
        onTabChange();
        return;
    }

    state.currentTag = await getDetail(data.id);
    state.activeTabName = TagDetail;
};

const onShowSaveTagDialog = (data: any) => {
    if (data) {
        state.saveTabDialog.form.pid = data.id;
        state.saveTabDialog.title = t('tag.createSubTagTitle', { codePath: data.codePath });
    } else {
        state.saveTabDialog.title = useI18nCreateTitle('tag.rootTag');
    }
    state.saveTabDialog.visible = true;
};

const onShowEditTagDialog = (data: any) => {
    state.saveTabDialog.form.id = data.id;
    state.saveTabDialog.form.code = data.code;
    state.saveTabDialog.form.name = data.name;
    state.saveTabDialog.form.remark = data.remark;
    state.saveTabDialog.title = useI18nEditTitle(data.codePath);
    state.saveTabDialog.visible = true;
};

const onSaveTag = async () => {
    await useI18nFormValidate(tagForm);
    const form = state.saveTabDialog.form;
    await tagApi.saveTagTree.request(form);
    useI18nSaveSuccessMsg();
    search();
    onCancelSaveTag();
    state.currentTag = null;
};

const onCancelSaveTag = () => {
    state.saveTabDialog.visible = false;
    state.saveTabDialog.form = {} as any;
    tagForm.value.resetFields();
};

const onDeleteTag = async (data: any) => {
    await useI18nDeleteConfirm(data.codePath);
    await tagApi.delTagTree.request({ id: data.id });
    useI18nDeleteSuccessMsg();
    search();
};

// 节点被展开时触发的事件
const onNodeExpand = (data: any, node: any) => {
    const id: any = node.data.id;
    if (!state.defaultExpandedKeys.includes(id)) {
        state.defaultExpandedKeys.push(id);
    }
};

// 关闭节点
const onNodeCollapse = (data: any, node: any) => {
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

const removeDeafultExpandId = (id: any) => {
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
