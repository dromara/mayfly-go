import Api from '@/common/Api';
import { VNode, ref, toValue } from 'vue';

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
 * 表单组件可选项的api信息
 */
export class OptionsApi {
    /**
     * 请求获取options的api
     */
    api: Api;

    /**
     * 请求参数
     */
    params: any;

    /**
     * 是否立即执行，否则在组件focus事件中获取
     */
    immediate: boolean = false;

    /**
     * 是否只获取一次，即若以获取则不继续调用该api
     */
    once: boolean = true;

    /**
     * 转换函数，主要用于将响应的api结果转换为满足组件options的结构
     */
    convertFn: (apiResp: any) => any;

    // remote: boolean = false;

    /**
     * 远程方法参数属性字段，存在该值，则说明使用remote-method进行远程搜索
     */
    remoteMethodParamProp: string;

    withConvertFn(fn: (apiResp: any) => any) {
        this.convertFn = fn;
        return this;
    }

    /**
     *  立即获取该可选值
     * @returns
     */
    withImmediate() {
        this.immediate = true;
        return this;
    }

    /**
     * 设为非一次性api，即每次组件focus获取的时候都允许重新获取options
     * @returns this
     */
    withNoOnce() {
        this.once = false;
        return this;
    }

    /**
     * 是否使用select的remote方式远程搜索调用
     * @param remoteReqParamKey  remote请求参数对应的prop，需要将输入的value赋值给params[paramProp]进行远程搜索
     */
    isRemote(paramProp: string) {
        this.remoteMethodParamProp = paramProp;
        return this;
    }

    /**
     * 调用api获取组件可选项
     * @returns 组件可选项信息
     */
    async getOptions() {
        let res = await this.api.request(toValue(this.params));
        if (this.convertFn) {
            res = this.convertFn(res);
        }
        return res;
    }

    static new(api: Api, params: any): OptionsApi {
        const oa = new OptionsApi();
        oa.api = api;
        oa.params = params;
        return oa;
    }
}

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
     * 获取可选项的api信息
     */
    optionsApi: OptionsApi;

    /**
     * 插槽名
     */
    slot: string;

    /**
     * 搜索项参数，根据 element plus 官方文档来传递，该属性所有值会透传到组件
     */
    props?: any;

    /**
     * 搜索项事件，根据 element plus 官方文档来传递，该属性所有值会透传到组件
     */
    events?: any;

    /**
     * 搜索提示
     */
    tooltip?: string;

    /**
     * 搜索项所占用的列数，默认为 1 列
     */
    span?: number;

    /**
     * 搜索字段左侧偏移列数
     */
    offset?: number;

    /**
     * 指定 label && value && children 的 key 值，用于select等类型组件
     */
    fieldNames: FieldNamesProps;

    /**
     * 自定义搜索内容渲染（tsx语法）
     */
    render?: (scope: any) => VNode;

    constructor(prop: string, label: string) {
        this.prop = prop;
        this.label = label;
    }

    static new(prop: string, label: string): SearchItem {
        return new SearchItem(prop, label);
    }

    static input(prop: string, label: string): SearchItem {
        const tq = new SearchItem(prop, label);
        tq.type = 'input';
        return tq;
    }

    static select(prop: string, label: string): SearchItem {
        const tq = new SearchItem(prop, label);
        tq.type = 'select';
        tq.withOneProps('filterable', true);
        return tq;
    }

    static datePicker(prop: string, label: string): SearchItem {
        const tq = new SearchItem(prop, label);
        tq.type = 'date-picker';
        return tq;
    }

    static slot(prop: string, label: string, slotName: string): SearchItem {
        const tq = new SearchItem(prop, label);
        tq.slot = slotName;
        return tq;
    }

    /**
     * 为组件设置一个props属性
     * @param propsKey 属性key
     * @param propsValue 属性value
     * @returns
     */
    withOneProps(propsKey: string, propsValue: any): SearchItem {
        if (!this.props) {
            this.props = {};
        }
        this.props[propsKey] = propsValue;
        return this;
    }

    /**
     * 为组件传递组件自身的props属性 (根据 element plus 官方文档来传递，该属性所有值会透传到组件)
     * @returns this
     */
    withProps(props: any = {}): SearchItem {
        this.props = props;
        return this;
    }

    /**
     * 为组件传递组件自身事件函数
     * @param event 事件名称
     * @param fn 事件处理函数
     * @returns
     */
    bindEvent(event: string, eventFn: any): SearchItem {
        if (!this.events) {
            this.events = {};
        }
        this.events[event] = eventFn;
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

    /**
     * 设置获取组件options可选项值的api配置
     * @param optionsApi 可选项api配置
     * @returns this
     */
    withOptionsApi(optionsApi: OptionsApi): SearchItem {
        this.optionsApi = optionsApi;
        // 使用api获取组件可选项需要将options转为响应式，否则组件无法响应式获取组件可选项
        this.options = ref(null);

        // 存在远程搜索请求参数prop，则为使用远程搜索可选项
        if (optionsApi.remoteMethodParamProp) {
            return this.withOneProps('remote', true).withOneProps('remote-method', async (value: any) => {
                if (!value) {
                    this.options.value = [];
                    return;
                }
                // 将输入的内容赋值为真实api请求参数中指定的属性字段
                optionsApi.params[optionsApi.remoteMethodParamProp] = value;
                this.options.value = await this.optionsApi.getOptions();
            });
        }

        // 立即执行，则直接调用api获取并赋值options
        if (this.optionsApi.immediate) {
            this.optionsApi.getOptions().then((res) => {
                this.options.value = res;
            });
        } else {
            // 注册focus事件，在触发focus时赋值options
            this.bindEvent('focus', async () => {
                if (!toValue(this.options) || !optionsApi.once) {
                    this.options.value = await this.optionsApi.getOptions();
                }
            });
        }
        return this;
    }

    withSpan(span: number): SearchItem {
        this.span = span;
        return this;
    }

    withOptions(options: any): SearchItem {
        this.options = options;
        return this;
    }

    /**
     * 赋值placeholder
     * @param val placeholder
     * @returns
     */
    withPlaceholder(val: string): SearchItem {
        return this.withOneProps('placeholder', val);
    }
}
