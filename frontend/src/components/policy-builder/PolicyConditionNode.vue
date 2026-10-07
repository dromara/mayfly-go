<template>
    <div class="policy-condition-tree" :class="`depth-${depth}`">
        <template v-if="node.kind === 'group'">
            <div class="group-head">
                <span class="group-label">{{ $t(`flow.policy.logic.${node.logic ?? 'all'}`) }}</span>
                <div class="logic-select">
                    <el-select v-model="node.logic" size="small" :disabled="disabled">
                        <el-option v-for="logic in LOGICS" :key="logic" :value="logic" :label="$t(`flow.policy.logic.${logic}`)" />
                    </el-select>
                </div>
                <!-- 分组必须可删：只给添加入口不给删除入口，用户误点一次「添加条件组」就被空分组卡死（后端判非法） -->
                <el-button v-if="!disabled && depth > 0" link type="danger" size="small" class="group-remove" @click="emit('remove')">
                    {{ $t('flow.policy.removeGroup') }}
                </el-button>
            </div>

            <div class="group-body">
                <div v-if="!node.items?.length" class="group-empty">{{ $t('flow.policy.noCondition') }}</div>
                <PolicyConditionNode
                    v-for="(item, index) in node.items ?? []"
                    :key="index"
                    :node="item"
                    :scenario="scenario"
                    :depth="depth + 1"
                    :disabled="disabled"
                    @remove="removeItem(index)"
                />
            </div>

            <div v-if="!disabled" class="group-actions">
                <el-button link type="primary" size="small" icon="plus" @click="addItem('condition')">{{ $t('flow.policy.addCondition') }}</el-button>
                <el-button link type="primary" size="small" icon="share" @click="addItem('group')">{{ $t('flow.policy.addGroup') }}</el-button>
                <!-- 该场景没有可引用的条件组时不显示入口：点了只会得到一个空的引用节点 -->
                <el-button v-if="availableSegments.length" link type="primary" size="small" icon="collection-tag" @click="addItem('segment')">
                    {{ $t('flow.policy.addSegment') }}
                </el-button>
            </div>
        </template>

        <div v-else-if="node.kind === 'segment'" class="condition-row">
            <div class="cell segment-cell">
                <el-select
                    :model-value="node.ref"
                    size="small"
                    filterable
                    :disabled="disabled"
                    :placeholder="$t('flow.policy.chooseSegment')"
                    @update:model-value="(ref: string) => (node.ref = ref)"
                >
                    <el-option v-for="segment in availableSegments" :key="segment.ref" :value="segment.ref" :label="segment.name">
                        <span>{{ segment.name }}</span>
                        <span v-if="segment.remark" class="segment-remark">{{ segment.remark }}</span>
                    </el-option>
                </el-select>
            </div>
            <el-button
                v-if="!disabled"
                link
                type="danger"
                size="small"
                icon="delete"
                :title="$t('flow.policy.removeCondition')"
                :aria-label="$t('flow.policy.removeCondition')"
                @click="emit('remove')"
            />
        </div>

        <div v-else class="condition-row">
            <div class="cell field-cell">
                <el-select
                    :model-value="node.field"
                    size="small"
                    filterable
                    :disabled="disabled"
                    :placeholder="$t('flow.policy.chooseField')"
                    @update:model-value="onFieldChange"
                >
                    <el-option-group v-for="group in fieldGroups" :key="group.key" :label="$t(group.label)">
                        <el-option v-for="field in group.fields" :key="field.key" :value="field.key" :label="$t(field.titleKey)" />
                    </el-option-group>
                </el-select>
            </div>

            <div class="cell op-cell">
                <el-select
                    :model-value="node.op"
                    size="small"
                    :disabled="disabled || !field"
                    :placeholder="$t('flow.policy.chooseOperator')"
                    @update:model-value="onOperatorChange"
                >
                    <el-option v-for="operator in field?.ops ?? []" :key="operator.name" :value="operator.name" :label="$t(operator.labelKey)" />
                </el-select>
            </div>

            <div v-if="operator?.valueKind !== 'none'" class="cell value-cell">
                <PolicyValueEditor
                    v-if="field && operator"
                    :field="field"
                    :operator="operator"
                    :model-value="node.value"
                    :disabled="disabled"
                    @update:model-value="(value) => (node.value = value)"
                />
            </div>

            <el-button
                v-if="!disabled"
                link
                type="danger"
                size="small"
                icon="delete"
                :title="$t('flow.policy.removeCondition')"
                :aria-label="$t('flow.policy.removeCondition')"
                @click="emit('remove')"
            />
        </div>
    </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue';

