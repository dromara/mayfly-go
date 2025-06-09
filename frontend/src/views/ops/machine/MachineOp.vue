<template>
    <div class="h-full">
        <ResourceOpPanel @resize="onResizeTagTree">
            <template #left>
                <tag-tree
                    ref="tagTreeRef"
                    :resource-type="TagResourceTypePath.MachineAuthCert"
                    :tag-path-node-type="NodeTypeTagPath"
                    :default-expanded-keys="state.defaultExpendKey"
                >
                    <template #prefix="{ data }">
                        <SvgIcon
                            v-if="data.icon && data.params.status == 1 && data.params.protocol == MachineProtocolEnum.Ssh.value"
                            :name="data.icon.name"
                            :color="data.icon.color"
                        />
                        <SvgIcon
                            v-if="data.icon && data.params.status == -1 && data.params.protocol == MachineProtocolEnum.Ssh.value"
                            :name="data.icon.name"
                            color="var(--el-color-danger)"
                        />
                        <SvgIcon v-if="data.icon && data.params.protocol != MachineProtocolEnum.Ssh.value" :name="data.icon.name" :color="data.icon.color" />
                    </template>

                    <template #suffix="{ data }">
                        <span v-if="data.type.value == MachineNodeType.AuthCert">{{
                            ` ${data.params.selectAuthCert.username}@${data.params.ip}:${data.params.port}`
                        }}</span>
                    </template>
                </tag-tree>
            </template>

            <template #right>
                <el-card class="h-full" body-class="machine-terminal-tabs h-full !p-1 flex flex-col flex-1">
                    <el-tabs v-if="state.tabs.size > 0" type="card" @tab-remove="onRemoveTab" v-model="state.activeTermName" class="!h-full w-full">
                        <el-tab-pane class="!h-full flex flex-col" closable v-for="dt in state.tabs.values()" :label="dt.label" :name="dt.key" :key="dt.key">
                            <template #label>
                                <el-popconfirm @confirm="handleReconnect(dt, true)" :title="$t('machine.reConnTips')">
                                    <template #reference>
                                        <el-icon
                                            class="mr-1"
                                            :color="EnumValue.getEnumByValue(TerminalStatusEnum, dt.status)?.extra?.iconColor"
                                            :title="dt.status == TerminalStatusEnum.Connected.value ? '' : $t('machine.clickReConn')"
                                            ><Connection />
                                        </el-icon>
                                    </template>
                                </el-popconfirm>
                                <el-popover :show-after="1000" placement="bottom-start" trigger="hover" :width="250">
                                    <template #reference>
                                        <div>
                                            <span class="machine-terminal-tab-label">{{ dt.label }}</span>
                                        </div>
                                    </template>
                                    <template #default>
                                        <el-descriptions :column="1" size="small">
                                            <el-descriptions-item :label="$t('common.name')"> {{ dt.params?.name }} </el-descriptions-item>
                                            <el-descriptions-item label="host"> {{ dt.params?.ip }} : {{ dt.params?.port }} </el-descriptions-item>
                                            <el-descriptions-item label="username"> {{ dt.params?.selectAuthCert.username }} </el-descriptions-item>
                                            <el-descriptions-item label="remark"> {{ dt.params?.remark }} </el-descriptions-item>
                                        </el-descriptions>
                                    </template>
                                </el-popover>
                            </template>

                            <div :ref="(el: any) => setTerminalWrapperRef(el, dt.key)" class="terminal-wrapper flex-1 h-[calc(100vh-155px)]">
                                <TerminalBody
                                    v-if="dt.params.protocol == MachineProtocolEnum.Ssh.value"
                                    :mount-init="false"
                                    @status-change="terminalStatusChange(dt.key, $event)"
                                    :ref="(el: any) => setTerminalRef(el, dt.key)"
                                    :socket-url="dt.socketUrl"
                                />
                                <machine-rdp
                                    v-if="dt.params.protocol != MachineProtocolEnum.Ssh.value"
                                    :machine-id="dt.params.id"
                                    :auth-cert="dt.authCert"
                                    :ref="(el: any) => setTerminalRef(el, dt.key)"
                                    @status-change="terminalStatusChange(dt.key, $event)"
                                />
                            </div>
                        </el-tab-pane>
                    </el-tabs>

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

                            <el-descriptions-item :span="1.5" :label="$t('machine.sshTunnel')"
                                >{{ infoDialog.data.sshTunnelMachineId > 0 ? $t('common.yes') : $t('common.no') }}
                            </el-descriptions-item>
                            <el-descriptions-item :span="1.5" :label="$t('machine.terminalPlayback')"
                                >{{ infoDialog.data.enableRecorder == 1 ? $t('common.yes') : $t('common.no') }}
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

                    <process-list v-model:visible="processDialog.visible" v-model:machineId="processDialog.machineId" />

                    <script-manage
                        :title="serviceDialog.title"
                        v-model:visible="serviceDialog.visible"
                        v-model:machineId="serviceDialog.machineId"
                        :auth-cert-name="serviceDialog.authCertName"
                    />

                    <file-conf-list
                        :title="fileDialog.title"
                        :auth-cert-name="fileDialog.authCertName"
                        v-model:visible="fileDialog.visible"
                        v-model:machineId="fileDialog.machineId"
                        :protocol="fileDialog.protocol"
                    />

                    <el-dialog
                        destroy-on-close
                        :title="state.filesystemDialog.title"
                        v-model="state.filesystemDialog.visible"
                        :close-on-click-modal="false"
                        width="70%"
                    >
                        <machine-file
                            :machine-id="state.filesystemDialog.machineId"
                            :auth-cert-name="state.filesystemDialog.authCertName"
                            :protocol="state.filesystemDialog.protocol"
                            :file-id="state.filesystemDialog.fileId"
                            :path="state.filesystemDialog.path"
                        />
                    </el-dialog>

                    <machine-stats v-model:visible="machineStatsDialog.visible" :machineId="machineStatsDialog.machineId" :title="machineStatsDialog.title" />

                    <machine-rec v-model:visible="machineRecDialog.visible" :machineId="machineRecDialog.machineId" :title="machineRecDialog.title" />
                </el-card>
            </template>
        </ResourceOpPanel>
    </div>
