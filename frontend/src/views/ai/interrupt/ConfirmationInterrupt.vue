<template>
    <Card class="confirmation-interrupt">
        <!-- 紧凑头部 -->
        <div class="confirmation-interrupt__header">
            <div class="confirmation-interrupt__header-left">
                <Badge variant="secondary" class="confirmation-interrupt__type">{{ t('ai.interrupt.confirmation.title') }}</Badge>
                <span class="confirmation-interrupt__desc">{{ interrupt?.description }}</span>
            </div>
            <Badge v-if="!readonly" variant="secondary" class="confirmation-interrupt__pending">{{ t('ai.interrupt.confirmation.pendingConfirmation') }}</Badge>
            <Badge v-else variant="outline">{{ t('ai.interrupt.confirmation.resolved') }}</Badge>
        </div>

        <div class="confirmation-interrupt__body">
            <!-- 确认选项 -->
            <div v-if="options.length > 0" class="confirmation-interrupt__options">
                <div class="confirmation-interrupt__options-label">{{ t('ai.interrupt.confirmation.pleaseSelect') }}</div>
                <el-radio-group v-model="selectedOption" :disabled="readonly">
                    <el-radio v-for="option in options" :key="option.value" :value="option.value" class="confirmation-interrupt__radio">
                        {{ option.label }}
                    </el-radio>
                </el-radio-group>
            </div>
        </div>

        <!-- 操作按钮 -->
        <div v-if="!readonly" class="confirmation-interrupt__footer">
            <Button size="xs" :disabled="!selectedOption && options.length > 0" @click="handleAction('confirm')">
                {{ t('ai.interrupt.confirmation.confirm') }}
            </Button>
            <Button size="xs" variant="outline" @click="handleAction('cancel')">{{ t('ai.interrupt.confirmation.cancel') }}</Button>
        </div>
    </Card>
</template>

<script setup lang="ts">
/**
 * 确认类型中断组件
 * 用于需要用户从多个选项中选择的场景
 */
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import type { InterruptComponentProps } from './types';

const props = withDefaults(defineProps<InterruptComponentProps>(), {
    readonly: false,
});

const { t } = useI18n();
const selectedOption = ref<string>();

/** 从 metadata 中提取选项列表 */
const options = computed(() => {
    const opts = props.interrupt?.metadata?.options as Array<{ value: string; label: string }> | undefined;
    return opts || [];
});

const handleAction = (action: string) => {
    props.onAction({
        turnId: props.turnId,
        interruptId: props.interrupt?.actionId || '',
        interruptType: props.interrupt?.type || '',
        action,
        payload: selectedOption.value ? { selected: selectedOption.value } : undefined,
        toolCallId: props.interrupt?.toolCallId,
    });
};
</script>

<style scoped>
.confirmation-interrupt {
    gap: 0;
    padding: 0;
    box-shadow: none;
}

.confirmation-interrupt__type {
    color: var(--el-color-primary);
}

.confirmation-interrupt__pending {
    color: var(--el-color-warning);
}

.confirmation-interrupt__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    border-bottom: 1px solid var(--el-border-color-extra-light);
    background: var(--el-fill-color-light);
}

.confirmation-interrupt__header-left {
    display: flex;
    align-items: center;
    gap: 8px;
}

.confirmation-interrupt__desc {
    font-size: 13px;
    font-weight: 500;
}

.confirmation-interrupt__body {
    padding: 8px 12px;
    flex: 1;
}

.confirmation-interrupt__options {
    background: var(--el-color-primary-light-9);
    border-radius: 6px;
    padding: 8px;
    border: 1px solid var(--el-color-primary-light-7);
}

.confirmation-interrupt__options-label {
    font-size: 12px;
    font-weight: 500;
    color: var(--el-color-primary);
    margin-bottom: 4px;
}

.confirmation-interrupt__radio {
    display: block;
    margin-bottom: 4px;
    font-size: 12px;
}

.confirmation-interrupt__footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding: 8px 12px;
    border-top: 1px solid var(--el-border-color-extra-light);
}
</style>
