<template>
    <div class="db-backup-history">
        <page-table
            height="100%"
            ref="pageTableRef"
            :page-api="dbApi.getDbBackupHistories"
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
                <el-button type="primary" icon="back" @click="restoreDbBackupHistory(null)">立即恢复</el-button>
                <el-button type="danger" icon="delete" @click="deleteDbBackupHistory(null)">删除</el-button>
            </template>

            <template #action="{ data }">
                <div>
                    <el-button @click="restoreDbBackupHistory(data)" type="primary" link>立即恢复</el-button>
                    <el-button @click="deleteDbBackupHistory(data)" type="danger" link>删除</el-button>
                </div>
            </template>
        </page-table>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, Ref, ref } from 'vue';
import { dbApi } from './api';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { SearchItem } from '@/components/SearchForm';
import { ElMessage, ElMessageBox } from 'element-plus';

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
    TableColumn.new('name', '备份名称'),
    TableColumn.new('createTime', '创建时间').isTime(),
    TableColumn.new('lastResult', '恢复结果'),
    TableColumn.new('lastTime', '恢复时间').isTime(),
    TableColumn.new('action', '操作').isSlot().setMinWidth(160).fixedRight(),
];

const emptyQuery = {
    dbId: 0,
    dbName: '',
    pageNum: 1,
    pageSize: 10,
};

const state = reactive({
    data: [],
    total: 0,
    query: emptyQuery,
    /**
     * 选中的数据
     */
    selectedData: [],
});

const { query } = toRefs(state);

const beforeQueryFn = (query: any) => {
    query.dbId = props.dbId;
    return query;
};

const search = async () => {
    await pageTableRef.value.search();
};

const deleteDbBackupHistory = async (data: any) => {
    let backupHistoryId: string;
    if (data) {
        backupHistoryId = data.id;
    } else if (state.selectedData.length > 0) {
        backupHistoryId = state.selectedData.map((x: any) => x.id).join(' ');
    } else {
        ElMessage.error('请选择需要删除的数据库备份历史');
        return;
    }
    await ElMessageBox.confirm(`确定删除 “数据库备份历史” 吗？`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    });
    await dbApi.deleteDbBackupHistory.request({ dbId: props.dbId, backupHistoryId: backupHistoryId });
    await search();
    ElMessage.success('删除成功');
};

const restoreDbBackupHistory = async (data: any) => {
    let backupHistoryId: string;
    if (data) {
        backupHistoryId = data.id;
    } else if (state.selectedData.length > 0) {
        const pluralDbNames: string[] = [];
        const dbNames: Map<string, boolean> = new Map();
        state.selectedData.forEach((item: any) => {
            if (!dbNames.has(item.dbName)) {
                dbNames.set(item.dbName, false);
                return;
            }
            if (!dbNames.get(item.dbName)) {
                dbNames.set(item.dbName, true);
                pluralDbNames.push(item.dbName);
            }
        });
        if (pluralDbNames.length > 0) {
            ElMessage.error('多次选择相同数据库：' + pluralDbNames.join(', '));
            return;
        }
        backupHistoryId = state.selectedData.map((x: any) => x.id).join(' ');
    } else {
        ElMessage.error('请选择需要恢复的数据库备份历史');
        return;
    }
    await ElMessageBox.confirm(`确定从 “数据库备份历史” 中恢复数据库吗？`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    });

    await dbApi.restoreDbBackupHistory.request({
        dbId: props.dbId,
        backupHistoryId: backupHistoryId,
    });
    await search();
    ElMessage.success('成功创建数据库恢复任务');
};
</script>
<style lang="scss"></style>
