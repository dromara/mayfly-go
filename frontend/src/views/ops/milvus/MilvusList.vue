<template>
    <div class="h-full">
        <data-table
            ref="dataTableRef"
            :page-api="milvusApi.list"
            :before-query-fn="checkRouteTagPath"
            :data-handler-fn="handleData"
            :searchItems="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="columns"
            lazy
        >
            <template #tableHeader>
                <el-button v-auth="perms.inst_save" type="primary" icon="plus" @click="editEntity()" plain>{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.inst_del" type="danger" icon="delete" :disabled="selectionData.length < 1" @click="onDelete" plain>
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #name="{ data }">
                <TagCodePath :code="data.code" show-popover />
                {{ data.name }}
            </template>

            <template #authCert="{ data }">
                <ResourceAuthCert v-model:select-auth-cert="data.selectAuthCert" :auth-certs="data.authCerts" />
            </template>

            <!-- 行内操作由表格统一渲染（见 columns 里的 setActions），列宽与按钮同源 -->
        </data-table>

        <milvus-edit @val-change="search()" :title="editDialog.title" v-model:visible="editDialog.visible" v-model:data="editDialog.data" />
    </div>
</template>

<script setup lang="ts">
import { TableColumn } from '@/components/data-table';
import DataTable from '@/components/data-table/DataTable.vue';
import { SearchItem } from '@/components/data-table/SearchForm';
import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { useEditDialog, useRouteTagPath } from '@/hooks/useResourceForm';
import { defineAsyncComponent, ref, useTemplateRef } from 'vue';
import ResourceAuthCert from '../component/ResourceAuthCert.vue';
import TagCodePath from '../component/TagCodePath.vue';
import { milvusApi, perms } from './api';
import type { Milvus } from './types';
import type { PageResult } from '@/types/common';

const MilvusEdit = defineAsyncComponent(() => import('./MilvusEdit.vue'));

const dataTableRef = useTemplateRef<InstanceType<typeof DataTable>>('dataTableRef');
const checkRouteTagPath = useRouteTagPath();
const { editDialog, editEntity } = useEditDialog<Milvus>('Milvus');

const query = ref({
    pageNum: 1,
    pageSize: 0,
});

const selectionData = ref<Milvus[]>([]);

const searchItems = [SearchItem.input('keyword', 'common.keyword').withPlaceholder('db.keywordPlaceholder')];

const columns = ref([
    TableColumn.new('name', 'common.name').isSlot('name'),
    TableColumn.new('host', 'milvus.host').setMinWidth(200),
    // 授权凭证列插槽渲染 el-select（下拉箭头+padding+边框回填量不到）：+x 预留控件占位
    TableColumn.new('authCerts[0].username', 'db.acName').isSlot('authCert').setAddWidth(25),
    TableColumn.new('createTime', 'common.createTime').setMinWidth(180).isTime(),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('code', 'Code').setMinWidth(150),
    TableColumn.new('action', 'common.operation')
        .setActions<Milvus>([{ key: 'edit', label: 'common.edit', type: 'primary', perm: perms.inst_save, onClick: (row) => editEntity(row) }])
        .fixedRight()
        .alignCenter(),
]);

const handleData = (res: PageResult<Milvus>) => {
    const dataList = res.list;
    // 赋值授权凭证
    for (let x of dataList) {
        x.selectAuthCert = x.authCerts?.[0];
    }
    return res;
};

const onDelete = async () => {
    const records = selectionData.value || [];
    if (records.length === 0) {
        Msg.warning('milvus.pleaseSelectDelete');
        return;
    }
    const names = records.map((r) => r.name).join('、');
    if (!(await useI18nDeleteConfirm(names))) {
        // 取消或关掉弹窗：不继续后续操作
        return;
    }
    await milvusApi.delete.request({ ids: records.map((r) => r.id).join(',') });
    Msg.deleteSuccess();
    search();
};

const search = () => {
    dataTableRef.value?.search();
};

defineExpose({ search });
</script>
