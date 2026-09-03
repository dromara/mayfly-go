<template>
    <div class="reasoning-part">
        <!-- 内容为空时：显示思考中占位（icon 脉冲呼吸） -->
        <div v-if="!content" class="reasoning-part--active">
            <BrainIcon class="reasoning-part__icon" :class="{ 'reasoning-part__icon--pulse': active }" />
            <span class="reasoning-part__label">{{ t('ai.chat.thinkingActive') }}</span>
        </div>

        <!-- 有内容：CollapsibleSection 折叠（对齐 tokhub CollapsibleSection header） -->
        <CollapsibleSection v-else :title="titleText">
            <template #icon>
                <BrainIcon
                    class="reasoning-part__icon"
                    :class="{ 'reasoning-part__icon--pulse': active }"
                />
            </template>
            <pre class="reasoning-part__content">{{ content }}</pre>
        </CollapsibleSection>
    </div>
</template>

<script setup lang="ts">
/**
 * ReasoningPart - 思考过程展示组件
 * 对齐 tokhub ReasoningPart：CollapsibleSection 折叠展示，标题为内容摘要
 */
import { BrainIcon } from '@lucide/vue';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import CollapsibleSection from '../CollapsibleSection.vue';

const props = defineProps<{
    content: string;
    active?: boolean;
}>();

const { t } = useI18n();

/** 标题摘要：取第一行或前 80 字符 */
const titleText = computed(() => {
    const text = props.content.trim();
    const firstLine = text.split('\n')[0];
    if (firstLine.length > 80) return firstLine.slice(0, 80) + '\u2026';
    return firstLine || text.slice(0, 80);
});
</script>

<style scoped>
.reasoning-part {
    border-radius: 6px;
}

.reasoning-part--active {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 2px 6px;
    font-size: 12px;
    color: var(--el-text-color-placeholder);
}

.reasoning-part__label {
    font-size: 12px;
    color: var(--el-text-color-placeholder);
}

.reasoning-part__icon {
    width: 13px;
    height: 13px;
    flex-shrink: 0;
}

/* 脉冲呼吸动画（对齐 tokhub animate-pulse） */
.reasoning-part__icon--pulse {
    animation: reasoning-pulse 1.5s ease-in-out infinite;
}

@keyframes reasoning-pulse {
    0%, 100% {
        opacity: 1;
    }
    50% {
        opacity: 0.4;
    }
}

/* 底色由 CollapsibleSection 内容面板提供；过程性内容降一档字号（12px，对齐 tokhub 11px/13px 正文的层级比例）。
   长思考面板内独立滚动（外层 CollapsibleSection 不再裁剪，滚动职责在内容面板自身） */
.reasoning-part__content {
    font-size: 12px;
    line-height: 1.55;
    color: var(--el-text-color-regular);
    white-space: pre-wrap;
    word-break: break-word;
    margin: 0;
    font-family: inherit;
    max-height: 200px;
    overflow-y: auto;
    overflow-x: hidden;
}
</style>
