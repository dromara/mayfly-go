/**
 * AutoForm 类型定义
 *
 * AutoFormItem 同时描述字段渲染与校验规则，是表单的唯一数据源。
 * 通过配置化方式声明表单，避免各处重复维护相似的表单模板。
 *
 * 本文件与 shared.ts / json/ 构成框架无关的 core 层（由 __tests__/architecture.test.ts 守护）：
 * 禁止 import 任何 UI 框架（element-plus 等），UI 能力由适配层组件消费与翻译。
 */
import type { EnumValue } from '@/common/Enum';

/**
 * 校验规则（async-validator 结构化子集，框架无关契约）。
 * 当前由 element-plus 适配层直接消费；替换 UI 框架时仅需适配层做规则格式转换，
 * 全站 items.rules 调用点无需变动（Rules.* 等对象字面量天然兼容）。
 */
export interface AutoFormItemRule {
    required?: boolean;
    message?: string | (() => string);
    trigger?: string | string[];
    validator?: (rule: any, value: any, callback: (error?: string | Error) => void) => void;
    /** 透传底层校验引擎的原生规则字段（min/max/pattern 等），保证适配层可用完整能力 */
    [key: string]: unknown;
}

// ── 字段类型 ────────────────────────────────────────────────────

export type AutoFormItemType =
    | 'input' // 文本输入框（默认）
    | 'password' // 密码输入框
    | 'number' // 数字输入框
    | 'textarea' // 文本域
    | 'select' // 下拉选择（options）
    | 'enum' // 枚举下拉选择（enums，EnumValue 对象）
    | 'radio' // 单选组（enums）
    | 'switch' // 开关
    | 'date' // 日期选择
    | 'datetime' // 日期时间选择
    | 'time' // 时间选择
    | 'monaco' // Monaco 代码编辑器
    | 'tags' // 标签输入器（el-input-tag，字符串数组）
    | 'divider' // 分隔标题（非字段，仅渲染 el-divider + label）
    | 'group' // 分组容器（非字段，标题 + 可选描述 + 带边框容器包裹后续字段，直到下一个 group）
    | 'custom'; // 自定义内容插槽（插槽名为 slot ?? prop）

/** 表单数据对象（动态字段，值类型由控件决定） */
export type AutoFormData = Record<string, any>;

export interface AutoFormSelectOption {
    value: string | number | boolean;
    label: string;
    /** 选项禁用（如联动约束：某选项在特定表单值下不可选） */
    disabled?: boolean;
}

// ── 字段配置 ────────────────────────────────────────────────────

export interface AutoFormItem {
    /** 字段名（divider 类型可省略） */
    prop?: string;

    /** 标签文本 i18n key（divider/group 类型时为分隔标题/分组标题） */
    label?: string;

    /** 标签 i18n 插值参数（如提示可用资源量的动态文案） */
    labelParams?: Record<string, unknown>;

    /** 控件类型，默认 'input' */
    type?: AutoFormItemType;

    /** 占位文本 i18n key（缺省时根据类型自动生成 请输入/请选择{label}） */
    placeholder?: string;

    /** 帮助提示 i18n key 或 key 数组（多行，显示在标签旁的 tooltip 图标） */
    tooltip?: string | string[];

    /** 辅助说明 i18n key（显示在控件下方） */
    description?: string;

    /** 分组描述 i18n key（type='group' 时显示在分组标题下方） */
    groupDescription?: string;

    /** 是否必填（自动生成 requiredInput / requiredSelect 校验规则；支持根据表单值动态计算） */
    required?: boolean | ((form: AutoFormData) => boolean);

    /** 额外的校验规则（与 required 生成的规则合并，经适配层翻译为 UI 框架规则格式） */
    rules?: AutoFormItemRule | AutoFormItemRule[];

    /** 自定义校验函数（rules 无法满足的复杂逻辑）：返回 true 通过，false/string（错误提示 i18n key）不通过；
     *  也可返回 Promise 做异步校验（如远程重名/连通性检查），resolve 同步语义，reject 视为不通过（字段默认文案） */
    validate?: (value: unknown, form: AutoFormData) => boolean | string | Promise<boolean | string>;

    /** 字段默认值（buildDefaultForm 构建新表单时使用） */
    defaultValue?: unknown;

