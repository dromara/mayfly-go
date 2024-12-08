export interface TerminalExpose {
    /** 连接 */
    init(width: number, height: number, force: boolean): void;

    /** 短开连接 */
    close(): void;

    blur(): void;

    focus(): void;
}
