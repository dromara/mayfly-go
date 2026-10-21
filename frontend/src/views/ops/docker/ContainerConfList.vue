<template>
    <div class="h-full">
        <data-table
            ref="dataTableRef"
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
                <el-button v-auth="'container:save'" type="primary" icon="plus" @click="editEntity()" plain>{{ $t('common.create') }}</el-button>
                <el-button v-auth="'container:del'" type="danger" icon="delete" :disabled="selectionData.length < 1" @click="onDelete" plain>
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #name="{ data }">
                <TagCodePath :code="data.code" show-popover />
                {{ data.name }}
            </template>

            <!-- 行内操作由表格统一渲染（见 columns 里的 setActions），列宽与按钮同源 -->
        </data-table>

        <el-dialog v-if="detailDialog.visible" v-model="detailDialog.visible">
            <el-descriptions v-if="detailDialog.data" :title="$t('common.detail')" :column="3" border label-width="90">
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

        <ContainerConfEdit @val-change="search()" :title="editDialog.title" v-model:visible="editDialog.visible" v-model:data="editDialog.data" />
    </div>
</template>

<script lang="ts" setup>
import { formatDate } from '@/common/utils/format';
import { TableColumn } from '@/components/data-table';
import DataTable from '@/components/data-table/DataTable.vue';
import { SearchItem } from '@/components/data-table/SearchForm';
import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { useEditDialog, useRouteTagPath } from '@/hooks/useResourceForm';
import TagCodePath from '@/views/ops/component/TagCodePath.vue';
import { defineAsyncComponent, onMounted, ref, useTemplateRef } from 'vue';
import { dockerApi } from './api';
import type { Container } from './types';

const ContainerConfEdit = defineAsyncComponent(() => import('./CotainerConfEdit.vue'));

const props = defineProps({
    lazy: {
        type: Boolean,
        default: false,
    },
});

const dataTableRef = useTemplateRef<InstanceType<typeof DataTable>>('dataTableRef');
const checkRouteTagPath = useRouteTagPath();
const { editDialog, editEntity } = useEditDialog<Container>('docker.containerConf');

const query = ref({
    tagPath: '',
    pageNum: 1,
    pageSize: 0,
});

const selectionData = ref<Container[]>([]);
const detailDialog = ref({
    visible: false,
    data: null as Container | null,
});

const searchItems = [SearchItem.input('keyword', 'common.keyword').withPlaceholder('redis.keywordPlaceholder')];

const columns = ref([
    TableColumn.new('name', 'common.name').isSlot('name'),
    TableColumn.new('addr', 'docker.addr'),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('code', 'common.code'),
    TableColumn.new('action', 'common.operation')
        .setActions<Container>([
            { key: 'detail', label: 'common.detail', onClick: (row) => showDetail(row) },
            { key: 'edit', label: 'common.edit', type: 'primary', perm: 'container:save', onClick: (row) => editEntity(row) },
        ])
        .fixedRight()
        .alignCenter(),
]);

onMounted(() => {
    if (!props.lazy) {
        search();
    }
});

const showDetail = (detail: Container) => {
    detailDialog.value.data = detail;
    detailDialog.value.visible = true;
};

const onDelete = async () => {
    const records = selectionData.value || [];
    if (records.length === 0) return;
    if (!(await useI18nDeleteConfirm(records.map((x) => x.name).join('、')))) {
        // 取消或关掉弹窗：不继续后续操作
        return;
    }
    await dockerApi.delConf.request({ id: records.map((x) => x.id).join(',') });
    Msg.deleteSuccess();
    search();
};

const search = (tagPath?: string) => {
    // tagPath 为 undefined 时（如"所有资源"节点），清空过滤条件查询全部；为具体值时按标签过滤
    query.value.tagPath = tagPath ?? '';
    dataTableRef.value?.search();
};

defineExpose({ search });
</script>
