<template>
    <div v-if="props.tags">
        <el-row v-for="(tag, idx) in props.tags?.slice(0, 1)" :key="idx">
            <TagInfo :tag-path="tag.tagPath" />
            <span class="ml3">{{ tag.tagPath }}</span>

            <!-- 展示剩余的标签信息 -->
            <el-popover :show-after="300" v-if="props.tags.length > 1 && idx == 0" placement="top-start" width="230" trigger="hover">
                <template #reference>
                    <SvgIcon class="mt5 ml5" color="var(--el-color-primary)" name="MoreFilled" />
                </template>

                <el-row v-for="i in props.tags.slice(1)" :key="i">
                    <TagInfo :tag-path="i.tagPath" />
                    <span class="ml3">{{ i.tagPath }}</span>
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
    },
});
</script>

<style lang="scss"></style>
