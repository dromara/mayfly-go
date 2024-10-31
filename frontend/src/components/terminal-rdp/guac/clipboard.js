import Guacamole from './guacamole-common';
import { ElMessage } from 'element-plus';

const clipboard = {};

clipboard.install = (client) => {
    if (!navigator.clipboard) {
        return false;
    }

    clipboard.getLocalClipboard().then((data) => (clipboard.cache = data));

    window.addEventListener('load', clipboard.update(client), true);
    window.addEventListener('copy', clipboard.update(client));
    window.addEventListener('cut', clipboard.update(client));
    window.addEventListener(
        'focus',
        (e) => {
            if (e.target === window) {
                clipboard.update(client)();
            }
        },
        true
    );

    return true;
};

clipboard.update = (client) => {
    return () => {
        clipboard.getLocalClipboard().then((data) => {
            clipboard.cache = data;
            clipboard.setRemoteClipboard(client);
        });
    };
};

clipboard.sendRemoteClipboard = (client, text) => {
    clipboard.cache = {
        type: 'text/plain',
        data: text,
    };

    clipboard.setRemoteClipboard(client);
};

clipboard.setRemoteClipboard = (client) => {
    if (!clipboard.cache) {
        return;
    }

    let writer;

    const stream = client.createClipboardStream(clipboard.cache.type);

    if (typeof clipboard.cache.data === 'string') {
        writer = new Guacamole.StringWriter(stream);
        writer.sendText(clipboard.cache.data);
        writer.sendEnd();

        clipboard.appendClipboardList('up', clipboard.cache.data);
    } else {
        writer = new Guacamole.BlobWriter(stream);
        writer.oncomplete = function clipboardSent() {
            writer.sendEnd();
        };
        writer.sendBlob(clipboard.cache.data);
    }
};

clipboard.getLocalClipboard = async () => {
    // 获取本地剪贴板数据
    if (navigator.clipboard && navigator.clipboard.readText) {
        const text = await navigator.clipboard.readText();
        return {
            type: 'text/plain',
            data: text,
        };
    } else {
        ElMessage.warning('只有https才可以访问剪贴板');
    }
};

clipboard.setLocalClipboard = async (data) => {
    if (data.type === 'text/plain') {
        if (navigator.clipboard && navigator.clipboard.writeText) {
            await navigator.clipboard.writeText(data.data);
        }
    }
};

// 获取到远程服务器剪贴板变动
clipboard.onClipboard = (stream, mimetype) => {
    let reader;

    if (/^text\//.exec(mimetype)) {
        reader = new Guacamole.StringReader(stream);

        // Assemble received data into a single string
        let data = '';
        reader.ontext = (text) => {
            data += text;
        };

        // Set clipboard contents once stream is finished
        reader.onend = () => {
            clipboard.setLocalClipboard({
                type: mimetype,
                data: data,
            });

            clipboard.setClipboardFn && typeof clipboard.setClipboardFn === 'function' && clipboard.setClipboardFn(data);

            clipboard.appendClipboardList('down', data);
        };
    } else {
        reader = new Guacamole.BlobReader(stream, mimetype);
        reader.onend = () => {
            clipboard.setLocalClipboard({
                type: mimetype,
                data: reader.getBlob(),
            });
        };
    }
};

/***
 * 注册剪贴板监听器，如果有本地或远程剪贴板变动，则会更新剪贴板列表
 */
clipboard.installWatcher = (clipboardList, setClipboardFn) => {
    clipboard.clipboardList = clipboardList;
    clipboard.setClipboardFn = setClipboardFn;
};

clipboard.appendClipboardList = (src, data) => {
    clipboard.clipboardList = clipboard.clipboardList || [];
    // 循环判断是否重复
    for (let i = 0; i < clipboard.clipboardList.length; i++) {
        if (clipboard.clipboardList[i].data === data) {
            return;
        }
    }

    clipboard.clipboardList.push({ type: 'text/plain', data, src });
};

export default clipboard;
