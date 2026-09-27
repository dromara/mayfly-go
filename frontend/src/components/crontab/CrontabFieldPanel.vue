<template>
    <div class="flex flex-col gap-3">
        <el-radio-group :model-value="rule.kind" size="small" :disabled="disabled" @change="onKindChange">
            <el-radio-button v-for="kind in field.kinds" :key="kind" :value="kind">{{ $t(RULE_KIND_LABEL_KEYS[kind]) }}</el-radio-button>
        </el-radio-group>

        <div v-if="rule.kind === 'cycle'" class="flex items-center gap-2">
            <span class="text-sm text-muted-foreground">{{ $t('components.crontab.fromLabel') }}</span>
            <el-input-number v-model="cycleFrom" class="w-28!" size="small" :min="field.min" :max="field.max" :disabled="disabled" controls-position="right" />
            <span class="text-sm text-muted-foreground">{{ $t('components.crontab.toLabel') }}</span>
            <el-input-number v-model="cycleTo" class="w-28!" size="small" :min="field.min" :max="field.max" :disabled="disabled" controls-position="right" />
        </div>

        <div v-else-if="rule.kind === 'step'" class="flex items-center gap-2">
            <span class="text-sm text-muted-foreground">{{ $t('components.crontab.startLabel') }}</span>
            <el-input-number v-model="stepStart" class="w-28!" size="small" :min="field.min" :max="field.max" :disabled="disabled" controls-position="right" />
            <span class="text-sm text-muted-foreground">{{ $t('components.crontab.everyLabel') }}</span>
            <el-input-number v-model="stepEvery" class="w-28!" size="small" :min="1" :max="stepMax" :disabled="disabled" controls-position="right" />
        </div>

        <div v-else-if="rule.kind === 'list'">
            <div class="mb-1.5 flex items-center gap-3">
                <el-link type="primary" underline="never" :disabled="disabled" @click="onSelectAll">{{ $t('components.crontab.selectAll') }}</el-link>
                <el-link type="primary" underline="never" :disabled="disabled" @click="onClearAll">{{ $t('components.crontab.clearAll') }}</el-link>
            </div>
            <div class="cron-chips" :style="{ gridTemplateColumns: `repeat(${field.columns}, minmax(0, 1fr))` }">
                <button
                    v-for="value in values"
                    :key="value"
                    type="button"
                    class="cron-chip"
                    :class="{ 'is-active': selectedValues.includes(value) }"
                    :disabled="disabled"
                    @click="onToggleValue(value)"
                >
                    {{ valueText(value) }}
                </button>
            </div>
        </div>

        <p class="text-xs text-muted-foreground">{{ description }}</p>
        <p v-if="hint" class="text-xs text-muted-foreground">{{ hint }}</p>
    </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { describeRule, fieldValues, RULE_KIND_LABEL_KEYS, ruleValueLabelKey, ruleValues, type CronField, type CronRule, type CronRuleKind } from './cronSpec';

const props = withDefaults(
    defineProps<{
        /** 字段元数据（边界、可选规则、取值名称） */
        field: CronField;
        /** 该字段的当前规则，真源在父级，本组件只读并回写完整的新规则 */
        rule: CronRule;
        disabled?: boolean;
        /** 字段级补充说明（如日/周互斥），为空则不占行 */
        hint?: string;
    }>(),
    { disabled: false, hint: '' }
);

const emits = defineEmits<{
    /** 规则变更 */
    'update:rule': [rule: CronRule];
}>();

const { t } = useI18n();

const values = computed(() => fieldValues(props.field));

const selectedValues = computed<number[]>(() => (props.rule.kind === 'list' ? props.rule.values : []));

/** 步长上界即取值区间宽度，再大等价于只命中起始值 */
const stepMax = computed(() => props.field.max - props.field.min + 1);

const description = computed(() => {
    const desc = describeRule(props.field, props.rule);
    return t(desc.key, { unit: t(props.field.labelKey), ...desc.params });
});

/** 当前规则覆盖到的区间端点，用于切换规则类型时延续用户已选的范围 */
const edges = computed(() => {
    const covered = ruleValues(props.field, props.rule);
    const first = covered[0] ?? props.field.min;
    return { first, last: Math.max(first, covered[covered.length - 1] ?? props.field.min) };
});

