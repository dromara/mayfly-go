import EnumValue from '@/common/Enum';
import { formatDate } from '@/common/utils/format';
import { getValueByPath } from '@/common/utils/object';
import { getTextWidth } from '@/common/utils/string';
import { i18n } from '@/i18n';

// 列宽自动计算相关常量
const CELL_PADDING = 30; // 单元格左右内间距等
const HEADER_PADDING = 60; // 表头文本计宽的额外留白
const MAX_AUTO_WIDTH = 400; // 自动计宽上限
const TAG_CHROME_WIDTH = 22; // el-tag 相对纯文本额外占用的边框 + 内边距

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
     * 是否为插槽，若slotName为空则插槽名为prop属性名
     */
    slot: boolean = false;

    /**
     * 插槽名，
     */
    slotName: string = '';

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
    typeParam: unknown;

    /**
     * 列宽自动计算的计宽扩展点：计算某一行该列「实际渲染内容」所占宽度(px)。
     * 默认按「格式化后的展示文本」测宽（见 autoCalculateMinWidth）；若某列的渲染形态与展示文本
     * 差异较大（如 tag 列：展示文本需按枚举翻译，且渲染为带边框/内边距的标签），则由该列在构造时
     * 覆写此函数即可，无需改动通用列宽算法（开闭原则）。入参为整行数据，与渲染器入参保持一致。
     */
    measureContentWidth?: (rowData: Record<string, unknown>) => number;

    width: number | string;

    fixed: string | boolean;

    align: string = 'left';

    /**
     * 指定格式化函数对原始值进行格式化，如时间格式化等
     * param1: data, param2: prop
     */
    formatFunc?: (data: any, prop: string) => unknown;

    /**
     * 是否显示该列,1显示 0不显示
     */
    show: number = 1;

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
    getValueByData(rowData: Record<string, unknown>) {
        if (this.formatFunc) {
            return this.formatFunc(rowData, this.prop);
        }
        return getValueByPath(rowData, this.prop);
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
    typeTag(param: unknown): TableColumn {
        this.type = 'tag';
        this.typeParam = param;
        // tag 展示文本为枚举翻译后的标签，且渲染为带边框/内边距的 el-tag，故按「标签文本 + 标签外框」计宽。
        // 不能复用 formatFunc：渲染器需将原始枚举值传给 enum-tag 自行解析，formatFunc 会污染该入参。
        const enums = param as Record<string, EnumValue>;
        this.measureContentWidth = (rowData: Record<string, unknown>) => {
            const value = this.getValueByData(rowData); // tag 列无 formatFunc，等价于原始枚举值，与渲染入参一致
            return getTextWidth(i18n.global.t(EnumValue.getLabelByValue(enums, value as string | number))) + TAG_CHROME_WIDTH;
        };
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
    isSlot(slotName: string = ''): TableColumn {
        this.slot = true;
        this.slotName = slotName;
        return this;
    }

    /**
     * 设置该列的格式化回调函数
     * @param func 格式化回调函数(参数为 -> data: 该行对应的数据，prop: 该列对应的prop属性值)
     * @returns
     */
    setFormatFunc(func: (data: any, prop: string) => unknown): TableColumn {
        this.formatFunc = func;
        return this;
    }

    /**
     * 为时间字段，则使用默认时间格式函数
     * @returns this
     */
    isTime(): TableColumn {
        this.setFormatFunc((data: Record<string, unknown>, prop: string) => {
            return formatDate(getValueByPath(data, prop) as string | number | Date);
        });
        return this;
    }

    /**
     * 标识该列枚举类，需进行枚举值转换
     * @returns this
     */
    isEnum(enums: Record<string, EnumValue>): TableColumn {
        this.setFormatFunc((data: Record<string, unknown>, prop: string) => {
            return i18n.global.t(EnumValue.getLabelByValue(enums, getValueByPath(data, prop) as string | number));
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
     * 自动计算该列最小宽度：遍历数据行取「实际渲染内容」最大宽度，与表头宽度取较大值后设置上限。
     * 渲染内容的度量委托给 measureContentWidth（列可覆写），默认按格式化后的展示文本测宽，
     * 因此新增特殊渲染形态的列只需提供自己的 measureContentWidth，无需改动本算法。
     * @param tableData 表数据
     */
    autoCalculateMinWidth = (tableData: Record<string, unknown>[]) => {
        if (!tableData || tableData.length === 0) {
            return 0;
        }

        // 计宽策略：列自带渲染宽度解析器(如 tag 列)则用之，否则按「格式化后的展示文本」测宽
        const measure =
            this.measureContentWidth ??
            ((rowData: Record<string, unknown>) => {
                const value = this.getValueByData(rowData);
                const text = value !== null && typeof value === 'object' ? JSON.stringify(value) : String(value ?? '');
                return getTextWidth(text);
            });

        let maxContentWidth = 0;
        for (const rowData of tableData) {
            const value = getValueByPath(rowData, this.prop);
            // 空内容不参与计宽（0 为有效展示值，需参与）
            if (value === null || value === undefined || value === '') {
                continue;
            }
            maxContentWidth = Math.max(maxContentWidth, measure(rowData));
        }

        // 内容宽度需加上单元格内间距，再与表头宽度取较大值
        const contentWidth = maxContentWidth + CELL_PADDING;
        const columnWidth = getTextWidth(i18n.global.t(this.label)) + HEADER_PADDING;
        const flexWidth = Math.max(contentWidth, columnWidth);
        // 设置上限并累加需要额外增加的宽度
        this.minWidth = Math.min(flexWidth, MAX_AUTO_WIDTH) + this.addWidth;
    };
}
