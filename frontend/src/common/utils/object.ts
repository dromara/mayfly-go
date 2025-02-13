/**
 * 根据对象访问路径，获取对应的值
 *
 * @param obj 对象，如 {user: {name: 'xxx'}, orderNo: 1212211, products: [{id: 12}]}
 * @param path 访问路径，如 orderNo 或者 user.name 或者product[0].id
 * @returns 路径对应的值
 */
export function getValueByPath(obj: any, path: string) {
    const keys = path.split('.');
    let result = obj;
    for (let key of keys) {
        if (!result) {
            return undefined;
        }
        // 如果是字符串，则尝试使用json解析
        if (typeof result == 'string') {
            try {
                result = JSON.parse(result);
            } catch (e) {
                console.error(e);
                return undefined;
            }
        }
        if (typeof result !== 'object') {
            return undefined;
        }

        if (key.includes('[') && key.includes(']')) {
            // 处理包含数组索引的情况
            const arrayKey = key.substring(0, key.indexOf('['));
            const matchIndex = key.match(/\[(.*?)\]/);

            if (!matchIndex) {
                return undefined;
            }

            const index = parseInt(matchIndex[1]);

            let arrValue = result[arrayKey];
            if (typeof arrValue == 'string') {
                try {
                    arrValue = JSON.parse(arrValue);
                } catch (e) {
                    result = undefined;
                    break;
                }
            }

            result = Array.isArray(arrValue) ? arrValue[index] : undefined;
        } else {
            result = result[key];
        }
    }

    return result;
}

/**
 * 根据字段路径设置字段值，若路径不存在，则建对应的路径对象信息
 * @param obj 对象
 * @param path 字段路径
 * @param value 字段值
 */
export function setValueByPath(obj: any, path: string[], value: any) {
    for (let i = 0; i < path.length - 1; i++) {
        const key = path[i];
        if (!obj[key]) {
            obj[key] = {};
        }
        obj = obj[key];
    }
    obj[path[path.length - 1]] = value;
}

/**
 * 使用递归函数进行深度克隆，并支持通过回调函数进行指定字段值的调整
 *
 * @param obj 要克隆的对象
 * @param callback 回调函数，在每个字段被克隆之前调用，可以调整字段的值
 * @param hash 用于处理循环引用的 WeakMap
 * @returns 深度克隆后的对象
 */
export function deepClone(
    obj: any,
    callback: (key: string | number, value: any) => any = (key: string | number, value: any) => value,
    hash = new WeakMap()
): any {
    if (Object(obj) !== obj) return obj; // 基本数据类型直接返回
    if (hash.has(obj)) return hash.get(obj); // 处理循环引用

    let result: any;

    if (obj instanceof Set) {
        result = new Set();
        hash.set(obj, result);
        obj.forEach((val) => result.add(deepClone(val, callback, hash)));
    } else if (obj instanceof Map) {
        result = new Map();
        hash.set(obj, result);
        obj.forEach((val, key) => result.set(key, deepClone(val, callback, hash)));
    } else if (obj instanceof Date) {
        result = new Date(obj.getTime());
    } else if (obj instanceof RegExp) {
        result = new RegExp(obj);
    } else if (typeof obj === 'object') {
        result = Array.isArray(obj) ? [] : {};
        hash.set(obj, result);
        for (let key in obj) {
            if (obj.hasOwnProperty(key)) {
                let value = obj[key];
                value = callback(key, value);
                result[key] = deepClone(value, callback, hash);
            }
        }
    } else {
        result = obj;
    }

    return result;
}
