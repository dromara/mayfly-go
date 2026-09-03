<template>
    <div class="generic-param-input">
        <div class="generic-param-input__hint">
            {{ t('ai.interrupt.paramCompletion.enterParamsHint') }}
        </div>

        <el-form label-position="top" size="small">
            <el-form-item v-for="param in params" :key="param.param" :label="param.name || param.param">
                <el-input
                    v-model="paramValues[param.param]"
                    :placeholder="t('ai.interrupt.paramCompletion.enterParamPlaceholder', { name: param.name || param.param })"
                    :disabled="readonly"
                    @input="onInput(param.param, $event)"
                />
            </el-form-item>
        </el-form>
    </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

interface ParamInfo {
    param: string;
    name: string;
    cacheable?: boolean;
}

interface Props {
    params: ParamInfo[];
    readonly?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
    readonly: false,
});

const emit = defineEmits<{
    change: [values: Record<string, any>];
}>();

const { t } = useI18n();

// 使用 defineModel 实现双向绑定
const paramValues = defineModel<Record<string, any>>('modelValue', {
    default: {},
});

// 输入事件
const onInput = (param: string, value: any) => {
    emit('change', { ...paramValues.value });
};

// 获取当前值
const getValues = () => ({ ...paramValues.value });

// 检查是否所有必填参数都有值
const isValid = () => {
    return props.params.every((p) => {
        const val = paramValues.value[p.param];
        return val !== undefined && val !== '';
    });
};

// 暴露方法给父组件
defineExpose({
    getValues,
    isValid,
});
</script>

<style scoped>
.generic-param-input {
    padding: 8px;
}

.generic-param-input__hint {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin-bottom: 8px;
}
</style>
