/**
 * AutoForm JSON Schema 类型定义（v1）
 *
 * 纯 JSON 可序列化的表单定义，供后端下发（如系统配置 params、脚本入参定义）
 * 或代码内以纯数据方式声明表单。经 compileJsonForm 编译为 AutoFormItem[] 后
 * 由 AutoForm / AutoFormDialog / AutoFormDrawer 渲染。
 *
 * 与 AutoFormItem 的关系：JSON Schema 是其可序列化子集——
 * 函数型配置（when / disabled(fn) / options(fn) / onChange）由结构化声明
 * （JsonCondition / JsonOptionsSource）编译生成，不使用 eval。
 */
import type { AutoFormItemType, AutoFormSelectOption } from '../types';

/** Schema 版本号，前端据此识别格式，后端迁移据此幂等跳过已升级数据 */
export const JSON_FORM_SCHEMA_VERSION = 1;

// ── 条件表达式 ──────────────────────────────────────────────────

/** 简单条件操作符 */
export type JsonConditionOp =
    | 'eq' // 等于（宽松比较：'1' 与 1 视为相等，null 与 undefined 视为相等）
    | 'ne' // 不等于（宽松比较）
    | 'in' // 值在指定数组内
    | 'notIn' // 值不在指定数组内
    | 'empty' // 值为空（undefined / null / 空字符串 / 空数组）
    | 'notEmpty' // 值非空
    | 'gt' // 大于（数值比较，无法转为数字时恒为 false）
    | 'gte' // 大于等于
    | 'lt' // 小于
    | 'lte'; // 小于等于

/** 简单条件：单字段单操作符 */
export interface JsonSimpleCondition {
    /** 目标字段名（表单数据中的 prop） */
    field: string;
    /** 操作符 */
    op: JsonConditionOp;
    /** 比较值（eq/ne/in/notIn 及数值比较时使用；empty/notEmpty 可省略） */
    value?: unknown;
}

/** 组合条件：all 内全部满足、any 内任一满足、not 取反 */
export interface JsonGroupCondition {
    /** 与组合：所有子条件均满足 */
    all?: JsonCondition[];
    /** 或组合：任一子条件满足 */
    any?: JsonCondition[];
    /** 非组合：子条件不满足 */
    not?: JsonCondition;
}

export type JsonCondition = JsonSimpleCondition | JsonGroupCondition;

// ── 声明式选项数据源 ────────────────────────────────────────────

/** select 类字段的声明式选项数据源（编译为异步 options 函数） */
export interface JsonOptionsSource {
    /**
     * 选项数据接口地址（必填，仅允许站内相对路径如 '/api/...'，
     * 走统一 request 封装发出，自动附带认证信息）
     */
    url: string;
    /** 请求方法，默认 'get' */
    method?: 'get' | 'post';
    /** 请求附加参数 */
    params?: Record<string, unknown>;
    /** 从响应中提取选项数组的字段名（如 'list'），不传则响应本身需为数组 */
    dataField?: string;
    /** 每项中作为 value 的字段名（默认 'value'） */
    valueField?: string;
    /** 每项中作为 label 的字段名（默认 'label'） */
    labelField?: string;
    /** 依赖的字段名：这些表单字段值变化时重新加载选项（值会合并进请求参数） */
    deps?: string[];
}

// ── 校验规则 ────────────────────────────────────────────────────

/** 校验规则（JSON 可序列化子集，编译为 element-plus FormItemRule） */
export interface JsonRules {
    /** 是否必填 */
    required?: boolean;
    /** 最小长度（字符串） */
    minLength?: number;
    /** 最大长度（字符串） */
    maxLength?: number;
    /** 正则表达式（字符串形式，编译时 new RegExp） */
    pattern?: string;
    /** 校验失败提示（i18n key 或原文，作用于 length / pattern / 数值范围规则） */
    message?: string;
    /** 最小值（数值校验，与控件级 min 字段区分） */
    min?: number;
    /** 最大值（数值校验） */
    max?: number;
}

