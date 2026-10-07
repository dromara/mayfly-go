<template>
    <el-tabs v-model="activeTabName">
        <el-tab-pane :label="$t('common.basic')" :name="basicTabName">
            <div class="edge-condition">
                <div class="condition-label">{{ $t('flow.condition.edgeLabel') }}</div>
                <div class="condition-tip">{{ $t('flow.condition.edgeTip') }}</div>
                <FlowConditionEditor v-model="condition" :scenario="scenario" :disabled="disabled" />
                <div v-if="issues.length" class="condition-issues">
                    <div v-for="(issue, index) in issues" :key="index">{{ issueText(issue) }}</div>
                </div>
            </div>
        </el-tab-pane>
    </el-tabs>
</template>

<script lang="ts" setup>
import { isTrue } from '@/common/assert';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { conditionScenarioOf, usePolicySchema, validateConditionTree, type PolicyIssue, type RuleNode } from '@/components/policy-builder';
import { FLOW_INSTANCE_BIZ_TYPE } from '@/views/flow/enums';
import { procdefApi } from '@/views/flow/api';
import FlowConditionEditor from '@/views/flow/components/FlowConditionEditor.vue';

/**
 * 连线属性：跳转条件。
 *
 * 条件写在这里而不是流程定义上，是因为「走哪条线」本质是图结构的一部分：
 * 它跟连线一起增删、一起在流程图上可见，放到流程定义级别就会与「这条线连的谁到谁」脱节。
 */
defineProps<{ disabled?: boolean }>();

const basicTabName = 'basic';
const activeTabName = ref(basicTabName);

const form = defineModel<Record<string, unknown>>('modelValue', { required: true });

const { policySchema, load } = usePolicySchema(() => procdefApi.policySchema.request());
const { t } = useI18n();

onMounted(load);

const scenario = computed(() => conditionScenarioOf(policySchema.value, FLOW_INSTANCE_BIZ_TYPE));

const condition = computed<RuleNode | null>({
    get: () => (form.value.condition as RuleNode) ?? null,
    set: (value) => {
        form.value.condition = value;
    },
});

const issues = computed<PolicyIssue[]>(() => validateConditionTree(condition.value, scenario.value));
// 与策略编辑器同一口径：定位串 + 原因，多个条件行时用户能知道是哪一行
const issueText = (issue: PolicyIssue) => `${issue.source} ${t(issue.reasonKey, issue.params ?? {})}`;

const confirm = () => {
    // 允许「不配条件」（默认流转），但不允许一份填了一半的条件落库：
    // 那种条件在运行期只会判出不成立，流程停在原地，而看起来像是配置成功了
    isTrue(issues.value.length === 0, 'flow.conditionIncomplete');
};

defineExpose({ confirm });
</script>

<style lang="scss" scoped>
.edge-condition {
    display: flex;
    flex-direction: column;
    gap: 8px;

    .condition-label {
        font-size: 13px;
        color: var(--el-text-color-regular);
    }

    .condition-tip {
        font-size: 12px;
        color: var(--el-text-color-secondary);
        line-height: 18px;
    }

    .condition-issues {
        font-size: 12px;
        color: var(--el-color-danger);
    }
}
</style>
