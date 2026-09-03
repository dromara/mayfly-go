<template>
    <template v-if="fabVisible">
        <!-- 悬浮球：默认位于系统通知球上方错位摆放，可拖拽 + 位置记忆 -->
        <div
            class="ai-fab fixed z-[2000]"
            :style="{ bottom: position.bottom + 'px', right: position.right + 'px' }"
        >
            <button
                type="button"
                class="ai-fab__button"
                :class="{ 'ai-fab__button--dragging': isDragging }"
                :title="t('ai.assistant.entryTitle')"
                :aria-label="t('ai.assistant.entryTitle')"
                @mousedown="startDrag"
                @click="onFabClick"
            >
                <SparklesIcon class="size-5" />
                <!-- 会话进行中：语义色脉冲圆点（悬浮球唯一的状态色） -->
                <span v-if="hasRunningConv" class="ai-fab__dot" />
            </button>
        </div>

        <!-- 右侧抽屉：复用 AiAssistantBody（页面与抽屉单一实现） -->
        <el-drawer
            v-model="drawerVisible"
            size="min(720px, 94vw)"
            :with-header="false"
            append-to-body
            class="ai-assistant-drawer"
            @opened="onDrawerOpened"
        >
            <div class="ai-fab-drawer">
                <header class="ai-fab-drawer__header">
                    <h3 class="ai-fab-drawer__title">
                        <SparklesIcon class="size-4 text-primary" />
                        {{ t('ai.assistant.entryTitle') }}
                    </h3>
                    <div class="flex items-center gap-1">
                        <Button
                            size="icon-sm"
                            variant="ghost"
                            :title="t('ai.assistant.sessions')"
                            @click="sidebarVisible = !sidebarVisible"
                        >
                            <PanelLeftIcon class="size-4" />
                        </Button>
                        <Button size="icon-sm" variant="ghost" :title="t('common.close')" @click="drawerVisible = false">
                            <XIcon class="size-4" />
                        </Button>
                    </div>
                </header>
                <div class="ai-fab-drawer__body">
                    <!-- 重内容（TipTap 编辑器初始化等）延迟到滑入动画结束后挂载，
                         避免与入场动画争抢主线程导致首次滑入掉帧；挂载一次后保留，
                         后续打开直接复用，首尾观感一致 -->
                    <AiAssistantBody v-if="bodyRendered" :show-sidebar="sidebarVisible" />
                    <div v-else class="p-4">
                        <el-skeleton :rows="6" animated />
                    </div>
                </div>
            </div>
        </el-drawer>
    </template>
</template>

<script setup lang="ts" name="AiAssistantFab">
/**
 * AiAssistantFab - 全局 AI 助手入口（悬浮球 + 右侧抽屉）
 *
 * - 可见性：系统已配置 AI 模型（useAiEnabled）且非登录页才渲染
 * - 拖拽/位置记忆复用 useDraggableFab（与 GlobalNotificationFab 同源实现）
 * - 抽屉内容复用 AiAssistantBody（与 /ai 页面单一实现，chatStore 共享会话状态）；
 *   抽屉不销毁内容（el-drawer 默认懒渲染后保留），关闭不中断进行中的流式会话
 */
import { Button } from '@/components/ui/button';
import { useDraggableFab } from '@/hooks/useDraggableFab';
import { PanelLeftIcon, SparklesIcon, XIcon } from '@lucide/vue';
import { computed, ref } from 'vue';
import { useRoute } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { URL_LOGIN } from '@/router/staticRouter';
import { useAiEnabled } from './hooks/useAiEnabled';
import { useChatStore } from './stores/chatStore';
import AiAssistantBody from './components/AiAssistantBody.vue';

const { t } = useI18n();
const route = useRoute();
const chatStore = useChatStore();
const { aiEnabled } = useAiEnabled();

// 登录页不渲染（探测请求在未登录态也会失败，此处双保险避免闪烁）
const isLoginPage = computed(() => route.path === URL_LOGIN);
const fabVisible = computed(() => aiEnabled.value && !isLoginPage.value);

// 拖拽与位置持久化（与通知悬浮球同源 composable，独立 storage key）
const { position, isDragging, hasMoved, startDrag } = useDraggableFab({
    storageKey: 'ai-assistant-fab-position',
    defaultPosition: { bottom: 88, right: 20 },
});

const drawerVisible = ref(false);
const sidebarVisible = ref(true);

/** 重内容是否已挂载：首次滑入动画结束（@opened）后置 true，此后不再重置 */
const bodyRendered = ref(false);

const onDrawerOpened = () => {
    bodyRendered.value = true;
};

const onFabClick = () => {
    // 拖拽松手后的 click 不触发开关
    if (hasMoved.value) {
        hasMoved.value = false;
        return;
    }
    drawerVisible.value = !drawerVisible.value;
};

/** 是否有会话在执行中（turn 级事件驱动 + 进入抽屉时服务端校正） */
const hasRunningConv = computed(() => chatStore.conversations.some((conv) => chatStore.isConvRunning(conv.id)));
</script>

<style scoped>
.ai-fab__button {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 48px;
    height: 48px;
    border-radius: 9999px;
    border: none;
    cursor: pointer;
    color: var(--el-color-white);
    background: var(--el-color-primary);
    box-shadow: var(--el-box-shadow-light);
    transition:
        transform 0.2s cubic-bezier(0.25, 1, 0.5, 1),
        box-shadow 0.2s cubic-bezier(0.25, 1, 0.5, 1);
}

.ai-fab__button:hover {
    transform: scale(1.05);
    box-shadow: var(--el-box-shadow);
}

.ai-fab__button:active {
    transform: scale(0.97);
}

.ai-fab__button:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
}

.ai-fab__button--dragging {
    transform: scale(1.05);
    cursor: grabbing;
}

.ai-fab__dot {
    position: absolute;
    top: 4px;
    right: 4px;
    width: 10px;
    height: 10px;
    border-radius: 9999px;
    background: var(--el-color-success);
    box-shadow: 0 0 0 2px var(--el-color-primary);
    animation: ai-fab-dot-pulse 2s ease-in-out infinite;
}

@keyframes ai-fab-dot-pulse {
    0%,
    100% {
        opacity: 1;
    }
    50% {
        opacity: 0.5;
    }
}

@media (prefers-reduced-motion: reduce) {
    .ai-fab__button,
    .ai-fab__button:hover,
    .ai-fab__button:active {
        transition: none;
        transform: none;
    }

    .ai-fab__dot {
        animation: none;
    }
}
</style>

<style>
/* el-drawer append-to-body 挂载在 body 下，抽屉壳样式需全局作用域 */
.ai-assistant-drawer .el-drawer__body {
    padding: 0;
    overflow: hidden;
}

.ai-fab-drawer {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--el-bg-color);
}

.ai-fab-drawer__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-shrink: 0;
    padding: 8px 12px;
    border-bottom: 1px solid var(--el-border-color-lighter);
}

.ai-fab-drawer__title {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
}

.ai-fab-drawer__body {
    flex: 1;
    min-height: 0;
}
</style>
