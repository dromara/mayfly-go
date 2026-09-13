<template>
    <div class="string-input-container w-full!" v-if="dataType == DataType.String || dataType == DataType.Number">
        <el-input
            :ref="
                (el: unknown) => {
                    nextTick(() => {
                        focus && (el as InstanceType<typeof ElInput>)?.$el?.querySelector?.('input')?.focus() || (el as HTMLInputElement)?.focus?.();
                    });
                }
            "
            :disabled="disabled"
            @blur="handleBlur"
            :class="`w-full! mb-1! ${showEditorIcon ? 'string-input-container-show-icon' : ''}`"
            size="small"
            v-model="itemValue"
            :placeholder="placeholder ?? $t('common.pleaseInput')"
        />
        <SvgIcon v-if="showEditorIcon" @mousedown="openEditor" class="string-input-container-icon" name="FullScreen" :size="10" />
    </div>

    <el-date-picker
        v-else-if="dataType == DataType.Date"
        :ref="
            (el: unknown) => {
                nextTick(() => {
                    focus && (el as HTMLInputElement)?.focus?.();
                });
            }
        "
        :disabled="disabled"
        @change="handleBlur"
        @blur="handleBlur"
        class="edit-time-picker mb-1!"
        popper-class="edit-time-picker-popper"
        size="small"
        v-model="itemValue"
        :clearable="false"
        type="date"
        value-format="YYYY-MM-DD"
        :placeholder="`date-${placeholder ?? $t('common.pleaseSelect')}`"
    />

    <el-date-picker
        v-else-if="dataType == DataType.DateTime"
        :ref="
            (el: unknown) => {
                nextTick(() => {
                    focus && (el as HTMLInputElement)?.focus?.();
                });
            }
        "
        :disabled="disabled"
        @change="handleBlur"
        @blur="handleBlur"
        class="edit-time-picker mb-1!"
        popper-class="edit-time-picker-popper"
        size="small"
        v-model="itemValue"
        :clearable="false"
        type="datetime"
        value-format="YYYY-MM-DD HH:mm:ss"
        :placeholder="`datetime-${placeholder ?? $t('common.pleaseSelect')}`"
    />

    <el-time-picker
        v-else-if="dataType == DataType.Time"
        :ref="
            (el: unknown) => {
                nextTick(() => {
                    focus && (el as HTMLInputElement)?.focus?.();
                });
            }
        "
        :disabled="disabled"
        @change="handleBlur"
        @blur="handleBlur"
        class="edit-time-picker mb-1!"
        popper-class="edit-time-picker-popper"
        size="small"
        v-model="itemValue"
        :clearable="false"
        value-format="HH:mm:ss"
        :placeholder="`time-${placeholder ?? $t('common.pleaseSelect')}`"
    />
</template>

<script lang="ts" setup>
/**
 * 列值录入控件：按方言数据类型分派到文本框 / 日期 / 日期时间 / 时间选择器，
 * 长文本（>50 字符）额外给出全屏编辑器入口。
 *
 * 定位是跨子模块复用的部件，故置于 widgets/：数据网格的单元格就地编辑（DbTableData）
 * 与数据表单弹窗（DbTableDataForm）都在用，与「表结构编辑」无关。
 * 它只负责值本身，不含 el-form-item——标签与校验交由父级表单容器。
 */
