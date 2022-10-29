<template>
    <div>
        <el-card>
            <div>
                <el-button v-auth="'machine:add'" type="primary" icon="plus" @click="openFormDialog(false)" plain>添加
                </el-button>
                <el-button v-auth="'machine:update'" type="primary" icon="edit" :disabled="!currentId"
                    @click="openFormDialog(currentData)" plain>编辑</el-button>
                <el-button v-auth="'machine:del'" :disabled="!currentId" @click="deleteMachine(currentId)" type="danger"
                    icon="delete">删除</el-button>
                <div style="float: right">
                    <el-select @focus="getTags" v-model="params.tagPath" placeholder="请选择标签" @clear="search" filterable
                        clearable>
                        <el-option v-for="item in tags" :key="item" :label="item" :value="item"> </el-option>
                    </el-select>
                    <el-input class="ml5" placeholder="请输入名称" style="width: 150px" v-model="params.name" @clear="search"
                        plain clearable></el-input>
                    <el-input class="ml5" placeholder="请输入ip" style="width: 150px" v-model="params.ip" @clear="search"
                        plain clearable></el-input>
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
                <el-table-column prop="tagPath" label="标签路径" min-width="150" show-overflow-tooltip></el-table-column>
                <el-table-column prop="name" label="名称" min-width="140" show-overflow-tooltip></el-table-column>
                <el-table-column prop="ip" label="ip:port" min-width="150">
                    <template #default="scope">
                        <el-link :disabled="scope.row.status == -1" @click="showMachineStats(scope.row)" type="primary"
                            :underline="false">{{
                                    `${scope.row.ip}:${scope.row.port}`
                            }}</el-link>
                    </template>
                </el-table-column>
                <el-table-column prop="status" label="状态" min-width="75">
                    <template #default="scope">
                        <el-switch v-auth:disabled="'machine:update'" :width="47" v-model="scope.row.status"
                            :active-value="1" :inactive-value="-1" inline-prompt active-text="启用" inactive-text="停用"
                            style="--el-switch-on-color: #13ce66; --el-switch-off-color: #ff4949"
                            @change="changeStatus(scope.row)"></el-switch>
                    </template>
                </el-table-column>
                <el-table-column prop="username" label="用户名" min-width="90"></el-table-column>
                <el-table-column prop="remark" label="备注" min-width="250" show-overflow-tooltip></el-table-column>
                <el-table-column prop="createTime" label="创建时间" min-width="165">
                    <template #default="scope">
                        {{ dateFormat(scope.row.createTime) }}
                    </template>
                </el-table-column>
                <el-table-column prop="creator" label="创建者" min-width="80"></el-table-column>
                <el-table-column label="操作" min-width="235" fixed="right">
                    <template #default="scope">
                        <span v-auth="'machine:terminal'">
                            <el-link :disabled="scope.row.status == -1" type="primary" @click="showTerminal(scope.row)"
                                plain size="small" :underline="false">终端</el-link>
                            <el-divider direction="vertical" border-style="dashed" />
                        </span>

                        <span v-auth="'machine:file'">
                            <el-link type="success" :disabled="scope.row.status == -1"
                                @click="showFileManage(scope.row)" plain size="small" :underline="false">文件</el-link>
                            <el-divider direction="vertical" border-style="dashed" />
                        </span>

                        <el-link :disabled="scope.row.status == -1" type="warning" @click="serviceManager(scope.row)"
                            plain size="small" :underline="false">脚本</el-link>
                        <el-divider direction="vertical" border-style="dashed" />

                        <el-dropdown>
                            <span class="el-dropdown-link-machine-list">
                                更多
                                <el-icon class="el-icon--right">
                                    <arrow-down />
                                </el-icon>
                            </span>
                            <template #dropdown>
                                <el-dropdown-menu>
                                    <el-dropdown-item>
                                        <el-link @click="showProcess(scope.row)" :disabled="scope.row.status == -1"
                                            plain :underline="false" size="small">进程</el-link>
                                    </el-dropdown-item>

                                    <el-dropdown-item v-if="scope.row.enableRecorder == 1">
                                        <el-link v-auth="'machine:update'" @click="showRec(scope.row)" plain
                                            :underline="false" size="small">终端回放</el-link>
                                    </el-dropdown-item>

                                    <el-dropdown-item>
                                        <el-link :disabled="!scope.row.hasCli || scope.row.status == -1" type="danger"
                                            @click="closeCli(scope.row)" plain size="small" :underline="false">关闭连接
                                        </el-link>
                                    </el-dropdown-item>
                                </el-dropdown-menu>
                            </template>
                        </el-dropdown>
                    </template>
                </el-table-column>
            </el-table>
            <el-row style="margin-top: 20px" type="flex" justify="end">
                <el-pagination style="text-align: right" :total="data.total" layout="prev, pager, next, total, jumper"
                    v-model:current-page="params.pageNum" :page-size="params.pageSize"
                    @current-change="handlePageChange"></el-pagination>
            </el-row>
        </el-card>

        <machine-edit :title="machineEditDialog.title" v-model:visible="machineEditDialog.visible"
            v-model:machine="machineEditDialog.data" @valChange="submitSuccess"></machine-edit>

        <process-list v-model:visible="processDialog.visible" v-model:machineId="processDialog.machineId" />

        <service-manage :title="serviceDialog.title" v-model:visible="serviceDialog.visible"
            v-model:machineId="serviceDialog.machineId" />

        <file-manage :title="fileDialog.title" v-model:visible="fileDialog.visible"
            v-model:machineId="fileDialog.machineId" />

        <machine-stats v-model:visible="machineStatsDialog.visible" :machineId="machineStatsDialog.machineId"
            :title="machineStatsDialog.title"></machine-stats>

        <machine-rec v-model:visible="machineRecDialog.visible" :machineId="machineRecDialog.machineId"
            :title="machineRecDialog.title"></machine-rec>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage, ElMessageBox } from 'element-plus';
