<template>
    <div class="procdef-policy-editor">
        <div class="policy-default">
            <span class="default-label">{{ $t('flow.policy.defaultSeverity') }}</span>
            <div class="default-select">
                <el-select v-model="policy.defaultSeverity" size="small" :teleported="false" :disabled="disabled">
                    <el-option :value="SEVERITY_REQUIRED" :label="$t(SEVERITY_LABEL_KEYS[SEVERITY_REQUIRED])" />
                    <el-option :value="SEVERITY_DISABLED" :label="$t(SEVERITY_LABEL_KEYS[SEVERITY_DISABLED])" />
                </el-select>
            </div>
            <span class="default-tip">{{ $t('flow.policy.defaultSeverityTip') }}</span>
        </div>

        <!-- 一屏 9 张场景卡 × 3 个 tab 是这里最大的负荷来源：先给一句「配了多少」，
             再把没配规则的场景折起来，管理员的视线才能落在真正在管的那几个场景上 -->
        <div class="policy-summary">
            <span class="summary-text">{{ $t('flow.policy.summary', { rules: ruleCount, scenes: configuredSceneCount }) }}</span>
            <el-button v-if="scenarios.length > 1" link type="primary" size="small" @click="toggleAll">
                {{ allExpanded ? $t('flow.policy.collapseAll') : $t('flow.policy.expandAll') }}
            </el-button>
        </div>

        <el-tabs v-model="activeTab" class="policy-tabs">
            <el-tab-pane name="checks" :label="$t('flow.policy.tabChecks')">
                <div v-for="scenario in scenarios" :key="scenario.bizType" class="scenario-card">
                    <div class="scenario-head">
                        <span class="scenario-name">{{ scenarioLabel(scenario.bizType) }}</span>
                        <span v-if="!hasAnyRule(scenario.bizType)" class="scenario-state">{{ $t('flow.policy.noRule') }}</span>
                        <!-- 兜底级别在这个场景落不了地时，引擎会在运行期收敛成该场景最严的可落地级别。
                             不在这里说出来，管理员会以为兜底已经管住了这台机器，实际等到的是整台机器被硬拒 -->
                        <span v-if="fallbackClamped(scenario)" class="scenario-state">
                            {{ $t('flow.policy.fallbackClamped', { severity: $t(SEVERITY_LABEL_KEYS[scenario.failClosedSeverity as Severity]) }) }}
                        </span>
                        <el-button
                            link
                            type="primary"
                            size="small"
                            :aria-expanded="String(isExpanded(scenario.bizType))"
                            @click="toggleScenario(scenario.bizType)"
                        >
                            {{ isExpanded(scenario.bizType) ? $t('flow.policy.collapseDetail') : $t('flow.policy.expandDetail') }}
                        </el-button>
                        <div class="scenario-templates">
                            <span class="templates-label">{{ $t('flow.policy.presetLabel') }}</span>
                            <el-tooltip v-for="preset in scenario.presets ?? []" :key="preset.key" :content="presetText(preset)" placement="top">
                                <el-button link type="primary" size="small" :disabled="disabled" @click="applyPreset(scenario, preset)">
                                    {{ $t(preset.titleKey) }}
                                </el-button>
                            </el-tooltip>
                        </div>
                    </div>
                    <PolicyCheckList v-show="isExpanded(scenario.bizType)" :policy="policy" :scenario="scenario" :disabled="disabled" />
                </div>
            </el-tab-pane>

            <el-tab-pane name="custom" :label="$t('flow.policy.tabCustom')">
                <div v-for="scenario in scenarios" :key="scenario.bizType" class="scenario-card">
                    <div class="scenario-head">
                        <span class="scenario-name">{{ scenarioLabel(scenario.bizType) }}</span>
                        <el-switch
                            :model-value="Boolean(customOf(policy, scenario.bizType))"
                            :aria-label="scenarioLabel(scenario.bizType)"
                            size="small"
                            :disabled="disabled"
                            @update:model-value="(value: boolean | string | number) => onToggleCustom(scenario, Boolean(value))"
                        />
                    </div>

                    <div v-if="customOf(policy, scenario.bizType)" class="custom-body">
                        <div class="custom-part">
                            <div class="part-head">
                                <span class="part-label">{{ $t('flow.policy.when') }}</span>
                                <div class="part-severity">
                                    <el-select v-model="customOf(policy, scenario.bizType)!.severity" size="small" :disabled="disabled">
                                        <el-option
                                            v-for="severity in severitiesOf(scenario)"
                                            :key="severity"
                                            :value="severity"
                                            :label="$t(SEVERITY_LABEL_KEYS[severity as Severity])"
                                        />
                                    </el-select>
                                </div>
                            </div>
                            <PolicyConditionNode :node="ensureWhen(scenario.bizType)" :scenario="scenario" :disabled="disabled" />
                        </div>

                        <div class="custom-part">
                            <div class="part-head">
                                <span class="part-label">{{ $t('flow.policy.unless') }}</span>
                                <el-button link type="primary" size="small" :disabled="disabled" @click="toggleUnless(scenario.bizType)">
                                    {{ unlessOf(scenario.bizType) ? $t('flow.policy.removeUnless') : $t('flow.policy.addUnless') }}
                                </el-button>
                            </div>
                            <PolicyConditionNode
                                v-if="unlessOf(scenario.bizType)"
                                :node="unlessOf(scenario.bizType)!"
                                :scenario="scenario"
                                :disabled="disabled"
                            />
                            <div v-else class="unless-empty">{{ $t('flow.policy.unlessTip') }}</div>
                        </div>
                    </div>
                </div>
            </el-tab-pane>
            <el-tab-pane name="simulate" :label="$t('flow.policy.tabSimulate')">
                <div v-for="scenario in simulatableScenarios" :key="scenario.bizType" class="scenario-card">
                    <div class="scenario-head">
                        <span class="scenario-name">{{ scenarioLabel(scenario.bizType) }}</span>
                    </div>

                    <template v-if="scenario.simulateFields?.length">
                        <div v-for="input in scenario.simulateFields" :key="input.key" class="simulate-field">
                            <div class="simulate-label">
                                <span v-if="input.required" class="label-required">*</span>
                                {{ $t(input.titleKey) }}
                            </div>
                            <!-- 资源引用类输入（editorKey 由场景声明）用资源树选择：库/机器 id 是内部主键，
                                 运维只知道资源叫什么名字，id 输入框等于不可用。其余控件形态取自下发的 editorKey / multiline，
                                 数值输入用 type=number 而不是 el-input-number：试算请求的 raw 是字符串表，
                                 数字组件会把值变成 number 而打不开接口 -->
                            <simulate-db-select
                                v-if="input.editorKey === 'db-select'"
                                v-model:id="rawOf(scenario.bizType)[input.key]"
                                @pick="onSimulateNamePicked(scenario.bizType, input, $event)"
                            />
                            <simulate-machine-select
                                v-else-if="input.editorKey === 'machine-select'"
                                v-model:id="rawOf(scenario.bizType)[input.key]"
                            />
                            <el-input
                                v-else
                                v-model="rawOf(scenario.bizType)[input.key]"
                                size="small"
                                :type="input.multiline ? 'textarea' : input.editorKey === 'number' ? 'number' : 'text'"
                                :rows="input.multiline ? 3 : undefined"
                                :placeholder="input.placeholderKey ? $t(input.placeholderKey) : undefined"
                            />
                        </div>
                        <el-button
                            type="primary"
                            size="small"
                            :loading="simulating === scenario.bizType"
                            :disabled="!simulateReady(scenario.simulateFields, rawOf(scenario.bizType))"
                            @click="onSimulate(scenario)"
                        >
                            {{ $t('flow.policy.runSimulate') }}
                        </el-button>

                        <div v-if="simulations[scenario.bizType]" class="simulate-result">
                            <div class="result-verdict">
                                <!-- 与列表徽标同用 light：dark 实心底配白字只有 2.78:1，结论级别是给人读的正文 -->
                                <el-tag size="small" :type="SEVERITY_TAG_TYPES[simulations[scenario.bizType]!.decision.severity]" effect="light">
                                    {{ $t(SEVERITY_LABEL_KEYS[simulations[scenario.bizType]!.decision.severity]) }}
                                </el-tag>
                                <span class="result-source">{{
                                    simulations[scenario.bizType]!.decision.matched ? $t('flow.policy.matchedRule') : $t('flow.policy.noMatchRule')
                                }}</span>
                            </div>
                            <ul v-if="simulations[scenario.bizType]!.decision.findings?.length" class="result-findings">
                                <li v-for="finding in simulations[scenario.bizType]!.decision.findings" :key="finding.source">
                                    {{ findingText(finding) }}
                                </li>
                            </ul>
                            <div v-else class="result-findings">{{ $t('flow.policy.noFinding') }}</div>
                            <div class="result-fields">
                                <span class="fields-label">{{ $t('flow.policy.resolvedFields') }}</span>
                                <code v-for="(value, key) in simulations[scenario.bizType]!.fields" :key="key" class="field-item">
                                    {{ key }} = {{ fmtValue(value) }}
                                </code>
                            </div>
                            <div v-if="simulations[scenario.bizType]!.decision.unknown?.length" class="result-unknown">
                                {{ $t('flow.policy.unknownFields', { fields: (simulations[scenario.bizType]!.decision.unknown ?? []).join(', ') }) }}
                            </div>
                        </div>
                    </template>
                </div>
                <div v-if="unsimulatableCount" class="simulate-unsupported">
                    {{ $t('flow.policy.simulateUnsupported', { count: unsimulatableCount }) }}
                </div>
            </el-tab-pane>
        </el-tabs>

        <div class="policy-preview">
            <div class="preview-title">{{ $t('flow.policy.preview') }}</div>
            <ul class="preview-list">
                <li v-for="(line, index) in summaryLines" :key="index">{{ line }}</li>
            </ul>
        </div>

        <div v-if="issues.length" class="policy-issues">
            <div v-for="(issue, index) in issues" :key="index" class="issue-item">{{ issueText(issue) }}</div>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import {
    createConditionNode,
    createGroupNode,
    customOf,
    describePolicy,
    disableCheck,
    dropCustomIfEmpty,
    enableCheck,
    ensureCustom,
    SEVERITY_DISABLED,
    SEVERITY_LABEL_KEYS,
    severitiesOf,
    SEVERITY_TAG_TYPES,
    SEVERITY_REQUIRED,
    simulateReady,
    validatePolicy,
    type Finding,
    labelIssueSource,
    type PolicyIssue,
    type PolicyPreset,
    type PolicyScenario,
    type PolicySchema,
    type PolicySimulateInput,
    type RuleNode,
    type Severity,
    type SimulateResult,
    type TriggerPolicy,
    PolicyCheckList,
    PolicyConditionNode,
} from '@/components/policy-builder';
import { procdefApi } from '../api';
import SimulateDbSelect from './SimulateDbSelect.vue';
import SimulateMachineSelect from './SimulateMachineSelect.vue';

