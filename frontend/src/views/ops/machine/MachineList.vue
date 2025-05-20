<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="machineApi.list"
            :before-query-fn="checkRouteTagPath"
            :data-handler-fn="handleData"
            :search-items="searchItems"
            v-model:query-form="params"
            :show-selection="true"
            v-model:selection-data="state.selectionData"
            :columns="columns"
            :lazy="true"
        >
            <template #tableHeader>
                <el-button v-auth="perms.addMachine" type="primary" icon="plus" @click="openFormDialog(false)" plain>{{ $t('common.create') }} </el-button>
                <el-button v-auth="perms.delMachine" :disabled="selectionData.length < 1" @click="deleteMachine()" type="danger" icon="delete">
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #ipPort="{ data }">
                <el-link :disabled="data.status == -1" @click="showMachineStats(data)" type="primary" underline="never">
                    {{ `${data.ip}:${data.port}` }}
                </el-link>
            </template>

            <template #stat="{ data }">
                <span v-if="!data.stat">-</span>
                <div v-else>
                    <el-row>
                        <el-text size="small" class="!text-[11px]">
                            {{ $t('machine.memberInfo') }}:
                            <span :class="getStatsFontClass(data.stat.memAvailable, data.stat.memTotal)"
                                >{{ formatByteSize(data.stat.memAvailable, 1) }}/{{ formatByteSize(data.stat.memTotal, 1) }}
                            </span>
                        </el-text>
                    </el-row>
                    <el-row>
                        <el-text class="!text-[11px]" size="small">
                            {{ $t('machine.cpuInfo') }}: <span :class="getStatsFontClass(data.stat.cpuIdle, 100)">{{ data.stat.cpuIdle.toFixed(0) }}%</span>
                        </el-text>
                    </el-row>
                </div>
            </template>

            <template #fs="{ data }">
                <span v-if="!data.stat?.fsInfos">-</span>
                <div v-else>
                    <el-row v-for="(i, idx) in data.stat.fsInfos.slice(0, 2)" :key="i.mountPoint">
                        <el-text class="!text-[11px]" size="small" :class="getStatsFontClass(i.free, i.used + i.free)">
                            {{ i.mountPoint }} => {{ formatByteSize(i.free, 0) }}/{{ formatByteSize(i.used + i.free, 0) }}
                        </el-text>

                        <!-- 展示剩余的磁盘信息 -->
                        <el-popover :show-after="300" v-if="data.stat.fsInfos.length > 2 && idx == 1" placement="top-start" width="230" trigger="hover">
                            <template #reference>
                                <SvgIcon class="mt-1 ml-1" color="var(--el-color-primary)" name="MoreFilled" />
                            </template>

                            <el-row v-for="i in data.stat.fsInfos.slice(2)" :key="i.mountPoint">
                                <el-text class="!text-[11px]" size="small" :class="getStatsFontClass(i.free, i.used + i.free)">
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
                    v-model="data.status"
                    :active-value="1"
                    :inactive-value="-1"
                    inline-prompt
                    :active-text="$t('common.enable')"
                    :inactive-text="$t('common.disable')"
                    style="--el-switch-on-color: #13ce66; --el-switch-off-color: #ff4949"
                    @change="changeStatus(data)"
                ></el-switch>
            </template>

            <template #tagPath="{ data }">
                <ResourceTags :tags="data.tags" />
            </template>

            <template #authCert="{ data }">
                <ResourceAuthCert v-model:select-auth-cert="data.selectAuthCert" :auth-certs="data.authCerts" />
            </template>

            <template #action="{ data }">
                <span v-auth="'machine:terminal'">
                    <el-tooltip
                        v-if="data.protocol == MachineProtocolEnum.Ssh.value"
                        :show-after="500"
                        :content="$t('machine.newOpenTabTerminalTips')"
                        placement="top"
                    >
                        <el-button :disabled="data.status == -1" type="primary" @click="showTerminal(data, $event)" link>SSH</el-button>
                    </el-tooltip>

                    <el-button v-if="data.protocol == MachineProtocolEnum.Rdp.value" type="primary" @click="showRDP(data)" link>RDP</el-button>
                    <el-button v-if="data.protocol == MachineProtocolEnum.Vnc.value" type="primary" @click="showRDP(data)" link>VNC</el-button>

                    <el-divider direction="vertical" border-style="dashed" />
                </span>

                <span v-auth="'machine:file'">
                    <el-button type="success" :disabled="data.status == -1" @click="showFileManage(data)" link>{{ $t('machine.file') }}</el-button>
                    <el-divider direction="vertical" border-style="dashed" />
                </span>

                <el-button
                    v-if="data.protocol == MachineProtocolEnum.Ssh.value"
                    :disabled="data.status == -1"
                    type="warning"
                    @click="serviceManager(data)"
                    link
                    >{{ $t('machine.script') }}</el-button
                >
                <el-divider direction="vertical" border-style="dashed" />

                <el-dropdown @command="handleCommand">
                    <span class="el-dropdown-link-machine-list">
                        {{ $t('common.more') }}
                        <el-icon class="el-icon--right">
                            <arrow-down />
                        </el-icon>
                    </span>
                    <template #dropdown>
                        <el-dropdown-menu>
                            <el-dropdown-item :command="{ type: 'detail', data }">
                                {{ $t('common.detail') }}
                            </el-dropdown-item>

                            <el-dropdown-item :command="{ type: 'rdp-blank', data }" v-if="data.protocol == MachineProtocolEnum.Rdp.value">
                                RDP({{ $t('machine.newTab') }})
                            </el-dropdown-item>

                            <el-dropdown-item :command="{ type: 'edit', data }" v-if="actionBtns[perms.updateMachine]">
                                {{ $t('common.edit') }}
                            </el-dropdown-item>

                            <el-dropdown-item
                                v-if="data.protocol == MachineProtocolEnum.Ssh.value"
                                :command="{ type: 'process', data }"
                                :disabled="data.status == -1"
                            >
                                {{ $t('machine.process') }}
                            </el-dropdown-item>

                            <el-dropdown-item :command="{ type: 'terminalRec', data }" v-if="actionBtns[perms.updateMachine] && data.enableRecorder == 1">
                                {{ $t('machine.terminalPlayback') }}
                            </el-dropdown-item>
                        </el-dropdown-menu>
                    </template>
                </el-dropdown>
            </template>
        </page-table>

        <el-dialog v-if="infoDialog.visible" v-model="infoDialog.visible">
            <el-descriptions :title="$t('common.detail')" :column="3" border>
                <el-descriptions-item :span="1.5" label="ID">{{ infoDialog.data.id }}</el-descriptions-item>
                <el-descriptions-item :span="1.5" :label="$t('common.name')">{{ infoDialog.data.name }}</el-descriptions-item>

                <el-descriptions-item :span="3" :label="$t('tag.relateTag')">
                    <ResourceTags :tags="infoDialog.data.tags" />
                </el-descriptions-item>

                <el-descriptions-item :span="2" label="IP">{{ infoDialog.data.ip }}</el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('machine.port')">{{ infoDialog.data.port }}</el-descriptions-item>

                <el-descriptions-item :span="3" :label="$t('common.remark')">{{ infoDialog.data.remark }}</el-descriptions-item>

                <el-descriptions-item :span="1.5" :label="$t('machine.sshTunnel')">
                    {{ infoDialog.data.sshTunnelMachineId > 0 ? $t('common.yes') : $t('common.no') }}
                </el-descriptions-item>
                <el-descriptions-item :span="1.5" :label="$t('machine.terminalPlayback')">
                    {{ infoDialog.data.enableRecorder == 1 ? $t('common.yes') : $t('common.no') }}
                </el-descriptions-item>

                <el-descriptions-item :span="2" :label="$t('common.createTime')">
                    {{ formatDate(infoDialog.data.createTime) }}
                </el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('common.creator')">
                    {{ infoDialog.data.creator }}
                </el-descriptions-item>

                <el-descriptions-item :span="2" :label="$t('common.updateTime')">
                    {{ formatDate(infoDialog.data.updateTime) }}
                </el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('common.modifier')">
                    {{ infoDialog.data.modifier }}
                </el-descriptions-item>
            </el-descriptions>
        </el-dialog>

        <terminal-dialog ref="terminalDialogRef" :visibleMinimize="true">
            <template #headerTitle="{ terminalInfo }">
                {{ `${(terminalInfo.terminalId + '').slice(-2)}` }}
                <el-divider direction="vertical" />
                {{ `${terminalInfo.meta.selectAuthCert.username}@${terminalInfo.meta.ip}:${terminalInfo.meta.port}` }}
                <el-divider direction="vertical" />
                {{ terminalInfo.meta.name }}
            </template>
        </terminal-dialog>

        <machine-edit
            :title="$t(machineEditDialog.title)"
            v-model:visible="machineEditDialog.visible"
            v-model:machine="machineEditDialog.data"
            @valChange="submitSuccess"
        ></machine-edit>

        <process-list v-model:visible="processDialog.visible" v-model:machineId="processDialog.machineId" />

        <script-manage
            :title="serviceDialog.title"
            v-model:visible="serviceDialog.visible"
            v-model:machineId="serviceDialog.machineId"
            :auth-cert-name="serviceDialog.authCertName"
        />

        <file-conf-list
            :title="fileDialog.title"
            v-model:visible="fileDialog.visible"
            v-model:machineId="fileDialog.machineId"
            :auth-cert-name="fileDialog.authCertName"
        />

        <machine-stats v-model:visible="machineStatsDialog.visible" :machineId="machineStatsDialog.machineId" :title="machineStatsDialog.title"></machine-stats>

        <machine-rec v-model:visible="machineRecDialog.visible" :machineId="machineRecDialog.machineId" :title="machineRecDialog.title"></machine-rec>

        <machine-rdp-dialog-comp
            :title="machineRdpDialog.title"
            v-model:visible="machineRdpDialog.visible"
            v-model:machine-id="machineRdpDialog.machineId"
            v-model:auth-cert="machineRdpDialog.authCert"
        >
            <template #headerTitle="{ terminalInfo }">
                {{ `${(terminalInfo.terminalId + '').slice(-2)}` }}
                <el-divider direction="vertical" />
                {{ `${terminalInfo.meta.username}@${terminalInfo.meta.ip}:${terminalInfo.meta.port}` }}
                <el-divider direction="vertical" />
                {{ terminalInfo.meta.name }}
            </template>
        </machine-rdp-dialog-comp>

        <el-dialog destroy-on-close :title="filesystemDialog.title" v-model="filesystemDialog.visible" :close-on-click-modal="false" width="70%">
            <machine-file
                :machine-id="filesystemDialog.machineId"
                :protocol="filesystemDialog.protocol"
                :file-id="filesystemDialog.fileId"
                :path="filesystemDialog.path"
            />
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { defineAsyncComponent, onMounted, reactive, ref, Ref, toRefs } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { getMachineTerminalSocketUrl, machineApi } from './api';
import ResourceTags from '../component/ResourceTags.vue';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import { formatByteSize, formatDate } from '@/common/utils/format';
import { TagResourceTypePath } from '@/common/commonEnum';
import { SearchItem } from '@/components/SearchForm';
import { getTagPathSearchItem } from '../component/tag';
import MachineFile from '@/views/ops/machine/file/MachineFile.vue';
import ResourceAuthCert from '../component/ResourceAuthCert.vue';
import { MachineProtocolEnum } from './enums';
import MachineRdpDialogComp from '@/components/terminal-rdp/MachineRdpDialog.vue';
import { useI18n } from 'vue-i18n';
import { useI18nDeleteConfirm, useI18nDeleteSuccessMsg } from '@/hooks/useI18n';

