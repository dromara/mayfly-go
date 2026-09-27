<template>
    <auto-form-drawer
        ref="drawerRef"
        v-model:visible="visible"
        :title="isEdit ? $t('ai.integration.editMcpServer') : $t('ai.integration.newMcpServer')"
        :items="formItems"
        :data="editData"
        size="560px"
        append-to-body
        :confirm-api="handleSave"
        @submitted="emit('saved')"
        @opened="onOpened"
    >
        <template #headers="{ form }">
            <div class="headers-editor">
                <MonacoEditor v-model="form.headers" language="json" height="140px" />
                <div class="field-tip">{{ $t('ai.integration.mcpHeadersPlaceholder') }}</div>
            </div>
        </template>
        <template #timeoutSec="{ form }">
            <el-input-number v-model="form.timeoutSec" :min="1" :max="600" controls-position="right" />
            <span class="unit-label">s</span>
        </template>
        <!-- 测试连接（编辑态可用） -->
        <template #testArea>
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

        <template #footer>
            <el-button @click="visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" :loading="drawerRef?.submitting" @click="drawerRef?.submit()">{{ $t('common.save') }}</el-button>
        </template>
    </auto-form-drawer>
</template>

<script lang="ts" setup>
import { computed, nextTick, ref, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { Msg } from '@/hooks/useI18n';
import { AutoFormDrawer, defineFormItems } from '@/components/auto-form';
import { useAutoFormModel } from '@/hooks/useAutoFormModel';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { pluginApi, type PluginInstance } from './api';

const { t } = useI18n();

const props = defineProps<{
    modelValue: boolean;
    /** 编辑态传入插件实例（连接配置内联在 config），新建态为 null */
    server: PluginInstance | null;
    /** 打开时自动执行连接测试并回显工具（列表页「查看工具」入口） */
    autoDiscover?: boolean;
}>();

const emit = defineEmits<{
    (e: 'update:modelValue', v: boolean): void;
    (e: 'saved'): void;
}>();

const visible = defineModel<boolean>({ default: false });

const isEdit = computed(() => !!props.server);

const drawerRef = useTemplateRef<{
    validate: (...args: unknown[]) => Promise<unknown>;
    clearValidate?: () => void;
    submitting: boolean;
    submit: () => Promise<void>;
}>('drawerRef');
const testing = ref(false);
const tested = ref(false);
const discoveredTools = ref<{ name: string; description: string }[]>([]);

// 编辑态原始启停值（保存后若变化需经启停接口同步）
const origEnabled = ref(1);

/** MCP 实例连接配置（内联在 instance.config 里，后端为跨插件类型共用的自由 map） */
interface McpInstanceConfig {
    url: string;
    headers: string;
    timeoutSec: number;
}

/** MCP 实例编辑表单 */
interface McpForm {
    code: string;
    name: string;
    description: string;
    enabled: number;
}

/**
 * 从实例 config 读出 MCP 连接配置
 *
 * config 是后端自由 map，本函数是唯一的边界转换点（逐键校类型，代替在读取点反复断言）。
 */
function mcpConfigOf(config?: Record<string, unknown>): McpInstanceConfig {
    return {
        url: typeof config?.url === 'string' ? config.url : '',
        headers: typeof config?.headers === 'string' ? config.headers : '',
        timeoutSec: typeof config?.timeoutSec === 'number' ? config.timeoutSec : 30,
    };
}

/** 表单声明（enabled/测试连接区仅编辑态展示，headers/timeoutSec 为 custom 插槽） */
const formItems = computed(() =>
    defineFormItems<McpForm & McpInstanceConfig>([
        {
            prop: 'name',
            label: 'ai.integration.mcpName',
            required: true,
            placeholder: 'ai.integration.mcpNamePlaceholder',
            rules: [{ required: true, message: () => t('common.pleaseInput', { label: t('ai.integration.mcpName') }), trigger: 'blur' }],
        },
        {
            prop: 'code',
            label: 'ai.integration.mcpCode',
            required: true,
            placeholder: 'ai.integration.mcpCodePlaceholder',
            disabled: () => isEdit.value,
            rules: [
                { required: true, message: () => t('common.pleaseInput', { label: t('ai.integration.mcpCode') }), trigger: 'blur' },
                { pattern: /^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$/, message: () => t('ai.integration.mcpCodePattern'), trigger: 'blur' },
            ],
        },
        {
            prop: 'description',
            label: 'ai.integration.mcpDescription',
            type: 'textarea',
            props: { rows: 2 },
            placeholder: 'ai.integration.mcpDescriptionPlaceholder',
        },
        {
            prop: 'url',
            label: 'ai.integration.mcpUrl',
            required: true,
            placeholder: 'https://example.com/mcp',
            rules: [
                { required: true, message: () => t('common.pleaseInput', { label: t('ai.integration.mcpUrl') }), trigger: 'blur' },
                { pattern: /^https?:\/\//, message: () => t('ai.integration.mcpUrlPattern'), trigger: 'blur' },
            ],
        },
        { prop: 'headers', label: 'ai.integration.mcpHeaders', type: 'custom' },
        { prop: 'timeoutSec', label: 'ai.integration.mcpTimeout', type: 'custom' },
        { prop: 'enabled', label: 'ai.integration.mcpEnabled', type: 'switch', when: () => isEdit.value, props: { 'active-value': 1, 'inactive-value': 0 } },
        { prop: 'testArea', type: 'custom', when: () => isEdit.value },
    ])
);

// 紧凑 JSON 格式化为缩进形式（非法 JSON 原样返回，交给保存校验提示）
const prettyJson = (s: string) => {
    try {
        return JSON.stringify(JSON.parse(s), null, 2);
    } catch {
        return s;
    }
};

/** 传给 AutoFormDrawer 的回填数据：连接配置内联在实例 config */
const editData = computed<McpForm & McpInstanceConfig>(() => {
    if (props.server) {
        const cfg = mcpConfigOf(props.server.config);
        return {
            code: props.server.code,
            name: props.server.name,
            description: props.server.description,
            url: cfg.url,
            headers: prettyJson(cfg.headers),
            timeoutSec: cfg.timeoutSec,
            enabled: props.server.enabled,
        };
    }
    return { code: '', name: '', description: '', url: '', headers: '', timeoutSec: 30, enabled: 1 };
});

// 宿主抽屉的内部表单在 @opened 接管（提交组装基于它）
const { openedWith, requireForm } = useAutoFormModel<McpForm & McpInstanceConfig>();

// 打开时重置测试状态，编辑态记录原始启停值；列表页「查看工具」入口自动执行连接测试
watch(visible, (open) => {
    if (!open) return;
    tested.value = false;
    discoveredTools.value = [];
    if (props.server) {
        origEnabled.value = props.server.enabled;
        if (props.autoDiscover) {
            nextTick(() => handleTest());
        }
    }
});

/** 抽屉打开且回填完成后清残留校验状态 */
const onOpened = openedWith(() => {
    drawerRef.value?.clearValidate?.();
});

const handleTest = async () => {
    testing.value = true;
    tested.value = false;
    try {
        discoveredTools.value = await pluginApi.discoverInstance.request({ id: props.server!.id });
        tested.value = true;
    } catch {
        // 测试失败（请求层已 toast），tested 保持 false
    } finally {
        testing.value = false;
    }
};

// confirmApi 提交动作：headers 前置校验（失败抛错中止，组件保持抽屉打开）+ 组装后提交；成功提示与关闭抽屉由组件内置逻辑处理
const handleSave = async () => {
    const form = requireForm();
    // headers 需为合法 JSON 对象（后端同样校验）
    if (form.headers?.trim()) {
        try {
            JSON.parse(form.headers);
        } catch {
            Msg.error('ai.integration.mcpHeadersInvalid');
            throw new Error('invalid mcp headers');
        }
    }
    // 连接配置内联进 instance.config，提交统一实例接口
    const body = {
        pluginType: 'mcp' as const,
        code: form.code,
        name: form.name,
        description: form.description,
        config: { url: form.url, headers: form.headers, timeoutSec: form.timeoutSec },
    };
    if (isEdit.value && props.server) {
        await pluginApi.updateInstance.request({ ...body, id: props.server.id });
        // 启停独立于实例元数据，保存后若变化经启停接口同步
        if (form.enabled !== origEnabled.value) {
            await pluginApi.toggleInstance.request({ id: props.server.id, enabled: form.enabled });
        }
    } else {
        // 新建实例默认启用（enabled 开关仅编辑态展示）
        await pluginApi.createInstance.request(body);
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
