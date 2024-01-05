export function dateFormat2(fmt: string, date: Date) {
    let ret;
    const opt = {
        'y+': date.getFullYear().toString(), // 年
        'M+': (date.getMonth() + 1).toString(), // 月
        'd+': date.getDate().toString(), // 日
        'H+': date.getHours().toString(), // 时
        'm+': date.getMinutes().toString(), // 分
        's+': date.getSeconds().toString(), // 秒
        'S+': date.getMilliseconds() ? date.getMilliseconds().toString() : '', // 毫秒
        // 有其他格式化字符需求可以继续添加，必须转化成字符串
    };
    for (const k in opt) {
        ret = new RegExp('(' + k + ')').exec(fmt);
        if (ret) {
            fmt = fmt.replace(ret[1], ret[1].length == 1 ? opt[k] : opt[k].padStart(ret[1].length, '0'));
        }
    }
    return fmt;
}

export function dateStrFormat(fmt: string, dateStr: string) {
    return dateFormat2(fmt, new Date(dateStr));
}

export function dateFormat(dateStr: string) {
    if (!dateStr) {
        return '';
    }
    return dateFormat2('yyyy-MM-dd HH:mm:ss', new Date(dateStr));
}
