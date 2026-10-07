<template>
    <div class="policy-value-editor">
        <!-- 枚举：候选值来自后端字段字典，多选形态由操作符决定 -->
        <div v-if="kind === 'select'" class="value-cell">
            <el-select
                :model-value="selectValue"
                size="small"
                :multiple="isMulti"
                :collapse-tags="isMulti"
                :collapse-tags-tooltip="isMulti"
                :clearable="!isMulti"
                :disabled="disabled"
                :placeholder="$t('flow.policy.chooseValue')"
                @update:model-value="onSelectChange"
            >
                <el-option v-for="option in options" :key="option" :label="optionLabel(option)" :value="option" />
            </el-select>
        </div>

        <!-- 数值：区间形态渲染左右端点 -->
        <div v-else-if="kind === 'number'" class="value-row">
            <div class="value-cell">
                <el-input-number
                    :model-value="numberAt(0)"
                    size="small"
                    :controls="false"
                    :disabled="disabled"
                    :placeholder="$t('flow.policy.numberPlaceholder')"
                    @update:model-value="(value: number | undefined) => writeNumber(0, value)"
                />
            </div>
            <!-- 只提示不钳位：el-input-number 的 min/max 会在输入瞬间静默改值（填 99999999、落库变 1048576，
                 而框里还显示着原值），那比「保存时被拦下并告知越界」更难排查；边界统一交给校验与后端 -->
            <span v-if="rangeHint" class="editor-hint">{{ rangeHint }}</span>
            <template v-if="isRange">
                <span class="range-sep">~</span>
                <div class="value-cell">
                    <el-input-number
                        :model-value="numberAt(1)"
                        size="small"
                        :controls="false"
                        :disabled="disabled"
                        :placeholder="$t('flow.policy.numberPlaceholder')"
                        @update:model-value="(value: number | undefined) => writeNumber(1, value)"
                    />
                </div>
            </template>
        </div>

        <!-- 布尔：只允许真布尔，避免与数值互认导致条件语义漂移 -->
        <div v-else-if="kind === 'switch'" class="value-cell">
            <el-switch
                :model-value="Boolean(model)"
                size="small"
                :disabled="disabled"
                @update:model-value="writeBool"
            />
        </div>

        <!-- 列表：成员由使用者输入，量词语义下期望值本身也是集合 -->
        <div v-else-if="kind === 'tags'" class="value-cell">
            <el-select
                :model-value="stringList"
                size="small"
                multiple
                filterable
                allow-create
                default-first-option
                :reserve-keyword="false"
                :disabled="disabled"
                :placeholder="$t('flow.policy.tagsPlaceholder')"
                @update:model-value="writeTags"
            />
        </div>

        <!-- 文本与未注册类型（后端可先上线，前端降级为文本输入不崩） -->
        <div v-else class="value-row">
            <div class="value-cell">
                <el-input
                    :model-value="textAt(0)"
                    size="small"
                    :disabled="disabled"
                    :placeholder="$t('flow.policy.textPlaceholder')"
                    @update:model-value="(value: string) => writeText(0, value)"
                />
            </div>
            <template v-if="isRange">
                <span class="range-sep">~</span>
                <div class="value-cell">
                    <el-input
                        :model-value="textAt(1)"
                        size="small"
                        :disabled="disabled"
                        :placeholder="$t('flow.policy.textPlaceholder')"
                        @update:model-value="(value: string) => writeText(1, value)"
                    />
                </div>
            </template>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import type { PolicyField, PolicyOperator, ValueKind } from './policyModel';
import { emptyValueFor, VALUE_EDITORS } from './policyModel';

/**
 * 策略条件值编辑器：按字段类型分派控件形态，按操作符的期望值形态决定单值/多值/区间。
 *
 * 控件由字段/参数携带的 editorKey 决定（后端注册表把字段类型换算成编辑器标识后随 schema 下发），
 * 未登记的标识降级为文本输入，因此后端新增字段类型可以先上线，前端后续再补专门编辑器。
 * 宽度一律由外层 div 承载：EP 组件根元素自带宽度声明，工具类直接写在组件上会被覆盖。
 * 下拉不做 :teleported="false"：宿主是可滚动的抽屉，关掉 teleport 会让弹层被卡片边界裁切。
 */
