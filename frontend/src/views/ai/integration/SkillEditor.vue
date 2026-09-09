<template>
    <auto-form-drawer
        ref="drawerRef"
        v-model="visible"
        :title="drawerTitle"
        size="900px"
        append-to-body
        v-model:active-tab="activeTab"
        :tabs="drawerTabs"
        :data="defaultFormData"
        @opened="onOpened"
    >
        <!-- 头部右侧：状态信息 + 编辑态操作（发布 / zip 导入导出） -->
        <template #header-extra>
            <div class="top-bar">
                <el-tag size="small" type="info">{{ internalForm.code || '-' }}</el-tag>
                <el-tag v-if="skill" size="small">v{{ skill.version }}</el-tag>
                <el-tag size="small" :type="formStatus === 'published' ? 'success' : 'warning'">
                    {{ formStatus === 'published' ? $t('ai.integration.statusPublished') : $t('ai.integration.statusDraft') }}
                </el-tag>
                <template v-if="isEdit">
                    <el-button v-if="formStatus === 'published'" size="small" plain @click="togglePublish(false)">{{ $t('ai.integration.unpublish') }}</el-button>
                    <el-button v-else size="small" type="success" plain @click="togglePublish(true)">{{ $t('ai.integration.publish') }}</el-button>
                    <el-button size="small" plain @click="triggerImport">{{ $t('ai.integration.importZip') }}</el-button>
                    <el-button size="small" plain @click="handleExport">{{ $t('ai.integration.exportZip') }}</el-button>
                </template>
            </div>
            <input ref="zipInputRef" type="file" accept=".zip" class="zip-input" @change="handleImportFile" />
        </template>

        <!-- 元数据 instructions 字段（编辑态元数据加载中上遮罩，防止加载完成前编辑被覆盖） -->
        <template #instructions="{ form }">
            <div v-loading="loading" class="w-full">
                <MonacoEditor v-model="form.instructions" language="markdown" height="320px" />
            </div>
        </template>

        <!-- 资源文件 Tab（左文件树 / 右编辑器） -->
        <template #resourcePanel>
            <div class="resource-panel">
                <!-- 左侧文件树 -->
                <div class="tree-side">
                    <div class="tree-header">
                        <span class="tree-title">
                            {{ $t('ai.integration.resources') }}
                            <el-badge v-if="resourceCount > 0" :value="resourceCount" type="info" class="count-badge" />
                        </span>
                        <el-button size="small" text type="primary" @click="openNewFile">
                            <el-icon><Plus /></el-icon>
                            {{ $t('ai.integration.addResource') }}
                        </el-button>
                    </div>
                    <!-- 内联新增路径输入（Enter 确认 / Esc 取消，无按钮） -->
                    <div v-if="newFileOpen" class="new-file-row">
                        <el-input
                            ref="newFileInputRef"
                            v-model="newFilePath"
                            size="small"
                            :placeholder="$t('ai.integration.newResourcePlaceholder')"
                            @keydown.enter.prevent="handleCreateFile"
                            @keydown.esc.stop="closeNewFile"
                        />
                    </div>
                    <el-tree
                        v-if="treeData.length"
                        :data="treeData"
                        node-key="path"
                        :props="{ key: 'path', label: 'name', children: 'children' }"
                        :current-node-key="activeFilePath ?? undefined"
                        :default-expanded-keys="[...expandedItems]"
                        :expand-on-click-node="false"
                        highlight-current
                        @node-click="handleNodeClick"
                        class="resource-tree"
                    >
                        <template #default="{ data }">
                            <div class="tree-node">
                                <el-icon :size="14" :color="data.isDir ? 'var(--el-color-primary)' : fileTypeColor(data.name)">
                                    <Folder v-if="data.isDir" />
                                    <Document v-else />
                                </el-icon>
                                <span class="node-name">{{ data.name }}</span>
                                <el-icon v-if="!data.isDir" class="node-delete" :size="12" @click.stop="handleDeleteFile(data.path)">
                                    <Delete />
                                </el-icon>
                            </div>
                        </template>
                    </el-tree>
                    <div v-else class="tree-empty">{{ $t('ai.integration.noResources') }}</div>
                </div>

                <!-- 右侧编辑区 -->
                <div class="editor-side">
                    <template v-if="activeFilePath">
                        <div class="editor-header">
                            <span class="file-path">{{ activeFilePath }}</span>
                            <el-tag v-if="isDirty && isEdit" size="small" type="warning">{{ $t('ai.integration.unsaved') }}</el-tag>
                            <el-button v-if="isDirty && isEdit" size="small" type="primary" @click="handleSaveFileContent">{{ $t('common.save') }}</el-button>
                        </div>
                        <MonacoEditor :model-value="fileContent" :language="languageFromPath(activeFilePath)" height="100%" @update:model-value="handleContentChange" />
                    </template>
                    <div v-else class="editor-empty">{{ $t('ai.integration.noResources') }}</div>
                </div>
            </div>
        </template>

        <template #footer>
            <el-button @click="visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" :loading="saving" @click="handleSubmit">{{ $t('common.save') }}</el-button>
        </template>
    </auto-form-drawer>
