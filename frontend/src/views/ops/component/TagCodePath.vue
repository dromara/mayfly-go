<template>
    <el-row v-for="(path, idx) in codePaths?.slice(0, 1)" :key="idx">
        <span v-for="item in path" :key="item.code">
            <SvgIcon
                :name="EnumValue.getEnumByValue(TagResourceTypeEnum, item.type)?.extra.icon"
                :color="EnumValue.getEnumByValue(TagResourceTypeEnum, item.type)?.extra.iconColor"
                class="mr-0.5"
            />
            <span> {{ item.name ? item.name : item.code }}</span>
            <SvgIcon v-if="!item.isEnd" class="mr-1 ml-1" name="arrow-right" />
        </span>

        <!-- 展示剩余的标签信息 -->
        <el-popover :show-after="300" v-if="paths.length > 1 && idx == 0" placement="bottom" width="500" trigger="hover">
            <template #reference>
                <SvgIcon class="mt-1 ml-1" color="var(--el-color-primary)" name="MoreFilled" />
            </template>

            <el-row v-for="i in paths.slice(1)" :key="i">
                <span v-for="item in parseTagPath(i)" :key="item.code">
                    <SvgIcon
                        :name="EnumValue.getEnumByValue(TagResourceTypeEnum, item.type)?.extra.icon"
                        :color="EnumValue.getEnumByValue(TagResourceTypeEnum, item.type)?.extra.iconColor"
                        class="mr-0.5"
                    />
                    <span> {{ item.name ? item.name : item.code }}</span>
                    <SvgIcon v-if="!item.isEnd" class="mr-1 ml-1" name="arrow-right" />
                </span>
            </el-row>
        </el-popover>
    </el-row>
</template>

<script lang="ts" setup>
import { TagResourceTypeEnum } from '@/common/commonEnum';
import EnumValue from '@/common/Enum';
import { computed, onMounted, ref, watch } from 'vue';
import { getAllTagInfoByCodePaths } from './tag';

const props = defineProps({
    path: {
        type: [String, Array<string>, Array<Object>],
    },
    tagInfos: {
        type: Object, // key: code , value: code info
    },
});

const codePaths: any = ref([]);
let allTagInfos: any = {};

const paths = computed(() => {
    if (Array.isArray(props.path)) {
        const ps = [];
        // 兼容["default/test1/test2/"] 与 [{id: 1, codePath: "default/test1/test2/"}]
        for (let p of props.path as any) {
            if (typeof p === 'string') {
                ps.push(p);
            } else {
                ps.push(p.codePath);
            }
        }
        return ps;
    }

    return [props.path];
});

onMounted(() => {
    setCodePaths();
});

watch(
    () => props.path,
    () => {
        setCodePaths();
    }
);

const setCodePaths = async () => {
    if (!paths.value) {
        return;
    }

    if (!props.tagInfos || Object.keys(props.tagInfos).length == 0) {
        const tagInfos = await getAllTagInfoByCodePaths(paths.value as any);
        allTagInfos = tagInfos;
    } else {
        allTagInfos = props.tagInfos;
    }

    codePaths.value = paths.value.map((p) => parseTagPath(p));
};

const parseTagPath = (tagPath: string = '') => {
    if (!tagPath) {
        return [];
    }
    const res = [] as any;
    const codes = tagPath.split('/');
    for (let code of codes) {
        const typeAndCode = code.split('|');

        let tagInfo;
        if (typeAndCode.length == 1) {
            const tagCode = typeAndCode[0];
            if (!tagCode) {
                continue;
            }

            tagInfo = {
                type: TagResourceTypeEnum.Tag.value,
                code: typeAndCode[0],
            };
            res.push(tagInfo);
            continue;
        } else {
            tagInfo = {
                type: typeAndCode[0],
                code: typeAndCode[1],
                name: '',
            };
        }

        const ti = getTagInfo(tagInfo.type, tagInfo.code);
        if (ti) {
            tagInfo.name = ti.name;
        }

        res.push(tagInfo);
    }

    res[res.length - 1].isEnd = true;
    return res;
};

const getTagInfo = (type: any, code: string) => {
    if (type == TagResourceTypeEnum.Tag.value) {
        return {};
    }

    if (allTagInfos && Object.keys(allTagInfos).length > 0) {
        const key = `${type}|${code}`;
        if (allTagInfos[key]) {
            return allTagInfos[key];
        }
    }

    return {};
};
</script>
<style lang="scss"></style>
