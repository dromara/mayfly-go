<template>
    <div v-if="props.tags">
        <el-row v-for="(tag, idx) in props.tags?.slice(0, 1)" :key="idx">
            <TagInfo :tag-path="tag.codePath" />
            <span class="ml-0.5">{{ tag.codePath }}</span>

            <!-- 展示剩余的标签信息 -->
            <el-popover :show-after="300" v-if="props.tags.length > 1 && idx == 0" placement="top-start" width="230" trigger="hover">
                <template #reference>
                    <SvgIcon class="mt-1 ml-1" color="var(--el-color-primary)" name="MoreFilled" />
                </template>

                <el-row v-for="i in props.tags.slice(1)" :key="i">
                    <TagInfo :tag-path="i.codePath" />
                    <span class="ml-0.5">{{ i.codePath }}</span>
                </el-row>
            </el-popover>
        </el-row>
    </div>
</template>

<script lang="ts" setup>
import SvgIcon from '@/components/svgIcon/index.vue';
import TagInfo from './TagInfo.vue';
const props = defineProps({
    tags: {
        type: [Array<any>],
        required: true,
        default: () => [],
    },
});
</script>

<style lang="scss"></style>
