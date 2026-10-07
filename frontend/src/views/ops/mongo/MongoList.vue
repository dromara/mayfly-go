<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="mongoApi.mongoList"
            :before-query-fn="checkRouteTagPath"
            :search-items="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="columns"
            lazy
        >
            <template #tableHeader>
                <el-button v-auth="perms.mongoSave" type="primary" icon="plus" @click="editEntity()" plain>{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.mongoDel" type="danger" icon="delete" :disabled="selectionData.length < 1" @click="onDelete" plain>
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #name="{ data }">
                <TagCodePath :code="data.code" show-popover />
                {{ data.name }}
            </template>

            <template #action="{ data }">
                <el-button @click="showDatabases(data)" link>{{ $t('mongo.db') }}</el-button>
                <el-button @click="showCommands(data.id)" link type="success">{{ $t('mongo.commandConsole') }}</el-button>
                <el-button v-auth="perms.mongoSave" @click="editEntity(data)" link type="primary">{{ $t('common.edit') }}</el-button>
            </template>
        </page-table>

        <mongo-dbs v-model:visible="dbsVisible" :id="dbId" :name="dbName" :code="dbCode" />
        <mongo-run-command v-model:visible="usersVisible" :id="dbId" />
        <mongo-edit @val-change="search()" :title="editDialog.title" v-model:visible="editDialog.visible" v-model:data="editDialog.data" />
    </div>
</template>

<script lang="ts" setup>
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { useEditDialog, useRouteTagPath } from '@/hooks/useResourceForm';
import { defineAsyncComponent, onMounted, ref, useTemplateRef } from 'vue';
import TagCodePath from '../component/TagCodePath.vue';
import { mongoApi } from './api';
import { perms } from './perms';
import type { Mongo } from './types';

const MongoEdit = defineAsyncComponent(() => import('./MongoEdit.vue'));
const MongoDbs = defineAsyncComponent(() => import('./MongoDbs.vue'));
const MongoRunCommand = defineAsyncComponent(() => import('./MongoRunCommand.vue'));

const props = defineProps({
    lazy: {
        type: Boolean,
        default: false,
    },
});

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');
const checkRouteTagPath = useRouteTagPath();
const { editDialog, editEntity } = useEditDialog<Mongo>('mongo.mongo');

const query = ref({
    pageNum: 1,
    pageSize: 0,
    tagPath: '',
});

const selectionData = ref<Mongo[]>([]);

const searchItems = [SearchItem.input('keyword', 'common.keyword').withPlaceholder('mongo.keywordPlaceholder')];

const columns = [
    TableColumn.new('name', 'common.name').isSlot('name').setAddWidth(25),
    // 连接串已脱敏（密码固定为 ****），明文凭证不再回传前端
    TableColumn.new('uri', 'mongo.connUrl'),
    TableColumn.new('createTime', 'common.createTime').isTime(),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('code', 'common.code'),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(210).fixedRight().alignCenter(),
];

const dbsVisible = ref(false);
const usersVisible = ref(false);
const dbId = ref(0);
const dbName = ref('');
const dbCode = ref('');

onMounted(() => {
    if (!props.lazy) {
        search();
    }
});

const showDatabases = (data: Mongo) => {
    dbId.value = data.id;
    // 实例名与 code 随弹窗传入：从集合跳操作 tab 时要用同一把键，否则会给同一实例再挂一个面板
    dbName.value = data.name;
    dbCode.value = data.code;
    dbsVisible.value = true;
};

const showCommands = (id: number) => {
    dbId.value = id;
    usersVisible.value = true;
};

const onDelete = async () => {
    const records = selectionData.value || [];
    if (records.length === 0) return;
    if (!(await useI18nDeleteConfirm(records.map((x) => x.name).join('、')))) {
        // 取消或关掉弹窗：不继续后续操作
        return;
    }
    await mongoApi.deleteMongo.request({ id: records.map((x) => x.id).join(',') });
    Msg.deleteSuccess();
    search();
};

const search = (tagPath?: string) => {
    // tagPath 为 undefined 时（如"所有资源"节点），清空过滤条件查询全部；为具体值时按标签过滤
    query.value.tagPath = tagPath ?? '';
    pageTableRef.value?.search();
};

defineExpose({ search });
</script>
