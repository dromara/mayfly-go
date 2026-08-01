<template>
    <div class="h-full flex flex-col p-5 justify-center">
        <div class="w-full flex-1 overflow-hidden" v-if="messages.length > 0">
            <BubbleList v-loading="msgLoading" ref="bubbleListRef" :key="bubbleListKey" :list="messages" max-height="100%" :virtual="false">
                <template #avatar="{ item }">
                    <SvgIcon v-if="item.role == ROLE.AI" :size="24" name="icon ai/assistant" color="var(--el-color-primary)" />
                    <img v-else class="size-10 max-w-none rounded-full" :src="useUserInfo().userInfo.photo" alt="avatar" />
                </template>

                <template #header="{ item }">
                    <ThoughtChain :thinking-items="item.thinks as never[]" dot-size="small" class="min-w-150 max-w-300" :max-width="BUBBLE_MAX_WIDTH" row-key="id">
                    </ThoughtChain>

                    <!-- 中断组件：横向 flex 排列，宽度缩小 -->
                    <div class="flex flex-wrap gap-2">
                        <template v-for="internal in item.internals" :key="internal.id || internal.extra?.interruptId">
                            <component
                                class="max-w-120 shrink-0"
                                v-if="internal.extra?.type?.startsWith('interrupt_')"
                                :is="getInterruptComponent(internal.extra?.type)"
                                :data="internal"
                                :readonly="internal.extra?.resumeInfo"
                                @action="handleInterruptAction"
                            />
                        </template>
                    </div>

                    <!-- 其他类型 internal 纵向排列 -->
                    <div v-for="internal in item.internals" :key="internal.id || internal.extra?.interruptId" class="mt-1">
                        <div
                            v-if="internal.extra?.type === 'notification'"
                            class="p-3 bg-blue-50 dark:bg-blue-900/20 rounded border border-blue-200 dark:border-blue-800"
                        >
                            <div class="text-sm font-medium text-blue-700 dark:text-blue-300">
                                {{ internal.extra?.content?.title }}
                            </div>
                            <div class="text-xs text-blue-600 dark:text-blue-400 mt-1">
                                {{ internal.extra?.content?.description }}
                            </div>
                        </div>

                        <!-- 未知类型的降级展示 -->
                        <div
                            v-else-if="!internal.extra?.type?.startsWith('interrupt_')"
                            class="px-2 py-1.5 bg-gray-50 dark:bg-gray-800 rounded border border-gray-200 dark:border-gray-700"
                        >
                            <div class="text-xs text-gray-500 dark:text-gray-400">
                                <span class="font-medium">{{ t('ai.chat.type') }}:</span> {{ internal.extra?.type || internal.type || 'unknown' }}
                            </div>
                            <div v-if="internal.content || internal.extra?.content" class="text-xs text-gray-600 dark:text-gray-300 mt-0.5">
                                {{ internal.content || internal.extra?.content?.description || JSON.stringify(internal.extra?.content || internal.content) }}
                            </div>
                        </div>
                    </div>

                    <!-- 中断处理进度提示：当有未处理的中断时显示进度 -->
                    <div
                        v-if="(item.unprocessedInterruptCount ?? 0) > 0"
                        class="mt-2 mb-2 px-2 py-1 bg-blue-50 dark:bg-blue-900/20 rounded border border-blue-200 dark:border-blue-800"
                    >
                        <div class="flex items-center justify-between">
                            <span class="text-xs text-blue-700 dark:text-blue-300">
                                {{ t('ai.chat.processed') }} {{ item.pendingResumes?.length || 0 }} / {{ item.unprocessedInterruptCount }}
                                {{ t('ai.chat.interrupts') }}
                                <span
                                    v-if="item.pendingResumes?.length === item.unprocessedInterruptCount"
                                    class="ml-1 text-xs text-green-600 dark:text-green-400"
                                >
                                    {{ t('ai.chat.submitting') }}
                                </span>
                            </span>
                        </div>
                    </div>
                </template>

                <template #content="{ item }">
                    <!-- chat 内容走 markdown -->
                    <MarkdownRenderer
                        v-if="item.role === ROLE.AI || item.role == ROLE.INTERNAL"
                        :markdown="item.content"
                        :enable-animate="true"
                        :is-dark="isDark"
                        :themes="{ light: 'github-light', dark: 'github-dark' }"
                        :default-theme-mode="isDark ? 'dark' : 'light'"
                        class="max-w-300"
                    />

                    <!-- user 内容 纯文本 -->
                    <div v-if="item.role === ROLE.USER" class="whitespace-pre-wrap">
                        {{ item.content }}
                    </div>
                </template>

                <template #footer="{ item }">
                    <div class="flex justify-between items-center">
                        <div>
                            <el-button @click="copyToClipboard(item.content)" color="#626aef" icon="DocumentCopy" size="small" circle />
                        </div>
                        <div class="ml-2 mt-1 text-xs">
                            {{ item.time }}
                        </div>
                    </div>
                </template>
            </BubbleList>
        </div>

        <!-- 输入框区域：固定在底部 -->
        <div class="w-full mt-4">
            <XSender
                ref="senderRef"
                style="border-radius: 24px"
                @submit="onSubmit"
                :loading="senderLoading"
                :disabled="senderLoading"
                :custom-dialog="true"
                :custom-trigger="customSenderTrigger"
                @show-tag-dialog="showTagDialog"
                submit-type="enter"
                :auto-focus="true"
                variant="updown"
                clearable
            >
            </XSender>

            <el-dialog v-model="dialogCustomVisible" :title="t('ai.chat.customTriggerDialogTitle')" width="500">
                <template v-for="option of customSenderTrigger" :key="option.prefix">
                    <p v-for="tag of option.tagList" :key="tag.id" @click="checkTag(tag)">
                        {{ tag.name }}
                    </p>
                </template>
            </el-dialog>
        </div>
    </div>
