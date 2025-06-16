<template>
    <div class="h-full flex flex-col flex-1 overflow-hidden">
        <transition name="el-zoom-in-top">
            <!-- 查询表单 -->
            <SearchForm v-if="isShowSearch" :items="tableSearchItems" v-model="queryForm" :search="search" :reset="reset" :search-col="searchCol">
                <!-- 遍历父组件传入的 solts 透传给子组件 -->
                <template v-for="(_, key) in useSlots()" v-slot:[key]>
                    <slot :name="key"></slot>
                </template>
            </SearchForm>
        </transition>

        <el-card class="h-full" body-class="h-full flex flex-col">
            <!-- 表格头部 操作按钮 -->
            <div class="flex justify-between">
                <div>
                    <slot name="tableHeader" />
                </div>

                <slot v-if="toolButton" name="toolButton">
                    <div class="flex">
                        <!-- 简易单个搜索项 -->
                        <div v-if="nowSearchItem" class="flex">
                            <el-dropdown v-if="searchItems?.length > 1">
                                <SvgIcon :size="16" name="CaretBottom" class="!mr-1 !mt-1.5 simple-search-form-btn" />
                                <template #dropdown>
                                    <el-dropdown-menu>
                                        <el-dropdown-item v-for="searchItem in searchItems" :key="searchItem.prop" @click="changeSimpleFormItem(searchItem)">
                                            {{ $t(searchItem.label) }}
                                        </el-dropdown-item>
                                    </el-dropdown-menu>
                                </template>
                            </el-dropdown>

                            <div class="text-right mr-1.5 mt-1">
                                <el-text truncated tag="b">{{ `${$t(nowSearchItem?.label)} : ` }}</el-text>
                            </div>

                            <el-form-item class="w-[200px]" :key="nowSearchItem.prop">
                                <SearchFormItem
                                    @keyup.enter.native="searchFormItemKeyUpEnter"
                                    v-if="!nowSearchItem.slot"
                                    :item="nowSearchItem"
                                    v-model="queryForm[nowSearchItem.prop]"
                                />

                                <slot @keyup.enter.native="searchFormItemKeyUpEnter" v-else :name="nowSearchItem.slot"> </slot>
                            </el-form-item>
                        </div>

                        <div class="ml-2">
                            <el-button v-if="showToolButton('search') && searchItems?.length" icon="Search" circle @click="search" />

                            <!-- <el-button v-if="showToolButton('refresh')" icon="Refresh" circle @click="execQuery()" /> -->

                            <el-button
                                v-if="showToolButton('search') && searchItems?.length > 1"
                                :icon="isShowSearch ? 'ArrowDown' : 'ArrowUp'"
                                circle
                                @click="isShowSearch = !isShowSearch"
                            />

                            <el-popover
                                placement="bottom"
                                title="表格配置"
                                popper-style="max-height: 550px; overflow: auto; max-width: 450px"
                                width="auto"
                                trigger="click"
                            >
                                <div v-for="(item, index) in tableColumns" :key="index">
                                    <el-checkbox v-model="item.show" :label="$t(item.label)" :true-value="1" :false-value="0" />
                                </div>
                                <template #reference>
                                    <el-button icon="Operation" circle :size="props.size"></el-button>
                                </template>
                            </el-popover>
                        </div>
                    </div>
                </slot>
            </div>

            <div class="flex-1 overflow-auto">
                <el-table
                    v-show="showTable"
                    ref="tableRef"
                    v-bind="$attrs"
                    height="100%"
                    @selection-change="handleSelectionChange"
                    :data="tableData"
                    highlight-current-row
                    v-loading="loading"
                    :size="props.size as any"
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
                            <!-- 插槽：预留功能 -->
                            <template #default="scope" v-if="item.slot">
                                <slot :name="item.slotName ? item.slotName : item.prop" :data="scope.row"></slot>
                            </template>

                            <!-- 枚举类型使用tab展示 -->
                            <template #default="scope" v-else-if="item.type == 'tag'">
                                <enum-tag :size="props.size" :enums="item.typeParam" :value="item.getValueByData(scope.row)"></enum-tag>
                            </template>

                            <template #default="scope" v-else>
                                <!-- 配置了美化文本按钮以及文本内容大于指定长度，则显示美化按钮 -->
                                <el-popover
                                    v-if="item.isBeautify && item.getValueByData(scope.row)?.length > 35"
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
                </el-table>
            </div>

            <el-row v-if="props.pageable" class="mt-4" type="flex" justify="end">
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
        </el-card>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, watch, reactive, onMounted, Ref, ref, useSlots, toValue, h } from 'vue';
import { TableColumn } from './index';
import EnumTag from '@/components/enumtag/EnumTag.vue';
import { useThemeConfig } from '@/store/themeConfig';
import { storeToRefs } from 'pinia';
import Api from '@/common/Api';
import SearchForm from '@/components/SearchForm/index.vue';
import { SearchItem } from '../SearchForm/index';
import SearchFormItem from '../SearchForm/components/SearchFormItem.vue';
import SvgIcon from '@/components/svgIcon/index.vue';
import { usePageTable } from '@/hooks/usePageTable';
import { ElInput, ElTable } from 'element-plus';

