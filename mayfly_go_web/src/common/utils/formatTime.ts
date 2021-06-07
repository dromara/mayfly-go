/*
 * 年(Y) 可用1-4个占位符
 * 月(m)、日(d)、小时(H)、分(M)、秒(S) 可用1-2个占位符
 * 星期(W) 可用1-3个占位符
 * 季度(q为阿拉伯数字，Q为中文数字)可用1或4个占位符
 *
 * let date = new Date()
 * formatDate(date, "YYYY-mm-dd HH:MM:SS")           // 2020-02-09 14:04:23
 * formatDate(date, "YYYY-mm-dd HH:MM:SS Q")         // 2020-02-09 14:09:03 一
 * formatDate(date, "YYYY-mm-dd HH:MM:SS WWW")       // 2020-02-09 14:45:12 星期日
 * formatDate(date, "YYYY-mm-dd HH:MM:SS QQQQ")      // 2020-02-09 14:09:36 第一季度
 * formatDate(date, "YYYY-mm-dd HH:MM:SS WWW QQQQ")  // 2020-02-09 14:46:12 星期日 第一季度
 */
export function formatDate(date: Date, format: string) {
    let we = date.getDay(); // 星期
    let qut = Math.floor((date.getMonth() + 3) / 3).toString(); // 季度
    const opt: any = {
        'Y+': date.getFullYear().toString(), // 年
        'm+': (date.getMonth() + 1).toString(), // 月(月份从0开始，要+1)
        'd+': date.getDate().toString(), // 日
        'H+': date.getHours().toString(), // 时
        'M+': date.getMinutes().toString(), // 分
        'S+': date.getSeconds().toString(), // 秒
        'q+': qut, // 季度
    };
    // 中文数字 (星期)
    const week: any = {
        '0': '日',
        '1': '一',
        '2': '二',
        '3': '三',
        '4': '四',
        '5': '五',
        '6': '六',
    };
    // 中文数字（季度）
    const quarter: any = {
        '1': '一',
        '2': '二',
        '3': '三',
        '4': '四',
    };
    if (/(W+)/.test(format))
        format = format.replace(RegExp.$1, RegExp.$1.length > 1 ? (RegExp.$1.length > 2 ? '星期' + week[we] : '周' + week[we]) : week[we]);
    if (/(Q+)/.test(format)) format = format.replace(RegExp.$1, RegExp.$1.length == 4 ? '第' + quarter[qut] + '季度' : quarter[qut]);
    for (let k in opt) {
        let r = new RegExp('(' + k + ')').exec(format);
        // 若输入的长度不为1，则前面补零
        if (r) format = format.replace(r[1], RegExp.$1.length == 1 ? opt[k] : opt[k].padStart(RegExp.$1.length, '0'));
    }
    return format;
}

/**
 * 10秒：  10 * 1000
 * 1分：   60 * 1000
 * 1小时： 60 * 60 * 1000
 * 24小时：60 * 60 * 24 * 1000
 * 3天：   60 * 60* 24 * 1000 * 3
 *
 * let data = new Date()
 * formatPast(data)                                           // 刚刚
 * formatPast(data - 11 * 1000)                               // 11秒前
 * formatPast(data - 2 * 60 * 1000)                           // 2分钟前
 * formatPast(data - 60 * 60 * 2 * 1000)                      // 2小时前
 * formatPast(data - 60 * 60 * 2 * 1000)                      // 2小时前
 * formatPast(data - 60 * 60 * 71 * 1000)                     // 2天前
 * formatPast("2020-06-01")                                   // 2020-06-01
 * formatPast("2020-06-01", "YYYY-mm-dd HH:MM:SS WWW QQQQ")   // 2020-06-01 08:00:00 星期一 第二季度
 */
export function formatPast(param: any, format: string = 'YYYY-mm-dd') {
    // 传入格式处理、存储转换值
    let t: any, s: any;
    // 获取js 时间戳
    let time: any = new Date().getTime();
    // 是否是对象
    typeof param === 'string' || 'object' ? (t = new Date(param).getTime()) : (t = param);
    // 当前时间戳 - 传入时间戳
    time = Number.parseInt(`${time - t}`);
    if (time < 10000) {
        // 10秒内
        return '刚刚';
    } else if (time < 60000 && time >= 10000) {
        // 超过10秒少于1分钟内
        s = Math.floor(time / 1000);
        return `${s}秒前`;
    } else if (time < 3600000 && time >= 60000) {
        // 超过1分钟少于1小时
        s = Math.floor(time / 60000);
        return `${s}分钟前`;
    } else if (time < 86400000 && time >= 3600000) {
        // 超过1小时少于24小时
        s = Math.floor(time / 3600000);
        return `${s}小时前`;
    } else if (time < 259200000 && time >= 86400000) {
        // 超过1天少于3天内
        s = Math.floor(time / 86400000);
        return `${s}天前`;
    } else {
        // 超过3天
        let date = typeof param === 'string' || 'object' ? new Date(param) : param;
        return formatDate(date, format);
    }
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
