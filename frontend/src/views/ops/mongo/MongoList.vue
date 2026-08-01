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
                <el-button v-auth="'mongo:save'" type="primary" icon="plus" @click="editMongo(false)" plain>{{ $t('common.create') }}</el-button>
                <el-button v-auth="'mongo:del'" type="danger" icon="delete" :disabled="selectionData.length < 1" @click="deleteMongo" plain>
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #name="{ data }">
                <TagCodePath :code="data.code" show-popover />
                {{ data.name }}
            </template>

            <template #action="{ data }">
                <el-button @click="showDatabases(data.id)" link>{{ $t('mongo.db') }}</el-button>

                <el-button @click="showUsers(data.id)" link type="success">cmd</el-button>

                <el-button v-auth="'mongo:save'" @click="editMongo(data)" link type="primary">{{ $t('common.edit') }}</el-button>
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
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import { defineAsyncComponent, onMounted, reactive, toRefs, useTemplateRef } from 'vue';
import { useRoute } from 'vue-router';
import TagCodePath from '../component/TagCodePath.vue';
import { mongoApi } from './api';
import type { Mongo } from './types';

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
const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');

const searchItems = [SearchItem.input('keyword', 'common.keyword').withPlaceholder('mongo.keywordPlaceholder')];

const columns = [
    TableColumn.new('name', 'common.name').isSlot('name').setAddWidth(25),
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
        data: null as Mongo | null,
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

const checkRouteTagPath = (query: Record<string, unknown>) => {
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
        await useI18nDeleteConfirm(state.selectionData.map((x: Mongo) => x.name).join('、'));
        await mongoApi.deleteMongo.request({ id: state.selectionData.map((x: Mongo) => x.id).join(',') });
        Msg.deleteSuccess();
        search();
    } catch (err) {
        //
    }
};

const search = async (tagPath: string = '') => {
    if (tagPath) {
        state.query.tagPath = tagPath;
    }
    pageTableRef.value?.search();
};

const editMongo = async (data: Mongo | false) => {
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