const emit = defineEmits(['update:selectionData', 'pageSizeChange', 'pageNumChange']);

export interface PageTableProps {
    size?: string;
    pageApi?: Api; // 请求表格数据的 api
    columns: TableColumn[] | any[]; // 列配置项  ==> 必传
    showSelection?: boolean;
    selectable?: (row: any) => boolean; // 是否可选
    pageable?: boolean;
    showSearch?: boolean; // 是否显示搜索表单
    data?: any[]; // 静态 table data 数据，若存在则不会使用 requestApi 返回的 data ==> 非必传
    lazy?: boolean; // 是否自动执行请求 api ==> 非必传（默认为false）
    beforeQueryFn?: (params: any) => any; // 执行查询时对查询参数进行处理，调整等
    dataHandlerFn?: (data: any) => any; // 数据处理回调函数，用于将请求回来的数据二次加工处理等
    searchItems?: SearchItem[];
    border?: boolean; // 是否带有纵向边框 ==> 非必传（默认为false）
    toolButton?: ('setting' | 'search')[] | boolean; // 是否显示表格功能按钮 ==> 非必传（默认为true）
    searchCol?: any; // 表格搜索项 每列占比配置 ==> 非必传 { xs: 1, sm: 2, md: 2, lg: 3, xl: 4 } | number 如 3
}

// 接受父组件参数，配置默认值
const props = withDefaults(defineProps<PageTableProps>(), {
    columns: () => [],
    pageable: true,
    showSelection: false,
    lazy: false,
    border: false,
    toolButton: true,
    showSearch: false,
    searchItems: () => [],
    searchCol: () => ({ xs: 1, sm: 3, md: 3, lg: 4, xl: 5 }),
});

// 查询表单参数 ==> 非必传（默认为{pageNum:1, pageSize: 10}）
const queryForm: Ref<any> = defineModel('queryForm', {
    default: {
        pageNum: 1,
        pageSize: 0,
    },
});

// table 实例
const tableRef = ref<InstanceType<typeof ElTable>>();

// 接收 columns 并设置为响应式
const tableColumns = reactive<TableColumn[]>(props.columns);

// 接收 searchItems 并设置为响应式
const tableSearchItems = reactive<SearchItem[]>(props.searchItems);

const { themeConfig } = storeToRefs(useThemeConfig());

// 是否显示搜索模块
const isShowSearch = ref(props.showSearch);

// 控制 ToolButton 显示
const showToolButton = (key: 'setting' | 'search') => {
    return Array.isArray(props.toolButton) ? props.toolButton.includes(key) : props.toolButton;
};

const nowSearchItem: Ref<SearchItem> = ref(null) as any;

// 是否已经计算列宽度
const isCalculatedWidth: Ref<boolean> = ref(false);
const showTable: Ref<boolean> = ref(false);

/**
 * 改变当前的搜索项
 * @param searchItem 当前点击的搜索项
 */
const changeSimpleFormItem = (searchItem: SearchItem) => {
    // 将之前的值置为空，避免因为只显示一个搜索项却搜索多个条件
    queryForm.value[nowSearchItem.value.prop] = null;
    nowSearchItem.value = searchItem;
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
    pageSizes: [] as any, // 可选每页显示的数据量
    // 输入框宽度
    formatVal: '', // 格式化后的值
});

const { pageSizes, formatVal } = toRefs(state);

watch(tableData, (newValue: any) => {
    calculateTableColumnMinWidth();
    // 需要计算完才能显示表格，否则会有表格闪烁的问题
    if (!showTable.value) {
        showTable.value = true;
    }
});

/**
 * 计算表格列宽
 */
const calculateTableColumnMinWidth = () => {
    if (isCalculatedWidth.value || !tableData.value || tableData.value.length === 0) {
        return;
    }

    // 计算表格列宽
    props.columns.forEach((item) => {
        if (item.autoWidth && item.show) {
            item.autoCalculateMinWidth(tableData.value);
        }
    });

    isCalculatedWidth.value = true;
};

watch(
    () => props.data,
    (newValue: any) => {
        tableData = newValue;
    }
);

onMounted(async () => {
    if (props.searchItems.length > 0) {
        nowSearchItem.value = props.searchItems[0];
    }

    let pageSize = queryForm.value.pageSize;
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

const searchFormItemKeyUpEnter = (event: any) => {
    event.preventDefault();
    search();
};

const formatText = (data: any) => {
    state.formatVal = '';
    try {
        state.formatVal = JSON.stringify(JSON.parse(data), null, 4);
    } catch (e) {
        state.formatVal = data;
    }
};

const handleSelectionChange = (val: any) => {
    emit('update:selectionData', val);
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
<style scoped lang="scss"></style>
