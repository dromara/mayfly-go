<template>
    <div class="h-full">
        <data-table
            ref="dataTableRef"
            :page-api="agentProposalApi.list"
            :search-items="searchItems"
            v-model:query-form="params"
            :show-selection="true"
            v-model:selection-data="state.selectionData"
            :columns="columns"
        >
            <template #tableHeader>
                <el-button v-auth="perms.del" :disabled="selectionData.length < 1" type="danger" icon="delete" @click="deleteProposal()">
                    {{ $t('common.delete') }}
                </el-button>
            </template>
        </data-table>

        <!-- 提案详情抽屉：展示锁定参数/理由/执行结果/校验结论/错误等列表页裁剪掉的大字段 -->
        <el-drawer v-model="detail.visible" :title="$t('agentops.proposalDetailTitle')" size="50%">
            <el-descriptions v-if="detail.data" :column="2" border label-width="110">
                <el-descriptions-item :label="$t('agentops.title')" :span="2">{{ detail.data.title }}</el-descriptions-item>
                <el-descriptions-item :label="$t('agentops.action')">{{ detail.data.action }}</el-descriptions-item>
                <el-descriptions-item :label="$t('agentops.risk')">{{ enumLabel(ProposalRiskEnum, detail.data.risk, t) }}</el-descriptions-item>
                <el-descriptions-item :label="$t('agentops.proposalStatus')">{{ enumLabel(ProposalStatusEnum, detail.data.status, t) }}</el-descriptions-item>
                <el-descriptions-item :label="$t('agentops.owner')">{{ detail.data.ownerUsername }}</el-descriptions-item>
                <!-- 审批链上的四个 id 全部给成可点开的链接：扫数字只能证明「有这条记录」，拿不到证据。
                     审计要的是从一条提案一路走到来源运行 / 审批工单 / 执行轨迹 / 校验轨迹，
                     而不是让人回到各列表页按 id 碰运气（列表页也没把这几个 id 当筛选维度开放） -->
                <el-descriptions-item v-if="detail.data.runId" :label="$t('agentops.sourceRunId')">
                    <el-link type="primary" underline="never" @click="openRunDetail(detail.data.runId)">{{ detail.data.runId }}</el-link>
                </el-descriptions-item>
                <el-descriptions-item v-if="detail.data.procInstId" :label="$t('agentops.procInstId')">
                    <el-link type="primary" underline="never" @click="openProcInst(detail.data.procInstId)">{{ detail.data.procInstId }}</el-link>
                </el-descriptions-item>
                <el-descriptions-item v-if="detail.data.execRunId" :label="$t('agentops.execRunId')">
                    <el-link type="primary" underline="never" @click="openRunDetail(detail.data.execRunId)">{{ detail.data.execRunId }}</el-link>
                </el-descriptions-item>
                <el-descriptions-item v-if="detail.data.verifyRunId" :label="$t('agentops.verifyRunId')">
                    <el-link type="primary" underline="never" @click="openRunDetail(detail.data.verifyRunId)">{{ detail.data.verifyRunId }}</el-link>
                </el-descriptions-item>
                <el-descriptions-item :label="$t('common.createTime')">{{ formatDate(detail.data.createTime || '') }}</el-descriptions-item>
                <el-descriptions-item :label="$t('agentops.params')" :span="2">
                    <pre class="agentops-proposal-pre">{{ detail.data.params }}</pre>
                </el-descriptions-item>
                <el-descriptions-item v-if="detail.data.reason" :label="$t('agentops.reason')" :span="2">
                    <MarkdownView class="agentops-proposal-md" :content="detail.data.reason" />
                </el-descriptions-item>
                <el-descriptions-item v-if="detail.data.execResult" :label="$t('agentops.execResult')" :span="2">
                    <MarkdownView class="agentops-proposal-md" :content="detail.data.execResult" />
                </el-descriptions-item>
                <el-descriptions-item v-if="detail.data.verifyConclusion" :label="$t('agentops.verifyConclusion')" :span="2">
                    <MarkdownView class="agentops-proposal-md" :content="detail.data.verifyConclusion" />
                </el-descriptions-item>
                <el-descriptions-item v-if="detail.data.err" :label="$t('agentops.err')" :span="2">
                    <pre class="agentops-proposal-pre agentops-proposal-pre--err">{{ detail.data.err }}</pre>
                </el-descriptions-item>
            </el-descriptions>
        </el-drawer>

        <!-- 发起该提案的那次运行的详情抽屉（只读轨迹）：与执行历史共用同一个详情容器，
             不另写一套回放界面 -->
        <RunDetailDrawer v-model:visible="runDetail.visible" :run-id="runDetail.runId" initial-tab="transcript" />

        <!-- 审批工单详情：与流程页共用同一个容器（基本信息/业务信息/审批节点/审批记录）。
             挂在根层级而非提案抽屉内：ProcinstDetail 的 append-to-body 为 false，嵌在父抽屉里会被裁切 -->
        <ProcinstDetail
            v-model:visible="procDetail.visible"
            :title="procDetail.title"
            :procinst-id="procDetail.procinstId"
            @cancel="procDetail.procinstId = 0"
        />
    </div>
</template>

