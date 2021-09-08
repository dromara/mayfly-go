<template>
    <el-dialog :title="keyValue.key" v-model="dialogVisible" :before-close="cancel" :show-close="false" width="800px">
        <el-form>
            <el-form-item>
                <!-- <el-input v-model="keyValue.value" type="textarea" :autosize="{ minRows: 10, maxRows: 20 }" autocomplete="off"></el-input> -->

                
            </el-form-item>
            <vue3-json-editor v-model="keyValue2.jsonValue" @json-change="valueChange" :show-btns="false" :expandedOnStart="true" />
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
import { Vue3JsonEditor } from 'vue3-json-editor';

export default defineComponent({
    name: 'ValueDialog',
    components: {
        Vue3JsonEditor,
    },
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
            dialogVisible: false,
            keyValue2: {} as any,
        });
        const cancel = () => {
            emit('update:visible', false);
            emit('cancel');
        };

        watch(
            () => props.visible,
            (val) => {
                state.dialogVisible = val;
            }
        );

        watch(
            () => props.keyValue,
            (val) => {
                state.keyValue2 = val;
                if (typeof val.value == 'string') {
                    state.keyValue2.jsonValue = JSON.parse(val.value)
                } else {
                    state.keyValue2.jsonValue = val.value;
                }
            }
        );

        const saveValue = async () => {
            isTrue(state.keyValue2.type == 'string', '暂不支持除string外其他类型修改');

            await redisApi.saveStringValue.request(state.keyValue2);
            ElMessage.success('保存成功');
            cancel();
        };

        const valueChange = (val: any) => {
            state.keyValue2.value = JSON.stringify(val);
        };

        return {
            ...toRefs(state),
            saveValue,
            valueChange,
            cancel,
        };
    },
});
</script>