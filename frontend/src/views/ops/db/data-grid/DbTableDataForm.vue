<template>
    <el-dialog v-model="visible" :title="title" :destroy-on-close="true" @opened="onOpened" @close="resetForm" width="600px" body-class="h-[65vh] overflow-auto">
        <!--
            validate-on-rule-change=false：required 规则在渲染时生成，若保持默认 true，
            el-form 会在「规则变化」这一刻立刻校验一次，表现为弹窗刚打开（用户还没碰过）
            必填项就全变红。校验应只在点「确定」或用户 blur/change 后发生。
        -->
        <el-form
            ref="dataForm"
            :model="modelValue"
            :validate-on-rule-change="false"
            scroll-to-error
            :show-message="false"
            label-width="auto"
            size="small"
        >
            <el-form-item
                v-for="column in columns"
                :key="column.columnName"
                :prop="column.columnName"
                :class="{ 'is-required': isColumnRequired(column) }"
            >
                <template #label>
                    <span class="cursor-pointer" :title="column?.columnComment ? `${column.columnType} | ${column.columnComment}` : column.columnType">
                        {{ column.columnName }}
                    </span>
                </template>

                <ColumnValueInput
                    v-model="modelValue[`${column.columnName}`]"
                    :data-type="dbInst.getDialect().getDataType(column.dataType ?? '')"
                    :placeholder="column?.columnComment ? `${column.columnType} | ${column.columnComment}` : column.columnType"
                    :column-name="column.columnName"
                    :binary="isBinaryColumnType(column.columnType)"
                    :disabled="column.autoIncrement || column.masked"
                />
            </el-form-item>
        </el-form>
        <template #footer v-if="props.tableName">
            <el-button @click="onCloseDialog">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
import { ref, watch, onMounted, nextTick, useTemplateRef } from 'vue';
import type { FormInstance } from 'element-plus';
import ColumnValueInput from '../widgets/ColumnValueInput.vue';
import { isBinaryColumnType } from '../widgets/binaryValue';
import { DbInst } from '../db';
import { firstMissingRequired, isRequiredColumn, resolveFormAction } from './formMode';
import { Msg } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';
import type { TableColumnDef } from '../types';

/**
 * 数据表单弹窗 props。
 *
 * 原名 ColumnFormItemProps，与本文件引用的列值录入控件的 props 接口重名（两者字段完全不同），
 * 同名异构会让「这到底是谁的 props」只能靠猜，故按组件名命名。
 */
export interface DbTableDataFormProps {
    /** 目标实例：取方言归一数据类型、生成并提交 SQL */
    dbInst: DbInst;
    dbName: string;
    tableName: string;
    /** 列定义，决定表单渲染哪些字段 */
    columns: TableColumnDef[];
    /** 表单用途：create=新增（生成 INSERT）；edit=查看/编辑既有行（生成差异 UPDATE） */
    mode?: 'create' | 'edit';
    title?: string; // dialog title
}

const props = withDefaults(defineProps<DbTableDataFormProps>(), {
    /** 表单用途：create=新增（必须生成 INSERT）；edit=查看/编辑既有行（生成差异 UPDATE） */
    mode: 'edit',
    title: '',
});

const modelValue = defineModel<Record<string, unknown>>('modelValue', { default: () => ({}) });

const visible = defineModel<boolean>('visible', {
    default: false,
});

const emit = defineEmits<{
    /** 表格数据提交成功，父级据此刷新列表 / 清理编辑态 */
    submitSuccess: [];
}>();

const dataForm = useTemplateRef<FormInstance>('dataForm');
const { t } = useI18n();

/**
 * 是否必填：只用于「显示星号」，不再作为 el-form-item 的 required 规则。
 *
 * 一旦把 required 交给 el-form-item，EP 会为字段生成校验规则，并在动态创建字段
 * （本表单是 destroy-on-close）时立刻跑一次校验 —— 用户还没看到界面就全红。
 * 因此校验时机改为完全由提交触发（见 validateRequired）。
 */
