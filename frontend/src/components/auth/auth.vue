<template>
    <div v-if="hasAuth">
        <slot />
    </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import { useUserInfo } from '@/store/userInfo';

const props = defineProps<{
    value: string;
}>();

// 获取 Pinia 中的用户权限
const hasAuth = computed(() => {
    return useUserInfo().userInfo.authBtnList.some((v: string) => v === props.value);
});
</script>
