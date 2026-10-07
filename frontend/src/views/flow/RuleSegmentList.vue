<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="ruleSegmentApi.list"
            :search-items="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="columns"
        >
            <template #empty>
                <div class="table-empty">
                    <p>{{ $t('flow.ruleSegment.empty') }}</p>
                    <p class="empty-tip">{{ $t('flow.ruleSegment.emptyTip') }}</p>
                </div>
            </template>

            <template #tableHeader>
                <el-button v-auth="perms.save" type="primary" icon="plus" @click="onEdit(false)">{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.del" :disabled="selectionData.length < 1" type="danger" icon="delete" @click="onDelete()">
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #bizType="{ data }">
                <span>{{ scenarioText(data.bizType) }}</span>
            </template>

            <template #ruleNode="{ data }">
                <span class="rule-preview">{{
                    describeNode(data.ruleNode, scenarioOf(policySchema, data.bizType), t) || $t('flow.ruleSegment.emptyRule')
                }}</span>
            </template>

            <template #action="{ data }">
                <el-button v-if="actionBtns[perms.save]" link type="primary" @click="onEdit(data)">{{ $t('common.edit') }}</el-button>
            </template>
        </page-table>

        <el-drawer
            v-model="editorVisible"
            :title="editorTitle"
            :size="FLOW_DRAWER.form"
            :close-on-click-modal="false"
            :destroy-on-close="true"
            header-class="mb-2!"
            body-class="pt-2!"
        >
            <template #header>
                <DrawerHeader :header="editorTitle" :back="() => (editorVisible = false)" />
            </template>

            <el-form ref="formRef" :model="form" label-position="top">
                <div class="form-grid">
                    <el-form-item :label="$t('common.name')" prop="name" :rules="[Rules.requiredInput('common.name')]">
                        <el-input v-model="form.name" clearable maxlength="100" show-word-limit />
                    </el-form-item>

                    <el-form-item prop="bizType" :label="$t('flow.ruleSegment.bizType')" :rules="[Rules.requiredSelect('flow.ruleSegment.bizType')]">
                        <el-select v-model="form.bizType" class="w-full" @change="onBizTypeChange">
                            <el-option v-for="item in scenarioOptions" :key="item.bizType" :value="item.bizType" :label="item.label" />
                        </el-select>
                    </el-form-item>

                    <el-form-item prop="ref" :label="$t('flow.ruleSegment.ref')" :rules="[Rules.requiredInput('flow.ruleSegment.ref')]">
                        <el-input v-model="form.ref" :disabled="isEdit" clearable placeholder="dba_members" />
                        <div class="field-tip">{{ isEdit ? $t('flow.ruleSegment.refImmutableTip') : $t('flow.ruleSegment.refTip') }}</div>
                    </el-form-item>

                    <el-form-item :label="$t('common.remark')" prop="remark">
                        <el-input v-model="form.remark" clearable maxlength="255" />
                    </el-form-item>
                </div>

                <el-form-item :label="$t('flow.ruleSegment.condition')" prop="ruleNode">
                    <div class="condition-block">
                        <div class="condition-tip">{{ $t('flow.ruleSegment.conditionTip') }}</div>
                        <PolicyConditionNode v-if="currentScenario && ruleNode" :node="ruleNode" :scenario="currentScenario" />
                        <el-button v-else-if="currentScenario" size="small" icon="plus" plain @click="startEditing">{{
                            $t('flow.condition.addCondition')
                        }}</el-button>
                        <div v-else class="condition-tip">{{ $t('flow.policy.loading') }}</div>
                        <div v-if="issues.length" class="condition-issues">
                            <div v-for="(issue, index) in issues" :key="index">{{ t(issue.reasonKey, issue.params ?? {}) }}</div>
                        </div>
                    </div>
                </el-form-item>
            </el-form>

            <template #footer>
                <el-button @click="editorVisible = false">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="saving" @click="onSave">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';

import { hasPerms } from '@/components/auth/auth';
import { Rules } from '@/common/rule';
import {
    allScenariosOf,
    cloneRuleNode,
    createConditionNode,
    createGroupNode,
    describeNode,
    scenarioOf,
    usePolicySchema,
    validateConditionTree,
    type PolicyIssue,
    type PolicyScenario,
    type RuleNode,
} from '@/components/policy-builder';
import PolicyConditionNode from '@/components/policy-builder/PolicyConditionNode.vue';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import PageTable from '@/components/page-table/PageTable.vue';
import { TableColumn } from '@/components/page-table';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import { procdefApi, ruleSegmentApi, type RuleSegmentForm } from './api';
import type { RuleSegment } from './types';
import { FLOW_DRAWER } from '@/views/flow/drawerSize';

/**
 * 可复用条件组：把一段常用判断（如「DBA 成员」「工作时段之外」）单独维护，
 * 触发策略与流程图里的条件通过 segment 节点引用它。
 *
 * 条件组只存判断、不存处置级别：同一个判断在不同规则里可以定不同级别。
 */
const { t } = useI18n();

const perms = {
    save: 'flow:ruleSegment:save',
    del: 'flow:ruleSegment:del',
};

const actionBtns = hasPerms([perms.save, perms.del]) as Record<string, boolean>;

const searchItems = [SearchItem.input('name', 'common.name')];
const columns = [
    TableColumn.new('name', 'common.name'),
    TableColumn.new('ref', 'flow.ruleSegment.ref'),
    TableColumn.new('bizType', 'flow.ruleSegment.bizType').isSlot(),
    TableColumn.new('ruleNode', 'flow.ruleSegment.condition').isSlot().setMinWidth('240px'),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('createTime', 'common.createTime').isTime(),
];

const pageTableRef = useTemplateRef<{ search: () => void }>('pageTableRef');
const formRef = useTemplateRef<{ validate: () => Promise<boolean> }>('formRef');

const { policySchema, load, reload: reloadPolicySchema } = usePolicySchema(() => procdefApi.policySchema.request());

/** 编辑态：新增与编辑共用一份结构，类型在此声明一次，不在赋值处做形态断言 */
interface SegmentEditorState {
    selectionData: RuleSegment[];
    query: { name: string; pageNum: number; pageSize: number };
    editorVisible: boolean;
    editorTitle: string;
    saving: boolean;
    form: RuleSegmentForm;
    ruleNode: RuleNode | null;
}

const emptyForm = (): RuleSegmentForm => ({ ref: '', name: '', bizType: '', remark: '' });

const state = reactive<SegmentEditorState>({
    selectionData: [],
    query: { name: '', pageNum: 1, pageSize: 0 },
    editorVisible: false,
    editorTitle: '',
    saving: false,
    form: emptyForm(),
    ruleNode: null,
});

const { selectionData, query, editorVisible, editorTitle, saving, form, ruleNode } = toRefs(state);

const isEdit = computed(() => (state.form.id ?? 0) > 0);

// 条件组只能被同场景的规则引用，因此可选的字段字典必须覆盖两类条件用到的全部字典
const scenarioOptions = computed(() => allScenariosOf(policySchema.value).map((item) => ({ bizType: item.bizType, label: scenarioText(item.bizType) })));

const currentScenario = computed<PolicyScenario | null>(() => scenarioOf(policySchema.value, state.form.bizType));

const issues = computed<PolicyIssue[]>(() => validateConditionTree(state.ruleNode, currentScenario.value));

onMounted(async () => {
    await load();
    if (Object.keys(actionBtns).length > 0) {
        columns.push(TableColumn.new('action', 'common.operation').isSlot().fixedRight().setMinWidth(120).alignCenter());
    }
});

const scenarioText = (bizType: string) => {
    const key = `flow.bizTypeName.${bizType}`;
    const label = t(key);
    return label === key ? bizType : label;
};

const search = () => pageTableRef.value?.search();

const resetRuleNode = (bizType: string) => {
    // 换字段字典后原条件里的字段大概率不再存在，保留只会得到一串「字段已不存在」
    state.ruleNode = null;
    state.form.bizType = bizType;
};

const onBizTypeChange = (bizType: string) => resetRuleNode(bizType);

const startEditing = () => {
    // 建组即给一行：空分组会被保存校验判非法，点一下「添加条件」就飘红等于把设计缺陷甩给用户
    // （与 FlowConditionEditor.startEditing、ProcdefPolicyEditor.toggleUnless 同一约定）
    const group = createGroupNode('all');
    group.items = [createConditionNode(currentScenario.value?.fields[0])];
    state.ruleNode = group;
};

const onEdit = (data: RuleSegment | false) => {
    if (!data) {
        state.form = { ...emptyForm(), bizType: scenarioOptions.value[0]?.bizType ?? '' };
        state.ruleNode = null;
        state.editorTitle = useI18nCreateTitle('flow.ruleSegment.title');
    } else {
        state.form = { id: data.id, ref: data.ref, name: data.name, bizType: data.bizType, remark: data.remark };
        // 深拷贝：直接引用列表行对象会让未保存的编辑结果显示在列表里
        state.ruleNode = cloneRuleNode(data.ruleNode);
        state.editorTitle = useI18nEditTitle('flow.ruleSegment.title');
    }
    state.editorVisible = true;
};

const onSave = async () => {
    if (!(await formRef.value?.validate())) return;

    // 空条件组没有任何判断价值，却会让引用它的规则恒真或恒假，必须在保存前挡下
    if (!state.ruleNode) {
        Msg.error(t('flow.ruleSegment.conditionRequired'));
        return;
    }
    if (issues.value.length > 0) {
        Msg.error(t(issues.value[0].reasonKey, issues.value[0].params ?? {}));
        return;
    }

    state.saving = true;
    try {
        await ruleSegmentApi.save.request({ ...state.form, ruleNode: state.ruleNode });
        // 条件组清单随 policy-schema 一起下发：保存后必须重取，否则本页的「条件内容」摘要
        // 会降级成「未配置条件」、编辑抽屉永远停在加载中，别的策略编辑器也看不到新条件组
        await reloadPolicySchema();
        Msg.saveSuccess();
        state.editorVisible = false;
        search();
    } finally {
        state.saving = false;
    }
};

const onDelete = async () => {
    try {
        if (!(await useI18nDeleteConfirm(state.selectionData.map((item: RuleSegment) => item.name).join(', ')))) {
            return;
        }
        await ruleSegmentApi.del.request({ id: state.selectionData.map((item: RuleSegment) => item.id).join(',') });
        await reloadPolicySchema();
        Msg.deleteSuccess();
        search();
    } catch (err) {
        // 被引用时后端返回带引用方名称的错误，由请求层统一提示
    }
};
</script>

<style lang="scss" scoped>
.form-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0 12px;
}

.field-tip {
    font-size: 12px;
    line-height: 18px;
    color: var(--el-text-color-secondary);
}

.condition-block {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 8px;
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

.rule-preview {
    font-size: 12px;
    color: var(--el-text-color-secondary);
}
</style>
