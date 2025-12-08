<template>
    <div>
        <el-drawer :title="props.title" v-model="visible" :before-close="cancel" :destroy-on-close="true" :close-on-click-modal="false" size="50%">
            <template #header>
                <DrawerHeader :header="title" :back="cancel" />
            </template>

            <el-form :model="modelValue" ref="formRef" :rules="rules" label-width="auto">
                <el-form-item prop="bizType" :label="$t('flow.bizType')">
                    <EnumSelect v-model="modelValue.bizType" :enums="FlowBizType" @change="changeBizType" />
                </el-form-item>

                <el-form-item prop="remark" :label="$t('common.remark')">
                    <el-input v-model.trim="modelValue.remark" type="textarea" auto-complete="off" clearable></el-input>
                </el-form-item>

                <el-divider content-position="left">{{ $t('flow.bizInfo') }}</el-divider>
                <component
                    ref="bizFormRef"
                    v-if="modelValue.bizType"
                    :is="bizComponents[modelValue.bizType]"
                    v-model:bizForm="modelValue.bizForm"
                    @changeResourceCode="changeResourceCode"
                >
                </component>
            </el-form>

            <span v-if="flowProcdef || !modelValue.procdefId">
                <el-divider content-position="left">{{ $t('flow.approvalNode') }}</el-divider>

                <FlowDesign height="300px" v-if="flowProcdef" :data="flowProcdef.flowDef" disabled center />

                <el-result v-if="!modelValue.procdefId" icon="error" :title="$t('flow.approvalNodeNotExist')" :sub-title="$t('flow.resourceNotExistFlow')">
                </el-result>
            </span>

            <template #footer>
                <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="saveBtnLoading" @click="btnOk" :disabled="!modelValue.procdefId">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, defineAsyncComponent, shallowReactive, useTemplateRef, watch, onMounted } from 'vue';
import { procdefApi, procinstApi } from './api';
import { ElMessage } from 'element-plus';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { FlowBizType } from './enums';
import EnumSelect from '@/components/enumselect/EnumSelect.vue';
import RedisRunCmdFlowBizForm from './flowbiz/redis/RedisRunCmdFlowBizForm.vue';
import { useI18n } from 'vue-i18n';
import { Rules } from '@/common/rule';
import FlowDesign from './components/flowdesign/FlowDesign.vue';

const DbSqlExecFlowBizForm = defineAsyncComponent(() => import('./flowbiz/dbms/DbSqlExecFlowBizForm.vue'));

const { t } = useI18n();

const props = defineProps({
    title: {
        type: String,
    },
});

const visible = defineModel<boolean>('visible', { default: false });

const modelValue = defineModel('modelValue', {
    default: () => ({
        bizType: FlowBizType.DbSqlExec.value,
        procdefId: 0,
        status: null,
        remark: '',
        bizKey: '',
        bizForm: {},
    }),
});

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
    bizType: [Rules.requiredSelect('flow.bizType')],
    remark: [Rules.requiredInput('common.remark')],
};

const state = reactive({
    tasks: [] as any,
    flowProcdef: null as any,
    sortable: '' as any,
});

const { flowProcdef } = toRefs(state);

const { isFetching: saveBtnLoading, execute: procinstStart } = procinstApi.start.useApi(modelValue);

watch(
    () => modelValue.value.procdefId,
    async () => {
        if (!modelValue.value.procdefId || state.flowProcdef) {
            return;
        }
        state.flowProcdef = await procdefApi.detail.request({ id: modelValue.value.procdefId });
    }
);

const changeResourceCode = async (resourceType: any, code: string) => {
    state.flowProcdef = await procdefApi.getByResource.request({ resourceType, resourceCode: code });
    if (!state.flowProcdef) {
        modelValue.value.procdefId = 0;
    } else {
        modelValue.value.procdefId = state.flowProcdef.id;
    }
};

const changeBizType = () => {
    //重置流程定义ID
    modelValue.value.procdefId = 0;
    state.flowProcdef = null;
    modelValue.value.bizForm = {};
};

const btnOk = async () => {
    try {
        await formRef.value.validate();
        await bizFormRef.value.validateBizForm();
    } catch (e: any) {
        ElMessage.error(t('flow.procinstFormError'));
        return false;
    }

    await procinstStart();
    ElMessage.success(t('flow.procinstStartSuccess'));
    emit('val-change', modelValue.value);
    //重置表单域
    cancel();
};

const cancel = () => {
    visible.value = false;
    emit('cancel');
    state.flowProcdef = null;
    formRef.value.resetFields();
    bizFormRef.value.resetBizForm();

    setTimeout(() => {
        modelValue.value = {} as any;
    }, 500);
};
</script>
<style lang="scss"></style>