/**
 * 流程定义触发策略编辑器：以「勾选内置检查项 + 调参 + 定级」为主，自定义条件树为辅。
 *
 * 场景卡片、检查项清单、字段字典全部来自后端 policy-schema，
 * 因此新增 Mongo/ES/机器等资源场景时本组件无需改动。
 * 直接读写宿主表单传入的策略草稿，组件内不保存副本，保证界面回显与提交内容是同一份对象。
 */
const props = withDefaults(
    defineProps<{
        policy: TriggerPolicy;
        schema: PolicySchema | null;
        disabled?: boolean;
    }>(),
    { disabled: false }
);

const { t } = useI18n();

const activeTab = ref<'checks' | 'custom' | 'simulate'>('checks');

// 试算按场景各自维护一份输入与结果：草稿未保存时也能带着当前策略去求值
const rawInputs = ref<Record<string, Record<string, string>>>({});
const simulations = ref<Record<string, SimulateResult>>({});
const simulating = ref('');

const rawOf = (bizType: string) => {
    if (!rawInputs.value[bizType]) rawInputs.value[bizType] = {};
    return rawInputs.value[bizType];
};

/** 资源选择控件选中后，把资源显示名回填进场景声明 nameKey 的输入项（如库名），免二次手抄 */
const onSimulateNamePicked = (bizType: string, input: PolicySimulateInput, name: string) => {
    if (!input.nameKey) return;
    rawOf(bizType)[input.nameKey] = name;
};