import MonacoEditorBox from '@/components/monaco/MonacoEditorBox';
import SvgIcon from '@/components/svg-icon/index.vue';
import { Msg } from '@/hooks/useI18n';
import { ElInput } from 'element-plus';
import { computed, nextTick, ref, Ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { DataType } from '../dialect/index';

const { t } = useI18n();

export interface ColumnValueInputProps {
    /** 数据类型，决定渲染哪种录入控件（由方言的 getDataType 归一后传入） */
    dataType: DataType;
    /** 挂载后是否立即聚焦（单元格就地编辑场景需要） */
    focus?: boolean;
    placeholder?: string;
    /** 列名，用于全屏编辑器的标题 */
    columnName?: string;
    disabled?: boolean;
}

const props = withDefaults(defineProps<ColumnValueInputProps>(), {
    focus: false,
    dataType: DataType.String,
    disabled: false,
});

const emit = defineEmits<{
    /** 列名输入框失焦，父级据此触发重命名占用检查 */
    blur: [];
}>();

const modelValue = defineModel<unknown>();

const itemValue: Ref<string | number> = ref((modelValue.value ?? '') as string | number);

const showEditorIcon = computed(() => {
    return typeof itemValue.value === 'string' && itemValue.value.length > 50;
});

const editorOpening = ref(false);

const openEditor = () => {
    editorOpening.value = true;
    // 编辑器语言，如：json、html、text
    let editorLang = getEditorLangByValue(itemValue.value);
    MonacoEditorBox({
        content: String(itemValue.value),
        title: `${t('db.editField')} [${props.columnName}]`,
        language: editorLang,
        confirmFn: (newVal: string) => {
            itemValue.value = newVal;
            closeEditorDialog();
        },
        closeFn: closeEditorDialog,
        useDrawer: true,
    });
};

const closeEditorDialog = () => {
    editorOpening.value = false;
    handleBlur();
};

const handleBlur = () => {
    if (editorOpening.value) {
        return;
    }
    if (props.dataType == DataType.Number && itemValue.value && !/^-?\d*\.?\d+$/.test(String(itemValue.value))) {
        Msg.error('db.valueTypeNoMatch');
        return;
    }
    modelValue.value = itemValue.value;
    emit('blur');
};

const getEditorLangByValue = (value: string | number) => {
    // 判断是否是json
    try {
        if (typeof JSON.parse(String(value)) === 'object') {
            return 'json';
        }
    } catch (e) {
        /* empty */
    }

    // 判断是否是html
    try {
        const doc = new DOMParser().parseFromString(String(value), 'text/html');
        if (Array.from(doc.body.childNodes).some((node) => node.nodeType === 1)) {
            return 'html';
        }
    } catch (e) {
        /* empty */
    }

    return 'text';
};
</script>

<style lang="scss">
/*
 * 刻意不加 scoped：.edit-time-picker-popper 是 el-date-picker/el-time-picker 的 popper-class，
 * 弹层挂在 body 下（组件 DOM 树之外），加 scoped 后选择器带上 data-v 属性会整体失效。
 */
.string-input-container {
    position: relative;

    .el-input__wrapper {
        padding: 1px 3px;
    }
}
.string-input-container-show-icon {
    .el-input__inner {
        padding-right: 10px;
    }
}
.string-input-container-icon {
    position: absolute;
    top: 5px; /* 调整图标的垂直位置 */
    right: 3px; /* 调整图标的水平位置 */
    color: var(--el-color-primary);
}
.string-input-container-icon:hover {
    color: var(--el-color-success);
}

.edit-time-picker {
    height: 26px;
    width: 100% !important;
    .el-input__prefix {
        display: none;
    }

    .el-input__wrapper {
        padding: 1px 3px;
    }
}

.edit-time-picker-popper {
    .el-date-picker {
        width: 250px !important;
        .el-date-picker__header {
            margin: 0 5px;
        }
        .el-picker-panel__content {
            width: unset;
            margin: 0 5px;
        }
        .el-date-picker__header-label {
            font-size: 13px;
        }
        .el-picker-panel__footer {
            padding: 0 5px;
            button {
                font-size: 11px;
                padding: 5px 6px;
                height: 20px;
            }
        }
    }
    .el-date-table {
        th {
            font-size: 10px;
            font-weight: 600;
            padding: 0;
        }
        td {
            padding: 0;
        }
    }
    .el-time-panel {
        width: 100px;

        .el-time-spinner__list {
            &::after,
            &::before {
                height: 10px;
            }
            .el-time-spinner__item {
                height: 20px;
                line-height: 20px;
            }
        }
    }
}
</style>
