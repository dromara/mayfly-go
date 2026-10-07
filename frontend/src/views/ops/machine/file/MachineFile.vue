<template>
    <div class="machine-file h-full">
        <div class="h-full flex flex-col">
            <!-- 工具行：位置在左，目录级动作在右（过滤框在名称列头，不占整行） -->
            <div class="file-toolbar mb-2 flex flex-wrap items-center gap-2">
                <el-breadcrumb class="min-w-0 flex-1" separator-icon="ArrowRight">
                    <el-breadcrumb-item v-for="path in filePathNav" :key="path">
                        <el-link @click="setFiles(path.path)" class="cursor-pointer! font-bold!">{{ path.name }}</el-link>
                    </el-breadcrumb-item>
                </el-breadcrumb>

                <div class="tool-group flex items-center">
                    <el-tooltip :content="$t('common.refresh')" placement="top">
                        <el-button class="tool-btn" text :aria-label="$t('common.refresh')" icon="Refresh" @click="refresh()" />
                    </el-tooltip>

                    <!-- 文件&文件夹上传 -->
                    <el-dropdown class="machine-file-upload-exec" trigger="click" size="small">
                        <span>
                            <el-tooltip :content="$t('machine.fileUpload')" placement="top">
                                <el-button v-auth="FILE_PERM.upload" class="tool-btn" text :aria-label="$t('machine.fileUpload')" icon="Upload" />
                            </el-tooltip>
                        </span>
                        <template #dropdown>
                            <el-dropdown-menu>
                                <el-dropdown-item>
                                    <el-upload
                                        :before-upload="beforeUpload"
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
                                            ref="folderUploadRef"
                                            type="file"
                                            webkitdirectory
                                            directory
                                            :multiple="true"
                                            @change="handleFolderUpload"
                                            class="hidden"
                                        />
                                    </div>
                                </el-dropdown-item>
                            </el-dropdown-menu>
                        </template>
                    </el-dropdown>

                    <el-tooltip :content="$t('machine.createName')" placement="top">
                        <el-button
                            v-auth="FILE_PERM.write"
                            class="tool-btn"
                            text
                            :aria-label="$t('machine.createName')"
                            icon="FolderAdd"
                            @click="showCreateFileDialog()"
                        />
                    </el-tooltip>

                    <!-- 操作说明：hover 即出弹层，不再另套 tooltip（两层弹层会互抢焦点） -->
                    <el-popover placement="bottom-end" :width="320" trigger="hover">
                        <template #reference>
                            <el-button class="tool-btn" text :aria-label="$t('machine.fileOpTips')" icon="QuestionFilled" />
                        </template>
                        <p class="text-xs leading-6 text-muted-foreground">{{ $t('machine.fileOpTipsContent') }}</p>
                    </el-popover>
                </div>
            </div>

            <!-- 文件列表：容器可聚焦，为 F2 / Enter 行级快捷键提供落点 -->
            <div class="file-table-wrap min-h-0 flex-1 overflow-auto" tabindex="0" @keydown="onTableKeydown">
                <el-table
                    ref="fileTableRef"
                    row-key="path"
                    :data="filterFiles"
                    height="100%"
                    highlight-current-row
                    v-loading="loading"
                    @row-contextmenu="onRowContextmenu"
                    @selection-change="handleSelectionChange"
                    @current-change="onCurrentRowChange"
                >
                    <el-table-column type="selection" width="45" />

                    <!-- 文件名：过滤框放在列头右侧，省掉整行高度 -->
                    <el-table-column prop="name" min-width="320">
                        <template #header>
                            <div class="flex items-center justify-between gap-2">
                                <span class="shrink-0 whitespace-nowrap">{{ $t('common.name') }}</span>
                                <!-- 宽度必须包在外层 div 上：EP 的 .el-input { width: 100% } 会覆盖写在组件上的宽度工具类，
                                     否则输入框撑满整格把「名称」挤到换行 -->
                                <div class="w-[150px] shrink-0">
                                    <el-input v-model="fileNameFilter" size="small" :placeholder="$t('machine.fileNameFilterPlaceholder')" clearable />
                                </div>
                            </div>
                        </template>

                        <template #default="scope">
                            <div class="file-name-cell flex min-w-0 cursor-pointer items-center gap-1.5" :title="scope.row.path" @click="getFile(scope.row)">
                                <SvgIcon
                                    :size="15"
                                    :name="scope.row.icon"
                                    :color="scope.row.isFolder ? 'var(--el-color-primary)' : undefined"
                                    class="shrink-0"
                                />

                                <span class="min-w-0 flex-1">
                                    <div v-if="scope.row.nameEdit" :class="renameEditClass">
                                        <el-input
                                            v-model="scope.row.name"
                                            size="small"
                                            :ref="focusRenameInput"
                                            @keyup.enter="submitRename"
                                            @keyup.esc="cancelRename"
                                        />
                                    </div>
                                    <span v-else class="block truncate font-medium">{{ scope.row.name }}</span>
                                </span>
                            </div>
                        </template>
                    </el-table-column>

                    <el-table-column prop="size" :label="$t('machine.size')" min-width="90" sortable>
                        <template #default="scope">
                            <span v-if="isFile(scope.row) || scope.row.dirSize" class="font-mono text-xs">{{
                                isFile(scope.row) ? formatByteSize(scope.row.size) : scope.row.dirSize
                            }}</span>
                            <el-button
                                v-else
                                text
                                size="small"
                                :loading="scope.row.loadingDirSize"
                                :aria-label="$t('machine.calculate')"
                                @click="getDirSize(scope.row)"
                            >
                                {{ $t('machine.calculate') }}
                            </el-button>
                        </template>
                    </el-table-column>

                    <el-table-column prop="mode" :label="$t('machine.attribute')" width="110" class-name="font-mono text-xs"> </el-table-column>

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

                    <el-table-column :label="$t('common.operation')" width="120">
                        <template #default="scope">
                            <div class="tool-group flex items-center">
                                <!-- 文件详情：弹层打开时才拉 stat，单一入口 -->
                                <el-popover
                                    placement="top-start"
                                    :title="`${scope.row.path} - ${$t('machine.fileDetail')}`"
                                    :width="520"
                                    trigger="click"
                                    @show="showFileStat(scope.row)"
                                >
                                    <template #reference>
                                        <el-button
                                            class="tool-btn"
                                            text
                                            :loading="scope.row.loadingStat"
                                            :aria-label="$t('machine.fileDetail')"
                                            icon="InfoFilled"
                                        />
                                    </template>
                                    <pre class="file-stat">{{ scope.row.stat }}</pre>
                                </el-popover>

                                <!-- 行内动作：清单与权限/适用条件均来自声明表，新增操作无需改这里 -->
                                <el-tooltip v-for="action in rowActions(scope.row)" :key="action.id" :content="$t(action.labelKey)" placement="top">
                                    <el-button
                                        class="tool-btn"
                                        :class="{ 'tool-btn--danger': action.danger }"
                                        text
                                        :aria-label="$t(action.labelKey)"
                                        :icon="action.icon"
                                        @click="action.run([scope.row])"
                                    />
                                </el-tooltip>
                            </div>
                        </template>
                    </el-table-column>

                    <!-- 空态分叉：目录真空与过滤无命中是两件不同的事 -->
                    <template #empty>
                        <div v-if="fileNameFilter" class="py-6">
                            <p class="text-sm">{{ $t('machine.emptyFilter', { keyword: fileNameFilter }) }}</p>
                            <el-button class="mt-2" size="small" @click="fileNameFilter = ''">{{ $t('machine.clearFilter') }}</el-button>
                        </div>
                        <p v-else class="py-6 text-sm text-muted-foreground">{{ $t('machine.emptyDir') }}</p>
                    </template>
                </el-table>
            </div>

            <!-- 批量动作条：只在有可执行动作或有剪贴板内容时出现，平时不占行 -->
            <div v-if="copyOrMvFile.paths.length > 0 || batchActions.length > 0" class="list-bar mt-1">
                <template v-if="copyOrMvFile.paths.length > 0">
                    <el-tooltip effect="customized" raw-content placement="top">
                        <template #content>
                            <div v-for="path in copyOrMvFile.paths" :key="path">{{ path }}</div>
                        </template>
                        <el-button size="small" type="primary" @click="pasteFile">
                            {{ isCpFile() ? $t('machine.copy') : $t('machine.move') }}{{ $t('machine.paste') }}{{ copyOrMvFile.paths.length }}
                        </el-button>
                    </el-tooltip>
                    <el-button size="small" text icon="CloseBold" @click="cancelCopy">{{ $t('common.cancel') }}</el-button>
                </template>
                <template v-else>
                    <el-button
                        v-for="action in batchActions"
                        :key="action.id"
                        size="small"
                        :type="action.danger ? 'danger' : 'default'"
                        :plain="action.danger"
                        :icon="action.icon"
                        @click="action.run(selectionFiles)"
                    >
                        {{ $t(action.labelKey) }}
                    </el-button>
                </template>
            </div>
        </div>

        <el-dialog
            :destroy-on-close="true"
            :title="$t('machine.createName')"
            v-model="createFileDialog.visible"
            :before-close="closeCreateFileDialog"
            :close-on-click-modal="false"
            top="5vh"
            width="400px"
        >
            <auto-form v-model="createFileDialog" :items="createFileItems" label-width="auto" />

            <!-- 把目标路径摆在眼前：否则用户无法确认「确定」之后东西建到哪了 -->
            <p class="mt-1 pl-2 text-xs text-muted-foreground">{{ $t('machine.createTo', { path: nowPath }) }}</p>

            <template #footer>
                <div>
                    <el-button @click="closeCreateFileDialog">{{ $t('common.cancel') }}</el-button>
                    <el-button v-auth="FILE_PERM.write" type="primary" @click="createFile">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-dialog>

        <!-- 行右键菜单：与资源树同一交互约定，重命名入口不再占用操作列图标 -->
        <Contextmenu ref="contextmenuRef" :items="fileMenuItems" :dropdown="dropdown" />

        <machine-file-content
            v-model:visible="fileContent.contentVisible"
            :machine-id="machineId"
            :auth-cert-name="props.authCertName"
            :file-id="fileId"
            :path="fileContent.path"
            :protocol="protocol"
            @saved="refresh()"
        />
    </div>
