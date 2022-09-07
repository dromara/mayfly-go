<template>
    <div>
        <el-card>
            <div>
                <el-button v-auth="'machine:add'" type="primary" icon="plus" @click="openFormDialog(false)" plain>添加</el-button>
                <el-button
                    v-auth="'machine:update'"
                    type="primary"
                    icon="edit"
                    :disabled="currentId == null"
                    @click="openFormDialog(currentData)"
                    plain
                    >编辑</el-button
                >
                <el-button v-auth="'machine:del'" :disabled="currentId == null" @click="deleteMachine(currentId)" type="danger" icon="delete"
                    >删除</el-button
                >
                <div style="float: right">
                    <el-select @focus="getProjects" v-model="params.projectId" placeholder="请选择项目" @clear="search" filterable clearable>
                        <el-option v-for="item in projects" :key="item.id" :label="`${item.name} [${item.remark}]`" :value="item.id"> </el-option>
                    </el-select>
                    <el-input
                        class="ml5"
                        placeholder="请输入名称"
                        style="width: 150px"
                        v-model="params.name"
                        @clear="search"
                        plain
                        clearable
                    ></el-input>
                    <el-input class="ml5" placeholder="请输入ip" style="width: 150px" v-model="params.ip" @clear="search" plain clearable></el-input>
                    <el-button class="ml5" @click="search" type="success" icon="search"></el-button>
                </div>
            </div>

            <el-table :data="data.list" stripe style="width: 100%" @current-change="choose">
                <el-table-column label="选择" width="55px">
                    <template #default="scope">
                        <el-radio v-model="currentId" :label="scope.row.id">
                            <i></i>
                        </el-radio>
                    </template>
                </el-table-column>
                <el-table-column prop="name" label="名称" min-width="140" show-overflow-tooltip></el-table-column>
                <el-table-column prop="ip" label="ip:port" min-width="150">
                    <template #default="scope">
                        <el-link :disabled="scope.row.status == -1" @click="showMachineStats(scope.row)" type="primary" :underline="false">{{
                            `${scope.row.ip}:${scope.row.port}`
                        }}</el-link>
                    </template>
                </el-table-column>
                <el-table-column prop="status" label="状态" min-width="75">
                    <template #default="scope">
                        <el-switch
                            v-auth:disabled="'machine:update'"
                            :width="47"
                            v-model="scope.row.status"
                            :active-value="1"
                            :inactive-value="-1"
                            inline-prompt
                            active-text="启用"
                            inactive-text="停用"
                            style="--el-switch-on-color: #13ce66; --el-switch-off-color: #ff4949"
                            @change="changeStatus(scope.row)"
                        ></el-switch>
                    </template>
                </el-table-column>
                <el-table-column prop="username" label="用户名" min-width="90"></el-table-column>
                <el-table-column prop="projectName" label="项目" min-width="120"></el-table-column>
                <el-table-column prop="remark" label="备注" min-width="250" show-overflow-tooltip></el-table-column>
                <el-table-column prop="createTime" label="创建时间" min-width="165">
                    <template #default="scope">
                        {{ $filters.dateFormat(scope.row.createTime) }}
                    </template>
                </el-table-column>
                <el-table-column prop="creator" label="创建者" min-width="80"></el-table-column>
                <el-table-column label="操作" min-width="335" fixed="right">
                    <template #default="scope">
                        <span v-auth="'machine:terminal'">
                            <el-link
                                :disabled="scope.row.status == -1"
                                type="primary"
                                @click="showTerminal(scope.row)"
                                plain
                                size="small"
                                :underline="false"
                                >终端</el-link
                            >
                            <el-divider direction="vertical" border-style="dashed" />
                        </span>

                        <span v-auth="'machine:update'" v-if="scope.row.enableRecorder == 1">
                            <el-link @click="showRec(scope.row)" plain :underline="false" size="small">终端回放</el-link>
                            <el-divider direction="vertical" border-style="dashed" />
                        </span>

                        <span v-auth="'machine:file'">
                            <el-link
                                type="success"
                                :disabled="scope.row.status == -1"
                                @click="fileManage(scope.row)"
                                plain
                                size="small"
                                :underline="false"
                                >文件</el-link
                            >
                            <el-divider direction="vertical" border-style="dashed" />
                        </span>

                        <el-link
                            :disabled="scope.row.status == -1"
                            type="warning"
                            @click="serviceManager(scope.row)"
                            plain
                            size="small"
                            :underline="false"
                            >脚本</el-link
                        >
                        <el-divider direction="vertical" border-style="dashed" />

                        <el-link @click="showProcess(scope.row)" :disabled="scope.row.status == -1" plain :underline="false" size="small"
                            >进程</el-link
                        >
                        <el-divider direction="vertical" border-style="dashed" />

                        <el-link
                            :disabled="!scope.row.hasCli || scope.row.status == -1"
                            type="danger"
                            @click="closeCli(scope.row)"
                            plain
                            size="small"
                            :underline="false"
                            >关闭连接</el-link
                        >
                    </template>
                </el-table-column>
            </el-table>
            <el-row style="margin-top: 20px" type="flex" justify="end">
                <el-pagination
                    style="text-align: right"
                    :total="data.total"
                    layout="prev, pager, next, total, jumper"
                    v-model:current-page="params.pageNum"
                    :page-size="params.pageSize"
                    @current-change="handlePageChange"
                ></el-pagination>
            </el-row>
        </el-card>

        <machine-edit
            :title="machineEditDialog.title"
            :projects="projects"
            v-model:visible="machineEditDialog.visible"
            v-model:machine="machineEditDialog.data"
            @valChange="submitSuccess"
        ></machine-edit>

        <process-list v-model:visible="processDialog.visible" v-model:machineId="processDialog.machineId" />

        <service-manage :title="serviceDialog.title" v-model:visible="serviceDialog.visible" v-model:machineId="serviceDialog.machineId" />

        <file-manage :title="fileDialog.title" v-model:visible="fileDialog.visible" v-model:machineId="fileDialog.machineId" />

        <machine-stats
            v-model:visible="machineStatsDialog.visible"
            :machineId="machineStatsDialog.machineId"
            :title="machineStatsDialog.title"
        ></machine-stats>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, onMounted, defineComponent } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage, ElMessageBox } from 'element-plus';
