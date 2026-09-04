<template>
    <Card class="approval-interrupt">
        <!-- 紧凑头部 -->
        <div class="approval-interrupt__header">
            <div class="approval-interrupt__header-left">
                <Badge variant="secondary" class="approval-interrupt__type">{{ t('ai.interrupt.approval.title') }}</Badge>
                <span class="approval-interrupt__desc">{{ interrupt?.description }}</span>
            </div>
            <Badge v-if="!readonly" variant="secondary" class="approval-interrupt__pending">{{ t('ai.interrupt.approval.pendingApproval') }}</Badge>
            <Badge v-else variant="outline">{{ t('ai.interrupt.approval.resolved') }}</Badge>
        </div>

        <div class="approval-interrupt__body">
            <!-- 工具名 -->
            <div v-if="interrupt?.toolName" class="approval-interrupt__tool-name">
                <span>{{ t('ai.interrupt.approval.toolName') }}:</span>
                <span class="approval-interrupt__tool-value">{{ interrupt.toolName }}</span>
            </div>

            <!-- 元数据中的参数 -->
            <div v-if="interrupt?.metadata?.arguments" class="approval-interrupt__params">
                <div class="approval-interrupt__params-label">{{ t('ai.interrupt.approval.executionParams') }}</div>
                <div class="approval-interrupt__params-content">
                    <pre class="approval-interrupt__pre">{{
                        formatJson(interrupt.metadata.arguments)
                    }}</pre>
                </div>
            </div>
        </div>

        <!-- 操作按钮 -->
        <div v-if="!readonly" class="approval-interrupt__footer">
            <Button size="xs" variant="outline" @click="handleAction('approve')">{{ t('ai.interrupt.approval.approve') }}</Button>
            <RejectReasonPopover :label="t('ai.interrupt.approval.reject')" @confirm="(reason: string) => handleAction('reject', { reason })" />
        </div>
    </Card>
</template>

<script setup lang="ts">
/**
 * 审批类型中断组件
 * 用于需要用户确认的高危操作场景
 */
import { formatJson } from '@/common/utils/format';
import { useI18n } from 'vue-i18n';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import type { InterruptComponentProps } from './types';
import RejectReasonPopover from './RejectReasonPopover.vue';

const props = withDefaults(defineProps<InterruptComponentProps>(), {
    readonly: false,
});

const { t } = useI18n();

const handleAction = (action: string, payload?: Record<string, unknown>) => {
    props.onAction({
        turnId: props.turnId,
        interruptId: props.interrupt?.actionId || '',
        interruptType: props.interrupt?.type || '',
        action,
        payload,
        toolCallId: props.interrupt?.toolCallId,
    });
};
</script>

<style scoped>
.approval-interrupt {
    gap: 0;
    padding: 0;
    box-shadow: none;
}

.approval-interrupt__type {
    color: var(--el-color-warning);
}

.approval-interrupt__pending {
    color: var(--el-color-warning);
}

.approval-interrupt__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    border-bottom: 1px solid var(--el-border-color-extra-light);
    background: var(--el-fill-color-light);
}

.approval-interrupt__header-left {
    display: flex;
    align-items: center;
    gap: 8px;
}

.approval-interrupt__desc {
    font-size: 13px;
    font-weight: 500;
}

.approval-interrupt__body {
    padding: 8px 12px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    flex: 1;
}

.approval-interrupt__tool-name {
    font-size: 12px;
    color: var(--el-text-color-secondary);
}

.approval-interrupt__tool-value {
    font-family: monospace;
    color: var(--el-color-primary);
    margin-left: 4px;
}

.approval-interrupt__params-label {
    font-size: 12px;
    color: var(--el-text-color-placeholder);
    margin-bottom: 4px;
}

.approval-interrupt__params-content {
    max-height: 100px;
    overflow-y: auto;
}

.approval-interrupt__pre {
    margin: 0;
    padding: 6px;
    background: var(--el-bg-color);
    border-radius: 4px;
    border: 1px solid var(--el-border-color-light);
    font-size: 12px;
    line-height: 1.5;
    overflow-x: auto;
    white-space: pre-wrap;
    word-break: break-all;
}

.approval-interrupt__footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding: 8px 12px;
    border-top: 1px solid var(--el-border-color-extra-light);
}
</style>
