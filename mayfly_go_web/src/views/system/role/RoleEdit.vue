<template>
    <div class="role-dialog">
        <el-dialog :title="title" v-model="dvisible" :show-close="false" :before-close="cancel" width="500px" :destroy-on-close="true">
            <el-form ref="roleForm" :model="form" label-width="auto">
                <el-form-item prop="name" label="角色名称" required>
                    <el-input v-model="form.name" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="code" label="角色code" required>
                    <el-input :disabled="form.id != null" v-model="form.code" placeholder="COMMON开头则为所有账号共有角色" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="status" label="状态" required>
                    <el-select v-model="form.status" placeholder="请选择状态" class="w100">
                        <el-option v-for="item in RoleStatusEnum" :key="item.value" :label="item.label" :value="item.value"> </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="角色描述">
                    <el-input v-model="form.remark" type="textarea" :rows="3" placeholder="请输入角色描述"></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, watch } from 'vue';
import { roleApi } from '../api';
import { RoleStatusEnum } from '../enums';

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

const roleForm: any = ref(null);
const state = reactive({
    dvisible: false,
    form: {
        id: null,
        name: '',
        code: '',
        status: 1,
        remark: '',
    },
});

const { dvisible, form } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveRoleExec } = roleApi.save.useApi(form);

watch(props, (newValue: any) => {
    state.dvisible = newValue.visible;
    if (newValue.data) {
        state.form = { ...newValue.data };
    } else {
        state.form = {} as any;
    }
});

const cancel = () => {
    // 更新父组件visible prop对应的值为false
    emit('update:visible', false);
    // 若父组件有取消事件，则调用
    emit('cancel');
};

const btnOk = async () => {
    roleForm.value.validate(async (valid: boolean) => {
        if (valid) {
            await saveRoleExec();
            emit('val-change', state.form);
            cancel();
        }
    });
};
</script>
<style lang="scss"></style>
