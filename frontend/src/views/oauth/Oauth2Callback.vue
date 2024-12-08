<template>
    <div></div>
</template>

<script lang="ts" setup>
import { onMounted } from 'vue';
import { useRoute } from 'vue-router';
import { ElMessage } from 'element-plus';
import openApi from '@/common/openApi';
import { useI18n } from 'vue-i18n';

const route = useRoute();

const { t } = useI18n();

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

        const res: any = await openApi.oauth2Callback(queryParam);
        ElMessage.success(t('system.oauth.authSuccess'));
        top?.opener.postMessage(res);
        window.close();
    } catch (e: any) {
        setTimeout(() => {
            window.close();
        }, 1500);
    }
});
</script>
<style lang="scss"></style>
