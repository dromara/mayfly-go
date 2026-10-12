<template>
    <div class="h-full flex flex-col flex-1 overflow-hidden">
        <!-- 查询表单：常驻渲染 + grid rows 0fr↔1fr 高度过渡，展开收起时下方内容随高度平滑推移 -->
        <div class="search-form-collapse" :class="{ 'is-open': isShowSearch }">
            <!-- 内层承担裁剪：0fr 轨道的最小尺寸含 item 自身 padding/margin（会把塌缩态撑出空白带），故裁剪层必须零内边距 -->
            <div class="search-form-collapse-inner">
                <SearchForm :items="tableSearchItems" v-model="queryForm" :search="search" :reset="reset" :search-col="searchCol">
                    <!-- 遍历父组件传入的 solts 透传给子组件 -->
                    <template v-for="(_, key) in useSlots()" v-slot:[key]>
                        <slot :name="key"></slot>
                    </template>
                </SearchForm>
            </div>
        </div>

        <!-- 顶层列表页用 el-card 提供边框/留白；抽屉、弹窗等内嵌列表传 :card="false" 走裸容器，避免「卡片套卡片」 -->
        <component
            :is="props.card ? ElCard : 'div'"
            class="h-full"
            :class="props.card ? '' : 'flex flex-col'"
            :body-class="props.card ? 'h-full flex flex-col' : undefined"
        >
            <!-- 表格头部 操作按钮：与下方表格之间留一条呼吸带（.data-table__toolbar），避免工具条控件与表头黏成一片 -->
            <div class="data-table__toolbar flex flex-wrap items-center justify-between gap-y-2">
                <div>
                    <slot name="tableHeader" />
                </div>

                <slot v-if="toolButton" name="toolButton">
                    <div class="flex">
                        <!-- 简易单个搜索项：完整搜索表单展开时隐藏，避免双搜索入口同屏重复（渐进披露） -->
                        <transition name="pt-fade">
                            <div v-if="nowSearchItem && !isShowSearch" class="flex items-center gap-2 simple-search-bar">
                                <!-- 字段名即触发器（filter 惯例）：ghost 文字钮 + 开合旋转小箭头，无独立圆钮 -->
                                <el-dropdown
                                    v-if="props.searchItems?.length > 1"
                                    trigger="click"
                                    popper-class="data-table-field-menu"
                                    @visible-change="onFieldMenuVisible"
                                >
                                    <span class="simple-search-field" :title="$t('common.searchFields')" :aria-label="$t('common.searchFields')">
                                        {{ $t(nowSearchItem?.label) }}
                                        <SvgIcon :size="12" name="ArrowDown" class="simple-search-field-arrow" :class="{ 'is-open': fieldMenuOpen }" />
                                    </span>
                                    <template #dropdown>
                                        <el-dropdown-menu>
                                            <el-dropdown-item
                                                v-for="searchItem in searchItems"
                                                :key="searchItem.prop"
                                                :class="{ 'is-current': searchItem.prop === nowSearchItem?.prop }"
                                                @click="changeSimpleFormItem(searchItem)"
                                            >
                                                <span class="field-menu-check">
                                                    <SvgIcon v-if="searchItem.prop === nowSearchItem?.prop" name="Check" :size="12" />
                                                </span>
                                                {{ $t(searchItem.label) }}
                                            </el-dropdown-item>
                                        </el-dropdown-menu>
                                    </template>
                                </el-dropdown>
                                <span v-else class="simple-search-field is-static">{{ $t(nowSearchItem?.label) }}</span>

                                <el-form-item class="w-[200px]" :key="nowSearchItem.prop">
                                    <SearchFormItem
                                        @keyup.enter="searchFormItemKeyUpEnter"
                                        v-if="!nowSearchItem.slot"
                                        :item="nowSearchItem"
                                        v-model="queryForm[nowSearchItem.prop]"
                                    />

                                    <!-- 事件不能直接绑在 <slot> 上（Vue 会编成插槽作用域 props、不会挂到外部内容），故包一层真元素接住回车 -->
                                    <div v-else class="w-full" @keyup.enter="searchFormItemKeyUpEnter">
                                        <slot :name="nowSearchItem.slot"></slot>
                                    </div>
                                </el-form-item>
                            </div>
                        </transition>

                        <div class="ml-2">
                            <el-button
                                v-if="showToolButton('search') && searchItems?.length"
                                icon="Search"
                                circle
                                :title="$t('common.search')"
                                :aria-label="$t('common.search')"
                                @click="search"
                            />
                            
                            <!-- 箭头固定用 ArrowUp，展开态旋转 180° 过渡，避免图标硬切换 -->
                            <el-button
                                v-if="showToolButton('search') && props.searchItems?.length > 1"
                                circle
                                :title="$t(isShowSearch ? 'common.collapse' : 'common.expand')"
                                :aria-label="$t(isShowSearch ? 'common.collapse' : 'common.expand')"
                                @click="isShowSearch = !isShowSearch"
                            >
                                <SvgIcon :size="13" name="ArrowUp" class="search-toggle-icon" :class="{ 'is-open': isShowSearch }" />
                            </el-button>

                            <el-popover
                                v-if="showToolButton('setting')"
                                placement="bottom"
                                :title="$t('common.columnConfig')"
                                popper-class="data-table-column-config"
                                width="auto"
                                trigger="click"
                            >
                                <div class="column-config-item column-config-header">
                                    <el-checkbox
                                        v-model="allColumnsSelected"
                                        :indeterminate="someColumnsSelected && !allColumnsSelected"
                                        :label="$t('common.selectAll')"
                                    />
                                </div>
                                <div v-for="(item, index) in tableColumns" :key="index" class="column-config-item">
                                    <el-checkbox v-model="item.show" :label="$t(item.label)" :true-value="1" :false-value="0" />
                                </div>
                                <template #reference>
                                    <el-button
                                        icon="Operation"
                                        circle
                                        :size="props.size"
                                        :title="$t('common.columnConfig')"
                                        :aria-label="$t('common.columnConfig')"
                                    ></el-button>
                                </template>
                            </el-popover>
                        </div>
                    </div>
                </slot>
            </div>

            <!-- v-loading 挂在外层包裹（而非 v-show 隐藏的 el-table）：异步列表「假空不显示」阶段 el-table 隐藏，
                 若 loading 挂在 el-table 上则加载期间无任何反馈；挂包裹层才能既防列宽跳变又保留加载转圈 -->
            <div class="flex-1 overflow-auto" ref="tableWrapRef" v-loading="loading">
                <el-table
                    v-show="showTable"
                    ref="tableRef"
                    v-bind="$attrs"
                    height="100%"
                    @selection-change="handleSelectionChange"
                    :data="tableData"
                    highlight-current-row
                    :size="tableSize"
                    :border="border"
                >
                    <el-table-column v-if="props.showSelection" :selectable="selectable" type="selection" width="40" />

                    <template v-for="(item, index) in tableColumns">
                        <el-table-column
                            :key="index"
                            v-if="item.show"
                            :prop="item.prop"
                            :label="$t(item.label)"
                            :fixed="item.fixed"
                            :align="item.align"
                            :show-overflow-tooltip="item.showOverflowTooltip"
                            :min-width="item.minWidth"
                            :sortable="item.sortable || false"
                            :type="item.type"
                            :width="item.width"
                        >
                            <!-- 带帮助提示的表头（jargon 列如 VSZ/RSS/STAT）：文案 + 问号 tooltip；无 headerTip 时回落 :label -->
                            <template #header v-if="item.headerTip">
                                <span>{{ $t(item.label) }}</span>
                                <el-tooltip :content="$t(item.headerTip)" placement="top">
                                    <SvgIcon name="QuestionFilled" :size="13" class="ml-1 align-middle" />
                                </el-tooltip>
                            </template>

                            <!-- 行内操作列：按钮由表格统一渲染，页面只需声明 setActions -->
                            <template #default="scope" v-if="item.actions">
                                <RowActions :items="item.actions" :options="item.actionsOptions" :data="scope.row" />
                            </template>

                            <!-- 插槽：预留功能 -->
                            <template #default="scope" v-else-if="item.slot">
                                <slot :name="item.slotName ? item.slotName : item.prop" :data="scope.row"></slot>
                            </template>

                            <!-- 内置值渲染器：按列 type 从注册表解析（tag 等），新增内置形态改注册表不改本模板 -->
                            <template #default="scope" v-else-if="resolveCellRenderer(item)">
                                <component
                                    :is="resolveCellRenderer(item)"
                                    :item="item"
                                    :row="scope.row"
                                    :value="item.getValueByData(scope.row)"
                                    :size="props.size"
                                />
                            </template>

                            <template #default="scope" v-else>
                                <!-- 配置了美化文本按钮以及文本内容大于指定长度，则显示美化按钮 -->
                                <el-popover
                                    v-if="item.isBeautify && String(item.getValueByData(scope.row) ?? '').length > 35"
                                    effect="light"
                                    trigger="click"
                                    placement="top"
                                    width="600px"
                                >
                                    <template #default>
                                        <el-input :autosize="{ minRows: 3, maxRows: 15 }" disabled v-model="formatVal" type="textarea" />
                                    </template>
                                    <template #reference>
                                        <el-link
                                            @click="formatText(item.getValueByData(scope.row))"
                                            underline="never"
                                            type="success"
                                            icon="MagicStick"
                                            class="mr-1"
                                        ></el-link>
                                    </template>
                                </el-popover>

                                <span>{{ item.getValueByData(scope.row) }}</span>
                            </template>
                        </el-table-column>
                    </template>
                    <!-- 空态由页面自己说明：列表类界面的「暂无数据」不告诉用户这里能放什么、去哪儿创建 -->
                    <template #empty>
                        <slot name="empty">{{ $t('common.empty') }}</slot>
                    </template>
                </el-table>
            </div>

            <!-- 分页条：仅分页且确有取数 api 时出现；静态 :data 列表（无 api）自动隐藏；异步「假空不显示」阶段随表格隐藏，不留孤立分页器 -->
            <el-row v-if="showTable && props.pageable && !!props.pageApi" class="mt-4" justify="end">
                <el-pagination
                    :small="props.size == 'small'"
                    @current-change="pageNumChange"
                    @size-change="pageSizeChange"
                    layout="prev, pager, next, total, sizes"
                    :total="total"
                    v-model:current-page="queryForm.pageNum"
                    v-model:page-size="queryForm.pageSize"
                    :page-sizes="pageSizes"
                />
            </el-row>
        </component>
    </div>
