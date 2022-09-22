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

            <!-- <el-button @click="onAddListValue" icon="plus" size="small" plain class="mt10">添加</el-button> -->
            <div v-if="operationType == 2" class="mt10" style="float: left">
                <span>len: {{ len }}</span>
            </div>
            <el-table :data="value" stripe style="width: 100%">
                <el-table-column prop="value" label="value" min-width="200">
                    <template #default="scope">
                        <el-input v-model="scope.row.value" clearable type="textarea" :autosize="{ minRows: 2, maxRows: 10 }" size="small"></el-input>
                    </template>
                </el-table-column>
                <el-table-column label="操作" width="140">
                    <template #default="scope">
                        <el-button
                            v-if="operationType == 2"
                            type="success"
                            @click="lset(scope.row, scope.$index)"
                            icon="check"
                            size="small"
                            plain
                        ></el-button>
                        <!-- <el-button type="danger" @click="set.value.splice(scope.$index, 1)" icon="delete" size="small" plain></el-button> -->
                    </template>
                </el-table-column>
            </el-table>
            <el-row style="margin-top: 20px" type="flex" justify="end">
                <el-pagination
                    style="text-align: right"
                    :total="len"
                    layout="prev, pager, next, total"
                    @current-change="handlePageChange"
                    v-model:current-page="pageNum"
                    :page-size="pageSize"
                ></el-pagination>
            </el-row>
        </el-form>
        <!-- <template #footer>
            <div class="dialog-footer">
                <el-button @click="cancel()">取 消</el-button>
                <el-button @click="saveValue" type="primary" v-auth="'redis:data:save'">确 定</el-button>
            </div>
        </template> -->
    </el-dialog>
</template>
<script lang="ts">
import { defineComponent, reactive, watch, toRefs } from 'vue';
import { redisApi } from './api';
import { ElMessage } from 'element-plus';
import { isTrue, notEmpty } from '@/common/assert';

export default defineComponent({
    name: 'ListValue',
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
        listValue: {
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
            value: [{ value: '' }],
            len: 0,
            start: 0,
            stop: 0,
            pageNum: 1,
            pageSize: 10,
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
                state.value = [];
            }, 500);
        };

        watch(props, async (newValue) => {
            state.dialogVisible = newValue.visible;
            state.key = newValue.key;
            state.redisId = newValue.redisId;
            state.key = newValue.keyInfo;
            state.operationType = newValue.operationType;
            // 如果是查看编辑操作，则获取值
            if (state.dialogVisible && state.operationType == 2) {
                getListValue();
            }
        });

        const getListValue = async () => {
            const pageNum = state.pageNum;
            const pageSize = state.pageSize;
            const res = await redisApi.getListValue.request({
                id: state.redisId,
                key: state.key.key,
                start: (pageNum - 1) * pageSize,
                stop: pageNum * pageSize - 1,
            });
            state.len = res.len;
            state.value = res.list.map((x: any) => {
                return {
                    value: x,
                };
            });
        };

        const lset = async (row: any, rowIndex: number) => {
            await redisApi.setListValue.request({
                id: state.redisId,
                key: state.key.key,
                index: (state.pageNum - 1) * state.pageSize + rowIndex,
                value: row.value,
            });
            ElMessage.success('数据保存成功');
        };

        const saveValue = async () => {
            notEmpty(state.key.key, 'key不能为空');
            isTrue(state.value.length > 0, 'list内容不能为空');
            // const sv = { value: state.value.map((x) => x.value), id: state.redisId };
            // Object.assign(sv, state.key);
            // await redisApi.saveSetValue.request(sv);

            ElMessage.success('数据保存成功');
            cancel();
            emit('valChange');
        };

        const onAddListValue = () => {
            state.value.unshift({ value: '' });
        };

        const handlePageChange = (curPage: number) => {
            state.pageNum = curPage;
            getListValue();
        };

        return {
            ...toRefs(state),
            saveValue,
            handlePageChange,
            cancel,
            lset,
            onAddListValue,
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