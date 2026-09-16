<template>
    <div>
        <auto-form-drawer
            ref="drawerRef"
            v-model:visible="dialogVisible"
            :title="title"
            :items="items"
            :data="editData"
            size="50%"
            :confirm-api="onConfirm"
            @cancel="emit('cancel')"
        >
            <!-- 告警条件自定义 -->
            <template #condition="{ form }">
                <AlertConditionEdit v-model="form.condition" :resource-type="form.resourceType" />
            </template>

            <!-- 范围值：标签路径多选 -->
            <template #scopeValue="{ form }">
                <TagTreeCheck :key="form.resourceType" height-mode="fixed" height="200px" :tag-type="`${form.resourceType}`" v-model="tagPathValue" @update:model-value="(val) => { formRef = form; tagPathValue = val; }" />
            </template>

            <!-- 规则标签 -->
            <template #labels="{ form }">
                <LabelAssociation v-model="form.labels" />
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { computed, ref, useTemplateRef, type PropType, watch } from 'vue';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import { alertRuleApi } from '../api';
import type { AlertCondition, AlertRuleForm, AlertRuleVO } from '../types';
import { AlertPriorityEnum } from '../enums';
import { parseLabelJson, supportedResourceTypes } from '../utils';
import AlertConditionEdit from '../components/AlertConditionEdit.vue';
import LabelAssociation from '@/views/ops/label/components/LabelAssociation.vue';
import TagTreeCheck from '../../component/TagTreeCheck.vue';
import { labelApi } from '@/views/ops/label/api';

