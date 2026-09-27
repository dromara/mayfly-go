<template>
    <div>
        <auto-form-drawer
            ref="drawerRef"
            v-model:visible="dialogVisible"
            :title="title"
            :items="items"
            :data="editData"
            size="1000px"
            :confirm-api="onConfirm"
            @opened="onOpened"
            @submitted="emit('submitSuccess')"
            @cancel="emit('cancel')"
        >
            <!-- 脚本入参表单定义（v1 JSON Schema 表格编辑器） -->
            <template #params>
                <auto-form-schema-edit v-model="params" />
            </template>

            <template #footer>
                <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
                <el-button v-auth="'machine:script:save'" type="primary" :loading="drawerRef?.submitting" @click="drawerRef?.submit?.()">
                    {{ $t('common.save') }}
                </el-button>
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { AutoFormDrawer, AutoFormSchemaEdit, defineFormItems, type AutoFormInstance } from '@/components/auto-form';
import { isJsonFormSchema, type AutoFormJsonSchema } from '@/components/auto-form/json';
import { useAutoFormModel } from '@/hooks/useAutoFormModel';
import { computed, ref, watch, useTemplateRef, type PropType } from 'vue';
import { machineApi } from './api';
import { COMMON_SCRIPT_MACHINE_ID, ScriptResultEnum } from './enums';
import type { MachineScriptForm, MachineScriptVO } from './types';

const props = defineProps({
    data: {
        type: Object as PropType<MachineScriptVO | null>,
        default: null,
    },
    title: {
        type: String,
    },
    machineId: {
        type: Number,
    },
    isCommon: {
        type: Boolean,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits(['cancel', 'submitSuccess']);

const categorys = ref<string[]>([]);

const drawerRef = useTemplateRef<AutoFormInstance>('drawerRef');

/** 表单声明（defineFormItems<MachineScriptForm>，渲染 + 校验唯一数据源；params 走插槽承载 AutoFormSchemaEdit） */
const items = defineFormItems<MachineScriptForm>([
    { prop: 'name', label: 'common.name', required: true },
    { prop: 'description', label: 'common.remark', required: true },
    { prop: 'type', label: 'common.type', type: 'enum', enums: ScriptResultEnum, required: true },
    {
        prop: 'category',
        label: 'machine.category',
        type: 'select',
        placeholder: 'machine.categoryTips',
        // 允许输入新分类（可创建）
        props: { allowCreate: true },
        options: () => Promise.resolve(categorys.value.map((x) => ({ value: x, label: x }))),
    },
    { prop: 'params', label: 'machine.scriptParam', type: 'custom', tooltip: ['machine.scriptParamTips1', 'machine.scriptParamTips2'] },
    { prop: 'script', label: 'machine.script', type: 'monaco', required: true, props: { language: 'shell', height: '300px' } },
]);

/** 脚本入参表单定义（v1 JSON Schema，与表单字段并行维护，提交时序列化进 form.params） */
const params = ref<AutoFormJsonSchema>({ version: 1, fields: [] });

const defaultForm: MachineScriptForm = {
    id: null,
    name: '',
    machineId: 0,
    description: '',
    script: '',
    params: '',
    type: null,
    category: '',
};

/** 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成） */
const editData = computed<MachineScriptForm>(() => {
    const data = props.data;
    if (!data) {
        return { ...defaultForm };
    }
    // 逐字段映射而非整包展开：脚本 VO 字段多为可选，表单必填字段统一由 defaultForm 兑底
    return {
        ...defaultForm,
        id: data.id ?? null,
        name: data.name ?? '',
        machineId: data.machineId ?? 0,
        description: data.description ?? '',
        script: data.script ?? '',
        params: data.params ?? '',
        type: data.type ?? null,
        category: data.category ?? '',
    };
});

// 宿主内部表单在 @opened 接管（提交组装基于它）
const { onOpened, requireForm } = useAutoFormModel<MachineScriptForm>();

// 抽屉打开时加载分类选项，并解析编辑数据中的入参 schema
watch(dialogVisible, (v) => {
    if (!v) {
        return;
    }
    machineApi.scriptCategorys.request().then((res) => {
        // 无任何脚本时接口返回 null，直接赋值会让 options 回调 .map 抛错
        categorys.value = res ?? [];
    });
    const raw = props.data?.params;
    if (raw) {
        try {
            const parsed = JSON.parse(raw);
            params.value = isJsonFormSchema(parsed) ? parsed : { version: 1, fields: [] };
        } catch {
            params.value = { version: 1, fields: [] };
        }
    } else {
        params.value = { version: 1, fields: [] };
    }
});

// confirmApi 提交动作：machineId/params 由页面派生（不在表单字段里）；成功提示、关闭抽屉由组件内置逻辑处理
const onConfirm = async () => {
    const form = requireForm();
    form.machineId = props.isCommon ? COMMON_SCRIPT_MACHINE_ID : (props.machineId ?? 0);
    if (params.value) {
        form.params = JSON.stringify(params.value);
    }
    await machineApi.saveScript.request(form);
};
</script>
<style lang="scss"></style>
