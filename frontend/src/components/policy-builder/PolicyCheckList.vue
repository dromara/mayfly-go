<template>
    <div class="policy-check-list">
        <div v-if="!scenario?.checks?.length" class="check-empty">{{ $t('flow.policy.noCheck') }}</div>

        <div v-for="check in scenario?.checks ?? []" :key="check.key" class="check-item" :class="{ 'is-enabled': isEnabled(check.key) }">
            <div class="check-head">
                <el-switch
                    :model-value="isEnabled(check.key)"
                    :aria-label="$t(check.titleKey)"
                    size="small"
                    :disabled="disabled"
                    @update:model-value="(value: boolean | string | number) => onToggle(check, Boolean(value))"
                />
                <div class="check-title">
                    <span class="check-name">{{ $t(check.titleKey) }}</span>
                    <span v-if="check.descriptionKey" class="check-desc">{{ $t(check.descriptionKey) }}</span>
                </div>
                <div class="check-severity">
                    <el-select
                        :model-value="severityOf(check)"
                        size="small"
                        :disabled="disabled || !isEnabled(check.key)"
                        @update:model-value="(value: unknown) => onSeverity(check, value as Severity)"
                    >
                        <el-option
                            v-for="severity in enforceSeverities"
                            :key="severity"
                            :label="$t(severityLabelKeys[severity as Severity])"
                            :value="severity"
                        />
                    </el-select>
                </div>
            </div>

            <div v-if="isEnabled(check.key) && check.params?.length" class="check-params">
                <div v-for="param in check.params" :key="param.key" class="param-item">
                    <span class="param-label">{{ $t(param.titleKey) }}</span>
                    <PolicyValueEditor
                        :field="editorOf(param).field"
                        :operator="editorOf(param).operator"
                        :disabled="disabled"
                        :model-value="paramValue(check, param)"
                        @update:model-value="(value: unknown) => onParam(check, param, value)"
                    />
                </div>
            </div>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue';

import PolicyValueEditor from './PolicyValueEditor.vue';
import {
    disableCheck,
    enableCheck,
    paramEditor,
    SEVERITY_LABEL_KEYS,
    severitiesOf,
    type PolicyCheck,
    type PolicyCheckParam,
    type PolicyScenario,
    type Severity,
    type TriggerPolicy,
} from './policyModel';

/**
 * 内置检查项清单：勾选 + 参数 + 处置级别。
 *
 * 清单内容完全由后端 policy-schema 下发，新增检查项（含新资源场景的检查项）本组件零改动；
 * 勾选状态以「配置是否存在于策略里」为唯一真源，不另存开关字段，避免出现两处状态分叉。
 */
const props = withDefaults(
    defineProps<{
        policy: TriggerPolicy;
        scenario: PolicyScenario | null;
        disabled?: boolean;
    }>(),
    { disabled: false }
);

const enforceSeverities = computed(() => severitiesOf(props.scenario));
const severityLabelKeys = SEVERITY_LABEL_KEYS;

// 「勾选态」就是「配置是否存在」，直接从 configOf 派生：两处各写一遍匹配条件会让它们有朝一日分叉
const configOf = (checkKey: string) => props.policy.checks?.find((item) => item.bizType === props.scenario?.bizType && item.key === checkKey) ?? null;

const isEnabled = (checkKey: string) => configOf(checkKey) !== null;

const severityOf = (check: PolicyCheck): Severity => configOf(check.key)?.severity ?? check.default;

const editorOf = (param: PolicyCheckParam) => paramEditor(param);

const paramValue = (check: PolicyCheck, param: PolicyCheckParam) => {
    const config = configOf(check.key);
    const value = config?.params?.[param.key];
    return value === undefined ? param.default : value;
};

const onToggle = (check: PolicyCheck, enabled: boolean) => {
    if (!props.scenario) return;
    if (enabled) enableCheck(props.policy, props.scenario.bizType, check);
    else disableCheck(props.policy, props.scenario.bizType, check.key);
};

const onSeverity = (check: PolicyCheck, severity: Severity) => {
    const config = configOf(check.key);
    if (config) config.severity = severity;
    else if (props.scenario) {
        enableCheck(props.policy, props.scenario.bizType, check);
        const created = configOf(check.key);
        if (created) created.severity = severity;
    }
};

const onParam = (check: PolicyCheck, param: PolicyCheckParam, value: unknown) => {
    const config = configOf(check.key);
    if (!config) return;
    config.params = { ...(config.params ?? {}), [param.key]: value };
};
</script>

<style lang="scss" scoped>
.policy-check-list {
    display: flex;
    flex-direction: column;
    gap: 8px;

    .check-empty {
        font-size: 12px;
        color: var(--el-text-color-secondary);
        padding: 8px 0;
    }

    .check-item {
        border: 1px solid var(--el-border-color-lighter);
        border-radius: var(--el-border-radius-base);
        padding: 8px 10px;
        transition: border-color 0.2s ease;

        &.is-enabled {
            border-color: var(--el-color-primary-light-7);
            background: var(--el-fill-color-lighter);
        }
    }

    .check-head {
        display: flex;
        align-items: flex-start;
        gap: 10px;
        min-width: 0;
    }

    .check-title {
        flex: 1;
        min-width: 0;
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .check-name {
        font-size: 13px;
        line-height: 24px;
        // 名称与括号内的取值示例同宽换行时，缩进对齐到首行文字而不是顶到左边
        text-wrap: pretty;
    }

    .check-desc {
        font-size: 12px;
        color: var(--el-text-color-secondary);
    }

    .check-severity {
        width: 104px;
        flex-shrink: 0;
    }

    .check-params {
        display: flex;
        flex-wrap: wrap;
        gap: 12px;
        margin-top: 8px;
        padding-left: 42px;
    }

    .param-item {
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .param-label {
        font-size: 12px;
        color: var(--el-text-color-secondary);
    }
}
</style>
