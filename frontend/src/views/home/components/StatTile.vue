<template>
    <el-card shadow="never" class="stat-tile" :class="[`stat-tile--${tone}`, { 'is-clickable': clickable }]">
        <div
            class="stat-tile__inner"
            :role="clickable ? 'button' : undefined"
            :tabindex="clickable ? 0 : undefined"
            @click="onClick"
            @keydown.enter="onClick"
            @keydown.space.prevent="onClick"
        >
            <div class="stat-tile__icon">
                <SvgIcon :name="icon" :size="18" />
            </div>

            <div class="stat-tile__content">
                <div class="stat-tile__label">{{ label }}</div>
                <div class="stat-tile__value">
                    <span v-if="loading" class="stat-tile__skeleton" aria-hidden="true"></span>
                    <template v-else-if="failed">--</template>
                    <template v-else>{{ value ?? '—' }}</template>
                </div>
                <div class="stat-tile__hint">{{ failed ? $t('home.loadFailed') : hint }}</div>
            </div>

            <SvgIcon v-if="clickable" name="ArrowRight" class="stat-tile__arrow" :size="14" />
        </div>
    </el-card>
</template>

<script lang="ts" setup>
import SvgIcon from '@/components/svg-icon/index.vue';

/** 指标色调：只影响图标底色与强调色，文字仍走主题文本色以保证暗色/玻璃主题下的对比度 */
type StatTone = 'primary' | 'danger' | 'warning' | 'success';

const props = withDefaults(
    defineProps<{
        icon: string;
        label: string;
        /** null 表示数据尚未就绪，渲染为占位横线而不是 0 */
        value?: number | null;
        hint?: string;
        tone?: StatTone;
        clickable?: boolean;
        loading?: boolean;
        failed?: boolean;
    }>(),
    {
        value: null,
        hint: '',
        tone: 'primary',
        clickable: false,
        loading: false,
        failed: false,
    }
);

const emit = defineEmits<{
    (e: 'select'): void;
}>();

const onClick = () => {
    if (props.clickable) {
        emit('select');
    }
};
</script>

<style scoped lang="scss">
.stat-tile {
    --el-card-border-radius: 16px;
    --tile-tone: var(--el-color-primary);
    height: 100%;
    transition: transform 0.2s ease;

    &.is-clickable {
        cursor: pointer;

        &:hover {
            transform: translateY(-2px);

            .stat-tile__arrow {
                opacity: 1;
                transform: translateX(2px);
            }
        }
    }

    &--danger {
        --tile-tone: var(--el-color-danger);
    }

    &--warning {
        --tile-tone: var(--el-color-warning);
    }

    &--success {
        --tile-tone: var(--el-color-success);
    }
}

.stat-tile__inner {
    display: flex;
    align-items: center;
    gap: 14px;
    height: 100%;
    border-radius: 12px;

    &:focus-visible {
        outline: 2px solid var(--tile-tone);
        outline-offset: 4px;
    }
}

.stat-tile__icon {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    width: 40px;
    height: 40px;
    border-radius: 12px;
    color: var(--tile-tone);
    background: color-mix(in srgb, var(--tile-tone) 14%, transparent);
}

.stat-tile__content {
    flex: 1;
    min-width: 0;
}

.stat-tile__label {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
}

.stat-tile__value {
    margin-top: 2px;
    font-size: 26px;
    font-weight: 700;
    line-height: 1.15;
    font-variant-numeric: tabular-nums;
    color: var(--el-text-color-primary);
}

.stat-tile__hint {
    margin-top: 2px;
    font-size: 11px;
    line-height: 1.4;
    color: var(--el-text-color-placeholder);
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
}

.stat-tile__skeleton {
    display: inline-block;
    width: 56px;
    height: 24px;
    border-radius: 6px;
    background: var(--el-fill-color);
    animation: stat-tile-pulse 1.2s ease-in-out infinite;
}

.stat-tile__arrow {
    flex-shrink: 0;
    color: var(--tile-tone);
    opacity: 0;
    transition:
        opacity 0.2s ease,
        transform 0.2s ease;
}

@keyframes stat-tile-pulse {
    0%,
    100% {
        opacity: 0.45;
    }
    50% {
        opacity: 1;
    }
}

@media (prefers-reduced-motion: reduce) {
    .stat-tile.is-clickable:hover {
        transform: none;
    }

    .stat-tile__skeleton {
        animation: none;
    }
}
</style>
