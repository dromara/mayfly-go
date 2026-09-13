<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="dbApi.instances"
            :data-handler-fn="handleData"
            :searchItems="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="columns"
            lazy
        >
            <template #tableHeader>
                <el-button v-auth="perms.saveInstance" type="primary" icon="plus" @click="editInstance()" plain>{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.delInstance" :disabled="selectionData.length < 1" @click="deleteInstance" type="danger" icon="delete" plain>
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

            <template #type="{ data }">
                <el-tooltip :content="getDbDialect(data.type).getInfo().name" placement="top">
                    <SvgIcon :name="getDbDialect(data.type).getInfo().icon" :size="20" />
                </el-tooltip>
            </template>

            <template #action="{ data }">
                <el-button @click="showInfo(data)" link>{{ $t('common.detail') }}</el-button>
                <el-button v-if="actionBtns[perms.saveInstance]" @click="editInstance(data)" type="primary" link>{{ $t('common.edit') }}</el-button>
                <el-button v-if="actionBtns[perms.saveDb]" @click="editDb(data)" type="primary" link>{{ $t('db.dbManage') }}</el-button>
            </template>
        </page-table>

        <el-dialog v-model="infoDialog.visible" :title="$t('common.detail')">
            <el-descriptions v-if="infoDialog.data" :column="3" border>
                <el-descriptions-item :span="2" :label="$t('common.name')">{{ infoDialog.data.name }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="ID">{{ infoDialog.data.id }}</el-descriptions-item>
                <el-descriptions-item :span="2" label="Host">{{ infoDialog.data.host }}</el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('db.port')">{{ infoDialog.data.port }}</el-descriptions-item>

                <el-descriptions-item :span="1" :label="$t('common.type')">{{ infoDialog.data.type }}</el-descriptions-item>

                <el-descriptions-item :span="3" :label="$t('db.connParam')">{{ infoDialog.data.params }}</el-descriptions-item>
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

        <!-- 必须写成 search()：search 的首个形参是标签树传来的 tagPath，若用 @val-change="search"
             会把保存的表单对象当 tagPath 塞进 query.tagPath，此后该列表的查询条件一直被污染 -->
        <instance-edit
            @val-change="search()"
            :title="instanceEditDialog.title"
            v-model:visible="instanceEditDialog.visible"
            v-model:data="instanceEditDialog.data"
        ></instance-edit>

        <DbList :title="dbEditDialog.title" v-model:visible="dbEditDialog.visible" :instance="dbEditDialog.instance" />
    </div>
</template>

<script lang="ts" setup>
import { formatDate } from '@/common/utils/format';
import { hasPerms } from '@/components/auth/auth';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import SvgIcon from '@/components/svg-icon/index.vue';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import { useRouteTagPath } from '@/hooks/useResourceForm';
import { defineAsyncComponent, onMounted, ref, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import ResourceAuthCert from '../../component/ResourceAuthCert.vue';
import TagCodePath from '../../component/TagCodePath.vue';
import { dbApi } from '../api';
import { getDbDialect } from '../dialect';
import type { DbInstance } from '../types';
import type { PageResult } from '@/types/common';

const InstanceEdit = defineAsyncComponent(() => import('./InstanceEdit.vue'));
const DbList = defineAsyncComponent(() => import('./DbList.vue'));

const { t } = useI18n();

const props = defineProps({
    lazy: {
        type: [Boolean],
        default: false,
    },
});

const perms = {
    saveInstance: 'db:instance:save',
    delInstance: 'db:instance:del',
    saveDb: 'db:save',
};

const searchItems = [SearchItem.input('keyword', 'common.keyword').withPlaceholder('db.keywordPlaceholder')];

const columns = ref([
    TableColumn.new('name', 'common.name').isSlot('name').setAddWidth(15),
    TableColumn.new('type', 'common.type').isSlot().setAddWidth(-15).alignCenter(),
    TableColumn.new('host', 'host:port').setFormatFunc((data: DbInstance) => `${data.host}:${data.port}`),
    TableColumn.new('authCerts[0].username', 'db.acName').isSlot('authCert').setAddWidth(10),
    TableColumn.new('params', 'db.connParam'),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('code', 'common.code'),
]);

// 该用户拥有的的操作列按钮权限
const actionBtns: Record<string, boolean> = hasPerms(Object.values(perms));
const actionColumn = TableColumn.new('action', 'common.operation').isSlot().setMinWidth(180).fixedRight().noShowOverflowTooltip().alignCenter();
const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');
const checkRouteTagPath = useRouteTagPath();

const selectionData = ref<DbInstance[]>([]);
const query = ref({
    name: null,
    tagPath: '',
    pageNum: 1,
    pageSize: 0,
});

const infoDialog = ref({
    visible: false,
    data: null as DbInstance | null,
});

const instanceEditDialog = ref({
    visible: false,
    data: null as DbInstance | null,
    title: '',
});

const dbEditDialog = ref({
    visible: false,
    instance: {} as DbInstance,
    title: '',
});

onMounted(async () => {
    if (Object.keys(actionBtns).length > 0) {
        columns.value.push(actionColumn);
    }
    if (!props.lazy) {
        search();
    }
});

const handleData = (res: PageResult<DbInstance>) => {
    const dataList = res.list;
    for (let x of dataList) {
        x.selectAuthCert = x.authCerts?.[0];
    }
    return res;
};

const showInfo = (info: DbInstance) => {
    infoDialog.value.data = info;
    infoDialog.value.visible = true;
};

const editInstance = (data: DbInstance | false = false) => {
    instanceEditDialog.value.data = data || null;
    instanceEditDialog.value.title = data ? useI18nEditTitle('db.dbInst') : useI18nCreateTitle('db.dbInst');
    instanceEditDialog.value.visible = true;
};

const deleteInstance = async () => {
    const records = selectionData.value || [];
    if (records.length === 0) return;
    try {
        await useI18nDeleteConfirm(records.map((x) => x.name).join('、'));
    } catch {
        return; // 用户取消
    }
    await dbApi.deleteInstance.request({ id: records.map((x) => x.id).join(',') });
    Msg.deleteSuccess();
    search();
};

const editDb = (data: DbInstance) => {
    dbEditDialog.value.instance = data;
    dbEditDialog.value.title = t('db.manageDbTitle', { instName: data.name });
    dbEditDialog.value.visible = true;
};

const search = (tagPath?: string) => {
    // tagPath 为 undefined 时（如"所有资源"节点），清空过滤条件查询全部；为具体值时按标签过滤
    query.value.tagPath = tagPath ?? '';
    pageTableRef.value?.search();
};

defineExpose({ search });
</script>
<style lang="scss"></style>
