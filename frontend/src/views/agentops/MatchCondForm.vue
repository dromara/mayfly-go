<template>
    <div class="agentops-matchcond">
        <!-- 脏数据降级：库里存了表单认识不了的 JSON（非法/含扩展键）时不猜不吞，退回原始编辑并说明原因。
             静默用空对象覆盖等于用户保存一次就无痕丢掉既有条件 -->
        <template v-if="rawMode">
            <el-alert type="warning" :closable="false" show-icon :title="$t('agentops.matchCondInvalid')" class="mb-2" />
            <el-input type="textarea" :rows="3" v-model="rawText" @input="emitRaw" />
            <el-button link type="primary" class="mt-1" @click="leaveRawMode">{{ $t('agentops.matchCondUseVisual') }}</el-button>
        </template>
        <div v-else class="agentops-matchcond__grid">
            <!-- 字段名用独立小标签行：不占 el-select 的 #label 插槽（那是「已选项展示」位，放了字段名会盖掉选中值） -->
            <div class="agentops-matchcond__field">
                <span class="agentops-matchcond__label">{{ $t('agentops.matchResType') }}</span>
                <el-select v-model="cond.resourceType" clearable filterable :placeholder="$t('agentops.matchResTypePh')" @change="onTypeChange">
                    <el-option v-for="rt in typeOptions" :key="rt.key" :label="$t(rt.label)" :value="rt.key">
                        <span>{{ $t(rt.label) }}</span>
                        <span class="agentops-matchcond__opt-key">{{ rt.key }}</span>
                    </el-option>
                </el-select>
            </div>

            <!-- 资源白名单按名字选（id 只作落库值）：不选类型时资源无从列起，禁用并提示原因 -->
            <div class="agentops-matchcond__field">
                <span class="agentops-matchcond__label">{{ $t('agentops.matchResIds') }}</span>
                <el-select
                    v-model="cond.resourceIds"
                    multiple
                    filterable
                    clearable
                    collapse-tags
                    collapse-tags-tooltip
                    :disabled="!cond.resourceType"
                    :placeholder="$t(cond.resourceType ? 'agentops.matchResIdsPh' : 'agentops.matchResIdsNeedType')"
                    :loading="resourcesLoading"
                >
                    <el-option v-for="r in resourceOptions" :key="r.id" :label="resourceLabel(r)" :value="Number(r.id)" />
                </el-select>
            </div>

            <div class="agentops-matchcond__field">
                <span class="agentops-matchcond__label">{{ $t('agentops.matchSeverities') }}</span>
                <el-select v-model="cond.severities" multiple collapse-tags collapse-tags-tooltip clearable :placeholder="$t('agentops.matchSeveritiesPh')">
                    <!-- alert.priority* 文案自带 P0~P3 前缀（如「P0 紧急」），不再拼 code 避免「P0 · P0 紧急」 -->
                    <el-option v-for="p in priorityOptions" :key="p.code" :label="$t(p.label)" :value="p.code" />
                </el-select>
            </div>

            <!-- 指标允许自造（allow-create）：告警事件的 metric 不止评估器注册的这些，webhook 来源可自由命名 -->
            <div class="agentops-matchcond__field">
                <span class="agentops-matchcond__label">{{ $t('agentops.matchMetrics') }}</span>
                <el-select v-model="cond.metrics" multiple filterable allow-create default-first-option collapse-tags collapse-tags-tooltip clearable :placeholder="$t('agentops.matchMetricsPh')">
                    <el-option v-for="m in metricOptions" :key="m.key" :label="m.display" :value="m.key" />
                </el-select>
            </div>
        </div>
        <div class="agentops-matchcond__preview">
            {{ $t('agentops.matchCondPreview', { json: rawMode ? rawText || '-' : serialized || $t('agentops.matchCondAllEvents') }) }}
        </div>
    </div>
</template>

<script setup lang="ts">
/**
 * 触发器匹配条件的可视化编辑器。
 *
 * 替代原「手敲 JSON」的 Monaco 输入：resourceType 的合法取值、指标名、资源 id 都藏在后端
 * 代码与数据库里，配置者根本无从得知。这里把四个维度落成控件，数据全部来自既有真源——
 * 资源类型与指标来自告警的 metrics 元信息接口（后端下发，不手抄清单），资源白名单复用
 * 聊天 `@` 菜单的 chat resources 接口（选名字存 id），严重级用告警优先级枚举（P0~P3 是
 * 后端 Code() 的中立标识）。表单值仍是 JSON 字符串，后端契约零改动。
 */
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { chatResourceApi } from './api';
import { parseMatchCond, serializeMatchCond, type MatchCond } from './matchCond';
import { AlertPriorityEnum } from '@/views/ops/alert/enums';
import { fetchResourceMetrics, getResourceTypeI18nKey } from '@/views/ops/alert/utils';
import type { ResourceMetrics } from '@/views/ops/alert/types';
import type { ChatResource } from './types';

const props = defineProps<{ modelValue?: string | null }>();
const emit = defineEmits<{ 'update:modelValue': [string] }>();

const { t } = useI18n();

const cond = ref<MatchCond>({});
const rawMode = ref(false);
const rawText = ref('');

