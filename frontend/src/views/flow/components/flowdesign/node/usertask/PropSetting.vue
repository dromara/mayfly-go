<template>
    <el-tabs v-model="activeTabName">
        <el-tab-pane :name="approvalRecordTabName" v-if="activeTabName == approvalRecordTabName" :label="$t('flow.approvalRecord')">
            <el-table :data="props.node?.properties?.tasks" stripe width="100%">
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
            <el-form-item prop="completionCondition" :label="$t('flow.approvalMode')" :rules="[Rules.requiredSelect('flow.approvalMode')]">
                <el-radio-group v-model="form.completionCondition">
                    <el-radio value="{{ eq .nrOfCompleted 1.0 }}">{{ $t('flow.orSign') }}</el-radio>
                    <el-radio value="{{ eq .nrOfAll .nrOfCompleted }}">{{ $t('flow.andSign') }}</el-radio>
                    <!-- <el-radio value="3">{{ $t('flow.voteSign') }}</el-radio> -->
                </el-radio-group>
            </el-form-item>

            <el-form-item label-position="top" :label="$t('flow.taskCandidate')">
                <el-table :data="taskCandidates" stripe>
                    <el-table-column :label="$t('common.type')" width="150">
                        <template #header>
                            <el-button class="ml-0" type="primary" circle size="small" icon="Plus" @click="onAddCandidate"> </el-button>
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
                            <el-button type="danger" @click="onDeleteCandidate(scope.$index, scope.row)" icon="delete" plain></el-button>
                        </template>
                    </el-table-column>
                </el-table>
            </el-form-item>
        </el-tab-pane>
    </el-tabs>
</template>
<script lang="ts" setup>
import { notEmpty } from '@/common/assert';
import { Rules } from '@/common/rule';
import { formatDate } from '@/common/utils/format';
import EnumSelect from '@/components/enumselect/EnumSelect.vue';
import EnumTag from '@/components/enumtag/EnumTag.vue';
import { useI18nPleaseSelect } from '@/hooks/useI18n';
import { ProcinstTaskStatus, UserTaskCandidateType } from '@/views/flow/enums';
import AccountInfo from '@/views/system/account/components/AccountInfo.vue';
import AccountSelectFormItem from '@/views/system/account/components/AccountSelectFormItem.vue';
import RoleSelectFormItem from '@/views/system/role/components/RoleSelectFormItem.vue';
import { computed, onMounted, Ref, ref, watch } from 'vue';

const props = defineProps({
    // 节点信息
    node: {
        type: Object,
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
