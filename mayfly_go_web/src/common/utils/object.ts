// 根据对象访问路径，获取对应的值
export function getValueByPath(obj: any, path: string) {
    const keys = path.split('.');
    let result = obj;
    for (let key of keys) {
        if (!result || typeof result !== 'object') {
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
            result = Array.isArray(result[arrayKey]) ? result[arrayKey][index] : undefined;
        } else {
            result = result[key];
        }
    }

    return result;
}