</template>

<script lang="ts" setup>
import { computed, toRefs, watch, reactive, onMounted, Ref, ref, useSlots, useTemplateRef, toValue } from 'vue';
import { TableColumn } from './index';
import RowActions from './RowActions.vue';
import { useColumnAutoWidth } from './useColumnAutoWidth';
import { resolveCellRenderer } from './cellRenderers';
import { useThemeConfig } from '@/store/themeConfig';
import { storeToRefs } from 'pinia';
import Api from '@/common/Api';
import SearchForm from '@/components/data-table/SearchForm/index.vue';
import { SearchItem } from './SearchForm/index';
import SearchFormItem from './SearchForm/components/SearchFormItem.vue';
import SvgIcon from '@/components/svg-icon/index.vue';
import { usePageTable } from '@/hooks/usePageTable';
import { ElCard, ElInput, type TableInstance } from 'element-plus';

const emit = defineEmits(['pageSizeChange', 'pageNumChange']);

/**
 * el-table 只接受三个字面量尺寸，而 props.size 是宽 string。
 *
 * 原来在模板里写 as 断言（`:size="(props.size as 'large' | ...)"`），靠一层括号活着：
 * 格式化一旦吃掉括号，vue 模板解析就把竖线当过滤器读（vue/no-deprecated-filter），
 * 所以类型收敛放在脚本里，模板只留一个普通绑定
 */
