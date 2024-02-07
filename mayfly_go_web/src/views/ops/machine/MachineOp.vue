<template>
    <div class="flex-all-center">
        <!--    文档： https://antoniandre.github.io/splitpanes/    -->
        <Splitpanes class="default-theme" @resized="onResizeTagTree">
            <Pane size="20" max-size="30">
                <tag-tree
                    class="machine-terminal-tree"
                    ref="tagTreeRef"
                    :resource-type="TagResourceTypeEnum.Machine.value"
                    :tag-path-node-type="NodeTypeTagPath"
                >
                    <template #prefix="{ data }">
                        <SvgIcon v-if="data.icon && data.params.status == 1" :name="data.icon.name" :color="data.icon.color" />
                        <SvgIcon v-if="data.icon && data.params.status == -1" :name="data.icon.name" color="var(--el-color-danger)" />
                    </template>

                    <template #suffix="{ data }">
                        <span style="color: #c4c9c4; font-size: 9px" v-if="data.type.value == MachineNodeType.Machine">{{
                            ` ${data.params.username}@${data.params.ip}:${data.params.port}`
                        }}</span>
                    </template>
                </tag-tree>
            </Pane>

            <Pane>
                <div class="machine-terminal-tabs card pd5">
                    <el-tabs
                        v-if="state.tabs.size > 0"
                        type="card"
                        @tab-remove="onRemoveTab"
                        @tab-change="onTabChange"
                        style="width: 100%"
                        v-model="state.activeTermName"
                        class="h100"
                    >
                        <el-tab-pane class="h100" closable v-for="dt in state.tabs.values()" :label="dt.label" :name="dt.key" :key="dt.key">
                            <template #label>
                                <el-popconfirm @confirm="handleReconnect(dt.key)" title="确认重新连接?">
                                    <template #reference>
                                        <el-icon class="mr5" :color="dt.status == 1 ? '#67c23a' : '#f56c6c'" :title="dt.status == 1 ? '' : '点击重连'"
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
                                            <el-descriptions-item label="机器名"> {{ dt.params?.name }} </el-descriptions-item>
                                            <el-descriptions-item label="host"> {{ dt.params?.ip }} : {{ dt.params?.port }} </el-descriptions-item>
                                            <el-descriptions-item label="username"> {{ dt.params?.username }} </el-descriptions-item>
                                            <el-descriptions-item label="remark"> {{ dt.params?.remark }} </el-descriptions-item>
                                        </el-descriptions>
                                    </template>
                                </el-popover>
                            </template>

                            <div class="terminal-wrapper" :style="{ height: `calc(100vh - 155px)` }">
                                <TerminalBody
                                    @status-change="terminalStatusChange(dt.key, $event)"
                                    :ref="(el) => setTerminalRef(el, dt.key)"
                                    :socket-url="dt.socketUrl"
                                />
                            </div>
                        </el-tab-pane>
                    </el-tabs>

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

                    <process-list v-model:visible="processDialog.visible" v-model:machineId="processDialog.machineId" />

                    <script-manage :title="serviceDialog.title" v-model:visible="serviceDialog.visible" v-model:machineId="serviceDialog.machineId" />

                    <file-conf-list :title="fileDialog.title" v-model:visible="fileDialog.visible" v-model:machineId="fileDialog.machineId" />

                    <machine-stats v-model:visible="machineStatsDialog.visible" :machineId="machineStatsDialog.machineId" :title="machineStatsDialog.title" />

                    <machine-rec v-model:visible="machineRecDialog.visible" :machineId="machineRecDialog.machineId" :title="machineRecDialog.title" />
                </div>
            </Pane>
        </Splitpanes>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, defineAsyncComponent } from 'vue';
import { useRouter } from 'vue-router';
import { machineApi, getMachineTerminalSocketUrl } from './api';
import { dateFormat } from '@/common/utils/date';
import { hasPerms } from '@/components/auth/auth';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { NodeType, TagTreeNode } from '../component/tag';
import TagTree from '../component/TagTree.vue';
import { Splitpanes, Pane } from 'splitpanes';
import { ContextmenuItem } from '@/components/contextmenu/index';
// 组件
const ScriptManage = defineAsyncComponent(() => import('./ScriptManage.vue'));
const FileConfList = defineAsyncComponent(() => import('./file/FileConfList.vue'));
const MachineStats = defineAsyncComponent(() => import('./MachineStats.vue'));
const MachineRec = defineAsyncComponent(() => import('./MachineRec.vue'));
const ProcessList = defineAsyncComponent(() => import('./ProcessList.vue'));
import TerminalBody from '@/components/terminal/TerminalBody.vue';
import { TerminalStatus } from '@/components/terminal/common';

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
}

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

