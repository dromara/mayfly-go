/**
 * 枚举类
 * @author meilin.huang
 */
export class Enum {
    /**
     * 添加枚举字段
     * 
     * @param {string} field  枚举字段名
     * @param {string} label  枚举名称
     * @param {Object} value  枚举值
     */
    add(field: string, label: string, value: any) {
        this[field] = { label, value }
        return this
    }

    /**
     * 根据枚举value获取其label
     * 
     * @param {Object} value 
     */
    getLabelByValue(value: any) {
        // 字段不存在返回‘’
        if (value === undefined || value === null) {
            return ''
        }
        for (const i in this) {
            const e: any = this[i]
            if (e && e.value === value) {
                return e.label
            }
        }

        return ''
    }
}