import PolicyValueEditor from './PolicyValueEditor.vue';
import {
    createConditionNode,
    createGroupNode,
    createSegmentNode,
    emptyValueFor,
    fieldOf,
    operatorOf,
    segmentsOf,
    type GroupLogic,
    type PolicyField,
    type PolicyScenario,
    type RuleNode,
} from './policyModel';

/**
 * 条件树编辑器（递归）：分组按 全部/任一/全不 组合，条件行为 字段 + 操作符 + 值。
 *
 * 直接读写传入的策略草稿（与数据网格过滤构建器同一套草稿模式），
 * 组件内不保存第二份条件状态，因此不存在「界面与提交内容不一致」的分叉点。
 * 字段候选与操作符集合均来自后端 policy-schema，新增业务场景本组件零改动。
 */
defineOptions({ name: 'PolicyConditionNode' });

const props = withDefaults(
    defineProps<{
        node: RuleNode;
        scenario: PolicyScenario | null;
        depth?: number;
        disabled?: boolean;
    }>(),
    { depth: 0, disabled: false }
);

const emit = defineEmits<{ remove: [] }>();

const LOGICS: GroupLogic[] = ['all', 'any', 'none'];

// 条件组候选项由后端随 schema 下发，本组件不持有条件组数据
const availableSegments = computed(() => segmentsOf(props.scenario));

const field = computed(() => fieldOf(props.scenario, props.node.field));
const operator = computed(() => operatorOf(field.value, props.node.op));

// 字段按后端下发的 group 分组展示，分组顺序沿用注册顺序
const fieldGroups = computed(() => {
    const groups: { key: string; label: string; fields: PolicyField[] }[] = [];
    for (const item of props.scenario?.fields ?? []) {
        const existing = groups.find((group) => group.key === item.group);
        if (existing) existing.fields.push(item);
        else groups.push({ key: item.group, label: `flow.fieldGroup.${item.group}`, fields: [item] });
    }
    return groups;
});

const onFieldChange = (key: string) => {
    const next = fieldOf(props.scenario, key);
    props.node.field = key;
    const firstOperator = next?.ops[0];
    props.node.op = firstOperator?.name ?? '';
    props.node.value = emptyValueFor(firstOperator?.valueKind ?? 'single', next?.type);
};

const onOperatorChange = (name: string) => {
    props.node.op = name;
    const next = operatorOf(field.value, name);
    // 期望值形态随操作符改变（单值/多值/区间），残留旧形态会被后端判为非法取值
    props.node.value = emptyValueFor(next?.valueKind ?? 'single', field.value?.type);
};

const addItem = (kind: 'condition' | 'group' | 'segment') => {
    const items = props.node.items ?? (props.node.items = []);
    if (kind === 'group') items.push(createGroupNode('all'));
    else if (kind === 'segment') items.push(createSegmentNode(availableSegments.value[0]?.ref ?? ''));
    else items.push(createConditionNode(props.scenario?.fields[0]));
};

const removeItem = (index: number) => {
    props.node.items = (props.node.items ?? []).filter((_, itemIndex) => itemIndex !== index);
};
</script>

<style lang="scss" scoped>
.policy-condition-tree {
    &.depth-0 > .group-body {
        padding-left: 0;
        border-left: none;
    }

    .group-head {
        display: flex;
        align-items: center;
        gap: 6px;
        margin-bottom: 6px;
        min-width: 0;
    }

    .group-remove {
        margin-left: auto;
    }

    .group-label {
        font-size: 12px;
        color: var(--el-text-color-secondary);
    }

    .logic-select {
        width: 130px;
    }

    .group-body {
        display: flex;
        flex-direction: column;
        gap: 6px;
        padding-left: 14px;
        margin-left: 4px;
        border-left: 2px solid var(--el-color-primary-light-7);
    }

    .group-empty {
        font-size: 12px;
        color: var(--el-text-color-placeholder);
    }

    .group-actions {
        display: flex;
        gap: 10px;
        margin-top: 6px;
    }

    .segment-cell {
        width: 220px;
        flex-shrink: 0;
    }

    .segment-remark {
        float: right;
        margin-left: 12px;
        font-size: 12px;
        color: var(--el-text-color-secondary);
    }

    .condition-row {
        display: flex;
        align-items: center;
        gap: 6px;
        flex-wrap: wrap;
    }

    .cell {
        min-width: 0;

        :deep(.el-select),
        :deep(.el-input) {
            width: 100%;
        }
    }

    .field-cell {
        width: 158px;
        flex-shrink: 0;
    }

    .op-cell {
        width: 108px;
        flex-shrink: 0;
    }

    .value-cell {
        flex: 1 1 150px;
        min-width: 150px;
    }
}
</style>
