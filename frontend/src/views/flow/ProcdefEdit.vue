<template>
    <div>
        <auto-form-drawer
            v-model:visible="visible"
            :title="title"
            :items="items"
            :data="editData"
            size="40%"
            :confirm-api="onConfirm"
            @submitted="emit('cancel')"
            @opened="onOpened"
            @cancel="emit('cancel')"
        >
            <!-- 消息模板选择 -->
            <template #msgTmplId="{ form }">
                <MsgTmplSelect v-model="form.msgTmplId" clearable />
            </template>

            <!-- 关联标签 -->
            <template #codePaths="{ form }">
                <tag-tree-check height="300px" v-model="form.codePaths" :tag-type="[TagResourceTypePath.Db, TagResourceTypeEnum.Redis.value]" />
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { TagResourceTypeEnum, TagResourceTypePath } from '@/common/commonEnum';
import { Rules } from '@/common/rule';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import { Msg } from '@/hooks/useI18n';
import { computed, ref, useTemplateRef, type PropType } from 'vue';
import { useI18n } from 'vue-i18n';
import MsgTmplSelect from '../msg/components/MsgTmplSelect.vue';
import TagTreeCheck from '../ops/component/TagTreeCheck.vue';
import { procdefApi } from './api';
import { ProcdefStatus } from './enums';
import type { Procdef } from './types';

const { t } = useI18n();

const props = defineProps({
    data: {
        type: Object as PropType<Procdef | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const visible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

interface ProcdefForm extends Partial<Procdef> {
    msgTmplId?: number | null;
    codePaths?: string[];
}

/** 表单声明（AutoFormItem[]，渲染 + 校验唯一数据源；消息模板/关联标签走 custom 插槽） */
const items: AutoFormItem[] = [
    { prop: 'name', label: 'common.name', rules: [Rules.requiredInput('common.name')] },
    { prop: 'defKey', label: 'Key', disabled: (f) => !!f.id, rules: [Rules.requiredInput('key')] },
    { prop: 'status', label: 'common.status', type: 'enum', enums: ProcdefStatus },
    {
        prop: 'condition',
        label: 'flow.triggeringCondition',
        type: 'textarea',
        rows: 10,
        tooltip: 'flow.triggeringConditionTips',
        placeholder: 'flow.conditionPlaceholder',
    },
    { prop: 'remark', label: 'common.remark' },
    { prop: 'msgTmplId', label: 'flow.notify', type: 'custom' },
    { prop: 'codePaths', label: 'tag.relateTag', type: 'custom' },
];

const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('drawerRef');

/** 传给 AutoFormDrawer 的回填数据（编辑态完整详情由 onOpened 异步补充回填） */
const editData = computed<AutoFormData>(() => {
    if (props.data) {
        const tags = (props.data as { tags?: Array<{ codePath: string }> }).tags;
        return {
            ...props.data,
            msgTmplId: null,
            codePaths: tags?.map((tag) => tag.codePath) || [],
        } as AutoFormData;
    }
    return {
        status: ProcdefStatus.Enable.value,
        condition: t('flow.conditionDefault'),
        msgTmplId: null,
        codePaths: [],
    } as AutoFormData;
});

/** 抽屉打开后暂存的内部表单引用（提交组装基于它） */
const internalForm = ref<AutoFormData>({});

const onOpened = async (form: AutoFormData) => {
    internalForm.value = form;
    // 编辑态异步补充回填详情（触发条件等列表行数据未携带的字段）
    if (props.data?.id) {
        const detail = await procdefApi.detail.request({ id: props.data.id });
        const tags = (props.data as { tags?: Array<{ codePath: string }> }).tags;
        Object.assign(form, detail, { codePaths: tags?.map((tag) => tag.codePath) });
    }
};

const submitForm = computed(() => internalForm.value as ProcdefForm);

const { execute: saveFlowDefExec } = procdefApi.save.useApi(submitForm);

// confirmApi 提交动作（无参，从内部表单读取提交数据）；成功提示与关闭抽屉由组件内置逻辑处理
const onConfirm = async () => {
    await saveFlowDefExec();
    emit('val-change', submitForm.value);
};
</script>
<style lang="scss"></style>
