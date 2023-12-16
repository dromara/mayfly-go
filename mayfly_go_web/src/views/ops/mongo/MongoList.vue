<template>
    <div>
        <page-table
            ref="pageTableRef"
            :page-api="mongoApi.mongoList"
            :before-query-fn="checkRouteTagPath"
            :search-items="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="columns"
        >
            <template #tableHeader>
                <el-button type="primary" icon="plus" @click="editMongo(true)" plain>添加</el-button>
                <el-button type="danger" icon="delete" :disabled="selectionData.length < 1" @click="deleteMongo" plain>删除 </el-button>
            </template>

            <template #tagPath="{ data }">
                <resource-tag :resource-code="data.code" :resource-type="TagResourceTypeEnum.Mongo.value" />
            </template>

            <template #action="{ data }">
                <el-button @click="showDatabases(data.id)" link>数据库</el-button>

                <el-button @click="showUsers(data.id)" link type="success">cmd</el-button>

                <el-button @click="editMongo(data)" link type="primary">编辑</el-button>
            </template>
        </page-table>

        <mongo-dbs v-model:visible="dbsVisible" :id="state.dbOps.dbId"></mongo-dbs>

        <mongo-run-command v-model:visible="usersVisible" :id="state.dbOps.dbId" />

        <mongo-edit
            @val-change="search"
            :title="mongoEditDialog.title"
            v-model:visible="mongoEditDialog.visible"
            v-model:mongo="mongoEditDialog.data"
        ></mongo-edit>
    </div>
</template>

<script lang="ts" setup>
import { mongoApi } from './api';
import { defineAsyncComponent, ref, toRefs, reactive, onMounted, Ref } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import ResourceTag from '../component/ResourceTag.vue';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { useRoute } from 'vue-router';
import { getTagPathSearchItem } from '../component/tag';

const MongoEdit = defineAsyncComponent(() => import('./MongoEdit.vue'));
const MongoDbs = defineAsyncComponent(() => import('./MongoDbs.vue'));
const MongoRunCommand = defineAsyncComponent(() => import('./MongoRunCommand.vue'));

const route = useRoute();
const pageTableRef: Ref<any> = ref(null);

const searchItems = [getTagPathSearchItem(TagResourceTypeEnum.Mongo.value)];

const columns = [
    TableColumn.new('name', '名称'),
    TableColumn.new('uri', '连接uri'),
    TableColumn.new('tagPath', '关联标签').isSlot().setAddWidth(20).alignCenter(),
    TableColumn.new('createTime', '创建时间').isTime(),
    TableColumn.new('creator', '创建人'),
    TableColumn.new('action', '操作').isSlot().setMinWidth(170).fixedRight().alignCenter(),
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
        title: '新增mongo',
    },
    dbsVisible: false,
    usersVisible: false,
});

const { selectionData, query, mongoEditDialog, dbsVisible, usersVisible } = toRefs(state);

onMounted(async () => {});

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
        await ElMessageBox.confirm(`确定删除【${state.selectionData.map((x: any) => x.name).join(', ')}】mongo信息?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await mongoApi.deleteMongo.request({ id: state.selectionData.map((x: any) => x.id).join(',') });
        ElMessage.success('删除成功');
        search();
    } catch (err) {
        //
    }
};

const search = async () => {
    pageTableRef.value.search();
};

const editMongo = async (data: any) => {
    if (!data) {
        state.mongoEditDialog.data = null;
        state.mongoEditDialog.title = '新增mongo';
    } else {
        state.mongoEditDialog.data = data;
        state.mongoEditDialog.title = '修改mongo';
    }
    state.mongoEditDialog.visible = true;
};
</script>

<style></style>
