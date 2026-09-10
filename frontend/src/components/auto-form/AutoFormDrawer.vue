<template>
    <ADrawer v-model="visible" :size="props.size" :direction="props.direction" :destroy-on-close="true" :before-close="onCancel" :append-to-body="props.appendToBody" :close-on-click-modal="props.closeOnClickModal">
        <template #header>
            <!-- 统一头部：返回箭头 + 标题（对齐 DrawerHeader 全站样式）；header-extra 插槽渲染头部右侧操作区 -->
            <DrawerHeader :header="props.title" :back="onCancel">
                <template v-if="slots['header-extra']" #extra>
                    <slot name="header-extra"></slot>
                </template>
            </DrawerHeader>
        </template>
        <AutoForm ref="autoFormRef" v-model="state.form" :items="items" :tabs="props.tabs" :cols="props.cols" :label-position="labelPosition" v-model:active-tab="activeTab" v-bind="$attrs">
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
import { useAutoFormHost } from '@/hooks/useAutoFormHost';
import type { AutoFormJsonSchema, JsonField } from './json';
import type { AutoFormData, AutoFormItem, AutoFormTab } from './types';

const props = withDefaults(
    defineProps<{
        /** 抽屉标题 */
        title?: string;
        /** 字段配置（与 schema 二选一） */
        items?: AutoFormItem[];
        /** v1 JSON Schema 表单定义（编译为 items，与 items 二选一，优先 schema） */
        schema?: AutoFormJsonSchema | JsonField[];
        /** Tab 页签布局（每个 Tab 为一组字段，共享表单数据与校验） */
        tabs?: AutoFormTab[];
        /** 编辑数据（对象回填表单；false/null 表示新增，按字段 defaultValue 回填。boolean 仅为兼容历史调用链，新代码建议传 null） */
        data?: AutoFormData | boolean | null;
        /** 抽屉宽度/尺寸（size 透传 el-drawer） */
        size?: string | number;
        /** 抽屉方向（rtl 右侧滑出 / ltr 左侧滑出 / ttb 顶部 / btt 底部） */
        direction?: 'rtl' | 'ltr' | 'ttb' | 'btt';
        /** 栅格列数 */
        cols?: number;
        /** 确认按钮 loading（由父组件的保存请求状态驱动） */
        confirmLoading?: boolean;
        /** 是否挂载到 body（嵌套弹层场景需开启以避免层叠上下文问题） */
        appendToBody?: boolean;
        /** 点击遮罩是否关闭（默认 false：防止误点遮罩丢失已填表单数据） */
        closeOnClickModal?: boolean;
        /** label 位置（默认 top 展示在输入项上方，label 长短不齐时观感更统一；可传 right 恢复水平对齐） */
        labelPosition?: 'left' | 'right' | 'top';
        /** 统一提交 API（可选）：传入后点击确认走内置默认提交逻辑（校验 → 调 API → 成功提示 → 触发 submitted → 关闭抽屉；
         *  失败时保留抽屉供修改重提），confirm 事件不再触发；不传则走 confirm 事件由父组件自行处理 */
        confirmApi?: (form: AutoFormData) => Promise<unknown>;
    }>(),
    {
        size: '500px',
        direction: 'rtl',
        cols: undefined,
        confirmLoading: false,
        appendToBody: false,
        closeOnClickModal: false,
        labelPosition: 'top',
    }
);

const visible = defineModel<boolean>('visible', { default: false });

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
    visible,
    autoFormRef,
    onCancel: () => emit('cancel'),
    onOpened: (form) => emit('opened', form),
    onSubmitted: (form) => emit('submitted', form),
});
defineExpose(exposed);
</script>
<style lang="scss" scoped></style>
