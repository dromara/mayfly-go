import dayjs from 'dayjs';

/**
 * 格式化日期
 * @param date 日期 字符串 Date 时间戳等
 * @param format 格式化格式  默认 YYYY-MM-DD HH:mm:ss
 * @returns 格式化后内容
 */
export function formatDate(date: any, format: string = 'YYYY-MM-DD HH:mm:ss') {
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
    const units = {
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
export function formatAxis(param: any) {
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
