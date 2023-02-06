<template>
    <el-dialog :title="title" v-model="dialogVisible" :before-close="cancel" width="800px" :destroy-on-close="true">
        <el-form label-width="85px">
            <el-form-item prop="key" label="key:">
                <el-input :disabled="operationType == 2" v-model="key.key"></el-input>
            </el-form-item>
            <el-form-item prop="timed" label="过期时间:">
                <el-input v-model.number="key.timed" type="number"></el-input>
            </el-form-item>
            <el-form-item prop="dataType" label="数据类型:">
                <el-input v-model="key.type" disabled></el-input>
            </el-form-item>

            <div id="string-value-text" style="width: 100%">
                <format-input :title="`type:【${key.type}】key:【${key.key}】`" v-model="string.value" :autosize="{ minRows: 10, maxRows: 20 }"></format-input>
                <!-- <el-select class="text-type-select" @change="onChangeTextType" v-model="string.type">
                    <el-option key="text" label="text" value="text"> </el-option>
                    <el-option key="json" label="json" value="json"> </el-option>
                </el-select> -->
            </div>
        </el-form>
        <template #footer>
            <div class="dialog-footer">
                <el-button @click="cancel()">取 消</el-button>
                <el-button @click="saveValue" type="primary" v-auth="'redis:data:save'">确 定</el-button>
            </div>
        </template>
    </el-dialog>
</template>
<script lang="ts" setup>
import { reactive, watch, toRefs } from 'vue';
import { redisApi } from './api';
import { ElMessage } from 'element-plus';
import { notEmpty } from '@/common/assert';
import FormatInput from './FormatInput.vue';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    title: {
        type: String,
    },
    redisId: {
        type: [Number],
        require: true,
    },
    db: {
        type: [String],
        require: true,
    },
    keyInfo: {
        type: [Object],
    },
    // 操作类型，1：新增，2：修改
    operationType: {
        type: [Number],
    },
})

const emit = defineEmits(['update:visible', 'cancel', 'valChange'])

const state = reactive({
    dialogVisible: false,
    operationType: 1,
    redisId: '',
    db: '0',
    key: {
        key: '',
        type: 'string',
        timed: -1,
    },
    string: {
        type: 'text',
        value: '',
    },
});

const {
    dialogVisible,
    operationType,
    key,
    string,
} = toRefs(state)

const cancel = () => {
    emit('update:visible', false);
    emit('cancel');
    setTimeout(() => {
        state.key = {
            key: '',
            type: 'string',
            timed: -1,
        };
        state.string.value = '';
        state.string.type = 'text';
    }, 500);
};

watch(
    () => props.visible,
    (val) => {
        state.dialogVisible = val;
    }
);

watch(
    () => props.redisId,
    (val) => {
        state.redisId = val as any;
    }
);

watch(
    () => props.db,
    (val) => {
        state.db = val as any;
    }
);

watch(props, async (newValue: any) => {
    state.dialogVisible = newValue.visible;
    state.redisId = newValue.redisId;
    state.db = newValue.db;
    state.key = newValue.keyInfo;
    state.operationType = newValue.operationType;
    // 如果是查看编辑操作，则获取值
    if (state.dialogVisible && state.operationType == 2) {
        getStringValue();
    }
});

const getStringValue = async () => {
    state.string.value = await redisApi.getStringValue.request({
        id: state.redisId,
        db: state.db,
        key: state.key.key,
    });
};

const saveValue = async () => {
    notEmpty(state.key.key, 'key不能为空');

    notEmpty(state.string.value, 'value不能为空');
    const sv = { value: state.string.value, id: state.redisId, db: state.db };
    Object.assign(sv, state.key);
    await redisApi.saveStringValue.request(sv);
    ElMessage.success('数据保存成功');
    cancel();
    emit('valChange');
};
</script>
<style lang="scss">
#string-value-text {
    flex-grow: 1;
    display: flex;
    position: relative;

    .text-type-select {
        position: absolute;
        z-index: 2;
        right: 10px;
        top: 10px;
        max-width: 70px;
    }
}
</style>
