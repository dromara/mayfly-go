<template>
    <el-drawer v-model="visible" :append-to-body="true" :title="$t('agentops.runDetailTitle')" size="70%">
        <!-- 运行详情抽屉（RunList 页与任务执行历史弹窗共用）：按 id 拉取全量字段。
             两个页签：概览（列表页裁掉的大字段）/ 执行轨迹（只读回放本次运行的对话）。
             轨迹留在「发现它的面上」而不是跳聊天页：查看一次运维运行不该顺带获得改它的能力。
             append-to-body 必须开：本抽屉现在会从另一个抽屉内部被打开（工单业务信息里的关联运行链接），
             留在父抽屉 body 里会被父容器的 overflow/背板层级影响而裁切或遮住 -->
        <el-tabs v-if="run" v-model="activeTab" class="agentops-run-detail">
            <el-tab-pane :label="$t('agentops.runTabOverview')" name="overview">
                <el-descriptions :column="2" border v-loading="loading" label-width="120">
                    <el-descriptions-item :label="$t('agentops.runNo')">{{ run.runNo }}</el-descriptions-item>
                    <el-descriptions-item :label="$t('agentops.title')">{{ run.title }}</el-descriptions-item>
                    <el-descriptions-item :label="$t('agentops.runStatus')">{{ enumLabel(RunStatusEnum, run.status, t) }}</el-descriptions-item>
                    <el-descriptions-item :label="$t('agentops.triggerType')">{{ enumLabel(TriggerTypeEnum, run.triggerType, t) }}</el-descriptions-item>
                    <el-descriptions-item :label="$t('agentops.triggerRef')">{{ run.triggerRef }}</el-descriptions-item>
                    <el-descriptions-item :label="$t('agentops.profile')">{{ enumLabel(ProfileEnum, run.profileId, t) }}</el-descriptions-item>
                    <el-descriptions-item v-if="run.dryRun" :label="$t('agentops.runMode')">
                        <el-tag type="warning" effect="plain">{{ $t('agentops.dryRun') }}</el-tag>
                    </el-descriptions-item>
                    <el-descriptions-item :label="$t('agentops.owner')">{{ run.ownerUsername }}</el-descriptions-item>
                    <el-descriptions-item :label="$t('agentops.totalTokens')">{{ formatTokenCount(run.totalTokens, true) }}</el-descriptions-item>
                    <el-descriptions-item :label="$t('agentops.startedAt')">{{ formatDate(run.startedAt || '') }}</el-descriptions-item>
                    <el-descriptions-item :label="$t('agentops.finishedAt')">{{ formatDate(run.finishedAt || '') }}</el-descriptions-item>
                    <el-descriptions-item :label="$t('agentops.costMs')" :span="2">{{ formatDurationMs(run.costMs) }}</el-descriptions-item>
                    <!-- 结论是模型产出的 Markdown（标题/表格/代码块），必须渲染后展示；
                         输入与任务指令是给模型的原文、错误是诊断文本，保持 <pre> 精确呈现 -->
                    <el-descriptions-item :label="$t('agentops.summary')" :span="2">
                        <MarkdownView class="agentops-run-md" :content="run.summary" />
                    </el-descriptions-item>
                    <el-descriptions-item v-if="run.input" :label="$t('agentops.input')" :span="2">
                        <pre class="agentops-run-pre">{{ run.input }}</pre>
                    </el-descriptions-item>
                    <el-descriptions-item v-if="run.instruction" :label="$t('agentops.instruction')" :span="2">
                        <pre class="agentops-run-pre">{{ run.instruction }}</pre>
                    </el-descriptions-item>
                    <el-descriptions-item v-if="run.err" :label="$t('agentops.err')" :span="2">
                        <pre class="agentops-run-pre agentops-run-pre--err">{{ run.err }}</pre>
                    </el-descriptions-item>
                </el-descriptions>
            </el-tab-pane>

            <!-- 无痕运行（conversationId=0）没有可回放的会话，不给该页签；
                 lazy 保证只在切到该页签时才拉消息 -->
            <el-tab-pane v-if="run.conversationId" :label="$t('agentops.runTabTranscript')" name="transcript" lazy>
                <div class="agentops-run-detail__transcript">
                    <RunTranscript :conversation-id="run.conversationId" />
                </div>
            </el-tab-pane>
        </el-tabs>

        <template #footer>
            <div v-if="run" class="agentops-run-detail__footer">
                <span class="agentops-run-detail__footer-tip">{{ $t('agentops.forkTip') }}</span>
                <el-button type="primary" plain @click="forkToChat">{{ $t('agentops.forkToChat') }}</el-button>
            </div>
        </template>
    </el-drawer>
</template>

