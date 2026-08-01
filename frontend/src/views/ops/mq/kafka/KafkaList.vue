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
                <el-button v-auth="'mq:kafka:save'" type="primary" icon="plus" @click="editKafka(false)" plain>{{ $t('common.create') }}</el-button>
                <el-button v-auth="'mq:kafka:del'" type="danger" icon="delete" :disabled="selectionData.length < 1" @click="deleteKafka" plain>
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #name="{ data }">
                <TagCodePath :code="data.code" show-popover />
                {{ data.name }}
            </template>

            <template #action="{ data }">
                <el-button v-auth="'mq:kafka:save'" @click="editKafka(data)" link type="primary">{{ $t('common.edit') }}</el-button>
            </template>
        </page-table>

        <kafka-edit @val-change="search()" :title="kafkaEditDialog.title" v-model:visible="kafkaEditDialog.visible" v-model:kafka="kafkaEditDialog.data" />
    </div>
</template>

<script lang="ts" setup>
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import { mqApi } from '@/views/ops/mq/api';
import type { Kafka } from '@/views/ops/mq/types';
import { defineAsyncComponent, onMounted, reactive, toRefs, useTemplateRef } from 'vue';
import { useRoute } from 'vue-router';
import TagCodePath from '../../component/TagCodePath.vue';

const KafkaEdit = defineAsyncComponent(() => import('./KafkaEdit.vue'));

const props = defineProps({
    lazy: {
        type: [Boolean],
        default: false,
    },
});

const route = useRoute();
const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');

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

const state = reactive({
    dbOps: {
        dbId: 0,
        db: '',
    },
    selectionData: [],
    query: {
        pageNum: 1,
        pageSize: 0,
        tagPath: '',
    },
    kafkaEditDialog: {
        visible: false,
        data: null as Kafka | null,
        title: '',
    },
});

const { selectionData, query, kafkaEditDialog } = toRefs(state);

const checkRouteTagPath = (query: Record<string, unknown>) => {
    if (route.query.tagPath) {
        query.tagPath = route.query.tagPath as string;
    }
    return query;
};

const deleteKafka = async () => {
    try {
        await useI18nDeleteConfirm(state.selectionData.map((x: Kafka) => x.name).join('、'));
        await mqApi.kafkaDel.request({ id: state.selectionData.map((x: Kafka) => x.id).join(',') });
        Msg.deleteSuccess();
        search();
    } catch (err) {
        //
    }
};

const search = async (tagPath: string = '') => {
    if (tagPath) {
        state.query.tagPath = tagPath;
    }
    pageTableRef.value?.search();
};

const editKafka = async (data: Kafka | false) => {
    if (!data) {
        state.kafkaEditDialog.data = null;
        state.kafkaEditDialog.title = useI18nCreateTitle('Kafka');
    } else {
        state.kafkaEditDialog.data = data;
        state.kafkaEditDialog.title = useI18nEditTitle('Kafka');
    }
    state.kafkaEditDialog.visible = true;
};

onMounted(() => {
    if (!props.lazy) {
        search();
    }
});

defineExpose({ search });
</script>

<style></style>
