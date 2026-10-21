<template>
    <div v-if="visibleActions.length" class="quick-actions">
        <button
            v-for="(action, index) of visibleActions"
            :key="action.key"
            type="button"
            class="quick-action"
            :class="[`quick-action--${action.tone}`, { 'is-wide': index === visibleActions.length - 1 && visibleActions.length % 2 === 1 }]"
            @click="$emit('navigate', action)"
        >
            <span class="quick-action__icon">
                <SvgIcon :name="action.icon" :size="16" />
            </span>
            <span class="quick-action__text">
                <span class="quick-action__title">{{ $t(action.titleKey) }}</span>
                <span class="quick-action__desc">{{ $t(action.descKey) }}</span>
            </span>
            <SvgIcon name="ArrowRight" class="quick-action__arrow" :size="12" />
        </button>
    </div>

    <HomeEmpty v-else icon="Lock" :title="$t('home.noQuickAction')" :desc="$t('home.noQuickActionDesc')" tone="primary" />
</template>

<script lang="ts" setup>
import { hasPerm } from '@/components/auth/auth';
import SvgIcon from '@/components/svg-icon/index.vue';
import { computed } from 'vue';
import HomeEmpty from './HomeEmpty.vue';

/** 快捷入口项：路由名用于跳转，权限码决定是否展示（菜单授权时会连带下发基础权限码） */
export interface QuickActionItem {
    key: string;
    icon: string;
    tone: 'primary' | 'success' | 'warning' | 'danger';
    titleKey: string;
    descKey: string;
    routeName: string;
    /** 基础权限码：没有该码说明连对应菜单都没授权，入口不应展示 */
    permission: string;
}

defineEmits<{
    (e: 'navigate', action: QuickActionItem): void;
}>();

const actions: QuickActionItem[] = [
    {
        key: 'resource',
        icon: 'Menu',
        tone: 'primary',
        titleKey: 'home.goMyResource',
        descKey: 'home.goMyResourceDesc',
        routeName: 'ResourceOp',
        permission: 'resource:base',
    },
    {
        key: 'alert',
        icon: 'Bell',
        tone: 'danger',
        titleKey: 'home.goAlerts',
        descKey: 'home.goAlertsDesc',
        routeName: 'AlertEventList',
        permission: 'alert:event',
    },
    {
        key: 'ai',
        icon: 'MagicStick',
        tone: 'success',
        titleKey: 'home.goAiAssistant',
        descKey: 'home.goAiAssistantDesc',
        routeName: 'AiAssistant',
        permission: 'ai:chat',
    },
    {
        key: 'flow',
        icon: 'Document',
        tone: 'primary',
        titleKey: 'home.goFlow',
        descKey: 'home.goFlowDesc',
        routeName: 'ProcinstList',
        permission: 'flow:base',
    },
];

/**
 * 入口可见性只看基础权限码：没有权限码说明对应菜单本身就没有授权，
 * 不需要再叠路由/菜单结构判断（这样每个入口的配置也只有一条）
 */
const visibleActions = computed(() => actions.filter((action) => hasPerm(action.permission)));
</script>

<style scoped lang="scss">
.quick-actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
    padding: 16px 18px 18px;

    @media (max-width: 640px) {
        grid-template-columns: minmax(0, 1fr);
    }
}

.quick-action {
    --action-tone: var(--el-color-primary);
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 12px;
    font: inherit;
    text-align: left;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 12px;
    background: transparent;
    cursor: pointer;
    transition:
        border-color 0.2s ease,
        background-color 0.2s ease,
        transform 0.2s ease;

    /* 入口为奇数时最后一个占满整行，避免 2 列网格留下空位 */
    &.is-wide {
        grid-column: 1 / -1;
    }

    &--success {
        --action-tone: var(--el-color-success);
    }

    &--warning {
        --action-tone: var(--el-color-warning);
    }

    &--danger {
        --action-tone: var(--el-color-danger);
    }

    &:hover {
        transform: translateY(-1px);
        border-color: color-mix(in srgb, var(--action-tone) 40%, var(--el-border-color-lighter));
        background: color-mix(in srgb, var(--action-tone) 6%, transparent);

        .quick-action__title {
            color: var(--action-tone);
        }

        .quick-action__arrow {
            color: var(--action-tone);
            transform: translateX(2px);
        }
    }

    &:focus-visible {
        outline: 2px solid var(--action-tone);
        outline-offset: 2px;
    }
}

.quick-action__icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    width: 32px;
    height: 32px;
    border-radius: 10px;
    color: var(--action-tone);
    background: color-mix(in srgb, var(--action-tone) 12%, transparent);
}

.quick-action__text {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
    min-width: 0;
}

.quick-action__title {
    font-size: 13px;
    font-weight: 600;
    line-height: 1.4;
    color: var(--el-text-color-primary);
    transition: color 0.2s ease;
}

.quick-action__desc {
    font-size: 11px;
    line-height: 1.4;
    color: var(--el-text-color-placeholder);
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
}

.quick-action__arrow {
    flex-shrink: 0;
    align-self: center;
    color: var(--el-text-color-placeholder);
    transition:
        color 0.2s ease,
        transform 0.2s ease;
}

@media (prefers-reduced-motion: reduce) {
    .quick-action:hover {
        transform: none;
    }
}
</style>
