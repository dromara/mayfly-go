/**
 * 不符合业务断言错误
 */
class AssertError extends Error {
    constructor(message: string) {
        super(message);
        // 错误类名
        this.name = 'AssertError';
    }
}

/**
 * 断言表达式为true
 *
 * @param condition 条件表达式
 * @param msg 错误消息
 */
export function isTrue(condition: boolean, msg: string) {
    if (!condition) {
        throw new AssertError(msg);
    }
}

/**
 * 断言不能为空值，即null,0,''等
 *
 * @param obj 对象1
 * @param msg 错误消息
 */
export function notBlank(obj: any, msg: string) {
    isTrue(obj, msg);
}

/**
 * 断言两对象相等
 *
 * @param obj1 对象1
 * @param obj2 对象2
 * @param msg 错误消息
 */
export function isEquals(obj1: any, obj2: any, msg: string) {
    isTrue(obj1 === obj2, msg);
}

/**
 * 断言对象不为null或undefiend
 *
 * @param obj 对象
 * @param msg 错误提示
 */
export function notNull(obj: any, msg: string) {
    if (obj == null || obj == undefined) {
        throw new AssertError(msg);
    }
}

/**
 * 断言字符串不能为空
 *
 * @param str 字符串
 * @param msg 错误提示
 */
export function notEmpty(str: string, msg: string) {
    if (str == null || str == undefined || str == '') {
        throw new AssertError(msg);
    }
}
