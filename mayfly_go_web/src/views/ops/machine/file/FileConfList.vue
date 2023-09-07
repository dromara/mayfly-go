<template>
    <div class="file-manage">
        <el-dialog v-if="dialogVisible" :title="title" v-model="dialogVisible" :show-close="true" :before-close="handleClose" width="50%">
            <el-table :data="fileTable" stripe style="width: 100%" v-loading="loading">
                <el-table-column prop="name" label="名称" min-width="100px">
                    <template #header>
                        <el-button class="ml0" type="primary" circle size="small" icon="Plus" @click="add()"> </el-button>
                        <span class="ml10">名称</span>
                    </template>
                    <template #default="scope">
                        <el-input v-model="scope.row.name" :disabled="scope.row.id != null" clearable> </el-input>
                    </template>
                </el-table-column>
                <el-table-column prop="name" label="类型" width="130px">
                    <template #default="scope">
                        <el-select :disabled="scope.row.id != null" v-model="scope.row.type" style="width: 100px" placeholder="请选择">
                            <el-option v-for="item in FileTypeEnum as any" :key="item.value" :label="item.label" :value="item.value"></el-option>
                        </el-select>
                    </template>
                </el-table-column>
                <el-table-column prop="path" label="路径" min-width="150px" show-overflow-tooltip>
                    <template #default="scope">
                        <el-input v-model="scope.row.path" :disabled="scope.row.id != null" clearable> </el-input>
                    </template>
                </el-table-column>
                <el-table-column label="操作" min-wdith="120px">
                    <template #default="scope">
                        <el-button v-if="scope.row.id == null" @click="addFiles(scope.row)" type="success" icon="success-filled" plain></el-button>
                        <el-button v-if="scope.row.id != null" @click="getConf(scope.row)" type="primary" icon="tickets" plain></el-button>
                        <el-button v-auth="'machine:file:del'" type="danger" @click="deleteRow(scope.$index, scope.row)" icon="delete" plain></el-button>
                    </template>
                </el-table-column>
            </el-table>
            <el-row style="margin-top: 10px" type="flex" justify="end">
                <el-pagination
                    style="text-align: center"
                    :total="total"
                    layout="prev, pager, next, total, jumper"
                    v-model:current-page="query.pageNum"
                    :page-size="query.pageSize"
                    @current-change="handlePageChange"
                >
                </el-pagination>
            </el-row>

            <el-dialog destroy-on-close :title="fileDialog.title" v-model="fileDialog.visible" :close-on-click-modal="false" width="70%">
                <machine-file :title="fileDialog.title" :machine-id="machineId" :file-id="fileDialog.fileId" :path="fileDialog.path" />
            </el-dialog>

            <machine-file-content
                :title="fileContent.title"
                v-model:visible="fileContent.contentVisible"
                :machine-id="machineId"
                :file-id="fileContent.fileId"
                :path="fileContent.path"
            />
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { machineApi } from '../api';
import { FileTypeEnum } from '../enums';
import MachineFile from './MachineFile.vue';
import MachineFileContent from './MachineFileContent.vue';

const props = defineProps({
    visible: { type: Boolean },
    machineId: { type: Number },
    title: { type: String },
});

const emit = defineEmits(['update:visible', 'cancel', 'update:machineId']);

const addFile = machineApi.addConf;
const delFile = machineApi.delConf;
const files = machineApi.files;

const state = reactive({
    dialogVisible: false,
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
    fileTable: [] as any,
    fileDialog: {
        visible: false,
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

const { dialogVisible, loading, query, total, fileTable, fileDialog, fileContent } = toRefs(state);

watch(props, async (newValue) => {
    state.dialogVisible = newValue.visible;
    if (newValue.machineId && newValue.visible) {
        await getFiles();
    }
});

const getFiles = async () => {
    try {
        state.loading = true;
        state.query.id = props.machineId as any;
        const res = await files.request(state.query);
        state.fileTable = res.list || [];
        state.total = res.total;
    } finally {
        state.loading = false;
    }
};

const handlePageChange = (curPage: number) => {
    state.query.pageNum = curPage;
    getFiles();
};

const add = () => {
    // 往数组头部添加元素
    state.fileTable = [{}].concat(state.fileTable);
};

const addFiles = async (row: any) => {
    row.machineId = props.machineId;
    await addFile.request(row);
    ElMessage.success('添加成功');
    getFiles();
};

const deleteRow = (idx: any, row: any) => {
    if (row.id) {
        ElMessageBox.confirm(`此操作将删除 [${row.name}], 是否继续?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        }).then(() => {
            // 删除配置文件
            delFile
                .request({
                    machineId: props.machineId,
                    id: row.id,
                })
                .then(() => {
                    getFiles();
                });
        });
    } else {
        state.fileTable.splice(idx, 1);
    }
};

const getConf = async (row: any) => {
    if (row.type == 1) {
        state.fileDialog.fileId = row.id;
        state.fileDialog.title = row.name;
        state.fileDialog.path = row.path;
        state.fileDialog.title = `${props.title} => ${row.path}`;
        state.fileDialog.visible = true;
        return;
    }

    showFileContent(row.id, row.path);
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
    emit('update:visible', false);
    emit('update:machineId', null);
    emit('cancel');
    state.fileTable = [];
};
</script>
<style lang="scss">
.machine-file-upload-exec {
    display: inline-flex;
    flex-direction: row;
    align-items: center;
    justify-content: center;
    vertical-align: middle;
    position: relative;
    text-decoration: none;
}
.inline-block {
    display: inline-block;
    margin-right: 10px;
}
.margin-change {
    display: inline-block;
    margin-left: 10px;
}
</style>
