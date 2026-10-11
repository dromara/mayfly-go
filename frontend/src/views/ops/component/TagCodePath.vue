<template>
    <!-- 普通模式：直接展示标签路径 -->
    <template v-if="!showPopover">
        <el-row v-for="(path, idx) in codePaths?.slice(0, 1)" :key="idx" :ref="bindFoldRow" :class="{ 'tag-path-autofold': autoFold }">
            <!-- 折叠由容器宽度驱动：放得下不折，放不下才把前置段收进「…」（悬浮可见段名） -->
            <template v-if="foldedOf(path).length">
                <el-tooltip :show-after="300" placement="top" :content="foldedTextOf(path)">
                    <span class="tag-path-fold-marker" :class="foldTextClass">…</span>
                </el-tooltip>
                <SvgIcon color="var(--el-text-color-placeholder)" :size="iconSize" :class="arrowMarginClass" name="arrow-right" />
            </template>
            <TagPathItem :path="visibleOf(path)" :size="size" />

            <!-- 展示剩余的标签信息 -->
            <el-popover :show-after="300" v-if="paths.length > 1 && idx == 0" placement="bottom" :width="popoverWidth" trigger="hover">
                <template #reference>
                    <SvgIcon :size="iconSize" :class="moreIconMarginClass" color="var(--el-color-primary)" name="MoreFilled" />
                </template>

                <el-row v-for="(opath, oi) in codePaths.slice(1)" :key="oi" class="mb-2 tag-path-row">
                    <TagPathItem :path="opath" :size="size" />
                </el-row>
            </el-popover>
        </el-row>
    </template>

    <!-- popover模式：默认显示按钮，悬浮时显示所有详细信息 -->
    <el-popover v-else :show-after="300" placement="bottom" :width="popoverWidth" trigger="hover" @show="handlePopoverShow">
        <template #reference>
            <SvgIcon :size="iconSize" color="var(--el-color-primary)" name="location" />
        </template>

        <div v-loading="isPopoverVisible && !codePaths.length">
            <el-row v-for="(path, idx) in codePaths" :key="idx" class="mb-2 tag-path-row">
                <TagPathItem :path="path" :size="size" />
            </el-row>
        </div>
    </el-popover>
</template>

<script lang="ts" setup>
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { tagApi } from '@/views/ops/tag/api';
import type { TagTree } from '@/views/ops/tag/types';
import { computed, nextTick, onMounted, ref, watch, type PropType } from 'vue';
import TagPathItem from './TagPathItem.vue';

interface TagPathInfo {
    type: number | string;
    code: string;
    codePath: string;
    name: string;
    isEnd?: boolean;
}

type TagInfoMap = Record<string, TagTree>;

const props = defineProps({
    // 兼容["default/test1/test2/"] 与 [{id: 1, codePath: "default/test1/test2/"}]（后者为标签信息列表）
    path: {
        type: [String, Array] as PropType<string | string[] | TagTree[]>,
    },
    // code，可直接设置该值展示路径信息
    code: {
        type: String,
    },
    // 尺寸: normal(默认) | small
    size: {
        type: String,
        default: 'small',
    },
    // 是否使用popover模式: 默认false，true时默认只显示icon，悬浮时才显示详细信息并请求接口
    showPopover: {
        type: Boolean,
        default: false,
    },
    // 宽度自适应折叠：放得下全段展示，放不下才把前置段收进「…」（悬浮可见段名，至少保留末段）；仅普通模式生效
    autoFold: {
        type: Boolean,
        default: false,
    },
});

const codePath = ref(props.path);
const codePaths = ref<TagPathInfo[][]>([]);
let allTagInfos: TagInfoMap = {};
const popoverTagInfos = ref<TagInfoMap>({});
const isPopoverVisible = ref(false);

const iconSize = computed(() => {
    return props.size === 'small' ? 14 : 15;
});

const moreIconMarginClass = computed(() => {
    return props.size === 'small' ? 'mt-2 ml-1' : 'mt-1 ml-1';
});

const arrowMarginClass = computed(() => {
    return props.size === 'small' ? 'mx-0.5' : 'mx-1';
});

const foldTextClass = computed(() => {
    return props.size === 'small' ? 'text-sm' : '';
});

const popoverWidth = computed(() => {
    return props.size === 'small' ? 400 : 500;
});

const handlePopoverShow = () => {
    if (props.showPopover) {
        loadPopoverTagInfo();
    }
};

const paths = computed<string[]>(() => {
    const value = codePath.value;
    // 兼容["default/test1/test2/"] 与 [{id: 1, codePath: "default/test1/test2/"}]
    if (Array.isArray(value)) {
        return value.map((p) => (typeof p === 'string' ? p : p.codePath));
    }

    return value ? [value] : [];
});

