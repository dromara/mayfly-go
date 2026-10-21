<template>
    <div class="h-full">
        <data-table ref="dataTableRef" :page-api="agentRunApi.list" :search-items="searchItems" v-model:query-form="params" :columns="columns">
            <template #tableHeader>
                <!-- 多 Agent 编排入口（阶段 4.1）：Planner 拆解 → 步骤执行 → Reviewer 审查 -->
                <el-button v-auth="perms.orchestrate" type="primary" icon="VideoPlay" plain @click="openOrchDialog()">
                    {{ $t('agentops.orchestration') }}
                </el-button>
            </template>

            <!-- 运行标题 + 形态标签（后端 runRole 按 i18n 渲染，不再从标题前缀反推） -->
            <template #title="{ data }">
                <el-tag v-if="data.runRole" size="small" effect="plain" :type="enumTagType(RunRoleEnum, data.runRole)">
                    {{ enumLabel(RunRoleEnum, data.runRole, t) }}
                </el-tag>
                <span class="ml-1">{{ data.title }}</span>
            </template>
        </data-table>

        <!-- 运行详情抽屉（与任务执行记录弹窗共用） -->
        <RunDetailDrawer v-model:visible="detail.visible" :run-id="detail.runId" :initial-tab="detail.tab" />

        <!-- 发起编排对话框：目标 + 可选编排约束/资源限制 -->
        <el-dialog v-model="orchDialog.visible" :title="$t('agentops.orchestration')" width="560px" destroy-on-close>
            <el-form label-width="120px">
                <el-form-item :label="$t('agentops.orchGoal')" required>
                    <el-input v-model="orchDialog.goal" type="textarea" :rows="4" :placeholder="$t('agentops.orchGoalTip')" />
                </el-form-item>
                <el-form-item :label="$t('agentops.instruction')">
                    <el-input v-model="orchDialog.instruction" type="textarea" :rows="2" :placeholder="$t('agentops.orchInstructionTip')" />
                </el-form-item>
                <el-form-item :label="$t('agentops.timeout')">
                    <el-input-number v-model="orchDialog.timeoutSeconds" :min="0" :step="30" />
                </el-form-item>
                <el-form-item :label="$t('agentops.maxStep')">
                    <el-input-number v-model="orchDialog.maxStep" :min="0" />
                </el-form-item>
                <el-form-item :label="$t('agentops.tokenBudget')">
                    <el-input-number v-model="orchDialog.tokenBudget" :min="0" :step="10000" />
                    <span class="agentops-gov-tip">{{ $t('agentops.tokenBudgetTip') }}</span>
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="orchDialog.visible = false">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="orchDialog.submitting" :disabled="!orchDialog.goal.trim()" @click="submitOrchestration">
                    {{ $t('common.confirm') }}
                </el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { TableColumn } from '@/components/data-table';
import DataTable from '@/components/data-table/DataTable.vue';
import { SearchItem } from '@/components/data-table/SearchForm';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { defineAsyncComponent, reactive, ref, toRefs, useTemplateRef, watch } from 'vue';
import { ElMessageBox } from 'element-plus';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';
import { agentOrchestrationApi, agentRunApi } from './api';
import { CANCELABLE_RUN_STATUS, ProfileEnum, RunFeedbackEnum, RunRoleEnum, RunStatusEnum, TERMINAL_RUN_STATUS, TriggerTypeEnum } from './enums';
import { enumLabel, enumTagType } from '@/common/Enum';
import { profileOptionsApi } from './useAgentProfiles';
import { AgentopsPerms as P } from './perms';
import type { AgentRun } from './types';

const RunDetailDrawer = defineAsyncComponent(() => import('./RunDetailDrawer.vue'));

const { t } = useI18n();
const route = useRoute();

// 权限码取自集中声明（与后端路由、迁移资源三方同源）
const perms = {
    cancel: P.runCancel,
    feedback: P.runFeedback,
    eval: P.runEval,
    orchestrate: P.orchestrationRun,
};