</template>

<script lang="ts" setup>
import { computed, nextTick, ref, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { Plus, Folder, Document, Delete } from '@element-plus/icons-vue';
import config from '@/common/config';
import { joinClientParams } from '@/common/request';
import { downloadFile } from '@/common/utils/file';
import { Msg, useI18nConfirm, useI18nFormValidate } from '@/hooks/useI18n';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem, type AutoFormTab } from '@/components/auto-form';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { pluginApi } from './api';
import { buildTree, languageFromPath, fileTypeColor, parentDirs } from './utils';
import type { Skill, SkillResource, LocalResource } from './types';
import type { TreeNode } from './types';

const { t } = useI18n();

const props = defineProps<{
    modelValue: boolean;
    /** 编辑态传入技能 id，新建态为 null */
    skillId: string | null;
}>();

const emit = defineEmits<{
    (e: 'update:modelValue', v: boolean): void;
    (e: 'saved'): void;
}>();

const visible = defineModel<boolean>({ default: false });

const isEdit = computed(() => !!props.skillId);
const drawerTitle = computed(() => (isEdit.value ? t('ai.integration.editSkill') : t('ai.integration.newSkill')));

// 新建态创建成功后的技能 id：资源批量落库部分失败时保留，重试不再重复建技能
const createdSkillId = ref<string | null>(null);

// ── 元数据 ────────────────────────────────────────────────
const loading = ref(false);
const saving = ref(false);
const skill = ref<Skill | null>(null);
const formStatus = ref('draft');
const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => Promise<unknown> }>('drawerRef');

/** 表单声明（instructions 为 custom 插槽，description 字段承载原 field-tip 提示） */
const formItems = computed<AutoFormItem[]>(() => [
    {
        prop: 'code',
        label: 'ai.integration.skillCode',
        required: true,
        placeholder: 'e.g. code-review',
        description: 'ai.integration.skillCodeHint',
        disabled: () => isEdit.value,
        rules: [
            { required: true, message: () => t('common.pleaseInput', { label: t('ai.integration.skillCode') }), trigger: 'blur' },
            { pattern: /^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$/, message: () => t('ai.integration.skillCodePattern'), trigger: 'blur' },
        ],
    },
    { prop: 'description', label: 'ai.integration.skillDescription', type: 'textarea', props: { rows: 2 } },
    { prop: 'allowedTools', label: 'ai.integration.skillAllowedTools', placeholder: 'ai.integration.skillAllowedToolsPlaceholder' },
    {
        prop: 'instructions',
        label: 'ai.integration.skillInstructions',
        type: 'custom',
        required: true,
        description: 'ai.integration.skillInstructionsHint',
        rules: [{ required: true, message: () => t('common.pleaseInput', { label: t('ai.integration.skillInstructions') }), trigger: 'blur' }],
    },
]);

