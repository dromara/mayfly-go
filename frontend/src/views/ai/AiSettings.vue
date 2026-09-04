<template>
    <div class="h-full p-4 overflow-auto">
        <el-card class="max-w-5xl! mx-auto">
            <template #header>
                <div class="flex justify-between items-center">
                    <span>{{ $t('menu.aiSettings') }}</span>
                    <el-button v-auth="'config:save'" type="primary" :loading="state.saving" @click="onSave">
                        {{ $t('common.save') }}
                    </el-button>
                </div>
            </template>

            <!-- label 置顶：长标签（如"上下文窗口（Token）"）不受固定 label 宽度截断 -->
            <el-form label-position="top" v-loading="state.loading">
                <!-- 主模型（AiModelConfig.value 顶层字段） -->
                <el-divider content-position="left">{{ $t('ai.settings.modelSection') }}</el-divider>
                <el-row :gutter="16">
                    <el-col :xs="24" :md="12">
                        <el-form-item :label="$t('system.sysconf.aiModel')" required>
                            <el-input v-model="state.model.model" :placeholder="$t('system.sysconf.aiModelPlaceholder')" clearable />
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :md="12">
                        <el-form-item :label="$t('common.name')">
                            <el-input v-model="state.model.name" clearable />
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :md="16">
                        <el-form-item :label="$t('system.sysconf.aiBaseUrl')">
                            <el-input v-model="state.model.baseUrl" :placeholder="$t('system.sysconf.aiBaseUrlPlaceholder')" clearable />
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :md="8">
                        <el-form-item :label="$t('system.sysconf.aiEnableThinking')">
                            <el-select v-model="state.model.enableThinking" clearable class="w-full!">
                                <el-option label="true" value="true" />
                                <el-option label="false" value="false" />
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :md="16">
                        <el-form-item :label="$t('system.sysconf.aiApiKey')">
                            <el-input v-model="state.model.apiKey" type="password" show-password autocomplete="new-password" clearable />
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :md="8">
                        <el-form-item :label="$t('ai.settings.timeOut')">
                            <el-input-number v-model="state.model.timeOut" :min="1" controls-position="right" class="w-full!" />
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="8">
                        <el-form-item :label="$t('ai.settings.temperature')">
                            <el-input-number v-model="state.model.temperature" :min="0" :max="2" :step="0.1" controls-position="right" class="w-full!" />
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="8">
                        <el-form-item :label="$t('system.sysconf.aiMaxTokens')">
                            <el-input-number v-model="state.model.maxTokens" :min="0" :step="1024" controls-position="right" class="w-full!" />
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="8">
                        <el-form-item :label="$t('ai.settings.contextWindow')">
                            <el-input-number v-model="state.model.contextWindow" :min="0" :step="1000" controls-position="right" class="w-full!" />
                        </el-form-item>
                    </el-col>
                </el-row>

                <!-- 调用重试与故障转移（AiModelConfig.value.retry / failover） -->
                <el-divider content-position="left">{{ $t('ai.settings.failoverSection') }}</el-divider>
                <el-row :gutter="16">
                    <el-col :xs="24" :sm="12" :lg="8">
                        <el-form-item :label="$t('ai.settings.maxRetries')">
                            <el-input-number v-model="state.model.maxRetries" :min="0" controls-position="right" class="w-full!" />
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :lg="8">
                        <el-form-item :label="$t('ai.settings.maxFailovers')">
                            <el-input-number v-model="state.model.maxFailovers" :min="0" controls-position="right" class="w-full!" />
                            <div class="form-tip">{{ $t('ai.settings.failoverDesc') }}</div>
                        </el-form-item>
                    </el-col>
                </el-row>

                <el-form-item :label="$t('ai.settings.fallbacks')">
                    <div class="w-full">
                        <el-table :data="state.model.fallbacks" stripe class="w-full!">
                            <el-table-column :label="$t('system.sysconf.aiModel')" min-width="180">
                                <template #default="scope">
                                    <el-input v-model="scope.row.model" :placeholder="$t('system.sysconf.aiModelPlaceholder')" clearable />
                                </template>
                            </el-table-column>
                            <el-table-column :label="$t('system.sysconf.aiBaseUrl')" min-width="160">
                                <template #default="scope">
                                    <el-input v-model="scope.row.baseUrl" clearable />
                                </template>
                            </el-table-column>
                            <el-table-column :label="$t('system.sysconf.aiApiKey')" min-width="150">
                                <template #default="scope">
                                    <el-input v-model="scope.row.apiKey" type="password" show-password autocomplete="new-password" clearable />
                                </template>
                            </el-table-column>
                            <el-table-column :label="$t('common.name')" min-width="90">
                                <template #default="scope">
                                    <el-input v-model="scope.row.name" clearable />
                                </template>
                            </el-table-column>
                            <el-table-column :label="$t('ai.settings.timeOut')" width="140">
                                <template #default="scope">
                                    <el-input-number v-model="scope.row.timeOut" :min="1" controls-position="right" class="w-full!" />
                                </template>
                            </el-table-column>
                            <el-table-column :label="$t('common.operation')" width="60" align="center">
                                <template #default="scope">
                                    <el-button type="danger" icon="delete" plain size="small" @click="removeFallback(scope.$index)" />
                                </template>
                            </el-table-column>
                        </el-table>
                        <el-button class="mt-2" type="primary" icon="plus" plain size="small" @click="addFallback">
                            {{ $t('ai.settings.addFallback') }}
                        </el-button>
                    </div>
                </el-form-item>

                <!-- Agent 运行时（AiAgentConfig.value.toolSearchThreshold） -->
                <el-divider content-position="left">{{ $t('ai.settings.agentSection') }}</el-divider>
                <el-row :gutter="16">
                    <el-col :xs="24" :sm="12" :lg="8">
                        <el-form-item :label="$t('ai.settings.toolSearchThreshold')">
                            <el-input-number v-model="state.agent.toolSearchThreshold" :min="0" controls-position="right" class="w-full!" />
                            <div class="form-tip">{{ $t('ai.settings.toolSearchThresholdPlaceholder') }}</div>
                        </el-form-item>
                    </el-col>
                </el-row>
            </el-form>
        </el-card>
    </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive } from 'vue';
