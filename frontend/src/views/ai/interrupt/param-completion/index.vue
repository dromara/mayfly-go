<template>
    <Card class="param-completion-interrupt">
        <!-- 紧凑头部 -->
        <div class="param-completion-interrupt__header">
            <div class="param-completion-interrupt__header-left">
                <Badge variant="secondary" class="param-completion-interrupt__type">{{ t('ai.interrupt.paramCompletion.title') }}</Badge>
                <span class="param-completion-interrupt__desc">{{ interrupt?.description || t('ai.interrupt.paramCompletion.completeParam') }}</span>
            </div>
            <Badge v-if="!readonly" variant="secondary" class="param-completion-interrupt__pending">{{ t('ai.interrupt.paramCompletion.pending') }}</Badge>
            <Badge v-else variant="outline">{{ t('ai.interrupt.paramCompletion.completed') }}</Badge>
        </div>

        <div class="param-completion-interrupt__body">
            <!-- 工具名 -->
            <div v-if="interrupt?.toolName" class="param-completion-interrupt__tool-name">
                <span>{{ t('ai.interrupt.paramCompletion.toolName') }}:</span>
                <span class="param-completion-interrupt__tool-value">{{ interrupt.toolName }}</span>
            </div>

            <!-- 通用选项选择（对齐 tokhub ask_user options 模式） -->
            <OptionsParamInput
                v-if="hasOptions"
                ref="paramInputRef"
                :options="availableOptions"
                :description="interrupt?.description"
                :readonly="readonly"
                :is-confirmed="readonly"
                @change="onParamChange"
            />
            <!-- 参数输入（旧模式 fallback） -->
            <component
                :is="paramInputComponent"
                v-else-if="paramInputComponent"
                ref="paramInputRef"
                v-model:model-value="paramInputValues"
                :params="missingParams"
                :readonly="readonly"
                :is-confirmed="readonly"
                @change="onParamChange"
            />
            <div v-else class="param-completion-interrupt__unsupported">
                {{ t('ai.interrupt.paramCompletion.unsupportedType') }}: {{ paramType }}
            </div>
        </div>

        <!-- 操作按钮 -->
        <div class="param-completion-interrupt__footer">
            <Button size="xs" variant="outline" :disabled="readonly" @click="handleCancel">
                {{ t('ai.interrupt.paramCompletion.cancel') }}
            </Button>
            <Button size="xs" :disabled="readonly || !formValid" @click="handleConfirm">
                {{ t('ai.interrupt.paramCompletion.confirm') }}
            </Button>
        </div>
    </Card>
</template>

<script setup lang="ts">
import { computed, markRaw, ref, watch, type Component, type ComponentPublicInstance } from 'vue';
import { useI18n } from 'vue-i18n';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import type { InterruptComponentProps } from '../types';
import DbParamInput from './DbParamInput.vue';
import GenericParamInput from './GenericParamInput.vue';
import MachineParamInput from './MachineParamInput.vue';
import OptionsParamInput from './OptionsParamInput.vue';

const { t } = useI18n();

const props = withDefaults(defineProps<InterruptComponentProps>(), {
    readonly: false,
});

/** 从 metadata 提取参数信息 */
const missingParams = computed(() => {
    return (props.interrupt?.metadata?.payload as unknown[]) || [];
});

const paramType = computed(() => {
    return String(props.interrupt?.metadata?.paramType || '');
});

/** 后端传递的可选项列表（通用 ask_user 模式） */
const availableOptions = computed(() => {
    const opts = props.interrupt?.metadata?.options as Array<{ label: string; value: string }> | undefined;
    return opts || [];
});

/** 是否有可选项（优先使用通用选项模式） */
const hasOptions = computed(() => availableOptions.value.length > 0);

const paramInputComponents: Record<string, Component> = {
    db: markRaw(DbParamInput),
    machine: markRaw(MachineParamInput),
};

const paramInputComponent = computed(() => {
    const type = paramType.value.toLowerCase();
    return paramInputComponents[type] || markRaw(GenericParamInput);
});

interface ParamInputInstance extends ComponentPublicInstance {
    isValid?: () => boolean;
    getValues?: () => Record<string, unknown>;
    getCacheableParams?: () => string[];
}

const paramInputRef = ref<ParamInputInstance | null>(null);
const paramInputValues = ref<Record<string, unknown>>({});
const formValid = ref(false);

watch(paramInputValues, () => {
    formValid.value = paramInputRef.value?.isValid?.() ?? false;
}, { deep: true });

const onParamChange = (values: Record<string, unknown>) => {
    paramInputValues.value = values;
    formValid.value = paramInputRef.value?.isValid?.() ?? Object.keys(values).length > 0;
};

const handleConfirm = () => {
    if (props.readonly) return;
    formValid.value = paramInputRef.value?.isValid?.() ?? false;
    if (!formValid.value) return;

    let inputValues = paramInputRef.value?.getValues?.();
    if (!inputValues) {
        inputValues = { params: { ...paramInputValues.value } };
    }

    const payload: Record<string, unknown> = { ...inputValues };
    const cacheableParams = paramInputRef.value?.getCacheableParams?.() || [];
    if (cacheableParams.length > 0) payload.caches = cacheableParams;

    props.onAction({
        turnId: props.turnId,
        interruptId: props.interrupt?.actionId || '',
        interruptType: props.interrupt?.type || '',
        action: 'complete',
        payload,
        toolCallId: props.interrupt?.toolCallId,
    });
};

const handleCancel = () => {
    if (props.readonly) return;
    props.onAction({
        turnId: props.turnId,
        interruptId: props.interrupt?.actionId || '',
        interruptType: props.interrupt?.type || '',
        action: 'cancel',
        toolCallId: props.interrupt?.toolCallId,
    });
};
</script>

<style scoped>
.param-completion-interrupt {
    gap: 0;
    padding: 0;
    box-shadow: none;
}

.param-completion-interrupt__type {
    color: var(--el-color-primary);
}

.param-completion-interrupt__pending {
    color: var(--el-color-warning);
}

.param-completion-interrupt__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 10px;
    border-bottom: 1px solid var(--el-border-color-extra-light);
    background: var(--el-fill-color-light);
}

.param-completion-interrupt__header-left {
    display: flex;
    align-items: center;
    gap: 6px;
}

.param-completion-interrupt__desc {
    font-size: 13px;
    font-weight: 500;
}

.param-completion-interrupt__body {
    padding: 6px 10px;
    display: flex;
    flex-direction: column;
    gap: 6px;
    flex: 1;
}

.param-completion-interrupt__tool-name {
    font-size: 12px;
    color: var(--el-text-color-secondary);
}

.param-completion-interrupt__tool-value {
    font-family: monospace;
    color: var(--el-color-primary);
    margin-left: 4px;
}

.param-completion-interrupt__unsupported {
    text-align: center;
    font-size: 12px;
    color: var(--el-text-color-placeholder);
    padding: 8px 0;
}

.param-completion-interrupt__footer {
    display: flex;
    justify-content: flex-end;
    gap: 6px;
    padding: 6px 10px;
    border-top: 1px solid var(--el-border-color-extra-light);
}
</style>
