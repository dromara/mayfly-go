<template>
    <div>
        <el-dialog :title="title" v-model="dialogVisible" :before-close="cancel" :close-on-click-modal="false" width="38%" :destroy-on-close="true">
            <el-form :model="form" ref="mongoForm" :rules="rules" label-width="85px">
                <el-tabs v-model="tabActiveName">
                    <el-tab-pane label="基础信息" name="basic">
                        <el-form-item prop="tagId" label="标签:" required>
                            <tag-select v-model="form.tagId" v-model:tag-path="form.tagPath" style="width: 100%" />
                        </el-form-item>

                        <el-form-item prop="name" label="名称" required>
                            <el-input v-model.trim="form.name" placeholder="请输入名称" auto-complete="off"></el-input>
                        </el-form-item>
                        <el-form-item prop="uri" label="uri" required>
                            <el-input
                                type="textarea"
                                :rows="2"
                                v-model.trim="form.uri"
                                placeholder="形如 mongodb://username:password@host1:port1"
                                auto-complete="off"
                            ></el-input>
                        </el-form-item>
                    </el-tab-pane>

                    <el-tab-pane label="其他配置" name="other">
                        <el-form-item prop="sshTunnelMachineId" label="SSH隧道:">
                            <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
                        </el-form-item>
                    </el-tab-pane>
                </el-tabs>
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
import { toRefs, reactive, watch, ref } from 'vue';
import { mongoApi } from './api';
import { ElMessage } from 'element-plus';
import TagSelect from '../component/TagSelect.vue';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    mongo: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['update:visible', 'cancel', 'val-change']);

const rules = {
    tagId: [
        {
            required: true,
            message: '请选择标签',
            trigger: ['change', 'blur'],
        },
    ],
    name: [
        {
            required: true,
            message: '请输入名称',
            trigger: ['change', 'blur'],
        },
    ],
    uri: [
        {
            required: true,
            message: '请输入mongo uri',
            trigger: ['change', 'blur'],
        },
    ],
};

const mongoForm: any = ref(null);
const state = reactive({
    dialogVisible: false,
    tabActiveName: 'basic',
    form: {
        id: null,
        name: null,
        uri: null,
        sshTunnelMachineId: null as any,
        tagId: null as any,
        tagPath: null as any,
    },
    btnLoading: false,
});

const { dialogVisible, tabActiveName, form, btnLoading } = toRefs(state);

watch(props, async (newValue: any) => {
    state.dialogVisible = newValue.visible;
    if (!state.dialogVisible) {
        return;
    }
    state.tabActiveName = 'basic';
    if (newValue.mongo) {
        state.form = { ...newValue.mongo };
    } else {
        state.form = { db: 0 } as any;
    }
});

const btnOk = async () => {
    mongoForm.value.validate(async (valid: boolean) => {
        if (valid) {
            const reqForm = { ...state.form };
            if (!state.form.sshTunnelMachineId || state.form.sshTunnelMachineId <= 0) {
                reqForm.sshTunnelMachineId = -1;
            }
            // reqForm.uri = await RsaEncrypt(reqForm.uri);
            mongoApi.saveMongo.request(reqForm).then(() => {
                ElMessage.success('保存成功');
                emit('val-change', state.form);
                state.btnLoading = true;
                setTimeout(() => {
                    state.btnLoading = false;
                }, 1000);

                cancel();
            });
        } else {
            ElMessage.error('请正确填写信息');
            return false;
        }
    });
};

const cancel = () => {
    emit('update:visible', false);
    emit('cancel');
};
</script>
<style lang="scss"></style>
