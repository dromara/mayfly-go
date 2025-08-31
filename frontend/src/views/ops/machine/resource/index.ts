import { ContextmenuItem } from '@/components/contextmenu';
import { ResourceTypeEnum } from '@/common/commonEnum';
import { defineAsyncComponent } from 'vue';
import { NodeType, TagTreeNode, ResourceComponentConfig, ResourceConfig } from '@/views/ops/component/tag';
import { machineApi } from '@/views/ops/machine/api';
import { MachineProtocolEnum } from '@/views/ops/machine/enums';

const MachineList = defineAsyncComponent(() => import('../MachineList.vue'));
const MachineOp = defineAsyncComponent(() => import('./MachineOp.vue'));

const NodeMachineAc = defineAsyncComponent(() => import('./NodeMachineAc.vue'));

const MachineIcon = {
    name: ResourceTypeEnum.Machine.extra.icon,
    color: ResourceTypeEnum.Machine.extra.iconColor,
};

export const MachineOpComp: ResourceComponentConfig = {
    name: 'tag.machineOp',
    component: MachineOp,
    icon: MachineIcon,
};

export const NodeTypeMachineTag = new NodeType(TagTreeNode.TagPath).withLoadNodesFunc(async (node: TagTreeNode) => {
    node.ctx?.addResourceComponent(MachineOpComp);
    // 加载标签树下的机器列表
    const res = await machineApi.list.request({ tagPath: node.params.tagPath });
    // 把list 根据name字段排序
    return res?.list
        .sort((a: any, b: any) => a.name.localeCompare(b.name))
        .map((x: any) =>
            TagTreeNode.new(node, x.code, x.name, NodeTypeMachine)
                .withParams(x)
                .withDisabled(x.status == -1 && x.protocol == MachineProtocolEnum.Ssh.value)
                .withIcon(MachineIcon)
        );
});

const NodeTypeMachine = new NodeType(11)
    .withLoadNodesFunc((node: TagTreeNode) => {
        const machine = node.params;
        // 获取授权凭证列表
        const authCerts = machine.authCerts;
        return authCerts.map((x: any) =>
            TagTreeNode.new(node, x.name, x.username, NodeTypeAuthCert)
                .withNodeComponent(NodeMachineAc)
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
        new ContextmenuItem('detail', 'common.detail').withIcon('More').withOnClick(async (node: TagTreeNode) => {
            (await node.ctx?.addResourceComponent(MachineOpComp)).showInfo(node.params);
        }),

        new ContextmenuItem('status', 'common.status')
            .withIcon('Compass')
            .withHideFunc((node: any) => node.params.protocol != MachineProtocolEnum.Ssh.value)
            .withOnClick(async (node: TagTreeNode) => {
                (await node.ctx?.addResourceComponent(MachineOpComp)).showMachineStats(node.params);
            }),

        new ContextmenuItem('process', 'machine.process')
            .withIcon('DataLine')
            .withHideFunc((node: any) => node.params.protocol != MachineProtocolEnum.Ssh.value)
            .withOnClick(async (node: TagTreeNode) => {
                (await node.ctx?.addResourceComponent(MachineOpComp)).showProcess(node.params);
            }),

        new ContextmenuItem('edit', 'machine.terminalPlayback')
            .withIcon('Compass')
            .withOnClick(async (node: TagTreeNode) => {
                (await node.ctx?.addResourceComponent(MachineOpComp)).showRec(node.params);
            })
            .withHideFunc((node: any) => node.params.enableRecorder == 1),
    ]);

const NodeTypeAuthCert = new NodeType(12)
    .withNodeDblclickFunc(async (node: TagTreeNode) => {
        (await node.ctx?.addResourceComponent(MachineOpComp)).openTerminal(node.params);
    })
    .withContextMenuItems([
        new ContextmenuItem('term', 'machine.openTerminal').withIcon('Monitor').withOnClick(async (node: TagTreeNode) => {
            (await node.ctx?.addResourceComponent(MachineOpComp))?.openTerminal(node.params);
        }),
        new ContextmenuItem('term-ex', 'machine.newTabOpenTerminal').withIcon('Monitor').withOnClick(async (node: TagTreeNode) => {
            (await node.ctx?.addResourceComponent(MachineOpComp))?.openTerminal(node.params, true);
        }),
        new ContextmenuItem('files', 'machine.fileManage').withIcon('FolderOpened').withOnClick(async (node: any) => {
            (await node.ctx?.addResourceComponent(MachineOpComp)).showFileManage(node.params);
        }),

        new ContextmenuItem('scripts', 'machine.scriptManage')
            .withIcon('Files')
            .withHideFunc((node: any) => node.params.protocol != MachineProtocolEnum.Ssh.value)
            .withOnClick(async (node: any) => {
                (await node.ctx?.addResourceComponent(MachineOpComp)).serviceManager(node.params);
            }),
    ]);

export default {
    order: 1,
    resourceType: ResourceTypeEnum.Machine.value,
    rootNodeType: NodeTypeMachineTag,
    manager: {
        componentConf: {
            component: MachineList,
            icon: MachineIcon,
            name: 'tag.machine',
        },
        permCode: 'machine',
        countKey: 'machine',
    },
} as ResourceConfig;
