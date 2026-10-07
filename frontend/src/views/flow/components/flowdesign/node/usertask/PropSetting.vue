<template>
    <el-tabs v-model="activeTabName">
        <el-tab-pane :name="approvalRecordTabName" v-if="activeTabName == approvalRecordTabName" :label="$t('flow.approvalRecord')">
            <el-table :data="props.node?.properties?.tasks" stripe width="100%">
                <el-table-column :label="$t('common.createTime')" min-width="135">
                    <template #default="scope">
                        {{ formatDate(scope.row.createTime) }}
                    </template>
                </el-table-column>

                <el-table-column :label="$t('common.time')" min-width="135">
                    <template #default="scope">
                        {{ formatDate(scope.row.endTime) }}
                    </template>
                </el-table-column>

                <el-table-column :label="$t('flow.approver')" min-width="100">
                    <template #default="scope">
                        <AccountInfo :username="scope.row.handler || ''" />
                    </template>
                </el-table-column>

                <el-table-column :label="$t('flow.approveResult')" width="80">
                    <template #default="scope">
                        <EnumTag :enums="ProcinstTaskStatus" :value="scope.row.status" />
                    </template>
                </el-table-column>

                <el-table-column :label="$t('flow.approvalRemark')" min-width="150">
                    <template #default="scope">
                        {{ scope.row.remark }}
                    </template>
                </el-table-column>
            </el-table>
        </el-tab-pane>

        <el-tab-pane :label="$t('common.basic')" :name="basicTabName">
            <!-- 审批模式：预设一键套用，套完仍可继续改阈值（如把会签调成过半通过） -->
            <div class="approval-mode">
                <div class="mode-label">{{ $t('flow.approvalMode') }}</div>
                <FlowConditionEditor v-model="completionCondition" :scenario="scenario" :disabled="disabled" show-presets />
                <div v-if="conditionIssues.length" class="mode-issues">
                    <div v-for="(issue, index) in conditionIssues" :key="index">{{ issueText(issue) }}</div>
                </div>
            </div>

            <el-form-item class="mt-4" label-position="top" :label="$t('flow.taskCandidate')">
                <el-table :data="taskCandidates" stripe>
                    <el-table-column :label="$t('common.type')" width="150">
                        <template #header>
                            <el-button
                                class="ml-0"
                                type="primary"
                                circle
                                size="small"
                                icon="Plus"
                                :title="$t('flow.addCandidate')"
                                :aria-label="$t('flow.addCandidate')"
                                @click="onAddCandidate"
                            ></el-button>
                            <span class="ml-2">{{ $t('common.type') }}</span>
                        </template>
                        <template #default="scope">
                            <EnumSelect :enums="UserTaskCandidateType" v-model="scope.row.type" />
                        </template>
                    </el-table-column>

                    <el-table-column :label="$t('common.name')" min-width="150">
                        <template #default="scope">
                            <AccountSelectFormItem label="" v-if="scope.row.type == UserTaskCandidateType.Account.value" v-model="scope.row.id" />
                            <RoleSelectFormItem label="" v-else-if="scope.row.type == UserTaskCandidateType.Role.value" v-model="scope.row.id" />
                            <el-input v-else v-model="scope.row.name" clearable> </el-input>
                        </template>
                    </el-table-column>

                    <el-table-column :label="$t('common.operation')" min-width="50">
                        <template #default="scope">
                            <el-button
                                type="danger"
                                :title="$t('flow.removeCandidate')"
                                :aria-label="$t('flow.removeCandidate')"
                                @click="onDeleteCandidate(scope.$index, scope.row)"
                                icon="delete"
                                plain
                            ></el-button>
                        </template>
                    </el-table-column>
                </el-table>
            </el-form-item>
        </el-tab-pane>
    </el-tabs>
