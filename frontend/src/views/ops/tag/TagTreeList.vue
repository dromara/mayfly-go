<template>
    <div class="tag-tree-list card !p-2 h-full flex">
        <el-splitter>
            <el-splitter-panel size="30%" min="25%" max="35%" class="flex flex-col flex-1">
                <div class="card !p-1 !mb-1 !mr-1 flex justify-between">
                    <div class="mb-1">
                        <el-input v-model="filterTag" clearable :placeholder="$t('tag.nameFilterPlaceholder')" class="mr-2 !w-[200px]" />
                        <el-button
                            v-if="useUserInfo().userInfo.username == 'admin'"
                            v-auth="'tag:save'"
                            type="primary"
                            icon="plus"
                            @click="onShowSaveTagDialog(null)"
                        ></el-button>
                    </div>
                    <div>
                        <el-tooltip placement="top">
                            <template #content>
                                {{ $t('tag.tagTips1') }}
                                <br />
                                {{ $t('tag.tagTips2') }} <br />
                                {{ $t('tag.tagTips3') }}
                            </template>
                            <SvgIcon name="question-filled" />
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
                        draggable
                        :allow-drop="allowDrop"
                        :allow-drag="allowDrag"
                        @node-drop="onNodeDrop"
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
                                    <span style="color: #3c8dbc">【</span>
                                    {{ data.code }}
                                    <span style="color: #3c8dbc">】</span>
                                    <el-tag v-if="data.children !== null" size="small">{{ data.children.length }}</el-tag>
                                </span>
                            </span>
                        </template>
                    </el-tree>
                </el-scrollbar>
            </el-splitter-panel>

            <el-splitter-panel>
                <div class="ml-2 h-full">
                    <el-tabs class="h-full" @tab-change="onTabChange" v-model="state.activeTabName" v-if="currentTag">
                        <el-tab-pane :label="$t('common.detail')" :name="TagDetail">
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
                            :label="`${$t('tag.machine')} (${resourceCount.machine || 0})`"
                            :name="MachineTag"
                        >
                            <MachineList lazy ref="machineListRef" />
                        </el-tab-pane>

                        <el-tab-pane
                            class="h-full"
                            :disabled="currentTag.type != TagResourceTypeEnum.Tag.value"
                            :label="`${$t('tag.db')} (${resourceCount.db || 0})`"
                            :name="DbTag"
                        >
                            <InstanceList lazy ref="dbInstanceListRef" />
                        </el-tab-pane>

                        <el-tab-pane
                            class="h-full"
                            :disabled="currentTag.type != TagResourceTypeEnum.Tag.value"
                            :label="`${$t('tag.es')} (${resourceCount.es || 0})`"
                            :name="EsTag"
                        >
                            <EsInstanceList lazy ref="esInstanceListRef" />
                        </el-tab-pane>

                        <el-tab-pane
                            class="h-full"
                            :disabled="currentTag.type != TagResourceTypeEnum.Tag.value"
                            :label="`Redis (${resourceCount.redis || 0})`"
                            :name="RedisTag"
                        >
                            <RedisList lazy ref="redisListRef" />
                        </el-tab-pane>

                        <el-tab-pane
                            class="h-full"
                            :disabled="currentTag.type != TagResourceTypeEnum.Tag.value"
                            :label="`Mongo (${resourceCount.mongo || 0})`"
                            :name="MongoTag"
                        >
                            <MongoList lazy ref="mongoListRef" />
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
                <div class="dialog-footer">
                    <el-button @click="onCancelSaveTag()">{{ $t('common.cancel') }}</el-button>
                    <el-button @click="onSaveTag" type="primary">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-dialog>

        <contextmenu :dropdown="state.contextmenu.dropdown" :items="state.contextmenu.items" ref="contextmenuRef" />
    </div>
</template>

<script lang="ts" setup>
import { toRefs, ref, watch, reactive, onMounted, Ref, defineAsyncComponent } from 'vue';
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

const MachineList = defineAsyncComponent(() => import('../machine/MachineList.vue'));
const InstanceList = defineAsyncComponent(() => import('../db/InstanceList.vue'));
const EsInstanceList = defineAsyncComponent(() => import('../es/EsInstanceList.vue'));
const RedisList = defineAsyncComponent(() => import('../redis/RedisList.vue'));
const MongoList = defineAsyncComponent(() => import('../mongo/MongoList.vue'));

const { t } = useI18n();

interface Tree {
    id: number;
    codePath: string;
    name: string;
    children?: Tree[];
}

