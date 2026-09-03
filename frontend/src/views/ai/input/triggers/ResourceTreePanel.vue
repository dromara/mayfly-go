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
            <el-tree
                ref="treeRef"
                class="resource-tree-panel__tree"
                lazy
                :load="loadNode"
                :props="treeProps"
                :filter-node-method="filterNode"
                :empty-text="t('ai.chat.resourceTreeEmpty')"
                :expand-on-click-node="false"
                :indent="12"
                @node-click="onNodeClick"
            >
                <template #default="{ data }">
                    <span class="resource-tree-panel__node" :class="{ 'is-selectable': isSelectable(data), 'is-disabled': data.disabled }">
                        <SvgIcon v-if="data.icon" :size="13" :name="data.icon.name" :color="data.icon.color" />
                        <span class="resource-tree-panel__label" :title="data.label">{{ data.label }}</span>
                    </span>
                </template>
            </el-tree>
            <div class="resource-tree-panel__tip">{{ t('ai.chat.resourceTreeTip') }}</div>
        </div>
    </Transition>
</template>

<script lang="ts" setup>
/**
 * ResourceTreePanel - `@` 触发的资源引用树面板
 *
 * 复用团队编辑的资源树数据链路（loadResourceTags + TagTreeNode 懒加载），
 * 与 ops 资源管理树同源，引用层级对齐团队资源分配的授权粒度：
 * - 机器：tag 分组 → 机器节点 → 授权凭证叶子（选到凭证才算完整引用，
 *   使 MachineCommandExec 的 authCertName 参数不再缺失触发中断补全）
 * - 数据库：tag 分组 → 数据库配置节点 → 物理 database 叶子
 *   （选到库使 db 工具的 dbName 参数直接可取；库记录绑定的授权账号
 *   username/authCertName 一并透传进引用元数据）
 */
import { nextTick, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import SvgIcon from '@/components/svg-icon/index.vue';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { TagTreeNode } from '@/views/ops/component/tag';
import { loadResourceTags } from '@/views/ops/resource/resource';

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
const treeRef = ref<{ filter: (val: string) => void } | null>(null);

/** 供父组件用 floating-ui 定位（floating 参照元素必须是面板自身 DOM） */
defineExpose({ panelRef });

const treeProps = {
    label: 'label',
    children: 'zones',
    isLeaf: 'isLeaf',
    disabled: 'disabled',
};

// ops 数据库资源树的物理库节点类型值（ops/db/resource/index.ts 中硬编码，
// 未进 TagResourceTypeEnum；引用面板将物理库叶子化——不展开表/schema，
// 引用粒度到库即满足 db 工具 dbId+dbName 定位）
const NODE_TYPE_DB = 223;

const loadNode = async (node: { level: number; data: TagTreeNode }, resolve: (data: TagTreeNode[]) => void) => {
    try {
        let nodes: TagTreeNode[];
        if (node.level === 0) {
            // 机器与数据库两类资源树（叶子分别为授权凭证 / 物理 database）
            nodes = (await loadResourceTags([TagResourceTypeEnum.Machine.value, TagResourceTypeEnum.DbInstance.value])) as TagTreeNode[];
        } else {
            nodes = (await node.data.loadChildren()) as TagTreeNode[];
        }
        const parentParams = (node.data.params as Record<string, any> | undefined) || {};
        nodes.forEach((n) => {
            if (n.type?.value === NODE_TYPE_DB) {
                // 物理 database 叶子化：引用选到库（不展开表/schema）
                n.withIsLeaf(true);
                // 库记录层（父节点）绑定的授权账号透传进引用元数据
                const p = (n.params as Record<string, any>) || {};
                p.username = p.username || parentParams.username || '';
                p.authCertName = p.authCertName || parentParams.authCertName || '';
            }
        });
        resolve(nodes);
    } catch (e) {
        console.error('[ai] load resource tree failed:', e);
        resolve([]);
    }
};

/**
 * 可选叶子：机器授权凭证 / 物理 database。
 * 注意：ops 树中凭证节点的 NodeType 硬编码 12（不等于 TagResourceTypeEnum.AuthCert 的 5），
 * 因此按 params 语义判定而非类型枚举：凭证叶子必携带 selectAuthCert；
 * 物理 database 叶子（223）携带 id+db（已在 loadNode 中叶子化）
 */
const isSelectable = (data: TagTreeNode): boolean => {
    if (data.disabled) return false;
    const m = data.params as Record<string, any> | undefined;
    if (!data.isLeaf || !m) return false;
    return !!m.selectAuthCert || (m.id != null && m.code != null);
};

const onNodeClick = (data: TagTreeNode) => {
    const m = data.params as Record<string, any> | undefined;
    if (!isSelectable(data) || !m) return;

    if (m.selectAuthCert) {
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

// 查询文本过滤树节点（懒加载树只过滤已加载节点）
const filterNode = (value: string, data: TagTreeNode): boolean => {
    if (!value) return true;
    return data.label.includes(value);
};

watch(
    () => props.query,
    (val) => {
        nextTick(() => treeRef.value?.filter(val || ''));
    },
);
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
        overflow: auto;
        padding: 4px 6px;

        :deep(.el-tree-node__content) {
            height: 28px;
            border-radius: 6px;
        }
    }

    &__node {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        min-width: 0;
        font-size: 13px;

        &.is-selectable {
            cursor: pointer;

            &:hover {
                color: var(--el-color-primary);
            }
        }

        &.is-disabled {
            opacity: 0.45;
            cursor: not-allowed;
        }
    }

    &__label {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    &__tip {
        padding: 4px 12px 2px;
        font-size: 10px;
        color: var(--el-text-color-placeholder);
        border-top: 1px solid var(--el-border-color-lighter);
    }
}
</style>
