import dayjs from 'dayjs';

/**
 * 格式化日期
 * @param date 日期 字符串 Date 时间戳等
 * @param format 格式化格式  默认 YYYY-MM-DD HH:mm:ss
 * @returns 格式化后内容
 */
export function formatDate(date?: string | number | Date, format: string = 'YYYY-MM-DD HH:mm:ss') {
    if (!date) {
        return '';
    }
    return dayjs(date).format(format);
}

/**
 * 格式化字节单位
 * @param size byte size
 * @returns
 */
export function formatByteSize(size: number, fixed = 2) {
    if (size === 0) {
        return '0B';
    }

    const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
    const base = 1024;
    const exponent = Math.floor(Math.log(size) / Math.log(base));

    return parseFloat((size / Math.pow(base, exponent)).toFixed(fixed)) + units[exponent];
}

export function formatDocSize(size: number, fixed = 2) {
    if (size === 0) {
        return '0';
    }

    const units = ['', 'K', 'M', 'G', 'T', 'P', 'E', 'Z', 'Y'];
    const base = 1000;
    const exponent = Math.floor(Math.log(size) / Math.log(base));

    return parseFloat((size / Math.pow(base, exponent)).toFixed(fixed)) + units[exponent];
}

/**
 * 计数千分位精确展示（详情/摘要卡用）：token 数、调用次数等需要读出精确值的场景
 * @example 91606 -> '91,606'
 */
export function formatThousands(val?: number | null): string {
    if (val === undefined || val === null || Number.isNaN(val)) {
        return '-';
    }
    return val.toLocaleString('en-US');
}

/**
 * 计数缩放的单一实现点：单位表 + 取整去尾零只有一份，所有「按量级缩写」的口径都仍从这里走。
 *
 * 为什么抽出来：单位表一旦被多个入口各自抄一份，就会出现「一处改了 K/M/B 另一处没改」
 * 的真源分叉；而小数位策略确实是不同读数需要的，所以只把这一项参数化（pickDigits）
 */
function compactWithDigits(val: number, pickDigits: (scaled: number) => number): string {
    const units = [
        { limit: 1e9, suffix: 'B' },
        { limit: 1e6, suffix: 'M' },
        { limit: 1e3, suffix: 'K' },
    ];
    for (const { limit, suffix } of units) {
        if (val >= limit) {
            const scaled = val / limit;
            return parseFloat(scaled.toFixed(pickDigits(scaled))).toString() + suffix;
        }
    }
    return formatThousands(val);
}

/**
 * token 数量的**唯一展示入口**：所有界面看到 token 都走它，不得自己挑 formatThousands。
 *
 * （原先还有一个通用导出 `formatCountCompact`，在 token 全部收编后零消费者，已删除：
 * 留着一个导出但禁止使用就是双重死代码，还会诱导后来者把新的计数展示接到它上面）
 *
 * 为什么需要单独一个入口：token 量级天然很大（一次编排动辄上百万），裸千分位在概览位上根本读不出
 * 量级；而「列表用紧凑、详情用精确」这个挑法一旦散到各页面，每个页面都会自己决定用哪个——
 * 治理看板的「Token 消耗」就是这么变成 1,957,362 这种不可读数值的。收成一个函数后新增展示位无需再判断
 *
 * 精度取**3 位有效数字**而不是固定小数位：固定 2 位小数会让同一屏出现
 * 「20.73K」与「29.6K」两种位数（看着像两种口径），而 3 位有效数字在任何量级下都是
 * 「20.7K / 29.6K / 2.29M」这样对齐的形式，也更接近成本读数的惯常精度
 *
 * @param withExact 为 true 时在紧凑值后附精确值（概览卡/详情用：一眼看出量级，也能读到准确数做成本对账）；
 *                  列表列不传（列宽有限，紧凑值已足够定位量级，精确值可在详情抽屉看）
 * @example 2292327 -> '2.29M'；(2292327, true) -> '2.29M (2,292,327)'；20734 -> '20.7K'；8234 -> '8,234'
 */
export function formatTokenCount(val?: number | null, withExact = false): string {
    if (val === undefined || val === null || Number.isNaN(val)) {
        return '-';
    }
    const compact = val < 10000 ? formatThousands(val) : compactWithDigits(val, (scaled) => (scaled >= 100 ? 0 : scaled >= 10 ? 1 : 2));
    if (!withExact) {
        return compact;
    }
    const exact = formatThousands(val);
    // 小于万位时 compact 已等于精确值，不重复输出成「8,234 (8,234)」
    return compact === exact ? compact : `${compact} (${exact})`;
}

