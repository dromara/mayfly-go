<template>
    <div>
        <auto-form-drawer
            ref="drawerRef"
            v-model:visible="visible"
            :title="props.title"
            :items="items"
            :data="modelValue"
            size="50%"
            :confirm-api="btnOk"
            @submitted="onSubmitted"
            @opened="onOpened"
            @cancel="onCancel"
        >
            <!-- 业务表单区（动态组件承载，自含校验）+ 审批节点展示 -->
            <template #body-extra="{ form }">
                <el-divider content-position="left">{{ $t('flow.bizInfo') }}</el-divider>
                <component
                    ref="bizFormRef"
                    v-if="form.bizType"
                    :is="bizComponents[form.bizType]"
                    v-model:bizForm="form.bizForm"
                    @changeResourceCode="changeResourceCode"
                >
                </component>

                <span v-if="flowProcdef || !form.procdefId">
                    <el-divider content-position="left">{{ $t('flow.approvalNode') }}</el-divider>

                    <FlowDesign height="300px" v-if="flowProcdef" :data="flowProcdef.flowDef" disabled center />

                    <el-result v-if="!form.procdefId" icon="error" :title="$t('flow.approvalNodeNotExist')" :sub-title="$t('flow.resourceNotExistFlow')"> </el-result>
                </span>
            </template>

            <template #footer>
                <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="drawerRef?.submitting" @click="drawerRef?.submit()" :disabled="!internalForm?.procdefId">{{ $t('common.confirm') }}</el-button>
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import { Msg } from '@/hooks/useI18n';
import { computed, defineAsyncComponent, reactive, ref, shallowReactive, toRefs, useTemplateRef, watch } from 'vue';
import { procdefApi, procinstApi } from './api';
import FlowDesign from './components/flowdesign/FlowDesign.vue';
import { FlowBizType } from './enums';
import type { Procdef, ProcInstStartForm } from './types';
import RedisRunCmdFlowBizForm from './flowbiz/redis/RedisRunCmdFlowBizForm.vue';

const DbSqlExecFlowBizForm = defineAsyncComponent(() => import('./flowbiz/dbms/DbSqlExecFlowBizForm.vue'));

const props = defineProps({
    title: {
        type: String,
    },
});

const visible = defineModel<boolean>('visible', { default: false });

const modelValue = defineModel<ProcInstStartForm>('modelValue', {
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

const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown; submitting: boolean; submit: () => Promise<void> }>('drawerRef');
const bizFormRef = useTemplateRef<{ validateBizForm: () => Promise<void>; resetBizForm: () => void }>('bizFormRef');

// 业务组件
const bizComponents = shallowReactive<Record<string, unknown>>({
    db_sql_exec_flow: DbSqlExecFlowBizForm,
    redis_run_cmd_flow: RedisRunCmdFlowBizForm,
});

/** 表单声明（AutoFormItem[]；业务表单由 bizComponents 动态组件承载，自含校验） */
const items: AutoFormItem[] = [
    { prop: 'bizType', label: 'flow.bizType', type: 'enum', enums: FlowBizType, required: true, onChange: (_value, form) => changeBizType(form) },
    { prop: 'remark', label: 'common.remark', type: 'textarea', required: true },
];

const state = reactive({
    flowProcdef: null as Procdef | null,
});

const { flowProcdef } = toRefs(state);

/** 抽屉打开后暂存的内部表单引用（业务表单绑定、提交组装均基于它） */
const internalForm = ref<AutoFormData>();

const onOpened = (form: AutoFormData) => {
    internalForm.value = form;
};

const submitForm = computed(() => internalForm.value as ProcInstStartForm);

const { execute: procinstStart } = procinstApi.start.useApi(submitForm);

watch(
    () => internalForm.value?.procdefId,
    async () => {
        const procdefId = internalForm.value?.procdefId;
        if (!procdefId || state.flowProcdef) {
            return;
        }
        state.flowProcdef = await procdefApi.detail.request({ id: procdefId });
    }
);

const changeResourceCode = async (resourceType: string, code: string) => {
    state.flowProcdef = await procdefApi.getByResource.request({ resourceType, resourceCode: code });
    if (!internalForm.value) {
        return;
    }
    if (!state.flowProcdef) {
        internalForm.value.procdefId = 0;
    } else {
        internalForm.value.procdefId = state.flowProcdef.id;
    }
};

const changeBizType = (form: AutoFormData) => {
    //重置流程定义ID
    form.procdefId = 0;
    state.flowProcdef = null;
    form.bizForm = {};
};

// confirmApi 提交动作：业务表单自含校验（失败抛错中止，组件保持抽屉打开）→ 发起流程；成功提示与关闭抽屉由组件内置逻辑处理
const btnOk = async () => {
    try {
        await bizFormRef.value?.validateBizForm();
    } catch (e: unknown) {
        Msg.error('flow.procinstFormError');
        throw new Error('biz form invalid');
    }

    await procinstStart();
};

// 提交成功后通知父组件并重置抽屉状态（抽屉随即由组件关闭）
const onSubmitted = () => {
    emit('val-change', submitForm.value);
    resetState();
};

/** 重置抽屉状态（打开时 AutoFormDrawer 克隆回填 :data，无需延迟重置） */
const resetState = () => {
    state.flowProcdef = null;
    modelValue.value = {
        bizType: FlowBizType.DbSqlExec.value,
        procdefId: 0,
        status: null,
        remark: '',
        bizKey: '',
        bizForm: {},
    };
};

// @cancel 时抽屉已由 AutoFormDrawer 关闭，仅需重置状态
const onCancel = () => {
    resetState();
};
</script>
<style lang="scss"></style>
