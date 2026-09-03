<template>
    <el-drawer v-model="visible" :title="isEdit ? $t('ai.integration.editMcpServer') : $t('ai.integration.newMcpServer')" size="560px" append-to-body :close-on-click-modal="false">
        <el-form ref="formRef" :model="form" :rules="rules" label-width="110px" label-position="left">
            <el-form-item :label="$t('ai.integration.mcpName')" prop="name">
                <el-input v-model="form.name" :placeholder="$t('ai.integration.mcpNamePlaceholder')" />
            </el-form-item>
            <el-form-item :label="$t('ai.integration.mcpCode')" prop="code">
                <el-input v-model="form.code" :disabled="isEdit" :placeholder="$t('ai.integration.mcpCodePlaceholder')" />
            </el-form-item>
            <el-form-item :label="$t('ai.integration.mcpDescription')">
                <el-input v-model="form.description" type="textarea" :rows="2" :placeholder="$t('ai.integration.mcpDescriptionPlaceholder')" />
            </el-form-item>
            <el-form-item :label="$t('ai.integration.mcpUrl')" prop="url">
                <el-input v-model="form.url" placeholder="https://example.com/mcp" />
            </el-form-item>
            <el-form-item :label="$t('ai.integration.mcpHeaders')">
                <div class="headers-editor">
                    <MonacoEditor v-model="form.headers" language="json" height="140px" />
                    <div class="field-tip">{{ $t('ai.integration.mcpHeadersPlaceholder') }}</div>
                </div>
            </el-form-item>
            <el-form-item :label="$t('ai.integration.mcpTimeout')">
                <el-input-number v-model="form.timeoutSec" :min="1" :max="600" controls-position="right" />
                <span class="unit-label">s</span>
            </el-form-item>
            <el-form-item :label="$t('ai.integration.mcpEnabled')">
                <el-switch v-model="form.enabled" :active-value="1" :inactive-value="0" />
            </el-form-item>

            <!-- 测试连接（编辑态可用） -->
            <template v-if="isEdit">
                <el-divider content-position="left">{{ $t('ai.integration.mcpConnectionTest') }}</el-divider>
                <div class="test-block">
                    <el-button type="primary" plain :loading="testing" @click="handleTest">{{ $t('ai.integration.mcpTestConnect') }}</el-button>
                    <span v-if="tested" class="tool-count">{{ $t('ai.integration.mcpToolCount', { count: discoveredTools.length }) }}</span>
                    <div v-if="discoveredTools.length" class="tool-list">
                        <div v-for="tool in discoveredTools" :key="tool.name" class="tool-item">
                            <span class="tool-name">{{ tool.name }}</span>
                            <span class="tool-desc">{{ tool.description }}</span>
                        </div>
                    </div>
                    <div v-else-if="tested && !testing" class="tool-empty">{{ $t('ai.integration.mcpNoTools') }}</div>
                </div>
            </template>
        </el-form>

        <template #footer>
            <el-button @click="visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" :loading="saving" @click="handleSave">{{ $t('common.save') }}</el-button>
        </template>
    </el-drawer>
</template>

<script lang="ts" setup>
import { computed, nextTick, reactive, ref, watch } from 'vue';
import type { FormInstance, FormRules } from 'element-plus';
import { useI18n } from 'vue-i18n';
import { Msg } from '@/hooks/useI18n';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { pluginApi } from './api';
import type { McpServer } from './types';

const { t } = useI18n();

const props = defineProps<{
    modelValue: boolean;
    /** 编辑态传入服务器实体，新建态为 null */
    server: McpServer | null;
    /** 打开时自动执行连接测试并回显工具（列表页「查看工具」入口） */
    autoDiscover?: boolean;
}>();

const emit = defineEmits<{
    (e: 'update:modelValue', v: boolean): void;
    (e: 'saved'): void;
}>();

const visible = computed({
    get: () => props.modelValue,
    set: (v: boolean) => emit('update:modelValue', v),
});

const isEdit = computed(() => !!props.server);

const formRef = ref<FormInstance>();
const saving = ref(false);
const testing = ref(false);
const tested = ref(false);
const discoveredTools = ref<{ name: string; description: string }[]>([]);