/** 星号显示与提交校验都基于同一判据（挂在 formMode 的纯函数上，便于单测） */
const isColumnRequired = (column: TableColumnDef): boolean => props.tableName !== '' && isRequiredColumn(column);

/** 提交前校验必填项（唯一校验入口） */
const validateRequired = (data: Record<string, unknown>): boolean => {
    const missing = firstMissingRequired(props.columns, data);
    if (missing) {
        Msg.warning(t('common.fieldNotEmpty', { field: missing }));
        return false;
    }
    return true;
};

let oldValue = null as Record<string, unknown> | null;

onMounted(() => {
    setOldValue();
});

watch(visible, async (newValue) => {
    if (!newValue) {
        return;
    }
    setOldValue();
    // 打开时清一次校验态：required 规则由 columns 动态生成，el-form 会在规则建立那一刻
    // 隐式跑一次校验，用户尚未输入就把必填项标红（"一打开就报错"）。
    // clearValidate 把状态复位为空，之后只由用户 blur/change 与提交时的 validate 驱动。
    await nextTick();
    dataForm.value?.clearValidate();
});

const setOldValue = () => {
    // create 模式一律按新增处理；edit 模式以「打开时的行数据」作为旧值基线。
    //
    // 必须显式置 null：原实现只在「有值」时赋值，上一次编辑留下的 oldValue 会残留到
    // 下一次新增，使新增被生成为 UPDATE（`WHERE id = NULL`，执行 0 行却看似成功）。
    if (props.mode === 'create') {
        oldValue = null;
        return;
    }
    oldValue = Object.keys(modelValue.value).length > 0 ? Object.assign({}, modelValue.value) : null;
};

// 关闭即丢弃表单内容与旧值基线：下一次打开必须是一张干净的表。
// 挂在 el-dialog 的 @close 上，覆盖 × / Esc / 点遮罩 三种关闭方式——
// 只清「取消」按钮会让右上角 × 关掉后数据仍在，下次打开带着上次的残留值。
const resetForm = () => {
    modelValue.value = {};
    oldValue = null;
};

// 打开后把滚动容器归零：弹窗 body 是复用的同一个 DOM，任何原因（历史滚动位置、
// 控件自动聚焦）留下的偏移都会让表单一打开就停在底部。这里显式回到顶部。
const onOpened = () => {
    const body = (dataForm.value?.$el as HTMLElement | undefined)?.closest('.el-dialog__body') as HTMLElement | null;
    if (body) {
        body.scrollTop = 0;
    }
};

const onCloseDialog = () => {
    visible.value = false;
    resetForm();
};

const onConfirm = async () => {
    if (!validateRequired(modelValue.value)) {
        return;
    }

    const dbInst = props.dbInst;
    const data = modelValue.value;
    const db = props.dbName;
    const tableName = props.tableName;

    let sql = '';
    if (resolveFormAction(props.mode, oldValue) === 'update') {
        const old = oldValue as Record<string, unknown>;
        const updateColumnValue: Record<string, unknown> = {};
        Object.keys(old).forEach((key) => {
            // 如果新旧值不相等，则为需要更新的字段
            if (old[key] !== modelValue.value[key]) {
                updateColumnValue[key] = modelValue.value[key];
            }
        });
        sql = await dbInst.genUpdateSql(db, tableName, updateColumnValue, old);
    } else {
        sql = await dbInst.genInsertSql(db, tableName, [data], true);
    }

    if (!sql) {
        // 更新场景无主键：无法安全定位单行，禁止提交
        Msg.warning('db.needPkToOperate');
        return;
    }

    // 新增/编辑单行也以原子批次提交：失败即整批不生效，避免半条记录落库
    dbInst.promptExeSql(
        db,
        sql,
        undefined,
        () => {
            onCloseDialog();
            emit('submitSuccess');
        },
        true
    );
};
</script>

<style lang="scss"></style>
