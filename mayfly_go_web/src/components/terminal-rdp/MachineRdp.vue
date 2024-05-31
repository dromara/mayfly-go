<template>
    <div>
        <div ref="viewportRef" class="viewport" :style="{ width: state.size.width + 'px', height: state.size.height + 'px' }">
            <div ref="displayRef" class="display" tabindex="0" />
            <div class="btn-box">
                <SvgIcon name="DocumentCopy" @click="openPaste" :size="20" class="pointer-icon mr10" title="剪贴板" />
                <SvgIcon name="FolderOpened" @click="openFilesystem" :size="20" class="pointer-icon mr10" title="文件管理" />
                <SvgIcon name="FullScreen" @click="state.fullscreen ? closeFullScreen() : openFullScreen()" :size="20" class="pointer-icon mr10" title="全屏" />

                <el-dropdown>
                    <SvgIcon name="Monitor" :size="20" class="pointer-icon mr10" title="发送快捷键" style="color: #fff" />
                    <template #dropdown>
                        <el-dropdown-menu>
                            <el-dropdown-item @click="openSendKeyboard(['65507', '65513', '65535'])"> Ctrl + Alt + Delete </el-dropdown-item>
                            <el-dropdown-item @click="openSendKeyboard(['65507', '65513', '65288'])"> Ctrl + Alt + Backspace </el-dropdown-item>
                            <el-dropdown-item @click="openSendKeyboard(['65515', '100'])"> Windows + D </el-dropdown-item>
                            <el-dropdown-item @click="openSendKeyboard(['65515', '101'])"> Windows + E </el-dropdown-item>
                            <el-dropdown-item @click="openSendKeyboard(['65515', '114'])"> Windows + R </el-dropdown-item>
                            <el-dropdown-item @click="openSendKeyboard(['65515'])"> Windows </el-dropdown-item>
                        </el-dropdown-menu>
                    </template>
                </el-dropdown>

                <SvgIcon name="Refresh" @click="connect(0, 0)" :size="20" class="pointer-icon mr10" title="重新连接" />
            </div>
            <clipboard-dialog ref="clipboardRef" v-model:visible="state.clipboardDialog.visible" @close="closePaste" @submit="onsubmitClipboard" />

            <el-dialog
                v-if="!state.fullscreen"
                destroy-on-close
                :title="state.filesystemDialog.title"
                v-model="state.filesystemDialog.visible"
                :close-on-click-modal="false"
                width="70%"
            >
                <machine-file
                    :machine-id="state.filesystemDialog.machineId"
                    :auth-cert-name="state.filesystemDialog.authCertName"
                    :protocol="state.filesystemDialog.protocol"
                    :file-id="state.filesystemDialog.fileId"
                    :path="state.filesystemDialog.path"
                />
            </el-dialog>
        </div>

        <el-dialog
            v-if="!state.fullscreen"
            destroy-on-close
            :title="state.filesystemDialog.title"
            v-model="state.filesystemDialog.visible"
            :close-on-click-modal="false"
            width="70%"
        >
            <machine-file
                :machine-id="state.filesystemDialog.machineId"
                :auth-cert-name="state.filesystemDialog.authCertName"
                :protocol="state.filesystemDialog.protocol"
                :file-id="state.filesystemDialog.fileId"
                :path="state.filesystemDialog.path"
            />
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import Guacamole from './guac/guacamole-common';
import { getMachineRdpSocketUrl } from '@/views/ops/machine/api';
import clipboard from './guac/clipboard';
import { reactive, ref } from 'vue';
import { TerminalStatus } from '@/components/terminal/common';
import ClipboardDialog from '@/components/terminal-rdp/guac/ClipboardDialog.vue';
import { TerminalExpose } from '@/components/terminal-rdp/index';
import SvgIcon from '@/components/svgIcon/index.vue';
import MachineFile from '@/views/ops/machine/file/MachineFile.vue';
import { exitFullscreen, launchIntoFullscreen, unWatchFullscreenChange, watchFullscreenChange } from '@/components/terminal-rdp/guac/screen';
import { useEventListener } from '@vueuse/core';
import { debounce } from 'lodash';
import { ClientState, TunnelState } from '@/components/terminal-rdp/guac/states';
import { ElMessage } from 'element-plus';
import { joinClientParams } from '@/common/request';

