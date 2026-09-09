<template>
    <div class="message-list-wrapper">
        <!-- ScrollerProvider 由 ChatPanel 提供（包裹 MessageList + ChatInput，
             ChatPanel 在发送后调用这里暴露的 scrollToEnd，onScrollReady 链路） -->
        <MessageScroller class="flex-1">
            <MessageScrollerViewport class="min-h-0 w-full max-w-4xl mx-auto" @scroll="onViewportScroll">
                <!-- 滚动容器与消息列同宽（max-w-4xl，同基准 56rem）：
                     滚动条贴消息列右缘，大屏下两侧自然留白 -->
                <MessageScrollerContent class="w-full gap-2 px-6 py-4">
                    <!-- 顶部加载更多指示器（data-scroll-ignore 避免引擎
                         把它当作 prepend 消息项） -->
                    <div
                        v-if="loadingMore"
                        data-scroll-ignore
                        class="flex justify-center py-2"
                    >
                        <Spinner class="size-4 text-muted-foreground" />
                    </div>

                    <!-- 空状态（仅非加载时渲染，避免污染引擎 children 计数）。
                         建议芯片承担教学职责：点击回填输入框，不直接发送 -->
                    <div v-if="messages.length === 0 && !loading" class="message-list__empty">
                        <Empty class="py-10">
                            <EmptyMedia variant="icon">
                                <MessageCircleMoreIcon />
                            </EmptyMedia>
                            <EmptyContent>
                                <EmptyDescription>{{ t('ai.assistant.startNewConversation') }}</EmptyDescription>
                            </EmptyContent>
                        </Empty>
                        <div class="message-list__suggestions">
                            <button
                                v-for="key in suggestionKeys"
                                :key="key"
                                type="button"
                                class="message-list__suggestion"
                                @click="$emit('suggest', t(key))"
                            >
                                {{ t(key) }}
                            </button>
                        </div>
                    </div>

                    <!-- 消息列表（不设 scroll-anchor：发送后由引擎
                         following-bottom 自动贴底；带 anchor 会被引擎锚定到视口顶部） -->
                    <MessageScrollerItem
                        v-for="msg in messages"
                        :key="msg.id"
                        :message-id="msg.id"
                    >
                        <MessageBubble
                            :id="msg.id"
                            :role="msg.role === 'user' ? 'user' : 'assistant'"
                            :content="msg.content"
                            :parts="msg.parts"
                            :segments="msg.segments"
                            :attachments="msg.attachments"
                            :time="msg.time"
                            :turn-id="msg.turnId"
                            :pending-interrupts="pendingInterrupts"
                            :streaming="msg.streaming"
                            :is-editing="editingMessageId === msg.id"
                            @interrupt-action="(action) => $emit('interrupt-action', action)"
                            @edit="$emit('edit-message', msg.id)"
                            @edit-send="(c) => $emit('edit-send', c)"
                            @cancel-edit="$emit('cancel-edit')"
                        />
                    </MessageScrollerItem>

                    <!-- 等待回复指示（loading item：作为带 messageId 的 Item 渲染） -->
                    <MessageScrollerItem v-if="pendingReply" message-id="__loading" class="flex justify-center py-2">
                        <Spinner class="size-5 text-muted-foreground" />
                    </MessageScrollerItem>
                </MessageScrollerContent>
            </MessageScrollerViewport>

            <!-- 滚动到底部按钮（rounded-full 圆钮 + 显式图标） -->
            <MessageScrollerButton direction="end" class="rounded-full">
                <ArrowDownIcon />
                <span class="sr-only">{{ t('ai.chat.scrollToEnd') }}</span>
            </MessageScrollerButton>
        </MessageScroller>
    </div>
</template>

<script setup lang="ts">
/**
 * MessageList - 消息列表组件（shadcn MessageScroller 驱动）
 * MessageList：MessageScrollerProvider 内渲染，自研滚动逻辑全部移除
 *
 * - 自动跟随 / 轮次锚定 / 上翻不拉回：由 MessageScroller 引擎处理（autoScroll + last-anchor）
 * - 历史分页（懒加载）：scrollTop < 20 触发 + 加载完成后 300ms 续检
 *   （useInfiniteScrollTrigger），位置恢复由 Viewport 的
 *   preserveScrollOnPrepend 保证，不跳动
 * - 滚动到底部按钮：MessageScrollerButton（方向 end，自动显隐）
 * - scrollToEnd 经 defineExpose 暴露给 ChatPanel，发送后贴底（ChatInput scrollToEnd）
 */