const tableSize = computed(() => (props.size === 'large' || props.size === 'small' ? props.size : 'default'));

export interface DataTableProps {
    size?: string;
    pageApi?: Api<any, any>; // 请求表格数据的 api
    columns: TableColumn[]; // 列配置项  ==> 必传
    showSelection?: boolean;
    selectable?: (row: Record<string, unknown>) => boolean; // 是否可选
    pageable?: boolean; // 是否分页：true 走 pageApi 的分页返回（res.list/res.total）并显示分页条；false 或无 pageApi 时按整份列表渲染 ==> 非必传（默认 true）
    showSearch?: boolean; // 是否显示搜索表单
    data?: Record<string, unknown>[]; // 静态/受控列表数据：直接渲染该数组（响应式，上游过滤后自动更新），无需 pageApi 与分页 ==> 非必传
    lazy?: boolean; // 是否自动执行请求 api ==> 非必传（默认为false）
    beforeQueryFn?: (params: Record<string, unknown>) => Record<string, unknown>; // 执行查询时对查询参数进行处理，调整等
    dataHandlerFn?: (data: any) => any; // 数据处理回调函数，用于将请求回来的数据（分页时为 PageResult）二次加工处理等
    searchItems?: SearchItem[];
    border?: boolean; // 是否带有纵向边框 ==> 非必传（默认为false）
    card?: boolean; // 是否用 el-card 外壳包裹：顶层列表页保持 true；抽屉/弹窗内嵌列表传 false 走裸容器，避免「卡片套卡片」 ==> 非必传（默认为true）
    toolButton?: ('setting' | 'search')[] | boolean; // 是否显示表格功能按钮 ==> 非必传（默认为true）
    searchCol?: Record<string, number> | number; // 表格搜索项 每列占比配置 ==> 非必传 { xs: 1, sm: 2, md: 2, lg: 3, xl: 4 } | number 如 3
}