</template>

<script setup lang="ts" name="AiChat">
import { copyToClipboard } from '@/common/utils/string';
import { useThemeConfig } from '@/store/themeConfig';
import { useUserInfo } from '@/store/userInfo';
import { computed, reactive, ref, toRefs, useTemplateRef, watch } from 'vue';
import { useDebounceFn } from '@vueuse/core';
import { BubbleList, ThoughtChain, XSender } from 'vue-element-plus-x';
import type { BubbleListInstance } from 'vue-element-plus-x/types/BubbleList';
import { useI18n } from 'vue-i18n';
import { MarkdownRenderer } from 'x-markdown-vue';
import 'x-markdown-vue/style';
import { getInterruptComponent } from './interrupt';
import { useAiChatWebSocket } from './hooks/useAiChatWebSocket';
import { useAiChatMessages } from './hooks/useAiChatMessages';

const { t } = useI18n();

const BUBBLE_MAX_WIDTH = '80%';

// 模板中使用的常量
const ROLE = {
    AI: 'assistant',
    USER: 'user',
    INTERNAL: 'internal',
} as const;

const props = defineProps({
    sessionId: {
        type: String,
        default: '',
    },
});

const emit = defineEmits(['activate']);

// 内部跟踪当前 sessionId，支持新会话时从空值被后端赋值
const currentSessionId = ref('');

// 新会话标识（统一用 '-1' 表示）
const isNewSession = computed(() => props.sessionId === '-1');

const themeConfig = useThemeConfig();
const isDark = computed(() => themeConfig.themeConfig.isDark);

const senderRef = useTemplateRef<InstanceType<typeof XSender>>('senderRef');
const bubbleListRef = useTemplateRef<BubbleListInstance>('bubbleListRef');

/**
 * 滚动到底部（防抖处理，500ms 内只执行一次）
 */
const scrollToBottom = useDebounceFn(() => {
    bubbleListRef.value?.scrollToBottom();
}, 500);

// 使用 WebSocket hook
const messageHandler = (data: any) => {
    messageHook.handleChunk(data);
};

const { initSocket, sendMessage, reconnectAttempts } = useAiChatWebSocket(messageHandler, currentSessionId, isNewSession);

// 使用消息管理 hook
const messageHook = useAiChatMessages(props, emit, currentSessionId, sendMessage);

// 直接使用 hook 的 state，保持响应式连接
const { msgLoading, senderLoading } = toRefs(messageHook.state);

// 监听消息变化，自动滚动到底部
watch(
    () => messageHook.messages.value.length,
    () => {
        scrollToBottom();
    }
);

const state = reactive({
    dialogCustomVisible: false,
    tagPrefix: '',
});

const { dialogCustomVisible } = toRefs(state);

// 提供中断操作处理器给子组件
const handleInterruptAction = messageHook.handleInterruptAction;

let isIniting = false;

const init = async () => {
    if (isIniting) {
        return;
    }
    isIniting = true;
    try {
        if (!reconnectAttempts.value || reconnectAttempts.value === 0) {
            initSocket();
        }
        if (isNewSession.value) {
            messageHook.state.senderLoading = false;
        }
        messageHook.loadMessage();
    } finally {
        isIniting = false;
    }
};

watch(
    () => props.sessionId,
    async (newVal: string) => {
        if (!newVal || newVal === currentSessionId.value) {
            return;
        }
        currentSessionId.value = newVal;
        if (isNewSession.value) {
            messageHook.resetMessages();
        }
        init();
    },
    { immediate: true }
);

// 自定义触发符相关逻辑
const customSenderTrigger = ref([
    {
        dialogTitle: t('ai.interrupt.assetSelection.title'),
        prefix: '#',
        tagList: [
            { id: 'ht1', name: '话题一' },
            { id: 'ht2', name: '话题二' },
        ],
    },
]);

const showTagDialog = (prefix: string) => {
    state.tagPrefix = prefix;
    state.dialogCustomVisible = true;
};

const checkTag = (tag: any) => {
    senderRef.value?.customSetTag(state.tagPrefix, tag);
    dialogCustomVisible.value = false;
};

/**
 * 转为组件需要的数据
 */
const messages = messageHook.messages;
const bubbleListKey = messageHook.bubbleListKey;

const clearSenderEditor = () => {
    senderRef.value?.clear();
};

const onSubmit = () => {
    try {
        const content = senderRef.value?.getModelValue().text;
        // 先添加消息到 UI
        messageHook.addUserMessage(content);
        // 再发送 WebSocket 消息
        sendMessage('text', content);
        messageHook.state.senderLoading = true;
    } finally {
        clearSenderEditor();
    }
};
</script>

<style></style>
