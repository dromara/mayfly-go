<template>
    <el-dialog :title="keyValue.key" v-model="visible" :before-close="cancel" :show-close="false" width="750px">
        <el-form>
            <el-form-item>
                <el-input v-model="keyValue.value" type="textarea" :autosize="{ minRows: 10, maxRows: 20 }" autocomplete="off"></el-input>
            </el-form-item>
        </el-form>
        <template #footer>
            <div class="dialog-footer">
                <el-button @click="saveValue" type="primary" size="mini">确 定</el-button>
                <el-button @click="cancel()" size="mini">取 消</el-button>
            </div>
        </template>
    </el-dialog>
</template>
<script lang="ts">
import { defineComponent, reactive, watch, toRefs } from 'vue';
import { redisApi } from './api';
import { ElMessage } from 'element-plus';
import { isTrue } from '@/common/assert';
export default defineComponent({
    name: 'ValueDialog',
    props: {
        visible: {
            type: Boolean,
        },
        title: {
            type: String,
        },
        keyValue: {
            type: [String, Object],
        },
    },
    setup(props: any, { emit }) {
        const state = reactive({
            visible: false,
            keyValue: {} as any,
        });
        const cancel = () => {
            emit('update:visible', false);
            emit('cancel');
        };

        watch(
            () => props.visible,
            (val) => {
                state.visible = val;
            }
        );

        watch(
            () => props.keyValue,
            (val) => {
                state.keyValue = val;
                if (state.keyValue.type != 'string') {
                    state.keyValue.value = JSON.stringify(val.value, undefined, 2)
                }
                // state.keyValue.value = JSON.stringify(val.value, undefined, 2)
            }
        );

        const saveValue = async () => {
            isTrue(state.keyValue.type == 'string', "暂不支持除string外其他类型修改")
            await redisApi.saveStringValue.request(state.keyValue);
            ElMessage.success('保存成功');
            cancel();
        };

        return {
            ...toRefs(state),
            saveValue,
            cancel,
        };
    },
});
</script>