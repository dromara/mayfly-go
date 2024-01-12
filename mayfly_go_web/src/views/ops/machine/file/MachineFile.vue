<template>
    <div class="machine-file">
        <div>
            <el-progress v-if="uploadProgressShow" style="width: 90%; margin-left: 20px" :text-inside="true" :stroke-width="20" :percentage="progressNum" />

            <el-row class="mb10">
                <el-breadcrumb separator-icon="ArrowRight">
                    <el-breadcrumb-item v-for="path in filePathNav" :key="path">
                        <el-link @click="setFiles(path.path)" style="font-weight: bold">{{ path.name }}</el-link>
                    </el-breadcrumb-item>
                </el-breadcrumb>
            </el-row>

            <el-table
                ref="fileTableRef"
                @cell-dblclick="cellDbclick"
                @selection-change="handleSelectionChange"
                height="65vh"
                :data="filterFiles"
                highlight-current-row
                v-loading="loading"
            >
                <el-table-column type="selection" width="30" />

                <el-table-column prop="name" label="名称">
                    <template #header>
                        <div class="machine-file-table-header">
                            <div>
                                <el-button :disabled="nowPath == basePath" type="primary" circle size="small" icon="Back" @click="back()"> </el-button>
                                <el-button class="ml5" type="primary" circle size="small" icon="Refresh" @click="refresh()"> </el-button>

                                <!-- 文件&文件夹上传 -->
                                <el-dropdown class="machine-file-upload-exec" trigger="click" size="small">
                                    <span>
                                        <el-button
                                            v-auth="'machine:file:upload'"
                                            class="ml5"
                                            type="primary"
                                            circle
                                            size="small"
                                            icon="Upload"
                                            title="上传"
                                        ></el-button>
                                    </span>
                                    <template #dropdown>
                                        <el-dropdown-menu>
                                            <el-dropdown-item>
                                                <el-upload
                                                    :before-upload="beforeUpload"
                                                    :on-success="uploadSuccess"
                                                    action=""
                                                    :http-request="uploadFile"
                                                    :headers="{ token }"
                                                    :show-file-list="false"
                                                    name="file"
                                                    class="machine-file-upload-exec"
                                                >
                                                    <el-link>文件</el-link>
                                                </el-upload>
                                            </el-dropdown-item>

                                            <el-dropdown-item>
                                                <div>
                                                    <el-link @click="addFinderToList">文件夹</el-link>
                                                    <input
                                                        type="file"
                                                        id="folderUploadInput"
                                                        ref="folderUploadRef"
                                                        webkitdirectory
                                                        directory
                                                        @change="uploadFolder"
                                                        style="display: none"
                                                    />
                                                </div>
                                            </el-dropdown-item>
                                        </el-dropdown-menu>
                                    </template>
                                </el-dropdown>

                                <el-button
                                    :disabled="state.selectionFiles.length == 0"
                                    v-auth="'machine:file:rm'"
                                    @click="copyFile(state.selectionFiles)"
                                    class="ml5"
                                    type="primary"
                                    circle
                                    size="small"
                                    icon="CopyDocument"
                                    title="复制"
                                >
                                </el-button>

                                <el-button
                                    :disabled="state.selectionFiles.length == 0"
                                    v-auth="'machine:file:rm'"
                                    @click="mvFile(state.selectionFiles)"
                                    class="ml5"
                                    type="primary"
                                    circle
                                    size="small"
                                    icon="Rank"
                                    title="移动"
                                >
                                </el-button>

                                <el-button
                                    v-auth="'machine:file:write'"
                                    @click="showCreateFileDialog()"
                                    class="ml5"
                                    type="primary"
                                    circle
                                    size="small"
                                    icon="FolderAdd"
                                    title="新建"
                                >
                                </el-button>

                                <el-button
                                    :disabled="state.selectionFiles.length == 0"
                                    v-auth="'machine:file:rm'"
                                    @click="deleteFile(state.selectionFiles)"
                                    class="ml5"
                                    type="danger"
                                    circle
                                    size="small"
                                    icon="delete"
                                    title="删除"
                                >
                                </el-button>

                                <el-button-group v-if="state.copyOrMvFile.paths.length > 0" size="small" class="ml5">
                                    <el-tooltip effect="customized" raw-content placement="top">
                                        <template #content>
                                            <div v-for="path in state.copyOrMvFile.paths" v-bind:key="path">{{ path }}</div>
                                        </template>

                                        <el-button @click="pasteFile" type="primary"
                                            >{{ isCpFile() ? '复制' : '移动' }}粘贴{{ state.copyOrMvFile.paths.length }}</el-button
                                        >
                                    </el-tooltip>

                                    <el-button icon="CloseBold" @click="cancelCopy" />
                                </el-button-group>
                            </div>

                            <div style="width: 150px">
                                <el-input v-model="fileNameFilter" size="small" placeholder="名称: 输入可过滤" clearable />
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

                        <span class="ml5" style="display: inline-block; width: 90%">
                            <div v-if="scope.row.nameEdit">
                                <el-input
                                    @keyup.enter="fileRename(scope.row)"
                                    :ref="(el: any) => el?.focus()"
                                    @blur="filenameBlur(scope.row)"
                                    v-model="scope.row.name"
                                />
                            </div>
                            <el-link v-else @click="getFile(scope.row)" style="font-weight: bold" :underline="false">{{ scope.row.name }}</el-link>
                        </span>
                    </template>
                </el-table-column>

                <el-table-column prop="size" label="大小" width="100" sortable>
                    <template #default="scope">
                        <span style="color: #67c23a; font-weight: bold" v-if="scope.row.type == '-'"> {{ formatByteSize(scope.row.size) }} </span>
                        <span style="color: #67c23a; font-weight: bold" v-if="scope.row.type == 'd' && scope.row.dirSize"> {{ scope.row.dirSize }} </span>
                        <span style="color: #67c23a; font-weight: bold" v-if="scope.row.type == 'd' && !scope.row.dirSize">
                            <el-button @click="getDirSize(scope.row)" type="primary" link :loading="scope.row.loadingDirSize">计算</el-button>
                        </span>
                    </template>
                </el-table-column>

                <el-table-column prop="mode" label="属性" width="110"> </el-table-column>
                <el-table-column prop="modTime" label="修改时间" width="165" sortable> </el-table-column>

                <el-table-column width="100">
                    <template #header>
                        <el-popover placement="top" :width="270" trigger="hover">
                            <template #reference>
                                <SvgIcon name="QuestionFilled" :size="18" class="pointer-icon mr10" />
                            </template>
                            <div>rename: 双击文件名单元格修改后回车</div>
                        </el-popover>
                        操作
                    </template>
                    <template #default="scope">
                        <el-link
                            @click="deleteFile([scope.row])"
                            v-if="!dontOperate(scope.row)"
                            v-auth="'machine:file:rm'"
                            type="danger"
                            icon="delete"
                            :underline="false"
                            title="删除"
                        ></el-link>

                        <el-link
                            @click="downloadFile(scope.row)"
                            v-if="scope.row.type == '-'"
                            v-auth="'machine:file:write'"
                            type="primary"
                            icon="download"
                            :underline="false"
                            class="ml10"
                            title="下载"
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
                            <el-input disabled autosize v-model="scope.row.stat" type="textarea" />
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
                <el-form-item prop="name" label="名称">
                    <el-input v-model.trim="createFileDialog.name" placeholder="请输入名称" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="type" label="类型">
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
import { ref, toRefs, reactive, onMounted, computed } from 'vue';
import { ElMessage, ElMessageBox, ElInput } from 'element-plus';
import { machineApi } from '../api';

