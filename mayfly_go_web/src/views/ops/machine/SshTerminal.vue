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
import { toRefs, watch, computed, reactive, defineComponent, onMounted, onBeforeUnmount } from 'vue';

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

        watch(props, (newValue) => {
            state.machineId = newValue.machineId;
            state.cmd = newValue.cmd;
            state.height = newValue.height;
            if (state.machineId) {
                initSocket();
            }
        });

        onMounted(() => {
            state.machineId = props.machineId;
            state.height = props.height;
            state.cmd = props.cmd;
            if (state.machineId) {
                initSocket();
            }
        });

        onBeforeUnmount(() => {
            closeAll();
        });

        const store = useStore();

        // 获取布局配置信息
        const getThemeConfig: any = computed(() => {
            return store.state.themeConfig.themeConfig;
        });

        function initXterm() {
            const term: any = new Terminal({
                fontSize: getThemeConfig.value.terminalFontSize || 15,
                // fontWeight: getThemeConfig.value.terminalFontWeight || 'normal',
                fontFamily: 'JetBrainsMono, Consolas, Menlo, Monaco',
                cursorBlink: true,
                // cursorStyle: 'underline', //光标样式
                disableStdin: false,
                theme: {
                    foreground: getThemeConfig.value.terminalForeground || '#c5c8c6', //字体
                    background: getThemeConfig.value.terminalBackground || '#121212', //背景色
                    cursor: getThemeConfig.value.terminalCursor || '#f0cc09', //设置光标
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
            // 为解决窗体resize方法才会向后端发送列数和行数，所以页面加载时也要触发此方法
            send({
                type: 'resize',
                Cols: parseInt(term.cols),
                Rows: parseInt(term.rows),
            });
            // 如果有初始要执行的命令，则发送执行命令
            if (state.cmd) {
                sendCmd(state.cmd + ' \r');
            }
        }

        function initSocket() {
            state.socket = new WebSocket(`${config.baseWsUrl}/machines/${state.machineId}/terminal?token=${getSession('token')}`);
            // 监听socket连接
            state.socket.onopen = open;
            // 监听socket错误信息
            state.socket.onerror = error;
            // 监听socket消息
            state.socket.onmessage = getMessage;
            // 发送socket消息
            state.socket.onsend = send;
        }

        function open() {
            console.log('socket连接成功');
            initXterm();
            //开启心跳
            //   this.start();
        }

        function error() {
            console.log('连接错误');
            //重连
            // reconnect();
        }

        function close() {
            if (state.socket) {
                state.socket.close();
                console.log('socket关闭');
            }

            //重连
            //   this.reconnect()
        }

        function getMessage(msg: string) {
            //   console.log(msg)
            state.term.write(msg['data']);
            //msg是返回的数据
            //   msg = JSON.parse(msg.data);
            //   this.socket.send("ping");//有事没事ping一下，看看ws还活着没
            //   //switch用于处理返回的数据，根据返回数据的格式去判断
            //   switch (msg["operation"]) {
            //     case "stdout":
            //       this.term.write(msg["data"]);//这里write也许不是固定的，失败后找后端看一下该怎么往term里面write
            //       break;
            //     default:
            //       console.error("Unexpected message type:", msg);//但是错误是固定的。。。。
            //   }
            //收到服务器信息，心跳重置
            //   this.reset();
        }

        function send(msg: any) {
            state.socket.send(JSON.stringify(msg));
        }

        function sendCmd(key: any) {
            send({
                type: 'cmd',
                msg: key,
            });
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