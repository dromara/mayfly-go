<template>
    <div>
        <el-dialog v-if="dialogVisible" :title="title" v-model="dialogVisible" :show-close="true" :before-close="handleClose" width="50%">
            <el-table :data="fileTable" stripe v-loading="loading">
                <el-table-column prop="name" :label="$t('common.name')" min-width="100px">
                    <template #header>
                        <el-button class="ml0" type="primary" circle size="small" icon="Plus" @click="add()"> </el-button>
                        <span class="ml-2">{{ $t('common.name') }}</span>
                    </template>
                    <template #default="scope">
                        <el-input v-model="scope.row.name" :disabled="scope.row.id != null" clearable> </el-input>
                    </template>
                </el-table-column>
                <el-table-column prop="name" :label="$t('common.type')" width="130px">
                    <template #default="scope">
                        <EnumSelect :enums="FileTypeEnum" :disabled="scope.row.id != null" v-model="scope.row.type" />
                    </template>
                </el-table-column>
                <el-table-column prop="path" :label="$t('common.path')" min-width="180" show-overflow-tooltip>
                    <template #default="scope">
                        <el-input v-model="scope.row.path" :disabled="scope.row.id != null" clearable> </el-input>
                    </template>
                </el-table-column>
                <el-table-column :label="$t('common.operation')" min-width="130">
                    <template #default="scope">
                        <el-button v-if="scope.row.id == null" @click="addFiles(scope.row)" type="success" icon="success-filled" plain></el-button>
                        <el-button v-if="scope.row.id != null" @click="getConf(scope.row)" type="primary" icon="tickets" plain></el-button>
                        <el-button v-auth="'machine:file:del'" type="danger" @click="deleteRow(scope.$index, scope.row)" icon="delete" plain></el-button>
                    </template>
                </el-table-column>
            </el-table>
            <el-row class="mt-2" type="flex" justify="end">
                <el-pagination
                    :total="total"
                    layout="prev, pager, next, total, jumper"
                    v-model:current-page="query.pageNum"
                    :page-size="query.pageSize"
                    @current-change="handlePageChange"
                >
                </el-pagination>
            </el-row>
        </el-dialog>

        <el-drawer
            :append-to-body="false"
            resizable
            destroy-on-close
            :title="fileDialog.title"
            v-model="fileDialog.visible"
            :close-on-click-modal="false"
            size="70%"
            header-class="!mb-0"
        >
            <machine-file
                :title="fileDialog.title"
                :machine-id="machineId ?? undefined"
                :auth-cert-name="props.authCertName"
                :file-id="fileDialog.fileId"
                :path="fileDialog.path"
                :protocol="protocol"
            />
        </el-drawer>

        <machine-file-content
            :title="fileContent.title"
            v-model:visible="fileContent.contentVisible"
            :machine-id="machineId ?? undefined"
            :auth-cert-name="props.authCertName"
            :file-id="fileContent.fileId"
            :path="fileContent.path"
        />
    </div>
</template>

<script lang="ts" setup>
import EnumSelect from '@/components/enum-select/EnumSelect.vue';
import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import {defineAsyncComponent, onMounted, reactive, toRefs, watch} from 'vue';
import { machineApi } from '../api';
import { FileTypeEnum } from '../enums';
import type { MachineFileVO } from '../types';

const MachineFile = defineAsyncComponent(() => import('./MachineFile.vue'));
const MachineFileContent = defineAsyncComponent(() => import('./MachineFileContent.vue'));

const props = defineProps({
    protocol: { type: Number, default: 1 },
    authCertName: { type: String },
    title: { type: String },
    openFileManager: { type: Boolean, default: true }, // 是否打开文件管理器
});

const dialogVisible = defineModel<boolean>('visible', { default: false });
const machineId = defineModel<number | null>('machineId');

const emit = defineEmits(['cancel', 'select']);

const addFile = machineApi.addConf;
const delFile = machineApi.delConf;
const files = machineApi.files;

const state = reactive({
    query: {
        id: 0,
        pageNum: 1,
        pageSize: 8,
    },
    loading: false,
    form: {
        id: null,
        type: null,
        name: '',
        remark: '',
    },
    total: 0,
    fileTable: [] as MachineFileVO[],
    fileDialog: {
        visible: false,
        protocol: 1,
        title: '',
        fileId: 0,
        path: '',
    },
    fileContent: {
        title: '',
        fileId: 0,
        contentVisible: false,
        path: '',
    },
});

const { loading, query, total, fileTable, fileDialog, fileContent } = toRefs(state);

watch(machineId, async (newValue) => {
    if (newValue && dialogVisible.value) {
        await getFiles();
    }
});

const getFiles = async () => {
    try {
        state.query.id = machineId.value || 0;
        if (!state.query.id){
            return
        }
        state.loading = true;
        const res = await files.request(state.query);
        state.fileTable = res.list || [];
        state.total = res.total;
    } finally {
        state.loading = false;
    }
};

onMounted(getFiles)

const handlePageChange = (curPage: number) => {
    state.query.pageNum = curPage;
    getFiles();
};

const add = () => {
    // 往数组头部添加元素
    state.fileTable = [{} as MachineFileVO].concat(state.fileTable);
};

const addFiles = async (row: MachineFileVO) => {
    row.machineId = machineId.value ?? 0;
    await addFile.request(row);
    Msg.saveSuccess();
    getFiles();
};

const deleteRow = async (idx: number, row: MachineFileVO) => {
    if (row.id) {
        await useI18nDeleteConfirm(row.name);
        // 删除配置文件
        await delFile.request({
            machineId: machineId.value,
            id: row.id,
        });
        getFiles();
    } else {
        state.fileTable.splice(idx, 1);
    }
};

const getConf = async (row: MachineFileVO) => {
    if (row.type != 1) {
        showFileContent(row.id, row.path);
        return;
    }

    // 如果打开文件管理器模式，在drawer中打开
    if (props.openFileManager) {
        state.fileDialog.fileId = row.id;
        state.fileDialog.title = row.name;
        state.fileDialog.path = row.path;
        state.fileDialog.title = `${props.title} => ${row.path}`;
        state.fileDialog.visible = true;
        return;
    }

    // 否则触发select事件，让父组件在tab中打开
    emit('select', {
        fileId: row.id,
        path: row.path,
        name: row.name,
        type: row.type,
    });
    dialogVisible.value = false;
};

const showFileContent = async (fileId: number, path: string) => {
    state.fileContent.fileId = fileId;
    state.fileContent.path = path;
    state.fileContent.title = `${props.title} => ${path}`;
    state.fileContent.contentVisible = true;
};

/**
 * 关闭取消按钮触发的事件
 */
const handleClose = () => {
    dialogVisible.value = false;
    machineId.value = null;
    emit('cancel');
    state.fileTable = [];
};
</script>
<style lang="scss"></style>
