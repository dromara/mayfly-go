<template>
    <div>
        <div class="layout-view-bg-white flex layout-view-link">
            <a :href="currentRouteMeta.link" target="_blank" class="flex-margin">{{ currentRouteMeta.title }}：{{ currentRouteMeta.link }}</a>
        </div>
    </div>
</template>

<script lang="ts">
import { defineComponent, computed } from 'vue';
export default defineComponent({
    name: 'layoutLinkView',
    props: {
        meta: {
            type: Object,
            default: () => {},
        },
    },
    setup(props) {
        // 获取父级菜单数据
        const currentRouteMeta = computed(() => {
            return props.meta;
        });
        return {
            currentRouteMeta,
        };
    },
});
</script>