<script lang="ts" setup>
import { enumLabel } from '@/common/Enum';
import { formatDate } from '@/common/utils/format';
import { TableColumn } from '@/components/data-table';
import DataTable from '@/components/data-table/DataTable.vue';
import { SearchItem } from '@/components/data-table/SearchForm';
import { MarkdownView } from '@/components/markdown';
import { Msg, useI18nDeleteConfirm, useI18nDetailTitle } from '@/hooks/useI18n';
import { defineAsyncComponent, reactive, ref, toRefs, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import { agentProposalApi } from './api';
import { DELETABLE_PROPOSAL_STATUS, ProposalRiskEnum, ProposalStatusEnum } from './enums';
import { AgentopsPerms as P } from './perms';
import type { AgentProposal } from './types';

const RunDetailDrawer = defineAsyncComponent(() => import('./RunDetailDrawer.vue'));
// 工单详情组件较重（内含 db/redis/提案三类业务表单子组件），按需加载，不让它进提案页首屏 chunk
const ProcinstDetail = defineAsyncComponent(() => import('@/views/flow/ProcinstDetail.vue'));

const { t } = useI18n();

// 权限码取自集中声明（与后端路由、迁移资源三方同源）
const perms = {
    del: P.proposalDel,
};

const searchItems = [
    SearchItem.input('title', 'agentops.title'),
    SearchItem.input('action', 'agentops.action'),
    SearchItem.select('status', 'agentops.proposalStatus').withEnum(ProposalStatusEnum),
    SearchItem.select('risk', 'agentops.risk').withEnum(ProposalRiskEnum),
];

const columns = ref([
    TableColumn.new('title', 'agentops.title'),
    TableColumn.new('action', 'agentops.action'),
    TableColumn.new('risk', 'agentops.risk').typeTag(ProposalRiskEnum),
    TableColumn.new('status', 'agentops.proposalStatus').typeTag(ProposalStatusEnum),
    TableColumn.new('ownerUsername', 'agentops.owner'),
    TableColumn.new('createTime', 'common.createTime').isTime(),
    TableColumn.new('operation', 'common.operation')
        .setActions<AgentProposal>([
            { key: 'detail', label: 'agentops.detail', type: 'primary', onClick: (row) => openDetail(row) },
            {
                // 回放 = 打开本次提案来源运行的执行轨迹（只读），不再跳进 AI 助手
                key: 'replay',
                label: 'agentops.replay',
                type: 'primary',
                disabled: (row) => !row.runId,
                onClick: (row) => openRunReplay(row),
            },
            {
                key: 'delete',
                label: 'common.delete',
                type: 'danger',
                perm: perms.del,
                // 仅终态提案可删（在途审批拒绝删除，与后端 entity.IsProposalTerminal 一致）
                disabled: (row) => !DELETABLE_PROPOSAL_STATUS.includes(row.status),
                onClick: (row) => deleteProposal(row),
            },
        ])
        .fixedRight()
        .noShowOverflowTooltip()
        .alignCenter(),
]);

const dataTableRef = useTemplateRef<InstanceType<typeof DataTable>>('dataTableRef');

const state = reactive({
    params: {
        pageNum: 1,
        pageSize: 0,
        title: null,
        action: null,
        status: null,
        risk: null,
    },
    selectionData: [] as AgentProposal[],
    detail: {
        visible: false,
        data: null as AgentProposal | null,
    },
    // 来源/执行/校验运行的详情抽屉（轨迹只读回放），与行内「回放」共用
    runDetail: {
        visible: false,
        runId: null as number | null,
    },
    // 审批工单详情抽屉
    procDetail: {
        visible: false,
        title: '',
        procinstId: 0,
    },
});

const { selectionData, params, detail, runDetail, procDetail } = toRefs(state);

/** 打开某次关联运行的详情（来源/执行/校验同一入口）：无关联运行时链接根本不会渲染 */
const openRunDetail = (runId: number) => {
    state.runDetail.runId = runId;
    state.runDetail.visible = true;
};

/** 打开审批工单详情：审计要能从提案走到「谁在哪一步批了什么」 */
const openProcInst = (procinstId: number) => {
    state.procDetail.procinstId = procinstId;
    state.procDetail.title = useI18nDetailTitle('flow.proc');
    state.procDetail.visible = true;
};

/** 打开提案来源运行的执行轨迹（无关联运行的行按钮已禁用） */
const openRunReplay = (row: AgentProposal) => openRunDetail(row.runId);

// 枚举文案统一走 common/Enum 的 enumLabel（颜色走 enumTagType），不再各页写一份

const openDetail = async (row: AgentProposal) => {
    state.detail.data = await agentProposalApi.detail.request({ id: row.id });
    state.detail.visible = true;
};

/**
 * 删除提案：传 row 删单条，否则删表格勾选项（批量）。
 * 仅终态提案可删（在途审批的 pending/approved 由后端拒绝，行内按钮已禁用）。
 */
const deleteProposal = async (row?: AgentProposal) => {
    const targets = row ? [row] : state.selectionData;
    if (targets.length < 1) {
        return;
    }
    try {
        if (!(await useI18nDeleteConfirm(targets.map((x) => x.title).join('、')))) {
            return;
        }
        await agentProposalApi.del.request({ ids: targets.map((x) => x.id).join(',') });
        Msg.deleteSuccess();
        search();
    } catch (err) {
        // 取消删除
    }
};

const search = () => {
    dataTableRef.value?.search();
};
</script>

<style lang="scss" scoped>
/* Markdown 文本（理由/执行结果/校验结论）：限高滚动，排版交给 MarkdownView */
.agentops-proposal-md {
    display: block;
    max-height: 320px;
    overflow: auto;
}

.agentops-proposal-pre {
    margin: 0;
    max-height: 240px;
    overflow: auto;
    white-space: pre-wrap;
    word-break: break-word;
    font-family: var(--el-font-family-mono, monospace);
    font-size: 12px;
    line-height: 1.6;
}

.agentops-proposal-pre--err {
    color: var(--el-color-danger);
}
</style>
