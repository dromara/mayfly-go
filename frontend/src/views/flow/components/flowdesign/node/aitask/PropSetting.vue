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
                        {{ scope.row.handler || '' }}
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
            <el-form-item prop="auditRule" :label="$t('flow.aiAuditRule')">
                <el-input v-model="form.auditRule" type="textarea" :rows="10" :placeholder="$t('flow.aiAuditRuleTip')" clearable />
            </el-form-item>
        </el-tab-pane>
    </el-tabs>
</template>
<script lang="ts" setup>
import { notEmpty } from '@/common/assert';
import { formatDate } from '@/common/utils/format';
import EnumTag from '@/components/enumtag/EnumTag.vue';
import { useI18nPleaseInput } from '@/hooks/useI18n';
import { ProcinstTaskStatus } from '@/views/flow/enums';
import { computed } from 'vue';

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
    console.log(props.node);
    // 如果存在审批记录 tasks 且长度大于0，则激活审批记录 tab
    if (props.node?.properties?.opLog) {
        return approvalRecordTabName;
    }
    return basicTabName;
});

const form: any = defineModel<any>('modelValue', { required: true });

const confirm = () => {
    notEmpty(form.value.auditRule, useI18nPleaseInput('flow.aiAuditRule'));
};

defineExpose({
    confirm,
});
</script>
