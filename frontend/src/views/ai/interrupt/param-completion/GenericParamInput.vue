<template>
    <div class="generic-param-input">
        <div class="generic-param-input__hint">
            {{ t('ai.interrupt.paramCompletion.enterParamsHint') }}
        </div>

        <el-form label-position="top" size="small">
            <el-form-item v-for="param in params" :key="param.param" :label="param.name || param.param">
                <el-input
                    :model-value="paramValues[param.param]"
                    :placeholder="t('ai.interrupt.paramCompletion.enterParamPlaceholder', { name: param.name || param.param })"
                    :disabled="readonly || isConfirmed"
                    @update:model-value="patch(param.param, $event)"
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
    /**
     * 已确认（历史/轨迹里的终态）：必须参与禁用判定。
     *
     * 父层一直在传 `:is-confirmed`，但本组件没声明该 prop → 属性落进 $attrs 被丢弃，
     * 只读轨迹里的参数输入框仍可直接编辑（与 Db/Machine 两个兄弟组件行为不一致）
     */
    isConfirmed?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
    readonly: false,
    isConfirmed: false,
});

const emit = defineEmits<{
    change: [values: Record<string, any>];
}>();

const { t } = useI18n();

// 使用 defineModel 实现双向绑定
const paramValues = defineModel<Record<string, any>>('modelValue', {
    default: () => ({}),
});

/**
 * 写回：整体替换值对象后报 change。
 *
 * 不用 `v-model="paramValues[param.param]"`：那是原地 mutate 传入对象的属性，
 * update:modelValue 永不触发，父层拿不到变更事件；而 onInput 里发的是 mutate 前的
 * `{...paramValues.value}` 拷贝，与输入值存时差一个周期
 */
const patch = (param: string, value: unknown) => {
    paramValues.value = { ...paramValues.value, [param]: value };
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
