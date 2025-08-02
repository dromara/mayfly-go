<template>
    <el-popover @before-enter="getDetail" width="500" :title="$t('common.detail')" trigger="click">
        <template #reference>
            <el-link type="primary" :style="{ fontSize: props.size == 'small' ? '12px' : '14px', verticalAlign: 'baseline' }">
                <slot></slot>
            </el-link>
        </template>

        <el-descriptions v-loading="state.loading" :size="props.size" :column="3" border>
            <el-descriptions-item :span="1" label="ID">{{ state.detail.id }}</el-descriptions-item>
            <el-descriptions-item :span="1" :label="$t('common.code')">{{ state.detail.code }}</el-descriptions-item>
            <el-descriptions-item :span="1" :label="$t('common.name')">{{ state.detail.name }}</el-descriptions-item>

            <!-- <el-descriptions-item :span="3" :label="$t('tag.relateTag')"><ResourceTags :tags="state.detail.tags" /></el-descriptions-item> -->

            <el-descriptions-item :span="3" label="Host">
                <SvgIcon :name="getDbDialect(state.detail.type).getInfo().icon" :size="20" />
                {{ state.detail.host }}:{{ state.detail.port }}
            </el-descriptions-item>

            <el-descriptions-item :span="3" :label="$t('db.acName')">{{ state.detail.authCertName }}</el-descriptions-item>

            <el-descriptions-item :span="3" :label="$t('common.remark')">{{ state.detail.remark }}</el-descriptions-item>

            <el-descriptions-item :span="2" :label="$t('common.createTime')">{{ formatDate(state.detail.createTime) }} </el-descriptions-item>
            <el-descriptions-item :span="1" :label="$t('common.creator')">{{ state.detail.creator }}</el-descriptions-item>

            <el-descriptions-item :span="2" :label="$t('common.updateTime')">{{ formatDate(state.detail.updateTime) }} </el-descriptions-item>
            <el-descriptions-item :span="1" :label="$t('common.modifier')">{{ state.detail.modifier }}</el-descriptions-item>
        </el-descriptions>
    </el-popover>
</template>

<script lang="ts" setup>
import { reactive } from 'vue';
import { dbApi } from '../api';
import { formatDate } from '@/common/utils/format';
import { getDbDialect } from '../dialect/index';

const props = defineProps({
    id: {
        type: Number,
        requierd: true,
    },
    size: {
        type: String,
        default: 'default',
    },
});

const state = reactive({
    loading: false,
    detail: {} as any,
});

const getDetail = async () => {
    try {
        state.detail = {};
        state.loading = true;
        const res = await dbApi.dbs.request({
            id: props.id,
        });
        if (res.total == 0) {
            return;
        }

        state.detail = res.list?.[0];
    } finally {
        state.loading = false;
    }
};
</script>

<style></style>