import { ArrowDownIcon, MessageCircleMoreIcon } from '@lucide/vue';
import { computed, onBeforeUnmount, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { Spinner } from '@/components/ui/spinner';
import {
    Empty,
    EmptyContent,
    EmptyDescription,
    EmptyMedia,
} from '@/components/ui/empty';
import {
    MessageScroller,
    MessageScrollerButton,
    MessageScrollerContent,
    MessageScrollerItem,
    MessageScrollerViewport,
    useMessageScroller,
} from '@/components/ui/message-scroller';
import type { ChatMessage, InterruptEvent } from '../protocol/types';
import type { InterruptActionEvent } from '../interrupt/types';
import MessageBubble from './MessageBubble.vue';

const props = defineProps<{
    messages: ChatMessage[];
    loading: boolean;
    /** 发送方忙（等待回复，首条 assistant item 到达前显示 Spinner） */
    senderLoading?: boolean;
    hasMore: boolean;
    /** 正在加载更早的历史消息（顶部 Spinner + 触发守卫） */
    loadingMore?: boolean;
    pendingInterrupts?: InterruptEvent[];
    /** 原地编辑中的消息 ID（仅用户消息，editingMessageId） */
    editingMessageId?: string | null;
    /** 会话 ID：作为滚动锚点恢复的触发 key（变化时恢复阅读位置） */
    conversationId?: number | null;
    /** 恢复目标：离开时视口顶部可见消息 ID + 视口内偏移（ScrollAnchor） */
    scrollAnchor?: { id: string; offset: number } | null;
}>();

const emit = defineEmits<{
    (e: 'load-more'): void;
    (e: 'suggest', text: string): void;
    (e: 'interrupt-action', action: InterruptActionEvent): void;
    (e: 'scroll-anchor-change', anchor: { id: string; offset: number }): void;
    (e: 'edit-message', messageId: string): void;
    (e: 'edit-send', content: string): void;
    (e: 'cancel-edit'): void;
}>();

const { t } = useI18n();
const { scrollToEnd, scrollToMessage } = useMessageScroller();

/** 空态建议（i18n key 列表，点击回填输入框） */
const suggestionKeys = ['ai.chat.suggestions.logCheck', 'ai.chat.suggestions.slowSql', 'ai.chat.suggestions.script'];

/** 暴露给 ChatPanel：发送后贴底（MessageList onScrollReady） */
defineExpose({
    scrollToEnd: (options?: { behavior?: ScrollBehavior }) => scrollToEnd(options),
});

/** 等待回复：发送后、assistant 流式消息创建前（最后一条仍是用户消息） */
const pendingReply = computed(() => {
    if (!props.senderLoading || props.messages.length === 0) return false;
    return props.messages[props.messages.length - 1].role === 'user';
});

/** 顶部翻页 + 锚点保存（useInfiniteScrollTrigger + ScrollPersistenceArea）。
 *  翻页防重复：store.isLoadingMoreMessages 守卫 + loadingMore prop */
let viewportEl: HTMLElement | null = null;
let anchorSaveRafId = 0;
const onViewportScroll = (e: Event) => {
    const el = e.target as HTMLElement;
    viewportEl = el;
    if (el.scrollTop < 20 && props.hasMore && !props.loading && !props.loadingMore) {
        emit('load-more');
    }
    // 锚点保存（ScrollPersistenceArea：RAF 节流，恢复期抑制）
    if (restoring || props.conversationId == null) return;
    // 捕获当时会话：切会话后 pending RAF 不得把旧视口位置错存到新会话
    const convIdAtScroll = props.conversationId;
    cancelAnimationFrame(anchorSaveRafId);
    anchorSaveRafId = requestAnimationFrame(() => {
        if (restoring || props.conversationId !== convIdAtScroll) return;
        const anchor = findTopVisibleAnchor(el);
        if (anchor) emit('scroll-anchor-change', anchor);
    });
};

onBeforeUnmount(() => cancelAnimationFrame(anchorSaveRafId));

/** 加载完成后续检：preserveScrollOnPrepend 保持 scrollTop
 *  不变导致 onScroll 不再触发，故延迟 300ms 检查是否仍在顶部，是则继续加载下一批。
 *  以 isLoadingMore 状态驱动（而非 scroll 事件），保证分批节奏 */
let continueCheckTimer: ReturnType<typeof setTimeout> | null = null;
watch(
    () => props.loadingMore,
    (isLoadingMore) => {
        if (continueCheckTimer) {
            clearTimeout(continueCheckTimer);
            continueCheckTimer = null;
        }
        if (isLoadingMore) return;
        const viewport = viewportEl;
        if (!viewport) return;
        continueCheckTimer = setTimeout(() => {
            continueCheckTimer = null;
            if (viewport.isConnected && viewport.scrollTop < 20 && props.hasMore && !props.loading) {
                emit('load-more');
            }
        }, 300);
    },
);

/** 视口顶部第一条可见消息 + 视口内偏移（findTopVisibleAnchor）。
 *  不用 scrollTop：消息项动态高度下数值不稳定，锚定到消息 ID + 偏移 */
function findTopVisibleAnchor(viewport: HTMLElement): { id: string; offset: number } | null {
    const content = viewport.querySelector<HTMLElement>('[data-slot="message-scroller-content"]');
    if (!content) return null;
    const viewportTop = viewport.getBoundingClientRect().top;
    // 与恢复侧 scrollMargin 计算一致：扣除 content 顶部 padding，否则残差恒等于 padding 值
    const pad = Number.parseFloat(window.getComputedStyle(content).paddingBlockStart || '0') || 0;
    for (const el of Array.from(content.children)) {
        if (!(el instanceof HTMLElement)) continue;
        const id = el.dataset.messageId;
        if (!id) continue;
        const rect = el.getBoundingClientRect();
        if (rect.bottom > viewportTop) {
            // offset 可为负：视口顶停在长消息中部时表示消息内深度，恢复时反向补偿
            return { id, offset: Math.round(rect.top - viewportTop - pad) };
        }
    }
    return null;
}

// ==================== 滚动锚点恢复（ScrollRestoreAnchor） ====================

/** 恢复期间抑制锚点保存：defaultScroll 按底的 scroll 不得覆盖恢复目标 */
let restoring = false;

watch(
    () => props.conversationId,
    () => {
        restoring = true;
        // 双帧延迟：等引擎 defaultScroll（贴底）先落位，再锚定覆盖
        requestAnimationFrame(() => requestAnimationFrame(() => tryRestore()));
    },
    { immediate: true },
);

/** 消息可能异步到达（分页/重载），元素未注册时按帧重试（MAX_RETRY_FRAMES） */
const MAX_RETRY_FRAMES = 90;

function tryRestore(frame = 0) {
    const anchor = props.scrollAnchor;
    const viewport = viewportEl;
    if (!anchor || !viewport || props.messages.length === 0) {
        // 无锚点：消息就绪后贴底；有锚点但消息未就绪：按帧等待，重试耗尽兑底贴底
        if (props.messages.length > 0 || frame >= MAX_RETRY_FRAMES) {
            scrollToEnd();
            restoring = false;
        } else {
            requestAnimationFrame(() => tryRestore(frame + 1));
        }
        return;
    }
    // align:'start' + scrollMargin=保存的视口内偏移一步完成「锚定消息 + 消息内深度」定位
    const ok = scrollToMessage(anchor.id, { align: 'start', scrollMargin: anchor.offset, behavior: 'auto' });
    if (ok) {
        requestAnimationFrame(() => {
            restoring = false;
        });
    } else if (frame < MAX_RETRY_FRAMES) {
        requestAnimationFrame(() => tryRestore(frame + 1));
    } else {
        scrollToEnd();
        restoring = false;
    }
}
</script>

<style scoped>
.message-list-wrapper {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

/* 空态建议：居中列，芯片为描边淡底按钮（避免与用户消息气泡撞形） */
.message-list__empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding-top: 24px;
}

.message-list__suggestions {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 8px;
    max-width: 32rem;
}

.message-list__suggestion {
    padding: 6px 14px;
    font-size: 12px;
    line-height: 1.4;
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 9999px;
    cursor: pointer;
    transition: background-color 0.15s ease-out, border-color 0.15s ease-out, color 0.15s ease-out;
}

.message-list__suggestion:hover {
    background: var(--el-fill-color);
    border-color: var(--el-border-color-light);
    color: var(--el-text-color-primary);
}

.message-list__suggestion:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 1px;
}
</style>
