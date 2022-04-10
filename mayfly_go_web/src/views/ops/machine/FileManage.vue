<template>
    <div class="file-manage">
        <el-dialog :title="title" v-model="dialogVisible" :show-close="true" :before-close="handleClose" width="800px">
            <div class="toolbar">
                <div style="float: right">
                    <el-button v-auth="'machine:file:add'" type="primary" @click="add" icon="plus" size="small" plain>添加</el-button>
                </div>
            </div>
            <!-- <div style="float: right;">
       
      </div> -->
            <el-table :data="fileTable" stripe style="width: 100%">
                <el-table-column prop="name" label="名称" width>
                    <template #default="scope">
                        <el-input v-model="scope.row.name" size="small" :disabled="scope.row.id != null" clearable></el-input>
                    </template>
                </el-table-column>
                <el-table-column prop="name" label="类型" min-width="50px">
                    <template #default="scope">
                        <el-select :disabled="scope.row.id != null" size="small" v-model="scope.row.type" style="width: 100px" placeholder="请选择">
                            <el-option v-for="item in enums.FileTypeEnum" :key="item.value" :label="item.label" :value="item.value"></el-option>
                        </el-select>
                    </template>
                </el-table-column>
                <el-table-column prop="path" label="路径" width>
                    <template #default="scope">
                        <el-input v-model="scope.row.path" :disabled="scope.row.id != null" size="small" clearable></el-input>
                    </template>
                </el-table-column>
                <el-table-column label="操作" width>
                    <template #default="scope">
                        <el-button v-if="scope.row.id == null" @click="addFiles(scope.row)" type="success" icon="success-filled" size="small" plain
                            >确定</el-button
                        >
                        <el-button v-if="scope.row.id != null" @click="getConf(scope.row)" type="primary" icon="tickets" size="small" plain
                            >查看</el-button
                        >
                        <el-button
                            v-auth="'machine:file:del'"
                            type="danger"
                            @click="deleteRow(scope.$index, scope.row)"
                            icon="delete"
                            size="small"
                            plain
                            >删除</el-button
                        >
                    </template>
                </el-table-column>
            </el-table>
        </el-dialog>

        <el-dialog :title="tree.title" v-model="tree.visible" :close-on-click-modal="false" width="680px">
            <el-progress
                v-if="uploadProgressShow"
                style="width: 90%; margin-left: 20px"
                :text-inside="true"
                :stroke-width="20"
                :percentage="progressNum"
            />
            <div style="height: 45vh; overflow: auto">
                <el-tree v-if="tree.visible" ref="fileTree" :load="loadNode" :props="props" lazy node-key="id" :expand-on-click-node="true">
                    <template #default="{ node, data }">
                        <span class="custom-tree-node">
                            <el-dropdown size="small" @visible-change="getFilePath(data, $event)" trigger="contextmenu">
                                <span class="el-dropdown-link">
                                    <span v-if="data.type == 'd' && !node.expanded">
                                        <SvgIcon name="folder" />
                                    </span>
                                    <span v-if="data.type == 'd' && node.expanded">
                                        <SvgIcon name="folder-opened" />
                                    </span>
                                    <span v-if="data.type == '-'">
                                        <SvgIcon name="document" />
                                    </span>

                                    <span style="display: inline-block">
                                        {{ node.label }}
                                        <span style="color: #67c23a" v-if="data.type == '-'">&nbsp;&nbsp;[{{ formatFileSize(data.size) }}]</span>
                                    </span>
                                </span>

                                <template #dropdown>
                                    <el-dropdown-menu>
                                        <el-dropdown-item v-if="data.type == '-' && data.size < 1 * 1024 * 1024">
                                            <el-link
                                                @click.prevent="getFileContent(tree.folder.id, data.path)"
                                                type="info"
                                                icon="view"
                                                :underline="false"
                                            >
                                                查看
                                            </el-link>
                                        </el-dropdown-item>

                                        <span v-auth="'machine:file:upload'">
                                            <el-dropdown-item v-if="data.type == 'd'">
                                                <el-upload
                                                    :before-upload="beforeUpload"
                                                    :on-success="uploadSuccess"
                                                    action=""
                                                    :http-request="getUploadFile"
                                                    :headers="{ token }"
                                                    :show-file-list="false"
                                                    name="file"
                                                    style="display: inline-block; margin-left: 2px"
                                                >
                                                    <el-link icon="upload" :underline="false"> 上传 </el-link>
                                                </el-upload>
                                            </el-dropdown-item>
                                        </span>

                                        <span v-auth="'machine:file:write'">
                                            <el-dropdown-item v-if="data.type == '-'">
                                                <el-link
                                                    @click.prevent="downloadFile(node, data)"
                                                    type="primary"
                                                    icon="download"
                                                    :underline="false"
                                                    style="margin-left: 2px"
                                                    >下载</el-link
                                                >
                                            </el-dropdown-item>
                                        </span>

                                        <span v-auth="'machine:file:rm'">
                                            <el-dropdown-item v-if="!dontOperate(data)">
                                                <el-link
                                                    @click.prevent="deleteFile(node, data)"
                                                    type="danger"
                                                    icon="delete"
                                                    :underline="false"
                                                    style="margin-left: 2px"
                                                    >删除
                                                </el-link>
                                            </el-dropdown-item>
                                        </span>
                                    </el-dropdown-menu>
                                </template>
                            </el-dropdown>
                        </span>
                    </template>
                </el-tree>
            </div>
        </el-dialog>

        <el-dialog
            :destroy-on-close="true"
            :title="fileContent.dialogTitle"
            v-model="fileContent.contentVisible"
            :close-on-click-modal="false"
            top="5vh"
            width="70%"
        >
            <div>
                <codemirror :can-change-mode="true" ref="cmEditor" v-model="fileContent.content" :language="fileContent.type" />
            </div>

            <template #footer>
                <div class="dialog-footer">
                    <el-button v-auth="'machine:file:write'" type="primary" @click="updateContent">保 存</el-button>
                    <el-button @click="fileContent.contentVisible = false">关 闭</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts">