import { joinClientParams } from '@/common/request';
import config from '@/common/config';
import { isTrue } from '@/common/assert';
import MachineFileContent from './MachineFileContent.vue';
import { notBlank } from '@/common/assert';
import { getToken } from '@/common/utils/storage';
import { formatByteSize, convertToBytes } from '@/common/utils/format';
import { getMachineConfig } from '@/common/sysconfig';

const props = defineProps({
    machineId: { type: Number },
    fileId: { type: Number, default: 0 },
    path: { type: String, default: '' },
    isFolder: { type: Boolean, default: true },
});

const token = getToken();
const folderUploadRef: any = ref();

const folderType = 'd';

// 路径分隔符
const pathSep = '/';

const state = reactive({
    basePath: '', // 基础路径
    nowPath: '', // 当前路径
    loading: true,
    progressNum: 0,
    uploadProgressShow: false,
    fileNameFilter: '',
    files: [] as any,
    selectionFiles: [] as any,
    copyOrMvFile: {
        paths: [] as any,
        type: 'cp',
        fromPath: '',
    },
    renameFile: {
        oldname: '',
    },
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
    machineConfig: { uploadMaxFileSize: '1GB' },
});

const { basePath, nowPath, loading, fileNameFilter, progressNum, uploadProgressShow, fileContent, createFileDialog } = toRefs(state);