</template>

<script lang="ts" setup>
import { Msg } from '@/hooks/useI18n';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import { ElInput } from 'element-plus';
import { computed, defineAsyncComponent, getCurrentInstance, onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';
import { machineApi } from '../api';

import { isTrue } from '@/common/assert';
import { Contextmenu } from '@/components/contextmenu';
import { getMachineConfig } from '@/common/sysconfig';
import { formatByteSize } from '@/common/utils/format';
import { getToken } from '@/common/utils/storage';
import { fuzzyMatchField } from '@/common/utils/string';
import { useI18n } from 'vue-i18n';
import type { ComponentPublicInstance } from 'vue';
import { MachineProtocolEnum } from '../enums';
import type { FileRowVM, MachineFileInfo, MachineUserInfo, MachineGroupInfo } from '../types';
import { getFileIcon } from './utils/fileIcon';
import { useFileOperations, isDir, isFile } from './composables/useFileOperations';
import { useFileActions } from './composables/useFileActions';
import { renameSelectRange, useFileRename } from './composables/useFileRename';
import { DIR_TYPE, FILE_PERM, PATH_SEP, PREVIEW_MAX_SIZE } from './constants';

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

const userMap = ref(new Map<number, MachineUserInfo>());
const groupMap = ref(new Map<number, MachineGroupInfo>());

const state = reactive({
    basePath: '', // 基础路径
    nowPath: '', // 当前路径
    loading: true,
    fileNameFilter: '',
    files: [] as FileRowVM[],
    selectionFiles: [] as FileRowVM[],
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
        type: DIR_TYPE,
        data: null as Record<string, unknown> | null,
    },
    machineConfig: { uploadMaxFileSize: '1GB' },
});

