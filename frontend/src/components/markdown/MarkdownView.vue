<template>
    <div class="markdown-body" v-html="html" />
</template>

<script setup lang="ts" name="MarkdownView">
/**
 * MarkdownView - Markdown 文本的只读渲染视图
 *
 * 契约：只接 `content`（Markdown 原文）。尺寸类策略（限高、滚动、边框、语义色）一律由调用方
 * 通过 class 决定——聊天气泡里不限高、抽屉里要 max-height + 滚动；本组件只负责「内容长什么样」，
 * 因此新增一处结论展示不需要再复制一遍标题/表格/代码块的排版规则。
 *
 * 消费方：AI 助手消息气泡、运行详情的结论、提案详情的理由/执行结果/校验结论。
 */
import { computed } from 'vue';
import { renderMarkdown } from './markdown';

const props = defineProps<{
    /** Markdown 原文（模型产出的结论、指令等） */
    content?: string;
}>();

const html = computed(() => renderMarkdown(props.content ?? ''));
</script>

<style scoped>
.markdown-body {
    word-break: break-word;
}

.markdown-body :deep(pre) {
    background: var(--el-fill-color-lighter);
    border-radius: 6px;
    padding: 12px;
    margin: 8px 0;
    overflow-x: auto;
    font-size: 13px;
    line-height: 1.5;
}

.markdown-body :deep(code) {
    background: var(--el-fill-color-lighter);
    border-radius: 3px;
    padding: 2px 6px;
    font-size: 13px;
    font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
}

.markdown-body :deep(pre code) {
    background: transparent;
    padding: 0;
}

.markdown-body :deep(a) {
    color: var(--el-color-primary);
    text-decoration: none;
}

.markdown-body :deep(a:hover) {
    text-decoration: underline;
}

.markdown-body :deep(strong) {
    font-weight: 600;
}

.markdown-body :deep(em) {
    font-style: italic;
}

/* markdown 标题/段落间距 */
.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3),
.markdown-body :deep(h4) {
    margin: 12px 0 6px;
    font-weight: 600;
    line-height: 1.4;
}

.markdown-body :deep(h1):first-child,
.markdown-body :deep(h2):first-child,
.markdown-body :deep(h3):first-child {
    margin-top: 0;
}

.markdown-body :deep(h1) {
    font-size: 18px;
}

.markdown-body :deep(h2) {
    font-size: 16px;
}

.markdown-body :deep(h3),
.markdown-body :deep(h4) {
    font-size: 14px;
}

.markdown-body :deep(p) {
    margin: 4px 0;
}

/* 列表（Tailwind preflight 会重置 list-style，需恢复） */
.markdown-body :deep(ul),
.markdown-body :deep(ol) {
    margin: 4px 0;
    padding-left: 20px;
}

.markdown-body :deep(ul) {
    list-style-type: disc;
}

.markdown-body :deep(ol) {
    list-style-type: decimal;
}

.markdown-body :deep(li) {
    margin: 2px 0;
}

/* 表格 */
.markdown-body :deep(table) {
    border-collapse: collapse;
    margin: 8px 0;
    width: 100%;
    font-size: 13px;
}

.markdown-body :deep(th),
.markdown-body :deep(td) {
    border: 1px solid var(--el-border-color-lighter);
    padding: 6px 10px;
    text-align: left;
}

.markdown-body :deep(th) {
    background: var(--el-fill-color-lighter);
    font-weight: 600;
}

/* 引用块：1px 全边框淡底取代侧边色条（detect: side-tab） */
.markdown-body :deep(blockquote) {
    margin: 8px 0;
    padding: 6px 12px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    background: var(--el-fill-color-lighter);
    color: var(--el-text-color-secondary);
}

.markdown-body :deep(hr) {
    border: none;
    border-top: 1px solid var(--el-border-color-lighter);
    margin: 12px 0;
}
</style>