// 组件
const TerminalDialog = defineAsyncComponent(() => import('@/components/terminal/TerminalDialog.vue'));
const MachineEdit = defineAsyncComponent(() => import('./MachineEdit.vue'));
const ScriptManage = defineAsyncComponent(() => import('./ScriptManage.vue'));
const FileConfList = defineAsyncComponent(() => import('./file/FileConfList.vue'));
const MachineStats = defineAsyncComponent(() => import('./MachineStats.vue'));
const MachineRec = defineAsyncComponent(() => import('./MachineRec.vue'));
const ProcessList = defineAsyncComponent(() => import('./ProcessList.vue'));

const { t } = useI18n();

const props = defineProps({
    lazy: {
        type: [Boolean],
        default: false,
    },
});

const router = useRouter();
const route = useRoute();
const terminalDialogRef: any = ref(null);
const pageTableRef: Ref<any> = ref(null);

const perms = {
    addMachine: 'machine:add',
    updateMachine: 'machine:update',
    delMachine: 'machine:del',
    terminal: 'machine:terminal',
};

const searchItems = [
    SearchItem.input('keyword', 'common.keyword').withPlaceholder('machine.keywordPlaceholder'),
    getTagPathSearchItem(TagResourceTypePath.MachineAuthCert),
];

const columns = [
    TableColumn.new('tags[0].tagPath', 'tag.relateTag').isSlot('tagPath').setAddWidth(20),
    TableColumn.new('name', 'common.name'),
    TableColumn.new('ipPort', 'Ip:Port').isSlot().setAddWidth(55),
    TableColumn.new('authCerts[0].username', 'machine.acName').isSlot('authCert').setAddWidth(10),
    TableColumn.new('status', 'common.status').isSlot().setAddWidth(5),
    TableColumn.new('stat', 'machine.runningStat').isSlot().setAddWidth(55),
    TableColumn.new('fs', 'machine.fs').isSlot().setAddWidth(25),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('code', 'common.code'),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(258).fixedRight().alignCenter().noShowOverflowTooltip(),
];

