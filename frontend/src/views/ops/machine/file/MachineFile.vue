<template>
    <div class="machine-file h-full">
        <div class="h-full flex flex-col">
            <!-- 文件路径 -->
            <el-row class="mb-2 ml-4">
                <el-breadcrumb separator-icon="ArrowRight">
                    <el-breadcrumb-item v-for="path in filePathNav" :key="path">
                        <el-link @click="setFiles(path.path)" class="cursor-pointer! font-bold!">{{ path.name }}</el-link>
                    </el-breadcrumb-item>
                </el-breadcrumb>
            </el-row>

            <!-- 文件列表 -->
            <div class="flex-1 overflow-auto">
                <el-table
                    ref="fileTableRef"
                    @cell-dblclick="cellDbclick"
                    @selection-change="handleSelectionChange"
                    height="100%"
                    :data="filterFiles"
                    highlight-current-row
                    v-loading="loading"
                >
                    <el-table-column type="selection" width="30" />

                    <!-- 文件名 -->
                    <el-table-column prop="name" :label="$t('common.name')" min-width="380">
                        <template #header>
                            <div class="machine-file-table-header">
                                <div>
                                    <el-button :disabled="nowPath == basePath" type="primary" circle size="small" icon="Back" @click="back()"> </el-button>
                                    <el-button class="ml-1!" type="primary" circle size="small" icon="Refresh" @click="refresh()"> </el-button>

                                    <!-- 文件&文件夹上传 -->
                                    <el-dropdown class="machine-file-upload-exec" trigger="click" size="small">
                                        <span>
                                            <el-button
                                                v-auth="'machine:file:upload'"
                                                class="ml-1!"
                                                type="primary"
                                                circle
                                                size="small"
                                                icon="Upload"
                                                :title="$t('machine.upload')"
                                            ></el-button>
                                        </span>
                                        <template #dropdown>
                                            <el-dropdown-menu>
                                                <el-dropdown-item>
                                                    <el-upload
                                                        :before-upload="beforeUpload"
                                                        :on-success="uploadSuccess"
                                                        action=""
                                                        :http-request="handleFileUpload"
                                                        :headers="{ token }"
                                                        :show-file-list="false"
                                                        name="file"
                                                        class="machine-file-upload-exec"
                                                    >
                                                        <el-link>{{ $t('machine.file') }}</el-link>
                                                    </el-upload>
                                                </el-dropdown-item>

                                                <el-dropdown-item>
                                                    <div>
                                                        <el-link @click="addFinderToList">{{ $t('machine.folder') }}</el-link>
                                                        <input
                                                            type="file"
                                                            id="folderUploadInput"
                                                            ref="folderUploadRef"
                                                            webkitdirectory
                                                            directory
                                                            @change="handleFolderUpload"
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
                                        class="ml-1!"
                                        type="primary"
                                        circle
                                        size="small"
                                        icon="CopyDocument"
                                        :title="$t('machine.copy')"
                                    >
                                    </el-button>

                                    <el-button
                                        :disabled="state.selectionFiles.length == 0"
                                        v-auth="'machine:file:rm'"
                                        @click="mvFile(state.selectionFiles)"
                                        class="ml-1!"
                                        type="primary"
                                        circle
                                        size="small"
                                        icon="Rank"
                                        :title="$t('machine.move')"
                                    >
                                    </el-button>

                                    <el-button
                                        v-auth="'machine:file:write'"
                                        @click="showCreateFileDialog()"
                                        class="ml-1!"
                                        type="primary"
                                        circle
                                        size="small"
                                        icon="FolderAdd"
                                        :title="$t('common.create')"
                                    >
                                    </el-button>

                                    <el-button
                                        :disabled="state.selectionFiles.length == 0"
                                        v-auth="'machine:file:rm'"
                                        @click="deleteFile(state.selectionFiles)"
                                        class="ml-1!"
                                        type="danger"
                                        circle
                                        size="small"
                                        icon="delete"
                                        :title="$t('common.delete')"
                                    >
                                    </el-button>

                                    <el-button-group v-if="copyOrMvFile.paths.length > 0" size="small" class="ml-1!">
                                        <el-tooltip effect="customized" raw-content placement="top">
                                            <template #content>
                                                <div v-for="path in copyOrMvFile.paths" v-bind:key="path">{{ path }}</div>
                                            </template>

                                            <el-button @click="pasteFile" type="primary">
                                                {{ isCpFile() ? $t('machine.copy') : $t('machine.move') }}
                                                {{ $t('machine.paste') }}{{ copyOrMvFile.paths.length }}</el-button
                                            >
                                        </el-tooltip>

                                        <el-button icon="CloseBold" @click="cancelCopy" />
                                    </el-button-group>
                                </div>

                                <div class="w-[150px]">
                                    <el-input v-model="fileNameFilter" size="small" :placeholder="$t('machine.fileNameFilterPlaceholder')" clearable />
                                </div>
                            </div>
                        </template>

                        <template #default="scope">
                            <div class="w-full cursor-pointer" @click="getFile(scope.row)">
                                <span v-if="scope.row.isFolder">
                                    <SvgIcon :size="15" name="folder" color="#007AFF" />
                                </span>
                                <span v-else>
                                    <SvgIcon :size="15" :name="scope.row.icon" />
                                </span>

                                <span class="ml-1! inline-block w-[90%]">
                                    <div v-if="scope.row.nameEdit">
                                        <el-input
                                            @keyup.enter="fileRename(scope.row)"
                                            :ref="focusRenameInput"
                                            @blur="filenameBlur(scope.row)"
                                            v-model="scope.row.name"
                                        />
                                    </div>
                                    <el-link v-else style="font-weight: bold" underline="never">{{ scope.row.name }}</el-link>
                                </span>
                            </div>
                        </template>
                    </el-table-column>

                    <el-table-column prop="size" label="Size" min-width="90" sortable>
                        <template #default="scope">
                            <span style="color: #67c23a; font-weight: bold" v-if="scope.row.type == '-'"> {{ formatByteSize(scope.row.size) }} </span>
                            <span style="color: #67c23a; font-weight: bold" v-if="scope.row.type == 'd' && scope.row.dirSize"> {{ scope.row.dirSize }} </span>
                            <span style="color: #67c23a; font-weight: bold" v-if="scope.row.type == 'd' && !scope.row.dirSize">
                                <el-button @click="getDirSize(scope.row)" type="primary" link :loading="scope.row.loadingDirSize">
                                    {{ $t('machine.calculate') }}
                                </el-button>
                            </span>
                        </template>
                    </el-table-column>

                    <el-table-column prop="mode" :label="$t('machine.attribute')" width="110"> </el-table-column>

                    <el-table-column v-if="$props.protocol == MachineProtocolEnum.Ssh.value" :label="$t('machine.user')" min-width="70" show-overflow-tooltip>
                        <template #default="scope">
                            {{ userMap.get(scope.row.uid)?.uname || scope.row.uid }}
                        </template>
                    </el-table-column>

                    <el-table-column v-if="$props.protocol == MachineProtocolEnum.Ssh.value" :label="$t('machine.group')" min-width="70" show-overflow-tooltip>
                        <template #default="scope">
                            {{ groupMap.get(scope.row.gid)?.gname || scope.row.gid }}
                        </template>
                    </el-table-column>

                    <el-table-column prop="modTime" :label="$t('machine.modificationTime')" width="160" sortable> </el-table-column>

                    <el-table-column :width="100">
                        <template #header>
                            <el-popover placement="top" :width="270" trigger="hover">
                                <template #reference>
                                    <SvgIcon name="QuestionFilled" :size="18" class="pointer-icon mr-2" />
                                </template>
                                <div>{{ $t('machine.renameTips') }}</div>
                            </el-popover>
                            {{ $t('common.operation') }}
                        </template>

                        <template #default="scope">
                            <div class="flex gap-1.5">
                                <!-- 基础信息 -->
                                <el-popover
                                    placement="top-start"
                                    :title="`${scope.row.path} - ${$t('machine.fileDetail')}`"
                                    :width="520"
                                    trigger="click"
                                    @show="showFileStat(scope.row)"
                                >
                                    <template #reference>
                                        <span style="color: #67c23a; font-weight: bold">
                                            <el-link
                                                @click="showFileStat(scope.row)"
                                                icon="InfoFilled"
                                                underline="never"
                                                link
                                                :loading="scope.row.loadingStat"
                                            ></el-link>
                                        </span>
                                    </template>
                                    <el-input disabled autosize v-model="scope.row.stat" type="textarea" />
                                </el-popover>

                                <!-- 下载文件 -->
                                <el-link
                                    @click="downloadFile(scope.row)"
                                    v-if="scope.row.type == '-'"
                                    v-auth="'machine:file:write'"
                                    type="primary"
                                    icon="download"
                                    underline="never"
                                    :title="$t('machine.download')"
                                ></el-link>

                                <!-- 删除文件 -->
                                <el-link
                                    @click="deleteFile([scope.row])"
                                    v-if="!dontOperate(scope.row)"
                                    v-auth="'machine:file:rm'"
                                    type="danger"
                                    icon="delete"
                                    underline="never"
                                    :title="$t('common.delete')"
                                ></el-link>
                            </div>
                        </template>
                    </el-table-column>
                </el-table>
            </div>
        </div>

        <el-dialog
            :destroy-on-close="true"
            :title="$t('machine.createFile')"
            v-model="createFileDialog.visible"
            :before-close="closeCreateFileDialog"
            :close-on-click-modal="false"
            top="5vh"
            width="400px"
        >
            <div>
                <el-form-item prop="name" :label="$t('common.name')">
                    <el-input v-model.trim="createFileDialog.name" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="type" :label="$t('common.type')">
                    <el-radio-group v-model="createFileDialog.type">
                        <el-radio value="d" label="d">{{ $t('machine.folder') }}</el-radio>
                        <el-radio value="-" label="-">{{ $t('machine.file') }}</el-radio>
                    </el-radio-group>
                </el-form-item>
            </div>

            <template #footer>
                <div>
                    <el-button @click="closeCreateFileDialog">{{ $t('common.cancel') }}</el-button>
                    <el-button v-auth="'machine:file:write'" type="primary" @click="createFile">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-dialog>

        <machine-file-content
            v-model:visible="fileContent.contentVisible"
            :machine-id="machineId"
            :auth-cert-name="props.authCertName"
            :file-id="fileId"
            :path="fileContent.path"
            :protocol="protocol"
        />
    </div>