/** 资源类型选项：key（落库/匹配用）+ 数值（指标联动用）+ i18n label，全部来自后端元信息 */
const resourceTypes = ref<{ key: string; value: number; label: string }[]>([]);
/** 指标元信息原始列表（按类型收窄候选时直接查它，不另发请求也不另缓存） */
const metricsMeta = ref<ResourceMetrics[]>([]);
/** 全类型指标并集（未选类型时的候选；选了类型再按类型收窄） */
const allMetrics = ref<{ key: string; display: string }[]>([]);
const chatResources = ref<ChatResource[]>([]);
const resourcesLoading = ref(false);

// 外部值 → 内部状态：编辑抽屉回填、以及保存失败后表单值被父级重置的场景
watch(
    () => props.modelValue,
    (val) => {
        const { cond: parsed, invalid } = parseMatchCond(val);
        if (invalid) {
            rawMode.value = true;
            rawText.value = (val ?? '').trim();
            cond.value = {};
        } else if (!rawMode.value) {
            cond.value = parsed;
        }
    },
    { immediate: true }
);

const serialized = computed(() => serializeMatchCond(cond.value));

// 内部状态 → 表单值（raw 模式下由输入事件负责，避免双向回写打断输入）
watch([cond, serialized], () => {
    if (!rawMode.value) {
        emit('update:modelValue', serialized.value);
    }
}, { deep: true });

onMounted(async () => {
    // 指标元信息：类型清单与指标清单一次取回（alert 模块已做进程内缓存，这里直接复用）
    const list = await fetchResourceMetrics();
    metricsMeta.value = list;
    resourceTypes.value = list
        .map((it) => ({ key: it.resourceKey, value: Number(it.resourceType), label: getResourceTypeI18nKey(Number(it.resourceType)) }))
        .filter((it) => it.key)
        .sort((a, b) => a.value - b.value);
    const seen = new Set<string>();
    allMetrics.value = list
        .flatMap((it) => it.metrics)
        .filter((m) => (seen.has(m.key) ? false : (seen.add(m.key), true)))
        .map((m) => metricDisplay(m));

    resourcesLoading.value = true;
    try {
        chatResources.value = (await chatResourceApi.list.request()) ?? [];
    } finally {
        resourcesLoading.value = false;
    }
});

/** 指标 label 可能是 i18n key（含点）也可能是裸指标名（后端未配文案时），按形态分流 */
function metricDisplay(m: { key: string; label: string }): { key: string; display: string } {
    return { key: m.key, display: m.label.includes('.') ? t(m.label) : m.label };
}

/** 已存值里的类型可能不在评估器清单里（如只发 webhook 的自定义类型）：补一个裸键选项，回填不丢值 */
const typeOptions = computed(() => {
    const opts = [...resourceTypes.value];
    const cur = cond.value.resourceType;
    if (cur && !opts.some((o) => o.key === cur)) {
        opts.unshift({ key: cur, value: -1, label: cur });
    }
    return opts;
});

const metricOptions = computed(() => {
    const rt = cond.value.resourceType;
    if (!rt) {
        return allMetrics.value;
    }
    const hit = metricsMeta.value.find((it) => it.resourceKey === rt);
    return hit ? hit.metrics.map((m) => metricDisplay(m)) : allMetrics.value;
});

const resourceOptions = computed(() => {
    const rt = cond.value.resourceType;
    return rt ? chatResources.value.filter((r) => r.resourceType === rt) : [];
});

/** 资源展示名带定位信息（同名资源靠 ip/code 区分），落库仍是 id */
function resourceLabel(r: ChatResource): string {
    const detail = r.ip ? `${r.ip}:${r.port || 22}` : r.code || r.description || '';
    return detail ? `${r.name}（${detail}）` : r.name;
}

const priorityOptions = Object.values(AlertPriorityEnum).map((ev) => ({
    // 事件侧严重级是后端 AlertPriority.Code() 的 P0~P3（数值即优先级），文案复用告警优先级枚举
    code: `P${ev.value as number}`,
    label: ev.label,
}));

/** 类型决定可用资源与指标语义，换类型后原选择不再成立，清空避免残留脏值（与告警规则表单同判据） */
function onTypeChange() {
    cond.value.resourceIds = undefined;
    cond.value.metrics = undefined;
}

function emitRaw() {
    emit('update:modelValue', rawText.value.trim());
}

/** 从降级模式切回可视化：以当前文本再解析一次，能解析才切（切不过去保留原文本不丢数据） */
function leaveRawMode() {
    const { cond: parsed, invalid } = parseMatchCond(rawText.value);
    if (invalid) {
        return;
    }
    rawMode.value = false;
    cond.value = parsed;
    emit('update:modelValue', serializeMatchCond(parsed));
}
</script>

<style lang="scss" scoped>
.agentops-matchcond {
    &__grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 8px;
    }

    &__field {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    &__label {
        font-size: 12px;
        color: var(--el-text-color-secondary);
    }

    &__opt-key {
        float: right;
        margin-left: 12px;
        font-size: 12px;
        color: var(--el-text-color-secondary);
    }

    &__preview {
        margin-top: 6px;
        font-size: 12px;
        color: var(--el-text-color-secondary);
        word-break: break-all;
    }
}
</style>
