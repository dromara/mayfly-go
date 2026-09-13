<template>
    <ADrawer class="auto-form-drawer" v-model="modelVisible" :size="props.size" :direction="props.direction" :destroy-on-close="true" :before-close="onCancel" :append-to-body="props.appendToBody" :close-on-click-modal="props.closeOnClickModal">
        <template #header>
            <!-- 统一头部：返回箭头 + 标题（对齐 DrawerHeader 全站样式）；header-extra 插槽渲染头部右侧操作区 -->
            <DrawerHeader :header="props.title" :back="onCancel">
                <template v-if="slots['header-extra']" #extra>
                    <slot name="header-extra"></slot>
                </template>
            </DrawerHeader>
        </template>
        <AutoForm ref="autoFormRef" class="auto-form-drawer__form" v-model="state.form" :items="items" :tabs="props.tabs" :cols="props.cols" :label-position="labelPosition" v-model:active-tab="activeTab" v-bind="$attrs">
            <!-- 透传父组件插槽给 AutoForm（自定义字段渲染） -->
            <template v-for="(_, key) in slots" #[key]="scope">
                <slot :name="key" v-bind="scope"></slot>
            </template>
        </AutoForm>

        <!-- 表单下方的附加内容插槽（如同页业务信息区块），scope 暴露内部表单供直接绑定 -->
        <slot name="body-extra" :form="state.form"></slot>

        <template v-if="slots.footer" #footer>
            <slot name="footer" :form="state.form"></slot>
        </template>
        <template v-else #footer>
            <div class="drawer-footer">
                <AButton @click="onCancel()">{{ $t('common.cancel') }}</AButton>
                <AButton type="primary" :loading="confirming || props.confirmLoading" @click="onConfirm">{{ $t('common.confirm') }}</AButton>
            </div>
        </template>
    </ADrawer>
</template>

<script lang="ts" setup>
import { useSlots, useTemplateRef } from 'vue';
import { ADrawer, AButton } from './ui/adapter';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import AutoForm from './AutoForm.vue';
import { useAutoFormHost, type AutoFormHostProps } from '@/hooks/useAutoFormHost';
import type { AutoFormData } from './types';

const props = withDefaults(
    defineProps<
        AutoFormHostProps & {
            /** 抽屉标题 */
            title?: string;
            /** 抽屉宽度/尺寸（size 透传 el-drawer） */
            size?: string | number;
            /** 抽屉方向（rtl 右侧滑出 / ltr 左侧滑出 / ttb 顶部 / btt 底部） */
            direction?: 'rtl' | 'ltr' | 'ttb' | 'btt';
            /** 是否挂载到 body（嵌套弹层场景需开启以避免层叠上下文问题） */
            appendToBody?: boolean;
        }
    >(),
    {
        size: '500px',
        direction: 'rtl',
        confirmLoading: false,
        appendToBody: false,
        closeOnClickModal: false,
        labelPosition: 'top',
    }
);

/** 可见性模型（由父组件 v-model 传入） */
const modelVisible = defineModel<boolean>('visible', { default: false });

/** 当前激活 Tab（tabs 模式下供父组件实现分步向导：设置后抽屉内 AutoForm 切换到对应页签） */
const activeTab = defineModel<string>('activeTab', { default: '' });

const emit = defineEmits<{
    /** 校验通过后点击确认触发，参数为表单数据（保存及关闭抽屉由父组件负责） */
    confirm: [form: AutoFormData];
    cancel: [];
    /** 抽屉打开且回填完成后触发，参数为内部表单引用（供父组件异步补充回填字段） */
    opened: [form: AutoFormData];
    /** confirmApi 默认提交成功后触发（参数为表单数据，父组件通常在此刷新列表） */
    submitted: [form: AutoFormData];
}>();

const autoFormRef = useTemplateRef('autoFormRef');

const slots = useSlots();

// 回填/统一提交/防重/expose 逻辑收敛在 useAutoFormHost，与 AutoFormDialog 单一出处（宿主仅保留模板与形态差异）
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
<style lang="scss">
// AutoFormDrawer 专属抽屉样式（全局块：el-drawer teleport 到 body，scoped :deep() 无法穿透）
// 通过 .auto-form-drawer 类名限定作用域，避免污染其他抽屉
.auto-form-drawer {
    // 头部间距收紧（el-drawer__header 默认 margin-bottom: 32px，el-page-header 自带大 padding）
    .el-drawer__header {
        margin-bottom: 0 !important;
        padding: 0 20px !important;
    }

    .el-drawer__header .el-page-header {
        padding: 10px 0;
    }

    // 主体内边距收紧（默认 30px → 10px 20px，减少顶部空白）
    .el-drawer__body {
        padding: 10px 20px !important;
    }

    // 底部按钮区内边距同步
    .el-drawer__footer {
        padding: 10px 20px !important;
    }
}
</style>
<style lang="scss" scoped>
// 表单区域：占满抽屉宽度，由抽屉 body 统一滚动
.auto-form-drawer__form {
    width: 100%;
}

// 底部按钮栏
.drawer-footer {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
}
</style>
