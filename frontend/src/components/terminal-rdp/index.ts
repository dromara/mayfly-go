import { TerminalStatus } from '@/components/terminal/common';

export interface TerminalExpose {
    /** 连接 */
    connect(width?: number, height?: number, force?: boolean): void;

    /** 断开连接 */
    disconnect(): void;

    /** 连接（connect 别名） */
    init(width?: number, height?: number, force?: boolean): void;

    /** 断开连接（disconnect 别名） */
    close(): void;

    /** 自适应尺寸 */
    fitTerminal(): void;

    blur(): void;

    focus(): void;

    /** 设置远程剪贴板 */
    setRemoteClipboard?: (val: string) => void;

    /** 获取当前连接状态（多窗格容器聚合状态时使用） */
    getStatus?: () => TerminalStatus;
}