const cycle = computed(() => (props.rule.kind === 'cycle' ? { from: props.rule.from, to: props.rule.to } : { from: edges.value.first, to: edges.value.last }));

const step = computed(() => (props.rule.kind === 'step' ? { start: props.rule.start, every: props.rule.every } : { start: edges.value.first, every: 1 }));

// 数值控件绑定「读规则 / 写回整条新规则」的计算属性，不留本地副本，避免与真源分叉
// 区间不弹回用户刚输入的值：改起始值时顺带抬高截止值，改截止值时顺带压低起始值，且始终不产生后端拒绝的回绕写法
const cycleFrom = computed<number | undefined>({
    get: () => cycle.value.from,
    set: (value) => {
        if (value === undefined) return;
        emitRule({ kind: 'cycle', from: value, to: Math.max(value, cycle.value.to) });
    },
});

const cycleTo = computed<number | undefined>({
    get: () => cycle.value.to,
    set: (value) => {
        if (value === undefined) return;
        emitRule({ kind: 'cycle', from: Math.min(value, cycle.value.from), to: value });
    },
});

const stepStart = computed<number | undefined>({
    get: () => step.value.start,
    set: (value) => {
        if (value === undefined) return;
        emitRule({ kind: 'step', start: value, every: step.value.every });
    },
});

const stepEvery = computed<number | undefined>({
    get: () => step.value.every,
    set: (value) => {
        if (value === undefined) return;
        emitRule({ kind: 'step', start: step.value.start, every: Math.max(1, value) });
    },
});

function emitRule(rule: CronRule) {
    emits('update:rule', rule);
}

function onKindChange(kind: string | number | boolean | undefined) {
    // el-radio-group 的 change 载荷不泛型化，但候选项只能来自 field.kinds，此处必为 CronRuleKind
    emitRule(ruleForKind(kind as CronRuleKind));
}

/** 新规则以当前覆盖的取值起步，切一次类型不必再从头输入 */
function ruleForKind(kind: CronRuleKind): CronRule {
    const { first, last } = edges.value;
    switch (kind) {
        case 'none':
            return { kind: 'none' };
        case 'all':
            return { kind: 'all' };
        case 'cycle':
            return { kind: 'cycle', from: first, to: last };
        case 'step':
            return { kind: 'step', start: first, every: step.value.every };
        case 'list':
            return { kind: 'list', values: [first] };
    }
}

/** 全选即不加约束，回写 `*` 而不是全量列表：后端只给 `*`/`?` 带 star 位，`1-31` 这类全量列表语义不同 */
function onSelectAll() {
    emitRule({ kind: 'all' });
}

/** 清空即「没有指定值」，回到该字段的缺省形态，避免产出无法序列化的空列表 */
function onClearAll() {
    emitRule(props.field.allowNone ? { kind: 'none' } : { kind: 'all' });
}

function onToggleValue(value: number) {
    const selected = new Set(selectedValues.value);
    if (selected.has(value)) {
        selected.delete(value);
    } else {
        selected.add(value);
    }
    if (!selected.size) {
        onClearAll();
        return;
    }
    emitRule({ kind: 'list', values: [...selected].sort((a, b) => a - b) });
}

function valueText(value: number): string {
    const labelKey = ruleValueLabelKey(props.field, value);
    return labelKey ? t(labelKey) : String(value);
}
</script>

<style scoped>
.cron-chips {
    display: grid;
    gap: 4px;
}

.cron-chip {
    height: 26px;
    border: 1px solid var(--el-border-color);
    border-radius: 4px;
    background: var(--el-fill-color-blank);
    color: var(--el-text-color-regular);
    font-size: 12px;
    line-height: 24px;
    cursor: pointer;
    user-select: none;
    transition:
        border-color 0.15s,
        background-color 0.15s,
        color 0.15s;
}

.cron-chip:hover:not(:disabled) {
    border-color: var(--el-color-primary);
    color: var(--el-color-primary);
}

.cron-chip.is-active {
    border-color: var(--el-color-primary);
    background: var(--el-color-primary);
    color: var(--el-color-white);
}

.cron-chip:disabled {
    cursor: not-allowed;
    opacity: 0.5;
}
</style>
