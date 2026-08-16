<template>
    <div>
        <div ref="viewportRef" class="viewport" :style="{ width: state.size.width + 'px', height: state.size.height + 'px' }">
            <div ref="displayRef" class="display" tabindex="0" />
            <div class="btn-box">
                <SvgIcon name="DocumentCopy" @click="openPaste" :size="20" class="pointer-icon mr-2" :title="$t('components.terminal-rdp.clipboard')" />
                <SvgIcon name="FolderOpened" @click="openFilesystem" :size="20" class="pointer-icon mr-2" :title="$t('components.terminal-rdp.fileManager')" />
                <SvgIcon name="FullScreen" @click="state.fullscreen ? closeFullScreen() : openFullScreen()" :size="20" class="pointer-icon mr-2" :title="$t('components.terminal.fullScreenTitle')" />

                <el-dropdown>
                    <SvgIcon name="Monitor" :size="20" class="pointer-icon mr-2" :title="$t('components.terminal-rdp.sendShortcut')" style="color: #fff" />
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

                <SvgIcon name="Refresh" @click="connect(0, 0)" :size="20" class="pointer-icon mr-2" :title="$t('components.terminal.reConnTips')" />
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
    </div>
</template>

<script lang="ts" setup>
import Guacamole from './guac/guacamole-common';
import { getMachineRdpSocketUrl } from '@/views/ops/machine/api';
import clipboard from './guac/clipboard';
import { onUnmounted, reactive, ref } from 'vue';
import { TerminalStatus } from '@/components/terminal/common';
import ClipboardDialog from '@/components/terminal-rdp/guac/ClipboardDialog.vue';
import { TerminalExpose } from '@/components/terminal-rdp/index';
import SvgIcon from '@/components/svg-icon/index.vue';
import MachineFile from '@/views/ops/machine/file/MachineFile.vue';
import { exitFullscreen, launchIntoFullscreen, unWatchFullscreenChange, watchFullscreenChange } from '@/components/terminal-rdp/guac/screen';
import { useDebounceFn, useEventListener } from '@vueuse/core';
import { ClientState, TunnelState } from '@/components/terminal-rdp/guac/states';
import { Msg } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';
import { joinClientParams } from '@/common/request';
import { MachineProtocolEnum } from '@/views/ops/machine/enums';

// Guacamole JS 库最小接口定义
interface GuacClient {
    connect: (queryString: string) => void;
    disconnect: () => void;
    sendKeyEvent: (pressed: number, keysym: number) => void;
    sendSize: (width: number, height: number) => void;
    sendMouseState: (state: GuacMouseState) => void;
    getDisplay: () => GuacDisplay;
    createArgumentValueStream: (mimetype: string, name: string) => GuacStream;
    onclipboard?: (stream: GuacStream, mimetype: string) => void;
    onstatechange?: (state: number) => void;
    onerror?: (status: { message: string }) => void;
    onsync?: () => void;
    onargv?: (stream: GuacStream, mimetype: string, name: string) => void;
    currentState: number;
}
interface GuacDisplay {
    getElement: () => HTMLElement;
    showCursor: (show: boolean) => void;
    scale: (s: number) => void;
    resize: (w: number, h: number) => void;
    getWidth: () => number;
    getHeight: () => number;
    getScale: () => number;
}
interface GuacMouseState {
    x: number;
    y: number;
    left: boolean;
    middle: boolean;
    right: boolean;
    up: boolean;
    down: boolean;
}
interface GuacTunnel {
    onerror: (status: { message: string }) => void;
    onstatechange: (state: number) => void;
}
interface GuacStream {
    onack: (status: { isError: () => boolean }) => void;
}
interface GuacStringReader {
    ontext: (text: string) => void;
    onend: () => void;
}
interface GuacKeyboard {
    onkeydown: (keysym: number) => void;
    onkeyup: (keysym: number) => void;
}
interface GuacMouse {
    onmousedown: (state: GuacMouseState, showCursor?: boolean) => void;
    onmouseup: (state: GuacMouseState, showCursor?: boolean) => void;
    onmousemove: (state: GuacMouseState, showCursor?: boolean) => void;
    onmouseout: () => void;
}
interface GuacTouchpad {
    onmousedown: (state: GuacMouseState, showCursor?: boolean) => void;
    onmouseup: (state: GuacMouseState, showCursor?: boolean) => void;
    onmousemove: (state: GuacMouseState, showCursor?: boolean) => void;
}
interface GuacClipboard {
    onClipboard: (data: string) => void;
    setText: (text: string) => void;
}

