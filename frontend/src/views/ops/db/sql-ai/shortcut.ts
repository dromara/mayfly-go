/**
 * 浮层文案里的快捷键前缀。
 *
 * 单独一个纯函数而不是在组件里读 navigator：界面提示要写「⌘K」还是「Ctrl+K」取决于宿主平台，
 * 而组件里直接取 navigator 既没法在 jsdom 之外断言，也会让同一条提示在不同组件里写成两种样式。
 * 判据收在这里，调用点传入 UA 字符串即可被表驱动单测钉住。
 */

/** mac 系平台（含 iOS/iPadOS UA）用 ⌘，其余用 Ctrl */
export function modKeyLabel(userAgent: string): string {
    return /Mac|iPhone|iPad|iPod/i.test(userAgent) ? '⌘' : 'Ctrl';
}

/** 当前浏览器的快捷键前缀 */
export function currentModKeyLabel(): string {
    // SSR / 测试环境下可能没有 navigator，退化成最常见的 Ctrl 写法
    return modKeyLabel(typeof navigator === 'undefined' ? '' : navigator.userAgent);
}
