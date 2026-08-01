<template>
    <div class="param-completion-interrupt border border-gray-200 dark:border-gray-700 rounded flex flex-col">
        <!-- 紧凑头部 -->
        <div class="flex items-center justify-between px-3 py-2 border-b border-gray-100 dark:border-gray-800 bg-gray-50/50 dark:bg-gray-800/50">
            <div class="flex items-center gap-2">
                <el-tag type="primary" size="small">{{ t('ai.interrupt.paramCompletion.title') }}</el-tag>
                <span class="text-sm font-medium">{{ interruptData?.title || t('ai.interrupt.paramCompletion.completeParam') }}</span>
            </div>
            <!-- 根据操作类型显示不同状态 -->
            <el-tag v-if="isProcessed && currentAction === 'complete'" type="success" size="small">{{ t('ai.interrupt.paramCompletion.completed') }}</el-tag>
            <el-tag v-else-if="isProcessed && currentAction === 'cancel'" type="info" size="small">{{ t('ai.interrupt.action.cancel') }}</el-tag>
            <el-tag v-else-if="hasPending" type="info" size="small">待提交</el-tag>
            <el-tag v-else type="warning" size="small">{{ t('ai.interrupt.paramCompletion.pending') }}</el-tag>
        </div>

        <div class="px-3 py-2 space-y-2 flex-1">
            <!-- 描述信息 -->
            <div v-if="interruptData?.description" class="text-xs text-gray-500 dark:text-gray-400">
                {{ interruptData.description }}
            </div>

            <!-- 工具名一行带过 -->
            <div v-if="interruptData?.toolInfo" class="text-xs text-gray-500 dark:text-gray-400">
                <span>{{ t('ai.interrupt.paramCompletion.toolName') }}:</span>
                <span class="font-mono text-blue-600 dark:text-blue-400 ml-1">{{ interruptData.toolInfo.name }}</span>
            </div>

            <!-- 参数输入 -->
            <component
                :is="paramInputComponent"
                v-if="paramInputComponent"
                ref="paramInputRef"
                v-model:model-value="paramInputValues"
                :params="missingParams"
                :readonly="isProcessed"
                :is-confirmed="isProcessed"
                @change="onParamChange"
            />
            <div v-else class="text-center text-xs text-gray-500 dark:text-gray-400 py-2">
                {{ t('ai.interrupt.paramCompletion.unsupportedType') }}: {{ paramType }}
            </div>
        </div>

        <!-- 操作按钮 -->
        <div class="flex justify-end gap-2 px-3 py-2 border-t border-gray-100 dark:border-gray-800">
            <el-button size="small" @click="handleCancel" :disabled="isProcessed || hasPending">
                {{ t('ai.interrupt.paramCompletion.cancel') }}
            </el-button>
            <el-button type="primary" size="small" @click="handleConfirm" :disabled="isProcessed || hasPending || !formValid">
                {{ t('ai.interrupt.paramCompletion.confirm') }}
            </el-button>
        </div>
    </div>
</template>

<script setup lang="ts">
import { computed, inject, markRaw, nextTick, ref, watch, type ComponentPublicInstance } from 'vue';
import { useI18n } from 'vue-i18n';
import type { InternalMessage, InterruptActionEvent } from '../types';
import type { Component } from 'vue';

// 引入参数输入组件
import DbParamInput from './DbParamInput.vue';
import GenericParamInput from './GenericParamInput.vue';
import MachineParamInput from './MachineParamInput.vue';

const { t } = useI18n();

interface Props {
    data: InternalMessage;
    readonly?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
    readonly: false,
});

const emit = defineEmits<{
    action: [data: { turnId: string; interruptId: string; interruptType: string; action: string; payload?: Record<string, unknown> | null }];
}>();

// 注入父组件提供的中断操作处理器，绕过 Vue 动态组件事件传递问题
const handleInterruptAction = inject<(action: InterruptActionEvent) => void>('handleInterruptAction');