onMounted(() => {
    codePath.value = props.path;
    if (!props.showPopover) {
        setCodePaths();
    }
});

watch(
    () => props.path,
    () => {
        codePath.value = props.path;
        if (!props.showPopover) {
            setCodePaths();
        }
    }
);

watch(
    () => props.code,
    () => {
        if (!props.code) {
            clear();
            return;
        }
        if (!props.showPopover) {
            setCodePaths();
        }
    }
);

const setCodePaths = async () => {
    if (props.code) {
        const tagInfos = await tagApi.listByQuery.request({ codes: props.code });
        if (tagInfos.length == 0) {
            clear();
            return;
        }
        codePath.value = tagInfos;
    }

    if (!paths.value) {
        clear();
        return;
    }

    allTagInfos = (await getAllCodePaths(paths.value as string[])) || {};
    codePaths.value = paths.value.filter((p): p is string => typeof p === 'string' && p !== undefined).map((p: string) => parseTagPath(p));
};

// popover模式下，悬浮时加载tag信息
const loadPopoverTagInfo = async () => {
    // if (isPopoverVisible.value) return; // 已经加载过

    isPopoverVisible.value = true;

    if (props.code) {
        const tagInfos = await tagApi.listByQuery.request({ codes: props.code });
        if (tagInfos.length > 0) {
            codePath.value = tagInfos;
        }
    }

    if (!paths.value) {
        return;
    }

    popoverTagInfos.value = (await getAllCodePaths(paths.value as string[])) || {};
    codePaths.value = paths.value
        .filter((p): p is string => typeof p === 'string' && p !== undefined)
        .map((p: string) => parseTagPathWithInfo(p, popoverTagInfos.value));
};

const clear = () => {
    codePath.value = '';
    codePaths.value = [];
};

const parseTagPath = (tagPath: string = '') => {
    return parseTagPathWithInfo(tagPath, allTagInfos);
};

const parseTagPathWithInfo = (tagPath: string = '', tagInfos: TagInfoMap) => {
    if (!tagPath) {
        return [];
    }
    const res: TagPathInfo[] = [];
    let codePath = '';
    const codes = tagPath.split('/');
    for (let code of codes) {
        codePath += code + '/';
        const typeAndCode = code.split('|');

        let tagInfo: TagPathInfo;
        if (typeAndCode.length == 1) {
            const tagCode = typeAndCode[0];
            if (!tagCode) {
                continue;
            }

            tagInfo = {
                type: TagResourceTypeEnum.Tag.value,
                code: typeAndCode[0],
                codePath: codePath,
                name: '',
            };
        } else {
            tagInfo = {
                type: typeAndCode[0],
                code: typeAndCode[1],
                codePath: codePath,
                name: '',
            };
        }

        const ti = tagInfos[codePath];
        if (ti) {
            tagInfo.name = ti.name;
        }

        res.push(tagInfo);
    }

    res[res.length - 1].isEnd = true;
    return res;
};

/** 当前折叠隐藏的前置段数（autoFold 关闭时恒为 0） */
const foldedCount = ref(0);
let foldRowEl: HTMLElement | null = null;
let foldObserver: ResizeObserver | null = null;
let settlingFold = false;
let foldAgain = false;

/** 当前隐藏段数：由容器宽度收敛而来，任何情况下至少保留末段 */
const hiddenCountOf = (path: TagPathInfo[]) => Math.min(foldedCount.value, Math.max(path.length - 1, 0));

/** 折叠出的前置段（悬浮提示用它回填被折叠的上下文） */
const foldedOf = (path: TagPathInfo[]) => path.slice(0, hiddenCountOf(path));

/** 行内可见段 */
const visibleOf = (path: TagPathInfo[]) => path.slice(hiddenCountOf(path));

/** 折叠段悬浮提示：按面包屑顺序拼接段名 */
const foldedTextOf = (path: TagPathInfo[]) =>
    foldedOf(path)
        .map((segment) => segment.name || segment.code)
        .join(' > ');

/** 行元素挂载/卸载：顺带重建尺寸观察，元素出现即触发一轮收敛 */
const bindFoldRow = (el: unknown) => {
    const target = ((el as { $el?: HTMLElement } | null)?.$el ?? (el as HTMLElement | null)) ?? null;
    if (target === foldRowEl) {
        return;
    }
    foldRowEl = target;
    foldObserver?.disconnect();
    foldObserver = null;
    if (target && props.autoFold && typeof ResizeObserver !== 'undefined') {
        foldObserver = new ResizeObserver(() => void settleFold());
        foldObserver.observe(target);
    }
    if (target) {
        void settleFold();
    }
};

