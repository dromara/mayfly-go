<template>
    <div>
        <transition name="el-zoom-in-top">
            <!-- 查询表单 -->
            <SearchForm v-show="isShowSearch" :items="searchItems" v-model="queryForm_" :search="queryData" :reset="reset" :search-col="searchCol">
                <!-- 遍历父组件传入的 solts 透传给子组件 -->
                <template v-for="(_, key) in useSlots()" v-slot:[key]>
                    <slot :name="key"></slot>
                </template>
            </SearchForm>
        </transition>

        <el-card>
            <div class="table-main">
                <!-- 表格头部 操作按钮 -->
                <div class="table-header">
                    <div class="header-button-lf">
                        <slot name="tableHeader" />
                    </div>

                    <div v-if="toolButton" class="header-button-ri">
                        <slot name="toolButton">
                            <el-button v-if="showToolButton('refresh')" icon="Refresh" circle @click="execQuery()" />

                            <el-button v-if="showToolButton('search') && searchItems?.length" icon="Search" circle @click="isShowSearch = !isShowSearch" />

                            <el-popover
                                placement="bottom"
                                title="表格配置"
                                popper-style="max-height: 550px; overflow: auto; max-width: 450px"
                                width="auto"
                                trigger="click"
                            >
                                <div v-for="(item, index) in tableColumns" :key="index">
                                    <el-checkbox v-model="item.show" :label="item.label" :true-label="true" :false-label="false" />
                                </div>
                                <template #reference>
                                    <el-button icon="Operation" circle :size="props.size"></el-button>
                                </template>
                            </el-popover>
                        </slot>
                    </div>
                </div>

                <el-table
                    v-bind="$attrs"
                    :max-height="tableMaxHeight"
                    @selection-change="handleSelectionChange"
                    :data="state.data"
                    highlight-current-row
                    v-loading="state.loading"
                    :size="props.size"
                    :border="border"
                >
                    <el-table-column v-if="props.showSelection" type="selection" width="40" />

                    <template v-for="(item, index) in tableColumns">
                        <el-table-column
                            :key="index"
                            v-if="item.show"
                            :prop="item.prop"
                            :label="item.label"
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
                                <slot :name="item.prop" :data="scope.row"></slot>
                            </template>

                            <!-- 枚举类型使用tab展示 -->
                            <template #default="scope" v-else-if="item.type == 'tag'">
                                <enum-tag :size="props.size" :enums="item.typeParam" :value="scope.row[item.prop]"></enum-tag>
                            </template>

                            <template #default="scope" v-else>
                                <!-- 配置了美化文本按钮以及文本内容大于指定长度，则显示美化按钮 -->
                                <el-popover
                                    v-if="item.isBeautify && scope.row[item.prop]?.length > 35"
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
                                            @click="formatText(scope.row[item.prop])"
                                            :underline="false"
                                            type="success"
                                            icon="MagicStick"
                                            class="mr5"
                                        ></el-link>
                                    </template>
                                </el-popover>

                                <span>{{ item.getValueByData(scope.row) }}</span>
                            </template>
                        </el-table-column>
                    </template>
                </el-table>
            </div>

            <el-row class="mt20" type="flex" justify="end">
                <el-pagination
                    :small="props.size == 'small'"
                    @current-change="handlePageChange"
                    @size-change="handleSizeChange"
                    style="text-align: right"
                    layout="prev, pager, next, total, sizes, jumper"
                    :total="state.total"
                    v-model:current-page="queryForm_.pageNum"
                    v-model:page-size="queryForm_.pageSize"
                    :page-sizes="pageSizes"
                />
            </el-row>
        </el-card>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, watch, reactive, onMounted, Ref, ref, useSlots } from 'vue';
import { TableColumn } from './index';
import EnumTag from '@/components/enumtag/EnumTag.vue';
import { useThemeConfig } from '@/store/themeConfig';
import { storeToRefs } from 'pinia';
import { useVModel, useEventListener } from '@vueuse/core';
import Api from '@/common/Api';
import SearchForm from '@/components/SearchForm/index.vue';
import { SearchItem } from '../SearchForm/index';

const emit = defineEmits(['update:queryForm', 'update:selectionData', 'pageChange']);

export interface PageTableProps {
    size?: string;
    showSelection?: boolean;
    showSearch?: boolean; // 是否显示搜索表单
    columns: TableColumn[]; // 列配置项  ==> 必传
    data?: any[]; // 静态 table data 数据，若存在则不会使用 requestApi 返回的 data ==> 非必传
    pageApi: Api; // 请求表格数据的 api
    lazy?: boolean; // 是否自动执行请求 api ==> 非必传（默认为false）
    beforeQueryFn?: (params: any) => any; // 执行查询时对查询参数进行处理，调整等
    dataHandlerFn?: (data: any) => any; // 数据处理回调函数，用于将请求回来的数据二次加工处理等
    searchItems?: SearchItem[];
    queryForm?: any; // 查询表单参数 ==> 非必传（默认为{pageNum:1, pageSize: 10}）
    border?: boolean; // 是否带有纵向边框 ==> 非必传（默认为false）
    toolButton?: ('refresh' | 'setting' | 'search')[] | boolean; // 是否显示表格功能按钮 ==> 非必传（默认为true）
    searchCol?: any; // 表格搜索项 每列占比配置 ==> 非必传 { xs: 1, sm: 2, md: 2, lg: 3, xl: 4 }
}

