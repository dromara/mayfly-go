<template>
    <!-- 普通模式：直接展示标签路径 -->
    <template v-if="!showPopover">
        <el-row v-for="(path, idx) in codePaths?.slice(0, 1)" :key="idx">
            <TagPathItem :path="path" :size="size" />

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
import { computed, onMounted, ref, watch, type PropType } from 'vue';
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
    // 兼容["default/test1/test2/"] 与 [{id: 1, codePath: "default/test1/test2/"}]
    path: {
        type: [String, Array] as PropType<string | string[] | Record<string, unknown>[]>,
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

const popoverWidth = computed(() => {
    return props.size === 'small' ? 400 : 500;
});

const handlePopoverShow = () => {
    if (props.showPopover) {
        loadPopoverTagInfo();
    }
};

const paths = computed(() => {
    if (Array.isArray(codePath.value)) {
        const ps = [];
        // 兼容["default/test1/test2/"] 与 [{id: 1, codePath: "default/test1/test2/"}]
        for (let p of codePath.value as (string | { codePath: string })[]) {
            if (typeof p === 'string') {
                ps.push(p);
            } else {
                ps.push(p.codePath);
            }
        }
        return ps;
    }

    return [codePath.value];
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
        codePath.value = tagInfos as unknown as Record<string, unknown>[];
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
            codePath.value = tagInfos as unknown as Record<string, unknown>[];
        }
    }

    if (!paths.value) {
        return;
    }

    popoverTagInfos.value = (await getAllCodePaths(paths.value as string[])) || {};
    codePaths.value = paths.value.filter((p): p is string => typeof p === 'string' && p !== undefined).map((p: string) => parseTagPathWithInfo(p, popoverTagInfos.value));
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
.tag-path-row {
    padding-bottom: 8px;
    border-bottom: 1px solid var(--el-border-color-lighter);

    &:last-child {
        padding-bottom: 0;
        border-bottom: none;
    }
}
</style>
