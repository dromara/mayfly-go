import { Msg } from '@/hooks/useI18n';
import { i18n } from '@/i18n';
import { v1 as uuidv1 } from 'uuid';

/**
 * 模板字符串解析，如：template = 'hahaha{name}_{id}' ,param = {name: 'hh', id: 1}
 * 解析后为 hahahahh_1
 * @param template 模板字符串
 * @param param   参数占位符
 * @returns
 */
export function templateResolve(template: string, param: Record<string, unknown> | FormData) {
    return template.replace(/\{\w+\}/g, (word) => {
        const key = word.substring(1, word.length - 1);
        let value: unknown;
        // 兼容FormData类型的参数
        if (param instanceof FormData) {
            value = param.get(key);
        } else {
            value = param[key];
        }
        if (value != null) {
            return String(value);
        }
        return '';
    });
}

// 首字符头像
export function letterAvatar(name: string, size = 60, color = '') {
    name = name || '';
    size = size || 60;
    var colours = [
            '#1abc9c',
            '#2ecc71',
            '#3498db',
            '#9b59b6',
            '#34495e',
            '#16a085',
            '#27ae60',
            '#2980b9',
            '#8e44ad',
            '#2c3e50',
            '#f1c40f',
            '#e67e22',
            '#e74c3c',
            '#00bcd4',
            '#95a5a6',
            '#f39c12',
            '#d35400',
            '#c0392b',
            '#bdc3c7',
            '#7f8c8d',
        ],
        nameSplit = String(name).split(' '),
        initials,
        charIndex,
        colourIndex,
        canvas,
        context,
        dataURI;

    if (nameSplit.length == 1) {
        initials = nameSplit[0] ? nameSplit[0].charAt(0) : '?';
    } else {
        initials = nameSplit[0].charAt(0) + nameSplit[1].charAt(0);
    }
    if (window.devicePixelRatio) {
        size = size * window.devicePixelRatio;
    }
    initials = initials.toLocaleUpperCase();
    charIndex = (initials == '?' ? 72 : initials.charCodeAt(0)) - 64;
    colourIndex = charIndex % 20;
    canvas = document.createElement('canvas');
    canvas.width = size;
    canvas.height = size;
    context = canvas.getContext('2d') as CanvasRenderingContext2D;

    context.fillStyle = color ? color : colours[colourIndex - 1];
    context.fillRect(0, 0, canvas.width, canvas.height);
    context.font = Math.round(canvas.width / 2) + "px 'Microsoft Yahei'";
    context.textAlign = 'center';
    context.fillStyle = '#FFF';
    context.fillText(initials, size / 2, size / 1.5);
    dataURI = canvas.toDataURL();
    canvas = null;
    return dataURI;
}

/**
 * 计算文本所占用的宽度（px） -> 该种方式较为准确
 * 使用span标签包裹内容，然后计算span的宽度 width： px
 * @param str
 */
export function getTextWidth(str: string) {
    let width = 0;
    let html = document.createElement('span');
    html.innerText = str;
    html.className = 'getTextWidth';
    document?.querySelector('body')?.appendChild(html);
    width = (document?.querySelector('.getTextWidth') as HTMLElement).offsetWidth;
    document?.querySelector('.getTextWidth')?.remove();
    return width;
}

/**
 *
 * @returns uuid
 */
export function randomUuid() {
    return uuidv1();
}

/**
 * 从剪贴板粘贴文本
 * @returns Promise<string> 剪贴板中的文本
 * @throws Error 当无法访问剪贴板时抛出异常
 */
export async function pasteFromClipboard(): Promise<string> {
    // navigator clipboard 需要https等安全上下文
    if (navigator.clipboard && window.isSecureContext) {
        // navigator clipboard 从剪贴板读文本
        try {
            const text = await navigator.clipboard.readText();
            return text;
        } catch (e: unknown) {
            throw new Error(i18n.global.t('common.pasteFailed'));
        }
    }

    // 非安全上下文（HTTP 环境），无法读取剪贴板
    throw new Error(i18n.global.t('common.pasteNotSupported'));
}