// 接受父组件参数，配置默认值
const props = withDefaults(defineProps<PageTableProps>(), {
    columns: () => [],
    showSelection: false,
    lazy: false,
    initParam: {},
    queryForm: {
        pageNum: 1,
        pageSize: 0,
    },
    border: false,
    toolButton: true,
    showSearch: false,
    searchItems: () => [],
    searchCol: () => ({ xs: 1, sm: 3, md: 3, lg: 4, xl: 4 }),
});

// 接收 columns 并设置为响应式
const tableColumns = reactive<TableColumn[]>(props.columns);

const { themeConfig } = storeToRefs(useThemeConfig());

const state = reactive({
    pageSizes: [] as any, // 可选每页显示的数据量
    isOpenMoreQuery: false,
    defaultQueryCount: 2, // 默认显示的查询参数个数，展开后每行显示查询条件个数为该值加1。第一行用最后一列来占用按钮
    loading: false,
    data: [],
    total: 0,
    // 输入框宽度
    inputWidth_: '200px' as any,
    formatVal: '', // 格式化后的值
    tableMaxHeight: '500px',
});

// 是否显示搜索模块
const isShowSearch = ref(props.showSearch);

// 控制 ToolButton 显示
const showToolButton = (key: 'refresh' | 'setting' | 'search') => {
    return Array.isArray(props.toolButton) ? props.toolButton.includes(key) : props.toolButton;
};

const { pageSizes, formatVal, tableMaxHeight } = toRefs(state);

const queryForm_: Ref<any> = useVModel(props, 'queryForm', emit);

watch(
    () => state.data,
    (newValue: any) => {
        if (newValue && newValue.length > 0) {
            props.columns.forEach((item) => {
                if (item.autoWidth && item.show) {
                    item.autoCalculateMinWidth(state.data);
                }
            });
        }
    }
);

watch(
    () => isShowSearch.value,
    () => {
        console.log('watch show sa');
        calcuTableHeight();
    }
);

onMounted(async () => {
    calcuTableHeight();
    useEventListener(window, 'resize', calcuTableHeight);

    let pageSize = queryForm_.value.pageSize;
    // 如果pageSize设为0，则使用系统全局配置的pageSize
    if (!pageSize) {
        pageSize = themeConfig.value.defaultListPageSize;
        // 可能storage已经存在配置json，则可能没值，需要清storage重试
        if (!pageSize) {
            pageSize = 10;
        }
    }

    queryForm_.value.pageNum = 1;
    queryForm_.value.pageSize = pageSize;
    state.pageSizes = [pageSize, pageSize * 2, pageSize * 3, pageSize * 4, pageSize * 5];

    if (!props.lazy) {
        await reqPageApi();
    }
});

const calcuTableHeight = () => {
    const headerHeight = isShowSearch.value ? 320 : 240;
    state.tableMaxHeight = window.innerHeight - headerHeight + 'px';
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

const reqPageApi = async () => {
    try {
        state.loading = true;

        let qf = queryForm_.value;
        if (props.beforeQueryFn) {
            qf = await props.beforeQueryFn(qf);
        }

        const res = await props.pageApi?.request(qf);
        if (!res) {
            return;
        }

        state.total = res.total;
        if (props.dataHandlerFn) {
            state.data = await props.dataHandlerFn(res.list);
        } else {
            state.data = res.list;
        }
    } finally {
        state.loading = false;
    }
};

const handlePageChange = (val: number) => {
    queryForm_.value.pageNum = val;
    execQuery();
};

const handleSizeChange = () => {
    changePageNum(1);
    execQuery();
};

const queryData = () => {
    changePageNum(1);
    execQuery();
};

const reset = () => {
    // 将查询参数绑定的值置空，并重新粗发查询接口
    for (let qi of props.searchItems) {
        queryForm_.value[qi.prop] = null;
    }

    changePageNum(1);
    execQuery();
};

const changePageNum = (pageNum: number) => {
    queryForm_.value.pageNum = pageNum;
};

const execQuery = async () => {
    await reqPageApi();
};

defineExpose({
    search: execQuery,
});
</script>
<style scoped lang="scss">
.table-box,
.table-main {
    display: flex;
    flex: 1;
    flex-direction: column;
    width: 100%;
    height: 100%;

    // 表格 header 样式
    .table-header {
        width: 100%;
        .header-button-lf {
            float: left;
        }

        .header-button-ri {
            float: right;
        }

        .el-button {
            margin-bottom: 10px;
        }
    }

    // el-table 表格样式
    .el-table {
        flex: 1;

        // 修复 safari 浏览器表格错位 https://github.com/HalseySpicy/Geeker-Admin/issues/83
        table {
            width: 100%;
        }

        // .el-table__header th {
        //     height: 45px;
        //     font-size: 15px;
        //     font-weight: bold;
        //     color: var(--el-text-color-primary);
        //     background: var(--el-fill-color-light);
        // }

        // .el-table__row {
        //     height: 45px;
        //     font-size: 14px;

        //     .move {
        //         cursor: move;

        //         .el-icon {
        //             cursor: move;
        //         }
        //     }
        // }

        // 设置 el-table 中 header 文字不换行，并省略
        .el-table__header .el-table__cell > .cell {
            // white-space: nowrap;
            white-space: wrap;
        }

        // 解决表格数据为空时样式不居中问题(仅在element-plus中)
        // .el-table__empty-block {
        //     position: absolute;
        //     top: 50%;
        //     left: 50%;
        //     transform: translate(-50%, -50%);

        //     .table-empty {
        //         line-height: 30px;
        //     }
        // }

        // table 中 image 图片样式
        .table-image {
            width: 50px;
            height: 50px;
            border-radius: 50%;
        }
    }
}

::v-deep(.el-form-item__label) {
    font-weight: bold;
}
</style>
