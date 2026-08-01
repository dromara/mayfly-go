import { registerMachineFileUploadProgress } from './machine-file-upload-progress';
import { registerFolderUploadProgressHandler } from './machine-folder-upload-progress';

export function initMachineSysMsgs() {
    registerMachineFileUploadProgress();
    registerFolderUploadProgressHandler();
}
