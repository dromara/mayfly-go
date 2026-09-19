/**
 * 内置日志解析器（策略模式实现）
 *
 * 提供常见日志格式的解析器，业务模块可直接使用或继承扩展。
 * 符合开闭原则：新增格式无需修改现有代码，只需实现 LogParser 接口。
 */

import type { LogParser, ParsedLogLine } from './types';
import { LogLevel } from './types';

/**
 * 通用时间戳+级别解析器
 *
 * 支持格式：
 * - [2024-01-01 12:00:00] INFO message
 * - [2024-01-01 12:00:00.123] WARN message
 * - [12:00:00] ERROR message
 * - [12:00:00.123] DEBUG message
 */
export class TimestampLevelParser implements LogParser {
    // 匹配 [timestamp] LEVEL message 或 [timestamp] message
    private static readonly PATTERN = /^\[(\d{1,4}[-/]\d{1,2}[-/]\d{1,2}\s+\d{1,2}:\d{1,2}:\d{1,2}(?:\.\d{1,3})?|\d{1,2}:\d{1,2}:\d{1,2}(?:\.\d{1,3})?)\]\s*(?:(DEBUG|INFO|WARN|WARNING|ERROR|FATAL)\s+)?(.*)$/i;

    parse(line: string, lineIndex: number): ParsedLogLine {
        const match = line.match(TimestampLevelParser.PATTERN);

        if (match) {
            const [, timestamp, levelStr, message] = match;
            return {
                raw: line,
                timestamp: timestamp.trim(),
                level: this.normalizeLevel(levelStr),
                message: message || '',
                lineIndex,
            };
        }

        // 无结构化匹配时，尝试从内容推断级别
        return {
            raw: line,
            level: this.inferLevel(line),
            message: line,
            lineIndex,
        };
    }

    parseAll(text: string, startIndex: number): ParsedLogLine[] {
        const lines = text.split('\n');
        return lines.map((line, i) => this.parse(line, startIndex + i));
    }

    private normalizeLevel(levelStr?: string): LogLevel {
        if (!levelStr) return LogLevel.INFO;
        const upper = levelStr.toUpperCase();
        if (upper === 'WARNING') return LogLevel.WARN;
        return (Object.values(LogLevel) as string[]).includes(upper) ? (upper as LogLevel) : LogLevel.INFO;
    }

    /** 从内容推断日志级别（用于无显式级别的日志） */
    private inferLevel(line: string): LogLevel {
        const lower = line.toLowerCase();
        if (lower.includes('error') || lower.includes('fail') || lower.includes('exception')) {
            return LogLevel.ERROR;
        }
        if (lower.includes('warn')) {
            return LogLevel.WARN;
        }
        if (lower.includes('debug')) {
            return LogLevel.DEBUG;
        }
        return LogLevel.INFO;
    }
}

/**
 * 纯文本解析器（无结构化信息）
 *
 * 用于简单日志格式，仅按行分割，级别默认为 INFO。
 */
export class PlainTextParser implements LogParser {
    parse(line: string, lineIndex: number): ParsedLogLine {
        return {
            raw: line,
            level: LogLevel.INFO,
            message: line,
            lineIndex,
        };
    }

    parseAll(text: string, startIndex: number): ParsedLogLine[] {
        return text.split('\n').map((line, i) => this.parse(line, startIndex + i));
    }
}

/**
 * 同步运行日志解析器
 *
 * 专门解析 mayfly-go 同步任务的 RunLog 格式：
 * [HH:MM:SS.mmm] 消息内容
 */
export class SyncRunLogParser implements LogParser {
    private static readonly PATTERN = /^\[(\d{1,2}:\d{1,2}:\d{1,2}(?:\.\d{1,3})?)\]\s*(.*)$/;

    parse(line: string, lineIndex: number): ParsedLogLine {
        const match = line.match(SyncRunLogParser.PATTERN);

        if (match) {
            const [, timestamp, message] = match;
            return {
                raw: line,
                timestamp,
                level: this.inferLevel(message),
                message,
                lineIndex,
            };
        }

        return {
            raw: line,
            level: this.inferLevel(line),
            message: line,
            lineIndex,
        };
    }

    parseAll(text: string, startIndex: number): ParsedLogLine[] {
        return text
            .split('\n')
            .filter((line) => line.trim().length > 0) // 过滤空行
            .map((line, i) => this.parse(line, startIndex + i));
    }

    /** 从消息内容推断级别 */
    private inferLevel(message: string): LogLevel {
        // 失败/错误相关
        if (message.includes('失败') || message.includes('错误') || message.includes('failed') || message.includes('error')) {
            return LogLevel.ERROR;
        }
        // 警告相关（排除统计摘要中的"跳过行数"）
        if (message.includes('警告') || message.includes('warn')) {
            return LogLevel.WARN;
        }
        // 只有实际描述跳过动作时才当作警告（如"跳过某行"），统计摘要不算
        if (message.includes('跳过') && !message.includes('行数') && !message.includes('：') && !message.includes(':')) {
            return LogLevel.WARN;
        }
        // 成功/完成相关
        if (message.includes('成功') || message.includes('完成') || message.includes('通过')) {
            return LogLevel.INFO;
        }
        // 开始/进行中
        if (message.includes('开始') || message.includes('执行') || message.includes('批次')) {
            return LogLevel.INFO;
        }
        return LogLevel.INFO;
    }
}
