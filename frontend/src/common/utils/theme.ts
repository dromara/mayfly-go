import { ElMessage } from 'element-plus';

// hex颜色转rgb颜色
export function hexToRgb(str: string): number[] {
    const reg = /^\#?[0-9A-Fa-f]{6}$/;
    if (!reg.test(str)) {
        ElMessage({ type: 'warning', message: '输入错误的hex' });
        return [0, 0, 0];
    }
    const matched = str.replace('#', '').match(/../g)!;
    return [parseInt(matched[0], 16), parseInt(matched[1], 16), parseInt(matched[2], 16)];
}

// rgb颜色转Hex颜色
export function rgbToHex(r: number, g: number, b: number): string {
    const reg = /^\d{1,3}$/;
    if (!reg.test(String(r)) || !reg.test(String(g)) || !reg.test(String(b))) {
        ElMessage({ type: 'warning', message: '输入错误的rgb颜色值' });
        return '#000000';
    }
    const hexs = [r.toString(16), g.toString(16), b.toString(16)];
    for (let i = 0; i < 3; i++) if (hexs[i].length == 1) hexs[i] = `0${hexs[i]}`;
    return `#${hexs.join('')}`;
}

// 加深颜色值，level为加深的程度，限0-1之间
export function getDarkColor(color: string, level: number): string {
    const reg = /^\#?[0-9A-Fa-f]{6}$/;
    if (!reg.test(color)) return ElMessage({ type: 'warning', message: '输入错误的hex颜色值' }) as unknown as string;
    const rgb = hexToRgb(color);
    for (let i = 0; i < 3; i++) rgb[i] = Math.floor(rgb[i] * (1 - level));
    return rgbToHex(rgb[0], rgb[1], rgb[2]);
}

// 变浅颜色值，level为加深的程度，限0-1之间
export function getLightColor(color: string, level: number): string {
    const reg = /^\#?[0-9A-Fa-f]{6}$/;
    if (!reg.test(color)) return ElMessage({ type: 'warning', message: '输入错误的hex颜色值' }) as unknown as string;
    const rgb = hexToRgb(color);
    for (let i = 0; i < 3; i++) rgb[i] = Math.floor((255 - rgb[i]) * level + rgb[i]);
    return rgbToHex(rgb[0], rgb[1], rgb[2]);
}
