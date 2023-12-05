<template>
    <div>
        <page-table
            ref="pageTableRef"
            :query="queryConfig"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :data="list"
            :columns="columns"
            :total="total"
            v-model:page-size="query.pageSize"
            v-model:page-num="query.pageNum"
            @pageChange="search()"
        >
            <template #tagPathSelect>
                <el-select @focus="getTags" v-model="query.tagPath" placeholder="请选择标签" @clear="search" filterable clearable style="width: 200px">
                    <el-option v-for="item in tags" :key="item" :label="item" :value="item"> </el-option>
                </el-select>
            </template>

            <template #queryRight>
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
            @val-change="valChange"
            :title="mongoEditDialog.title"
            v-model:visible="mongoEditDialog.visible"
            v-model:mongo="mongoEditDialog.data"
        ></mongo-edit>
    </div>
</template>

<script lang="ts" setup>
import { mongoApi } from './api';
import { defineAsyncComponent, ref, toRefs, reactive, onMounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import ResourceTag from '../component/ResourceTag.vue';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn, TableQuery } from '@/components/pagetable';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { tagApi } from '../tag/api';
import { useRoute } from 'vue-router';

const MongoEdit = defineAsyncComponent(() => import('./MongoEdit.vue'));
const MongoDbs = defineAsyncComponent(() => import('./MongoDbs.vue'));
const MongoRunCommand = defineAsyncComponent(() => import('./MongoRunCommand.vue'));

const pageTableRef: any = ref(null);
const route = useRoute();

const queryConfig = [TableQuery.slot('tagPath', '标签', 'tagPathSelect')];
const columns = ref([
    TableColumn.new('name', '名称'),
    TableColumn.new('uri', '连接uri'),
    TableColumn.new('tagPath', '关联标签').isSlot().setAddWidth(20).alignCenter(),
    TableColumn.new('createTime', '创建时间').isTime(),
    TableColumn.new('creator', '创建人'),
    TableColumn.new('action', '操作').isSlot().setMinWidth(170).fixedRight().alignCenter(),
]);

const state = reactive({
    tags: [],
    dbOps: {
        dbId: 0,
        db: '',
    },
    list: [],
    total: 0,
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

const { tags, list, total, selectionData, query, mongoEditDialog, dbsVisible, usersVisible } = toRefs(state);

onMounted(async () => {
    search();
});

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
    try {
        pageTableRef.value.loading(true);

        if (route.query.tagPath) {
            state.query.tagPath = route.query.tagPath as string;
        }

        const res = await mongoApi.mongoList.request(state.query);
        state.list = res.list;
        state.total = res.total;
    } finally {
        pageTableRef.value.loading(false);
    }
};

const getTags = async () => {
    state.tags = await tagApi.getResourceTagPaths.request({ resourceType: TagResourceTypeEnum.Mongo.value });
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

const valChange = () => {
    search();
};
</script>

<style></style>
