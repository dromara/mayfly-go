import { ResourceTypeEnum } from '@/common/commonEnum';
import router from '@/router';
import { machineApi } from '@/views/ops/machine/api';
import { MachineProtocolEnum } from '@/views/ops/machine/enums';
import type { MachineVO, MachineAuthCert } from '@/views/ops/machine/types';
import { registerCommand, registerContributor, registerMenu, type TreeCommandCtx, type TreeNode, type TreeNodeData } from '@/views/ops/resource/tree';

export interface MachineNodeParams {
    id: number;
    code: string;
    name: string;
    ip: string;
    port?: number;
    protocol: number;
    status?: number;
    enableRecorder?: number;
    username?: string;
    tagPath?: string;
    selectAuthCert?: MachineAuthCert;
    authCerts?: MachineAuthCert[];
    [key: string]: unknown;
}
import { defineResourceConfig } from '@/views/ops/resource/resourceRegistry';
import { defineAsyncComponent } from 'vue';
import { createResourceOpTab, showResourceOpOverlay } from '../../resource/resourceOp';

const MachineList = defineAsyncComponent(() => import('../MachineList.vue'));
const TerminalTab = defineAsyncComponent(() => import('./tabs/TerminalTab.vue'));
const TerminalTabLabel = defineAsyncComponent(() => import('./tabs/TerminalTabLabel.vue'));
const FileTab = defineAsyncComponent(() => import('./tabs/FileTab.vue'));
const ScriptManage = defineAsyncComponent(() => import('../ScriptManage.vue'));
const MachineDetailDialog = defineAsyncComponent(() => import('./tabs/MachineDetailDialog.vue'));
const MachineStats = defineAsyncComponent(() => import('../MachineStats.vue'));
const ProcessList = defineAsyncComponent(() => import('../ProcessList.vue'));
const MachineRec = defineAsyncComponent(() => import('../MachineRec.vue'));
const NodeMachineAc = defineAsyncComponent(() => import('./NodeMachineAc.vue'));

const MachineIcon = {
    name: ResourceTypeEnum.Machine.extra.icon,
    color: ResourceTypeEnum.Machine.extra.iconColor,
};

const FileIcon = { name: 'FolderOpened', color: '#E6A23C' };
const AuthCertIcon = { name: 'Ticket', color: '#409eff' };

/**
 * machine 资源树节点 kind 常量（贡献者协议字符串，跨模块复用如 MachineSelectTree）
 */
export const MachineKind = 'machine';
export const MachineAuthCertKind = 'machine-auth-cert';

/** 创建终端 tab（双击凭证节点与"打开终端"菜单共用） */
const openTerminalTab = async (ctx: TreeCommandCtx) => {
    const m = ctx.node.params as unknown as MachineNodeParams;
    const key = `${m.code}.${m.selectAuthCert!.name}.${new Date().getTime()}`;
    createResourceOpTab({
        key,
        nodeKey: ctx.node.key,
        name: `${m.selectAuthCert!.username}@${m.name}`,

        component: TerminalTab,
        componentProps: {
            tabKey: key,
            machineId: m.id,
            authCertName: m.selectAuthCert!.name,
            protocol: m.protocol,
        },

        tabComponent: TerminalTabLabel,
        tabComponentProps: {
            icon: MachineIcon,
            status: 'disconnected',
        },
    });
};

// ---------------------------------- 命令注册 ----------------------------------

// 双击凭证节点打开终端
registerCommand({
    id: 'machine.terminal.open',
    txt: '',
    handler: openTerminalTab,
});

registerCommand({
    id: 'machine.terminal.menu',
    txt: 'machine.openTerminal',
    icon: 'Monitor',
    handler: openTerminalTab,
});

registerCommand({
    id: 'machine.terminal.newTab',
    txt: 'machine.newTabOpenTerminal',
    icon: 'Monitor',
    handler: async (ctx: TreeCommandCtx) => {
        const m = machineParams(ctx.node);
        const ac = m.selectAuthCert?.name || '';

        if (m.protocol == MachineProtocolEnum.Ssh.value) {
            const { href } = router.resolve({
                path: `/machine/terminal`,
                query: {
                    ac,
                    name: m.name,
                },
            });
            window.open(href, '_blank');
            return;
        }
        if (m.protocol == MachineProtocolEnum.Rdp.value) {
            const { href } = router.resolve({
                path: `/machine/terminal-rdp`,
                query: {
                    machineId: m.id,
                    ac: ac,
                    name: m.name,
                },
            });
            window.open(href, '_blank');
            return;
        }
    },
});

/** 命令回调统一取参（node.params 可选，machine 节点均携带 params） */
const machineParams = (node: TreeNode) => (node.params ?? {}) as MachineNodeParams;

registerCommand({
    id: 'machine.file.manage',
    txt: 'machine.fileManage',
    icon: 'FolderOpened',
    handler: async (ctx: TreeCommandCtx) => {
        const m = machineParams(ctx.node);
        const acName = m.selectAuthCert?.name || '';

        // 直接打开文件管理 tab，FileTab 内部会处理配置选择
        const tabKey = `${m.code}.${acName}`;
        createResourceOpTab({
            key: tabKey,
            nodeKey: ctx.node.key,
            name: `${m.selectAuthCert?.username || ''}@${m.name}`,
            component: FileTab,
            tabComponentProps: {
                icon: FileIcon,
            },
            componentProps: {
                tabKey: tabKey,
                machineId: m.id,
                authCertName: acName,
                protocol: m.protocol,
            },
        });
    },
});