/** 页签布局：basic 元数据表单 / resources 资源文件管理（resourcePanel 为 custom 插槽） */
const drawerTabs = computed<AutoFormTab[]>(() => [
    { name: 'basic', label: 'ai.integration.tabBasic', items: formItems.value },
    { name: 'resources', label: 'ai.integration.tabResources', items: [{ prop: 'resourcePanel', type: 'custom' }] },
]);

/** 回填数据恒为空默认值：编辑态元数据经 @opened 异步加载后填充（内部 form 引用暂存于 internalForm） */
const defaultFormData = { code: '', description: '', allowedTools: '', instructions: '' } as unknown as AutoFormData;

/** 抽屉打开后暂存的内部表单引用（header-extra 状态展示、异步回填与提交均基于它） */
const internalForm = ref<AutoFormData>({});

const onOpened = async (form: AutoFormData) => {
    internalForm.value = form;
    resetAll(form);
    if (!props.skillId) return;
    loading.value = true;
    try {
        const [s, ins] = await Promise.all([
            pluginApi.getSkill.request({ id: props.skillId }),
            pluginApi.getInstructions.request({ id: props.skillId }),
        ]);
        skill.value = s;
        formStatus.value = s.status;
        Object.assign(form, {
            code: s.code,
            description: s.description || '',
            allowedTools: s.allowedTools || '',
            instructions: ins?.content || '',
        });
    } finally {
        loading.value = false;
    }
};

// ── 资源状态（编辑态 DB / 新建态本地暂存） ──────
const resources = ref<SkillResource[]>([]);
const localResources = ref<LocalResource[]>([]);
const resourcesLoaded = ref(false);
const activeTab = ref('basic');
const resourceCount = computed(() => (isEdit.value ? resources.value.length : localResources.value.length));
const treeData = computed<TreeNode[]>(() => buildTree(isEdit.value ? resources.value : localResources.value));

// ── 树 / 编辑区状态 ───────────────────────────────────────
const expandedItems = ref(new Set<string>());
const activeFilePath = ref<string | null>(null);
const fileContent = ref('');
const isDirty = ref(false);

// ── 新建文件 ──────────────────────────────────────────────
const newFileOpen = ref(false);
const newFilePath = ref('');
const newFileInputRef = ref();
const zipInputRef = ref();

const openNewFile = () => {
    newFileOpen.value = true;
    newFilePath.value = '';
    nextTick(() => newFileInputRef.value?.focus());
};

const closeNewFile = () => {
    newFileOpen.value = false;
    newFilePath.value = '';
};

const resetAll = (form: AutoFormData) => {
    skill.value = null;
    formStatus.value = 'draft';
    createdSkillId.value = null;
    Object.assign(form, { code: '', description: '', allowedTools: '', instructions: '' });
    resources.value = [];
    localResources.value = [];
    resourcesLoaded.value = false;
    expandedItems.value = new Set();
    activeFilePath.value = null;
    fileContent.value = '';
    isDirty.value = false;
    activeTab.value = 'basic';
    closeNewFile();
};

// 资源 Tab 首次激活时懒加载（编辑态）
watch(activeTab, (tab) => {
    if (tab !== 'resources') return;
    if (isEdit.value && !resourcesLoaded.value) {
        resourcesLoaded.value = true;
        loadResources();
    }
});

const loadResources = async () => {
    if (!props.skillId) return;
    try {
        resources.value = (await pluginApi.listResources.request({ id: props.skillId })) || [];
    } catch {
        resources.value = [];
    }
};

// ── 文件选择 / 编辑 / 保存 ─────────────────────────────────
const handleNodeClick = (data: TreeNode) => {
    if (data.isDir) return;
    activeFilePath.value = data.path;
    isDirty.value = false;
    const all = isEdit.value ? resources.value : localResources.value;
    const res = all.find((r) => r.path === data.path);
    fileContent.value = res ? ('content' in res ? res.content || '' : '') : '';
};

