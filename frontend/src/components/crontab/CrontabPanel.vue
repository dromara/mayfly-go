<template>
    <div class="flex gap-4 max-lg:flex-col">
        <section class="min-w-0 flex-1">
            <div class="mb-2 flex flex-wrap items-center gap-1.5">
                <span class="text-xs text-muted-foreground">{{ $t('components.crontab.presetsLabel') }}</span>
                <el-button v-for="preset in presets" :key="preset.key" size="small" plain @click="emits('applyPreset', preset)">
                    {{ $t(preset.labelKey) }}
                </el-button>
            </div>

            <el-tabs v-model="activeField" type="border-card" class="cron-tabs">
                <el-tab-pane v-for="field in fields" :key="field.key" :label="$t(field.labelKey)" :name="field.key">
                    <CrontabFieldPanel
                        :field="field"
                        :rule="spec[field.key]"
                        :disabled="!editable"
                        :hint="field.allowNone ? $t('components.crontab.dayWeekHint') : ''"
                        @update:rule="(rule) => emits('changeRule', field.key, rule)"
                    />
                </el-tab-pane>
            </el-tabs>
        </section>

        <aside class="flex max-h-full w-[230px] shrink-0 flex-col gap-3 overflow-y-auto max-lg:w-full">
            <div v-if="editable" class="cron-cells">
                <button
                    v-for="(field, index) in fields"
                    :key="field.key"
                    type="button"
                    class="cron-cell"
                    :class="{ 'is-active': activeField === field.key }"
                    @click="activeField = field.key"
                >
                    <span class="cron-cell-label">{{ $t(field.labelKey) }}</span>
                    <span class="cron-cell-value">{{ segments[index] }}</span>
                </button>
            </div>
            <p class="cron-expression">{{ displayExpression }}</p>

            <template v-if="error">
                <el-alert type="warning" show-icon :closable="false" :title="$t('components.crontab.parseErrorTitle')" :description="errorText" />
                <p class="text-xs text-muted-foreground">{{ $t('components.crontab.parseErrorHint') }}</p>
            </template>
            <el-alert
                v-else-if="!editable"
                type="info"
                show-icon
                :closable="false"
                :title="$t('components.crontab.descriptorTitle')"
                :description="$t('components.crontab.descriptorHint')"
            />

            <div v-if="editable" class="cron-preview">
                <p class="cron-preview-title">{{ $t('components.crontab.previewTitle', { count: PREVIEW_COUNT }) }}</p>
                <ul class="cron-preview-list">
                    <li v-for="time in previewTimes" :key="time">{{ time }}</li>
                    <li v-if="!previewTimes.length" class="text-muted-foreground">{{ $t('components.crontab.previewNone', { years: PREVIEW_YEARS }) }}</li>
                </ul>
                <p v-if="previewTimes.length && preview.exhausted" class="text-xs text-muted-foreground">
                    {{ $t('components.crontab.previewFew', { years: PREVIEW_YEARS, count: previewTimes.length }) }}
                </p>
            </div>
        </aside>
    </div>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import CrontabFieldPanel from './CrontabFieldPanel.vue';
import {
    CRON_FIELDS,
    CRON_PANEL_FIELDS,
    CRON_PRESETS,
    formatRule,
    PREVIEW_COUNT,
    PREVIEW_YEARS,
    type CronError,
    type CronFieldKey,
    type CronPreset,
    type CronPreview,
    type CronRule,
    type CronSpec,
} from './cronSpec';
import { formatDate } from '@/common/utils/format';

const props = defineProps<{
    /** 面板草稿的规则集（唯一真源） */
    spec: CronSpec;
    /** 由规则集派生的完整表达式 */
    expression: string;
    /** 打开面板时的原始表达式，描述符只原样展示 */
    sourceExpression: string;
    /** 载入原表达式时的解析错误 */
    error: CronError | null;
    /** 描述符表达式只能原样保留，面板只读 */
    descriptorOnly: boolean;
    /** 运行时间预览 */
    preview: CronPreview;
}>();

const emits = defineEmits<{
    /** 某字段的规则变更 */
    changeRule: [key: CronFieldKey, rule: CronRule];
    /** 应用常用预设 */
    applyPreset: [preset: CronPreset];
}>();

const { t } = useI18n();

const fields = CRON_PANEL_FIELDS;
const presets = CRON_PRESETS;

const activeField = ref<CronFieldKey>('second');

const editable = computed(() => !props.descriptorOnly);

const segments = computed(() => fields.map((field) => formatRule(props.spec[field.key])));

// 描述符不由规则集表达，展示原表达式而不是面板默认值
const displayExpression = computed(() => (props.descriptorOnly ? props.sourceExpression : props.expression));

const previewTimes = computed(() => props.preview.times.map((time) => formatDate(time)));

/** 错误文案里的字段名需按 i18n key 二次翻译后插值 */
const errorText = computed(() => {
    const error = props.error;
    if (!error) return '';
    const params = { ...error.params, field: error.field ? t(CRON_FIELDS[error.field].labelKey) : '' };
    return t(error.key, params);
});
</script>

<style scoped>
/* 页签内容区定高：不同规则类型的体量差异很大（指定要摆 60 个取值），不固定会让 dialog 高度反复跳动；
   280px 为最高的那档（秒/分 的 5 行取值）刚好不需滚动，再高则靠 overflow-y 兜底 */
.cron-tabs :deep(.el-tabs__content) {
    height: 280px;
    overflow-y: auto;
}

.cron-cells {
    display: grid;
    grid-template-columns: repeat(6, minmax(0, 1fr));
    gap: 4px;
}

.cron-cell {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
    padding: 4px 2px;
    border: 1px solid var(--el-border-color);
    border-radius: 4px;
    background: var(--el-fill-color-blank);
    cursor: pointer;
    transition:
        border-color 0.15s,
        background-color 0.15s;
}

.cron-cell:hover {
    border-color: var(--el-color-primary-light-5);
}

.cron-cell.is-active {
    border-color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
}

.cron-cell-label {
    font-size: 11px;
    color: var(--el-text-color-secondary);
}

.cron-cell-value {
    font-family: var(--el-font-family-mono, monospace);
    font-size: 12px;
    word-break: break-all;
}

.cron-expression {
    padding: 6px 8px;
    border-radius: 4px;
    background: var(--el-fill-color-light);
    font-family: var(--el-font-family-mono, monospace);
    font-size: 12px;
    text-align: center;
    word-break: break-all;
}

.cron-preview-title {
    margin-bottom: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
}

.cron-preview-list {
    display: flex;
    flex-direction: column;
    gap: 2px;
    font-family: var(--el-font-family-mono, monospace);
    font-size: 12px;
    line-height: 20px;
}
</style>
