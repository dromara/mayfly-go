<template>
    <el-dialog :title="props.title" v-model="visible" :show-close="false" :before-close="onCancel" :width="props.width" :destroy-on-close="true">
        <AutoForm ref="autoFormRef" v-model="state.form" :items="props.items" :cols="props.cols" v-bind="$attrs">
            <!-- 透传父组件插槽给 AutoForm（自定义字段渲染） -->
            <template v-for="(_, key) in useSlots()" #[key]="scope">
                <slot :name="key" v-bind="scope"></slot>
            </template>
        </AutoForm>

        <template #footer>
            <div class="dialog-footer">
                <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="props.confirmLoading" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
            </div>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
import { reactive, useSlots, useTemplateRef, watchEffect } from 'vue';
import { useI18nFormValidate } from '@/hooks/useI18n';
import AutoForm from './AutoForm.vue';
import { buildDefaultForm, type AutoFormData, type AutoFormItem } from './types';

const props = withDefaults(
    defineProps<{
        /** 弹窗标题 */
        title?: string;
        /** 字段配置 */
        items: AutoFormItem[];
        /** 编辑数据（编辑时传入对象回填表单，新增时传 false/null 使用字段 defaultValue） */
        data?: AutoFormData | boolean | null;
        /** 弹窗宽度 */
        width?: string;
        /** 栅格列数 */
        cols?: number;
        /** 确认按钮 loading（由父组件的保存请求状态驱动） */
        confirmLoading?: boolean;
    }>(),
    {
        width: '600px',
        confirmLoading: false,
    }
);

const visible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits<{
    /** 校验通过后点击确认触发，参数为表单数据（保存及关闭弹窗由父组件负责） */
    confirm: [form: AutoFormData];
    cancel: [];
}>();

const autoFormRef = useTemplateRef('autoFormRef');

const state = reactive({
    form: {} as AutoFormData,
});

// 弹窗打开时回填编辑数据或应用字段默认值
watchEffect(() => {
    if (!visible.value) {
        return;
    }
    if (props.data && typeof props.data === 'object') {
        state.form = { ...props.data };
    } else {
        state.form = buildDefaultForm(props.items);
    }
});

const onCancel = () => {
    visible.value = false;
    emit('cancel');
};

const onConfirm = async () => {
    await useI18nFormValidate(autoFormRef);
    emit('confirm', state.form);
};
</script>
<style lang="scss" scoped></style>
