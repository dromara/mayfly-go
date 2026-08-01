/**
 * guac/clipboard.js 类型声明：本地与远程剪贴板同步工具。
 */
interface GuacClipboard {
    install: (client: any) => boolean;
    update: (client: any) => () => void;
    sendRemoteClipboard: (client: any, text: string) => void;
    setRemoteClipboard: (client: any) => void;
    getLocalClipboard: () => Promise<{ type: string; data: string } | undefined>;
    setLocalClipboard: (data: { type: string; data: unknown }) => Promise<void>;
    onClipboard: (stream: any, mimetype: string) => void;
    installWatcher: (clipboardList: unknown[], setClipboardFn: (data: string) => void) => void;
    appendClipboardList: (src: string, data: string) => void;
    cache?: { type: string; data: unknown };
    clipboardList?: unknown[];
    setClipboardFn?: (data: string) => void;
}

declare const clipboard: GuacClipboard;
export default clipboard;
