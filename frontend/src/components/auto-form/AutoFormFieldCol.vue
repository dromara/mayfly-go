<template>
    <ACol v-if="visible" :span="toGridSpan(effectiveSpan)">
        <!-- 分隔标题（非字段） -->
        <ADivider v-if="item.type == 'divider'" content-position="left">
            {{ item.label ? $t(item.label, item.labelParams ?? {}) : '' }}
        </ADivider>

        <AFormItem v-else :prop="item.prop" :label="item.label ? $t(item.label, item.labelParams ?? {}) : ''" :required="isRequired || undefined" :class="{ 'is-narrow-col': effectiveSpan < 24, 'has-tooltip-label': !!item.tooltip }">
            <!-- 标签旁 tooltip 提示 -->
            <template v-if="item.tooltip" #label>
                <div class="flex items-center">
                    {{ item.label ? $t(item.label, item.labelParams ?? {}) : '' }}
                    <ATooltip placement="top" popper-class="auto-form-label-tooltip">
                        <template #content>
                            <span style="white-space: pre-line">{{ tooltipContent }}</span>
                        </template>
                        <SvgIcon name="QuestionFilled" class="ml-1" />
                    </ATooltip>
                </div>
            </template>

            <!-- 自定义插槽（type='custom' 或父组件提供了同名插槽）；
                 外层 w-full 保证插槽内容作为 el-form-item__content(flex) 的子项占满整行，
                 避免 flex item 收缩到内容宽度导致表格/选择器等无法撑满 -->
            <div v-if="slotName" class="w-full">
                <slot :name="slotName" :form="form" :item="item" />
            </div>
            <AutoFormControl v-else-if="item.prop" v-model="form[item.prop]" :item="item" :form="form" :readonly="readonly" />
            <!-- 辅助说明（控件下方） -->
            <div v-if="item.description" class="w-full text-xs text-gray-400 leading-5">{{ $t(item.description) }}</div>
        </AFormItem>
    </ACol>
</template>

<script lang="ts" setup>
import { computed, useSlots } from 'vue';
import { useI18n } from 'vue-i18n';
import { ACol, ADivider, AFormItem, ATooltip, toGridSpan } from './ui/adapter';
import AutoFormControl from './AutoFormControl.vue';
import { isItemRequired, isItemVisible } from './shared';
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

const { t, tm } = useI18n();

/** tooltip 内容：
 *  - item.tooltip 为单个 i18n key 或 key 数组；
 *  - key 对应的消息若为数组（逐行文案），逐行展开；否则按普通 key 翻译；
 *  - 最终以换行拼接，配合气泡的 white-space: pre-line 渲染成多行 */
const tooltipContent = computed(() => {
    const tips = props.item.tooltip;
    if (!tips) {
        return '';
    }
    const keys = Array.isArray(tips) ? tips : [tips];
    return keys
        .flatMap((key) => {
            const message = tm(key);
            return Array.isArray(message) ? (message as string[]) : [t(key)];
        })
        .join('\n');
});

/** 动态 required（星号显示；校验规则由 AutoForm formRules 生成，判定逻辑与 AutoForm 共用） */
const isRequired = computed(() => isItemRequired(props.item, props.form));

/** 条件显隐（hidden 字段不渲染但保留在表单数据中；判定逻辑与 AutoFormFields 分组空壳过滤共用） */
const visible = computed(() => isItemVisible(props.item, props.form));

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