async function onSimulate(scenario: PolicyScenario) {
    simulating.value = scenario.bizType;
    try {
        simulations.value[scenario.bizType] = await procdefApi.policySimulate.request({
            bizType: scenario.bizType,
            policy: props.policy,
            raw: rawOf(scenario.bizType),
        });
    } finally {
        simulating.value = '';
    }
}

// 检查项命中回显其名称，引擎/自定义条件结论回显固定文案
const findingText = (finding: Finding) => `${t(finding.title)} → ${t(SEVERITY_LABEL_KEYS[finding.severity])}`;

const scenarios = computed(() => props.schema?.scenarios ?? []);

/**
 * 场景卡的折叠状态：默认只展开「已经配了规则」的场景。
 *
 * 不折叠时一屏要滚过 9 张卡 × 每张 4-9 行检查项，管理员实际在管的常常只有两三个场景；
 * 预设按钮仍留在折叠行里，未配置场景的起步路径没有被藏起来
 */
const expandedScenes = ref<Record<string, boolean>>({});

const hasCheckRules = (bizType: string) => (props.policy.checks ?? []).some((item) => item.bizType === bizType);

// 与引擎 HasRuleFor 同语义：checks 或自定义条件任一存在即算「已配置规则」，兜底级别只在一条都没配时生效。
// 只看 checks 会让配了自定义条件的场景仍显示「未配置任何规则/兜底按禁止执行生效」，
// 而实际未命中即放行——管理员的安全认知会被带反
const hasAnyRule = (bizType: string) => hasCheckRules(bizType) || !!customOf(props.policy, bizType);