const form = reactive({
    code: '',
    name: '',
    description: '',
    url: '',
    headers: '',
    timeoutSec: 30,
    enabled: 1,
});

const rules: FormRules = {
    name: [{ required: true, message: () => t('common.pleaseInput', { label: t('ai.integration.mcpName') }), trigger: 'blur' }],
    code: [
        { required: true, message: () => t('common.pleaseInput', { label: t('ai.integration.mcpCode') }), trigger: 'blur' },
        { pattern: /^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$/, message: () => t('ai.integration.mcpCodePattern'), trigger: 'blur' },
    ],
    url: [
        { required: true, message: () => t('common.pleaseInput', { label: t('ai.integration.mcpUrl') }), trigger: 'blur' },
        { pattern: /^https?:\/\//, message: () => t('ai.integration.mcpUrlPattern'), trigger: 'blur' },
    ],
};

// 紧凑 JSON 格式化为缩进形式（非法 JSON 原样返回，交给保存校验提示）
const prettyJson = (s: string) => {
    try {
        return JSON.stringify(JSON.parse(s), null, 2);
    } catch {
        return s;
    }
};

watch(
    () => props.modelValue,
    (open) => {
        if (!open) return;
        tested.value = false;
        discoveredTools.value = [];
        formRef.value?.clearValidate();
        if (props.server) {
            Object.assign(form, {
                code: props.server.code,
                name: props.server.name,
                description: props.server.description,
                url: props.server.url,
                headers: prettyJson(props.server.headers || ''),
                timeoutSec: props.server.timeoutSec,
                enabled: props.server.enabled,
            });
            if (props.autoDiscover) {
                nextTick(() => handleTest());
            }
        } else {
            Object.assign(form, { code: '', name: '', description: '', url: '', headers: '', timeoutSec: 30, enabled: 1 });
        }
    }
);

const handleTest = async () => {
    testing.value = true;
    tested.value = false;
    try {
        discoveredTools.value = await pluginApi.discoverMcpTools.request({ id: props.server!.id });
        tested.value = true;
    } catch {
        // 测试失败（请求层已 toast），tested 保持 false
    } finally {
        testing.value = false;
    }
};

const handleSave = async () => {
    const valid = await formRef.value?.validate().then(() => true).catch(() => false);
    if (!valid) return;
    // headers 需为合法 JSON 对象（后端同样校验）
    if (form.headers.trim()) {
        try {
            JSON.parse(form.headers);
        } catch {
            Msg.error('ai.integration.mcpHeadersInvalid');
            return;
        }
    }
    saving.value = true;
    try {
        const body = { ...form };
        if (isEdit.value && props.server) {
            await pluginApi.updateMcpServer.request({ ...body, id: props.server.id });
        } else {
            await pluginApi.createMcpServer.request(body);
        }
        Msg.success('common.saveSuccess');
        visible.value = false;
        emit('saved');
    } catch {
        // 保存失败（请求层已 toast），抽屉保持打开以便修改重试
    } finally {
        saving.value = false;
    }
};
</script>

<style lang="scss" scoped>
.headers-editor {
    width: 100%;

    .field-tip {
        margin-top: 4px;
        font-size: 12px;
        line-height: 1.4;
        color: var(--el-text-color-secondary);
    }
}

.unit-label {
    margin-left: 8px;
    color: var(--el-text-color-secondary);
}

.test-block {
    display: flex;
    flex-direction: column;
    gap: 8px;

    .tool-count {
        font-size: 12px;
        color: var(--el-text-color-secondary);
    }

    .tool-list {
        display: flex;
        flex-direction: column;
        gap: 4px;
        max-height: 220px;
        overflow: auto;
        padding: 8px;
        border: 1px solid var(--el-border-color-lighter);
        border-radius: 6px;

        .tool-item {
            display: flex;
            flex-direction: column;

            .tool-name {
                font-size: 13px;
                font-weight: 600;
                color: var(--el-text-color-primary);
            }

            .tool-desc {
                font-size: 12px;
                color: var(--el-text-color-secondary);
            }
        }
    }

    .tool-empty {
        font-size: 12px;
        color: var(--el-text-color-placeholder);
    }
}
</style>