    /** 禁用（支持根据表单值动态计算） */
    disabled?: boolean | ((form: AutoFormData) => boolean);

    /** 只读（同 disabled 语义禁用控件，支持根据表单值动态计算） */
    readonly?: boolean | ((form: AutoFormData) => boolean);

    /** 隐藏：不渲染控件但保留字段值（如透传 tenant_id 等不展示的参数） */
    hidden?: boolean;

    /** 条件显隐：返回 false 时隐藏该字段（隐藏时不参与校验） */
    when?: (form: AutoFormData) => boolean;

    /** 栅格跨度（el-col span，缺省时由 AutoForm cols 均分 24） */
    span?: number;

    // ── select 专用 ──
    /** 选项：静态数组或异步加载函数（函数入参为当前表单值，值变化时自动重新加载） */
    options?: AutoFormSelectOption[] | ((form: AutoFormData) => Promise<AutoFormSelectOption[]>);

    /** 是否多选（select / enum 类型） */
    multiple?: boolean;

    // ── radio 专用 ──
    // 选项使用 options 静态数组（无异步需求）或 enums 枚举对象，select/enum/radio 均支持 excludeValues / optionDisabled

    // ── enum 专用 ──
    /** 枚举对象或枚举子集数组（type='enum' 时使用，数组仅渲染传入的枚举项） */
    enums?: Record<string, EnumValue> | EnumValue[];

    // ── select / enum 通用 ──
    /** 从选项中剔除的值（如按场景隐藏某类凭证类型） */
    excludeValues?: unknown[];

    /** 选项级禁用（返回 true 时该选项不可选，用于选项联动约束） */
    optionDisabled?: (value: unknown, form: AutoFormData) => boolean;

    // ── textarea 专用 ──
    /** 文本域行数（默认 3） */
    rows?: number;

    // ── number 专用 ──
    min?: number;
    max?: number;

    // ── input / number 前后缀 ──
    /** 输入框前缀文本（如 '¥'、'http://'，input / number 类型生效） */
    prefix?: string;
    /** 输入框后缀文本（如 'ms'、'次/天'，input / number 类型生效） */
    suffix?: string;

    // ── custom 专用 ──
    /** 自定义插槽名（缺省使用 prop 作为插槽名） */
    slot?: string;

    /** 透传给底层控件的额外属性（如 monaco 的 language/height、switch 的 active-value 等） */
    props?: Record<string, unknown>;

    /** 值变化回调：用于字段联动（修改 form 中其他字段值等） */
    onChange?: (value: unknown, form: AutoFormData) => void;
}

// ── 控件类型注册表（新增控件类型的“类型知识”单一出处） ──────

/**
 * 控件类型能力描述
 *
 * 新增 AutoFormItemType 联合成员时：在 AutoFormControl 增加渲染分支后，
 * 同步在本表登记能力位，isSelectLikeItem / JSON 编译白名单即自动生效，
 * 无需再散点修改多处判断（渲染分支因各控件 props 绑定异构，保留显式 v-if 以保类型安全）。
 */
export interface ControlDescriptor {
    /** 选择类控件：必填提示语生成“请选择{label}”（否则为“请输入{label}”） */
    selectLike: boolean;
    /** 允许 JSON Schema 下发（无需插槽 / enums 函数对象即可纯 JSON 表达；radio 需 enums，暂不开放） */
    jsonCompilable: boolean;
}

export const CONTROL_REGISTRY: Record<AutoFormItemType, ControlDescriptor> = {
    input: { selectLike: false, jsonCompilable: true },
    password: { selectLike: false, jsonCompilable: true },
    number: { selectLike: false, jsonCompilable: true },
    textarea: { selectLike: false, jsonCompilable: true },
    select: { selectLike: true, jsonCompilable: true },
    enum: { selectLike: true, jsonCompilable: false },
    radio: { selectLike: true, jsonCompilable: false },
    switch: { selectLike: false, jsonCompilable: true },
    date: { selectLike: true, jsonCompilable: true },
    datetime: { selectLike: true, jsonCompilable: true },
    time: { selectLike: true, jsonCompilable: true },
    monaco: { selectLike: false, jsonCompilable: true },
    tags: { selectLike: false, jsonCompilable: true },
    divider: { selectLike: false, jsonCompilable: true },
    group: { selectLike: false, jsonCompilable: true },
    custom: { selectLike: false, jsonCompilable: false },
};

