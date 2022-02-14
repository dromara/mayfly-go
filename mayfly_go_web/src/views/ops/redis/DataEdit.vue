<template>
    <el-dialog :title="title" v-model="dialogVisible" :before-close="cancel" :show-close="false" width="750px" :destroy-on-close="true">
        <el-form label-width="85px">
            <el-form-item prop="key" label="key:">
                <el-input :disabled="operationType == 2" v-model="key.key"></el-input>
            </el-form-item>
            <el-form-item prop="timed" label="过期时间:">
                <el-input v-model.number="key.timed" type="number"></el-input>
            </el-form-item>
            <el-form-item prop="dataType" label="数据类型:">
                <el-select :disabled="operationType == 2" style="width: 100%" v-model="key.type" placeholder="请选择数据类型">
                    <el-option key="string" label="string" value="string"> </el-option>
                    <el-option key="hash" label="hash" value="hash"> </el-option>
                    <el-option key="set" label="set" value="set"> </el-option>
                </el-select>
            </el-form-item>

            <el-form-item v-if="keyInfo.type == 'string'" prop="value" label="内容:">
                <div id="string-value-text" style="width: 100%">
                    <el-input class="json-text" v-model="string.value" type="textarea" :autosize="{ minRows: 10, maxRows: 20 }"></el-input>
                    <el-select class="text-type-select" @change="onChangeTextType" v-model="string.type">
                        <el-option key="text" label="text" value="text"> </el-option>
                        <el-option key="json" label="json" value="json"> </el-option>
                    </el-select>
                </div>
            </el-form-item>

            <span v-if="keyInfo.type == 'hash'">
                <el-button @click="onAddHashValue" icon="plus" size="small" plain class="mt10">添加</el-button>
                <el-table :data="hashValue" stripe style="width: 100%">
                    <el-table-column prop="key" label="key" width>
                        <template #default="scope">
                            <el-input v-model="scope.row.key" clearable size="small"></el-input>
                        </template>
                    </el-table-column>
                    <el-table-column prop="value" label="value" min-width="200">
                        <template #default="scope">
                            <el-input
                                v-model="scope.row.value"
                                clearable
                                type="textarea"
                                :autosize="{ minRows: 2, maxRows: 10 }"
                                size="small"
                            ></el-input>
                        </template>
                    </el-table-column>
                    <el-table-column label="操作" width="90">
                        <template #default="scope">
                            <el-button type="danger" @click="hash.value.splice(scope.$index, 1)" icon="delete" size="small" plain>删除</el-button>
                        </template>
                    </el-table-column>
                </el-table>
            </span>

            <span v-if="keyInfo.type == 'set'">
                <el-button @click="onAddSetValue" icon="plus" size="small" plain class="mt10">添加</el-button>
                <el-table :data="setValue" stripe style="width: 100%">
                    <el-table-column prop="value" label="value" min-width="200">
                        <template #default="scope">
                            <el-input
                                v-model="scope.row.value"
                                clearable
                                type="textarea"
                                :autosize="{ minRows: 2, maxRows: 10 }"
                                size="small"
                            ></el-input>
                        </template>
                    </el-table-column>
                    <el-table-column label="操作" width="90">
                        <template #default="scope">
                            <el-button type="danger" @click="set.value.splice(scope.$index, 1)" icon="delete" size="small" plain>删除</el-button>
                        </template>
                    </el-table-column>
                </el-table>
            </span>
        </el-form>
        <template #footer>
            <div class="dialog-footer">
                <el-button @click="saveValue" type="primary" v-auth="'redis:data:save'">确 定</el-button>
                <el-button @click="cancel()">取 消</el-button>
            </div>
        </template>
    </el-dialog>
</template>
<script lang="ts">
import { defineComponent, reactive, watch, toRefs } from 'vue';
import { redisApi } from './api';
import { ElMessage } from 'element-plus';
import { isTrue, notEmpty } from '@/common/assert';
import { formatJsonString } from '@/common/utils/format';

export default defineComponent({
    name: 'DateEdit',
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
        stringValue: {
            type: [String],
        },
        setValue: {
            type: [Array, Object],
        },
        hashValue: {
            type: [Array, Object],
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
            hash: {
                value: [
                    {
                        key: '',
                        value: '',
                    },
                ],
            },
            set: {
                value: [{ value: '' }],
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
                state.hash.value = [
                    {
                        key: '',
                        value: '',
                    },
                ];
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

        watch(
            () => props.operationType,
            (val) => {
                state.operationType = val;
            }
        );

        watch(
            () => props.keyInfo,
            (val) => {
                if (val) {
                    state.key = { ...val };
                }
            },
            {
                deep: true, // 深度监听的参数
            }
        );

        watch(
            () => props.stringValue,
            (val) => {
                if (val) {
                    state.string.value = val;
                }
            },
            {
                deep: true, // 深度监听的参数
            }
        );

        watch(
            () => props.setValue,
            (val) => {
                if (val) {
                    state.set.value = val;
                }
            },
            {
                deep: true, // 深度监听的参数
            }
        );

        watch(
            () => props.hashValue,
            (val) => {
                if (val) {
                    state.hash.value = val;
                }
            },
            {
                deep: true, // 深度监听的参数
            }
        );

        const saveValue = async () => {
            notEmpty(state.key.key, 'key不能为空');

            if (state.key.type == 'string') {
                notEmpty(state.string.value, 'value不能为空');
                const sv = { value: formatJsonString(state.string.value, true), id: state.redisId };
                Object.assign(sv, state.key);
                await redisApi.saveStringValue.request(sv);
            }

            if (state.key.type == 'hash') {
                isTrue(state.hash.value.length > 0, 'hash内容不能为空');
                const sv = { value: state.hash.value, id: state.redisId };
                Object.assign(sv, state.key);
                await redisApi.saveHashValue.request(sv);
            }

            if (state.key.type == 'set') {
                isTrue(state.set.value.length > 0, 'set内容不能为空');
                const sv = { value: state.set.value.map((x) => x.value), id: state.redisId };
                Object.assign(sv, state.key);
                await redisApi.saveSetValue.request(sv);
            }

            ElMessage.success('数据保存成功');
            cancel();
            emit('valChange');
        };

        const onAddHashValue = () => {
            state.hash.value.push({ key: '', value: '' });
        };

        const onAddSetValue = () => {
            state.set.value.push({ value: '' });
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
            onAddHashValue,
            onAddSetValue,
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