</template>

<script lang="ts" setup>
import { defineAsyncComponent, nextTick, onMounted, reactive, ref, toRefs, watch } from 'vue';
import { useRouter } from 'vue-router';
import { getMachineTerminalSocketUrl, machineApi } from './api';
import { formatDate } from '@/common/utils/format';
import { hasPerms } from '@/components/auth/auth';
import { TagResourceTypeEnum, TagResourceTypePath } from '@/common/commonEnum';
import { NodeType, TagTreeNode, getTagTypeCodeByPath } from '../component/tag';
import TagTree from '../component/TagTree.vue';
import { ContextmenuItem } from '@/components/contextmenu/index';
import TerminalBody from '@/components/terminal/TerminalBody.vue';
import { TerminalStatus, TerminalStatusEnum } from '@/components/terminal/common';
import MachineRdp from '@/components/terminal-rdp/MachineRdp.vue';
import MachineFile from '@/views/ops/machine/file/MachineFile.vue';
import ResourceTags from '../component/ResourceTags.vue';
import { MachineProtocolEnum } from './enums';
import { useAutoOpenResource } from '@/store/autoOpenResource';
import { storeToRefs } from 'pinia';
import EnumValue from '@/common/Enum';
import { useI18n } from 'vue-i18n';
import ResourceOpPanel from '../component/ResourceOpPanel.vue';

// 组件
const ScriptManage = defineAsyncComponent(() => import('./ScriptManage.vue'));
const FileConfList = defineAsyncComponent(() => import('./file/FileConfList.vue'));
const MachineStats = defineAsyncComponent(() => import('./MachineStats.vue'));
const MachineRec = defineAsyncComponent(() => import('./MachineRec.vue'));
const ProcessList = defineAsyncComponent(() => import('./ProcessList.vue'));