// 接受父组件参数，配置默认值
const props = withDefaults(defineProps<DataTableProps>(), {
    columns: () => [],
    pageable: true,
    showSelection: false,
    lazy: false,
    border: false,
    card: true,
    toolButton: true,
    showSearch: false,
    searchItems: () => [],
    searchCol: () => ({ xs: 1, sm: 3, md: 3, lg: 4, xl: 5 }),
});

const selectionData = defineModel<Record<string, unknown>[]>('selectionData', { default: () => [] });

// 查询表单参数 ==> 非必传（默认为{pageNum:1, pageSize: 0}）
const queryForm = defineModel<Record<string, unknown>>('queryForm', {
    default: () => ({
        pageNum: 1,
        pageSize: 0,
    }),
});

// table 实例
const tableRef = ref<TableInstance>();

// 接收 columns 并设置为响应式
const tableColumns = reactive<TableColumn[]>(props.columns);

// 列配置全选三态：与逐列复选框经 show 字段（1 显示 / 0 隐藏）双向联动
const allColumnsSelected = computed({
    get: () => tableColumns.length > 0 && tableColumns.every((c) => c.show === 1),
    set: (checked) => {
        const val = checked ? 1 : 0;
        tableColumns.forEach((c) => (c.show = val));
    },
});
const someColumnsSelected = computed(() => tableColumns.some((c) => c.show === 1));

// 接收 searchItems 并设置为响应式
const tableSearchItems = reactive<SearchItem[]>(props.searchItems);

const { themeConfig } = storeToRefs(useThemeConfig());

// 是否显示搜索模块
const isShowSearch = ref(props.showSearch);

// 控制 ToolButton 显示
const showToolButton = (key: 'setting' | 'search') => {
    return Array.isArray(props.toolButton) ? props.toolButton.includes(key) : props.toolButton;
};

const nowSearchItem: Ref<SearchItem | null> = ref<SearchItem | null>(null);

const showTable: Ref<boolean> = ref(false);

/**
 * 改变当前的搜索项
 * @param searchItem 当前点击的搜索项
 */
const changeSimpleFormItem = (searchItem: SearchItem) => {
    // 将之前的值置为空，避免因为只显示一个搜索项却搜索多个条件
    if (nowSearchItem.value) {
        queryForm.value[nowSearchItem.value.prop] = null;
    }
    nowSearchItem.value = searchItem;
};

// 字段下拉开合状态：驱动触发器箭头旋转过渡
const fieldMenuOpen = ref(false);
const onFieldMenuVisible = (visible: boolean) => {
    fieldMenuOpen.value = visible;
};

const pageSizeChange = (val: number) => {
    emit('pageSizeChange', val);
    handlePageSizeChange(val);
};
const pageNumChange = (val: number) => {
    emit('pageNumChange', val);
    handlePageNumChange(val);
};

