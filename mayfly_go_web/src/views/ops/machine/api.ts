import Api from '@/common/Api';

export const machineApi = {
    // 获取权限列表
    list: Api.create("/machines", 'get'),
    getMachinePwd: Api.create("/machines/{id}/pwd", 'get'),
    info: Api.create("/machines/{id}/sysinfo", 'get'),
    stats: Api.create("/machines/{id}/stats", 'get'),
    process: Api.create("/machines/{id}/process", 'get'),
    // 终止进程
    killProcess: Api.create("/machines/{id}/process", 'delete'),
    closeCli: Api.create("/machines/{id}/close-cli", 'delete'),
    // 保存按钮
    saveMachine: Api.create("/machines", 'post'),
    // 调整状态
    changeStatus: Api.create("/machines/{id}/{status}", 'put'),
    // 删除机器
    del: Api.create("/machines/{id}", 'delete'),
    scripts: Api.create("/machines/{machineId}/scripts", 'get'),
    runScript: Api.create("/machines/{machineId}/scripts/{scriptId}/run", 'get'),
    saveScript: Api.create("/machines/{machineId}/scripts", 'post'),
    deleteScript: Api.create("/machines/{machineId}/scripts/{scriptId}", 'delete'),
    // 获取配置文件列表
    files: Api.create("/machines/{id}/files", 'get'),
    lsFile: Api.create("/machines/{machineId}/files/{fileId}/read-dir", 'get'),
    rmFile: Api.create("/machines/{machineId}/files/{fileId}/remove", 'delete'),
    uploadFile: Api.create("/machines/{machineId}/files/{fileId}/upload?token={token}", 'post'),
    fileContent: Api.create("/machines/{machineId}/files/{fileId}/read", 'get'),
    createFile: Api.create("/machines/{machineId}/files/{id}/create-file", 'post'),
    // 修改文件内容
    updateFileContent: Api.create("/machines/{machineId}/files/{id}/write", 'post'),
    // 添加文件or目录
    addConf: Api.create("/machines/{machineId}/files", 'post'),
    // 删除配置的文件or目录
    delConf: Api.create("/machines/{machineId}/files/{id}", 'delete'),
    terminal: Api.create("/api/machines/{id}/terminal", 'get'),
    recDirNames: Api.create("/machines/rec/names", 'get')
}