/** 收敛调度：整个过程在微任务内完成，渲染不会出现「先折后展」的中间帧 */
const settleFold = async () => {
    if (!props.autoFold || props.showPopover) {
        return;
    }
    if (settlingFold) {
        foldAgain = true;
        return;
    }
    settlingFold = true;
    try {
        do {
            foldAgain = false;
            await convergeFold();
        } while (foldAgain);
    } finally {
        settlingFold = false;
    }
};

/**
 * 放得下判定：行自身不得横向溢出；宿主是选择框时（DbSelectTree 场景）库名标签
 * 也不得被挤压——库名由 EP 绝对定位的 placeholder 承载（脱离文档流、静默 ellipsis），
 * 被挤窄不会冒泡成任何祖先的 scrollWidth 溢出，必须单独测量，否则会「路径全展示、库名只剩一个字」
 */
const fitsRow = (el: HTMLElement) => {
    if (el.scrollWidth > el.clientWidth) {
        return false;
    }
    const label = el.closest('.el-select__wrapper')?.querySelector<HTMLElement>('.el-select__placeholder');
    // is-transparent：无值时的占位文案截断无妨；+1 容忍亚像素取整
    return !label || label.classList.contains('is-transparent') || label.scrollWidth <= label.clientWidth + 1;
};

/** 单轮收敛：先折到放得下，再回展到刚好放得下（保证折叠数最小），极限处保留末段 */
const convergeFold = async () => {
    await nextTick();
    const el = foldRowEl;
    const segments = codePaths.value[0] ?? [];
    if (!el || segments.length < 2) {
        return;
    }
    const maxHidden = segments.length - 1;
    for (let guard = 0; guard <= maxHidden + 2; guard++) {
        if (!fitsRow(el)) {
            if (foldedCount.value >= maxHidden) {
                // 只剩末段仍放不下：由外层省略号兜底，不再折
                return;
            }
            foldedCount.value += 1;
        } else if (foldedCount.value > 0) {
            foldedCount.value -= 1;
            await nextTick();
            if (!fitsRow(el)) {
                // 回展一段就放不下了：退回刚放得下的折数
                foldedCount.value += 1;
                return;
            }
            continue;
        } else {
            return;
        }
        await nextTick();
    }
};

// 内容变化（选中新库/段名异步到货）都从全展开重新收敛，折叠数始终取最小可用值
watch(
    () => codePaths.value,
    () => {
        if (props.showPopover || !props.autoFold) {
            return;
        }
        foldedCount.value = 0;
        void settleFold();
    }
);

/**
 * 获取所有标签路径信息，如 default/test1/1|machinecode -> ['default/', 'default/test1/', 'default/test1/1|machinecode']
 * @param codePath 标签路径
 * @returns 所有层级路径数组
 */
function getAllCodePath(codePath: string) {
    if (!codePath) return [];

    const parts = codePath.split('/');
    const result: string[] = [];
    let currentPath = '';

    for (const part of parts) {
        if (!part) {
            continue;
        }
        currentPath += part + '/';
        result.push(currentPath);
    }

    return result;
}

/**
 * 完善标签路径信息
 * @param codePaths 标签路径
 * @returns
 */
async function getAllCodePaths(codePaths: string[]) {
    if (!codePaths) return;
    const allCodePaths: string[] = [];

    // 收集所有层级路径并去重
    for (const codePath of codePaths) {
        allCodePaths.push(...getAllCodePath(codePath));
    }

    const codepath2CodeInfo: TagInfoMap = {};
    // 去重
    const uniqueCodePaths = [...new Set(allCodePaths)];
    if (uniqueCodePaths.length == 0) {
        return codepath2CodeInfo;
    }

    const tagInfos = await tagApi.listByQuery.request({ tagPaths: uniqueCodePaths });

    for (const tagInfo of tagInfos) {
        codepath2CodeInfo[tagInfo.codePath] = tagInfo;
    }

    return codepath2CodeInfo;
}
</script>

<style lang="scss" scoped>
.tag-path-fold-marker {
    cursor: default;
}

.tag-path-row {
    padding-bottom: 8px;
    border-bottom: 1px solid var(--el-border-color-lighter);

    &:last-child {
        padding-bottom: 0;
        border-bottom: none;
    }
}
</style>

<style lang="scss">
/* autoFold 的行必须 nowrap：wrap 会把横向溢出藏成折行，宽度测量就失真；
   行内子项不允许收缩，溢出必须如实体现为 scrollWidth > clientWidth。
   子项模板在 TagPathItem 里，不在本组件 scoped 作用域内，故用非 scoped 块 */
.el-row.tag-path-autofold {
    flex-wrap: nowrap;
    white-space: nowrap;

    > * {
        flex-shrink: 0;
    }
}
</style>
