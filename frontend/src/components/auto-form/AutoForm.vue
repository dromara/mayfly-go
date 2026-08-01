<template>
    <el-form ref="formRef" :model="model" :rules="formRules" label-width="auto" v-bind="$attrs">
        <el-row :gutter="16">
            <template v-for="(item, index) in props.items" :key="item.prop ?? `item-${index}`">
                <el-col v-if="isVisible(item)" :span="item.span ?? defaultSpan">
                    <!-- 分隔标题 -->
                    <el-divider v-if="item.type == 'divider'" content-position="left">
                        {{ item.label ? $t(item.label) : '' }}
                    </el-divider>

                    <el-form-item v-else :prop="item.prop" :label="item.label ? $t(item.label) : ''">
                        <!-- 标签旁 tooltip 提示 -->
                        <template v-if="item.tooltip" #label>
                            <div class="flex items-center">
                                {{ item.label ? $t(item.label) : '' }}
                                <el-tooltip :content="$t(item.tooltip)" placement="top">
                                    <SvgIcon name="QuestionFilled" class="ml-1" />
                                </el-tooltip>
                            </div>
                        </template>

                        <!-- 自定义插槽（type='custom' 或父组件提供了同名插槽） -->
                        <slot v-if="slotNameOf(item)" :name="slotNameOf(item)!" :form="model" :item="item" />
                        <AutoFormControl v-else-if="item.prop" v-model="model[item.prop]" :item="item" :form="model" />
                    </el-form-item>
                </el-col>
            </template>
        </el-row>
    </el-form>
</template>

<script lang="ts" setup>
import { computed, useSlots, useTemplateRef } from 'vue';
import type { FormInstance, FormItemRule } from 'element-plus';
import { Rules } from '@/common/rule';
import AutoFormControl from './AutoFormControl.vue';
import { isSelectLikeItem, type AutoFormData, type AutoFormItem } from './types';

const props = defineProps<{
    /** 字段配置（唯一数据源：渲染 + 校验） */
    items: AutoFormItem[];
    /** 栅格列数（默认 1，字段可用 span 单独覆盖） */
    cols?: number;
}>();

/** 表单数据（v-model 双向绑定） */
const model = defineModel<AutoFormData>({ default: () => ({}) });

const slots = useSlots();

const formRef = useTemplateRef<FormInstance>('formRef');

/** 每个字段默认占据的栅格跨度 */
const defaultSpan = computed(() => Math.floor(24 / (props.cols ?? 1)));

/** 条件显隐 */
const isVisible = (item: AutoFormItem) => !item.when || item.when(model.value);

/** custom 类型或父组件提供了 prop 同名插槽时，返回插槽名 */
const slotNameOf = (item: AutoFormItem): string | null => {
    const name = item.slot ?? item.prop;
    if (!name) {
        return null;
    }
    if (item.type == 'custom' || slots[name]) {
        return name;
    }
    return null;
};

/** 根据字段配置生成 element-plus 校验规则（required + 自定义 rules 合并） */
const formRules = computed(() => {
    const rules: Record<string, FormItemRule[]> = {};
    for (const item of props.items) {
        if (!item.prop) {
            continue;
        }
        const itemRules: FormItemRule[] = [];
        if (item.required) {
            itemRules.push(isSelectLikeItem(item) ? Rules.requiredSelect(item.label) : Rules.requiredInput(item.label));
        }
        if (item.rules) {
            itemRules.push(...(Array.isArray(item.rules) ? item.rules : [item.rules]));
        }
        if (itemRules.length > 0) {
            rules[item.prop] = itemRules;
        }
    }
    return rules;
});

const validate = async () => {
    return await formRef.value?.validate();
};

const resetFields = () => {
    formRef.value?.resetFields();
};

const clearValidate = () => {
    formRef.value?.clearValidate();
};

defineExpose({
    validate,
    resetFields,
    clearValidate,
});
</script>
<style lang="scss" scoped></style>
