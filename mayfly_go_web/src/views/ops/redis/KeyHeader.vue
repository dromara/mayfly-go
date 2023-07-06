<template>
    <div>
        <!-- key name -->
        <div class="key-header-item key-name-input">
            <el-input ref="keyNameInput" v-model="keyInfo.key" title="点击重命名" placeholder="KeyName">
                <template #prepend>
                    <span class="key-detail-type">{{ keyInfo.type }}</span>
                </template>

                <template #suffix>
                    <SvgIcon v-auth="'redis:data:save'" @click="renameKey" title="点击重命名" name="check" class="cursor-pointer" />
                </template>
            </el-input>
        </div>

        <!-- key ttl -->
        <div class="key-header-item key-ttl-input">
            <el-input type="number" v-model.number="keyInfo.timed" placeholder="单位(秒),负数永久" title="点击修改过期时间">
                <template #prepend>
                    <span slot="prepend">TTL</span>
                </template>

                <template #suffix>
                    <!-- save ttl -->
                    <SvgIcon v-auth="'redis:data:save'" @click="ttlKey" title="点击修改过期时间" name="check" />
                </template>
            </el-input>
        </div>

        <!-- del & refresh btn -->
        <div class="key-header-item key-header-btn-con">
            <el-button slot="reference" ref="refreshBtn" type="success" @click="refreshKey" icon="refresh" title="刷新"></el-button>
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

const emit = defineEmits(['refreshContent', 'changeKey', 'valChange']);

const state = reactive({
    redisId: 0,
    keyInfo: {
        key: '',
        type: '',
        timed: -1,
    } as any,
    oldKey: '',
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

const renameKey = async () => {
    if (!state.oldKey || state.keyInfo.key == state.oldKey) {
        return;
    }
    await redisApi.renameKey.request({
        id: props.redisId,
        db: props.db,
        newKey: state.keyInfo.key,
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
    if (state.keyInfo.timed <= 0) {
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
        state.keyInfo.timed = -1;
        return;
    }

    await redisApi.expireKey.request({
        id: props.redisId,
        db: props.db,
        key: state.keyInfo.key,
        seconds: state.keyInfo.timed,
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

const { keyInfo, oldKey } = toRefs(state);

// watch(
//     () => props.keyInfo,
//     (val) => {
//         state.keyInfo = val;
//         state.keyName = state.keyInfo.key;
//     }
// );
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
    margin-right: 15px;
    margin-bottom: 10px;
}

.key-header-item.key-ttl-input {
    width: 220px;
    margin-right: 15px;
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
