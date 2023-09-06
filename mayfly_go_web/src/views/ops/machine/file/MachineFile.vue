<template>
    <div class="machine-file">
        <div>
            <el-progress v-if="uploadProgressShow" style="width: 90%; margin-left: 20px" :text-inside="true" :stroke-width="20" :percentage="progressNum" />
            <el-row class="mb10">
                <el-breadcrumb separator-icon="ArrowRight">
                    <el-breadcrumb-item v-for="path in filePathNav">
                        <el-link @click="setFiles(path.path)">{{ path.name }}</el-link>
                    </el-breadcrumb-item>
                </el-breadcrumb>
            </el-row>
            <el-table ref="fileTableRef" height="65vh" :data="files" style="width: 100%" highlight-current-row v-loading="loading">
                <el-table-column prop="name" label="名称" show-overflow-tooltip>
                    <template #header>
                        <div class="machine-file-table-header">
                            <div>
                                <el-button :disabled="nowPath == basePath" type="primary" circle size="small" icon="Back" @click="back()"> </el-button>
                                <el-button class="ml0" type="primary" circle size="small" icon="Refresh" @click="refresh()"> </el-button>

                                <el-upload
                                    :before-upload="beforeUpload"
                                    :on-success="uploadSuccess"
                                    action=""
                                    :http-request="getUploadFile"
                                    :headers="{ token }"
                                    :show-file-list="false"
                                    name="file"
                                    class="machine-file-upload-exec"
                                >
                                    <el-button v-auth="'machine:file:upload'" class="ml10" type="primary" circle size="small" icon="Upload"> </el-button>
                                </el-upload>

                                <el-button
                                    v-auth="'machine:file:write'"
                                    @click="showCreateFileDialog()"
                                    class="ml10"
                                    type="primary"
                                    circle
                                    size="small"
                                    icon="FolderAdd"
                                >
                                </el-button>
                            </div>
                        </div>
                    </template>

                    <template #default="scope">
                        <span v-if="scope.row.isFolder">
                            <SvgIcon :size="15" name="folder" color="#007AFF" />
                        </span>
                        <span v-else>
                            <SvgIcon :size="15" name="document" />
                        </span>

                        <span class="ml5" style="font-weight: bold">
                            <el-link @click="getFile(scope.row)" :underline="false">{{ scope.row.name }}</el-link>
                        </span>
                    </template>
                </el-table-column>

                <el-table-column prop="size" label="大小" width="100" sortable>
                    <template #default="scope">
                        <span style="color: #67c23a; font-weight: bold" v-if="scope.row.type == '-'"> {{ formatFileSize(scope.row.size) }} </span>
                        <span style="color: #67c23a; font-weight: bold" v-if="scope.row.type == 'd' && scope.row.dirSize"> {{ scope.row.dirSize }} </span>
                        <span style="color: #67c23a; font-weight: bold" v-if="scope.row.type == 'd' && !scope.row.dirSize">
                            <el-button @click="getDirSize(scope.row)" type="primary" link :loading="scope.row.loadingDirSize">计算</el-button>
                        </span>
                    </template>
                </el-table-column>

                <el-table-column prop="mode" label="属性" width="110"> </el-table-column>
                <el-table-column prop="modTime" label="修改时间" width="165" sortable> </el-table-column>

                <el-table-column label="操作" width="100">
                    <template #default="scope">
                        <el-link
                            @click="downloadFile(scope.row)"
                            v-if="scope.row.type == '-'"
                            v-auth="'machine:file:write'"
                            type="primary"
                            icon="download"
                            :underline="false"
                        ></el-link>

                        <el-link
                            @click="deleteFile(scope.row)"
                            v-if="!dontOperate(scope.row)"
                            v-auth="'machine:file:rm'"
                            type="danger"
                            icon="delete"
                            :underline="false"
                            class="ml10"
                        ></el-link>

                        <el-popover placement="top-start" :title="`${scope.row.path}-文件详情`" :width="520" trigger="click" @show="showFileStat(scope.row)">
                            <template #reference>
                                <span style="color: #67c23a; font-weight: bold">
                                    <el-link
                                        @click="showFileStat(scope.row)"
                                        icon="InfoFilled"
                                        :underline="false"
                                        link
                                        class="ml10"
                                        :loading="scope.row.loadingStat"
                                    ></el-link>
                                </span>
                            </template>
                            <el-input :input-style="{ color: 'black' }" disabled autosize v-model="scope.row.stat" type="textarea" />
                        </el-popover>
                    </template>
                </el-table-column>
            </el-table>
        </div>

        <el-dialog
            :destroy-on-close="true"
            title="新建文件"
            v-model="createFileDialog.visible"
            :before-close="closeCreateFileDialog"
            :close-on-click-modal="false"
            top="5vh"
            width="400px"
        >
            <div>
                <el-form-item prop="name" label="名称:">
                    <el-input v-model.trim="createFileDialog.name" placeholder="请输入名称" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="type" label="类型:">
                    <el-radio-group v-model="createFileDialog.type">
                        <el-radio label="d">文件夹</el-radio>
                        <el-radio label="-">文件</el-radio>
                    </el-radio-group>
                </el-form-item>
            </div>

            <template #footer>
                <div>
                    <el-button @click="closeCreateFileDialog">关闭</el-button>
                    <el-button v-auth="'machine:file:write'" type="primary" @click="createFile">确定</el-button>
                </div>
            </template>
        </el-dialog>

        <machine-file-content v-model:visible="fileContent.contentVisible" :machine-id="machineId" :file-id="fileId" :path="fileContent.path" />
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, onMounted, computed } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { machineApi } from '../api';

