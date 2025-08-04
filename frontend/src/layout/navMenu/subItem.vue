<template>
    <template v-for="val in chils">
        <el-sub-menu :index="val.path" :key="val.path" v-if="val.children && val.children.length > 0">
            <template #title>
                <SvgIcon :name="val.meta.icon" />
                <span>{{ $t(val.meta.title) }}</span>
            </template>
            <sub-item :chil="val.children" />
        </el-sub-menu>
        <el-menu-item :index="val.path" :key="val?.path" v-else>
            <template v-if="!val.meta.link || (val.meta.link && val.meta.linkType == 1)">
                <SvgIcon :name="val.meta.icon" />
                <span>{{ $t(val.meta.title) }}</span>
            </template>
            <template v-else>
                <a class="w-full" :href="val.meta.link" target="_blank">
                    <SvgIcon :name="val.meta.icon" />
                    {{ $t(val.meta.title) }}
                </a>
            </template>
        </el-menu-item>
    </template>
</template>

<script setup lang="ts" name="navMenuSubItem">
import { computed } from 'vue';

// 定义 props
interface Props {
    chil?: any[];
}

const props = withDefaults(defineProps<Props>(), {
    chil: () => [],
});

// 获取父级菜单数据
const chils = computed(() => {
    return props.chil as any;
});
</script>
