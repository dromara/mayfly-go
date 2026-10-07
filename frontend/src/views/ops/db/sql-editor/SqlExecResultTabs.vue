<template>
    <div class="sql-exec-res h-full!">
        <el-tabs class="h-full! w-full!" v-if="execResTabs.length > 0" @tab-remove="onRemoveTab" @tab-change="onTabChange" :model-value="activeTab">
            <el-tab-pane class="h-full!" closable v-for="dt in execResTabs" :label="dt.id" :name="dt.id" :key="dt.id">
                <template #label>
                    <el-popover :show-after="1000" placement="top-start" :title="$t('db.execInfo')" trigger="hover" :width="300">
                        <template #reference>
                            <div>
                                <span>
                                    <span v-if="dt.loading">
                                        <SvgIcon class="mb-0.5! is-loading" name="Loading" color="var(--el-color-primary)" />
                                    </span>
                                    <span v-else>
                                        <SvgIcon class="mb-0.5!" v-if="dt.notice && !dt.errorMsg" name="Warning" color="var(--el-color-warning)" />
                                        <SvgIcon class="mb-0.5!" v-else-if="dt.errorMsg" name="CircleClose" color="var(--el-color-error)" />
                                        <SvgIcon class="mb-0.5!" v-else name="CircleCheck" color="var(--el-color-success)" />
                                    </span>
                                </span>

                                <span> {{ $t('db.result') }}-{{ dt.id }} </span>
                            </div>
                        </template>
                        <template #default>
                            <el-descriptions v-if="dt.sql" :column="1" size="small">
                                <el-descriptions-item>
                                    <div style="width: 280px">
                                        <el-text size="small" truncated :title="dt.sql"> {{ dt.sql }} </el-text>
                                    </div>
                                </el-descriptions-item>
                                <el-descriptions-item :label="`${$t('db.times')} :`"> {{ dt.execTime }}ms </el-descriptions-item>
                                <el-descriptions-item :label="`${$t('db.resultSet')} :`">
                                    {{ dt.data?.length }}
                                </el-descriptions-item>
                            </el-descriptions>
                        </template>
                    </el-popover>
                </template>

                <el-row>
                    <span v-if="dt.hasUpdatedFields" class="mt-1">
                        <span>
                            <el-link type="success" underline="never" @click="emit('submitUpdateFields', dt)"
                                ><span style="font-size: 12px">{{ $t('common.submit') }}</span></el-link
                            >
                        </span>
                        <span>
                            <el-divider direction="vertical" border-style="dashed" />
                            <el-link type="warning" underline="never" @click="emit('cancelUpdateFields', dt)"
                                ><span style="font-size: 12px">{{ $t('common.cancel') }}</span></el-link
                            >
                        </span>
                    </span>
                </el-row>
                <!-- 批量执行时逐条标注哪些语句被要求审批，并给出直接提单入口 -->
                <div v-if="needApprovalOf(dt).length" class="approval-tip">
                    <span>{{ $t('db.needApprovalTip', { count: needApprovalOf(dt).length }) }}</span>
                    <el-button link type="primary" @click="emit('submitTicket', needApprovalOf(dt)[0], dt)">
                        {{ $t('flow.submitTicket') }}
                    </el-button>
                </div>

                <db-table-data
                    v-if="!dt.errorMsg && !dt.notice"
                    :ref="(el: unknown) => setDbTableRef(dt, el)"
                    :db-id="dbId"
                    :db="db"
                    :data="dt.data"
                    :table="dt.table"
                    :columns="dt.tableColumn"
                    :column-more-actions="['fixed']"
                    :loading="dt.loading"
                    :abort-fn="dt.abortFn"
                    :height="tableDataHeight"
                    :empty-text="tableDataEmptyText"
                    @change-updated-field="emit('changeUpdatedField', $event, dt)"
                    @data-delete="emit('dataDelete', $event, dt)"
                ></db-table-data>

                <!-- 命中「仅提醒」但选择不执行：黄色提醒而不是红色报错，因为它根本没失败 -->
                <el-result v-else-if="dt.notice" icon="warning" :title="$t('flow.warnAckTitle')" :sub-title="dt.notice">
                    <template #extra>
                        <el-button v-if="needApprovalOf(dt).length" type="primary" @click="onSubmitTicket(dt)">
                            {{ $t('flow.submitTicket') }}
                        </el-button>
                    </template>
                </el-result>

                <el-result v-else icon="error" :title="$t('db.execFail')" :sub-title="dt.errorMsg">
                    <template #extra>
                        <!-- 只有策略要求审批的失败才给提单入口：语法错误、权限不足提单也没用 -->
                        <el-button v-if="needApprovalOf(dt).length" type="primary" @click="onSubmitTicket(dt)">
                            {{ $t('flow.submitTicket') }}
                        </el-button>
                    </template>
                </el-result>
            </el-tab-pane>
        </el-tabs>
    </div>
</template>

<script lang="ts" setup>
import SvgIcon from '@/components/svg-icon/index.vue';
import DbTableData from '@/views/ops/db/data-grid/DbTableData.vue';
import type { ExecResTabLike, ExecResTabState, DbTableDataRef } from './composables/useSqlExec';

const props = defineProps<{
    execResTabs: ExecResTabState[];
    activeTab: number;
    dbId: number;
    db: string;
    tableDataHeight: string;
    tableDataEmptyText: string;
}>();

const emit = defineEmits<{
    tabRemove: [targetId: number];
    submitTicket: [sql: string, dt: ExecResTabLike];
    tabChange: [];
    submitUpdateFields: [dt: ExecResTabLike];
    cancelUpdateFields: [dt: ExecResTabLike];
    changeUpdatedField: [hasUpdatedFields: boolean, dt: ExecResTabLike];
    dataDelete: [deleteDatas: Record<string, unknown>[], dt: ExecResTabLike];
}>();

const onRemoveTab = (targetId: number) => {
    emit('tabRemove', targetId);
};

const onTabChange = () => {
    emit('tabChange');
};

// needApprovalOf 该结果里被触发策略要求审批的语句（与「已被禁止执行」互斥）
const needApprovalOf = (dt: ExecResTabState) => dt.needApprovalSqls ?? [];

const onSubmitTicket = (dt: ExecResTabState) => {
    const sqls = needApprovalOf(dt);
    if (sqls.length) {
        emit('submitTicket', sqls[0], dt);
    }
};

const setDbTableRef = (dt: ExecResTabState, el: unknown) => {
    dt.dbTableRef = el as DbTableDataRef | null;
};
</script>
