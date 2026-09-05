<template>
    <div>
        <auto-form-drawer v-model:visible="dialogVisible" :title="title" :items="items" :data="editData" size="1000px" :confirm-api="onConfirm" @submitted="emit('submitSuccess')" @cancel="emit('cancel')">
            <!-- 脚本入参表单定义（v1 JSON Schema 表格编辑器） -->
            <template #params>
                <auto-form-schema-edit v-model="params" />
            </template>

            <template #footer="{ form }">
                <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
                <el-button v-auth="'machine:script:save'" type="primary" @click="onConfirm(form)">
                    {{ $t('common.save') }}
                </el-button>
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { AutoFormDrawer, AutoFormSchemaEdit, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import { isJsonFormSchema, type AutoFormJsonSchema } from '@/components/auto-form/json';
import { computed, ref, toRefs, watch } from 'vue';
import { machineApi } from './api';
import { ScriptResultEnum } from './enums';
import type { MachineScriptForm, MachineScriptVO } from './types';

const props = defineProps({
    data: {
        type: Object as () => MachineScriptVO | null,
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

const { isCommon, machineId } = toRefs(props);
const categorys = ref([] as string[]);

/** 表单声明（AutoFormItem[]，渲染 + 校验唯一数据源；params 走插槽承载 AutoFormSchemaEdit） */
const items: AutoFormItem[] = [
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
];

/** 脚本入参表单定义（v1 JSON Schema，与表单字段并行维护，提交时序列化进 form.params） */
const params = ref({ version: 1, fields: [] } as AutoFormJsonSchema);

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
const editData = computed<AutoFormData | null>(() => {
    if (props.data) {
        return { ...props.data } as unknown as AutoFormData;
    }
    return { ...defaultForm } as unknown as AutoFormData;
});

// 抽屉打开时加载分类选项，并解析编辑数据中的入参 schema
watch(dialogVisible, (v) => {
    if (!v) {
        return;
    }
    machineApi.scriptCategorys.request().then((res: string[]) => {
        categorys.value = res;
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

// confirmApi 提交动作：组装 machineId/params 后走统一提交；成功提示、关闭抽屉由组件内置逻辑处理
const onConfirm = async (rawForm: AutoFormData) => {
    const form = rawForm as MachineScriptForm;
    form.machineId = isCommon.value ? 9999999 : (machineId?.value as number);
    if (params.value) {
        form.params = JSON.stringify(params.value);
    }
    await machineApi.saveScript.request(form);
};
</script>
<style lang="scss"></style>