// 用户编辑：编辑态标 dirty 等手动保存；新建态实时写回本地暂存（程序赋值不走此函数）
const handleContentChange = (v: string | null | undefined) => {
    const val = v ?? '';
    fileContent.value = val;
    if (isEdit.value) {
        isDirty.value = true;
    } else if (activeFilePath.value) {
        const idx = localResources.value.findIndex((r) => r.path === activeFilePath.value);
        if (idx >= 0) localResources.value[idx].content = val;
    }
};

const handleSaveFileContent = async () => {
    if (!props.skillId || !activeFilePath.value || !isDirty.value) return;
    try {
        await pluginApi.upsertResource.request({
            id: props.skillId,
            path: activeFilePath.value,
            content: fileContent.value,
            size: new Blob([fileContent.value]).size,
        });
        isDirty.value = false;
        Msg.success('common.saveSuccess');
    } catch {
        // 失败原因请求层已 toast，保持 isDirty 以便重试
    }
};

// ── 新建文件 ──────────────────────────────────────────────
const handleCreateFile = async () => {
    const path = newFilePath.value.trim();
    if (!path) return;
    const all = isEdit.value ? resources.value : localResources.value;
    if (all.some((r) => r.path === path)) {
        Msg.error('ai.integration.resourceExists');
        return;
    }
    try {
        if (isEdit.value && props.skillId) {
            await pluginApi.upsertResource.request({ id: props.skillId, path, content: '', size: 0 });
            await loadResources();
        } else {
            localResources.value.push({ _tempId: `t_${Date.now()}`, path, content: '' });
        }
    } catch {
        return; // 落库失败不进树/不激活编辑器（请求层已 toast）
    }
    parentDirs(path).forEach((p) => expandedItems.value.add(p));
    activeFilePath.value = path;
    fileContent.value = '';
    isDirty.value = false;
    closeNewFile();
};

// ── 删除文件 ──────────────────────────────────────────────
const handleDeleteFile = async (path: string) => {
    try {
        await useI18nConfirm('ai.integration.deleteResourceConfirm');
        if (isEdit.value && props.skillId) {
            await pluginApi.deleteResource.request({ id: props.skillId, path });
            await loadResources();
        } else {
            localResources.value = localResources.value.filter((r) => r.path !== path);
        }
        if (activeFilePath.value === path) {
            activeFilePath.value = null;
            fileContent.value = '';
            isDirty.value = false;
        }
    } catch {
        // 确认取消或删除失败（请求层已 toast），静默退出
    }
};

// ── zip 导入导出（编辑态，导入为覆盖语义） ──────────────────
const triggerImport = () => zipInputRef.value?.click();

const handleImportFile = async (e: Event) => {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    input.value = '';
    if (!file || !props.skillId) return;
    const fd = new FormData();
    fd.append('file', file);
    loading.value = true;
    try {
        const imported = await pluginApi.importSkillZip.request(fd);
        if (imported) {
            skill.value = imported;
            formStatus.value = imported.status;
            Object.assign(internalForm.value, { code: imported.code, description: imported.description || '' });
            const ins = await pluginApi.getInstructions.request({ id: imported.id });
            internalForm.value.instructions = ins?.content || '';
        }
        await loadResources();
        resourcesLoaded.value = true;
        Msg.success('ai.integration.importSuccess');
    } catch {
        // 导入失败（请求层已 toast），仅复位 loading
    } finally {
        loading.value = false;
    }
};

const handleExport = () => {
    if (!props.skillId) return;
    downloadFile(`${config.baseApiUrl}/ai/plugin/skills/${props.skillId}/export?${joinClientParams()}`);
};

// ── 发布 / 转草稿 ─────────────────────────────────────────
const togglePublish = async (publish: boolean) => {
    if (!props.skillId) return;
    try {
        if (publish) await pluginApi.publishSkill.request({ id: props.skillId });
        else await pluginApi.unpublishSkill.request({ id: props.skillId });
        formStatus.value = publish ? 'published' : 'draft';
        Msg.success('common.operateSuccess');
    } catch {
        // 失败（请求层已 toast），状态保持不变
    }
};

