import { dateFormat } from '@/common/utils/date';
import { getTextWidth } from '@/common/utils/string';

export class TableColumn {

    /**
     * 属性字段
     */
    prop: string;

    /**
     * 显示表头
     */
    label: string;

    /**
     * 是否自动计算宽度
     */
    autoWidth: boolean = true;

    /**
     * 自动计算宽度时，累加该值（可能列值会进行转换添加图标等，宽度需要比计算出来的更宽些）
     */
    addWidth: number = 0;

    /**      
     * 最小宽度
     */
    minWidth: number | string;

    /**
     * 是否插槽，是的话插槽名则为prop属性名
     */
    slot: boolean = false;

    showOverflowTooltip: boolean = true;

    sortable: boolean = false;

    type: string;

    width: number | string;

    fixed: any;

    align: string = "center"

    /**
     * 指定格式化函数对原始值进行格式化，如时间格式化等
     * param1: data, param2: prop
     */
    formatFunc: Function

    /**
     * 是否显示该列
     */
    show: boolean = true

    constructor(prop: string, label: string) {
        this.prop = prop;
        this.label = label;
    }

    /**
     * 获取该列在指定行数据中的值
     * @param rowData 该行对应的数据
     * @returns 该列对应的值
     */
    getValueByData(rowData: any) {
        if (this.formatFunc) {
            return this.formatFunc(rowData, this.prop);
        }
        return rowData[this.prop];
    }

    static new(prop: string, label: string): TableColumn {
        return new TableColumn(prop, label)
    }

    setMinWidth(minWidth: number | string): TableColumn {
        this.minWidth = minWidth
        this.autoWidth = false;
        return this;
    }

    setAddWidth(addWidth: number): TableColumn {
        this.addWidth = addWidth
        return this;
    }

    /**
     * 标识该列为插槽
     * @returns this
     */
    isSlot(): TableColumn {
        this.slot = true
        return this;
    }

    /**
     * 设置该列的格式化回调函数
     * @param func 格式化回调函数(参数为 -> data: 该行对应的数据，prop: 该列对应的prop属性值)
     * @returns 
     */
    setFormatFunc(func: Function): TableColumn {
        this.formatFunc = func;
        return this;
    }

    /**
     * 为时间字段，则使用默认时间格式函数
     * @returns this
     */
    isTime(): TableColumn {
        this.setFormatFunc((data: any, prop: string) => {
            return dateFormat(data[prop])
        })
        return this;
    }

    fixedRight(): TableColumn {
        this.fixed = "right";
        return this;
    }

    fixedLeft(): TableColumn {
        this.fixed = "left";
        return this;
    }


    /**
     * 自动计算最小宽度
     * @param str 字符串
     * @param tableData 表数据
     * @param label 表头label也参与宽度计算
     * @returns 列宽度
     */
    autoCalculateMinWidth = (tableData: any) => {
        const prop = this.prop
        const label = this.label

        if (!tableData || !tableData.length || tableData.length === 0 || tableData === undefined) {
            return 0;
        }

        let maxWidthText = ""
        let maxWidthValue
        // 为了兼容formatFunc格式化回调函数
        let maxData
        // 获取该列中最长的数据(内容)
        for (let i = 0; i < tableData.length; i++) {
            let nowData = tableData[i]
            let nowValue = nowData[prop]
            if (!nowValue) {
                continue;
            }
            // 转为字符串比较长度
            let nowText = nowValue + "";
            if (nowText.length > maxWidthText.length) {
                maxWidthText = nowText;
                maxWidthValue = nowValue;
                maxData = nowData;
            }
        }
        if (this.formatFunc && maxWidthValue) {
            maxWidthText = this.formatFunc(maxData, prop) + ""
        }
        // 需要加上表格的内间距等，视情况加
        const contentWidth: number = getTextWidth(maxWidthText) + 30;
        // 获取label的宽度，取较大的宽度
        const columnWidth: number = getTextWidth(label) + 60;
        const flexWidth: number = contentWidth > columnWidth ? contentWidth : columnWidth;
        // 设置上限与累加需要额外增加的宽度
        this.minWidth = (flexWidth > 400 ? 400 : flexWidth) + this.addWidth;
    };
}

export class TableQuery {

    /**
     * 属性字段
     */
    prop: string;

    /**
     * 显示表头
     */
    label: string;

    /**
     * 查询类型，text、select、date
     */
    type: string;

    /**
     * select可选值
     */
    options: any;

    /**
     * 插槽名
     */
    slot: string;


    constructor(prop: string, label: string) {
        this.prop = prop;
        this.label = label;
    }

    static new(prop: string, label: string): TableQuery {
        return new TableQuery(prop, label)
    }

    static text(prop: string, label: string): TableQuery {
        const tq = new TableQuery(prop, label)
        tq.type = 'text';
        return tq;
    }

    static select(prop: string, label: string): TableQuery {
        const tq = new TableQuery(prop, label)
        tq.type = 'select';
        return tq;
    }

    static date(prop: string, label: string): TableQuery {
        const tq = new TableQuery(prop, label)
        tq.type = 'date';
        return tq;
    }

    static slot(prop: string, label: string, slotName: string): TableQuery {
        const tq = new TableQuery(prop, label)
        tq.slot = slotName;
        return tq;
    }

    setOptions(options: any): TableQuery {
        this.options = options;
        return this
    }
}