import { configApi } from '@/views/system/api';
import type { SysConfig } from '@/views/system/types';
import { Msg } from '@/hooks/useI18n';

/** 备用模型表单行（对齐后端 config.ModelConfig 的 failover.fallbacks 条目） */
interface FallbackForm {
    name?: string;
    model?: string;
    baseUrl?: string;
    apiKey?: string;
    timeOut?: number;
    temperature?: number;
    maxTokens?: number;
}

const emptyFallback = (): FallbackForm => ({});

const state = reactive({
    loading: false,
    saving: false,
    /** AiModelConfig / AiAgentConfig 配置行（保存时需携带 id） */
    modelRow: null as SysConfig | null,
    agentRow: null as SysConfig | null,
    /** 解析后的原始 value（保存时合并覆盖，保留本表单未覆盖的字段） */
    rawModelValue: {} as Record<string, unknown>,
    rawAgentValue: {} as Record<string, unknown>,
    model: {
        name: '',
        model: '',
        baseUrl: '',
        apiKey: '',
        timeOut: undefined as number | undefined,
        temperature: undefined as number | undefined,
        maxTokens: undefined as number | undefined,
        contextWindow: undefined as number | undefined,
        /** 三态字符串：''=未配置（对齐通用动态表单存字符串的既有行为） */
        enableThinking: '',
        maxRetries: 0,
        maxFailovers: 0,
        fallbacks: [] as FallbackForm[],
    },
    agent: {
        toolSearchThreshold: 0,
    },
});

const addFallback = () => {
    state.model.fallbacks.push(emptyFallback());
};

const removeFallback = (index: number) => {
    state.model.fallbacks.splice(index, 1);
};

const toNumber = (v: unknown): number | undefined => {
    const n = Number(v);
    return Number.isFinite(n) && n !== 0 ? n : undefined;
};

const load = async () => {
    state.loading = true;
    try {
        const [modelRes, agentRes] = await Promise.all([
            configApi.list.request({ key: 'AiModelConfig', pageNum: 1, pageSize: 1 } as any),
            configApi.list.request({ key: 'AiAgentConfig', pageNum: 1, pageSize: 1 } as any),
        ]);
        state.modelRow = modelRes.list?.[0] ?? null;
        state.agentRow = agentRes.list?.[0] ?? null;

        // value 解析失败按空配置处理（与 sysconfig.ts getAiModelConfig 容错一致）
        const parseValue = (row: SysConfig | null): Record<string, unknown> => {
            try {
                return row?.value ? JSON.parse(row.value) : {};
            } catch (e) {
                console.warn('[ai-settings] parse config value failed:', e);
                return {};
            }
        };
        state.rawModelValue = parseValue(state.modelRow);
        state.rawAgentValue = parseValue(state.agentRow);

        const m = state.rawModelValue;
        const retry = (m.retry ?? {}) as Record<string, unknown>;
        const failover = (m.failover ?? {}) as Record<string, unknown>;
        const fallbacks = Array.isArray(failover.fallbacks) ? (failover.fallbacks as Record<string, unknown>[]) : [];
        state.model = {
            name: (m.name as string) ?? '',
            model: (m.model as string) ?? '',
            baseUrl: (m.baseUrl as string) ?? '',
            apiKey: (m.apiKey as string) ?? '',
            timeOut: toNumber(m.timeOut),
            temperature: toNumber(m.temperature),
            maxTokens: toNumber(m.maxTokens),
            contextWindow: toNumber(m.contextWindow),
            // 通用动态表单存的是字符串 "true"/"false"，归一化为三态字符串回显
            enableThinking: m.enableThinking === true ? 'true' : m.enableThinking === false ? 'false' : String(m.enableThinking ?? ''),
            maxRetries: Number(retry.maxRetries) || 0,
            maxFailovers: Number(failover.maxFailovers) || 0,
            fallbacks: fallbacks.map((f) => ({
                name: (f.name as string) ?? '',
                model: (f.model as string) ?? '',
                baseUrl: (f.baseUrl as string) ?? '',
                apiKey: (f.apiKey as string) ?? '',
                timeOut: toNumber(f.timeOut),
                temperature: toNumber(f.temperature),
                maxTokens: toNumber(f.maxTokens),
            })),
        };
        state.agent.toolSearchThreshold = Number(state.rawAgentValue.toolSearchThreshold) || 0;
    } finally {
        state.loading = false;
    }
};

