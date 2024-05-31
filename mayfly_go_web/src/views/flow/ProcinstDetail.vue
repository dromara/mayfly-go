<template>
    <div>
        <el-drawer :title="props.title" v-model="visible" :before-close="cancel" size="40%" :close-on-click-modal="!props.instTaskId">
            <template #header>
                <DrawerHeader :header="title" :back="cancel" />
            </template>

            <div>
                <el-divider content-position="left">流程信息</el-divider>
                <el-descriptions :column="2" border>
                    <el-descriptions-item label="流程名">{{ procinst.procdefName }}</el-descriptions-item>
                    <el-descriptions-item label="业务">
                        <enum-tag :enums="FlowBizType" :value="procinst.bizType"></enum-tag>
                    </el-descriptions-item>

                    <el-descriptions-item label="发起人">
                        <AccountInfo :account-id="procinst.creatorId" :username="procinst.creator" />
                        <!-- {{ procinst.creator }} -->
                    </el-descriptions-item>
                    <el-descriptions-item label="发起时间">{{ formatDate(procinst.createTime) }}</el-descriptions-item>

                    <div v-if="procinst.duration">
                        <el-descriptions-item label="持续时间">{{ formatTime(procinst.duration) }}</el-descriptions-item>
                        <el-descriptions-item label="结束时间">{{ formatDate(procinst.endTime) }}</el-descriptions-item>
                    </div>

                    <el-descriptions-item label="流程状态">
                        <enum-tag :enums="ProcinstStatus" :value="procinst.status"></enum-tag>
                    </el-descriptions-item>
                    <el-descriptions-item label="业务状态">
                        <enum-tag :enums="ProcinstBizStatus" :value="procinst.bizStatus"></enum-tag>
                    </el-descriptions-item>

                    <el-descriptions-item label="备注">
                        {{ procinst.remark }}
                    </el-descriptions-item>
                </el-descriptions>
            </div>

            <div>
                <el-divider content-position="left">审批节点</el-divider>
                <procdef-tasks :tasks="procinst?.procdef?.tasks" :procinst-tasks="procinst.procinstTasks" />
            </div>

            <div>
                <el-divider content-position="left">业务信息</el-divider>
                <component
                    v-if="procinst.bizType"
                    ref="keyValueRef"
                    :is="bizComponents[procinst.bizType]"
                    :biz-key="procinst.bizKey"
                    :biz-form="procinst.bizForm"
                >
                </component>
            </div>

            <div v-if="props.instTaskId">
                <el-divider content-position="left">审批表单</el-divider>
                <el-form :model="form" label-width="auto">
                    <el-form-item prop="status" label="结果" required>
                        <el-select v-model="form.status" placeholder="请选择审批结果">
                            <el-option :label="ProcinstTaskStatus.Pass.label" :value="ProcinstTaskStatus.Pass.value"> </el-option>
                            <!-- <el-option :label="ProcinstTaskStatus.Back.label" :value="ProcinstTaskStatus.Back.value"> </el-option> -->
                            <el-option :label="ProcinstTaskStatus.Reject.label" :value="ProcinstTaskStatus.Reject.value"> </el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item prop="remark" label="备注">
                        <el-input v-model.trim="form.remark" placeholder="备注" type="textarea" clearable></el-input>
                    </el-form-item>
                </el-form>
            </div>

            <template #footer v-if="props.instTaskId">
                <div>
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">确 定</el-button>
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

const DbSqlExecBiz = defineAsyncComponent(() => import('./flowbiz/DbSqlExecBiz.vue'));
const RedisRunWriteCmdBiz = defineAsyncComponent(() => import('./flowbiz/RedisRunWriteCmdBiz.vue'));

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
const bizComponents = shallowReactive({
    db_sql_exec_flow: DbSqlExecBiz,
    redis_run_write_cmd_flow: RedisRunWriteCmdBiz,
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
