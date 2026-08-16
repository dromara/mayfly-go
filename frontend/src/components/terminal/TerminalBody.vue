<template>
    <div class="h-full w-full flex">
        <div ref="terminalRef" class="h-full w-full" :style="{ background: getTerminalTheme().background }" />

        <TerminalSearch ref="terminalSearchRef" :search-addon="state.addon.search" @close="focus" />

        <!-- 右键菜单 -->
        <Contextmenu ref="contextmenuRef" :dropdown="state.contextmenu.dropdown" :items="state.contextmenu.items" />
    </div>
</template>

<script lang="ts" setup>
import { FitAddon } from '@xterm/addon-fit';
import { SearchAddon } from '@xterm/addon-search';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { FontWeight, ITheme, Terminal } from '@xterm/xterm';
import '@xterm/xterm/css/xterm.css';

import config from '@/common/config';
import { createWebSocket, joinClientParams } from '@/common/request';
import { downloadFile } from '@/common/utils/file';
import { copyToClipboard, pasteFromClipboard } from '@/common/utils/string';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import { Msg } from '@/hooks/useI18n';
import { useThemeConfig } from '@/store/themeConfig';
import { machineApi, uploadFile, uploadFolder } from '@/views/ops/machine/api';
import { useDebounceFn, useEventListener } from '@vueuse/core';
import { ElMessageBox } from 'element-plus';
import { storeToRefs } from 'pinia';
import { nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import TerminalSearch from './TerminalSearch.vue';
import { TerminalStatus } from './common';
import themes from './themes';

const { t } = useI18n();

const props = withDefaults(
    defineProps<{
        /** mounted时，是否执行init方法 */
        mountInit?: boolean;
        /** 初始化执行命令 */
        cmd?: string;
        /** 连接url */
        socketUrl?: string;
        /** 机器ID（用于文件传输） */
        machineId?: number;
        /** 授权凭证名（用于文件传输） */
        authCertName?: string;
        /** 文件ID（用于文件传输） */
        fileId?: number;
        /** 协议类型（用于文件传输） */
        protocol?: number;
    }>(),
    { mountInit: true, machineId: 0, authCertName: '', fileId: 0, protocol: 1 }
);

const emit = defineEmits(['statusChange']);

const terminalRef = ref<HTMLElement | null>(null);
const terminalSearchRef = ref<InstanceType<typeof TerminalSearch> | null>(null);
const contextmenuRef = ref<InstanceType<typeof Contextmenu> | null>(null);

const { themeConfig } = storeToRefs(useThemeConfig());

// 终端实例
let term: Terminal;
let socket: WebSocket;
let heartbeatTimer: ReturnType<typeof setInterval> | null = null;

// 静默模式标志：用于发送不显示的命令（如 pwd）
let silentMode = false;
let silentResolve: ((value: string) => void) | null = null;
let silentBuffer = '';

const state = reactive({
    // 插件
    addon: {
        fit: null as FitAddon | null,
        search: null as SearchAddon | null,
        weblinks: null as WebLinksAddon | null,
    },
    status: -11,
    // 右键菜单
    contextmenu: {
        dropdown: {
            x: 0,
            y: 0,
        },
        items: [] as ContextmenuItem[],
        selectedItem: '',
    },
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
        close();
    }
    nextTick(() => {
        initTerm();
    });
};

const initTerm = async () => {
    term = new Terminal({
        fontSize: themeConfig.value.terminalFontSize || 15,
        fontWeight: (themeConfig.value.terminalFontWeight || 'normal') as FontWeight,
        fontFamily: 'JetBrainsMono, monaco, Consolas, Lucida Console, monospace',
        cursorBlink: true,
        disableStdin: false,
        allowProposedApi: true,
        theme: getTerminalTheme(),
    });

    if (terminalRef.value) {
        term.open(terminalRef.value);
    }

    // 注册自适应组件
    const fitAddon = new FitAddon();
    state.addon.fit = fitAddon;
    term.loadAddon(fitAddon);
    fitTerminal();
    // 注册窗口大小监听器
    useEventListener('resize', useDebounceFn(fitTerminal, 400));

    await initSocket();
    // 注册其他插件
    loadAddon();

    // 注册自定义快捷键
    term.attachCustomKeyEventHandler((event: KeyboardEvent) => {
        // 注册搜索键 ctrl + f
        if (event.key === 'f' && (event.ctrlKey || event.metaKey) && event.type === 'keydown') {
            event.preventDefault();
            terminalSearchRef.value?.open();
        }

        return true;
    });
};

const initSocket = async () => {
    if (!props.socketUrl) {
        return;
    }
    try {
        socket = await createWebSocket(`${props.socketUrl}${props.socketUrl.includes('?') ? '&' : '?'}rows=${term?.rows}&cols=${term?.cols}`);
    } catch (e) {
        term.writeln(`\r\n\x1b[31m${t('components.terminal.connErrMsg')}`);
        state.status = TerminalStatus.Error;
        return;
    }

    // 注册心跳
    startHeartbeat();

    state.status = TerminalStatus.Connected;

    focus();
    fitTerminal();

    // 如果有初始要执行的命令，则发送执行命令
    if (props.cmd) {
        sendData(props.cmd + ' \r');
    }

    socket.onclose = (e: CloseEvent) => {
        state.status = TerminalStatus.Disconnected;
    };

    // 监听 WebSocket 消息，将服务器输出写入终端
    socket.onmessage = (e: MessageEvent) => {
        // 如果是静默模式，捕获输出但不显示
        if (silentMode && silentResolve) {
            silentBuffer += e.data;

            // 使用正则表达式匹配绝对路径
            // 匹配以 / 开头，不包含空格、换行符的连续字符
            const pathMatch = silentBuffer.match(/(\/[\w\-\./_]*)/);

            if (pathMatch && pathMatch[1]) {
                const path = pathMatch[1];
                silentResolve(path);
                silentMode = false;
                silentResolve = null;
                silentBuffer = '';
                return; // 不写入终端
            }

            // 如果缓冲区太大，超时处理
            if (silentBuffer.length > 500) {
                silentResolve('~');
                silentMode = false;
                silentResolve = null;
                silentBuffer = '';
            }

            return; // 不写入终端
        }

        // 正常模式，写入终端显示
        term.write(e.data);
    };
};

const startHeartbeat = () => {
    stopHeartbeat();
    heartbeatTimer = setInterval(() => {
        sendPing();
    }, 10000);
};

const stopHeartbeat = () => {
    if (heartbeatTimer) {
        clearInterval(heartbeatTimer);
        heartbeatTimer = null;
    }
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

    // 注册终端输入事件监听（将用户输入发送到 socket）
    term.onData((data: string) => sendData(data));
    term.onBinary((data: string) => sendData(data));

    // 注册终端大小变化事件
    term.onResize((size: { cols: number; rows: number }) => {
        sendResize(size.cols, size.rows);
    });

    // enable drag files or directories to upload
    terminalRef.value?.addEventListener('dragover', (event: Event) => event.preventDefault());
    terminalRef.value?.addEventListener('drop', (event: DragEvent) => {
        event.preventDefault();
        if (event.dataTransfer) {
            handleFileDrop(event.dataTransfer.items);
        }
    });

    // 添加右键菜单支持文件下载和上传
    setupContextMenu();
};

// 写入内容至终端
const write2Term = (data: string | Uint8Array) => {
    term.write(data);
};

const writeln2Term = (data: string | Uint8Array) => {
    term.writeln(data);
};

const getTerminalTheme = () => {
    const terminalTheme = themeConfig.value.terminalTheme;
    // 如果不是自定义主题，则返回内置主题
    if (terminalTheme != 'custom') {
        return (themes as Record<string, ITheme>)[terminalTheme];
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
    state.addon.fit?.fit();
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

const send2Socket = (data: string | ArrayBuffer) => {
    state.status == TerminalStatus.Connected && socket?.send(data);
};

const sendResize = (cols: number, rows: number) => {
    send2Socket(`${MsgType.Resize}|${rows}|${cols}`);
};

const sendPing = () => {
    send2Socket(`${MsgType.Ping}|ping`);
};

const sendData = (key: string) => {
    send2Socket(`${MsgType.Data}|${key}`);
};

const closeSocket = () => {
    stopHeartbeat();
    // 关闭 websocket
    socket && socket.readyState === 1 && socket.close();
};

// 设置右键菜单
const setupContextMenu = () => {
    terminalRef.value?.addEventListener('contextmenu', async (event: MouseEvent) => {
        event.preventDefault();

        showContextMenu(event, term.getSelection());
    });
};

// 显示组合右键菜单
const showContextMenu = (event: MouseEvent, selectedText: string) => {
    state.contextmenu.selectedItem = selectedText;
    state.contextmenu.dropdown = {
        x: event.clientX,
        y: event.clientY,
    };

    // 始终添加上传文件和上传文件夹按钮
    state.contextmenu.items = [
        new ContextmenuItem('copy', 'common.copy')
            .withIcon('CopyDocument')
            .withHideFunc(() => !selectedText)
            .withOnClick(() => {
                copyToClipboard(selectedText);
            }),
        new ContextmenuItem('paste', 'common.paste').withIcon('Document').withOnClick(async () => {
            let text = '';
            try {
                // 尝试从剪贴板读取
                text = await pasteFromClipboard();
            } catch (error) {
                // 读取失败（非 HTTPS 环境），弹出输入框让用户手动粘贴
                try {
                    const { value: manualText } = await ElMessageBox.prompt(t('components.terminal.pasteManualHint'), t('components.terminal.manualPaste'), {
                        confirmButtonText: t('common.confirm'),
                        cancelButtonText: t('common.cancel'),
                        inputType: 'textarea',
                        inputPlaceholder: t('components.terminal.pasteHere'),
                    });
                    text = manualText;
                } catch {
                    // 用户取消
                }
            }
            if (text) {
                term.paste(text);
                focus();
            }
        }),
        new ContextmenuItem('download', 'components.terminal.downloadSelectedFile')
            .withIcon('Download')
            .withHideFunc(() => !selectedText)
            .withOnClick(() => {
                downloadSelectedFile(state.contextmenu.selectedItem);
            })
            .withPermission('machine:file:upload'),
        new ContextmenuItem('uploadFile', 'components.terminal.uploadFileToCurrentDir')
            .withIcon('Upload')
            .withHideFunc(() => !props.machineId || !props.authCertName)
            .withOnClick(() => {
                triggerFilesUpload();
            })
            .withPermission('machine:file:upload'),
        new ContextmenuItem('uploadFolder', 'components.terminal.uploadFolderToCurrentDir')
            .withIcon('Upload')
            .withHideFunc(() => !props.machineId || !props.authCertName)
            .withOnClick(() => {
                triggerFolderUpload();
            })
            .withPermission('machine:file:upload'),
    ];

    // 打开右键菜单
    contextmenuRef.value?.openContextmenu({});
};

// 下载选中的文件
const downloadSelectedFile = async (filePath: string) => {
    if (!props.machineId || !props.authCertName) {
        Msg.error('components.terminal.downloadFailed', { error: '缺少机器信息' });
        return;
    }

    try {
        // 获取当前路径
        const currentPath = await getCurrentPathOrDefault();

        // 从完整路径中提取文件名
        const filename = filePath.trim().split('/').pop() || filePath.trim();

        // 拼接完整路径
        const fullPath = currentPath.endsWith('/') ? `${currentPath}${filename}` : `${currentPath}/${filename}`;

        // 先验证文件是否存在
        try {
            await machineApi.fileStat.request({
                machineId: props.machineId,
                protocol: props.protocol,
                fileId: props.fileId,
                authCertName: props.authCertName,
                path: fullPath,
            });
        } catch (error: unknown) {
            const msg = error instanceof Error ? error.message : String(error);
            Msg.error('components.terminal.downloadFailed', { error: msg });
            return;
        }

        // 下载文件
        downloadFile(
            `${config.baseApiUrl}/machines/${props.machineId}/files/${props.fileId}/download?path=${encodeURIComponent(fullPath)}&machineId=${props.machineId}&authCertName=${props.authCertName}&fileId=${props.fileId}&protocol=${props.protocol}&${joinClientParams()}`
        );

        Msg.success('components.terminal.startDownload', { file: fullPath });
    } catch (error: unknown) {
        const msg = error instanceof Error ? error.message : String(error);
        Msg.error('components.terminal.downloadFailed', { error: msg });
    }
};

// 触发文件上传
const triggerFilesUpload = () => {
    const input = document.createElement('input');
    input.type = 'file';
    input.multiple = true;
    input.style.display = 'none';

    input.addEventListener('change', () => {
        if (input.files && input.files.length > 0) {
            uploadFilesToCurrentPath(input.files);
        }
        document.body.removeChild(input);
    });

    document.body.appendChild(input);
    input.click();
};

// 触发文件夹上传
const triggerFolderUpload = () => {
    const input = document.createElement('input');
    input.type = 'file';
    input.multiple = true;
    const htmlInput = input as HTMLInputElement & { webkitdirectory: boolean; directory: boolean };
    htmlInput.webkitdirectory = true;
    htmlInput.directory = true;
    input.style.display = 'none';

    input.addEventListener('change', () => {
        if (input.files && input.files.length > 0) {
            uploadFolderToCurrentPath(input.files);
        }
        document.body.removeChild(input);
    });

    document.body.appendChild(input);
    input.click();
};

// 上传文件到当前路径
const uploadFilesToCurrentPath = async (files: FileList) => {
    try {
        // 获取当前路径
        const currentPath = await getCurrentPathOrDefault();

        const file = files[0];

        uploadFile(
            file,
            {
                machineId: props.machineId as number,
                authCertName: props.authCertName as string,
                protocol: props.protocol,
                fileId: props.fileId as number,
                path: currentPath,
                filename: file.name,
            },
            {
                onSuccess: () => {
                    Msg.success('components.terminal.uploadSuccess');
                },
                onError: (error) => {
                    Msg.error('components.terminal.uploadFailed', { error: error.message });
                },
            }
        );
    } catch (error: unknown) {
        const msg = error instanceof Error ? error.message : String(error);
        Msg.error('components.terminal.uploadFailed', { error: msg });
    }
};

// 上传文件夹到当前路径
const uploadFolderToCurrentPath = async (files: FileList) => {
    try {
        // 获取当前路径
        const currentPath = await getCurrentPathOrDefault();

        uploadFolder(
            files,
            {
                machineId: props.machineId as number,
                authCertName: props.authCertName as string,
                protocol: props.protocol,
                fileId: props.fileId as number,
                path: currentPath,
            },
            {
                onSuccess: () => {
                    Msg.success('components.terminal.uploadSuccess');
                },
                onError: (error: Error) => {
                    Msg.error('components.terminal.uploadFailed', { error: error.message });
                },
            }
        );
    } catch (error: unknown) {
        const msg = error instanceof Error ? error.message : String(error);
        Msg.error('components.terminal.uploadFailed', { error: msg });
    }
};

// 获取当前路径（静默模式，失败返回默认值）
const getCurrentPathOrDefault = async (): Promise<string> => {
    try {
        return await getCurrentPath();
    } catch (e) {
        return '~';
    }
};

// 获取当前路径（静默模式，不在终端显示）
const getCurrentPath = (): Promise<string> => {
    return new Promise((resolve, reject) => {
        if (!socket || socket.readyState !== WebSocket.OPEN) {
            reject('WebSocket 未连接');
            return;
        }

        // 设置静默模式
        silentMode = true;
        silentResolve = resolve;
        silentBuffer = '';

        // 发送 pwd 命令（使用 \r 模拟回车）
        sendData('pwd\r');

        // 设置超时，防止永远等待
        setTimeout(() => {
            if (silentMode) {
                silentMode = false;
                silentResolve = null;
                silentBuffer = '';
                resolve('~'); // 超时返回默认路径
            }
        }, 2000); // 2秒超时
    });
};

// 处理文件拖拽上传
const handleFileDrop = async (items: DataTransferItemList) => {
    if (!props.machineId || !props.authCertName) {
        Msg.error('components.terminal.uploadFailed', { error: '缺少机器信息' });
        return;
    }

    const files: File[] = [];
    for (let i = 0; i < items.length; i++) {
        const item = items[i];
        if (item.kind === 'file') {
            const file = item.getAsFile();
            if (file) {
                files.push(file);
            }
        }
    }

    if (files.length > 0) {
        await uploadFilesToCurrentPath(files as unknown as FileList);
    }
};

const close = () => {
    closeSocket();
    if (term) {
        state.addon.search?.dispose();
        state.addon.fit?.dispose();
        state.addon.weblinks?.dispose();
        term.dispose();
    }
};

const getStatus = (): TerminalStatus => {
    return state.status;
};

defineExpose({ init, fitTerminal, focus, clear, close, getStatus, sendResize, write2Term, writeln2Term });
</script>
<style lang="scss" scoped>
// 终端容器样式
</style>
