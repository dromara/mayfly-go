<template>
    <div class="db-backup">
        <page-table
            height="100%"
            ref="pageTableRef"
            :page-api="dbApi.getDbBackups"
            :show-selection="true"
            v-model:selection-data="state.selectedData"
            :searchItems="searchItems"
            :before-query-fn="beforeQueryFn"
            v-model:query-form="query"
            :columns="columns"
        >
            <template #dbSelect>
                <el-select v-model="query.dbName" placeholder="请选择数据库" style="width: 200px" filterable clearable>
                    <el-option v-for="item in props.dbNames" :key="item" :label="`${item}`" :value="item"> </el-option>
                </el-select>
            </template>

            <template #tableHeader>
                <el-button type="primary" icon="plus" @click="createDbBackup()">添加</el-button>
                <el-button type="primary" icon="video-play" @click="enableDbBackup(null)">启用</el-button>
                <el-button type="primary" icon="video-pause" @click="disableDbBackup(null)">禁用</el-button>
            </template>

            <template #action="{ data }">
                <el-button @click="editDbBackup(data)" type="primary" link>编辑</el-button>
                <el-button @click="enableDbBackup(data)" v-if="!data.enabled" type="success" link>启用</el-button>
                <el-button @click="disableDbBackup(data)" v-if="data.enabled" type="warning" link>禁用</el-button>
            </template>
        </page-table>

        <db-backup-edit
            @val-change="search"
            :title="dbBackupEditDialog.title"
            :dbId="dbId"
            :data="dbBackupEditDialog.data"
            v-model:visible="dbBackupEditDialog.visible"
        ></db-backup-edit>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, defineAsyncComponent, Ref, ref } from 'vue';
import { dbApi } from './api';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { SearchItem } from '@/components/SearchForm';
import { ElMessage } from 'element-plus';

const DbBackupEdit = defineAsyncComponent(() => import('./DbBackupEdit.vue'));
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

const searchItems = [SearchItem.slot('dbName', '数据库名称', 'dbSelect')];

const columns = [
    TableColumn.new('dbName', '数据库名称'),
    TableColumn.new('name', '任务名称'),
    TableColumn.new('startTime', '启动时间').isTime(),
    TableColumn.new('intervalDay', '备份周期'),
    TableColumn.new('enabled', '是否启用'),
    TableColumn.new('lastResult', '执行结果'),
    TableColumn.new('lastTime', '执行时间').isTime(),
    TableColumn.new('action', '操作').isSlot().setMinWidth(160).fixedRight().alignCenter(),
];

const emptyQuery = {
    dbId: 0,
    dbName: '',
    pageNum: 1,
    pageSize: 10,
    repeated: true,
};

const state = reactive({
    data: [],
    total: 0,
    query: emptyQuery,
    dbBackupEditDialog: {
        visible: false,
        data: null as any,
        title: '创建数据库备份任务',
    },
    /**
     * 选中的数据
     */
    selectedData: [],
});

const { query, dbBackupEditDialog } = toRefs(state);

const beforeQueryFn = (query: any) => {
    query.dbId = props.dbId;
    return query;
};

const search = async () => {
    await pageTableRef.value.search();
};

const createDbBackup = async () => {
    state.dbBackupEditDialog.data = null;
    state.dbBackupEditDialog.title = '创建数据库备份任务';
    state.dbBackupEditDialog.visible = true;
};

const editDbBackup = async (data: any) => {
    state.dbBackupEditDialog.data = data;
    state.dbBackupEditDialog.title = '修改数据库备份任务';
    state.dbBackupEditDialog.visible = true;
};

const enableDbBackup = async (data: any) => {
    let backupId: String;
    if (data) {
        backupId = data.id;
    } else if (state.selectedData.length > 0) {
        backupId = state.selectedData.map((x: any) => x.id).join(' ');
    } else {
        ElMessage.error('请选择需要启用的备份任务');
        return;
    }
    await dbApi.enableDbBackup.request({ dbId: props.dbId, backupId: backupId });
    await search();
    ElMessage.success('启用成功');
};

const disableDbBackup = async (data: any) => {
    let backupId: String;
    if (data) {
        backupId = data.id;
    } else if (state.selectedData.length > 0) {
        backupId = state.selectedData.map((x: any) => x.id).join(' ');
    } else {
        ElMessage.error('请选择需要禁用的备份任务');
        return;
    }
    await dbApi.disableDbBackup.request({ dbId: props.dbId, backupId: backupId });
    await search();
    ElMessage.success('禁用成功');
};
</script>
<style lang="scss"></style>