const saveConfig = async (row: SysConfig | null, key: string, name: string, value: Record<string, unknown>) => {
    const payload: Record<string, unknown> = { key, name, value: JSON.stringify(value) };
    if (row?.id) {
        payload.id = row.id;
    }
    await configApi.save.request(payload as any);
};

const onSave = async () => {
    if (!state.model.model?.trim()) {
        Msg.warning('ai.settings.modelRequired');
        return;
    }
    state.saving = true;
    try {
        await saveConfig(state.modelRow, 'AiModelConfig', 'system.sysconf.aiModelConf', buildModelValue());
        await saveConfig(state.agentRow, 'AiAgentConfig', 'menu.aiSettings.agentConfig', buildAgentValue());
        Msg.saveSuccess();
        await load();
    } finally {
        state.saving = false;
    }
};

/** 组装 AiModelConfig.value：在原始对象上覆盖表单字段，保留未知字段不丢失 */
const buildModelValue = (): Record<string, unknown> => {
    const value = { ...state.rawModelValue };
    const setOrDelete = (key: string, v: unknown) => {
        if (v === undefined || v === null || v === '') {
            delete value[key];
        } else {
            value[key] = v;
        }
    };

    setOrDelete('name', state.model.name?.trim());
    setOrDelete('model', state.model.model?.trim());
    setOrDelete('baseUrl', state.model.baseUrl?.trim());
    setOrDelete('apiKey', state.model.apiKey?.trim());
    setOrDelete('timeOut', toNumber(state.model.timeOut));
    setOrDelete('temperature', toNumber(state.model.temperature));
    setOrDelete('maxTokens', toNumber(state.model.maxTokens));
    setOrDelete('contextWindow', toNumber(state.model.contextWindow));
    // 思考模式：''=未配置（删除字段走后端 qwen 默认逻辑），否则存 boolean
    setOrDelete('enableThinking', state.model.enableThinking === '' ? undefined : state.model.enableThinking === 'true');

    // retry：未配置（0）删除字段，保持零重试行为
    const maxRetries = toNumber(state.model.maxRetries) ?? 0;
    if (maxRetries > 0) {
        value.retry = { maxRetries };
    } else {
        delete value.retry;
    }

    // failover：无有效备用模型删除字段；model 为空的行跳过（后端解析同样忽略）
    const fallbacks = state.model.fallbacks
        .filter((f) => f.model?.trim())
        .map((f) => ({
            ...(f.name?.trim() ? { name: f.name.trim() } : {}),
            model: f.model!.trim(),
            ...(f.baseUrl?.trim() ? { baseUrl: f.baseUrl.trim() } : {}),
            ...(f.apiKey?.trim() ? { apiKey: f.apiKey.trim() } : {}),
            ...(toNumber(f.timeOut) ? { timeOut: toNumber(f.timeOut) } : {}),
            ...(toNumber(f.temperature) ? { temperature: toNumber(f.temperature) } : {}),
            ...(toNumber(f.maxTokens) ? { maxTokens: toNumber(f.maxTokens) } : {}),
        }));
    if (fallbacks.length > 0) {
        const maxFailovers = toNumber(state.model.maxFailovers) ?? 0;
        value.failover = maxFailovers > 0 ? { maxFailovers, fallbacks } : { fallbacks };
    } else {
        delete value.failover;
    }
    return value;
};

/** 组装 AiAgentConfig.value */
const buildAgentValue = (): Record<string, unknown> => {
    const value = { ...state.rawAgentValue };
    const threshold = toNumber(state.agent.toolSearchThreshold) ?? 0;
    if (threshold > 0) {
        value.toolSearchThreshold = threshold;
    } else {
        delete value.toolSearchThreshold;
    }
    return value;
};

onMounted(load);
</script>
<style lang="scss" scoped>
/* 表单项内的说明文字：独占一行，颜色走 Element token 保证深浅主题一致 */
:deep(.el-form-item__content) .form-tip {
    width: 100%;
    margin-top: 2px;
    font-size: 12px;
    line-height: 1.6;
    color: var(--el-text-color-secondary);
}
</style>
