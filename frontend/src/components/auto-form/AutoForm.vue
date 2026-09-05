<template>
    <el-form ref="formRef" :model="model" :rules="formRules" :label-position="labelPosition ?? 'right'" :label-width="labelWidth ?? 'auto'" v-bind="$attrs">
        <!-- Tab 分组布局（非懒渲染：未激活 Tab 的字段同样挂载，validate 全量生效） -->
        <el-tabs v-if="activeTabs.length" v-model="activeTab">
            <el-tab-pane v-for="tab in activeTabs" :key="tab.name" :name="tab.name" :disabled="isTabDisabled(tab)">
                <template #label>
                    <span class="flex items-center">
                        <SvgIcon v-if="tab.icon" :name="tab.icon" class="mr-1" />
                        {{ $t(tab.label) }}
                    </span>
                </template>
                <AutoFormFields :items="tab.items" :form="model" :cols="props.cols" :readonly="props.readonly ?? false">
                    <template v-for="(_, name) in $slots" :key="name" #[name]="slotProps">
                        <slot :name="name" v-bind="slotProps ?? {}" />
                    </template>
                </AutoFormFields>
            </el-tab-pane>
        </el-tabs>

        <!-- 平铺布局（支持 group 分组容器） -->
        <AutoFormFields v-else :items="allItems" :form="model" :cols="props.cols" :readonly="props.readonly ?? false">
            <template v-for="(_, name) in $slots" :key="name" #[name]="slotProps">
                <slot :name="name" v-bind="slotProps ?? {}" />
            </template>
        </AutoFormFields>
    </el-form>
</template>

<script lang="ts" setup>
import { computed, ref, watchEffect, useTemplateRef } from 'vue';
import type { FormInstance, FormItemRule } from 'element-plus';
import { useI18n } from 'vue-i18n';
import { Rules } from '@/common/rule';
import SvgIcon from '@/components/svg-icon/index.vue';
import AutoFormFields from './AutoFormFields.vue';
import { compileJsonTabs, type AutoFormJsonSchema, type JsonField } from './json';
import { isItemRequired, resolveFormItems } from './shared';
import { isSelectLikeItem, type AutoFormData, type AutoFormItem, type AutoFormInstance, type AutoFormTab } from './types';

const props = defineProps<{
        /** 字段配置（渲染 + 校验数据源），与 schema 二选一 */
        items?: AutoFormItem[];
        /** v1 JSON Schema 表单定义（经编译层转为 items），与 items 二选一，优先 schema */
        schema?: AutoFormJsonSchema | JsonField[];
        /** Tab 页签布局（每个 Tab 为一组字段，共享表单数据与校验），与 schema 内置 tabs 等效 */
        tabs?: AutoFormTab[];
        /** 栅格列数（默认 1，字段可用 span 单独覆盖） */
        cols?: number;
        /** 全局只读模式（所有字段禁用，字段级 readonly 同样生效） */
        readonly?: boolean;
        /** label 位置（right 右侧水平对齐 / top 输入项上方；缺省 right，抽屉场景由 AutoFormDrawer 默认传 top） */
        labelPosition?: 'left' | 'right' | 'top';
        /** label 宽度（默认 auto：mirror 测量全表单最宽 label 后统一右对齐；嵌套弹窗等场景测量漂移导致个别 label 溢出压线时，可传固定宽度如 '80px'） */
        labelWidth?: string;
}>();

const { t } = useI18n();

/** 表单数据（v-model 双向绑定） */
const model = defineModel<AutoFormData>({ default: () => ({}) });

const formRef = useTemplateRef<FormInstance>('formRef');

/** 生效的 Tab 布局：props.tabs 优先，其次 schema 内置 tabs（编译为 AutoFormTab[]） */
const activeTabs = computed<AutoFormTab[]>(() => {
    if (props.tabs?.length) {
        return props.tabs;
    }
    const schema = props.schema;
    if (schema && !Array.isArray(schema) && schema.tabs?.length) {
        return compileJsonTabs(schema);
    }
    return [];
});

/** 当前激活 Tab（支持 v-model:active-tab 外部控制，如向导式上一步/下一步；缺省自动激活第一个 Tab） */
const activeTab = defineModel<string>('activeTab', { default: '' });

// 未指定激活 Tab 时自动激活第一个，保证外部向导逻辑读到的始终是有效 Tab 名
watchEffect(() => {
    if (!activeTab.value && activeTabs.value.length) {
        activeTab.value = activeTabs.value[0].name;
    }
});

/** Tab 禁用（静态布尔或根据表单值动态计算） */
const isTabDisabled = (tab: AutoFormTab): boolean =>
    typeof tab.disabled === 'function' ? tab.disabled(model.value) : !!tab.disabled;

/** 生效的字段配置：schema 优先编译，否则使用 items（与 Dialog/Drawer 共用同一解析逻辑） */
const allItems = resolveFormItems(props);

/** 根据字段配置生成 element-plus 校验规则（required + validate 函数 + 自定义 rules 合并） */
const formRules = computed(() => {
    const rules: Record<string, FormItemRule[]> = {};
    // Tab 布局时（props.tabs 或 schema.tabs）校验字段为全部 Tab 字段的合集，否则用平铺 items
    const ruleItems = activeTabs.value.length ? activeTabs.value.flatMap((tab) => tab.items) : allItems.value;
    for (const item of ruleItems) {
        if (!item.prop) {
            continue;
        }
        const itemRules: FormItemRule[] = [];
        // 动态 required：根据表单值实时计算（如条件必填字段，与 FieldCol 星号显示共用同一判定）
        if (isItemRequired(item, model.value)) {
            itemRules.push(isSelectLikeItem(item) ? Rules.requiredSelect(item.label) : Rules.requiredInput(item.label));
        }
        if (item.validate) {
            const validateFn = item.validate;
            const label = item.label ?? '';
            itemRules.push({
                validator: (_rule: unknown, value: unknown, callback: (error?: Error) => void) => {
                    // 同步与异步校验统一收敛：true 通过；false/字符串/reject 均不通过（Promise reject 用字段默认文案，避免 unhandled rejection）
                    const settle = (res: boolean | string) => {
                        if (res === true) {
                            callback();
                        } else {
                            callback(new Error(res === false ? t(label) : t(res)));
                        }
                    };
                    const res = validateFn(value, model.value);
                    if (res instanceof Promise) {
                        res.then(settle, () => settle(false));
                    } else {
                        settle(res);
                    }
                },
            });
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

/** 单字段校验（缺省全部字段）：向导式分步场景只校验当前步字段 */
const validateField = async (fields?: string | string[]) => {
    return await formRef.value?.validateField(fields);
};

const resetFields = () => {
    formRef.value?.resetFields();
};

const clearValidate = () => {
    formRef.value?.clearValidate();
};

/** 暴露内部表单方法（AutoFormInstance 契约编译期校验，外部编程式校验/单字段校验/重置/清校验） */
const exposed: AutoFormInstance = {
    validate,
    validateField,
    resetFields,
    clearValidate,
};
defineExpose(exposed);
</script>
<style lang="scss" scoped></style>
