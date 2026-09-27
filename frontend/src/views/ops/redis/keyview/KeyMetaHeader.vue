<template>
    <div class="key-meta card p-3!">
        <!-- 第一行：身份。key 名可截断收缩，其余固定宽，保证永远一行 -->
        <div class="flex min-w-0 items-center gap-2">
            <span class="type-badge shrink-0" :style="{ backgroundColor: viewColor }">{{ typeText }}</span>

            <span class="key-name min-w-0 flex-1 truncate font-mono text-sm font-semibold" :title="keyName">{{ keyName }}</span>

            <el-tooltip :content="$t('common.copy')" placement="top">
                <el-button class="shrink-0" link icon="CopyDocument" @click="onCopyKey" />
            </el-tooltip>

            <el-select
                v-if="viewOptions.length > 1"
                :model-value="view"
                size="small"
                class="view-select shrink-0"
                :placeholder="$t('redis.switchView')"
                @change="(val: string) => emit('switchView', val)"
            >
                <template #prefix>
                    <SvgIcon name="Grid" />
                </template>
                <el-option v-for="item in viewOptions" :key="item.view" :label="$t(item.label)" :value="item.view" />
            </el-select>
        </div>

        <!-- 第二行：统计靠左、工具靠右，同一行内不再折行 -->
        <div class="mt-2 flex min-w-0 flex-wrap items-center gap-y-1 gap-1">
            <span v-for="chip in chips" :key="chip.label" class="meta-chip shrink-0" :class="{ 'meta-chip--warn': chip.warn }" :title="$t(chip.label)">
                <SvgIcon :name="chip.icon" :size="12" />
                <b>{{ chip.value }}</b>
            </span>

            <div class="tool-group ml-auto flex shrink-0 items-center">
                <el-tooltip :content="$t('common.refresh')" placement="top">
                    <el-button class="tool-btn" text :aria-label="$t('common.refresh')" @click="emit('refresh')">
                        <SvgIcon name="refresh" :size="15" />
                    </el-button>
                </el-tooltip>
                <el-tooltip :content="$t('redis.renameKey')" placement="top">
                    <el-button v-auth="PERM_DATA_SAVE" class="tool-btn" text :disabled="missing" :aria-label="$t('redis.renameKey')" @click="emit('rename')">
                        <SvgIcon name="EditPen" :size="15" />
                    </el-button>
                </el-tooltip>
                <el-tooltip :content="$t('redis.ttlKey')" placement="top">
                    <el-button v-auth="PERM_DATA_SAVE" class="tool-btn" text :disabled="missing" :aria-label="$t('redis.ttlKey')" @click="emit('ttl')">
                        <SvgIcon name="Timer" :size="15" />
                    </el-button>
                </el-tooltip>
                <el-tooltip :content="$t('redis.copyKey')" placement="top">
                    <el-button v-auth="PERM_DATA_SAVE" class="tool-btn" text :disabled="missing" :aria-label="$t('redis.copyKey')" @click="emit('copy')">
                        <SvgIcon name="DocumentAdd" :size="15" />
                    </el-button>
                </el-tooltip>
                <el-tooltip :content="$t('common.delete')" placement="top">
                    <el-button
                        v-auth="PERM_DATA_DEL"
                        class="tool-btn tool-btn--danger"
                        text
                        :disabled="missing"
                        :aria-label="$t('common.delete')"
                        @click="emit('del')"
                    >
                        <SvgIcon name="delete" :size="15" />
                    </el-button>
                </el-tooltip>
            </div>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { formatByteSize } from '@/common/utils/format';
