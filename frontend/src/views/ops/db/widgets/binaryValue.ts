/**
 * 二进制列值的呈现辅助（hex ↔ 字节 ↔ base64/UTF-8）。
 *
 * 后端在迁移/查询链路里把二进制列回读成 **hex 字符串**（见 dbi.ValuerBytes），所以单元格拿到的
 * 就是那串 hex。这里只把它换算成更好读的形态，不写回任何新语义：编辑仍按 hex 提交。
 *
 * 判据集中在 BINARY_BASE_TYPES 一处：新增方言的二进制类型是往集合里登记一条，
 * 而不是在渲染分支里按字符串前缀各猜一遍。
 */

/** 各已支持方言的二进制/大对象类型基础名（已去掉长度参数与大小写差异） */
const BINARY_BASE_TYPES = new Set([
    'blob',
    'tinyblob',
    'mediumblob',
    'longblob',
    'binary',
    'varbinary',
    'bytea',
    'image',
    'raw',
    'long raw',
    'longbinary',
    'bfile',
]);

/**
 * 该列类型是否二进制。
 *
 * 刻意不匹配 FixedString/String 这类「可能被用来存字节」的类型：判成二进制就会把一个正常
 * 文本列的编辑入口换成只读预览，误伤比缺功能更糟。
 */
export function isBinaryColumnType(columnType?: string): boolean {
    if (!columnType) {
        return false;
    }
    const base = columnType
        .toLowerCase()
        .replace(/\([^)]*\)/, '')
        .trim();
    // pg 的数组形态只认 bytea 数组（_bytea / bytea[]），int[] 之类不是二进制
    if (base.startsWith('_')) {
        return base === '_bytea';
    }
    if (base.endsWith('[]')) {
        return base === 'bytea[]';
    }
    return BINARY_BASE_TYPES.has(base);
}

/** hex 字符串 → 字节；形态不是 hex（奇数长度/含非 hex 字符）时返回 null，由调用方如实说明 */
export function hexToBytes(hex: string): Uint8Array | null {
    if (!hex || hex.length % 2 !== 0) {
        return null;
    }
    const bytes = new Uint8Array(hex.length / 2);
    for (let i = 0; i < bytes.length; i++) {
        const byte = Number.parseInt(hex.slice(i * 2, i * 2 + 2), 16);
        if (Number.isNaN(byte)) {
            return null;
        }
        bytes[i] = byte;
    }
    return bytes;
}

/** 字节 → base64（分块转换：一次性 apply 大数组会触发栈溢出） */
export function bytesToBase64(bytes: Uint8Array): string {
    const chunk = 0x8000;
    let binary = '';
    for (let i = 0; i < bytes.length; i += chunk) {
        binary += String.fromCharCode(...bytes.subarray(i, i + chunk));
    }
    return btoa(binary);
}

/** UTF-8 预览的字节上限：只影响读，不影响取值 */
const UTF8_PREVIEW_LIMIT = 4096;

export interface Utf8Preview {
    text: string;
    /** 内容超出预览上限，后面还有未显示的部分 */
    truncated: boolean;
    /** 可打印字符占比（0~1），供界面提示「这大概率不是文本」 */
    printableRatio: number;
}

/**
 * 按 UTF-8 解读字节做预览。
 *
 * 明确标注为「猜测性解读」：二进制列没有编码声明，非法字节会被替换成 U+FFFD，
 * 因此它只能当参考，不能当成「这就是原文」的结论。
 */
export function utf8Preview(bytes: Uint8Array): Utf8Preview {
    const truncated = bytes.length > UTF8_PREVIEW_LIMIT;
    const slice = truncated ? bytes.subarray(0, UTF8_PREVIEW_LIMIT) : bytes;
    const text = new TextDecoder('utf-8', { fatal: false }).decode(slice);
    const total = [...text];
    const printable = total.filter((ch) => ch === '\n' || ch === '\t' || ch === '\r' || (ch >= ' ' && ch !== '\uFFFD')).length;
    return {
        text,
        truncated,
        printableRatio: total.length ? printable / total.length : 1,
    };
}
