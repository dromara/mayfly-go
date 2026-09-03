<template>
    <div class="component-container card p-2!">
        <div class="plugin-header">
            <span class="plugin-title">{{ $t('ai.integration.pluginManagement') }}</span>
            <div class="header-actions">
                <el-button plain icon="upload" @click="zipInputRef?.click()">{{ $t('ai.integration.importZip') }}</el-button>
                <el-button type="primary" icon="plus" @click="pickerOpen = true">{{ $t('ai.integration.newPlugin') }}</el-button>
            </div>
        </div>
        <input ref="zipInputRef" type="file" accept=".zip" class="zip-input" @change="handleImportFile" />

        <!-- 统一插件列表（对齐 tokhub：技能与 MCP 服务器混合渲染，类型标签区分） -->
        <div v-loading="loading" class="plugin-body">
            <template v-if="items.length">
                <div v-for="item in items" :key="item.kind + item.id" class="plugin-card" @click="openEdit(item)">
                    <div class="card-head">
                        <span class="type-icon" :class="item.kind === 'skill' ? 'is-skill' : 'is-mcp'">
                            <el-icon :size="18"><FolderOpened v-if="item.kind === 'skill'" /><Connection v-else /></el-icon>
                        </span>
                        <span class="name">{{ item.name }}</span>
                        <el-tag v-if="item.kind === 'skill'" size="small" :type="item.status === 'published' ? 'success' : 'warning'">
                            {{ item.status === 'published' ? $t('ai.integration.statusPublished') : $t('ai.integration.statusDraft') }}
                        </el-tag>
                        <el-tag v-else size="small" :type="item.enabled === 1 ? 'success' : 'info'">
                            {{ item.enabled === 1 ? $t('common.enabled') : $t('common.disabled') }}
                        </el-tag>
                    </div>
                    <div class="card-desc">{{ item.description || '-' }}</div>
                    <div class="card-meta">
                        <el-tag size="small" effect="plain" :type="item.kind === 'skill' ? 'primary' : 'success'">
                            {{ $t(item.kind === 'skill' ? 'ai.integration.skillPlugin' : 'ai.integration.mcpPlugin') }}
                        </el-tag>
                        <span class="code">{{ item.code }}</span>
                        <span v-if="item.kind === 'skill' && item.version" class="version">v{{ item.version }}</span>
                    </div>
                    <div class="card-footer" @click.stop>
                        <!-- MCP：连接测试 / 工具查看 -->
                        <template v-if="item.kind === 'mcp'">
                            <el-button size="small" plain @click="openMcp(item)">{{ $t('ai.integration.testConnect') }}</el-button>
                            <el-button size="small" plain @click="openMcp(item, true)">{{ $t('ai.integration.viewTools') }}</el-button>
                        </template>
                        <!-- 技能：发布 / 取消发布 / 导出 -->
                        <template v-else>
                            <el-button v-if="item.status === 'draft'" size="small" plain type="success" @click="togglePublish(item, true)">{{ $t('ai.integration.publish') }}</el-button>
                            <el-button v-else size="small" plain @click="togglePublish(item, false)">{{ $t('ai.integration.unpublish') }}</el-button>
                            <el-button size="small" plain @click="exportSkill(item)">{{ $t('ai.integration.exportZip') }}</el-button>
                        </template>
                        <el-button size="small" plain type="primary" @click="openEdit(item)">{{ $t('common.edit') }}</el-button>
                        <el-button size="small" plain type="danger" @click="remove(item)">{{ $t('common.delete') }}</el-button>
                    </div>
                </div>
            </template>
            <div v-else-if="!loading" class="plugin-empty">
                <el-icon :size="42" color="var(--el-text-color-placeholder)"><Box /></el-icon>
                <div class="empty-title">{{ $t('ai.integration.emptyText') }}</div>
                <div class="empty-desc">{{ $t('ai.integration.emptyDesc') }}</div>
            </div>
        </div>

        <!-- 类型选择 -->
        <PluginTypePicker v-model="pickerOpen" @pick="handlePick" />
        <!-- MCP 表单（autoDiscover：打开后自动连接测试并回显工具） -->
        <McpFormDrawer v-model="mcpDrawerOpen" :server="editingMcp" :auto-discover="mcpAutoDiscover" @saved="loadAll" />
        <!-- 技能编辑器 -->
        <SkillEditor v-model="skillEditorOpen" :skill-id="editingSkillId" @saved="loadAll" />
    </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue';