</template>

<script lang="ts" setup>
import { Msg } from '@/hooks/useI18n';
import { ElInput } from 'element-plus';
import { computed, defineAsyncComponent, getCurrentInstance, onMounted, reactive, ref, toRefs } from 'vue';
import { machineApi } from '../api';

import { isTrue, notBlank } from '@/common/assert';
import { getMachineConfig } from '@/common/sysconfig';
import { formatByteSize } from '@/common/utils/format';
import { getToken } from '@/common/utils/storage';
import { fuzzyMatchField } from '@/common/utils/string';
import { useI18n } from 'vue-i18n';
import type { ComponentPublicInstance } from 'vue';
import { MachineProtocolEnum } from '../enums';
import type { MachineFileInfo, MachineUserInfo, MachineGroupInfo } from '../types';
import { getFileIcon } from './utils/fileIcon';
import { useFileOperations } from './composables/useFileOperations';

const MachineFileContent = defineAsyncComponent(() => import('./MachineFileContent.vue'));

const { t } = useI18n();

const props = defineProps({
    machineId: { type: Number },
    authCertName: { type: String },
    protocol: { type: Number, default: 1 },
    fileId: { type: Number, default: 0 },
    path: { type: String, default: '' },
    isFolder: { type: Boolean, default: true },
    tabKey: { type: String, default: '' },
});