/** 该场景是否会触发「兜底级别收敛」：配了需要落地能力的兜底，但这个场景没有对应的处置通道 */
const fallbackClamped = (scenario: PolicyScenario) => {
    // 已配置规则时兜底根本不参与判定，收敛提示再显示就是误导
    if (hasAnyRule(scenario.bizType)) {
        return false;
    }
    const strictest = scenario.failClosedSeverity;
    if (strictest === undefined || props.policy.defaultSeverity === SEVERITY_DISABLED) return false;
    return !(scenario.severities ?? []).includes(props.policy.defaultSeverity) && strictest !== props.policy.defaultSeverity;
};
const isExpanded = (bizType: string) => expandedScenes.value[bizType] ?? hasAnyRule(bizType);
const toggleScenario = (bizType: string) => {
    expandedScenes.value[bizType] = !isExpanded(bizType);
};
const allExpanded = computed(() => scenarios.value.every((item) => isExpanded(item.bizType)));
function toggleAll() {
    const next = !allExpanded.value;
    for (const item of scenarios.value) expandedScenes.value[item.bizType] = next;
}

const ruleCount = computed(() => (props.policy.checks?.length ?? 0) + (props.policy.customs?.length ?? 0));
const configuredSceneCount = computed(
    () => new Set([...(props.policy.checks ?? []).map((item) => item.bizType), ...(props.policy.customs ?? []).map((item) => item.bizType)]).size
);
const simulatableScenarios = computed(() => scenarios.value.filter((item) => item.simulateFields?.length));
const unsimulatableCount = computed(() => scenarios.value.length - simulatableScenarios.value.length);

const scenarioLabel = (bizType: string) => {
    const key = `flow.bizTypeName.${bizType}`;
    const label = t(key);
    return label === key ? bizType : label;
};

// 预置规则包由后端场景注册表随 schema 下发：新增资源场景自带自己的模板，本组件不持有任何 check key
function applyPreset(scenario: PolicyScenario, preset: PolicyPreset) {
    // 只重置本场景：按钮长在场景卡片上，跨场景改动会让其它卡片里的手工配置被静默清掉
    for (const check of scenario.checks) {
        if (preset.checkKeys.includes(check.key)) enableCheck(props.policy, scenario.bizType, check);
        else disableCheck(props.policy, scenario.bizType, check.key);
    }
}

function scenarioOf(bizType: string) {
    return scenarios.value.find((item) => item.bizType === bizType) ?? null;
}

function onToggleCustom(scenario: PolicyScenario, enabled: boolean) {
    if (enabled) ensureCustom(props.policy, scenario.bizType, scenario);
    else props.policy.customs = (props.policy.customs ?? []).filter((item) => item.bizType !== scenario.bizType);
}

const fmtValue = (value: unknown) => (typeof value === 'object' && value !== null ? JSON.stringify(value) : String(value));

// 预置按钮只显示短名，差异内容放 tooltip：三个长文案按钮并排会把卡片头挤到溢出
const presetText = (preset: PolicyPreset) => preset.checkKeys.map((key) => checkTitle(key)).join('、');

function checkTitle(checkKey: string) {
    for (const scenario of scenarios.value) {
        const found = scenario.checks.find((item) => item.key === checkKey);
        if (found) return t(found.titleKey);
    }
    return checkKey;
}

function ensureWhen(bizType: string): RuleNode {
    const custom = ensureCustom(props.policy, bizType, scenarioOf(bizType));
    if (!custom.when) custom.when = createGroupNode('all');
    return custom.when;
}

function unlessOf(bizType: string): RuleNode | undefined {
    return customOf(props.policy, bizType)?.unless ?? undefined;
}

function toggleUnless(bizType: string) {
    const custom = customOf(props.policy, bizType);
    if (!custom) return;
    if (custom.unless) {
        custom.unless = undefined;
        dropCustomIfEmpty(props.policy, bizType);
    } else {
        // 建组即给一行（与 FlowConditionEditor.startEditing 同约定）：空分组会被保存校验判非法，
        // 点「添加豁免条件」立刻飘红等于把设计缺陷甩给用户
        const group = createGroupNode('all');
        group.items = [createConditionNode(scenarioOf(bizType)?.fields[0])];
        custom.unless = group;
    }
}

