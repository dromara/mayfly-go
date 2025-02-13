<template>
    <div>
        <el-drawer :title="props.title" v-model="visible" :before-close="cancel" size="50%" :close-on-click-modal="!props.instTaskId">
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
                        <AccountInfo :account-id="procinst.creatorId" :username="procinst.creator" />
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
                <el-divider content-position="left">{{ $t('flow.approveNode') }}</el-divider>
                <procdef-tasks :tasks="procinst?.procdef?.tasks" :procinst-tasks="procinst.procinstTasks" />
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

            <template #footer v-if="props.instTaskId">
                <div>
                    <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, defineAsyncComponent, shallowReactive } from 'vue';
import { procinstApi } from './api';
import { ElMessage } from 'element-plus';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { FlowBizType, ProcinstBizStatus, ProcinstTaskStatus, ProcinstStatus } from './enums';
import ProcdefTasks from './components/ProcdefTasks.vue';
import { formatTime } from '@/common/utils/format';
import EnumTag from '@/components/enumtag/EnumTag.vue';
import AccountInfo from '@/views/system/account/components/AccountInfo.vue';
import { formatDate } from '@/common/utils/format';

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
    tasks: [] as any,
    form: {
        status: ProcinstTaskStatus.Pass.value,
        remark: '',
    },
    saveBtnLoading: false,
    sortable: '' as any,
});

const { procinst, form, saveBtnLoading } = toRefs(state);

watch(
    () => props.procinstId,
    async (newValue: any) => {
        if (newValue) {
            state.procinst = await procinstApi.detail.request({ id: newValue });
        } else {
            state.procinst = {};
        }
    }
);

const btnOk = async () => {
    const status = state.form.status;
    let api = procinstApi.completeTask;
    if (status === ProcinstTaskStatus.Back.value) {
        api = procinstApi.backTask;
    } else if (status === ProcinstTaskStatus.Reject.value) {
        api = procinstApi.rejectTask;
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
