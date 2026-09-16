/** 标签视图对象 */
export interface LabelVO {
    id: number;
    labelKey: string;
    labelValue: string;
    extra?: {
        color?: string;
        description?: string;
    };
    createTime: string;
}

/** 标签查询条件 */
export interface LabelQuery {
    labelKey?: string;
    pageNum?: number;
    pageSize?: number;
}

/** 标签表单 */
export interface LabelForm {
    id?: number;
    labelKey: string;
    labelValue: string;
    extra?: {
        color?: string;
        description?: string;
    };
}

/** 标签绑定视图对象 */
export interface LabelBindingVO {
    labelId: number;
    labelKey: string;
    labelValue: string;
}

/** 标签值明细（颜色/描述归属于具体的 key+value 标签） */
export interface LabelValueDetail {
    value: string;
    description?: string;
    color?: string;
}

/** 自动补全项 */
export interface LabelAutocompleteItem {
    key: string;
    values: string[];
    description?: string;
    color?: string;
    /** 值级别明细，与 values 同序 */
    valueDetails?: LabelValueDetail[];
}

/** 标签选择器值 */
export interface LabelSelectValue {
    key: string;
    value: string;
}