const viewportRef = ref({} as any);
const displayRef = ref({} as any);
const clipboardRef = ref({} as any);

const props = defineProps({
    machineId: {
        type: Number,
        required: true,
    },
    authCert: {
        type: String,
        required: true,
    },
    clipboardList: {
        type: Array,
        default: () => [],
    },
});

const emit = defineEmits(['statusChange']);

const state = reactive({
    client: null as any,
    display: null as any,
    displayElm: {} as any,
    clipboard: {} as any,
    keyboard: {} as any,
    mouse: null as any,
    touchpad: null as any,
    errorMessage: '',
    arguments: {},
    status: TerminalStatus.NoConnected,
    size: {
        height: 710,
        width: 1024,
        force: false,
    },
    enableClipboard: true,
    clipboardDialog: {
        visible: false,
    },
    filesystemDialog: {
        visible: false,
        authCertName: '',
        machineId: 0,
        protocol: 1,
        title: '',
        fileId: 0,
        path: '',
    },
    fullscreen: false,
    beforeFullSize: {
        height: 710,
        width: 1024,
    },
});

const installKeyboard = () => {
    state.keyboard = new Guacamole.Keyboard(state.displayElm);
    uninstallKeyboard();
    state.keyboard.onkeydown = (keysym: any) => {
        state.client.sendKeyEvent(1, keysym);
    };
    state.keyboard.onkeyup = (keysym: any) => {
        state.client.sendKeyEvent(0, keysym);
    };
};
const uninstallKeyboard = () => {
    state.keyboard!.onkeydown = state.keyboard!.onkeyup = () => {};
};

const installMouse = () => {
    state.mouse = new Guacamole.Mouse(state.displayElm);
    // Hide software cursor when mouse leaves display
    state.mouse.onmouseout = () => {
        if (!state.display) return;
        state.display.showCursor(false);
    };
    state.mouse.onmousedown = state.mouse.onmouseup = state.mouse.onmousemove = handleMouseState;
};

const installTouchpad = () => {
    state.touchpad = new Guacamole.Mouse.Touchpad(state.displayElm);

    state.touchpad.onmousedown =
        state.touchpad.onmouseup =
        state.touchpad.onmousemove =
            (st: any) => {
                // 记录按下时，光标所在位置
                console.log(st);
                handleMouseState(st, true);
            };

    // 记录单指按压时候手在屏幕的位置
    state.displayElm.ontouchend = (event: TouchEvent) => {
        console.log('end', event);
        state.displayElm.ontouchend = () => {};
    };
};

const setClipboard = (data: string) => {
    clipboardRef.value.setValue(data);
};

const installClipboard = () => {
    state.enableClipboard = clipboard.install(state.client) as any;
    clipboard.installWatcher(props.clipboardList, setClipboard);
    state.client.onclipboard = clipboard.onClipboard;
};

const installResize = () => {
    // 在resize事件结束后300毫秒执行
    useEventListener('resize', debounce(resize, 300));
};

const installDisplay = () => {
    let { width, height, force } = state.size;
    state.display = state.client.getDisplay();
    const displayElm = displayRef.value;
    displayElm.appendChild(state.display.getElement());
    displayElm.addEventListener('contextmenu', (e: any) => {
        e.stopPropagation();
        if (e.preventDefault) {
            e.preventDefault();
        }
        e.returnValue = false;
    });
    state.client.connect('width=' + width + '&height=' + height + '&force=' + force + '&' + joinClientParams());
    window.onunload = () => state.client.disconnect();

    // allows focusing on the display div so that keyboard doesn't always go to session
    displayElm.onclick = () => {
        displayElm.focus();
    };
    displayElm.onfocus = () => {
        displayElm.className = 'focus';
    };
    displayElm.onblur = () => {
        displayElm.className = '';
    };

    state.displayElm = displayElm;
};