const props = withDefaults(
    defineProps<{
        field: PolicyField;
        operator: PolicyOperator | null;
        disabled?: boolean;
    }>(),
    { disabled: false }
);

// 条件值（v-model）：单值/区间/多选形态由操作符的 valueKind 决定，写入统一走下方 write* 函数
const model = defineModel<unknown>();

const { t } = useI18n();

const valueKind = computed<ValueKind>(() => props.operator?.valueKind ?? 'single');
const isMulti = computed(() => valueKind.value === 'multi');
const isRange = computed(() => valueKind.value === 'range');

const kind = computed(() => {
    if (isMulti.value && props.field.type !== 'enum') return 'tags';
    // 未登记的标识按文本框处理：后端只会下发注册表里的标识，落到这里说明它没登记，
    // 此时渲染文本框比整块条件编辑区报错更可用，也更接近「新类型先注册再用」的约定
    return VALUE_EDITORS[props.field.editorKey] ?? 'input';
});

const options = computed(() => props.field.options ?? []);

// rangeHint 把 schema 声明的数值边界写在控件旁（边界由后端注册表随 schema 下发，前端不另维一份）。
// 只提示不钳位：越界要由校验给出可定位的中文提示，而不是让控件静默改写用户填的数
const rangeHint = computed(() => {
    const { min, max } = props.field;
    if (min === undefined && max === undefined) return '';
    if (min !== undefined && max !== undefined) return t('flow.policy.rangeBoth', { min, max });
    if (min !== undefined) return t('flow.policy.rangeMin', { min });
    return t('flow.policy.rangeMax', { max });
});

const optionLabel = (option: string) => {
    const key = `flow.option.${props.field.key}.${option}`;
    const label = t(key);
    // 文案缺失时回退原始取值：新场景不阻塞前端发版，也不显示无意义的 key 串
    return label === key ? option : label;
};

const asArray = (value: unknown): unknown[] => (Array.isArray(value) ? value : value === undefined || value === null ? [] : [value]);

const stringList = computed<string[]>(() => asArray(model.value).map((item) => String(item)));
const selectValue = computed(() => (isMulti.value ? stringList.value : (stringList.value[0] ?? '')));

const textAt = (index: number) => String(asArray(isRange.value ? model.value : [model.value])[index] ?? '');

const writeText = (index: number, value: string) => {
    if (!isRange.value) {
        model.value = value;
        return;
    }
    const bounds = [textAt(0), textAt(1)];
    bounds[index] = value;
    model.value = bounds;
};

const numberAt = (index: number) => {
    const raw = asArray(isRange.value ? model.value : [model.value])[index];
    if (raw === undefined || raw === null || raw === '') return undefined;
    const numeric = Number(raw);
    return Number.isNaN(numeric) ? undefined : numeric;
};

const writeNumber = (index: number, value: number | undefined) => {
    if (!isRange.value) {
        model.value = value;
        return;
    }
    const bounds: (number | undefined)[] = [numberAt(0), numberAt(1)];
    bounds[index] = value;
    model.value = bounds;
};

const writeBool = (value: boolean | string | number) => {
    model.value = Boolean(value);
};

const writeTags = (value: string[]) => {
    model.value = value;
};

// 切换字段或操作符后可能留下与新形态不匹配的取值，这里统一按新形态收敛
const onSelectChange = (value: unknown) => {
    if (isMulti.value) {
        model.value = asArray(value).map((item) => String(item));
        return;
    }
    const single = asArray(value)[0];
    model.value = single === undefined ? emptyValueFor('single') : single;
};
</script>

<style lang="scss" scoped>
.policy-value-editor {
    display: inline-flex;
    min-width: 0;

    .value-row {
        display: flex;
        align-items: center;
        gap: 4px;
        min-width: 0;
    }

    .value-cell {
        width: 170px;
        min-width: 0;

        :deep(.el-select),
        :deep(.el-input),
        :deep(.el-input-number) {
            width: 100%;
        }
    }

    .editor-hint {
        // 范围提示是 .value-cell 的兄弟节点，写在 .value-cell 里会编译成后代选择器而永不命中
        font-size: 12px;
        color: var(--el-text-color-secondary);
        white-space: nowrap;
    }

    .range-sep {
        color: var(--el-text-color-secondary);
        font-size: 12px;
    }
}
</style>
