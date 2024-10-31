<template>
    <div v-if="getUserAuthBtnList">
        <slot />
    </div>
</template>

<script lang="ts">
import { computed } from 'vue';
import { useUserInfo } from '@/store/userInfo';
export default {
    name: 'auths',
    props: {
        value: {
            type: Array,
            default: () => [],
        },
    },
    setup(props) {
        // 获取 vuex 中的用户权限
        const getUserAuthBtnList = computed(() => {
            let flag = false;
            useUserInfo().userInfo.authBtnList.map((val: any) => {
                props.value.map((v) => {
                    if (val === v) flag = true;
                });
            });
            return flag;
        });
        return {
            getUserAuthBtnList,
        };
    },
};
</script>
