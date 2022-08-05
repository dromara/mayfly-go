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
                <el-input class="json-text" v-model="string.value" type="textarea" :autosize="{ minRows: 10, maxRows: 20 }"></el-input>
                <el-select class="text-type-select" @change="onChangeTextType" v-model="string.type">
                    <el-option key="text" label="text" value="text"> </el-option>
                    <el-option key="json" label="json" value="json"> </el-option>
                </el-select>
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
<script lang="ts">
import { defineComponent, reactive, watch, toRefs } from 'vue';
import { redisApi } from './api';
import { ElMessage } from 'element-plus';
import { notEmpty } from '@/common/assert';
import { formatJsonString } from '@/common/utils/format';

export default defineComponent({
    name: 'StringValue',
    components: {},
    props: {
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
        keyInfo: {
            type: [Object],
        },
        // 操作类型，1：新增，2：修改
        operationType: {
            type: [Number],
        },
    },
    emits: ['valChange', 'cancel', 'update:visible'],
    setup(props: any, { emit }) {
        const state = reactive({
            dialogVisible: false,
            operationType: 1,
            redisId: '',
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
                state.redisId = val;
            }
        );

        watch(props, async (newValue) => {
            state.dialogVisible = newValue.visible;
            state.key = newValue.key;
            state.redisId = newValue.redisId;
            state.key = newValue.keyInfo;
            state.operationType = newValue.operationType
            // 如果是查看编辑操作，则获取值
            if (state.dialogVisible && state.operationType == 2) {
                getStringValue();
            }
        });

        const getStringValue = async () => {
            state.string.value = await redisApi.getStringValue.request({
                id: state.redisId,
                key: state.key.key,
            });
        };

        const saveValue = async () => {
            notEmpty(state.key.key, 'key不能为空');

            notEmpty(state.string.value, 'value不能为空');
            const sv = { value: formatJsonString(state.string.value, true), id: state.redisId };
            Object.assign(sv, state.key);
            await redisApi.saveStringValue.request(sv);
            ElMessage.success('数据保存成功');
            cancel();
            emit('valChange');
        };

        // 更改文本类型
        const onChangeTextType = (val: string) => {
            if (val == 'json') {
                state.string.value = formatJsonString(state.string.value, false);
                return;
            }
            if (val == 'text') {
                state.string.value = formatJsonString(state.string.value, true);
            }
        };

        return {
            ...toRefs(state),
            saveValue,
            cancel,
            onChangeTextType,
        };
    },
});
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