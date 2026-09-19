<template>
    <div class="log-viewer" :class="{ 'is-loading': loading }" :style="containerStyle">
        <!-- 顶部工具栏 -->
        <div class="log-viewer__toolbar">
            <!-- 左侧：级别过滤 -->
            <div class="log-viewer__filters">
                <el-checkbox-group v-model="activeLevels" size="small" @change="onFilterChange">
                    <el-checkbox-button v-for="level in allLevels" :key="level" :value="level" :class="`level-${level.toLowerCase()}`">
                        <el-icon class="mr-1"><component :is="getLevelIcon(level)" /></el-icon>
                        {{ level }}
                    </el-checkbox-button>
                </el-checkbox-group>

                <!-- 搜索框 -->
                <el-input
                    v-model="searchKeyword"
                    :placeholder="$t('components.logViewer.searchPlaceholder')"
                    size="small"
                    clearable
                    class="log-viewer__search"
                    @input="onSearchChange"
                >
                    <template #prefix>
                        <el-icon><Search /></el-icon>
                    </template>
                </el-input>
            </div>

            <!-- 右侧：操作按钮 -->
            <div class="log-viewer__actions">
                <!-- 匹配计数 -->
                <span v-if="searchKeyword" class="log-viewer__match-count">
                    {{ filteredLines.length }} {{ $t('components.logViewer.matchedLines') }}
                </span>

                <!-- 自动滚动开关 -->
                <el-tooltip :content="autoScroll ? $t('components.logViewer.disableAutoScroll') : $t('components.logViewer.enableAutoScroll')">
                    <el-switch v-model="autoScroll" size="small" :active-action-icon="Bottom" />
                </el-tooltip>

                <!-- 刷新按钮 -->
                <el-tooltip :content="$t('common.refresh')">
                    <el-button :icon="Refresh" size="small" circle :loading="loading" @click="refresh" />
                </el-tooltip>

                <!-- 下载按钮 -->
                <el-tooltip :content="$t('components.logViewer.download')">
                    <el-button :icon="Download" size="small" circle @click="download" />
                </el-tooltip>

                <!-- 清空按钮 -->
                <el-tooltip :content="$t('common.clear')">
                    <el-button :icon="Delete" size="small" circle @click="clear" />
                </el-tooltip>
            </div>
        </div>

        <!-- 指标面板（可选插槽） -->
        <div v-if="$slots.metrics" class="log-viewer__metrics">
            <slot name="metrics" />
        </div>

        <!-- 日志流主体（虚拟滚动） -->
        <div ref="containerRef" class="log-viewer__content" @scroll="onScroll">
            <!-- 空状态 -->
            <el-empty v-if="filteredLines.length === 0 && !loading" :description="$t('components.logViewer.empty')" />

            <!-- 加载中 -->
            <div v-if="loading && filteredLines.length === 0" class="log-viewer__loading">
                <el-icon class="is-loading"><Loading /></el-icon>
                <span>{{ $t('components.logViewer.loading') }}</span>
            </div>

            <!-- 虚拟滚动列表 -->
            <div v-else class="log-viewer__list" :style="{ height: `${totalHeight}px` }">
                <div class="log-viewer__spacer" :style="{ transform: `translateY(${offsetY}px)` }">
                    <div
                        v-for="line in visibleLines"
                        :key="line.lineIndex"
                        :class="['log-viewer__line', `level-${line.level.toLowerCase()}`]"
                        :style="{ height: `${props.lineHeight}px` }"
                    >
                        <!-- 行号 -->
                        <span v-if="theme.showLineNumbers" class="log-viewer__line-number">{{ line.lineIndex + 1 }}</span>

                        <!-- 时间戳 -->
                        <span v-if="theme.showTimestamp && line.timestamp" class="log-viewer__timestamp">{{ line.timestamp }}</span>

                        <!-- 级别图标 -->
                        <el-icon v-if="theme.showLevelIcon" :class="['log-viewer__level-icon', `level-${line.level.toLowerCase()}`]">
                            <component :is="getLevelIcon(line.level)" />
                        </el-icon>

                        <!-- 消息内容（支持搜索高亮） -->
                        <span class="log-viewer__message" v-html="highlightText(line.message)" />
                    </div>
                </div>
            </div>
        </div>

        <!-- 底部状态栏 -->
        <div class="log-viewer__statusbar">
            <span class="log-viewer__status">
                <el-icon v-if="loading" class="is-loading"><Loading /></el-icon>
                <el-icon v-else-if="finished"><CircleCheck /></el-icon>
                <el-icon v-else><MoreFilled /></el-icon>
                {{ statusText }}
            </span>
            <span class="log-viewer__count">
                {{ totalLines }} {{ $t('components.logViewer.totalLines') }}
            </span>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { Bottom, CircleCheck, CircleCloseFilled, Delete, Download, Loading, MoreFilled, Refresh, Search, WarningFilled } from '@element-plus/icons-vue';
