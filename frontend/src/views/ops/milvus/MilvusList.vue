<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="milvusApi.list"
            :data-handler-fn="handleData"
            :searchItems="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="columns"
            lazy
        >
            <template #tableHeader>
                <el-button v-auth="perms.inst_save" type="primary" icon="plus" @click="editMilvus()" plain>{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.inst_del" type="danger" icon="delete" :disabled="selectionData.length < 1" @click="deleteMilvus" plain>
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
                <el-button v-auth="perms.inst_save" @click="editMilvus(data)" link type="primary">{{ $t('common.edit') }}</el-button>
            </template>
        </page-table>

        <milvus-edit @val-change="search()" :title="milvusEditDialog.title" v-model:visible="milvusEditDialog.visible" v-model:milvus="milvusEditDialog.data" />
    </div>
</template>

<script setup lang="ts">
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import { defineAsyncComponent, ref, useTemplateRef } from 'vue';
import ResourceAuthCert from '../component/ResourceAuthCert.vue';
import TagCodePath from '../component/TagCodePath.vue';
import { milvusApi, perms } from './api';
import type { Milvus } from './types';
import type { PageResult } from '@/types/common';

const MilvusEdit = defineAsyncComponent(() => import('./MilvusEdit.vue'));

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');

const query = ref({
    pageNum: 1,
    pageSize: 0,
});

const selectionData = ref([]);

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

const milvusEditDialog = ref({
    title: '',
    visible: false,
    data: null as Milvus | null,
});

const editMilvus = (data?: Milvus) => {
    milvusEditDialog.value = {
        title: data ? useI18nEditTitle('Milvus') : useI18nCreateTitle('Milvus'),
        visible: true,
        data: data || null,
    };
};

const handleData = (res: PageResult<Milvus>) => {
    const dataList = res.list;
    // 赋值授权凭证
    for (let x of dataList) {
        x.selectAuthCert = x.authCerts?.[0];
    }
    return res;
};

const deleteMilvus = async () => {
    const records = selectionData.value || [];
    if (records.length === 0) {
        Msg.warning('请选择要删除的数据');
        return;
    }
    const ids = records.map((r: Milvus) => r.id).join(',');

    await useI18nDeleteConfirm('Milvus: ' + ids);
    milvusApi.delete.request({ ids }).then(() => {
        Msg.deleteSuccess();
        search();
    });
};

const search = () => {
    pageTableRef.value?.search();
};

defineExpose({ search });
</script>