onMounted(async () => {
    state.basePath = props.path;
    setFiles(props.path);
    state.machineConfig = await getMachineConfig();
});

const filterFiles = computed(() =>
    state.files.filter((data: any) => !state.fileNameFilter || data.name.toLowerCase().includes(state.fileNameFilter.toLowerCase()))
);

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

const handleSelectionChange = (val: any) => {
    state.selectionFiles = val;
};

const isCpFile = () => {
    return state.copyOrMvFile.type == 'cp';
};

const copyFile = (files: any[]) => {
    setCopyOrMvFile(files);
};

const mvFile = (files: any[]) => {
    setCopyOrMvFile(files, 'mv');
};

const setCopyOrMvFile = (files: any[], type = 'cp') => {
    for (let file of files) {
        const path = file.path;
        if (!state.copyOrMvFile.paths.includes(path)) {
            state.copyOrMvFile.paths.push(path);
        }
    }
    state.copyOrMvFile.type = type;
    state.copyOrMvFile.fromPath = state.nowPath;
};

const pasteFile = async () => {
    const cmFile = state.copyOrMvFile;
    isTrue(state.nowPath != cmFile.fromPath, '同目录下不能粘贴');
    const api = isCpFile() ? machineApi.cpFile : machineApi.mvFile;
    try {
        state.loading = true;
        await api.request({
            machineId: props.machineId,
            fileId: props.fileId,
            path: cmFile.paths,
            toPath: state.nowPath,
        });
        ElMessage.success('粘贴成功');
        state.copyOrMvFile.paths = [];
        refresh();
    } finally {
        state.loading = false;
    }
};

const cancelCopy = () => {
    state.copyOrMvFile.paths = [];
};

const cellDbclick = (row: any, column: any) => {
    // 双击名称列可修改名称
    if (column.property == 'name') {
        state.renameFile.oldname = row.name;
        row.nameEdit = true;
    }
};

const filenameBlur = (row: any) => {
    const oldname = state.renameFile.oldname;
    // 如果存在旧名称，则说明未回车修改文件名，则还原旧文件名
    if (oldname) {
        row.name = oldname;
        state.renameFile.oldname = '';
    }
    row.nameEdit = false;
};

const fileRename = async (row: any) => {
    if (row.name == state.renameFile.oldname) {
        row.nameEdit = false;
        return;
    }
    notBlank(row.name, '新名称不能为空');
    try {
        await machineApi.renameFile.request({
            machineId: props.machineId,
            fileId: props.fileId,
            oldname: state.nowPath + pathSep + state.renameFile.oldname,
            newname: state.nowPath + pathSep + row.name,
        });
        ElMessage.success('重命名成功');
        // 修改路径上的文件名
        row.path = state.nowPath + pathSep + row.name;
        state.renameFile.oldname = '';
    } catch (e) {
        row.name = state.renameFile.oldname;
    }
    row.nameEdit = false;
};

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
        state.fileNameFilter = '';
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

