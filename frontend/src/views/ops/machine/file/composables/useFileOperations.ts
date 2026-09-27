import { isTrue, notBlank } from '@/common/assert';
import { downloadFile as downloadByIframe } from '@/common/utils/file';
import { convertToBytes } from '@/common/utils/format';
import { Msg } from '@/hooks/useI18n';
import { useI18nDeleteConfirm } from '@/hooks/useI18n';
import { reactive } from 'vue';
import { useI18n } from 'vue-i18n';
import { buildFileDownloadUrl, machineApi, uploadFile, uploadFolder } from '../../api';
import { DIR_TYPE, FILE_TYPE, PATH_SEP, PROTECTED_PATHS } from '../constants';
import type { FileRowVM, MachineFileInfo } from '../../types';

/** 目录/普通文件判定：类型标记只在此处解释，模板与业务代码不重复比较字面量 */
export const isDir = (file: MachineFileInfo) => file.type === DIR_TYPE;

export const isFile = (file: MachineFileInfo) => file.type === FILE_TYPE;

export interface UseFileOperationsOptions {
    machineId: () => number | undefined;
    authCertName: () => string | undefined;
    fileId: () => number;
    protocol: () => number;
    nowPath: () => string;
    setLoading: (loading: boolean) => void;
    refresh: () => void;
    uploadMaxFileSize: () => string;
}

