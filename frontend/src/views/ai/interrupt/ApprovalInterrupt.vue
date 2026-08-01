<template>
    <div class="approval-interrupt border border-gray-200 dark:border-gray-700 rounded flex flex-col">
        <!-- 紧凑头部 -->
        <div class="flex items-center justify-between px-3 py-2 border-b border-gray-100 dark:border-gray-800 bg-gray-50/50 dark:bg-gray-800/50">
            <div class="flex items-center gap-2">
                <el-tag type="warning" size="small">{{ t('ai.interrupt.approval.title') }}</el-tag>
                <span class="text-sm font-medium">{{ interruptData?.title }}</span>
            </div>
            <enum-tag v-if="isProcessed" :enums="InterruptAction" :value="currentAction" />
            <el-tag v-else-if="hasPending" type="info" size="small">{{ t('ai.interrupt.approval.pendingSubmit') }}</el-tag>
            <el-tag v-else type="warning" size="small">{{ t('ai.interrupt.approval.pendingApproval') }}</el-tag>
        </div>

        <div class="px-3 py-2 space-y-2 flex-1">
            <!-- 描述信息 -->
            <div v-if="interruptData?.description" class="text-xs text-gray-500 dark:text-gray-400">
                {{ interruptData.description }}
            </div>

            <!-- 工具名一行带过 -->
            <div v-if="interruptData?.toolInfo" class="text-xs text-gray-500 dark:text-gray-400">
                <span>{{ t('ai.interrupt.approval.toolName') }}:</span>
                <span class="font-mono text-blue-600 dark:text-blue-400 ml-1">{{ interruptData.toolInfo.name }}</span>
            </div>

            <!-- 执行参数：固定高度常驻显示 -->
            <div v-if="interruptData?.arguments" class="text-xs">
                <div class="text-gray-400 text-xs mb-1">{{ t('ai.interrupt.approval.executionParams') }}</div>
                <div class="h-25 overflow-y-auto">
                    <pre class="p-1.5 bg-white dark:bg-gray-900 rounded border border-gray-200 dark:border-gray-700 overflow-x-auto text-xs">{{
                        formatJson(interruptData.arguments)
                    }}</pre>
                </div>
            </div>

            <!-- 待提交状态 -->
            <div v-if="hasPending && !isProcessed" class="flex items-center gap-2 text-xs text-yellow-600 dark:text-yellow-400">
                <span>{{ t('ai.interrupt.approval.selected') }}</span>
                <enum-tag :enums="InterruptAction" :value="pendingResumeInfo?.action" />
                <span v-if="pendingResumeInfo?.action === 'reject' && pendingResumeInfo?.payload?.reason" class="truncate max-w-40" :title="String(pendingResumeInfo?.payload?.reason)">
                    ({{ pendingResumeInfo?.payload?.reason }})
                </span>
            </div>

            <!-- 操作结果记录 -->
            <div v-if="resumeInfo" class="flex items-center gap-2 text-xs">
                <span class="text-gray-500 dark:text-gray-400">{{ t('ai.interrupt.approval.operationType') }}:</span>
                <enum-tag :enums="InterruptAction" :value="resumeInfo.action" />
                <span v-if="resumeInfo.action === 'reject' && resumeInfo.payload?.reason" class="text-gray-700 dark:text-gray-300 truncate max-w-40" :title="String(resumeInfo.payload.reason)">
                    ({{ resumeInfo.payload.reason }})
                </span>
            </div>
        </div>

        <!-- 操作按钮 -->
        <div v-if="!readonly && !isProcessed && !hasPending" class="flex justify-end gap-2 px-3 py-2 border-t border-gray-100 dark:border-gray-800">
            <el-button size="small" @click="handleApprove">{{ t('ai.interrupt.approval.approve') }}</el-button>
            <el-button size="small" type="danger" @click="handleReject">{{ t('ai.interrupt.approval.reject') }}</el-button>
        </div>
    </div>
</template>

<script setup lang="ts">
/**
 * 审批类型中断组件
 * 用于需要用户确认的高危操作场景
 */

import EnumValue from '@/common/Enum';
import { formatJson } from '@/common/utils/format';
import EnumTag from '@/components/enum-tag/EnumTag.vue';
import { ElMessageBox } from 'element-plus';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import type { InternalMessage, InterruptActionEvent } from './types';

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

const { t } = useI18n();

// 从 data 中提取常用字段
const interruptId = computed(() => props.data.actionId || props.data.extra?.actionId || '');
const turnId = computed(() => props.data.turnId || props.data.extra?.turnId || '');
const interruptData = computed(() => props.data.extra?.content);
const resumeInfo = computed(() => props.data.extra?.resumeInfo);
const pendingResumeInfo = computed(() => props.data.extra?.pendingResumeInfo);
const currentAction = computed(() => resumeInfo.value?.action || pendingResumeInfo.value?.action);
const isProcessed = computed(() => !!resumeInfo.value);
const hasPending = computed(() => !!pendingResumeInfo.value);
const interruptType = computed(() => props.data.extra?.type || '');

const InterruptAction = {
    Approve: EnumValue.of('approve', 'ai.interrupt.action.approve').tagTypeSuccess(),
    Reject: EnumValue.of('reject', 'ai.interrupt.action.reject').tagTypeDanger(),
};

/**
 * 处理审批通过操作
 */
const handleApprove = () => {
    handleAction('approve');
};

/**
 * 处理审批拒绝操作，弹出输入框
 */
const handleReject = async () => {
    try {
        const { value: reason } = await ElMessageBox.prompt(t('ai.interrupt.approval.rejectReasonPlaceholder'), t('ai.interrupt.approval.rejectTitle'), {
            confirmButtonText: t('common.confirm'),
            cancelButtonText: t('common.cancel'),
            inputType: 'textarea',
            inputPlaceholder: t('ai.interrupt.approval.rejectReasonPlaceholder'),
            inputValidator: (value: string) => {
                if (!value || !value.trim()) {
                    return t('ai.interrupt.approval.rejectReasonRequired');
                }
                return true;
            },
        });

        // 用户输入了拒绝原因，提交操作
        handleAction('reject', { reason: reason?.trim() });
    } catch {
        // 用户取消了操作，不做任何处理
    }
};

/**
 * 处理用户操作
 * @param action 操作类型
 * @param payload 额外数据
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
</script>

<style scoped>
.approval-interrupt {
    @apply transition-all duration-300;
}
</style>
