<template>
    <div class="h-full">
        <page-table ref="pageTableRef" :page-api="labelApi.list" :search-items="searchItems" v-model:query-form="query" :columns="columns">
            <template #tableHeader>
                <el-button v-auth="'label:save'" type="primary" icon="plus" @click="onAdd">{{ $t('label.addLabel') }}</el-button>
            </template>

            <template #color="{ data }">
                <el-tag
                    v-if="data.extra?.color"
                    :style="{
                        backgroundColor: data.extra.color + '20',
                        borderColor: data.extra.color,
                        color: data.extra.color,
                    }"
                    size="small"
                >
                    {{ data.labelValue }}
                </el-tag>
                <span v-else>{{ data.labelValue }}</span>
            </template>

            <template #description="{ data }">
                {{ data.extra?.description || '-' }}
            </template>

            <template #action="{ data }">
                <el-button link @click="onEdit(data)" type="primary">{{ $t('common.edit') }}</el-button>
                <el-button link @click="onDelete(data)" type="danger">{{ $t('common.delete') }}</el-button>
            </template>
        </page-table>

        <el-dialog v-model="editVisible" :title="$t(editTitle)" width="500px" destroy-on-close>
            <el-form :model="editForm" label-width="100px">
                <el-form-item :label="$t('label.labelKey')" required>
                    <el-input
                        v-model="editForm.labelKey"
                        :placeholder="$t('label.labelKeyPlaceholder')"
                        :disabled="isEditMode"
                    />
                </el-form-item>
                <el-form-item :label="$t('label.labelValue')" required>
                    <el-input
                        v-model="editForm.labelValue"
                        :placeholder="$t('label.labelValuePlaceholder')"
                        :disabled="isEditMode"
                    />
                </el-form-item>
                <el-form-item :label="$t('label.color')">
                    <el-color-picker v-model="editForm.extra!.color" />
                </el-form-item>
                <el-form-item :label="$t('label.description')">
                    <el-input v-model="editForm.extra!.description" type="textarea" :rows="3" :placeholder="$t('label.descriptionPlaceholder')" />
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="editVisible = false">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" @click="onSave">{{ $t('common.save') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { reactive, ref, useTemplateRef } from 'vue';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg } from '@/hooks/useI18n';
import { useI18nDeleteConfirm } from '@/hooks/useI18n';
import { refreshLabelColors } from '@/hooks/useLabelFill';
import { labelApi } from './api';
import type { LabelVO, LabelForm } from './types';

const searchItems = [SearchItem.input('labelKey', 'label.labelKey').withPlaceholder('label.labelKeyPlaceholder')];

const columns = [
    TableColumn.new('labelKey', 'label.labelKey'),
    TableColumn.new('labelValue', 'label.labelValue').isSlot('color'),
    TableColumn.new('description', 'label.description').isSlot(),
    TableColumn.new('createTime', 'common.createTime').isTime(),
    TableColumn.new('action', 'common.operation').isSlot().fixedRight().setMinWidth(150).alignCenter(),
];

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');
const query = reactive({
    labelKey: '',
});

const editVisible = ref(false);
const editTitle = ref('label.addLabel');
const isEditMode = ref(false);
const editForm = reactive<LabelForm>({
    id: undefined,
    labelKey: '',
    labelValue: '',
    extra: { color: '', description: '' },
});

const resetForm = () => {
    editForm.id = undefined;
    editForm.labelKey = '';
    editForm.labelValue = '';
    editForm.extra = { color: '', description: '' };
};

const onAdd = () => {
    editTitle.value = 'label.addLabel';
    isEditMode.value = false;
    resetForm();
    editVisible.value = true;
};

const onEdit = (row: LabelVO) => {
    editTitle.value = 'label.editLabel';
    isEditMode.value = true;
    editForm.id = row.id;
    editForm.labelKey = row.labelKey;
    editForm.labelValue = row.labelValue;
    editForm.extra = {
        color: row.extra?.color || '',
        description: row.extra?.description || '',
    };
    editVisible.value = true;
};

const onSave = async () => {
    if (!editForm.labelKey || !editForm.labelValue) {
        Msg.warning('label.labelRequired');
        return;
    }
    if (editForm.id) {
        await labelApi.update.request(editForm);
    } else {
        await labelApi.save.request(editForm);
    }
    Msg.saveSuccess();
    editVisible.value = false;
    refreshLabelColors();
    pageTableRef.value?.search();
};

const onDelete = async (row: LabelVO) => {
    await useI18nDeleteConfirm(`${row.labelKey}=${row.labelValue}`);
    await labelApi.del.request({ id: row.id });
    Msg.deleteSuccess();
    refreshLabelColors();
    pageTableRef.value?.search();
};
</script>

<style lang="scss" scoped>
</style>
