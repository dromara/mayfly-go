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
                <el-button type="primary" icon="plus" @click="editContainerConf(false)" plain>{{ $t('common.create') }}</el-button>
                <el-button type="danger" icon="delete" :disabled="selectionData.length < 1" @click="deleteConf" plain>{{ $t('common.delete') }}</el-button>
            </template>

            <template #tagPath="{ data }">
                <resource-tags :tags="data.tags" />
            </template>

            <template #action="{ data }">
                <el-button @click="showDetail(data)" link>{{ $t('common.detail') }}</el-button>
                <el-button type="primary" link @click="editContainerConf(data)">{{ $t('common.edit') }}</el-button>
            </template>
        </page-table>

        <el-dialog v-if="detailDialog.visible" v-model="detailDialog.visible">
            <el-descriptions :title="$t('common.detail')" :column="3" border>
                <el-descriptions-item :span="1.5" label="id">{{ detailDialog.data.id }}</el-descriptions-item>
                <el-descriptions-item :span="1.5" :label="$t('common.name')">{{ detailDialog.data.name }}</el-descriptions-item>

                <el-descriptions-item :span="3" :label="$t('tag.relateTag')"><ResourceTags :tags="detailDialog.data.tags" /></el-descriptions-item>

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
import { dockerApi } from './api';
import { defineAsyncComponent, onMounted, reactive, ref, Ref, toRefs } from 'vue';
import { formatDate } from '@/common/utils/format';
import ResourceTags from '../component/ResourceTags.vue';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { useRoute } from 'vue-router';
import { getTagPathSearchItem } from '../component/tag';
import { SearchItem } from '@/components/pagetable/SearchForm';
import { useI18nCreateTitle, useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nEditTitle } from '@/hooks/useI18n';

const ContainerConfEdit = defineAsyncComponent(() => import('./CotainerConfEdit.vue'));

const props = defineProps({
    lazy: {
        type: [Boolean],
        default: false,
    },
});

const route = useRoute();
const pageTableRef: Ref<any> = ref(null);

const searchItems = [
    SearchItem.input('keyword', 'common.keyword').withPlaceholder('redis.keywordPlaceholder'),
    getTagPathSearchItem(TagResourceTypeEnum.Container.value),
];

const columns = ref([
    TableColumn.new('tags[0].tagPath', 'tag.relateTag').isSlot('tagPath').setAddWidth(20),
    TableColumn.new('name', 'common.name'),
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
        data: null as any,
    },
    containerConfEditDialog: {
        visible: false,
        data: null as any,
        title: '',
    },
});

const { selectionData, query, detailDialog, containerConfEditDialog } = toRefs(state);

onMounted(() => {
    if (!props.lazy) {
        search();
    }
});

const checkRouteTagPath = (query: any) => {
    if (route.query.tagPath) {
        query.tagPath = route.query.tagPath as string;
    }
    return query;
};

const showDetail = (detail: any) => {
    state.detailDialog.data = detail;
    state.detailDialog.visible = true;
};

const deleteConf = async () => {
    try {
        await useI18nDeleteConfirm(state.selectionData.map((x: any) => x.name).join('、'));
        await dockerApi.delConf.request({ id: state.selectionData.map((x: any) => x.id).join(',') });
        useI18nDeleteSuccessMsg();
        search();
    } catch (err) {
        //
    }
};

const search = async (tagPath: string = '') => {
    if (tagPath) {
        state.query.tagPath = tagPath;
    }
    pageTableRef.value.search();
};

const editContainerConf = async (data: any) => {
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