export function useFileOperations(options: UseFileOperationsOptions) {
    const { t } = useI18n();

    /**
     * 剪贴板（待复制/待移动的路径集）。
     *
     * 必须是 reactive：它直接驱动模板里的粘贴条与按钮文案，用普通对象时 push/清空不会触发重渲染，
     * 表现为「点了复制但粘贴条不出现，要再碰一下表格才冒出来」。
     */
    const copyOrMvFile = reactive({
        paths: [] as string[],
        type: 'cp',
        fromPath: '',
    });

    const isCpFile = () => {
        return copyOrMvFile.type == 'cp';
    };

    const setCopyOrMvFile = (files: MachineFileInfo[], type = 'cp') => {
        for (let file of files) {
            const path = file.path;
            if (!copyOrMvFile.paths.includes(path)) {
                copyOrMvFile.paths.push(path);
            }
        }
        copyOrMvFile.type = type;
        copyOrMvFile.fromPath = options.nowPath();
    };

    const copyFile = (files: MachineFileInfo[]) => {
        setCopyOrMvFile(files);
    };

    const mvFile = (files: MachineFileInfo[]) => {
        setCopyOrMvFile(files, 'mv');
    };

    const pasteFile = async () => {
        isTrue(options.nowPath() != copyOrMvFile.fromPath, 'machine.sameDirNoPaste');
        const api = isCpFile() ? machineApi.cpFile : machineApi.mvFile;
        try {
            options.setLoading(true);
            await api.request({
                machineId: options.machineId(),
                fileId: options.fileId(),
                authCertName: options.authCertName(),
                paths: copyOrMvFile.paths,
                toPath: options.nowPath(),
                protocol: options.protocol(),
            });
            Msg.success('machine.pasteSuccess');
            copyOrMvFile.paths = [];
            options.refresh();
        } finally {
            options.setLoading(false);
        }
    };

    const cancelCopy = () => {
        copyOrMvFile.paths = [];
    };

    const fileRename = async (row: MachineFileInfo, oldname: string) => {
        notBlank(row.name, t('machine.newFileNameNotEmpty'));
        await machineApi.renameFile.request({
            machineId: options.machineId(),
            authCertName: options.authCertName(),
            fileId: options.fileId(),
            path: options.nowPath() + PATH_SEP + oldname,
            newname: options.nowPath() + PATH_SEP + row.name,
            protocol: options.protocol(),
        });
        Msg.success('machine.renameSuccess');
        options.refresh();
    };

    const deleteFile = async (files: MachineFileInfo[]) => {
        let confirmMsg = files.map((x: MachineFileInfo) => `[${x.path}]`).join('\n');
        if (confirmMsg.length > 400) {
            confirmMsg = confirmMsg.substring(0, 400) + '...';
        }
        await useI18nDeleteConfirm(confirmMsg);
        options.setLoading(true);
        try {
            await machineApi.rmFile.request({
                fileId: options.fileId(),
                paths: files.map((x: MachineFileInfo) => x.path),
                machineId: options.machineId(),
                authCertName: options.authCertName(),
                protocol: options.protocol(),
            });
            Msg.deleteSuccess();
            options.refresh();
        } finally {
            options.setLoading(false);
        }
    };

    /**
     * 在当前目录下创建文件/目录
     *
     * 名称在此集中校验（入口不止弹层里的「确定」一个），带分隔符的名称会拼出与预期不同的路径，
     * 在前置拦下比把后端错误透给用户更好定位
     * @returns 创建后的完整路径，供调用方给出可核对的结果提示
     */
    const createFile = async (name: string, type: string) => {
        // notBlank 收已翻译的文案（isTrue 才自己走 i18n），两者传参形式不同
        notBlank(name, t('machine.newFileNameNotEmpty'));
        isTrue(!name.includes(PATH_SEP), 'machine.fileNameNoSeparator');
        const path = options.nowPath() + PATH_SEP + name;
        await machineApi.createFile.request({
            machineId: options.machineId(),
            authCertName: options.authCertName(),
            id: options.fileId(),
            protocol: options.protocol(),
            path,
            type,
        });
        options.refresh();
        return path;
    };

    /**
     * 触发浏览器下载，走全站统一的隐藏 iframe 入口。
     *
     * 自己拼 `a[target=_blank]` 会先开一个新标签页再由浏览器自给关闭（响应是附件下载），
     * 表现为点一下下载整页闪动；镜像导出 / 技能导出 / 终端录制下载都用的是这个入口。
     */
    const downloadFile = (data: MachineFileInfo) => {
        downloadByIframe(
            buildFileDownloadUrl({
                machineId: options.machineId() as number,
                fileId: options.fileId(),
                path: data.path,
                authCertName: options.authCertName(),
                protocol: options.protocol(),
            })
        );
    };

    const getDirSize = async (data: FileRowVM) => {
        try {
            data.loadingDirSize = true;
            const res = await machineApi.dirSize.request({
                machineId: options.machineId(),
                fileId: options.fileId(),
                path: data.path,
                protocol: options.protocol(),
                authCertName: options.authCertName(),
            });
            data.dirSize = res;
        } finally {
            data.loadingDirSize = false;
        }
    };

    const showFileStat = async (data: FileRowVM) => {
        try {
            if (data.stat) {
                return;
            }
            data.loadingStat = true;
            const res = await machineApi.fileStat.request({
                machineId: options.machineId(),
                fileId: options.fileId(),
                path: data.path,
                protocol: options.protocol(),
                authCertName: options.authCertName(),
            });
            data.stat = res;
        } finally {
            data.loadingStat = false;
        }
    };

    const checkUploadFileSize = (fileSize: number) => {
        const bytes = convertToBytes(options.uploadMaxFileSize());
        if (fileSize > bytes) {
            Msg.error('machine.fileExceedsSysConf', { uploadMaxFileSize: options.uploadMaxFileSize() });
            return false;
        }
        return true;
    };

    const handleFileUpload = (content: { file: File }) => {
        const file = content.file;
        const path = options.nowPath();

        if (!checkUploadFileSize(file.size)) {
            return;
        }

        uploadFile(
            file,
            {
                machineId: options.machineId() as number,
                authCertName: options.authCertName() as string,
                protocol: options.protocol(),
                fileId: options.fileId(),
                path: path,
                filename: file.name,
            },
            {
                onSuccess: () => {
                    Msg.success('machine.uploadSuccess');
                    setTimeout(() => {
                        options.refresh();
                    }, 1000);
                },
                onError: (error: Error) => {
                    Msg.error(error.message);
                },
            }
        );
    };

    const handleFolderUpload = (files: FileList) => {
        if (!files || files.length === 0) {
            return;
        }

        let totalFileSize = 0;
        for (let file of files) {
            totalFileSize += file.size;
        }

        if (!checkUploadFileSize(totalFileSize)) {
            return;
        }

        uploadFolder(
            files,
            {
                machineId: options.machineId() as number,
                authCertName: options.authCertName() as string,
                protocol: options.protocol(),
                fileId: options.fileId(),
                path: options.nowPath(),
            },
            {
                onSuccess: () => {
                    Msg.success('machine.uploadSuccess');
                    setTimeout(() => {
                        options.refresh();
                    }, 1000);
                },
                onError: (error) => {
                    Msg.error(error.message);
                },
            }
        );
    };

    const dontOperate = (data: MachineFileInfo) => PROTECTED_PATHS.has(data.path);

    return {
        copyOrMvFile,
        isCpFile,
        copyFile,
        mvFile,
        pasteFile,
        cancelCopy,
        fileRename,
        deleteFile,
        createFile,
        downloadFile,
        getDirSize,
        showFileStat,
        checkUploadFileSize,
        handleFileUpload,
        handleFolderUpload,
        dontOperate,
        isDir,
        isFile,
    };
}
