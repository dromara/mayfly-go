<template>
    <div class="h-full">
        <page-table ref="pageTableRef" :page-api="alertRuleApi.list" :search-items="searchItems" v-model:query-form="query" :columns="columns" :data-handler-fn="handleListData">
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

            <template #action="{ data }">
                <el-button link @click="onEdit(data)" type="primary">{{ $t('alert.edit') }}</el-button>
                <el-button link :type="data.status === AlertRuleStatusEnable ? 'warning' : 'success'" @click="onToggleStatus(data)">
                    {{ data.status === AlertRuleStatusEnable ? $t('alert.disable') : $t('alert.enable') }}
                </el-button>
                <el-button link @click="onDelete(data)" type="danger">{{ $t('alert.delete') }}</el-button>
            </template>
        </page-table>

        <AlertRuleEdit v-model:visible="editVisible" :data="editData" :title="editTitle" @cancel="editVisible = false" @val-change="onEditSuccess" />
    </div>
</template>

<script lang="ts" setup>
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
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
    TableColumn.new('action', 'common.operation').isSlot().fixedRight().setMinWidth(180).alignCenter(),
];

const { t } = useI18n();

const AlertRuleStatusEnable = AlertRuleStatusEnum.Enable.value;
const AlertRuleStatusDisable = AlertRuleStatusEnum.Disable.value;

// 加载后端支持的资源类型与指标元信息，供搜索与规则表单共用
ensureResourceMetadata();

// 标签批量填充（一行搞定，颜色加载 + dataHandlerFn）
const handleListData = useLabelFill<AlertRuleVO>('alert_rule');

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');

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
    await useI18nConfirm(i18nKey);
    await alertRuleApi.changeStatus.request({ id: row.id, status: newStatus });
    Msg.operateSuccess();
    pageTableRef.value?.search();
};

const onDelete = async (row: AlertRuleVO) => {
    await useI18nConfirm('alert.confirmDelete');
    await alertRuleApi.del.request({ id: row.id });
    Msg.deleteSuccess();
    pageTableRef.value?.search();
};

const onEditSuccess = () => {
    editVisible.value = false;
    pageTableRef.value?.search();
};
</script>
