<template>
    <div>
        <el-dialog :title="title" v-model="dvisible" :show-close="false" :before-close="cancel" width="750px" :destroy-on-close="true">
            <el-form ref="configForm" :model="form" label-width="auto">
                <el-form-item prop="name" label="配置项:" required>
                    <el-input v-model="form.name"></el-input>
                </el-form-item>
                <el-form-item prop="key" label="配置key:" required>
                    <el-input :disabled="form.id != null" v-model="form.key"></el-input>
                </el-form-item>

                <el-row style="margin-left: 30px; margin-bottom: 5px">
                    <el-button @click="onAddParam" size="small" type="success">新增配置项</el-button>
                </el-row>
                <el-form-item :key="param" v-for="(param, index) in params" prop="params" :label="`参数${index + 1}`">
                    <el-row>
                        <el-col :span="5">
                            <el-input v-model="param.model" placeholder="model"></el-input>
                        </el-col>
                        <span :span="1">
                            <el-divider direction="vertical" border-style="dashed" />
                        </span>
                        <el-col :span="4">
                            <el-input v-model="param.name" placeholder="字段名"></el-input>
                        </el-col>
                        <span :span="1">
                            <el-divider direction="vertical" border-style="dashed" />
                        </span>
                        <el-col :span="4">
                            <el-input v-model="param.placeholder" placeholder="字段说明"></el-input>
                        </el-col>
                        <span :span="1">
                            <el-divider direction="vertical" border-style="dashed" />
                        </span>
                        <el-col :span="4">
                            <el-input v-model="param.options" placeholder="可选值 ,分割"></el-input>
                        </el-col>
                        <span :span="1">
                            <el-divider direction="vertical" border-style="dashed" />
                        </span>
                        <el-col :span="2">
                            <el-button @click="onDeleteParam(index)" size="small" type="danger">删除</el-button>
                        </el-col>
                    </el-row>
                </el-form-item>
                <!-- <el-form-item prop="value" label="配置值:" required>
                    <el-input v-model="form.value"></el-input>
                </el-form-item> -->
                <el-form-item label="备注:">
                    <el-input v-model="form.remark" type="textarea" :rows="2"></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="btnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, watch } from 'vue';
import { configApi } from '../api';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    data: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['update:visible', 'cancel', 'val-change']);

const configForm: any = ref(null);

const state = reactive({
    dvisible: false,
    params: [] as any,
    form: {
        id: null,
        name: '',
        key: '',
        params: '',
        value: '',
        remark: '',
    },
    btnLoading: false,
});

const { dvisible, params, form, btnLoading } = toRefs(state);

watch(props, (newValue: any) => {
    state.dvisible = newValue.visible;
    if (newValue.data) {
        state.form = { ...newValue.data };
        if (state.form.params) {
            state.params = JSON.parse(state.form.params);
        } else {
            state.params = [];
        }
    } else {
        state.form = {} as any;
        state.params = [];
    }
});

const onAddParam = () => {
    state.params.push({ name: '', model: '', placeholder: '' });
};

const onDeleteParam = (idx: number) => {
    state.params.splice(idx, 1);
};

const cancel = () => {
    // 更新父组件visible prop对应的值为false
    emit('update:visible', false);
    // 若父组件有取消事件，则调用
    emit('cancel');
};

const btnOk = async () => {
    configForm.value.validate(async (valid: boolean) => {
        if (valid) {
            if (state.params) {
                state.form.params = JSON.stringify(state.params);
            }
            await configApi.save.request(state.form);
            emit('val-change', state.form);
            cancel();
            state.btnLoading = true;
            setTimeout(() => {
                state.btnLoading = false;
            }, 1000);
        }
    });
};
</script>
<style lang="scss"></style>
