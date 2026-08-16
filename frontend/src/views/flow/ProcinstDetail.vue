<template>
    <el-drawer
        :append-to-body="false"
        :title="props.title"
        v-model="visible"
        :before-close="cancel"
        size="50%"
        body-class="p-2!"
        header-class="mb-2!"
        :destroy-on-close="true"
        :close-on-click-modal="!props.instTaskId"
    >
        <template #header>
            <DrawerHeader :header="title" :back="cancel" />
        </template>

        <el-tabs v-model="state.activeTab">
            <el-tab-pane :label="$t('common.basic')" name="basic">
                <div>
                    <el-divider content-position="left">{{ $t('flow.proc') }}</el-divider>
                    <el-descriptions :column="3" border>
                        <el-descriptions-item :span="1" :label="$t('flow.procdefName')">{{ procinst.procdefName }}</el-descriptions-item>
                        <el-descriptions-item :span="1" :label="$t('flow.bizType')">
                            <enum-tag :enums="FlowBizType" :value="procinst.bizType"></enum-tag>
                        </el-descriptions-item>
                        <el-descriptions-item :span="1" :label="$t('flow.initiator')">
                            <AccountInfo :username="procinst.creator || ''" />
                        </el-descriptions-item>

                        <el-descriptions-item :span="1" :label="$t('flow.procinstStatus')">
                            <enum-tag :enums="ProcinstStatus" :value="procinst.status"></enum-tag>
                        </el-descriptions-item>
                        <el-descriptions-item :span="1" :label="$t('flow.bizStatus')">
                            <enum-tag :enums="ProcinstBizStatus" :value="procinst.bizStatus"></enum-tag>
                        </el-descriptions-item>
                        <el-descriptions-item :span="1" :label="$t('flow.startingTime')">{{ formatDate(procinst.createTime) }}</el-descriptions-item>

                        <div v-if="procinst.duration">
                            <el-descriptions-item :span="1.5" :label="$t('flow.endTime')">{{ formatDate(procinst.endTime) }}</el-descriptions-item>
                            <el-descriptions-item :span="1.5" :label="$t('flow.duration')">{{ formatTime(procinst.duration) }}</el-descriptions-item>
                        </div>

                        <el-descriptions-item :span="3" :label="$t('common.remark')">
                            {{ procinst.remark }}
                        </el-descriptions-item>
                    </el-descriptions>
                </div>

                <div>
                    <el-divider content-position="left">{{ $t('flow.bizInfo') }}</el-divider>
                    <component v-if="procinst.bizType" ref="keyValueRef" :is="bizComponents[procinst.bizType]" :procinst="procinst"> </component>
                </div>

                <div v-if="props.instTaskId">
                    <el-divider content-position="left">{{ $t('flow.approveForm') }}</el-divider>
                    <el-form :model="form" label-width="auto">
                        <el-form-item prop="status" :label="$t('flow.approveResult')" required>
                            <el-select v-model="form.status">
                                <el-option :label="$t(ProcinstTaskStatus.Pass.label)" :value="ProcinstTaskStatus.Pass.value"> </el-option>
                                <el-option :label="$t(ProcinstTaskStatus.Back.label)" :value="ProcinstTaskStatus.Back.value"> </el-option>
                                <el-option :label="$t(ProcinstTaskStatus.Reject.label)" :value="ProcinstTaskStatus.Reject.value"> </el-option>
                            </el-select>
                        </el-form-item>
                        <el-form-item prop="remark" :label="$t('common.remark')">
                            <el-input v-model.trim="form.remark" :placeholder="$t('common.remark')" type="textarea" clearable></el-input>
                        </el-form-item>
                    </el-form>
                </div>

                <div v-if="flowDef" class="h-75">
                    <el-divider content-position="left">{{ $t('flow.approveNode') }}</el-divider>
                    <FlowDesign disabled center :data="flowDef" />
                </div>
            </el-tab-pane>

            <el-tab-pane :label="$t('flow.approvalRecord')" name="approvalRecord">
                <el-timeline>
                    <el-timeline-item
                        v-for="task in procinst.procinstTasks"
                        :key="task.id"
                        :timestamp="formatDate(task.createTime)"
                        :type="getTaskStatusType(task.status)"
                        :icon="getTaskStatusIcon(task.status)"
                        size="large"
                        placement="top"
                    >
                        <el-card shadow="hover" class="hover:shadow-md transition-shadow">
                            <div>
                                <div class="flex justify-between">
                                    <div>
                                        <el-text tag="b" size="large">{{ task.nodeName }}</el-text> -
                                        <el-text tag="b" size="large" type="primary">{{ task.handler || '/' }}</el-text>
                                    </div>
                                    <enum-tag :enums="ProcinstTaskStatus" :value="task.status" />
                                </div>

                                <div class="mt-2">
                                    <el-text class="ml-5" tag="b">{{ task.remark }}</el-text>
                                </div>
                            </div>
                        </el-card>
                    </el-timeline-item>
                </el-timeline>
            </el-tab-pane>
        </el-tabs>

        <template #footer v-if="props.instTaskId">
            <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-drawer>
