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
                    <div v-if="props.query.length > 0" class="query-head">
                        <div style="display: flex; align-items: center;">
                            <el-form :model="props.queryForm" label-width="70px" style="display: flex;">
                                <el-form-item :label="item.label" style="margin-right: 20px; margin-bottom: 0px;"
                                    v-for="item in props.query?.slice(0, defaultQueryCount)" :key="item.prop">
                                    <!-- 这里只获取指定个数的筛选条件 -->
                                    <el-input v-model="queryForm[item.prop]" :placeholder="'输入' + item.label + '关键字'"
                                        clearable v-if="item.type == 'text'"></el-input>

                                    <el-select-v2 v-model="queryForm[item.prop]" :options="item.options" clearable
                                        :placeholder="'选择' + item.label + '关键字'" v-else-if="item.type == 'select'" />

                                    <el-date-picker v-model="queryForm[item.prop]" clearable type="datetimerange"
                                        format="YYYY-MM-DD hh:mm:ss" value-format="x" range-separator="至"
                                        start-placeholder="开始时间" end-placeholder="结束时间" v-else-if="item.type == 'date'" />

                                    <slot :name="item.slot"></slot>
                                </el-form-item>
                            </el-form>

                            <template v-if="props.query?.length > defaultQueryCount">
                                <el-button @click="isOpenMoreQuery = !isOpenMoreQuery" v-if="!isOpenMoreQuery"
                                    icon="ArrowDownBold" circle></el-button>
                                <el-button @click="isOpenMoreQuery = !isOpenMoreQuery" v-else icon="ArrowUpBold"
                                    circle></el-button>
                            </template>

                            <el-button @click="queryData()" type="primary" plain>查询</el-button>
                            <el-button @click="reset()">重置</el-button>
                        </div>
                    </div>
                    <!-- 这里做的是一个类似于折叠面板的功能 -->
                    <div class="query-content" :class="isOpenMoreQuery ? 'is-open' : ''">
                        <el-form :model="props.queryForm" label-width="70px" style="display: flex; flex-wrap: wrap;">
                            <el-form-item :label="item.label" style="margin-right: 20px; margin-bottom: 0px;"
                                v-for="item in props.query?.slice(defaultQueryCount)" :key="item.prop">

                                <!-- 这里获取除前两个以外所有的筛选条件 -->
                                <el-input v-model="queryForm[item.prop]" :placeholder="'输入' + item.label + '关键字'" clearable
                                    v-if="item.type == 'text'"></el-input>

                                <el-select-v2 v-model="queryForm[item.prop]" :options="item.options" clearable
                                    :placeholder="'选择' + item.label + '关键字'" v-else-if="item.type == 'select'" />

                                <el-date-picker v-model="queryForm[item.prop]" clearable type="datetimerange"
                                    format="YYYY-MM-DD hh:mm:ss" value-format="x" range-separator="至"
                                    start-placeholder="开始时间" end-placeholder="结束时间" v-else-if="item.type == 'date'" />

                                <slot :name="item.slot"></slot>
                            </el-form-item>
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
                    <el-popover placement="bottom" title="表格配置" :width="200" trigger="click">
                        <div v-for="(item, index) in props.columns" :key="index">
                            <el-checkbox v-model="item.show" :label="item.label" :true-label="true" :false-label="false" />
                        </div>
                        <template #reference>
                            <!-- 一个Element Plus中的图标 -->
                            <el-button icon="Operation"></el-button>
                        </template>
                    </el-popover>
                </div>
            </div>

            <el-table v-bind="$attrs" max-height="700" @current-change="choose" :data="props.data" border
                highlight-current-row show-overflow-tooltip>
                <el-table-column v-if="props.showChooseColumn" label="选择" align="center" width="53px">
                    <template #default="scope">
                        <el-radio v-model="state.chooseId" :label="scope.row.id">
                            <i></i>
                        </el-radio>
                    </template>
                </el-table-column>

                <template v-for="(item, index) in columns">
                    <el-table-column :key="index" v-if="item.show" :prop="item.prop" :label="item.label" :fixed="item.fixed"
                        :align="item.align" :show-overflow-tooltip="item.showOverflowTooltip || true"
                        :min-width="item.minWidth" :sortable="item.sortable || false" :type="item.type" :width="item.width">

                        <!-- 插槽：预留功能 -->
                        <template #default="scope" v-if="item.slot">
                            <slot :name="item.slot" :data="scope.row"></slot>
                        </template>

                        <template #default="scope" v-else>
                            <span>{{ item.formatFunc ? item.formatFunc(scope.row[item.prop]) : scope.row[item.prop]
                                }}</span>
                        </template>

                    </el-table-column>
                </template>
            </el-table>

            <el-row style="margin-top: 20px" type="flex" justify="end">
                <el-pagination @current-change="handlePageChange" @size-change="handleSizeChange" style="text-align: right"
                    layout="prev, pager, next, total, sizes, jumper" :total="props.total"
                    v-model:current-page="state.pageNum" v-model:page-size="state.pageSize"
                    :page-sizes="[10, 15, 20, 25]" />
            </el-row>
        </el-card>
    </div>
