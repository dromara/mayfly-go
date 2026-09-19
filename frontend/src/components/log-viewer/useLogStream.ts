/**
 * 日志流数据管理组合式函数
 *
 * 职责：
 * - 数据源适配：支持轮询、手动刷新、静态数据
 * - 增量加载：只获取新增日志行
 * - 解析调度：调用注入的 LogParser 解析原始文本
 * - 状态管理：加载状态、错误状态、完成状态
 *
 * 设计原则：
 * - 单一职责：只管数据流，不管 UI 展示
 * - 依赖注入：Parser 和 Source 通过参数注入
 * - 响应式：返回 Vue 响应式状态
 */

import { useIntervalFn, whenever } from '@vueuse/core';
import { computed, ref, type Ref } from 'vue';

import type { LogParser, ParsedLogLine, LogSourceConfig } from './types';
import { PlainTextParser } from './parsers';

export interface UseLogStreamOptions {
    /** 日志数据源配置 */
    source: LogSourceConfig;
    /** 日志解析器（策略模式），默认 PlainTextParser */
    parser?: LogParser;
    /** 是否立即开始加载，默认 true */
    immediate?: boolean;
    /** 最大保留行数（防止内存溢出），默认 10000 */
    maxLines?: number;
}

export interface UseLogStreamReturn {
    /** 解析后的日志行（响应式） */
    lines: Ref<ParsedLogLine[]>;
    /** 是否正在加载 */
    loading: Ref<boolean>;
    /** 是否已加载完成（数据源结束） */
    finished: Ref<boolean>;
    /** 最后错误信息 */
    error: Ref<string | null>;
    /** 总行数 */
    totalLines: Ref<number>;
    /** 手动刷新（获取增量） */
    refresh: () => Promise<void>;
    /** 清空日志 */
    clear: () => void;
    /** 暂停轮询 */
    pause: () => void;
    /** 恢复轮询 */
    resume: () => void;
    /** 是否暂停中 */
    isPaused: Ref<boolean>;
}

/**
 * 日志流数据管理
 *
 * @example
 * ```ts
 * // 轮询模式
 * const { lines, loading, refresh } = useLogStream({
 *   source: {
 *     pollingInterval: 2000,
 *     fetcher: async (lastLine) => {
 *       const res = await logApi.detail({ id: logId });
 *       return res.resp;
 *     },
 *     finished: () => log.value?.type !== LogTypeEnum.Running.value,
 *   },
 *   parser: new TimestampLevelParser(),
 * });
 *
 * // 静态模式（一次性加载）
 * const { lines } = useLogStream({
 *   source: {
 *     fetcher: async () => staticLogText,
 *   },
 *   parser: new SyncRunLogParser(),
 * });
 * ```
 */
export function useLogStream(options: UseLogStreamOptions): UseLogStreamReturn {
    const { source, immediate = true, maxLines = 10000 } = options;
    const parser = options.parser || new PlainTextParser();

    // 是否启用轮询（仅当 pollingInterval > 0 时启用）
    const isPollingMode = (source.pollingInterval ?? 0) > 0;

    // 状态
    const lines = ref<ParsedLogLine[]>([]);
    const loading = ref(false);
    const finished = ref(false);
    const error = ref<string | null>(null);
    const isPaused = ref(false);

    // 行计数器（用于增量加载）
    let lastLineIndex = 0;

    // 计算属性
    const totalLines = computed(() => lines.value.length);

    /** 获取并解析增量日志 */
    const fetchAndParse = async () => {
        if (loading.value || finished.value) return;

        loading.value = true;
        error.value = null;

        try {
            // 调用数据源获取增量文本
            const newText = await source.fetcher(lastLineIndex);

            if (!newText) {
                // 无新内容
                if (source.finished?.()) {
                    finished.value = true;
                }
                return;
            }

            // 解析新文本
            const newLines = parser.parseAll
                ? parser.parseAll(newText, lastLineIndex)
                : newText.split('\n').map((line, i) => parser.parse(line, lastLineIndex + i));

            // 追加到日志列表
            lines.value = [...lines.value, ...newLines];

            // 更新行计数器
            lastLineIndex += newLines.length;

            // 防止内存溢出：保留最新 N 行
            if (lines.value.length > maxLines) {
                lines.value = lines.value.slice(-maxLines);
            }

            // 检查是否已结束
            if (source.finished?.()) {
                finished.value = true;
            }
        } catch (e) {
            error.value = e instanceof Error ? e.message : String(e);
        } finally {
            loading.value = false;
        }
    };

    // 轮询控制（仅在轮询模式下启用，避免 interval=0 导致疯狂触发）
    const { pause: pauseInterval, resume: resumeInterval } = useIntervalFn(
        () => {
            if (!isPaused.value) {
                fetchAndParse();
            }
        },
        source.pollingInterval ?? 1000,
        { immediate: false }
    );

    // 手动刷新
    const refresh = async () => {
        await fetchAndParse();
    };

    // 清空日志
    const clear = () => {
        lines.value = [];
        lastLineIndex = 0;
        finished.value = false;
        error.value = null;
    };

    // 监听 finished 状态，自动停止轮询
    whenever(finished, () => {
        if (isPollingMode) {
            pauseInterval();
        }
    });

    // 立即加载
    if (immediate) {
        if (isPollingMode) {
            resumeInterval();
        }
        fetchAndParse();
    }

    return {
        lines,
        loading,
        finished,
        error,
        totalLines,
        refresh,
        clear,
        pause: () => {
            if (isPollingMode) {
                pauseInterval();
            }
            isPaused.value = true;
        },
        resume: () => {
            isPaused.value = false;
            if (isPollingMode) {
                resumeInterval();
            }
        },
        isPaused,
    };
}
