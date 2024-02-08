<template>
    <div>
        <div class="terminal-dialog-container" v-for="openTerminal of terminals" :key="openTerminal.terminalId">
            <el-dialog
                title="终端"
                v-model="openTerminal.visible"
                top="32px"
                class="terminal-dialog"
                width="75%"
                :close-on-click-modal="false"
                :modal="true"
                :show-close="false"
                :fullscreen="openTerminal.fullscreen"
            >
                <template #header>
                    <div class="terminal-title-wrapper">
                        <!-- 左侧 -->
                        <div class="title-left-fixed">
                            <!-- title信息 -->
                            <div>
                                <slot name="headerTitle" :terminalInfo="openTerminal">
                                    {{ openTerminal.headerTitle }}
                                </slot>
                            </div>
                        </div>

                        <!-- 右侧 -->
                        <div class="title-right-fixed">
                            <el-popconfirm @confirm="reConnect(openTerminal.terminalId)" title="确认重新连接?">
                                <template #reference>
                                    <div class="mr15 pointer">
                                        <el-tag v-if="openTerminal.status == TerminalStatus.Connected" type="success" effect="light" round> 已连接 </el-tag>
                                        <el-tag v-else type="danger" effect="light" round> 未连接 </el-tag>
                                    </div>
                                </template>
                            </el-popconfirm>

                            <el-popover placement="bottom" :width="200" trigger="hover">
                                <template #reference>
                                    <SvgIcon name="QuestionFilled" :size="20" class="pointer-icon mr10" />
                                </template>
                                <div>ctrl | command + f (搜索)</div>
                                <div class="mt5">点击连接状态可重连</div>
                            </el-popover>

                            <SvgIcon
                                name="ArrowDown"
                                v-if="props.visibleMinimize"
                                @click="minimize(openTerminal.terminalId)"
                                :size="20"
                                class="pointer-icon mr10"
                                title="最小化"
                            />

                            <SvgIcon name="FullScreen" @click="handlerFullScreen(openTerminal)" :size="20" class="pointer-icon mr10" title="全屏|退出全屏" />

                            <SvgIcon name="Close" class="pointer-icon" @click="close(openTerminal.terminalId)" title="关闭" :size="20" />
                        </div>
                    </div>
                </template>
                <div class="terminal-wrapper" :style="{ height: `calc(100vh - ${openTerminal.fullscreen ? '49px' : '200px'})` }">
                    <TerminalBody
                        @status-change="terminalStatusChange(openTerminal.terminalId, $event)"
                        :ref="(el) => setTerminalRef(el, openTerminal.terminalId)"
                        :cmd="openTerminal.cmd"
                        :socket-url="openTerminal.socketUrl"
                    />
                </div>
            </el-dialog>
        </div>

        <!-- 终端最小化 -->
        <div class="terminal-minimize-container">
            <el-card
                v-for="minimizeTerminal of minimizeTerminals"
                :key="minimizeTerminal.terminalId"
                :class="`terminal-minimize-item pointer ${minimizeTerminal.styleClass}`"
                size="small"
                @click="maximize(minimizeTerminal.terminalId)"
            >
                <el-tooltip :content="minimizeTerminal.desc" placement="top">
                    <span>
                        {{ minimizeTerminal.title }}
                    </span>
                </el-tooltip>

                <!-- 关闭按钮 -->
                <SvgIcon name="CloseBold" @click.stop="closeMinimizeTerminal(minimizeTerminal.terminalId)" class="ml10 pointer-icon fr" :size="20" />
            </el-card>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive } from 'vue';
import TerminalBody from '@/components/terminal/TerminalBody.vue';
import SvgIcon from '@/components/svgIcon/index.vue';
import { TerminalStatus } from './common';

const props = defineProps({
    visibleMinimize: {
        type: Boolean,
        default: false,
    },
});

const emit = defineEmits(['close', 'minimize']);

const openTerminalRefs: any = {};

/**
terminal对象信息:

visible: false,
machineId: null as any,
terminalId: null as any,
machine: {} as any,
fullscreen: false,
 */

const state = reactive({
    terminals: {} as any, // key -> terminalId  value -> terminal
    minimizeTerminals: {} as any, // key -> terminalId  value -> 简易terminal
});

const { terminals, minimizeTerminals } = toRefs(state);

const setTerminalRef = (el: any, terminalId: any) => {
    if (terminalId) {
        openTerminalRefs[terminalId] = el;
    }
};

