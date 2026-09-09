<template>
    <Transition name="trigger-menu">
        <div
            v-if="visible"
            ref="panelRef"
            class="trigger-menu resource-tree-panel"
            :style="menuStyle"
            @mousedown.prevent
        >
            <div class="resource-tree-panel__header">{{ t('ai.chat.resourceTreeTitle') }}</div>
            <TreeContainer
                class="resource-tree-panel__tree"
                :load-root="loadRoot"
                :filter-text="queryText"
                :show-actions="false"
                :interactive="false"
                :transform-node="decorateNode"
                event-scope="ai"
                @node-click="onNodeClick"
            />
            <div class="resource-tree-panel__tip">{{ t('ai.chat.resourceTreeTip') }}</div>
        </div>
    </Transition>
</template>

<script lang="ts" setup>
/**
 * ResourceTreePanel - `@` 触发的资源引用树面板
 *
 * 复用 ops 资源树的贡献者数据链路（TreeContainer + Contributor），
 * 与 ops 资源管理树同源，引用层级对齐团队资源分配的授权粒度：
 * - 机器：tag 分组 → 机器节点 → 授权凭证叶子（选到凭证才算完整引用，
 *   使 MachineCommandExec 的 authCertName 参数不再缺失触发中断补全）
 * - 数据库：tag 分组 → 数据库配置节点 → 物理 database 叶子
 *   （选到库使 db 工具的 dbName 参数直接可取；库记录绑定的授权账号
 *   username/authCertName 一并透传进引用元数据）
 */
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { loadResourceTags } from '@/views/ops/resource/resource';
import TreeContainer from '@/views/ops/resource/tree/TreeContainer.vue';
import { isNodeSelectable } from '@/views/ops/resource/tree';
import type { TreeNode, TreeNodeData } from '@/views/ops/resource/tree/types';
// db 资源树物理库节点 kind：引用面板将库节点叶子化（不展开表/schema）
import { DbKind } from '@/views/ops/db/resource';
import { MachineAuthCertKind } from '@/views/ops/machine/resource';

/** 树叶子选中后上抛的引用数据（extra 随芯片下发后端） */
export interface ResourceTreeSelectPayload {
    resourceType: 'machine' | 'db';
    label: string;
    description?: string;
    extra: Record<string, string>;
}

const props = defineProps<{
    visible: boolean;
    menuStyle?: Record<string, string>;
    /** 触发词后的查询文本，用于过滤树节点 */
    query?: string;
}>();

const emit = defineEmits<{
    (e: 'select', payload: ResourceTreeSelectPayload): void;
    (e: 'close'): void;
}>();

const { t } = useI18n();
const panelRef = ref<HTMLElement | null>(null);

/** 供父组件用 floating-ui 定位（floating 参照元素必须是面板自身 DOM） */
defineExpose({ panelRef });

// 机器与数据库两类资源树（叶子分别为授权凭证 / 物理 database）
const loadRoot = () => loadResourceTags([TagResourceTypeEnum.Machine.value, TagResourceTypeEnum.DbInstance.value]);

/** 物理 database 节点叶子化：引用选到库，不展开表/schema */
const decorateNode = (node: TreeNodeData): TreeNodeData => {
    if (node.kind === DbKind) {
        return { ...node, hasChildren: false };
    }
    return node;
};

/**
 * 可选叶子：由贡献者 selectable 声明单源判定（机器凭证/物理 database 均声明为选择目标），
 * 面板不再硬编码 kind 清单
 */
const onNodeClick = (node: TreeNode) => {
    if (!isNodeSelectable(node)) {
        return;
    }
    const m = node.params as Record<string, any>;

    if (node.kind === MachineAuthCertKind) {
        // 机器凭证叶子：params = {...machineVO, selectAuthCert}
        const cert = m.selectAuthCert || {};
        const username = cert.username || cert.name || '';
        emit('select', {
            resourceType: 'machine',
            label: `${m.name}(${username})`,
            description: `${username}@${m.ip}:${m.port}`,
            extra: {
                resourceType: 'machine',
                id: String(m.id ?? ''),
                code: String(m.code ?? ''),
                ip: String(m.ip ?? ''),
                port: String(m.port ?? ''),
                authCertName: String(cert.name ?? ''),
                username: String(username),
            },
        });
        emit('close');
        return;
    }
    // 物理 database 叶子：label 对齐 ops 打开 tab 的 `配置名/库名` 形式；
    // extra 携带 dbId/dbName/账号 完整定位标识，db 工具参数直接可取不中断
    const dbName = String(m.db ?? '');
    emit('select', {
        resourceType: 'db',
        label: dbName ? `${m.name}/${dbName}` : String(m.name ?? ''),
        description: String(m.code ?? ''),
        extra: {
            resourceType: 'db',
            id: String(m.id ?? ''),
            code: String(m.code ?? ''),
            db: dbName,
            authCertName: String(m.authCertName ?? ''),
            username: String(m.username ?? ''),
        },
    });
    emit('close');
};

// 查询文本过滤树节点（容器内部防抖；懒加载树只过滤已加载节点）
const queryText = computed(() => props.query || '');
</script>

<style lang="scss" scoped>
.resource-tree-panel {
    /* 悬浮浮层：与 TriggerMenu 的 .trigger-menu 一致（scoped 隔离，需在此重复声明，
       否则静态定位挤占文档流把输入框顶起，floating-ui 的 top/left 也不生效） */
    position: fixed;
    z-index: 1000;
    width: 300px;
    max-height: 320px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    padding: 6px 0 4px;
    background: var(--el-bg-color-overlay);
    border: 1px solid var(--el-border-color-light);
    border-radius: 10px;
    box-shadow: var(--el-box-shadow-light);

    &__header {
        padding: 2px 12px 6px;
        font-size: 11px;
        color: var(--el-text-color-secondary);
        border-bottom: 1px solid var(--el-border-color-lighter);
    }

    &__tree {
        flex: 1;
        min-height: 0;
        overflow: hidden;
        padding: 4px 6px;
    }

    &__tip {
        padding: 4px 12px 2px;
        font-size: 10px;
        color: var(--el-text-color-placeholder);
        border-top: 1px solid var(--el-border-color-lighter);
    }
}
</style>