const installClient = () => {
    let tunnel = new Guacamole.WebSocketTunnel(getMachineRdpSocketUrl(props.authCert)) as any;
    if (state.client) {
        state.display?.scale(0);
        uninstallKeyboard();
        state.client.disconnect();
    }

    state.client = new Guacamole.Client(tunnel);

    tunnel.onerror = (status: any) => {
        // eslint-disable-next-line no-console
        console.error(`Tunnel failed ${JSON.stringify(status)}`);
        // state.connectionState = states.TUNNEL_ERROR;
    };

    tunnel.onstatechange = (st: any) => {
        console.log('statechange', st);
        state.status = st;
        switch (st) {
            case TunnelState.CONNECTING: // 'CONNECTING'
                break;
            case TunnelState.OPEN: // 'OPEN'
                state.status = TerminalStatus.Connected;
                emit('statusChange', TerminalStatus.Connected);
                break;
            case TunnelState.CLOSED: // 'CLOSED'
                state.status = TerminalStatus.Disconnected;
                emit('statusChange', TerminalStatus.Disconnected);
                break;
            case TunnelState.UNSTABLE: // 'UNSTABLE'
                state.status = TerminalStatus.Error;
                emit('statusChange', TerminalStatus.Error);
                break;
        }
    };

    state.client.onstatechange = (clientState: any) => {
        console.log('clientState', clientState);
        switch (clientState) {
            case ClientState.IDLE:
                console.log('连接空闲');
                break;
            case ClientState.CONNECTING:
                console.log('连接中...');
                break;
            case ClientState.WAITING:
                console.log('等待服务器响应...');
                break;
            case ClientState.CONNECTED:
                console.log('连接成功...');
                break;
            // eslint-disable-next-line no-fallthrough
            case ClientState.DISCONNECTING:
                console.log('断开连接中...');
                break;
            case ClientState.DISCONNECTED:
                console.log('已断开连接...');
                break;
        }
    };

    state.client.onerror = (error: any) => {
        state.client.disconnect();
        console.error(`Client error ${JSON.stringify(error)}`);
        state.errorMessage = error.message;
        // state.connectionState = states.CLIENT_ERROR;
    };

    state.client.onsync = () => {};

    state.client.onargv = (stream: any, mimetype: any, name: any) => {
        if (mimetype !== 'text/plain') return;

        const reader = new Guacamole.StringReader(stream);

        // Assemble received data into a single string
        let value = '';
        reader.ontext = (text: any) => {
            value += text;
        };

        // Test mutability once stream is finished, storing the current value for the argument only if it is mutable
        reader.onend = () => {
            const stream = state.client.createArgumentValueStream('text/plain', name);
            stream.onack = (status: any) => {
                if (status.isError()) {
                    // ignore reject
                    return;
                }
                state.arguments[name] = value;
            };
        };
    };
};

const resize = () => {
    const elm = viewportRef.value;
    if (!elm || !elm.offsetWidth) {
        // resize is being called on the hidden window
        return;
    }

    let box = elm.parentElement;

    state.size.width = box.clientWidth;
    state.size.height = box.clientHeight;

    const width = parseInt(String(box.clientWidth));
    const height = parseInt(String(box.clientHeight));

    if (state.display.getWidth() !== width || state.display.getHeight() !== height) {
        if (state.status !== TerminalStatus.Connected) {
            connect(width, height);
        } else {
            state.client.sendSize(width, height);
        }
    }
    // setting timeout so display has time to get the correct size
    // setTimeout(() => {
    //     const scale = Math.min(box.clientWidth / Math.max(state.display.getWidth(), 1), box.clientHeight / Math.max(state.display.getHeight(), 1));
    //     state.display.scale(scale);
    //     console.log(state.size, scale);
    // }, 100);
};

