<template>
    <div class="trigger-hints">
        <span v-for="hint in hints" :key="hint.char" class="trigger-hints__item">
            <kbd>{{ hint.char }}</kbd> {{ hint.label }}
        </span>
    </div>
</template>

<script setup lang="ts" name="TriggerHints">
/**
 * TriggerHints - 输入框的引用快捷键提示（`/` 选择技能、`@` 引用资源）
 *
 * 为什么单独成组件：这份提示在聊天输入框与表单提示词编辑器各写了一遍（模板 + kbd 样式），
 * 两处都没给 kbd 设 line-height，于是继承页面行高（≈1.6）把小方框撑成 23px 高、17px 宽的
 * 竖长方形，而且改一处另一处不变。数据源已是 useTriggerHints，这里把模板与样式一并收口。
 *
 * 单根 div：消费方通过 class 挂自己的定位/过渡样式（多根组件无法继承外层 class）。
 */
import { computed } from 'vue';
import { triggerHints } from './useTriggerHints';

const hints = computed(() => triggerHints());
</script>

<style scoped>
.trigger-hints {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 4px 10px;
    font-size: 12px;
    /* 与 kbd 同高：不写就会继承外层 1.5~1.6 的行高，把整行提示条撑高 */
    line-height: 18px;
    color: var(--el-text-color-placeholder);
}

.trigger-hints__item {
    display: inline-flex;
    align-items: center;
    gap: 4px;
}

/* 固定 18×18 正方形：字符本身宽度不一（`/` 窄、`@` 宽），只靠 padding 撑会得到宽窄不同的长方形 */
.trigger-hints__item kbd {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    box-sizing: border-box;
    width: 18px;
    height: 18px;
    line-height: 1;
    font-size: 11px;
    font-family: inherit;
    background: var(--el-fill-color);
    border: 1px solid var(--el-border-color);
    border-radius: 4px;
}
</style>
