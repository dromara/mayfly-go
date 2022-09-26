<template>
    <div :style="{ height: height }" id="xterm" class="xterm" />
</template>

<script lang="ts">
import 'xterm/css/xterm.css';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import { getSession } from '@/common/utils/storage.ts';
import config from '@/common/config';
import { useStore } from '@/store/index.ts';
import { nextTick, toRefs, watch, computed, reactive, defineComponent, onMounted, onBeforeUnmount } from 'vue';

export default defineComponent({
    name: 'SshTerminal',
    props: {
        machineId: { type: Number },
        cmd: { type: String },
        height: { type: String },
    },
    setup(props: any) {
        const state = reactive({
            machineId: 0,
            cmd: '',
            height: '',
            term: null as any,
            socket: null as any,
        });

        const resize = 1;
        const data = 2;
        const ping = 3;

        watch(props, (newValue) => {
            state.machineId = newValue.machineId;
            state.cmd = newValue.cmd;
            state.height = newValue.height;
        });

        onMounted(() => {
            state.machineId = props.machineId;
            state.height = props.height;
            state.cmd = props.cmd;
        });

        onBeforeUnmount(() => {
            closeAll();
        });

        const store = useStore();

        // 获取布局配置信息
        const getThemeConfig: any = computed(() => {
            return store.state.themeConfig.themeConfig;
        });

        nextTick(() => {
            initXterm();
            initSocket();
        });

        function initXterm() {
            const term: any = new Terminal({
                fontSize: getThemeConfig.value.terminalFontSize || 15,
                fontWeight: getThemeConfig.value.terminalFontWeight || 'normal',
                fontFamily: 'JetBrainsMono, monaco, Consolas, Lucida Console, monospace',
                cursorBlink: true,
                disableStdin: false,
                theme: {
                    foreground: getThemeConfig.value.terminalForeground || '#7e9192', //字体
                    background: getThemeConfig.value.terminalBackground || '#002833', //背景色
                    cursor: getThemeConfig.value.terminalCursor || '#268F81', //设置光标
                    // cursorAccent: "red",  // 光标停止颜色
                } as any,
            });
            const fitAddon = new FitAddon();
            term.loadAddon(fitAddon);
            term.open(document.getElementById('xterm'));
            fitAddon.fit();
            term.focus();
            state.term = term;

            // 监听窗口resize
            window.addEventListener('resize', () => {
                try {
                    // 窗口大小改变时，触发xterm的resize方法使自适应
                    fitAddon.fit();
                    if (state.term) {
                        state.term.focus();
                        send({
                            type: resize,
                            Cols: parseInt(state.term.cols),
                            Rows: parseInt(state.term.rows),
                        });
                    }
                } catch (e) {
                    console.log(e);
                }
            });

            // / **
            //     *添加事件监听器，用于按下键时的事件。事件值包含
            //     *将在data事件以及DOM事件中发送的字符串
            //     *触发了它。
            //     * @返回一个IDisposable停止监听。
            //  * /
            //   / ** 更新：xterm 4.x（新增）
            //  *为数据事件触发时添加事件侦听器。发生这种情况
            //  *用户输入或粘贴到终端时的示例。事件值
            //  *是`string`结果的结果，在典型的设置中，应该通过
            //  *到支持pty。
            //  * @返回一个IDisposable停止监听。
            //  * /
            // 支持输入与粘贴方法
            term.onData((key: any) => {
                sendCmd(key);
            });
        }

        let pingInterval: any;
        function initSocket() {
            state.socket = new WebSocket(
                `${config.baseWsUrl}/machines/${state.machineId}/terminal?token=${getSession('token')}&cols=${state.term.cols}&rows=${
                    state.term.rows
                }`
            );

            // 监听socket连接
            state.socket.onopen = () => {
                // 如果有初始要执行的命令，则发送执行命令
                if (state.cmd) {
                    sendCmd(state.cmd + ' \r');
                }
                // 开启心跳
                pingInterval = setInterval(() => {
                    send({ type: ping, msg: 'ping' });
                }, 8000);
            };

            // 监听socket错误信息
            state.socket.onerror = (e: any) => {
                console.log('连接错误', e);
            };

            state.socket.onclose = () => {
                if (state.term) {
                    state.term.writeln('\r\n\x1b[31m提示: 连接已关闭...');
                }
                if (pingInterval) {
                    clearInterval(pingInterval);
                }
            };

            // 发送socket消息
            state.socket.onsend = send;

            // 监听socket消息
            state.socket.onmessage = getMessage;
        }

        function getMessage(msg: any) {
            // msg.data是真正后端返回的数据
            state.term.write(msg.data);
        }

        function send(msg: any) {
            state.socket.send(JSON.stringify(msg));
        }

        function sendCmd(key: any) {
            send({
                type: data,
                msg: key,
            });
        }

        function close() {
            if (state.socket) {
                state.socket.close();
                console.log('socket关闭');
            }
        }

        function closeAll() {
            close();
            if (state.term) {
                state.term.dispose();
                state.term = null;
            }
        }

        return {
            ...toRefs(state),
        };
    },
});
</script>