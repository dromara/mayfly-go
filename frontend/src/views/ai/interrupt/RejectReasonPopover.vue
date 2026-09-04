<template>
    <el-popover v-model:visible="visible" trigger="click" :width="280" placement="top-end">
        <template #reference>
            <Button size="xs" variant="destructive">{{ label }}</Button>
        </template>
        <div class="reject-reason">
            <div class="reject-reason__title">{{ t('ai.interrupt.generic.rejectTitle') }}</div>
            <el-input
                v-model="reason"
                type="textarea"
                :rows="3"
                :placeholder="t('ai.interrupt.generic.rejectReasonPlaceholder')"
                @keydown.enter.exact.prevent
            />
            <div v-if="error" class="reject-reason__error">{{ error }}</div>
            <div class="reject-reason__actions">
                <Button size="xs" variant="outline" @click="visible = false">{{ t('common.cancel') }}</Button>
                <Button size="xs" @click="handleConfirm">{{ t('common.confirm') }}</Button>
            </div>
        </div>
    </el-popover>
</template>

<script setup lang="ts">
/**
 * 拒绝原因就近输入组件
 * 拒绝按钮直接挂载 popover，避免居中大弹窗打断视觉焦点
 */
import { ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { Button } from '@/components/ui/button';

defineProps<{ label: string }>();

const emit = defineEmits<{ (e: 'confirm', reason: string): void }>();

const { t } = useI18n();

const visible = ref(false);
const reason = ref('');
const error = ref('');

watch(visible, (val) => {
    if (val) error.value = '';
});

const handleConfirm = () => {
    const trimmed = reason.value.trim();
    if (!trimmed) {
        error.value = t('ai.interrupt.generic.rejectReasonRequired');
        return;
    }
    emit('confirm', trimmed);
    visible.value = false;
    reason.value = '';
};
</script>

<style scoped>
.reject-reason {
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.reject-reason__title {
    font-size: 13px;
    font-weight: 500;
}

.reject-reason__error {
    font-size: 12px;
    color: var(--el-color-danger);
}

.reject-reason__actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
}
</style>
