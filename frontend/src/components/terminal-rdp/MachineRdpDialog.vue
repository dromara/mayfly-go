<template>
    <div class="rdpDialog" ref="dialogRef">
        <el-dialog
            v-model="dialogVisible"
            :before-close="handleClose"
            :close-on-click-modal="false"
            :destroy-on-close="true"
            :close-on-press-escape="false"
            :show-close="false"
            width="1024"
            @open="connect()"
        >
            <template #header>
                <div class="terminal-title-wrapper">
                    <!-- 左侧 -->
                    <div class="title-left-fixed">
                        <!-- title信息 -->
                        <div>
                            {{ title }}
                        </div>
                    </div>

                    <!-- 右侧 -->
                    <div class="title-right-fixed">
                        <el-popconfirm @confirm="connect(true)" title="确认重新连接?">
                            <template #reference>
                                <div class="mr-2 cursor-pointer">
                                    <el-tag v-if="state.status == TerminalStatus.Connected" type="success" effect="light" round> 已连接 </el-tag>
                                    <el-tag v-else type="danger" effect="light" round> 未连接，点击重连 </el-tag>
                                </div>
                            </template>
                        </el-popconfirm>

                        <el-popconfirm @confirm="handleClose" title="确认关闭?">
                            <template #reference>
                                <SvgIcon name="Close" class="pointer-icon" title="关闭" :size="20" />
                            </template>
                        </el-popconfirm>
                    </div>
                </div>
            </template>

            <machine-rdp ref="rdpRef" :machine-id="machineId" :auth-cert="authCert" @status-change="handleStatusChange" />
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { reactive, ref, toRefs, watch } from 'vue';
import MachineRdp from '@/components/terminal-rdp/MachineRdp.vue';
import { TerminalStatus } from '@/components/terminal/common';
import SvgIcon from '@/components/svgIcon/index.vue';

const rdpRef = ref({} as any);
const dialogRef = ref({} as any);

const props = defineProps({
    visible: { type: Boolean },
    machineId: {
        type: Number,
        required: true,
    },
    authCert: {
        type: String,
        required: true,
    },
    title: { type: String },
});

const emit = defineEmits(['update:visible', 'cancel', 'update:machineId']);

const state = reactive({
    dialogVisible: false,
    title: '',
    status: TerminalStatus.NoConnected,
});

const { dialogVisible } = toRefs(state);

watch(props, async (newValue: any) => {
    const visible = newValue.visible;
    state.dialogVisible = visible;
    if (visible) {
        state.title = newValue.title;
    }
});

const connect = (force = false) => {
    rdpRef.value?.disconnect();

    let width = 1024;
    let height = 710;
    rdpRef.value?.connect(width, height, force);
};

const handleStatusChange = (status: TerminalStatus) => {
    state.status = status;
};

/**
 * 关闭取消按钮触发的事件
 */
const handleClose = () => {
    emit('update:visible', false);
    emit('update:machineId', null);
    emit('cancel');
    rdpRef.value?.disconnect();
};
</script>
<style lang="scss">
.rdpDialog {
    .el-dialog {
        padding: 0;
        .el-dialog__header {
            padding: 10px;
        }
    }

    .el-overlay .el-overlay-dialog .el-dialog .el-dialog__body {
        max-height: 100% !important;
        padding: 0 !important;
    }

    .terminal-title-wrapper {
        display: flex;
        justify-content: space-between;
        font-size: 16px;

        .title-right-fixed {
            display: flex;
            align-items: center;
            font-size: 20px;
            text-align: end;
        }
    }
}
</style>