const { t } = useI18n();

const router = useRouter();

const perms = {
    addMachine: 'machine:add',
    updateMachine: 'machine:update',
    delMachine: 'machine:del',
    terminal: 'machine:terminal',
    closeCli: 'machine:close-cli',
};

// 该用户拥有的的操作列按钮权限，使用v-if进行判断，v-auth对el-dropdown-item无效
const actionBtns = hasPerms([perms.updateMachine, perms.closeCli]);

class MachineNodeType {
    static Machine = 1;
    static AuthCert = 2;
}

const state = reactive({
    defaultExpendKey: [] as any,
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
        protocol: 1,
        title: '',
        authCertName: '',
    },
    filesystemDialog: {
        visible: false,
        machineId: 0,
        authCertName: '',
        protocol: 1,
        title: '',
        fileId: 0,
        path: '',
    },
    machineStatsDialog: {
        visible: false,
        stats: null,
        title: '',
        machineId: 0,
    },
    machineRecDialog: {
        visible: false,
        machineId: 0,
        title: '',
    },
    activeTermName: '',
    tabs: new Map<string, any>(),
});

const { infoDialog, serviceDialog, processDialog, fileDialog, machineStatsDialog, machineRecDialog } = toRefs(state);

const tagTreeRef: any = ref(null);

const autoOpenResourceStore = useAutoOpenResource();
const { autoOpenResource } = storeToRefs(autoOpenResourceStore);

let openIds: any = {};

const NodeTypeTagPath = new NodeType(TagTreeNode.TagPath).withLoadNodesFunc(async (node: TagTreeNode) => {
    // 加载标签树下的机器列表
    state.params.tagPath = node.key;
    state.params.pageNum = 1;
    state.params.pageSize = 1000;
    const res = await search();
    // 把list 根据name字段排序
    res.list = res.list.sort((a: any, b: any) => a.name.localeCompare(b.name));
    return res.list.map((x: any) =>
        new TagTreeNode(x.code, x.name, NodeTypeMachine)
            .withParams(x)
            .withDisabled(x.status == -1 && x.protocol == MachineProtocolEnum.Ssh.value)
            .withIcon({
                name: 'Monitor',
                color: '#409eff',
            })
    );
});

const NodeTypeMachine = new NodeType(MachineNodeType.Machine)
    .withLoadNodesFunc((node: TagTreeNode) => {
        const machine = node.params;
        // 获取授权凭证列表
        const authCerts = machine.authCerts;
        return authCerts.map((x: any) =>
            new TagTreeNode(x.name, x.username, NodeTypeAuthCert)
                .withParams({ ...machine, selectAuthCert: x })
                .withDisabled(machine.status == -1 && machine.protocol == MachineProtocolEnum.Ssh.value)
                .withIcon({
                    name: 'Ticket',
                    color: '#409eff',
                })
                .withIsLeaf(true)
        );
    })
    .withContextMenuItems([
        new ContextmenuItem('detail', 'common.detail').withIcon('More').withOnClick((node: any) => showInfo(node.params)),

        new ContextmenuItem('status', 'common.status')
            .withIcon('Compass')
            .withHideFunc((node: any) => node.params.protocol != MachineProtocolEnum.Ssh.value)
            .withOnClick((node: any) => showMachineStats(node.params)),

        new ContextmenuItem('process', 'machine.process')
            .withIcon('DataLine')
            .withHideFunc((node: any) => node.params.protocol != MachineProtocolEnum.Ssh.value)
            .withOnClick((node: any) => showProcess(node.params)),

        new ContextmenuItem('edit', 'machine.terminalPlayback')
            .withIcon('Compass')
            .withOnClick((node: any) => showRec(node.params))
            .withHideFunc((node: any) => actionBtns[perms.updateMachine] && node.params.enableRecorder == 1),
    ]);

