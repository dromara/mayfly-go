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
                <el-button type="primary" icon="plus" @click="editMongo(false)" plain>{{ $t('common.create') }}</el-button>
                <el-button type="danger" icon="delete" :disabled="selectionData.length < 1" @click="deleteMongo" plain>{{ $t('common.delete') }}</el-button>
            </template>

            <template #tagPath="{ data }">
                <resource-tags :tags="data.tags" />
            </template>

            <template #action="{ data }">
                <el-button @click="showDatabases(data.id)" link>{{ $t('mongo.db') }}</el-button>

                <el-button @click="showUsers(data.id)" link type="success">cmd</el-button>

                <el-button @click="editMongo(data)" link type="primary">{{ $t('common.edit') }}</el-button>
            </template>
        </page-table>

        <mongo-dbs v-model:visible="dbsVisible" :id="state.dbOps.dbId"></mongo-dbs>

        <mongo-run-command v-model:visible="usersVisible" :id="state.dbOps.dbId" />

        <mongo-edit
            @val-change="search()"
            :title="mongoEditDialog.title"
            v-model:visible="mongoEditDialog.visible"
            v-model:mongo="mongoEditDialog.data"
        ></mongo-edit>
    </div>
</template>

<script lang="ts" setup>
import { mongoApi } from './api';
import { defineAsyncComponent, onMounted, reactive, ref, Ref, toRefs } from 'vue';
import ResourceTags from '../component/ResourceTags.vue';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { useRoute } from 'vue-router';
import { getTagPathSearchItem } from '../component/tag';
import { SearchItem } from '@/components/SearchForm';
import { useI18nCreateTitle, useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nEditTitle } from '@/hooks/useI18n';

const MongoEdit = defineAsyncComponent(() => import('./MongoEdit.vue'));
const MongoDbs = defineAsyncComponent(() => import('./MongoDbs.vue'));
const MongoRunCommand = defineAsyncComponent(() => import('./MongoRunCommand.vue'));

const props = defineProps({
    lazy: {
        type: [Boolean],
        default: false,
    },
});

const route = useRoute();
const pageTableRef: Ref<any> = ref(null);

const searchItems = [
    SearchItem.input('keyword', 'common.keyword').withPlaceholder('mongo.keywordPlaceholder'),
    getTagPathSearchItem(TagResourceTypeEnum.Mongo.value),
];

const columns = [
    TableColumn.new('tags[0].tagPath', 'tag.relateTag').isSlot('tagPath').setAddWidth(20),
    TableColumn.new('name', 'common.name'),
    TableColumn.new('uri', 'mongo.connUrl'),
    TableColumn.new('createTime', 'common.createTime').isTime(),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('code', 'common.code'),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(170).fixedRight().alignCenter(),
];

const state = reactive({
    dbOps: {
        dbId: 0,
        db: '',
    },
    selectionData: [],
    query: {
        pageNum: 1,
        pageSize: 0,
        tagPath: '',
    },
    mongoEditDialog: {
        visible: false,
        data: null as any,
        title: '',
    },
    dbsVisible: false,
    usersVisible: false,
});

const { selectionData, query, mongoEditDialog, dbsVisible, usersVisible } = toRefs(state);

onMounted(() => {
    if (!props.lazy) {
        search();
    }
});

const checkRouteTagPath = (query: any) => {
    if (route.query.tagPath) {
        query.tagPath = route.query.tagPath as string;
    }
    return query;
};

const showDatabases = async (id: number) => {
    state.dbOps.dbId = id;
    state.dbsVisible = true;
};

const showUsers = async (id: number) => {
    state.dbOps.dbId = id;
    state.usersVisible = true;
};

const deleteMongo = async () => {
    try {
        await useI18nDeleteConfirm(state.selectionData.map((x: any) => x.name).join('、'));
        await mongoApi.deleteMongo.request({ id: state.selectionData.map((x: any) => x.id).join(',') });
        useI18nDeleteSuccessMsg();
        search();
    } catch (err) {
        //
    }
};

const search = async (tagPath: string = '') => {
    if (tagPath) {
        state.query.tagPath = tagPath;
    }
    pageTableRef.value.search();
};

const editMongo = async (data: any) => {
    if (!data) {
        state.mongoEditDialog.data = null;
        state.mongoEditDialog.title = useI18nCreateTitle('mongo.mongo');
    } else {
        state.mongoEditDialog.data = data;
        state.mongoEditDialog.title = useI18nEditTitle('mongo.mongo');
    }
    state.mongoEditDialog.visible = true;
};

defineExpose({ search });
</script>

<style></style>
