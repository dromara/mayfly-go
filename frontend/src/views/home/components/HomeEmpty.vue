<template>
    <div class="home-empty">
        <span class="home-empty__icon" :class="`home-empty__icon--${tone}`">
            <SvgIcon :name="icon" :size="20" />
        </span>
        <div class="home-empty__title">{{ title }}</div>
        <div v-if="desc" class="home-empty__desc">{{ desc }}</div>
    </div>
</template>

<script lang="ts" setup>
import SvgIcon from '@/components/svg-icon/index.vue';

/** 面板空状态：统一的图标底座 + 标题 + 说明，避免各面板各写一套留白 */
withDefaults(
    defineProps<{
        icon: string;
        title: string;
        desc?: string;
        tone?: 'success' | 'primary';
    }>(),
    {
        desc: '',
        tone: 'success',
    }
);
</script>

<style scoped lang="scss">
.home-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 4px;
    padding: 36px 20px;
    text-align: center;
}

.home-empty__icon {
    --empty-tone: var(--el-color-success);
    display: flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    margin-bottom: 8px;
    border-radius: 14px;
    color: var(--empty-tone);
    background: color-mix(in srgb, var(--empty-tone) 12%, transparent);

    &--primary {
        --empty-tone: var(--el-color-primary);
    }
}

.home-empty__title {
    font-size: 13px;
    font-weight: 600;
    color: var(--el-text-color-regular);
}

.home-empty__desc {
    font-size: 12px;
    color: var(--el-text-color-placeholder);
}
</style>