const NodeTypeTagPath = new NodeType(TagTreeNode.TagPath).withLoadNodesFunc(async (node: any) => {
    // 加载标签树下的机器列表
    state.params.tagPath = node.key;
    state.params.pageNum = 1;
    state.params.pageSize = 1000;
    const res = await search();
    // 把list 根据name字段排序
    res.list = res.list.sort((a: any, b: any) => a.name.localeCompare(b.name));
    return res.list.map((x: any) =>
        new TagTreeNode(x.id, x.name, NodeTypeMachine(x))
            .withParams(x)
            .withDisabled(x.status == -1)
            .withIcon({
                name: 'Monitor',
                color: '#409eff',
            })
            .withIsLeaf(true)
    );
});

let openIds = {};

const NodeTypeMachine = (machine: any) => {
    let contextMenuItems = [];
    contextMenuItems.push(new ContextmenuItem('term', '打开终端').withIcon('Monitor').withOnClick(() => openTerminal(machine)));
    contextMenuItems.push(new ContextmenuItem('term-ex', '打开终端(新窗口)').withIcon('Monitor').withOnClick(() => openTerminal(machine, true)));
    contextMenuItems.push(new ContextmenuItem('detail', '详情').withIcon('More').withOnClick(() => showInfo(machine)));
    contextMenuItems.push(new ContextmenuItem('status', '状态').withIcon('Compass').withOnClick(() => showMachineStats(machine)));
    contextMenuItems.push(new ContextmenuItem('process', '进程').withIcon('DataLine').withOnClick(() => showProcess(machine)));

    if (actionBtns[perms.updateMachine] && machine.enableRecorder == 1) {
        contextMenuItems.push(new ContextmenuItem('edit', '终端回放').withIcon('Compass').withOnClick(() => showRec(machine)));
    }

    contextMenuItems.push(new ContextmenuItem('files', '文件管理').withIcon('FolderOpened').withOnClick(() => showFileManage(machine)));
    contextMenuItems.push(new ContextmenuItem('scripts', '脚本管理').withIcon('Files').withOnClick(() => serviceManager(machine)));
    return new NodeType(MachineNodeType.Machine).withContextMenuItems(contextMenuItems).withNodeDblclickFunc(() => {
        // for (let k of state.tabs.keys()) {
        //     // 存在该机器相关的终端tab，则直接激活该tab
        //     if (k.startsWith(`${machine.id}_${machine.username}_`)) {
        //         state.activeTermName = k;
        //         onTabChange();
        //         return;
        //     }
        // }

        openTerminal(machine);
    });
};

const openTerminal = (machine: any, ex?: boolean) => {
    // 新窗口打开
    if (ex) {
        const { href } = router.resolve({
            path: `/machine/terminal`,
            query: {
                id: machine.id,
                name: machine.name,
            },
        });
        window.open(href, '_blank');
        return;
    }

    let { name, id, username } = machine;

    // 同一个机器的终端打开多次，key后添加下划线和数字区分
    openIds[id] = openIds[id] ? ++openIds[id] : 1;
    let sameIndex = openIds[id];

    let key = `${id}_${username}_${sameIndex}`;
    // 只保留name的10个字，超出部分只保留前后4个字符，中间用省略号代替
    let label = name.length > 10 ? name.slice(0, 4) + '...' + name.slice(-4) : name;

    state.tabs.set(key, {
        key,
        label: `${label}${sameIndex === 1 ? '' : ':' + sameIndex}`, // label组成为:总打开term次数+name+同一个机器打开的次数
        params: machine,
        socketUrl: getMachineTerminalSocketUrl(id),
    });
    state.activeTermName = key;
    fitTerminal();
};

const serviceManager = (row: any) => {
    state.serviceDialog.machineId = row.id;
    state.serviceDialog.visible = true;
    state.serviceDialog.title = `${row.name} => ${row.ip}`;
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
    const res = await machineApi.list.request(state.params);
    return res;
};

const showFileManage = (selectionData: any) => {
    state.fileDialog.visible = true;
    state.fileDialog.machineId = selectionData.id;
    state.fileDialog.title = `${selectionData.name} => ${selectionData.ip}`;
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

const onRemoveTab = (targetName: string) => {
    let activeTermName = state.activeTermName;
    const tabNames = [...state.tabs.keys()];
    for (let i = 0; i < tabNames.length; i++) {
        const tabName = tabNames[i];
        if (tabName !== targetName) {
            continue;
        }
        const nextTab = tabNames[i + 1] || tabNames[i - 1];
        if (nextTab) {
            activeTermName = nextTab;
        } else {
            activeTermName = '';
        }

        let info = state.tabs.get(targetName);
        if (info) {
            terminalRefs[info.key]?.close();
        }

        state.tabs.delete(targetName);
        state.activeTermName = activeTermName;
        onTabChange();
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

const onResizeTagTree = () => {
    fitTerminal();
};

const onTabChange = () => {
    fitTerminal();
};

const fitTerminal = () => {
    setTimeout(() => {
        let info = state.tabs.get(state.activeTermName);
        if (info) {
            terminalRefs[info.key]?.resize();
            terminalRefs[info.key]?.focus();
        }
    }, 100);
};

const handleReconnect = (key: string) => {
    terminalRefs[key].init();
};
</script>

<style lang="scss">
.machine-terminal-tabs {
    height: calc(100vh - 108px);
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