registerCommand({
    id: 'machine.script.manage',
    txt: 'machine.scriptManage',
    icon: 'Files',
    handler: async (ctx: TreeCommandCtx) => {
        const m = machineParams(ctx.node);

        // 显示脚本管理弹窗（已存在则更新 props，否则注册）
        showResourceOpOverlay('script_manage', ScriptManage, {
            machineId: m.id,
            authCertName: m.selectAuthCert?.name || '',
            title: `${m.name} => ${m.selectAuthCert?.username || ''}@${m.ip}`,
        });
    },
});

registerCommand({
    id: 'machine.detail',
    txt: 'common.detail',
    icon: 'More',
    handler: async (ctx: TreeCommandCtx) => {
        const m = machineParams(ctx.node);

        // 显示机器详情弹窗（已存在则更新 props，否则注册）
        showResourceOpOverlay('machine_detail', MachineDetailDialog, {
            code: m.code,
        });
    },
});

registerCommand({
    id: 'machine.status',
    txt: 'common.status',
    icon: 'Compass',
    handler: async (ctx: TreeCommandCtx) => {
        const m = machineParams(ctx.node);

        // 显示机器状态弹窗
        showResourceOpOverlay('machine_stats', MachineStats, {
            machineId: m.id,
            title: `${m.name} => ${m.ip}`,
        });
    },
});

registerCommand({
    id: 'machine.process',
    txt: 'machine.process',
    icon: 'DataLine',
    handler: async (ctx: TreeCommandCtx) => {
        const m = machineParams(ctx.node);

        // 显示进程列表弹窗
        showResourceOpOverlay('machine_process', ProcessList, {
            machineId: m.id,
            title: `${m.name} => ${m.ip}`,
        });
    },
});

registerCommand({
    id: 'machine.rec',
    txt: 'machine.terminalPlayback',
    icon: 'Compass',
    handler: async (ctx: TreeCommandCtx) => {
        const m = machineParams(ctx.node);

        showResourceOpOverlay('machine_rec', MachineRec, {
            machineId: m.id,
            title: `${m.name} => ${m.ip}`,
        });
    },
});

// ---------------------------------- 菜单挂载 ----------------------------------

const isSsh = (ctx: TreeCommandCtx) => ctx.node.params?.protocol == MachineProtocolEnum.Ssh.value;

registerMenu({ command: 'machine.detail', kinds: [MachineKind], order: 1 });
registerMenu({ command: 'machine.status', kinds: [MachineKind], order: 2, when: isSsh });
registerMenu({ command: 'machine.process', kinds: [MachineKind], order: 3, when: isSsh });
// 终端回放：仅开启录制（enableRecorder == 1）时隐藏
registerMenu({ command: 'machine.rec', kinds: [MachineKind], order: 4, when: (ctx) => ctx.node.params?.enableRecorder != 1 });

registerMenu({ command: 'machine.terminal.menu', kinds: [MachineAuthCertKind], order: 1, permission: 'machine:terminal' });
registerMenu({ command: 'machine.terminal.newTab', kinds: [MachineAuthCertKind], order: 2, permission: 'machine:terminal' });
registerMenu({ command: 'machine.file.manage', kinds: [MachineAuthCertKind], order: 3 });
registerMenu({ command: 'machine.script.manage', kinds: [MachineAuthCertKind], order: 4, when: isSsh });
registerMenu({ command: 'machine.terminal.open', kinds: [MachineAuthCertKind], trigger: 'dblclick' });

// ---------------------------------- 贡献者注册 ----------------------------------

// 机器节点：loadRoots 供类型分组节点展开（列出标签下机器），loadChildren 展开机器列出授权凭证
registerContributor({
    kind: MachineKind,
    resourceType: ResourceTypeEnum.Machine.value,
    hasChildren: true,
    icon: MachineIcon,
    loadRoots: async (groupNode) => {
        const res = await machineApi.list.request({ tagPath: groupNode.params?.tagPath as string });
        return (res?.list ?? [])
            .sort((a: MachineVO, b: MachineVO) => a.name.localeCompare(b.name))
            .map((x: MachineVO) => ({
                key: `${x.code}`,
                kind: MachineKind,
                label: x.name,
                icon: MachineIcon,
                disabled: x.status == -1 && x.protocol == MachineProtocolEnum.Ssh.value,
                params: x as unknown as Record<string, unknown>,
            }));
    },
    loadChildren: async (node) => {
        const machine = node.params as unknown as MachineNodeParams;
        const authCerts = machine.authCerts || [];
        return authCerts.map(
            (x: MachineAuthCert): TreeNodeData => ({
                key: `${node.key}.${x.name}`,
                kind: MachineAuthCertKind,
                label: x.username,
                icon: AuthCertIcon,
                disabled: machine.status == -1 && machine.protocol == MachineProtocolEnum.Ssh.value,
                params: { ...machine, selectAuthCert: x },
            })
        );
    },
});

// 授权凭证节点（叶子，双击打开终端）：选择/引用场景的终级粒度（选到凭证才是完整授权目标）
registerContributor({
    kind: MachineAuthCertKind,
    selectable: true,
    renderer: NodeMachineAc,
});

export default defineResourceConfig({
    order: 1,
    resourceType: ResourceTypeEnum.Machine.value,
    manager: {
        componentConf: {
            component: MachineList,
            icon: MachineIcon,
            name: 'tag.machine',
        },
        permCode: 'machine',
        countKey: 'machine',
    },
});
