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
     * 弹性列：保留 min-width 参与 el-table fit 的容器富余摊派。
     * 页面声明了弹性列时，其余数值 minWidth 列在首轮定型时固化为 width（贴内容不摊派），
     * 富余集中到弹性列——长文本主识别列（任务名/描述类）是天然候选。
     */
    flexible: boolean = false;

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
    typeParam?: Record<string, EnumValue>;

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

    /**
     * 声明本列为弹性列：保留 min-width 吸收容器富余，其余数值 minWidth 列随之固化为 width。
     * 未声明时全员 min-width 均摊（el-table 既有默认）。
     */
    setFlexible(): TableColumn {
        this.flexible = true;
        return this;
    }

    /**
     * 声明插槽列的渲染文本提取器（供自动计宽打底）。
     * 插槽列的实际渲染形态由页面模板决定，与原始字段值往往不同
     * （如单元格渲染 `${dbName}.${tableName}` 而数据里是两个独立字段），
     * 默认按原始字段值计宽会失真，需显式声明渲染文本。
     * 文本估算只覆盖文字部分；图标/tag 等渲染占位由渲染后的 DOM 实测回填（refineMinWidthByDom）自动补齐，
     * 无需声明图标宽度。
     * @param getText 从行数据提取单元格实际渲染的文本；T 由回调参数标注推断，页面侧无需断言
     */
    setSlotText<T>(getText: (rowData: T) => string): TableColumn {
        this.measureContentWidth = (rowData: Record<string, unknown>) => getTextWidth(getText(rowData as T));
        return this;
    }

    /**
     * 渲染后按单元格真实内容占位宽向上修正 minWidth。
     * 入参是渲染后的整块内容实测宽（文本+图标+间隙等，取自溢出叶子元素的 scrollWidth），
     * 无需关心插槽里渲染了什么；
     * 隐藏（display:none）时量到 0，直接跳过，不把隐藏态量到的错误值写进列宽。
     * 仅向上修正：文本估算已给出的宽度不会被实测缩小。
     */
    refineMinWidthByDom(cellScrollWidth: number) {
        if (!cellScrollWidth || cellScrollWidth <= 0) {
            return;
        }
        // 显式声明的字符串宽（如 '10%'）是调用方的明确意图，不参与 DOM 回填；
        // 未初始化（undefined）的自动计宽列按 0 起步参与回填
        if (typeof this.minWidth === 'string') {
            return;
        }
        const measured = Math.min(cellScrollWidth + CELL_PADDING, MAX_AUTO_WIDTH);
        // 首轮定型已把 minWidth 固化为 width 的列（紧凑贴内容），生效约束是 width，
        // 只修 minWidth 不改变渲染宽度，须同步向上修 width
        if (typeof this.width === 'number') {
            if (measured > this.width) {
                this.width = measured;
            }
            return;
        }
        if (measured > (this.minWidth ?? 0)) {
            this.minWidth = measured;
        }
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
    typeTag(param: Record<string, EnumValue>): TableColumn {
        this.type = 'tag';
        this.typeParam = param;
        // tag 展示文本为枚举翻译后的标签，且渲染为带边框/内边距的 el-tag，故按「标签文本 + 标签外框」计宽。
        // 不能复用 formatFunc：渲染器需将原始枚举值传给 enum-tag 自行解析，formatFunc 会污染该入参。
        this.measureContentWidth = (rowData: Record<string, unknown>) => {
            const value = this.getValueByData(rowData); // tag 列无 formatFunc，等价于原始枚举值，与渲染入参一致
            return getTextWidth(i18n.global.t(EnumValue.getLabelByValue(param, value as string | number))) + TAG_CHROME_WIDTH;
        };
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
    autoCalculateMinWidth(tableData: Record<string, unknown>[]) {
        if (!tableData || tableData.length === 0) {
            return 0;
        }

        // 计宽策略：列自带渲染宽度解析器(如 tag 列)则用之，否则按「格式化后的展示文本」测宽
        const measure =
            this.measureContentWidth ??
            ((rowData: Record<string, unknown>) => {
                const value = this.getValueByData(rowData);
                const text = value !== null && typeof value === 'object' ? JSON.stringify(value) : String(value ?? '');
                // 超长文本（如 JSON 大字段）精确量宽代价高，且其宽度必超上限被封顶，
                // 最窄字符 12px 字号下也 >3px，200 字符宽必 > MAX_AUTO_WIDTH，直接按上限计不再量宽
                return text.length > 200 ? MAX_AUTO_WIDTH : getTextWidth(text);
            });

        let maxContentWidth = 0;
        for (const rowData of tableData) {
            // 自带计宽扩展点的列（插槽列/tag 列）量什么由提取器自己决定，
            // 渲染文本可能来自任意字段甚至多字段拼接，不能按 prop 是否有值过滤，
            // 否则插槽列（行数据里往往没有 prop 对应字段）会被整行跳过、扩展点形同虚设；
            // 默认路径仍按原始字段值过滤空值（空内容不参与计宽）
            if (!this.measureContentWidth) {
                const value = getValueByPath(rowData, this.prop);
                if (value === null || value === undefined || value === '') {
                    continue;
                }
            }
            maxContentWidth = Math.max(maxContentWidth, measure(rowData));
        }

        // 内容宽度需加上单元格内间距，再与表头宽度取较大值
        const contentWidth = maxContentWidth + CELL_PADDING;
        const flexWidth = Math.max(contentWidth, headerMinWidth(this.label));
        // 上限封顶；图标/tag 等估算覆盖不到的渲染占位由渲染后的 DOM 实测回填（refineMinWidthByDom）精修
        this.minWidth = Math.min(flexWidth, MAX_AUTO_WIDTH);
        // 紧凑固化列的生效约束是 width：估算结果同步向上（基线不被后续估算缩小）
        if (typeof this.width === 'number' && this.minWidth > this.width) {
            this.width = this.minWidth;
        }
        return this.minWidth;
    }

    /**
     * 仅按表头文本估算 minWidth（无数据时的空态展示用）：内容估不到，
     * 表头宽度让空列表的表头有合理列宽；不定型，首轮非空数据仍按内容重估。
     */
    autoCalculateMinWidthByHeader() {
        if (typeof this.minWidth === 'string') {
            return;
        }
        this.minWidth = headerMinWidth(this.label);
        if (typeof this.width === 'number' && this.minWidth > this.width) {
            this.width = this.minWidth;
        }
    }
}

/** 表头文本估算 minWidth：内容估不到时的列宽下限（空态展示与内容估算共用） */
function headerMinWidth(label: string): number {
    return Math.min(getTextWidth(i18n.global.t(label)) + HEADER_PADDING, MAX_AUTO_WIDTH);
}
