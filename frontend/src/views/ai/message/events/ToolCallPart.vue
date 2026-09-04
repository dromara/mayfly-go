<template>
    <div
        class="tool-call-part"
        :class="{ 'tool-call-part--interrupted': !!pendingInterrupt, 'tool-call-part--running': isRunning }"
    >
        <CollapsibleSection :title="toolName" :force-open="!!pendingInterrupt">
            <template #icon>
                <WrenchIcon
                    class="tool-call-part__icon"
                    :class="{ 'is-running': isRunning, 'is-interrupted': !!pendingInterrupt }"
                />
            </template>
            <template #extra>
                <!-- 终态状态圆点（对齐 tokhub StatusDot：执行中/中断由图标脉冲表达） -->
                <span v-if="statusDotClass" class="tool-call-part__dot" :class="statusDotClass" />
                <!-- 待决策中断：warning 徽章；已决策中断：决议色小徽章（对齐 ResumeStatusBadge） -->
                <span v-if="pendingInterrupt" class="tool-call-part__badge bg-warning/10 text-warning border-warning/20">
                    {{ pendingBadgeLabel }}
                </span>
                <span
                    v-else-if="decidedBadge"
                    class="tool-call-part__badge"
                    :class="decidedBadge.class"
                >
                    {{ decidedBadge.label }}
                </span>
                <span v-if="durationText" class="tool-call-part__duration">{{ durationText }}</span>
            </template>

            <!-- 展开详情：备注 + 参数 + 输出 -->
            <div v-if="remarkText" class="tool-call-part__section">
                <div class="tool-call-part__label">{{ t('ai.chat.remark') }}</div>
                <div class="tool-call-part__remark">{{ remarkText }}</div>
            </div>
            <div v-if="arguments" class="tool-call-part__section">
                <div class="tool-call-part__label">{{ t('ai.chat.arguments') }}</div>
                <div class="tool-call-part__panel">
                    <pre class="tool-call-part__pre">{{ arguments }}</pre>
                </div>
            </div>
            <div v-if="output" class="tool-call-part__section">
                <div class="tool-call-part__label">{{ t('ai.chat.result') }}</div>
                <div class="tool-call-part__panel">
                    <pre class="tool-call-part__pre">{{ output }}</pre>
                </div>
            </div>
        </CollapsibleSection>

        <!-- 内联中断组件：始终可见（对齐 tokhub ToolCallEntry 内联中断，组件自带 Card 外壳） -->
        <div v-if="pendingInterrupt" class="tool-call-part__interrupt">
            <component
                :is="getInterruptComponent(pendingInterrupt.type)"
                :interrupt="pendingInterrupt"
                :turn-id="turnId || pendingInterrupt.turnId || ''"
                :readonly="false"
                :on-action="handleInterruptAction"
            />
        </div>
    </div>
</template>

<script setup lang="ts">
/**
 * ToolCallPart - 工具调用展示组件
 * 对齐 tokhub ToolCallEntry：固定 Wrench 图标 + 执行中脉冲 + 终态语义色圆点 +
 * 决议/耗时 meta，CollapsibleSection 折叠详情；状态视觉不走大徽章
 */
import { WrenchIcon } from '@lucide/vue';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { getInterruptComponent } from '../../interrupt';
import { getInterruptHandler } from '../../registries/interruptRegistry';
import type { InterruptEvent } from '../../protocol/types';
import type { InterruptActionEvent, InterruptActionHandler } from '../../interrupt/types';
import { InterruptResumeStatus, ToolCallStatus } from '../../registries/statuses';
import { isInterruptDecided } from '../../registries/statuses';
import CollapsibleSection from '../CollapsibleSection.vue';

const props = defineProps<{
    toolName: string;
    arguments?: string;
    output?: string;
    status: string;
    durationMs?: number;
    toolCallId?: string;
    turnId?: string;
    /** 恢复决策类型（对齐 tokhub extra.interrupt.resume.type，历史回放的权威数据源） */
    resumeType?: string;
    pendingInterrupts?: InterruptEvent[];
}>();

const emit = defineEmits<{
    (e: 'interrupt-action', action: InterruptActionEvent): void;
}>();

const { t } = useI18n();

/** 解析备注（MachineCommandExec/ExecSql 的 remark 入参，帮助用户理解执行意图；
 *  流式期间 arguments 可能残缺，解析失败静默降级） */