const handleMouseState = (mouseState: any, showCursor = false) => {
    state.client.getDisplay().showCursor(showCursor);

    const scaledMouseState = Object.assign({}, mouseState, {
        x: mouseState.x / state.display.getScale(),
        y: mouseState.y / state.display.getScale(),
    });
    state.client.sendMouseState(scaledMouseState);
};

const connect = (width: number, height: number, force = false) => {
    if (!width && !height) {
        if (state.size && state.size.width && state.size.height) {
            width = state.size.width;
            height = state.size.height;
        } else {
            // 获取当前viewportRef宽高
            width = viewportRef.value.clientWidth;
            height = viewportRef.value.clientHeight;
        }
    }
    state.size = { width, height, force };

    installClient();
    installDisplay();
    installKeyboard();
    installMouse();
    installTouchpad();
    installClipboard();
    installResize();
};

const disconnect = () => {
    uninstallKeyboard();
    state.client?.disconnect();
};

const blur = () => {
    uninstallKeyboard();
};

const focus = () => {};

const openPaste = async () => {
    state.clipboardDialog.visible = true;
};

const closePaste = async () => {
    installKeyboard();
};

const onsubmitClipboard = (val: string) => {
    state.clipboardDialog.visible = false;
    installKeyboard();
    clipboard.sendRemoteClipboard(state.client, val);
};

const openFilesystem = async () => {
    state.filesystemDialog.protocol = 2;
    state.filesystemDialog.machineId = props.machineId;
    state.filesystemDialog.authCertName = props.authCert;
    state.filesystemDialog.fileId = props.machineId;
    state.filesystemDialog.path = '/';
    state.filesystemDialog.title = `远程桌面文件管理`;
    state.filesystemDialog.visible = true;
};

const openFullScreen = function () {
    launchIntoFullscreen(viewportRef.value);
    state.fullscreen = true;

    // 记录原始尺寸
    state.beforeFullSize = {
        width: state.size.width,
        height: state.size.height,
    };

    // 使用新的宽高重新连接
    setTimeout(() => {
        connect(viewportRef.value.clientWidth, viewportRef.value.clientHeight, false);
    }, 500);

    watchFullscreenChange(watchFullscreen);
};

function watchFullscreen(event: Event, isFull: boolean) {
    if (!isFull) {
        closeFullScreen();
    }
}

const closeFullScreen = function () {
    exitFullscreen();

    state.fullscreen = false;

    // 使用新的宽高重新连接
    setTimeout(() => {
        connect(state.beforeFullSize.width, state.beforeFullSize.height, false);
    }, 500);

    // 取消注册esc事件，退出全屏
    unWatchFullscreenChange(watchFullscreen);
};

const openSendKeyboard = (keys: string[]) => {
    if (!state.client) {
        return;
    }
    for (let i = 0; i < keys.length; i++) {
        state.client.sendKeyEvent(1, keys[i]);
    }
    for (let j = 0; j < keys.length; j++) {
        state.client.sendKeyEvent(0, keys[j]);
    }
    ElMessage.success('发送组合键成功');
};

const exposes = {
    connect,
    disconnect,
    init: connect,
    close: disconnect,
    fitTerminal: resize,
    focus,
    blur,
    setRemoteClipboard: onsubmitClipboard,
} as TerminalExpose;

defineExpose(exposes);
</script>

<style lang="scss">
.viewport {
    position: relative;
    width: 1024px;
    min-height: 710px;
    z-index: 1;
}
.display {
    overflow: hidden;
    width: 100%;
    height: 100%;
}
.btn-box {
    position: absolute;
    top: 20px;
    right: 30px;
    padding: 5px 0 5px 10px;
    background: #dddddd4a;
    color: #fff;
    border-radius: 3px;
}
</style>