const tagForm: any = ref(null);
const tagTreeRef: any = ref(null);
const filterTag = ref('');
const contextmenuRef = ref();
const machineListRef: Ref<any> = ref(null);
const dbInstanceListRef: Ref<any> = ref(null);
const esInstanceListRef: Ref<any> = ref(null);
const redisListRef: Ref<any> = ref(null);
const mongoListRef: Ref<any> = ref(null);

const TagDetail = 'tagDetail';
const MachineTag = 'machineTag';
const DbTag = 'dbTag';
const EsTag = 'EsTag';
const RedisTag = 'redisTag';
const MongoTag = 'mongoTag';

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
});

watch(filterTag, (val) => {
    tagTreeRef.value!.filter(val);
});

watch(
    () => state.currentTag,
    (val: any) => {
        if (val.type == TagResourceTypeEnum.Tag.value) {
            tagApi.countTagResource.request({ tagPath: val.codePath }).then((res: any) => {
                state.resourceCount = res;
            });
        }

        setNowTabData();
    }
);

const allowDrop = (draggingNode: any, dropNode: any, type: any) => {
    // 不允许同层级移动
    if (type != 'inner') {
        return false;
    }

    const dropNodeData = dropNode.data;
    const draggingNodeData = draggingNode.data;
    const dropTagType = dropNodeData.type;
    const draggingTagType = draggingNodeData.type;

    // 目标节点只允许为标签类型
    if (dropTagType != TagResourceTypeEnum.Tag.value) {
        return false;
    }

    // 目标节点下没有子节点
    if (!dropNodeData.children) {
        // 都为标签类型允许移动
        if (dropTagType == draggingTagType && dropTagType == TagResourceTypeEnum.Tag.value) {
            return true;
        }

        // 目标节点为标签，允许移动
        if (dropTagType == TagResourceTypeEnum.Tag.value) {
            return true;
        }

        return false;
    }

    for (let child of dropNodeData.children) {
        // 当前移动节点若在目标节点下有相同code，则不允许移动
        if (draggingNodeData.code == child.code) {
            return false;
        }

        const childType = child.type;
        // 移动节点非标签类型时（资源标签）,并且子节点存在标签类型，则不允许移动，因为资源只允许放在叶子标签类型下
        if (draggingTagType != TagResourceTypeEnum.Tag.value && childType == TagResourceTypeEnum.Tag.value) {
            return false;
        }

        // 移动节点为标签类型时（资源标签）,并且子节点存在资源类型，则不允许移动
        if (draggingTagType == TagResourceTypeEnum.Tag.value && childType != TagResourceTypeEnum.Tag.value) {
            return false;
        }
    }
    return true;
};

const allowDrag = (node: any) => {
    const tagType = node.data.type;
    return (
        tagType == TagResourceTypeEnum.Tag.value ||
        tagType == TagResourceTypeEnum.DbInstance.value ||
        tagType == TagResourceTypeEnum.Redis.value ||
        tagType == TagResourceTypeEnum.Machine.value ||
        tagType == TagResourceTypeEnum.Mongo.value
    );
};

const onNodeDrop = async (draggingNode: any, dropNode: any) => {
    const draggingData = draggingNode.data;
    const dropData = dropNode.data;

    try {
        await tagApi.movingTag.request({
            fromPath: draggingData.codePath,
            toPath: dropData.codePath,
        });
    } finally {
        search();
    }
};

const onTabChange = () => {
    setNowTabData();
};

const setNowTabData = () => {
    const tagPath = state.currentTag.codePath;
    switch (state.activeTabName) {
        case MachineTag:
            machineListRef.value.search(tagPath);
            break;
        case DbTag:
            dbInstanceListRef.value.search(tagPath);
            break;
        case EsTag:
            esInstanceListRef.value.search(tagPath);
            break;
        case RedisTag:
            redisListRef.value.search(tagPath);
            break;
        case MongoTag:
            mongoListRef.value.search(tagPath);
            break;
        default:
            break;
    }
};

const filterNode = (value: string, data: Tree) => {
    return !value || isPrefixSubsequence(value, data.codePath) || isPrefixSubsequence(value, data.name);
};

const search = async () => {
    let res = await tagApi.getTagTrees.request(null);
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
    state.currentTag = await getDetail(data.id);
    state.activeTabName = TagDetail;
    // 关闭可能存在的右击菜单
    contextmenuRef.value.closeContextmenu();
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
<style lang="scss">
.tag-tree-list {
    .tag-tree-data {
        .el-tree-node__content {
            height: 40px;
            line-height: 40px;
        }
    }
}
</style>
