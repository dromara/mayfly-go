<template>
    <div class="h-full w-full flex">
        <div ref="terminalRef" class="h-full w-full" :style="{ background: getTerminalTheme().background }" />

        <TerminalSearch ref="terminalSearchRef" :search-addon="state.addon.search" @close="focus" />
    </div>
</template>

<script lang="ts" setup>
import '@xterm/xterm/css/xterm.css';
import { Terminal, ITheme } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { SearchAddon } from '@xterm/addon-search';
import { WebLinksAddon } from '@xterm/addon-web-links';

import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import { ref, nextTick, reactive, onMounted, onBeforeUnmount, watch } from 'vue';
import TerminalSearch from './TerminalSearch.vue';
import { TerminalStatus } from './common';
import { useDebounceFn, useEventListener, useIntervalFn } from '@vueuse/core';
import themes from './themes';
import { TrzszFilter } from 'trzsz';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
    // mounted时，是否执行init方法
    mountInit: {
        type: Boolean,
        default: true,
    },
    /**
     * 初始化执行命令
     */
    cmd: { type: String },
    /**
     * 连接url
     */
    socketUrl: {
        type: String,
    },
});

const emit = defineEmits(['statusChange']);

const terminalRef: any = ref(null);
const terminalSearchRef: any = ref(null);

const { themeConfig } = storeToRefs(useThemeConfig());

// 终端实例
let term: Terminal;
let socket: WebSocket;

const state = reactive({
    // 插件
    addon: {
        fit: null as any,
        search: null as any,
        weblinks: null as any,
    },
    status: -11,
});

onMounted(() => {
    if (props.mountInit) {
        init();
    }
});

watch(
    () => state.status,
    () => {
        emit('statusChange', state.status);
    }
);

// 监听 themeConfig terminalTheme配置的变化
watch(
    () => themeConfig.value.terminalTheme,
    () => {
        if (term) {
            term.options.theme = getTerminalTheme();
        }
    }
);

onBeforeUnmount(() => {
    close();
});

const init = () => {
    state.status = TerminalStatus.NoConnected;
    if (term) {
        console.log('重新连接...');
        close();
    }
    nextTick(() => {
        initTerm();
    });
};

const initTerm = async () => {
    term = new Terminal({
        fontSize: themeConfig.value.terminalFontSize || 15,
        fontWeight: themeConfig.value.terminalFontWeight || 'normal',
        fontFamily: 'JetBrainsMono, monaco, Consolas, Lucida Console, monospace',
        cursorBlink: true,
        disableStdin: false,
        allowProposedApi: true,
        fastScrollModifier: 'ctrl',
        theme: getTerminalTheme(),
    });

    term.open(terminalRef.value);

    // 注册自适应组件
    const fitAddon = new FitAddon();
    state.addon.fit = fitAddon;
    term.loadAddon(fitAddon);
    fitTerminal();
    // 注册窗口大小监听器
    useEventListener('resize', useDebounceFn(fitTerminal, 400));

    initSocket();
    // 注册其他插件
    loadAddon();

    // 注册自定义快捷键
    term.attachCustomKeyEventHandler((event: KeyboardEvent) => {
        // 注册搜索键 ctrl + f
        if (event.key === 'f' && (event.ctrlKey || event.metaKey) && event.type === 'keydown') {
            event.preventDefault();
            terminalSearchRef.value.open();
        }

        return true;
    });
};

const initSocket = () => {
    if (!props.socketUrl) {
        return;
    }
    socket = new WebSocket(`${props.socketUrl}&rows=${term?.rows}&cols=${term?.cols}`);
    // 监听socket连接
    socket.onopen = () => {
        // 注册心跳
        useIntervalFn(sendPing, 15000);

        state.status = TerminalStatus.Connected;

        focus();
        fitTerminal();

        // 如果有初始要执行的命令，则发送执行命令
        if (props.cmd) {
            sendData(props.cmd + ' \r');
        }
    };

    // 监听socket错误信息
    socket.onerror = (e: Event) => {
        term.writeln(`\r\n\x1b[31m${t('components.terminal.connErrMsg')}`);
        state.status = TerminalStatus.Error;
        console.log('连接错误', e);
    };

    socket.onclose = (e: CloseEvent) => {
        console.log('terminal socket close...', e.reason);
        state.status = TerminalStatus.Disconnected;
    };
};