const searchItems = [
    // 触发源引用：从任务页「执行记录」跳转时会预填任务 key，必须可见可清除——
    // 曾因该条件隐形（URL 携带但表单不展示）导致「内置模板从未运行 → 过滤 0 条」
    // 被误认为执行历史没有数据
    SearchItem.input('triggerRef', 'agentops.triggerRef'),
    SearchItem.select('triggerType', 'agentops.triggerType').withEnum(TriggerTypeEnum),
    SearchItem.select('status', 'agentops.runStatus').withEnum(RunStatusEnum),
    // 反馈态可筛：点踩过的运行是需要人工复盘的样本，拿不到这个入口只能逐行点开看
    SearchItem.select('feedback', 'agentops.runFeedback').withEnum(RunFeedbackEnum),
    // 档位选项由后端目录驱动：历史运行可能用「当时可选、现在需许可」的档跑出来，筛选项不置灰
    SearchItem.select('profileId', 'agentops.profile').withOptionsApi(profileOptionsApi()),
];

const columns = ref([
    TableColumn.new('runNo', 'agentops.runNo'),
    // 标题带形态标签（规划/步骤/审查、重放、执行/校验）：同一目标会派生多条运行，
    // 没有标签时列表里是一排同名行，看不出哪条是容器哪条在干活
    TableColumn.new('title', 'agentops.title').isSlot(),
    TableColumn.new('triggerType', 'agentops.triggerType').typeTag(TriggerTypeEnum),
    TableColumn.new('profileId', 'agentops.profile').typeTag(ProfileEnum),
    TableColumn.new('status', 'agentops.runStatus').typeTag(RunStatusEnum),
    // 人类反馈列：up/down 分别渲成绿/红标签，未反馈就是空白格（不给每条都打「未反馈」标签，
    // 避免刷屏）；紧跟状态是因为两者合起来才是一条运行的完整结论：系统自判结果 + 人工判定结果
    TableColumn.new('feedback', 'agentops.runFeedback').typeTag(RunFeedbackEnum).alignCenter(),
    TableColumn.new('ownerUsername', 'agentops.owner'),
    TableColumn.new('totalTokens', 'agentops.totalTokens').alignCenter().isToken(),
    TableColumn.new('costMs', 'agentops.costMs').alignCenter().isDuration(),
    TableColumn.new('startedAt', 'agentops.startedAt').isTime(),
    TableColumn.new('action', 'common.operation')
        .setActions<AgentRun>([
            { key: 'detail', label: 'agentops.detail', type: 'primary', onClick: (row) => openDetail(row) },
            {
                // 回放 = 直接打开运行详情的「执行轨迹」页签（只读），不再跳进 AI 助手：
                // 查看一次运维运行不该顺带拿到改它的能力（输入框/中断决策）
                key: 'replay',
                label: 'agentops.replay',
                type: 'primary',
                disabled: (row) => !row.conversationId,
                onClick: (row) => openDetail(row, 'transcript'),
            },
            {
                key: 'cancel',
                label: 'agentops.cancelRun',
                type: 'danger',
                perm: perms.cancel,
                disabled: (row) => !CANCELABLE_RUN_STATUS.includes(row.status),
                onClick: (row) => cancelRun(row),
            },
            {
                key: 'feedbackUp',
                label: 'agentops.feedbackUp',
                type: 'success',
                perm: perms.feedback,
                disabled: (row) => !TERMINAL_RUN_STATUS.includes(row.status),
                onClick: (row) => feedback(row, 'up'),
            },
            {
                key: 'feedbackDown',
                label: 'agentops.feedbackDown',
                type: 'warning',
                perm: perms.feedback,
                disabled: (row) => !TERMINAL_RUN_STATUS.includes(row.status),
                onClick: (row) => feedback(row, 'down'),
            },
            {
                key: 'eval',
                label: 'agentops.eval',
                type: 'primary',
                perm: perms.eval,
                disabled: (row) => row.status !== RunStatusEnum.Succeeded.value,
                onClick: (row) => replayEval(row),
            },
        ])
        .fixedRight()
        .noShowOverflowTooltip()
        .alignCenter(),
]);

const dataTableRef = useTemplateRef<InstanceType<typeof DataTable>>('dataTableRef');

