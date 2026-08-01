/**
 * AutoForm 类型定义
 *
 * AutoFormItem 同时描述字段渲染与校验规则，是表单的唯一数据源。
 * 通过配置化方式声明表单，避免各处重复维护相似的 el-form 模板。
 */
import type { FormItemRule } from 'element-plus';
import type { EnumValue } from '@/common/Enum';

// ── 字段类型 ────────────────────────────────────────────────────

export type AutoFormItemType =
    | 'input' // 文本输入框（默认）
    | 'password' // 密码输入框
    | 'number' // 数字输入框
    | 'textarea' // 文本域
    | 'select' // 下拉选择（options）
    | 'enum' // 枚举下拉选择（enums，EnumValue 对象）
    | 'switch' // 开关
    | 'date' // 日期选择
    | 'datetime' // 日期时间选择
    | 'time' // 时间选择
    | 'monaco' // Monaco 代码编辑器
    | 'divider' // 分隔标题（非字段，仅渲染 el-divider + label）
    | 'custom'; // 自定义内容插槽（插槽名为 slot ?? prop）

/** 表单数据对象（动态字段，值类型由控件决定） */
export type AutoFormData = Record<string, any>;

export interface AutoFormSelectOption {
    value: string | number | boolean;
    label: string;
}

// ── 字段配置 ────────────────────────────────────────────────────

export interface AutoFormItem {
    /** 字段名（divider 类型可省略） */
    prop?: string;

    /** 标签文本 i18n key（divider 类型时为分隔标题） */
    label?: string;

    /** 控件类型，默认 'input' */
    type?: AutoFormItemType;

    /** 占位文本 i18n key（缺省时根据类型自动生成 请输入/请选择{label}） */
    placeholder?: string;

    /** 帮助提示 i18n key（显示在标签旁的 tooltip 图标） */
    tooltip?: string;

    /** 是否必填（自动生成 requiredInput / requiredSelect 校验规则） */
    required?: boolean;

    /** 额外的 element-plus 校验规则（与 required 生成的规则合并） */
    rules?: FormItemRule | FormItemRule[];

    /** 字段默认值（buildDefaultForm 构建新表单时使用） */
    defaultValue?: unknown;

    /** 禁用（支持根据表单值动态计算） */
    disabled?: boolean | ((form: AutoFormData) => boolean);

    /** 条件显隐：返回 false 时隐藏该字段（隐藏时不参与校验） */
    when?: (form: AutoFormData) => boolean;

    /** 栅格跨度（el-col span，缺省时由 AutoForm cols 均分 24） */
    span?: number;

    // ── select 专用 ──
    /** 选项：静态数组或异步加载函数（函数入参为当前表单值，值变化时自动重新加载） */
    options?: AutoFormSelectOption[] | ((form: AutoFormData) => Promise<AutoFormSelectOption[]>);

    /** 是否多选（select / enum 类型） */
    multiple?: boolean;

    // ── enum 专用 ──
    /** 枚举对象（type='enum' 时使用） */
    enums?: Record<string, EnumValue>;

    // ── textarea 专用 ──
    /** 文本域行数（默认 3） */
    rows?: number;

    // ── number 专用 ──
    min?: number;
    max?: number;

    // ── custom 专用 ──
    /** 自定义插槽名（缺省使用 prop 作为插槽名） */
    slot?: string;

    /** 透传给底层控件的额外属性（如 monaco 的 language/height、switch 的 active-value 等） */
    props?: Record<string, unknown>;

    /** 值变化回调：用于字段联动（修改 form 中其他字段值等） */
    onChange?: (value: unknown, form: AutoFormData) => void;
}

// ── 工具函数 ────────────────────────────────────────────────────

/** 判断字段是否为“选择类”控件（用于生成 请选择{label} 提示语） */
export const isSelectLikeItem = (item: AutoFormItem): boolean => {
    const type = item.type ?? 'input';
    return ['select', 'enum', 'date', 'datetime', 'time'].includes(type);
};

/**
 * 根据字段配置构建默认表单数据（应用各字段 defaultValue）
 *
 * 用于新增场景初始化表单，编辑场景直接使用回填数据。
 */
export const buildDefaultForm = (items: AutoFormItem[]): AutoFormData => {
    const form: AutoFormData = {};
    for (const item of items) {
        if (item.prop && item.defaultValue !== undefined) {
            form[item.prop] = item.defaultValue;
        }
    }
    return form;
};