function open(terminalInfo: any, cmd: string = '') {
    let terminalId = terminalInfo.terminalId;
    if (!terminalId) {
        terminalId = Date.now();
    }
    state.terminals[terminalId] = {
        ...terminalInfo,
        terminalId,
        visible: true,
        cmd,
        status: TerminalStatus.NoConnected,
    };
}

const terminalStatusChange = (terminalId: string, status: TerminalStatus) => {
    const terminal = state.terminals[terminalId];
    if (terminal) {
        terminal.status = status;
    }

    const minTerminal = state.minimizeTerminals[terminalId];
    if (!minTerminal) {
        return;
    }
    minTerminal.styleClass = getTerminalStatysStyleClass(terminalId, status);
};

const getTerminalStatysStyleClass = (terminalId: any, status: any = null) => {
    if (status == null) {
        status = openTerminalRefs[terminalId].getStatus();
    }
    if (status == TerminalStatus.Connected) {
        return 'terminal-status-success';
    }

    if (status == TerminalStatus.NoConnected) {
        return 'terminal-status-no-connect';
    }

    return 'terminal-status-error';
};

const reConnect = (terminalId: any) => {
    openTerminalRefs[terminalId].init();
};

function close(terminalId: any) {
    console.log('in terminal dialog close');
    delete state.terminals[terminalId];

    // 关闭终端，并删除终端ref
    const terminalRef = openTerminalRefs[terminalId];
    terminalRef && terminalRef.close();
    delete openTerminalRefs[terminalId];

    emit('close', terminalId);
}

function minimize(terminalId: number) {
    console.log('in terminal dialog minimize: ', terminalId);

    const terminal = state.terminals[terminalId];
    if (!terminal) {
        console.warn('不存在该终端信息: ', terminalId);
        return;
    }
    terminal.visible = false;

    const minTerminalInfo = {
        terminalId: terminal.terminalId,
        title: terminal.minTitle, // 截取terminalId最后两位区分多个terminal
        desc: terminal.minDesc,
        styleClass: getTerminalStatysStyleClass(terminalId),
    };
    state.minimizeTerminals[terminalId] = minTerminalInfo;

    emit('minimize', minTerminalInfo);
}

function maximize(terminalId: any) {
    console.log('in terminal dialog maximize: ', terminalId);
    const minTerminal = state.minimizeTerminals[terminalId];
    if (!minTerminal) {
        console.log('no min terminal...');
        return;
    }
    delete state.minimizeTerminals[terminalId];

    // 显示终端信息
    state.terminals[terminalId].visible = true;

    const terminalRef = openTerminalRefs[terminalId];
    // fit
    setTimeout(() => {
        terminalRef.fitTerminal();
        terminalRef.focus();
    }, 250);
}

const handlerFullScreen = (terminal: any) => {
    terminal.fullscreen = !terminal.fullscreen;
    const terminalRef = openTerminalRefs[terminal.terminalId];
    // fit
    setTimeout(() => {
        terminalRef?.fitTerminal();
        terminalRef?.focus();
    }, 250);
};

const closeMinimizeTerminal = (terminalId: any) => {
    delete state.minimizeTerminals[terminalId];
    close(terminalId);
};

defineExpose({
    open,
    close,
    minimize,
    maximize,
});
</script>

<style lang="scss">
.terminal-dialog-container {
    .el-dialog__header {
        padding: 10px;
    }

    .el-dialog {
        padding: 1px 1px;
    }

    // 取消body最大高度，否则全屏有问题
    .el-dialog__body {
        max-height: 100% !important;
        overflow: hidden !important;
    }

    .el-overlay .el-overlay-dialog .el-dialog .el-dialog__body {
        padding: 0px !important;
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

.terminal-minimize-container {
    position: absolute;
    right: 16px;
    bottom: 16px;
    z-index: 10;
    display: flex;
    flex-wrap: wrap-reverse;
    justify-content: flex-end;

    .terminal-minimize-item {
        min-width: 120px;
        // box-shadow: 0 3px 4px #dee2e6;
        border-radius: 4px;
        margin: 1px 1px;
    }

    .terminal-status-error {
        box-shadow: 0 3px 4px var(--el-color-danger);
        border-color: var(--el-color-danger);
    }

    .terminal-status-no-connect {
        box-shadow: 0 3px 4px var(--el-color-warning);
        border-color: var(--el-color-warning);
    }

    .terminal-status-success {
        box-shadow: 0 3px 4px var(--el-color-success);
        border-color: var(--el-color-success);
    }

    .el-card__body {
        padding: 15px !important;
    }
}
</style>
