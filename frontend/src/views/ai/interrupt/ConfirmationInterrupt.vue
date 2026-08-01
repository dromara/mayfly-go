<template>
    <div class="confirmation-interrupt border border-gray-200 dark:border-gray-700 rounded flex flex-col">
        <!-- 紧凑头部 -->
        <div class="flex items-center justify-between px-3 py-2 border-b border-gray-100 dark:border-gray-800 bg-gray-50/50 dark:bg-gray-800/50">
            <div class="flex items-center gap-2">
                <el-tag type="primary" size="small">{{ t('ai.interrupt.confirmation.title') }}</el-tag>
                <span class="text-sm font-medium">{{ interruptData?.title }}</span>
            </div>
            <enum-tag v-if="isProcessed" :enums="InterruptAction" :value="currentAction" />
            <el-tag v-else-if="hasPending" type="info" size="small">待提交</el-tag>
            <el-tag v-else type="warning" size="small">{{ t('ai.interrupt.confirmation.pendingConfirmation') }}</el-tag>
        </div>

        <div class="px-3 py-2 space-y-2 flex-1">
            <!-- 描述信息 -->
            <div v-if="interruptData?.description" class="text-xs text-gray-500 dark:text-gray-400">
                {{ interruptData.description }}
            </div>

            <!-- 确认选项 -->
            <div v-if="interruptData?.options" class="bg-blue-50 dark:bg-blue-900/20 rounded p-2 border border-blue-200 dark:border-blue-800">
                <div class="text-xs font-medium text-blue-600 dark:text-blue-400 mb-1">{{ t('ai.interrupt.confirmation.pleaseSelect') }}</div>
                <el-radio-group v-model="selectedOption" :disabled="readonly || isProcessed || hasPending">
                    <el-radio v-for="option in interruptData.options" :key="option.value" :value="option.value" class="block mb-1 text-xs">
                        {{ option.label }}
                    </el-radio>
                </el-radio-group>
            </div>

            <!-- 操作结果记录 -->
            <div v-if="resumeInfo" class="flex items-center gap-2 text-xs">
                <span class="text-gray-500 dark:text-gray-400">{{ t('ai.interrupt.confirmation.operationType') }}:</span>
                <enum-tag :enums="InterruptAction" :value="resumeInfo.action" />
                <span v-if="resumeInfo.payload" class="text-gray-500 dark:text-gray-400 ml-1">{{ resumeInfo.payload }}</span>
            </div>
        </div>

        <!-- 操作按钮 -->
        <div v-if="!readonly && !isProcessed && !hasPending" class="flex justify-end gap-2 px-3 py-2 border-t border-gray-100 dark:border-gray-800">
            <el-button size="small" type="primary" @click="handleAction('confirm', selectedOption)" :disabled="!selectedOption">
                {{ t('ai.interrupt.confirmation.confirm') }}
            </el-button>
            <el-button size="small" @click="handleAction('cancel')">{{ t('ai.interrupt.confirmation.cancel') }}</el-button>
        </div>
    </div>
</template>

<script setup lang="ts">
/**
 * 确认类型中断组件
 * 用于需要用户从多个选项中选择的场景
 */

import { EnumValue } from '@/common/Enum';
import EnumTag from '@/components/enum-tag/EnumTag.vue';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
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

const selectedOption = ref<string>();

// 从 data 对象中提取常用字段
const interruptData = computed(() => props.data.extra?.content);
const interruptId = computed(() => props.data.actionId || props.data.extra?.actionId || '');
const turnId = computed(() => props.data.turnId || props.data.extra?.turnId || '');
const resumeInfo = computed(() => props.data.extra?.resumeInfo);
const pendingResumeInfo = computed(() => props.data.extra?.pendingResumeInfo);
const interruptType = computed(() => props.data.extra?.type || '');

// 根据 resumeInfo.action 计算当前状态
const currentAction = computed(() => resumeInfo.value?.action || pendingResumeInfo.value?.action);

// 判断是否已处理（有 resumeInfo 表示已处理）
const isProcessed = computed(() => !!resumeInfo.value);
const hasPending = computed(() => !!pendingResumeInfo.value);

// 从 pendingResumeInfo 恢复已选择的选项
watch(
    () => pendingResumeInfo.value?.payload,
    (payload) => {
        if (payload && typeof payload === 'string') {
            selectedOption.value = payload;
        }
    },
    { immediate: true }
);

const InterruptAction = {
    Confirm: EnumValue.of('confirm', 'ai.interrupt.action.confirm').tagTypeSuccess(),
    Cancel: EnumValue.of('cancel', 'ai.interrupt.action.cancel').tagTypeDanger(),
};

/**
 * 处理用户操作
 * @param action 操作类型
 * @param payload 额外数据（如选中的选项值）
 */
const handleAction = (action: string, payload?: unknown) => {
    emit('action', {
        turnId: turnId.value || '',
        interruptId: interruptId.value || '',
        interruptType: interruptType.value || '',
        action,
        payload: payload as Record<string, unknown> | undefined,
    });
};
</script>

<style scoped>
.confirmation-interrupt {
    @apply transition-all duration-300;
}
</style>
