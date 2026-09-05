<template>
    <el-col v-if="visible" :span="effectiveSpan">
        <!-- 分隔标题（非字段） -->
        <el-divider v-if="item.type == 'divider'" content-position="left">
            {{ item.label ? $t(item.label, item.labelParams ?? {}) : '' }}
        </el-divider>

        <el-form-item v-else :prop="item.prop" :label="item.label ? $t(item.label, item.labelParams ?? {}) : ''" :required="isRequired || undefined" :class="{ 'is-narrow-col': effectiveSpan < 24 }">
            <!-- 标签旁 tooltip 提示 -->
            <template v-if="item.tooltip" #label>
                <div class="flex items-center">
                    {{ item.label ? $t(item.label, item.labelParams ?? {}) : '' }}
                    <el-tooltip placement="top">
                        <template #content>
                            <span style="white-space: pre-line">{{ tooltipContent }}</span>
                        </template>
                        <SvgIcon name="QuestionFilled" class="ml-1" />
                    </el-tooltip>
                </div>
            </template>

            <!-- 自定义插槽（type='custom' 或父组件提供了同名插槽） -->
            <slot v-if="slotName" :name="slotName" :form="form" :item="item" />
            <AutoFormControl v-else-if="item.prop" v-model="form[item.prop]" :item="item" :form="form" :readonly="readonly" />
            <!-- 辅助说明（控件下方） -->
            <div v-if="item.description" class="w-full text-xs text-gray-400 leading-5">{{ $t(item.description) }}</div>
        </el-form-item>
    </el-col>
</template>

<script lang="ts" setup>
import { computed, useSlots } from 'vue';
import { useI18n } from 'vue-i18n';
import AutoFormControl from './AutoFormControl.vue';
import { isItemRequired } from './shared';
import type { AutoFormData, AutoFormItem } from './types';

const props = defineProps<{
    item: AutoFormItem;
    /** 表单数据对象（共享引用，直接读写字段值） */
    form: AutoFormData;
    /** 默认栅格跨度（由 AutoForm cols 均分 24） */
    defaultSpan: number;
    /** 只读（由 AutoForm 解析全局 readonly + 字段级 readonly 后传入） */
    readonly: boolean;
}>();

const slots = useSlots();

const { t } = useI18n();

/** tooltip 内容（支持多 key 多行） */
const tooltipContent = computed(() => {
    const tips = props.item.tooltip;
    if (!tips) {
        return '';
    }
    return (Array.isArray(tips) ? tips : [tips]).map((key) => t(key)).join('\n');
});

/** 动态 required（星号显示；校验规则由 AutoForm formRules 生成，判定逻辑与 AutoForm 共用） */
const isRequired = computed(() => isItemRequired(props.item, props.form));

/** 条件显隐（hidden 字段不渲染但保留在表单数据中） */
const visible = computed(() => !props.item.hidden && (!props.item.when || props.item.when(props.form)));

/** 实际栅格跨度：窄列（span < 24，多字段并排）标记 is-narrow-col，
 *  脱离 label-width=auto 的全局 label 右对齐（其偏移按全表单最宽 label 计算，
 *  窄列可用宽小于「偏移 + label」时 content 被挤为 0，控件不可见且触发横向滚动） */
const effectiveSpan = computed(() => props.item.span ?? props.defaultSpan);

/** custom 类型或父组件提供了 prop 同名插槽时，返回插槽名 */
const slotName = computed<string | null>(() => {
    const name = props.item.slot ?? props.item.prop;
    if (!name) {
        return null;
    }
    return props.item.type == 'custom' || slots[name] ? name : null;
});
</script>