// 从 data 中提取常用字段（fallback 到 extra 中的字段，防止 actionId/turnId 顶层丢失）
const interruptData = computed(() => props.data.extra?.content);
const resumeInfo = computed(() => props.data.extra?.resumeInfo);
const pendingResumeInfo = computed(() => props.data.extra?.pendingResumeInfo);
const turnId = computed(() => props.data.turnId || props.data.extra?.turnId || '');
const interruptId = computed(() => props.data.actionId || props.data.extra?.actionId || '');
const interruptType = computed(() => props.data.extra?.type || '');

// 缺失参数列表
const missingParams = computed(() => {
    return interruptData.value?.payload || [];
});

// 参数类型
const paramType = computed(() => {
    return String(interruptData.value?.paramType || '');
});

// 参数输入组件映射
const paramInputComponents: Record<string, Component> = {
    db: markRaw(DbParamInput),
    machine: markRaw(MachineParamInput),
    // 后续可以添加更多类型
    // redis: markRaw(RedisParamInput),
};

// 动态获取参数输入组件（默认为通用输入框）
const paramInputComponent = computed(() => {
    const type = paramType.value.toLowerCase();
    return paramInputComponents[type] || markRaw(GenericParamInput);
});

/** 参数输入组件实例方法 */
interface ParamInputInstance extends ComponentPublicInstance {
    isValid?: () => boolean;
    getValues?: () => Record<string, unknown>;
    getCacheableParams?: () => string[];
}

// 参数输入组件引用
const paramInputRef = ref<ParamInputInstance | null>(null);

// 初始参数值（从 resumeInfo 或 pendingResumeInfo 恢复）
const paramInputValues = ref<Record<string, unknown>>({});

// 监听 resumeInfo / pendingResumeInfo 变化，更新 paramInputValues
watch(
    () => resumeInfo.value?.payload?.params || pendingResumeInfo.value?.payload?.params,
    (newParams) => {
        if (newParams) {
            paramInputValues.value = { ...(newParams as Record<string, unknown>) };
        }
    },
    { immediate: true, deep: true }
);

// 是否已处理（仅基于 resumeInfo）
const isProcessed = computed(() => !!resumeInfo.value);
const hasPending = computed(() => !!pendingResumeInfo.value);

// 当前操作类型（complete/cancel）
const currentAction = computed(() => resumeInfo.value?.action || pendingResumeInfo.value?.action || '');

// 参数填写校验状态
const formValid = ref(false);

// 监听参数值变化，实时更新校验状态
watch(
    paramInputValues,
    () => {
        formValid.value = paramInputRef.value?.isValid?.() ?? false;
    },
    { deep: true }
);

// 参数变化回调
const onParamChange = (_values: Record<string, unknown>) => {
    // Param changed
};

// 处理确认操作
const handleConfirm = () => {
    if (isProcessed.value) {
        return;
    }

    // 实时校验
    formValid.value = paramInputRef.value?.isValid?.() ?? false;
    if (!formValid.value) {
        return;
    }

    // 获取参数输入组件的值，优先使用 ref 获取，失败则回退到 paramInputValues
    let inputValues = paramInputRef.value?.getValues?.();
    if (!inputValues) {
        inputValues = { params: { ...paramInputValues.value } };
    }

    // 构建 payload
    const payload: Record<string, unknown> = {
        ...inputValues,
    };

    // 添加需要缓存的参数名
    const cacheableParams = paramInputRef.value?.getCacheableParams?.() || [];
    if (cacheableParams.length > 0) {
        payload.caches = cacheableParams;
    }

    const actionData = {
        turnId: turnId.value || '',
        interruptId: interruptId.value || '',
        interruptType: interruptType.value || '',
        action: 'complete',
        payload,
    };
    if (handleInterruptAction) {
        handleInterruptAction(actionData);
    } else {
        emit('action', actionData);
    }
};

// 处理取消操作
const handleCancel = () => {
    if (isProcessed.value) {
        return;
    }
    const actionData = {
        turnId: turnId.value || '',
        interruptId: interruptId.value || '',
        interruptType: interruptType.value || '',
        action: 'cancel',
        payload: undefined as Record<string, unknown> | undefined,
    };
    if (handleInterruptAction) {
        handleInterruptAction(actionData);
    } else {
        emit('action', actionData);
    }
};
</script>

<style scoped>
.param-completion-interrupt {
    transition: all 0.3s ease;
}
</style>