// 该用户拥有的的操作列按钮权限，使用v-if进行判断，v-auth对el-dropdown-item无效
const actionBtns: any = hasPerms([perms.updateMachine]);

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
        authCertName: '',
        title: '',
    },
    processDialog: {
        visible: false,
        machineId: 0,
    },
    fileDialog: {
        visible: false,
        machineId: 0,
        authCertName: '',
        title: '',
    },
    filesystemDialog: {
        visible: false,
        machineId: 0,
        protocol: 1,
        title: '',
        fileId: 0,
        authCertName: '',
        path: '',
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
        title: '',
    },
    machineRecDialog: {
        visible: false,
        machineId: 0,
        title: '',
    },
    machineRdpDialog: {
        visible: false,
        machineId: 0,
        title: '',
        authCert: '',
    },
});

const {
    params,
    infoDialog,
    selectionData,
    serviceDialog,
    processDialog,
    fileDialog,
    machineStatsDialog,
    machineEditDialog,
    machineRecDialog,
    machineRdpDialog,
    filesystemDialog,
} = toRefs(state);

onMounted(async () => {
    if (!props.lazy) {
        search();
    }
});

const checkRouteTagPath = (query: any) => {
    if (route.query.tagPath) {
        query.tagPath = route.query.tagPath as string;
    }
    return query;
};