import { useElementSize } from '@vueuse/core';
import { computed, nextTick, onMounted, ref, watch, type Component } from 'vue';
import { useI18n } from 'vue-i18n';

import type { LogLevel, LogViewerTheme, ParsedLogLine } from './types';
import { LogLevel as LogLevelEnum } from './types';

const { t } = useI18n();

// Props
const props = withDefaults(
    defineProps<{
        /** 日志行数据 */
        lines: ParsedLogLine[];
        /** 是否正在加载 */
        loading?: boolean;
        /** 是否已结束 */
        finished?: boolean;
        /** 总行数（用于状态栏显示） */
        totalLines?: number;
        /** 行高（px） */
        lineHeight?: number;
        /** 主题配置 */
        theme?: Partial<LogViewerTheme>;
    }>(),
    {
        loading: false,
        finished: false,
        totalLines: 0,
        lineHeight: 24,
        theme: () => ({}),
    }
);

// Emits
const emit = defineEmits<{
    refresh: [];
    clear: [];
    download: [];
}>();

// 默认主题
const defaultTheme: LogViewerTheme = {
    background: 'var(--el-fill-color-blank)',
    fontSize: '13px',
    lineHeight: '24px',
    showLineNumbers: true,
    showTimestamp: true,
    showLevelIcon: true,
};

const theme = computed(() => ({ ...defaultTheme, ...props.theme }));

// 容器样式
const containerStyle = computed(() => ({
    background: theme.value.background,
    fontSize: theme.value.fontSize,
}));

// 所有日志级别
const allLevels = Object.values(LogLevelEnum);

// 过滤状态
const activeLevels = ref<LogLevel[]>([]);
const searchKeyword = ref('');

// 自动滚动
const autoScroll = ref(true);

// DOM 引用
const containerRef = ref<HTMLElement | null>(null);

// 虚拟滚动状态
const scrollTop = ref(0);
const containerHeight = ref(500);

// 过滤后的日志行
const filteredLines = computed(() => {
    let result = props.lines;

    // 级别过滤
    if (activeLevels.value.length > 0) {
        result = result.filter((line) => activeLevels.value.includes(line.level));
    }

    // 搜索过滤（归一化匹配：去掉 _ 和 - 分隔符后比较，使 "syslog" 能匹配 "sys_log"）
    if (searchKeyword.value) {
        const keyword = normalizeText(searchKeyword.value);
        result = result.filter((line) => normalizeText(line.message).includes(keyword) || normalizeText(line.raw).includes(keyword));
    }

    return result;
});

/** 归一化文本：转小写并移除常见分隔符，用于模糊匹配 */
const normalizeText = (text: string): string => {
    return text.toLowerCase().replace(/[_\-\s]/g, '');
};

// 虚拟滚动计算
const { height: measuredHeight } = useElementSize(containerRef);
watch(measuredHeight, (h) => {
    if (h > 0) {
        containerHeight.value = h;
        // 容器高度变化时（如 dialog 打开后），重新滚动到底部
        if (autoScroll.value && props.lines.length > 0) {
            nextTick(scrollToBottom);
        }
    }
});

const totalHeight = computed(() => filteredLines.value.length * props.lineHeight);

const visibleCount = computed(() => Math.ceil(containerHeight.value / props.lineHeight) + 5);

const startIndex = computed(() => Math.floor(scrollTop.value / props.lineHeight));

const offsetY = computed(() => startIndex.value * props.lineHeight);

const visibleLines = computed(() => {
    const start = startIndex.value;
    const end = Math.min(start + visibleCount.value, filteredLines.value.length);
    return filteredLines.value.slice(start, end);
});

