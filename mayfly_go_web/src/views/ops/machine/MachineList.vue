<template>
    <div class="machine-list">
        <page-table
            ref="pageTableRef"
            :page-api="machineApi.list"
            :before-query-fn="checkRouteTagPath"
            :search-items="searchItems"
            v-model:query-form="params"
            :show-selection="true"
            v-model:selection-data="state.selectionData"
            :columns="columns"
        >
            <template #tableHeader>
                <el-button v-auth="perms.addMachine" type="primary" icon="plus" @click="openFormDialog(false)" plain>添加 </el-button>
                <el-button v-auth="perms.delMachine" :disabled="selectionData.length < 1" @click="deleteMachine()" type="danger" icon="delete">删除</el-button>
            </template>

            <template #ipPort="{ data }">
                <el-link :disabled="data.status == -1" @click="showMachineStats(data)" type="primary" :underline="false">
                    {{ `${data.ip}:${data.port}` }}
                </el-link>
            </template>

            <template #stat="{ data }">
                <span v-if="!data.stat">-</span>
                <div v-else>
                    <el-row>
                        <el-text size="small" class="font11">
                            内存(可用/总):
                            <span :class="getStatsFontClass(data.stat.memAvailable, data.stat.memTotal)"
                                >{{ formatByteSize(data.stat.memAvailable, 1) }}/{{ formatByteSize(data.stat.memTotal, 1) }}
                            </span>
                        </el-text>
                    </el-row>
                    <el-row>
                        <el-text class="font11" size="small">
                            CPU(空闲): <span :class="getStatsFontClass(data.stat.cpuIdle, 100)">{{ data.stat.cpuIdle.toFixed(0) }}%</span>
                        </el-text>
                    </el-row>
                </div>
            </template>

            <template #fs="{ data }">
                <span v-if="!data.stat?.fsInfos">-</span>
                <div v-else>
                    <el-row v-for="(i, idx) in data.stat.fsInfos.slice(0, 2)" :key="i.mountPoint">
                        <el-text class="font11" size="small" :class="getStatsFontClass(i.free, i.used + i.free)">
                            {{ i.mountPoint }} => {{ formatByteSize(i.free, 0) }}/{{ formatByteSize(i.used + i.free, 0) }}
                        </el-text>

                        <!-- 展示剩余的磁盘信息 -->
                        <el-popover :show-after="300" v-if="data.stat.fsInfos.length > 2 && idx == 1" placement="top-start" width="230" trigger="hover">
                            <template #reference>
                                <SvgIcon class="mt5 ml5" color="var(--el-color-primary)" name="MoreFilled" />
                            </template>

                            <el-row v-for="i in data.stat.fsInfos.slice(2)" :key="i.mountPoint">
                                <el-text class="font11" size="small" :class="getStatsFontClass(i.free, i.used + i.free)">
                                    {{ i.mountPoint }} => {{ formatByteSize(i.free, 0) }}/{{ formatByteSize(i.used + i.free, 0) }}
                                </el-text>
                            </el-row>
                        </el-popover>
                    </el-row>
                </div>
            </template>

            <template #status="{ data }">
                <el-switch
                    v-auth:disabled="'machine:update'"
                    :width="52"
                    v-model="data.status"
                    :active-value="1"
                    :inactive-value="-1"
                    inline-prompt
                    active-text="启用"
                    inactive-text="停用"
                    style="--el-switch-on-color: #13ce66; --el-switch-off-color: #ff4949"
                    @change="changeStatus(data)"
                ></el-switch>
            </template>

            <template #tagPath="{ data }">
                <resource-tag :resource-code="data.code" :resource-type="TagResourceTypeEnum.Machine.value" />
            </template>

            <template #action="{ data }">
                <span v-auth="'machine:terminal'">
                    <el-tooltip :show-after="500" content="按住ctrl则为新标签打开" placement="top">
                        <el-button :disabled="data.status == -1" type="primary" @click="showTerminal(data, $event)" link>终端</el-button>
                    </el-tooltip>

                    <el-divider direction="vertical" border-style="dashed" />
                </span>

                <span v-auth="'machine:file'">
                    <el-button type="success" :disabled="data.status == -1" @click="showFileManage(data)" link>文件</el-button>
                    <el-divider direction="vertical" border-style="dashed" />
                </span>

                <el-button :disabled="data.status == -1" type="warning" @click="serviceManager(data)" link>脚本</el-button>
                <el-divider direction="vertical" border-style="dashed" />

                <el-dropdown @command="handleCommand">
                    <span class="el-dropdown-link-machine-list">
                        更多
                        <el-icon class="el-icon--right">
                            <arrow-down />
                        </el-icon>
                    </span>
                    <template #dropdown>
                        <el-dropdown-menu>
                            <el-dropdown-item :command="{ type: 'detail', data }"> 详情 </el-dropdown-item>

                            <el-dropdown-item :command="{ type: 'edit', data }" v-if="actionBtns[perms.updateMachine]"> 编辑 </el-dropdown-item>

                            <el-dropdown-item :command="{ type: 'process', data }" :disabled="data.status == -1"> 进程 </el-dropdown-item>

                            <el-dropdown-item :command="{ type: 'terminalRec', data }" v-if="actionBtns[perms.updateMachine] && data.enableRecorder == 1">
                                终端回放
                            </el-dropdown-item>

                            <el-dropdown-item
                                :command="{ type: 'closeCli', data }"
                                v-if="actionBtns[perms.closeCli]"
                                :disabled="!data.hasCli || data.status == -1"
                            >
                                关闭连接
                            </el-dropdown-item>
                        </el-dropdown-menu>
                    </template>
                </el-dropdown>
            </template>
        </page-table>

        <el-dialog v-model="infoDialog.visible">
            <el-descriptions title="详情" :column="3" border>
                <el-descriptions-item :span="1.5" label="机器id">{{ infoDialog.data.id }}</el-descriptions-item>
                <el-descriptions-item :span="1.5" label="名称">{{ infoDialog.data.name }}</el-descriptions-item>

                <el-descriptions-item :span="3" label="标签路径">{{ infoDialog.data.tagPath }}</el-descriptions-item>

                <el-descriptions-item :span="2" label="IP">{{ infoDialog.data.ip }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="端口">{{ infoDialog.data.port }}</el-descriptions-item>

                <el-descriptions-item :span="2" label="用户名">{{ infoDialog.data.username }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="认证方式">
                    {{ infoDialog.data.authCertId > 1 ? '授权凭证' : '密码' }}
                </el-descriptions-item>

                <el-descriptions-item :span="3" label="备注">{{ infoDialog.data.remark }}</el-descriptions-item>

                <el-descriptions-item :span="1.5" label="SSH隧道">{{ infoDialog.data.sshTunnelMachineId > 0 ? '是' : '否' }} </el-descriptions-item>
                <el-descriptions-item :span="1.5" label="终端回放">{{ infoDialog.data.enableRecorder == 1 ? '是' : '否' }} </el-descriptions-item>

                <el-descriptions-item :span="2" label="创建时间">{{ dateFormat(infoDialog.data.createTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" label="创建者">{{ infoDialog.data.creator }}</el-descriptions-item>

                <el-descriptions-item :span="2" label="更新时间">{{ dateFormat(infoDialog.data.updateTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" label="修改者">{{ infoDialog.data.modifier }}</el-descriptions-item>
            </el-descriptions>
        </el-dialog>

        <terminal-dialog ref="terminalDialogRef" :visibleMinimize="true">
            <template #headerTitle="{ terminalInfo }">
                {{ `${(terminalInfo.terminalId + '').slice(-2)}` }}
                <el-divider direction="vertical" />
                {{ `${terminalInfo.meta.username}@${terminalInfo.meta.ip}:${terminalInfo.meta.port}` }}
                <el-divider direction="vertical" />
                {{ terminalInfo.meta.name }}
            </template>
        </terminal-dialog>

        <machine-edit
            :title="machineEditDialog.title"
            v-model:visible="machineEditDialog.visible"
            v-model:machine="machineEditDialog.data"
            @valChange="submitSuccess"
        ></machine-edit>

        <process-list v-model:visible="processDialog.visible" v-model:machineId="processDialog.machineId" />

        <script-manage :title="serviceDialog.title" v-model:visible="serviceDialog.visible" v-model:machineId="serviceDialog.machineId" />

        <file-conf-list :title="fileDialog.title" v-model:visible="fileDialog.visible" v-model:machineId="fileDialog.machineId" />

        <machine-stats v-model:visible="machineStatsDialog.visible" :machineId="machineStatsDialog.machineId" :title="machineStatsDialog.title"></machine-stats>

        <machine-rec v-model:visible="machineRecDialog.visible" :machineId="machineRecDialog.machineId" :title="machineRecDialog.title"></machine-rec>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted, defineAsyncComponent, Ref } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { ElMessage, ElMessageBox } from 'element-plus';
import { machineApi, getMachineTerminalSocketUrl } from './api';
import { dateFormat } from '@/common/utils/date';
import ResourceTag from '../component/ResourceTag.vue';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import { formatByteSize } from '@/common/utils/format';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { SearchItem } from '@/components/SearchForm';
import { getTagPathSearchItem } from '../component/tag';

// 组件
const TerminalDialog = defineAsyncComponent(() => import('@/components/terminal/TerminalDialog.vue'));
const MachineEdit = defineAsyncComponent(() => import('./MachineEdit.vue'));
const ScriptManage = defineAsyncComponent(() => import('./ScriptManage.vue'));
const FileConfList = defineAsyncComponent(() => import('./file/FileConfList.vue'));
const MachineStats = defineAsyncComponent(() => import('./MachineStats.vue'));
const MachineRec = defineAsyncComponent(() => import('./MachineRec.vue'));
const ProcessList = defineAsyncComponent(() => import('./ProcessList.vue'));

const router = useRouter();
const route = useRoute();
const terminalDialogRef: any = ref(null);
const pageTableRef: Ref<any> = ref(null);

const perms = {
    addMachine: 'machine:add',
    updateMachine: 'machine:update',
    delMachine: 'machine:del',
    terminal: 'machine:terminal',
    closeCli: 'machine:close-cli',
};

const searchItems = [getTagPathSearchItem(TagResourceTypeEnum.Machine.value), SearchItem.input('ip', 'IP'), SearchItem.input('name', '名称')];

const columns = [
    TableColumn.new('name', '名称'),
    TableColumn.new('ipPort', 'ip:port').isSlot().setAddWidth(50),
    TableColumn.new('stat', '运行状态').isSlot().setAddWidth(55),
    TableColumn.new('fs', '磁盘(挂载点=>可用/总)').isSlot().setAddWidth(25),
    TableColumn.new('username', '用户名'),
    TableColumn.new('status', '状态').isSlot().setMinWidth(85),
    TableColumn.new('tagPath', '关联标签').isSlot().setAddWidth(10).alignCenter(),
    TableColumn.new('remark', '备注'),
    TableColumn.new('action', '操作').isSlot().setMinWidth(238).fixedRight().alignCenter(),
];

// 该用户拥有的的操作列按钮权限，使用v-if进行判断，v-auth对el-dropdown-item无效
const actionBtns = hasPerms([perms.updateMachine, perms.closeCli]);

const state = reactive({
    params: {
        pageNum: 1,
        pageSize: 0,
        ip: null,
        name: null,
        tagPath: '',
    },
    infoDialog: {
        visible: false,
        data: null as any,
    },
    // 当前选中数据
    selectionData: [],
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

const { params, infoDialog, selectionData, serviceDialog, processDialog, fileDialog, machineStatsDialog, machineEditDialog, machineRecDialog } = toRefs(state);

onMounted(async () => {});

const checkRouteTagPath = (query: any) => {
    if (route.query.tagPath) {
        query.tagPath = route.query.tagPath as string;
    }
    return query;
};

const handleCommand = (commond: any) => {
    const data = commond.data;
    const type = commond.type;
    switch (type) {
        case 'detail': {
            showInfo(data);
            return;
        }
        case 'edit': {
            openFormDialog(data);
            return;
        }
        case 'process': {
            showProcess(data);
            return;
        }
        case 'terminalRec': {
            showRec(data);
            return;
        }
        case 'closeCli': {
            closeCli(data);
            return;
        }
    }
};

const showTerminal = (row: any, event: PointerEvent) => {
    // 按住ctrl点击，则新建标签页打开, metaKey对应mac command键
    if (event.ctrlKey || event.metaKey) {
        const { href } = router.resolve({
            path: `/machine/terminal`,
            query: {
                id: row.id,
                name: row.name,
            },
        });
        window.open(href, '_blank');
        return;
    }

    const terminalId = Date.now();
    terminalDialogRef.value.open({
        terminalId,
        socketUrl: getMachineTerminalSocketUrl(row.id),
        minTitle: `${row.name} [${(terminalId + '').slice(-2)}]`, // 截取terminalId最后两位区分多个terminal
        minDesc: `${row.username}@${row.ip}:${row.port} (${row.name})`,
        meta: row,
    });
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

const openFormDialog = async (machine: any) => {
    let dialogTitle;
    if (machine) {
        state.machineEditDialog.data = machine;
        dialogTitle = '编辑机器';
    } else {
        state.machineEditDialog.data = null;
        dialogTitle = '添加机器';
    }

    state.machineEditDialog.title = dialogTitle;
    state.machineEditDialog.visible = true;
};

const deleteMachine = async () => {
    try {
        await ElMessageBox.confirm(
            `确定删除【${state.selectionData.map((x: any) => x.name).join(', ')}】机器信息? 该操作将同时删除脚本及文件配置信息`,
            '提示',
            {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'warning',
            }
        );
        await machineApi.del.request({ id: state.selectionData.map((x: any) => x.id).join(',') });
        ElMessage.success('操作成功');
        search();
    } catch (err) {
        //
    }
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
    if (!row.id) {
        return;
    }
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

const search = async () => {
    pageTableRef.value.search();
};

const submitSuccess = () => {
    search();
};

const showFileManage = (selectionData: any) => {
    state.fileDialog.visible = true;
    state.fileDialog.machineId = selectionData.id;
    state.fileDialog.title = `${selectionData.name} => ${selectionData.ip}`;
};

const getStatsFontClass = (availavle: number, total: number) => {
    const p = availavle / total;
    if (p < 0.1) {
        return 'color-danger';
    }
    if (p < 0.2) {
        return 'color-warning';
    }

    return 'color-success';
};

const showInfo = (info: any) => {
    state.infoDialog.data = info;
    state.infoDialog.visible = true;
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
.el-dropdown-link-machine-list {
    cursor: pointer;
    color: var(--el-color-primary);
    display: flex;
    align-items: center;
    margin-top: 6px;
}
</style>
