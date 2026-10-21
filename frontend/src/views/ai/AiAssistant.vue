<template>
    <AiAssistantBody route-driven />
</template>

<script setup lang="ts" name="AiAssistant">
/**
 * AiAssistant - AI 聊天助手主页面
 *
 * 主体（会话侧栏 + 聊天区）收敛在 AiAssistantBody，
 * 与全局 AiAssistantFab 抽屉共用同一实现，避免双路径维护；route-driven 声明本实例由路由挂载，
 * 路由级意图（追问草稿）只由它处理。
 *
 * 注意：本组件是路由页，模板必须保持**单根**——根层多一个注释节点就会变成 fragment，
 * 布局层的 `<transition mode="out-in">` 拿不到唯一根元素，离开本页后 router-view 会永久
 * 只剩空占位节点（实测踩过，且不产生任何异常日志）
 */
import AiAssistantBody from './components/AiAssistantBody.vue';
</script>
