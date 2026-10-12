<template>
    <div class="h-full">
        <data-table
            ref="dataTableRef"
            :page-api="alertRuleApi.list"
            :search-items="searchItems"
            v-model:query-form="query"
            :columns="columns"
            :data-handler-fn="handleListData"
        >
            <template #tableHeader>
                <el-button v-auth="'alert:rule:save'" type="primary" icon="plus" @click="onAdd">{{ $t('alert.addRule') }}</el-button>
            </template>

            <template #resourceType="{ data }">
                {{ getResourceTypeLabel(data.resourceType) }}
            </template>

            <template #priority="{ data }">
                <enum-tag :enums="AlertPriorityEnum" :value="data.priority" />
            </template>

            <template #status="{ data }">
                <enum-tag :enums="AlertRuleStatusEnum" :value="data.status" />
            </template>

            <template #labels="{ data }">
                <LabelTags :labels="data.labels" />
            </template>

            <!-- 行内操作由表格统一渲染（见 columns 里的 setActions），列宽与按钮同源 -->
        </data-table>

        <AlertRuleEdit v-model:visible="editVisible" :data="editData" :title="editTitle" @cancel="editVisible = false" @val-change="onEditSuccess" />
    </div>
</template>

<script lang="ts" setup>
import { TableColumn } from '@/components/data-table';
import DataTable from '@/components/data-table/DataTable.vue';
import { SearchItem } from '@/components/data-table/SearchForm';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { computed, ref, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import { alertRuleApi } from '../api';
import AlertRuleEdit from './AlertRuleEdit.vue';
import { AlertRuleStatusEnum, AlertPriorityEnum } from '../enums';
import type { AlertRuleVO } from '../types';
import { ensureResourceMetadata, getResourceTypeLabel, supportedResourceTypes } from '../utils';
import EnumTag from '@/components/enum-tag/EnumTag.vue';
import { LabelTags } from '@/components/label-tags';
import { useLabelFill } from '@/hooks/useLabelFill';

// 资源类型选项依赖后端下发结果，故为 computed 以便加载完成后刷新下拉
const searchItems = computed(() => [
    SearchItem.input('name', 'alert.ruleName').withPlaceholder('common.keyword'),
    SearchItem.select('resourceType', 'alert.resourceType').withOptions(supportedResourceTypes.value),
    SearchItem.select('status', 'alert.status').withOptions(Object.values(AlertRuleStatusEnum).map((e) => ({ value: e.value, label: e.label }))),
]);

const columns = [
    TableColumn.new('name', 'alert.ruleName'),
    TableColumn.new('resourceType', 'alert.resourceType').isSlot().alignCenter(),
    TableColumn.new('priority', 'alert.priority').isSlot().alignCenter(),
    TableColumn.new('status', 'alert.status').isSlot().alignCenter(),
    TableColumn.new('labels', 'alert.labels').isSlot().setMinWidth(200),
    TableColumn.new('remark', 'alert.remark'),
    TableColumn.new('action', 'common.operation')
        .setActions<AlertRuleVO>([
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

const { t } = useI18n();

const AlertRuleStatusEnable = AlertRuleStatusEnum.Enable.value;
const AlertRuleStatusDisable = AlertRuleStatusEnum.Disable.value;

// 加载后端支持的资源类型与指标元信息，供搜索与规则表单共用
ensureResourceMetadata();

// 标签批量填充（一行搞定，颜色加载 + dataHandlerFn）
const handleListData = useLabelFill<AlertRuleVO>('alert_rule');

const dataTableRef = useTemplateRef<InstanceType<typeof DataTable>>('dataTableRef');

const query = ref({
    name: '',
    resourceType: undefined as number | undefined,
    status: undefined as number | undefined,
    pageNum: 1,
    pageSize: 0,
});

const editVisible = ref(false);
const editData = ref<AlertRuleVO | null>(null);
const editTitle = ref('');

const onAdd = () => {
    editData.value = null;
    editTitle.value = t('alert.addRule');
    editVisible.value = true;
};

const onEdit = (row: AlertRuleVO) => {
    editData.value = row;
    editTitle.value = t('alert.editRule');
    editVisible.value = true;
};

const onToggleStatus = async (row: AlertRuleVO) => {
    const newStatus = row.status === AlertRuleStatusEnable ? AlertRuleStatusDisable : AlertRuleStatusEnable;
    const i18nKey = newStatus === 1 ? 'alert.confirmEnable' : 'alert.confirmDisable';
    if (!(await useI18nConfirm(i18nKey))) {
        return;
    }
    await alertRuleApi.changeStatus.request({ id: row.id, status: newStatus });
    Msg.operateSuccess();
    dataTableRef.value?.search();
};

const onDelete = async (row: AlertRuleVO) => {
    if (!(await useI18nConfirm('alert.confirmDelete'))) {
        return;
    }
    await alertRuleApi.del.request({ id: row.id });
    Msg.deleteSuccess();
    dataTableRef.value?.search();
};

const onEditSuccess = () => {
    editVisible.value = false;
    dataTableRef.value?.search();
};
</script>
