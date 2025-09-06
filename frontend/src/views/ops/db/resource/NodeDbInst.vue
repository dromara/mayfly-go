<template>
    <BaseTreeNode v-bind="$attrs">
        <template #prefix="{ data }">
            <el-popover @show="showDbInfo(data.params)" :show-after="500" placement="right-start" :title="$t('db.dbInstInfo')" trigger="hover" :width="250">
                <template #reference>
                    <SvgIcon :name="getDbDialect(data.params.type).getInfo().icon" :size="18" />
                </template>
                <template #default>
                    <el-descriptions :column="1" size="small">
                        <el-descriptions-item :label="$t('common.name')">
                            {{ data.params.name }}
                        </el-descriptions-item>
                        <el-descriptions-item label="Host">
                            {{ `${data.params.host}:${data.params.port}` }}
                        </el-descriptions-item>
                        <el-descriptions-item label="version">
                            <span v-loading="loadingServerInfo"> {{ `${dbServerInfo?.version}` }}</span>
                        </el-descriptions-item>
                        <!-- <el-descriptions-item :label="$t('db.acName')">
                            {{ data.params.authCertName }}
                        </el-descriptions-item> -->
                        <el-descriptions-item :label="$t('common.remark')">
                            {{ data.params.remark }}
                        </el-descriptions-item>
                    </el-descriptions>
                </template>
            </el-popover>
        </template>
    </BaseTreeNode>
</template>

<script lang="ts" setup>
import { ref } from 'vue';
import { dbApi } from '../api';
import { getDbDialect } from '../dialect/index';
import BaseTreeNode from '@/views/ops/resource/BaseTreeNode.vue';

const serverInfoReqParam = ref({
    instanceId: 0,
});

const { execute: getDbServerInfo, isFetching: loadingServerInfo, data: dbServerInfo } = dbApi.getInstanceServerInfo.useApi<any>(serverInfoReqParam);

const showDbInfo = async (db: any) => {
    if (dbServerInfo.value) {
        dbServerInfo.value.version = '';
    }
    serverInfoReqParam.value.instanceId = db.id;
    await getDbServerInfo();
};
</script>

<style lang="scss"></style>
