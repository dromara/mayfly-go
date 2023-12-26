import EnumValue from '@/common/Enum';
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
     * 自动计算宽度时，累加该值（可能列值会进行转换 如添加图标等，宽度需要比计算出来的更宽些）
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

    /**
     * 官方：对应列的类型。 如果设置了selection则显示多选框；
     * 如果设置了 index 则显示该行的索引（从 1 开始计算）；
     *
     * 新增 tag类型，用于枚举值转换后用tag进行展示
     *
     */
    type: string;

    /**
     * 类型展示需要的额外参数，如枚举转换的EnumValue值等
     */
    typeParam: any;

    width: number | string;

    fixed: any;

    align: string = 'left';

    /**
     * 指定格式化函数对原始值进行格式化，如时间格式化等
     * param1: data, param2: prop
     */
    formatFunc: Function;

    /**
     * 是否显示该列
     */
    show: boolean = true;

    /**
     * 是否展示美化按钮（主要用于美化json文本等）
     */
    isBeautify: boolean = false;

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
        return new TableColumn(prop, label);
    }

    noShowOverflowTooltip(): TableColumn {
        this.showOverflowTooltip = false;
        return this;
    }

    setMinWidth(minWidth: number | string): TableColumn {
        this.minWidth = minWidth;
        this.autoWidth = false;
        return this;
    }

    setAddWidth(addWidth: number): TableColumn {
        this.addWidth = addWidth;
        return this;
    }

    /**
     * 居中对齐
     * @returns this
     */
    alignCenter(): TableColumn {
        this.align = 'center';
        return this;
    }

    /**
     * 使用标签类型展示该列（用于枚举值友好展示）
     * @param param 枚举对象, 如AccountStatusEnum
     * @returns this
     */
    typeTag(param: any): TableColumn {
        this.type = 'tag';
        this.typeParam = param;
        return this;
    }

    typeText(): TableColumn {
        this.type = 'text';
        return this;
    }

    typeJson(): TableColumn {
        this.type = 'jsonText';
        return this;
    }

    /**
     * 标识该列为插槽
     * @returns this
     */
    isSlot(): TableColumn {
        this.slot = true;
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
            return dateFormat(data[prop]);
        });
        return this;
    }

    /**
     * 标识该列枚举类，需进行枚举值转换
     * @returns this
     */
    isEnum(enums: any): TableColumn {
        this.setFormatFunc((data: any, prop: string) => {
            return EnumValue.getLabelByValue(enums, data[prop]);
        });
        return this;
    }

    fixedRight(): TableColumn {
        this.fixed = 'right';
        return this;
    }

    fixedLeft(): TableColumn {
        this.fixed = 'left';
        return this;
    }

    canBeautify(): TableColumn {
        this.isBeautify = true;
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
        const prop = this.prop;
        const label = this.label;

        if (!tableData || !tableData.length || tableData.length === 0 || tableData === undefined) {
            return 0;
        }

        let maxWidthText = '';
        let maxWidthValue;
        // 为了兼容formatFunc格式化回调函数
        let maxData;
        // 获取该列中最长的数据(内容)
        for (let i = 0; i < tableData.length; i++) {
            let nowData = tableData[i];
            let nowValue = nowData[prop];
            if (!nowValue) {
                continue;
            }
            // 转为字符串比较长度
            let nowText = nowValue + '';
            if (nowText.length > maxWidthText.length) {
                maxWidthText = nowText;
                maxWidthValue = nowValue;
                maxData = nowData;
            }
        }
        if (this.formatFunc && maxWidthValue) {
            maxWidthText = this.formatFunc(maxData, prop) + '';
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