// ── 字段配置 ────────────────────────────────────────────────────

/** JSON 表单字段配置（渲染 + 校验的唯一数据源，严格 JSON 可序列化） */
export interface JsonField {
    /** 字段名 */
    prop: string;

    /** 标签文本（i18n key 或原文） */
    label?: string;

    /** 控件类型，默认 'input'（JSON 下发场景不支持 'custom' 插槽） */
    type?: AutoFormItemType;

    /** 占位文本（i18n key 或原文，缺省时根据类型自动生成 请输入/请选择{label}） */
    placeholder?: string;

    /** 帮助提示（i18n key 或原文，显示在标签旁的 tooltip 图标） */
    tooltip?: string;

    /** 辅助说明（i18n key 或原文，显示在控件下方） */
    description?: string;

    /** 分组描述（i18n key 或原文，type='group' 时显示在分组标题下方） */
    groupDescription?: string;

    /** 校验规则 */
    rules?: JsonRules;

    /** 字段默认值 */
    defaultValue?: unknown;

    /** 禁用：布尔值或条件表达式（条件满足时禁用） */
    disabled?: boolean | JsonCondition;

    /** 只读：布尔值或条件表达式（条件满足时只读禁用） */
    readonly?: boolean | JsonCondition;

    /** 隐藏：不渲染控件但保留字段值（如透传 tenant_id） */
    hidden?: boolean;

    /** 条件显隐：条件不满足时隐藏该字段（隐藏时不参与校验） */
    when?: JsonCondition;

    /** 栅格跨度（el-col span，缺省时由 cols 均分 24） */
    span?: number;

    // ── select 专用 ──
    /** 静态选项（与 optionsSource 二选一，同时存在时优先 options） */
    options?: AutoFormSelectOption[];

    /** 声明式选项数据源 */
    optionsSource?: JsonOptionsSource;

    /** 是否多选（select 类型） */
    multiple?: boolean;

    // ── textarea 专用 ──
    /** 文本域行数（默认 3） */
    rows?: number;

    // ── number 专用 ──
    min?: number;
    max?: number;

    // ── input / number 前后缀 ──
    /** 输入框前缀文本（如 '¥'、'http://'，i18n key 或原文） */
    prefix?: string;
    /** 输入框后缀文本（如 'ms'，i18n key 或原文） */
    suffix?: string;

    /** 透传给底层控件的额外属性（如 monaco 的 language/height、switch 的 active-value 等） */
    props?: Record<string, unknown>;
}

/** JSON 表单 Tab 页签配置（容器级布局，每个 Tab 为一组字段） */
export interface JsonTab {
    /** Tab 唯一标识 */
    name: string;

    /** Tab 显示标签（i18n key 或原文） */
    label: string;

    /** Tab 图标名（可选） */
    icon?: string;

    /** 该 Tab 下的字段配置 */
    fields: JsonField[];
}

// ── 表单 Schema ────────────────────────────────────────────────

/** JSON 表单 Schema（v1） */
export interface AutoFormJsonSchema {
    /** Schema 版本号，当前固定为 1 */
    version: number;

    /** 表单字段配置（tabs 存在时以 tabs 为准，fields 可为空数组） */
    fields: JsonField[];

    /** Tab 页签布局（可选；存在时渲染为 el-tabs，每个 Tab 独立字段组） */
    tabs?: JsonTab[];

    /** 栅格列数（默认 1） */
    cols?: number;
}

/** 判断未知数据是否为 v1 JSON 表单 Schema（对象且 version === 1） */
export const isJsonFormSchema = (data: unknown): data is AutoFormJsonSchema => {
    return typeof data === 'object' && data !== null && !Array.isArray(data) && (data as AutoFormJsonSchema).version === JSON_FORM_SCHEMA_VERSION;
};
