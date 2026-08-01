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
}
