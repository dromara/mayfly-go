<template>
    <div>
        <div class="toolbar">
            <div style="float: left">
                <el-button v-auth="'machine:add'" type="primary" icon="el-icon-plus" size="mini" @click="openFormDialog(false)" plain>添加</el-button>
                <el-button
                    v-auth="'machine:update'"
                    type="primary"
                    icon="el-icon-edit"
                    size="mini"
                    :disabled="currentId == null"
                    @click="openFormDialog(currentData)"
                    plain
                    >编辑</el-button
                >
                <el-button
                    v-auth="'machine:del'"
                    :disabled="currentId == null"
                    @click="deleteMachine(currentId)"
                    type="danger"
                    icon="el-icon-delete"
                    size="mini"
                    >删除</el-button
                >
                <el-button v-auth="'machine:file'" type="success" :disabled="currentId == null" @click="fileManage(currentData)" size="mini" plain
                    >文件管理</el-button
                >
            </div>

            <div style="float: right">
                <el-input placeholder="host" size="mini" style="width: 140px" v-model="params.host" @clear="search" plain clearable></el-input>
                <el-button @click="search" type="success" icon="el-icon-search" size="mini"></el-button>
            </div>
        </div>

        <el-table :data="data.list" border stripe style="width: 100%" @current-change="choose">
            <el-table-column label="选择" width="55px">
                <template #default="scope">
                    <el-radio v-model="currentId" :label="scope.row.id">
                        <i></i>
                    </el-radio>
                </template>
            </el-table-column>
            <el-table-column prop="name" label="名称" width></el-table-column>
            <el-table-column prop="ip" label="IP" width></el-table-column>
            <el-table-column prop="port" label="端口" :min-width="40"></el-table-column>
            <el-table-column prop="username" label="用户名" :min-width="40"></el-table-column>
            <el-table-column prop="createTime" label="创建时间" :min-width="100">
                <template #default="scope">
                    {{ $filters.dateFormat(scope.row.createTime) }}
                </template>
            </el-table-column>
            <el-table-column prop="creator" label="创建者" :min-width="50"></el-table-column>
            <el-table-column prop="updateTime" label="更新时间" :min-width="100">
                <template #default="scope">
                    {{ $filters.dateFormat(scope.row.updateTime) }}
                </template>
            </el-table-column>
            <el-table-column prop="modifier" label="修改者" :min-width="50"></el-table-column>
            <el-table-column label="操作" min-width="200px">
                <template #default="scope">
                    <el-button type="primary" @click="monitor(scope.row.id)" icom="el-icon-tickets" size="mini" plain>监控</el-button>
                    <el-button type="success" @click="serviceManager(scope.row)" size="mini" plain>脚本管理</el-button>
                    <el-button v-auth="'machine:terminal'" type="success" @click="showTerminal(scope.row)" size="mini" plain>终端</el-button>
                </template>
            </el-table-column>
        </el-table>

        <el-pagination
            style="text-align: center"
            background
            layout="prev, pager, next, total, jumper"
            :total="data.total"
            v-model:current-page="params.pageNum"
            :page-size="params.pageSize"
        />

        <!-- <el-dialog @close="closeMonitor" title="监控信息" v-model="monitorDialog.visible" width="60%">
			<monitor ref="monitorDialogRef" :machineId="monitorDialog.machineId" />
		</el-dialog> -->

        <service-manage :title="serviceDialog.title" v-model:visible="serviceDialog.visible" v-model:machineId="serviceDialog.machineId" />

        <file-manage :title="fileDialog.title" v-model:visible="fileDialog.visible" v-model:machineId="fileDialog.machineId" />

        <dynamic-form-dialog
            v-model:visible="formDialog.visible"
            :title="formDialog.title"
            :formInfo="formDialog.formInfo"
            v-model:formData="formDialog.formData"
            @submitSuccess="submitSuccess"
        ></dynamic-form-dialog>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, onMounted, defineComponent } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage } from 'element-plus';
import { DynamicFormDialog } from '@/components/dynamic-form';
// import Monitor from './Monitor.vue';
import { machineApi } from './api';
import SshTerminal from './SshTerminal.vue';
import ServiceManage from './ServiceManage.vue';
import FileManage from './FileManage.vue';

export default defineComponent({
    name: 'MachineList',
    components: {
        // Monitor,
        SshTerminal,
        ServiceManage,
        FileManage,
        DynamicFormDialog,
    },
    setup() {
        const router = useRouter();
        // const monitorDialogRef = ref();
        const state = reactive({
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
            fileDialog: {
                visible: false,
                machineId: 0,
                title: '',
            },
            monitorDialog: {
                visible: false,
                machineId: 0,
            },
            formDialog: {
                visible: false,
                title: '',
                formInfo: {
                    createApi: machineApi.save,
                    updateApi: machineApi.save,
                    formRows: [
                        [
                            {
                                type: 'input',
                                label: '名称：',
                                name: 'name',
                                placeholder: '请输入名称',
                                rules: [
                                    {
                                        required: true,
                                        message: '请输入名称',
                                        trigger: ['blur', 'change'],
                                    },
                                ],
                            },
                        ],
                        [
                            {
                                type: 'input',
                                label: 'ip：',
                                name: 'ip',
                                placeholder: '请输入ip',
                                rules: [
                                    {
                                        required: true,
                                        message: '请输入ip',
                                        trigger: ['blur', 'change'],
                                    },
                                ],
                            },
                        ],
                        [
                            {
                                type: 'input',
                                label: '端口号：',
                                name: 'port',
                                placeholder: '请输入端口号',
                                inputType: 'number',
                                rules: [
                                    {
                                        required: true,
                                        message: '请输入ip',
                                        trigger: ['blur', 'change'],
                                    },
                                ],
                            },
                        ],
                        [
                            {
                                type: 'input',
                                label: '用户名：',
                                name: 'username',
                                placeholder: '请输入用户名',
                                rules: [
                                    {
                                        required: true,
                                        message: '请输入用户名',
                                        trigger: ['blur', 'change'],
                                    },
                                ],
                            },
                        ],
                        [
                            {
                                type: 'input',
                                label: '密码：',
                                name: 'password',
                                placeholder: '请输入密码',
                                inputType: 'password',
                            },
                        ],
                    ],
                },
                formData: { port: 22 },
            },
        });

        onMounted(() => {
            search();
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
            // router.push(`/machine/${row.id}/terminal?id=${row.id}&name=${row.name}&time=${new Date().getTime()}`);
            const { href } = router.resolve({
                path: `/machine/terminal`,
                query: {
                    id: row.id,
                    name: row.name,
                },
            });
            window.open(href, '_blank');
        };

        const openFormDialog = (redis: any) => {
            let dialogTitle;
            if (redis) {
                state.formDialog.formData = state.currentData as any;
                dialogTitle = '编辑机器';
            } else {
                state.formDialog.formData = { port: 22 };
                dialogTitle = '添加机器';
            }

            state.formDialog.title = dialogTitle;
            state.formDialog.visible = true;
        };

        const deleteMachine = async (id: number) => {
            await machineApi.del.request({ id });
            ElMessage.success('操作成功');
            search();
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

        return {
            ...toRefs(state),
            choose,
            // monitor,
            // closeMonitor,
            showTerminal,
            openFormDialog,
            deleteMachine,
            serviceManager,
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
