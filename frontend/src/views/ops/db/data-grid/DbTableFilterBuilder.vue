<template>
    <div class="db-filter-builder">
        <!-- 组合逻辑 + 添加条件 -->
        <div class="flex items-center justify-between mb-2">
            <el-radio-group v-model="draft.logic" size="small">
                <el-radio-button value="AND">{{ $t('db.logicAnd') }}</el-radio-button>
                <el-radio-button value="OR">{{ $t('db.logicOr') }}</el-radio-button>
            </el-radio-group>
            <el-button link type="primary" icon="Plus" size="small" @click="addCondition">{{ $t('db.addCondition') }}</el-button>
        </div>

        <div v-if="!draft.conditions.length" class="py-4 text-center text-[12px] text-gray-400">{{ $t('db.noFilter') }}</div>

        <!-- 条件行：[列] [操作符] [类型感知值编辑器] [删除]。
             宽度一律由外层 div 承载：EP 的 .el-select/.el-input/.el-date-editor 根元素自带 width 声明，
             工具类直接写组件上会被覆盖（项目既有惯例）；值编辑器的形态切换内聚在 DbTableFilterValueEditor -->
        <div v-for="cond in draft.conditions" :key="cond.id" class="flex items-center gap-2 mb-2">
            <div class="w-[150px] shrink-0">
                <el-select
                    v-model="cond.columnName"
                    filterable
                    size="small"
                    :placeholder="$t('db.filterSelectColumn')"
                    :teleported="false"
                    @change="onColumnChange(cond)"
                >
                    <el-option v-for="col in columns" :key="col.columnName" :label="col.columnName" :value="col.columnName">
                        <span>{{ col.columnName }}</span>
                        <span v-if="col.columnComment" class="float-right ml-3 text-[12px] text-gray-400">{{ col.columnComment }}</span>
                    </el-option>
                </el-select>
            </div>

            <div class="w-[110px] shrink-0">
                <el-select v-model="cond.operator" size="small" :teleported="false">
                    <el-option v-for="op in operatorsOf(cond)" :key="op.value" :label="op.symbol ?? $t(op.labelKey)" :value="op.value" />
                </el-select>
            </div>

            <!-- 单值 -->
            <div v-if="valueKindOf(cond) === 'single'" class="flex-1 min-w-0">
                <DbTableFilterValueEditor :cond="cond" :column="columnOf(cond)" :dialect="dialect" :placeholder="valuePlaceholderOf(cond)" @enter="apply" />
            </div>

            <!-- 区间 BETWEEN -->
            <template v-else-if="valueKindOf(cond) === 'range'">
                <div class="flex-1 min-w-0">
                    <DbTableFilterValueEditor
                        :cond="cond"
                        :column="columnOf(cond)"
                        :dialect="dialect"
                        endpoint="value"
                        :placeholder="valuePlaceholderOf(cond)"
                        @enter="apply"
                    />
                </div>
                <span class="shrink-0 text-[12px] text-gray-400">{{ $t('db.filterTo') }}</span>
                <div class="flex-1 min-w-0">
                    <DbTableFilterValueEditor
                        :cond="cond"
                        :column="columnOf(cond)"
                        :dialect="dialect"
                        endpoint="value2"
                        :placeholder="valuePlaceholderOf(cond)"
                        @enter="apply"
                    />
                </div>
            </template>

            <!-- 多值 IN：可手输创建 tag -->
            <div v-else-if="valueKindOf(cond) === 'multi'" class="flex-1 min-w-0">
                <el-select
                    v-model="cond.inValues"
                    multiple
                    filterable
                    allow-create
                    default-first-option
                    size="small"
                    :teleported="false"
                    :placeholder="$t('db.filterValuePlaceholder')"
                />
            </div>

            <el-button
                link
                icon="Close"
                size="small"
                class="shrink-0"
                :title="$t('common.delete')"
                :aria-label="$t('common.delete')"
                @click="removeCondition(cond)"
            />
        </div>

        <div class="flex justify-end mt-1">
            <el-button type="primary" size="small" @click="apply">{{ $t('db.applyFilter') }}</el-button>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { reactive } from 'vue';
import { useI18n } from 'vue-i18n';

import type { DbDialect } from '@/views/ops/db/dialect';
import type { TableColumnDef } from '@/views/ops/db/types';
import DbTableFilterValueEditor from './DbTableFilterValueEditor.vue';
import {
    cloneFilterGroup,
    columnKind,
    createFilterCondition,
    getOperatorDef,
    operatorsForKind,
    type FilterOperatorDef,
    type TableFilterCondition,
    type TableFilterGroup,
} from './filterModel';

// 过滤条件构建器（popover 内容）：编辑草稿条件组，应用时把快照交给父组件提交查询。
// 父组件以 v-if 控制挂载 → 每次打开 popover 都从当前生效条件组重新初始化草稿。
const props = defineProps<{
    columns: TableColumnDef[];
    modelValue: TableFilterGroup;
    dialect: DbDialect;
}>();

const emit = defineEmits<{
    apply: [group: TableFilterGroup];
}>();

const { t } = useI18n();

const draft = reactive<TableFilterGroup>(cloneFilterGroup(props.modelValue));

const columnOf = (cond: TableFilterCondition) => props.columns.find((x) => x.columnName === cond.columnName);

const operatorsOf = (cond: TableFilterCondition): FilterOperatorDef[] => operatorsForKind(columnKind(columnOf(cond), props.dialect));

const valueKindOf = (cond: TableFilterCondition) => getOperatorDef(cond.operator)?.valueKind ?? 'single';

// 值输入框 placeholder 带上列类型，输入前即可感知类型约束
const valuePlaceholderOf = (cond: TableFilterCondition) => {
    const column = columnOf(cond);
    return column?.columnType || column?.columnComment || t('db.filterValuePlaceholder');
};

const addCondition = () => {
    draft.conditions.push(createFilterCondition());
};

const removeCondition = (cond: TableFilterCondition) => {
    draft.conditions = draft.conditions.filter((x) => x.id !== cond.id);
};

// 换列后清空旧值并回退不适配的操作符：旧值语义随列类型变化（如字符串值挂在日期列上会拼出非法 WHERE）
const onColumnChange = (cond: TableFilterCondition) => {
    const operators = operatorsOf(cond);
    if (!operators.some((x) => x.value === cond.operator)) {
        cond.operator = operators[0].value;
    }
    cond.value = '';
    cond.value2 = '';
    cond.inValues = [];
};

const apply = () => {
    emit('apply', cloneFilterGroup(draft));
};
</script>