import { Box, Connection, FolderOpened } from '@element-plus/icons-vue';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import config from '@/common/config';
import { joinClientParams } from '@/common/request';
import { downloadFile } from '@/common/utils/file';
import { pluginApi } from './api';
import PluginTypePicker from './PluginTypePicker.vue';
import McpFormDrawer from './McpFormDrawer.vue';
import SkillEditor from './SkillEditor.vue';
import type { Skill, McpServer, PluginType } from './types';

/** 统一插件视图（技能 + MCP 服务器混合渲染，对齐 tokhub 单一卡片列表） */
interface UnifiedPlugin {
    kind: PluginType;
    id: string;
    name: string;
    code: string;
    description: string;
    /** 技能：published / draft */
    status?: string;
    version?: string;
    /** MCP：1 启用 / 0 停用 */
    enabled?: number;
}

const loading = ref(false);
const skills = ref<Skill[]>([]);
const mcpServers = ref<McpServer[]>([]);

const items = computed<UnifiedPlugin[]>(() => [
    ...skills.value.map((s) => ({ kind: 'skill' as const, id: s.id, name: s.name, code: s.code, description: s.description || '', status: s.status, version: s.version })),
    ...mcpServers.value.map((m) => ({ kind: 'mcp' as const, id: m.id, name: m.name, code: m.code, description: m.description || '', enabled: m.enabled })),
]);

const pickerOpen = ref(false);
const mcpDrawerOpen = ref(false);
const mcpAutoDiscover = ref(false);
const editingMcp = ref<McpServer | null>(null);
const skillEditorOpen = ref(false);
const editingSkillId = ref<string | null>(null);
const zipInputRef = ref();

const loadAll = async () => {
    loading.value = true;
    try {
        [skills.value, mcpServers.value] = await Promise.all([pluginApi.listSkills.request(), pluginApi.listMcpServers.request()]);
    } finally {
        loading.value = false;
    }
};

onMounted(loadAll);

// ── 新建（类型选择器分发） ─────────────────────────────────
const handlePick = (type: PluginType) => {
    if (type === 'skill') {
        editingSkillId.value = null;
        skillEditorOpen.value = true;
    } else {
        mcpAutoDiscover.value = false;
        editingMcp.value = null;
        mcpDrawerOpen.value = true;
    }
};

// ── 编辑 ──────────────────────────────────────────────────
const openMcp = (item: UnifiedPlugin, autoDiscover = false) => {
    mcpAutoDiscover.value = autoDiscover;
    editingMcp.value = mcpServers.value.find((m) => m.id === item.id) || null;
    mcpDrawerOpen.value = true;
};

const openSkillEdit = (item: UnifiedPlugin) => {
    editingSkillId.value = item.id;
    skillEditorOpen.value = true;
};

const openEdit = (item: UnifiedPlugin) => (item.kind === 'skill' ? openSkillEdit(item) : openMcp(item));

// ── 技能发布 / 取消发布 / 导出 ─────────────────────────────
const togglePublish = async (item: UnifiedPlugin, publish: boolean) => {
    try {
        if (publish) await pluginApi.publishSkill.request({ id: item.id });
        else await pluginApi.unpublishSkill.request({ id: item.id });
        Msg.success('common.operateSuccess');
        await loadAll();
    } catch {
        // 失败（请求层已 toast），列表状态保持不变
    }
};