const { loading, fileNameFilter, fileContent, createFileDialog, selectionFiles, nowPath } = toRefs(state);

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

// 行内重命名状态机：提交委派给 useFileOperations 的 rename（成功会刷新列表）
const {
    start: startRename,
    submit: submitRename,
    cancel: cancelRename,
    close: closeRename,
    editClass: renameEditClass,
} = useFileRename({
    rename: (row, oldname) => doFileRename(row, oldname),
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

/**
 * 重命名输入框渲染后聚焦，并在首次拿到焦点时预选文件名主干（不含扩展名），改主名不误伤后缀。
 *
 * 函数式 ref 会随单元格重渲染被重放，所以只在输入框尚未聚焦时设选区；否则用户每敲一个
 * 字符选区都会被抢回主干，名字根本改不了。
 */
const focusRenameInput = (el: Element | ComponentPublicInstance | null) => {
    const input = el as InstanceType<typeof ElInput> | null;
    const nativeEl = input?.$el?.querySelector('input') as HTMLInputElement | null;
    if (!nativeEl) {
        return;
    }
    const hadFocus = document.activeElement === nativeEl;
    input?.focus();
    if (hadFocus) {
        return;
    }
    const [start, end] = renameSelectRange(nativeEl.value);
    nativeEl.setSelectionRange(start, end);
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

    const paths = state.nowPath.split(PATH_SEP).splice(1);
    let nowPath = '';
    for (let path of paths) {
        if (!nowPath) {
            nowPath = PATH_SEP + path;
        } else {
            nowPath = nowPath + PATH_SEP + path;
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

const handleSelectionChange = (val: FileRowVM[]) => {
    state.selectionFiles = val;
};

// 当前高亮行：键盘入口（F2 重命名 / Enter 打开）作用于它，与鼠标右键菜单共用同一套动作实现
const currentRow = ref<FileRowVM | null>(null);

const onCurrentRowChange = (row: FileRowVM | null) => {
    currentRow.value = row;
};

const onTableKeydown = (event: KeyboardEvent) => {
    const row = currentRow.value;
    if (!row || row.nameEdit) {
        return;
    }
    if (event.key === 'F2') {
        event.preventDefault();
        startRename(row);
    } else if (event.key === 'Enter') {
        event.preventDefault();
        getFile(row);
    }
};

// 行右键菜单：项均为文字表述，入口与资源树同一交互约定
const contextmenuRef = useTemplateRef<InstanceType<typeof Contextmenu>>('contextmenuRef');
const dropdown = ref({ x: 0, y: 0 });

// 操作清单单一真源：行内动作组、右键菜单、批量动作条均从 useFileActions 的声明表派生
const { actionsOn, menuItems } = useFileActions({
    rename: startRename,
    download: downloadFile,
    remove: deleteFile,
    copy: copyFile,
    move: mvFile,
    isProtected: dontOperate,
});

const rowActions = (row: FileRowVM) => actionsOn('row', [row]);

const batchActions = computed(() => actionsOn('batch', state.selectionFiles));

const fileMenuItems = menuItems();

/** el-table 的 row-contextmenu 签名为 (row, column, event)，不像 cell-* 事件会传 cell，多取一个形参就会拿不到 event */
const onRowContextmenu = (row: FileRowVM, _column: unknown, event: MouseEvent) => {
    // 重命名编辑态下不抢占输入
    if (row.nameEdit) {
        return;
    }
    // 必须取消默认行为，否则浏览器原生菜单会弹出并抢焦点把自绘菜单直接 dismiss
    event.preventDefault();
    dropdown.value = { x: event.clientX, y: event.clientY };
    contextmenuRef.value?.openContextmenu(row);
};

const showFileContent = async (path: string) => {
    state.fileContent.dialogTitle = path;
    state.fileContent.path = path;
    state.fileContent.contentVisible = true;
};

const getFile = async (row: FileRowVM) => {
    // 行处于重命名编辑态时，单元格内的点击不作为打开/预览操作
    if (row.nameEdit) {
        return;
    }
    if (isDir(row)) {
        await setFiles(row.path);
    } else {
        isTrue(row.size < PREVIEW_MAX_SIZE, 'machine.fileTooLargeTips');
        await showFileContent(row.path);
    }
};

// 目录请求代际号：连续切目录时响应可能乱序返回，只允许最新一次请求落库
let filesRequestSeq = 0;

const setFiles = async (path: string) => {
    const target = path || PATH_SEP;
    // 切目录会重建行对象，未提交的编辑态在此丢弃（同时摘掉全局 pointerdown 监听）
    closeRename();
    state.fileNameFilter = '';
    const seq = ++filesRequestSeq;
    state.loading = true;
    try {
        const files = await lsFile(target);
        if (seq !== filesRequestSeq) {
            // 已被更新的请求取代，丢弃迟到的响应；否则会停在旧目录内容但路径已是新目录
            return;
        }
        // 行对象整体替换，配合表格 row-key 定位行；不在请求前清空旧数据，避免白闪与丢勾选
        state.files = files;
        state.nowPath = target;
    } finally {
        if (seq === filesRequestSeq) {
            state.loading = false;
        }
    }
};

const lsFile = async (path: string): Promise<FileRowVM[]> => {
    const res = await machineApi.lsFile.request({
        fileId: props.fileId,
        machineId: props.machineId,
        authCertName: props.authCertName,
        protocol: props.protocol,
        path,
    });
    // 后端只给事实，渲染与交互态在此一次性补齐，行视图模型的字段因此始终有值
    return res.map((file) => ({
        ...file,
        isFolder: isDir(file),
        icon: isDir(file) ? 'folder' : getFileIcon(file.name),
        nameEdit: false,
        dirSize: '',
        loadingDirSize: false,
        stat: '',
        loadingStat: false,
    }));
};

const refresh = async () => {
    setFiles(state.nowPath);
};

const showCreateFileDialog = () => {
    state.createFileDialog.data = {};
    state.createFileDialog.visible = true;
};

/** 新建文件/目录表单声明 */
const createFileItems: AutoFormItem[] = [
    { prop: 'name', label: 'common.name', required: true, props: { autocomplete: 'off' } },
    {
        prop: 'type',
        label: 'common.type',
        type: 'radio',
        options: [
            { label: 'machine.directory', value: 'd' },
            { label: 'machine.file', value: '-' },
        ],
    },
];

const createFile = async () => {
    const name = state.createFileDialog.name;
    const type = state.createFileDialog.type;
    try {
        await doCreateFile(name, type);
    } catch {
        // 校验或接口失败时弹层保持打开，用户可就地改名重试（错误文案由提示层给出）
        return;
    }
    closeCreateFileDialog();
    // 明确说出建了什么、建在哪，列表刷新后新行夹在中间，没提示就等于没反馈
    Msg.success(type === DIR_TYPE ? 'machine.createDirSuccess' : 'machine.createFileSuccess', { name });
};

const closeCreateFileDialog = () => {
    state.createFileDialog.visible = false;
    state.createFileDialog.data = null;
    state.createFileDialog.name = '';
    state.createFileDialog.type = DIR_TYPE;
};

function addFinderToList() {
    folderUploadRef.value?.click();
}

function handleFolderUpload(e: Event) {
    const input = e.target as HTMLInputElement;
    const files = input.files;
    if (!files || files.length === 0) {
        return;
    }
    doFolderUpload(files);
    // 清空已选择的文件夹，否则同一目录再选一次不会触发 change
    input.value = '';
}

const beforeUpload = (file: File) => {
    return checkUploadFileSize(file.size);
};

defineExpose({ showFileContent, onRefresh: refresh });
</script>
<style lang="scss" scoped>
@use '@/theme/common/ops-toolbar.scss' as *;

.machine-file-upload-exec {
    display: inline-flex;
    align-items: center;
    vertical-align: middle;
}

// 键盘入口需要看得到焦点在哪
.file-table-wrap:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: -2px;
}

// stat 原文是多行定长字段对齐输出，用 pre 保留空白并允许换行
.file-stat {
    max-height: 320px;
    margin: 0;
    overflow: auto;
    color: var(--el-text-color-regular);
    font-family: var(--font-mono, ui-monospace, monospace);
    font-size: 12px;
    line-height: 1.6;
    white-space: pre-wrap;
    word-break: break-all;
}
</style>
