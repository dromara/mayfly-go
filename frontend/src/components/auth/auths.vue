<template>
    <div v-if="hasAuthAny">
        <slot />
    </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import { useUserInfo } from '@/store/userInfo';

const props = defineProps<{
    value: string[];
}>();

// 获取 Pinia 中的用户权限（任一匹配即通过）
const hasAuthAny = computed(() => {
    const authBtnList: string[] = useUserInfo().userInfo.authBtnList;
    return props.value.some((v) => authBtnList.includes(v));
});
</script>