import { getSession } from '@/common/utils/storage';
import config from '@/common/config';
import { isTrue } from '@/common/assert';
import MachineFileContent from './MachineFileContent.vue';

const props = defineProps({
    machineId: { type: Number },
    fileId: { type: Number, default: 0 },
    path: { type: String, default: '' },
    isFolder: { type: Boolean, default: true },
});

const token = getSession('token');

const folderType = 'd';
const fileType = '-';
// 路径分隔符
const pathSep = '/';

const state = reactive({
    basePath: '', // 基础路径
    nowPath: '', // 当前路径
    loading: true,
    progressNum: 0,
    uploadProgressShow: false,
    files: [] as any,
    fileContent: {
        content: '',
        contentVisible: false,
        dialogTitle: '',
        path: '',
        type: 'shell',
    },
    createFileDialog: {
        visible: false,
        name: '',
        type: folderType,
        data: null as any,
    },
    file: null as any,
});

const { basePath, nowPath, loading, files, progressNum, uploadProgressShow, fileContent, createFileDialog } = toRefs(state);

onMounted(() => {
    state.basePath = props.path;
    setFiles(props.path);
});

const filePathNav = computed(() => {
    let basePath = state.basePath;
    const pathNavs = [
        {
            path: basePath,
            name: basePath,
        },
    ];
    if (basePath == state.nowPath) {
        return pathNavs;
    }

    const paths = state.nowPath.split(pathSep).splice(1);
    let nowPath = '';
    for (let path of paths) {
        if (!nowPath) {
            nowPath = pathSep + path;
        } else {
            nowPath = nowPath + pathSep + path;
        }
        // 最多只能点击到basePath
        if (nowPath.length <= basePath.length) {
            continue;
        }

        pathNavs.push({
            name: path,
            path: nowPath,
        });
    }

    return pathNavs;
});

const showFileContent = async (path: string) => {
    state.fileContent.dialogTitle = path;
    state.fileContent.path = path;
    state.fileContent.contentVisible = true;
};

const getFile = async (row: any) => {
    if (row.type == folderType) {
        await setFiles(row.path);
    } else {
        isTrue(row.size < 1 * 1024 * 1024, '文件太大, 请下载使用');
        await showFileContent(row.path);
    }
};

const setFiles = async (path: string) => {
    try {
        if (!path) {
            path = pathSep;
        }
        state.loading = true;
        state.files = await lsFile(path);
        state.nowPath = path;
    } finally {
        state.loading = false;
    }
};

const lsFile = async (path: string) => {
    const res = await machineApi.lsFile.request({
        fileId: props.fileId,
        machineId: props.machineId,
        path,
    });
    for (const file of res) {
        const type = file.type;
        if (type == folderType) {
            file.isFolder = true;
        } else {
            file.isFolder = false;
        }
    }
    return res;
};

const back = () => {
    setFiles(getParentPath(state.nowPath));
};

const refresh = async () => {
    setFiles(state.nowPath);
};

const getDirSize = async (data: any) => {
    try {
        data.loadingDirSize = true;
        const res = await machineApi.dirSize.request({
            machineId: props.machineId,
            fileId: props.fileId,
            path: data.path,
        });
        data.dirSize = res;
    } finally {
        data.loadingDirSize = false;
    }
};