</template>

<script lang='ts' setup>
import { toRefs, watch, reactive, onMounted } from 'vue';
import { TableColumn, TableQuery } from './index';

const emit = defineEmits(['update:queryForm', 'update:pageNum', 'update:pageSize', 'update:chooseData', 'pageChange'])

const props = defineProps({
    // 是否显示选择列
    showChooseColumn: {
        type: Boolean,
        default: false,
    },
    // 选择列绑定的主键key字段名
    chooseDataIdKey: {
        type: String,
        default: "id"
    },
    // 当前选择的数据
    chooseData: {
        type: Object
    },
    // 列信息
    columns: {
        type: Array<TableColumn>,
        default: function () {
            return [];
        }
    },
    // 表格数据
    data: {
        type: Array,
    },
    total: {
        type: [Number],
        default: 0,
    },
    pageNum: {
        type: Number,
        default: 1,
    },
    pageSize: {
        type: [Number],
        default: 10,
    },
    // 绑定的查询表单
    queryForm: {
        type: Object,
        default: function () {
            return {};
        }
    },
    // 查询条件配置
    query: {
        type: Array<TableQuery>,
        default: function () {
            return [];
        }
    }
})

const state = reactive({
    pageSize: 10,
    pageNum: 1,
    chooseData: null as any,
    chooseId: 0 as any,
    isOpenMoreQuery: false,
    defaultQueryCount: 2, // 默认显示的查询参数个数
    queryForm: {} as any,
})

const {
    isOpenMoreQuery,
    defaultQueryCount,
    queryForm,
} = toRefs(state)

watch(() => props.queryForm, (newValue: any) => {
    state.queryForm = newValue
})

watch(() => props.chooseData, (newValue: any) => {
    state.chooseData = newValue
    if (newValue) {
        state.chooseId = state.chooseData[props.chooseDataIdKey]
    } else {
        state.chooseId = 0;
    }
})

watch(() => props.pageNum, (newValue: any) => {
    state.pageNum = newValue
})

watch(() => props.pageSize, (newValue: any) => {
    state.pageSize = newValue
})

watch(() => props.data, (newValue: any) => {
    if (newValue.length > 0) {
        props.columns.forEach(item => {
            if (item.autoWidth && item.show) {
                item.minWidth = TableColumn.flexColumnWidth(item.prop, item.label, props.data) as any
            }
        })
    }
})

onMounted(() => {
    state.pageNum = props.pageNum;
    state.pageSize = props.pageSize;
    state.queryForm = props.queryForm;
})

// 处理选中了列表中的某一条数据
const choose = (item: any) => {
    if (!item || !props.showChooseColumn) {
        return;
    }
    state.chooseData = item;
    state.chooseId = item[props.chooseDataIdKey]
    emit('update:chooseData', state.chooseData)
};

const handlePageChange = () => {
    emit('update:pageNum', state.pageNum)
    emit("pageChange")
}

const handleSizeChange = () => {
    emit('update:pageSize', state.pageSize)
    emit("pageChange")
}

const queryData = () => {
    // 触发重新调用查询接口即可
    emit("pageChange")
}

const reset = () => {
    // 触发重新调用查询接口即可
    state.queryForm = {};
    emit('update:queryForm', state.queryForm)
    emit("pageChange")
}

</script>
<style scoped lang="scss">
.page-table {
    .query {
        margin-bottom: 10px;
        overflow: hidden;

        .query-head {
            display: flex;
            align-items: center;
            justify-content: space-between;
        }

        .query-content {
            width: 100%;
            max-height: 0px;
            transition: all 0.8s;
        }

        .is-open {
            padding: 10px 0;
            max-height: 200px;
        }

        display: flex;
        align-items: flex-start;
        justify-content: space-between;
        margin-bottom: 10px;

        .query-content {
            display: flex;
            align-items: flex-start;

            .query-form {
                .el-form-item {
                    margin: 0px;
                    margin-right: 20px;
                }
            }
        }

        .slot {
            display: flex;
            justify-content: flex-end;
            padding-right: 20px;
        }
    }

    .page {
        margin-top: 10px;
    }
}

::v-deep(.el-form-item__label) {
    font-weight: bold;
}

.el-input {
    width: 200px;
}

.el-select-v2 {
    width: 200px;
}

::v-deep(.el-date-editor) {
    width: 380px !important;
}
</style>