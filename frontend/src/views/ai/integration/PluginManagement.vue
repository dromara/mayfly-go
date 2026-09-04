<template>
    <div class="component-container card p-2!">
        <input ref="zipInputRef" type="file" accept=".zip" class="zip-input" @change="handleImportFile" />

        <!-- 统一插件列表：滚动分页 + 搜索（对齐 tokhub InfiniteCardList + useInfiniteScroll） -->
        <InfiniteCardList
            v-bind="bindCardList"
            :search-fields="searchFields"
            row-key="id"
            grid-class="plugin-grid"
            :empty-text="$t('ai.integration.emptyText')"
            :empty-desc="$t('ai.integration.emptyDesc')"
        >
            <template #default="{ item }">
                <PluginCard
                    :item="item"
                    @edit="openEdit(item)"
                    @test="openMcp(item)"
                    @tools="openMcp(item, true)"
                    @publish="togglePublish(item, true)"
                    @unpublish="togglePublish(item, false)"
                    @export="exportSkill(item)"
                    @delete="remove(item)"
                />
            </template>
            <template #actions>
                <el-button plain icon="upload" @click="zipInputRef?.click()">{{ $t('ai.integration.importZip') }}</el-button>
                <el-button type="primary" icon="plus" @click="pickerOpen = true">{{ $t('ai.integration.newPlugin') }}</el-button>
            </template>
        </InfiniteCardList>

        <!-- 类型选择 -->
        <PluginTypePicker v-model="pickerOpen" @pick="handlePick" />
        <!-- MCP 表单（autoDiscover：打开后自动连接测试并回显工具） -->
        <McpFormDrawer v-model="mcpDrawerOpen" :server="editingMcp" :auto-discover="mcpAutoDiscover" @saved="reload" />
        <!-- 技能编辑器 -->
        <SkillEditor v-model="skillEditorOpen" :skill-id="editingSkillId" @saved="reload" />
    </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import config from '@/common/config';
import { joinClientParams } from '@/common/request';
import { downloadFile } from '@/common/utils/file';
import { useInfiniteScroll } from '@/hooks/useInfiniteScroll';
import InfiniteCardList from '@/components/infinite-card-list/index.vue';
import { pluginApi, type PluginInstance, type PluginTypeInfo } from './api';
import PluginCard from './PluginCard.vue';
import PluginTypePicker from './PluginTypePicker.vue';
import McpFormDrawer from './McpFormDrawer.vue';
import SkillEditor from './SkillEditor.vue';
import type { PluginType } from './types';

const { t } = useI18n();

// ── 滚动分页数据源（t_ai_plugin_instance 单表分页接口，搜索词跨页保持） ─────────
const { reload, bindCardList } = useInfiniteScroll<PluginInstance, { keyword: string; type: string }>({
    fetcher: ({ page, pageSize, search }) =>
        pluginApi.listInstances.request({
            pageNum: page,
            pageSize,
            keyword: search.keyword || undefined,
            type: search.type || undefined,
        }),
    defaultSearch: { keyword: '', type: '' },
    pageSize: 12,
});

// ── 类型元数据（对齐 tokhub：前端仅维护展示映射，类型清单由后端注册表下发） ──
const typeMeta: Record<string, { labelKey?: string; fallback: string }> = {
    skill: { labelKey: 'ai.integration.skillPlugin', fallback: 'skill' },
    mcp: { labelKey: 'ai.integration.mcpPlugin', fallback: 'mcp' },
};
const typeLabel = (code: string) => {
    const meta = typeMeta[code];
    return meta?.labelKey ? t(meta.labelKey) : meta?.fallback || code;
};

const registeredTypes = ref<PluginTypeInfo[]>([]);

// computed 包裹：运行时切换语言时 placeholder/label 同步更新
const searchFields = computed(() => [
    { key: 'keyword', type: 'input' as const, placeholder: t('ai.integration.searchPlaceholder') },
    {
        key: 'type',
        type: 'select' as const,
        placeholder: t('ai.integration.allTypes'),
        options: registeredTypes.value.map((it) => ({ label: typeLabel(it.code), value: it.code })),
    },
]);

onMounted(async () => {
    await reload();
    try {
        // 类型过滤下拉数据源 → 后端类型注册表（新增类型零改前端核心）
        registeredTypes.value = (await pluginApi.listTypes.request()) || [];
    } catch {
        // 拉取失败（请求层已 toast），下拉为空不影响列表
    }
});

const pickerOpen = ref(false);
const mcpDrawerOpen = ref(false);
const mcpAutoDiscover = ref(false);
const editingMcp = ref<PluginInstance | null>(null);
const skillEditorOpen = ref(false);
const editingSkillId = ref<string | null>(null);
const zipInputRef = ref();

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
// 分页后本地列表可能不含目标项（如搜索前滚页外），编辑 MCP 前先拉实例详情（config 内联，无需再查专业表）
const openMcp = async (item: PluginInstance, autoDiscover = false) => {
    try {
        mcpAutoDiscover.value = autoDiscover;
        editingMcp.value = (await pluginApi.getInstance.request({ id: item.id })) || null;
        mcpDrawerOpen.value = true;
    } catch {
        // 详情拉取失败（请求层已 toast），不打开表单
    }
};

const openSkillEdit = (item: PluginInstance) => {
    // 技能编辑走技能接口：用填充的 skillId（实例 id 与技能 id 不同）
    editingSkillId.value = item.skillId || null;
    skillEditorOpen.value = true;
};

const openEdit = (item: PluginInstance) => (item.pluginType === 'skill' ? openSkillEdit(item) : openMcp(item));

// ── 技能发布 / 取消发布 / 导出 ─────────────────────────────
const togglePublish = async (item: PluginInstance, publish: boolean) => {
    try {
        if (publish) await pluginApi.publishSkill.request({ id: item.skillId! });
        else await pluginApi.unpublishSkill.request({ id: item.skillId! });
        Msg.success('common.operateSuccess');
        await reload();
    } catch {
        // 失败（请求层已 toast），列表状态保持不变
    }
};

const exportSkill = (item: PluginInstance) => {
    downloadFile(`${config.baseApiUrl}/ai/plugin/skills/${item.skillId}/export?${joinClientParams()}`);
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
        await reload();
    } catch {
        // 导入失败（请求层已 toast），静默退出
    }
};

// ── 删除 ──────────────────────────────────────────────────
const remove = async (item: PluginInstance) => {
    try {
        if (item.pluginType === 'skill') {
            // 技能删除走技能接口，后端联动删除引用实例
            await useI18nConfirm('ai.integration.deleteSkillConfirm', { name: item.name });
            await pluginApi.deleteSkill.request({ id: item.skillId! });
        } else {
            await useI18nConfirm('ai.integration.deleteInstanceConfirm', { name: item.name });
            await pluginApi.deleteInstance.request({ id: item.id });
        }
        Msg.success('common.deleteSuccess');
        await reload();
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
    overflow: hidden;

    .zip-input {
        display: none;
    }

    :deep(.plugin-grid) {
        padding-bottom: 4px;
    }
}
</style>
