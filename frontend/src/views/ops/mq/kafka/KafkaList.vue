<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="mqApi.kafkaList"
            :before-query-fn="checkRouteTagPath"
            :search-items="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="columns"
            lazy
        >
            <template #tableHeader>
                <el-button v-auth="'mq:kafka:save'" type="primary" icon="plus" @click="editEntity()" plain>{{ $t('common.create') }}</el-button>
                <el-button v-auth="'mq:kafka:del'" type="danger" icon="delete" :disabled="selectionData.length < 1" @click="onDelete" plain>
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #name="{ data }">
                <TagCodePath :code="data.code" show-popover />
                {{ data.name }}
            </template>

            <template #action="{ data }">
                <el-button v-auth="'mq:kafka:save'" @click="editEntity(data)" link type="primary">{{ $t('common.edit') }}</el-button>
            </template>
        </page-table>

        <kafka-edit @val-change="search()" :title="editDialog.title" v-model:visible="editDialog.visible" v-model:data="editDialog.data" />
    </div>
</template>

<script lang="ts" setup>
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { useEditDialog, useRouteTagPath } from '@/hooks/useResourceForm';
import { mqApi } from '@/views/ops/mq/api';
import type { Kafka } from '@/views/ops/mq/types';
import { defineAsyncComponent, onMounted, ref, useTemplateRef } from 'vue';
import TagCodePath from '../../component/TagCodePath.vue';

const KafkaEdit = defineAsyncComponent(() => import('./KafkaEdit.vue'));

const props = defineProps({
    lazy: {
        type: Boolean,
        default: false,
    },
});

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');
const checkRouteTagPath = useRouteTagPath();
const { editDialog, editEntity } = useEditDialog<Kafka>('mq.kafka');

const query = ref({
    pageNum: 1,
    pageSize: 0,
    tagPath: '',
});

const selectionData = ref<Kafka[]>([]);

const searchItems = [SearchItem.input('keyword', 'common.keyword').withPlaceholder('mq.kafka.keywordPlaceholder')];

const columns = [
    TableColumn.new('name', 'common.name').isSlot('name').setAddWidth(15),
    TableColumn.new('hosts', 'Hosts'),
    TableColumn.new('username', 'mq.kafka.username'),
    TableColumn.new('password', 'common.password'),
    TableColumn.new('createTime', 'common.createTime').isTime(),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('code', 'common.code'),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(170).fixedRight().alignCenter(),
];

onMounted(() => {
    if (!props.lazy) {
        search();
    }
});

const onDelete = async () => {
    const records = selectionData.value || [];
    if (records.length === 0) return;
    try {
        await useI18nDeleteConfirm(records.map((x) => x.name).join('、'));
    } catch {
        return; // 用户取消
    }
    await mqApi.kafkaDel.request({ id: records.map((x) => x.id).join(',') });
    Msg.deleteSuccess();
    search();
};

const search = (tagPath?: string) => {
    // tagPath 为 undefined 时（如"所有资源"节点），清空过滤条件查询全部；为具体值时按标签过滤
    query.value.tagPath = tagPath ?? '';
    pageTableRef.value?.search();
};

defineExpose({ search });
</script>
