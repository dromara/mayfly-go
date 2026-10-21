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

/**
 * 枚举值 → 本地化标签（各页同名 helper 的单点实现）
 *
 * 取不到登记项时返回原值而不是空串：后端新增状态而枚举未跟上时，空标签会让人以为数据丢了，
 * 原值至少能看出是哪个新值。translate 由调用方传入（而不是在本层 import i18n）：common 层被
 * 广泛依赖，静态引入 i18n 一旦成环，就会表现为只在首屏偶发的初始化报错
 */
export function enumLabel(
    enumObj: Record<string, EnumValue>,
    value: EnumValueBase | null | undefined,
    translate: (key: string) => string,
): string {
    const found = EnumValue.getEnumByValue(enumObj, value as EnumValueBase);
    return found ? translate(found.label) : String(value ?? '');
}

/**
 * 枚举值 → 标签色（读枚举上登记的 tag 元数据，未登记或未设色回退 info）
 *
 * 颜色属于枚举自身的展示语义（成功=success、危险=danger），不该在页面里再 switch 一遍字面量：
 * 新增一个状态值时枚举与页面 switch 必然分叉，而分叉表现只是「颜色不对」，没有任何报错信号
 */
export function enumTagType(enumObj: Record<string, EnumValue>, value: EnumValueBase): string {
    return EnumValue.getEnumByValue(enumObj, value)?.tag?.type ?? 'info';
}

export default EnumValue;
