<template>
    <el-dialog :title="props.title" v-model="visible" :show-close="false" :before-close="onCancel" :width="props.width" :destroy-on-close="true" :close-on-click-modal="props.closeOnClickModal">
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
                <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="props.confirmLoading" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
            </div>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
import { reactive, useSlots, useTemplateRef, watch } from 'vue';
import { useI18nFormValidate } from '@/hooks/useI18n';
import AutoForm from './AutoForm.vue';
import { buildDefaultForm, type AutoFormData, type AutoFormInstance } from './types';
import type { AutoFormJsonSchema, JsonField } from './json';
import { cloneFormData, resolveFormItems } from './shared';
import type { AutoFormItem, AutoFormTab } from './types';

const props = withDefaults(
    defineProps<{
        /** 弹窗标题 */
        title?: string;
        /** 字段配置（与 schema 二选一） */
        items?: AutoFormItem[];
        /** v1 JSON Schema 表单定义（编译为 items，与 items 二选一，优先 schema） */
        schema?: AutoFormJsonSchema | JsonField[];
        /** Tab 页签布局（每个 Tab 为一组字段，共享表单数据与校验） */
        tabs?: AutoFormTab[];
        /** 编辑数据（对象回填表单；false/null 表示新增，按字段 defaultValue 回填。boolean 仅为兼容历史调用链，新代码建议传 null） */
        data?: AutoFormData | boolean | null;
        /** 弹窗宽度 */
        width?: string;
        /** 栅格列数 */
        cols?: number;
        /** 确认按钮 loading（由父组件的保存请求状态驱动） */
        confirmLoading?: boolean;
        /** 点击遮罩是否关闭（默认 false：防止误点遮罩丢失已填表单数据） */
        closeOnClickModal?: boolean;
        /** label 位置（默认 right 右侧水平对齐，可传 top 展示在输入项上方） */
        labelPosition?: 'left' | 'right' | 'top';
    }>(),
    {
        width: '600px',
        confirmLoading: false,
        closeOnClickModal: false,
        labelPosition: 'right',
    }
);

const visible = defineModel<boolean>('visible', { default: false });

/** 当前激活 Tab（tabs 模式下供父组件实现分步向导：设置后弹窗内 AutoForm 切换到对应页签） */
const activeTab = defineModel<string>('activeTab', { default: '' });

const emit = defineEmits<{
    /** 校验通过后点击确认触发，参数为表单数据（保存及关闭弹窗由父组件负责） */
    confirm: [form: AutoFormData];
    cancel: [];
    /** 弹窗打开且回填完成后触发，参数为内部表单引用（供父组件异步补充回填字段，与 AutoFormDrawer 对齐） */
    opened: [form: AutoFormData];
}>();

const autoFormRef = useTemplateRef('autoFormRef');

const slots = useSlots();

/** 生效的字段配置：schema 优先编译，否则使用 items（tabs 模式下合并所有 Tab 字段，供 buildDefaultForm 使用，与 AutoForm 共用解析逻辑） */
const items = resolveFormItems(props);

const state = reactive({
    form: {} as AutoFormData,
});

// 弹窗打开时回填编辑数据或应用字段默认值（深拷贝，避免嵌套对象编辑中突变污染外部行数据）。
// 仅在打开瞬间回填：打开期间外部 data 引用变化（如列表刷新）不重置表单，避免丢失用户已填内容
watch(visible, (v) => {
    if (!v) {
        return;
    }
    if (props.data && typeof props.data === 'object') {
        state.form = cloneFormData(props.data);
    } else {
        state.form = buildDefaultForm(items.value);
    }
    // 回填完成后抛出内部表单引用（与 AutoFormDrawer 契约一致）
    emit('opened', state.form);
});

const onCancel = () => {
    visible.value = false;
    emit('cancel');
};

const onConfirm = async () => {
    // 校验失败内部已 toast 并抛出；返回 false 表示表单 ref 未就绪（如销毁中），静默忽略本次确认
    const valid = await useI18nFormValidate(autoFormRef);
    if (valid === false) {
        return;
    }
    emit('confirm', state.form);
};

/** 暴露内部表单方法（AutoFormInstance 契约编译期校验，外部编程式校验/重置/清校验） */
const exposed: AutoFormInstance = {
    validate: async () => autoFormRef.value?.validate(),
    resetFields: () => autoFormRef.value?.resetFields(),
    clearValidate: () => autoFormRef.value?.clearValidate(),
};
defineExpose(exposed);
</script>
<style lang="scss" scoped></style>
