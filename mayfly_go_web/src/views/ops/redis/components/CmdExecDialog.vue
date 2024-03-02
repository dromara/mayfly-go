<template>
    <div>
        <el-dialog title="待执行cmd" v-model="dialogVisible" :show-close="false" width="600px" @close="cancel">
            <el-input type="textarea" disabled v-model="state.cmdStr" class="mt5" rows="5" />
            <el-input @keyup.enter="runCmd" ref="remarkInputRef" v-model="remark" placeholder="请输入执行备注" class="mt5" />

            <div v-if="props.flowProcdefKey">
                <el-divider content-position="left">审批节点</el-divider>
                <procdef-tasks :procdef-key="props.flowProcdefKey" />
            </div>

            <template #footer>
                <span class="dialog-footer">
                    <el-button @click="cancel">取 消</el-button>
                    <el-button @click="runCmd" type="primary" :loading="btnLoading">执 行</el-button>
                </span>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, ref, reactive, onMounted } from 'vue';
import { ElDialog, ElButton, ElInput, ElMessage, InputInstance, ElDivider } from 'element-plus';

import ProcdefTasks from '@/views/flow/components/ProcdefTasks.vue';
import { redisApi } from '../api';
import { CmdExecProps } from './CmdExecBox';

const props = withDefaults(defineProps<CmdExecProps>(), {});

const remarkInputRef = ref<InputInstance>();
const state = reactive({
    dialogVisible: false,
    flowProcdefKey: '' as any,
    cmdStr: '',
    remark: '',
    btnLoading: false,
});

const { dialogVisible, remark, btnLoading } = toRefs(state);

onMounted(() => {
    show(props);
});

const show = (props: CmdExecProps) => {
    const cmdArr = props.cmd.map((item: any, index: number) => {
        if (index === 0) {
            return item; // 第一个元素直接返回原值
        }
        if (typeof item === 'string') {
            return `'${item}'`; // 字符串加单引号
        }
        return item; // 其他类型直接返回
    });
    state.cmdStr = cmdArr.join('  ');

    state.dialogVisible = props.visible || true;
    setTimeout(() => {
        remarkInputRef.value?.focus();
    }, 200);
};

/**
 * 执行cmd
 */
const runCmd = async () => {
    if (!state.remark) {
        ElMessage.error('请输入执行备注信息');
        return;
    }

    try {
        state.btnLoading = true;
        await redisApi.runCmd.request({
            id: props.id,
            db: props.db,
            cmd: props.cmd,
            remark: state.remark,
        });
        props.runSuccessFn && props.runSuccessFn();
        ElMessage.success('工单提交成功');
    } finally {
        state.btnLoading = false;
        cancel();
    }
};

const cancel = () => {
    state.dialogVisible = false;
    props.cancelFn && props.cancelFn();
};
</script>
<style lang="scss"></style>
