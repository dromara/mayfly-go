<template>
    <div class="options-param-input">
        <!-- 问题描述 -->
        <div v-if="description" class="options-param-input__question">
            {{ description }}
        </div>

        <!-- 可选项列表（Questionnaire choices） -->
        <div v-if="options.length > 0" class="options-param-input__choices">
            <div
                v-for="(opt, idx) in options"
                :key="idx"
                class="options-param-input__choice"
                :class="{ 'is-selected': selectedValue === opt.value }"
                @click="selectOption(opt)"
            >
                <span class="options-param-input__choice-label">{{ opt.label }}</span>
            </div>
        </div>

        <!-- 自由输入（Questionnaire input） -->
        <div class="options-param-input__input-row">
            <input
                v-model="freeText"
                class="options-param-input__input"
                :placeholder="t('ai.interrupt.paramCompletion.freeInputPlaceholder')"
                :disabled="readonly"
                @input="onFreeTextInput"
            />
        </div>
    </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

interface OptionItem {
    label: string;
    value: string;
}

interface Props {
    options: OptionItem[];
    description?: string;
    readonly?: boolean;
    isConfirmed?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
    readonly: false,
    isConfirmed: false,
    description: '',
});

const emit = defineEmits<{
    (e: 'change', values: Record<string, unknown>): void;
}>();

const { t } = useI18n();

const selectedValue = ref<string>('');
const freeText = ref('');

const selectOption = (opt: OptionItem) => {
    if (props.readonly) return;
    selectedValue.value = opt.value;
    freeText.value = '';
    emitChange();
};

const onFreeTextInput = () => {
    if (props.readonly) return;
    if (freeText.value) {
        selectedValue.value = '';
    }
    emitChange();
};

const emitChange = () => {
    let payload: Record<string, unknown> = {};
    if (selectedValue.value) {
        try {
            payload = JSON.parse(selectedValue.value);
        } catch {
            payload = { value: selectedValue.value };
        }
    } else if (freeText.value) {
        try {
            payload = JSON.parse(freeText.value);
        } catch {
            payload = { value: freeText.value };
        }
    }
    emit('change', payload);
};

const isValid = () => {
    return !!(selectedValue.value || freeText.value.trim());
};

const getValues = () => {
    if (selectedValue.value) {
        try {
            return JSON.parse(selectedValue.value);
        } catch {
            return { value: selectedValue.value };
        }
    }
    if (freeText.value) {
        try {
            return JSON.parse(freeText.value);
        } catch {
            return { value: freeText.value };
        }
    }
    return {};
};

defineExpose({
    isValid,
    getValues,
});
</script>

<style scoped>
.options-param-input {
    padding: 4px;
    display: flex;
    flex-direction: column;
    gap: 6px;
}

.options-param-input__question {
    font-size: 12px;
    color: var(--el-text-color-regular);
    line-height: 1.4;
}

.options-param-input__choices {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
}

.options-param-input__choice {
    display: inline-flex;
    align-items: center;
    padding: 2px 8px;
    border: 1px solid var(--el-border-color);
    border-radius: 4px;
    font-size: 11px;
    cursor: pointer;
    transition: all 0.2s;
    background: var(--el-bg-color);
    color: var(--el-text-color-regular);
    line-height: 1.6;
}

.options-param-input__choice:hover {
    border-color: var(--el-color-primary);
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
}

.options-param-input__choice.is-selected {
    border-color: var(--el-color-primary);
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-8);
    font-weight: 500;
}

.options-param-input__choice-label {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 200px;
}

.options-param-input__input-row {
    display: flex;
    gap: 4px;
}

.options-param-input__input {
    flex: 1;
    height: 24px;
    padding: 0 6px;
    border: 1px solid var(--el-border-color);
    border-radius: 4px;
    font-size: 11px;
    outline: none;
    background: var(--el-bg-color);
    color: var(--el-text-color-regular);
    transition: border-color 0.2s;
}

.options-param-input__input:focus {
    border-color: var(--el-color-primary);
}

.options-param-input__input:disabled {
    background: var(--el-fill-color-light);
    cursor: not-allowed;
}
</style>