const deleteFile = async (files: any) => {
    try {
        await ElMessageBox.confirm(`此操作将删除 ${files.map((x: any) => `[${x.path}]`).join('\n')}, 是否继续?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        state.loading = true;
        await machineApi.rmFile.request({
            fileId: props.fileId,
            path: files.map((x: any) => x.path),
            machineId: props.machineId,
        });
        ElMessage.success('删除成功');
        refresh();
    } catch (e) {
        //
    } finally {
        state.loading = false;
    }
};

const downloadFile = (data: any) => {
    const a = document.createElement('a');
    a.setAttribute('href', `${config.baseApiUrl}/machines/${props.machineId}/files/${props.fileId}/download?path=${data.path}&${joinClientParams()}`);
    a.click();
};

function addFinderToList() {
    folderUploadRef.value.click();
}

function uploadFolder(e: any) {
    //e.target.files为文件夹里面的文件
    // 把文件夹数据放到formData里面，下面的files和paths字段根据接口来定
    var form = new FormData();
    form.append('basePath', state.nowPath);

    let totalFileSize = 0;
    for (let file of e.target.files) {
        totalFileSize += file.size;
        form.append('files', file);
        form.append('paths', file.webkitRelativePath);
    }

    try {
        if (!checkUploadFileSize(totalFileSize)) {
            return;
        }

        // 上传操作
        machineApi.uploadFile
            .xhrReq(form, {
                url: `${config.baseApiUrl}/machines/${props.machineId}/files/${props.fileId}/upload-folder?${joinClientParams()}`,
                headers: { 'Content-Type': 'multipart/form-data; boundary=----WebKitFormBoundaryF1uyUD0tWdqmJqpl' },
                onUploadProgress: onUploadProgress,
                baseURL: '',
                timeout: 3 * 60 * 60 * 1000,
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
    } finally {
        //无论上传成功与否，都把已选择的文件夹清空，否则选择同一文件夹没有反应
        const folderEle: any = document.getElementById('folderUploadInput');
        if (folderEle) {
            folderEle.value = '';
        }
    }
}

const onUploadProgress = (progressEvent: any) => {
    state.uploadProgressShow = true;
    let complete = ((progressEvent.loaded / progressEvent.total) * 100) | 0;
    state.progressNum = complete;
};

const uploadFile = (content: any) => {
    const params = new FormData();
    const path = state.nowPath;
    params.append('file', content.file);
    params.append('path', path);
    params.append('machineId', props.machineId as any);
    params.append('fileId', props.fileId as any);
    params.append('token', token);
    machineApi.uploadFile
        .xhrReq(params, {
            url: `${config.baseApiUrl}/machines/${props.machineId}/files/${props.fileId}/upload?${joinClientParams()}`,
            headers: { 'Content-Type': 'multipart/form-data; boundary=----WebKitFormBoundaryF1uyUD0tWdqmJqpl' },
            onUploadProgress: onUploadProgress,
            baseURL: '',
            timeout: 3 * 60 * 60 * 1000,
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
    return checkUploadFileSize(file.size);
};

const checkUploadFileSize = (fileSize: number) => {
    const bytes = convertToBytes(state.machineConfig.uploadMaxFileSize);
    if (fileSize > bytes) {
        ElMessage.error(`上传的文件超过系统配置的[${state.machineConfig.uploadMaxFileSize}]`);
        return false;
    }
    return true;
};

const dontOperate = (data: any) => {
    const path = data.path;
    const ls = ['/', '//', '/usr', '/usr/', '/usr/bin', '/opt', '/run', '/etc', '/proc', '/var', '/mnt', '/boot', '/dev', '/home', '/media', '/root'];
    return ls.indexOf(path) != -1;
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
