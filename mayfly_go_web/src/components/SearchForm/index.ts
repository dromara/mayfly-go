import { VNode } from 'vue';

export type FieldNamesProps = {
    label: string;
    value: string;
    children?: string;
};

export type SearchItemType =
    | 'input'
    | 'input-number'
    | 'select'
    | 'select-v2'
    | 'tree-select'
    | 'cascader'
    | 'date-picker'
    | 'time-picker'
    | 'time-select'
    | 'switch'
    | 'slider';

/**
 * 搜索项
 */
export class SearchItem {
    /**
     * 属性字段
     */
    prop: string;

    /**
     * 当前项搜索框的 label
     */
    label: string;

    /**
     * 表单项类型，input、select、date等
     */
    type: SearchItemType;

    /**
     * select等组件的可选值
     */
    options: any;

    /**
     * 插槽名
     */
    slot: string;

    props?: any; // 搜索项参数，根据 element plus 官方文档来传递，该属性所有值会透传到组件

    tooltip?: string; // 搜索提示

    span?: number; // 搜索项所占用的列数，默认为 1 列

    offset?: number; // 搜索字段左侧偏移列数

    fieldNames: FieldNamesProps; // 指定 label && value && children 的 key 值，用于select等类型组件

    render?: (scope: any) => VNode; // 自定义搜索内容渲染（tsx语法）

    constructor(prop: string, label: string) {
        this.prop = prop;
        this.label = label;
    }

    static new(prop: string, label: string): SearchItem {
        return new SearchItem(prop, label);
    }

    static text(prop: string, label: string): SearchItem {
        const tq = new SearchItem(prop, label);
        tq.type = 'input';
        return tq;
    }

    static select(prop: string, label: string): SearchItem {
        const tq = new SearchItem(prop, label);
        tq.type = 'select';
        return tq;
    }

    static date(prop: string, label: string): SearchItem {
        const tq = new SearchItem(prop, label);
        tq.type = 'date-picker';
        return tq;
    }

    static slot(prop: string, label: string, slotName: string): SearchItem {
        const tq = new SearchItem(prop, label);
        tq.slot = slotName;
        return tq;
    }

    withSpan(span: number): SearchItem {
        this.span = span;
        return this;
    }

    /**
     * 设置枚举值用于选择等
     * @param enumValues 枚举值对象
     * @returns
     */
    withEnum(enumValues: any): SearchItem {
        this.options = Object.values(enumValues);
        return this;
    }

    setOptions(options: any): SearchItem {
        this.options = options;
        return this;
    }
}
