<template>
    <div>
        <!-- 提案快照取自工单 bizForm（发起工单时锁定，之后提案再怎么流转都不影响审批人看到的判断依据）。
             与 dbms/redis 面板不同，这里无需回查资源实例：锁定参数原文就是审批要审的全部内容 -->
        <el-alert class="mb-2" type="info" :closable="false" show-icon :title="$t('flow.approvalNotice')" />

        <el-descriptions :column="2" border label-width="110">
            <el-descriptions-item :span="2" :label="$t('flow.proposalTitle')">{{ form.title }}</el-descriptions-item>
            <el-descriptions-item :label="$t('flow.proposalAction')">
                <el-tag effect="plain">{{ form.action }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item :label="$t('flow.proposalRisk')">
                <enum-tag :enums="ProposalRiskEnum" :value="form.risk" />
            </el-descriptions-item>
            <el-descriptions-item :label="$t('flow.proposalOwner')">{{ form.ownerUsername }}</el-descriptions-item>
            <el-descriptions-item v-if="sourceRunId" :label="$t('flow.procFromRun')">
                <!-- 来源运行给成可点开：审批人要判断「这个变更凭什么被提出来」，得看到那次运行的轨迹。
                     只放一个裸 id 等于让审批人回运维列表按 id 碰运气（而运行列表也没把该 id 当筛选开放） -->
                <el-link type="primary" underline="never" @click="openRunDetail(sourceRunId)">{{ sourceRunId }}</el-link>
            </el-descriptions-item>
            <el-descriptions-item :span="2" :label="$t('flow.proposalParams')">
                <pre class="flow-proposal-pre">{{ paramsText }}</pre>
            </el-descriptions-item>
            <el-descriptions-item v-if="form.reason" :span="2" :label="$t('flow.proposalReason')">
                <MarkdownView class="flow-proposal-md" :content="form.reason" />
            </el-descriptions-item>
        </el-descriptions>

        <!-- 处理结果：受控执行是异步的，回调当时只能记「已受理」回执；终态完成后后端会把
             真实执行/校验结果回写到本列（ticketHandleResult 形态），旧回执则降级为等待提示 -->
        <template v-if="finalRes">
            <el-divider content-position="left">{{ $t('flow.handleResult') }}</el-divider>
            <el-descriptions :column="2" border label-width="110">
                <el-descriptions-item :label="$t('flow.handleResultStatus')">
                    <enum-tag :enums="ProposalStatusEnum" :value="finalRes.status" />
                </el-descriptions-item>
                <el-descriptions-item v-if="execRunId || verifyRunId" :label="$t('flow.handleResRuns')">
                    <!-- 执行/校验各给一条独立链接：写进一句文案就点不开；只跑到一个阶段时（失败终止的
                         工单只有 execRunId）也不该再拼出一个「校验 #-」的空位 -->
                    <el-link v-if="execRunId" type="primary" underline="never" @click="openRunDetail(execRunId)">
                        {{ $t('flow.handleResRunExec', { id: execRunId }) }}
                    </el-link>
                    <el-link v-if="verifyRunId" type="primary" underline="never" class="ml-2" @click="openRunDetail(verifyRunId)">
                        {{ $t('flow.handleResRunVerify', { id: verifyRunId }) }}
                    </el-link>
                </el-descriptions-item>
                <el-descriptions-item v-if="finalRes.err" :span="2" :label="$t('flow.handleResErr')">
                    <pre class="flow-proposal-pre flow-proposal-pre--err">{{ finalRes.err }}</pre>
                </el-descriptions-item>
                <el-descriptions-item v-if="finalRes.execResult" :span="2" :label="$t('flow.handleResExecResult')">
                    <MarkdownView class="flow-proposal-md" :content="finalRes.execResult" />
                </el-descriptions-item>
                <el-descriptions-item v-if="finalRes.verifyConclusion" :span="2" :label="$t('flow.handleResVerifyConclusion')">
                    <MarkdownView class="flow-proposal-md" :content="finalRes.verifyConclusion" />
                </el-descriptions-item>
            </el-descriptions>
        </template>
        <div v-else-if="handleResText">
            <el-divider content-position="left">{{ $t('flow.handleResult') }}</el-divider>
            <!-- 异步执行尚未收尾：明确告诉审批人在等什么，而不是把 queued 回执当结果直出 -->
            <el-alert v-if="isQueuedAck" type="info" :closable="false" show-icon :title="$t('flow.handleQueuedTip')" />
            <pre v-else class="flow-proposal-pre">{{ handleResText }}</pre>
        </div>

        <!-- 关联运行（来源/执行/校验）的详情抽屉：与提案页、执行历史共用同一个只读容器。
             必须挂在面板根层级而不是「处理结果」块里：待审批工单还没有终态回写（finalRes 为空），
             放进那个 v-if 里就等于「审批最需要看来源轨迹的时候，点链接什么也不会发生」 -->
        <RunDetailDrawer v-model:visible="runDetail.visible" :run-id="runDetail.runId" initial-tab="transcript" />
    </div>
</template>

<script lang="ts" setup>
import { computed, defineAsyncComponent, reactive } from 'vue';
import { ProposalRiskEnum, ProposalStatusEnum } from '@/views/agentops/enums';
import { ACK_EXECUTIONS, ACK_KEY_EXECUTION, ACK_KEY_MESSAGE, ACK_MESSAGES } from './proposalAck';

const EnumTag = defineAsyncComponent(() => import('@/components/enum-tag/EnumTag.vue'));
const MarkdownView = defineAsyncComponent(() => import('@/components/markdown').then((m) => m.MarkdownView));
// 运行详情抽屉体量大（含轨迹回放与追问草稿），按需加载，不拖慢工单详情首屏
const RunDetailDrawer = defineAsyncComponent(() => import('@/views/agentops/RunDetailDrawer.vue'));

/** 与后端 agentops.AgentRemediationBizForm 同构的提案快照 */
interface RemediationBizForm {
    proposalId?: number;
    title?: string;
    action?: string;
    params?: string;
    risk?: string;
    reason?: string;
    runId?: number;
    conversationId?: number;
    ownerUsername?: string;
}

/** 与后端 agentops.ticketHandleResult 同构的终态回写结果 */
interface TicketHandleResult {
    proposalId?: number;
    status?: string;
    execRunId?: number;
    execResult?: string;
    verifyRunId?: number;
    verifyConclusion?: string;
    err?: string;
}

const props = defineProps({
    procinst: {
        type: [Object],
        default: () => {},
    },
});

const EMPTY_FORM: RemediationBizForm = {};

/**
 * bizForm 理论上必为合法 JSON（后端 jsonx.ToStr 写入），但坏一串就打断整个详情抽屉不值得，
 * 解析失败按空表单降级（面板只剩须知，读不出内容胜过白屏）
 */
const form = computed<RemediationBizForm>(() => {
    try {
        return props.procinst.bizForm ? JSON.parse(props.procinst.bizForm) : EMPTY_FORM;
    } catch {
        return EMPTY_FORM;
    }
});

/** 处理结果按原文缩进展示（非终态回写形态的兜底）；非 JSON（历史脏数据）时原样透出 */
const handleResText = computed(() => {
    const raw = props.procinst.bizHandleRes;
    if (!raw) {
        return '';
    }
    try {
        return JSON.stringify(JSON.parse(raw), null, 2);
    } catch {
        return raw;
    }
});

/** 解析后的处理结果对象（非 JSON / 标量时为 null），终态判定与回执判定共用这一次解析 */
const handleResObj = computed<Record<string, unknown> | null>(() => {
    const raw = props.procinst.bizHandleRes;
    if (!raw) {
        return null;
    }
    try {
        const parsed = JSON.parse(raw);
        return parsed && typeof parsed === 'object' ? parsed : null;
    } catch {
        return null;
    }
});

// 提案终态才是「真实结果」；状态值取 ProposalStatusEnum 登记项而不是手抄字面量（同仓 enums.ts 自立的规则）
const finalRes = computed<TicketHandleResult | null>(() => {
    const obj = handleResObj.value as TicketHandleResult | null;
    return obj && (obj.status === ProposalStatusEnum.Executed.value || obj.status === ProposalStatusEnum.Failed.value) ? obj : null;
});

// 三个关联运行 id 派生为独立 computed：模板里拿 finalRes.execRunId 做 v-if 后，子节点里再取一次
// 仍是 `number | undefined`（Vue 模板不对属性链做跳节点收窄），要么写 ! 要么写 `|| 0`，
// 两种都是在可空性上含糊；收拢成「取不到就是 0」后，模板只需判真值
const sourceRunId = computed(() => form.value.runId ?? 0);
const execRunId = computed(() => finalRes.value?.execRunId ?? 0);
const verifyRunId = computed(() => finalRes.value?.verifyRunId ?? 0);

const runDetail = reactive({ visible: false, runId: null as number | null });

/** 打开某次关联运行的只读详情（来源/执行/校验共用一个入口）：链接只在 id 非零时渲染，无需再防 0 */
const openRunDetail = (runId: number) => {
    runDetail.runId = runId;
    runDetail.visible = true;
};

// 受理回执的全部形态（与后端 FlowBizHandle 的非终态返回同源，字面值集中在 ./proposalAck.ts）：
// queued/requeued 是执行已排队/重投，already_processed/already_claimed 是幂等落败——它们都没有
// 结论内容，一律降级为等待提示而非裸直出 JSON
const isQueuedAck = computed(() => {
    const obj = handleResObj.value;
    return !!obj && (ACK_EXECUTIONS.includes(String(obj[ACK_KEY_EXECUTION])) || ACK_MESSAGES.includes(String(obj[ACK_KEY_MESSAGE])));
});

/**
 * 锁定参数缩进展示 + 转义还原。
 *
 * 模型提交的 params 原文里常带字面 `\u0026`（JSON 字符串转义被模型写成了转义文本），执行时工具
 * 解 JSON 会还原成 `&`，但审批人直接读原文会把命令读花——而锁定参数正是审批要判断的全部内容。
 * 这里只改展示不改存储：执行收到的参数一个字节没动；非 JSON / 解码失败时原样透出，不编造内容
 */
const paramsText = computed(() => {
    const raw = form.value.params;
    if (!raw) {
        return '';
    }
    let text = raw;
    try {
        text = JSON.stringify(JSON.parse(raw), null, 2);
    } catch {
        return raw;
    }
    return text.replace(/\\u0026/g, '&').replace(/\\u003c/g, '<').replace(/\\u003e/g, '>');
});
</script>

<style lang="scss" scoped>
.flow-proposal-pre {
    margin: 0;
    max-height: 240px;
    overflow: auto;
    white-space: pre-wrap;
    word-break: break-word;
    font-family: var(--el-font-family-mono, monospace);
    font-size: 12px;
    line-height: 1.6;
}

.flow-proposal-pre--err {
    color: var(--el-color-danger);
}

.flow-proposal-md {
    display: block;
    max-height: 320px;
    overflow: auto;
}
</style>
