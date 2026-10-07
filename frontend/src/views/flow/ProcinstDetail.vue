<template>
    <el-drawer
        :append-to-body="false"
        :title="props.title"
        v-model="visible"
        :before-close="cancel"
        :size="FLOW_DRAWER.detail"
        body-class="p-2!"
        header-class="mb-2!"
        :destroy-on-close="true"
        :close-on-click-modal="!props.instTaskId"
    >
        <template #header>
            <DrawerHeader :header="title" :back="cancel" />
        </template>

        <el-tabs v-model="state.activeTab">
            <el-tab-pane :label="$t('common.basic')" name="basic">
                <div>
                    <el-divider content-position="left">{{ $t('flow.proc') }}</el-divider>
                    <el-descriptions :column="3" border>
                        <el-descriptions-item :span="1" :label="$t('flow.procdefName')">{{ procinst.procdefName }}</el-descriptions-item>
                        <el-descriptions-item :span="1" :label="$t('flow.bizType')">
                            <enum-tag :enums="FlowBizType" :value="procinst.bizType"></enum-tag>
                        </el-descriptions-item>
                        <el-descriptions-item :span="1" :label="$t('flow.initiator')">
                            <AccountInfo :username="procinst.creator || ''" />
                        </el-descriptions-item>

                        <el-descriptions-item :span="1" :label="$t('flow.procinstStatus')">
                            <enum-tag :enums="ProcinstStatus" :value="procinst.status"></enum-tag>
                        </el-descriptions-item>
                        <el-descriptions-item :span="1" :label="$t('flow.bizStatus')">
                            <enum-tag :enums="ProcinstBizStatus" :value="procinst.bizStatus"></enum-tag>
                        </el-descriptions-item>
                        <el-descriptions-item :span="1" :label="$t('flow.startingTime')">{{ formatDate(procinst.createTime) }}</el-descriptions-item>

                        <div v-if="procinst.duration">
                            <el-descriptions-item :span="1.5" :label="$t('flow.endTime')">{{ formatDate(procinst.endTime) }}</el-descriptions-item>
                            <el-descriptions-item :span="1.5" :label="$t('flow.duration')">{{ formatTime(procinst.duration) }}</el-descriptions-item>
                        </div>

                        <el-descriptions-item :span="3" :label="$t('common.remark')">
                            {{ procinst.remark }}
                        </el-descriptions-item>
                    </el-descriptions>
                </div>
            </el-tab-pane>

            <!-- 执行结果（业务信息）与流程图此前内联在「基本」里，和「审批记录」同级的信息却藏在不同层，
                 现在各自成 tab；审批态默认落在业务信息，因为那才是审批人要判断的内容 -->
            <el-tab-pane :label="$t('flow.bizInfo')" name="bizInfo">
                <component v-if="procinst.bizType" :is="bizComponents[procinst.bizType]" :procinst="procinst"></component>
            </el-tab-pane>

            <el-tab-pane :label="$t('flow.approveNode')" name="approveNode">
                <!-- 画布在隐藏容器里量到的宽度是 0，因此只在页签激活时挂载 -->
                <div v-if="flowDef && state.activeTab === 'approveNode'" class="h-75">
                    <FlowDesign disabled center :data="flowDef" />
                </div>
                <div v-else-if="!flowDef" class="empty-tip">{{ $t('flow.noFlowDiagram') }}</div>
            </el-tab-pane>

            <el-tab-pane :label="$t('flow.approvalRecord')" name="approvalRecord">
                <el-timeline>
                    <el-timeline-item
                        v-for="task in procinst.procinstTasks"
                        :key="task.id"
                        :timestamp="formatDate(task.createTime)"
                        :type="getTaskStatusType(task.status)"
                        :icon="getTaskStatusIcon(task.status)"
                        size="large"
                        placement="top"
                    >
                        <el-card shadow="hover" class="hover:shadow-md transition-shadow">
                            <div>
                                <div class="flex justify-between">
                                    <div>
                                        <el-text tag="b" size="large">{{ task.nodeName }}</el-text> -
                                        <el-text tag="b" size="large" type="primary">{{ task.handler || '/' }}</el-text>
                                    </div>
                                    <enum-tag :enums="ProcinstTaskStatus" :value="task.status" />
                                </div>

                                <div class="mt-2">
                                    <el-text class="ml-5" tag="b">{{ task.remark }}</el-text>
                                </div>
                            </div>
                        </el-card>
                    </el-timeline-item>
                </el-timeline>
            </el-tab-pane>
        </el-tabs>

        <template #footer v-if="props.instTaskId">
            <!-- 审批动作常驻底部：不随页签切换而消失，否则「填意见」和「点确定」分处两个页签 -->
            <div class="approve-footer">
                <auto-form v-model="form" :items="approveItems" label-position="top" />
                <div class="approve-actions">
                    <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">{{ $t('common.confirm') }}</el-button>
                </div>
            </div>
        </template>
    </el-drawer>
</template>

<script lang="ts" setup>
import { formatDate, formatTime } from '@/common/utils/format';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import EnumTag from '@/components/enum-tag/EnumTag.vue';
import { Msg } from '@/hooks/useI18n';
import AccountInfo from '@/views/system/account/components/AccountInfo.vue';
import { defineAsyncComponent, reactive, shallowReactive, toRefs, watch } from 'vue';
import { procinstApi, procinstTaskApi } from './api';
import FlowDesign from './components/flowdesign/FlowDesign.vue';
import { FlowBizType, ProcinstBizStatus, ProcinstStatus, ProcinstTaskStatus } from './enums';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import type { Procinst, ProcinstTask, HisProcinstOp, FlowNode, FlowDef } from './types';
import { FLOW_DRAWER } from '@/views/flow/drawerSize';