const props = defineProps({
    data: {
        type: Object as PropType<AlertRuleVO | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });
const emit = defineEmits<{
    cancel: [];
    'val-change': [form: AlertRuleForm];
}>();

const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('drawerRef');

// 标签路径值（数组形式，用于 TagTreeCheck 多选）
const tagPathValue = ref<string[]>([]);

// 标签绑定数据（从 t_label_binding 查询）
const labelBindingsJson = ref('');

// 捕获 slot 中的 form 引用，用于将 tagPathValue 同步到 form.scopeValue
const formRef = ref<Record<string, unknown> | null>(null);

// 抽屉打开时，如果是编辑模式，从 t_label_binding 查询标签
watch(dialogVisible, async (visible) => {
    if (visible && props.data?.id) {
        try {
            const bindings = await labelApi.bindings.request({ targetType: 'alert_rule', targetId: props.data.id });
            // 转换为 JSON 格式：{"key": "value"}
            const labelsObj: Record<string, string> = {};
            for (const b of bindings) {
                labelsObj[b.labelKey] = b.labelValue;
            }
            labelBindingsJson.value = Object.keys(labelsObj).length ? JSON.stringify(labelsObj) : '';
        } catch {
            labelBindingsJson.value = '';
        }
    } else {
        labelBindingsJson.value = '';
    }
});

// tagPathValue 变化时同步到 form.scopeValue，确保 AutoForm 的 required 校验能感知到值
watch(tagPathValue, (val) => {
    if (formRef.value) {
        formRef.value.scopeValue = JSON.stringify(val);
    }
});

/**
 * 表单配置。
 *
 * 数值区间与 server 侧 alert_rule.go 的校验常量保持一致
 * （minEvalInterval/maxEvalInterval、minLimitCount/maxLimitCount），避免前端填得出、后端存不下
 */
const items: AutoFormItem[] = [
    { prop: 'name', label: 'alert.ruleName', required: true },
    {
        prop: 'resourceType',
        label: 'alert.resourceType',
        type: 'select',
        required: true,
        // 可选项来自后端已注册评估器的资源类型：无评估器的类型规则不会被任何评估周期处理
        options: async () => supportedResourceTypes.value,
        // 资源类型决定可用指标与范围语义，变更后原有条件/范围不再成立，需清空避免残留脏值
        onChange: (_value, form) => {
            form.condition = { operator: 'and', items: [] };
            form.scopeValue = '';
            tagPathValue.value = [];
        },
    },
    {
        prop: 'scopeValue',
        label: 'alert.scopeValue',
        type: 'custom',
        required: true,
        tooltip: 'alert.scopeValueTips',
        validate: (value) => {
            // TagTreeCheck 多选：至少选一个标签路径
            return tagPathValue.value.length > 0 ? true : 'alert.scopeValueRequired';
        },
    },
    {
        prop: 'priority',
        label: 'alert.priority',
        type: 'enum',
        enums: AlertPriorityEnum,
        required: true,
    },
    {
        prop: 'condition',
        label: 'alert.condition',
        type: 'custom',
        required: true,
        tooltip: 'alert.conditionTips',
        validate: (value) => {
            const condition = value as AlertCondition | undefined;
            const conditionItems = condition?.items ?? [];
            if (!conditionItems.length) {
                return 'alert.conditionRequired';
            }
            // 指标或比较方式留空时后端会拒绝保存，这里提前给出定位到行的提示
            return conditionItems.every((item) => item.metric && item.compare) ? true : 'alert.conditionRequired';
        },
    },
    {
        prop: 'notifyConfig.groupWait',
        label: 'alert.groupWait',
        type: 'number',
        span: 8,
        min: 0,
        max: 3600,
        suffix: 's',
        tooltip: 'alert.groupWaitTips',
        props: { placeholder: 'alert.groupWaitPlaceholder' },
    },
    {
        prop: 'notifyConfig.groupInterval',
        label: 'alert.groupInterval',
        type: 'number',
        span: 8,
        min: 0,
        max: 86400,
        suffix: 's',
        tooltip: 'alert.groupIntervalTips',
        props: { placeholder: 'alert.groupIntervalPlaceholder' },
    },
    {
        prop: 'notifyConfig.repeatInterval',
        label: 'alert.repeatInterval',
        type: 'number',
        span: 8,
        min: 0,
        max: 86400,
        suffix: 's',
        tooltip: 'alert.repeatIntervalTips',
        props: { placeholder: 'alert.repeatIntervalPlaceholder' },
    },
    {
        prop: 'evalInterval',
        label: 'alert.evalInterval',
        type: 'number',
        span: 8,
        min: 10,
        max: 86400,
        suffix: 's',
        tooltip: 'alert.evalIntervalTips',
    },
    {
        prop: 'triggerCount',
        label: 'alert.triggerCount',
        type: 'number',
        span: 8,
        min: 1,
        max: 1000,
        tooltip: 'alert.triggerCountTips',
    },
    {
        prop: 'recoveryCount',
        label: 'alert.recoveryCount',
        type: 'number',
        span: 8,
        min: 1,
        max: 1000,
        tooltip: 'alert.recoveryCountTips',
        props: { placeholder: 'alert.recoveryCountPlaceholder' },
    },
    {
        prop: 'labels',
        label: 'alert.ruleLabels',
        type: 'custom',
        slot: 'labels',
        // 后端以 map[string]string 解析该字段，非法 JSON 会让静默规则永不命中且无任何报错
        validate: (value) => (parseLabelJson(String(value ?? '')) ? true : 'alert.labelsInvalidJson'),
    },
    { prop: 'remark', label: 'alert.remark', type: 'textarea', rows: 2 },
];

const defaultForm: AlertRuleForm = {
    name: '',
    status: 1,
    priority: 2,
    remark: '',
    resourceType: 1,
    scopeType: 2,
    scopeValue: '',
    condition: { operator: 'and', items: [] },
    notifyConfig: {
        groupWait: 30,
        groupInterval: 300,
        repeatInterval: 300,
    },
    evalInterval: 60,
    triggerCount: 1,
    recoveryCount: 1,
    labels: '',
};

/** 回填数据 */
const editData = computed<AutoFormData | null>(() => {
    const rule = props.data;
    if (!rule) {
        tagPathValue.value = [];
        labelBindingsJson.value = '';
        return { ...defaultForm } as unknown as AutoFormData;
    }
    // 始终为标签路径模式，解析 JSON 数组
    tagPathValue.value = rule.scopeValue ? (() => {
        try { return JSON.parse(rule.scopeValue); } catch { return []; }
    })() : [];
    return {
        ...rule,
        condition: rule.condition || { operator: 'and', items: [] },
        notifyConfig: { ...defaultForm.notifyConfig, ...(rule.notifyConfig || {}) },
        labels: labelBindingsJson.value || rule.labels || '',
    } as unknown as AutoFormData;
});

const { execute: saveRuleExec } = alertRuleApi.save.useApi();

/** 提交 */
const onConfirm = async (rawForm: AutoFormData) => {
    const form = rawForm as unknown as AlertRuleForm;
    // 始终为标签路径模式，序列化多选结果为 JSON 数组
    form.scopeValue = JSON.stringify(tagPathValue.value);
    form.labels = String(form.labels ?? '').trim();
    await saveRuleExec(form);
    emit('val-change', form);
};

// 保留 drawer ref 以便外部按需触发校验
void drawerRef;
</script>
