export interface EnumValueTag {
    color?: string;
    type?: string;
}

/** 枚举值的基础类型（支持 string 和 number） */
export type EnumValueBase = string | number;

/**
 * 枚举值
 */
export class EnumValue<V extends EnumValueBase = EnumValueBase> {
    /**
     * 枚举值
     */
    value: V;

    /**
     * 枚举描述
     */
    label: string;

    /**
     * 展示的标签信息
     */
    tag: EnumValueTag;

    extra: any;

    constructor(value: V, label: string) {
        this.value = value;
        this.label = label;
    }

    setTagType(type: string = 'primary'): this {
        this.tag = { type };
        return this;
    }

    tagTypeInfo(): this {
        return this.setTagType('info');
    }

    tagTypeSuccess(): this {
        return this.setTagType('success');
    }

    tagTypeDanger(): this {
        return this.setTagType('danger');
    }

    tagTypeWarning(): this {
        return this.setTagType('warning');
    }

    setTagColor(color: string): this {
        this.tag = { color };
        return this;
    }

    setExtra(extra: any): this {
        this.extra = extra;
        return this;
    }

    public static of<V extends EnumValueBase>(value: V, label: string): EnumValue<V> {
        return new EnumValue(value, label);
    }

    /**
     * 根据枚举值获取指定枚举值对象
     *
     * @param enums 枚举对象
     * @param value 需要匹配的枚举值
     * @returns 枚举值对象
     */
    static getEnumByValue(enums: Record<string, EnumValue>, value: EnumValueBase): EnumValue | null {
        const enumValues = Object.values(enums);
        for (let enumValue of enumValues) {
            if (enumValue.value == value) {
                return enumValue;
            }
        }
        return null;
    }

    /**
     * 根据枚举值获取枚举描述
     *
     * @param enums 枚举对象
     * @param value 枚举值
     * @returns 枚举描述
     */
    static getLabelByValue(enums: Record<string, EnumValue>, value: EnumValueBase) {
        const enumValues = Object.values(enums);
        for (let enumValue of enumValues) {
            if (enumValue['value'] == value) {
                return enumValue['label'];
            }
        }
        return '';
    }
}

export default EnumValue;
