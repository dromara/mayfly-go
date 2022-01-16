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
                <el-button v-auth="'machine:file'" type="success" icon="files" :disabled="currentId == null" @click="fileManage(currentData)" plain
                    >文件</el-button
                >
                <div style="float: right">
                    <el-select v-model="params.projectId" placeholder="请选择项目" @clear="search" filterable clearable>
                        <el-option v-for="item in projects" :key="item.id" :label="`${item.name} [${item.remark}]`" :value="item.id"> </el-option>
                    </el-select>
                    <el-input class="ml5" placeholder="请输入ip" style="width: 200px" v-model="params.ip" @clear="search" plain clearable></el-input>
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
                <el-table-column prop="name" label="名称" min-width="130"></el-table-column>
                <el-table-column prop="ip" label="ip:port" min-width="130">
                    <template #default="scope">
                        {{ `${scope.row.ip}:${scope.row.port}` }}
                    </template>
                </el-table-column>
                <el-table-column prop="username" label="用户名" min-width="75"></el-table-column>
                <el-table-column prop="projectName" label="项目" min-width="120"></el-table-column>
                <el-table-column prop="ip" label="hasCli" width="70">
                    <template #default="scope">
                        {{ `${scope.row.hasCli ? '是' : '否'}` }}
                    </template>
                </el-table-column>
                <el-table-column prop="createTime" label="创建时间" width="160">
                    <template #default="scope">
                        {{ $filters.dateFormat(scope.row.createTime) }}
                    </template>
                </el-table-column>
                <el-table-column prop="creator" label="创建者" min-width="60"></el-table-column>
                <el-table-column label="操作" min-width="260" fixed="right">
                    <template #default="scope">
                        <el-button type="success" @click="serviceManager(scope.row)" plain size="small">脚本</el-button>
                        <el-button v-auth="'machine:terminal'" type="primary" @click="showTerminal(scope.row)" plain size="small">终端</el-button>
                        <el-button @click="showProcess(scope.row)" plain size="small">进程</el-button>
                        <el-button :disabled="!scope.row.hasCli" type="danger" @click="closeCli(scope.row)" plain size="small">关闭连接</el-button>
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

        <!-- <el-dialog @close="closeMonitor" title="监控信息" v-model="monitorDialog.visible" width="60%">
			<monitor ref="monitorDialogRef" :machineId="monitorDialog.machineId" />
		</el-dialog> -->

        <process-list v-model:visible="processDialog.visible" v-model:machineId="processDialog.machineId" />

        <service-manage :title="serviceDialog.title" v-model:visible="serviceDialog.visible" v-model:machineId="serviceDialog.machineId" />

        <file-manage :title="fileDialog.title" v-model:visible="fileDialog.visible" v-model:machineId="fileDialog.machineId" />
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, onMounted, defineComponent } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage, ElMessageBox } from 'element-plus';
// import Monitor from './Monitor.vue';
import { machineApi } from './api';
import { projectApi } from '../project/api.ts';
import ServiceManage from './ServiceManage.vue';
import FileManage from './FileManage.vue';
import MachineEdit from './MachineEdit.vue';
import ProcessList from './ProcessList.vue';

export default defineComponent({
    name: 'MachineList',
    components: {
        ServiceManage,
        ProcessList,
        FileManage,
        MachineEdit,
    },
    setup() {
        const router = useRouter();
        const state = reactive({
            projects: [],
            params: {
                pageNum: 1,
                pageSize: 10,
                host: null,
                clusterId: null,
            },
            // 列表数据
            data: {
                list: [],
                total: 10,
            },
            // 当前选中数据id
            currentId: null,
            currentData: null,
            infoDialog: {
                visible: false,
                info: '',
            },
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
            monitorDialog: {
                visible: false,
                machineId: 0,
            },
            machineEditDialog: {
                visible: false,
                data: null,
                title: '新增机器',
            },
        });

        onMounted(async () => {
            search();
            state.projects = (await projectApi.projects.request({ pageNum: 1, pageSize: 100 })).list;
        });

        const choose = (item: any) => {
            if (!item) {
                return;
            }
            state.currentId = item.id;
            state.currentData = item;
        };

        // const monitor = (id: number) => {
        // 	state.monitorDialog.machineId = id;
        // 	state.monitorDialog.visible = true;
        // 	// 如果重复打开同一个则开启定时任务
        // 	const md: any = monitorDialogRef;
        // 	if (md) {
        // 		md.startInterval();
        // 	}
        // };

        // const closeMonitor = () => {
        // 	// 关闭窗口，取消定时任务
        // 	const md: any = monitorDialogRef;
        // 	md.cancelInterval();
        // };

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
            await machineApi.closeCli.request({ id: row.id });
            ElMessage.success('关闭成功');
            search();
        };

        const openFormDialog = (redis: any) => {
            let dialogTitle;
            if (redis) {
                state.machineEditDialog.data = state.currentData as any;
                dialogTitle = '编辑机器';
            } else {
                state.machineEditDialog.data = { port: 22 } as any;
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

        const showProcess = (row: any) => {
            state.processDialog.machineId = row.id;
            state.processDialog.visible = true;
        };

        return {
            ...toRefs(state),
            choose,
            // monitor,
            // closeMonitor,
            showTerminal,
            openFormDialog,
            deleteMachine,
            closeCli,
            serviceManager,
            showProcess,
            submitSuccess,
            fileManage,
            search,
        };
    },
});
</script>

<style>
.el-dialog__body {
    padding: 2px 2px;
}
</style>