const NodeTypeAuthCert = new NodeType(MachineNodeType.AuthCert)
    .withNodeDblclickFunc((node: TagTreeNode) => {
        openTerminal(node.params);
    })
    .withContextMenuItems([
        new ContextmenuItem('term', 'machine.openTerminal').withIcon('Monitor').withOnClick((node: any) => openTerminal(node.params)),
        new ContextmenuItem('term-ex', 'machine.newTabOpenTerminal').withIcon('Monitor').withOnClick((node: any) => openTerminal(node.params, true)),
        new ContextmenuItem('files', 'machine.fileManage').withIcon('FolderOpened').withOnClick((node: any) => showFileManage(node.params)),

        new ContextmenuItem('scripts', 'machine.scriptManage')
            .withIcon('Files')
            .withHideFunc((node: any) => node.params.protocol != MachineProtocolEnum.Ssh.value)
            .withOnClick((node: any) => serviceManager(node.params)),
    ]);

watch(
    () => autoOpenResource.value.machineCodePath,
    (codePath: any) => {
        autoOpenTerminal(codePath);
    }
);

watch(
    () => state.activeTermName,
    (newValue, oldValue) => {
        fitTerminal();

        oldValue && terminalRefs[oldValue]?.blur && terminalRefs[oldValue]?.blur();
        terminalRefs[newValue]?.focus && terminalRefs[newValue]?.focus();

        const nowTab = state.tabs.get(state.activeTermName);
        tagTreeRef.value.setCurrentKey(nowTab?.authCert);
    }
);

onMounted(() => {
    autoOpenTerminal(autoOpenResource.value.machineCodePath);
});

const autoOpenTerminal = (codePath: string) => {
    if (!codePath) {
        return;
    }

    const typeAndCodes = getTagTypeCodeByPath(codePath);
    const tagPath = typeAndCodes[TagResourceTypeEnum.Tag.value].join('/') + '/';

    const machineCode = typeAndCodes[TagResourceTypeEnum.Machine.value][0];
    state.defaultExpendKey = [tagPath, machineCode];

    const authCertName = typeAndCodes[TagResourceTypeEnum.AuthCert.value][0];
    setTimeout(() => {
        // 置空
        autoOpenResourceStore.setMachineCodePath('');
        tagTreeRef.value.setCurrentKey(authCertName);

        const acNode = tagTreeRef.value.getNode(authCertName);
        openTerminal(acNode.data.params);
    }, 1000);
};

const openTerminal = (machine: any, ex?: boolean) => {
    // 授权凭证名
    const ac = machine.selectAuthCert.name;

    // 新窗口打开
    if (ex) {
        if (machine.protocol == MachineProtocolEnum.Ssh.value) {
            const { href } = router.resolve({
                path: `/machine/terminal`,
                query: {
                    ac,
                    name: machine.name,
                },
            });
            window.open(href, '_blank');
            return;
        }
        if (machine.protocol == MachineProtocolEnum.Rdp.value) {
            const { href } = router.resolve({
                path: `/machine/terminal-rdp`,
                query: {
                    machineId: machine.id,
                    ac: ac,
                    name: machine.name,
                },
            });
            window.open(href, '_blank');
            return;
        }
    }

    let { name } = machine;
    const labelName = `${machine.selectAuthCert.username}@${name}`;

    // 同一个机器的终端打开多次，key后添加下划线和数字区分
    openIds[ac] = openIds[ac] ? ++openIds[ac] : 1;
    let sameIndex = openIds[ac];

    let key = `${ac}_${sameIndex}`;
    // 只保留name的15个字，超出部分只保留前后10个字符，中间用省略号代替
    const label = labelName.length > 15 ? labelName.slice(0, 10) + '...' + labelName.slice(-10) : labelName;

    let tab = {
        key,
        label: `${label}${sameIndex === 1 ? '' : ':' + sameIndex}`, // label组成为:总打开term次数+name+同一个机器打开的次数
        params: machine,
        authCert: ac,
        socketUrl: getMachineTerminalSocketUrl(ac),
    };

    state.tabs.set(key, tab);
    state.activeTermName = key;

    nextTick(() => {
        handleReconnect(tab);
    });
};