const remarkText = computed(() => {
    if (!props.arguments) return '';
    try {
        const remark = JSON.parse(props.arguments)?.remark;
        return typeof remark === 'string' ? remark.trim() : '';
    } catch {
        return '';
    }
});

/** 匹配当前工具调用的待处理中断（对齐 tokhub ToolCallEntry.interrupt）
 *  已决策的中断不再渲染待审批表单（决策状态已双写到 part） */
const pendingInterrupt = computed(() => {
    if (!props.pendingInterrupts?.length || !props.toolCallId) return undefined;
    const found = props.pendingInterrupts.find((i) => i.toolCallId === props.toolCallId);
    return found && !isInterruptDecided(found.status) ? found : undefined;
});

/** 中断操作回调（向上冒泡） */
const handleInterruptAction: InterruptActionHandler = (action) => {
    emit('interrupt-action', action);
};

/** 待决策徽章文案：委托类型 handler 覆盖（对齐 tokhub getPendingLabel），回退通用文案 */
const pendingBadgeLabel = computed(() => {
    const handler = pendingInterrupt.value ? getInterruptHandler(pendingInterrupt.value.type) : undefined;
    return handler?.getPendingLabel?.(t) ?? t('common.waitingInput');
});

// ==================== 状态 meta（对齐 tokhub STATUS_DOT_COLORS / ResumeStatusBadge） ====================

/** 执行中：pending/running 均视为执行中（图标脉冲表达，不显示圆点） */
const isRunning = computed(
    () => props.status === ToolCallStatus.Pending || props.status === ToolCallStatus.Running,
);

/** 终态状态圆点色（pending/running/interrupted 由脉冲图标表达，不显示） */
const DOT_CLASSES: Record<string, string> = {
    [ToolCallStatus.Success]: 'bg-success',
    [ToolCallStatus.Failed]: 'bg-destructive',
    [ToolCallStatus.Cancelled]: 'bg-muted-foreground/40',
};
const statusDotClass = computed(() => DOT_CLASSES[props.status]);

/** 通用决议徽章配置（对齐 tokhub RESUME_STATUS_CONFIG，i18n key + 语义色淡底） */
const RESUME_STATUS_CONFIG: Record<string, { class: string; labelKey: string }> = {
    [InterruptResumeStatus.Approved]: {
        class: 'bg-success/10 text-success border-success/20',
        labelKey: 'ai.chat.resumeStatus.approved',
    },
    [InterruptResumeStatus.Resolved]: {
        class: 'bg-success/10 text-success border-success/20',
        labelKey: 'ai.chat.resumeStatus.approved',
    },
    [InterruptResumeStatus.ParamsCompleted]: {
        class: 'bg-success/10 text-success border-success/20',
        labelKey: 'ai.chat.resumeStatus.paramsCompleted',
    },
    [InterruptResumeStatus.Answered]: {
        class: 'bg-success/10 text-success border-success/20',
        labelKey: 'ai.chat.resumeStatus.answered',
    },
    [InterruptResumeStatus.Rejected]: {
        class: 'bg-destructive/10 text-destructive border-destructive/20',
        labelKey: 'ai.chat.resumeStatus.rejected',
    },
    [InterruptResumeStatus.Skipped]: {
        class: 'bg-muted text-muted-foreground border-border',
        labelKey: 'ai.chat.resumeStatus.skipped',
    },
};

/** 解析决议徽章（对齐 tokhub：handler 覆盖 → 通用配置 → 未知状态归入 cancelled 灰色） */
function resumeBadgeOf(status: string, interruptType?: string) {
    const handler = interruptType ? getInterruptHandler(interruptType) : undefined;
    const override = handler?.getResumeBadge?.(status, t);
    if (override) return { class: override.badgeClass, label: override.label };
    const config = RESUME_STATUS_CONFIG[status];
    if (config) return { class: config.class, label: t(config.labelKey) };
    return {
        class: 'bg-muted text-muted-foreground border-border',
        label: t('ai.chat.resumeStatus.cancelled'),
    };
}

/** 已决策中断的决议小徽章：历史回放以 part.resumeType（extra.interrupt.resume.type）
 *  为权威数据源；实时路径从 pendingInterrupts 已决策项派生 */
