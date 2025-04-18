<template>
    <div class="sync-task-logs">
        <el-dialog v-model="dialogVisible" :before-close="cancel" :destroy-on-close="false" width="1120px">
            <template #header>
                <span class="mr-2">{{ $t('db.log') }}</span>
                <el-switch v-model="realTime" @change="watchPolling" inline-prompt :active-text="$t('db.realTime')" :inactive-text="$t('db.noRealTime')" />
                <el-button @click="search" icon="Refresh" circle size="small" :loading="realTime" class="ml-2"></el-button>
            </template>
            <page-table ref="logTableRef" :page-api="dbApi.datasyncLogs" v-model:query-form="query" :tool-button="false" :columns="columns" size="small">
            </page-table>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { reactive, Ref, ref, toRefs, watch } from 'vue';
import { dbApi } from '@/views/ops/db/api';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { DbDataSyncLogStatusEnum } from './enums';

const props = defineProps({
    taskId: {
        type: Number,
    },
    running: {
        type: Boolean,
        default: false,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

const columns = ref([
    // 状态:1.成功  -1.失败
    TableColumn.new('status', 'common.status').alignCenter().typeTag(DbDataSyncLogStatusEnum),
    TableColumn.new('createTime', 'Time').alignCenter().isTime(),
    TableColumn.new('errText', 'db.log'),
    TableColumn.new('dataSqlFull', 'SQL').alignCenter(),
    TableColumn.new('resNum', 'Rows'),
]);

watch(dialogVisible, (newValue: any) => {
    if (!newValue) {
        state.polling = false;
        watchPolling(false);
        return;
    }

    state.query.taskId = props.taskId!;
    search();
    state.realTime = props.running;
    watchPolling(props.running);
});

const startPolling = () => {
    if (!state.polling) {
        state.polling = true;
        state.pollingIndex = setInterval(search, 1000);
    }
};
const stopPolling = () => {
    if (state.polling) {
        state.polling = false;
        clearInterval(state.pollingIndex);
    }
};

const watchPolling = (polling: boolean) => {
    if (polling) {
        startPolling();
    } else {
        stopPolling();
    }
};

const logTableRef: Ref<any> = ref(null);

const search = () => {
    try {
        logTableRef.value.search();
    } catch (e) {
        /* empty */
    }
};

const emit = defineEmits(['update:visible', 'cancel', 'val-change']);
//定义事件
const cancel = () => {
    dialogVisible.value = false;
    emit('cancel');
    watchPolling(false);
};

const state = reactive({
    polling: false,
    pollingIndex: 0 as any,
    realTime: props.running,
    /**
     * 查询条件
     */
    query: {
        taskId: 0,
        name: null,
        pageNum: 1,
        pageSize: 0,
    },
});

const { query, realTime } = toRefs(state);
</script>