import { machineApi } from './api';
import { tagApi } from '../tag/api.ts';
import ServiceManage from './ServiceManage.vue';
import FileManage from './FileManage.vue';
import MachineEdit from './MachineEdit.vue';
import ProcessList from './ProcessList.vue';
import MachineStats from './MachineStats.vue';
import MachineRec from './MachineRec.vue';
import { dateFormat } from '@/common/utils/date';

const router = useRouter();
const state = reactive({
    tags: [] as any,
    params: {
        pageNum: 1,
        pageSize: 10,
        ip: null,
        name: null,
        tagPath: null,
    },
    // 列表数据
    data: {
        list: [],
        total: 10,
    },
    // 当前选中数据id
    currentId: 0,
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
        data: null as any,
        title: '新增机器',
    },
    machineRecDialog: {
        visible: false,
        machineId: 0,
        title: '',
    },
});

const {
    tags,
    params,
    data,
    currentId,
    currentData,
    serviceDialog,
    processDialog,
    fileDialog,
    machineStatsDialog,
    machineEditDialog,
    machineRecDialog,
} = toRefs(state)

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

const getTags = async () => {
    state.tags = await tagApi.getAccountTags.request(null);
};

const openFormDialog = async (machine: any) => {
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
        state.currentId = 0;
        state.currentData = null;
        search();
    } catch (err) { }
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
    state.currentId = 0;
    state.currentData = null;
    search();
};

const showFileManage = (currentData: any) => {
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
    state.machineRecDialog.title = `${row.name}[${row.ip}]-终端回放记录`;
    state.machineRecDialog.machineId = row.id;
    state.machineRecDialog.visible = true;
};
</script>

<style>
.el-dialog__body {
    padding: 2px 2px;
}

.el-dropdown-link-machine-list {
    cursor: pointer;
    color: var(--el-color-primary);
    display: flex;
    align-items: center;
    margin-top: 6px;
}
</style>