const handleData = (res: any) => {
    const dataList = res.list;
    // 赋值授权凭证
    for (let x of dataList) {
        x.selectAuthCert = x.authCerts[0];
    }
    return res;
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
        case 'rdp': {
            showRDP(data);
            return;
        }
        case 'rdp-blank': {
            showRDP(data, true);
            return;
        }
    }
};

const showTerminal = (row: any, event: PointerEvent) => {
    const ac = row.selectAuthCert.name;
    // 按住ctrl点击，则新建标签页打开, metaKey对应mac command键
    if (event.ctrlKey || event.metaKey) {
        const { href } = router.resolve({
            path: `/machine/terminal`,
            query: {
                ac,
                name: row.name,
            },
        });
        window.open(href, '_blank');
        return;
    }

    const terminalId = Date.now();
    terminalDialogRef.value.open({
        terminalId,
        socketUrl: getMachineTerminalSocketUrl(ac),
        minTitle: `${row.name} [${(terminalId + '').slice(-2)}]`, // 截取terminalId最后两位区分多个terminal
        minDesc: `${row.selectAuthCert.username}@${row.ip}:${row.port} (${row.name})`,
        meta: row,
    });
};

const openFormDialog = async (machine: any) => {
    let dialogTitle;
    if (machine) {
        state.machineEditDialog.data = machine;
        dialogTitle = 'machine.editMachine';
    } else {
        state.machineEditDialog.data = null;
        dialogTitle = 'machine.createMachine';
    }

    state.machineEditDialog.title = dialogTitle;
    state.machineEditDialog.visible = true;
};

