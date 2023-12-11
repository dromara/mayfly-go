<template>
    <div class="page-table">
        <!-- 
            实现：通过我们配置好的 查询条件
                首先去创建form表单，根据我们配置的查询条件去做一个循环判断，展示出不用类型所对应不同的输入框
                比如：text对应普通的输入框，select对应下拉选择，dateTime对应日期时间选择器
                在使用时，父组件会传来一个queryForm空的对象，
                循环出来的输入框会绑定表格配置中的prop字段绑定在queryForm对象中
         -->
        <el-card>
            <div class="query" ref="queryRef">
                <div>
                    <div v-if="props.query.length > 0">
                        <el-form :model="queryForm_" label-width="auto" :size="props.size">
                            <el-row
                                v-for="i in Math.ceil((props.query.length + 1) / (defaultQueryCount + 1))"
                                :key="i"
                                v-show="i == 1 || isOpenMoreQuery"
                                :class="i > 1 && isOpenMoreQuery ? 'is-open' : ''"
                            >
                                <el-form-item
                                    :label="item.label"
                                    style="margin-right: 12px; margin-bottom: 0px"
                                    v-for="item in getRowQueryItem(i)"
                                    :key="item.prop"
                                >
                                    <!-- 这里只获取指定个数的筛选条件 -->
                                    <el-input
                                        v-model="queryForm_[item.prop]"
                                        :placeholder="'输入' + item.label + '关键字'"
                                        clearable
                                        v-if="item.type == 'text'"
                                    ></el-input>

                                    <el-select-v2
                                        v-model="queryForm_[item.prop]"
                                        :options="item.options"
                                        clearable
                                        :placeholder="'选择' + item.label + '关键字'"
                                        v-else-if="item.type == 'select'"
                                    />

                                    <el-date-picker
                                        v-model="queryForm_[item.prop]"
                                        clearable
                                        type="datetimerange"
                                        format="YYYY-MM-DD hh:mm:ss"
                                        value-format="x"
                                        range-separator="至"
                                        start-placeholder="开始时间"
                                        end-placeholder="结束时间"
                                        v-else-if="item.type == 'date'"
                                    />

                                    <template v-else-if="item.slot == 'queryBtns'">
                                        <template v-if="props.query?.length > defaultQueryCount">
                                            <el-button
                                                @click="isOpenMoreQuery = !isOpenMoreQuery"
                                                v-if="!isOpenMoreQuery"
                                                icon="ArrowDownBold"
                                                circle
                                            ></el-button>
                                            <el-button @click="isOpenMoreQuery = !isOpenMoreQuery" v-else icon="ArrowUpBold" circle></el-button>
                                        </template>

                                        <el-button @click="queryData()" type="primary" icon="search" plain>查询</el-button>
                                        <el-button @click="reset()" icon="RefreshRight">重置</el-button>
                                    </template>

                                    <slot :name="item.slot"></slot>
                                </el-form-item>
                            </el-row>
                        </el-form>
                    </div>
                </div>

                <div class="slot">
                    <!-- 查询栏右侧slot插槽（用来添加表格其他操作，比如，新增数据，删除数据等其他操作） -->
                    <slot name="queryRight"></slot>

                    <!-- 
                    动态表头显示，根据表格每条配置项中的show字段来决定改列是否显示或者隐藏 
                    columns 就是我们表格配置的数组对象
                    -->
                    <el-popover
                        placement="bottom"
                        title="表格配置"
                        popper-style="max-height: 550px; overflow: auto; max-width: 450px"
                        width="auto"
                        trigger="click"
                    >
                        <div v-for="(item, index) in props.columns" :key="index">
                            <el-checkbox v-model="item.show" :label="item.label" :true-label="true" :false-label="false" />
                        </div>
                        <template #reference>
                            <!-- 一个Element Plus中的图标 -->
                            <el-button icon="Operation" :size="props.size"></el-button>
                        </template>
                    </el-popover>
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
            >
                <el-table-column v-if="props.showSelection" type="selection" width="40" />

                <template v-for="(item, index) in columns">
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

            <el-row style="margin-top: 20px" type="flex" justify="end">
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
import { toRefs, watch, reactive, onMounted, Ref } from 'vue';
import { TableColumn, TableQuery } from './index';
import EnumTag from '@/components/enumtag/EnumTag.vue';
import { useThemeConfig } from '@/store/themeConfig';
import { storeToRefs } from 'pinia';
import { useVModel } from '@vueuse/core';
import Api from '@/common/Api';

