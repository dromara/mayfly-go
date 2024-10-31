<template>
    <div>
        <el-drawer :title="props.title" v-model="visible" :before-close="cancel" :destroy-on-close="true" :close-on-click-modal="false" size="50%">
            <template #header>
                <DrawerHeader :header="title" :back="cancel" />
            </template>

            <el-form :model="form" ref="formRef" :rules="rules" label-width="auto">
                <el-form-item prop="bizType" label="业务类型">
                    <EnumSelect v-model="form.bizType" :enums="FlowBizType" placeholder="请选择业务类型" />
                </el-form-item>

                <el-form-item prop="remark" label="备注">
                    <el-input v-model.trim="form.remark" type="textarea" placeholder="备注" auto-complete="off" clearable></el-input>
                </el-form-item>

                <el-divider content-position="left">业务信息</el-divider>
                <component
                    ref="bizFormRef"
                    v-if="form.bizType"
                    :is="bizComponents[form.bizType]"
                    v-model:bizForm="form.bizForm"
                    @changeResourceCode="changeResourceCode"
                >
                </component>
            </el-form>

            <span v-if="flowProcdef || !state.form.procdefId">
                <el-divider content-position="left">审批节点</el-divider>

                <ProcdefTasks v-if="flowProcdef" :procdef="flowProcdef" />

                <el-result v-if="!state.form.procdefId" icon="error" title="不存在审批节点" sub-title="该资源无需审批操作"> </el-result>
            </span>

            <template #footer>
                <div>
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk" :disabled="!state.form.procdefId">确 定</el-button>
                </div>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, defineAsyncComponent, shallowReactive, useTemplateRef } from 'vue';
import { procdefApi, procinstApi } from './api';
import { ElMessage } from 'element-plus';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { FlowBizType } from './enums';
import EnumSelect from '@/components/enumselect/EnumSelect.vue';
import ProcdefTasks from './components/ProcdefTasks.vue';
import RedisRunCmdFlowBizForm from './flowbiz/redis/RedisRunCmdFlowBizForm.vue';

const DbSqlExecFlowBizForm = defineAsyncComponent(() => import('./flowbiz/dbms/DbSqlExecFlowBizForm.vue'));

const props = defineProps({
    title: {
        type: String,
    },
});

const visible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

const formRef: any = useTemplateRef('formRef');
const bizFormRef: any = useTemplateRef('bizFormRef');

// 业务组件
const bizComponents: any = shallowReactive({
    db_sql_exec_flow: DbSqlExecFlowBizForm,
    redis_run_cmd_flow: RedisRunCmdFlowBizForm,
});

const rules = {
    bizType: [
        {
            required: true,
            message: '请选择流程业务类型',
            trigger: ['change', 'blur'],
        },
    ],
    remark: [
        {
            required: true,
            message: '请输入申请备注',
            trigger: ['change', 'blur'],
        },
    ],
};

const defaultForm = {
    bizType: FlowBizType.DbSqlExec.value,
    procdefId: -1,
    status: null,
    remark: '',
    bizForm: {},
};

const state = reactive({
    tasks: [] as any,
    form: { ...defaultForm },
    flowProcdef: null as any,
    sortable: '' as any,
});

const { form, flowProcdef } = toRefs(state);

const { isFetching: saveBtnLoading, execute: procinstStart } = procinstApi.start.useApi(form);

const changeResourceCode = async (resourceType: any, code: string) => {
    state.flowProcdef = await procdefApi.getByResource.request({ resourceType, resourceCode: code });
    if (!state.flowProcdef) {
        state.form.procdefId = 0;
    } else {
        state.form.procdefId = state.flowProcdef.id;
    }
};

const btnOk = async () => {
    try {
        await formRef.value.validate();
        await bizFormRef.value.validateBizForm();
    } catch (e: any) {
        ElMessage.error('请正确填写信息');
        return false;
    }

    await procinstStart();
    ElMessage.success('流程发起成功');
    emit('val-change', state.form);
    //重置表单域
    cancel();
};

const cancel = () => {
    visible.value = false;
    emit('cancel');
    state.flowProcdef = null;
    formRef.value.resetFields();
    bizFormRef.value.resetBizForm();

    state.form = { ...defaultForm };
};
</script>
<style lang="scss"></style>