/**
 * 毫秒耗时人类可读展示（<1s 用 ms，1 分钟内用秒，跨分显示「Xm Ys」），
 * 替代裸的「87902 ms」直出
 * @example 876 -> '876ms'；8200 -> '8.2s'；87902 -> '1m 28s'
 */
export function formatDurationMs(ms?: number | null): string {
    if (ms === undefined || ms === null || Number.isNaN(ms)) {
        return '-';
    }
    if (ms < 1000) {
        return `${Math.round(ms)}ms`;
    }
    if (ms < 60_000) {
        return `${parseFloat((ms / 1000).toFixed(1))}s`;
    }
    const totalSeconds = Math.round(ms / 1000);
    return `${Math.floor(totalSeconds / 60)}m ${totalSeconds % 60}s`;
}

/**
 * 容量转为对应的字节大小，如 1KB转为 1024
 * @param sizeString  1kb 1gb等
 * @returns
 */
export function convertToBytes(sizeStr: string) {
    sizeStr = sizeStr.trim();
    const unit = sizeStr.slice(-2);

    const valueStr = sizeStr.slice(0, -2);
    const value = parseInt(valueStr, 10);

    let bytes = 0;

    switch (unit.toUpperCase()) {
        case 'KB':
            bytes = value * 1024;
            break;
        case 'MB':
            bytes = value * 1024 * 1024;
            break;
        case 'GB':
            bytes = value * 1024 * 1024 * 1024;
            break;
        default:
            throw new Error('Invalid size unit');
    }

    return bytes;
}

/**
 * 格式化指定时间数为人性化可阅读的内容(默认time为秒单位)
 *
 * @param time 时间数
 * @param unit time对应的单位
 * @returns
 */
export function formatTime(time: number, unit: string = 's') {
    const units: Record<string, number> = {
        y: 31536000,
        M: 2592000,
        d: 86400,
        h: 3600,
        m: 60,
        s: 1,
    };

    if (!units[unit]) {
        return 'Invalid unit';
    }

    let seconds = time * units[unit];
    let result = '';

    const timeUnits = Object.entries(units).map(([unit, duration]) => {
        const value = Math.floor(seconds / duration);
        seconds %= duration;
        return { value, unit };
    });

    timeUnits.forEach(({ value, unit }) => {
        if (value > 0) {
            result += `${value}${unit} `;
        }
    });

    return result;
}

/**
 * formatAxis(new Date())   // 上午好
 */
export function formatAxis(param: string | number | Date) {
    let hour: number = new Date(param).getHours();
    if (hour < 6) return '凌晨好';
    else if (hour < 9) return '早上好';
    else if (hour < 12) return '上午好';
    else if (hour < 14) return '中午好';
    else if (hour < 17) return '下午好';
    else if (hour < 19) return '傍晚好';
    else if (hour < 22) return '晚上好';
    else return '夜里好';
}

/**
 * 格式化数据为美观的 JSON 字符串
 *
 * - 如果输入是对象，直接格式化为缩进 JSON
 * - 如果输入是 JSON 字符串，先解析为对象再格式化
 * - 如果解析失败，返回原始值的字符串形式
 * - 如果输入为空值，返回空字符串
 *
 * @param val - 要格式化的数据（对象或 JSON 字符串）
 * @returns 格式化后的 JSON 字符串，带 2 空格缩进
 *
 * @example
 * ```ts
 * // 格式化对象
 * formatJson({ name: 'test', value: 123 })
 * // 输出: '{\n  "name": "test",\n  "value": 123\n}'
 *
 * // 格式化 JSON 字符串
 * formatJson('{"name":"test"}')
 * // 输出: '{\n  "name": "test"\n}'
 *
 * // 处理无效输入
 * formatJson(null)
 * // 输出: ''
 * ```
 */
export function formatJson(val: unknown) {
    if (!val) {
        return '';
    }

    try {
        // 如果val是字符串，尝试解析为对象后再格式化
        let data = val;
        if (typeof val === 'string') {
            data = JSON.parse(val);
        }
        return JSON.stringify(data, null, 2);
    } catch {
        // 如果解析失败，直接返回原始值的字符串形式
        return String(val);
    }
}