import { ref, toRefs, reactive, watch, defineComponent } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { machineApi } from './api';

import { codemirror } from '@/components/codemirror';
import { getSession } from '@/common/utils/storage';
import enums from './enums';
import config from '@/common/config';

export default defineComponent({
    name: 'FileManage',
    components: {
        codemirror,
    },
    props: {
        visible: { type: Boolean },
        machineId: { type: Number },
        title: { type: String },
    },

    setup(props: any, { emit }) {
        const addFile = machineApi.addConf;
        const delFile = machineApi.delConf;
        const updateFileContent = machineApi.updateFileContent;
        const files = machineApi.files;
        const fileTree: any = ref(null);
        const token = getSession('token');

        const cmOptions = {
            tabSize: 2,
            mode: 'text/x-sh',
            theme: 'panda-syntax',
            line: true,
            // 开启校验
            lint: true,
            gutters: ['CodeMirror-lint-markers'],
            indentWithTabs: true,
            smartIndent: true,
            matchBrackets: true,
            autofocus: true,
            styleSelectedText: true,
            styleActiveLine: true, // 高亮选中行
            foldGutter: true, // 块槽
            hintOptions: {
                // 当匹配只有一项的时候是否自动补全
                completeSingle: true,
            },
        };

        const state = reactive({
            dialogVisible: false,
            form: {
                id: null,
                type: null,
                name: '',
                remark: '',
            },
            fileTable: [] as any,
            btnLoading: false,
            fileContent: {
                fileId: 0,
                content: '',
                contentVisible: false,
                dialogTitle: '',
                path: '',
                type: 'shell',
            },
            tree: {
                title: '',
                visible: false,
                folder: { id: 0 },
                node: {
                    childNodes: [],
                },
                resolve: {},
            },
            props: {
                label: 'name',
                children: 'zones',
                isLeaf: 'leaf',
            },
            progressNum: 0,
            uploadProgressShow: false,
            dataObj: {
                name: '',
                path: '',
                type: '',
            },
            file: null as any,
        });

        watch(props, (newValue) => {
            if (newValue.machineId) {
                getFiles();
            }
            state.dialogVisible = newValue.visible;
        });

        const getFiles = async () => {
            const res = await files.request({ id: props.machineId });
            state.fileTable = res.list;
        };

        /**
         * tab切换触发事件
         * @param {Object} tab
         * @param {Object} event
         */
        // handleClick(tab, event) {
        //   // if (tab.name == 'file-manage') {
        //   //   this.fileManage.node.childNodes = [];
        //   //   this.loadNode(this.fileManage.node, this.fileManage.resolve);
        //   // }
        // }

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
                            state.fileTable.splice(idx, 1);
                        });
                });
            } else {
                state.fileTable.splice(idx, 1);
            }
        };

        const getConf = (row: any) => {
            if (row.type == 1) {
                state.tree.folder = row;
                state.tree.title = row.name;
                loadNode(state.tree.node, state.tree.resolve);
                state.tree.visible = true;
                return;
            }
            getFileContent(row.id, row.path);
        };

        const getFileContent = async (fileId: number, path: string) => {
            const res = await machineApi.fileContent.request({
                fileId,
                path,
                machineId: props.machineId,
            });
            state.fileContent.content = res;
            state.fileContent.fileId = fileId;
            state.fileContent.dialogTitle = path;
            state.fileContent.path = path;
            state.fileContent.type = getFileType(path);
            state.fileContent.contentVisible = true;
        };

        const getFileType = (path: string) => {
            if (path.endsWith('.sh')) {
                return 'shell';
            }
            if (path.endsWith('js') || path.endsWith('json')) {
                return 'javascript';
            }
            if (path.endsWith('Dockerfile')) {
                return 'dockerfile';
            }
            if (path.endsWith('nginx.conf')) {
                return 'nginx';
            }
            if (path.endsWith('sql')) {
                return 'sql';
            }
            if (path.endsWith('yaml') || path.endsWith('yml')) {
                return 'yaml';
            }
            if (path.endsWith('xml') || path.endsWith('html')) {
                return 'html';
            }
            return 'text';
        };

        const updateContent = async () => {
            await updateFileContent.request({
                content: state.fileContent.content,
                id: state.fileContent.fileId,
                path: state.fileContent.path,
                machineId: props.machineId,
            });
            ElMessage.success('修改成功');
            state.fileContent.contentVisible = false;
            state.fileContent.content = '';
        };

        /**
         * 关闭取消按钮触发的事件
         */
        const handleClose = () => {
            emit('update:visible', false);
            emit('update:machineId', null);
            emit('cancel');
            // state.activeName = 'conf-file'
            state.fileTable = [];
            state.tree.folder = { id: 0 };
        };

        /**
         * 加载文件树节点
         * @param {Object} node
         * @param {Object} resolve
         */
        const loadNode = async (node: any, resolve: any) => {
            if (typeof resolve !== 'function') {
                return;
            }

            const folder: any = state.tree.folder;
            if (node.level === 0) {
                state.tree.node = node;
                state.tree.resolve = resolve;

                // let folder: any = this.tree.folder
                const path = folder ? folder.path : '/';
                return resolve([
                    {
                        name: path,
                        type: 'd',
                        path: path,
                    },
                ]);
            }

            let path;
            const data = node.data;
            // 只有在第一级节点时，name==path，即上述level==0时设置的
            if (!data || data.name == data.path) {
                path = folder.path;
            } else {
                path = data.path;
            }

            const res = await machineApi.lsFile.request({
                fileId: folder.id,
                machineId: props.machineId,
                path,
            });
            for (const file of res) {
                const type = file.type;
                if (type != 'd') {
                    file.leaf = true;
                }
            }
            return resolve(res);
        };

        const deleteFile = (node: any, data: any) => {
            const file = data.path;
            ElMessageBox.confirm(`此操作将删除 [${file}], 是否继续?`, '提示', {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'warning',
            })
                .then(() => {
                    machineApi.rmFile
                        .request({
                            fileId: state.tree.folder.id,
                            path: file,
                            machineId: props.machineId,
                        })
                        .then(() => {
                            ElMessage.success('删除成功');
                            fileTree.value.remove(node);
                        });
                })
                .catch(() => {
                    // skip
                });
        };

        const downloadFile = (node: any, data: any) => {
            const a = document.createElement('a');
            // a.setAttribute('target', '_blank')
            a.setAttribute(
                'href',
                `${config.baseApiUrl}/machines/${props.machineId}/files/${state.tree.folder.id}/read?type=1&path=${data.path}&token=${token}`
            );
            a.click();
        };

        const onUploadProgress = (progressEvent: any) => {
            state.uploadProgressShow = true;
            let complete = ((progressEvent.loaded / progressEvent.total) * 100) | 0;
            state.progressNum = complete;
        };

        const getUploadFile = (content: any) => {
            const params = new FormData();
            params.append('file', content.file);
            params.append('path', state.dataObj.path);
            params.append('machineId', props.machineId);
            params.append('fileId', state.tree.folder.id as any);
            params.append('token', token);
            machineApi.uploadFile
                .request(params, {
                    url: `${config.baseApiUrl}/machines/${props.machineId}/files/${state.tree.folder.id}/upload?token=${token}`,
                    headers: { 'Content-Type': 'multipart/form-data; boundary=----WebKitFormBoundaryF1uyUD0tWdqmJqpl' },
                    onUploadProgress: onUploadProgress,
                    baseURL: '',
                    timeout: 60 * 60 * 1000,
                })
                .then(() => {
                    ElMessage.success('上传成功');
                    setTimeout(() => {
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
            // ElMessage.success(`'${file.name}' 上传中,请关注结果通知`);
        };
        const getFilePath = (data: object, visible: boolean) => {
            if (visible) {
                state.dataObj = data as any;
            }
        };
        const dontOperate = (data: any) => {
            const path = data.path;
            const ls = [
                '/',
                '//',
                '/usr',
                '/usr/',
                '/usr/bin',
                '/opt',
                '/run',
                '/etc',
                '/proc',
                '/var',
                '/mnt',
                '/boot',
                '/dev',
                '/home',
                '/media',
                '/root',
            ];
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

        return {
            ...toRefs(state),
            fileTree,
            enums,
            token,
            cmOptions,
            add,
            getFiles,
            addFiles,
            deleteRow,
            getConf,
            getFileContent,
            updateContent,
            handleClose,
            loadNode,
            deleteFile,
            downloadFile,
            getUploadFile,
            beforeUpload,
            getFilePath,
            uploadSuccess,
            dontOperate,
            formatFileSize,
        };
    },
});
</script>
<style lang="scss">
</style>
