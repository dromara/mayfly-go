<template>
    <div class="db-restore">
        <page-table
            height="100%"
            ref="pageTableRef"
            :page-api="dbApi.getDbRestores"
            :show-selection="true"
            v-model:selection-data="state.selectedData"
            :searchItems="searchItems"
            :before-query-fn="beforeQueryFn"
            v-model:query-form="query"
            :columns="columns"
        >
            <template #dbSelect>
                <el-select v-model="query.dbName" placeholder="请选择数据库" style="width: 200px" filterable clearable>
                    <el-option v-for="item in dbNames" :key="item" :label="`${item}`" :value="item"> </el-option>
                </el-select>
            </template>

            <template #tableHeader>
                <el-button type="primary" icon="plus" @click="createDbRestore()">添加</el-button>
                <el-button type="primary" icon="video-play" @click="enableDbRestore(null)">启用</el-button>
                <el-button type="primary" icon="video-pause" @click="disableDbRestore(null)">禁用</el-button>
                <el-button type="danger" icon="delete" @click="deleteDbRestore(null)">删除</el-button>
            </template>

            <template #action="{ data }">
                <el-button @click="showDbRestore(data)" type="primary" link>详情</el-button>
                <el-button @click="enableDbRestore(data)" v-if="!data.enabled" type="primary" link>启用</el-button>
                <el-button @click="disableDbRestore(data)" v-if="data.enabled" type="primary" link>禁用</el-button>
                <el-button @click="deleteDbRestore(data)" type="danger" link>删除</el-button>
            </template>
        </page-table>

        <db-restore-edit
            @val-change="search"
            :title="dbRestoreEditDialog.title"
            :dbId="dbId"
            :dbNames="dbNames"
            :data="dbRestoreEditDialog.data"
            v-model:visible="dbRestoreEditDialog.visible"
        ></db-restore-edit>

        <el-dialog v-model="infoDialog.visible" title="数据库恢复">
            <el-descriptions :column="1" border>
                <el-descriptions-item :span="1" label="数据库名称">{{ infoDialog.data.dbName }}</el-descriptions-item>
                <el-descriptions-item v-if="infoDialog.data.pointInTime" :span="1" label="恢复时间点">{{
                    dateFormat(infoDialog.data.pointInTime)
                }}</el-descriptions-item>
                <el-descriptions-item v-if="!infoDialog.data.pointInTime" :span="1" label="数据库备份">{{
                    infoDialog.data.dbBackupHistoryName
                }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="开始时间">{{ dateFormat(infoDialog.data.startTime) }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="是否启用">{{ infoDialog.data.enabledDesc }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="执行时间">{{ dateFormat(infoDialog.data.lastTime) }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="执行结果">{{ infoDialog.data.lastResult }}</el-descriptions-item>
            </el-descriptions>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, defineAsyncComponent, Ref, ref } from 'vue';
import { dbApi } from './api';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { SearchItem } from '@/components/SearchForm';
import { ElMessage, ElMessageBox } from 'element-plus';
import { dateFormat } from '@/common/utils/date';
const DbRestoreEdit = defineAsyncComponent(() => import('./DbRestoreEdit.vue'));
const pageTableRef: Ref<any> = ref(null);

const props = defineProps({
    dbId: {
        type: [Number],
        required: true,
    },
    dbNames: {
        type: [Array<String>],
        required: true,
    },
});

// const queryConfig = [TableQuery.slot('dbName', '数据库名称', 'dbSelect')];
const searchItems = [SearchItem.slot('dbName', '数据库名称', 'dbSelect')];

const columns = [
    TableColumn.new('dbName', '数据库名称'),
    TableColumn.new('startTime', '启动时间').isTime(),
    TableColumn.new('enabledDesc', '是否启用'),
    TableColumn.new('lastTime', '执行时间').isTime(),
    TableColumn.new('lastResult', '执行结果'),
    TableColumn.new('action', '操作').isSlot().setMinWidth(220).fixedRight().alignCenter(),
];

const emptyQuery = {
    dbId: props.dbId,
    dbName: '',
    pageNum: 1,
    pageSize: 10,
    repeated: false,
};

const state = reactive({
    data: [],
    total: 0,
    query: emptyQuery,
    dbRestoreEditDialog: {
        visible: false,
        data: null as any,
        title: '创建数据库恢复任务',
    },
    infoDialog: {
        visible: false,
        data: null as any,
    },
    /**
     * 选中的数据
     */
    selectedData: [],
});

const { query, dbRestoreEditDialog, infoDialog } = toRefs(state);

const beforeQueryFn = (query: any) => {
    query.dbId = props.dbId;
    return query;
};

const search = async () => {
    await pageTableRef.value.search();
};

const createDbRestore = async () => {
    state.dbRestoreEditDialog.data = null;
    state.dbRestoreEditDialog.title = '数据库恢复';
    state.dbRestoreEditDialog.visible = true;
};

const deleteDbRestore = async (data: any) => {
    let restoreId: string;
    if (data) {
        restoreId = data.id;
    } else if (state.selectedData.length > 0) {
        restoreId = state.selectedData.map((x: any) => x.id).join(' ');
    } else {
        ElMessage.error('请选择需要删除的数据库恢复任务');
        return;
    }
    await ElMessageBox.confirm(`确定删除 “数据库恢复任务” 吗？`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    });
    await dbApi.deleteDbRestore.request({ dbId: props.dbId, restoreId: restoreId });
    await search();
    ElMessage.success('删除成功');
};

const showDbRestore = async (data: any) => {
    state.infoDialog.data = data;
    state.infoDialog.visible = true;
};

const enableDbRestore = async (data: any) => {
    let restoreId: string;
    if (data) {
        restoreId = data.id;
    } else if (state.selectedData.length > 0) {
        restoreId = state.selectedData.map((x: any) => x.id).join(' ');
    } else {
        ElMessage.error('请选择需要启用的数据库恢复任务');
        return;
    }
    await dbApi.enableDbRestore.request({ dbId: props.dbId, restoreId: restoreId });
    await search();
    ElMessage.success('启用成功');
};

const disableDbRestore = async (data: any) => {
    let restoreId: string;
    if (data) {
        restoreId = data.id;
    } else if (state.selectedData.length > 0) {
        restoreId = state.selectedData.map((x: any) => x.id).join(' ');
    } else {
        ElMessage.error('请选择需要禁用的数据库恢复任务');
        return;
    }
    await dbApi.disableDbRestore.request({ dbId: props.dbId, restoreId: restoreId });
    await search();
    ElMessage.success('禁用成功');
};
</script>
<style lang="scss"></style>