const token = getToken();
const folderUploadRef = ref<HTMLInputElement | null>(null);

const folderType = 'd';

const userMap = ref(new Map<number, MachineUserInfo>());
const groupMap = ref(new Map<number, MachineGroupInfo>());

// 路径分隔符
const pathSep = '/';

const state = reactive({
    basePath: '', // 基础路径
    nowPath: '', // 当前路径
    loading: true,
    fileNameFilter: '',
    files: [] as MachineFileInfo[],
    selectionFiles: [] as MachineFileInfo[],
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
        data: null as Record<string, unknown> | null,
    },
    machineConfig: { uploadMaxFileSize: '1GB' },
});

const { basePath, nowPath, loading, fileNameFilter, fileContent, createFileDialog } = toRefs(state);

const emits = defineEmits(['init']);

// File operations composable
const {
    copyOrMvFile,
    isCpFile,
    copyFile,
    mvFile,
    pasteFile,
    cancelCopy,
    fileRename: doFileRename,
    deleteFile,
    createFile: doCreateFile,
    downloadFile,
    getDirSize,
    showFileStat,
    checkUploadFileSize,
    handleFileUpload,
    handleFolderUpload: doFolderUpload,
    dontOperate,
} = useFileOperations({
    machineId: () => props.machineId,
    authCertName: () => props.authCertName,
    fileId: () => props.fileId,
    protocol: () => props.protocol,
    nowPath: () => state.nowPath,
    setLoading: (l) => (state.loading = l),
    refresh: () => refresh(),
    uploadMaxFileSize: () => state.machineConfig.uploadMaxFileSize,
});