const emit = defineEmits(['update:queryForm', 'update:pageNum', 'update:pageSize', 'update:selectionData', 'pageChange']);

const props = defineProps({
    size: {
        type: String,
        default: '',
    },
    inputWidth: {
        type: [Number, String],
        default: '200px',
    },
    // 是否显示选择列
    showSelection: {
        type: Boolean,
        default: false,
    },
    // 当前选择的数据
    selectionData: {
        type: Array<any>,
    },
    // 列信息
    columns: {
        type: Array<TableColumn>,
        default: function () {
            return [];
        },
        required: true,
    },
    // 调用分页数据的api
    pageApi: {
        type: Api,
        required: true,
    },
    // 懒加载，即需要手动调用search方法才可调接口获取数据，不会在mounted的时候调用。
    lazy: {
        type: Boolean,
        default: false,
    },
    // 执行查询时对查询参数进行处理，调整等
    beforeQueryFn: {
        type: Function,
    },
    // 数据处理回调函数，用于将请求回来的数据二次加工处理等
    dataHandlerFn: {
        type: Function,
    },
    // 查询条件配置
    query: {
        type: Array<TableQuery>,
        default: function () {
            return [];
        },
    },
    // 绑定的查询表单
    queryForm: {
        type: Object,
        default: function () {
            return {
                pageNum: 1,
                pageSize: 10,
            };
        },
    },
});

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
    tableMaxHeight: window.innerHeight - 240 + 'px',
});

const { pageSizes, isOpenMoreQuery, defaultQueryCount, inputWidth_, formatVal, tableMaxHeight } = toRefs(state);

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

onMounted(async () => {
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

    // 如果没传输入框宽度，则根据组件size设置默认宽度
    if (!props.inputWidth) {
        state.inputWidth_ = props.size == 'small' ? '150px' : '200px';
    } else {
        state.inputWidth_ = props.inputWidth;
    }

    window.addEventListener('resize', () => {
        calcuTableHeight();
    });

    if (!props.lazy) {
        await reqPageApi();
    }
});

const calcuTableHeight = () => {
    state.tableMaxHeight = window.innerHeight - 240 + 'px';
};

const formatText = (data: any) => {
    state.formatVal = '';
    try {
        state.formatVal = JSON.stringify(JSON.parse(data), null, 4);
    } catch (e) {
        state.formatVal = data;
    }
};

const getRowQueryItem = (row: number) => {
    // 第一行需要加个查询等按钮列
    if (row === 1) {
        const res = props.query.slice(row - 1, defaultQueryCount.value);
        // 查询等按钮列
        res.push(TableQuery.slot('', '', 'queryBtns'));
        return res;
    }
    const columnCount = defaultQueryCount.value + 1;
    return props.query.slice((row - 1) * columnCount - 1, row * columnCount - 1);
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
    for (let qi of props.query) {
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
.page-table {
    .query {
        margin-bottom: 10px;
        overflow: hidden;

        .is-open {
            // padding: 10px 0;
            max-height: 200px;
            margin-top: 10px;
        }

        display: flex;
        align-items: flex-start;
        justify-content: space-between;

        .slot {
            display: flex;
            justify-content: flex-end;
        }
    }

    .page {
        margin-top: 10px;
    }
}

::v-deep(.el-form-item__label) {
    font-weight: bold;
}

.el-select-v2 {
    width: v-bind(inputWidth_);
}

.el-input {
    width: v-bind(inputWidth_);
}

.el-select {
    width: v-bind(inputWidth_);
}

.el-date-editor {
    width: 380px !important;
}
</style>
