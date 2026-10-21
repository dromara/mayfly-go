<template>
    <!-- 任务执行记录弹窗：与任务强关联（triggerRef=任务 key 固定过滤，不暴露清除入口），
         cron 与 manual 触发的运行 triggerRef 均为任务 key；取消/回放/反馈/评测等深度操作
         经「在执行历史中查看」跳转全局页 -->
    <el-dialog v-model="visible" :title="`${$t('agentops.execRecord')} - ${taskName}`" width="75%" destroy-on-close>
        <div class="run-history-dialog__body">
            <data-table ref="dataTableRef" :page-api="agentRunApi.list" :search-items="searchItems" v-model:query-form="params" :columns="columns">
                <!-- 运行标题 + 形态标签（与 RunList 同一渲染口径：后端 runRole 按 i18n 渲染） -->
                <template #title="{ data }">
                    <el-tag v-if="data.runRole" size="small" effect="plain" :type="enumTagType(RunRoleEnum, data.runRole)">
                        {{ enumLabel(RunRoleEnum, data.runRole, t) }}
                    </el-tag>
                    <span class="ml-1">{{ data.title }}</span>
                </template>
            </data-table>
        </div>
        <template #footer>
            <el-button @click="openFullHistory">{{ $t('agentops.runHistoryAll') }}</el-button>
            <el-button type="primary" @click="visible = false">{{ $t('common.close') }}</el-button>
        </template>

        <RunDetailDrawer v-model:visible="detailVisible" :run-id="detailRunId" />
    </el-dialog>
</template>

<script lang="ts" setup>
import { TableColumn } from '@/components/data-table';
import DataTable from '@/components/data-table/DataTable.vue';
import { SearchItem } from '@/components/data-table/SearchForm';
import { defineAsyncComponent, reactive, ref, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import { agentRunApi } from './api';
import { ProfileEnum, RunRoleEnum, RunStatusEnum, TriggerTypeEnum } from './enums';
import { enumLabel, enumTagType } from '@/common/Enum';
import { profileOptionsApi } from './useAgentProfiles';
import type { AgentRun } from './types';

const RunDetailDrawer = defineAsyncComponent(() => import('./RunDetailDrawer.vue'));

const props = defineProps<{
    /** 任务 key（triggerRef 过滤值） */
    taskKey: string;
    /** 任务名（弹窗标题展示） */
    taskName: string;
}>();

const visible = defineModel<boolean>('visible', { required: true });

const { t } = useI18n();
const router = useRouter();

const searchItems = [
    SearchItem.select('status', 'agentops.runStatus').withEnum(RunStatusEnum),
    SearchItem.select('profileId', 'agentops.profile').withOptionsApi(profileOptionsApi()),
];

const columns = ref([
    TableColumn.new('runNo', 'agentops.runNo'),
    // 标题带形态标签，与全局执行历史同口径（见 RunList.vue）
    TableColumn.new('title', 'agentops.title').isSlot(),
    TableColumn.new('triggerType', 'agentops.triggerType').typeTag(TriggerTypeEnum),
    TableColumn.new('profileId', 'agentops.profile').typeTag(ProfileEnum),
    TableColumn.new('status', 'agentops.runStatus').typeTag(RunStatusEnum),
    TableColumn.new('totalTokens', 'agentops.totalTokens').alignCenter().isToken(),
    TableColumn.new('costMs', 'agentops.costMs').alignCenter().isDuration(),
    TableColumn.new('startedAt', 'agentops.startedAt').isTime(),
    TableColumn.new('action', 'common.operation')
        .setActions<AgentRun>([{ key: 'detail', label: 'agentops.detail', type: 'primary', onClick: (row) => openDetail(row) }])
        .fixedRight()
        .alignCenter(),
]);

const dataTableRef = useTemplateRef<InstanceType<typeof DataTable>>('dataTableRef');

// destroy-on-close：每次打开重新挂载，setup 期即固化任务 key 过滤（早于 DataTable 首次加载）。
// 注意：defineAsyncComponent 常驻渲染下 setup 只执行一次（destroy-on-close 不重建 setup），
// 故 taskKey 变化须用 watch 同步，否则弹窗永远停留在首次打开任务的过滤值（曾致列表显示全量记录）
const params = reactive({
    pageNum: 1,
    pageSize: 0,
    triggerRef: props.taskKey,
    status: null,
    profileId: null,
});

watch(
    () => props.taskKey,
    (key) => {
        params.triggerRef = key;
        params.pageNum = 1;
        if (visible.value) {
            dataTableRef.value?.search();
        }
    }
);

const detailVisible = ref(false);
const detailRunId = ref<number | null>(null);

const openDetail = (row: AgentRun) => {
    detailRunId.value = row.id;
    detailVisible.value = true;
};

/** 深度操作（取消/回放/反馈/评测）在全局执行历史页完成，携带任务 key 过滤跳转 */
const openFullHistory = () => {
    visible.value = false;
    router.push({ name: 'AgentopsRunList', query: { triggerRef: props.taskKey } });
};

/** 弹窗内刷新（预留：后续在弹窗内接入触发/取消动作时使用） */
const search = () => {
    dataTableRef.value?.search();
};
defineExpose({ search });
</script>

<style lang="scss" scoped>
.run-history-dialog__body {
    height: 60vh;
}
</style>
