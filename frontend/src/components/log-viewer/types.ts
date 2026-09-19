/**
 * 通用日志查看器类型定义
 *
 * 设计原则：
 * - 数据源无关：支持轮询、WebSocket、静态数据等多种数据源
 * - 格式无关：通过 LogParser 策略注入解析逻辑
 * - 展示可配：通过 props/slots 自定义展示
 */

/** 日志级别 */
export enum LogLevel {
    DEBUG = 'DEBUG',
    INFO = 'INFO',
    WARN = 'WARN',
    ERROR = 'ERROR',
    FATAL = 'FATAL',
}

/** 解析后的单条日志 */
export interface ParsedLogLine {
    /** 原始文本 */
    raw: string;
    /** 时间戳（如有） */
    timestamp?: string;
    /** 日志级别 */
    level: LogLevel;
    /** 消息内容（去除时间戳和级别后的纯文本） */
    message: string;
    /** 行号（用于增量加载） */
    lineIndex: number;
}

/**
 * 日志解析器策略接口（开闭原则核心）
 *
 * 不同模块的日志格式各异，通过实现此接口注入自定义解析逻辑：
 * - 迁移日志：[2024-01-01 12:00:00] INFO message
 * - 同步日志：[13:56:22.130] message
 * - 系统日志：纯文本无结构化
 */
export interface LogParser {
    /** 解析单行日志文本为结构化数据 */
    parse(line: string, lineIndex: number): ParsedLogLine;
    /** 批量解析（默认逐行调用 parse，可覆写优化性能） */
    parseAll?(text: string, startIndex: number): ParsedLogLine[];
}

/** 日志数据源配置 */
export interface LogSourceConfig {
    /** 轮询间隔（ms），0 表示不轮询 */
    pollingInterval?: number;
    /** 获取日志的异步函数，返回增量文本 */
    fetcher: (lastLineIndex: number) => Promise<string>;
    /** 日志是否已结束（不再有新内容） */
    finished?: () => boolean;
}

/** 日志级别过滤配置 */
export interface LogFilterConfig {
    /** 显示的日志级别，空数组表示全部显示 */
    levels: LogLevel[];
    /** 搜索关键词 */
    keyword: string;
}

/** 日志查看器主题配置 */
export interface LogViewerTheme {
    /** 背景色 */
    background?: string;
    /** 字体大小 */
    fontSize?: string;
    /** 行高 */
    lineHeight?: string;
    /** 是否显示行号 */
    showLineNumbers?: boolean;
    /** 是否显示时间戳 */
    showTimestamp?: boolean;
    /** 是否显示级别图标 */
    showLevelIcon?: boolean;
}

/** 指标项定义 */
export interface MetricItem {
    /** 指标标签 */
    label: string;
    /** 指标值 */
    value: string | number;
    /** 单位（可选） */
    unit?: string;
    /** 状态（用于颜色标记） */
    status?: 'success' | 'warning' | 'error' | 'info';
}
