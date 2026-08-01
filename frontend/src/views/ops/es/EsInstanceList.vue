<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="esApi.instances"
            :data-handler-fn="handleData"
            :searchItems="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="state.selectionData"
            :columns="columns"
            lazy
        >
            <template #tableHeader>
                <el-button v-auth="perms.saveInstance" type="primary" icon="plus" @click="editInstance(false)" plain>{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.delInstance" :disabled="selectionData.length < 1" @click="deleteInstance()" type="danger" icon="delete" plain>
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
                <el-button @click="showInfo(data)" link>{{ $t('common.detail') }}</el-button>
                <el-button v-if="actionBtns[perms.saveInstance]" @click="editInstance(data)" type="primary" link>{{ $t('common.edit') }}</el-button>
            </template>
        </page-table>

        <el-dialog v-model="infoDialog.visible" :title="$t('common.detail')">
            <el-descriptions v-if="infoDialog.data" :column="3" border>
                <el-descriptions-item :span="2" :label="$t('common.name')">{{ infoDialog.data.name }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="ID">{{ infoDialog.data.id }}</el-descriptions-item>
                <el-descriptions-item :span="2" label="Host">{{ infoDialog.data.host }}</el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('es.port')">{{ infoDialog.data.port }}</el-descriptions-item>

                <el-descriptions-item :span="3" :label="$t('common.remark')">{{ infoDialog.data.remark }}</el-descriptions-item>

                <el-descriptions-item :span="3" :label="$t('machine.sshTunnel')">
                    {{ infoDialog.data.sshTunnelMachineId > 0 ? $t('common.yes') : $t('common.no') }}
                </el-descriptions-item>

                <el-descriptions-item :span="2" :label="$t('common.createTime')">{{ formatDate(infoDialog.data.createTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('common.creator')">{{ infoDialog.data.creator }}</el-descriptions-item>

                <el-descriptions-item :span="2" :label="$t('common.updateTime')">{{ formatDate(infoDialog.data.updateTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('common.modifier')">{{ infoDialog.data.modifier }}</el-descriptions-item>
            </el-descriptions>
        </el-dialog>

        <instance-edit
            @val-change="search()"
            :title="instanceEditDialog.title"
            v-model:visible="instanceEditDialog.visible"
            v-model:data="instanceEditDialog.data"
        />
    </div>
</template>

<script lang="ts" setup>
import { formatDate } from '@/common/utils/format';
import { hasPerms } from '@/components/auth/auth';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import { defineAsyncComponent, onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';
import ResourceAuthCert from '../component/ResourceAuthCert.vue';
import TagCodePath from '../component/TagCodePath.vue';
import { esApi } from './api';
import type { EsInstance } from './types';
import type { PageResult } from '@/types/common';

const InstanceEdit = defineAsyncComponent(() => import('./EsInstanceEdit.vue'));

const props = defineProps({
    lazy: {
        type: [Boolean],
        default: false,
    },
});

const perms = {
    saveInstance: 'es:instance:save',
    delInstance: 'es:instance:del',
};

const searchItems = [SearchItem.input('keyword', 'common.keyword').withPlaceholder('es.keywordPlaceholder')];

const columns = ref([
    TableColumn.new('name', 'common.name').isSlot('name').setAddWidth(15),
    TableColumn.new('host', 'host:port').setFormatFunc((data: EsInstance) => `${data.host}:${data.port}`),
    TableColumn.new('authCerts[0].username', 'es.acName').isSlot('authCert').setAddWidth(10),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('code', 'common.code'),
]);

// 该用户拥有的的操作列按钮权限
const actionBtns: Record<string, boolean> = hasPerms(Object.values(perms));
const actionColumn = TableColumn.new('action', 'common.operation').isSlot().setMinWidth(180).fixedRight().noShowOverflowTooltip().alignCenter();
const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');

const state = reactive({
    row: {},
    dbId: 0,
    db: '',
    /**
     * 选中的数据
     */
    selectionData: [],
    /**
     * 查询条件
     */
    query: {
        name: null,
        tagPath: '',
        pageNum: 1,
        pageSize: 0,
    },
    infoDialog: {
        visible: false,
        data: null as EsInstance | null,
    },
    instanceEditDialog: {
        visible: false,
        data: null as EsInstance | null,
        title: '',
    },
});

const { selectionData, query, infoDialog, instanceEditDialog } = toRefs(state);

onMounted(async () => {
    if (Object.keys(actionBtns).length > 0) {
        columns.value.push(actionColumn);
    }
    if (!props.lazy) {
        search();
    }
});

const search = (tagPath: string = '') => {
    if (tagPath) {
        state.query.tagPath = tagPath;
    }
    pageTableRef.value?.search();
};

const handleData = (res: PageResult<EsInstance>) => {
    const dataList = res.list;
    // 赋值授权凭证
    for (let x of dataList) {
        if (x.authCerts && x.authCerts.length > 0) {
            x.selectAuthCert = x.authCerts[0];
        }
    }
    return res;
};

const showInfo = (info: EsInstance) => {
    state.infoDialog.data = info;
    state.infoDialog.visible = true;
};

const editInstance = async (data: EsInstance | false) => {
    if (!data) {
        state.instanceEditDialog.data = null;
        state.instanceEditDialog.title = useI18nCreateTitle('es.instance');
    } else {
        state.instanceEditDialog.data = data;
        state.instanceEditDialog.title = useI18nEditTitle('es.instance');
    }
    state.instanceEditDialog.visible = true;
};

const deleteInstance = async () => {
    try {
        await useI18nDeleteConfirm(state.selectionData.map((x: EsInstance) => x.name).join('、'));
        await esApi.deleteInstance.request({ id: state.selectionData.map((x: EsInstance) => x.id).join(',') });
        Msg.deleteSuccess();
        search();
    } catch (err) {
        //
    }
};

defineExpose({ search });
</script>
<style lang="scss"></style>