const serviceManager = (row: any) => {
    const authCert = row.selectAuthCert;
    state.serviceDialog.machineId = row.id;
    state.serviceDialog.visible = true;
    state.serviceDialog.authCertName = authCert.name;
    state.serviceDialog.title = `${row.name} => ${authCert.username}@${row.ip}`;
};

/**
 * 显示机器状态统计信息
 */
const showMachineStats = async (machine: any) => {
    state.machineStatsDialog.machineId = machine.id;
    state.machineStatsDialog.title = `${t('machine.machineState')}: ${machine.name} => ${machine.ip}`;
    state.machineStatsDialog.visible = true;
};

const search = async () => {
    const res = await machineApi.list.request(state.params);
    return res;
};

const showFileManage = (selectionData: any) => {
    const authCert = selectionData.selectAuthCert;
    if (selectionData.protocol == 1) {
        state.fileDialog.visible = true;
        state.fileDialog.protocol = selectionData.protocol;
        state.fileDialog.machineId = selectionData.id;
        state.fileDialog.authCertName = authCert.name;
        state.fileDialog.title = `${selectionData.name} => ${authCert.username}@${selectionData.ip}`;
    }

    if (selectionData.protocol == 2) {
        state.filesystemDialog.protocol = 2;
        state.filesystemDialog.machineId = selectionData.id;
        state.filesystemDialog.authCertName = authCert.name;
        state.filesystemDialog.fileId = selectionData.id;
        state.filesystemDialog.path = '/';
        state.filesystemDialog.title = t('machine.remoteFileDesktopManage');
        state.filesystemDialog.visible = true;
    }
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

const onRemoveTab = (targetName: string) => {
    let activeTermName = state.activeTermName;
    const tabNames = [...state.tabs.keys()];
    for (let i = 0; i < tabNames.length; i++) {
        const tabName = tabNames[i];
        if (tabName !== targetName) {
            continue;
        }

        state.tabs.delete(targetName);
        let info = state.tabs.get(targetName);
        if (info) {
            terminalRefs[info.key]?.close();
        }

        if (activeTermName != targetName) {
            break;
        }

        // 如果删除的tab是当前激活的tab，则切换到前一个或后一个tab
        const nextTab = tabNames[i + 1] || tabNames[i - 1];
        if (nextTab) {
            activeTermName = nextTab;
        } else {
            activeTermName = '';
        }

        state.activeTermName = activeTermName;
        break;
    }
};

const terminalStatusChange = (key: string, status: TerminalStatus) => {
    state.tabs.get(key).status = status;
};

const terminalRefs: any = {};
const setTerminalRef = (el: any, key: any) => {
    if (key) {
        terminalRefs[key] = el;
    }
};

const terminalWrapperRefs: any = {};
const setTerminalWrapperRef = (el: any, key: any) => {
    if (key) {
        terminalWrapperRefs[key] = el;
    }
};

const onResizeTagTree = () => {
    fitTerminal();
};

const fitTerminal = () => {
    setTimeout(() => {
        let info = state.tabs.get(state.activeTermName);
        if (info) {
            terminalRefs[info.key]?.fitTerminal && terminalRefs[info.key]?.fitTerminal();
        }
    });
};

const handleReconnect = (tab: any, force = false) => {
    let width = terminalWrapperRefs[tab.key].offsetWidth;
    let height = terminalWrapperRefs[tab.key].offsetHeight;
    terminalRefs[tab.key]?.init(width, height, force);
};
</script>

<style lang="scss">
.machine-terminal-tabs {
    --el-tabs-header-height: 30px;

    .el-tabs {
        --el-tabs-header-height: 30px;
    }

    .machine-terminal-tab-label {
        font-size: 12px;
    }
    .el-tabs__header {
        margin-bottom: 5px;
    }
    .el-tabs__item {
        padding: 0 8px !important;
    }
}
</style>