</template>

<script lang="ts" setup>
import { formatDate, formatTime } from '@/common/utils/format';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import EnumTag from '@/components/enum-tag/EnumTag.vue';
import { Msg } from '@/hooks/useI18n';
import AccountInfo from '@/views/system/account/components/AccountInfo.vue';
import { defineAsyncComponent, reactive, shallowReactive, toRefs, watch } from 'vue';
import { procinstApi, procinstTaskApi } from './api';
import FlowDesign from './components/flowdesign/FlowDesign.vue';
import { FlowBizType, ProcinstBizStatus, ProcinstStatus, ProcinstTaskStatus } from './enums';
import type { Procinst, ProcinstTask, HisProcinstOp, FlowNode, FlowDef } from './types';

const DbSqlExecBiz = defineAsyncComponent(() => import('./flowbiz/dbms/DbSqlExecBiz.vue'));
const RedisRunCmdBiz = defineAsyncComponent(() => import('./flowbiz/redis/RedisRunCmdBiz.vue'));

const props = defineProps({
    procinstId: {
        type: Number,
    },
    // 流程实例任务id（存在则展示审批相关信息）
    instTaskId: {
        type: Number,
    },
    title: {
        type: String,
    },
});

const visible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

// 业务组件
const bizComponents = shallowReactive<Record<string, unknown>>({
    db_sql_exec_flow: DbSqlExecBiz,
    redis_run_cmd_flow: RedisRunCmdBiz,
});

const state = reactive({
    activeTab: 'basic',
    procinst: {} as Procinst,
    flowDef: null as FlowDef | null,
    tasks: [] as unknown[],
    form: {
        status: ProcinstTaskStatus.Pass.value as number,
        remark: '',
    },
    saveBtnLoading: false,
    sortable: '' as string,
});

const { procinst, flowDef, form, saveBtnLoading } = toRefs(state);

watch(
    () => props.procinstId,
    async (newValue: number | undefined) => {
        state.form.status = ProcinstTaskStatus.Pass.value;
        state.form.remark = '';

        if (!newValue) {
            state.procinst = {} as Procinst;
            state.flowDef = null;
            return;
        }

        state.procinst = await procinstApi.detail.request({ id: newValue });

        const flowdef = JSON.parse(state.procinst.flowDef) as FlowDef;
        procinstApi.hisOp.request({ id: newValue }).then((res: HisProcinstOp[]) => {
            const nodeKey2Ops = res.reduce(
                (acc: Record<string, HisProcinstOp[]>, item: HisProcinstOp) => {
                    const key = item.nodeKey;
                    if (!acc[key]) {
                        acc[key] = [];
                    }
                    acc[key].push(item);
                    return acc;
                },
                {} as Record<string, HisProcinstOp[]>
            );

            const nodeKey2Tasks = state.procinst.procinstTasks?.reduce(
                (acc: Record<string, ProcinstTask[]>, item: ProcinstTask) => {
                    const key = item.nodeKey;
                    if (!acc[key]) {
                        acc[key] = [];
                    }
                    acc[key].push(item);
                    return acc;
                },
                {} as Record<string, ProcinstTask[]>
            );

            flowdef.nodes.forEach((node: FlowNode) => {
                const key = node.key;
                if (nodeKey2Ops[key]) {
                    if (!node.extra) {
                        node.extra = {};
                    }
                    node.extra.opLog = nodeKey2Ops[key][0];
                    node.extra.tasks = nodeKey2Tasks?.[key];
                }
            });

            state.flowDef = flowdef;
        });
    }
);

const btnOk = async () => {
    const status = state.form.status;
    let api = procinstTaskApi.passTask;
    if (status === ProcinstTaskStatus.Back.value) {
        api = procinstTaskApi.backTask;
    } else if (status === ProcinstTaskStatus.Reject.value) {
        api = procinstTaskApi.rejectTask;
    }

    try {
        state.saveBtnLoading = true;
        await api.request({ id: props.instTaskId, remark: state.form.remark });
        Msg.operateSuccess();
        cancel();
        emit('val-change');
    } finally {
        state.saveBtnLoading = false;
    }
};

const cancel = () => {
    visible.value = false;
    state.activeTab = 'basic';
    emit('cancel');
};

const getTaskStatusIcon = (status: number) => {
    if (status === ProcinstTaskStatus.Pass.value) {
        return 'Check';
    } else if (status === ProcinstTaskStatus.Back.value) {
        return 'Close';
    } else if (status === ProcinstTaskStatus.Reject.value) {
        return 'Close';
    }
    return 'SemiSelect';
};

const getTaskStatusType = (status: number) => {
    if (status === ProcinstTaskStatus.Pass.value) {
        return 'success';
    } else if (status === ProcinstTaskStatus.Back.value) {
        return 'warning';
    } else if (status === ProcinstTaskStatus.Reject.value) {
        return 'danger';
    }
    return 'primary';
};
</script>
<style lang="scss"></style>
