<template>
    <div class="clipboard-dialog">
        <el-dialog
            v-model="dialogVisible"
            :title="$t('components.terminal-rdp.clipboardTitle')"
            :before-close="onclose"
            :close-on-click-modal="false"
            :close-on-press-escape="false"
            width="600"
        >
            <el-input v-model="state.modelValue" type="textarea" :rows="20" />

            <template #footer>
                <el-button type="primary" @click="onsubmit">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup lang="ts">
import { reactive, toRefs, watch } from 'vue';
import { Msg } from '@/hooks/useI18n';

const props = defineProps<{
    visible?: boolean;
}>();

const emits = defineEmits(['submit', 'close', 'update:visible']);

const state = reactive({
    dialogVisible: false,
    modelValue: '',
});

const { dialogVisible } = toRefs(state);

watch(
    () => props.visible,
    (val) => {
        state.dialogVisible = val ?? false;
    }
);

const onclose = () => {
    emits('update:visible', false);
    emits('close');
};

const onsubmit = () => {
    state.dialogVisible = false;
    if (state.modelValue) {
        Msg.success('components.terminal-rdp.clipboardSendSuccess');
        emits('submit', state.modelValue);
    } else {
        Msg.warning('components.terminal-rdp.clipboardInputRequired');
    }
};

const setValue = (val: string) => {
    state.modelValue = val;
};

defineExpose({ setValue });
</script>

<style lang="scss" scoped>
.clipboard-dialog {
}
</style>
