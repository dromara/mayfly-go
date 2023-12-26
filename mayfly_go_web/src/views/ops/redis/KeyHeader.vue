<template>
    <div>
        <!-- key name -->
        <div class="key-header-item key-name-input">
            <el-input ref="keyNameInput" v-model="ki.key" title="点击重命名" placeholder="KeyName">
                <template #prepend>
                    <span class="key-detail-type">{{ ki.type }}</span>
                </template>

                <template #suffix>
                    <SvgIcon v-auth="'redis:data:save'" @click="renameKey" title="点击重命名" name="check" class="cursor-pointer" />
                </template>
            </el-input>
        </div>

        <!-- key ttl -->
        <div class="key-header-item key-ttl-input">
            <el-input type="number" v-model.number="ki.timed" placeholder="单位(秒),负数永久" title="点击修改过期时间">
                <template #prepend>
                    <span>TTL</span>
                </template>

                <template #suffix>
                    <!-- 时间转换 -->
                    <el-tooltip effect="dark" placement="top">
                        <template #content>{{ ttlConveter(ki.timed) }}</template>
                        <span class="ml10">
                            <el-icon class="mr5"><InfoFilled /></el-icon>
                        </span>
                    </el-tooltip>

                    <!-- save ttl -->
                    <SvgIcon v-auth="'redis:data:save'" @click="ttlKey" title="点击修改过期时间" name="check" />
                </template>
            </el-input>
        </div>

        <!-- del & refresh btn -->
        <div class="key-header-item key-header-btn-con">
            <el-button type="success" @click="refreshKey" icon="refresh" title="刷新"></el-button>
            <el-button v-auth="'redis:data:del'" type="danger" @click="delKey" icon="delete" title="删除"></el-button>
        </div>
    </div>
</template>
<script lang="ts" setup>
import { reactive, watch, toRefs, onMounted } from 'vue';
import { redisApi } from './api';
import { ElMessage, ElMessageBox } from 'element-plus';

const props = defineProps({
    redisId: {
        type: Number,
    },
    db: {
        type: Number,
    },
    keyInfo: {
        type: [Object],
    },
});

const emit = defineEmits(['refreshContent', 'delKey', 'changeKey']);

const state = reactive({
    redisId: 0,
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
        id: props.redisId,
        db: props.db,
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
    await redisApi.renameKey.request({
        id: props.redisId,
        db: props.db,
        newKey: state.ki.key,
        key: state.oldKey,
    });
    ElMessage.success('设置成功');
    emit('changeKey');
};

const ttlKey = async () => {
    if (!state.oldKey) {
        return;
    }
    // ttl <= 0，则持久化该key
    if (state.ki.timed <= 0) {
        try {
            await ElMessageBox.confirm('确定持久化该key?', 'Warning', {
                confirmButtonText: '确认',
                cancelButtonText: '取消',
                type: 'warning',
            });
        } catch (err) {
            return;
        }
        await persistKey();
        state.ki.timed = -1;
        return;
    }

    await redisApi.expireKey.request({
        id: props.redisId,
        db: props.db,
        key: state.ki.key,
        seconds: state.ki.timed,
    });
    ElMessage.success('设置成功');
    emit('changeKey');
};

const persistKey = async () => {
    await redisApi.persistKey.request({
        id: props.redisId,
        db: props.db,
        key: state.keyInfo.key,
    });
    ElMessage.success('设置成功');
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
        return '永久';
    }
    if (!ttl) {
        ttl = 0;
    }
    let second = parseInt(ttl); // 秒
    let min = 0; // 分
    let hour = 0; // 小时
    let day = 0;
    if (second > 60) {
        min = parseInt(second / 60 + '');
        second = second % 60;
        if (min > 60) {
            hour = parseInt(min / 60 + '');
            min = min % 60;
            if (hour > 24) {
                day = parseInt(hour / 24 + '');
                hour = hour % 24;
            }
        }
    }
    let result = '' + second + 's';
    if (min > 0) {
        result = '' + min + 'm:' + result;
    }
    if (hour > 0) {
        result = '' + hour + 'h:' + result;
    }
    if (day > 0) {
        result = '' + day + 'd:' + result;
    }
    return result;
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
    width: calc(100% - 332px);
    min-width: 220px;
    max-width: 800px;
    margin-right: 10px;
    margin-bottom: 10px;
}

.key-header-item.key-ttl-input {
    width: 200px;
    margin-right: 10px;
    margin-bottom: 10px;
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
