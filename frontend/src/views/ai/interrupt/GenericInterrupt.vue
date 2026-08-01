<template>
    <div class="generic-interrupt border border-gray-200 dark:border-gray-700 rounded flex flex-col">
        <!-- 紧凑头部 -->
        <div class="flex items-center justify-between px-3 py-2 border-b border-gray-100 dark:border-gray-800 bg-gray-50/50 dark:bg-gray-800/50">
            <div class="flex items-center gap-2">
                <el-tag type="info" size="small">{{ interruptData?.type || t('ai.interrupt.generic.interrupt') }}</el-tag>
                <span class="text-sm font-medium">{{ interruptData?.title || t('ai.interrupt.generic.operationInterrupted') }}</span>
            </div>
            <el-tag v-if="isProcessed" :type="getActionTag(currentAction)" size="small">
                {{ getActionText(currentAction) }}
            </el-tag>
            <el-tag v-else-if="hasPending" type="info" size="small">待提交</el-tag>
            <el-tag v-else type="warning" size="small">{{ t('ai.interrupt.generic.pending') }}</el-tag>
        </div>

        <div class="px-3 py-2 space-y-2 flex-1">
            <!-- 描述信息 -->
            <div v-if="interruptData?.description" class="text-xs text-gray-500 dark:text-gray-400">
                {{ interruptData.description }}
            </div>

            <!-- 操作结果记录 -->
            <div v-if="resumeInfo" class="flex items-center gap-2 text-xs">
                <span class="text-gray-500 dark:text-gray-400">{{ t('ai.interrupt.generic.operationType') }}:</span>
                <el-tag :type="getActionTag(resumeInfo.action)" size="small">
                    {{ getActionText(resumeInfo.action) }}
                </el-tag>
                <span v-if="resumeInfo.action === 'reject' && resumeInfo.payload?.reason" class="text-gray-700 dark:text-gray-300 truncate max-w-40" :title="String(resumeInfo.payload.reason)">
                    ({{ resumeInfo.payload.reason }})
                </span>
            </div>
        </div>

        <!-- 操作按钮 -->
        <div v-if="!readonly && !isProcessed && !hasPending" class="flex justify-end gap-2 px-3 py-2 border-t border-gray-100 dark:border-gray-800">
            <el-button size="small" @click="handleAction('approve')">{{ t('ai.interrupt.generic.confirm') }}</el-button>
            <el-button size="small" type="danger" @click="handleReject">{{ t('ai.interrupt.generic.reject') }}</el-button>
        </div>
    </div>
</template>

<script setup lang="ts">
/**
 * 通用中断组件
 * 用于未注册特定类型的中断场景，作为降级方案
 */

import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { ElMessageBox } from 'element-plus';
import type { InternalMessage, InterruptActionEvent } from './types';

const { t } = useI18n();

interface Props {
    data: InternalMessage;
    readonly?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
    readonly: false,
});

const emit = defineEmits<{
    action: [action: InterruptActionEvent];
}>();

// 从 data 对象中提取常用字段
const interruptData = computed(() => props.data.extra?.content);
const interruptId = computed(() => props.data?.actionId || props.data?.extra?.actionId || '');
const turnId = computed(() => props.data?.turnId || props.data?.extra?.turnId || '');
const resumeInfo = computed(() => props.data.extra?.resumeInfo);
const pendingResumeInfo = computed(() => props.data.extra?.pendingResumeInfo);
const interruptType = computed(() => props.data.extra?.type || '');

// 根据 resumeInfo.action 计算当前动作
const currentAction = computed(() => resumeInfo.value?.action || pendingResumeInfo.value?.action || '');

// 判断是否已处理（有 resumeInfo 表示已处理）
const isProcessed = computed(() => !!resumeInfo.value);
const hasPending = computed(() => !!pendingResumeInfo.value);

/**
 * 处理用户操作
 */
const handleAction = (action: string, payload?: Record<string, unknown>) => {
    emit('action', {
        turnId: turnId.value || '',
        interruptId: interruptId.value || '',
        interruptType: interruptType.value || '',
        action,
        payload,
    });
};

/**
 * 获取操作类型对应的标签类型
 */
const getActionTag = (actionType: string): 'success' | 'danger' | 'info' => {
    switch (actionType) {
        case 'approve':
        case 'confirm':
            return 'success';
        case 'reject':
        case 'cancel':
            return 'danger';
        default:
            return 'info';
    }
};

/**
 * 处理拒绝操作，弹出输入框填写原因
 */
const handleReject = async () => {
    try {
        const { value: reason } = await ElMessageBox.prompt(t('ai.interrupt.generic.rejectReasonPlaceholder'), t('ai.interrupt.generic.rejectTitle'), {
            confirmButtonText: t('common.confirm'),
            cancelButtonText: t('common.cancel'),
            inputType: 'textarea',
            inputPlaceholder: t('ai.interrupt.generic.rejectReasonPlaceholder'),
            inputValidator: (value: string) => {
                if (!value || !value.trim()) {
                    return t('ai.interrupt.generic.rejectReasonRequired');
                }
                return true;
            },
        });
        handleAction('reject', { reason: reason?.trim() });
    } catch {
        // 用户取消，不做处理
    }
};

/**
 * 获取操作类型的显示文本
 */
const getActionText = (action: string): string => {
    switch (action) {
        case 'approve':
            return t('ai.interrupt.action.approve');
        case 'reject':
            return t('ai.interrupt.action.reject');
        case 'confirm':
            return t('ai.interrupt.action.confirm');
        case 'cancel':
            return t('ai.interrupt.action.cancel');
        default:
            return action;
    }
};
</script>

<style scoped>
.generic-interrupt {
    @apply transition-all duration-300;
}
</style>