import { copyToClipboard } from '@/common/utils/string';
import { computed, onUnmounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { PERM_DATA_DEL, PERM_DATA_SAVE } from './permission';
import { LAYOUT_VALUE } from './descriptor';
import { viewAppearance } from './appearance';
import type { RedisKeyMeta, RedisViewDescriptor } from '../types';

const props = defineProps<{
    keyName: string;
    meta?: RedisKeyMeta;
    view?: string;
    /** 当前视角的描述符，提供类型主色 */
    descriptor?: RedisViewDescriptor;
    /** 成员总数：取成员分页回包的总数，与列表同源；字段自然过期时它比元信息更实时 */
    size?: number;
    /** 同一原生类型下可切换的视角（string 可看作文本/位图/HLL，zset 可看作有序集合/GEO） */
    viewOptions: RedisViewDescriptor[];
    /** key 已不存在（被删除或过期）：类型徽标不能用视角名冒充类型 */
    missing?: boolean;
}>();

const emit = defineEmits<{
    refresh: [];
    rename: [];
    copy: [];
    ttl: [];
    del: [];
    switchView: [view: string];
}>();

const { t } = useI18n();

// TTL 是读取时刻的快照，页面停留期间会持续变小，因此本地每秒递减而不是等接口再返回
const elapsed = ref(0);
const timer = window.setInterval(() => (elapsed.value += 1), 1000);
onUnmounted(() => window.clearInterval(timer));

const ttlLeft = computed(() => {
    const ttl = props.meta?.ttl ?? -1;
    return ttl > 0 ? ttl - elapsed.value : ttl;
});

// 每次重新取到元信息都要归零本地计数：meta.ttl 已是服务端最新值，
// 继续叠加旧 elapsed 会把读数算小甚至算成负数，落到「永久」分支
watch(
    () => props.meta,
    () => {
        elapsed.value = 0;
    }
);

const isExpiring = computed(() => ttlLeft.value > 0 && ttlLeft.value < 60);

const ttlText = computed(() => {
    // key 已不存在时「永久」是误信息：它没有任何过期策略可读，给一个中性占位
    if (props.missing) {
        return '-';
    }
    const left = ttlLeft.value;
    if (left < 0) {
        return t('redis.permanent');
    }
    if (left >= 86400) {
        return `${Math.floor(left / 86400)}d ${formatLeft(left % 86400)}`;
    }
    return formatLeft(left);
});

function formatLeft(seconds: number): string {
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = seconds % 60;
    return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${pad(m)}:${pad(s)}`;
}

function pad(value: number): string {
    return String(value).padStart(2, '0');
}

const viewColor = computed(() => viewAppearance(props.descriptor?.view).color);

/**
 * 新增态下 key 还不存在（后端返回类型 none），徽标改用用户刚选的视角名，
 * 否则会显示成 "none" 这种存储层细节
 */
const typeText = computed(() => {
    if (props.meta?.exists) {
        return props.meta.type;
    }
    if (props.missing) {
        return t('redis.keyMissingBadge');
    }
    // 新增态展示视角的中文名（如「集合」），而不是 hash 这种存储层标识
    return props.descriptor?.label ? t(props.descriptor.label) : props.view || '-';
});

// 单值视角的「大小」就是正文本身，编辑器栏已有实时读数，这里再摆一颗只会重复
const isValueLayout = computed(() => props.descriptor?.layout === LAYOUT_VALUE);

// 统计读数集中声明：图标相同语义在全模块保持一致，新增一项只改这里
const chips = computed(() => {
    const items = [
        { icon: 'Clock', label: 'redis.ttl', value: ttlText.value, warn: isExpiring.value },
        { icon: 'Coin', label: 'redis.encoding', value: props.meta?.encoding || '-', warn: false },
        { icon: 'Odometer', label: 'redis.memuse', value: formatByteSize(props.meta?.memuse ?? 0), warn: false },
    ];
    if (!isValueLayout.value) {
        items.push({ icon: 'Files', label: 'redis.memberSize', value: String(props.size ?? props.meta?.size ?? 0), warn: false });
    }
    return items;
});

const onCopyKey = async () => {
    // 复制成功提示由 copyToClipboard 统一发出，这里再叠一条会出现两个一模一样的 toast
    await copyToClipboard(props.keyName);
};
</script>

<style lang="scss" scoped>
@use './toolbar.scss' as *;

.type-badge {
    padding: 2px 9px;
    border-radius: 7px;
    color: #fff;
    font-size: 11.5px;
    font-weight: 600;
    letter-spacing: 0.2px;
    text-transform: capitalize;
    line-height: 16px;
}

.view-select {
    width: 150px;
}

.meta-chip {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 2px 8px;
    border-radius: 999px;
    background-color: var(--el-fill-color-light);
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 16px;

    b {
        color: var(--el-text-color-primary);
        font-weight: 600;
    }
}

.meta-chip--warn {
    background-color: var(--el-color-warning-light-9);

    b,
    svg {
        color: var(--el-color-warning-dark-2);
    }
}

.key-meta {
    border: 1px solid var(--el-border-color-lighter);
    border-radius: $redis-radius;
}
</style>