let { tableData, total, loading, search, reset, getTableData, handlePageNumChange, handlePageSizeChange } = usePageTable(
    props.pageable,
    props.pageApi,
    queryForm,
    props.beforeQueryFn,
    props.dataHandlerFn
);

const state = reactive({
    pageSizes: [] as number[], // 可选每页显示的数据量
    // 输入框宽度
    formatVal: '', // 格式化后的值
});

const { pageSizes, formatVal } = toRefs(state);

/** 表格包裹层：DOM 实测单元格内容宽的取量入口 */
const tableWrapRef = useTemplateRef<HTMLElement>('tableWrapRef');

// 列宽自适应：首轮非空数据到达时估算打底（完成后回调显示表格防闪烁）→ 渲染后 DOM 回填精修 → 定型（翻页冻结）
useColumnAutoWidth({
    tableWrapRef,
    // 必须用模板实际渲染的 tableColumns（reactive 代理）而非原始 props.columns：
    // 页面以纯数组声明列时二者不是同一对象，写原始对象不触发代理 set 陷阱 → 回填结果不重渲染（插槽/tag 列丢宽）
    columns: () => tableColumns,
    tableData,
    showSelection: () => props.showSelection,
    // 异步列表：数据未到达的「假空」阶段不提前显示表格，避免「表头窄宽→内容宽」的列宽跳变（切换/挂载时的闪烁）
    loading,
    asyncLoad: () => !!props.pageApi,
    onEstimated: () => {
        // 需要计算完才能显示表格，否则会有表格闪烁的问题
        if (!showTable.value) {
            showTable.value = true;
        }
    },
});

// 静态/受控列表：把 props.data 同步进 tableData。immediate 让「无分页、无 api、仅传 :data」的普通列表
// 首挂载即渲染（否则 tableData 停在 usePageTable 初始的 []，要等 data 引用变化才出内容）；
// 分页页不传 data，immediate 仅把已是 [] 的 tableData 再置一次 []，零副作用。
watch(
    () => props.data,
    (newValue: Record<string, unknown>[] | undefined) => {
        tableData.value = newValue ?? [];
    },
    { immediate: true }
);

onMounted(async () => {
    if (props.searchItems.length > 0) {
        nowSearchItem.value = props.searchItems[0];
    }

    let pageSize = queryForm.value.pageSize as number;
    // 如果pageSize设为0，则使用系统全局配置的pageSize
    if (!pageSize) {
        pageSize = themeConfig.value.defaultListPageSize;
        // 可能storage已经存在配置json，则可能没值，需要清storage重试
        if (!pageSize) {
            pageSize = 10;
        }
    }

    queryForm.value.pageNum = 1;
    queryForm.value.pageSize = pageSize;
    state.pageSizes = [pageSize, pageSize * 2, pageSize * 3, pageSize * 4, pageSize * 5];

    if (!props.lazy) {
        await getTableData();
    }
});

const searchFormItemKeyUpEnter = (event: KeyboardEvent) => {
    event.preventDefault();
    search();
};

const formatText = (data: unknown) => {
    state.formatVal = '';
    try {
        state.formatVal = JSON.stringify(JSON.parse(data as string), null, 4);
    } catch (e) {
        state.formatVal = String(data);
    }
};

const handleSelectionChange = (val: Record<string, unknown>[]) => {
    selectionData.value = val;
};

const getData = () => {
    return toValue(tableData);
};

defineExpose({
    tableRef: tableRef,
    search: getTableData,
    getData,
    total,
});
</script>
<style scoped lang="scss">
// 展开收起：grid-template-rows 0fr↔1fr 是布局高度的平滑过渡（v-if 切换的瞬时塌陷是跳变的根源）。
// visibility 收起态延迟到动画结束再隐藏，避免内容在收拢过程中提前消失；
// collapsed 下表单常驻渲染（仅视觉隐藏），展开不再重建组件
.search-form-collapse {
    display: grid;
    grid-template-rows: 0fr;
    opacity: 0;
    visibility: hidden;
    transform: translateY(-6px);
    pointer-events: none;
    transition:
        grid-template-rows 0.3s ease-out,
        opacity 0.25s ease-out,
        transform 0.3s ease-out,
        visibility 0s linear 0.3s;

    // 内层是 grid item：必须零 padding/margin，0fr 轨道才能真正归零（轨道最小尺寸含 item 自身的 padding/margin）
    .search-form-collapse-inner {
        min-height: 0;
        overflow: hidden;
    }

    &.is-open {
        grid-template-rows: 1fr;
        opacity: 1;
        visibility: visible;
        transform: none;
        pointer-events: auto;
        transition:
            grid-template-rows 0.3s ease-out,
            opacity 0.25s ease-out,
            transform 0.3s ease-out,
            visibility 0s;
    }
}

