import { isTrue, notBlank } from '@/common/assert';
import config from '@/common/config';
import { joinClientParams } from '@/common/request';
import { convertToBytes } from '@/common/utils/format';
import { Msg } from '@/hooks/useI18n';
import { useI18nDeleteConfirm } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';
import { machineApi, uploadFile, uploadFolder } from '../../api';
import type { MachineFileInfo } from '../../types';

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
    const pathSep = '/';

    const copyOrMvFile = {
        paths: [] as string[],
        type: 'cp',
        fromPath: '',
    };

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
            machineId: parseInt(options.machineId() + ''),
            authCertName: options.authCertName(),
            fileId: parseInt(options.fileId() + ''),
            path: options.nowPath() + pathSep + oldname,
            newname: options.nowPath() + pathSep + row.name,
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

    const createFile = async (name: string, type: string) => {
        const path = options.nowPath() + pathSep + name;
        await machineApi.createFile.request({
            machineId: options.machineId(),
            authCertName: options.authCertName(),
            id: options.fileId(),
            protocol: options.protocol(),
            path,
            type,
        });
        options.refresh();
    };

    const downloadFile = (data: MachineFileInfo) => {
        const a = document.createElement('a');
        a.setAttribute(
            'href',
            `${config.baseApiUrl}/machines/${options.machineId()}/files/${options.fileId()}/download?path=${data.path}&machineId=${options.machineId()}&authCertName=${options.authCertName()}&fileId=${options.fileId()}&protocol=${options.protocol()}&${joinClientParams()}`
        );
        a.setAttribute('target', '_blank');
        a.click();
    };

    const getDirSize = async (data: MachineFileInfo) => {
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

    const showFileStat = async (data: MachineFileInfo) => {
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
                protocol: options.protocol() as number,
                fileId: options.fileId() as number,
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
                protocol: options.protocol() as number,
                fileId: options.fileId() as number,
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

    const dontOperate = (data: MachineFileInfo) => {
        const path = data.path;
        const ls = ['/', '//', '/usr', '/usr/', '/usr/bin', '/opt', '/run', '/etc', '/proc', '/var', '/mnt', '/boot', '/dev', '/home', '/media', '/root'];
        return ls.indexOf(path) != -1;
    };

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
    };
}
