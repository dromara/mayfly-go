/**
 * JSON 表单 Schema 编译器
 *
 * 将 v1 JSON Schema（AutoFormJsonSchema / JsonField[]）编译为 AutoFormItem[]，
 * 复用 AutoForm / AutoFormControl 的渲染与校验体系：
 * - when / disabled 条件 → 函数（基于 evalCondition 安全求值）
 * - optionsSource → 异步 options 函数（走统一 request 封装，仅允许站内相对路径）
 * - rules → element-plus FormItemRule
 */
import type { FormItemRule } from 'element-plus';
import { i18n } from '@/i18n';
import request from '@/common/request';
import type { AutoFormData, AutoFormItem, AutoFormSelectOption, AutoFormTab, ControlDescriptor } from '../types';
import { CONTROL_REGISTRY } from '../types';
import { evalCondition } from './condition';
import type { AutoFormJsonSchema, JsonCondition, JsonField, JsonOptionsSource, JsonRules } from './schema';

/** 条件编译：JSON 条件 → when/disabled 使用的求值函数 */
const compileCondition = (cond: JsonCondition | undefined) => (cond ? (form: AutoFormData) => evalCondition(cond, form) : undefined);

/** 规则失败提示：显式 message 优先（i18n key 或原文），否则使用默认 key。
 * 返回求值函数延迟翻译（async-validator 支持函数 message），语言切换后校验时取最新文案，避免编译期冻结 */
const ruleMessage = (message: string | undefined, defaultKey: string) => (): string => i18n.global.t(message ?? defaultKey);

/** 校验规则编译：required 提升为 item.required（由 AutoForm 生成 i18n 必填规则），其余编译为 FormItemRule */
const compileRules = (field: JsonField): { required?: boolean; rules?: FormItemRule[] } => {
    const jsonRules: JsonRules | undefined = field.rules;
    if (!jsonRules) {
        return {};
    }
    const rules: FormItemRule[] = [];
    const message = jsonRules.message;
    if (jsonRules.minLength != null || jsonRules.maxLength != null) {
        rules.push({
            min: jsonRules.minLength,
            max: jsonRules.maxLength,
            message: ruleMessage(message, 'common.lengthRuleMsg'),
            trigger: 'blur',
        });
    }
    if (jsonRules.pattern) {
        rules.push({ pattern: new RegExp(jsonRules.pattern), message: ruleMessage(message, 'common.patternRuleMsg'), trigger: 'blur' });
    }
    if (jsonRules.min != null || jsonRules.max != null) {
        rules.push({
            type: 'number',
            min: jsonRules.min,
            max: jsonRules.max,
            message: ruleMessage(message, 'common.numberRangeRuleMsg'),
            trigger: 'blur',
        });
    }
    return { required: jsonRules.required, rules: rules.length > 0 ? rules : undefined };
};

/** JSON Schema 允许编译的控件类型白名单：从控件注册表派生，新增类型无需修改本文件
 *（custom 需要插槽、enum/radio 需要 enums 函数对象，均无法纯 JSON 表达） */
const isJsonCompilableType = (type: string): boolean => (CONTROL_REGISTRY as Record<string, ControlDescriptor | undefined>)[type]?.jsonCompilable ?? false;

/** 加载声明式选项数据源（url 仅允许站内相对路径，deps 字段值合并进请求参数） */
const loadOptionsSource = async (source: JsonOptionsSource, form: AutoFormData): Promise<AutoFormSelectOption[]> => {
    const url = source.url.trim();
    // 必须以 '/' 开头且非 '//' 开头（协议相对路径），彻底杜绝绝对地址/大小写变体（如 HTTP://）绕过站内限制
    if (!url.startsWith('/') || url.startsWith('//')) {
        throw new Error(`optionsSource.url 仅允许站内相对路径（以 / 开头）: ${source.url}`);
    }
    const params: Record<string, unknown> = { ...(source.params ?? {}) };
    // deps 值的读取发生在 await 之前，AutoFormControl 的 watchEffect 可追踪到依赖字段变化并重新加载
    for (const dep of source.deps ?? []) {
        params[dep] = form[dep];
    }
    const res = await request.request<unknown>(source.method ?? 'get', url, params);
    const list = source.dataField ? (res as Record<string, unknown>)?.[source.dataField] : res;
    if (!Array.isArray(list)) {
        return [];
    }
    const valueField = source.valueField ?? 'value';
    const labelField = source.labelField ?? 'label';
    return list.map((item: unknown) => {
        const record = (item ?? {}) as Record<string, unknown>;
        return { value: record[valueField] as AutoFormSelectOption['value'], label: record[labelField] as string };
    });
};

/** 选项编译：静态数组直接使用，optionsSource 编译为异步加载函数 */
const compileOptions = (field: JsonField): AutoFormItem['options'] => {
    if (field.options) {
        return field.options;
    }
    if (field.optionsSource) {
        const source = field.optionsSource;
        return (form: AutoFormData) => loadOptionsSource(source, form);
    }
    return undefined;
};

/** 编译单个 JSON 字段为 AutoFormItem（不支持/未知类型返回 undefined 并告警，调用方跳过该字段） */
export const compileJsonField = (field: JsonField): AutoFormItem | undefined => {
    const type = field.type ?? 'input';
    if (!isJsonCompilableType(type)) {
        // custom / enum / 未知类型无法纯 JSON 表达，静默渲染会产生空白控件，显式跳过并留痕
        console.warn(`[AutoForm] 字段 ${field.prop} 的控件类型 ${type} 不支持 JSON 下发，已跳过`);
        return undefined;
    }
    const { required, rules } = compileRules(field);
    const disabled = typeof field.disabled === 'object' && field.disabled !== null ? compileCondition(field.disabled) : field.disabled;
    const readonly = typeof field.readonly === 'object' && field.readonly !== null ? compileCondition(field.readonly) : field.readonly;
    return {
        prop: field.prop,
        label: field.label,
        type,
        placeholder: field.placeholder,
        tooltip: field.tooltip,
        description: field.description,
        groupDescription: field.groupDescription,
        required,
        rules,
        defaultValue: field.defaultValue,
        disabled,
        readonly,
        hidden: field.hidden,
        when: compileCondition(field.when),
        span: field.span,
        options: compileOptions(field),
        multiple: field.multiple,
        enums: undefined,
        rows: field.rows,
        min: field.min,
        max: field.max,
        prefix: field.prefix,
        suffix: field.suffix,
        props: field.props,
    };
};

/**
 * 编译 JSON 表单定义为 AutoFormItem[]
 *
 * @param input 完整 Schema（AutoFormJsonSchema）或纯字段数组（JsonField[]）
 */
export const compileJsonForm = (input: AutoFormJsonSchema | JsonField[]): AutoFormItem[] => {
    const fields = Array.isArray(input) ? input : input.fields;
    return (fields ?? []).map(compileJsonField).filter((item): item is AutoFormItem => item !== undefined);
};

/**
 * 编译 JSON Schema 内置 tabs 为 AutoFormTab[]（无 tabs 时返回空数组）
 *
 * 供 AutoForm 的 schema 入口使用：schema.tabs 存在时渲染为 Tab 布局。
 */
export const compileJsonTabs = (schema: AutoFormJsonSchema): AutoFormTab[] => {
    return (schema.tabs ?? []).map((tab) => ({
        name: tab.name,
        label: tab.label,
        icon: tab.icon,
        items: compileJsonForm({ version: schema.version, fields: tab.fields }),
    }));
};
