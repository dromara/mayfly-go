<template>
    <div v-if="paths">
        <el-row v-for="(path, idx) in paths?.slice(0, 1)" :key="idx">
            <span v-for="item in parseTagPath(path)" :key="item.code">
                <SvgIcon
                    :name="EnumValue.getEnumByValue(TagResourceTypeEnum, item.type)?.extra.icon"
                    :color="EnumValue.getEnumByValue(TagResourceTypeEnum, item.type)?.extra.iconColor"
                    class="mr2"
                />
                <span> {{ item.code }}</span>
                <SvgIcon v-if="!item.isEnd" class="mr5 ml5" name="arrow-right" />
            </span>

            <!-- 展示剩余的标签信息 -->
            <el-popover :show-after="300" v-if="paths.length > 1 && idx == 0" placement="bottom" width="500" trigger="hover">
                <template #reference>
                    <SvgIcon class="mt5 ml5" color="var(--el-color-primary)" name="MoreFilled" />
                </template>

                <el-row v-for="i in paths.slice(1)" :key="i">
                    <span v-for="item in parseTagPath(i)" :key="item.code">
                        <SvgIcon
                            :name="EnumValue.getEnumByValue(TagResourceTypeEnum, item.type)?.extra.icon"
                            :color="EnumValue.getEnumByValue(TagResourceTypeEnum, item.type)?.extra.iconColor"
                            class="mr2"
                        />
                        <span> {{ item.code }}</span>
                        <SvgIcon v-if="!item.isEnd" class="mr5 ml5" name="arrow-right" />
                    </span>
                </el-row>
            </el-popover>
        </el-row>
    </div>
</template>

<script lang="ts" setup>
import { TagResourceTypeEnum } from '@/common/commonEnum';
import EnumValue from '@/common/Enum';
import { computed } from 'vue';

const props = defineProps({
    path: {
        type: [String, Array<string>],
    },
});

const paths = computed(() => {
    if (Array.isArray(props.path)) {
        return props.path;
    }

    return [props.path];
});

const parseTagPath = (tagPath: string = '') => {
    if (!tagPath) {
        return [];
    }
    const res = [] as any;
    const codes = tagPath.split('/');
    for (let code of codes) {
        const typeAndCode = code.split('|');

        if (typeAndCode.length == 1) {
            const tagCode = typeAndCode[0];
            if (!tagCode) {
                continue;
            }

            res.push({
                type: TagResourceTypeEnum.Tag.value,
                code: typeAndCode[0],
            });
            continue;
        }

        res.push({
            type: typeAndCode[0],
            code: typeAndCode[1],
        });
    }

    res[res.length - 1].isEnd = true;
    return res;
};
</script>
<style lang="scss"></style>
