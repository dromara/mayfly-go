<template>
    <el-card shadow="never" class="hero">
        <div class="hero__decor" aria-hidden="true"></div>

        <div class="hero__content">
            <div class="hero__main">
                <UserAvatarUpload :username="username" :name="name" :photo="photo" :size="64" />

                <div class="hero__intro">
                    <div class="hero__title-row">
                        <h1 class="hero__title">{{ greetingText }}</h1>
                        <span v-if="roleText" class="hero__role">{{ roleText }}</span>
                    </div>
                    <p class="hero__subtitle">{{ $t('home.heroSubtitle') }}</p>
                    <div class="hero__meta">
                        <span class="hero__meta-item">
                            <SvgIcon name="User" :size="13" />
                            {{ username }}
                        </span>
                        <span v-if="lastLoginTime" class="hero__meta-item">
                            <SvgIcon name="Clock" :size="13" />
                            {{ $t('home.lastLogin') }} {{ formatDate(lastLoginTime, 'YYYY-MM-DD HH:mm') }}
                        </span>
                        <span v-if="lastLoginIp" class="hero__meta-item">
                            <SvgIcon name="Link" :size="13" />
                            {{ lastLoginIp }}
                        </span>
                    </div>
                </div>
            </div>

            <div class="hero__side">
                <div class="hero__clock">
                    <span class="hero__clock-time">{{ clock }}</span>
                    <span class="hero__clock-date">{{ dateText }}</span>
                </div>
                <div class="hero__actions">
                    <el-button :loading="refreshing" icon="Refresh" circle :title="$t('home.refresh')" @click="$emit('refresh')" />
                    <el-button icon="User" @click="$emit('navigate-personal')">{{ $t('home.personalCenter') }}</el-button>
                </div>
            </div>
        </div>
    </el-card>
</template>

<script lang="ts" setup>
import { formatDate } from '@/common/utils/format';
import SvgIcon from '@/components/svg-icon/index.vue';
import UserAvatarUpload from '@/components/user-avatar-upload/UserAvatarUpload.vue';
import dayjs from 'dayjs';
import { computed, onUnmounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { greetingKeyOf } from '../utils';

const props = defineProps<{
    name: string;
    username: string;
    photo?: string;
    roleText?: string;
    lastLoginIp?: string;
    lastLoginTime?: string;
    refreshing?: boolean;
}>();

defineEmits<{
    (e: 'refresh'): void;
    (e: 'navigate-personal'): void;
}>();

const { t, locale } = useI18n();

// 每 30s 走一次：问候语与时钟跟着真实时间切档，又不必为分钟级刷新付出每秒渲染成本
const now = ref(new Date());
const timer = window.setInterval(() => {
    now.value = new Date();
}, 30 * 1000);
onUnmounted(() => window.clearInterval(timer));

const greeting = computed(() => t(greetingKeyOf(now.value)));
// 用户信息尚未回填时不拼一个空名字（会显示成「下午好，」）
const greetingText = computed(() => (props.name ? t('home.greetingName', { greeting: greeting.value, name: props.name }) : greeting.value));
const clock = computed(() => dayjs(now.value).format('HH:mm'));
const dateText = computed(() => {
    const localeCode = locale.value === 'en' ? 'en-US' : 'zh-CN';
    const weekday = new Intl.DateTimeFormat(localeCode, { weekday: 'long' }).format(now.value);
    return `${dayjs(now.value).format('YYYY-MM-DD')} ${weekday}`;
});
</script>

<style scoped lang="scss">
.hero {
    --el-card-border-radius: 18px;
    position: relative;
    overflow: hidden;
}

/* 装饰层独立成 div 而非 ::before：玻璃主题把 .el-card::before 用作了背板模糊层，占用同一伪元素会把它覆盖掉 */
.hero__decor {
    position: absolute;
    inset: 0;
    pointer-events: none;
    background:
        radial-gradient(60% 120% at 8% 0%, color-mix(in srgb, var(--el-color-primary) 22%, transparent), transparent 62%),
        radial-gradient(50% 110% at 88% 0%, color-mix(in srgb, var(--el-color-success) 16%, transparent), transparent 66%),
        linear-gradient(120deg, color-mix(in srgb, var(--el-color-primary) 6%, transparent), transparent 55%);
    opacity: 0.9;
}

.hero__content {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    gap: 20px;
    justify-content: space-between;

    @media (min-width: 900px) {
        flex-direction: row;
        align-items: center;
        gap: 24px;
    }
}

.hero__main {
    display: flex;
    align-items: center;
    gap: 16px;
    min-width: 0;
}

.hero__intro {
    min-width: 0;
}

.hero__title-row {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 10px;
}

.hero__title {
    margin: 0;
    font-size: clamp(20px, 1.6vw, 24px);
    font-weight: 700;
    line-height: 1.3;
    letter-spacing: -0.01em;
    color: var(--el-text-color-primary);
}

.hero__role {
    max-width: 240px;
    padding: 2px 10px;
    font-size: 11px;
    font-weight: 600;
    line-height: 18px;
    color: var(--el-color-primary);
    background: color-mix(in srgb, var(--el-color-primary) 14%, transparent);
    border: 1px solid color-mix(in srgb, var(--el-color-primary) 24%, transparent);
    border-radius: 999px;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
}

.hero__subtitle {
    margin: 6px 0 0;
    font-size: 13px;
    color: var(--el-text-color-secondary);
}

.hero__meta {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 6px 14px;
    margin-top: 10px;
}

.hero__meta-item {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: 12px;
    color: var(--el-text-color-placeholder);
}

.hero__side {
    display: flex;
    align-items: center;
    gap: 20px;
    flex-shrink: 0;
}

.hero__clock {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    line-height: 1.1;
}

.hero__clock-time {
    font-size: 30px;
    font-weight: 700;
    letter-spacing: -0.02em;
    font-variant-numeric: tabular-nums;
    color: var(--el-text-color-primary);
}

.hero__clock-date {
    margin-top: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
}

.hero__actions {
    display: flex;
    align-items: center;
    gap: 8px;
}

@media (max-width: 899px) {
    .hero__side {
        justify-content: space-between;
    }

    .hero__clock {
        align-items: flex-start;
    }
}

@media (max-width: 560px) {
    .hero__main {
        align-items: flex-start;
    }

    .hero__clock-time {
        font-size: 24px;
    }
}
</style>
