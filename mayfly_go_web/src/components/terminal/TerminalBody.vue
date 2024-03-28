<template>
    <div id="terminal-body" :style="{ height }">
        <div ref="terminalRef" class="terminal" />

        <TerminalSearch ref="terminalSearchRef" :search-addon="state.addon.search" @close="focus" />
    </div>
</template>

<script lang="ts" setup>
import 'xterm/css/xterm.css';
import { Terminal, ITheme } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import { SearchAddon } from 'xterm-addon-search';
import { WebLinksAddon } from 'xterm-addon-web-links';

import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import { ref, nextTick, reactive, onMounted, onBeforeUnmount, watch } from 'vue';
import TerminalSearch from './TerminalSearch.vue';
import { debounce } from 'lodash';
import { TerminalStatus } from './common';
import { useEventListener } from '@vueuse/core';
import themes from './themes';

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
    /**
     * 高度
     */
    height: {
        type: [String, Number],
        default: '100%',
    },
});

const emit = defineEmits(['statusChange']);

const terminalRef: any = ref(null);
const terminalSearchRef: any = ref(null);

const { themeConfig } = storeToRefs(useThemeConfig());

// 终端实例
let term: Terminal;
let socket: WebSocket;
let pingInterval: any;

const state = reactive({
    // 插件
    addon: {
        fit: null as any,
        search: null as any,
        weblinks: null as any,
    },
    status: TerminalStatus.NoConnected,
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
        term.options.theme = getTerminalTheme();
    }
);

onBeforeUnmount(() => {
    close();
});

function init() {
    if (term) {
        console.log('重新连接...');
        close();
    }
    nextTick(() => {
        initTerm();
        initSocket();
    });
}

function initTerm() {
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
    useEventListener('resize', debounce(fitTerminal, 400));

    // 注册搜索组件
    const searchAddon = new SearchAddon();
    state.addon.search = searchAddon;
    term.loadAddon(searchAddon);

    // 注册 url link组件
    const weblinks = new WebLinksAddon();
    state.addon.weblinks = weblinks;
    term.loadAddon(weblinks);

    // 注册自定义快捷键
    term.attachCustomKeyEventHandler((event: KeyboardEvent) => {
        // 注册搜索键 ctrl + f
        if (event.key === 'f' && (event.ctrlKey || event.metaKey) && event.type === 'keydown') {
            event.preventDefault();
            terminalSearchRef.value.open();
        }

        return true;
    });
}

function initSocket() {
    if (!props.socketUrl) {
        return;
    }

    socket = new WebSocket(`${props.socketUrl}&rows=${term?.rows}&cols=${term?.cols}`);
    // 监听socket连接
    socket.onopen = () => {
        // 注册心跳
        pingInterval = setInterval(sendPing, 15000);
        state.status = TerminalStatus.Connected;

        // 注册 terminal 事件
        term.onResize((event) => sendResize(event.cols, event.rows));
        term.onData((event) => sendCmd(event));

        focus();

        // 如果有初始要执行的命令，则发送执行命令
        if (props.cmd) {
            sendCmd(props.cmd + ' \r');
        }
    };

    // 监听socket错误信息
    socket.onerror = (e: Event) => {
        term.writeln('\r\n\x1b[31m提示: 连接错误...');
        state.status = TerminalStatus.Error;
        console.log('连接错误', e);
    };

    socket.onclose = (e: CloseEvent) => {
        console.log('terminal socket close...', e.reason);
        state.status = TerminalStatus.Disconnected;
    };

    // 监听socket消息
    socket.onmessage = (msg: any) => {
        // msg.data是真正后端返回的数据
        write2Term(msg.data);
    };
}

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
        return themes[terminalTheme];
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

const send = (msg: any) => {
    state.status == TerminalStatus.Connected && socket?.send(msg);
};

const sendResize = (cols: number, rows: number) => {
    send(`${MsgType.Resize}|${rows}|${cols}`);
};

const sendPing = () => {
    send(`${MsgType.Ping}|ping`);
};

function sendCmd(key: any) {
    send(`${MsgType.Data}|${key}`);
}

function closeSocket() {
    // 关闭 websocket
    socket && socket.readyState === 1 && socket.close();
    // 清除 ping
    pingInterval && clearInterval(pingInterval);
}

function close() {
    console.log('in terminal body close');
    closeSocket();
    if (term) {
        state.addon.search.dispose();
        state.addon.fit.dispose();
        state.addon.weblinks.dispose();
        term.dispose();
    }
}

const getStatus = (): TerminalStatus => {
    return state.status;
};

defineExpose({ init, fitTerminal, focus, clear, close, getStatus, sendResize, write2Term, writeln2Term });
</script>
<style lang="scss">
#terminal-body {
    width: 100%;

    .terminal {
        width: 100%;
        height: 100%;

        // .xterm .xterm-viewport {
        //     overflow-y: hidden;
        // }
    }
}
</style>
