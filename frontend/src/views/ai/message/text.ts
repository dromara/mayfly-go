/**
 * 文本截断工具（按 Unicode 码点，而非 UTF-16 码元）。
 *
 * 为什么不能直接用 `str.slice(0, n)`：JS 字符串是 UTF-16 码元序列，emoji 等增补平面字符
 * 占两个码元。按码元截断会把代理对劈开，界面上出现半个字符（渲染为 ''）。
 * `Array.from` 按码点迭代，截断后再拼回，保证输出始终是合法字符串。
 */

/** 按码点截断，超出时补省略号；未超出原样返回 */
export function truncateByCodePoints(text: string, max: number): string {
    if (max <= 0) return '';
    const chars = Array.from(text);
    if (chars.length <= max) return text;
    return chars.slice(0, max).join('') + '\u2026';
}