// ── 提交（新建态创建后批量 upsert 暂存资源；资源部分失败保留 createdSkillId，重试不重复建技能） ──
const handleSubmit = async () => {
    // 校验失败内部已 toast（catch 吞掉 reject，避免落入下方资源落库失败提示分支）
    const valid = await useI18nFormValidate(drawerRef).catch(() => false);
    if (valid === false) return;
    const form = internalForm.value as { code: string; description: string; allowedTools: string; instructions: string };
    saving.value = true;
    try {
        let skillId = props.skillId || createdSkillId.value;
        if (!skillId) {
            const created = await pluginApi.createSkill.request({ ...form });
            createdSkillId.value = created.id;
            skillId = created.id;
        } else if (isEdit.value) {
            await pluginApi.updateSkill.request({ ...form, id: skillId });
        }
        if (!props.skillId) {
            await Promise.all(
                localResources.value.map((res) =>
                    pluginApi.upsertResource.request({
                        id: skillId!,
                        path: res.path,
                        content: res.content,
                        size: new Blob([res.content]).size,
                    })
                )
            );
        }
        Msg.success('common.saveSuccess');
        visible.value = false;
        emit('saved');
    } catch {
        // 资源批量落库部分失败：技能已创建（createdSkillId 已暂存），提示用户重试而非重复建技能
        if (createdSkillId.value && !props.skillId) {
            Msg.error('ai.integration.createPartialFailed');
        }
    } finally {
        saving.value = false;
    }
};
</script>

<style lang="scss" scoped>
.top-bar {
    display: flex;
    align-items: center;
    gap: 8px;
}

.zip-input {
    display: none;
}

.resource-panel {
    display: flex;
    gap: 12px;
    height: calc(100vh - 250px);

    .tree-side {
        width: 260px;
        flex-shrink: 0;
        display: flex;
        flex-direction: column;
        border: 1px solid var(--el-border-color-light);
        border-radius: 6px;
        overflow: hidden;

        .tree-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 6px 10px;
            border-bottom: 1px solid var(--el-border-color-lighter);
            background: var(--el-fill-color-light);

            .tree-title {
                display: flex;
                align-items: center;
                gap: 4px;
                font-size: 13px;
                font-weight: 600;
            }
        }

        .new-file-row {
            display: flex;
            align-items: center;
            gap: 2px;
            padding: 4px 8px;
            border-bottom: 1px solid var(--el-border-color-lighter);
        }

        .resource-tree {
            flex: 1;
            overflow: auto;

            :deep(.el-tree-node__content) {
                height: 26px;
            }

            .tree-node {
                display: flex;
                align-items: center;
                gap: 4px;
                flex: 1;
                min-width: 0;
                padding-right: 4px;

                .node-name {
                    flex: 1;
                    overflow: hidden;
                    text-overflow: ellipsis;
                    white-space: nowrap;
                    font-size: 12px;
                }

                .node-delete {
                    display: none;
                    color: var(--el-color-danger);
                    cursor: pointer;
                }

                &:hover .node-delete {
                    display: inline-flex;
                }
            }
        }

        .tree-empty {
            flex: 1;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 12px;
            color: var(--el-text-color-placeholder);
        }
    }

    .editor-side {
        flex: 1;
        min-width: 0;
        display: flex;
        flex-direction: column;
        border: 1px solid var(--el-border-color-light);
        border-radius: 6px;
        overflow: hidden;

        .editor-header {
            display: flex;
            align-items: center;
            gap: 8px;
            padding: 6px 10px;
            border-bottom: 1px solid var(--el-border-color-lighter);
            background: var(--el-fill-color-light);

            .file-path {
                flex: 1;
                font-size: 12px;
                font-weight: 600;
                overflow: hidden;
                text-overflow: ellipsis;
                white-space: nowrap;
            }
        }

        .editor-empty {
            flex: 1;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 12px;
            color: var(--el-text-color-placeholder);
        }
    }
}
</style>
