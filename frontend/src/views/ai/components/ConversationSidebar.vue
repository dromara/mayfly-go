<template>
    <!-- 固定宽度 w-72（AgentChat 侧栏），避免与 chat 区 flex 平分 -->
    <Card class="conv-sidebar w-72 shrink-0 min-h-0">
        <CardContent class="p-3 relative flex flex-col h-full min-w-0 gap-3">
            <!-- 标题栏 + 新建按钮（AgentSidebar） -->
            <div class="shrink-0 flex items-center justify-between">
                <h3 class="text-sm font-semibold flex items-center gap-1.5 text-foreground/80">
                    <MessageSquareIcon class="size-3.5" />
                    {{ t('ai.assistant.sessions') }}
                </h3>
                <Button
                    size="icon-sm"
                    variant="ghost"
                    :title="t('ai.assistant.newSession')"
                    @click="$emit('create')"
                >
                    <PlusIcon class="size-4" />
                </Button>
            </div>

            <!-- 搜索 -->
            <Input v-model="searchText" :placeholder="t('ai.assistant.searchPlaceholder')" class="shrink-0" />

            <!-- 会话列表 -->
            <div ref="listRef" class="flex-1 min-h-0 overflow-y-auto -mx-1 px-1" @scroll="onListScroll">
                <Empty v-if="filteredItems.length === 0" class="py-10">
                    <EmptyMedia variant="icon">
                        <MessageSquareIcon />
                    </EmptyMedia>
                    <EmptyContent>
                        <EmptyDescription>{{ items.length === 0 ? t('ai.assistant.noSessions') : t('ai.assistant.searchNoMatch') }}</EmptyDescription>
                    </EmptyContent>
                </Empty>

                <div
                    v-for="item in filteredItems"
                    :key="item.key"
                    class="group flex items-center gap-1.5 px-2.5 py-2 rounded-lg text-sm cursor-pointer min-w-0 transition-colors mb-0.5"
                    :class="item.key === active ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-accent'"
                    @click="$emit('change', item)"
                    @dblclick.stop="startEdit(item)"
                >
                    <!-- 执行中指示器：替换会话图标为 primary 脉冲圆点（克制的呼吸感，不占额外空间）；
                         prefers-reduced-motion 下降级为静态圆点 -->
                    <span
                        v-if="item.running"
                        class="relative flex size-3.5 shrink-0 items-center justify-center"
                        :title="t('ai.chat.turnRunning')"
                    >
                        <span
                            class="absolute size-2 rounded-full bg-primary opacity-60 animate-ping motion-reduce:animate-none"
                            aria-hidden="true"
                        ></span>
                        <span class="relative size-2 rounded-full bg-primary"></span>
                    </span>
                    <MessageSquareIcon
                        v-else
                        class="size-3.5 shrink-0"
                        :class="item.key === active ? 'text-primary' : 'text-muted-foreground'"
                    />
                    <!-- 双击行内重命名（Enter/blur 提交，Esc 取消） -->
                    <input
                        v-if="editingKey === item.key"
                        :ref="setEditInputRef"
                        v-model="editTitle"
                        class="flex-1 min-w-0 bg-transparent border-none outline-none text-sm px-1"
                        @click.stop
                        @blur="commitEdit(item)"
                        @keydown.enter.prevent="commitEdit(item)"
                        @keydown.esc.prevent="cancelEdit"
                    >
                    <span v-else class="truncate flex-1" :title="`${item.label} · ${item.createTime}`">{{ item.label }}</span>
                    <span v-if="item.createTime && editingKey !== item.key" class="text-xs text-muted-foreground shrink-0">{{ item.createTime }}</span>
                    <!-- 删除确认：el-popconfirm 气泡（对齐项目删除确认惯例），@click.stop 阻断行切换 -->
                    <el-popconfirm
                        :title="t('ai.assistant.deleteConfirm', { name: item.label })"
                        :confirm-button-text="t('common.confirm')"
                        :cancel-button-text="t('common.cancel')"
                        width="240"
                        @confirm="$emit('delete', item)"
                    >
                        <template #reference>
                            <Button
                                size="icon-xs"
                                variant="ghost"
                                class="opacity-0 group-hover:opacity-100 shrink-0"
                                @click.stop
                            >
                                <Trash2Icon class="size-3" />
                            </Button>
                        </template>
                    </el-popconfirm>
                </div>
            </div>

            <!-- 回顶部 -->
            <Button
                v-if="showBackTop"
                size="icon-sm"
                variant="secondary"
                class="absolute bottom-3 left-1/2 -translate-x-1/2 z-10 rounded-full shadow"
                :title="t('ai.assistant.backTop')"
                @click="scrollToTop"
            >
                <ArrowUpIcon class="size-4" />
            </Button>
        </CardContent>
    </Card>
</template>

<script setup lang="ts">
/**
 * ConversationSidebar - 会话列表侧边栏
 * AgentSidebar：Card 壳 + 标题栏新建 + 行内 tailwind token（选中 bg-primary/10、
 * hover bg-accent）+ 双击行内重命名 + hover 删除按钮 + el-popconfirm 删除确认
 */
import { ArrowUpIcon, MessageSquareIcon, PlusIcon, Trash2Icon } from '@lucide/vue';
import { computed, nextTick, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Empty, EmptyContent, EmptyDescription, EmptyMedia } from '@/components/ui/empty';
import { Input } from '@/components/ui/input';

export interface ConversationSidebarItem {
    key: string;
    label: string;
    createTime?: string;
    updateTime?: string;
    /** 执行中（agent 正在生成回复）：显示脉冲圆点指示器 */
    running?: boolean;
}

const props = defineProps<{
    items: ConversationSidebarItem[];
    active?: string;
}>();

const emit = defineEmits<{
    (e: 'create'): void;
    (e: 'change', item: ConversationSidebarItem): void;
    (e: 'rename', item: ConversationSidebarItem, title: string): void;
    (e: 'delete', item: ConversationSidebarItem): void;
}>();

const { t } = useI18n();

// 搜索过滤（会话多时快速定位）
const searchText = ref('');
const filteredItems = computed(() => {
    const q = searchText.value.trim().toLowerCase();
    if (!q) return props.items;
    return props.items.filter((item) => item.label.toLowerCase().includes(q));
});

// 双击行内重命名
const editingKey = ref('');
const editTitle = ref('');
const editInputEl = ref<HTMLInputElement | null>(null);
const setEditInputRef = (el: unknown) => {
    editInputEl.value = (el as HTMLInputElement) ?? null;
};

const startEdit = (item: ConversationSidebarItem) => {
    editingKey.value = item.key;
    editTitle.value = item.label;
    nextTick(() => editInputEl.value?.focus());
};

const commitEdit = (item: ConversationSidebarItem) => {
    if (editingKey.value !== item.key) return;
    const title = editTitle.value.trim();
    editingKey.value = '';
    if (title && title !== item.label) {
        emit('rename', item, title);
    }
};

const cancelEdit = () => {
    editingKey.value = '';
};

// 回顶部
const showBackTop = ref(false);
const listRef = ref<HTMLElement | null>(null);

const onListScroll = (e: Event) => {
    showBackTop.value = (e.target as HTMLElement).scrollTop > 200;
};

const scrollToTop = () => {
    if (listRef.value) listRef.value.scrollTop = 0;
};
</script>

<style scoped>
.conv-sidebar {
    width: 260px;
    height: 100%;
}
</style>
