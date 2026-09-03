<template>
    <AiAssistantBody />
</template>

<script setup lang="ts" name="AiAssistant">
/**
 * AiAssistant - AI 聊天助手主页面
 *
 * 主体（会话侧栏 + 聊天区）收敛在 AiAssistantBody，
 * 与全局 AiAssistantFab 抽屉共用同一实现，避免双路径维护
 */
import AiAssistantBody from './components/AiAssistantBody.vue';
</script>
