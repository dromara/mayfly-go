<template>
    <ADialog :title="props.title" v-model="modelVisible" :show-close="false" :before-close="onCancel" :width="props.width" :destroy-on-close="true" :close-on-click-modal="props.closeOnClickModal">
        <AutoForm ref="autoFormRef" v-model="state.form" :items="items" :tabs="props.tabs" :cols="props.cols" :label-position="labelPosition" v-model:active-tab="activeTab" v-bind="$attrs">
            <!-- 透传父组件插槽给 AutoForm（自定义字段渲染） -->
            <template v-for="(_, key) in slots" #[key]="scope">
                <slot :name="key" v-bind="scope"></slot>
            </template>
        </AutoForm>

        <!-- 表单下方的附加内容插槽（与 AutoFormDrawer 契约对齐），scope 暴露内部表单供直接绑定 -->
        <slot name="body-extra" :form="state.form"></slot>

        <template #footer>
            <div class="dialog-footer">
                <AButton @click="onCancel()">{{ $t('common.cancel') }}</AButton>
                <AButton type="primary" :loading="confirming || props.confirmLoading" @click="onConfirm">{{ $t('common.confirm') }}</AButton>
            </div>
        </template>
    </ADialog>
</template>

<script lang="ts" setup>
import { useSlots, useTemplateRef } from 'vue';
import { ADialog, AButton } from './ui/adapter';
import AutoForm from './AutoForm.vue';
import { useAutoFormHost, type AutoFormHostProps } from '@/hooks/useAutoFormHost';
import type { AutoFormData } from './types';

const props = withDefaults(
    defineProps<
        AutoFormHostProps & {
            /** 弹窗标题 */
            title?: string;
            /** 弹窗宽度 */
            width?: string;
        }
    >(),
    {
        width: '600px',
        confirmLoading: false,
        closeOnClickModal: false,
        labelPosition: 'right',
    }
);

/** 可见性模型（由父组件 v-model 传入） */
const modelVisible = defineModel<boolean>('visible', { default: false });

/** 当前激活 Tab（tabs 模式下供父组件实现分步向导：设置后弹窗内 AutoForm 切换到对应页签） */
const activeTab = defineModel<string>('activeTab', { default: '' });

const emit = defineEmits<{
    /** 校验通过后点击确认触发，参数为表单数据（保存及关闭弹窗由父组件负责） */
    confirm: [form: AutoFormData];
    cancel: [];
    /** 弹窗打开且回填完成后触发，参数为内部表单引用（供父组件异步补充回填字段，与 AutoFormDrawer 对齐） */
    opened: [form: AutoFormData];
    /** confirmApi 默认提交成功后触发（参数为表单数据，父组件通常在此刷新列表） */
    submitted: [form: AutoFormData];
}>();

const autoFormRef = useTemplateRef('autoFormRef');

const slots = useSlots();

// 回填/统一提交/防重/expose 逻辑收敛在 useAutoFormHost，与 AutoFormDrawer 单一出处（宿主仅保留模板与形态差异）
const { items, state, confirming, onCancel, onConfirm, exposed } = useAutoFormHost({
    props,
    visible: modelVisible,
    autoFormRef,
    onCancel: () => emit('cancel'),
    onOpened: (form) => emit('opened', form),
    onSubmitted: (form) => emit('submitted', form),
});
defineExpose(exposed);
</script>
<style lang="scss" scoped></style>