// 状态文本
const statusText = computed(() => {
    if (props.loading) return t('components.logViewer.loading');
    if (props.finished) return t('components.logViewer.finished');
    return t('components.logViewer.streaming');
});

// 方法

/** 每个级别使用不同图标，便于视觉区分 */
const getLevelIcon = (level: LogLevel): Component => {
    const icons: Record<string, Component> = {
        [LogLevelEnum.DEBUG]: MoreFilled,
        [LogLevelEnum.INFO]: CircleCheck,
        [LogLevelEnum.WARN]: WarningFilled,
        [LogLevelEnum.ERROR]: CircleCloseFilled,
        [LogLevelEnum.FATAL]: CircleCloseFilled,
    };
    return icons[level] || CircleCheck;
};

/** 转义正则特殊字符，防止用户输入导致 RegExp 构造崩溃 */
const escapeRegex = (str: string): string => {
    return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
};

const escapeHtml = (text: string): string => {
    return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
};

/**
 * 搜索关键词高亮
 *
 * 构建灵活正则：关键词每个字符之间允许出现可选分隔符 [_\-]，
 * 使 "syslog" 能高亮原文中的 "sys_log"，"dbtransfer" 能高亮 "db_transfer"。
 */
const highlightText = (text: string): string => {
    const escaped = escapeHtml(text);
    if (!searchKeyword.value) return escaped;
    // 将关键词每个字符之间插入可选分隔符模式：syslog → s[_\-]?y[_\-]?s[_\-]?l[_\-]?o[_\-]?g
    const chars = escapeRegex(searchKeyword.value).split('');
    const flexiblePattern = chars.join('[_\\-]?');
    const regex = new RegExp(`(${flexiblePattern})`, 'gi');
    return escaped.replace(regex, '<mark class="log-viewer__highlight">$1</mark>');
};

const onScroll = () => {
    if (containerRef.value) {
        scrollTop.value = containerRef.value.scrollTop;
    }
};

const onFilterChange = () => {
    nextTick(() => scrollToBottom());
};

const onSearchChange = () => {
    nextTick(() => {
        if (filteredLines.value.length > 0) {
            scrollToIndex(0);
        }
    });
};

const scrollToBottom = () => {
    if (autoScroll.value && containerRef.value) {
        // 直接更新 scrollTop.value 以触发虚拟滚动更新
        scrollTop.value = totalHeight.value - containerHeight.value;
        if (scrollTop.value < 0) {
            scrollTop.value = 0;
        }
        // 同时设置 DOM 的 scrollTop
        nextTick(() => {
            if (containerRef.value) {
                containerRef.value.scrollTop = containerRef.value.scrollHeight;
            }
        });
    }
};

const scrollToIndex = (index: number) => {
    if (containerRef.value) {
        containerRef.value.scrollTop = index * props.lineHeight;
    }
};

// 监听日志变化，自动滚动到底部
watch(
    () => props.lines.length,
    (newLen, oldLen) => {
        if (autoScroll.value) {
            nextTick(scrollToBottom);
        }
        // 首次加载时自动滚动到底部
        if (oldLen === 0 && newLen > 0) {
            nextTick(scrollToBottom);
        }
    }
);

// 监听 finished 状态变化，完成后滚动到底部
watch(
    () => props.finished,
    (finished) => {
        if (finished && autoScroll.value) {
            nextTick(scrollToBottom);
        }
    }
);

// 组件挂载时滚动到底部（处理初始已有数据的场景）
onMounted(() => {
    if (props.lines.length > 0 && autoScroll.value) {
        // 延迟滚动，等待 dialog 动画完成和容器高度计算完成
        setTimeout(() => {
            scrollToBottom();
        }, 100);
    }
});

// 公共方法
const refresh = () => emit('refresh');
const clear = () => emit('clear');
const download = () => emit('download');

// 暴露方法供外部调用
defineExpose({
    scrollToBottom,
    scrollToIndex,
    refresh,
    clear,
});
</script>

