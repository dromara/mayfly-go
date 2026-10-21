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

/**
 * 选项文本 → 决策载荷的解析单点。
 *
 * 此前 emitChange 与 getValues 各写一份完全相同的 JSON.parse + 兜底：两处一旦分叉，
 * 「界面显示的选中项」与「真正提交给后端的参数」就会不一致，而且没有任何报错。
 *
 * 本组件不对外提供 modelValue 回显入口：已决策的中断在轨迹里只渲染决议徽章（ToolCallPart
 * 仅在 !isInterruptDecided 时下发 pendingInterrupt），卡片不会被重放，所以上层没有可绑的数据源。
 */
function parsePayload(raw: string): Record<string, unknown> {
    if (!raw) return {};
    try {
        const parsed = JSON.parse(raw);
        return parsed && typeof parsed === 'object' ? (parsed as Record<string, unknown>) : { value: parsed };
    } catch {
        return { value: raw };
    }
}

/** 当前原始文本：选中项优先，其次自由文本（两者互斥，由下方两个入口保证） */
const currentRaw = () => selectedValue.value || freeText.value.trim();

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
    // 载荷解析与 getValues 同源，保证「看到的选中项」与「提交的参数」不会分叉
    emit('change', parsePayload(currentRaw()));
};

const isValid = () => {
    return !!(selectedValue.value || freeText.value.trim());
};

const getValues = () => parsePayload(currentRaw());

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