const decidedBadge = computed(() => {
    if (props.resumeType) {
        return resumeBadgeOf(props.resumeType);
    }
    if (!props.pendingInterrupts?.length || !props.toolCallId) return undefined;
    const decided = props.pendingInterrupts.find(
        (i) => i.toolCallId === props.toolCallId && isInterruptDecided(i.status),
    );
    if (!decided) return undefined;
    return resumeBadgeOf(decided.status ?? '', decided.type);
});

/** 耗时展示：<1s 显示 ms，否则 s（对齐 tokhub durationMs 格式） */
const durationText = computed(() => {
    if (!props.durationMs) return '';
    return props.durationMs < 1000 ? `${props.durationMs}ms` : `${(props.durationMs / 1000).toFixed(1)}s`;
});
</script>

<style scoped>
.tool-call-part {
    border-radius: 6px;
    overflow: hidden;
    font-size: 12px;
}

.tool-call-part--interrupted {
    border: 1px solid color-mix(in srgb, var(--el-color-warning) 35%, transparent);
    border-radius: 6px;
}

/* 固定 Wrench 图标：终态半透明灰，hover 渐亮；执行中/中断脉冲 */
.tool-call-part__icon {
    width: 12px;
    height: 12px;
    flex-shrink: 0;
    opacity: 0.6;
    transition: opacity 0.15s ease-out;
}

.collapsible-section__trigger:hover .tool-call-part__icon {
    opacity: 0.85;
}

@keyframes tool-call-pulse {
    0%,
    100% {
        opacity: 0.8;
    }
    50% {
        opacity: 0.4;
    }
}

.tool-call-part__icon.is-running {
    animation: tool-call-pulse 1.6s ease-in-out infinite;
}

.tool-call-part__icon.is-interrupted {
    color: var(--el-color-warning);
    animation: tool-call-pulse 1.6s ease-in-out infinite;
}

/* 执行中标题提亮一档（对齐 tokhub isRunning 分层） */
.tool-call-part--running :deep(.collapsible-section__title) {
    color: var(--el-text-color-regular);
}

/* 终态状态圆点（6px，语义色由 tailwind 类提供） */
.tool-call-part__dot {
    width: 6px;
    height: 6px;
    flex-shrink: 0;
    border-radius: 9999px;
}

/* meta 徽章：10px 语义色淡底描边（底/字/框色由 tailwind 类提供） */
.tool-call-part__badge {
    flex-shrink: 0;
    padding: 2px 6px;
    border: 1px solid;
    border-radius: 4px;
    font-size: 10px;
    font-weight: 500;
    line-height: 1;
}

.tool-call-part__duration {
    flex-shrink: 0;
    font-size: 10px;
    color: var(--el-text-color-placeholder);
    font-variant-numeric: tabular-nums;
}

.tool-call-part__interrupt {
    padding: 4px 6px 6px;
}

.tool-call-part__section {
    display: flex;
    flex-direction: column;
    gap: 2px;
}

.tool-call-part__label {
    font-size: 10px;
    font-weight: 500;
    color: var(--el-text-color-placeholder);
}

/* 执行目的：淡底高亮行（非等宽字体，与代码参数面板区分） */
.tool-call-part__remark {
    padding: 5px 10px;
    border: 1px solid color-mix(in srgb, var(--el-color-primary) 20%, transparent);
    border-radius: 8px;
    background: color-mix(in srgb, var(--el-color-primary) 6%, transparent);
    font-size: 11px;
    line-height: 1.55;
    color: var(--el-text-color-regular);
    word-break: break-all;
}

/* 参数/结果面板：圆角描边淡底 + 面板内独立滚动（基础参考 tokhub max-h-24，
   放宽到 160px 并作为唯一滚动层：外层不再裁剪，确保长内容可完整滚动查看） */
.tool-call-part__panel {
    border: 1px solid color-mix(in srgb, var(--el-border-color) 40%, transparent);
    border-radius: 8px;
    background: color-mix(in srgb, var(--el-fill-color-lighter) 50%, transparent);
}

.tool-call-part__pre {
    margin: 0;
    padding: 5px 10px;
    max-height: 160px;
    overflow-y: auto;
    overflow-x: hidden;
    font-size: 11px;
    line-height: 1.55;
    white-space: pre-wrap;
    word-break: break-all;
    color: var(--el-text-color-regular);
    font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
}
</style>