const showFileStat = async (data: any) => {
    try {
        if (data.stat) {
            return;
        }
        data.loadingStat = true;
        const res = await machineApi.fileStat.request({
            machineId: props.machineId,
            fileId: props.fileId,
            path: data.path,
        });
        data.stat = res;
    } finally {
        data.loadingStat = false;
    }
};

const showCreateFileDialog = () => {
    state.createFileDialog.data = {};
    state.createFileDialog.visible = true;
};

const createFile = async () => {
    const name = state.createFileDialog.name;
    const type = state.createFileDialog.type;
    const path = state.nowPath + pathSep + name;
    await machineApi.createFile.request({
        machineId: props.machineId,
        id: props.fileId,
        path,
        type,
    });

    closeCreateFileDialog();
    refresh();
};

const closeCreateFileDialog = () => {
    state.createFileDialog.visible = false;
    state.createFileDialog.data = null;
    state.createFileDialog.name = '';
    state.createFileDialog.type = folderType;
};

function getParentPath(filePath: string) {
    const segments = filePath.split(pathSep);
    segments.pop(); // 移除最后一个路径段
    return segments.join(pathSep);
}

const deleteFile = (data: any) => {
    const file = data.path;
    ElMessageBox.confirm(`此操作将删除 [${file}], 是否继续?`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    })
        .then(() => {
            machineApi.rmFile
                .request({
                    fileId: props.fileId,
                    path: file,
                    machineId: props.machineId,
                })
                .then(async () => {
                    ElMessage.success('删除成功');
                    refresh();
                });
        })
        .catch(() => {
            // skip
        });
};

const downloadFile = (data: any) => {
    const a = document.createElement('a');
    a.setAttribute('href', `${config.baseApiUrl}/machines/${props.machineId}/files/${props.fileId}/read?type=1&path=${data.path}&token=${token}`);
    a.click();
};

const onUploadProgress = (progressEvent: any) => {
    state.uploadProgressShow = true;
    let complete = ((progressEvent.loaded / progressEvent.total) * 100) | 0;
    state.progressNum = complete;
};

const getUploadFile = (content: any) => {
    const params = new FormData();
    const path = state.nowPath;
    params.append('file', content.file);
    params.append('path', path);
    params.append('machineId', props.machineId as any);
    params.append('fileId', props.fileId as any);
    params.append('token', token);
    machineApi.uploadFile
        .request(params, {
            url: `${config.baseApiUrl}/machines/${props.machineId}/files/${props.fileId}/upload?token=${token}`,
            headers: { 'Content-Type': 'multipart/form-data; boundary=----WebKitFormBoundaryF1uyUD0tWdqmJqpl' },
            onUploadProgress: onUploadProgress,
            baseURL: '',
            timeout: 60 * 60 * 1000,
        })
        .then(() => {
            ElMessage.success('上传成功');
            setTimeout(() => {
                refresh();
                state.uploadProgressShow = false;
            }, 3000);
        })
        .catch(() => {
            state.uploadProgressShow = false;
        });
};

const uploadSuccess = (res: any) => {
    if (res.code !== 200) {
        ElMessage.error(res.msg);
    }
};

const beforeUpload = (file: File) => {
    state.file = file;
};

const dontOperate = (data: any) => {
    const path = data.path;
    const ls = ['/', '//', '/usr', '/usr/', '/usr/bin', '/opt', '/run', '/etc', '/proc', '/var', '/mnt', '/boot', '/dev', '/home', '/media', '/root'];
    return ls.indexOf(path) != -1;
};

/**
 * 格式化文件大小
 * @param {*} value
 */
const formatFileSize = (size: any) => {
    const value = Number(size);
    if (size && !isNaN(value)) {
        const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB', 'BB'];
        let index = 0;
        let k = value;
        if (value >= 1024) {
            while (k > 1024) {
                k = k / 1024;
                index++;
            }
        }
        return `${k.toFixed(2)}${units[index]}`;
    }
    return '-';
};

defineExpose({ showFileContent });
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
.machine-file-table-header {
    display: flex;
    justify-content: space-between;
    font-size: 16px;

    .title-right-fixed {
        display: flex;
        align-items: center;
        font-size: 20px;
        text-align: end;
    }
}
</style>
