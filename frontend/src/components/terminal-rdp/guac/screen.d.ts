/**
 * guac/screen.js 类型声明：全屏切换工具函数。
 */
export declare function launchIntoFullscreen(element: HTMLElement | null): void;
export declare function exitFullscreen(): void;
export declare function watchFullscreenChange(callback: (event: Event, isFull: boolean) => void): void;
export declare function unWatchFullscreenChange(callback: (event: Event, isFull: boolean) => void): void;