const deleteMachine = async () => {
    try {
        await useI18nDeleteConfirm(state.selectionData.map((x: any) => x.name).join('、'));
        await machineApi.del.request({ id: state.selectionData.map((x: any) => x.id).join(',') });
        useI18nDeleteSuccessMsg();
        search();
    } catch (err) {
        //
    }
};

const serviceManager = (row: any) => {
    const authCert = row.selectAuthCert;
    state.serviceDialog.machineId = row.id;
    state.serviceDialog.authCertName = authCert.name;
    state.serviceDialog.visible = true;
    state.serviceDialog.title = `${row.name} => ${authCert.username}@${row.ip}`;
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
    state.machineStatsDialog.title = `${t('machine.machineState')}: ${machine.name} => ${machine.ip}`;
    state.machineStatsDialog.visible = true;
};

const search = async (tagPath: string = '') => {
    if (tagPath) {
        state.params.tagPath = tagPath;
    }
    pageTableRef.value.search();
};

const submitSuccess = () => {
    search();
};

const showFileManage = (data: any) => {
    if (data.protocol === MachineProtocolEnum.Ssh.value) {
        // ssh
        state.fileDialog.visible = true;
        state.fileDialog.machineId = data.id;
        state.fileDialog.authCertName = data.selectAuthCert.name;
        state.fileDialog.title = `${data.name} => ${data.selectAuthCert.username}@${data.ip}`;
    } else if (data.protocol === MachineProtocolEnum.Rdp.value) {
        // rdp
        state.filesystemDialog.protocol = 2;
        state.filesystemDialog.machineId = data.id;
        state.filesystemDialog.fileId = data.id;
        state.filesystemDialog.authCertName = data.selectAuthCert.name;
        state.filesystemDialog.path = '/';
        state.filesystemDialog.title = `${data.name} => ${data.selectAuthCert.username}@${t('machine.remoteFileDesktopManage')}`;
        state.filesystemDialog.visible = true;
    }
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
    state.machineRecDialog.title = `${row.name}[${row.ip}]-${t('machine.terminalPlayback')}`;
    state.machineRecDialog.machineId = row.id;
    state.machineRecDialog.visible = true;
};

const showRDP = (row: any, blank = false) => {
    if (blank) {
        const { href } = router.resolve({
            path: `/machine/terminal-rdp`,
            query: {
                ac: row.selectAuthCert.name,
                name: row.name,
            },
        });
        window.open(href, '_blank');
        return;
    }
    state.machineRdpDialog.title = `${row.name}[${row.ip}]-${t('machine.remoteDesktop')}`;
    state.machineRdpDialog.machineId = row.id;
    state.machineRdpDialog.authCert = row.selectAuthCert.name;
    state.machineRdpDialog.visible = true;
};

defineExpose({ search });
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