</template>
<script lang="ts" setup>
import { isTrue, notEmpty } from '@/common/assert';
import { formatDate } from '@/common/utils/format';
import EnumSelect from '@/components/enum-select/EnumSelect.vue';
import EnumTag from '@/components/enum-tag/EnumTag.vue';
import { useI18nPleaseSelect } from '@/hooks/useI18n';
import { ProcinstTaskStatus, UserTaskCandidateType } from '@/views/flow/enums';
import AccountInfo from '@/views/system/account/components/AccountInfo.vue';
import AccountSelectFormItem from '@/views/system/account/components/AccountSelectFormItem.vue';
import RoleSelectFormItem from '@/views/system/role/components/RoleSelectFormItem.vue';
import { computed, onMounted, Ref, ref, type PropType } from 'vue';
import { useI18n } from 'vue-i18n';
import { conditionScenarioOf, usePolicySchema, validateConditionTree, type PolicyIssue, type RuleNode } from '@/components/policy-builder';
import { FLOW_INSTANCE_BIZ_TYPE } from '@/views/flow/enums';
import { procdefApi } from '@/views/flow/api';
import FlowConditionEditor from '@/views/flow/components/FlowConditionEditor.vue';
import type { FlowNode } from '@/views/flow/types';

const props = defineProps({
    // 节点信息
    node: {
        type: Object as PropType<FlowNode>,
        default: null,
    },
    // 只读模式（查看流程实例时禁止改配置）
    disabled: {
        type: Boolean,
        default: false,
    },
});

const basicTabName = 'basic';
const approvalRecordTabName = 'approvalRecord';

const activeTabName = computed(() => {
    // 如果存在审批记录 tasks 且长度大于0，则激活审批记录 tab
    if (props.node?.properties?.tasks && props.node.properties.tasks.length > 0) {
        return approvalRecordTabName;
    }
    return basicTabName;
});

const form: any = defineModel<any>('modelValue', { required: true });

const { t } = useI18n();

// 完成条件的可比较字段由后端流程实例变量字典下发：新增一类变量只改注册表，本组件不动
const { policySchema, load: loadSchema } = usePolicySchema(() => procdefApi.policySchema.request());
const scenario = computed(() => conditionScenarioOf(policySchema.value, FLOW_INSTANCE_BIZ_TYPE));

const completionCondition = computed<RuleNode | null>({
    get: () => (form.value?.completionCondition as RuleNode) ?? null,
    set: (value) => {
        form.value.completionCondition = value;
    },
});

const conditionIssues = computed<PolicyIssue[]>(() => validateConditionTree(completionCondition.value, scenario.value));
const issueText = (issue: PolicyIssue) => t(issue.reasonKey, issue.params ?? {});

onMounted(loadSchema);

const taskCandidates: Ref<any> = ref([]);

onMounted(() => {
    const rawCandidates = form.value?.candidates || [];
    taskCandidates.value = rawCandidates.map((item: any) => {
        if (item && typeof item === 'object') {
            return item;
        }

        if (item.indexOf(':') == -1) {
            return { type: UserTaskCandidateType.Account.value, id: Number.parseInt(item) };
        }

        let [type, id] = item.split(':');
        if (type == '') {
            type = UserTaskCandidateType.Account.value;
        }
        return { type: type, id: Number.parseInt(id) };
    });
});

const onAddCandidate = () => {
    // 往数组头部添加元素
    taskCandidates.value = [...(taskCandidates.value || []), {}];
};

const onDeleteCandidate = async (idx: any, row: any) => {
    taskCandidates.value.splice(idx, 1);
};

const confirm = () => {
    notEmpty(taskCandidates.value, useI18nPleaseSelect('flow.taskCandidate'));

    // 完成条件缺失时旧实现会把任务永远停在「无人完成」或「第一人即完成」之间摇摆，
    // 后端保存流程时同样拒绝，这里提前挡住并定位到具体控件
    isTrue(Boolean(completionCondition.value), 'flow.approvalModeRequired');
    isTrue(conditionIssues.value.length === 0, 'flow.conditionIncomplete');
    form.value.candidates = taskCandidates.value.map((x: any) => {
        if (x.type == UserTaskCandidateType.Account.value) {
            return `${x.id}`;
        }
        return `${x.type}:${x.id}`;
    });
};

defineExpose({
    confirm,
});
</script>
<style lang="scss" scoped>
.approval-mode {
    margin-bottom: 12px;

    .mode-label {
        margin-bottom: 6px;
        font-size: 13px;
        color: var(--el-text-color-regular);
    }

    .mode-issues {
        margin-top: 6px;
        font-size: 12px;
        color: var(--el-color-danger);
    }
}
</style>
