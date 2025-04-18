<template>
    <div>
        <!-- key name -->
        <div class="key-header-item key-name-input">
            <el-input ref="keyNameInput" v-model="ki.key" :title="$t('redis.renameTips')" placeholder="KeyName">
                <template #prepend>
                    <span class="key-detail-type">{{ ki.type }}</span>
                </template>

                <template #suffix>
                    <SvgIcon v-auth="'redis:data:save'" @click="renameKey" :title="$t('redis.renameTips')" name="check" class="cursor-pointer" />
                </template>
            </el-input>
        </div>

        <!-- key ttl -->
        <div class="key-header-item key-ttl-input">
            <el-input type="number" v-model.number="ki.timed" :placeholder="$t('redis.ttlPlaceholder')" :title="$t('redis.ttlTips')">
                <template #prepend>
                    <span>TTL</span>
                </template>

                <template #suffix>
                    <!-- 时间转换 -->
                    <el-tooltip effect="dark" placement="top">
                        <template #content>{{ ttlConveter(ki.timed) }}</template>
                        <span class="ml-2">
                            <el-icon class="mr-1"><InfoFilled /></el-icon>
                        </span>
                    </el-tooltip>

                    <!-- save ttl -->
                    <SvgIcon v-auth="'redis:data:save'" @click="ttlKey" :title="$t('redis.ttlTips')" name="check" />
                </template>
            </el-input>
        </div>

        <!-- del & refresh btn -->
        <div class="key-header-item key-header-btn-con">
            <el-button type="success" @click="refreshKey" icon="refresh" :title="$t('common.refresh')"></el-button>
            <el-button v-auth="'redis:data:del'" type="danger" @click="delKey" icon="delete" :title="$t('common.delete')"></el-button>
        </div>
    </div>
</template>
<script lang="ts" setup>
import { reactive, watch, toRefs, onMounted } from 'vue';
import { redisApi } from './api';
import { ElMessage, ElMessageBox } from 'element-plus';
import { formatTime } from '@/common/utils/format';
import { RedisInst } from './redis';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
    redis: {
        type: RedisInst,
        required: true,
    },
    keyInfo: {
        type: [Object],
    },
});

const emit = defineEmits(['refreshContent', 'delKey', 'changeKey']);

const state = reactive({
    keyInfo: {
        key: '',
        type: '',
        timed: -1,
    } as any,
    ki: {
        key: '',
        type: '',
        timed: -1,
    } as any,
    oldKey: '',
    memuse: 0,
});

onMounted(() => {
    state.keyInfo = props.keyInfo;
    state.oldKey = props.keyInfo?.key;
});

const refreshKey = async () => {
    const ttl = await redisApi.keyTtl.request({
        id: props.redis.id,
        db: props.redis.db,
        key: state.oldKey,
    });
    state.keyInfo.timed = ttl;
    emit('refreshContent');
};

const delKey = async () => {
    emit('delKey', state.ki.key);
};

const renameKey = async () => {
    if (!state.oldKey || state.ki.key == state.oldKey) {
        return;
    }
    // RENAME key newkey
    await props.redis.runCmd(['RENAME', state.oldKey, state.ki.key]);
    ElMessage.success(t('redis.settingSuccess'));
    emit('changeKey');
};

const ttlKey = async () => {
    if (!state.oldKey) {
        return;
    }
    // ttl <= 0，则持久化该key
    if (state.ki.timed <= 0) {
        try {
            await ElMessageBox.confirm(t('redis.persistenceConfirm'), 'Warning', {
                confirmButtonText: t('common.confirm'),
                cancelButtonText: t('common.cancel'),
                type: 'warning',
            });
        } catch (err) {
            return;
        }
        await persistKey();
        state.ki.timed = -1;
        return;
    }

    // EXPIRE key seconds [NX | XX | GT | LT]
    await props.redis.runCmd(['EXPIRE', state.ki.key, state.ki.timed]);
    ElMessage.success(t('redis.settingSuccess'));
    emit('changeKey');
};

const persistKey = async () => {
    // PERSIST key
    await props.redis.runCmd(['PERSIST', state.keyInfo.key]);
    ElMessage.success(t('redis.settingSuccess'));
    emit('changeKey');
};

const { ki } = toRefs(state);

const setKeyInfo = (val: any) => {
    state.ki.timed = val.timed;
    state.ki.key = val.key;
    state.oldKey = val.key;
    state.ki.type = val.type;
};

watch(
    () => props.keyInfo,
    (val: any) => {
        setKeyInfo(val);
    },
    { deep: true }
);

const ttlConveter = (ttl: any) => {
    if (ttl == -1 || ttl == 0) {
        return t('redis.permanent');
    }
    if (!ttl) {
        ttl = 0;
    }
    return formatTime(ttl);
};
</script>
<style lang="scss">
.key-detail-type {
    text-transform: capitalize;
    text-align: center;
    min-width: 34px;
    display: inline-block;
}

.cursor-pointer {
    cursor: pointer;
}

.key-header-item {
    /*padding-right: 15px;*/
    /*margin-bottom: 10px;*/
    float: left;
}

.key-header-item.key-name-input {
    width: calc(100% - 322px);
    min-width: 230px;
    max-width: 800px;
    margin-right: 5px;
    margin-bottom: 5px;
}

.key-header-item.key-ttl-input {
    width: 190px;
    margin-right: 5px;
    margin-bottom: 5px;
}

/*hide number input button*/
.key-header-item.key-ttl-input input::-webkit-inner-spin-button,
.key-header-item.key-ttl-input input::-webkit-outer-spin-button {
    appearance: none;
}

.key-header-item.key-header-btn-con .el-button + .el-button {
    margin-left: 4px;
}

/*refresh btn rotating*/
.key-header-info .key-header-btn-con .rotating .el-icon-refresh {
    animation: rotate 1.5s linear infinite;
}
</style>