const loadAddon = () => {
    // 注册搜索组件
    const searchAddon = new SearchAddon();
    state.addon.search = searchAddon;
    term.loadAddon(searchAddon);

    // 注册 url link组件
    const weblinks = new WebLinksAddon();
    state.addon.weblinks = weblinks;
    term.loadAddon(weblinks);

    // 注册 trzsz
    // initialize trzsz filter
    const trzsz = new TrzszFilter({
        // write the server output to the terminal
        writeToTerminal: (data: any) => term.write(typeof data === 'string' ? data : new Uint8Array(data)),
        // send the user input to the server
        sendToServer: sendData,
        // the terminal columns
        terminalColumns: term.cols,
        // there is a windows shell
        isWindowsShell: false,
    });

    // let trzsz process the server output
    socket?.addEventListener('message', (e) => trzsz.processServerOutput(e.data));
    // let trzsz process the user input
    term.onData((data) => trzsz.processTerminalInput(data));
    term.onBinary((data) => trzsz.processBinaryInput(data));
    term.onResize((size) => {
        sendResize(size.cols, size.rows);
        // tell trzsz the terminal columns has been changed
        trzsz.setTerminalColumns(size.cols);
    });
    // enable drag files or directories to upload
    terminalRef.value.addEventListener('dragover', (event: Event) => event.preventDefault());
    terminalRef.value.addEventListener('drop', (event: any) => {
        event.preventDefault();
        trzsz
            .uploadFiles(event.dataTransfer.items)
            .then(() => console.log('upload success'))
            .catch((err: any) => console.log(err));
    });
};

// 写入内容至终端
const write2Term = (data: any) => {
    term.write(data);
};

const writeln2Term = (data: any) => {
    term.writeln(data);
};

const getTerminalTheme = () => {
    const terminalTheme = themeConfig.value.terminalTheme;
    // 如果不是自定义主题，则返回内置主题
    if (terminalTheme != 'custom') {
        return (themes as any)[terminalTheme];
    }

    // 自定义主题
    return {
        foreground: themeConfig.value.terminalForeground || '#7e9192', //字体
        background: themeConfig.value.terminalBackground || '#002833', //背景色
        cursor: themeConfig.value.terminalCursor || '#268F81', //设置光标
        // cursorAccent: "red",  // 光标停止颜色
    } as ITheme;
};

// 自适应终端
const fitTerminal = () => {
    state.addon.fit.fit();
};

const focus = () => {
    setTimeout(() => term.focus(), 300);
};

const clear = () => {
    term.clear();
    term.clearSelection();
    term.focus();
};

enum MsgType {
    Resize = 1,
    Data = 2,
    Ping = 3,
}

const send2Socket = (data: any) => {
    state.status == TerminalStatus.Connected && socket?.send(data);
};

const sendResize = (cols: number, rows: number) => {
    send2Socket(`${MsgType.Resize}|${rows}|${cols}`);
};

const sendPing = () => {
    send2Socket(`${MsgType.Ping}|ping`);
};

const sendData = (key: any) => {
    send2Socket(`${MsgType.Data}|${key}`);
};

const closeSocket = () => {
    // 关闭 websocket
    socket && socket.readyState === 1 && socket.close();
};

const close = () => {
    console.log('in terminal body close');
    closeSocket();
    if (term) {
        state.addon.search.dispose();
        state.addon.fit.dispose();
        state.addon.weblinks.dispose();
        term.dispose();
    }
};

const getStatus = (): TerminalStatus => {
    return state.status;
};

defineExpose({ init, fitTerminal, focus, clear, close, getStatus, sendResize, write2Term, writeln2Term });
</script>
<style lang="scss"></style>