<style lang="scss" scoped>
.log-viewer {
    display: flex;
    flex-direction: column;
    height: 100%;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    overflow: hidden;

    &__toolbar {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 8px 12px;
        background: var(--el-fill-color-lighter);
        border-bottom: 1px solid var(--el-border-color-lighter);
        gap: 12px;
    }

    &__filters {
        display: flex;
        align-items: center;
        gap: 12px;
        flex: 1;
    }

    &__search {
        width: 200px;
    }

    &__actions {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    &__match-count {
        font-size: 12px;
        color: var(--el-text-color-secondary);
        padding: 0 8px;
    }

    &__metrics {
        padding: 12px;
        background: var(--el-fill-color-lighter);
        border-bottom: 1px solid var(--el-border-color-lighter);
    }

    &__content {
        flex: 1;
        overflow-y: auto;
        overflow-x: auto;
        position: relative;
    }

    &__list {
        position: relative;
        overflow: hidden;
    }

    &__spacer {
        position: absolute;
        left: 0;
        right: 0;
        top: 0;
    }

    &__line {
        display: flex;
        align-items: center;
        padding: 0 12px;
        border-bottom: 1px solid var(--el-border-color-extra-light);
        font-family: 'JetBrains Mono', 'Consolas', 'Monaco', monospace;
        white-space: nowrap;

        &:hover {
            background: var(--el-fill-color-light);
        }

        &.level-debug {
            color: var(--el-text-color-secondary);
        }

        &.level-info {
            color: var(--el-text-color-primary);
        }

        &.level-warn {
            color: var(--el-color-warning);
            background: var(--el-color-warning-light-9);
        }

        &.level-error {
            color: var(--el-color-danger);
            background: var(--el-color-danger-light-9);
        }

        &.level-fatal {
            color: var(--el-color-danger);
            background: var(--el-color-danger-light-8);
            font-weight: bold;
        }
    }

    &__line-number {
        min-width: 50px;
        padding-right: 12px;
        text-align: right;
        color: var(--el-text-color-placeholder);
        user-select: none;
        font-size: 12px;
    }

    &__timestamp {
        min-width: 120px;
        padding-right: 12px;
        color: var(--el-text-color-secondary);
        font-size: 12px;
    }

    &__level-icon {
        margin-right: 8px;
        font-size: 14px;

        &.level-debug {
            color: var(--el-text-color-secondary);
        }
        &.level-info {
            color: var(--el-color-success);
        }
        &.level-warn {
            color: var(--el-color-warning);
        }
        &.level-error {
            color: var(--el-color-danger);
        }
        &.level-fatal {
            color: var(--el-color-danger);
        }
    }

    &__message {
        flex: 1;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    &__loading {
        display: flex;
        align-items: center;
        justify-content: center;
        height: 200px;
        gap: 8px;
        color: var(--el-text-color-secondary);
    }

    &__statusbar {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 6px 12px;
        background: var(--el-fill-color-lighter);
        border-top: 1px solid var(--el-border-color-lighter);
        font-size: 12px;
        color: var(--el-text-color-secondary);
    }

    &__status {
        display: flex;
        align-items: center;
        gap: 4px;
    }
}

// 级别过滤按钮样式
:deep(.level-debug) {
    --el-button-bg-color: var(--el-fill-color);
    --el-button-border-color: var(--el-border-color);
    --el-button-text-color: var(--el-text-color-regular);
    &.is-active {
        --el-button-bg-color: var(--el-color-info-light-5);
        --el-button-border-color: var(--el-color-info);
        --el-button-text-color: var(--el-color-info);
    }
}

:deep(.level-info) {
    &.is-active {
        --el-button-bg-color: var(--el-color-success-light-5);
        --el-button-border-color: var(--el-color-success);
        --el-button-text-color: var(--el-color-success);
    }
}

:deep(.level-warn) {
    &.is-active {
        --el-button-bg-color: var(--el-color-warning-light-5);
        --el-button-border-color: var(--el-color-warning);
        --el-button-text-color: var(--el-color-warning);
    }
}

:deep(.level-error),
:deep(.level-fatal) {
    &.is-active {
        --el-button-bg-color: var(--el-color-danger-light-5);
        --el-button-border-color: var(--el-color-danger);
        --el-button-text-color: var(--el-color-danger);
    }
}
</style>

<!-- v-html 注入的 <mark> 标签不受 scoped CSS 控制，必须用非 scoped 块 -->
<style lang="scss">
.log-viewer__highlight {
    background: var(--el-color-warning-light-5);
    padding: 0 2px;
    border-radius: 2px;
    color: inherit;
}
</style>