const summaryLines = computed(() => describePolicy(props.policy, props.schema, t));

const issues = computed<PolicyIssue[]>(() => validatePolicy(props.policy, props.schema));

// 定位串与原因一起给出：一份策略里常有十余条规则，只说「参数不得小于 1」无法回答是哪条规则出错
const issueText = (issue: PolicyIssue) => `${labelIssueSource(issue.source, props.schema, t)} ${t(issue.reasonKey, issue.params ?? {})}`;

/**
 * 供宿主在提交前调用：有问题时返回第一条的完整提示文本，无问题返回 null。
 *
 * 这些判定此前已经实时显示在编辑器的问题面板里，却在点「保存」时不拦一手：
 * 用户要发一次请求、被后端拒了才知道没填对，白等一次往返
 */
defineExpose<{ validate: () => string | null }>({
    validate: () => (issues.value.length > 0 ? issueText(issues.value[0]) : null),
});
</script>

<style lang="scss" scoped>
.procdef-policy-editor {
    display: flex;
    flex-direction: column;
    gap: 8px;

    .policy-default {
        display: flex;
        align-items: center;
        gap: 8px;
        flex-wrap: wrap;
    }

    .default-label {
        font-size: 13px;
    }

    .default-select {
        width: 140px;
    }

    .default-tip {
        font-size: 12px;
        color: var(--el-text-color-secondary);
    }

    .scenario-card {
        border: 1px solid var(--el-border-color-lighter);
        border-radius: var(--el-border-radius-base);
        padding: 10px;
        margin-bottom: 10px;
    }

    // 间距挂在「有明细时」而不是挂在头部：折叠态的卡片不该白留一行空隙
    .scenario-head {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 6px 10px;
    }

    .scenario-name {
        font-size: 13px;
        font-weight: 600;
        white-space: nowrap;
    }

    // 明细的间距挂在这里，而不是挂在头部：折叠态卡片不该白留一行空隙
    :deep(.policy-check-list) {
        margin-top: 8px;
    }

    .scenario-templates {
        margin-left: auto;
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        justify-content: flex-end;
        gap: 4px;
        min-width: 0;
    }

    .templates-label {
        font-size: 12px;
        color: var(--el-text-color-secondary);
    }

    .custom-body {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .part-head {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 6px;
    }

    .part-label {
        font-size: 12px;
        color: var(--el-text-color-secondary);
    }

    .part-severity {
        width: 120px;
        margin-left: auto;
    }

    .unless-empty {
        font-size: 12px;
        color: var(--el-text-color-placeholder);
    }

    .simulate-field {
        margin-bottom: 8px;
    }

    .simulate-label {
        font-size: 12px;
        color: var(--el-text-color-secondary);
        margin-bottom: 4px;
    }

    // 折叠行上的「未配置规则」标记与试算不可用说明：都用次要色，不与场景名抢层级
    .scenario-state,
    .simulate-unsupported {
        font-size: 12px;
        color: var(--el-text-color-secondary);
    }

    .policy-summary {
        display: flex;
        align-items: center;
        gap: 10px;
        font-size: 12px;
        color: var(--el-text-color-secondary);
    }

    .simulate-result {
        margin-top: 10px;
        padding-top: 8px;
        border-top: 1px dashed var(--el-border-color-lighter);
        font-size: 12px;
    }

    .result-verdict {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .result-source {
        color: var(--el-text-color-secondary);
    }

    .result-findings {
        margin: 6px 0;
        padding-left: 18px;
    }

    .result-fields {
        display: flex;
        flex-wrap: wrap;
        gap: 4px 10px;
        margin-top: 6px;
        color: var(--el-text-color-secondary);
    }

    .fields-label {
        flex-basis: 100%;
    }

    .field-item {
        padding: 1px 6px;
        border-radius: 4px;
        background: var(--el-fill-color-light);
        font-size: 12px;
        overflow-wrap: anywhere;
    }

    .label-required {
        color: var(--el-color-danger);
        margin-right: 2px;
    }

    .result-unknown {
        color: var(--el-color-warning);
    }

    .policy-preview {
        border-top: 1px dashed var(--el-border-color-lighter);
        padding-top: 8px;
    }

    .preview-title {
        font-size: 12px;
        color: var(--el-text-color-secondary);
        margin-bottom: 4px;
    }

    .preview-list {
        margin: 0;
        padding-left: 18px;
        font-size: 12px;
        line-height: 20px;
    }

    .policy-issues {
        color: var(--el-color-danger);
        font-size: 12px;
    }
}
</style>
