<template>
    <Card class="generic-interrupt">
        <!-- 紧凑头部 -->
        <div class="generic-interrupt__header">
            <div class="generic-interrupt__header-left">
                <Badge variant="secondary">{{ interrupt?.type || t('ai.interrupt.generic.interrupt') }}</Badge>
                <span class="generic-interrupt__desc">{{ interrupt?.description || t('ai.interrupt.generic.operationInterrupted') }}</span>
            </div>
            <Badge v-if="!readonly" variant="secondary" class="generic-interrupt__pending">{{ t('ai.interrupt.generic.pending') }}</Badge>
            <Badge v-else variant="outline">{{ t('ai.interrupt.generic.resolved') }}</Badge>
        </div>

        <!-- 操作按钮 -->
        <div v-if="!readonly" class="generic-interrupt__footer">
            <Button size="xs" variant="outline" @click="handleAction('approve')">{{ t('ai.interrupt.generic.confirm') }}</Button>
            <RejectReasonPopover :label="t('ai.interrupt.generic.reject')" @confirm="(reason: string) => handleAction('reject', { reason })" />
        </div>
    </Card>
</template>

<script setup lang="ts">
/**
 * 通用中断组件
 * 用于未注册特定类型的中断场景，作为降级方案
 */
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
.generic-interrupt {
    gap: 0;
    padding: 0;
    box-shadow: none;
}

.generic-interrupt__pending {
    color: var(--el-color-warning);
}

.generic-interrupt__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    border-bottom: 1px solid var(--el-border-color-extra-light);
    background: var(--el-fill-color-light);
}

.generic-interrupt__header-left {
    display: flex;
    align-items: center;
    gap: 8px;
}

.generic-interrupt__desc {
    font-size: 13px;
    font-weight: 500;
}

.generic-interrupt__footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding: 8px 12px;
    border-top: 1px solid var(--el-border-color-extra-light);
}
</style>