import { machineApi } from './api';
import { projectApi } from '../project/api.ts';
import ServiceManage from './ServiceManage.vue';
import FileManage from './FileManage.vue';
import MachineEdit from './MachineEdit.vue';
import ProcessList from './ProcessList.vue';
import MachineStats from './MachineStats.vue';

export default defineComponent({
    name: 'MachineList',
    components: {
        ServiceManage,
        ProcessList,
        FileManage,
        MachineEdit,
        MachineStats,
    },
    setup() {
        const router = useRouter();
        const state = reactive({
            projects: [],
            stats: '',
            params: {
                pageNum: 1,
                pageSize: 10,
                ip: null,
                name: null,
            },
            // 列表数据
            data: {
                list: [],
                total: 10,
            },
            // 当前选中数据id
            currentId: null,
            currentData: null,
            serviceDialog: {
                visible: false,
                machineId: 0,
                title: '',
            },
            processDialog: {
                visible: false,
                machineId: 0,
            },
            fileDialog: {
                visible: false,
                machineId: 0,
                title: '',
            },
            machineStatsDialog: {
                visible: false,
                stats: null,
                title: '',
                machineId: 0,
            },
            machineEditDialog: {
                visible: false,
                data: null,
                title: '新增机器',
            },
            machineRecDialog: {
                visible: false,
                machineId: 0,
                title: '',
            },
        });

        onMounted(async () => {
            search();
        });

        const choose = (item: any) => {
            if (!item) {
                return;
            }
            state.currentId = item.id;
            state.currentData = item;
        };

        const showTerminal = (row: any) => {
            const { href } = router.resolve({
                path: `/machine/terminal`,
                query: {
                    id: row.id,
                    name: row.name,
                },
            });
            window.open(href, '_blank');
        };

        const closeCli = async (row: any) => {
            await ElMessageBox.confirm(`确定关闭该机器客户端连接?`, '提示', {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'warning',
            });
            await machineApi.closeCli.request({ id: row.id });
            ElMessage.success('关闭成功');
            search();
        };

        const getProjects = async () => {
            state.projects = await projectApi.accountProjects.request(null);
        };

        const openFormDialog = async (machine: any) => {
            await getProjects();
            let dialogTitle;
            if (machine) {
                state.machineEditDialog.data = state.currentData as any;
                dialogTitle = '编辑机器';
            } else {
                state.machineEditDialog.data = null;
                dialogTitle = '添加机器';
            }

            state.machineEditDialog.title = dialogTitle;
            state.machineEditDialog.visible = true;
        };

        const deleteMachine = async (id: number) => {
            try {
                await ElMessageBox.confirm(`确定删除该机器信息? 该操作将同时删除脚本及文件配置信息`, '提示', {
                    confirmButtonText: '确定',
                    cancelButtonText: '取消',
                    type: 'warning',
                });
                await machineApi.del.request({ id });
                ElMessage.success('操作成功');
                state.currentId = null;
                state.currentData = null;
                search();
            } catch (err) {}
        };

        const serviceManager = (row: any) => {
            state.serviceDialog.machineId = row.id;
            state.serviceDialog.visible = true;
            state.serviceDialog.title = `${row.name} => ${row.ip}`;
        };

        /**
         * 调整机器状态
         */
        const changeStatus = async (row: any) => {
            await machineApi.changeStatus.request({ id: row.id, status: row.status });
        };

        /**
         * 显示机器状态统计信息
         */
        const showMachineStats = async (machine: any) => {
            state.machineStatsDialog.machineId = machine.id;
            state.machineStatsDialog.title = `机器状态: ${machine.name} => ${machine.ip}`;
            state.machineStatsDialog.visible = true;
        };

        const submitSuccess = () => {
            state.currentId = null;
            state.currentData = null;
            search();
        };

        const fileManage = (currentData: any) => {
            state.fileDialog.visible = true;
            state.fileDialog.machineId = currentData.id;
            state.fileDialog.title = `${currentData.name} => ${currentData.ip}`;
        };

        const search = async () => {
            const res = await machineApi.list.request(state.params);
            state.data = res;
        };

        const handlePageChange = (curPage: number) => {
            state.params.pageNum = curPage;
            search();
        };

        const showProcess = (row: any) => {
            state.processDialog.machineId = row.id;
            state.processDialog.visible = true;
        };

        const showRec = (row: any) => {
            const { href } = router.resolve({
                path: `/machine/terminal-rec`,
                query: {
                    id: row.id,
                    name: `${row.name}[${row.ip}]-终端回放记录`,
                },
            });
            window.open(href, '_blank');
        };

        return {
            ...toRefs(state),
            choose,
            getProjects,
            showTerminal,
            openFormDialog,
            deleteMachine,
            closeCli,
            serviceManager,
            showMachineStats,
            showProcess,
            changeStatus,
            submitSuccess,
            fileManage,
            search,
            showRec,
            handlePageChange,
        };
    },
});
</script>

<style>
.el-dialog__body {
    padding: 2px 2px;
}
</style>
