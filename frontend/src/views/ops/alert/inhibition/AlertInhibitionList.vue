<template>
    <div class="h-full">
        <page-table ref="pageTableRef" :page-api="alertInhibitionApi.list" :search-items="searchItems" v-model:query-form="query" :columns="columns">
            <template #tableHeader>
                <el-button v-auth="'alert:inhibition:save'" type="primary" icon="plus" @click="onAdd">{{ $t('alert.addInhibition') }}</el-button>
            </template>

            <template #status="{ data }">
                <enum-tag :enums="AlertRuleStatusEnum" :value="data.status" />
            </template>

            <template #sourceMatch="{ data }">
                <LabelTags :labels="data.sourceMatch" />
            </template>

            <template #targetMatch="{ data }">
                <LabelTags :labels="data.targetMatch" />
            </template>

            <template #action="{ data }">
                <el-button link @click="onEdit(data)" type="primary">{{ $t('alert.edit') }}</el-button>
                <el-button link :type="data.status === AlertRuleStatusEnable ? 'warning' : 'success'" @click="onToggleStatus(data)">
                    {{ data.status === AlertRuleStatusEnable ? $t('alert.disable') : $t('alert.enable') }}
                </el-button>
                <el-button link @click="onDelete(data)" type="danger">{{ $t('alert.delete') }}</el-button>
            </template>
        </page-table>

        <auto-form-dialog
            v-model:visible="dialogVisible"
            :title="$t(editTitle)"
            :items="formItems"
            :data="editData"
            width="640px"
            :confirm-api="onConfirm"
            @submitted="onSubmitted"
        >
            <template #sourceMatch="{ form }">
                <LabelAssociation v-model="form.sourceMatch" />
            </template>
            <template #targetMatch="{ form }">
                <LabelAssociation v-model="form.targetMatch" />
            </template>
            <template #equalLabels>
                <el-input v-model="equalLabelsText" :placeholder="$t('alert.equalLabelsPlaceholder')" />
            </template>
        </auto-form-dialog>
    </div>
</template>

<script lang="ts" setup>
import { computed, ref, useTemplateRef, watch } from 'vue';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { AutoFormDialog, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import { alertInhibitionApi } from '../api';
import { AlertRuleStatusEnum } from '../enums';
import type { AlertInhibitionVO, AlertInhibitionForm } from '../types';
import LabelAssociation from '@/views/ops/label/components/LabelAssociation.vue';
import EnumTag from '@/components/enum-tag/EnumTag.vue';
import { LabelTags } from '@/components/label-tags';
import { useLabelColors } from '@/hooks/useLabelFill';

const AlertRuleStatusEnable = AlertRuleStatusEnum.Enable.value;
const AlertRuleStatusDisable = AlertRuleStatusEnum.Disable.value;

// 加载标签颜色（抑制规则的 sourceMatch/targetMatch 已随列表返回，无需 fill）
useLabelColors();

const searchItems = computed(() => [
    SearchItem.input('name', 'alert.inhibitionName').withPlaceholder('common.keyword'),
    SearchItem.select('status', 'alert.status').withOptions(Object.values(AlertRuleStatusEnum).map((e) => ({ value: e.value, label: e.label }))),
]);

const columns = [
    TableColumn.new('name', 'alert.inhibitionName'),
    TableColumn.new('sourceMatch', 'alert.sourceMatch').isSlot().setMinWidth(160),
    TableColumn.new('targetMatch', 'alert.targetMatch').isSlot().setMinWidth(160),
    TableColumn.new('equal', 'alert.equalLabels'),
    TableColumn.new('status', 'alert.status').isSlot().alignCenter(),
    TableColumn.new('remark', 'alert.remark'),
    TableColumn.new('action', 'common.operation').isSlot().fixedRight().setMinWidth(180).alignCenter(),
];

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');

const query = ref({
    name: '',
    status: undefined as number | undefined,
    pageNum: 1,
    pageSize: 0,
});

// ---- 表单配置 ----
const dialogVisible = ref(false);
const editTitle = ref('');
const editRow = ref<AlertInhibitionVO | null>(null);
const equalLabelsText = ref('');

const formItems: AutoFormItem[] = [
    { prop: 'name', label: 'alert.inhibitionName', required: true },
    {
        prop: 'sourceMatch',
        label: 'alert.sourceMatch',
        type: 'custom',
        required: true,
        tooltip: 'alert.sourceMatchTips',
    },
    {
        prop: 'targetMatch',
        label: 'alert.targetMatch',
        type: 'custom',
        required: true,
        tooltip: 'alert.targetMatchTips',
    },
    {
        prop: 'equalLabels',
        label: 'alert.equalLabels',
        type: 'custom',
        tooltip: 'alert.equalLabelsTips',
    },
    { prop: 'remark', label: 'alert.remark', type: 'textarea', rows: 3 },
];

const defaultForm: AlertInhibitionForm = {
    name: '',
    status: 1,
    sourceMatch: '',
    targetMatch: '',
    equal: '',
    remark: '',
};

const editData = computed<AutoFormData | null>(() => {
    if (!editRow.value) {
        return { ...defaultForm } as unknown as AutoFormData;
    }
    return { ...editRow.value } as unknown as AutoFormData;
});

// 弹窗打开时解析 equal 标签为逗号分隔文本
watch(dialogVisible, (visible) => {
    if (visible) {
        const equal = editRow.value?.equal;
        if (equal) {
            try {
                const arr = JSON.parse(equal);
                equalLabelsText.value = Array.isArray(arr) ? arr.join(',') : equal;
            } catch {
                equalLabelsText.value = equal;
            }
        } else {
            equalLabelsText.value = '';
        }
    }
});

const onAdd = () => {
    editRow.value = null;
    editTitle.value = 'alert.addInhibition';
    dialogVisible.value = true;
};

const onEdit = (row: AlertInhibitionVO) => {
    editRow.value = row;
    editTitle.value = 'alert.editInhibition';
    dialogVisible.value = true;
};

const onConfirm = async (rawForm: AutoFormData) => {
    const form = rawForm as unknown as AlertInhibitionForm;
    // 序列化 equal 标签
    const equalArr = equalLabelsText.value
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
    form.equal = equalArr.length ? JSON.stringify(equalArr) : '[]';
    await alertInhibitionApi.save.request(form);
};

const onSubmitted = () => {
    pageTableRef.value?.search();
};

const onToggleStatus = async (row: AlertInhibitionVO) => {
    const newStatus = row.status === AlertRuleStatusEnable ? AlertRuleStatusDisable : AlertRuleStatusEnable;
    const i18nKey = newStatus === 1 ? 'alert.confirmEnable' : 'alert.confirmDisable';
    await useI18nConfirm(i18nKey);
    await alertInhibitionApi.changeStatus.request({ id: row.id, status: newStatus });
    Msg.operateSuccess();
    pageTableRef.value?.search();
};

const onDelete = async (row: AlertInhibitionVO) => {
    await useI18nConfirm('alert.confirmDelete');
    await alertInhibitionApi.del.request({ id: row.id });
    Msg.deleteSuccess();
    pageTableRef.value?.search();
};
</script>
