<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
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

            <template #action="{ data }">
                <el-button v-auth="perms.inst_save" @click="editEntity(data)" link type="primary">{{ $t('common.edit') }}</el-button>
            </template>
        </page-table>

        <milvus-edit @val-change="search()" :title="editDialog.title" v-model:visible="editDialog.visible" v-model:data="editDialog.data" />
    </div>
</template>

<script setup lang="ts">
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { useEditDialog, useRouteTagPath } from '@/hooks/useResourceForm';
import { defineAsyncComponent, ref, useTemplateRef } from 'vue';
import ResourceAuthCert from '../component/ResourceAuthCert.vue';
import TagCodePath from '../component/TagCodePath.vue';
import { milvusApi, perms } from './api';
import type { Milvus } from './types';
import type { PageResult } from '@/types/common';

const MilvusEdit = defineAsyncComponent(() => import('./MilvusEdit.vue'));

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');
const checkRouteTagPath = useRouteTagPath();
const { editDialog, editEntity } = useEditDialog<Milvus>('Milvus');

const query = ref({
    pageNum: 1,
    pageSize: 0,
});

const selectionData = ref<Milvus[]>([]);

const searchItems = [SearchItem.input('keyword', 'common.keyword').withPlaceholder('db.keywordPlaceholder')];

const columns = ref([
    TableColumn.new('name', 'common.name').isSlot('name').setAddWidth(15),
    TableColumn.new('host', 'milvus.host').setMinWidth(200),
    TableColumn.new('authCerts[0].username', 'db.acName').isSlot('authCert').setAddWidth(10),
    TableColumn.new('createTime', 'common.createTime').setMinWidth(180).isTime(),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('code', 'Code').setMinWidth(150),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(100).fixedRight().alignCenter(),
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
    try {
        await useI18nDeleteConfirm(names);
    } catch {
        return; // 用户取消
    }
    await milvusApi.delete.request({ ids: records.map((r) => r.id).join(',') });
    Msg.deleteSuccess();
    search();
};

const search = () => {
    pageTableRef.value?.search();
};

defineExpose({ search });
</script>
