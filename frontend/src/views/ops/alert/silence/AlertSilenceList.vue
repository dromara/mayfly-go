<template>
    <div class="h-full">
        <data-table
            ref="dataTableRef"
            :page-api="alertSilenceApi.list"
            :search-items="searchItems"
            v-model:query-form="query"
            :columns="columns"
            :data-handler-fn="handleListData"
        >
            <template #tableHeader>
                <el-button v-auth="'alert:silence:save'" type="primary" icon="plus" @click="onAdd">{{ $t('alert.addSilence') }}</el-button>
            </template>

            <template #matchLabels="{ data }">
                <LabelTags :labels="data.matchLabels" />
            </template>

            <template #status="{ data }">
                <enum-tag :enums="AlertRuleStatusEnum" :value="data.status" />
            </template>

            <!-- 行内操作由表格统一渲染（见 columns 里的 setActions），列宽与按钮同源 -->
        </data-table>

        <el-dialog v-model="editVisible" :title="$t(editTitle)" width="640px" destroy-on-close>
            <el-form :model="editForm" label-width="110px">
                <el-form-item :label="$t('alert.silenceName')" required>
                    <el-input v-model="editForm.name" :placeholder="$t('alert.silenceName')" />
                </el-form-item>
                <el-form-item>
                    <template #label>
                        <div class="flex items-center">
                            {{ $t('alert.matchLabels') }}
                            <el-tooltip placement="top">
                                <template #content
                                    ><span style="white-space: pre-line">{{ $t('alert.matchLabelsTips') }}</span></template
                                >
                                <SvgIcon name="QuestionFilled" class="ml-1" />
                            </el-tooltip>
                        </div>
                    </template>
                    <LabelAssociation v-model="editForm.matchLabels" />
                </el-form-item>
                <el-form-item :label="$t('alert.effectiveTime')" required>
                    <div class="time-range">
                        <el-date-picker v-model="editForm.startTime" type="datetime" :placeholder="$t('alert.startTime')" value-format="YYYY-MM-DD HH:mm:ss" />
                        <span class="time-sep">{{ $t('alert.timeTo') }}</span>
                        <el-date-picker v-model="editForm.endTime" type="datetime" :placeholder="$t('alert.endTime')" value-format="YYYY-MM-DD HH:mm:ss" />
                    </div>
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
import { computed, reactive, ref, useTemplateRef } from 'vue';
import { TableColumn } from '@/components/data-table';
import DataTable from '@/components/data-table/DataTable.vue';
import { SearchItem } from '@/components/data-table/SearchForm';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { alertSilenceApi } from '../api';
import { AlertRuleStatusEnum } from '../enums';
import type { AlertSilenceVO, AlertSilenceForm } from '../types';
import EnumTag from '@/components/enum-tag/EnumTag.vue';
import LabelAssociation from '@/views/ops/label/components/LabelAssociation.vue';
import { LabelTags } from '@/components/label-tags';
import { useLabelFill } from '@/hooks/useLabelFill';

const AlertRuleStatusEnable = AlertRuleStatusEnum.Enable.value;
const AlertRuleStatusDisable = AlertRuleStatusEnum.Disable.value;

// 标签批量填充
const handleListData = useLabelFill<AlertSilenceVO>('alert_silence', { labelField: 'matchLabels' });

const searchItems = [
    SearchItem.input('name', 'alert.silenceName').withPlaceholder('common.keyword'),
    SearchItem.select('status', 'alert.status').withOptions(Object.values(AlertRuleStatusEnum).map((e) => ({ value: e.value, label: e.label }))),
];

const columns = [
    TableColumn.new('name', 'alert.silenceName'),
    TableColumn.new('matchLabels', 'alert.matchLabels').isSlot().setMinWidth(200),
    TableColumn.new('startTime', 'alert.startTime').isTime(),
    TableColumn.new('endTime', 'alert.endTime').isTime(),
    TableColumn.new('status', 'alert.status').isSlot().alignCenter(),
    TableColumn.new('remark', 'alert.remark'),
    TableColumn.new('action', 'common.operation')
        .setActions<AlertSilenceVO>([
            { key: 'edit', label: 'alert.edit', type: 'primary', onClick: (row) => onEdit(row) },
            // 启用/禁用互斥：两格各自带行级条件与配色，计宽按同组最宽的一格
            {
                key: 'disable',
                label: 'alert.disable',
                type: 'warning',
                group: 'status',
                show: (row) => row.status === AlertRuleStatusEnable,
                onClick: (row) => onToggleStatus(row),
            },
            {
                key: 'enable',
                label: 'alert.enable',
                type: 'success',
                group: 'status',
                show: (row) => row.status !== AlertRuleStatusEnable,
                onClick: (row) => onToggleStatus(row),
            },
            { key: 'del', label: 'alert.delete', type: 'danger', onClick: (row) => onDelete(row) },
        ])
        .fixedRight()
        .alignCenter(),
];

const dataTableRef = useTemplateRef<InstanceType<typeof DataTable>>('dataTableRef');

const query = ref({
    name: '',
    status: undefined as number | undefined,
    pageNum: 1,
    pageSize: 0,
});

const editVisible = ref(false);
const editTitle = ref('');

const createEmptyForm = (): AlertSilenceForm => ({
    id: null,
    name: '',
    status: 1,
    matchLabels: '',
    startTime: '',
    endTime: '',
    remark: '',
});

const editForm = reactive<AlertSilenceForm>(createEmptyForm());

const resetForm = (source?: AlertSilenceVO) => {
    Object.assign(editForm, createEmptyForm(), source ? { ...source } : {});
};

const onAdd = () => {
    resetForm();
    editTitle.value = 'alert.addSilence';
    editVisible.value = true;
};

const onEdit = (row: AlertSilenceVO) => {
    resetForm(row);
    editTitle.value = 'alert.editSilence';
    editVisible.value = true;
};

const validateForm = (): string | null => {
    if (!editForm.name.trim()) {
        return 'alert.silenceNameRequired';
    }
    if (!editForm.startTime || !editForm.endTime) {
        return 'alert.effectiveTimeRequired';
    }
    if (new Date(editForm.endTime).getTime() <= new Date(editForm.startTime).getTime()) {
        return 'alert.effectiveTimeInvalid';
    }
    return null;
};

const onSave = async () => {
    const errorKey = validateForm();
    if (errorKey) {
        Msg.error(errorKey);
        return;
    }
    await alertSilenceApi.save.request({ ...editForm, matchLabels: editForm.matchLabels?.trim() || '{}' });
    Msg.saveSuccess();
    editVisible.value = false;
    dataTableRef.value?.search();
};

const onToggleStatus = async (row: AlertSilenceVO) => {
    const newStatus = row.status === AlertRuleStatusEnable ? AlertRuleStatusDisable : AlertRuleStatusEnable;
    const i18nKey = newStatus === 1 ? 'alert.confirmEnable' : 'alert.confirmDisable';
    if (!(await useI18nConfirm(i18nKey))) {
        return;
    }
    await alertSilenceApi.changeStatus.request({ id: row.id, status: newStatus });
    Msg.operateSuccess();
    dataTableRef.value?.search();
};

const onDelete = async (row: AlertSilenceVO) => {
    if (!(await useI18nConfirm('alert.confirmDelete'))) {
        return;
    }
    await alertSilenceApi.del.request({ id: row.id });
    Msg.deleteSuccess();
    dataTableRef.value?.search();
};
</script>

<style lang="scss" scoped>
.time-range {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
}

.time-sep {
    color: var(--el-text-color-secondary);
}
</style>
