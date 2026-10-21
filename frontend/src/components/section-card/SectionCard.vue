<template>
    <el-card shadow="never" class="section-card">
        <div class="section-card__head">
            <div class="section-card__title">
                <span v-if="icon" class="section-card__icon">
                    <SvgIcon :name="icon" :size="15" />
                </span>
                <span class="section-card__text">{{ title }}</span>
                <span v-if="count !== undefined && count !== null" class="section-card__count">{{ count }}</span>
            </div>
            <div class="section-card__actions">
                <slot name="actions" />
            </div>
        </div>
        <div class="section-card__body">
            <slot />
        </div>
    </el-card>
</template>

<script lang="ts" setup>
import SvgIcon from '@/components/svg-icon/index.vue';

/**
 * 通用分区卡：标题栏（图标 + 标题 + 计数徽标 + 右侧动作）与内容区。
 * 外壳用 el-card 而不是自绘 div，是为了直接复用玻璃主题对 .el-card 的描边/高光/背板模糊规则。
 * 首页各面板与个人中心安全设置共用，避免两处各写一套标题栏
 */
defineProps<{
    title: string;
    icon?: string;
    count?: number | null;
}>();
</script>

<style scoped lang="scss">
.section-card {
    --el-card-border-radius: 16px;
    height: 100%;

    :deep(.el-card__body) {
        display: flex;
        flex-direction: column;
        height: 100%;
        padding: 0;
    }
}

.section-card__head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 14px 18px;
    border-bottom: 1px solid var(--el-border-color-extra-light);
}

.section-card__title {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
}

.section-card__text {
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
}

.section-card__icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    border-radius: 8px;
    color: var(--el-color-primary);
    background: color-mix(in srgb, var(--el-color-primary) 12%, transparent);
}

.section-card__count {
    flex-shrink: 0;
    padding: 0 7px;
    font-size: 12px;
    font-weight: 600;
    line-height: 18px;
    font-variant-numeric: tabular-nums;
    border-radius: 999px;
    color: var(--el-color-primary);
    background: color-mix(in srgb, var(--el-color-primary) 12%, transparent);
}

.section-card__actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
}

.section-card__body {
    flex: 1;
    min-height: 0;
}
</style>
