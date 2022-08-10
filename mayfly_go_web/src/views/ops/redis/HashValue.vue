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

            <el-row class="mt10">
                <el-form label-position="right" :inline="true">
                    <el-form-item label="field" label-width="40px" v-if="operationType == 2">
                        <el-input placeholder="支持*模糊field" style="width: 140px" v-model="scanParam.match" clearable size="small"></el-input>
                    </el-form-item>
                    <el-form-item label="count" v-if="operationType == 2">
                        <el-input placeholder="count" style="width: 62px" v-model.number="scanParam.count" size="small"></el-input>
                    </el-form-item>
                    <el-form-item>
                        <el-button v-if="operationType == 2" @click="reHscan()" type="success" icon="search" plain size="small"></el-button>
                        <el-button v-if="operationType == 2" @click="hscan()" icon="bottom" plain size="small">scan</el-button>
                        <el-button @click="onAddHashValue" icon="plus" size="small" plain>添加</el-button>
                    </el-form-item>
                    <div v-if="operationType == 2" class="mt10" style="float: right">
                        <span>fieldSize: {{ keySize }}</span>
                    </div>
                </el-form>
            </el-row>
            <el-table :data="hashValues" stripe style="width: 100%">
                <el-table-column prop="field" label="field" width>
                    <template #default="scope">
                        <el-input v-model="scope.row.field" clearable size="small"></el-input>
                    </template>
                </el-table-column>
                <el-table-column prop="value" label="value" min-width="200">
                    <template #default="scope">
                        <el-input v-model="scope.row.value" clearable type="textarea" :autosize="{ minRows: 2, maxRows: 10 }" size="small"></el-input>
                    </template>
                </el-table-column>
                <el-table-column label="操作" width="120">
                    <template #default="scope">
                        <el-button v-if="operationType == 2" type="success" @click="hset(scope.row)" icon="check" size="small" plain></el-button>
                        <el-button type="danger" @click="hdel(scope.row.field, scope.$index)" icon="delete" size="small" plain></el-button>
                    </template>
                </el-table-column>
            </el-table>
        </el-form>
        <template #footer v-if="operationType == 1">
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
import { ElMessage, ElMessageBox } from 'element-plus';
import { isTrue, notEmpty } from '@/common/assert';

export default defineComponent({
    name: 'HashValue',
    components: {},
    props: {
        visible: {
            type: Boolean,
        },
        title: {
            type: String,
        },
        // 操作类型，1：新增，2：修改
        operationType: {
            type: [Number],
            require: true,
        },
        redisId: {
            type: [Number],
            require: true,
        },
        keyInfo: {
            type: [Object],
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
            redisId: 0,
            key: {
                key: '',
                type: 'hash',
                timed: -1,
            },
            scanParam: {
                key: '',
                id: 0,
                cursor: 0,
                match: '',
                count: 10,
            },
            keySize: 0,
            hashValues: [
                {
                    field: '',
                    value: '',
                },
            ],
        });

        const cancel = () => {
            emit('update:visible', false);
            emit('cancel');
            setTimeout(() => {
                state.hashValues = [];
                state.key = {} as any;
            }, 500);
        };

        watch(props, async (newValue) => {
            const visible = newValue.visible;
            state.redisId = newValue.redisId;
            state.key = newValue.keyInfo;
            state.operationType = newValue.operationType;

            if (visible && state.operationType == 2) {
                state.scanParam.id = props.redisId;
                state.scanParam.key = state.key.key;
                await reHscan();
            }

            state.dialogVisible = visible;
        });

        const reHscan = async () => {
            state.scanParam.id = state.redisId;
            state.scanParam.cursor = 0;
            hscan();
        };

        const hscan = async () => {
            const match = state.scanParam.match;
            if (!match || match == '' || match == '*') {
                if (state.scanParam.count > 100) {
                    ElMessage.error('match为空或者*时, count不能超过100');
                    return;
                }
            } else {
                if (state.scanParam.count > 1000) {
                    ElMessage.error('count不能超过1000');
                    return;
                }
            }

            const scanRes = await redisApi.hscan.request(state.scanParam);
            state.scanParam.cursor = scanRes.cursor;
            state.keySize = scanRes.keySize;

            const keys = scanRes.keys;
            const hashValue = [];
            const fieldCount = keys.length / 2;
            let nextFieldIndex = 0;
            for (let i = 0; i < fieldCount; i++) {
                hashValue.push({ field: keys[nextFieldIndex++], value: keys[nextFieldIndex++] });
            }
            state.hashValues = hashValue;
        };

        const hdel = async (field: any, index: any) => {
            // 如果是新增操作，则直接数组移除即可
            if (state.operationType == 1) {
                state.hashValues.splice(index, 1);
                return;
            }
            await ElMessageBox.confirm(`确定删除[${field}]?`, '提示', {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'warning',
            });
            await redisApi.hdel.request({
                id: state.redisId,
                key: state.key.key,
                field,
            });
            ElMessage.success('删除成功');
            reHscan();
        };

        const hset = async (row: any) => {
            await redisApi.saveHashValue.request({
                id: state.redisId,
                key: state.key.key,
                timed: state.key.timed,
                value: [
                    {
                        field: row.field,
                        value: row.value,
                    },
                ],
            });
            ElMessage.success('保存成功');
        };

        const onAddHashValue = () => {
            state.hashValues.unshift({ field: '', value: '' });
        };

        const saveValue = async () => {
            notEmpty(state.key.key, 'key不能为空');
            isTrue(state.hashValues.length > 0, 'hash内容不能为空');
            const sv = { value: state.hashValues, id: state.redisId };
            Object.assign(sv, state.key);
            await redisApi.saveHashValue.request(sv);
            ElMessage.success('保存成功');

            cancel();
            emit('valChange');
        };

        return {
            ...toRefs(state),
            reHscan,
            hscan,
            cancel,
            hdel,
            hset,
            onAddHashValue,
            saveValue,
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