export interface EnumValueTag {
    color?: string;
    type?: string;
}

/**
 * 枚举值
 */
export class EnumValue {
    /**
     * 枚举值
     */
    value: any;

    /**
     * 枚举描述
     */
    label: string;

    /**
     * 展示的标签信息
     */
    tag: EnumValueTag;

    constructor(value: any, label: string) {
        this.value = value;
        this.label = label;
    }

    setTagType(type: string = 'primary'): EnumValue {
        this.tag = { type };
        return this;
    }

    tagTypeInfo(): EnumValue {
        return this.setTagType('info');
    }

    tagTypeSuccess(): EnumValue {
        return this.setTagType('success');
    }

    tagTypeDanger(): EnumValue {
        return this.setTagType('danger');
    }

    tagTypeWarning(): EnumValue {
        return this.setTagType('warning');
    }

    setTagColor(color: string): EnumValue {
        this.tag = { color };
        return this;
    }

    public static of(value: any, label: string): EnumValue {
        return new EnumValue(value, label);
    }

    /**
     * 根据枚举值获取指定枚举值对象
     *
     * @param enumValues 所有枚举值
     * @param value 需要匹配的枚举值
     * @returns 枚举值对象
     */
    static getEnumByValue(enumValues: EnumValue[], value: any): EnumValue | null {
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
    static getLabelByValue(enums: any, value: any) {
        const enumValues = Object.values(enums) as any;
        for (let enumValue of enumValues) {
            if (enumValue['value'] == value) {
                return enumValue['label'];
            }
        }
        return '';
    }
}

export default EnumValue;
