<template>
    <div>
        <el-drawer
            :title="props.title"
            v-model="visible"
            :before-close="cancel"
            size="50%"
            body-class="!p-2"
            header-class="!mb-2"
            :close-on-click-modal="!props.instTaskId"
        >
            <template #header>
                <DrawerHeader :header="title" :back="cancel" />
            </template>

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
                            <!-- <el-option :label="ProcinstTaskStatus.Back.label" :value="ProcinstTaskStatus.Back.value"> </el-option> -->
                            <el-option :label="$t(ProcinstTaskStatus.Reject.label)" :value="ProcinstTaskStatus.Reject.value"> </el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item prop="remark" :label="$t('common.remark')">
                        <el-input v-model.trim="form.remark" :placeholder="$t('common.remark')" type="textarea" clearable></el-input>
                    </el-form-item>
                </el-form>
            </div>

            <div v-if="flowDef">
                <el-divider content-position="left">{{ $t('flow.approveNode') }}</el-divider>
                <FlowDesign height="300px" disabled center :data="flowDef" />
            </div>

            <template #footer v-if="props.instTaskId">
                <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, defineAsyncComponent, shallowReactive } from 'vue';
import { procinstApi, procinstTaskApi } from './api';
import { ElMessage } from 'element-plus';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { FlowBizType, ProcinstBizStatus, ProcinstTaskStatus, ProcinstStatus } from './enums';
import { formatTime } from '@/common/utils/format';
import EnumTag from '@/components/enumtag/EnumTag.vue';
import AccountInfo from '@/views/system/account/components/AccountInfo.vue';
import { formatDate } from '@/common/utils/format';
import FlowDesign from './components/flowdesign/FlowDesign.vue';

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
const bizComponents: any = shallowReactive({
    db_sql_exec_flow: DbSqlExecBiz,
    redis_run_cmd_flow: RedisRunCmdBiz,
});

const state = reactive({
    procinst: {} as any,
    flowDef: null as any,
    tasks: [] as any,
    form: {
        status: ProcinstTaskStatus.Pass.value,
        remark: '',
    },
    saveBtnLoading: false,
    sortable: '' as any,
});

const { procinst, flowDef, form, saveBtnLoading } = toRefs(state);

watch(
    () => props.procinstId,
    async (newValue: any) => {
        if (!newValue) {
            state.procinst = {};
            state.flowDef = null;
            return;
        }

        state.procinst = await procinstApi.detail.request({ id: newValue });

        const flowdef = JSON.parse(state.procinst.flowDef);
        procinstApi.hisOp.request({ id: newValue }).then((res: any) => {
            const nodeKey2Ops = res.reduce(
                (acc: { [x: string]: any[] }, item: { nodeKey: any }) => {
                    const key = item.nodeKey;
                    if (!acc[key]) {
                        acc[key] = [];
                    }
                    acc[key].push(item);
                    return acc;
                },
                {} as Record<string, typeof res>
            );

            const nodeKey2Tasks = state.procinst.procinstTasks.reduce(
                (acc: { [x: string]: any[] }, item: { nodeKey: any }) => {
                    const key = item.nodeKey;
                    if (!acc[key]) {
                        acc[key] = [];
                    }
                    acc[key].push(item);
                    return acc;
                },
                {} as Record<string, typeof res>
            );

            flowdef.nodes.forEach((node: any) => {
                const key = node.key;
                if (nodeKey2Ops[key]) {
                    // 将操作记录挂载到 node 下，例如命名为 historyList
                    node.extra.opLog = nodeKey2Ops[key][0];
                    node.extra.tasks = nodeKey2Tasks[key];
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
        ElMessage.success('操作成功');
        cancel();
        emit('val-change');
    } finally {
        state.saveBtnLoading = false;
    }
};

const cancel = () => {
    visible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
