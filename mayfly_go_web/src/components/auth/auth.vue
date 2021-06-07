<template>
    <div v-if="getUserAuthBtnList">
        <slot />
    </div>
</template>

<script lang="ts">
import { computed } from 'vue';
import { useStore } from '/@/store/index.ts';
export default {
    name: 'auth',
    props: {
        value: {
            type: String,
            default: () => '',
        },
    },
    setup(props) {
        const store = useStore();
        // 获取 vuex 中的用户权限
        const getUserAuthBtnList = computed(() => {
            return store.state.userInfos.userInfos.authBtnList.some((v: any) => v === props.value);
        });
        return {
            getUserAuthBtnList,
        };
    },
};
</script>