/**
 * 拷贝文本至剪贴板
 * @param txt 需要拷贝到剪贴板的文本
 */
export async function copyToClipboard(txt: string) {
    // navigator clipboard 需要https等安全上下文
    if (navigator.clipboard && window.isSecureContext) {
        // navigator clipboard 向剪贴板写文本
        try {
            await navigator.clipboard.writeText(txt);
            Msg.success('common.copySuccess');
        } catch (e: unknown) {
            Msg.error('common.copyFailed');
        }
        return;
    }

    // 非安全上下文（HTTP 环境），无法使用 Clipboard API
    // 降级方案：创建临时 textarea 并使用 execCommand('copy')
    try {
        const textarea = document.createElement('textarea');
        textarea.value = txt;
        textarea.style.position = 'fixed';
        textarea.style.left = '-9999px';
        textarea.style.top = '-9999px';
        document.body.appendChild(textarea);
        textarea.focus();
        textarea.select();

        const success = document.execCommand('copy');
        document.body.removeChild(textarea);

        if (success) {
            Msg.success('common.copySuccess');
        } else {
            Msg.error('common.copyFailed');
        }
    } catch (e: unknown) {
        Msg.error('common.copyNotSupported');
    }
}

export function fuzzyMatchField<T>(keyword: string, fields: T[], ...valueExtractFuncs: ((field: T) => string | undefined)[]) {
    keyword = keyword?.toLowerCase();
    return fields.filter((field) => {
        for (let valueExtractFunc of valueExtractFuncs) {
            const value = valueExtractFunc(field);
            if (value && isPrefixSubsequence(keyword, value)) {
                return true;
            }
        }
        return false;
    });
}

/**
 * 匹配是否为前缀子序列 targetTemplate=username prefix=uname -> true，prefix=uname2 -> false
 * @param prefix 字符串前缀（不连续也可以,但不改变字符的相对顺序）
 * @param targetTemplate 目标模板
 * @returns 是否匹配
 */
export function isPrefixSubsequence(prefix: string, targetTemplate: string) {
    let i = 0; // 指向prefix的索引
    let j = 0; // 指向targetTemplate的索引

    while (i < prefix.length && j < targetTemplate.length) {
        if (prefix[i] === targetTemplate[j]) {
            // 字符匹配，两个指针都向前移动
            i++;
        }
        j++; // 目标字符串指针始终向前移动
    }

    // 如果prefix的所有字符都被找到，返回true
    return i === prefix.length;
}

/**
 *  生成随机密码
 * @param length 密码长度
 */
export function randomPassword(length = 10) {
    const lowerCase = 'abcdefghijklmnopqrstuvwxyz';
    const upperCase = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ';
    const numbers = '0123456789';
    const specialChars = '!@#$%^&*()-_=+[]{}|;:,.<>?';

    // 确保每个类别至少包含一个字符
    let password = [getRandomChar(lowerCase), getRandomChar(upperCase), getRandomChar(numbers), getRandomChar(specialChars)];

    // 剩余字符从所有字符集中随机选择
    const allChars = lowerCase + upperCase + numbers + specialChars;
    for (let i = 4; i < length; i++) {
        password.push(getRandomChar(allChars));
    }

    // 打乱数组顺序以增加随机性
    shuffleArray(password);

    return password.join('');
}

export function randomString(length = 8) {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    let result = '';
    for (let i = 0; i < length; i++) {
        result += getRandomChar(chars);
    }
    return result;
}

function getRandomChar(charSet: string) {
    const randomIndex = Math.floor(Math.random() * charSet.length);
    return charSet[randomIndex];
}

function shuffleArray(array: string[]) {
    for (let i = array.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [array[i], array[j]] = [array[j], array[i]];
    }
}
