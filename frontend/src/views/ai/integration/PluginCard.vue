<template>
    <div class="plugin-card" @click="emit('edit')">
        <div class="card-head">
            <span class="type-icon" :class="item.pluginType === 'skill' ? 'is-skill' : 'is-mcp'">
                <el-icon :size="18"><FolderOpened v-if="item.pluginType === 'skill'" /><Connection v-else /></el-icon>
            </span>
            <span class="name">{{ item.name }}</span>
            <!-- MCP：启停 + 实例健康状态；技能：引用技能发布状态 -->
            <template v-if="item.pluginType === 'mcp'">
                <el-tag size="small" :type="item.status === InstanceStatus.Healthy ? 'success' : item.status === InstanceStatus.Error ? 'danger' : 'info'">
                    {{ $t(item.status === InstanceStatus.Healthy ? 'ai.integration.instanceHealthy' : item.status === InstanceStatus.Error ? 'ai.integration.instanceError' : 'ai.integration.instanceUnknown') }}
                </el-tag>
                <el-tag size="small" :type="item.enabled === 1 ? 'success' : 'info'" effect="plain">
                    {{ item.enabled === 1 ? $t('common.enabled') : $t('common.disabled') }}
                </el-tag>
            </template>
            <el-tag v-else size="small" :type="item.skillStatus === 'published' ? 'success' : 'warning'">
                {{ item.skillStatus === 'published' ? $t('ai.integration.statusPublished') : $t('ai.integration.statusDraft') }}
            </el-tag>
        </div>
        <div class="card-desc">{{ item.description || '-' }}</div>
        <div class="card-meta">
            <el-tag size="small" effect="plain" :type="item.pluginType === 'skill' ? 'primary' : 'success'">
                {{ $t(item.pluginType === 'skill' ? 'ai.integration.skillPlugin' : 'ai.integration.mcpPlugin') }}
            </el-tag>
            <span class="code">{{ item.code }}</span>
            <span v-if="item.pluginType === 'skill' && item.skillVersion" class="version">v{{ item.skillVersion }}</span>
        </div>
        <div class="card-footer" @click.stop>
            <!-- MCP：连接测试 / 工具查看 -->
            <template v-if="item.pluginType === 'mcp'">
                <el-button size="small" plain @click="emit('test')">{{ $t('ai.integration.testConnect') }}</el-button>
                <el-button size="small" plain @click="emit('tools')">{{ $t('ai.integration.viewTools') }}</el-button>
            </template>
            <!-- 技能：发布 / 取消发布 / 导出 -->
            <template v-else>
                <el-button v-if="item.skillStatus === 'draft'" size="small" plain type="success" @click="emit('publish')">{{ $t('ai.integration.publish') }}</el-button>
                <el-button v-else size="small" plain @click="emit('unpublish')">{{ $t('ai.integration.unpublish') }}</el-button>
                <el-button size="small" plain @click="emit('export')">{{ $t('ai.integration.exportZip') }}</el-button>
            </template>
            <el-button size="small" plain type="primary" @click="emit('edit')">{{ $t('common.edit') }}</el-button>
            <el-button size="small" plain type="danger" @click="emit('delete')">{{ $t('common.delete') }}</el-button>
        </div>
    </div>
</template>

<script setup lang="ts">
/**
 * PluginCard - 统一插件实例卡片（skill / mcp 混合渲染，PluginCard）
 * 纯展示组件：操作语义经 emit 上抛，编排逻辑收敛在 PluginManagement 页面
 */
import { Connection, FolderOpened } from '@element-plus/icons-vue';
import type { PluginInstance } from './api';
import { InstanceStatus } from './types';

defineProps<{ item: PluginInstance }>();

const emit = defineEmits<{
    (e: 'edit'): void;
    /** MCP：连接测试 */
    (e: 'test'): void;
    /** MCP：查看工具 */
    (e: 'tools'): void;
    (e: 'publish'): void;
    (e: 'unpublish'): void;
    (e: 'export'): void;
    (e: 'delete'): void;
}>();
</script>

<style lang="scss" scoped>
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
</style>