// ── 工具函数 ────────────────────────────────────────────────────

/** 判断字段是否为“选择类”控件（用于生成 请选择{label} 提示语） */
export const isSelectLikeItem = (item: AutoFormItem): boolean => {
    return CONTROL_REGISTRY[item.type ?? 'input'].selectLike;
};

// ── 对外契约与 Tab 分组布局 ────────────────────────────────────

/**
 * AutoForm 对外暴露的方法契约
 *
 * 业务侧用 useTemplateRef<AutoFormInstance>('xxxFormRef') 统一声明表单 ref，
 * 避免各调用点手写结构体类型在契约变更时多处漂移。
 */
export interface AutoFormInstance {
    /** 触发全量校验（校验失败时 reject） */
    validate: (...args: unknown[]) => Promise<unknown>;
    /** 触发指定字段校验（缺省全部字段），向导式分步场景只校验当前步字段；底层实现不支持时为 undefined */
    validateField?: (props?: string | string[]) => Promise<unknown>;
    /** 重置字段到初始值并清除校验状态 */
    resetFields: () => void;
    /** 清除校验状态 */
    clearValidate: () => void;
    /** 触发统一提交流程（校验 → confirmApi/onConfirm → 成功提示 → submitted → 关闭），仅 Dialog/Drawer 宿主暴露，供 #footer 自定义确认按钮使用 */
    submit?: () => Promise<void>;
    /** 是否处于提交流程中（含校验期与提交请求期），仅 Dialog/Drawer 宿主暴露，供 #footer 自定义确认按钮 loading */
    submitting?: boolean;
}

/**
 * Tab 页签配置（对齐 tokhub TabConfig）
 *
 * 用于 AutoForm / AutoFormDialog / AutoFormDrawer 的 tabs prop，将字段按页签分组渲染。
 * 所有 Tab 共享同一表单数据与校验（非懒渲染，未激活 Tab 的字段同样参与校验）。
 */
export interface AutoFormTab {
    /** Tab 唯一标识 */
    name: string;

    /** Tab 显示标签 i18n key */
    label: string;

    /** Tab 图标名（SvgIcon，可选） */
    icon?: string;

    /** Tab 禁用（支持根据表单值动态计算，如向导式步骤前置条件） */
    disabled?: boolean | ((form: AutoFormData) => boolean);

    /** 该 Tab 下的字段配置 */
    items: AutoFormItem[];
}

// ── 嵌套路径读写（prop 支持点分路径，如 'extra.qywxUserId'） ──

/** 按点分路径读取嵌套值，中间节点不存在时返回 undefined */
export const getNestedValue = (obj: AutoFormData, path: string): unknown => {
    return path.split('.').reduce<unknown>((cur, key) => (cur == null ? undefined : (cur as AutoFormData)[key]), obj);
};

/** 按点分路径写入嵌套值，中间节点不存在时自动创建空对象 */
export const setNestedValue = (obj: AutoFormData, path: string, value: unknown): void => {
    const keys = path.split('.');
    let cur = obj;
    for (let i = 0; i < keys.length - 1; i++) {
        const key = keys[i];
        if (typeof cur[key] !== 'object' || cur[key] === null) {
            cur[key] = {};
        }
        cur = cur[key];
    }
    cur[keys[keys.length - 1]] = value;
};

/** 判断 prop 是否为嵌套路径（含点分符） */
export const isNestedPath = (prop: string | undefined): boolean => !!prop?.includes('.');

/**
 * 根据字段配置构建默认表单数据（应用各字段 defaultValue）
 *
 * 用于新增场景初始化表单，编辑场景直接使用回填数据。
 */
export const buildDefaultForm = (items: AutoFormItem[]): AutoFormData => {
    const form: AutoFormData = {};
    for (const item of items) {
        if (!item.prop) {
            continue;
        }
        if (item.defaultValue !== undefined) {
            setNestedValue(form, item.prop, item.defaultValue);
        } else if (item.multiple) {
            // 多选控件缺省值为数组，避免 undefined 传入 el-select multiple 产生异常
            setNestedValue(form, item.prop, []);
        }
    }
    return form;
};