// 工具条与表格之间留一条呼吸带：控件（32px）紧贴表头会让「搜索项」与「列名」糊成一片，
// 下边距把操作区与数据区分开；窄面板下工具条可 flex-wrap 换行，换行行距由 gap-y 兜底（可读性优先于单行）。
.data-table__toolbar {
    margin-bottom: 12px;
}

// 表头给一档浅灰底，让列名读作数据区的「标题带」（经典后台表格观感），不再与上方白底工具条糊在一起。
// --el-fill-color-light 三套主题各自自适应：浅为 #f5f5f5 轻灰、深为暗色填充、玻璃态为半透明白透出壁纸。
:deep(th.el-table__cell) {
    background-color: var(--el-fill-color-light);
}

// 简易搜索条：字段名即下拉触发器（filter 惯例），ghost 形态与输入框同高对齐
.simple-search-field {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    height: 32px;
    padding: 0 8px;
    border-radius: 6px;
    font-weight: 500;
    color: var(--el-text-color-regular);
    cursor: pointer;
    user-select: none;
    transition: background-color 0.15s ease-out, color 0.15s ease-out;

    &:hover {
        background: var(--el-fill-color-light);
        color: var(--el-color-primary);
    }

    // 单字段无下拉：纯文字占位，不可点
    &.is-static {
        padding: 0;
        cursor: default;

        &:hover {
            background: none;
            color: var(--el-text-color-regular);
        }
    }
}

.simple-search-field-arrow {
    color: var(--el-text-color-secondary);
    transition: transform 0.2s ease-out;

    &.is-open {
        transform: rotate(180deg);
    }
}

// 简易条复用 el-form-item 承载控件，须去掉表单的下边距，避免撑高工具行
.simple-search-bar .el-form-item {
    margin-bottom: 0;
}

// 展开按钮箭头：旋转过渡替代图标硬切换
.search-toggle-icon {
    transition: transform 0.3s ease-out;

    &.is-open {
        transform: rotate(180deg);
    }
}

// 简易搜索条出现/隐藏的轻淡入
.pt-fade-enter-active,
.pt-fade-leave-active {
    transition: opacity 0.2s ease-out;
}
.pt-fade-enter-from,
.pt-fade-leave-to {
    opacity: 0;
}

// 字段切换下拉（teleport 到 body）：scoped 下 :global 需平铺整条选择器，嵌套子规则不会被导出。
// 玻璃底/描边/阴影由全局 elevated 档提供，这里只收紧行距并标注当前字段
:global(.data-table-field-menu) {
    min-width: 150px;
}
:global(.data-table-field-menu .el-dropdown-menu) {
    padding: 4px;
}
:global(.data-table-field-menu .el-dropdown-menu__item) {
    padding: 6px 10px;
    border-radius: 6px;
}
:global(.data-table-field-menu .el-dropdown-menu__item.is-current) {
    color: var(--el-color-primary);
    font-weight: 600;
}
.field-menu-check {
    display: inline-flex;
    width: 14px;
    margin-right: 6px;
    vertical-align: -2px;
}

// 列配置 popover（teleport 到 body）：scoped 下 :global 需平铺整条选择器，嵌套子规则不会被导出
:global(.data-table-column-config) {
    max-width: 450px;
    max-height: 550px;
    overflow: auto;
}
:global(.data-table-column-config .column-config-item) {
    padding: 2px 0;
}
:global(.data-table-column-config .column-config-header) {
    margin-bottom: 4px;
    padding-bottom: 6px;
    border-bottom: 1px solid var(--el-border-color-lighter);
}
</style>
