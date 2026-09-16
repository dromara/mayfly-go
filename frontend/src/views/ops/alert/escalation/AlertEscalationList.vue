<template>
    <div class="h-full">
        <page-table ref="pageTableRef" :page-api="alertEscalationApi.list" :search-items="searchItems" v-model:query-form="query" :columns="columns" :data-handler-fn="handleListData">
            <template #tableHeader>
                <el-button v-auth="'alert:escalation:save'" type="primary" icon="plus" @click="onAdd">{{ $t('alert.addEscalation') }}</el-button>
            </template>

            <template #matchLabels="{ data }">
                <LabelTags :labels="data.matchLabels" />
            </template>

            <template #rules="{ data }">
                {{ escalationChainText(data.rules) }}
            </template>

            <template #status="{ data }">
                <enum-tag :enums="AlertRuleStatusEnum" :value="data.status" />
            </template>

            <template #action="{ data }">
                <el-button link @click="onEdit(data)" type="primary">{{ $t('alert.edit') }}</el-button>
                <el-button link :type="data.status === AlertRuleStatusEnable ? 'warning' : 'success'" @click="onToggleStatus(data)">
                    {{ data.status === AlertRuleStatusEnable ? $t('alert.disable') : $t('alert.enable') }}
                </el-button>
                <el-button link @click="onDelete(data)" type="danger">{{ $t('alert.delete') }}</el-button>
            </template>
        </page-table>

        <el-dialog v-model="editVisible" :title="$t(editTitle)" width="860px" destroy-on-close>
            <el-form :model="editForm" label-width="110px">
                <el-form-item :label="$t('alert.policyName')" required>
                    <el-input v-model="editForm.name" :placeholder="$t('alert.policyName')" />
                </el-form-item>
                <el-form-item>
                    <template #label>
                        <div class="flex items-center">
                            {{ $t('alert.matchLabels') }}
                            <el-tooltip placement="top">
                                <template #content><span style="white-space: pre-line">{{ $t('alert.matchLabelsTips') }}</span></template>
                                <SvgIcon name="QuestionFilled" class="ml-1" />
                            </el-tooltip>
                        </div>
                    </template>
                    <LabelAssociation v-model="editForm.matchLabels" />
                </el-form-item>
                <el-form-item required>
                    <template #label>
                        <div class="flex items-center">
                            {{ $t('alert.escalationRules') }}
                            <el-tooltip placement="top">
                                <template #content><span style="white-space: pre-line">{{ $t('alert.escalationRulesTips') }}</span></template>
                                <SvgIcon name="QuestionFilled" class="ml-1" />
                            </el-tooltip>
                        </div>
                    </template>
                    <el-table :data="editForm.rules" size="small" border class="escalation-table">
                        <el-table-column type="index" width="50" align="center" />
                        <el-table-column :label="$t('alert.delayMinutes')" width="150">
                            <template #default="{ row }">
                                <el-input-number v-model="row.delayMinutes" :min="1" :step="5" :controls="false" style="width: 100%" />
                            </template>
                        </el-table-column>
                        <el-table-column :label="$t('alert.notifyChannel')" min-width="220">
                            <template #default="{ row }">
                                <ChannelSelect v-model="row.channelIds" />
                            </template>
                        </el-table-column>
                        <el-table-column :label="$t('alert.notifyReceiver')" min-width="220">
                            <template #default="{ row }">
                                <ReceiverSelect v-model="row.receiverIds" />
                            </template>
                        </el-table-column>
                        <el-table-column :label="$t('common.operation')" width="80" align="center">
                            <template #default="{ $index }">
                                <el-button type="danger" link @click="editForm.rules.splice($index, 1)">{{ $t('alert.delete') }}</el-button>
                            </template>
                        </el-table-column>
                    </el-table>
                    <el-button type="primary" link @click="addLevel">+ {{ $t('alert.addEscalationLevel') }}</el-button>
                </el-form-item>
                <el-form-item :label="$t('alert.remark')">
                    <el-input v-model="editForm.remark" type="textarea" :rows="3" :placeholder="$t('alert.remark')" />
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="editVisible = false">{{ $t('alert.cancel') }}</el-button>
                <el-button type="primary" @click="onSave">{{ $t('alert.save') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { reactive, ref, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { alertEscalationApi } from '../api';
import { AlertRuleStatusEnum } from '../enums';
import type { AlertEscalationVO, AlertEscalationForm, EscalationRule } from '../types';
import ChannelSelect from '../components/ChannelSelect.vue';
import ReceiverSelect from '../components/ReceiverSelect.vue';
import EnumTag from '@/components/enum-tag/EnumTag.vue';
import LabelAssociation from '@/views/ops/label/components/LabelAssociation.vue';
import { LabelTags } from '@/components/label-tags';
import { useLabelFill } from '@/hooks/useLabelFill';

const { t } = useI18n();

const AlertRuleStatusEnable = AlertRuleStatusEnum.Enable.value;
const AlertRuleStatusDisable = AlertRuleStatusEnum.Disable.value;

// 标签批量填充
const handleListData = useLabelFill<AlertEscalationVO>('alert_escalation', { labelField: 'matchLabels' });

const searchItems = [
    SearchItem.input('name', 'alert.policyName').withPlaceholder('common.keyword'),
    SearchItem.select('status', 'alert.status').withOptions(Object.values(AlertRuleStatusEnum).map((e) => ({ value: e.value, label: e.label }))),
];

const columns = [
    TableColumn.new('name', 'alert.policyName'),
    TableColumn.new('matchLabels', 'alert.matchLabels').isSlot().setMinWidth(200),
    TableColumn.new('rules', 'alert.escalationRules').isSlot(),
    TableColumn.new('status', 'alert.status').isSlot().alignCenter(),
    TableColumn.new('remark', 'alert.remark'),
    TableColumn.new('action', 'common.operation').isSlot().fixedRight().setMinWidth(180).alignCenter(),
];

/** 升级链路摘要：延迟分钟 -> 渠道数，便于在列表一眼看出配置是否生效 */
const escalationChainText = (rules?: EscalationRule[] | null) => {
    if (!rules?.length) {
        return '-';
    }
    return rules.map((rule) => `${rule.delayMinutes}${t('alert.minutes')}`).join(' → ');
};

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');

const query = ref({
    name: '',
    status: undefined as number | undefined,
    pageNum: 1,
    pageSize: 0,
});

const editVisible = ref(false);
const editTitle = ref('');

const createEmptyForm = (): AlertEscalationForm => ({
    id: null,
    name: '',
    status: 1,
    matchLabels: '',
    rules: [],
    remark: '',
});

const editForm = reactive<AlertEscalationForm>(createEmptyForm());

const addLevel = () => {
    const last = editForm.rules[editForm.rules.length - 1];
    editForm.rules.push({ delayMinutes: last ? last.delayMinutes + 30 : 30, channelIds: [], receiverIds: [] });
};

const onAdd = () => {
    Object.assign(editForm, createEmptyForm());
    editTitle.value = 'alert.addEscalation';
    editVisible.value = true;
};

const onEdit = (row: AlertEscalationVO) => {
    Object.assign(editForm, createEmptyForm(), { ...row });
    // 深拷贝规则链：编辑过程中的增删改不能污染列表行数据
    editForm.rules = (row.rules ?? []).map((rule) => ({ ...rule, channelIds: [...(rule.channelIds ?? [])], receiverIds: [...(rule.receiverIds ?? [])] }));
    editTitle.value = 'alert.editEscalation';
    editVisible.value = true;
};

/**
 * 保存前校验。
 *
 * 某级别没有渠道时后端仍会推进升级级别但不发出通知，导致后续级别被判定为"已升级过"而永久漏报，
 * 因此渠道必填必须在前端拦住
 */
const validateForm = (): string | null => {
    if (!editForm.name.trim()) {
        return 'alert.policyNameRequired';
    }
    if (!editForm.rules.length) {
        return 'alert.escalationRulesRequired';
    }
    for (const rule of editForm.rules) {
        if (!rule.delayMinutes || rule.delayMinutes <= 0) {
            return 'alert.escalationDelayRequired';
        }
        // 渠道和接收人至少配置一个
        if (!rule.channelIds?.length && !rule.receiverIds?.length) {
            return 'alert.escalationChannelOrReceiverRequired';
        }
    }
    return null;
};

const onSave = async () => {
    const errorKey = validateForm();
    if (errorKey) {
        Msg.error(errorKey);
        return;
    }
    await alertEscalationApi.save.request({ ...editForm, name: editForm.name.trim(), matchLabels: editForm.matchLabels?.trim() || '{}' });
    Msg.saveSuccess();
    editVisible.value = false;
    pageTableRef.value?.search();
};

const onToggleStatus = async (row: AlertEscalationVO) => {
    const newStatus = row.status === AlertRuleStatusEnable ? AlertRuleStatusDisable : AlertRuleStatusEnable;
    const i18nKey = newStatus === 1 ? 'alert.confirmEnable' : 'alert.confirmDisable';
    await useI18nConfirm(i18nKey);
    await alertEscalationApi.changeStatus.request({ id: row.id, status: newStatus });
    Msg.operateSuccess();
    pageTableRef.value?.search();
};

const onDelete = async (row: AlertEscalationVO) => {
    await useI18nConfirm('alert.confirmDelete');
    await alertEscalationApi.del.request({ id: row.id });
    Msg.deleteSuccess();
    pageTableRef.value?.search();
};
</script>

<style lang="scss" scoped>
.escalation-table {
    width: 100%;
    margin-bottom: 8px;
}
</style>
