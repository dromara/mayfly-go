<template>
    <div class="h-full">
        <data-table
            ref="dataTableRef"
            :page-api="machineApi.list"
            :before-query-fn="checkRouteTagPath"
            :data-handler-fn="handleData"
            :search-items="searchItems"
            v-model:query-form="params"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="columns"
            :lazy="true"
        >
            <template #tableHeader>
                <el-button v-auth="perms.addMachine" type="primary" icon="plus" @click="editEntity()" plain>{{ $t('common.create') }} </el-button>
                <el-button v-auth="perms.delMachine" :disabled="selectionData.length < 1" @click="onDelete" type="danger" icon="delete" plain>
                    {{ $t('common.delete') }}
                </el-button>
                <el-button v-auth="perms.updateMachine" icon="key" plain @click="gotoHostKeys">{{ $t('machine.hostKeys') }}</el-button>
                <el-button
                    v-auth="perms.terminal"
                    icon="operation"
                    plain
                    type="warning"
                    @click="openBatchRun"
                >
                    {{ $t('machine.batchRunCmd') }}
                </el-button>
                <el-button
                    v-auth="perms.terminal"
                    icon="upload"
                    plain
                    type="warning"
                    @click="openBatchFile"
                >
                    {{ $t('machine.batchFile') }}
                </el-button>
                <el-button v-auth="perms.updateMachine" icon="data-line" plain @click="healthDialog.visible = true">
                    {{ $t('machine.healthOverview') }}
                </el-button>
            </template>

            <template #name="{ data }">
                <TagCodePath :code="data.code" show-popover />
                {{ data.name }}
            </template>

            <template #ipPort="{ data }">
                <el-link :disabled="data.status == -1" @click="showMachineStats(data)" type="primary" underline="never">
                    {{ `${data.ip}:${data.port}` }}
                </el-link>
            </template>

            <template #stats="{ data }">
                <span v-if="!data.stat">-</span>
                <div v-else>
                    <el-row>
                        <el-text size="small" class="text-[11px]!">
                            {{ $t('machine.memberInfo') }}:
                            <span :class="getStatsFontClass(data.stat.memAvailable, data.stat.memTotal)"
                                >{{ formatByteSize(data.stat.memAvailable, 1) }}/{{ formatByteSize(data.stat.memTotal, 1) }}
                            </span>
                        </el-text>
                    </el-row>
                    <el-row>
                        <el-text class="text-[11px]!" size="small">
                            {{ $t('machine.cpuInfo') }}: <span :class="getStatsFontClass(data.stat.cpuIdle, 100)">{{ Number(data.stat.cpuIdle).toFixed(0) }}%</span>
                        </el-text>
                    </el-row>
                </div>
            </template>

            <template #fs="{ data }">
                <span v-if="!data.stat?.fsInfos">-</span>
                <div v-else>
                    <el-row v-for="(i, idx) in data.stat.fsInfos.slice(0, 2)" :key="i.mountPoint">
                        <el-text class="text-[11px]!" size="small" :class="getStatsFontClass(i.free, i.used + i.free)">
                            {{ i.mountPoint }} => {{ formatByteSize(i.free, 0) }}/{{ formatByteSize(i.used + i.free, 0) }}
                        </el-text>

                        <!-- 展示剩余的磁盘信息 -->
                        <el-popover :show-after="300" v-if="data.stat.fsInfos.length > 2 && idx == 1" placement="top-start" width="230" trigger="hover">
                            <template #reference>
                                <SvgIcon class="mt-1 ml-1" color="var(--el-color-primary)" name="MoreFilled" />
                            </template>

                            <el-row v-for="i in data.stat.fsInfos.slice(2)" :key="i.mountPoint">
                                <el-text class="text-[11px]!" size="small" :class="getStatsFontClass(i.free, i.used + i.free)">
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
                    :model-value="data.status === 1 ? 1 : -1"
                    :active-value="1"
                    :inactive-value="-1"
                    inline-prompt
                    :active-text="$t('common.enable')"
                    :inactive-text="$t('common.disable')"
                    style="--el-switch-on-color: #13ce66; --el-switch-off-color: #ff4949"
                    @change="(val: 1 | -1) => changeStatus(data, val)"
                ></el-switch>
            </template>

            <template #authCert="{ data }">
                <ResourceAuthCert v-model:select-auth-cert="data.selectAuthCert" :auth-certs="data.authCerts" />
            </template>

            <!-- 行内操作由表格统一渲染（见 columns 里的 setActions），列宽与按钮同源 -->
        </data-table>

        <el-dialog v-if="infoDialog.visible" v-model="infoDialog.visible">
            <el-descriptions v-if="infoDialog.data" :title="$t('common.detail')" :column="3" border>
                <el-descriptions-item :span="1.5" label="ID">{{ infoDialog.data.id }}</el-descriptions-item>
                <el-descriptions-item :span="1.5" :label="$t('common.name')">{{ infoDialog.data.name }}</el-descriptions-item>

                <el-descriptions-item :span="3" :label="$t('tag.relateTag')">
                    <TagCodePath :code="infoDialog.data.code" />
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
                {{ `${terminalMeta(terminalInfo).selectAuthCert?.username}@${terminalMeta(terminalInfo).ip}:${terminalMeta(terminalInfo).port}` }}
                <el-divider direction="vertical" />
                {{ terminalMeta(terminalInfo).name }}
            </template>
        </terminal-dialog>

        <machine-edit :title="editDialog.title" v-model:visible="editDialog.visible" v-model:data="editDialog.data" @val-change="search()"></machine-edit>

        <process-list v-model:visible="processDialog.visible" v-model:machineId="processDialog.machineId" :title="processDialog.title" />

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

        <batch-run-dialog v-model:visible="batchRunCmdDialog.visible" :machines="batchRunCmdDialog.machines" />
        <batch-file-dialog v-model:visible="batchFileDispatchDialog.visible" :machines="batchFileDispatchDialog.machines" />
        <machine-health-dialog v-model:visible="healthDialog.visible" @view-trend="onHealthViewTrend" />
        <disk-analyze-dialog v-model:visible="diskDialog.visible" :machine-id="diskDialog.machineId" :machine-name="diskDialog.machineName" />

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
import { formatByteSize, formatDate } from '@/common/utils/format';
import { TableColumn } from '@/components/data-table';
import DataTable from '@/components/data-table/DataTable.vue';
import { SearchItem } from '@/components/data-table/SearchForm';
import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { useEditDialog, useRouteTagPath } from '@/hooks/useResourceForm';
import { defineAsyncComponent, ref, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import TagCodePath from '../component/TagCodePath.vue';
import { getMachineTerminalSocketUrl, machineApi } from './api';
import { MachineProtocolEnum } from './enums';
import type { MachineHealth, MachineVO } from './types';
import type { PageResult } from '@/types/common';

// 组件
const TerminalDialog = defineAsyncComponent(() => import('@/components/terminal/TerminalDialog.vue'));
const MachineEdit = defineAsyncComponent(() => import('./MachineEdit.vue'));
const ScriptManage = defineAsyncComponent(() => import('./ScriptManage.vue'));
const FileConfList = defineAsyncComponent(() => import('./file/FileConfList.vue'));
const MachineStats = defineAsyncComponent(() => import('./MachineStats.vue'));
const MachineRec = defineAsyncComponent(() => import('./MachineRec.vue'));
const ProcessList = defineAsyncComponent(() => import('./ProcessList.vue'));
const MachineFile = defineAsyncComponent(() => import('./file/MachineFile.vue'));
const ResourceAuthCert = defineAsyncComponent(() => import('../component/ResourceAuthCert.vue'));
const MachineRdpDialogComp = defineAsyncComponent(() => import('@/components/terminal-rdp/MachineRdpDialog.vue'));
const BatchRunDialog = defineAsyncComponent(() => import('./component/BatchRunDialog.vue'));
const BatchFileDialog = defineAsyncComponent(() => import('./component/BatchFileDialog.vue'));
const MachineHealthDialog = defineAsyncComponent(() => import('./component/MachineHealthDialog.vue'));
const DiskAnalyzeDialog = defineAsyncComponent(() => import('./component/DiskAnalyzeDialog.vue'));

const { t } = useI18n();

const props = defineProps({
    lazy: {
        type: [Boolean],
        default: false,
    },
});

const router = useRouter();
const terminalDialogRef = useTemplateRef<InstanceType<typeof TerminalDialog>>('terminalDialogRef');
const dataTableRef = useTemplateRef<InstanceType<typeof DataTable>>('dataTableRef');
const checkRouteTagPath = useRouteTagPath();
const { editDialog, editEntity } = useEditDialog<MachineVO>('machine');

const perms = {
    addMachine: 'machine:add',
    updateMachine: 'machine:update',
    delMachine: 'machine:del',
    terminal: 'machine:terminal',
};

const searchItems = [SearchItem.input('keyword', 'common.keyword').withPlaceholder('machine.keywordPlaceholder')];

const columns = [
    TableColumn.new('name', 'common.name').isSlot('name'),
    TableColumn.new('ipPort', 'Ip:Port').isSlot(),
    TableColumn.new('authCerts[0].username', 'machine.acName').isSlot('authCert'),
    // 状态列插槽渲染定宽 el-switch（圆钮不在字段文本里，离线估算与只量溢出回填都量不到）：+20 预留圆钮占位
    TableColumn.new('status', 'common.status').isSlot().setAddWidth(20),
    TableColumn.new('stats', 'machine.runningStat').isSlot(),
    TableColumn.new('fs', 'machine.fs').isSlot(),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('code', 'common.code'),
    TableColumn.new('action', 'common.operation')
        .setActions<MachineVO>([
            // 三种协议按机器类型互斥，同组只按最宽的一格计宽
            {
                key: 'ssh',
                label: 'machine.protocolSsh',
                type: 'primary',
                perm: perms.terminal,
                group: 'protocol',
                tooltip: 'machine.newOpenTabTerminalTips',
                disabled: (row) => row.status == -1,
                show: (row) => row.protocol == MachineProtocolEnum.Ssh.value,
                onClick: (row, event) => showTerminal(row, event),
            },
            {
                key: 'rdp',
                label: 'machine.protocolRdp',
                type: 'primary',
                perm: perms.terminal,
                group: 'protocol',
                show: (row) => row.protocol == MachineProtocolEnum.Rdp.value,
                onClick: (row) => showRDP(row),
            },
            {
                key: 'vnc',
                label: 'machine.protocolVnc',
                type: 'primary',
                perm: perms.terminal,
                group: 'protocol',
                show: (row) => row.protocol == MachineProtocolEnum.Vnc.value,
                onClick: (row) => showRDP(row),
            },
            {
                key: 'file',
                label: 'machine.file',
                type: 'success',
                perm: 'machine:file',
                disabled: (row) => row.status == -1,
                onClick: (row) => showFileManage(row),
            },
            {
                key: 'script',
                label: 'machine.script',
                type: 'warning',
                disabled: (row) => row.status == -1,
                show: (row) => row.protocol == MachineProtocolEnum.Ssh.value,
                onClick: (row) => serviceManager(row),
            },
            { key: 'detail', label: 'common.detail', onClick: (row) => showInfo(row) },
            { key: 'edit', label: 'common.edit', perm: perms.updateMachine, onClick: (row) => editEntity(row) },
            {
                key: 'process',
                label: 'machine.process',
                disabled: (row) => row.status == -1,
                show: (row) => row.protocol == MachineProtocolEnum.Ssh.value,
                onClick: (row) => showProcess(row),
            },
            {
                key: 'diskAnalyze',
                label: 'machine.diskAnalyze',
                disabled: (row) => row.status == -1,
                show: (row) => row.protocol == MachineProtocolEnum.Ssh.value,
                onClick: (row) => showDiskAnalyze(row),
            },
            {
                key: 'terminalRec',
                label: 'machine.terminalPlayback',
                perm: perms.updateMachine,
                show: (row) => row.enableRecorder == 1,
                onClick: (row) => showRec(row),
            },
            {
                key: 'rdpNewTab',
                label: 'machine.openRdpTab',
                group: 'protocol',
                show: (row) => row.protocol == MachineProtocolEnum.Rdp.value,
                onClick: (row) => showRDP(row, true),
            },
        ])
        .fixedRight()
        .alignCenter()
        .noShowOverflowTooltip(),
];

const params = ref({
    pageNum: 1,
    pageSize: 0,
    ip: null,
    name: null,
    tagPath: '',
});

const selectionData = ref<MachineVO[]>([]);

// --- 详情弹窗 ---
const infoDialog = ref({
    visible: false,
    data: null as MachineVO | null,
});

// --- 其他弹窗 ---
const serviceDialog = ref({
    visible: false,
    machineId: 0,
    authCertName: '',
    title: '',
});

const processDialog = ref({
    visible: false,
    machineId: 0,
    title: '',
});

const fileDialog = ref({
    visible: false,
    machineId: 0,
    authCertName: '',
    title: '',
});

const filesystemDialog = ref({
    visible: false,
    machineId: 0,
    protocol: 1,
    title: '',
    fileId: 0,
    authCertName: '',
    path: '',
});

const machineStatsDialog = ref({
    visible: false,
    stats: null,
    title: '',
    machineId: 0,
});

const machineRecDialog = ref({
    visible: false,
    machineId: 0,
    title: '',
});

const machineRdpDialog = ref({
    visible: false,
    machineId: 0,
    title: '',
    authCert: '',
});

const batchRunCmdDialog = ref({
    visible: false,
    machines: [] as MachineVO[],
});

const batchFileDispatchDialog = ref({
    visible: false,
    machines: [] as MachineVO[],
});


const healthDialog = ref({ visible: false });

const diskDialog = ref({ visible: false, machineId: 0, machineName: '' });

/** 批量执行：把当前勾选的机器交给对话框展示并执行 */
const openBatchRun = () => {
    if (!selectionData.value?.length) {
        Msg.warning('machine.selectMachineFirst');
        return;
    }
    batchRunCmdDialog.value.machines = selectionData.value;
    batchRunCmdDialog.value.visible = true;
};

/** 批量文件分发：把当前勾选的机器交给对话框 */
const openBatchFile = () => {
    if (!selectionData.value?.length) {
        Msg.warning('machine.selectMachineFirst');
        return;
    }
    batchFileDispatchDialog.value.machines = selectionData.value;
    batchFileDispatchDialog.value.visible = true;
};

// --- 数据操作 ---
const handleData = (res: PageResult<MachineVO>) => {
    const dataList = res.list;
    for (let x of dataList) {
        x.selectAuthCert = x.authCerts[0];
        // el-switch 要求 model-value 必须为 active-value 或 inactive-value，
        // 后端返回 null/undefined/其他值时强制归一为 inactive(-1)
        if (x.status !== 1 && x.status !== -1) {
            x.status = -1;
        }
    }
    return res;
};

/** event 为点击事件（Ctrl/Cmd 点开新标签页）；动作被折进「更多」时拿不到事件，按普通打开处理 */
const showTerminal = (row: MachineVO, event?: MouseEvent) => {
    const ac = row.selectAuthCert?.name || '';
    if (event?.ctrlKey || event?.metaKey) {
        const { href } = router.resolve({
            path: `/machine/terminal`,
            query: { ac, name: row.name },
        });
        window.open(href, '_blank');
        return;
    }

    const terminalId = Date.now();
    terminalDialogRef.value?.open({
        terminalId,
        socketUrl: getMachineTerminalSocketUrl(ac),
        minTitle: `${row.name} [${(terminalId + '').slice(-2)}]`,
        minDesc: `${row.selectAuthCert?.username || ''}@${row.ip}:${row.port} (${row.name})`,
        meta: { ...row },
    });
};

/** 将终端插槽透传的 meta 还原为 MachineVO */
const terminalMeta = (terminalInfo: { meta?: unknown }): MachineVO => terminalInfo.meta as MachineVO;

const onDelete = async () => {
    const records = selectionData.value || [];
    if (records.length === 0) return;
    if (!(await useI18nDeleteConfirm(records.map((x) => x.name).join('、')))) {
        // 取消或关掉弹窗：不继续后续操作
        return;
    }
    await machineApi.del.request({ id: records.map((x) => x.id).join(',') });
    Msg.deleteSuccess();
    search();
};

const serviceManager = (row: MachineVO) => {
    const authCert = row.selectAuthCert;
    serviceDialog.value.machineId = row.id;
    serviceDialog.value.authCertName = authCert?.name || '';
    serviceDialog.value.visible = true;
    serviceDialog.value.title = `${row.name} => ${authCert?.username || ''}@${row.ip}`;
};

const changeStatus = async (row: MachineVO, status: 1 | -1) => {
    if (!row.id) return;
    // 乐观更新本地状态，保证 el-switch 显示与用户操作同步（el-switch 使用 :model-value 单向绑定，需要显式回写）
    const prev = row.status;
    row.status = status;
    try {
        await machineApi.changeStatus.request({ id: row.id, status });
    } catch (err) {
        // 失败回滚，避免 UI 与后端不一致
        row.status = prev;
        throw err;
    }
};

const showMachineStats = async (machine: MachineVO) => {
    machineStatsDialog.value.machineId = machine.id;
    machineStatsDialog.value.title = `${t('machine.machineState')}: ${machine.name} => ${machine.ip}`;
    machineStatsDialog.value.visible = true;
};

// 从健康总览下钻到单机监控历史：先收起总览，避免两层模态窗口叠加
const onHealthViewTrend = (health: MachineHealth) => {
    healthDialog.value.visible = false;
    showMachineStats({ id: health.machineId, name: health.name, ip: health.ip } as MachineVO);
};

const search = (tagPath?: string) => {
    // tagPath 为 undefined 时（如"所有资源"节点），清空过滤条件查询全部；为具体值时按标签过滤
    params.value.tagPath = tagPath ?? '';
    dataTableRef.value?.search();
};

const showFileManage = (data: MachineVO) => {
    if (data.protocol === MachineProtocolEnum.Ssh.value) {
        fileDialog.value.visible = true;
        fileDialog.value.machineId = data.id;
        fileDialog.value.authCertName = data.selectAuthCert?.name || '';
        fileDialog.value.title = `${data.name} => ${data.selectAuthCert?.username || ''}@${data.ip}`;
    } else if (data.protocol === MachineProtocolEnum.Rdp.value) {
        filesystemDialog.value.protocol = 2;
        filesystemDialog.value.machineId = data.id;
        filesystemDialog.value.fileId = data.id;
        filesystemDialog.value.authCertName = data.selectAuthCert?.name || '';
        filesystemDialog.value.path = '/';
        filesystemDialog.value.title = `${data.name} => ${data.selectAuthCert?.username || ''}@${t('machine.remoteFileDesktopManage')}`;
        filesystemDialog.value.visible = true;
    }
};

const getStatsFontClass = (availavle: number, total: number) => {
    const p = availavle / total;
    if (p < 0.1) return 'color-danger';
    if (p < 0.2) return 'color-warning';
    return 'color-success';
};

const showInfo = (info: MachineVO) => {
    infoDialog.value.data = info;
    infoDialog.value.visible = true;
};

/** 主机密钥信任库入口：直达安全配置页的主机密钥标签页 */
const gotoHostKeys = () => {
    router.push({ path: '/machine/security', query: { tab: 'hostKey' } });
};

const showProcess = (row: MachineVO) => {
    processDialog.value.machineId = row.id;
    processDialog.value.title = `${row.name} (${row.ip}:${row.port})`;
    processDialog.value.visible = true;
};

const showDiskAnalyze = (row: MachineVO) => {
    diskDialog.value.machineId = row.id;
    diskDialog.value.machineName = row.name;
    diskDialog.value.visible = true;
};

const showRec = (row: MachineVO) => {
    machineRecDialog.value.title = `${row.name}[${row.ip}]-${t('machine.terminalPlayback')}`;
    machineRecDialog.value.machineId = row.id;
    machineRecDialog.value.visible = true;
};

const showRDP = (row: MachineVO, blank = false) => {
    if (blank) {
        const { href } = router.resolve({
            path: `/machine/terminal-rdp`,
            query: { ac: row.selectAuthCert?.name || '', name: row.name },
        });
        window.open(href, '_blank');
        return;
    }
    machineRdpDialog.value.title = `${row.name}[${row.ip}]-${t('machine.remoteDesktop')}`;
    machineRdpDialog.value.machineId = row.id;
    machineRdpDialog.value.authCert = row.selectAuthCert?.name || '';
    machineRdpDialog.value.visible = true;
};

defineExpose({ search });
</script>