const { t } = useI18n();

const viewportRef = ref<HTMLElement | null>(null);
const displayRef = ref<HTMLElement | null>(null);
const clipboardRef = ref<InstanceType<typeof ClipboardDialog> | null>(null);

const props = withDefaults(
    defineProps<{
        machineId: number;
        authCert: string;
        protocol?: number;
        clipboardList?: unknown[];
    }>(),
    { protocol: 2, clipboardList: () => [] }
);

const emit = defineEmits(['statusChange']);

const state = reactive({
    client: null as GuacClient | null,
    display: null as GuacDisplay | null,
    displayElm: {} as HTMLElement,
    clipboard: {} as GuacClipboard,
    keyboard: null as GuacKeyboard | null,
    mouse: null as GuacMouse | null,
    touchpad: null as GuacTouchpad | null,
    errorMessage: '',
    arguments: {} as Record<string, unknown>,
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
    if (!state.keyboard) {
        return;
    }
    state.keyboard.onkeydown = (keysym: number) => {
        state.client?.sendKeyEvent(1, keysym);
    };
    state.keyboard.onkeyup = (keysym: number) => {
        state.client?.sendKeyEvent(0, keysym);
    };
};
const uninstallKeyboard = () => {
    if (state.keyboard) {
        state.keyboard.onkeydown = state.keyboard.onkeyup = () => {};
    }
};

const installMouse = () => {
    state.mouse = new Guacamole.Mouse(state.displayElm);
    if (!state.mouse) {
        return;
    }
    // Hide software cursor when mouse leaves display
    state.mouse.onmouseout = () => {
        if (!state.display) return;
        state.display.showCursor(false);
    };
    state.mouse.onmousedown = state.mouse.onmouseup = state.mouse.onmousemove = handleMouseState;
};

const installTouchpad = () => {
    state.touchpad = new Guacamole.Mouse.Touchpad(state.displayElm);
    if (!state.touchpad) {
        return;
    }

    state.touchpad.onmousedown =
        state.touchpad.onmouseup =
        state.touchpad.onmousemove =
            (st: GuacMouseState) => {
                // 记录按下时，光标所在位置
                handleMouseState(st, true);
            };

    // 记录单指按压时候手在屏幕的位置
    state.displayElm.ontouchend = (event: TouchEvent) => {
        state.displayElm.ontouchend = () => {};
    };
};

const setClipboard = (data: string) => {
    clipboardRef.value?.setValue(data);
};

const installClipboard = () => {
    state.enableClipboard = clipboard.install(state.client) as boolean;
    clipboard.installWatcher(props.clipboardList, setClipboard);
    if (state.client) {
        state.client.onclipboard = clipboard.onClipboard;
    }
};

const installResize = () => {
    // 在 resize 事件结束后 300 毫秒执行，使用防抖
    useEventListener('resize', useDebounceFn(resize, 300));
};

const installDisplay = () => {
    if (!state.client) {
        return;
    }
    let { width, height, force } = state.size;
    state.display = state.client.getDisplay();
    const displayElm = displayRef.value;
    if (!displayElm) {
        return;
    }
    displayElm.appendChild(state.display.getElement());
    displayElm.addEventListener('contextmenu', (e: Event) => {
        e.stopPropagation();
        if (e.preventDefault) {
            e.preventDefault();
        }
        e.returnValue = false;
    });
    state.client.connect('width=' + width + '&height=' + height + '&force=' + force + '&' + joinClientParams());
    window.onunload = () => state.client?.disconnect();

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
    let tunnel = new Guacamole.WebSocketTunnel(getMachineRdpSocketUrl(props.authCert)) as GuacTunnel;
    if (state.client) {
        state.display?.scale(0);
        uninstallKeyboard();
        state.client.disconnect();
    }

    const client: GuacClient = new Guacamole.Client(tunnel);
    state.client = client;

    tunnel.onerror = (status: { message: string }) => {
        console.error(`Tunnel failed ${JSON.stringify(status)}`);
        // state.connectionState = states.TUNNEL_ERROR;
    };

    tunnel.onstatechange = (st: number) => {
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

    client.onstatechange = (clientState: number) => {
        switch (clientState) {
            case ClientState.IDLE:
                break;
            case ClientState.CONNECTING:
                break;
            case ClientState.WAITING:
                break;
            case ClientState.CONNECTED:
                break;
            case ClientState.DISCONNECTING:
                break;
            case ClientState.DISCONNECTED:
                break;
        }
    };

    client.onerror = (error: { message: string }) => {
        client.disconnect();
        console.error(`Client error ${JSON.stringify(error)}`);
        state.errorMessage = error.message;
        // state.connectionState = states.CLIENT_ERROR;
    };

    client.onsync = () => {};

    client.onargv = (stream: GuacStream, mimetype: string, name: string) => {
        if (mimetype !== 'text/plain') return;

        const reader = new Guacamole.StringReader(stream);

        // Assemble received data into a single string
        let value = '';
        reader.ontext = (text: string) => {
            value += text;
        };

        // Test mutability once stream is finished, storing the current value for the argument only if it is mutable
        reader.onend = () => {
            if (!state.client) {
                return;
            }
            const stream = state.client.createArgumentValueStream('text/plain', name);
            stream.onack = (status: { isError: () => boolean }) => {
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
    if (!box) {
        return;
    }

    state.size.width = box.clientWidth;
    state.size.height = box.clientHeight;

    const width = parseInt(String(box.clientWidth));
    const height = parseInt(String(box.clientHeight));

    // VNC 协议只发送尺寸，不重连；RDP 协议在连接状态下发送尺寸，未连接时重连
    if (state.display && (state.display.getWidth() !== width || state.display.getHeight() !== height)) {
        // VNC 协议（protocol=3）只发送尺寸变化，不触发重连
        if (props.protocol === MachineProtocolEnum.Vnc.value) {
            // VNC: 仅发送尺寸
            state.client?.sendSize(width, height);
        } else {
            // RDP: 未连接时重连，已连接时发送尺寸
            if (state.status !== TerminalStatus.Connected) {
                connect(width, height);
            } else {
                state.client?.sendSize(width, height);
            }
        }
    }
};

const handleMouseState = (mouseState: GuacMouseState, showCursor = false) => {
    if (!state.client || !state.display) {
        return;
    }
    state.client.getDisplay().showCursor(showCursor);

    const scaledMouseState = Object.assign({}, mouseState, {
        x: mouseState.x / state.display.getScale(),
        y: mouseState.y / state.display.getScale(),
    });
    state.client.sendMouseState(scaledMouseState);
};

const connect = (width?: number, height?: number, force = false) => {
    if (!width && !height) {
        if (state.size && state.size.width && state.size.height) {
            width = state.size.width;
            height = state.size.height;
        } else {
            // 获取当前viewportRef宽高
            width = viewportRef.value?.clientWidth;
            height = viewportRef.value?.clientHeight;
        }
    }
    state.size = { width: width ?? 0, height: height ?? 0, force };

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
    state.filesystemDialog.title = t('machine.remoteFileDesktopManage');
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
        connect(viewportRef.value?.clientWidth, viewportRef.value?.clientHeight, false);
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
        state.client.sendKeyEvent(1, Number(keys[i]));
    }
    for (let j = 0; j < keys.length; j++) {
        state.client.sendKeyEvent(0, Number(keys[j]));
    }
    Msg.success('components.terminal-rdp.sendCombinationKeySuccess');
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

onUnmounted(() => {
    disconnect();
});

defineExpose(exposes);
</script>

<style lang="scss" scoped>
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