// Init as MachineOp component
onMounted(async () => {
    emits('init', { name: 'tag.machineOp', tabKey: props.tabKey, ref: getCurrentInstance()?.exposed });

    state.basePath = props.path;
    const machineId = props.machineId;

    if (props.protocol == MachineProtocolEnum.Ssh.value) {
        machineApi.users.request({ id: machineId }).then((res: MachineUserInfo[]) => {
            for (let user of res) {
                userMap.value.set(user.uid, user);
            }
        });

        machineApi.groups.request({ id: machineId }).then((res: MachineGroupInfo[]) => {
            for (let group of res) {
                groupMap.value.set(group.gid, group);
            }
        });
    }

    setFiles(props.path);
    state.machineConfig = await getMachineConfig();
});

const filterFiles = computed(() => fuzzyMatchField(state.fileNameFilter, state.files, (file: MachineFileInfo) => file.name));

/** 重命名输入框渲染后自动聚焦 */
const focusRenameInput = (el: Element | ComponentPublicInstance | null) => {
    (el as InstanceType<typeof ElInput> | null)?.focus();
};

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

const handleSelectionChange = (val: MachineFileInfo[]) => {
    state.selectionFiles = val;
};

const cellDbclick = (row: MachineFileInfo, column: { property: string }) => {
    // 双击名称列可修改名称
    if (column.property == 'name') {
        state.renameFile.oldname = row.name;
        row.nameEdit = true;
    }
};

const filenameBlur = (row: MachineFileInfo) => {
    const oldname = state.renameFile.oldname;
    // 如果存在旧名称，则说明未回车修改文件名，则还原旧文件名
    if (oldname) {
        row.name = oldname;
        state.renameFile.oldname = '';
    }
    row.nameEdit = false;
};

const fileRename = async (row: MachineFileInfo) => {
    if (row.name == state.renameFile.oldname) {
        row.nameEdit = false;
        return;
    }
    try {
        await doFileRename(row, state.renameFile.oldname);
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

const getFile = async (row: MachineFileInfo) => {
    if (row.type == folderType) {
        await setFiles(row.path);
    } else {
        isTrue(row.size < 1 * 1024 * 1024, 'machine.fileTooLargeTips');
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
        state.files = []; // 清空旧数据，可能出现莫名其妙的文件展示错误，先这么处理
        state.nowPath = '';
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
        authCertName: props.authCertName,
        protocol: props.protocol,
        path,
    });
    for (const file of res) {
        const type = file.type;
        if (type == folderType) {
            file.isFolder = true;
            file.icon = 'folder';
        } else {
            file.isFolder = false;
            file.icon = getFileIcon(file.name);
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

const showCreateFileDialog = () => {
    state.createFileDialog.data = {};
    state.createFileDialog.visible = true;
};

const createFile = async () => {
    const name = state.createFileDialog.name;
    const type = state.createFileDialog.type;
    await doCreateFile(name, type);
    closeCreateFileDialog();
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

function addFinderToList() {
    folderUploadRef.value?.click();
}

function handleFolderUpload(e: Event) {
    const files = (e.target as HTMLInputElement).files;
    if (!files || files.length === 0) {
        return;
    }
    doFolderUpload(files);
    // 清空已选择的文件夹
    const folderEle: HTMLInputElement | null = document.getElementById('folderUploadInput') as HTMLInputElement | null;
    if (folderEle) {
        folderEle.value = '';
    }
}

const uploadSuccess = (res: Record<string, unknown>) => {
    if (res.code !== 200) {
        Msg.error(res.msg as string);
    }
};

const beforeUpload = (file: File) => {
    return checkUploadFileSize(file.size);
};

defineExpose({ showFileContent, onRefresh: refresh });
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