<script setup lang="ts" name="RunDetailDrawer">
/**
 * 运行详情抽屉：概览 + 只读执行轨迹 + 「带上下文追问」出口。
 *
 * 分层：本组件只做「按 id 取详情 + 页签编排 + 追问投递」；轨迹的取数与渲染归
 * RunTranscript（它自己只管 conversationId），草稿正文的组装归 forkDraft（只管取值与截断），
 * 三者互不知道对方的接口细节。
 */
import { enumLabel } from '@/common/Enum';
import { formatDate, formatDurationMs, formatTokenCount } from '@/common/utils/format';
import { MarkdownView } from '@/components/markdown';
import { useChatStore } from '@/views/ai/stores/chatStore';
import RunTranscript from '@/views/ai/transcript/RunTranscript.vue';
import { ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import { agentRunApi } from './api';
import { buildRunForkParams } from './forkDraft';
import { ProfileEnum, RunStatusEnum, TriggerTypeEnum } from './enums';
import type { AgentRun } from './types';

/** 页签名（与 el-tab-pane 的 name 一一对应）；声明在 props 前供类型引用 */
const TAB_OVERVIEW = 'overview';
const TAB_TRANSCRIPT = 'transcript';

const props = defineProps<{
    /** 运行 id；为空时抽屉不渲染内容（首次打开前无选中行） */
    runId: number | null;
    /** 打开时落在哪个页签（行内「回放」直达轨迹）；无会话的运行自动回落概览 */
    initialTab?: typeof TAB_OVERVIEW | typeof TAB_TRANSCRIPT;
}>();

const visible = defineModel<boolean>('visible', { required: true });

const { t } = useI18n();
const router = useRouter();
const chatStore = useChatStore();

const run = ref<AgentRun | null>(null);
const loading = ref(false);
const activeTab = ref(TAB_OVERVIEW);

// 每次打开按 id 拉全量字段（列表 VO 裁剪了大字段，行数据不足以支撑详情）
watch(
    () => [visible.value, props.runId] as const,
    async ([v, id]) => {
        if (!v || !id) {
            return;
        }
        // 先回概览再按 initialTab 定位：避免上一次打开停在轨迹页签残留
        activeTab.value = TAB_OVERVIEW;
        loading.value = true;
        try {
            run.value = await agentRunApi.detail.request({ id });
            if (props.initialTab === TAB_TRANSCRIPT && run.value.conversationId) {
                activeTab.value = TAB_TRANSCRIPT;
            }
        } finally {
            loading.value = false;
        }
    },
    { immediate: true }
);

// 枚举文案统一走 common/Enum 的 enumLabel（颜色走 enumTagType），不再各页写一份

/**
 * 带上下文追问：把本次运行整理成草稿，交给一个属于当前用户的**新会话**。
 *
 * 只做一次性投递 + 单次导航（不带 query 参数）：助手页在挂载/激活时自己取走草稿并建会话。
 * 刻意不用 `?compose=1`：那需要在跳转后再 replace 清参，同窗口两次路由变更会撞上布局层的
 * out-in 过渡，导致 router-view 永久空白（实测踩过）。
 *
 * 也不在原会话里继续：那次运行的会话是运维审计产物（服务端已拒绝对其发起新对话），
 * 且其运行身份属于运行 owner，不是正在浏览详情的这个人
 */
const forkToChat = () => {
    if (!run.value) return;
    const detail = run.value;
    chatStore.stageDraft(
        t('agentops.forkDraft', {
            ...buildRunForkParams(detail, {
                status: enumLabel(RunStatusEnum, detail.status, t),
                profile: enumLabel(ProfileEnum, detail.profileId, t),
                trigger: enumLabel(TriggerTypeEnum, detail.triggerType, t),
            }),
        })
    );
    visible.value = false;
    void router.push({ name: 'AiAssistant' });
};
</script>

<style lang="scss" scoped>
.agentops-run-detail {
    :deep(.el-tabs__content) {
        overflow: visible;
    }
}

.agentops-run-detail__transcript {
    display: flex;
    flex-direction: column;
    /* 抽屉为满高，减去头部/页签栏/底部操作条的固定预算，给轨迹一个确定高度的滚动视口
       （消息列表引擎必须有界高度，否则滚动与「上翻加载更早」都失效） */
    height: calc(100vh - 210px);
    min-height: 320px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    overflow: hidden;
}

.agentops-run-detail__footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 12px;
}

.agentops-run-detail__footer-tip {
    font-size: 12px;
    color: var(--el-text-color-secondary);
}

/* Markdown 结论容器：只负责限高与滚动，排版由 MarkdownView 提供 */
.agentops-run-md {
    display: block;
    max-height: 320px;
    overflow: auto;
}

.agentops-run-pre {
    margin: 0;
    max-height: 240px;
    overflow: auto;
    white-space: pre-wrap;
    word-break: break-word;
    font-family: var(--el-font-family-mono, monospace);
    font-size: 12px;
    line-height: 1.6;
}

.agentops-run-pre--err {
    color: var(--el-color-danger);
}
</style>
