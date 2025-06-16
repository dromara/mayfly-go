<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="dbApi.instances"
            :data-handler-fn="handleData"
            :searchItems="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="state.selectionData"
            :columns="columns"
            lazy
        >
            <template #tableHeader>
                <el-button v-auth="perms.saveInstance" type="primary" icon="plus" @click="editInstance(false)">{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.delInstance" :disabled="selectionData.length < 1" @click="deleteInstance()" type="danger" icon="delete">
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #tagPath="{ data }">
                <ResourceTags :tags="data.tags" />
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
            <el-descriptions :column="3" border>
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
import { defineAsyncComponent, onMounted, reactive, ref, Ref, toRefs } from 'vue';
import { dbApi } from './api';
import { formatDate } from '@/common/utils/format';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import SvgIcon from '@/components/svgIcon/index.vue';
import { getDbDialect } from './dialect';
import { SearchItem } from '@/components/SearchForm';
import ResourceAuthCert from '../component/ResourceAuthCert.vue';
import ResourceTags from '../component/ResourceTags.vue';
import { getTagPathSearchItem } from '../component/tag';
import { TagResourceTypePath } from '@/common/commonEnum';
import { useI18nCreateTitle, useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nEditTitle } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';

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

const searchItems = [SearchItem.input('keyword', 'common.keyword').withPlaceholder('db.keywordPlaceholder'), getTagPathSearchItem(TagResourceTypePath.Db)];

const columns = ref([
    TableColumn.new('tags[0].tagPath', 'tag.relateTag').isSlot('tagPath').setAddWidth(20),
    TableColumn.new('name', 'common.name'),
    TableColumn.new('type', 'common.type').isSlot().setAddWidth(-15).alignCenter(),
    TableColumn.new('host', 'host:port').setFormatFunc((data: any) => `${data.host}:${data.port}`),
    TableColumn.new('authCerts[0].username', 'db.acName').isSlot('authCert').setAddWidth(10),
    TableColumn.new('params', 'db.connParam'),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('code', 'common.code'),
]);

// 该用户拥有的的操作列按钮权限
const actionBtns: any = hasPerms(Object.values(perms));
const actionColumn = TableColumn.new('action', 'common.operation').isSlot().setMinWidth(180).fixedRight().noShowOverflowTooltip().alignCenter();
const pageTableRef: Ref<any> = ref(null);

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
        data: null as any,
    },
    instanceEditDialog: {
        visible: false,
        data: null as any,
        title: '',
    },
    dbEditDialog: {
        visible: false,
        instance: {},
        title: '',
    },
});

const { selectionData, query, infoDialog, instanceEditDialog, dbEditDialog } = toRefs(state);

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
    pageTableRef.value.search();
};

const handleData = (res: any) => {
    const dataList = res.list;
    // 赋值授权凭证
    for (let x of dataList) {
        x.selectAuthCert = x.authCerts[0];
    }
    return res;
};

const showInfo = (info: any) => {
    state.infoDialog.data = info;
    state.infoDialog.visible = true;
};

const editInstance = async (data: any) => {
    if (!data) {
        state.instanceEditDialog.data = null;
        state.instanceEditDialog.title = useI18nCreateTitle('db.dbInst');
    } else {
        state.instanceEditDialog.data = data;
        state.instanceEditDialog.title = useI18nEditTitle('db.dbInst');
    }
    state.instanceEditDialog.visible = true;
};

const deleteInstance = async () => {
    try {
        await useI18nDeleteConfirm(state.selectionData.map((x: any) => x.name).join('、'));
        await dbApi.deleteInstance.request({ id: state.selectionData.map((x: any) => x.id).join(',') });
        useI18nDeleteSuccessMsg();
        search();
    } catch (err) {
        //
    }
};

const editDb = (data: any) => {
    state.dbEditDialog.instance = data;
    state.dbEditDialog.title = t('db.manageDbTitle', { instName: data.name });
    state.dbEditDialog.visible = true;
};

defineExpose({ search });
</script>
<style lang="scss"></style>