const state = reactive({
    // triggerRef/triggerType 支持从任务列表「执行记录」跳转时经 route.query 预填（setup 期同步读取，
    // 早于 DataTable 子组件 onMounted 的首次加载，故首屏即带过滤）
    params: {
        pageNum: 1,
        pageSize: 0,
        triggerRef: (route.query.triggerRef as string) || null,
        triggerType: (route.query.triggerType as string) || null,
        status: null,
        profileId: null,
    },
    detail: {
        visible: false,
        runId: null as number | null,
        /** 打开详情时落在哪个页签（行内「回放」直达轨迹） */
        tab: 'overview' as 'overview' | 'transcript',
    },
    // 发起编排对话框（多 Agent 编排，阶段 4.1）
    orchDialog: {
        visible: false,
        submitting: false,
        goal: '',
        instruction: '',
        timeoutSeconds: 0,
        maxStep: 0,
        tokenBudget: 0,
    },
});

const { params, detail, orchDialog } = toRefs(state);

// keep-alive 复用实例时再次从任务列表跳转：query 变化后同步过滤条件并重查（仅在本页激活时响应）
watch(
    () => route.query,
    (q) => {
        if (route.name !== 'AgentopsRunList') {
            return;
        }
        state.params.triggerRef = (q.triggerRef as string) || null;
        state.params.triggerType = (q.triggerType as string) || null;
        search();
    }
);

/**
 * 打开运行详情抽屉（概览 / 执行轨迹）。
 *
 * 轨迹页签的可见性由抽屉自己按 conversationId 兜底（无会话自动回落概览），
 * 行内按钮已按同一条件禁用，两处不是各自拍板的两个真源
 */
const openDetail = (row: AgentRun, tab: 'overview' | 'transcript' = 'overview') => {
    state.detail.runId = row.id;
    state.detail.tab = tab;
    state.detail.visible = true;
};

/** 取消排队/运行中的运行（急停）：终态运行不可取消（按钮已禁用，此处再兜底确认） */
const cancelRun = async (row: AgentRun) => {
    if (!(await useI18nConfirm('agentops.cancelRunConfirm'))) {
        return;
    }
    await agentRunApi.cancel.request({ id: row.id });
    Msg.success('agentops.cancelRunSuccess');
    search();
};

const search = () => {
    dataTableRef.value?.search();
};

/** 打开发起编排对话框（重置上一次输入） */
const openOrchDialog = () => {
    state.orchDialog.goal = '';
    state.orchDialog.instruction = '';
    state.orchDialog.timeoutSeconds = 0;
    state.orchDialog.maxStep = 0;
    state.orchDialog.tokenBudget = 0;
    state.orchDialog.visible = true;
};

/** 发起编排：异步执行，成功后提示并刷新列表（父运行与子运行都会出现在执行历史） */
const submitOrchestration = async () => {
    state.orchDialog.submitting = true;
    try {
        await agentOrchestrationApi.orchestrate.request({
            goal: state.orchDialog.goal.trim(),
            instruction: state.orchDialog.instruction.trim() || undefined,
            timeoutSeconds: state.orchDialog.timeoutSeconds || undefined,
            maxStep: state.orchDialog.maxStep || undefined,
            tokenBudget: state.orchDialog.tokenBudget || undefined,
        });
        Msg.success('agentops.orchestrationStarted');
        state.orchDialog.visible = false;
        search();
    } finally {
        state.orchDialog.submitting = false;
    }
};

/** 人类反馈（点赞/点踩 + 可选备注）：回流沉淀为操作员经验记忆（学习闭环） */
const feedback = async (row: AgentRun, value: 'up' | 'down') => {
    let note = '';
    try {
        const { value: input } = await ElMessageBox.prompt(t('agentops.feedbackNoteTip'), t('agentops.feedback'), {
            confirmButtonText: t('common.confirm'),
            cancelButtonText: t('common.cancel'),
            inputPlaceholder: t('agentops.feedbackNoteTip'),
        });
        note = (input || '').trim();
    } catch {
        return; // 取消
    }
    await agentRunApi.feedback.request({ id: row.id, feedback: value, note: note || undefined });
    Msg.success('agentops.feedbackSaved');
    search();
};

/** 离线评测回放：诊断档重跑原始输入 + 审查档对比打分（异步执行） */
const replayEval = async (row: AgentRun) => {
    if (!(await useI18nConfirm('agentops.evalConfirm'))) {
        return;
    }
    await agentRunApi.eval.request({ id: row.id });
    Msg.success('agentops.evalStarted');
    search();
};
</script>
