<template>
    <div></div>
</template>

<script lang="ts" setup>
import openApi from '@/common/openApi';
import { Msg } from '@/hooks/useI18n';
import type { LoginResult } from '@/views/system/types';
import { onMounted, toRaw } from 'vue';
import { useRoute } from 'vue-router';

const route = useRoute();

onMounted(async () => {
    try {
        const queryParam = route.query;
        // 使用hash路由，回调code可能会被设置到search
        // 如 localhost:8888/?code=xxxx/oauth2/callback，导致route.query获取不到值
        if (location.search) {
            const searchParams = location.search.split('?')[1];
            if (searchParams) {
                for (let searchParam of searchParams.split('&')) {
                    const searchParamSplit = searchParam.split('=');
                    queryParam[searchParamSplit[0]] = searchParamSplit[1];
                }
            }
        }

        const res: LoginResult = await openApi.oauth2Callback({
            code: String(queryParam.code ?? ''),
            state: String(queryParam.state ?? ''),
        });
        Msg.success('system.oauth.authSuccess');
        top?.opener.postMessage(toRaw(res), '*');
        window.close();
    } catch (e: unknown) {
        console.error('oauth2 callback handle error: ', e);
        setTimeout(() => {
            window.close();
        }, 5000);
    }
});
</script>
<style lang="scss"></style>
