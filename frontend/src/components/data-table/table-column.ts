import EnumValue from '@/common/Enum';
import { formatDate } from '@/common/utils/format';
import { getValueByPath } from '@/common/utils/object';
import { getTextWidth } from '@/common/utils/string';
import { i18n } from '@/i18n';
import { CELL_PADDING, HEADER_PADDING, MAX_AUTO_WIDTH, TAG_CHROME_WIDTH } from './constants';
import { actionsColumnWidth, type ActionItem, type ActionOptions } from './actions';

/**
 * 表格列模型：一份声明同时驱动「渲染」与「自适应列宽」。
 *
 * 链式 setter（setActions/setSlotText/typeTag/setFlexible/setAddWidth…）把「这一列长什么样、要多宽」
 * 收敛到列自身；通用宽度算法（autoCalculateMinWidth / refineMinWidthByDom）不认识任何具体渲染形态，
 * 特殊形态由该列通过 measureContentWidth 扩展点自带算法（开闭原则）。
 */
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
     * 额外增量宽度（px）：叠加在自动估算的 minWidth 之上。用于「离线文本估算量不到、且 DOM 回填又不兜底」
     * 的定宽渲染占位——典型是插槽里的定宽交互控件（el-switch 圆钮、el-select 下拉箭头等）：
     * 这类控件在列被 el-table fit 撑宽时不溢出，「只量溢出」的回填那一帧量不到它，随后列塌回 min-width 就被裁。
     * 由该列显式声明控件相对文字估算的额外占位，让离线估算直接给出安全宽（也避免依赖回填导致的切换跳变）。
     */
    addWidth: number = 0;

    /**
     * 是否为插槽，若slotName为空则插槽名为prop属性名
     */
    slot: boolean = false;

    /**
     * 行内操作项：非空则该列由表格内置的 RowActions 渲染（无需页面提供同名插槽），列宽也从它推出
     */
    actions?: ActionItem[];

    /**
     * 行内操作列的展示选项（上限/图标模式/尺寸/外观）。
     * 由 setActions 存下并整体下发给 RowActions：计宽与渲染读同一个对象，
     * 逐个字段开 prop 会让以后每加一个选项都要改三层（字段 + prop + 下发）。
     */
    actionsOptions?: ActionOptions;

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
     * 新增 tag类型，用于枚举值转换后用tag进行展示（内置渲染器见 cellRenderers 注册表）
     *
     */
    type?: string;

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

    // 列固定宽；未设置（undefined）时由自适应列宽按 minWidth 估算/回填/固化
    width?: number | string;

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

    /**
     * 表头帮助提示的 i18n key：非空则该列表头渲染为「文案 + 问号 tooltip」，用于 ps/指标类 jargon 列（VSZ/RSS/STAT 等）。
     * 列宽离线估算仍只按文案量；问号图标/排序箭头等「文字之外的占位」由渲染后的表头实测回填
     * （columnWidthMeasure.measureHeaderCellWidth）自适应补齐——HEADER_PADDING 仅 30px 并不覆盖图标，勿删该回填链路。
     */
    headerTip?: string;

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
     * 无需声明图标宽度。操作列不用本方法，走 setActions（声明即渲染）。
     * @param getText 从行数据提取单元格实际渲染的文本；T 由回调参数标注推断，页面侧无需断言
     */
    setSlotText<T>(getText: (rowData: T) => string): TableColumn {
        this.measureContentWidth = (rowData: Record<string, unknown>) => getTextWidth(getText(rowData as T));
        return this;
    }

    /**
     * 声明该列在自动估算宽度之上再叠加的增量宽度（px）：用于定宽交互控件等「文本估不到、DOM 回填又不兜底」
     * 的渲染占位（如插槽里的 el-switch 圆钮、el-select 下拉箭头）。见 addWidth 字段说明。
     */
    setAddWidth(addWidth: number): TableColumn {
        this.addWidth = addWidth;
        return this;
    }

    /**
     * 指定该列为表格内置的行内操作列，由 RowActions 统一渲染，页面不再写按钮模板与宽度。
     * 展示形态（图标/图标+文字/文字）全部从 `items` + `options` 推出，见 ActionItem。
     * 计宽口径与渲染共用同一份声明：权限不足的按钮不计，互斥组只取最宽一格，
     * 超出内联上限的尾巴换成「更多」那一格（含下拉箭头占位）；
     * 行级 show 按并集计入，个别行因少一个按钮而不折叠时会退到渲染后的 DOM 实测回填继续加宽，不会裁按钮。
     * 格与格之间统一用 flex gap 而不画分隔线（全系统一致），所以间距与 RowActions 的 gap 必须同取 ACTION_BUTTON_GAP。
     * 泛型 Row：页面写 `setActions<MachineVO>([...])` 绑定一次，本列所有行回调即拿到 MachineVO 类型与补全；不写则默认 any。
     */
    setActions<Row = any>(items: ActionItem<Row>[], options: ActionOptions = {}): TableColumn {
        if (import.meta.env.DEV) {
            // 重复 key 会让「更多」菜单按 key 反查时命中错项（静默错路由），开发期直接报出来
            const duplicated = items.map((item) => item.key).filter((key, index, all) => all.indexOf(key) !== index);
            if (duplicated.length) {
                console.warn(`[data-table] 操作列存在重复 key（折叠菜单会路由到错的动作）: ${[...new Set(duplicated)].join(', ')}`);
            }
        }
        this.actions = items;
        this.actionsOptions = options;
        // 与裸表格的 actionColumnProps 共用 actionsColumnWidth：宽度算法只允许一份
        this.measureContentWidth = () => actionsColumnWidth(items, options);
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
        // 显式声明的字符串宽（如 '250px'；el-table 仅认数字前缀 px、不支持百分比）是调用方的明确意图，不参与 DOM 回填；
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
     * 声明该列可排序（客户端排序，透传给 el-table 的 sortable）。
     */
    setSortable(): TableColumn {
        this.sortable = true;
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
     * 声明该列表头的帮助提示（i18n key），渲染为「文案 + 问号 tooltip」。
     */
    setHeaderTip(tip: string): TableColumn {
        this.headerTip = tip;
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
        // 上限封顶后叠加该列声明的定宽控件增量（addWidth）：图标/tag 等可被 DOM 回填兜底的占位无需声明，
        // 但定宽交互控件（el-switch 圆钮等）回填量不到，靠此处显式预留
        this.minWidth = Math.min(flexWidth, MAX_AUTO_WIDTH) + this.addWidth;
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
