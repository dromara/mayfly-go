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
            @opened="loadDetailOnOpened"
            @cancel="emit('cancel')"
        >
            <!-- 消息模板选择 -->
            <template #msgTmplId="{ form }">
                <MsgTmplSelect v-model="form.msgTmplId" clearable />
            </template>

            <!-- 关联标签 -->
            <template #codePaths="{ form }">
                <tag-tree-check height-mode="fixed" height="300px" v-model="form.codePaths" :tag-type="[TagResourceTypePath.Db, TagResourceTypeEnum.Redis.value]" />
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { TagResourceTypeEnum, TagResourceTypePath } from '@/common/commonEnum';
import { Rules } from '@/common/rule';
import { AutoFormDrawer, defineFormItems } from '@/components/auto-form';
import { useAutoFormModel } from '@/hooks/useAutoFormModel';
import { computed, type PropType } from 'vue';
import { useI18n } from 'vue-i18n';
import MsgTmplSelect from '../msg/components/MsgTmplSelect.vue';
import TagTreeCheck from '../ops/component/TagTreeCheck.vue';
import { procdefApi } from './api';
import { ProcdefStatus } from './enums';
import type { ProcdefVO } from './types';

const { t } = useI18n();

const props = defineProps({
    data: {
        type: Object as PropType<ProcdefVO | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const visible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

interface ProcdefForm extends Partial<ProcdefVO> {
    msgTmplId?: number | null;
    codePaths?: string[];
}

/** 表单声明（defineFormItems<ProcdefForm>，渲染 + 校验唯一数据源；消息模板/关联标签走 custom 插槽） */
const items = defineFormItems<ProcdefForm>([
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
]);

/** 关联标签路径（列表行与详情均经 FillTagInfo 携带 tags） */
const codePathsOf = (procdef?: ProcdefVO | null) => procdef?.tags?.map((tag) => tag.codePath) ?? [];

/** 传给 AutoFormDrawer 的回填数据（编辑态完整详情由 loadDetailOnOpened 异步补充回填） */
const editData = computed<ProcdefForm>(() => {
    if (props.data) {
        return {
            ...props.data,
            msgTmplId: null,
            codePaths: codePathsOf(props.data),
        };
    }
    return {
        status: ProcdefStatus.Enable.value,
        condition: t('flow.conditionDefault'),
        msgTmplId: null,
        codePaths: [],
    };
});

// 宿主抽屉的内部表单在 @opened 接管（提交载荷即它）
const { requireForm, openedWith } = useAutoFormModel<ProcdefForm>();

/** 编辑态异步补充回填详情（触发条件/消息模板等列表行未携带的字段） */
const loadDetailOnOpened = openedWith(async (form) => {
    if (!props.data?.id) {
        return;
    }
    const detail = await procdefApi.detail.request({ id: props.data.id });
    Object.assign(form, detail, { codePaths: codePathsOf(detail) });
});

const submitForm = computed(requireForm);

const { execute: saveFlowDefExec } = procdefApi.save.useApi(submitForm);

// confirmApi 提交动作（无参，从内部表单读取提交数据）；成功提示与关闭抽屉由组件内置逻辑处理
const onConfirm = async () => {
    await saveFlowDefExec();
    emit('val-change', submitForm.value);
};
</script>
<style lang="scss"></style>
