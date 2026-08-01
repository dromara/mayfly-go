<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="dockerApi.page"
            :before-query-fn="checkRouteTagPath"
            :searchItems="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="columns"
            lazy
        >
            <template #tableHeader>
                <el-button v-auth="'container:save'" type="primary" icon="plus" @click="editContainerConf(false)" plain>{{ $t('common.create') }}</el-button>
                <el-button v-auth="'container:del'" type="danger" icon="delete" :disabled="selectionData.length < 1" @click="deleteConf" plain>
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #name="{ data }">
                <TagCodePath :code="data.code" show-popover />
                {{ data.name }}
            </template>

            <template #action="{ data }">
                <el-button @click="showDetail(data)" link>{{ $t('common.detail') }}</el-button>
                <el-button v-auth="'container:save'" type="primary" link @click="editContainerConf(data)">{{ $t('common.edit') }}</el-button>
            </template>
        </page-table>

        <el-dialog v-if="detailDialog.visible" v-model="detailDialog.visible">
            <el-descriptions v-if="detailDialog.data" :title="$t('common.detail')" :column="3" border>
                <el-descriptions-item :span="1.5" label="id">{{ detailDialog.data.id }}</el-descriptions-item>
                <el-descriptions-item :span="1.5" :label="$t('common.name')">{{ detailDialog.data.name }}</el-descriptions-item>

                <el-descriptions-item :span="3" :label="$t('tag.relateTag')"><TagCodePath :code="detailDialog.data.code" /></el-descriptions-item>

                <el-descriptions-item :span="3" :label="$t('docker.addr')">{{ detailDialog.data.addr }}</el-descriptions-item>

                <el-descriptions-item :span="3" :label="$t('common.remark')">{{ detailDialog.data.remark }}</el-descriptions-item>

                <el-descriptions-item :span="2" :label="$t('common.createTime')">{{ formatDate(detailDialog.data.createTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('common.creator')">{{ detailDialog.data.creator }}</el-descriptions-item>

                <el-descriptions-item :span="2" :label="$t('common.updateTime')">{{ formatDate(detailDialog.data.updateTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('common.modifier')">{{ detailDialog.data.modifier }}</el-descriptions-item>
            </el-descriptions>
        </el-dialog>

        <ContainerConfEdit
            @val-change="search()"
            :title="containerConfEditDialog.title"
            v-model:visible="containerConfEditDialog.visible"
            v-model:container="containerConfEditDialog.data"
        ></ContainerConfEdit>
    </div>
</template>

<script lang="ts" setup>
import { formatDate } from '@/common/utils/format';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import TagCodePath from '@/views/ops/component/TagCodePath.vue';
import { defineAsyncComponent, onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';
import { useRoute } from 'vue-router';
import { dockerApi } from './api';
import type { Container } from './types';

const ContainerConfEdit = defineAsyncComponent(() => import('./CotainerConfEdit.vue'));

const props = defineProps({
    lazy: {
        type: [Boolean],
        default: false,
    },
});

const route = useRoute();
const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');

const searchItems = [SearchItem.input('keyword', 'common.keyword').withPlaceholder('redis.keywordPlaceholder')];

const columns = ref([
    TableColumn.new('name', 'common.name').isSlot('name').setAddWidth(15),
    TableColumn.new('addr', 'docker.addr'),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('code', 'common.code'),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(200).fixedRight().alignCenter(),
]);

const state = reactive({
    selectionData: [],
    query: {
        tagPath: '',
        pageNum: 1,
        pageSize: 0,
    },
    detailDialog: {
        visible: false,
        data: null as Container | null,
    },
    containerConfEditDialog: {
        visible: false,
        data: null as Container | null,
        title: '',
    },
});

const { selectionData, query, detailDialog, containerConfEditDialog } = toRefs(state);

onMounted(() => {
    if (!props.lazy) {
        search();
    }
});

const checkRouteTagPath = (query: Record<string, unknown>) => {
    if (route.query.tagPath) {
        query.tagPath = route.query.tagPath as string;
    }
    return query;
};

const showDetail = (detail: Container) => {
    state.detailDialog.data = detail;
    state.detailDialog.visible = true;
};

const deleteConf = async () => {
    try {
        await useI18nDeleteConfirm(state.selectionData.map((x: Container) => x.name).join('、'));
        await dockerApi.delConf.request({ id: state.selectionData.map((x: Container) => x.id).join(',') });
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

const editContainerConf = async (data: Container | false) => {
    if (!data) {
        state.containerConfEditDialog.data = null;
        state.containerConfEditDialog.title = useI18nCreateTitle('docker.containerConf');
    } else {
        state.containerConfEditDialog.data = data;
        state.containerConfEditDialog.title = useI18nEditTitle('docker.containerConf');
    }
    state.containerConfEditDialog.visible = true;
};

defineExpose({ search });
</script>

<style></style>