const exportSkill = (item: UnifiedPlugin) => {
    downloadFile(`${config.baseApiUrl}/ai/plugin/skills/${item.id}/export?${joinClientParams()}`);
};

// ── 技能 zip 导入 ─────────────────────────────────────────
const handleImportFile = async (e: Event) => {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    input.value = '';
    if (!file) return;
    const fd = new FormData();
    fd.append('file', file);
    try {
        await pluginApi.importSkillZip.request(fd);
        Msg.success('ai.integration.importSuccess');
        await loadAll();
    } catch {
        // 导入失败（请求层已 toast），静默退出
    }
};

// ── 删除 ──────────────────────────────────────────────────
const remove = async (item: UnifiedPlugin) => {
    try {
        if (item.kind === 'skill') {
            await useI18nConfirm('ai.integration.deleteSkillConfirm', { name: item.name });
            await pluginApi.deleteSkill.request({ id: item.id });
        } else {
            await useI18nConfirm('ai.integration.deleteMcpConfirm', { name: item.name });
            await pluginApi.deleteMcpServer.request({ id: item.id });
        }
        Msg.success('common.deleteSuccess');
        await loadAll();
    } catch {
        // 确认取消或删除失败（请求层已 toast），静默退出
    }
};
</script>

<style lang="scss" scoped>
.component-container {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: auto;

    .plugin-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 14px;

        .plugin-title {
            font-size: 16px;
            font-weight: 600;
        }

        .header-actions {
            display: flex;
            gap: 0;
        }
    }

    .zip-input {
        display: none;
    }

    .plugin-body {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
        gap: 12px;
        align-content: start;
    }

    .plugin-card {
        display: flex;
        flex-direction: column;
        gap: 6px;
        padding: 14px;
        border: 1px solid var(--el-border-color-light);
        border-radius: 8px;
        cursor: pointer;
        transition: all 0.2s;

        &:hover {
            border-color: var(--el-color-primary);
            box-shadow: var(--el-box-shadow-light);
        }

        .card-head {
            display: flex;
            align-items: center;
            gap: 10px;

            .type-icon {
                display: flex;
                align-items: center;
                justify-content: center;
                width: 32px;
                height: 32px;
                border-radius: 6px;
                flex-shrink: 0;

                &.is-skill {
                    color: var(--el-color-primary);
                    background: var(--el-color-primary-light-9);
                }

                &.is-mcp {
                    color: var(--el-color-success);
                    background: var(--el-color-success-light-9);
                }
            }

            .name {
                flex: 1;
                min-width: 0;
                font-size: 14px;
                font-weight: 600;
                overflow: hidden;
                text-overflow: ellipsis;
                white-space: nowrap;
            }
        }

        .card-desc {
            font-size: 12px;
            color: var(--el-text-color-secondary);
            display: -webkit-box;
            -webkit-line-clamp: 2;
            -webkit-box-orient: vertical;
            overflow: hidden;
            min-height: 18px;
        }

        .card-meta {
            display: flex;
            align-items: center;
            gap: 8px;

            .code {
                font-size: 12px;
                color: var(--el-text-color-secondary);
                font-family: monospace;
                overflow: hidden;
                text-overflow: ellipsis;
                white-space: nowrap;
            }

            .version {
                font-size: 12px;
                color: var(--el-text-color-secondary);
            }
        }

        .card-footer {
            display: flex;
            align-items: center;
            gap: 0;
            margin-top: 2px;
        }
    }

    .plugin-empty {
        grid-column: 1 / -1;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 6px;
        padding: 48px 0;

        .empty-title {
            font-size: 14px;
            color: var(--el-text-color-primary);
            margin-top: 8px;
        }

        .empty-desc {
            font-size: 12px;
            color: var(--el-text-color-secondary);
        }
    }
}
</style>