const DbSqlExecBiz = defineAsyncComponent(() => import('./flowbiz/dbms/DbSqlExecBiz.vue'));
const RedisRunCmdBiz = defineAsyncComponent(() => import('./flowbiz/redis/RedisRunCmdBiz.vue'));

const props = defineProps({
    procinstId: {
        type: Number,
    },
    // 流程实例任务id（存在则展示审批相关信息）
    instTaskId: {
        type: Number,
    },
    title: {
        type: String,
    },
});

const visible = defineModel<boolean>('visible', { default: false });

// 本组件常驻复用（列表页只挂一份），构造期 props.instTaskId 还没赋值，所以默认页签必须每次打开重设：
// 审批态先落在业务信息（那才是要判断的内容），纯查看详情从基本信息看起
watch(visible, (opened) => {
    if (opened) state.activeTab = props.instTaskId ? 'bizInfo' : 'basic';
});

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

// 业务组件
const bizComponents = shallowReactive<Record<string, unknown>>({
    db_sql_exec_flow: DbSqlExecBiz,
    redis_run_cmd_flow: RedisRunCmdBiz,
});

const state = reactive({
    // 初值只是占位：真正落在哪个页签由打开时的 instTaskId 决定（见下方 watch）
    activeTab: 'basic',
    procinst: {} as Procinst,
    flowDef: null as FlowDef | null,
    tasks: [] as unknown[],
    form: {
        status: ProcinstTaskStatus.Pass.value as number,
        remark: '',
    },
    saveBtnLoading: false,
    sortable: '' as string,
});

const { procinst, flowDef, form, saveBtnLoading } = toRefs(state);

/** 审批表单声明 */
const approveItems: AutoFormItem[] = [
    // 枚举选项需用 type:'enum'（'select' 只读 options，会给 enums 导致下拉无选项）
    { prop: 'status', label: 'flow.approveResult', type: 'enum', required: true, enums: ProcinstTaskStatus },
    { prop: 'remark', label: 'common.remark', type: 'textarea', props: { clearable: true }, placeholder: 'common.remark' },
];

watch(
    () => props.procinstId,
    async (newValue: number | undefined) => {
        state.form.status = ProcinstTaskStatus.Pass.value;
        state.form.remark = '';

        if (!newValue) {
            state.procinst = {} as Procinst;
            state.flowDef = null;
            return;
        }

        state.procinst = await procinstApi.detail.request({ id: newValue });

        const flowdef = JSON.parse(state.procinst.flowDef) as FlowDef;
        procinstApi.hisOp.request({ id: newValue }).then((res: HisProcinstOp[]) => {
            const nodeKey2Ops = res.reduce(
                (acc: Record<string, HisProcinstOp[]>, item: HisProcinstOp) => {
                    const key = item.nodeKey;
                    if (!acc[key]) {
                        acc[key] = [];
                    }
                    acc[key].push(item);
                    return acc;
                },
                {} as Record<string, HisProcinstOp[]>
            );

            const nodeKey2Tasks = state.procinst.procinstTasks?.reduce(
                (acc: Record<string, ProcinstTask[]>, item: ProcinstTask) => {
                    const key = item.nodeKey;
                    if (!acc[key]) {
                        acc[key] = [];
                    }
                    acc[key].push(item);
                    return acc;
                },
                {} as Record<string, ProcinstTask[]>
            );

            flowdef.nodes.forEach((node: FlowNode) => {
                const key = node.key;
                if (nodeKey2Ops[key]) {
                    if (!node.extra) {
                        node.extra = {};
                    }
                    node.extra.opLog = nodeKey2Ops[key][0];
                    node.extra.tasks = nodeKey2Tasks?.[key];
                }
            });

            state.flowDef = flowdef;
        });
    }
);

const btnOk = async () => {
    const status = state.form.status;
    let api = procinstTaskApi.passTask;
    if (status === ProcinstTaskStatus.Back.value) {
        api = procinstTaskApi.backTask;
    } else if (status === ProcinstTaskStatus.Reject.value) {
        api = procinstTaskApi.rejectTask;
    }

    try {
        state.saveBtnLoading = true;
        await api.request({ id: props.instTaskId, remark: state.form.remark });
        Msg.operateSuccess();
        cancel();
        emit('val-change');
    } finally {
        state.saveBtnLoading = false;
    }
};

const cancel = () => {
    visible.value = false;
    emit('cancel');
};

const getTaskStatusIcon = (status: number) => {
    if (status === ProcinstTaskStatus.Pass.value) {
        return 'Check';
    } else if (status === ProcinstTaskStatus.Back.value) {
        return 'Close';
    } else if (status === ProcinstTaskStatus.Reject.value) {
        return 'Close';
    }
    return 'SemiSelect';
};

const getTaskStatusType = (status: number) => {
    if (status === ProcinstTaskStatus.Pass.value) {
        return 'success';
    } else if (status === ProcinstTaskStatus.Back.value) {
        return 'warning';
    } else if (status === ProcinstTaskStatus.Reject.value) {
        return 'danger';
    }
    return 'primary';
};
</script>
<style lang="scss">
/* 审批区常驻底部：表单占满宽度、按钮右对齐，避免抽屉 footer 的默认居中把按钮挤到表单上方 */
.approve-footer {
    text-align: left;
}

.approve-footer .approve-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 8px;
}

.empty-tip {
    font-size: 12px;
    color: var(--el-text-color-secondary);
}
</style>
