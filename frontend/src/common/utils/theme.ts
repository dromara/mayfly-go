import { ElMessage } from 'element-plus';

// hex颜色转rgb颜色
export function hexToRgb(str: any) {
    let hexs: any = '';
    let reg = /^\#?[0-9A-Fa-f]{6}$/;
    if (!reg.test(str)) return ElMessage({ type: 'warning', message: '输入错误的hex' });
    str = str.replace('#', '');
    hexs = str.match(/../g);
    for (let i = 0; i < 3; i++) hexs[i] = parseInt(hexs[i], 16);
    return hexs;
}

// rgb颜色转Hex颜色
export function rgbToHex(r: any, g: any, b: any) {
    let reg = /^\d{1,3}$/;
    if (!reg.test(r) || !reg.test(g) || !reg.test(b)) return ElMessage({ type: 'warning', message: '输入错误的rgb颜色值' });
    let hexs = [r.toString(16), g.toString(16), b.toString(16)];
    for (let i = 0; i < 3; i++) if (hexs[i].length == 1) hexs[i] = `0${hexs[i]}`;
    return `#${hexs.join('')}`;
}

// 加深颜色值，level为加深的程度，限0-1之间
export function getDarkColor(color: any, level: number) {
    let reg = /^\#?[0-9A-Fa-f]{6}$/;
    if (!reg.test(color)) return ElMessage({ type: 'warning', message: '输入错误的hex颜色值' });
    let rgb = hexToRgb(color);
    for (let i = 0; i < 3; i++) rgb[i] = Math.floor(rgb[i] * (1 - level));
    return rgbToHex(rgb[0], rgb[1], rgb[2]);
}

// 变浅颜色值，level为加深的程度，限0-1之间
export function getLightColor(color: any, level: number) {
    let reg = /^\#?[0-9A-Fa-f]{6}$/;
    if (!reg.test(color)) return ElMessage({ type: 'warning', message: '输入错误的hex颜色值' });
    let rgb = hexToRgb(color);
    for (let i = 0; i < 3; i++) rgb[i] = Math.floor((255 - rgb[i]) * level + rgb[i]);
    return rgbToHex(rgb[0], rgb[1], rgb[2]);
}
