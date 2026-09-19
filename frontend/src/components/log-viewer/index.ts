/**
 * 通用日志查看器模块
 *
 * 导出所有公共 API，供业务模块使用。
 *
 * @example
 * ```vue
 * <template>
 *   <LogViewer :lines="lines" :loading="loading" :finished="finished" />
 * </template>
 *
 * <script setup>
 * import { LogViewer, useLogStream, TimestampLevelParser } from '@/components/log-viewer';
 *
 * const { lines, loading, finished } = useLogStream({
 *   source: { pollingInterval: 2000, fetcher: fetchLog },
 *   parser: new TimestampLevelParser(),
 * });
 * </script>
 * ```
 */

// 核心组件
export { default as LogViewer } from './LogViewer.vue';

// 组合式函数
export { useLogStream, type UseLogStreamOptions, type UseLogStreamReturn } from './useLogStream';

// 类型定义
export { LogLevel } from './types';
export type { LogParser, ParsedLogLine, LogSourceConfig, LogFilterConfig, LogViewerTheme, MetricItem } from './types';

// 内置解析器
export { TimestampLevelParser, PlainTextParser, SyncRunLogParser } from './parsers';
